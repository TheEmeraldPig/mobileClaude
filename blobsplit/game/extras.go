package game

import (
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Visitors, timed buffs, the tutorial and the little animations that make taps feel
// good: buttons that squash, confetti and a shaking board.

// ---- buffs: timed boosts from visitors ----

type buffDef struct {
	id, name, desc string
	short          string  // for narrow screens
	secs           float64 // how long it lasts; 0 pays out at once
	col            uint32
}

var buffs = []buffDef{
	{"cash", "DOUBLE CASH", "BIOTE VALUE X2", "CASH X2", 120, 0x6dff7a},
	{"luck", "LUCKY STAR", "MUTATION CHANCE X3", "LUCK X3", 120, 0xffc94a},
	{"rush", "SPLIT RUSH", "SPLITS 2X FASTER", "RUSH", 90, 0x5fd0ff},
	{"bank", "PIGGY BANK", "2 MINUTES OF INCOME, NOW", "BANK", 0, 0xff6fb5},
}

func (g *Game) buff(id string) bool { return g.st.Buffs[id] > 0 }

// giveBuff starts buff i, k times as long (or k times the money).
func (g *Game) giveBuff(i int, k float64) {
	b := buffs[i]
	if b.secs == 0 {
		g.timeSkip(120 * k)
	} else {
		g.st.Buffs[b.id] += b.secs * k
		g.floats = append(g.floats, floater{s: b.name + "  " + fmtTime(b.secs*k), x: g.W / 2, y: g.H * 0.45, c: rgb(b.col), size: 26})
	}
	g.confetti(40)
	g.play("rebirth")
	g.save()
}

// ---- visitors: now and then something drifts across the screen. Tap it for a buff:
// a gift is free and an ad doubles it; an offer needs an ad. ----

type visitor struct {
	on   bool
	t0   float64 // when it appeared
	buff int
	gift bool
	dir  float64 // 1: left to right
	y    float64 // height, in units
}

const visitLen = 11.0 // seconds to cross the screen

func (g *Game) tickVisitors() {
	s := &g.st
	for id, left := range s.Buffs {
		if left -= dt; left <= 0 {
			delete(s.Buffs, id)
		} else {
			s.Buffs[id] = left
		}
	}
	if g.vis.on && g.t-g.vis.t0 > visitLen {
		g.vis.on = false
	}
	if g.vis.on || g.panel != "" || g.ad.on || s.Tut < len(tutSteps) {
		return
	}
	if s.NextVisit -= dt; s.NextVisit > 0 {
		return
	}
	g.sendVisitor()
}

func (g *Game) sendVisitor() {
	g.st.NextVisit = 75 + rand.Float64()*90
	dir := 1.0
	if rand.Intn(2) == 0 {
		dir = -1
	}
	g.vis = visitor{on: true, t0: g.t, buff: rand.Intn(len(buffs)), gift: rand.Intn(2) == 0, dir: dir, y: 90 + rand.Float64()*90}
	g.play("rare")
}

// visitorPos is where the visitor is at time t.
func (g *Game) visitorPos(t float64) (x, y float64) {
	k := (t - g.vis.t0) / visitLen
	if g.vis.dir < 0 {
		k = 1 - k
	}
	x = -30*g.u + (g.W+60*g.u)*k
	y = g.vis.y*g.u + math.Sin(k*math.Pi*5)*16*g.u
	return
}

// tapVisitor catches the visitor if (x, y) is on it.
func (g *Game) tapVisitor(x, y float64) bool {
	if !g.vis.on {
		return false
	}
	vx, vy := g.visitorPos(g.t)
	if math.Hypot(x-vx, y-vy) > 30*g.u {
		return false
	}
	g.offer = g.vis
	g.vis.on = false
	g.panel = "visitor"
	g.play("buy")
	return true
}

func (g *Game) drawVisitor(dst *ebiten.Image) {
	if !g.vis.on {
		return
	}
	u := g.u
	c := rgb(buffs[g.vis.buff].col)
	// a sparkly trail, then a glowing orb with the buff's symbol
	for k := 8; k > 0; k-- {
		x, y := g.visitorPos(g.t - float64(k)*0.06)
		a := uint8(120 - 13*k)
		vector.DrawFilledCircle(dst, float32(x), float32(y), float32((10-float64(k))*1.4*u), color.NRGBA{c.R, c.G, c.B, a}, true)
	}
	x, y := g.visitorPos(g.t)
	g.drawOrb(dst, g.vis.buff, x, y, 1)
	say(dst, "TAP!", face(fSmall, 7*u), x, y-26*u+math.Sin(g.t*6)*2*u, alCenter, colT, math.Round(u))
	if rand.Float64() < 0.5 {
		g.parts = append(g.parts, particle{x: x, y: y, vx: (rand.Float64() - 0.5) * u, vy: (rand.Float64() - 0.5) * u, life: 0.5, size: math.Max(2, 2*u), c: c, r: -1})
	}
}

// drawOrb is a buff's glowing orb with its symbol, k times the normal size.
func (g *Game) drawOrb(dst *ebiten.Image, buff int, x, y, k float64) {
	u := g.u * k
	c := rgb(buffs[buff].col)
	pulse := 1 + 0.12*math.Sin(g.t*8)
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(22*u*pulse), color.NRGBA{c.R, c.G, c.B, 60}, true)
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(15*u), colO, true)
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(13*u), c, true)
	vector.DrawFilledCircle(dst, float32(x-4*u), float32(y-4*u), float32(4*u), color.NRGBA{255, 255, 255, 150}, true)
	switch buffs[buff].id {
	case "luck":
		icon(dst, "star", x, y, 16*u)
	case "bank":
		icon(dst, "heart", x, y, 14*u)
	case "rush":
		say(dst, ">>", face(fBig, 16*u), x, y-u, alCenter, colW, math.Round(1.5*u))
	default:
		say(dst, "$", face(fBig, 18*u), x, y-u, alCenter, colW, math.Round(1.5*u))
	}
}

