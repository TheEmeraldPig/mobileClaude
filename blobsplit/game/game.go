// Package game is Biote: an idle game where you split biotes until the board is full
// and sell them. New biotes can gain mutations, which stack and multiply their value.
// Sales count towards rounds (Petri, Colony, Outbreak) that pay DNA for the card
// market; rebirths pay points for the skill tree; day, night and events change which
// mutations show up.
package game

import (
	"image/color"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Version is set at build time (-ldflags "-X .../game.Version=12").
var Version = "dev"

// DataDir is where the save lives. The Android app sets it; on desktop it defaults to
// the user's config folder.
var DataDir string

const dt = 1.0 / 60

type blob struct {
	m      int     // mutation mask
	born   float64 // game time it appeared
	squash float64 // last time it split (it wobbles)
	phase  float64
	// when the board grows, each biote glides from (fx, fy) to its new slot from time mt
	fx, fy, mt float64
}

type particle struct {
	x, y, vx, vy float64
	age, life    float64
	size         float64
	c            color.RGBA
	r            int // >= 0: a sold biote with this sprite flying from (x,y) to (tx,ty); -2: a ring
	tx, ty       float64
	s0           float64 // a sold biote's sprite scale when it was sold
}

type floater struct {
	s    string
	x, y float64
	t    float64
	c    color.RGBA
	size float64 // font size in units (0 = 20)
}

type Game struct {
	mu     sync.Mutex
	st     State
	m      mods // rules in effect this frame
	blobs  []blob
	t      float64
	frame  int
	cool   float64 // seconds since the last split
	full   float64 // seconds the board has been full (auto sell)
	W, H   float64 // screen size in device pixels
	u      float64 // one UI unit in device pixels; the screen is 360 units tall
	panel  string  // "", "tree", "challenge", "index", "rebirth", "blob", "world"
	sel    int     // the biote shown in the "blob" panel, or the card picked in "challenge"
	selN   string  // the skill tree node picked
	parts  []particle
	floats []floater
	banner struct {
		r    int
		t    float64
		text string
	}
	shown   float64 // money as displayed, counting up to the real amount
	cellA   float64 // biote size before the last split, which it grows or shrinks from
	cellT   float64 // time of the last split
	soldT   float64 // time of the last sell (the money counter bounces)
	eventT  float64 // time the current event started (its banner)
	holdOn  int     // upgrade card held down (-1 none)
	holdT   float64
	repeatT float64
	saveT   float64
	quit    bool
	sky     sky

	lastSale float64 // time of the last sale, for the income rate
	away     float64 // seconds away, shown on the welcome-back screen
	awayGain float64
	confirm  bool // the reset button was pressed once
	cardTab  int  // the card rarity shown in the CHALLENGE screen

	awayDoubled bool // the welcome-back gain was doubled with an ad
	ad          adState
	paid        chan string // items whose real-money purchase went through
	adDone      chan bool   // ads that finished (whether they were watched)
	restored    chan string // items the store says this account owns
	vis, offer  visitor     // the visitor on screen, and the one tapped
	shopTab     int
	drawerK     float64 // the upgrades drawer: 0 closed, 1 open
	scroll      float64 // how far a panel's list is scrolled, in units
	drag        struct {
		on, moved, bar bool
		x, y, s0       float64
	}
	press  struct{ x, y, t float64 } // the last press, for squashing buttons
	shakeT float64                   // the board shakes from this time
	playT  float64                   // seconds played towards the next minute (quests)
}

func NewGame() *Game {
	initFonts()
	initSprites()
	initAudio()
	g := &Game{st: loadState(), holdOn: -1, paid: make(chan string, 8), adDone: make(chan bool, 8), restored: make(chan string, 64), shakeT: -10}
	g.press.t = -10
	if !g.st.DrawerShut {
		g.drawerK = 1
	}
	g.banner.r, g.banner.t = -1, -10
	g.eventT = -10
	for _, m := range g.st.Board {
		g.blobs = append(g.blobs, blob{m: m, born: -10, mt: -10, squash: -10, phase: rand.Float64() * 7})
	}
	g.shown = g.st.Money
	g.cool = 99
	g.sky.init()
	g.hookInit()
	g.m = g.computeMods()
	g.checkQuests()
	g.checkAway()
	g.updateMusic()
	current = g
	return g
}

func (g *Game) Layout(ow, oh int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	g.W, g.H = math.Ceil(float64(ow)*s), math.Ceil(float64(oh)*s)
	g.u = g.H / 360
	return int(g.W), int(g.H)
}

// ---- rules ----

func (g *Game) capacity() int     { return capOf(g.st.CapLv) }
func (g *Game) cooldown() float64 { return baseCooldown(g.st.SpeedLv) * g.m.cool }

// baseValueAt is a plain biote's value with value level lv and everything that
// multiplies all biotes.
func (g *Game) baseValueAt(lv int) float64 {
	v := valueOf(lv) * rebirthMult(g.st.Rebirths) * g.m.value * (1 + g.m.crowd*float64(len(g.blobs)))
	if g.st.Perks["golden"] {
		v *= 2
	}
	if g.st.Perks["xmoney"] {
		v *= 1.5
	}
	return v
}

// mutMult is what mutation r multiplies a biote's value by, cards included.
func (g *Game) mutMult(r int) float64 { return rarities[r].Mult * g.m.worth[r] * g.m.genes }

func (g *Game) blobValue(mask int) float64 {
	v := g.baseValueAt(g.st.ValueLv)
	for _, r := range maskList(mask) {
		v *= g.mutMult(r)
	}
	if maskCount(mask) >= 2 {
		v *= g.m.hybrid
	}
	if maskCount(mask) >= 3 {
		v *= g.m.chimera
	}
	if mask == 0 {
		v *= g.m.plain
	}
	return v
}

func (g *Game) boardValue() float64 {
	v := 0.0
	for _, b := range g.blobs {
		v += g.blobValue(b.m)
	}
	return v
}

// gainChance is the chance that a new biote gains mutation r it doesn't have yet.
func (g *Game) gainChance(r int) float64 {
	return math.Min(0.5, mutationChance(r)*g.m.luck*g.m.chance[r])
}

// childMask is the mutations of a biote split from a parent with mask p: a copy of
// the parent's (unless the boss says otherwise), and a chance at each one it doesn't
// have yet.
func (g *Game) childMask(p int, near ...int) int {
	m := 0
	if g.m.inherit {
		m = p
	}
	for k := 0; k < g.m.rolls; k++ {
		for r := 1; r < nMut; r++ {
			if !maskHas(m, r) && rand.Float64() < g.gainChance(r) {
				m |= 1 << r
			}
		}
	}
	// neighbours on the board pass their mutations on now and then
	for _, n := range near {
		for _, r := range maskList(n &^ m) {
			if rand.Float64() < g.spreadChance(r) {
				m |= 1 << r
			}
		}
	}
	return m
}

// spreadChance is the chance that a neighbour's mutation r reaches a new biote.
func (g *Game) spreadChance(r int) float64 {
	return math.Min(0.5, spreadBase*g.m.spread*g.m.spreadMut[r])
}

// neighbours are the biotes next to biote i on the board (left, right, above, below).
func neighbours(L layout, i, n int) []int {
	var out []int
	cols := max(1, L.cols)
	if L.circle != nil {
		cols = n + 1 // a circle: only the ones either side
	}
	for _, j := range []int{i - 1, i + 1, i - cols, i + cols} {
		if j < 0 || j >= n || ((j == i-1 || j == i+1) && j/cols != i/cols) {
			continue
		}
		out = append(out, j)
	}
	return out
}

func (g *Game) trySplit(manual bool) {
	n, c := len(g.blobs), g.capacity()
	if n >= c {
		if manual {
			g.play("deny")
			g.float("FULL! SELL", g.lay().split, colT)
		}
		return
	}
	if g.cool < g.cooldown() {
		return
	}
	g.cool = 0
	add := n
	if n+add > c {
		add = c - n
	}
	// remember where everything is now, so biotes glide to their new slots
	L := g.lay()
	g.cellA, g.cellT = g.cellNow(L), g.t
	for i := range g.blobs {
		g.blobs[i].fx, g.blobs[i].fy, _ = g.blobPos(L, i)
		g.blobs[i].mt = g.t
	}
	// each new biote goes right after the one it split from, so they sit side by side
	best := 0
	var gains, kids []int
	next := make([]blob, 0, n+add)
	for i, p := range g.blobs {
		next = append(next, p)
		if i >= add {
			continue
		}
		next[len(next)-1].squash = g.t
		var near []int
		for _, j := range neighbours(L, i, n) {
			near = append(near, g.blobs[j].m)
		}
		m := g.childMask(p.m, near...)
		kids = append(kids, m)
		next = append(next, blob{m: m, born: g.t, squash: -10, phase: rand.Float64() * 7, fx: p.fx, fy: p.fy, mt: g.t})
		for _, r := range maskList(m &^ p.m) {
			g.st.Found[r]++
			if g.st.Found[r] == 1 && r > g.banner.r {
				g.banner.r, g.banner.t, g.banner.text = r, g.t, "NEW MUTATION!"
			}
			best = max(best, r)
			gains = append(gains, r)
		}
	}
	g.blobs = next
	for _, r := range gains {
		g.gained(r)
	}
	g.questAdd(qSplit, 1)
	g.questAdd(qMutate, float64(len(gains)))
	for _, m := range kids {
		if m != 0 {
			g.questAdd(qRare, comboOdds(m))
		}
	}
	g.play("split")
	if best > 0 {
		g.play("rare")
		if best >= 3 {
			ebiten.Vibrate(&ebiten.VibrateOptions{Duration: 60e6, Magnitude: 0.6})
		}
	}
}

func (g *Game) sell() {
	if len(g.blobs) < 2 {
		g.play("deny")
		g.float("SPLIT FIRST", g.lay().sell, colW)
		return
	}
	L := g.lay()
	v := g.boardValue()
	mutated := 0
	for _, b := range g.blobs {
		if b.m != 0 {
			mutated++
		}
	}
	jackpot := g.m.jackpot > 0 && rand.Float64() < g.m.jackpot
	if jackpot {
		v *= 10
	}
	g.st.Money += v
	g.st.Earned += v
	g.noteSale(v)
	// Every biote pops (a ring and a burst of coins), hops up, then flies in an arc to
	// the money counter, in a wave across the board.
	tx, ty := L.money.x+20*g.u, L.money.cy()
	for i, b := range g.blobs {
		x, y, s := g.blobPos(L, i)
		wait := 0.18 + float64(i%max(1, L.cols))*0.025 + float64(i/max(1, L.cols))*0.04
		top := maskTop(b.m)
		g.parts = append(g.parts, particle{x: x, y: y, r: top, tx: tx, ty: ty, age: -wait, life: 0.6, s0: spriteScale(s)})
		g.parts = append(g.parts, particle{x: x, y: y, r: -2, life: 0.4, size: s * 0.45, c: rgb(rarities[top].Label)})
		for k := 0; k < 3; k++ {
			a := rand.Float64() * 2 * math.Pi
			sp := (1.5 + rand.Float64()*2.5) * g.u
			g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 2*g.u, life: 0.5 + rand.Float64()*0.3, size: math.Max(2, 2.5*g.u), c: []color.RGBA{colG, colT}[k%2], r: -1})
		}
	}
	label := "+$" + fmtNum(v)
	if jackpot {
		label = "JACKPOT! " + label
	}
	g.floats = append(g.floats, floater{s: label, x: L.grid.cx(), y: L.grid.cy(), c: colT, size: 34})
	g.blobs = []blob{{m: 0, born: g.t, mt: -10, squash: -10, phase: rand.Float64() * 7}}
	if g.panel == "blob" {
		g.panel = ""
	}
	g.cool = 0 // the split timer starts again
	g.full = 0
	g.play("sell")
	g.questAdd(qSell, 1)
	g.roomSale(v, mutated)
}

