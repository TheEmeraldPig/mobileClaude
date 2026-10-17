package game

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Cards are 120x168 at scale 1: rarity frame, illustration window, name
// banner, rules box, cost orb and rarity gem.

const cardW, cardH = 120.0, 168.0

var kindBG = [][2]color.RGBA{
	{rgb(0x9a, 0x30, 0x24), rgb(0x1e, 0x06, 0x08)},
	{rgb(0x24, 0x5a, 0x9a), rgb(0x06, 0x0c, 0x1e)},
}

// drawCard draws card c with its top-left at (x,y), w wide.
func drawCard(dst *ebiten.Image, c Card, x, y, w float64, dim bool, t float64) {
	s := w / cardW
	h := cardH * s
	d := c.stats()
	base := c.def()
	rc := rarityCol[base.Rarity]
	R := func(a, b, ww, hh float64) rect { return rect{x + a*s, y + b*s, ww * s, hh * s} }
	full := R(0, 0, cardW, cardH)

	if c.Enh != eNone {
		ec := enhDefs[c.Enh].Col
		if c.Enh == eHolo {
			ec = hue(t*0.4 + x*0.003)
		}
		glow(dst, full.cx(), full.cy(), w*0.9, ec, 0.35+0.15*math.Sin(t*4))
	}
	shadow(dst, full.cx(), full.y+full.h+2*s, w*0.6, 10*s, 0.7)
	// frame
	fillPath(dst, chamfer(full.grow(3.5*s), 12*s), colInk)
	goldEdge(dst, chamfer(full.grow(2.5*s), 11*s), chamfer(full.grow(1*s), 10*s))
	texPath(dst, chamfer(full, 9*s), mix(rgb(0x30, 0x2c, 0x36), rc, 0.12), rgb(0x0c, 0x0a, 0x10))
	if base.Rarity >= rRare {
		strokePath(dst, chamfer(full.grow(-3*s), 7*s), 1.2*s, alpha(rc, 0.7))
	}
	if base.Rarity == rLegendary { // a light that runs around the frame
		a := t * 2
		px, py := full.cx()+math.Cos(a)*w*0.55, full.cy()+math.Sin(a)*h*0.55
		glow(dst, math.Max(x, math.Min(x+w, px)), math.Max(y, math.Min(y+h, py)), 26*s, colGoldHi, 0.8)
	}
	// art window
	aw := R(8, 24, cardW-16, 72)
	fillPath(dst, chamfer(aw.grow(2*s), 6*s), colInk)
	strokePath(dst, chamfer(aw.grow(1.5*s), 6*s), 1.2*s, colGoldLo)
	bg := kindBG[d.Kind]
	sub := clip(dst, aw)
	radial(sub, aw.cx(), aw.cy(), aw.w*0.75, aw.h*0.95, 0, -0.15, bg[0], bg[1])
	for i := 0; i < 8; i++ {
		a := float64(i)*math.Pi/4 + t*0.12
		cx, cy := aw.cx(), aw.cy()
		shape(sub, []float64{cx, cy, cx + math.Cos(a)*w, cy + math.Sin(a)*w, cx + math.Cos(a+0.16)*w, cy + math.Sin(a+0.16)*w}, alpha(light(bg[0], 0.4), 0.12), clear, 0)
	}
	cardArt(sub, base.Art, aw.cx(), aw.cy()+2*s, 30*s, t, base.Rarity)
	if c.Enh == eFoil || c.Enh == eHolo {
		sx := math.Mod(t*90*s, aw.w*3) - aw.w
		col := alpha(colWhite, 0.3)
		if c.Enh == eHolo {
			col = alpha(hue(t*0.5), 0.35)
		}
		shape(sub, []float64{aw.x + sx, aw.y, aw.x + sx + 16*s, aw.y, aw.x + sx - 20*s, aw.y + aw.h, aw.x + sx - 36*s, aw.y + aw.h}, col, alpha(col, 0.3), 0)
	}
	gradRect(sub, aw.x, aw.y+aw.h*0.6, aw.w, aw.h*0.4, clear, alpha(colInk, 0.55))
	// name
	ns := 13.5 * s
	if tw := titleWidth(base.Name, ns); tw > w-34*s {
		ns *= (w - 34*s) / tw
	}
	drawFont(dst, fTitle, base.Name, x+w/2+5*s, y+13.5*s, ns, color.RGBA{0, 0, 0, 0xc0}, 0.5)
	drawFont(dst, fTitle, base.Name, x+w/2+5*s, y+12.5*s, ns, colBone, 0.5)
	gradRect(dst, x+14*s, y+22*s, w/2-14*s, 1.2*s, clear, rc)
	gradRect(dst, x+w/2, y+22*s, w/2-14*s, 1.2*s, rc, clear)
	// rules box
	tb := R(8, 101, cardW-16, 58)
	fillPath(dst, chamfer(tb.grow(1.5*s), 5*s), colInk)
	texPath(dst, chamfer(tb, 4*s), rgb(0x2e, 0x26, 0x1e), rgb(0x12, 0x0e, 0x0a))
	gradRect(dst, tb.x+6*s, tb.y, tb.w-12*s, 1.2*s, alpha(colGold, 0.6), alpha(colGold, 0.6))
	txt := c.text()
	lines := 1 + strings.Count(txt, "\n")
	ts := 13.5 * s
	if lines >= 4 {
		ts = 11.5 * s
	}
	for _, ln := range strings.Split(txt, "\n") {
		if tw := textWidth(ln, ts); tw > tb.w-6*s {
			ts *= (tb.w - 6*s) / tw
		}
	}
	drawText(dst, txt, tb.cx(), tb.cy(), ts, colBone, 0.5)
	// cost crystal
	cc := colBlue
	if d.Cost < base.Cost {
		cc = colGreen
	}
	cx, cy := x+5*s, y+5*s
	hex := []float64{cx, cy - 15*s, cx + 13*s, cy - 7*s, cx + 13*s, cy + 7*s, cx, cy + 15*s, cx - 13*s, cy + 7*s, cx - 13*s, cy - 7*s}
	glow(dst, cx, cy, 24*s, cc, 0.4)
	shape(dst, scaleAround(hex, cx, cy, 1.18, 1.18), colGoldHi, colGoldLo, 1.2*s)
	shape(dst, hex, light(cc, 0.35), shade(cc, 0.35), 0)
	shape(dst, []float64{cx, cy - 15*s, cx + 13*s, cy - 7*s, cx, cy}, alpha(colWhite, 0.35), alpha(colWhite, 0.1), 0)
	say(dst, itoa(d.Cost), cx, cy-1*s, 17*s, colWhite, 0.5)
	// rarity gem
	gx, gy := x+w/2, y+h+1*s
	shape(dst, []float64{gx - 9*s, gy, gx, gy - 7*s, gx + 9*s, gy, gx, gy + 7*s}, colGoldHi, colGoldLo, 1.2*s)
	shape(dst, []float64{gx - 5.5*s, gy, gx, gy - 4.5*s, gx + 5.5*s, gy, gx, gy + 4.5*s}, light(rc, 0.4), shade(rc, 0.5), 0)
	glow(dst, gx, gy, 12*s, rc, 0.5)
	// enhancement tag
	if c.Enh != eNone {
		ed := enhDefs[c.Enh]
		r := rect{x + w - 46*s, y + h - 14*s, 42 * s, 16 * s}
		fillPath(dst, chamfer(r.grow(1.5*s), 5*s), colInk)
		gradPath(dst, chamfer(r, 4*s), shade(ed.Col, 0.5), shade(ed.Col, 0.2))
		strokePath(dst, chamfer(r, 4*s), 1*s, alpha(ed.Col, 0.9))
		drawFont(dst, fTitle, ed.Name, r.cx(), r.cy(), 10*s, light(ed.Col, 0.4), 0.5)
	}
	if dim {
		fillPath(dst, chamfer(full.grow(3.5*s), 12*s), color.RGBA{0x04, 0x02, 0x08, 0xa8})
	}
}

