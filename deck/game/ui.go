package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---- frames ----

// goldEdge draws a bevelled gold border along a path's outline.
func goldEdge(dst *ebiten.Image, outer, inner *vector.Path) {
	gradPath(dst, outer, colGoldHi, colGoldLo)
	gradPath(dst, inner, colGoldLo, colGold)
}

// cornerOrn is a small gold flourish at a frame corner; dx,dy point into the frame.
func cornerOrn(dst *ebiten.Image, x, y, dx, dy, s float64) {
	pts := []float64{x, y, x + dx*18*s, y + dy*3*s, x + dx*7*s, y + dy*7*s, x + dx*3*s, y + dy*18*s}
	shape(dst, pts, colGoldHi, colGoldLo, 1)
	shape(dst, []float64{x + dx*5*s, y + dy*5*s, x + dx*9*s, y + dy*9*s, x + dx*5*s, y + dy*13*s, x + dx*1*s, y + dy*9*s}, colInk, colInk, 0)
	disc(dst, x+dx*7*s, y+dy*7*s, 2*s, 2*s, colGoldHi)
}

// setGem draws a small faceted gem in a gold setting.
func setGem(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	shape(dst, []float64{x - r*1.5, y, x, y - r*1.2, x + r*1.5, y, x, y + r*1.2}, colGoldHi, colGoldLo, 1.2)
	shape(dst, []float64{x - r, y, x, y - r*0.8, x + r, y, x, y + r*0.8}, light(c, 0.4), shade(c, 0.5), 0)
	glow(dst, x, y, r*2.5, c, 0.35)
}

// frame draws an ornate dark panel: shadow, gold bevel, textured stone fill,
// an inner gold line, corner flourishes and an optional gem on top.
func frame(dst *ebiten.Image, r rect, fill, accent color.RGBA, gem bool) {
	c := math.Min(14, math.Min(r.w, r.h)/4)
	radial(dst, r.cx(), r.cy()+8, r.w*0.62, r.h*0.62, 0, 0, alpha(colInk, 0.7), clear)
	fillPath(dst, chamfer(r.grow(4.5), c+4), colInk)
	goldEdge(dst, chamfer(r.grow(3.5), c+3), chamfer(r.grow(1.5), c+1.5))
	texPath(dst, chamfer(r, c), light(fill, 0.08), shade(fill, 0.55))
	gradPath(dst, chamfer(rect{r.x + 2, r.y + 2, r.w - 4, math.Min(r.h*0.4, 60)}, c-2), alpha(colWhite, 0.07), clear)
	if r.w > 60 && r.h > 40 {
		strokePath(dst, chamfer(r.grow(-6), c-3), 1, alpha(colGold, 0.35))
	}
	if r.w > 90 && r.h > 70 {
		k := math.Min(1, math.Min(r.w, r.h)/120)
		cornerOrn(dst, r.x-2, r.y-2, 1, 1, k)
		cornerOrn(dst, r.x+r.w+2, r.y-2, -1, 1, k)
		cornerOrn(dst, r.x-2, r.y+r.h+2, 1, -1, k)
		cornerOrn(dst, r.x+r.w+2, r.y+r.h+2, -1, -1, k)
	}
	if gem && r.w > 140 {
		setGem(dst, r.cx(), r.y-2, 6, accent)
	}
}

// panel is the standard frame, with its fill tinted toward the given colour.
func panel(dst *ebiten.Image, r rect, top, _ color.RGBA, _ float64) {
	frame(dst, r, mix(colStone, shade(top, 0.6), 0.1), colRed, true)
}

// divider is a gold rule with a diamond in the middle.
func divider(dst *ebiten.Image, cx, y, w float64) {
	gradRect(dst, cx-w/2, y-1, w/2, 2, clear, colGold)
	gradRect(dst, cx, y-1, w/2, 2, colGold, clear)
	shape(dst, []float64{cx - 7, y, cx, y - 6, cx + 7, y, cx, y + 6}, colGoldHi, colGoldLo, 1)
}

