package game

import (
	"image/color"
	"math"
	"math/rand"
)

// The places you fight in (a painted backdrop per area) and the kitchen with its
// stations.

type areaDef struct {
	name         string
	kind         int
	wall0, wall1 color.RGBA
	floor        color.RGBA
	night        bool
}

const (
	akPantry = iota
	akGarden
	akBBQ
	akCandy
	akFridge
	akAlley
)

var areas = []areaDef{
	{"The Pantry", akPantry, rgb(0xc8, 0x8a, 0x50), rgb(0xa0, 0x64, 0x34), rgb(0x8a, 0x5a, 0x34), false},
	{"Veggie Garden", akGarden, rgb(0x6c, 0xc8, 0xff), rgb(0xd0, 0xf4, 0xff), rgb(0x78, 0xc8, 0x48), false},
	{"BBQ Yard", akBBQ, rgb(0xff, 0x8a, 0x5a), rgb(0xff, 0xd0, 0x90), rgb(0x9a, 0x94, 0x90), false},
	{"Candy Shop", akCandy, rgb(0xff, 0xc8, 0xe0), rgb(0xff, 0xe8, 0xf2), rgb(0xff, 0x9a, 0xc8), false},
	{"The Fridge", akFridge, rgb(0xc8, 0xec, 0xff), rgb(0x90, 0xc8, 0xf0), rgb(0xd8, 0xf4, 0xff), false},
	{"Dumpster Alley", akAlley, rgb(0x1e, 0x20, 0x4a), rgb(0x4a, 0x3a, 0x6a), rgb(0x3a, 0x38, 0x48), true},
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

func cloudSh(x, y, w float64) sh {
	s := ell(x, y, w*0.5, w*0.16)
	for _, b := range [][3]float64{{-0.22, -0.1, 0.2}, {0.05, -0.2, 0.26}, {0.28, -0.06, 0.18}} {
		s = s.melt(circ(x+b[0]*w, y+b[1]*w, b[2]*w), w*0.05)
	}
	return s.and(rrect(x-w, y-w, w*2, w+w*0.12, 0))
}

func jar(c *canvas, x, y, w, h float64, fill color.RGBA) {
	body := rrect(x-w/2, y-h, w, h, w*0.2)
	c.part(body, mat{base: rgb(0xd8, 0xf0, 0xff), flat: 0.5, gloss: 0.8}, 2)
	c.volOn(rrect(x-w/2+3, y-h*0.62, w-6, h*0.62-3, w*0.15), body, mat{base: fill, flat: 0.3})
	c.part(rrect(x-w/2-2, y-h-8, w+4, 10, 3), mat{base: rgb(0xe0, 0x50, 0x40)}, 2)
	c.fill(rrect(x-w/2+5, y-h+6, 4, h*0.5, 2), alpha(colWhite, 0.6))
}

func sack(c *canvas, x, y, s float64, col color.RGBA) {
	b := blob(x-10*s, y-40*s, x+10*s, y-40*s, x+24*s, y-14*s, x+20*s, y, x-20*s, y, x-24*s, y-14*s)
	c.part(b, mat{base: col, tex: 0.12, texF: 0.3}, 2.4)
	c.part(ell(x, y-41*s, 12*s, 5*s), mat{base: shade(col, 0.85)}, 2)
}

// paintScene paints an area's backdrop, w x h units with the floor at gy.
func paintScene(c *canvas, a areaDef, w, h, gy float64) {
	rng := rand.New(rand.NewSource(int64(a.kind)*977 + 5))
	all := rrect(-2, -2, w+4, h+4, 0)
	c.grad(all, a.wall0, a.wall1, 0, gy)
	floor := hillSh(-10, w+10, gy-30, h+10, func(x float64) float64 { return gy - 26 })
	switch a.kind {
	case akPantry:
		for x := 0.0; x < w; x += 54 {
			c.fill(rrect(x, 0, 3, gy, 0), alpha(rgb(0x6a, 0x3a, 0x1c), 0.5))
			c.vol(rrect(x+3, 0, 51, gy, 0), mat{base: mix(a.wall0, a.wall1, rng.Float64()), flat: 1, tex: 0.12, texF: 0.08})
		}
		for _, sy := range []float64{gy - 300, gy - 180} {
			c.part(rrect(-10, sy, w+20, 16, 3), mat{base: rgb(0x8a, 0x52, 0x2a), tex: 0.15, texF: 0.15}, 2.4)
			x := 20.0
			for x < w-30 {
				switch rng.Intn(3) {
				case 0:
					jw, jh := 34+rng.Float64()*12, 44+rng.Float64()*26
					jar(c, x+jw/2, sy, jw, jh, []color.RGBA{colTomato, rgb(0xff, 0xc0, 0x30), rgb(0x8a, 0x40, 0xc0), colLeaf}[rng.Intn(4)])
					x += jw + 14
				case 1:
					sack(c, x+24, sy, 1.1, rgb(0xf0, 0xe4, 0xcc))
					x += 62
				default:
					can := rrect(x, sy-50, 34, 50, 4)
					c.part(can, mat{base: rgb(0xc8, 0xcc, 0xd8), gloss: 1, shine: 30}, 2)
					c.volOn(rrect(x, sy-38, 34, 24, 0), can, mat{base: []color.RGBA{colRed, colBlue, colGreen}[rng.Intn(3)]})
					x += 48
				}
			}
		}
		c.vol(floor, mat{base: a.floor, flat: 0.5, depth: 20, tex: 0.12, texF: 0.06})
		for x := 0.0; x < w; x += 90 {
			c.fill(rrect(x, gy-24, 2.5, h, 0), alpha(rgb(0x4a, 0x2a, 0x14), 0.5))
		}
	case akGarden:
		c.glowIn(circ(450, 70, 30), rgb(0xff, 0xf4, 0xc0), 110, 0.45)
		c.vol(circ(450, 70, 30), mat{base: rgb(0xff, 0xf0, 0x90), dark: rgb(0xff, 0xd0, 0x60), flat: 0.6})
		hills := hillSh(-10, w+10, gy-200, h, func(x float64) float64 { return gy - 160 + waves(2, 18, 8)(x) })
		c.vol(hills, mat{base: rgb(0x9c, 0xdc, 0x8a), flat: 0.5, depth: 40})
		for x := -10.0; x < w+20; x += 34 {
			pk := poly(x, gy-40, x, gy-128, x+13, gy-142, x+26, gy-128, x+26, gy-40)
			c.part(pk, mat{base: rgb(0xfa, 0xf6, 0xee), rim: 0.3}, 2.2)
		}
		c.part(rrect(-10, gy-118, w+20, 12, 3), mat{base: rgb(0xf0, 0xea, 0xe0)}, 2.2)
		c.part(rrect(-10, gy-74, w+20, 12, 3), mat{base: rgb(0xf0, 0xea, 0xe0)}, 2.2)
		c.vol(floor, mat{base: rgb(0x8a, 0x5a, 0x34), flat: 0.4, depth: 20, tex: 0.2, texF: 0.1})
		for _, x := range []float64{40, 120, 420, 500} {
			c.part(circ(x, gy-24, 20).fuzz(1.5, 0.3), mat{base: rgb(0x8c, 0xd8, 0x5a), tex: 0.1}, 2.2)
			c.volOn(circ(x, gy-24, 11), circ(x, gy-24, 20), mat{base: rgb(0xc8, 0xf0, 0x90)})
		}
		for i, x := range []float64{14, 528} {
			st := line(5, 4, x, gy, x+4, gy-120, x, gy-200)
			c.part(st, mat{base: rgb(0x5a, 0xb0, 0x3a)}, 2)
			var ps sh
			for k := 0; k < 12; k++ {
				an := float64(k) / 12 * 2 * math.Pi
				p := ell(x+math.Cos(an)*22, gy-210+math.Sin(an)*22, 14, 7).rot(x+math.Cos(an)*22, gy-210+math.Sin(an)*22, an)
				if k == 0 {
					ps = p
				} else {
					ps = ps.or(p)
				}
			}
			c.part(ps, mat{base: rgb(0xff, 0xd0, 0x30)}, 2)
			c.part(circ(x, gy-210, 15), mat{base: rgb(0x7a, 0x4a, 0x2a), tex: 0.3, texF: 0.4}, 2)
			_ = i
		}
	case akBBQ:
		c.grad(rrect(-2, gy-250, w+4, 260, 0), rgb(0xb8, 0x5a, 0x40), rgb(0x8a, 0x3a, 0x30), gy-250, gy)
		for row := 0; row < 9; row++ {
			y := gy - 250 + float64(row)*26
			off := float64(row%2) * 30
			for x := -60.0 + off; x < w; x += 60 {
				b := rrect(x+2, y+2, 56, 22, 3)
				c.vol(b, mat{base: mix(rgb(0xc8, 0x60, 0x40), rgb(0xa8, 0x48, 0x30), rng.Float64()), flat: 0.6, depth: 4, tex: 0.15, texF: 0.2})
			}
		}
		var wire []float64
		for i := 0; i <= 12; i++ {
			x := float64(i) * w / 12
			wire = append(wire, x, 40+math.Sin(float64(i)/12*math.Pi)*40)
		}
		c.fill(line(1.5, 1.5, wire...), rgb(0x3a, 0x2a, 0x30))
		for i := 1; i < 12; i++ {
			x, y := wire[2*i], wire[2*i+1]+10
			col := []color.RGBA{colGold, colPink, colGreen, colBlue}[i%4]
			c.glowIn(circ(x, y, 5), col, 20, 0.5)
			c.part(ell(x, y, 5, 7), mat{base: col, flat: 0.5, gloss: 0.6}, 1.6)
		}
		c.vol(floor, mat{base: a.floor, flat: 0.5, depth: 20, tex: 0.1, texF: 0.08})
		for x := 0.0; x < w; x += 70 {
			c.fill(rrect(x, gy-24, 2.5, h, 0), alpha(rgb(0x5a, 0x54, 0x50), 0.6))
		}
		// a grill puffing smoke
		gx := 480.0
		c.part(line(3, 3, gx-30, gy, gx-20, gy-60).or(line(3, 3, gx+30, gy, gx+20, gy-60)), mat{base: rgb(0x3a, 0x3a, 0x44)}, 2)
		c.part(ell(gx, gy-70, 44, 26).and(rrect(gx-50, gy-70, 100, 40, 0)), mat{base: rgb(0x2a, 0x2a, 0x34), gloss: 0.8}, 2.4)
		for i := 0; i < 4; i++ {
			c.soft(circ(gx-10+float64(i)*8, gy-110-float64(i)*30, 16+float64(i)*6), alpha(rgb(0xd0, 0xc8, 0xd0), 0.45), 12)
		}
	case akCandy:
		for x := 0.0; x < w; x += 60 {
			c.fill(rrect(x, 0, 30, gy, 0), alpha(colWhite, 0.55))
		}
		sy := gy - 230
		c.part(rrect(-10, sy, w+20, 14, 3), mat{base: rgb(0xd8, 0x6a, 0xa0)}, 2.4)
		for i, x := range []float64{80, 170, 260, 350, 440} {
			col := []color.RGBA{colPink, rgb(0x8c, 0xe0, 0x60), rgb(0x60, 0xc8, 0xff), colGold, colPurple}[i]
			jar(c, x, sy, 54, 70, col)
			for k := 0; k < 5; k++ {
				c.part(circ(x-14+float64(k%3)*14, sy-12-float64(k/3)*14, 6), mat{base: light(col, 0.2*float64(k%2)), gloss: 1}, 1.2)
			}
		}
		for _, x := range []float64{30, 512} {
			c.part(rrect(x-4, gy-180, 8, 160, 3), mat{base: colWhite}, 2)
			lp := circ(x, gy-200, 38)
			c.part(lp, mat{base: rgb(0xff, 0x6a, 0xb0), gloss: 1}, 2.4)
			var sp []float64
			for k := 0; k < 30; k++ {
				an := float64(k) * 0.42
				r := float64(k) * 1.2
				sp = append(sp, x+math.Cos(an)*r, gy-200+math.Sin(an)*r)
			}
			c.fill(line(4, 4, sp...).and(lp), alpha(colWhite, 0.85))
		}
		for x := 0.0; x < w; x += 44 {
			for row := 0.0; row < 3; row++ {
				col := colWhite
				if (int(x/44)+int(row))%2 == 0 {
					col = a.floor
				}
				c.fill(rrect(x, gy-26+row*30, 44, 30, 0), col)
			}
		}
		c.fill(rrect(-2, gy-28, w+4, 4, 0), rgb(0xd8, 0x6a, 0xa0))
	case akFridge:
		for _, sy := range []float64{gy - 290, gy - 150} {
			c.part(rrect(-10, sy, w+20, 8, 3), mat{base: colWhite, gloss: 1}, 1.6)
		}
		y := gy - 150.0
		mk := rrect(40, y-110, 60, 110, 4).or(poly(40, y-110, 70, y-140, 100, y-110))
		c.part(mk, mat{base: colWhite, flat: 0.3}, 2.2)
		c.fill(rrect(40, y-80, 60, 34, 0), colBlue)
		c.part(rrect(150, y-40, 110, 40, 5), mat{base: rgb(0xd8, 0xb8, 0x90), tex: 0.1}, 2.2)
		for k := 0; k < 4; k++ {
			c.part(ell(166+float64(k)*26, y-42, 11, 13), mat{base: rgb(0xff, 0xf6, 0xe8)}, 1.6)
		}
		for i, x := range []float64{330, 380, 430} {
			b := rrect(x, y-120, 34, 120, 10).or(rrect(x+10, y-150, 14, 34, 4))
			c.part(b, mat{base: []color.RGBA{rgb(0x60, 0xd0, 0x80), rgb(0xff, 0xa0, 0x40), rgb(0xa0, 0xd8, 0xff)}[i], flat: 0.4, gloss: 1}, 2)
		}
		jar(c, 120, gy-290, 50, 60, rgb(0x9a, 0x30, 0x70))
		jar(c, 420, gy-290, 46, 54, rgb(0xff, 0xc0, 0x30))
		for _, p := range [][2]float64{{0, 0}, {w, 0}, {0, gy}, {w, gy}} {
			c.soft(circ(p[0], p[1], 50), alpha(colWhite, 0.8), 40)
		}
		c.vol(floor, mat{base: a.floor, flat: 0.3, depth: 24, gloss: 0.6})
		c.fill(rrect(-2, gy-26, w+4, 4, 0), alpha(colWhite, 0.9))
	case akAlley:
		for i := 0; i < 50; i++ {
			c.soft(circ(rng.Float64()*w, rng.Float64()*120, 0.6+rng.Float64()), alpha(colWhite, 0.4+rng.Float64()*0.5), 1.5)
		}
		c.glowIn(circ(430, 60, 26), rgb(0xa0, 0xb0, 0xff), 80, 0.35)
		c.vol(circ(430, 60, 26), mat{base: rgb(0xf8, 0xf4, 0xe0), dark: rgb(0xc8, 0xc0, 0xe0), flat: 0.5})
		for row := 0; row < 12; row++ {
			y := gy - 320 + float64(row)*26
			off := float64(row%2) * 30
			for x := -60.0 + off; x < w; x += 60 {
				c.vol(rrect(x+2, y+2, 56, 22, 3), mat{base: mix(rgb(0x4a, 0x34, 0x5a), rgb(0x3a, 0x2a, 0x4a), rng.Float64()), flat: 0.6, depth: 4, tex: 0.15, texF: 0.2})
			}
		}
		c.glowIn(circ(500, gy-260, 10), rgb(0xff, 0xd8, 0x80), 160, 0.35)
		c.part(rrect(496, gy-250, 8, 230, 2), mat{base: rgb(0x2a, 0x2a, 0x34)}, 2)
		c.part(ell(500, gy-260, 18, 10), mat{base: rgb(0xff, 0xe8, 0xa0), flat: 0.7}, 2)
		c.vol(floor, mat{base: a.floor, flat: 0.5, depth: 20, tex: 0.2, texF: 0.1})
		d := rrect(10, gy-130, 130, 110, 6)
		c.part(d, mat{base: rgb(0x3a, 0x8a, 0x5a), gloss: 0.6}, 2.4)
		c.part(rrect(4, gy-142, 142, 18, 4).rot(75, gy-133, -0.08), mat{base: rgb(0x2a, 0x6a, 0x44), gloss: 0.6}, 2.4)
		for _, b := range [][3]float64{{150, -10, 30}, {176, -14, 24}} {
			c.part(circ(b[0], gy+b[1]-b[2], b[2]).melt(poly(b[0]-6, gy+b[1]-b[2]*2, b[0]+6, gy+b[1]-b[2]*2, b[0], gy+b[1]-b[2]*2-14), 4), mat{base: rgb(0x2a, 0x2a, 0x34), gloss: 1}, 2)
		}
	}
	// soft shade at the floor line
	c.fill(rrect(-2, gy-26, w+4, 3, 0), alpha(rgb(0x1a, 0x0e, 0x20), 0.25))
}

func paintCloud(c *canvas) {
	s := cloudSh(0, 0, 120)
	c.part(s, mat{base: colWhite, dark: rgb(0xc8, 0xd8, 0xf0), flat: 0.3}, 2)
}

// ---- the kitchen ----

// stationPos is where station i stands in the kitchen (bottom centre).
func (g *Game) stationPos(i int) (float64, float64) {
	return 58 + float64(i)*106, g.fieldH() - 34
}

func paintKitchen(c *canvas, w, h float64) {
	all := rrect(-2, -2, w+4, h+4, 0)
	c.grad(all, rgb(0xd8, 0xf4, 0xf0), rgb(0xa8, 0xdc, 0xd8), 0, h)
	for x := 0.0; x < w; x += 36 {
		c.fill(rrect(x, 0, 2, h, 0), alpha(rgb(0x70, 0xb0, 0xb0), 0.6))
	}
	for y := 0.0; y < h; y += 36 {
		c.fill(rrect(0, y, w, 2, 0), alpha(rgb(0x70, 0xb0, 0xb0), 0.6))
	}
	// a window
	win := rrect(170, 70, 200, 130, 8)
	c.part(win.grow(8), mat{base: rgb(0xf8, 0xf4, 0xec)}, 2.4)
	c.grad(win, rgb(0x6c, 0xc8, 0xff), rgb(0xd0, 0xf4, 0xff), 70, 200)
	c.part(cloudSh(230, 130, 70), mat{base: colWhite, flat: 0.4}, 1.6)
	c.fill(rrect(268, 70, 6, 130, 0), rgb(0xf8, 0xf4, 0xec))
	c.fill(rrect(170, 132, 200, 6, 0), rgb(0xf8, 0xf4, 0xec))
	for _, s := range []float64{-1, 1} {
		cur := blob(270+s*110, 56, 270+s*80, 60, 270+s*92, 140, 270+s*84, 214, 270+s*118, 214)
		c.part(cur, mat{base: rgb(0xff, 0x7a, 0x7a), tex: 0.05}, 2.2)
	}
	c.part(rrect(150, 50, 240, 10, 4), mat{base: rgb(0xb8, 0x80, 0x40)}, 2)
	// hanging pans
	c.part(rrect(20, 40, 120, 6, 3), mat{base: rgb(0x9a, 0x9a, 0xa8), gloss: 1}, 1.6)
	c.part(rrect(400, 40, 120, 6, 3), mat{base: rgb(0x9a, 0x9a, 0xa8), gloss: 1}, 1.6)
	for i, x := range []float64{40, 90, 130, 420, 470, 510} {
		r := 16 + float64(i%3)*5
		c.part(line(2, 2, x, 46, x, 70), mat{base: rgb(0x5a, 0x5a, 0x66)}, 1.4)
		c.part(rrect(x-3, 66, 6, 30, 3), mat{base: rgb(0x3a, 0x3a, 0x44)}, 1.6)
		pan := circ(x, 96+r, r)
		c.part(pan, mat{base: []color.RGBA{rgb(0xc8, 0x70, 0x40), rgb(0x3a, 0x3a, 0x48), rgb(0xd0, 0xd4, 0xe0)}[i%3], gloss: 1, shine: 26}, 2)
	}
	// the floor
	c.vol(rrect(-2, h-48, w+4, 60, 0), mat{base: rgb(0xd8, 0xb8, 0x88), flat: 0.6, depth: 10, tex: 0.1, texF: 0.06})
	for x := 0.0; x < w; x += 60 {
		c.fill(rrect(x, h-48, 2, 60, 0), alpha(rgb(0x8a, 0x6a, 0x40), 0.6))
	}
}

// paintStation paints a kitchen station standing at 0,0 (about 96 wide).
func paintStation(c *canvas, st int) {
	ow := 2.4
	steel := mat{base: rgb(0xd0, 0xd8, 0xe4), gloss: 1, shine: 30, rim: 0.3}
	cab := func(top float64, col color.RGBA) sh {
		b := rrect(-46, top, 92, -top, 6)
		c.part(b, mat{base: col, rim: 0.3, tex: 0.06, texF: 0.1}, ow)
		return b
	}
	knob := func(x, y float64) { c.part(circ(x, y, 4.5), mat{base: rgb(0x3a, 0x3a, 0x44), gloss: 0.8}, 1.4) }
	switch st {
	case stCounter:
		cab(-90, rgb(0xf0, 0xd8, 0xa8))
		c.part(rrect(-38, -78, 34, 60, 4), mat{base: rgb(0xe0, 0xc4, 0x90)}, 1.6)
		c.part(rrect(4, -78, 34, 60, 4), mat{base: rgb(0xe0, 0xc4, 0x90)}, 1.6)
		knob(-8, -48)
		knob(8, -48)
		c.part(rrect(-50, -100, 100, 14, 4), mat{base: rgb(0xb8, 0x7a, 0x40), tex: 0.15, texF: 0.2}, ow)
		c.part(rrect(-36, -112, 62, 12, 4), mat{base: rgb(0xd8, 0x9a, 0x58), tex: 0.15, texF: 0.3}, ow)
		c.part(poly(-30, -118, 6, -122, 6, -114, -30, -114), mat{base: rgb(0xe0, 0xe8, 0xf8), gloss: 1}, 1.6)
		c.part(rrect(6, -122, 18, 8, 3), mat{base: rgb(0x5a, 0x30, 0x20)}, 1.6)
		c.part(circ(30, -120, 9), mat{base: colTomato, gloss: 0.8}, 1.8)
	case stStove:
		cab(-96, rgb(0xf8, 0xf8, 0xff))
		c.part(rrect(-36, -76, 72, 52, 6), mat{base: rgb(0x2a, 0x2a, 0x34), gloss: 0.8}, 1.8)
		for _, x := range []float64{-24, -8, 8, 24} {
			knob(x, -86)
		}
		c.part(rrect(-48, -104, 96, 10, 3), mat{base: rgb(0x3a, 0x3a, 0x44)}, ow)
		pot := rrect(-30, -152, 60, 48, 8)
		c.part(capsule(-36, -140, -30, -140, 4, 4).or(capsule(30, -140, 36, -140, 4, 4)), steel, 1.6)
		c.part(pot, steel, ow)
		c.part(ell(0, -152, 32, 7), mat{base: rgb(0xb8, 0xc0, 0xd0), gloss: 1}, ow)
	case stOven:
		cab(-150, rgb(0xd8, 0x50, 0x48))
		c.part(rrect(-38, -140, 76, 18, 4), mat{base: rgb(0x2a, 0x2a, 0x34)}, 1.6)
		for _, x := range []float64{-22, -6, 10, 26} {
			c.part(circ(x, -131, 5), mat{base: steel.base, gloss: 1}, 1.2)
		}
		c.part(rrect(-38, -114, 76, 80, 8), mat{base: rgb(0x3a, 0x22, 0x20), gloss: 0.5}, ow)
		c.part(rrect(-30, -106, 60, 46, 6), mat{base: rgb(0x4a, 0x30, 0x2a), gloss: 1.2, shine: 30, flat: 0.5}, 1.6)
		c.part(rrect(-30, -28, 60, 8, 4), steel, 1.6)
	case stGrill:
		c.part(line(3.4, 3.4, -30, 0, -24, -70).or(line(3.4, 3.4, 30, 0, 24, -70)), mat{base: rgb(0x3a, 0x3a, 0x44)}, ow)
		c.part(rrect(-34, -40, 68, 6, 2), mat{base: rgb(0x3a, 0x3a, 0x44)}, 1.6)
		bowl := ell(0, -80, 46, 30).and(rrect(-50, -80, 100, 40, 0))
		c.part(bowl, mat{base: rgb(0x2a, 0x2a, 0x34), gloss: 1, shine: 26}, ow)
		lid := ell(-10, -96, 40, 34).and(rrect(-60, -140, 100, 44, 0)).rot(-40, -90, -0.5)
		c.part(lid, mat{base: rgb(0x2a, 0x2a, 0x34), gloss: 1, shine: 26}, ow)
		for x := -38.0; x <= 38; x += 9 {
			c.fill(rrect(x, -86, 3, 6, 1), rgb(0x8a, 0x8a, 0x9a))
		}
		c.part(rrect(-44, -86, 88, 4, 2), mat{base: rgb(0x8a, 0x8a, 0x9a)}, 1.2)
	case stFreezer:
		b := rrect(-42, -190, 84, 190, 12)
		c.part(b, mat{base: rgb(0x9c, 0xdc, 0xff), rim: 0.3, gloss: 0.6}, ow)
		c.fill(rrect(-42, -128, 84, 3, 0), outlineOf(rgb(0x9c, 0xdc, 0xff)))
		c.part(rrect(24, -176, 8, 34, 4), steel, 1.6)
		c.part(rrect(24, -112, 8, 50, 4), steel, 1.6)
		c.fill(star5(-16, -160, 10, 3), alpha(colWhite, 0.9))
		c.soft(circ(-30, -40, 14), alpha(colWhite, 0.6), 10)
	}
}

func stationArt(st int) *pimg {
	return paintArt("station"+itoa(st), -60, -205, 60, 6, 1, func(c *canvas) { paintStation(c, st) })
}
