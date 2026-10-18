package game

import (
	"image/color"
	"math"
)

// The rotten food monsters. Each is painted standing with its feet at 0,0, in two
// layers: the body and the face (angry, or hurt). Each one drops ingredients.

type monster struct {
	name     string
	drop     []int // ingredients it drops
	col      color.RGBA
	e1x, e1y float64
	e2x, e2y float64
	er       float64
	mx, my   float64
	mw       float64
}

var monsters = []monster{
	{"Moldy Bread", []int{inFlour}, rgb(0x8a, 0xc8, 0x40), -22, -112, 22, -112, 15, 0, -70, 22},
	{"Rotten Tomato", []int{inTomato}, colTomato, -24, -96, 24, -96, 15, 0, -60, 22},
	{"Evil Broccoli", []int{inVeggie}, rgb(0x3a, 0xa0, 0x38), -14, -86, 14, -86, 12, 0, -56, 16},
	{"Stinky Cheese", []int{inCheese}, colCheese, -30, -84, 6, -88, 14, -12, -48, 20},
	{"Mean Meatball", []int{inMeat}, colMeat, -24, -92, 24, -92, 15, 0, -56, 22},
	{"Cracked Egg", []int{inEgg}, rgb(0xff, 0xe0, 0x60), -20, -96, 20, -96, 14, 0, -60, 18},
	{"Gummy Brute", []int{inSugar}, rgb(0xff, 0x50, 0x70), -16, -146, 16, -146, 11, 0, -122, 14},
	{"Hot Pepper", []int{inSpice}, rgb(0xe8, 0x28, 0x20), -8, -112, 20, -108, 12, 6, -78, 14},
	{"Angry Onion", []int{inVeggie, inSpice}, rgb(0xb0, 0x60, 0xc0), -20, -82, 20, -82, 14, 0, -46, 18},
	{"Burnt Toast", []int{inFlour, inSpice}, rgb(0x5a, 0x3a, 0x2a), -22, -112, 22, -112, 15, 0, -70, 22},
}

var bosses = []monster{
	{"Burger Boss", []int{inMeat, inFlour, inCheese, inVeggie}, colMeat, -32, -170, 32, -170, 18, 0, -134, 28},
	{"Cake Monster", []int{inSugar, inEgg, inFlour}, colPink, -34, -78, 34, -78, 17, 0, -38, 26},
	{"Donut King", []int{inSugar, inFlour, inEgg}, colPink, -46, -160, 46, -160, 17, 0, -48, 28},
	{"Hot Dog Hulk", []int{inMeat, inFlour, inSpice}, rgb(0xc8, 0x50, 0x3a), -14, -170, 16, -170, 13, 0, -134, 16},
	{"Pizza Tyrant", []int{inCheese, inTomato, inFlour}, colCheese, -30, -176, 30, -176, 17, 0, -126, 26},
}