// cardArt draws illustration a centred at (x,y); s is about half its size.
func cardArt(dst *ebiten.Image, a int, x, y, s, t float64, rarity int) {
	if litArt(dst, a, x, y, s/30, t, rarity) {
		return
	}
	k := s / 30
	switch a {
	case aSword:
		swordArt(dst, x, y, s, -0.75, colSteel)
		sparkle(dst, x+s*0.7, y-s*0.6, s*0.25, t)
	case aShield:
		shieldIcon(dst, x, y, s*0.95, rgb(0x3e, 0x8c, 0xf0))
		shape(dst, []float64{x - 3*k, y - 18*k, x + 3*k, y - 18*k, x + 3*k, y + 16*k, x - 3*k, y + 16*k}, colYellow, colOrange, 1.5*k)
		shape(dst, []float64{x - 14*k, y - 5*k, x + 14*k, y - 5*k, x + 14*k, y + 1*k, x - 14*k, y + 1*k}, colYellow, colOrange, 1.5*k)
	case aDagger:
		swordArt(dst, x, y+4*k, s*0.7, -0.9, colSteel)
		for i := 0; i < 3; i++ {
			yy := y + float64(i-1)*10*k
			line(dst, x-s*1.1, yy+8*k, x-s*0.55, yy+4*k, 2.5*k, alpha(colWhite, 0.7))
		}
	case aTwin:
		swordArt(dst, x, y, s*0.9, -0.75, colSteel)
		swordArt(dst, x, y, s*0.9, 0.75, colSteel)
		glow(dst, x, y-6*k, 18*k, colYellow, 0.8)
		sparkle(dst, x, y-6*k, s*0.3, t)
	case aDart:
		dart := rot([]float64{x - 26*k, y, x + 14*k, y - 4*k, x + 26*k, y, x + 14*k, y + 4*k}, x, y, -0.5)
		shape(dst, dart, rgb(0xc8, 0xd0, 0xe0), rgb(0x6a, 0x7a, 0x98), 1.8*k)
		tip := rot([]float64{x + 14*k, y - 5*k, x + 30*k, y, x + 14*k, y + 5*k}, x, y, -0.5)
		shape(dst, tip, rgb(0x9a, 0xff, 0x60), rgb(0x30, 0xa0, 0x20), 1.8*k)
		fl := rot([]float64{x - 26*k, y, x - 34*k, y - 9*k, x - 18*k, y}, x, y, -0.5)
		shape(dst, fl, colRed, shade(colRed, 0.6), 1.5*k)
		for i := 0; i < 2; i++ {
			dy := math.Mod(t*20+float64(i)*12, 24) * k
			drop(dst, x+16*k+float64(i)*6*k, y+6*k+dy, 4*k, colGreen)
		}
	case aBrace:
		orb(dst, x, y, 24*k, 24*k, rgb(0xa8, 0x6a, 0x38))
		disc(dst, x, y, 17*k, 17*k, shade(rgb(0xa8, 0x6a, 0x38), 0.8))
		orb(dst, x, y, 7*k, 7*k, colSteel)
		for _, d := range []float64{-1, 1} {
			curve(dst, x+d*30*k, y-14*k, x+d*38*k, y, x+d*30*k, y+14*k, 3*k, alpha(colWhite, 0.8))
		}
	case aHammer:
		h := rot([]float64{x - 3*k, y + 30*k, x - 3*k, y - 10*k, x + 3*k, y - 10*k, x + 3*k, y + 30*k}, x, y, 0.5)
		shape(dst, h, rgb(0xb0, 0x70, 0x40), rgb(0x60, 0x34, 0x18), 1.8*k)
		hd := rot([]float64{x - 20*k, y - 26*k, x + 20*k, y - 26*k, x + 20*k, y - 8*k, x - 20*k, y - 8*k}, x, y, 0.5)
		shape(dst, hd, rgb(0xd8, 0xe0, 0xf0), rgb(0x6a, 0x76, 0x90), 2*k)
		burst(dst, x-22*k, y-22*k, 14*k, colYellow, t)
	case aAxe:
		h := rot([]float64{x - 3*k, y + 32*k, x - 3*k, y - 28*k, x + 3*k, y - 28*k, x + 3*k, y + 32*k}, x, y, -0.4)
		shape(dst, h, rgb(0xb0, 0x70, 0x40), rgb(0x60, 0x34, 0x18), 1.8*k)
		p := &vector.Path{}
		bx, by := x+6*k, y-14*k
		p.MoveTo(float32(bx), float32(by-12*k))
		p.QuadTo(float32(bx+34*k), float32(by-20*k), float32(bx+28*k), float32(by+12*k))
		p.QuadTo(float32(bx+20*k), float32(by+8*k), float32(bx), float32(by+8*k))
		p.Close()
		strokePath(dst, p, 3.5*k, colInk)
		gradPath(dst, p, colWhite, rgb(0x7a, 0x86, 0xa0))
	case aTower:
		c := rgb(0x4a, 0x7a, 0xd8)
		if rarity >= rRare {
			c = rgb(0x8a, 0x5a, 0xe8)
		}
		roundRect(dst, rect{x - 19*k, y - 29*k, 38 * k, 58 * k}, 9*k, colInk)
		roundGrad(dst, rect{x - 16*k, y - 26*k, 32 * k, 52 * k}, 7*k, light(c, 0.35), shade(c, 0.6))
		roundGrad(dst, rect{x - 12*k, y - 22*k, 24 * k, 10 * k}, 4*k, alpha(colWhite, 0.4), clear)
		for _, p := range [][2]float64{{-10, -18}, {10, -18}, {-10, 18}, {10, 18}} {
			orb(dst, x+p[0]*k, y+p[1]*k, 2.5*k, 2.5*k, colGold)
		}
		orb(dst, x, y, 8*k, 8*k, colGold)
	case aHorn:
		p := &vector.Path{}
		p.MoveTo(float32(x-24*k), float32(y+6*k))
		p.QuadTo(float32(x-10*k), float32(y+24*k), float32(x+20*k), float32(y-2*k))
		p.LineTo(float32(x+26*k), float32(y-22*k))
		p.QuadTo(float32(x+6*k), float32(y+8*k), float32(x-20*k), float32(y-1*k))
		p.Close()
		strokePath(dst, p, 3.5*k, colInk)
		gradPath(dst, p, rgb(0xff, 0xf0, 0xd0), rgb(0xa8, 0x70, 0x40))
		orb(dst, x+24*k, y-12*k, 6*k, 11*k, rgb(0xd8, 0xa8, 0x50))
		for i := 1; i <= 3; i++ {
			r := float64(i) * 8 * k
			curve(dst, x+28*k+r*0.5, y-24*k-r*0.6, x+34*k+r, y-12*k, x+28*k+r*0.5, y+r*0.5-4*k, 2.5*k, alpha(colWhite, 0.8-float64(i)*0.2))
		}
	case aBook:
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x, y - 14*k, x + d*28*k, y - 20*k, x + d*28*k, y + 16*k, x, y + 22*k}, colCream, rgb(0xe0, 0xc8, 0x98), 2*k)
			for i := 0; i < 3; i++ {
				ly := y - 8*k + float64(i)*8*k
				line(dst, x+d*6*k, ly, x+d*22*k, ly-3*k, 1.6*k, alpha(rgb(0x6a, 0x4a, 0xa0), 0.7))
			}
		}
		glow(dst, x, y-10*k, 30*k, rgb(0x9a, 0x7a, 0xff), 0.7)
		sparkle(dst, x-14*k, y-26*k, 6*k, t)
		sparkle(dst, x+16*k, y-30*k, 5*k, t+1)
	case aFist:
		flames(dst, x, y-4*k, 22*k, t)
		orb(dst, x, y+4*k, 20*k, 18*k, rgb(0xff, 0xc0, 0x90))
		for i := 0; i < 4; i++ {
			orb(dst, x-12*k+float64(i)*8*k, y-10*k, 4.5*k, 5*k, rgb(0xff, 0xc8, 0x98))
		}
	case aFire:
		glow(dst, x, y, 40*k, colOrange, 0.8)
		flames(dst, x+6*k, y, 20*k, t)
		radial(dst, x-4*k, y+4*k, 18*k, 18*k, -0.2, -0.2, rgb(0xff, 0xf4, 0xa0), rgb(0xff, 0x6a, 0x10))
		radial(dst, x-6*k, y+2*k, 9*k, 9*k, 0, 0, colWhite, rgb(0xff, 0xe0, 0x60))
	case aCloud:
		c := rgb(0x8a, 0xe8, 0x4a)
		if rarity >= rEpic {
			c = rgb(0xb0, 0x60, 0xe8)
		}
		for i, b := range [][3]float64{{-16, 4, 14}, {14, 4, 15}, {0, -8, 17}, {-4, 10, 13}} {
			r := b[2] * k * (1 + 0.05*math.Sin(t*3+float64(i)))
			orb(dst, x+b[0]*k, y+b[1]*k, r, r*0.9, c)
		}
		disc(dst, x-6*k, y-2*k, 4*k, 5*k, colInk)
		disc(dst, x+6*k, y-2*k, 4*k, 5*k, colInk)
		shape(dst, []float64{x - 6*k, y + 8*k, x + 6*k, y + 8*k, x, y + 4*k}, colInk, colInk, 0)
	case aBolt:
		glow(dst, x, y, 36*k, colYellow, 0.8)
		boltIcon(dst, x, y, 26*k, colYellow)
		sparkle(dst, x+16*k, y-20*k, 6*k, t)
	case aFang:
		drop(dst, x, y+4*k, 16*k, colRed)
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*26*k, y - 22*k, x + d*14*k, y - 22*k, x + d*20*k, y + 2*k}, colWhite, rgb(0xd8, 0xd0, 0xc8), 2*k)
		}
	case aMeteor:
		for i := 0; i < 3; i++ {
			o := float64(i) * 7 * k
			shape(dst, []float64{x - 4*k + o, y - 8*k - o, x + 34*k + o, y - 36*k - o, x + 8*k + o, y + 6*k - o}, alpha(colOrange, 0.8-float64(i)*0.2), alpha(colYellow, 0.2), 0)
		}
		glow(dst, x-4*k, y+6*k, 34*k, colOrange, 0.7)
		orb(dst, x-6*k, y+8*k, 20*k, 19*k, rgb(0x8a, 0x6a, 0x5a))
		for _, c := range [][3]float64{{-12, 4, 4}, {0, 14, 3}, {-2, 0, 3}} {
			disc(dst, x+c[0]*k, y+c[1]*k, c[2]*k, c[2]*k, rgb(0xff, 0x8a, 0x30))
		}
	case aPhoenix:
		glow(dst, x, y, 40*k, colOrange, 0.7)
		flap := math.Sin(t*5) * 6 * k
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*4*k, y - 2*k, x + d*36*k, y - 26*k - flap, x + d*30*k, y - 6*k, x + d*38*k, y - 2*k - flap*0.5, x + d*16*k, y + 10*k}, rgb(0xff, 0xe0, 0x50), rgb(0xff, 0x40, 0x20), 2*k)
		}
		shape(dst, []float64{x - 6*k, y + 6*k, x + 6*k, y + 6*k, x + 10*k, y + 30*k, x, y + 22*k, x - 10*k, y + 30*k}, rgb(0xff, 0xa0, 0x30), rgb(0xd8, 0x20, 0x20), 2*k)
		orb(dst, x, y, 9*k, 12*k, rgb(0xff, 0x8a, 0x20))
		orb(dst, x, y-14*k, 7*k, 7*k, rgb(0xff, 0xb0, 0x30))
		shape(dst, []float64{x + 5*k, y - 15*k, x + 13*k, y - 13*k, x + 5*k, y - 11*k}, colYellow, colOrange, 1*k)
	case aExcalibur:
		for i := 0; i < 10; i++ {
			a := float64(i)*math.Pi/5 + t*0.6
			shape(dst, []float64{x, y - 4*k, x + math.Cos(a)*60*k, y - 4*k + math.Sin(a)*60*k, x + math.Cos(a+0.15)*60*k, y - 4*k + math.Sin(a+0.15)*60*k}, alpha(colYellow, 0.35), clear, 0)
		}
		glow(dst, x, y-4*k, 34*k, colYellow, 0.9)
		swordArt(dst, x, y, s*1.05, 0, rgb(0xff, 0xf0, 0xa0))
		sparkle(dst, x, y-30*k, 8*k, t)
	case aHourglass:
		roundGrad(dst, rect{x - 18*k, y - 28*k, 36 * k, 7 * k}, 3*k, colYellow, colOrange)
		roundGrad(dst, rect{x - 18*k, y + 21*k, 36 * k, 7 * k}, 3*k, colYellow, colOrange)
		shape(dst, []float64{x - 13*k, y - 21*k, x + 13*k, y - 21*k, x + 2*k, y, x + 13*k, y + 21*k, x - 13*k, y + 21*k, x - 2*k, y}, alpha(rgb(0xd8, 0xf0, 0xff), 0.7), alpha(rgb(0x9a, 0xc8, 0xf0), 0.7), 2*k)
		f := math.Mod(t*0.3, 1)
		shape(dst, []float64{x - 10*k*(1-f), y - 18*k + 16*k*f, x + 10*k*(1-f), y - 18*k + 16*k*f, x, y - 1*k}, rgb(0xff, 0xd8, 0x70), rgb(0xe8, 0xa0, 0x30), 0)
		shape(dst, []float64{x - 11*k*f, y + 19*k - 14*k*f, x + 11*k*f, y + 19*k - 14*k*f, x + 12*k, y + 19*k, x - 12*k, y + 19*k}, rgb(0xff, 0xd8, 0x70), rgb(0xe8, 0xa0, 0x30), 0)
		line(dst, x, y, x, y+19*k, 1.5*k, rgb(0xff, 0xd8, 0x70))
		glow(dst, x, y, 30*k, rgb(0x9a, 0xd0, 0xff), 0.5)
	case aAegis:
		glow(dst, x, y, 40*k, colYellow, 0.7)
		for _, d := range []float64{-1, 1} {
			for i := 0; i < 3; i++ {
				o := float64(i) * 7 * k
				shape(dst, []float64{x + d*14*k, y - 12*k + o, x + d*(40*k-o), y - 24*k + o*1.2, x + d*(34*k-o), y - 4*k + o, x + d*14*k, y + 2*k + o}, colWhite, rgb(0xd8, 0xe8, 0xff), 1.5*k)
			}
		}
		shieldIcon(dst, x, y, 22*k, colGold)
		sparkle(dst, x+10*k, y-14*k, 6*k, t)
	case aDragon:
		shape(dst, []float64{x + 4*k, y - 4*k, x + 40*k, y - 22*k, x + 46*k, y + 10*k, x + 6*k, y + 8*k}, alpha(colOrange, 0.9), alpha(colYellow, 0.4), 0)
		glow(dst, x+28*k, y-2*k, 26*k, colOrange, 0.8)
		head := []float64{x - 30*k, y - 10*k, x - 10*k, y - 22*k, x + 8*k, y - 10*k, x + 10*k, y + 4*k, x - 4*k, y + 6*k, x - 28*k, y + 14*k}
		shape(dst, head, rgb(0x6a, 0xd8, 0x6a), rgb(0x1a, 0x7a, 0x3a), 2.4*k)
		shape(dst, []float64{x - 14*k, y - 18*k, x - 26*k, y - 36*k, x - 6*k, y - 20*k}, colCream, rgb(0xc8, 0xb0, 0x88), 1.8*k)
		disc(dst, x-6*k, y-10*k, 4*k, 3*k, colInk)
		circle(dst, x-6*k, y-11*k, 2*k, colYellow)
		shape(dst, []float64{x + 2*k, y + 4*k, x + 6*k, y + 4*k, x + 4*k, y + 10*k}, colWhite, colCream, 1*k)
	}
}

