// Package game is Blob Split: an idle game where you split blobs until the board is
// full, sell them, buy upgrades and rebirth for bigger multipliers. Rare blobs can
// appear on every split.
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
	r      int     // rarity
	born   float64 // game time it appeared
	squash float64 // last time it split (it wobbles)
	phase  float64
	// when the board grows, each blob glides from (fx, fy) to its new slot from time mt
	fx, fy, mt float64
}

type particle struct {
	x, y, vx, vy float64
	age, life    float64
	size         float64
	c            color.RGBA
	r            int // >= 0: a sold blob of this rarity flying from (x,y) to (tx,ty); -2: a ring
	tx, ty       float64
	s0           float64 // a sold blob's sprite scale when it was sold
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
	blobs  []blob
	t      float64
	frame  int
	cool   float64 // seconds since the last split
	full   float64 // seconds the board has been full (auto sell)
	W, H   float64 // screen size in device pixels
	u      float64 // one UI unit in device pixels; the screen is 360 units tall
	panel  string  // "", "shop", "index" or "rebirth"
	parts  []particle
	floats []floater
	banner struct {
		r int
		t float64
	}
	shown   float64 // money as displayed, counting up to the real amount
	cellA   float64 // blob size before the last split, which it grows or shrinks from
	cellT   float64 // time of the last split
	soldT   float64 // time of the last sell (the money counter bounces)
	holdOn  int     // upgrade card held down (-1 none)
	holdT   float64
	repeatT float64
	saveT   float64
	quit    bool
	bg      *ebiten.Image
	bgKey   [2]int
	stars   [][3]float64
}

