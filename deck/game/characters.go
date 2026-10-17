package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
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

// stamp draws the buffer so its anchor (ax,ay) lands on (x,y).
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

func glowEye(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	glow(dst, x, y, r*4, c, 0.7)
	disc(dst, x, y, r*1.3, r*0.8, light(c, 0.3))
	disc(dst, x, y, r*0.6, r*0.4, colWhite)
}

var (
	steelHi = rgb(0x8a, 0x92, 0xa8)
	steelLo = rgb(0x16, 0x18, 0x22)
)

// ---- the hero: a horned dark knight ----

// drawHero draws the hero with feet at (x,y), facing right. swing is 0..1 for an attack.
func drawHero(dst *ebiten.Image, x, y, t, swing float64) {
	br := math.Sin(t*2.2) * 1.5
	sway := math.Sin(t*1.7) * 5
	shadow(dst, x, y, 52, 11, 0.6)
	// cape, torn at the hem
	cape := []float64{x - 16, y - 126 + br, x + 8, y - 124 + br, x + 6 + sway*0.3, y - 60, x - 4 + sway, y - 4, x - 14 + sway, y - 16, x - 22 + sway, y + 2, x - 32 + sway, y - 12, x - 42 + sway, y - 2, x - 46 + sway*0.8, y - 40, x - 34, y - 90}
	blob(dst, cape, rgb(0x7a, 0x14, 0x24), rgb(0x24, 0x04, 0x0a), 1.6)
	curve(dst, x-20, y-110, x-34+sway*0.5, y-60, x-30+sway, y-14, 1.2, alpha(rgb(0xc0, 0x30, 0x40), 0.5))
	// legs
	leg := func(hx, fx float64, c1, c2 color.RGBA) {
		shape(dst, []float64{hx - 9, y - 70, hx + 9, y - 70, fx + 7, y - 30, fx + 6, y - 8, fx - 6, y - 8, fx - 8, y - 30}, c1, c2, 1.4)
		shape(dst, []float64{fx - 8, y - 38, fx + 8, y - 40, fx + 6, y - 28, fx - 6, y - 26}, light(c1, 0.2), c2, 1.2)
		shape(dst, []float64{fx - 9, y - 10, fx + 8, y - 10, fx + 20, y - 2, fx + 18, y + 2, fx - 10, y + 2}, c1, c2, 1.4)
	}
	leg(x-8, x-16, shade(steelHi, 0.6), steelLo)
	leg(x+8, x+14, steelHi, steelLo)
	// tassets
	for i, dx := range []float64{-18, -6, 6} {
		shape(dst, []float64{x + dx, y - 74, x + dx + 13, y - 74, x + dx + 12, y - 52 + float64(i%2)*4, x + dx + 1, y - 54}, steelHi, steelLo, 1.2)
	}
	// breastplate with an ember sigil
	by := y - 122 + br
	chest := []float64{x - 24, by, x + 24, by, x + 21, by + 40, x + 12, by + 54, x - 12, by + 54, x - 21, by + 40}
	shape(dst, chest, rgb(0x6a, 0x72, 0x88), steelLo, 1.6)
	line(dst, x, by+4, x, by+50, 1.4, alpha(steelHi, 0.8))
	strokePath(dst, pathOf(scaleAround(chest, x, by+27, 0.82, 0.86)), 1.2, alpha(colGold, 0.7))
	glow(dst, x, by+22, 12, colOrange, 0.5)
	shape(dst, []float64{x - 5, by + 16, x, by + 10, x + 5, by + 16, x, by + 28}, rgb(0xff, 0xb0, 0x50), rgb(0xc0, 0x40, 0x10), 1)
	// helmet with horns
	hy := by - 4
	for _, d := range []float64{-1, 1} {
		horn := []float64{x + d*12, hy - 34, x + d*28, hy - 46, x + d*36, hy - 66, x + d*30, hy - 76, x + d*30, hy - 60, x + d*20, hy - 48, x + d*12, hy - 44}
		blob(dst, horn, rgb(0xd8, 0xcc, 0xb0), rgb(0x3a, 0x30, 0x28), 1.4)
	}
	helm := []float64{x - 17, hy, x + 17, hy, x + 20, hy - 22, x + 16, hy - 40, x + 6, hy - 50, x - 6, hy - 50, x - 16, hy - 40, x - 20, hy - 22}
	shape(dst, helm, rgb(0x7a, 0x82, 0x98), steelLo, 1.6)
	line(dst, x, hy-48, x, hy-30, 1.2, alpha(colWhite, 0.4))
	// glowing visor
	glow(dst, x+4, hy-24, 26, colOrange, 0.5)
	shape(dst, []float64{x - 12, hy - 27, x + 18, hy - 27, x + 18, hy - 21, x - 12, hy - 21}, rgb(0xff, 0xd0, 0x80), rgb(0xff, 0x60, 0x10), 0)
	shape(dst, []float64{x + 2, hy - 21, x + 7, hy - 21, x + 7, hy - 6, x + 2, hy - 6}, rgb(0xff, 0xa0, 0x40), rgb(0xff, 0x50, 0x10), 0)
	// pauldrons
	for _, d := range []float64{-1, 1} {
		px := x + d*24
		p := []float64{px - 14, by + 14, px - 16, by - 2, px - 6, by - 12, px + 10, by - 10, px + 16, by + 2, px + 12, by + 16}
		if d < 0 {
			p = mirror(p, px)
		}
		shape(dst, p, rgb(0x7a, 0x82, 0x98), steelLo, 1.4)
		shape(dst, []float64{px - 4 + d*4, by - 10, px + d*14, by - 26, px + 6 + d*4, by - 8}, steelHi, steelLo, 1.2)
	}
	// off arm hanging at the side
	blob(dst, []float64{x - 30, by + 6, x - 22, by + 4, x - 24, by + 40, x - 28, by + 58, x - 36, by + 56, x - 34, by + 36}, shade(steelHi, 0.7), steelLo, 1.4)
	orb(dst, x-32, by+60, 7, 7, rgb(0x3a, 0x3e, 0x4c))
	// sword arm and greatsword, held out and up
	ang := 0.42 - swing*0.2 + swing*1.4
	hx, hyy := x+40, by+40
	blob(dst, []float64{x + 22, by + 4, x + 32, by + 2, x + 40, by + 22, x + 46, by + 38, x + 36, by + 44, x + 28, by + 26}, steelHi, steelLo, 1.4)
	drawSword(dst, hx, hyy, ang, 1.15)
	orb(dst, hx, hyy, 7.5, 7, rgb(0x4a, 0x4e, 0x5e))
}