func swordArt(dst *ebiten.Image, x, y, s, ang float64, bladeCol color.RGBA) {
	k := s / 30
	blade := rot([]float64{x - 4*k, y + 14*k, x - 4*k, y - 28*k, x, y - 34*k, x + 4*k, y - 28*k, x + 4*k, y + 14*k}, x, y, ang)
	shape(dst, blade, light(bladeCol, 0.6), shade(bladeCol, 0.7), 2*k)
	rg := rot([]float64{x, y + 12*k, x, y - 30*k}, x, y, ang)
	line(dst, rg[0], rg[1], rg[2], rg[3], 1.2*k, alpha(colWhite, 0.8))
	guard := rot([]float64{x - 13*k, y + 13*k, x + 13*k, y + 13*k, x + 13*k, y + 19*k, x - 13*k, y + 19*k}, x, y, ang)
	shape(dst, guard, colYellow, colOrange, 1.8*k)
	grip := rot([]float64{x - 3*k, y + 19*k, x + 3*k, y + 19*k, x + 3*k, y + 30*k, x - 3*k, y + 30*k}, x, y, ang)
	shape(dst, grip, rgb(0x9a, 0x5a, 0x30), rgb(0x5a, 0x30, 0x18), 1.8*k)
	pm := rot([]float64{x, y + 33*k}, x, y, ang)
	orb(dst, pm[0], pm[1], 4*k, 4*k, colGold)
}

