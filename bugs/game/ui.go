package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Interface pieces, all painted: chunky glossy buttons with a pressable lip,
// rounded panels, progress bars and icons.

// btnArt is a button image 2h wide and h tall, for slice3.
func btnArt(h float64, col color.RGBA, pressed bool) *pimg {
	key := "btn" + num(h) + itoa(int(col.R)<<16|int(col.G)<<8|int(col.B))
	if pressed {
		key += "p"
	}
	return paintArt(key, 0, 0, 2*h, h, 1, func(c *canvas) {
		w := 2 * h
		r := h * 0.34
		lip := h * 0.12
		top := 0.0
		if pressed {
			top = lip * 0.8
		}
		ol := outlineOf(col)
		c.fill(rrect(0, top, w, h-top, r), ol)
		c.fill(rrect(2.6, top+2.6, w-5.2, h-top-5.2, r-2.6), shade(col, 0.66))
		face := rrect(2.6, top+2.6, w-5.2, h-top-5.2-lip+top, r-2.6)
		c.vol(face, mat{base: col, dark: shade(col, 0.82), flat: 0.55, depth: h * 0.3, rim: 0.3})
		c.fill(rrect(r*0.6, top+5, w-r*1.2, (h-lip)*0.3, (h-lip)*0.15), alpha(colWhite, 0.32))
	})
}

var btnLip = 0.12

// button draws a pressable button with a label; it returns nothing, taps are
// handled by the screen's hit boxes.
func button(dst *ebiten.Image, r rect, col color.RGBA, label string, size float64, pressed, off bool) {
	if off {
		col = rgb(0xa0, 0x98, 0xb8)
	}
	slice3(dst, btnArt(r.h, col, pressed), r.x, r.y, r.w, cs1)
	dy := -r.h * btnLip / 2
	if pressed {
		dy = r.h * btnLip * 0.3
	}
	if label != "" {
		say(dst, label, r.cx(), r.cy()+dy, size, colWhite, 0.5)
	}
}

// panelArt is a rounded panel for slice9 (corner 22 units).
func panelArt(style int) *pimg {
	return paintArt("panel"+itoa(style), 0, 0, 66, 66, 1, func(c *canvas) {
		switch style {
		case 0: // light card
			c.fill(rrect(0, 0, 66, 66, 20), rgb(0x3a, 0x22, 0x6e))
			c.fill(rrect(2.5, 2.5, 61, 61, 17.5), rgb(0xc8, 0xb4, 0xf0))
			c.vol(rrect(2.5, 2.5, 61, 58, 17.5), mat{base: rgb(0xf4, 0xee, 0xff), dark: rgb(0xe0, 0xd4, 0xf8), flat: 0.7, depth: 14})
		case 1: // deep purple tray
			c.fill(rrect(0, 0, 66, 66, 22), rgb(0x24, 0x12, 0x48))
			c.grad(rrect(3, 3, 60, 60, 19), rgb(0x6e, 0x4a, 0xc0), rgb(0x4a, 0x2c, 0x8e), 3, 63)
			c.fill(rrect(3, 3, 60, 60, 19).ring(2).and(rrect(0, 0, 66, 30, 0)), alpha(colWhite, 0.18))
		case 2: // golden highlight card
			c.fill(rrect(0, 0, 66, 66, 20), rgb(0x7a, 0x3a, 0x10))
			c.fill(rrect(2.5, 2.5, 61, 61, 17.5), rgb(0xf0, 0xa0, 0x30))
			c.vol(rrect(2.5, 2.5, 61, 58, 17.5), mat{base: rgb(0xff, 0xf0, 0xc0), dark: rgb(0xff, 0xd8, 0x80), flat: 0.7, depth: 14})
		case 3: // dark glass (counters on the battlefield)
			c.fill(rrect(0, 0, 66, 66, 22), alpha(rgb(0x1a, 0x0c, 0x34), 0.6))
		}
	})
}

func panel(dst *ebiten.Image, r rect, style int) {
	slice9(dst, panelArt(style), r, 20, cs1)
}