// drawSword draws a long blade held at (hx,hy) pointing up, rotated by ang, with glowing runes.
func drawSword(dst *ebiten.Image, hx, hy, ang, s float64) {
	R := func(p []float64) []float64 { return rot(p, hx, hy, ang) }
	tip := R([]float64{hx, hy - 60*s})
	glow(dst, tip[0], tip[1], 40*s, colOrange, 0.25)
	shape(dst, R([]float64{hx - 5*s, hy - 10*s, hx - 5*s, hy - 100*s, hx, hy - 112*s, hx + 5*s, hy - 100*s, hx + 5*s, hy - 10*s}), rgb(0xd0, 0xd8, 0xe8), rgb(0x3a, 0x40, 0x50), 1.4)
	r := R([]float64{hx, hy - 14*s, hx, hy - 100*s})
	line(dst, r[0], r[1], r[2], r[3], 1.6*s, alpha(colOrange, 0.9))
	shape(dst, R([]float64{hx - 18*s, hy - 12*s, hx - 22*s, hy - 18*s, hx, hy - 14*s, hx + 22*s, hy - 18*s, hx + 18*s, hy - 12*s, hx, hy - 6*s}), colGoldHi, colGoldLo, 1.2)
	shape(dst, R([]float64{hx - 3*s, hy - 6*s, hx + 3*s, hy - 6*s, hx + 3*s, hy + 14*s, hx - 3*s, hy + 14*s}), rgb(0x4a, 0x2a, 0x1a), rgb(0x1a, 0x0a, 0x04), 1)
	pm := R([]float64{hx, hy + 17*s})
	orb(dst, pm[0], pm[1], 4*s, 4*s, colGold)
}

// ---- enemies (they face left, toward the hero) ----

func claws(dst *ebiten.Image, x, y, s float64, c color.RGBA) {
	for i := 0; i < 3; i++ {
		a := -0.5 + float64(i)*0.45
		shape(dst, []float64{x, y - 3*s, x - math.Cos(a)*18*s, y + math.Sin(a)*18*s + 6*s, x + 2*s, y + 3*s}, light(c, 0.4), shade(c, 0.4), 1)
	}
}

func fangs(dst *ebiten.Image, x, y float64, n int, gap float64) {
	for i := 0; i < n; i++ {
		tx := x + float64(i)*gap
		shape(dst, []float64{tx, y, tx + gap*0.7, y, tx + gap*0.35, y + gap*1.3}, colBone, colCream, 0)
	}
}