// autoSlow is how much longer AUTO waits: twice the time until its skill is unlocked.
func autoSlow(skill bool) float64 {
	if skill {
		return 1
	}
	return 2
}

func (g *Game) unlock(id string) {
	n := nodeByID(id)
	if n == nil || !g.canUnlock(n) || g.st.Points < n.cost {
		g.play("deny")
		return
	}
	g.st.Points -= n.cost
	g.st.Tree[id] = true
	g.play("buy")
	g.save()
}

// rebirth rebirths if you have the money, or anyway when paid is set (bought in the
// shop).
func (g *Game) rebirth(paid bool) {
	s := &g.st
	if !paid && s.Money < rebirthReq(s.Rebirths) {
		g.play("deny")
		return
	}
	pts, dna, tokens := g.rebirthRewards()
	s.PtFrac = g.pointsDue() - float64(pts)
	s.Points += pts
	s.Gems += dna
	s.Tokens += tokens
	s.Rebirths++
	s.Money, g.shown = 0, 0
	s.ValueLv, s.SpeedLv, s.CapLv = 0, 0, 0
	s.Floor, s.Room, s.RunFloors = 1, 0, 0
	g.resetRoom()
	g.blobs = []blob{{m: 0, born: g.t, mt: -10, squash: -10}}
	g.cool = 99
	g.panel = "tree"
	g.confetti(80)
	for i := 0; i < 80; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := (2 + rand.Float64()*6) * g.u
		g.parts = append(g.parts, particle{x: g.W / 2, y: g.H / 2, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, life: 1 + rand.Float64(), size: 3 * g.u, c: []color.RGBA{colT, colL, colG}[i%3], r: -1})
	}
	g.play("rebirth")
	g.save()
}