// barArt is a pill-shaped bar image (track or fill) 2h wide.
func barArt(h float64, col color.RGBA, track bool) *pimg {
	key := "bar" + num(h) + itoa(int(col.R)<<16|int(col.G)<<8|int(col.B))
	if track {
		key += "t"
	}
	return paintArt(key, 0, 0, 2*h, h, 1, func(c *canvas) {
		s := rrect(0, 0, 2*h, h, h/2)
		if track {
			c.fill(s, rgb(0x24, 0x12, 0x40))
			c.fill(rrect(2, 2, 2*h-4, h-4, h/2-2), rgb(0x3e, 0x26, 0x66))
			return
		}
		in := rrect(2, 2, 2*h-4, h-4, h/2-2)
		c.vol(in, mat{base: col, flat: 0.5, depth: h * 0.4, gloss: 0.4})
		c.fill(rrect(h*0.3, 3.5, 2*h-h*0.6, h*0.22, h*0.11), alpha(colWhite, 0.4))
	})
}

// bar draws a progress bar filled to fraction f.
func bar(dst *ebiten.Image, x, y, w, h, f float64, col color.RGBA) {
	slice3(dst, barArt(h, col, true), x, y, w, cs1)
	f = clamp01(f)
	if f > 0 {
		fw := math.Max(h, w*f)
		slice3(dst, barArt(h, col, false), x, y, fw, cs1)
	}
}

// ---- icons ----

const (
	icCoin = iota
	icGem
	icEgg
	icGoldEgg
	icTap
	icStar
	icStarOff
	icLock
	icClock
	icChest
	icGift
	icSack
	icPlus
	icCheck
	icSword
	icCrown
)

// icon returns a painted icon size units across, anchored at its centre.
func icon(kind int, size float64) *pimg {
	r := size / 2
	return paintArt("ic"+itoa(kind)+"@"+num(size*10), -r-3, -r-3, r+3, r+3, 1, func(c *canvas) {
		u := size / 40 // icons are drawn on a 40-unit grid
		paintIcon(c, kind, u)
	})
}

func star5(cx, cy, r, ri float64) sh {
	var p []float64
	for i := 0; i < 10; i++ {
		a := -math.Pi/2 + float64(i)*math.Pi/5
		rr := r
		if i%2 == 1 {
			rr = ri
		}
		p = append(p, cx+math.Cos(a)*rr, cy+math.Sin(a)*rr)
	}
	return poly(p...)
}

