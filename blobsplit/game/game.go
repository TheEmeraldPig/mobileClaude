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
	"slices"
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
	panel  string  // "", "tree", "index", "rebirth", "market", "blob", "card", "world", "round"
	sel    int     // the biote or card shown in the "blob" and "card" panels
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
}

func NewGame() *Game {
	initFonts()
	initSprites()
	initAudio()
	g := &Game{st: loadState(), holdOn: -1}
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
	if g.st.Market && g.panel == "" {
		g.panel = "market"
	}
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
func (g *Game) sellsAllowed() int { return max(1, baseSells+g.m.sells) }

// baseValueAt is a plain biote's value with value level lv and everything that
// multiplies all biotes.
func (g *Game) baseValueAt(lv int) float64 {
	return valueOf(lv) * rebirthMult(g.st.Rebirths) * g.m.value * (1 + g.m.crowd*float64(len(g.blobs)))
}

// mutMult is what mutation r multiplies a biote's value by, cards included.
func (g *Game) mutMult(r int) float64 { return rarities[r].Mult * g.m.worth[r] }

func (g *Game) blobValue(mask int) float64 {
	v := g.baseValueAt(g.st.ValueLv)
	for _, r := range maskList(mask) {
		v *= g.mutMult(r)
	}
	if maskCount(mask) >= 2 {
		v *= g.m.hybrid
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
func (g *Game) childMask(p int) int {
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
	return m
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
	next := make([]blob, 0, n+add)
	for i, p := range g.blobs {
		next = append(next, p)
		if i >= add {
			continue
		}
		next[len(next)-1].squash = g.t
		m := g.childMask(p.m)
		next = append(next, blob{m: m, born: g.t, squash: -10, phase: rand.Float64() * 7, fx: p.fx, fy: p.fy, mt: g.t})
		for _, r := range maskList(m &^ p.m) {
			g.st.Found[r]++
			if g.st.Found[r] == 1 && r > g.banner.r {
				g.banner.r, g.banner.t, g.banner.text = r, g.t, "NEW MUTATION!"
			}
			best = max(best, r)
		}
	}
	g.blobs = next
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
	jackpot := g.m.jackpot > 0 && rand.Float64() < g.m.jackpot
	if jackpot {
		v *= 10
	}
	g.st.Money += v
	g.st.Earned += v
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
	g.roundSale(v)
}

// roundSale counts a sale towards the current round.
func (g *Game) roundSale(v float64) {
	s := &g.st
	s.RoundEarned += v
	s.RoundSells++
	if s.RoundEarned >= roundTarget(s.Ante, s.Round) {
		g.winRound()
	} else if s.RoundSells >= g.sellsAllowed() {
		g.floats = append(g.floats, floater{s: roundNames[s.Round] + " FAILED, TRY AGAIN", x: g.W / 2, y: g.H * 0.3, c: rgb(0xff8a8a), size: 24})
		s.RoundEarned, s.RoundSells = 0, 0
		g.play("deny")
	}
}

func (g *Game) winRound() {
	s := &g.st
	dna := roundReward(s.Round)
	s.Gems += dna
	msg := roundNames[s.Round] + " CLEARED!  +" + itoa(dna) + " DNA"
	if s.Round == 2 {
		// beating the Outbreak gives an ability card of a random rarity
		id := g.randomCard()
		c := cardByID(id)
		if len(s.Cards) < g.cardSlots() {
			s.Cards = append(s.Cards, id)
			msg = "OUTBREAK BEATEN!  +" + itoa(dna) + " DNA  +" + c.name + " card"
		} else {
			s.Gems += cardRarities[c.rar].price
			msg = "OUTBREAK BEATEN!  +" + itoa(dna+cardRarities[c.rar].price) + " DNA (card slots full)"
		}
		g.banner.r, g.banner.t, g.banner.text = c.mut, g.t, cardRarities[c.rar].name+" CARD: "+c.name
		s.RunBosses++
		s.Ante++
		s.Boss = rand.Intn(len(bosses))
	}
	s.Round = (s.Round + 1) % 3
	s.RoundEarned, s.RoundSells = 0, 0
	s.Market, s.MarketMsg, s.Rerolled = true, msg, false
	g.rollOffers()
	g.panel = "market"
	g.play("rebirth")
	g.save()
}

func (g *Game) rollOffers() {
	g.st.Offers = nil
	for tries := 0; len(g.st.Offers) < 3; tries++ {
		id := g.randomCard()
		if tries < 50 && slices.Contains(g.st.Offers, id) {
			continue // three different cards
		}
		g.st.Offers = append(g.st.Offers, id)
	}
}

func (g *Game) buyOffer(i int) {
	s := &g.st
	if i >= len(s.Offers) || s.Offers[i] == "" {
		return
	}
	c := cardByID(s.Offers[i])
	price := cardRarities[c.rar].price
	if s.Gems < price || len(s.Cards) >= g.cardSlots() {
		g.play("deny")
		if len(s.Cards) >= g.cardSlots() {
			g.floats = append(g.floats, floater{s: "CARD SLOTS FULL", x: g.W / 2, y: g.H * 0.5, c: rgb(0xff8a8a)})
		}
		return
	}
	s.Gems -= price
	s.Cards = append(s.Cards, c.id)
	s.Offers[i] = ""
	g.play("buy")
}

func (g *Game) reroll() {
	s := &g.st
	free := g.has("reroll") && !s.Rerolled
	if !free && s.Gems < 1 {
		g.play("deny")
		return
	}
	if !free {
		s.Gems--
	}
	s.Rerolled = true
	g.rollOffers()
	g.play("tap")
}

func (g *Game) closeMarket() {
	g.st.Market = false
	g.panel = ""
	g.play("tap")
}

func (g *Game) sellCard(i int) {
	s := &g.st
	if i >= len(s.Cards) {
		return
	}
	c := cardByID(s.Cards[i])
	if c != nil {
		s.Gems += cardRarities[c.rar].price / 2
	}
	s.Cards = append(s.Cards[:i], s.Cards[i+1:]...)
	g.panel = ""
	g.play("sell")
}

func (g *Game) unlock(id string) {
	n := nodeByID(id)
	if n == nil || !g.canUnlock(n) || g.st.Points < n.cost {
		g.play("deny")
		return
	}
	g.st.Points -= n.cost
	g.st.Tree[id] = true
	switch id {
	case "auto":
		g.st.Auto = true
	case "autosell":
		g.st.AutoSell = true
	}
	g.play("buy")
	g.save()
}

func (g *Game) rebirth() {
	s := &g.st
	if s.Money < rebirthReq(s.Rebirths) {
		g.play("deny")
		return
	}
	s.Points += rebirthPoints(s.RunBosses)
	s.Rebirths++
	s.Money, g.shown = 0, 0
	s.ValueLv, s.SpeedLv, s.CapLv = 0, 0, 0
	s.Ante, s.Round, s.RoundEarned, s.RoundSells, s.RunBosses = 1, 0, 0, 0, 0
	s.Market = false
	g.blobs = []blob{{m: 0, born: g.t, mt: -10, squash: -10}}
	g.cool = 99
	g.panel = "tree"
	for i := 0; i < 80; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := (2 + rand.Float64()*6) * g.u
		g.parts = append(g.parts, particle{x: g.W / 2, y: g.H / 2, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, life: 1 + rand.Float64(), size: 3 * g.u, c: []color.RGBA{colT, colL, colG}[i%3], r: -1})
	}
	g.play("rebirth")
	g.save()
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
	if g.panel != "" {
		g.tapPanel(L, x, y)
		return
	}
	switch {
	case L.split.in(x, y):
		if len(g.blobs) < g.capacity() && g.cool < g.cooldown() {
			g.play("tap")
		}
		g.trySplit(true)
	case L.auto.in(x, y):
		if g.has("auto") {
			g.st.Auto = !g.st.Auto
			g.play("tap")
		} else {
			g.selN = "auto"
			g.open("tree")
		}
	case L.autosell.in(x, y):
		if g.has("autosell") {
			g.st.AutoSell = !g.st.AutoSell
			g.play("tap")
		} else {
			g.selN = "autosell"
			g.open("tree")
		}
	case L.sell.in(x, y):
		g.sell()
	case L.shop.in(x, y):
		g.open("tree")
	case L.index.in(x, y):
		g.open("index")
	case L.rebirth.in(x, y):
		g.open("rebirth")
	case L.world.in(x, y):
		g.open("world")
	case L.round.in(x, y):
		g.open("round")
	case L.sound.in(x, y):
		g.st.Sound = !g.st.Sound
		g.play("tap")
	case L.music.in(x, y):
		g.st.NoMusic = !g.st.NoMusic
		g.updateMusic()
		g.play("tap")
	case L.buy.in(x, y):
		g.st.BuyMode = (g.st.BuyMode + 1) % len(buyModes)
		g.play("tap")
	default:
		for i, c := range L.cards[:nUps] {
			if c.in(x, y) {
				g.buyCard(i, false)
				g.holdOn, g.holdT, g.repeatT = i, 0, 0
				return
			}
		}
		for i, c := range L.hand {
			if c.in(x, y) {
				g.sel = i
				if i < len(g.st.Cards) {
					g.open("card")
				} else {
					g.open("tree") // an empty slot: more slots are in the tree
				}
				return
			}
		}
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
	g.tickEvents()
	g.m = g.computeMods()
	g.hookUpdate()
	L := g.lay()

	presses, down, hx, hy := pointer()
	for _, p := range presses {
		g.tap(L, p[0], p[1])
	}
	if g.holdOn >= 0 {
		if !down || !L.cards[g.holdOn].in(hx, hy) || g.panel != "" {
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
	if g.panel == "" {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.trySplit(true)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			g.sell()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.closePanel()
	}

	g.cool += dt
	if g.panel != "market" {
		if g.st.Auto && g.has("auto") {
			g.trySplit(false)
		}
		if len(g.blobs) >= g.capacity() {
			g.full += dt
			if g.st.AutoSell && g.has("autosell") && g.full > 0.6 {
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