// button draws a cut-corner metal button lit in colour c.
func button(dst *ebiten.Image, r rect, c color.RGBA, label string, size float64, off bool) {
	if off {
		c = rgb(0x40, 0x3e, 0x48)
	}
	k := math.Min(12, r.h/3.5)
	shadow(dst, r.cx(), r.y+r.h+2, r.w*0.55, 10, 0.6)
	fillPath(dst, chamfer(r.grow(3.5), k+3), colInk)
	goldEdge(dst, chamfer(r.grow(2.5), k+2.5), chamfer(r.grow(1), k+1))
	texPath(dst, chamfer(r, k), shade(c, 0.95), shade(c, 0.32))
	gradPath(dst, chamfer(rect{r.x + 3, r.y + 2, r.w - 6, r.h * 0.45}, k-2), alpha(colWhite, 0.22), alpha(colWhite, 0.02))
	gradRect(dst, r.x+k, r.y+r.h-3, r.w-2*k, 2, alpha(light(c, 0.4), 0.5), alpha(light(c, 0.4), 0.5))
	if r.w > 130 && r.h > 44 {
		for _, d := range []float64{-1, 1} {
			x := r.cx() + d*(r.w/2+3)
			shape(dst, []float64{x, r.cy() - 7, x + d*9, r.cy(), x, r.cy() + 7}, colGoldHi, colGoldLo, 1)
		}
	}
	if label != "" {
		if !off {
			glow(dst, r.cx(), r.cy(), r.w*0.35, light(c, 0.3), 0.18)
		}
		tc := colBone
		if off {
			tc = colDim
		}
		say(dst, label, r.cx(), r.cy(), size, tc, 0.5)
	}
}

// pill is a dark plate for a counter (value on the right, icon goes on the left).
func pill(dst *ebiten.Image, r rect, val string) {
	k := r.h / 2.6
	fillPath(dst, chamfer(r.grow(3), k+3), colInk)
	goldEdge(dst, chamfer(r.grow(2), k+2), chamfer(r.grow(0.8), k+0.8))
	texPath(dst, chamfer(r, k), rgb(0x22, 0x20, 0x2a), rgb(0x0c, 0x0b, 0x12))
	say(dst, val, r.x+r.w-12, r.cy(), 18, colBone, 1)
}

// ---- icons ----

func coinIcon(dst *ebiten.Image, x, y, r float64) {
	disc(dst, x, y+1.5, r+2, r+2, colInk)
	radial(dst, x, y, r, r, -0.3, -0.35, rgb(0xff, 0xe8, 0x9a), rgb(0x8a, 0x5a, 0x10))
	radial(dst, x, y, r*0.72, r*0.72, 0.3, 0.35, rgb(0xe8, 0xb8, 0x50), rgb(0xa8, 0x70, 0x18))
	shape(dst, []float64{x, y - r*0.45, x + r*0.4, y, x, y + r*0.45, x - r*0.4, y}, rgb(0xff, 0xf0, 0xc0), rgb(0xb8, 0x80, 0x20), 0)
	radial(dst, x-r*0.35, y-r*0.4, r*0.28, r*0.2, 0, 0, alpha(colWhite, 0.7), clear)
}

// gemIcon is a blood-red soul crystal (the premium currency).
func gemIcon(dst *ebiten.Image, x, y, r float64) {
	glow(dst, x, y, r*2.2, colGem, 0.35)
	out := []float64{x, y - r*1.3, x + r*0.8, y - r*0.4, x + r*0.6, y + r*0.8, x, y + r*1.3, x - r*0.6, y + r*0.8, x - r*0.8, y - r*0.4}
	shape(dst, out, rgb(0xff, 0x6a, 0x90), rgb(0x6a, 0x08, 0x28), 1.6)
	shape(dst, []float64{x, y - r*1.3, x + r*0.8, y - r*0.4, x, y}, rgb(0xff, 0xb0, 0xc8), rgb(0xff, 0x60, 0x8a), 0)
	shape(dst, []float64{x, y, x - r*0.6, y + r*0.8, x, y + r*1.3}, rgb(0x8a, 0x10, 0x38), rgb(0x4a, 0x04, 0x18), 0)
	line(dst, x-r*0.2, y-r*0.8, x-r*0.45, y-r*0.2, 1.2, alpha(colWhite, 0.8))
}

func boltIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	glow(dst, x, y, r*1.8, c, 0.35)
	pts := []float64{x + 0.25*r, y - r, x - 0.6*r, y + 0.15*r, x - 0.05*r, y + 0.15*r, x - 0.3*r, y + r, x + 0.62*r, y - 0.22*r, x + 0.08*r, y - 0.22*r}
	shape(dst, pts, light(c, 0.6), shade(c, 0.6), 1.5)
}

func heartIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	p := &vector.Path{}
	p.MoveTo(float32(x), float32(y+r))
	p.CubicTo(float32(x-r*1.5), float32(y-r*0.1), float32(x-r*0.6), float32(y-r*1.25), float32(x), float32(y-r*0.4))
	p.CubicTo(float32(x+r*0.6), float32(y-r*1.25), float32(x+r*1.5), float32(y-r*0.1), float32(x), float32(y+r))
	p.Close()
	glow(dst, x, y, r*1.8, c, 0.3)
	strokePath(dst, p, 2.5, colInk)
	gradPath(dst, p, light(c, 0.3), shade(c, 0.45))
	radial(dst, x-r*0.45, y-r*0.35, r*0.22, r*0.15, 0, 0, alpha(colWhite, 0.7), clear)
}

func shieldIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	pts := []float64{x - r*0.85, y - r*0.8, x, y - r*0.95, x + r*0.85, y - r*0.8, x + r*0.78, y + r*0.1, x, y + r, x - r*0.78, y + r*0.1}
	shape(dst, pts, colGoldHi, colGoldLo, 1.6)
	in := []float64{x - r*0.66, y - r*0.62, x, y - r*0.76, x + r*0.66, y - r*0.62, x + r*0.6, y + r*0.06, x, y + r*0.78, x - r*0.6, y + r*0.06}
	shape(dst, in, light(c, 0.15), shade(c, 0.35), 0)
	line(dst, x, y-r*0.6, x, y+r*0.6, r*0.12, alpha(colGoldHi, 0.8))
}

func swordIcon(dst *ebiten.Image, x, y, r float64) {
	blade := rot([]float64{x - r*0.1, y + r*0.35, x - r*0.1, y - r*0.8, x, y - r, x + r*0.1, y - r*0.8, x + r*0.1, y + r*0.35}, x, y, math.Pi/4)
	shape(dst, blade, colWhite, rgb(0x6a, 0x76, 0x90), 1.4)
	guard := rot([]float64{x - r*0.45, y + r*0.3, x, y + r*0.42, x + r*0.45, y + r*0.3, x, y + r*0.52}, x, y, math.Pi/4)
	shape(dst, guard, colGoldHi, colGoldLo, 1.2)
	grip := rot([]float64{x - r*0.07, y + r*0.5, x + r*0.07, y + r*0.5, x + r*0.07, y + r*0.85, x - r*0.07, y + r*0.85}, x, y, math.Pi/4)
	shape(dst, grip, rgb(0x5a, 0x34, 0x22), rgb(0x2a, 0x16, 0x0c), 1.2)
}