// drawBuffs shows the buffs running, with their time left, in a row at y over area.
func (g *Game) drawBuffs(dst *ebiten.Image, area rect, y float64) {
	u := g.u
	var on []buffDef
	for _, b := range buffs {
		if g.buff(b.id) {
			on = append(on, b)
		}
	}
	if len(on) == 0 {
		return
	}
	gap := 4 * u
	w := math.Min(104*u, (area.w-gap*float64(len(on)-1))/float64(len(on)))
	x := area.cx() - (float64(len(on))*(w+gap)-gap)/2
	for _, b := range on {
		name := b.name
		if w < 100*u {
			name = b.short
		}
		r := rect{x, y, w, 16 * u}
		left := g.st.Buffs[b.id]
		sk := skDark
		if left < 10 && math.Mod(g.t, 0.5) < 0.25 {
			sk = skGrey // about to run out
		}
		bevel(dst, r, sk, math.Max(1, math.Round(u)))
		say(dst, name, face(fSmall, 6*u), r.x+5*u, r.cy(), alLeft, rgb(b.col), math.Round(u))
		say(dst, fmtTime(left), face(fSmall, 6*u), r.x+r.w-5*u, r.cy(), alRight, colW, math.Round(u))
		x += w + gap
	}
}

// ---- the tutorial: a few steps for new players, each pointing at a button ----

type tutStep struct {
	text   string
	target func(g *Game, L layout) rect // what it points at (zero: nothing)
	done   func(g *Game) bool           // nil: a GOT IT button
}

