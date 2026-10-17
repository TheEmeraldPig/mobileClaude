package game

import (
	"image/color"
	"math"
)

// The slugs: grumpy, gooey, and in the way. Each is painted facing left with its
// foot at 0,0, in two layers: the body (with eye stalks) and the eyes, which have
// an open and a hurt state.

type slugKind struct {
	name string
	col  color.RGBA
	spot color.RGBA
	gear int // 0 none, 1 helmet, 2 snail shell, 3 back spikes, 4 shades, 5 horns, 6 crown
}

const (
	gearNone = iota
	gearHelmet
	gearShell
	gearSpikes
	gearShades
	gearHorns
	gearCrown
)

var slugKinds = []slugKind{
	{"Slimey", rgb(0x8e, 0xe0, 0x4a), rgb(0x5e, 0xb0, 0x30), gearNone},
	{"Grub", rgb(0xb4, 0x7a, 0xf0), rgb(0x8a, 0x50, 0xd0), gearNone},
	{"Rusty", rgb(0xff, 0x9a, 0x3c), rgb(0xd8, 0x6a, 0x20), gearHelmet},
	{"Shelly", rgb(0x5c, 0xd0, 0xc8), rgb(0x3a, 0xa8, 0xa0), gearShell},
	{"Spike", rgb(0xf0, 0x5a, 0x7a), rgb(0xc0, 0x38, 0x58), gearSpikes},
	{"Cool Slug", rgb(0x6a, 0xa0, 0xff), rgb(0x48, 0x78, 0xe0), gearShades},
	{"Brute", rgb(0xa8, 0xc8, 0x3a), rgb(0x7a, 0x98, 0x20), gearHorns},
}

// bossKinds are the big ones at the end of each stage.
var bossKinds = []slugKind{
	{"King Gloop", rgb(0x9a, 0x5a, 0xe8), rgb(0x6e, 0x38, 0xc0), gearCrown},
	{"Captain Ooze", rgb(0xff, 0x7a, 0x3a), rgb(0xd0, 0x50, 0x20), gearHelmet},
	{"Grand Shellby", rgb(0x3c, 0xc0, 0x9a), rgb(0x24, 0x90, 0x70), gearShell},
	{"Mega Muck", rgb(0x7a, 0xb8, 0x30), rgb(0x58, 0x88, 0x18), gearHorns},
	{"Duchess Drool", rgb(0xff, 0x5a, 0x9a), rgb(0xd0, 0x38, 0x70), gearCrown},
}

func slugBody() sh {
	foot := capsule(-44, -13, 62, -10, 14, 7)
	hump := ell(8, -34, 44, 28)
	neck := capsule(-30, -20, -34, -56, 22, 25)
	head := circ(-34, -62, 27)
	return foot.melt(hump, 14).melt(neck, 12).melt(head, 10)
}

const (
	slugEye1X, slugEye1Y = -52.0, -114.0
	slugEye2X, slugEye2Y = -20.0, -116.0
)

