package game

import (
	"image/color"
	"math"
	"math/rand"
)

// The places you fight in: a painted backdrop per area plus drifting clouds,
// swaying grass in front and floating pollen.

type areaDef struct {
	name             string
	sky0, sky1       color.RGBA
	far, mid, ground color.RGBA
	kind             int
	night            bool
}

const (
	akMeadow = iota
	akMushroom
	akPond
	akAutumn
	akBerry
	akMoon
)

var areas = []areaDef{
	{"Sunny Meadow", rgb(0x5c, 0xc0, 0xff), rgb(0xc8, 0xf0, 0xff), rgb(0x9c, 0xe0, 0x9a), rgb(0x6c, 0xc8, 0x4a), rgb(0x7c, 0xd4, 0x46), akMeadow, false},
	{"Mushroom Grove", rgb(0x6a, 0x8a, 0xe8), rgb(0xf0, 0xc8, 0xf0), rgb(0x9a, 0xa8, 0xe0), rgb(0x5a, 0xb0, 0x8a), rgb(0x6a, 0xc0, 0x6a), akMushroom, false},
	{"Lily Pond", rgb(0x48, 0xb8, 0xf0), rgb(0xd8, 0xf8, 0xf0), rgb(0x8a, 0xd8, 0xc0), rgb(0x48, 0xb8, 0x6a), rgb(0x6a, 0xcc, 0x58), akPond, false},
	{"Autumn Woods", rgb(0xff, 0x9a, 0x6a), rgb(0xff, 0xe0, 0xa0), rgb(0xf0, 0xb0, 0x7a), rgb(0xe0, 0x8a, 0x3a), rgb(0xc8, 0xb0, 0x48), akAutumn, false},
	{"Berry Bramble", rgb(0xb0, 0x7a, 0xf0), rgb(0xff, 0xd0, 0xe8), rgb(0xc8, 0x9a, 0xe0), rgb(0x5a, 0xa8, 0x70), rgb(0x78, 0xc0, 0x5a), akBerry, false},
	{"Moonlit Garden", rgb(0x1e, 0x24, 0x5a), rgb(0x5a, 0x4a, 0x9a), rgb(0x3a, 0x40, 0x8a), rgb(0x2e, 0x6a, 0x7a), rgb(0x3a, 0x8a, 0x6a), akMoon, true},
}

func areaOf(stage int) areaDef { return areas[area(stage)%len(areas)] }

// hillSh is everything below the curve y = f(x) between x0 and x1.
func hillSh(x0, x1, top, bot float64, f func(x float64) float64) sh {
	return sh{func(x, y float64) float64 {
		d := f(x) - y
		if x < x0 {
			d = math.Max(d, x0-x)
		}
		if x > x1 {
			d = math.Max(d, x-x1)
		}
		return d
	}, x0, top, x1, bot}
}

func waves(seed float64, amp ...float64) func(x float64) float64 {
	return func(x float64) float64 {
		v := 0.0
		for i, a := range amp {
			f := 0.006 * math.Pow(2.1, float64(i))
			v += math.Sin(x*f+seed*(float64(i)+1.7)) * a
		}
		return v
	}
}

func flower(c *canvas, x, y, r float64, petal, mid color.RGBA, stem float64) {
	if stem > 0 {
		st := line(r*0.16, r*0.2, x, y, x+r*0.3, y+stem*0.5, x, y+stem)
		c.part(st, mat{base: rgb(0x4a, 0xa8, 0x3a), gloss: 0.3}, 1.4)
		lf := ell(x+r*0.7, y+stem*0.6, r*0.6, r*0.28).rot(x+r*0.7, y+stem*0.6, -0.5)
		c.part(lf, mat{base: rgb(0x5a, 0xc0, 0x40), gloss: 0.4}, 1.4)
	}
	var ps sh
	for i := 0; i < 6; i++ {
		a := float64(i)/6*2*math.Pi + 0.3
		p := ell(x+math.Cos(a)*r*0.62, y+math.Sin(a)*r*0.62, r*0.5, r*0.36).rot(x+math.Cos(a)*r*0.62, y+math.Sin(a)*r*0.62, a)
		if i == 0 {
			ps = p
		} else {
			ps = ps.or(p)
		}
	}
	c.fill(ps.grow(1.6), outlineOf(petal))
	c.volOn(ps, circ(x, y, r*1.1), mat{base: petal, gloss: 0.4, rim: 0.3})
	c.part(circ(x, y, r*0.36), mat{base: mid, gloss: 0.8, rim: 0.4}, 1.4)
}