var tutSteps = []tutStep{
	{"TAP SPLIT. YOUR BIOTE MAKES A COPY OF ITSELF.",
		func(g *Game, L layout) rect { return L.split },
		func(g *Game) bool { return len(g.blobs) >= 2 }},
	{"KEEP SPLITTING UNTIL THE BOARD IS FULL. THE BAR SHOWS WHEN THE NEXT SPLIT IS READY.",
		func(g *Game, L layout) rect { return L.split },
		func(g *Game) bool { return len(g.blobs) >= g.capacity() || g.st.Earned > 0 }},
	{"NOW SELL THEM ALL FOR MONEY.",
		func(g *Game, L layout) rect { return L.sell },
		func(g *Game) bool { return g.st.Earned > 0 }},
	{"SPEND IT ON UPGRADES: MORE BIOTES, FASTER SPLITS AND MORE VALUE. THE TAB OPENS AND CLOSES THEM.",
		func(g *Game, L layout) rect {
			if g.st.DrawerShut {
				return L.handle
			}
			return L.cards[2]
		},
		func(g *Game) bool { return g.st.ValueLv+g.st.SpeedLv+g.st.CapLv > 0 }},
	{"EVERY SALE COUNTS TOWARDS THE CHALLENGE: CLEAR ROOMS TO CLIMB FLOORS. TAP IT TO SEE WHAT THIS ROOM NEEDS.",
		func(g *Game, L layout) rect { return L.challenge },
		func(g *Game) bool { return g.panel == "challenge" }},
	{"NEW BIOTES CAN MUTATE, AND RARE ONES ARE WORTH FAR MORE. TAP A BIOTE TO SEE ITS MUTATIONS. REBIRTH LATER FOR SKILLS AND DNA. HAVE FUN!",
		func(g *Game, L layout) rect { return rect{} }, nil},
}

func (g *Game) tutOn() bool { return g.st.Tut < len(tutSteps) && g.panel == "" && !g.ad.on }

func (g *Game) tickTut() {
	if g.st.Tut >= len(tutSteps) {
		return
	}
	if d := tutSteps[g.st.Tut].done; d != nil && d(g) {
		g.st.Tut++
		g.play("buy")
	}
}

// layTut places the tutorial box away from what it points at.
func (g *Game) layTut(L *layout) {
	if g.st.Tut >= len(tutSteps) {
		return
	}
	u := math.Max(g.u, 1)
	tg := tutSteps[g.st.Tut].target(g, *L)
	w := math.Min(320*u, L.grid.w)
	y := L.grid.y + 6*u
	lines := wrapLines(tutSteps[g.st.Tut].text, w-20*u, face(fSmall, 8*u))
	h := 26*u + float64(lines)*8*1.9*u + 26*u
	if tg.h > 0 && tg.cy() < g.H/2 {
		y = L.grid.y + L.grid.h - h - 6*u
	}
	L.tutBox = rect{L.grid.cx() - w/2, y, w, h}
	L.tutSkip = rect{L.tutBox.x + L.tutBox.w - 50*u, L.tutBox.y + L.tutBox.h - 22*u, 44 * u, 17 * u}
	L.tutOK = rect{L.tutBox.cx() - 40*u, L.tutBox.y + L.tutBox.h - 24*u, 80 * u, 20 * u}
}

// wrapLines is how many lines wrapSay needs for s in width w.
func wrapLines(s string, w float64, f *text.GoTextFace) int {
	n, line := 0, ""
	for _, word := range strings.Fields(s) {
		try := strings.TrimSpace(line + " " + word)
		if line != "" && textW(try, f) > w {
			n++
			line = word
		} else {
			line = try
		}
	}
	if line != "" {
		n++
	}
	return n
}

// tapTut handles a tap on the tutorial box; other taps go through to the game.
func (g *Game) tapTut(L layout, x, y float64) bool {
	if !g.tutOn() {
		return false
	}
	st := tutSteps[g.st.Tut]
	switch {
	case st.done == nil && L.tutOK.in(x, y):
		g.st.Tut++
	case st.done != nil && L.tutSkip.in(x, y):
		g.st.Tut = len(tutSteps)
	case L.tutBox.in(x, y):
	default:
		return false
	}
	g.play("tap")
	g.save()
	return true
}