// rebirthRewards is what rebirthing now pays: one skill point, DNA for every floor
// cleared this run, and a token for every time your money doubled the requirement.
func (g *Game) rebirthRewards() (points, dna, tokens int) {
	s := &g.st
	dna = max(1, s.RunFloors)
	if g.has("bank") {
		dna += dna / 2
	}
	if req := rebirthReq(s.Rebirths); s.Money >= req {
		tokens = int(math.Log2(s.Money / req))
	}
	if g.has("reroll") {
		tokens++
	}
	if s.Perks["xdna"] {
		dna = int(math.Ceil(float64(dna) * 1.5))
	}
	if s.Perks["xtokens"] {
		tokens = int(math.Ceil(float64(tokens) * 1.5))
	}
	return int(g.pointsDue()), dna, tokens
}

// pointsDue is the skill points a rebirth pays, with the part point carried over
// from the last one when the x1.5 boost is on.
func (g *Game) pointsDue() float64 {
	if g.st.Perks["xpoints"] {
		return 1.5 + g.st.PtFrac
	}
	return 1
}

// Upgrade cards: 0 biote value, 1 split speed, 2 capacity.

const nUps = 3

// upLv points at an upgrade's level in the save.
func (g *Game) upLv(i int) *int {
	return []*int{&g.st.ValueLv, &g.st.SpeedLv, &g.st.CapLv}[i]
}