// chestIcon: a dark wooden chest bound in iron; the royal one is black and gold.
func chestIcon(dst *ebiten.Image, x, y, s float64, royal bool) {
	wood, band, glowC := rgb(0x5a, 0x34, 0x1e), rgb(0x6a, 0x6e, 0x7a), colOrange
	if royal {
		wood, band, glowC = rgb(0x22, 0x18, 0x2e), colGold, colPurple
	}
	shadow(dst, x, y+40*s, 50*s, 10*s, 0.6)
	glow(dst, x, y-10*s, 70*s, glowC, 0.25)
	texPath(dst, chamfer(rect{x - 44*s, y - 8*s, 88 * s, 48 * s}, 4*s), light(wood, 0.15), shade(wood, 0.45))
	strokePath(dst, chamfer(rect{x - 44*s, y - 8*s, 88 * s, 48 * s}, 4*s), 2.4*s, colInk)
	lid := []float64{x - 46*s, y - 6*s, x - 42*s, y - 34*s, x - 24*s, y - 46*s, x + 24*s, y - 46*s, x + 42*s, y - 34*s, x + 46*s, y - 6*s}
	shape(dst, lid, light(wood, 0.25), shade(wood, 0.6), 1.8*s)
	for _, dx := range []float64{-30, 30} {
		shape(dst, []float64{x + dx*s - 5*s, y - 45*s, x + dx*s + 5*s, y - 45*s, x + dx*s + 5*s, y + 40*s, x + dx*s - 5*s, y + 40*s}, light(band, 0.3), shade(band, 0.5), 1.2*s)
		for _, ry := range []float64{-30, -2, 26} {
			disc(dst, x+dx*s, y+ry*s, 1.8*s, 1.8*s, light(band, 0.5))
		}
	}
	shape(dst, []float64{x - 48*s, y - 10*s, x + 48*s, y - 10*s, x + 48*s, y - 2*s, x - 48*s, y - 2*s}, light(band, 0.3), shade(band, 0.5), 1.2*s)
	shape(dst, []float64{x - 9*s, y - 14*s, x + 9*s, y - 14*s, x + 9*s, y + 6*s, x, y + 12*s, x - 9*s, y + 6*s}, colGoldHi, colGoldLo, 1.4*s)
	glow(dst, x, y-2*s, 12*s, glowC, 0.9)
	disc(dst, x, y-3*s, 2.5*s, 3.5*s, colInk)
}

// hpBar draws a framed health bar with tick marks and the numbers on it.
func hpBar(dst *ebiten.Image, x, y, w float64, hp, maxHP int, c color.RGBA) {
	r := rect{x - w/2, y, w, 20}
	fillPath(dst, chamfer(r.grow(3.5), 7), colInk)
	goldEdge(dst, chamfer(r.grow(2.5), 6), chamfer(r.grow(1), 5))
	fillPath(dst, chamfer(r, 4), rgb(0x08, 0x06, 0x0a))
	f := math.Max(0, math.Min(1, float64(hp)/float64(max(1, maxHP))))
	if f > 0 {
		fr := rect{r.x, r.y, math.Max(8, r.w*f), r.h}
		texPath(dst, chamfer(fr, 4), light(c, 0.15), shade(c, 0.4))
		gradRect(dst, fr.x+2, fr.y+2, fr.w-4, 5, alpha(colWhite, 0.35), alpha(colWhite, 0.05))
		glow(dst, fr.x+fr.w, fr.cy(), 16, c, 0.3)
	}
	for i := 1; i < 10; i++ {
		tx := r.x + r.w*float64(i)/10
		line(dst, tx, r.y+r.h-5, tx, r.y+r.h-1, 1, alpha(colInk, 0.6))
	}
	sayBody(dst, itoa(max(0, hp))+" / "+itoa(maxHP), x, y+10, 15, colBone, 0.5)
}