func NewGame() *Game {
	initFonts()
	initSprites()
	initAudio()
	g := &Game{st: loadState(), holdOn: -1, banner: struct {
		r int
		t float64
	}{-1, -10}}
	for _, r := range g.st.Blobs {
		g.blobs = append(g.blobs, blob{r: r, born: -10, mt: -10, squash: -10, phase: rand.Float64() * 7})
	}
	g.shown = g.st.Money
	g.cool = 99
	for i := 0; i < 70; i++ {
		g.stars = append(g.stars, [3]float64{rand.Float64(), rand.Float64() * 0.55, rand.Float64() * 7})
	}
	g.hookInit()
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

func (g *Game) gem(id string) int { return g.st.GemLv[id] }
func (g *Game) capacity() int     { return capOf(g.st.CapLv) }
func (g *Game) cooldown() float64 {
	return baseCooldown(g.st.SpeedLv) * math.Pow(0.9, float64(g.gem("quick")))
}
func (g *Game) luck() float64 { return luckOf(g.st.LuckLv) * (1 + 0.5*float64(g.gem("lucky"))) }
func (g *Game) blobValueAt(lv int) float64 {
	return valueOf(lv) * rebirthMult(g.st.Rebirths) * (1 + 0.5*float64(g.gem("golden")))
}
func (g *Game) boardValue() float64 {
	v := 0.0
	for _, b := range g.blobs {
		v += rarities[b.r].Mult
	}
	return v * g.blobValueAt(g.st.ValueLv)
}

// roll picks the rarity of a new blob split from a parent of rarity min: it is a
// copy of the parent, or with luck something rarer.
func (g *Game) roll(min int) int {
	l := g.luck()
	for r := len(rarities) - 1; r > min; r-- {
		if rand.Float64() < l/rarities[r].Odds {
			return r
		}
	}
	return min
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
	// remember where everything is now, so blobs glide to their new slots
	L := g.lay()
	g.cellA, g.cellT = g.cellNow(L), g.t
	for i := range g.blobs {
		g.blobs[i].fx, g.blobs[i].fy, _ = g.blobPos(L, i)
		g.blobs[i].mt = g.t
	}
	best := 0
	for i := 0; i < add; i++ {
		r := g.roll(g.blobs[i].r)
		g.blobs[i].squash = g.t
		g.blobs = append(g.blobs, blob{r: r, born: g.t, squash: -10, phase: rand.Float64() * 7, fx: g.blobs[i].fx, fy: g.blobs[i].fy, mt: g.t})
		g.st.Found[r]++
		if r > 0 && g.st.Found[r] == 1 && r > g.banner.r {
			g.banner.r, g.banner.t = r, g.t
		}
		if r > g.blobs[i].r {
			best = max(best, r) // only a new, rarer roll is celebrated
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
	g.st.Money += v
	g.st.Earned += v
	// Every blob pops (a ring and a burst of coins), hops up, then flies in an arc to
	// the money counter, in a wave across the board.
	tx, ty := L.money.x+20*g.u, L.money.cy()
	for i, b := range g.blobs {
		x, y, s := g.blobPos(L, i)
		wait := 0.18 + float64(i%L.cols)*0.025 + float64(i/L.cols)*0.04
		g.parts = append(g.parts, particle{x: x, y: y, r: b.r, tx: tx, ty: ty, age: -wait, life: 0.6, s0: spriteScale(s)})
		g.parts = append(g.parts, particle{x: x, y: y, r: -2, life: 0.4, size: s * 0.45, c: rgb(rarities[b.r].Label)})
		for k := 0; k < 3; k++ {
			a := rand.Float64() * 2 * math.Pi
			sp := (1.5 + rand.Float64()*2.5) * g.u
			g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 2*g.u, life: 0.5 + rand.Float64()*0.3, size: math.Max(2, 2.5*g.u), c: []color.RGBA{colG, colT}[k%2], r: -1})
		}
	}
	g.floats = append(g.floats, floater{s: "+$" + fmtNum(v), x: L.grid.cx(), y: L.grid.cy(), c: colT, size: 34})
	g.blobs = []blob{{r: 0, born: g.t, mt: -10, squash: -10, phase: rand.Float64() * 7}}
	g.cool = 99
	g.full = 0
	g.play("sell")
}

func (g *Game) rebirth() {
	if g.st.Money < rebirthReq(g.st.Rebirths) {
		g.play("deny")
		return
	}
	g.st.Gems += rebirthGems(g.st.Rebirths)
	g.st.Rebirths++
	g.st.Money, g.shown = 0, 0
	g.st.ValueLv, g.st.SpeedLv, g.st.CapLv, g.st.LuckLv = 0, 0, 0, 0
	g.blobs = []blob{{r: 0, born: g.t, mt: -10, squash: -10}}
	g.cool = 99
	g.panel = ""
	for i := 0; i < 80; i++ {
		a := rand.Float64() * 2 * math.Pi
		s := (2 + rand.Float64()*6) * g.u
		g.parts = append(g.parts, particle{x: g.W / 2, y: g.H / 2, vx: math.Cos(a) * s, vy: math.Sin(a) * s, life: 1 + rand.Float64(), size: 3 * g.u, c: []color.RGBA{colT, colL, colG}[i%3], r: -1})
	}
	g.play("rebirth")
	g.save()
}

// card i: 0 blob value, 1 split speed, 2 capacity, 3 luck
func (g *Game) cardInfo(i int) (title, from, to string, cost float64, maxed bool) {
	s := &g.st
	switch i {
	case 0:
		return "BLOB VALUE", "$" + fmtNum(g.blobValueAt(s.ValueLv)), "$" + fmtNum(g.blobValueAt(s.ValueLv+1)), valueCost(s.ValueLv), false
	case 1:
		q := math.Pow(0.9, float64(g.gem("quick")))
		f := func(l int) string { return trimZeros(sprintf1(baseCooldown(l)*q)) + " SEC" }
		return "SPLIT SPEED", f(s.SpeedLv), f(s.SpeedLv + 1), speedCost(s.SpeedLv), s.SpeedLv >= maxSpeedLv
	case 2:
		return "CAPACITY", fmtNum(float64(capOf(s.CapLv))) + " BLOBS", fmtNum(float64(capOf(s.CapLv + 1))), capCost(s.CapLv), s.CapLv >= maxCapLv
	default:
		m := 1 + 0.5*float64(g.gem("lucky"))
		f := func(l int) string { return "x" + trimZeros(sprintf1(luckOf(l)*m)) }
		return "LUCK", f(s.LuckLv), f(s.LuckLv + 1), luckCost(s.LuckLv), s.LuckLv >= maxLuckLv
	}
}

func (g *Game) buyCard(i int, repeat bool) bool {
	_, _, _, cost, maxed := g.cardInfo(i)
	if maxed || g.st.Money < cost {
		if !repeat {
			g.play("deny")
		}
		return false
	}
	g.st.Money -= cost
	switch i {
	case 0:
		g.st.ValueLv++
	case 1:
		g.st.SpeedLv++
	case 2:
		g.st.CapLv++
	case 3:
		g.st.LuckLv++
	}
	g.play("buy")
	r := g.lay().cards[i]
	for k := 0; k < 8; k++ {
		g.parts = append(g.parts, particle{x: r.x + rand.Float64()*r.w, y: r.y + r.h, vx: (rand.Float64() - 0.5) * 2 * g.u, vy: -(1 + rand.Float64()*2) * g.u, life: 0.5, size: 2 * g.u, c: colT, r: -1})
	}
	return true
}

func (g *Game) buyGem(i int) {
	u := gemUps[i]
	lv := g.gem(u.id)
	if lv >= u.max {
		// one-off unlocks become on/off switches once bought
		switch u.id {
		case "auto":
			g.st.Auto = !g.st.Auto
		case "autosell":
			g.st.AutoSell = !g.st.AutoSell
		}
		g.play("tap")
		return
	}
	c := gemCost(u, lv)
	if g.st.Gems < c {
		g.play("deny")
		return
	}
	g.st.Gems -= c
	g.st.GemLv[u.id] = lv + 1
	switch u.id {
	case "auto":
		g.st.Auto = true
	case "autosell":
		g.st.AutoSell = true
	}
	g.play("buy")
	g.save()
}

func (g *Game) float(s string, r rect, c color.RGBA) {
	g.floats = append(g.floats, floater{s: s, x: r.cx(), y: r.y - 4*g.u, t: 0, c: c})
}

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
	case L.sell.in(x, y):
		g.sell()
	case L.auto.in(x, y):
		if g.gem("auto") > 0 {
			g.st.Auto = !g.st.Auto
			g.play("tap")
		} else {
			g.panel = "shop"
			g.play("tap")
		}
	case L.shop.in(x, y):
		g.panel = "shop"
		g.play("tap")
	case L.index.in(x, y):
		g.panel = "index"
		g.play("tap")
	case L.rebirth.in(x, y), L.chip.in(x, y):
		g.panel = "rebirth"
		g.play("tap")
	case L.sound.in(x, y):
		g.st.Sound = !g.st.Sound
		g.play("tap")
	default:
		for i, c := range L.cards {
			if c.in(x, y) {
				g.buyCard(i, false)
				g.holdOn, g.holdT, g.repeatT = i, 0, 0
			}
		}
	}
}