func upCost(i, l int) float64 {
	return []func(int) float64{valueCost, speedCost, capCost}[i](l)
}

func upMax(i int) int { return []int{1 << 30, maxSpeedLv, maxCapLv}[i] }

// The buy amount button: x1, x5, x25 or as many as you can afford.
var buyModes = []string{"x1", "x5", "x25", "MAX"}

// buyPlan is how many levels of upgrade i the buy amount would buy now, and their
// total cost. With MAX and nothing affordable it shows the price of one level.
func (g *Game) buyPlan(i int) (n int, cost float64) {
	lv, left := *g.upLv(i), upMax(i)-*g.upLv(i)
	if g.st.BuyMode < 3 {
		n = min([]int{1, 5, 25}[g.st.BuyMode], left)
		for k := 0; k < n; k++ {
			cost += upCost(i, lv+k)
		}
		return n, cost
	}
	for n < left && n < 1000 && cost+upCost(i, lv+n) <= g.st.Money {
		cost += upCost(i, lv+n)
		n++
	}
	if n == 0 && left > 0 {
		return 1, upCost(i, lv)
	}
	return n, cost
}

func (g *Game) cardInfo(i int) (title, from, to string, cost float64, maxed bool) {
	lv := *g.upLv(i)
	n, cost := g.buyPlan(i)
	maxed = lv >= upMax(i)
	var f func(l int) string
	switch i {
	case 0:
		title = "BIOTE VALUE"
		f = func(l int) string { return "$" + fmtNum(g.baseValueAt(l)) }
	case 1:
		title = "SPLIT SPEED"
		f = func(l int) string { return sprintf1(baseCooldown(l)*g.m.cool) + "s" }
	default:
		title = "CAPACITY"
		f = func(l int) string { return fmtNum(float64(capOf(l))) }
	}
	from, to = f(lv), f(lv+n)
	if i == 2 {
		from += " BIOTES"
	}
	return title, from, to, cost, maxed
}

func (g *Game) buyCard(i int, repeat bool) bool {
	n, cost := g.buyPlan(i)
	if *g.upLv(i) >= upMax(i) || n == 0 || g.st.Money < cost {
		if !repeat {
			g.play("deny")
		}
		return false
	}
	g.st.Money -= cost
	*g.upLv(i) += n
	g.questAdd(qUpgrade, float64(n))
	g.play("buy")
	r := g.lay().cards[i]
	for k := 0; k < 8+min(n, 25); k++ {
		g.parts = append(g.parts, particle{x: r.x + rand.Float64()*r.w, y: r.y + r.h, vx: (rand.Float64() - 0.5) * 2 * g.u, vy: -(1 + rand.Float64()*2) * g.u, life: 0.5, size: 2 * g.u, c: colT, r: -1})
	}
	return true
}

