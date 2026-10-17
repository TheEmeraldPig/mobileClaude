package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

const groundY = 300.0 // horizon line in battle

// ridge returns a smooth wavy height for backgrounds.
func ridge(x float64, seed float64) float64 {
	return math.Sin(x*0.011+seed)*0.5 + math.Sin(x*0.023+seed*2.1)*0.3 + math.Sin(x*0.051+seed*3.7)*0.2
}

func ridgePoly(w, base, amp, seed, bottom float64) []float64 {
	pts := []float64{0, bottom}
	for x := 0.0; x <= w+10; x += 10 {
		pts = append(pts, x, base-amp*(0.5+0.5*ridge(x, seed)))
	}
	return append(pts, w, bottom)
}

func tree(dst *ebiten.Image, x, y, s float64, leaf color.RGBA) {
	shadow(dst, x, y, 20*s, 5*s, 0.25)
	roundRect(dst, rect{x - 5*s, y - 30*s, 10 * s, 32 * s}, 3*s, colInk)
	roundGrad(dst, rect{x - 3.5*s, y - 30*s, 7 * s, 30 * s}, 2*s, rgb(0x9a, 0x62, 0x38), rgb(0x5a, 0x34, 0x1a))
	for _, b := range [][3]float64{{-12, -34, 15}, {12, -36, 14}, {0, -50, 18}} {
		orb(dst, x+b[0]*s, y+b[1]*s, b[2]*s, b[2]*s*0.92, leaf)
	}
}

func pine(dst *ebiten.Image, x, y, s float64, leaf color.RGBA, snow bool) {
	roundGrad(dst, rect{x - 3*s, y - 14*s, 6 * s, 14 * s}, 2*s, rgb(0x8a, 0x5a, 0x34), rgb(0x5a, 0x34, 0x1a))
	for i := 0; i < 3; i++ {
		w, yy := (24-float64(i)*6)*s, y-12*s-float64(i)*14*s
		shape(dst, []float64{x - w, yy, x, yy - 24*s, x + w, yy}, light(leaf, 0.15), shade(leaf, 0.6), 2*s)
		if snow {
			shape(dst, []float64{x - w*0.45, yy - 13*s, x, yy - 24*s, x + w*0.45, yy - 13*s}, colWhite, rgb(0xd8, 0xe8, 0xf8), 0)
		}
	}
}