// ribbon is a title banner: a dark plate with gold edges, pointed ends and wings.
func ribbon(dst *ebiten.Image, cx, cy, w float64, s string, c color.RGBA) {
	for _, d := range []float64{-1, 1} {
		wing := []float64{cx + d*(w/2-6), cy - 18, cx + d*(w/2+40), cy - 30, cx + d*(w/2+26), cy - 10, cx + d*(w/2+46), cy - 2, cx + d*(w/2+24), cy + 6, cx + d*(w/2+36), cy + 22, cx + d*(w/2-6), cy + 16}
		shape(dst, wing, shade(c, 0.55), shade(c, 0.25), 1.6)
		line(dst, cx+d*(w/2), cy-14, cx+d*(w/2+34), cy-24, 1.2, alpha(colGold, 0.6))
	}
	body := []float64{cx - w/2, cy - 26, cx + w/2, cy - 26, cx + w/2 + 14, cy, cx + w/2, cy + 26, cx - w/2, cy + 26, cx - w/2 - 14, cy}
	fillPath(dst, pathOf(scaleAround(body, cx, cy, 1.04, 1.18)), colInk)
	gradPath(dst, pathOf(scaleAround(body, cx, cy, 1.025, 1.1)), colGoldHi, colGoldLo)
	texPath(dst, pathOf(body), shade(c, 0.75), shade(c, 0.28))
	gradRect(dst, cx-w/2+10, cy-24, w-20, 14, alpha(colWhite, 0.12), clear)
	say(dst, s, cx, cy-1, 32, colGoldHi, 0.5)
	setGem(dst, cx, cy-30, 6, c)
}

func scaleAround(p []float64, cx, cy, kx, ky float64) []float64 {
	o := make([]float64, len(p))
	for i := 0; i+1 < len(p); i += 2 {
		o[i], o[i+1] = cx+(p[i]-cx)*kx, cy+(p[i+1]-cy)*ky
	}
	return o
}

// runeCircle is a faint magic circle used behind important things.
func runeCircle(dst *ebiten.Image, x, y, r, t float64, c color.RGBA, a float64) {
	strokeCircle(dst, x, y, r, 1.5, alpha(c, a))
	strokeCircle(dst, x, y, r*0.82, 1, alpha(c, a*0.7))
	for i := 0; i < 12; i++ {
		ang := float64(i)*math.Pi/6 + t*0.2
		px, py := x+math.Cos(ang)*r*0.91, y+math.Sin(ang)*r*0.91
		shape(dst, []float64{px, py - 4, px + 3, py, px, py + 4, px - 3, py}, alpha(c, a), alpha(c, a), 0)
	}
	for i := 0; i < 3; i++ {
		ang := float64(i)*2*math.Pi/3 - t*0.15
		line(dst, x+math.Cos(ang)*r*0.82, y+math.Sin(ang)*r*0.82, x+math.Cos(ang+2*math.Pi/3)*r*0.82, y+math.Sin(ang+2*math.Pi/3)*r*0.82, 1, alpha(c, a*0.6))
	}
}

// Button colours: dark metals lit from inside.
var (
	btnGo     = rgb(0x2e, 0x5a, 0x34)
	btnFire   = rgb(0x9a, 0x40, 0x1a)
	btnBlood  = rgb(0x6a, 0x16, 0x20)
	btnArcane = rgb(0x24, 0x4a, 0x7a)
	btnStone  = rgb(0x3a, 0x34, 0x44)
)

// medallion is a map node: a gold-rimmed disc of dark enamel; the boss gets spikes.
func medallion(dst *ebiten.Image, x, y, r float64, base color.RGBA, spiky bool) {
	if spiky {
		for i := 0; i < 12; i++ {
			a := float64(i) * math.Pi / 6
			p := []float64{x + math.Cos(a-0.16)*r, y + math.Sin(a-0.16)*r, x + math.Cos(a)*r*1.35, y + math.Sin(a)*r*1.35, x + math.Cos(a+0.16)*r, y + math.Sin(a+0.16)*r}
			shape(dst, p, colGoldHi, colGoldLo, 1.2)
		}
	}
	disc(dst, x, y, r+3, r+3, colInk)
	radial(dst, x, y, r+1.5, r+1.5, -0.3, -0.4, colGoldHi, colGoldLo)
	radial(dst, x, y, r*0.84, r*0.84, 0.2, 0.3, colGoldLo, colGold)
	radial(dst, x, y, r*0.76, r*0.76, -0.2, -0.3, light(base, 0.25), shade(base, 0.25))
	strokeCircle(dst, x, y, r*0.66, 1, alpha(colGold, 0.35))
}