func sparkle(dst *ebiten.Image, x, y, r, t float64) {
	f := 0.6 + 0.4*math.Sin(t*5+x)
	r *= f
	shape(dst, []float64{x, y - r, x + r*0.22, y - r*0.22, x + r, y, x + r*0.22, y + r*0.22, x, y + r, x - r*0.22, y + r*0.22, x - r, y, x - r*0.22, y - r*0.22}, colWhite, rgb(0xff, 0xf0, 0xb0), 0)
	glow(dst, x, y, r*1.4, colWhite, 0.4*f)
}

func drop(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	p := &vector.Path{}
	p.MoveTo(float32(x), float32(y-r*1.6))
	p.QuadTo(float32(x+r*1.1), float32(y-r*0.3), float32(x+r), float32(y+r*0.2))
	p.Arc(float32(x), float32(y+r*0.2), float32(r), 0, math.Pi, vector.Clockwise)
	p.QuadTo(float32(x-r*1.1), float32(y-r*0.3), float32(x), float32(y-r*1.6))
	p.Close()
	strokePath(dst, p, 2.5, colInk)
	gradPath(dst, p, light(c, 0.4), shade(c, 0.65))
	radial(dst, x-r*0.35, y-r*0.1, r*0.25, r*0.35, 0, 0, alpha(colWhite, 0.7), clear)
}