func paintIcon(c *canvas, kind int, u float64) {
	ow := math.Max(1.2, 2.2*u)
	switch kind {
	case icCoin:
		gold := rgb(0xff, 0xc4, 0x2a)
		c.part(circ(0, 0, 18*u), mat{base: gold, gloss: 0.9, shine: 30, rim: 0.5}, ow)
		c.fill(circ(0, 0, 13*u).ring(2*u), alpha(rgb(0xc8, 0x80, 0x10), 0.7))
		c.volOn(star5(0, 0.5*u, 8*u, 3.6*u), circ(0, 0, 18*u), mat{base: rgb(0xff, 0xe8, 0x80), gloss: 0.6})
	case icGem:
		g := rgb(0x40, 0xe0, 0xd0)
		out := poly(-16*u, -6*u, -8*u, -15*u, 8*u, -15*u, 16*u, -6*u, 0, 17*u)
		c.fill(out.grow(ow), outlineOf(rgb(0x20, 0x80, 0xa0)))
		c.fill(out, g)
		c.fill(poly(-16*u, -6*u, 16*u, -6*u, 0, 17*u), rgb(0x20, 0xb0, 0xc8))
		c.fill(poly(-6*u, -6*u, 6*u, -6*u, 0, 17*u), rgb(0x50, 0xd8, 0xe8))
		c.fill(poly(-8*u, -15*u, 8*u, -15*u, 5*u, -6*u, -5*u, -6*u), rgb(0xb0, 0xff, 0xf4))
		c.fill(star5(-8*u, -10*u, 4*u, 1.2*u), colWhite)
	case icEgg, icGoldEgg:
		base := rgb(0xff, 0xf2, 0xdc)
		spot := rgb(0x7a, 0xc8, 0x4a)
		if kind == icGoldEgg {
			base, spot = rgb(0xff, 0xc8, 0x30), rgb(0xff, 0x8a, 0x2a)
		}
		egg := blob(0, -18*u, 11*u, -10*u, 14*u, 4*u, 8*u, 16*u, 0, 18*u, -8*u, 16*u, -14*u, 4*u, -11*u, -10*u)
		c.part(egg, mat{base: base, gloss: 0.8, rim: 0.4}, ow)
		for _, s := range [][3]float64{{-5, -6, 4}, {6, 2, 3.2}, {-3, 9, 3.6}, {4, -12, 2.4}} {
			c.volOn(circ(s[0]*u, s[1]*u, s[2]*u).and(egg), egg, mat{base: spot, gloss: 0.6})
		}
		if kind == icGoldEgg {
			c.fill(star5(10*u, -12*u, 5*u, 1.5*u), colWhite)
		}
	case icTap:
		s := star5(0, 1*u, 19*u, 9*u)
		c.part(s, mat{base: rgb(0xff, 0xa0, 0x30), gloss: 0.5}, ow)
		c.volOn(star5(0, 1*u, 11*u, 5*u), s, mat{base: rgb(0xff, 0xf0, 0x70), gloss: 0.6})
	case icStar, icStarOff:
		col := rgb(0xff, 0xcc, 0x2a)
		if kind == icStarOff {
			col = rgb(0x6a, 0x5a, 0x8a)
		}
		c.part(star5(0, 1.5*u, 19*u, 8.5*u), mat{base: col, gloss: 0.8, rim: 0.4}, ow)
	case icLock:
		c.fill(arc(0, -6*u, 8*u, 9*u, math.Pi, 2*math.Pi, 3.6*u).grow(ow), rgb(0x3a, 0x30, 0x50))
		c.vol(arc(0, -6*u, 8*u, 9*u, math.Pi, 2*math.Pi, 3.6*u), mat{base: rgb(0xb8, 0xb8, 0xd0), gloss: 0.8})
		b := rrect(-13*u, -6*u, 26*u, 22*u, 5*u)
		c.part(b, mat{base: rgb(0xff, 0xc0, 0x30), gloss: 0.8, rim: 0.4}, ow)
		c.fill(circ(0, 3*u, 3.4*u).or(rrect(-1.6*u, 3*u, 3.2*u, 7*u, 1.5*u)), rgb(0x7a, 0x44, 0x10))
	case icClock:
		c.part(circ(0, 0, 17*u), mat{base: rgb(0xff, 0xff, 0xff), dark: rgb(0xd8, 0xd0, 0xf0), gloss: 0.4}, ow)
		c.fill(circ(0, 0, 17*u).ring(3*u), rgb(0xff, 0x7a, 0x5a))
		c.fill(capsule(0, 0, 0, -10*u, 1.8*u, 1.6*u).or(capsule(0, 0, 7*u, 3*u, 1.8*u, 1.6*u)), rgb(0x3a, 0x22, 0x5a))
	case icChest:
		wd := rgb(0xb8, 0x6a, 0x30)
		lid := rrect(-17*u, -15*u, 34*u, 14*u, 6*u)
		box := rrect(-16*u, -4*u, 32*u, 20*u, 3*u)
		c.part(box, mat{base: wd, gloss: 0.4, tex: 0.2, texF: 0.4}, ow)
		c.part(lid, mat{base: light(wd, 0.1), gloss: 0.5, tex: 0.2, texF: 0.4}, ow)
		c.part(rrect(-17*u, -5*u, 34*u, 4*u, 1.5*u), mat{base: rgb(0xff, 0xc8, 0x30), gloss: 1}, ow*0.6)
		c.part(rrect(-4*u, -8*u, 8*u, 10*u, 2*u), mat{base: rgb(0xff, 0xd8, 0x40), gloss: 1}, ow*0.6)
		c.fill(star5(10*u, -15*u, 5*u, 1.5*u), colWhite)
	case icGift:
		b := rrect(-15*u, -6*u, 30*u, 22*u, 3*u)
		c.part(b, mat{base: rgb(0xff, 0x5a, 0x9a), gloss: 0.6}, ow)
		l := rrect(-17*u, -12*u, 34*u, 8*u, 3*u)
		c.part(l, mat{base: rgb(0xff, 0x7a, 0xb0), gloss: 0.6}, ow)
		rib := mat{base: rgb(0xff, 0xd8, 0x40), gloss: 0.8}
		c.part(rrect(-3*u, -12*u, 6*u, 28*u, 1*u), rib, ow*0.6)
		c.part(ell(-6*u, -15*u, 6*u, 4*u).rot(-6*u, -15*u, 0.5).or(ell(6*u, -15*u, 6*u, 4*u).rot(6*u, -15*u, -0.5)), rib, ow*0.6)
	case icSack:
		s := blob(-6*u, -12*u, 6*u, -12*u, 15*u, 4*u, 12*u, 16*u, -12*u, 16*u, -15*u, 4*u)
		c.part(s, mat{base: rgb(0xc8, 0x90, 0x58), gloss: 0.3, tex: 0.15, texF: 0.5}, ow)
		c.part(ell(0, -14*u, 8*u, 4*u), mat{base: rgb(0xb0, 0x78, 0x40)}, ow)
		c.part(circ(0, 5*u, 7*u), mat{base: rgb(0xff, 0xc4, 0x2a), gloss: 0.9}, ow*0.7)
	case icPlus:
		p := rrect(-4*u, -14*u, 8*u, 28*u, 3*u).or(rrect(-14*u, -4*u, 28*u, 8*u, 3*u))
		c.part(p, mat{base: rgb(0x8c, 0xf0, 0x50), gloss: 0.6}, ow)
	case icCheck:
		c.part(circ(0, 0, 16*u), mat{base: rgb(0x6c, 0xd8, 0x3a), gloss: 0.6}, ow)
		c.fill(line(3*u, 3*u, -7*u, 0, -2*u, 6*u, 8*u, -6*u), colWhite)
	case icSword:
		bl := poly(-3*u, 8*u, -3*u, -14*u, 0, -19*u, 3*u, -14*u, 3*u, 8*u)
		c.part(bl, mat{base: rgb(0xe0, 0xe8, 0xff), gloss: 1, shine: 30}, ow)
		c.part(rrect(-10*u, 7*u, 20*u, 4.5*u, 2*u), mat{base: rgb(0xff, 0xc0, 0x30), gloss: 0.8}, ow)
		c.part(rrect(-2.5*u, 11*u, 5*u, 8*u, 2*u), mat{base: rgb(0x8a, 0x4a, 0x2a)}, ow)
	case icCrown:
		cr := poly(-16*u, 10*u, -18*u, -10*u, -8*u, -2*u, 0, -16*u, 8*u, -2*u, 18*u, -10*u, 16*u, 10*u)
		c.part(cr, mat{base: rgb(0xff, 0xc8, 0x30), gloss: 1.2, shine: 30, flat: 0.2}, ow)
		c.part(circ(0, 3*u, 3.5*u), mat{base: rgb(0xff, 0x40, 0x7a), gloss: 1}, ow*0.5)
	}
}