func (g *Game) float(s string, r rect, c color.RGBA) {
	g.floats = append(g.floats, floater{s: s, x: r.cx(), y: r.y - 26*g.u, t: 0, c: c})
}

func itoa(n int) string { return fmtNum(float64(n)) }

// ---- input ----

// pointer returns this frame's new presses and where a held pointer is.
func pointer() (presses [][2]float64, down bool, hx, hy float64) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		presses = append(presses, [2]float64{float64(x), float64(y)})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		down, hx, hy = true, float64(x), float64(y)
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		presses = append(presses, [2]float64{float64(x), float64(y)})
	}
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		down, hx, hy = true, float64(x), float64(y)
	}
	return
}

func (g *Game) open(panel string) {
	g.panel = panel
	g.play("tap")
}

func (g *Game) tap(L layout, x, y float64) {
	g.press.x, g.press.y, g.press.t = x, y, g.t
	if g.ad.on {
		return // an ad is playing
	}
	if g.panel == "tree" {
		g.tapTree(x, y)
		return
	}
	if g.panel == "challenge" {
		g.tapChallenge(x, y)
		return
	}
	if g.panel != "" {
		g.tapPanel(L, x, y)
		return
	}
	if g.tapTut(L, x, y) || g.tapVisitor(x, y) {
		return
	}
	switch {
	case L.handle.in(x, y):
		g.st.DrawerShut = !g.st.DrawerShut
		g.play("tap")
	case L.split.in(x, y):
		if len(g.blobs) < g.capacity() && g.cool < g.cooldown() {
			g.play("tap")
		}
		g.trySplit(true)
	case L.auto.in(x, y):
		g.st.Auto = !g.st.Auto
		g.play("tap")
	case L.autosell.in(x, y):
		g.st.AutoSell = !g.st.AutoSell
		g.play("tap")
	case L.sell.in(x, y):
		g.sell()
	case L.shop.in(x, y):
		g.open("tree")
	case L.index.in(x, y):
		g.scroll = 0
		g.open("index")
	case L.rebirth.in(x, y):
		g.open("rebirth")
	case L.world.in(x, y):
		g.open("world")
	case L.challenge.in(x, y):
		g.open("challenge")
	case L.sound.in(x, y):
		g.confirm = false
		g.open("settings")
	case L.music.in(x, y):
		g.scroll = 0
		g.open("shop")
	case L.quests.in(x, y):
		g.checkQuests()
		g.open("quests")
	case L.gems.in(x, y):
		g.open("index") // the DNA lab
	case L.buy.in(x, y) && g.drawerK > 0.5:
		g.st.BuyMode = (g.st.BuyMode + 1) % len(buyModes)
		g.play("tap")
	case L.drawer.in(x, y) && g.drawerK > 0.5:
		for i, c := range L.cards {
			if c.in(x, y) {
				g.buyCard(i, false)
				g.holdOn, g.holdT, g.repeatT = i, 0, 0
				return
			}
		}
	default:
		// a biote: show its mutations
		for i := range g.blobs {
			bx, by, s := g.blobPos(L, i)
			if math.Abs(x-bx) < s/2 && math.Abs(y-by) < s/2 {
				g.sel = i
				g.open("blob")
				return
			}
		}
	}
}