// drawEnemy draws an enemy with its feet at (x,y).
func drawEnemy(dst *ebiten.Image, shapeID int, x, y, t, h float64) {
	switch shapeID {
	case shGhoul:
		skin := tint(rgb(0x7a, 0x8a, 0x70), h*0.5)
		hunch := math.Sin(t*2.4) * 3
		shadow(dst, x, y, 60, 12, 0.6)
		blob(dst, []float64{x + 10, y - 86 + hunch, x - 30, y - 56, x - 46, y - 28, x - 40, y - 24, x - 22, y - 50, x + 16, y - 74}, shade(skin, 0.6), shade(skin, 0.25), 1.4)
		claws(dst, x-44, y-26, 0.8, rgb(0xc8, 0xc0, 0xa8))
		blob(dst, []float64{x + 2, y - 50, x + 14, y - 52, x + 10, y - 20, x + 20, y + 1, x + 2, y + 1, x, y - 22}, shade(skin, 0.7), shade(skin, 0.3), 1.4)
		blob(dst, []float64{x + 18, y - 50, x + 30, y - 46, x + 34, y - 20, x + 42, y + 1, x + 26, y + 1, x + 22, y - 24}, shade(skin, 0.8), shade(skin, 0.3), 1.4)
		shape(dst, []float64{x - 2, y - 58, x + 34, y - 56, x + 30, y - 34, x + 22, y - 42, x + 14, y - 30, x + 6, y - 40, x - 2, y - 32}, rgb(0x4a, 0x3a, 0x30), rgb(0x1e, 0x16, 0x10), 1.2)
		blob(dst, []float64{x - 4, y - 56, x + 34, y - 60, x + 40, y - 92 + hunch, x + 20, y - 118 + hunch, x - 12, y - 116 + hunch, x - 24, y - 96 + hunch}, light(skin, 0.1), shade(skin, 0.35), 1.6)
		for i := 0; i < 4; i++ {
			sx, sy := x+26-float64(i)*12, y-114+hunch+math.Abs(float64(i)-1.5)*4
			shape(dst, []float64{sx - 4, sy + 2, sx, sy - 9, sx + 4, sy + 2}, light(skin, 0.3), shade(skin, 0.4), 1)
		}
		for i := 0; i < 3; i++ {
			curve(dst, x-6, y-96+hunch+float64(i)*9, x+8, y-88+hunch+float64(i)*9, x+22, y-92+hunch+float64(i)*9, 1.4, alpha(colInk, 0.5))
		}
		hx, hy := x-30, y-108+hunch
		blob(dst, []float64{hx - 20, hy - 4, hx - 8, hy - 22, hx + 14, hy - 20, hx + 20, hy - 2, hx + 10, hy + 14, hx - 14, hy + 16}, light(skin, 0.15), shade(skin, 0.4), 1.6)
		shape(dst, []float64{hx - 22, hy + 4, hx + 6, hy + 8, hx + 2, hy + 22, hx - 18, hy + 20}, shade(skin, 0.6), shade(skin, 0.3), 1.4)
		fangs(dst, hx-18, hy+6, 4, 6)
		glowEye(dst, hx-10, hy-6, 3.5, rgb(0xd0, 0xff, 0x60))
		glowEye(dst, hx+4, hy-6, 3, rgb(0xd0, 0xff, 0x60))
		blob(dst, []float64{x - 8, y - 96 + hunch, x - 40, y - 70, x - 58, y - 46, x - 50, y - 42, x - 30, y - 64, x, y - 84}, skin, shade(skin, 0.35), 1.4)
		claws(dst, x-56, y-44, 1, rgb(0xe0, 0xd8, 0xc0))
	case shWolf:
		fur := tint(rgb(0x3a, 0x38, 0x48), h*0.3)
		pant := math.Sin(t*5) * 2
		shadow(dst, x, y, 86, 13, 0.6)
		glow(dst, x, y-50, 90, rgb(0x60, 0x40, 0x90), 0.15)
		leg := func(lx, ly float64, c color.RGBA) {
			blob(dst, []float64{lx - 9, ly, lx + 9, ly, lx + 6, y - 20, lx + 4, y + 1, lx - 14, y + 1, lx - 6, y - 22}, c, shade(c, 0.3), 1.4)
		}
		leg(x+44, y-50, shade(fur, 0.7))
		leg(x-24, y-54, shade(fur, 0.7))
		body := []float64{x - 46, y - 52}
		for i := 0; i <= 10; i++ {
			f := float64(i) / 10
			by := y - 92 - math.Sin(f*math.Pi)*10
			if i%2 == 1 {
				by -= 12
			}
			body = append(body, x-46+f*112, by)
		}
		body = append(body, x+72, y-70, x+80, y-86, x+70, y-58, x+50, y-44, x-30, y-42)
		shape(dst, body, light(fur, 0.1), shade(fur, 0.3), 1.6)
		leg(x+28, y-46, fur)
		leg(x-36, y-48, fur)
		hx, hy := x-56, y-86+pant
		shape(dst, []float64{hx + 30, hy - 18, hx + 10, hy - 22, hx - 4, hy - 34, hx, hy - 16, hx - 30, hy - 8, hx - 40, hy + 2, hx - 10, hy + 6, hx + 24, hy + 14}, light(fur, 0.15), shade(fur, 0.3), 1.6)
		shape(dst, []float64{hx - 36, hy + 6, hx - 6, hy + 8 + pant, hx + 18, hy + 18, hx - 4, hy + 20 + pant}, shade(fur, 0.6), shade(fur, 0.3), 1.4)
		fangs(dst, hx-32, hy+4, 4, 7)
		shape(dst, []float64{hx + 2, hy - 24, hx + 12, hy - 44, hx + 18, hy - 22}, light(fur, 0.2), shade(fur, 0.4), 1.2)
		glowEye(dst, hx-6, hy-10, 3.5, colRed)
	case shWraith:
		spirit := tint(rgb(0x50, 0xd0, 0xd0), h)
		fl := math.Sin(t*1.8) * 8
		cy := y - 60 + fl
		shadow(dst, x, y, 50*(1-fl/40), 9, 0.4)
		glow(dst, x, cy-40, 110, spirit, 0.25)
		robe := []float64{x - 26, cy - 90, x + 26, cy - 90, x + 40, cy - 30}
		for i := 0; i <= 6; i++ {
			f := float64(i) / 6
			ry := cy + 30 + math.Sin(t*4+f*9)*8
			if i%2 == 1 {
				ry -= 22
			}
			robe = append(robe, x+44-f*88, ry)
		}
		robe = append(robe, x-40, cy-30)
		shape(dst, robe, rgb(0x2a, 0x2e, 0x40), alpha(rgb(0x10, 0x14, 0x20), 0.4), 1.4)
		gradPath(dst, pathOf(robe), clear, alpha(spirit, 0.25))
		hood := []float64{x - 30, cy - 82, x - 22, cy - 120, x, cy - 136, x + 22, cy - 120, x + 30, cy - 82, x + 14, cy - 94, x - 14, cy - 94}
		shape(dst, hood, rgb(0x3a, 0x3e, 0x54), rgb(0x14, 0x16, 0x22), 1.6)
		blob(dst, []float64{x - 16, cy - 92, x - 12, cy - 116, x + 10, cy - 118, x + 16, cy - 92}, rgb(0x02, 0x02, 0x06), rgb(0x06, 0x06, 0x0c), 0)
		glowEye(dst, x-7, cy-104, 3.5, spirit)
		glowEye(dst, x+7, cy-104, 3.5, spirit)
		for _, d := range []float64{-1, 1} {
			ax := x - 46
			if d > 0 {
				ax = x + 36
			}
			line(dst, x+d*20, cy-70, ax, cy-48+d*6, 4, rgb(0x2a, 0x2e, 0x40))
			claws(dst, ax, cy-48+d*6, 0.9, rgb(0xd8, 0xe8, 0xe0))
		}
		for i := 0; i < 3; i++ {
			wy := cy + 20 - math.Mod(t*30+float64(i)*30, 90)
			glow(dst, x-20+float64(i)*20, wy, 10, spirit, 0.3)
		}
	case shCultist:
		robe := tint(rgb(0x7a, 0x18, 0x22), h*0.4)
		flame := rgb(0x7a, 0xff, 0x50)
		sway := math.Sin(t*2) * 3
		shadow(dst, x, y, 48, 10, 0.6)
		sx := x - 40
		line(dst, sx, y+2, sx+6+sway, y-160, 7, colInk)
		line(dst, sx, y+2, sx+6+sway, y-160, 4, rgb(0x5a, 0x3a, 0x22))
		glow(dst, sx+6+sway, y-176, 40, flame, 0.6)
		orb(dst, sx+6+sway, y-166, 9, 8, rgb(0xd8, 0xd0, 0xb8))
		disc(dst, sx+3+sway, y-167, 2, 2.5, colInk)
		disc(dst, sx+9+sway, y-167, 2, 2.5, colInk)
		blob(dst, []float64{sx - 4 + sway, y - 178, sx + 6 + sway, y - 200 + math.Sin(t*9)*4, sx + 16 + sway, y - 178, sx + 6 + sway, y - 172}, light(flame, 0.5), flame, 0)
		r := []float64{x - 24, y - 120, x + 24, y - 120, x + 34, y - 30, x + 40, y, x + 14, y - 8, x, y + 2, x - 16, y - 8, x - 38, y, x - 32, y - 40}
		blob(dst, r, light(robe, 0.1), shade(robe, 0.25), 1.6)
		line(dst, x, y-110, x+2, y-6, 2, alpha(colGold, 0.6))
		for i := 0; i < 3; i++ {
			gx := x - 10 + float64(i)*10
			shape(dst, []float64{gx - 3, y - 80, gx, y - 86, gx + 3, y - 80, gx, y - 74}, colGoldHi, colGoldLo, 0)
		}
		shape(dst, []float64{x - 16, y - 112, x - 40 + sway, y - 92, x - 36 + sway, y - 84, x - 10, y - 100}, robe, shade(robe, 0.3), 1.2)
		hood := []float64{x - 26, y - 112, x - 22, y - 146, x - 4, y - 166, x + 16, y - 150, x + 24, y - 112}
		shape(dst, hood, light(robe, 0.15), shade(robe, 0.3), 1.6)
		mask := []float64{x - 16, y - 146, x + 8, y - 148, x + 10, y - 128, x, y - 112, x - 14, y - 116, x - 20, y - 130}
		shape(dst, mask, rgb(0xe8, 0xe0, 0xc8), rgb(0x8a, 0x80, 0x68), 1.4)
		glowEye(dst, x-10, y-134, 2.8, flame)
		glowEye(dst, x+2, y-134, 2.8, flame)
		line(dst, x-8, y-122, x+2, y-120, 1.4, colInk)
	case shSkeleton:
		bone := tint(rgb(0xd8, 0xcc, 0xb0), h*0.2)
		bob := math.Sin(t*2.4) * 2
		shadow(dst, x, y, 46, 10, 0.6)
		bl := func(x0, y0, x1, y1, w float64) {
			line(dst, x0, y0, x1, y1, w+2.4, colInk)
			line(dst, x0, y0, x1, y1, w, bone)
		}
		bl(x-8, y-60, x-14, y-30, 5)
		bl(x-14, y-30, x-18, y, 4)
		bl(x+8, y-60, x+14, y-30, 5)
		bl(x+14, y-30, x+20, y, 4)
		shape(dst, []float64{x - 16, y - 66, x + 16, y - 66, x + 12, y - 54, x - 12, y - 54}, bone, shade(bone, 0.4), 1.2)
		bl(x, y-64+bob, x, y-122+bob, 5)
		for i := 0; i < 4; i++ {
			ry := y - 112 + float64(i)*11 + bob
			curve(dst, x-20, ry+5, x, ry-6, x+20, ry+5, 6, colInk)
			curve(dst, x-20, ry+5, x, ry-6, x+20, ry+5, 3, bone)
		}
		bl(x+16, y-116+bob, x+36, y-90+bob, 4)
		swx, swy := x+38, y-90+bob
		R := func(p []float64) []float64 { return rot(p, swx, swy, 0.5) }
		shape(dst, R([]float64{swx - 4, swy - 6, swx - 4, swy - 76, swx, swy - 84, swx + 4, swy - 70, swx + 3, swy - 52, swx + 5, swy - 40, swx + 4, swy - 6}), rgb(0x9a, 0x88, 0x78), rgb(0x4a, 0x38, 0x2a), 1.2)
		shape(dst, R([]float64{swx - 12, swy - 6, swx + 12, swy - 6, swx + 12, swy - 1, swx - 12, swy - 1}), rgb(0x7a, 0x6a, 0x5a), rgb(0x3a, 0x2a, 0x1a), 1)
		sx, sy := x-30, y-84+bob
		disc(dst, sx, sy, 28, 30, colInk)
		radial(dst, sx, sy, 26, 28, -0.3, -0.3, rgb(0x6a, 0x4a, 0x30), rgb(0x2a, 0x18, 0x0c))
		strokeCircle(dst, sx, sy, 22, 2.5, rgb(0x7a, 0x5a, 0x40))
		orb(dst, sx, sy, 6, 6, rgb(0x8a, 0x7a, 0x6a))
		hy := y - 146 + bob
		blob(dst, []float64{x - 20, hy, x - 18, hy - 22, x, hy - 30, x + 18, hy - 22, x + 20, hy, x + 10, hy + 12, x - 10, hy + 12}, light(bone, 0.15), shade(bone, 0.5), 1.6)
		shape(dst, []float64{x - 12, hy + 8, x + 10, hy + 8, x + 8, hy + 20, x - 10, hy + 20}, shade(bone, 0.8), shade(bone, 0.45), 1.4)
		for i := 0; i < 4; i++ {
			line(dst, x-7+float64(i)*5, hy+10, x-7+float64(i)*5, hy+18, 1.2, alpha(colInk, 0.7))
		}
		for _, d := range []float64{-1, 1} {
			disc(dst, x+d*8-2, hy-4, 6, 7, colInk)
			glowEye(dst, x+d*8-2, hy-3, 2.5, colRed)
		}
		shape(dst, []float64{x - 4, hy + 6, x, hy, x + 2, hy + 6}, colInk, colInk, 0)
	case shKnight:
		void := tint(rgb(0xa0, 0x50, 0xff), h)
		br := math.Sin(t*1.8) * 2
		shadow(dst, x, y, 64, 13, 0.65)
		glow(dst, x, y-100, 120, void, 0.2)
		cape := []float64{x - 20, y - 160 + br, x + 30, y - 160 + br, x + 56, y - 4, x + 36, y - 16, x + 22, y + 2, x + 6, y - 14, x - 10, y}
		blob(dst, cape, rgb(0x2a, 0x10, 0x40), rgb(0x0a, 0x04, 0x12), 1.6)
		for _, d := range []float64{-1, 1} {
			lx := x + d*14
			shape(dst, []float64{lx - 12, y - 90, lx + 12, y - 90, lx + 10, y - 10, lx + 14, y, lx - 14, y, lx - 10, y - 10}, rgb(0x3a, 0x38, 0x48), rgb(0x0c, 0x0a, 0x12), 1.6)
			line(dst, lx, y-80, lx, y-20, 1.4, alpha(void, 0.7))
		}
		by := y - 160 + br
		chest := []float64{x - 34, by, x + 34, by, x + 30, by + 56, x + 16, by + 76, x - 16, by + 76, x - 30, by + 56}
		shape(dst, chest, rgb(0x4a, 0x46, 0x5a), rgb(0x0c, 0x0a, 0x14), 1.8)
		for _, c := range [][4]float64{{-20, 10, -6, 40}, {-6, 40, -14, 64}, {12, 14, 4, 46}, {4, 46, 14, 70}} {
			line(dst, x+c[0], by+c[1], x+c[2], by+c[3], 2, light(void, 0.4))
		}
		glow(dst, x, by+40, 30, void, 0.5)
		gx := x - 46
		glow(dst, gx, y-70, 50, void, 0.3)
		shape(dst, []float64{gx - 7, y - 4, gx - 7, y - 130, gx, y - 140, gx + 7, y - 130, gx + 7, y - 4}, rgb(0x5a, 0x5a, 0x6e), rgb(0x16, 0x14, 0x20), 1.4)
		line(dst, gx, y-10, gx, y-130, 1.6, alpha(void, 0.9))
		shape(dst, []float64{gx - 26, y - 132, gx + 26, y - 132, gx + 20, y - 124, gx - 20, y - 124}, rgb(0x6a, 0x66, 0x7a), rgb(0x1a, 0x18, 0x24), 1.2)
		shape(dst, []float64{gx - 4, y - 132, gx + 4, y - 132, gx + 4, y - 160, gx - 4, y - 160}, rgb(0x3a, 0x22, 0x1a), rgb(0x10, 0x08, 0x04), 1)
		line(dst, x-26, by+12, gx+4, y-150, 10, colInk)
		line(dst, x-26, by+12, gx+4, y-150, 7, rgb(0x3a, 0x38, 0x48))
		orb(dst, gx, y-150, 9, 8, rgb(0x3a, 0x38, 0x48))
		for _, d := range []float64{-1, 1} {
			px := x + d*34
			shape(dst, []float64{px - 18, by + 18, px - 20, by - 4, px, by - 16, px + 20, by - 4, px + 18, by + 18}, rgb(0x4a, 0x46, 0x5a), rgb(0x0c, 0x0a, 0x14), 1.6)
			for i := 0; i < 2; i++ {
				sx := px - 8 + float64(i)*14
				shape(dst, []float64{sx - 4, by - 8, sx + d*4, by - 30 - float64(i)*6, sx + 4, by - 10}, rgb(0x6a, 0x66, 0x7a), rgb(0x1a, 0x18, 0x24), 1.2)
			}
		}
		hy := by - 6
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*14, hy - 34, x + d*36, hy - 40, x + d*50, hy - 62, x + d*46, hy - 80, x + d*42, hy - 58, x + d*24, hy - 48}, rgb(0x2a, 0x26, 0x34), rgb(0x08, 0x06, 0x0c), 1.4)
		}
		helm := []float64{x - 20, hy, x + 20, hy, x + 24, hy - 30, x + 14, hy - 54, x, hy - 60, x - 14, hy - 54, x - 24, hy - 30}
		shape(dst, helm, rgb(0x52, 0x4e, 0x64), rgb(0x0c, 0x0a, 0x14), 1.8)
		glow(dst, x-4, hy-30, 26, void, 0.6)
		shape(dst, []float64{x - 18, hy - 33, x + 12, hy - 33, x + 12, hy - 27, x - 18, hy - 27}, light(void, 0.6), void, 0)
	case shGargoyle:
		stone := tint(rgb(0x6a, 0x6a, 0x78), h*0.3)
		ember := rgb(0xff, 0x8a, 0x30)
		flap := math.Sin(t*2) * 8
		shadow(dst, x, y, 70, 13, 0.6)
		for _, d := range []float64{-1, 1} {
			wing := []float64{x + d*14, y - 110, x + d*60, y - 150 - flap, x + d*96, y - 170 - flap, x + d*92, y - 130 - flap*0.5, x + d*104, y - 108, x + d*84, y - 104, x + d*88, y - 82, x + d*64, y - 88, x + d*56, y - 66, x + d*22, y - 84}
			shape(dst, wing, shade(stone, 0.7), shade(stone, 0.25), 1.6)
			line(dst, x+d*14, y-110, x+d*96, y-170-flap, 2.5, alpha(colInk, 0.6))
			line(dst, x+d*50, y-136-flap*0.7, x+d*84, y-104, 1.6, alpha(colInk, 0.5))
			line(dst, x+d*44, y-124-flap*0.6, x+d*64, y-88, 1.6, alpha(colInk, 0.5))
		}
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*10, y - 66, x + d*40, y - 56, x + d*36, y - 20, x + d*46, y + 1, x + d*20, y + 1, x + d*18, y - 30}, light(stone, 0.05), shade(stone, 0.3), 1.6)
			claws(dst, x+d*40-10, y-2, 0.7, rgb(0x2a, 0x2a, 0x30))
		}
		blob(dst, []float64{x - 30, y - 60, x - 32, y - 104, x - 10, y - 124, x + 16, y - 120, x + 32, y - 96, x + 28, y - 58}, light(stone, 0.12), shade(stone, 0.3), 1.8)
		for i := 0; i < 3; i++ {
			curve(dst, x-20, y-100+float64(i)*12, x, y-94+float64(i)*12, x+20, y-100+float64(i)*12, 1.6, alpha(colInk, 0.4))
		}
		line(dst, x-6, y-118, x+4, y-80, 1.5, alpha(ember, 0.7))
		glow(dst, x, y-100, 20, ember, 0.3)
		hy := y - 132
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*10, hy - 16, x + d*24, hy - 26, x + d*30, hy - 48, x + d*26, hy - 58, x + d*24, hy - 38, x + d*12, hy - 26}, rgb(0x4a, 0x48, 0x52), rgb(0x10, 0x10, 0x16), 1.2)
		}
		blob(dst, []float64{x - 20, hy - 6, x - 14, hy - 24, x + 12, hy - 26, x + 20, hy - 6, x + 8, hy + 14, x - 12, hy + 14}, light(stone, 0.15), shade(stone, 0.35), 1.6)
		shape(dst, []float64{x - 14, hy + 4, x + 6, hy + 4, x + 2, hy + 16, x - 10, hy + 16}, shade(stone, 0.4), shade(stone, 0.2), 1)
		fangs(dst, x-12, hy+4, 3, 6)
		glowEye(dst, x-9, hy-8, 3.5, ember)
		glowEye(dst, x+5, hy-8, 3.2, ember)
	case shDemon:
		skin := tint(rgb(0x6a, 0x1a, 0x1e), h*0.5)
		lava := rgb(0xff, 0x8a, 0x20)
		br := math.Sin(t*1.4) * 3
		flap := math.Sin(t*1.6) * 10
		shadow(dst, x, y, 96, 16, 0.7)
		glow(dst, x, y-120, 170, lava, 0.18)
		for _, d := range []float64{-1, 1} {
			wing := []float64{x + d*24, y - 170 + br, x + d*90, y - 220 - flap, x + d*150, y - 236 - flap, x + d*140, y - 180 - flap*0.5, x + d*156, y - 140, x + d*122, y - 138, x + d*126, y - 104, x + d*92, y - 114, x + d*84, y - 82, x + d*36, y - 120}
			shape(dst, wing, rgb(0x3a, 0x0e, 0x14), rgb(0x10, 0x02, 0x04), 1.8)
			for _, p := range [][2]float64{{122, -138}, {92, -114}, {150, -236}} {
				line(dst, x+d*30, y-160+br, x+d*p[0], y+p[1]-flap*0.4, 2.2, alpha(rgb(0x6a, 0x1a, 0x20), 0.9))
			}
		}
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*10, y - 84, x + d*48, y - 74, x + d*40, y - 34, x + d*56, y + 1, x + d*22, y + 1, x + d*24, y - 36}, light(skin, 0.05), shade(skin, 0.25), 1.8)
			shape(dst, []float64{x + d*24, y + 1, x + d*30, y + 8, x + d*36, y + 1}, colBone, colCream, 1)
			shape(dst, []float64{x + d*42, y + 1, x + d*50, y + 8, x + d*56, y + 1}, colBone, colCream, 1)
		}
		by := y - 186 + br
		torso := []float64{x - 50, by + 20, x - 40, by - 6, x + 40, by - 6, x + 50, by + 20, x + 36, by + 90, x + 20, by + 110, x - 20, by + 110, x - 36, by + 90}
		blob(dst, torso, light(skin, 0.12), shade(skin, 0.22), 2)
		curve(dst, x-30, by+30, x-14, by+46, x, by+34, 1.6, alpha(colInk, 0.5))
		curve(dst, x+30, by+30, x+14, by+46, x, by+34, 1.6, alpha(colInk, 0.5))
		for _, c := range [][4]float64{{-22, 20, -8, 60}, {-8, 60, -18, 96}, {18, 24, 6, 56}, {6, 56, 16, 92}, {-4, 70, 4, 104}} {
			line(dst, x+c[0], by+c[1], x+c[2], by+c[3], 2.5, lava)
			glow(dst, x+(c[0]+c[2])/2, by+(c[1]+c[3])/2, 14, lava, 0.4)
		}
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*38, by + 2, x + d*74, by + 36, x + d*76, by + 90, x + d*58, by + 94, x + d*54, by + 50, x + d*32, by + 28}, light(skin, 0.06), shade(skin, 0.25), 1.8)
			shape(dst, []float64{x + d*48, by + 16, x + d*66, by + 4, x + d*58, by + 22}, rgb(0x2a, 0x22, 0x20), rgb(0x0a, 0x06, 0x06), 1)
			claws(dst, x+d*66, by+92, 1.2, rgb(0x1a, 0x10, 0x10))
		}
		hy := by - 18
		for _, d := range []float64{-1, 1} {
			blob(dst, []float64{x + d*16, hy - 14, x + d*44, hy - 24, x + d*62, hy - 56, x + d*58, hy - 86, x + d*50, hy - 56, x + d*30, hy - 34, x + d*18, hy - 30}, rgb(0x2a, 0x22, 0x20), rgb(0x0a, 0x06, 0x06), 1.6)
		}
		head := []float64{x - 26, hy + 8, x - 30, hy - 18, x - 14, hy - 38, x + 14, hy - 38, x + 30, hy - 18, x + 26, hy + 8, x + 10, hy + 24, x - 10, hy + 24}
		blob(dst, head, light(skin, 0.15), shade(skin, 0.3), 1.8)
		for i := 0; i < 5; i++ {
			fx := x - 20 + float64(i)*10
			fh := 18 + 10*math.Sin(t*7+float64(i)*1.7)
			blob(dst, []float64{fx - 6, hy - 34, fx, hy - 34 - fh, fx + 6, hy - 34}, rgb(0xff, 0xe0, 0x80), lava, 0)
		}
		shape(dst, []float64{x - 16, hy + 8, x + 12, hy + 8, x + 8, hy + 18, x - 12, hy + 18}, rgb(0x2a, 0x04, 0x04), rgb(0x10, 0x00, 0x00), 1)
		glow(dst, x-2, hy+12, 14, lava, 0.6)
		fangs(dst, x-14, hy+8, 5, 6)
		glowEye(dst, x-12, hy-12, 4.5, rgb(0xff, 0xd0, 0x40))
		glowEye(dst, x+8, hy-12, 4.5, rgb(0xff, 0xd0, 0x40))
		shape(dst, []float64{x - 22, hy - 22, x - 4, hy - 16, x - 6, hy - 12, x - 22, hy - 16}, colInk, colInk, 0)
		shape(dst, []float64{x + 18, hy - 22, x, hy - 16, x + 2, hy - 12, x + 18, hy - 16}, colInk, colInk, 0)
	}
}