// battleBG paints a chapter's arena: sky, far mountains, themed scenery, a tiled floor.
func battleBG(ch int, w, h float64) *ebiten.Image {
	img := ebiten.NewImage(int(w*S), int(h*S))
	th := chapterTheme(ch)
	idx := (ch - 1) % len(themes)
	seed := float64(ch) * 1.7
	gradRect(img, 0, 0, w, groundY+20, th.Sky1, th.Sky2)
	sunC := rgb(0xff, 0xf4, 0xc0)
	if idx == 1 || idx == 5 {
		sunC = rgb(0xd8, 0xd0, 0xff)
	}
	glow(img, w*0.78, 96, 120, sunC, 0.55)
	disc(img, w*0.78, 96, 30, 30, alpha(sunC, 0.95))
	// far mountains
	shape(img, ridgePoly(w, groundY-70, 150, seed, groundY+20), mix(th.Far, th.Sky2, 0.25), mix(th.Far, th.Sky2, 0.6), 0)
	if idx == 4 || idx == 0 {
		for x := 40.0; x < w; x += 140 {
			pk := groundY - 70 - 150*(0.5+0.5*ridge(x, seed))
			shape(img, []float64{x - 22, pk + 26, x, pk, x + 22, pk + 26}, alpha(colWhite, 0.8), alpha(colWhite, 0.3), 0)
		}
	}
	// near hills
	shape(img, ridgePoly(w, groundY-10, 70, seed+4, groundY+30), th.Near, shade(th.Near, 0.75), 0)
	rng := rand.New(rand.NewSource(int64(ch)))
	switch idx {
	case 0: // woods
		for x := 10.0; x < w; x += 44 + rng.Float64()*30 {
			y := groundY - 10 - 70*(0.5+0.5*ridge(x, seed+4)) + 18
			tree(img, x, y, 0.7+rng.Float64()*0.4, th.Leaf)
		}
	case 1: // caves: stalactites and crystals
		for x := 0.0; x < w; x += 30 + rng.Float64()*30 {
			l := 30 + rng.Float64()*70
			shape(img, []float64{x - 14, 0, x + 14, 0, x + 2, l}, shade(th.Far, 0.9), shade(th.Far, 0.5), 1.5)
		}
		for i := 0; i < 6; i++ {
			x, y := rng.Float64()*w, groundY-10+rng.Float64()*20
			c := hue(0.55 + rng.Float64()*0.25)
			glow(img, x, y-14, 30, c, 0.5)
			shape(img, []float64{x - 8, y, x - 4, y - 30, x + 2, y - 38, x + 8, y}, light(c, 0.4), shade(c, 0.6), 1.5)
		}
	case 2: // pass: mesas and cacti
		for x := 30.0; x < w; x += 160 + rng.Float64()*60 {
			hh := 60 + rng.Float64()*50
			shape(img, []float64{x - 50, groundY + 10, x - 36, groundY - hh, x + 36, groundY - hh, x + 50, groundY + 10}, light(th.Near, 0.1), shade(th.Near, 0.6), 2)
		}
		for x := 70.0; x < w; x += 120 + rng.Float64()*80 {
			y := groundY + 6
			c := rgb(0x5a, 0xa0, 0x4a)
			roundRect(img, rect{x - 9, y - 50, 18, 52}, 9, colInk)
			roundGrad(img, rect{x - 7, y - 48, 14, 48}, 7, light(c, 0.2), shade(c, 0.7))
			roundRect(img, rect{x + 6, y - 38, 18, 10}, 5, colInk)
			roundRect(img, rect{x + 14, y - 50, 10, 20}, 5, colInk)
			roundGrad(img, rect{x + 7, y - 36, 16, 6}, 3, c, shade(c, 0.7))
			roundGrad(img, rect{x + 16, y - 48, 6, 16}, 3, c, shade(c, 0.7))
		}
	case 3: // swamp: dead trees and fog
		for x := 30.0; x < w; x += 90 + rng.Float64()*60 {
			y := groundY + 8
			c := rgb(0x4a, 0x3a, 0x34)
			line(img, x, y, x+4, y-70, 9, colInk)
			line(img, x, y, x+4, y-70, 6, c)
			line(img, x+2, y-40, x-20, y-60, 6, colInk)
			line(img, x+2, y-40, x-20, y-60, 3, c)
			line(img, x+4, y-56, x+24, y-80, 6, colInk)
			line(img, x+4, y-56, x+24, y-80, 3, c)
		}
		for i := 0; i < 3; i++ {
			gradRect(img, 0, groundY-30+float64(i)*14, w, 26, alpha(colWhite, 0), alpha(colWhite, 0.18))
		}
	case 4: // frost: snowy pines
		for x := 10.0; x < w; x += 40 + rng.Float64()*30 {
			y := groundY - 10 - 70*(0.5+0.5*ridge(x, seed+4)) + 16
			pine(img, x, y, 0.8+rng.Float64()*0.4, th.Leaf, true)
		}
	case 5: // keep: towers with lit windows
		for x := 40.0; x < w; x += 130 + rng.Float64()*40 {
			hh := 90 + rng.Float64()*60
			c := shade(th.Far, 0.8)
			roundGrad(img, rect{x - 24, groundY - hh, 48, hh + 10}, 2, light(c, 0.1), shade(c, 0.6))
			for i := 0; i < 4; i++ {
				roundRect(img, rect{x - 26 + float64(i)*14, groundY - hh - 12, 10, 14}, 1, c)
			}
			for i := 0; i < 2; i++ {
				wy := groundY - hh + 24 + float64(i)*34
				glow(img, x, wy, 14, colOrange, 0.6)
				roundRect(img, rect{x - 5, wy - 7, 10, 14}, 4, rgb(0xff, 0xc8, 0x60))
			}
		}
	}
	// floor
	gradRect(img, 0, groundY, w, h-groundY, th.Ground1, th.Ground2)
	rows := []float64{groundY}
	for y, step := groundY, 20.0; y < h; step *= 1.22 {
		y += step
		rows = append(rows, y)
	}
	for i := 0; i+1 < len(rows); i++ {
		y0, y1 := rows[i], rows[i+1]
		n := 9
		for j := -n; j <= n; j++ {
			xa0 := w/2 + float64(j)*(w/8)*(0.6+(y0-groundY)/400)
			xa1 := w/2 + float64(j+1)*(w/8)*(0.6+(y0-groundY)/400)
			xb0 := w/2 + float64(j)*(w/8)*(0.6+(y1-groundY)/400)
			xb1 := w/2 + float64(j+1)*(w/8)*(0.6+(y1-groundY)/400)
			v := 0.92 + rng.Float64()*0.16
			c := shade(mix(th.Ground1, th.Ground2, (y0-groundY)/(h-groundY)), v)
			shape(img, []float64{xa0 + 1, y0 + 1, xa1 - 1, y0 + 1, xb1 - 1, y1 - 1, xb0 + 1, y1 - 1}, light(c, 0.06), shade(c, 0.92), 0)
		}
	}
	gradRect(img, 0, groundY, w, 30, alpha(colInk, 0.35), clear)
	// spotlight on the arena and a vignette
	radial(img, w*0.5, 470, 260, 130, 0, 0, alpha(colWhite, 0.12), clear)
	gradRect(img, 0, h-260, w, 260, clear, alpha(colInk, 0.55))
	return img
}

