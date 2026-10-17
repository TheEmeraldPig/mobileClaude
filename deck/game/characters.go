package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func rectI(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

// Characters are drawn into a buffer first, then stamped onto the screen with
// a scale, a fade and a white hit-flash.
type sprite struct {
	w, h float64
	s    float64
	img  *ebiten.Image
}

func newSprite(w, h int) *sprite { return &sprite{w: float64(w), h: float64(h)} }

// buf returns the cleared buffer, sized for the current screen scale.
func (s *sprite) buf() *ebiten.Image {
	if s.img == nil || s.s != S {
		if s.img != nil {
			s.img.Deallocate()
		}
		s.img, s.s = ebiten.NewImage(int(s.w*S), int(s.h*S)), S
	}
	s.img.Clear()
	return s.img
}

// stamp draws the buffer centred on (x,y) (the buffer's centre-bottom anchor is ax,ay).
func (s *sprite) stamp(dst *ebiten.Image, x, y, ax, ay, scale, sx, sy, fade, flash float64) {
	cm := colorm.ColorM{}
	cm.Scale(1, 1, 1, fade)
	if flash > 0 {
		cm.Translate(flash*fade, flash*fade, flash*fade, 0)
	}
	op := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-ax*S, -ay*S)
	op.GeoM.Scale(scale*sx, scale*sy)
	op.GeoM.Translate(x*S, y*S)
	colorm.DrawImage(dst, s.img, cm, op)
}

// ---- the hero: a little knight ----

// drawHero draws the hero with feet at (x,y). swing is 0..1 for an attack.
func drawHero(dst *ebiten.Image, x, y, t, swing float64) {
	breath := math.Sin(t*2.6) * 1.5
	shadow(dst, x, y, 42, 10, 0.4)
	// cape
	sway := math.Sin(t*2) * 4
	shape(dst, []float64{x - 20, y - 62, x + 14, y - 62, x + 26 + sway, y - 8, x - 30 + sway, y - 6}, rgb(0xe8, 0x40, 0x48), rgb(0x8a, 0x18, 0x30), 2.5)
	// boots
	orb(dst, x-12, y-8, 11, 8, rgb(0x6a, 0x40, 0x28))
	orb(dst, x+12, y-8, 11, 8, rgb(0x6a, 0x40, 0x28))
	// body
	by := y - 34 + breath*0.5
	orb(dst, x, by, 25, 25-breath*0.3, rgb(0x3d, 0x7c, 0xe8))
	roundGrad(dst, rect{x - 24, by + 4, 48, 9}, 4, rgb(0xff, 0xd8, 0x50), rgb(0xc8, 0x8a, 0x10))
	roundRect(dst, rect{x - 6, by + 3, 12, 11}, 3, colInk)
	roundGrad(dst, rect{x - 4, by + 5, 8, 7}, 2, colWhite, colSteel)
	// sword arm
	ang := -0.9 + swing*2.2
	hx, hy := x+24, by-2
	drawSword(dst, hx, hy, ang, 1)
	orb(dst, hx, hy, 8, 8, colSkin)
	// shield
	sx, sy := x-24, by+2
	orb(dst, sx, sy, 17, 19, rgb(0xa8, 0x6a, 0x38))
	disc(dst, sx, sy, 12, 14, shade(rgb(0xa8, 0x6a, 0x38), 0.8))
	orb(dst, sx, sy, 6, 6, colGold)
	// head
	hy2 := y - 82 + breath
	orb(dst, x+2, hy2, 27, 25, colSkin)
	// helmet
	p := &vector.Path{}
	p.Arc(float32(x+2), float32(hy2-2), 30, math.Pi*1.02, math.Pi*1.98, vector.Clockwise)
	p.Close()
	strokePath(dst, p, 5, colInk)
	gradPath(dst, p, rgb(0xf0, 0xf6, 0xff), rgb(0x84, 0x94, 0xb4))
	roundGrad(dst, rect{x - 1, hy2 - 30, 6, 30}, 3, colWhite, colSteel)
	radial(dst, x-10, hy2-18, 8, 5, 0, 0, alpha(colWhite, 0.8), clear)
	// plume
	pw := math.Sin(t*3) * 3
	shape(dst, []float64{x, hy2 - 30, x - 6 + pw, hy2 - 54, x + 8 + pw, hy2 - 62, x + 22 + pw, hy2 - 50, x + 8, hy2 - 30}, rgb(0xff, 0x6a, 0x6a), rgb(0xc8, 0x20, 0x30), 2.5)
	// face
	for _, ex := range []float64{x + 6, x + 20} {
		disc(dst, ex, hy2+6, 4.6, 5.6, colInk)
		circle(dst, ex+1.2, hy2+4, 1.8, colWhite)
	}
	disc(dst, x-4, hy2+14, 5, 3, alpha(rgb(0xff, 0x80, 0x90), 0.6))
	curve(dst, x+9, hy2+15, x+13, hy2+18, x+17, hy2+15, 2, colInk)
}