func (g *Game) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.quit || ebiten.IsWindowBeingClosed() {
		g.save()
		return ebiten.Termination
	}
	g.t += dt
	g.frame++
	g.tickShop()
	g.tickEvents()
	g.tickVisitors()
	g.m = g.computeMods()
	g.hookUpdate()
	g.tickTut()
	g.tickQuests()
	want := 1.0
	if g.st.DrawerShut {
		want = 0
	}
	if g.drawerK += (want - g.drawerK) * math.Min(1, dt*12); math.Abs(want-g.drawerK) < 0.002 {
		g.drawerK = want
	}
	L := g.lay()

	presses, down, hx, hy := pointer()
	g.pointerInput(L, presses, down, hx, hy)
	if g.holdOn >= 0 {
		if !down || !L.cards[g.holdOn].in(hx, hy) || g.panel != "" || g.ad.on {
			g.holdOn = -1
		} else if g.holdT += dt; g.holdT > 0.4 {
			for g.repeatT += dt; g.repeatT > 0.07; g.repeatT -= 0.07 {
				if !g.buyCard(g.holdOn, true) {
					g.holdOn = -1
					break
				}
			}
		}
	}
	if g.panel == "" && !g.ad.on {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.trySplit(true)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			g.sell()
		}
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)) && !g.ad.on {
		g.closePanel()
	}

	g.cool += dt
	{
		// AUTO works from the start at half speed; its skills make it full speed
		if g.st.Auto && g.cool >= g.cooldown()*autoSlow(g.has("auto")) {
			g.trySplit(false)
		}
		if len(g.blobs) >= g.capacity() {
			g.full += dt
			if g.st.AutoSell && g.full >= g.cooldown()*autoSlow(g.has("autosell")) {
				g.sell()
			}
		} else {
			g.full = 0
		}
	}

	// sparkles around mutated biotes
	for i, b := range g.blobs {
		if top := maskTop(b.m); top > 0 && rand.Float64() < 0.04+0.01*float64(top) {
			x, y, s := g.blobPos(L, i)
			a := rand.Float64() * 2 * math.Pi
			g.parts = append(g.parts, particle{x: x + math.Cos(a)*s*0.5, y: y + math.Sin(a)*s*0.5, vy: -0.4 * g.u, life: 0.6, size: math.Max(2, 1.5*g.u), c: rgb(rarities[top].Label), r: -1})
		}
	}
	var spawn []particle
	live := g.parts[:0]
	for _, p := range g.parts {
		p.age += dt
		if p.r == -1 {
			p.x += p.vx
			p.y += p.vy
			p.vy += 0.05 * g.u
			p.vx *= 0.97
		} else if p.r == -3 { // confetti flutters down
			p.x += p.vx + math.Sin(p.age*7+p.tx)*0.6*g.u
			p.y += p.vy
			p.vy = math.Min(p.vy+0.03*g.u, 1.6*g.u)
			p.vx *= 0.98
		}
		if p.age < p.life {
			live = append(live, p)
		} else if p.r >= 0 {
			// a sold biote reached the money counter
			g.soldT = g.t
			for k := 0; k < 2; k++ {
				spawn = append(spawn, particle{x: p.tx, y: p.ty, vx: (rand.Float64() - 0.5) * 3 * g.u, vy: -rand.Float64() * 3 * g.u, life: 0.4, size: 2 * g.u, c: colG, r: -1})
			}
		}
	}
	g.parts = append(live, spawn...)
	fl := g.floats[:0]
	for _, f := range g.floats {
		if f.t += dt; f.t < 1.3 {
			fl = append(fl, f)
		}
	}
	g.floats = fl
	g.sky.update(g)
	g.tickMusic()
	g.shown += (g.st.Money - g.shown) * math.Min(1, dt*8)
	if math.Abs(g.st.Money-g.shown) < 0.5 || g.st.Money < g.shown {
		g.shown = g.st.Money
	}
	if g.saveT += dt; g.saveT > 10 {
		g.saveT = 0
		g.save()
	}
	return nil
}

// pointerInput hands this frame's presses to tap. In a panel with a scrolling list,
// a press there waits to see if it becomes a drag (which scrolls) before it counts
// as a tap; the mouse wheel scrolls too.
func (g *Game) pointerInput(L layout, presses [][2]float64, down bool, hx, hy float64) {
	scrolls := L.view.w > 0 && !g.ad.on
	if scrolls {
		if _, wy := ebiten.Wheel(); wy != 0 {
			g.scroll -= wy * 30
		}
	}
	for _, p := range presses {
		if scrolls && (L.view.in(p[0], p[1]) || L.bar.in(p[0], p[1])) {
			g.drag.on, g.drag.moved, g.drag.bar = true, false, L.bar.in(p[0], p[1])
			g.drag.x, g.drag.y, g.drag.s0 = p[0], p[1], g.scroll
			continue
		}
		g.tap(L, p[0], p[1])
	}
	if !g.drag.on {
		return
	}
	switch {
	case !scrolls:
		g.drag.on = false
	case down && g.drag.bar:
		g.drag.moved = true
		g.scroll = (hy - L.bar.y) / math.Max(1, L.bar.h) * L.scrollMax
	case down:
		if math.Abs(hy-g.drag.y) > 6*g.u {
			g.drag.moved = true
		}
		if g.drag.moved {
			g.scroll = g.drag.s0 - (hy-g.drag.y)/g.u
		}
	default:
		g.drag.on = false
		if !g.drag.moved {
			g.tap(L, g.drag.x, g.drag.y)
		}
	}
}