// mapBG paints the run's map: themed ground with scenery dotted around.
func mapBG(r *Run, w, h float64) *ebiten.Image {
	img := ebiten.NewImage(int(w*S), int(h*S))
	th := chapterTheme(r.Chapter)
	idx := (r.Chapter - 1) % len(themes)
	gradRect(img, 0, 0, w, h, mix(th.Ground1, th.Sky2, 0.25), shade(th.Ground2, 0.9))
	rng := rand.New(rand.NewSource(r.Seed))
	for i := 0; i < 26; i++ { // soft patches
		x, y := rng.Float64()*w, rng.Float64()*h
		radial(img, x, y, 60+rng.Float64()*80, 30+rng.Float64()*40, 0, 0, alpha(light(th.Leaf, 0.2), 0.35), clear)
	}
	for i := 0; i < 34; i++ {
		x, y := rng.Float64()*w, rng.Float64()*h
		if x > 60 && x < w-60 && rng.Intn(3) > 0 {
			continue // keep the middle clear for the paths
		}
		s := 0.5 + rng.Float64()*0.35
		switch idx {
		case 4:
			pine(img, x, y, s, th.Leaf, true)
		case 1, 5:
			orb(img, x, y, 14*s, 10*s, shade(th.Far, 1.2))
		default:
			tree(img, x, y, s, th.Leaf)
		}
	}
	return img
}

// homeSky paints the home screen sky.
func homeSky(dst *ebiten.Image, w, h, t float64) {
	gradRect(dst, 0, 0, w, h, rgb(0x4c, 0xb8, 0xff), rgb(0xc8, 0xee, 0xff))
	cx, cy := w*0.8, 120.0
	for i := 0; i < 12; i++ {
		a := float64(i)*math.Pi/6 + t*0.08
		shape(dst, []float64{cx, cy, cx + math.Cos(a)*900, cy + math.Sin(a)*900, cx + math.Cos(a+0.16)*900, cy + math.Sin(a+0.16)*900}, alpha(colWhite, 0.14), clear, 0)
	}
	glow(dst, cx, cy, 140, rgb(0xff, 0xf6, 0xc0), 0.7)
	disc(dst, cx, cy, 36, 36, rgb(0xff, 0xf8, 0xd8))
	for i := 0; i < 6; i++ {
		x := math.Mod(float64(i)*230+t*10*(1+float64(i%3)*0.3), w+300) - 150
		y := 140 + float64(i)*140
		cloud(dst, x, y, 0.8+float64(i%3)*0.3)
	}
}

func cloud(dst *ebiten.Image, x, y, s float64) {
	for _, b := range [][3]float64{{-34, 6, 24}, {0, -6, 32}, {34, 6, 24}, {14, 10, 22}} {
		radial(dst, x+b[0]*s, y+b[1]*s, b[2]*s, b[2]*s*0.85, -0.2, -0.4, colWhite, rgb(0xd0, 0xe4, 0xf8))
	}
}

// island is the floating grass island the hero stands on at home.
func island(dst *ebiten.Image, x, y, w float64, t float64) {
	bob := math.Sin(t*1.2) * 4
	y += bob
	shape(dst, []float64{x - w, y, x - w*0.6, y + 70, x - w*0.2, y + 110, x + w*0.25, y + 96, x + w*0.6, y + 64, x + w, y}, rgb(0xb0, 0x7a, 0x4a), rgb(0x5a, 0x3a, 0x22), 3)
	for _, r := range [][3]float64{{-0.5, 40, 9}, {0.2, 70, 7}, {0.55, 34, 6}} {
		orb(dst, x+r[0]*w, y+r[1], r[2], r[2]*0.8, rgb(0x9a, 0x9a, 0xa8))
	}
	radial(dst, x, y, w+3, 30, 0, 0, colInk, colInk)
	radial(dst, x, y-2, w, 26, -0.2, -0.5, rgb(0x9a, 0xe8, 0x5a), rgb(0x3a, 0x9a, 0x30))
	for i := 0; i < 7; i++ {
		fx := x - w*0.8 + float64(i)*w*0.27
		fy := y - 4 + math.Sin(float64(i)*2)*8
		c := []color.RGBA{colRed, colYellow, colWhite, colPurple}[i%4]
		line(dst, fx, fy, fx, fy-8, 2, rgb(0x2a, 0x7a, 0x2a))
		orb(dst, fx, fy-10, 3.5, 3.5, c)
	}
}