func drawSword(dst *ebiten.Image, hx, hy, ang, s float64) {
	blade := rot([]float64{hx - 4*s, hy - 8*s, hx - 4*s, hy - 52*s, hx, hy - 60*s, hx + 4*s, hy - 52*s, hx + 4*s, hy - 8*s}, hx, hy, ang)
	shape(dst, blade, colWhite, rgb(0x9a, 0xac, 0xc8), 2.2)
	ridge := rot([]float64{hx, hy - 10*s, hx, hy - 54*s}, hx, hy, ang)
	line(dst, ridge[0], ridge[1], ridge[2], ridge[3], 1.2, alpha(rgb(0x6a, 0x7a, 0x98), 0.8))
	guard := rot([]float64{hx - 12*s, hy - 10*s, hx + 12*s, hy - 10*s, hx + 12*s, hy - 4*s, hx - 12*s, hy - 4*s}, hx, hy, ang)
	shape(dst, guard, colYellow, colOrange, 2)
}

// ---- enemies ----

type epal struct{ main, dark, accent color.RGBA }

func pal(c color.RGBA, accent color.RGBA, h float64) epal {
	c, accent = tint(c, h), tint(accent, h)
	return epal{c, shade(c, 0.6), accent}
}

func eyes(dst *ebiten.Image, cx, cy, r, gap float64, iris color.RGBA, angry bool, look float64) {
	for _, d := range []float64{-1, 1} {
		ex := cx + d*gap
		disc(dst, ex, cy, r+2.2, r*1.1+2.2, colInk)
		radial(dst, ex, cy, r, r*1.1, -0.2, -0.3, colWhite, rgb(0xd8, 0xdc, 0xe8))
		radial(dst, ex+look*r*0.3, cy+r*0.15, r*0.58, r*0.62, -0.2, -0.3, light(iris, 0.3), shade(iris, 0.5))
		disc(dst, ex+look*r*0.3, cy+r*0.15, r*0.28, r*0.32, colInk)
		circle(dst, ex+look*r*0.3-r*0.2, cy-r*0.1, r*0.17, colWhite)
		if angry {
			shape(dst, []float64{ex - d*r*1.3, cy - r*1.0, ex + d*r*1.2, cy - r*1.5, ex + d*r*1.3, cy - r*0.95, ex - d*r*1.2, cy - r*0.55}, colInk, colInk, 0)
		}
	}
}

func teeth(dst *ebiten.Image, cx, cy, w, h float64, n int) {
	roundRect(dst, rect{cx - w/2 - 2, cy - 2, w + 4, h + 4}, h/2, colInk)
	roundGrad(dst, rect{cx - w/2, cy, w, h}, h/3, rgb(0x6a, 0x20, 0x30), rgb(0x3a, 0x10, 0x18))
	tw := w / float64(n)
	for i := 0; i < n; i++ {
		tx := cx - w/2 + tw*float64(i)
		shape(dst, []float64{tx + 1, cy, tx + tw - 1, cy, tx + tw/2, cy + h*0.6}, colWhite, rgb(0xd8, 0xd0, 0xc0), 0)
	}
}