func drawIcon(dst *ebiten.Image, kind int, x, y, size float64) {
	icon(kind, size).draw(dst, x, y, 1, cs1)
}

// glowArt is a soft round light, for sparkles and the hatch glow.
func glowArt() *pimg {
	return paintArt("glow", -50, -50, 50, 50, 0.6, func(c *canvas) {
		c.each(circ(0, 0, 50), func(i int, x, y, d float64) {
			t := clamp01(1 - math.Hypot(x, y)/50)
			c.put(i, 1, 1, 1, t*t)
		})
	})
}

// sparkArt is a four-pointed twinkle.
func sparkArt() *pimg {
	return paintArt("spark", -20, -20, 20, 20, 1, func(c *canvas) {
		s := poly(0, -18, 3.5, -3.5, 18, 0, 3.5, 3.5, 0, 18, -3.5, 3.5, -18, 0, -3.5, -3.5)
		c.soft(s, colWhite, 2)
	})
}

// raysArt is a burst of light rays behind a reward.
func raysArt() *pimg {
	return paintArt("rays", -200, -200, 200, 200, 0.7, func(c *canvas) {
		var s sh
		for i := 0; i < 12; i++ {
			a := float64(i) / 12 * 2 * math.Pi
			p := poly(0, 0, math.Cos(a-0.12)*200, math.Sin(a-0.12)*200, math.Cos(a+0.12)*200, math.Sin(a+0.12)*200)
			if i == 0 {
				s = p
			} else {
				s = s.or(p)
			}
		}
		c.each(s, func(i int, x, y, d float64) {
			t := clamp01(1 - math.Hypot(x, y)/200)
			c.put(i, 1, 1, 0.85, cov(d)*t*0.9)
		})
	})
}