// merchant draws the shopkeeper behind a counter.
func merchant(dst *ebiten.Image, x, y, t float64) {
	bob := math.Sin(t*2) * 2
	// backpack
	orb(dst, x+40, y-80+bob, 34, 40, rgb(0x9a, 0x6a, 0x3a))
	orb(dst, x+44, y-128+bob, 14, 10, rgb(0xd8, 0x4a, 0x4a))
	// robe
	shape(dst, []float64{x - 44, y, x - 30, y - 90 + bob, x + 30, y - 90 + bob, x + 44, y}, rgb(0x7a, 0x4a, 0xc8), rgb(0x3a, 0x1e, 0x6a), 3)
	// hood
	orb(dst, x, y-112+bob, 34, 32, rgb(0x6a, 0x3a, 0xb8))
	disc(dst, x, y-106+bob, 24, 20, rgb(0x1a, 0x10, 0x2a))
	for _, d := range []float64{-1, 1} {
		glow(dst, x+d*9, y-108+bob, 10, colYellow, 0.9)
		disc(dst, x+d*9, y-108+bob, 4, 3+math.Max(0, math.Sin(t*0.7))*1.5, rgb(0xff, 0xf0, 0x90))
	}
	// coin in hand
	coinIcon(dst, x-38, y-60+bob+math.Sin(t*3)*4, 10)
}

// campfire draws a fire with logs and a warm glow.
func campfire(dst *ebiten.Image, x, y, s, t float64) {
	glow(dst, x, y-20*s, 160*s, colOrange, 0.35+0.05*math.Sin(t*7))
	shadow(dst, x, y+6*s, 60*s, 12*s, 0.4)
	for i := 0; i < 7; i++ {
		a := float64(i) / 7 * 2 * math.Pi
		orb(dst, x+math.Cos(a)*44*s, y+math.Sin(a)*12*s, 9*s, 7*s, rgb(0x9a, 0x9a, 0xa8))
	}
	line(dst, x-40*s, y+6*s, x+40*s, y-8*s, 13*s, colInk)
	line(dst, x-40*s, y+6*s, x+40*s, y-8*s, 9*s, rgb(0x8a, 0x5a, 0x30))
	line(dst, x+40*s, y+6*s, x-40*s, y-8*s, 13*s, colInk)
	line(dst, x+40*s, y+6*s, x-40*s, y-8*s, 9*s, rgb(0x7a, 0x4a, 0x24))
	flames(dst, x, y-6*s, 34*s, t)
	flames(dst, x-14*s, y-2*s, 18*s, t+1.3)
	flames(dst, x+14*s, y-2*s, 20*s, t+2.1)
}

// ---- particles ----

type particle struct {
	x, y, vx, vy float64
	life, max    float64
	size         float64
	col          color.RGBA
	grav         float64
	kind         int // 0 dot, 1 star, 2 ring
}

func (p *particle) step() bool {
	p.x += p.vx
	p.y += p.vy
	p.vy += p.grav
	p.vx *= 0.98
	p.life--
	return p.life > 0
}

func (p *particle) draw(dst *ebiten.Image) {
	f := p.life / p.max
	switch p.kind {
	case 0:
		glow(dst, p.x, p.y, p.size*2.2, p.col, 0.6*f)
		circle(dst, p.x, p.y, p.size*f, alpha(light(p.col, 0.4), f))
	case 1:
		r := p.size * (0.5 + f*0.5)
		shape(dst, []float64{p.x, p.y - r, p.x + r*0.25, p.y - r*0.25, p.x + r, p.y, p.x + r*0.25, p.y + r*0.25, p.x, p.y + r, p.x - r*0.25, p.y + r*0.25, p.x - r, p.y, p.x - r*0.25, p.y - r*0.25}, alpha(colWhite, f), alpha(p.col, f), 0)
	case 2:
		strokeCircle(dst, p.x, p.y, p.size*(1.6-f), 4*f, alpha(p.col, f))
	}
}