func flames(dst *ebiten.Image, x, y, r, t float64) {
	for i, c := range []color.RGBA{rgb(0xff, 0x4a, 0x1a), rgb(0xff, 0x9a, 0x20), rgb(0xff, 0xe0, 0x60)} {
		f := 1 - float64(i)*0.28
		w := math.Sin(t*8+float64(i)) * r * 0.12
		shape(dst, []float64{x - r*f, y + r*0.4, x - r*0.6*f + w, y - r*0.5*f, x - r*0.2*f, y - r*0.1*f, x + w*1.5, y - r*1.5*f, x + r*0.3*f, y - r*0.2*f, x + r*0.7*f - w, y - r*0.7*f, x + r*f, y + r*0.4}, light(c, 0.2), c, 0)
	}
}

func burst(dst *ebiten.Image, x, y, r float64, c color.RGBA, t float64) {
	for i := 0; i < 8; i++ {
		a := float64(i)*math.Pi/4 + 0.2
		l := r * (0.8 + 0.3*math.Sin(t*8+float64(i)))
		line(dst, x+math.Cos(a)*r*0.4, y+math.Sin(a)*r*0.4, x+math.Cos(a)*l*1.4, y+math.Sin(a)*l*1.4, 2.5, alpha(c, 0.9))
	}
}