// Each area has its own monster family.
var areaMonsters = [][]int{{0, 5, 3, 9}, {1, 2, 8}, {4, 7, 9}, {6, 5, 0}, {3, 1, 4, 2}, {0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}

func monLeg(c *canvas, col color.RGBA, x float64) {
	s := capsule(x, -30, x, -8, 9, 8).or(ell(x+4, -6, 15, 8))
	c.part(s, mat{base: col, rim: 0.3}, 2.4)
}

func monArm(c *canvas, col color.RGBA, sx, sy, hx, hy float64) {
	s := line(7, 6, sx, sy, (sx+hx)/2, (sy+hy)/2+6, hx, hy).or(circ(hx, hy, 10))
	c.part(s, mat{base: col, rim: 0.3}, 2.4)
}

func paintMonster(c *canvas, id int, boss bool) {
	ow := 2.6
	P := func(s sh, col color.RGBA) { c.part(s, mat{base: col, rim: 0.4, gloss: 0.4}, ow) }
	limb := rgb(0x6a, 0x40, 0x30)
	if boss {
		switch id {
		case 0: // Burger Boss
			monLeg(c, colBread, -44)
			monLeg(c, colBread, 44)
			monArm(c, colBread, -88, -70, -122, -40)
			monArm(c, colBread, 88, -70, 124, -100)
			P(ell(0, -40, 92, 28).and(rrect(-100, -50, 200, 50, 0)).or(rrect(-92, -52, 184, 14, 6)), colBread)
			P(rrect(-100, -92, 200, 44, 22).tex(), rgb(0x6a, 0x34, 0x1c))
			c.part(poly(-96, -94, 96, -94, 70, -70, 50, -86, 10, -66, -30, -86, -60, -70), mat{base: colCheese, rim: 0.3}, ow)
			c.part(rrect(-104, -104, 208, 14, 7).fuzz(3, 0.12), mat{base: colLeaf}, ow)
			for _, x := range []float64{-50, 10, 60} {
				c.part(ell(x, -110, 28, 8), mat{base: colTomato, gloss: 0.6}, ow)
			}
			top := ell(0, -112, 96, 98).and(rrect(-110, -230, 220, 116, 0))
			P(top, colBread)
			for _, s := range [][2]float64{{-50, -170}, {-20, -196}, {16, -186}, {54, -164}, {-62, -132}, {66, -130}, {0, -150}} {
				c.part(ell(s[0], s[1], 6, 3.5).rot(s[0], s[1], s[0]*0.01), mat{base: rgb(0xff, 0xf6, 0xe0)}, 1.4)
			}
		case 1: // Cake Monster
			monLeg(c, limb, -40)
			monLeg(c, limb, 40)
			monArm(c, rgb(0xff, 0xf0, 0xe0), -92, -70, -126, -110)
			monArm(c, rgb(0xff, 0xf0, 0xe0), 92, -70, 126, -40)
			low := rrect(-96, -124, 192, 100, 16)
			P(low, rgb(0x8a, 0x48, 0x2a))
			c.part(blob(-98, -126, 98, -126, 98, -92, 80, -78, 60, -98, 30, -80, 0, -96, -30, -78, -60, -96, -84, -80, -98, -92), mat{base: colPink, rim: 0.4}, ow)
			up := rrect(-62, -196, 124, 74, 14)
			P(up, rgb(0xff, 0xf0, 0xe0))
			c.part(blob(-64, -198, 64, -198, 64, -176, 40, -164, 20, -180, -10, -166, -40, -180, -64, -170), mat{base: colPink, rim: 0.4}, ow)
			for _, x := range []float64{-34, 0, 34} {
				c.part(rrect(x-5, -232, 10, 34, 3), mat{base: rgb(0x8a, 0xd0, 0xff), stripe: func(_, y float64) float64 { return 0.8 + 0.2*math.Sin(y*0.6) }}, ow*0.7)
				c.glowIn(ell(x, -244, 6, 9), rgb(0xff, 0xd0, 0x40), 10, 0.5)
				c.part(ell(x, -244, 6, 9), mat{base: rgb(0xff, 0xc0, 0x30), flat: 0.6}, ow*0.6)
			}
		case 2: // Donut King
			monLeg(c, colBread, -40)
			monLeg(c, colBread, 40)
			monArm(c, colBread, -88, -100, -126, -70)
			monArm(c, colBread, 88, -100, 126, -140)
			ring := circ(0, -110, 92).minus(circ(0, -110, 28))
			P(ring, colBread)
			icing := ring.and(blob(-100, -130, -86, -200, 0, -220, 86, -200, 100, -130, 80, -90, 60, -110, 34, -70, 0, -96, -34, -70, -60, -110, -84, -86))
			c.volOn(icing, ring, mat{base: colPink, rim: 0.4, gloss: 0.6})
			c.fill(icing.ring(1.6), outlineOf(colPink))
			for i := 0; i < 22; i++ {
				a := float64(i)*2.4 + 0.3
				r := 40 + math.Mod(float64(i)*17, 44)
				x, y := math.Cos(a)*r, -110+math.Sin(a)*r
				if y > -110+math.Sin(a)*10 && y > -100 {
					continue
				}
				cols := []color.RGBA{colWhite, rgb(0x60, 0xd0, 0xff), colGold, rgb(0x8c, 0xe0, 0x50)}
				c.part(capsule(x-3*math.Cos(a+1), y-3*math.Sin(a+1), x+3*math.Cos(a+1), y+3*math.Sin(a+1), 2, 2), mat{base: cols[i%4]}, 1)
			}
			cr := poly(-40, -196, -48, -236, -24, -214, 0, -246, 24, -214, 48, -236, 40, -196)
			c.part(cr, mat{base: rgb(0xff, 0xc8, 0x30), gloss: 1.2, shine: 30, flat: 0.2}, ow)
			c.part(circ(0, -214, 5), mat{base: rgb(0x40, 0xd0, 0xff), gloss: 1}, 1.4)
		case 3: // Hot Dog Hulk
			monLeg(c, colBread, -36)
			monLeg(c, colBread, 36)
			monArm(c, colBread, -60, -110, -112, -150)
			monArm(c, colBread, 60, -110, 110, -70)
			P(capsule(-36, -60, -36, -170, 30, 30), colBread)
			P(capsule(0, -40, 0, -214, 34, 34), rgb(0xc8, 0x50, 0x3a))
			var zz []float64
			for i := 0; i <= 8; i++ {
				zz = append(zz, -16+float64(i%2)*32, -60-float64(i)*12)
			}
			c.part(line(3.5, 3.5, zz...), mat{base: rgb(0xff, 0xd0, 0x20)}, 1.6)
			P(capsule(36, -60, 36, -170, 30, 30), colBread)
		case 4: // Pizza Tyrant
			monLeg(c, limb, -30)
			monLeg(c, limb, 30)
			monArm(c, colCrust, -74, -160, -118, -120)
			monArm(c, colCrust, 74, -160, 118, -200)
			sl := poly(-96, -214, 96, -214, 0, -20).grow(6)
			P(sl, colCheese)
			drips := []float64{-56, 56}
			for _, x := range drips {
				c.part(capsule(x, -214, x*0.92, -150+math.Abs(x)*0.6, 8, 6), mat{base: colCheese, gloss: 0.5}, ow*0.7)
			}
			for _, p := range [][2]float64{{-72, -196}, {72, -196}, {-6, -84}, {24, -98}, {-26, -100}} {
				c.part(circ(p[0], p[1], 11), mat{base: rgb(0xd8, 0x30, 0x28), gloss: 0.6}, ow*0.8)
			}
			P(capsule(-104, -224, 104, -224, 16, 16).tex(), colCrust)
		}
		return
	}
	switch id {
	case 0, 9: // Moldy Bread, Burnt Toast
		crust, in := colCrust, colBread
		if id == 9 {
			crust, in = rgb(0x3a, 0x22, 0x18), rgb(0x6a, 0x40, 0x28)
		}
		monLeg(c, limb, -28)
		monLeg(c, limb, 28)
		monArm(c, crust, -60, -80, -90, -50)
		monArm(c, crust, 60, -80, 92, -110)
		sl := rrect(-62, -150, 124, 128, 14).or(circ(-34, -152, 32)).or(circ(34, -152, 32))
		c.part(sl, mat{base: crust, rim: 0.3}, ow)
		c.volOn(sl.grow(-10), sl, mat{base: in, tex: 0.12, texF: 0.35})
		if id == 0 {
			for _, m := range [][3]float64{{-34, -140, 14}, {36, -60, 12}, {16, -164, 9}, {-40, -46, 8}} {
				c.volOn(circ(m[0], m[1], m[2]).fuzz(2.5, 0.3).and(sl.grow(-4)), sl, mat{base: rgb(0x8a, 0xc8, 0x40), tex: 0.2})
			}
		} else {
			for _, y := range []float64{-130, -96, -62} {
				c.fill(line(3, 3, -40, y, 40, y-8).and(sl.grow(-12)), alpha(rgb(0x20, 0x10, 0x0a), 0.6))
			}
			for i, x := range []float64{-20, 10, 34} {
				c.soft(circ(x, -200-float64(i)*14, 10+float64(i)*3), alpha(rgb(0x80, 0x78, 0x80), 0.55), 8)
			}
		}
	case 1: // Rotten Tomato
		monLeg(c, rgb(0x3a, 0x8a, 0x30), -28)
		monLeg(c, rgb(0x3a, 0x8a, 0x30), 28)
		monArm(c, rgb(0x3a, 0x8a, 0x30), -64, -90, -96, -120)
		monArm(c, rgb(0x3a, 0x8a, 0x30), 64, -80, 98, -50)
		b := ell(0, -84, 74, 64)
		P(b, colTomato)
		for _, s := range [][3]float64{{40, -60, 14}, {-44, -50, 10}, {30, -126, 8}} {
			c.volOn(circ(s[0], s[1], s[2]).fuzz(2, 0.3).and(b), b, mat{base: rgb(0x8a, 0x5a, 0x2a)})
		}
		c.part(capsule(-20, -26, -22, -8, 7, 5).or(capsule(26, -30, 28, -14, 6, 4)), mat{base: rgb(0xd8, 0x30, 0x30), gloss: 0.8}, ow*0.7)
		c.part(poly(-30, -148, -8, -140, 0, -166, 8, -140, 30, -148, 12, -132, 0, -138, -12, -132), mat{base: colLeaf}, ow)
	case 2: // Evil Broccoli
		st := rgb(0xa8, 0xe0, 0x6a)
		monLeg(c, st, -20)
		monLeg(c, st, 20)
		monArm(c, st, -26, -70, -70, -96)
		monArm(c, st, 26, -70, 72, -40)
		stalk := rrect(-30, -130, 60, 110, 18).melt(capsule(-10, -110, -46, -146, 12, 10), 8).melt(capsule(10, -110, 48, -140, 12, 10), 8)
		P(stalk, st)
		fl := circ(-50, -150, 30)
		for _, b := range [][3]float64{{0, -176, 36}, {48, -152, 30}, {-26, -186, 26}, {28, -190, 26}, {-60, -126, 20}, {62, -128, 20}} {
			fl = fl.or(circ(b[0], b[1], b[2]))
		}
		c.fill(fl.fuzz(2.5, 0.25).grow(2.6*inkW), outlineOf(colLeaf))
		c.volOn(fl.fuzz(2.5, 0.25), fl, mat{base: rgb(0x3a, 0xa0, 0x38), rim: 0.4, tex: 0.25, texF: 0.3})
	case 3: // Stinky Cheese
		monLeg(c, limb, -30)
		monLeg(c, limb, 30)
		monArm(c, colCheese, -66, -70, -96, -100)
		monArm(c, colCheese, 66, -60, 96, -30)
		w := poly(-64, -18, -64, -112, 52, -146, 66, -18).grow(8)
		P(w, colCheese)
		top := poly(-64, -112, 52, -146, 66, -120, -50, -96).grow(4).and(w)
		c.volOn(top, w, mat{base: light(colCheese, 0.3)})
		for _, h := range [][3]float64{{-40, -40, 10}, {30, -60, 14}, {44, -110, 8}, {-10, -120, 7}} {
			c.volOn(circ(h[0], h[1], h[2]), circ(h[0], h[1]+h[2]*0.4, h[2]*1.6), mat{base: rgb(0xd8, 0x98, 0x20)})
		}
		for i, x := range []float64{-30, 10, 44} {
			var p []float64
			for k := 0; k <= 6; k++ {
				p = append(p, x+math.Sin(float64(k)+float64(i))*6, -160-float64(k)*9)
			}
			c.fill(line(2.4, 1, p...), alpha(rgb(0x8a, 0xd0, 0x40), 0.8))
		}
	case 4: // Mean Meatball
		monLeg(c, limb, -28)
		monLeg(c, limb, 28)
		monArm(c, colMeat, -62, -80, -94, -110)
		monArm(c, colMeat, 62, -76, 96, -46)
		b := circ(0, -84, 68).fuzz(3, 0.08)
		c.fill(b.grow(2.6*inkW), outlineOf(colMeat))
		c.volOn(b, circ(0, -84, 68), mat{base: colMeat, rim: 0.4, tex: 0.3, texF: 0.18})
		sauce := blob(-50, -126, -30, -150, 0, -156, 30, -150, 52, -124, 30, -118, 20, -104, 4, -116, -16, -106, -30, -118)
		c.volOn(sauce, circ(0, -84, 68), mat{base: rgb(0xd8, 0x2c, 0x20), gloss: 0.8})
		c.fill(sauce.ring(1.8), outlineOf(colTomato))
		c.part(ell(10, -156, 14, 6).rot(10, -156, -0.4), mat{base: colLeaf}, ow*0.8)
	case 5: // Cracked Egg
		monLeg(c, rgb(0xff, 0xc0, 0x30), -24)
		monLeg(c, rgb(0xff, 0xc0, 0x30), 24)
		monArm(c, rgb(0xff, 0xf6, 0xe8), -54, -80, -86, -110)
		monArm(c, rgb(0xff, 0xf6, 0xe8), 54, -80, 88, -50)
		egg := blob(0, -176, 46, -150, 62, -90, 46, -30, 0, -18, -46, -30, -62, -90, -46, -150)
		P(egg, rgb(0xff, 0xf6, 0xe8))
		var zz []float64
		for i := 0; i <= 8; i++ {
			zz = append(zz, -54+float64(i)*13, -138+float64(i%2)*14)
		}
		c.fill(line(2.4, 2.4, zz...).and(egg), outlineOf(rgb(0xff, 0xf6, 0xe8)))
		c.volOn(blob(-20, -138, 0, -126, 26, -138, 18, -124, 0, -116, -14, -124).and(egg), egg, mat{base: rgb(0xff, 0xc8, 0x30)})
	case 6: // Gummy Brute
		g := rgb(0xff, 0x50, 0x70)
		gm := mat{base: g, gloss: 1.2, shine: 24, rim: 0.6}
		for _, x := range []float64{-28, 28} {
			c.part(capsule(x, -40, x, -10, 18, 17), gm, ow)
		}
		monArm(c, g, -50, -100, -86, -70)
		monArm(c, g, 50, -100, 88, -130)
		body := ell(0, -76, 54, 58)
		c.part(body, gm, ow)
		c.volOn(ell(0, -70, 30, 34), body, mat{base: light(g, 0.3), gloss: 0.6})
		for _, s := range []float64{-1, 1} {
			c.part(circ(s*34, -176, 15), gm, ow)
		}
		c.part(circ(0, -140, 42), gm, ow)
	case 7: // Hot Pepper
		r := rgb(0xe8, 0x28, 0x20)
		monLeg(c, rgb(0x3a, 0x8a, 0x30), -22)
		monLeg(c, rgb(0x3a, 0x8a, 0x30), 22)
		monArm(c, rgb(0x3a, 0x8a, 0x30), -34, -90, -76, -120)
		monArm(c, rgb(0x3a, 0x8a, 0x30), 40, -86, 80, -56)
		b := line(42, 12, 0, -40, -6, -96, 10, -150, 34, -180)
		c.part(capsule(0, -40, 0, -26, 34, 30).or(b), mat{base: r, gloss: 1.1, shine: 30, rim: 0.5}, ow)
		c.part(blob(14, -176, 30, -194, 52, -192, 58, -176, 44, -166, 24, -166), mat{base: colLeaf}, ow)
		for i, x := range []float64{20, 34, 48} {
			f := blob(x, -236+float64(i%2)*10, x+9, -212, x+8, -194, x-8, -194, x-9, -212)
			c.part(f, mat{base: rgb(0xff, 0x8a, 0x20), flat: 0.6}, ow*0.7)
		}
	case 8: // Angry Onion
		o := rgb(0xb0, 0x60, 0xc0)
		monLeg(c, rgb(0xf0, 0xe0, 0xc8), -26)
		monLeg(c, rgb(0xf0, 0xe0, 0xc8), 26)
		monArm(c, o, -60, -70, -92, -40)
		monArm(c, o, 60, -76, 92, -108)
		b := ell(0, -70, 64, 58).melt(poly(-26, -110, 26, -110, 0, -176), 20)
		P(b, o)
		for _, s := range []float64{-1, 1} {
			c.fill(arc(s*20, -80, 30, 62, math.Pi*0.8-s*0.7, math.Pi*1.2-s*0.7, 1.8).and(b.grow(-4)), alpha(outlineOf(o), 0.5))
		}
		c.part(line(5, 3, 0, -170, 6, -196, -4, -214).or(line(4, 2, 0, -172, -14, -196)), mat{base: colLeaf}, ow)
		for _, x := range []float64{-10, 0, 10} {
			c.fill(line(1.4, 1, x, -14, x*1.6, -2), rgb(0xd8, 0xc8, 0xb0))
		}
	}
}

// tex is a shorthand for no change: it marks shapes that get grain from their material.
func (a sh) tex() sh { return a }

// paintMonFace paints the angry face: state 0 normal, 3 hurt.
func paintMonFace(c *canvas, m monster, boss bool, state int) {
	out := rgb(0x1a, 0x0e, 0x20)
	skin := m.col
	for i, e := range [][2]float64{{m.e1x, m.e1y}, {m.e2x, m.e2y}} {
		s := float64(i*2 - 1)
		paintEye(c, e[0], e[1], m.er, -s*0.2, state, skin)
		if state == 0 { // angry brows
			c.fill(line(m.er*0.24, m.er*0.18, e[0]-s*m.er*1.1, e[1]-m.er*0.85, e[0]+s*m.er*0.9, e[1]-m.er*1.6), out)
		}
	}
	x, y, w := m.mx, m.my, m.mw
	if state == 3 {
		mo := ell(x, y, w*0.6, w*0.6)
		c.fill(mo.grow(2.2), out)
		c.fill(mo, rgb(0x5a, 0x14, 0x24))
		c.fill(ell(x, y+w*0.35, w*0.35, w*0.2).and(mo), rgb(0xff, 0x6a, 0x7a))
		return
	}
	// a grumpy grimace full of teeth
	mo := ell(x, y+w*0.25, w, w*0.5).and(rrect(x-w-2, y-2, 2*w+4, w, 0))
	c.fill(mo.grow(2.2), out)
	c.fill(mo, rgb(0x5a, 0x14, 0x24))
	n := 4
	if boss {
		n = 5
	}
	for i := 0; i < n; i++ {
		tx := x - w*0.8 + (float64(i)+0.5)*w*1.6/float64(n)
		tw := w * 0.8 / float64(n)
		c.fill(poly(tx-tw, y-1, tx+tw, y-1, tx, y+w*0.32).and(mo), colWhite)
	}
}

type monSprite struct {
	body *pimg
	face [2]*pimg // angry, hurt
}

func monArt(id int, boss bool, z float64) *monSprite {
	m := monsters[id%len(monsters)]
	key := "mon" + itoa(id)
	if boss {
		m = bosses[id%len(bosses)]
		key = "boss" + itoa(id)
	}
	key += "@" + num(z*100)
	s := &monSprite{}
	s.body = paintArt(key, -140, -260, 140, 12, z, func(c *canvas) { paintMonster(c, id, boss) })
	for i, st := range []int{0, 3} {
		st := st
		s.face[i] = paintArt(key+"f"+itoa(st), -90, -230, 90, -10, z, func(c *canvas) { paintMonFace(c, m, boss, st) })
	}
	return s
}

func monOf(id int, boss bool) monster {
	if boss {
		return bosses[id%len(bosses)]
	}
	return monsters[id%len(monsters)]
}