func (g *Game) tapPanel(L layout, x, y float64) {
	if L.close.in(x, y) || !L.panel.in(x, y) {
		g.panel = ""
		g.play("tap")
		return
	}
	switch g.panel {
	case "shop":
		for i, r := range L.rows {
			if r.in(x, y) {
				g.buyGem(i)
			}
		}
	case "rebirth":
		if L.action.in(x, y) {
			g.rebirth()
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
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.panel == "" {
		g.trySplit(true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.panel == "" {
		g.sell()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.panel = ""
	}

	g.cool += dt
	if g.st.Auto && g.gem("auto") > 0 {
		g.trySplit(false)
	}
	if len(g.blobs) >= g.capacity() {
		g.full += dt
		if g.st.AutoSell && g.gem("autosell") > 0 && g.full > 0.6 {
			g.sell()
		}
	} else {
		g.full = 0
	}

	// sparkles around rare blobs
	for i, b := range g.blobs {
		if b.r > 0 && rand.Float64() < 0.04+0.01*float64(b.r) {
			x, y, s := g.blobPos(L, i)
			a := rand.Float64() * 2 * math.Pi
			g.parts = append(g.parts, particle{x: x + math.Cos(a)*s*0.5, y: y + math.Sin(a)*s*0.5, vy: -0.4 * g.u, life: 0.6, size: math.Max(2, 1.5*g.u), c: rgb(rarities[b.r].Label), r: -1})
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
			// a sold blob reached the money counter
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

func sprintf1(f float64) string {
	return trimZeros(fmtFixed(f, 1))
}