// drawEnemy draws an enemy with its feet at (x,y).
func drawEnemy(dst *ebiten.Image, shapeID int, x, y, t, h float64) {
	switch shapeID {
	case shSlime:
		p := pal(rgb(0x6a, 0xe0, 0x50), rgb(0xc8, 0xff, 0x80), h)
		sq := 1 + math.Sin(t*4)*0.07
		shadow(dst, x, y, 56, 12, 0.4)
		rx, ry := 54*sq, 44/sq
		cy := y - ry + 4
		p2 := &vector.Path{}
		p2.MoveTo(float32(x-rx), float32(y-6))
		p2.CubicTo(float32(x-rx), float32(cy-ry*0.9), float32(x+rx), float32(cy-ry*0.9), float32(x+rx), float32(y-6))
		p2.QuadTo(float32(x), float32(y+6), float32(x-rx), float32(y-6))
		p2.Close()
		strokePath(dst, p2, 5, colInk)
		gradPath(dst, p2, light(p.main, 0.3), shade(p.main, 0.7))
		radial(dst, x, cy+4, rx*0.8, ry*0.7, -0.3, -0.5, alpha(p.accent, 0.5), clear)
		for i, b := range [][3]float64{{-24, 8, 6}, {18, 18, 4}, {28, -2, 3}} {
			by := cy + b[1] - math.Mod(t*10+float64(i)*7, 20)
			disc(dst, x+b[0], by, b[2], b[2], alpha(colWhite, 0.35))
		}
		radial(dst, x-rx*0.45, cy-ry*0.45, rx*0.3, ry*0.18, 0, 0, alpha(colWhite, 0.85), clear)
		eyes(dst, x+4, cy+2, 9, 15, rgb(0x30, 0x60, 0x20), true, 0.4)
		curve(dst, x-4, cy+22, x+6, cy+18, x+16, cy+22, 3, colInk)
	case shGoblin:
		p := pal(rgb(0x8c, 0xc8, 0x3c), rgb(0xff, 0xd0, 0x40), h)
		bob := math.Sin(t*3) * 2
		shadow(dst, x, y, 40, 10, 0.4)
		orb(dst, x-10, y-7, 10, 7, rgb(0x5a, 0x3a, 0x22))
		orb(dst, x+12, y-7, 10, 7, rgb(0x5a, 0x3a, 0x22))
		orb(dst, x, y-34+bob, 26, 24, rgb(0x8a, 0x5a, 0x34))
		roundGrad(dst, rect{x - 24, y - 30 + bob, 48, 7}, 3, rgb(0x5a, 0x3a, 0x22), rgb(0x3a, 0x22, 0x10))
		// dagger
		dx, dy := x+30, y-36+bob
		blade := rot([]float64{dx - 3, dy, dx - 3, dy - 30, dx, dy - 36, dx + 3, dy - 30, dx + 3, dy}, dx, dy, 0.5)
		shape(dst, blade, colWhite, colSteel, 2)
		orb(dst, dx, dy, 7, 7, p.main)
		hy := y - 80 + bob
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*20, hy - 8, x + d*62, hy - 26, x + d*24, hy + 10}, light(p.main, 0.1), p.dark, 2.5)
			shape(dst, []float64{x + d*26, hy - 4, x + d*52, hy - 18, x + d*28, hy + 4}, rgb(0xff, 0xa0, 0xa0), rgb(0xc8, 0x60, 0x70), 0)
		}
		orb(dst, x, hy, 32, 28, p.main)
		eyes(dst, x+2, hy-4, 8, 13, p.accent, true, 0.3)
		orb(dst, x+4, hy+8, 7, 6, shade(p.main, 0.85))
		teeth(dst, x+3, hy+15, 22, 7, 4)
	case shBat:
		p := pal(rgb(0x9b, 0x6b, 0xe0), rgb(0xff, 0x40, 0x50), h)
		hover := math.Sin(t*3) * 8
		cy := y - 80 + hover
		shadow(dst, x, y, 34*(1-hover/40), 8, 0.3)
		flap := math.Sin(t*14) * 16
		for _, d := range []float64{-1, 1} {
			wing := []float64{x + d*16, cy - 6, x + d*80, cy - 34 - flap, x + d*70, cy + 4 - flap*0.3, x + d*54, cy - 2, x + d*44, cy + 14, x + d*30, cy + 6, x + d*16, cy + 16}
			shape(dst, wing, light(p.dark, 0.2), shade(p.dark, 0.6), 2.5)
			line(dst, x+d*16, cy-4, x+d*80, cy-34-flap, 2, alpha(colInk, 0.6))
			line(dst, x+d*40, cy-16-flap*0.5, x+d*54, cy-2, 1.6, alpha(colInk, 0.5))
		}
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*8, cy - 18, x + d*24, cy - 40, x + d*24, cy - 12}, p.main, p.dark, 2)
		}
		orb(dst, x, cy, 27, 25, p.main)
		radial(dst, x, cy+10, 14, 10, 0, 0, alpha(light(p.main, 0.4), 0.6), clear)
		glow(dst, x-9, cy-4, 14, p.accent, 0.5)
		glow(dst, x+9, cy-4, 14, p.accent, 0.5)
		eyes(dst, x, cy-4, 6.5, 9, p.accent, true, 0)
		shape(dst, []float64{x - 7, cy + 9, x - 3, cy + 9, x - 5, cy + 16}, colWhite, colCream, 1)
		shape(dst, []float64{x + 3, cy + 9, x + 7, cy + 9, x + 5, cy + 16}, colWhite, colCream, 1)
	case shShroom:
		p := pal(rgb(0xe8, 0x40, 0x40), rgb(0xff, 0xf0, 0xd8), h)
		sq := 1 + math.Sin(t*3)*0.04
		shadow(dst, x, y, 44, 10, 0.4)
		orb(dst, x, y-26, 26*sq, 28/sq, rgb(0xff, 0xec, 0xd0))
		eyes(dst, x+2, y-30, 7, 11, rgb(0x60, 0x30, 0x20), true, 0.3)
		curve(dst, x-6, y-12, x+3, y-17, x+12, y-12, 3, colInk)
		cy := y - 66
		p2 := &vector.Path{}
		p2.MoveTo(float32(x-58*sq), float32(cy+16))
		p2.CubicTo(float32(x-60*sq), float32(cy-50), float32(x+60*sq), float32(cy-50), float32(x+58*sq), float32(cy+16))
		p2.QuadTo(float32(x), float32(cy+28), float32(x-58*sq), float32(cy+16))
		p2.Close()
		strokePath(dst, p2, 5, colInk)
		gradPath(dst, p2, light(p.main, 0.25), shade(p.main, 0.65))
		for _, sp := range [][3]float64{{-30, -4, 9}, {6, -22, 11}, {32, -2, 8}, {-6, 4, 6}} {
			radial(dst, x+sp[0], cy+sp[1], sp[2], sp[2]*0.8, -0.3, -0.3, colWhite, p.accent)
		}
		radial(dst, x-24, cy-18, 16, 8, 0, 0, alpha(colWhite, 0.5), clear)
	case shSkeleton:
		bone := tint(rgb(0xf2, 0xec, 0xd8), h*0.2)
		bob := math.Sin(t*2.5) * 2
		shadow(dst, x, y, 40, 10, 0.4)
		// legs and spine
		line(dst, x-10, y-4, x-8, y-34, 6, colInk)
		line(dst, x+10, y-4, x+8, y-34, 6, colInk)
		line(dst, x-10, y-4, x-8, y-34, 3, bone)
		line(dst, x+10, y-4, x+8, y-34, 3, bone)
		line(dst, x, y-34+bob, x, y-74+bob, 7, colInk)
		line(dst, x, y-34+bob, x, y-74+bob, 4, bone)
		for i := 0; i < 3; i++ {
			ry := y - 66 + float64(i)*9 + bob
			curve(dst, x-18, ry+4, x, ry-6, x+18, ry+4, 6, colInk)
			curve(dst, x-18, ry+4, x, ry-6, x+18, ry+4, 3, bone)
		}
		// club
		cx, cyy := x+32, y-50+bob
		line(dst, cx, cyy, cx+14, cyy-38, 9, colInk)
		line(dst, cx, cyy, cx+14, cyy-38, 5, bone)
		orb(dst, cx+15, cyy-42, 9, 8, bone)
		orb(dst, cx, cyy, 6, 6, bone)
		hy := y - 100 + bob
		orb(dst, x, hy, 28, 26, bone)
		roundRect(dst, rect{x - 16, hy + 12, 32, 16}, 6, colInk)
		roundGrad(dst, rect{x - 14, hy + 13, 28, 13}, 5, bone, shade(bone, 0.75))
		for i := 0; i < 4; i++ {
			line(dst, x-9+float64(i)*6, hy+14, x-9+float64(i)*6, hy+25, 1.5, alpha(colInk, 0.7))
		}
		for _, d := range []float64{-1, 1} {
			disc(dst, x+d*10, hy-2, 8, 9, colInk)
			glow(dst, x+d*10, hy-1, 9, colRed, 0.8)
			circle(dst, x+d*10, hy-1, 2.5, rgb(0xff, 0x80, 0x60))
		}
		shape(dst, []float64{x - 3, hy + 9, x + 3, hy + 9, x, hy + 4}, colInk, colInk, 0)
	case shOrc:
		p := pal(rgb(0xc0, 0x6a, 0x48), rgb(0xff, 0xf0, 0xd0), h)
		bob := math.Sin(t*2) * 2
		shadow(dst, x, y, 62, 14, 0.45)
		orb(dst, x-20, y-10, 15, 11, rgb(0x4a, 0x32, 0x22))
		orb(dst, x+20, y-10, 15, 11, rgb(0x4a, 0x32, 0x22))
		orb(dst, x, y-56+bob, 48, 44, p.main)
		radial(dst, x+4, y-46+bob, 28, 22, -0.2, -0.3, alpha(light(p.main, 0.3), 0.5), clear)
		roundGrad(dst, rect{x - 46, y - 40 + bob, 92, 12}, 5, rgb(0x6a, 0x48, 0x2a), rgb(0x3a, 0x26, 0x14))
		orb(dst, x, y-34+bob, 9, 8, colGold)
		// club
		cx, cy := x+50, y-60+bob
		shape(dst, rot([]float64{cx - 6, cy + 20, cx - 12, cy - 50, cx + 12, cy - 50, cx + 6, cy + 20}, cx, cy, 0.35), rgb(0xa8, 0x6a, 0x38), rgb(0x5a, 0x34, 0x18), 2.5)
		for i := 0; i < 4; i++ {
			sx, sy := cx+12+float64(i)*5, cy-40+float64(i)*10
			shape(dst, []float64{sx, sy - 4, sx + 12, sy, sx, sy + 4}, colSteel, shade(colSteel, 0.6), 1.5)
		}
		orb(dst, cx-4, cy+14, 11, 11, p.main)
		hy := y - 114 + bob
		orb(dst, x, hy, 36, 32, p.main)
		// helmet with horns
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*26, hy - 22, x + d*58, hy - 54, x + d*46, hy - 18}, colCream, rgb(0xc8, 0xb0, 0x88), 2.5)
		}
		hp := &vector.Path{}
		hp.Arc(float32(x), float32(hy-6), 38, math.Pi*1.05, math.Pi*1.95, vector.Clockwise)
		hp.Close()
		strokePath(dst, hp, 5, colInk)
		gradPath(dst, hp, rgb(0xb8, 0xc0, 0xd0), rgb(0x5a, 0x62, 0x78))
		eyes(dst, x+2, hy+2, 7.5, 13, rgb(0xff, 0xd0, 0x30), true, 0.2)
		teeth(dst, x+2, hy+16, 26, 8, 4)
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*9, hy + 22, x + d*15, hy + 22, x + d*12, hy + 6}, colWhite, colCream, 2)
		}
	case shGolem:
		stone := tint(rgb(0x9a, 0xa0, 0xb0), h*0.3)
		rune := tint(rgb(0x50, 0xe8, 0xff), h)
		bob := math.Sin(t*1.8) * 2
		shadow(dst, x, y, 64, 14, 0.45)
		block := func(r rect, rad float64) {
			roundRect(dst, r.grow(3), rad+3, colInk)
			roundGrad(dst, r, rad, light(stone, 0.2), shade(stone, 0.6))
			roundGrad(dst, rect{r.x + 4, r.y + 3, r.w - 8, r.h * 0.3}, rad, alpha(colWhite, 0.25), clear)
		}
		block(rect{x - 34, y - 26, 26, 24}, 6)
		block(rect{x + 8, y - 26, 26, 24}, 6)
		block(rect{x - 46, y - 100 + bob, 92, 76}, 14)
		line(dst, x-20, y-80+bob, x-4, y-60+bob, 3, alpha(rune, 0.9))
		line(dst, x-4, y-60+bob, x+14, y-76+bob, 3, alpha(rune, 0.9))
		glow(dst, x-4, y-62+bob, 30, rune, 0.35)
		for _, d := range []float64{-1, 1} {
			block(rect{x + d*58 - 16, y - 94 + bob, 32, 36}, 8)
			block(rect{x + d*62 - 18, y - 56 + bob, 36, 32}, 10)
		}
		block(rect{x - 30, y - 150 + bob, 60, 50}, 12)
		for _, d := range []float64{-1, 1} {
			roundRect(dst, rect{x + d*12 - 7, y - 130 + bob, 14, 8}, 3, colInk)
			glow(dst, x+d*12, y-126+bob, 16, rune, 0.9)
			roundRect(dst, rect{x + d*12 - 5, y - 128 + bob, 10, 4}, 2, light(rune, 0.6))
		}
	case shKing:
		p := pal(rgb(0x5a, 0xa8, 0x90), rgb(0xff, 0xd8, 0x40), h)
		bob := math.Sin(t*1.6) * 3
		shadow(dst, x, y, 76, 16, 0.5)
		// cape
		shape(dst, []float64{x - 44, y - 120 + bob, x + 44, y - 120 + bob, x + 74, y - 6, x - 74, y - 6}, rgb(0xa0, 0x30, 0xc8), rgb(0x4a, 0x10, 0x60), 3)
		orb(dst, x-24, y-12, 17, 12, shade(p.main, 0.7))
		orb(dst, x+24, y-12, 17, 12, shade(p.main, 0.7))
		orb(dst, x, y-64+bob, 56, 52, p.main)
		radial(dst, x+4, y-50+bob, 34, 28, 0, -0.2, alpha(light(p.main, 0.35), 0.6), clear)
		// club
		cx, cy := x+60, y-70+bob
		shape(dst, rot([]float64{cx - 7, cy + 26, cx - 16, cy - 60, cx + 16, cy - 60, cx + 7, cy + 26}, cx, cy, 0.3), rgb(0x8a, 0x5a, 0x34), rgb(0x4a, 0x2a, 0x14), 3)
		orb(dst, cx-6, cy+18, 13, 13, p.main)
		hy := y - 132 + bob
		orb(dst, x, hy, 42, 36, p.main)
		eyes(dst, x+2, hy-4, 9, 16, rgb(0xff, 0x50, 0x30), true, 0.2)
		orb(dst, x+4, hy+12, 11, 9, shade(p.main, 0.85))
		teeth(dst, x+3, hy+22, 30, 8, 5)
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*12, hy + 30, x + d*19, hy + 30, x + d*16, hy + 12}, colWhite, colCream, 2)
		}
		// crown
		cr := []float64{x - 34, hy - 26, x - 38, hy - 66, x - 18, hy - 46, x, hy - 74, x + 18, hy - 46, x + 38, hy - 66, x + 34, hy - 26}
		shape(dst, cr, rgb(0xff, 0xf0, 0x80), rgb(0xd8, 0x8a, 0x10), 3)
		for i, gx := range []float64{-20, 0, 20} {
			c := []color.RGBA{colRed, colBlue, colRed}[i]
			orb(dst, x+gx, hy-34, 5, 5, c)
		}
		glow(dst, x, hy-50, 40, colYellow, 0.25+0.1*math.Sin(t*3))
	}
}