func paintSlug(c *canvas, k slugKind, boss bool) {
	body := slugBody()
	base := k.col
	goo := mat{base: base, gloss: 1.1, shine: 26, rim: 0.65}
	// back eye stalk, then the body
	stalk := func(x0, y0, x1, y1 float64) {
		s := line(4.6, 3.8, x0, y0, (x0+x1)/2+2, (y0+y1)/2, x1, y1)
		c.part(s, goo, 2.4)
	}
	stalk(-24, -80, -21, slugEye2Y+6)
	if k.gear == gearShell {
		sc := circ(18, -54, 34)
		c.part(sc, mat{base: rgb(0xf0, 0xb8, 0x6a), gloss: 0.8, rim: 0.5, tex: 0.1}, 2.6)
		var p []float64
		for i := 0; i < 34; i++ {
			a := float64(i) / 34 * math.Pi * 3.6
			r := 4 + float64(i)*0.82
			p = append(p, 18+math.Cos(a)*r, -54+math.Sin(a)*r)
		}
		c.fill(line(2.2, 2.6, p...).and(sc.grow(-1)), rgb(0xa8, 0x62, 0x30))
	}
	if k.gear == gearSpikes {
		for i := 0; i < 5; i++ {
			x := -12 + float64(i)*14
			y := -58 + math.Abs(float64(i)-1.5)*4
			sp := poly(x-8, y+8, x+1, y-20, x+9, y+8)
			c.part(sp, mat{base: rgb(0xff, 0xe8, 0xd0), gloss: 0.6}, 2)
		}
	}
	c.fill(body.grow(2.8), outlineOf(base))
	c.vol(body, goo)
	for _, s := range [][3]float64{{14, -46, 7}, {30, -30, 5}, {-4, -32, 4.5}, {40, -18, 3.5}} {
		if k.gear == gearShell || k.gear == gearSpikes {
			break
		}
		c.volOn(circ(s[0], s[1], s[2]).and(body), body, mat{base: k.spot, gloss: 0.8, shine: 26})
	}
	// slime shine
	c.shine(ell(-44, -76, 8, 4).rot(-44, -76, -0.6), 0.55)
	c.shine(ell(4, -56, 12, 4).rot(4, -56, -0.2), 0.35)
	stalk(-40, -78, -50, slugEye1Y+6)
	// mouth: a grumpy frown, or a toothy grin for bosses
	if boss {
		m := ell(-50, -46, 13, 8).and(rrect(-70, -46, 40, 20, 2))
		c.fill(m.grow(2), outlineOf(base))
		c.fill(m, rgb(0x5a, 0x16, 0x30))
		c.fill(poly(-58, -46, -54, -46, -56, -40), colWhite)
		c.fill(poly(-47, -46, -43, -46, -45, -41), colWhite)
	} else {
		c.fill(arc(-50, -40, 8, 6, math.Pi*1.15, math.Pi*1.85, 1.8), outlineOf(base))
	}
	c.soft(ell(-32, -48, 6, 3.5), alpha(rgb(0xff, 0x5a, 0x8a), 0.45), 3)
	switch k.gear {
	case gearHelmet:
		hm := ell(-34, -70, 31, 26).and(rrect(-70, -100, 80, 32, 0))
		c.part(hm, mat{base: rgb(0xb8, 0xc0, 0xd8), gloss: 1.3, shine: 40, rim: 0.6}, 2.4)
		c.part(rrect(-68, -74, 68, 8, 4), mat{base: rgb(0x9a, 0xa4, 0xc0), gloss: 1}, 2)
		for _, x := range []float64{-60, -46, -22, -8} {
			c.part(circ(x, -70, 2.2), mat{base: rgb(0xe0, 0xe4, 0xf0), gloss: 1}, 0.8)
		}
		c.part(poly(-40, -94, -34, -108, -28, -94), mat{base: rgb(0xd8, 0xdc, 0xe8), gloss: 1.2}, 2)
	case gearHorns:
		for _, s := range []float64{-1, 1} {
			x := -34 + s*20
			h := line(6, 2, x, -80, x+s*14, -92, x+s*14, -108)
			c.part(h, mat{base: rgb(0xff, 0xf0, 0xd8), gloss: 0.8, rim: 0.4}, 2.2)
		}
		c.part(ell(-34, -84, 26, 9).and(rrect(-64, -92, 60, 9, 0)), mat{base: rgb(0x9a, 0x6a, 0x40), gloss: 0.7, tex: 0.1}, 2.2)
	case gearCrown:
		cr := poly(-56, -82, -60, -104, -48, -94, -36, -112, -24, -94, -12, -104, -14, -82)
		c.part(cr, mat{base: rgb(0xff, 0xd0, 0x30), gloss: 1.4, shine: 36, flat: 0.25}, 2.4)
		c.part(circ(-36, -92, 3.6), mat{base: rgb(0x40, 0xd0, 0xff), gloss: 1}, 1.2)
		for _, x := range []float64{-60, -36, -12} {
			c.part(circ(x, -106+math.Abs(x+36)*0.15, 2.6), mat{base: rgb(0xff, 0x40, 0x70), gloss: 1}, 1)
		}
	}
}

// paintSlugEyes paints the eyeballs on the stalks: state 0 open, 3 hurt; shades go on top.
func paintSlugEyes(c *canvas, k slugKind, boss bool, state int) {
	skin := k.col
	for _, e := range [][2]float64{{slugEye2X, slugEye2Y}, {slugEye1X, slugEye1Y}} {
		paintEye(c, e[0], e[1], 11, -1, state, skin)
	}
	if state == 0 && (boss || k.gear == gearHorns) { // angry brows
		out := rgb(0x24, 0x14, 0x36)
		c.fill(line(2.4, 1.8, slugEye1X-10, slugEye1Y-16, slugEye1X+10, slugEye1Y-10), out)
		c.fill(line(1.8, 2.4, slugEye2X-10, slugEye2Y-10, slugEye2X+10, slugEye2Y-16), out)
	}
	if k.gear == gearShades {
		g := mat{base: rgb(0x2a, 0x24, 0x40), gloss: 1.4, shine: 30, flat: 0.4}
		lens := rrect(slugEye1X-13, slugEye1Y-9, 24, 17, 7).or(rrect(slugEye2X-11, slugEye2Y-9, 24, 17, 7))
		c.part(lens, g, 2)
		c.fill(line(1.6, 1.6, slugEye1X+10, slugEye1Y-3, slugEye2X-10, slugEye2Y-3), rgb(0x2a, 0x24, 0x40))
		c.shine(ell(slugEye1X-6, slugEye1Y-4, 5, 2.4).rot(slugEye1X-6, slugEye1Y-4, -0.5), 0.6)
		c.shine(ell(slugEye2X-4, slugEye2Y-4, 5, 2.4).rot(slugEye2X-4, slugEye2Y-4, -0.5), 0.6)
	}
}

type slugSprite struct {
	body *pimg
	face [2]*pimg // open, hurt
}

func slugArt(kind int, boss bool, z float64) *slugSprite {
	k := slugKinds[kind%len(slugKinds)]
	key := "slug" + itoa(kind)
	if boss {
		k = bossKinds[kind%len(bossKinds)]
		key = "boss" + itoa(kind)
	}
	key += "@" + num(z*100)
	s := &slugSprite{}
	s.body = paintArt(key, -84, -126, 80, 6, z, func(c *canvas) { paintSlug(c, k, boss) })
	for i, st := range []int{0, 3} {
		st := st
		s.face[i] = paintArt(key+"f"+itoa(st), -72, -136, 0, -96, z, func(c *canvas) { paintSlugEyes(c, k, boss, st) })
	}
	return s
}

func slugName(kind int, boss bool) string {
	if boss {
		return bossKinds[kind%len(bossKinds)].name
	}
	return slugKinds[kind%len(slugKinds)].name
}