// charmIcon draws a charm as an amulet: a gem in a gold hexagon.
func charmIcon(dst *ebiten.Image, id string, x, y, r, t float64) {
	ch := charmByID(id)
	glow(dst, x, y, r*1.8, ch.Col, 0.3+0.1*math.Sin(t*3+x))
	hex := func(k float64) []float64 {
		var p []float64
		for i := 0; i < 6; i++ {
			a := float64(i)*math.Pi/3 - math.Pi/2
			p = append(p, x+math.Cos(a)*r*k, y+math.Sin(a)*r*k)
		}
		return p
	}
	shape(dst, hex(1), colGoldHi, colGoldLo, 1.2)
	shape(dst, hex(0.72), light(ch.Col, 0.3), shade(ch.Col, 0.3), 0)
	shape(dst, []float64{x, y - r*0.72, x + r*0.62, y - r*0.36, x, y}, alpha(colWhite, 0.4), alpha(colWhite, 0.05), 0)
	drawFont(dst, fTitle, string(ch.Name[0]), x, y+r*0.05, r*0.8, alpha(colWhite, 0.85), 0.5)
}

// nodeIcon draws a map node's symbol.
func nodeIcon(dst *ebiten.Image, kind int, x, y, r, t float64) {
	k := r / 30
	switch kind {
	case nFight:
		swordArt(dst, x, y, 24*k, -0.75, colSteel)
		swordArt(dst, x, y, 24*k, 0.75, colSteel)
	case nElite:
		orb(dst, x, y+2*k, 20*k, 18*k, rgb(0xd8, 0xd0, 0xc0))
		for _, d := range []float64{-1, 1} {
			shape(dst, []float64{x + d*12*k, y - 10*k, x + d*26*k, y - 30*k, x + d*20*k, y - 6*k}, colCream, rgb(0xa0, 0x90, 0x70), 1.5*k)
			disc(dst, x+d*7*k, y, 5*k, 6*k, colInk)
			glow(dst, x+d*7*k, y, 6*k, colRed, 0.9)
		}
	case nShop:
		orb(dst, x, y+6*k, 20*k, 18*k, rgb(0xb0, 0x7a, 0x40))
		shape(dst, []float64{x - 9*k, y - 12*k, x + 9*k, y - 12*k, x + 5*k, y - 4*k, x - 5*k, y - 4*k}, rgb(0xb0, 0x7a, 0x40), rgb(0x7a, 0x4a, 0x20), 1.5*k)
		coinIcon(dst, x, y+8*k, 9*k)
	case nRest:
		glow(dst, x, y, 30*k, colOrange, 0.6)
		line(dst, x-16*k, y+14*k, x+16*k, y+6*k, 5*k, rgb(0x7a, 0x4a, 0x20))
		line(dst, x+16*k, y+14*k, x-16*k, y+6*k, 5*k, rgb(0x6a, 0x3a, 0x18))
		flames(dst, x, y+6*k, 16*k, t)
	case nEvent:
		glow(dst, x, y, 22*k, colBlue, 0.5)
		say(dst, "?", x, y, 34*k, rgb(0x9a, 0xd8, 0xff), 0.5)
	case nTreasure:
		chestIcon(dst, x, y+6*k, 0.34*k, false)
	case nBoss:
		cr := []float64{x - 22*k, y + 10*k, x - 26*k, y - 18*k, x - 11*k, y - 4*k, x, y - 24*k, x + 11*k, y - 4*k, x + 26*k, y - 18*k, x + 22*k, y + 10*k}
		shape(dst, cr, rgb(0xff, 0xf0, 0x80), rgb(0xd8, 0x8a, 0x10), 2.5*k)
		orb(dst, x, y+2*k, 4*k, 4*k, colRed)
	}
}