func (g *Game) drawTut(dst *ebiten.Image, L layout) {
	if !g.tutOn() {
		return
	}
	u, p := g.u, L.p
	st := tutSteps[g.st.Tut]
	if tg := st.target(g, L); tg.w > 0 {
		// dim everything but the target, and pulse a ring round it
		dim := color.NRGBA{5, 2, 15, 120}
		fill(dst, 0, 0, g.W, tg.y, dim)
		fill(dst, 0, tg.y+tg.h, g.W, g.H-tg.y-tg.h, dim)
		fill(dst, 0, tg.y, tg.x, tg.h, dim)
		fill(dst, tg.x+tg.w, tg.y, g.W-tg.x-tg.w, tg.h, dim)
		k := math.Mod(g.t, 1.0)
		grow := 4*u + 10*u*k
		vector.StrokeRect(dst, float32(tg.x-grow), float32(tg.y-grow), float32(tg.w+2*grow), float32(tg.h+2*grow), float32(math.Max(1, 3*u*(1-k))), color.NRGBA{255, 201, 74, uint8(255 * (1 - k))}, false)
		vector.StrokeRect(dst, float32(tg.x-2*u), float32(tg.y-2*u), float32(tg.w+4*u), float32(tg.h+4*u), float32(math.Max(1, 2*u)), colT, false)
	}
	b := L.tutBox
	bevel(dst, b, skViolet, p)
	say(dst, "TUTORIAL "+itoa(g.st.Tut+1)+"/"+itoa(len(tutSteps)), face(fSmall, 6*u), b.x+10*u, b.y+10*u, alLeft, colT, math.Round(u))
	g.wrapSay(dst, st.text, b.x+10*u, b.y+26*u, b.w-20*u, 8, colW)
	if st.done == nil {
		bevel(dst, L.tutOK, skGreen, math.Max(1, p-1))
		say(dst, "GOT IT", face(fSmall, 8*u), L.tutOK.cx(), L.tutOK.cy(), alCenter, colW, math.Round(u))
	} else {
		bevel(dst, L.tutSkip, skGrey, math.Max(1, p-1))
		say(dst, "SKIP", face(fSmall, 6*u), L.tutSkip.cx(), L.tutSkip.cy(), alCenter, colW, math.Round(u))
	}
}

// ---- small animations ----

// pop is r squashed a little for a moment after it was pressed.
func (g *Game) pop(r rect) rect {
	a := g.t - g.press.t
	if a >= 0.18 || !r.in(g.press.x, g.press.y) {
		return r
	}
	return grow(r, 1-0.1*math.Sin(math.Pi*a/0.18))
}

// grow scales r by k about its centre.
func grow(r rect, k float64) rect {
	w, h := r.w*k, r.h*k
	return rect{math.Round(r.cx() - w/2), math.Round(r.cy() - h/2), math.Round(w), math.Round(h)}
}

// confetti rains n pieces down from the top of the screen.
func (g *Game) confetti(n int) {
	cols := []color.RGBA{colT, colG, colL, rgb(0xff6fb5), rgb(0x5fe8ff), colW}
	for i := 0; i < n; i++ {
		g.parts = append(g.parts, particle{x: rand.Float64() * g.W, y: -rand.Float64() * 40 * g.u, vx: (rand.Float64() - 0.5) * 3 * g.u, vy: rand.Float64() * 2 * g.u,
			life: 1.4 + rand.Float64(), size: math.Max(2, (2+rand.Float64()*2)*g.u), c: cols[i%len(cols)], r: -3, tx: rand.Float64() * 7})
	}
}

// burst throws n sparks out from (x, y).
func (g *Game) burst(x, y float64, n int, c color.RGBA) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := (1.5 + rand.Float64()*3) * g.u
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - g.u, life: 0.6 + rand.Float64()*0.4, size: math.Max(2, 2.5*g.u), c: c, r: -1})
	}
}

// shakeAt is how far the board is shaken right now.
func (g *Game) shakeAt() (dx, dy float64) {
	a := g.t - g.shakeT
	if a >= 0.45 {
		return 0, 0
	}
	amp := 5 * g.u * (1 - a/0.45)
	return math.Sin(g.t*70) * amp, math.Cos(g.t*57) * amp
}

// notice is a message in the middle of the screen.
func (g *Game) notice(s string) {
	g.floats = append(g.floats, floater{s: s, x: g.W / 2, y: g.H * 0.5, c: colW, size: 16})
}