func mushroom(c *canvas, x, y, h float64, capCol color.RGBA) {
	st := rrect(x-h*0.13, y-h*0.62, h*0.26, h*0.62, h*0.1)
	c.part(st, mat{base: rgb(0xff, 0xf0, 0xd8), gloss: 0.4, rim: 0.4}, 2)
	cp := ell(x, y-h*0.62, h*0.42, h*0.3).and(rrect(x-h, y-h*1.2, h*2, h*0.62, 0)).melt(ell(x, y-h*0.62, h*0.42, h*0.06), 4)
	c.part(cp, mat{base: capCol, gloss: 0.9, rim: 0.5}, 2.2)
	for _, s := range [][3]float64{{-0.2, -0.75, 0.07}, {0.12, -0.84, 0.06}, {0.26, -0.68, 0.045}, {-0.05, -0.68, 0.035}} {
		c.volOn(circ(x+s[0]*h, y+s[1]*h, s[2]*h).and(cp), cp, mat{base: rgb(0xff, 0xfa, 0xf0), gloss: 0.5})
	}
}

func tree(c *canvas, x, y, h float64, leaf color.RGBA, seed int64) {
	r := rand.New(rand.NewSource(seed))
	tr := line(h*0.07, h*0.05, x, y, x+h*0.04, y-h*0.5)
	c.part(tr, mat{base: rgb(0x8a, 0x5a, 0x3a), gloss: 0.3, tex: 0.15}, 2)
	var cn sh
	for i := 0; i < 6; i++ {
		cx := x + (r.Float64()-0.5)*h*0.6
		cy := y - h*0.6 - r.Float64()*h*0.35
		b := circ(cx, cy, h*(0.18+r.Float64()*0.1))
		if i == 0 {
			cn = b
		} else {
			cn = cn.melt(b, h*0.06)
		}
	}
	cn = cn.fuzz(h*0.012, 0.15)
	c.fill(cn.grow(2), outlineOf(leaf))
	c.volOn(cn, cn, mat{base: leaf, gloss: 0.2, rim: 0.4, tex: 0.1, texF: 0.12})
}

func cloudSh(x, y, w float64) sh {
	s := ell(x, y, w*0.5, w*0.16)
	for _, b := range [][3]float64{{-0.22, -0.1, 0.2}, {0.05, -0.2, 0.26}, {0.28, -0.06, 0.18}} {
		s = s.melt(circ(x+b[0]*w, y+b[1]*w, b[2]*w), w*0.05)
	}
	return s.and(rrect(x-w, y-w, w*2, w+w*0.12, 0))
}