// eventArt draws the scene for mystery event e centred at (x,y).
func eventArt(dst *ebiten.Image, e int, x, y, t float64) {
	switch e {
	case 0: // wishing well
		shadow(dst, x, y+70, 110, 18, 0.4)
		for _, d := range []float64{-1, 1} {
			roundRect(dst, rect{x + d*70 - 7, y - 90, 14, 130}, 4, colInk)
			roundGrad(dst, rect{x + d*70 - 5, y - 88, 10, 126}, 3, rgb(0x4a, 0x30, 0x1e), rgb(0x1a, 0x10, 0x08))
		}
		shape(dst, []float64{x - 96, y - 74, x, y - 120, x + 96, y - 74, x + 86, y - 66, x - 86, y - 66}, rgb(0x3a, 0x1e, 0x1a), rgb(0x12, 0x08, 0x08), 3)
		line(dst, x, y-70, x, y-20, 2, colInk)
		roundRect(dst, rect{x - 12, y - 28, 24, 20}, 4, colInk)
		roundGrad(dst, rect{x - 10, y - 26, 20, 16}, 3, rgb(0x4a, 0x34, 0x22), rgb(0x1e, 0x12, 0x0a))
		radial(dst, x, y+14, 88, 30, 0, 0, colInk, colInk)
		radial(dst, x, y+14, 82, 25, 0, -0.3, rgb(0x2a, 0x8a, 0x9a), rgb(0x04, 0x14, 0x1e))
		for i := 0; i < 5; i++ {
			gx := x - 50 + float64(i)*25
			sparkle(dst, gx, y+14+math.Sin(float64(i))*8, 6, t+float64(i))
		}
		for i := 0; i < 11; i++ {
			a := float64(i)/11*math.Pi + 0.05
			orb(dst, x-math.Cos(a)*88, y+30+math.Sin(a)*26, 15, 12, rgb(0x4a, 0x48, 0x54))
		}
	case 1: // blood altar
		glow(dst, x, y-10, 140, colRed, 0.45+0.1*math.Sin(t*3))
		roundRect(dst, rect{x - 92, y + 6, 184, 60}, 10, colInk)
		roundGrad(dst, rect{x - 89, y + 9, 178, 54}, 8, rgb(0x4a, 0x48, 0x54), rgb(0x16, 0x14, 0x1a))
		roundRect(dst, rect{x - 110, y - 10, 220, 22}, 8, colInk)
		roundGrad(dst, rect{x - 107, y - 7, 214, 16}, 6, rgb(0x5a, 0x58, 0x64), rgb(0x22, 0x20, 0x28))
		for i := 0; i < 3; i++ {
			rx := x - 50 + float64(i)*50
			glow(dst, rx, y+36, 18, colRed, 0.8)
			line(dst, rx-8, y+28, rx+8, y+44, 3, rgb(0xff, 0x50, 0x50))
			line(dst, rx+8, y+28, rx-8, y+44, 3, rgb(0xff, 0x50, 0x50))
		}
		orb(dst, x, y-40, 30, 27, rgb(0xf0, 0xea, 0xd8))
		for _, d := range []float64{-1, 1} {
			disc(dst, x+d*11, y-42, 8, 9, colInk)
			glow(dst, x+d*11, y-42, 9, colRed, 0.9)
			roundRect(dst, rect{x + d*86 - 7, y - 46, 14, 36}, 4, colInk)
			roundGrad(dst, rect{x + d*86 - 5, y - 44, 10, 32}, 3, colCream, rgb(0xd8, 0xc0, 0x98))
			flames(dst, x+d*86, y-46, 8, t+d)
		}
	case 2: // traveling trader
		merchant(dst, x, y+96, t)
	case 3: // fairy spring
		radial(dst, x, y+40, 150, 44, 0, 0, colInk, colInk)
		radial(dst, x, y+40, 144, 40, 0, -0.4, rgb(0x3a, 0x9a, 0x9a), rgb(0x04, 0x18, 0x22))
		for i := 0; i < 3; i++ {
			f := math.Mod(t*0.5+float64(i)/3, 1)
			strokeCircle(dst, x, y+40, 20+f*110, 3*(1-f), alpha(colWhite, 0.6*(1-f)))
		}
		for _, lp := range [][2]float64{{-100, 46}, {96, 34}} {
			orb(dst, x+lp[0], y+lp[1], 22, 9, rgb(0x2a, 0x4a, 0x2a))
		}
		fy := y - 50 + math.Sin(t*2)*10
		glow(dst, x, fy, 70, rgb(0xc8, 0xff, 0xd0), 0.8)
		for _, d := range []float64{-1, 1} {
			radial(dst, x+d*16, fy-6, 16, 9, 0, 0, alpha(colWhite, 0.8), alpha(rgb(0xa0, 0xe0, 0xff), 0.4))
		}
		orb(dst, x, fy, 8, 8, rgb(0xff, 0xf8, 0xd0))
	case 4: // gambler's cups
		roundRect(dst, rect{x - 150, y + 30, 300, 40}, 12, colInk)
		roundGrad(dst, rect{x - 147, y + 33, 294, 34}, 10, rgb(0x3a, 0x22, 0x1a), rgb(0x14, 0x0a, 0x06))
		lift := math.Max(0, math.Sin(t*2)) * 26
		coinIcon(dst, x, y+28, 14)
		for i, cx := range []float64{x - 90, x, x + 90} {
			cy := y + 34
			if i == 1 {
				cy -= lift
			}
			shape(dst, []float64{cx - 34, cy, cx - 22, cy - 66, cx + 22, cy - 66, cx + 34, cy}, rgb(0x5a, 0x3a, 0x26), rgb(0x1e, 0x10, 0x08), 3)
			roundGrad(dst, rect{cx - 26, cy - 50, 52, 8}, 3, rgb(0xff, 0xd0, 0x50), rgb(0xc8, 0x8a, 0x10))
		}
	case 5: // old forge
		glow(dst, x, y, 140, colOrange, 0.45)
		shape(dst, []float64{x - 80, y - 20, x + 100, y - 20, x + 70, y + 4, x + 30, y + 4, x + 30, y + 40, x + 56, y + 70, x - 56, y + 70, x - 30, y + 40, x - 30, y + 4, x - 60, y + 4}, rgb(0x4a, 0x4e, 0x5a), rgb(0x12, 0x12, 0x18), 3)
		swordArt(dst, x+10, y-30, 34, 1.57, rgb(0xff, 0xa0, 0x50))
		h := rot([]float64{x + 40, y - 60, x + 46, y - 60, x + 46, y - 130, x + 40, y - 130}, x+43, y-60, 0.8+math.Sin(t*4)*0.3)
		shape(dst, h, rgb(0xb0, 0x70, 0x40), rgb(0x60, 0x34, 0x18), 2)
		burst(dst, x+10, y-30, 30, colYellow, t)
	}
}