// paintScene paints an area's backdrop, w x h units with the ground at gy.
func paintScene(c *canvas, a areaDef, w, h, gy float64) {
	rng := rand.New(rand.NewSource(int64(a.kind)*977 + 5))
	all := rrect(-2, -2, w+4, h+4, 0)
	c.grad(all, a.sky0, a.sky1, 0, gy-40)
	// sun or moon
	if a.night {
		for i := 0; i < 60; i++ {
			x, y := rng.Float64()*w, rng.Float64()*gy*0.7
			c.soft(circ(x, y, 0.6+rng.Float64()*1.2), alpha(colWhite, 0.5+rng.Float64()*0.5), 1.5)
		}
		c.glowIn(circ(420, 90, 34), rgb(0xa0, 0xb0, 0xff), 90, 0.35)
		mo := circ(420, 90, 34)
		c.vol(mo, mat{base: rgb(0xf8, 0xf4, 0xe0), dark: rgb(0xc8, 0xc0, 0xe0), flat: 0.5, tex: 0.12, texF: 0.1})
	} else {
		c.glowIn(circ(440, 80, 30), rgb(0xff, 0xf4, 0xc0), 120, 0.45)
		c.vol(circ(440, 80, 30), mat{base: rgb(0xff, 0xf2, 0xa0), dark: rgb(0xff, 0xd0, 0x60), flat: 0.6})
	}
	// far and middle hills
	fy := gy - 150
	far := hillSh(-10, w+10, fy-60, h, func(x float64) float64 { return fy + waves(float64(a.kind), 26, 12, 5)(x) })
	c.vol(far, mat{base: a.far, flat: 0.55, depth: 40, rim: 0.2})
	switch a.kind {
	case akMushroom:
		for _, m := range [][3]float64{{60, 0, 120}, {470, 0, 150}, {300, 0, 80}} {
			mushroom(c, m[0], gy-80, m[2], mix(rgb(0xd8, 0x6a, 0xc8), a.far, 0.45))
		}
	case akAutumn:
		for i, x := range []float64{30, 140, 260, 390, 500} {
			tree(c, x, gy-70, 150+float64(i%2)*40, mix([]color.RGBA{rgb(0xff, 0x8a, 0x2a), rgb(0xe8, 0x5a, 0x30), rgb(0xff, 0xc0, 0x3a)}[i%3], a.far, 0.25), int64(i))
		}
	case akMoon:
		for i, x := range []float64{40, 200, 480} {
			tree(c, x, gy-70, 160, mix(rgb(0x2a, 0x5a, 0x8a), a.far, 0.3), int64(i+9))
		}
	case akMeadow, akPond, akBerry:
		for i, x := range []float64{70, 480} {
			tree(c, x, gy-90, 130+float64(i)*30, mix(rgb(0x4a, 0xb0, 0x4a), a.far, 0.35), int64(i+3))
		}
	}
	my := gy - 70
	mid := hillSh(-10, w+10, my-40, h, func(x float64) float64 { return my + waves(float64(a.kind)+3, 16, 8, 3)(x) })
	c.vol(mid, mat{base: a.mid, flat: 0.45, depth: 34, rim: 0.3, tex: 0.06, texF: 0.05})
	if a.kind == akPond {
		water := hillSh(-10, w+10, gy-48, gy+10, func(x float64) float64 { return gy - 44 + math.Sin(x*0.02)*3 })
		c.grad(water, rgb(0x6a, 0xd0, 0xf8), rgb(0x2a, 0x90, 0xd8), gy-48, gy)
		for i := 0; i < 14; i++ {
			x := rng.Float64() * w
			c.fill(capsule(x, gy-30+rng.Float64()*20, x+12+rng.Float64()*20, gy-30+rng.Float64()*20, 1.2, 1.2), alpha(colWhite, 0.5))
		}
		for _, x := range []float64{90, 300, 460} {
			pad := ell(x, gy-28, 26, 8).minus(poly(x, gy-28, x+30, gy-36, x+30, gy-22))
			c.part(pad, mat{base: rgb(0x5a, 0xc0, 0x50), gloss: 0.5, flat: 0.3}, 1.6)
		}
		flower(c, 304, gy-36, 8, rgb(0xff, 0xb0, 0xd8), rgb(0xff, 0xe0, 0x60), 0)
	}
	// the ground the brigade stands on
	ground := hillSh(-10, w+10, gy-34, h+10, func(x float64) float64 { return gy - 24 + waves(float64(a.kind)+7, 5, 3)(x) })
	c.vol(ground, mat{base: a.ground, flat: 0.4, depth: 26, rim: 0.3, tex: 0.08, texF: 0.08})
	path := ell(w/2, gy+18, w*0.62, 22)
	c.volOn(path.and(ground), ground, mat{base: mix(a.ground, rgb(0xf0, 0xd8, 0x90), 0.5), flat: 0.5, tex: 0.12, texF: 0.15})
	// grass blades along the ground's top edge
	gc := shade(a.ground, 0.78)
	for i := 0; i < 70; i++ {
		x := rng.Float64() * w
		y := gy - 22 + waves(float64(a.kind)+7, 5, 3)(x) + rng.Float64()*30
		hh := 6 + rng.Float64()*8
		c.fill(line(1.6, 0.4, x, y, x+(rng.Float64()-0.5)*6, y-hh), alpha(gc, 0.8))
	}
	// details
	switch a.kind {
	case akMeadow, akPond:
		for i := 0; i < 7; i++ {
			x := 20 + rng.Float64()*(w-40)
			y := gy - 10 + rng.Float64()*40
			cols := []color.RGBA{colWhite, rgb(0xff, 0xe0, 0x50), rgb(0xff, 0x9a, 0xc8)}
			flower(c, x, y, 5+rng.Float64()*3, cols[i%3], rgb(0xff, 0xb0, 0x30), 0)
		}
		flower(c, 24, gy-60, 18, rgb(0xff, 0xd0, 0x30), rgb(0x9a, 0x5a, 0x2a), 70)
		flower(c, 516, gy-70, 16, colWhite, rgb(0xff, 0xc0, 0x30), 80)
	case akMushroom:
		mushroom(c, 30, gy+10, 70, rgb(0xf0, 0x4a, 0x4a))
		mushroom(c, 515, gy+6, 56, rgb(0xb0, 0x6a, 0xf0))
		mushroom(c, 60, gy+20, 30, rgb(0xff, 0x9a, 0x3a))
	case akAutumn:
		for i := 0; i < 16; i++ {
			x, y := rng.Float64()*w, gy-10+rng.Float64()*50
			lf := ell(x, y, 5, 2.6).rot(x, y, rng.Float64()*3)
			c.part(lf, mat{base: []color.RGBA{rgb(0xff, 0x8a, 0x2a), rgb(0xe0, 0x4a, 0x2a), rgb(0xff, 0xc8, 0x3a)}[i%3], gloss: 0.3}, 1)
		}
	case akBerry:
		for _, b := range [][2]float64{{20, 0}, {520, 0}} {
			bush := circ(b[0]-20, gy, 34).melt(circ(b[0]+16, gy-14, 36), 10).melt(circ(b[0]+30, gy+10, 26), 10).fuzz(1.5, 0.3)
			c.fill(bush.grow(2), outlineOf(rgb(0x3a, 0x8a, 0x50)))
			c.volOn(bush, bush, mat{base: rgb(0x3a, 0x8a, 0x50), rim: 0.3, tex: 0.1})
			for i := 0; i < 9; i++ {
				x := b[0] - 30 + rng.Float64()*60
				y := gy - 30 + rng.Float64()*40
				c.part(circ(x, y, 5), mat{base: []color.RGBA{rgb(0xf0, 0x3a, 0x6a), rgb(0x6a, 0x3a, 0xd0)}[i%2], gloss: 1.1, shine: 30}, 1.2)
			}
		}
	case akMoon:
		for i := 0; i < 8; i++ {
			x, y := 20+rng.Float64()*(w-40), gy-6+rng.Float64()*40
			c.glowIn(circ(x, y, 4), rgb(0x80, 0xe0, 0xff), 16, 0.4)
			flower(c, x, y, 5, rgb(0x9a, 0xe8, 0xff), rgb(0xff, 0xff, 0xd0), 0)
		}
	}
}

// paintTuft paints a clump of tall grass for the foreground.
func paintTuft(c *canvas, col color.RGBA, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < 9; i++ {
		x := (r.Float64() - 0.5) * 40
		hh := 40 + r.Float64()*50
		lean := (r.Float64() - 0.5) * 40
		bl := line(4.5, 0.6, x, 4, x+lean*0.3, -hh*0.5, x+lean, -hh)
		cc := mix(col, shade(col, 0.7), r.Float64())
		c.fill(bl.grow(1.5), outlineOf(cc))
		c.vol(bl, mat{base: cc, gloss: 0.3, flat: 0.3})
	}
}

func paintCloud(c *canvas, night bool) {
	s := cloudSh(0, 0, 120)
	base := colWhite
	if night {
		base = rgb(0x8a, 0x8a, 0xc8)
	}
	c.vol(s, mat{base: base, dark: mix(base, rgb(0xb0, 0xa8, 0xe8), 0.7), flat: 0.35, rim: 0.4})
}
