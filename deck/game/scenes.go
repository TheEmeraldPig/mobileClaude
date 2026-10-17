package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

const groundY = 300.0 // horizon line in battle

// ridge returns a wavy height for skylines.
func ridge(x float64, seed float64) float64 {
	return math.Sin(x*0.011+seed)*0.5 + math.Sin(x*0.023+seed*2.1)*0.3 + math.Sin(x*0.061+seed*3.7)*0.2
}

func ridgePoly(w, base, amp, seed, bottom float64, jag bool) []float64 {
	pts := []float64{0, bottom}
	for x := 0.0; x <= w+12; x += 12 {
		y := base - amp*(0.5+0.5*ridge(x, seed))
		if jag && int(x/12)%2 == 1 {
			y -= amp * 0.12
		}
		pts = append(pts, x, y)
	}
	return append(pts, w, bottom)
}

// deadTree is a twisted, leafless silhouette.
func deadTree(dst *ebiten.Image, x, y, s float64, c color.RGBA, rng *rand.Rand) {
	var branch func(x0, y0, ang, l, w float64, d int)
	branch = func(x0, y0, ang, l, w float64, d int) {
		x1, y1 := x0+math.Cos(ang)*l, y0+math.Sin(ang)*l
		cx, cy := (x0+x1)/2+(rng.Float64()-0.5)*l*0.4, (y0+y1)/2+(rng.Float64()-0.5)*l*0.3
		curve(dst, x0, y0, cx, cy, x1, y1, w, c)
		if d == 0 || w < 1 {
			return
		}
		n := 2 + rng.Intn(2)
		for i := 0; i < n; i++ {
			branch(x1, y1, ang+(rng.Float64()-0.5)*1.4, l*(0.55+rng.Float64()*0.2), w*0.62, d-1)
		}
	}
	branch(x, y, -math.Pi/2+(rng.Float64()-0.5)*0.3, 46*s, 8*s, 4)
}

func pine(dst *ebiten.Image, x, y, s float64, c color.RGBA, snow bool) {
	line(dst, x, y, x, y-14*s, 4*s, c)
	for i := 0; i < 4; i++ {
		w, yy := (22-float64(i)*5)*s, y-10*s-float64(i)*13*s
		shape(dst, []float64{x - w, yy, x - w*0.4, yy - 8*s, x, yy - 24*s, x + w*0.4, yy - 8*s, x + w, yy}, c, c, 0)
		if snow {
			shape(dst, []float64{x - w*0.5, yy - 10*s, x, yy - 24*s, x + w*0.5, yy - 10*s, x, yy - 14*s}, alpha(colWhite, 0.7), alpha(colWhite, 0.4), 0)
		}
	}
}

func spire(dst *ebiten.Image, x, base, w, h float64, c color.RGBA, win color.RGBA) {
	shape(dst, []float64{x - w/2, base, x - w/2, base - h, x - w/2 - 6, base - h, x, base - h - w*1.6, x + w/2 + 6, base - h, x + w/2, base - h, x + w/2, base}, c, c, 0)
	for i := 0; i < int(h/40); i++ {
		wy := base - h + 20 + float64(i)*40
		glow(dst, x, wy, 12, win, 0.5)
		shape(dst, []float64{x - 3, wy + 6, x - 3, wy - 2, x, wy - 6, x + 3, wy - 2, x + 3, wy + 6}, light(win, 0.3), win, 0)
	}
}

func moonOf(idx int) color.RGBA {
	switch idx {
	case 2:
		return rgb(0xff, 0x9a, 0x50)
	case 3, 5:
		return rgb(0xff, 0x50, 0x50)
	case 1:
		return rgb(0xc0, 0x9a, 0xff)
	}
	return rgb(0xd8, 0xe4, 0xff)
}

// battleBG paints a chapter's arena: night sky, moon, skylines, themed scenery,
// fog, a stone floor and a heavy vignette.
func battleBG(ch int, w, h float64) *ebiten.Image {
	img := ebiten.NewImage(int(w*S), int(h*S))
	th := chapterTheme(ch)
	idx := (ch - 1) % len(themes)
	seed := float64(ch) * 1.7
	rng := rand.New(rand.NewSource(int64(ch)))
	gradRect(img, 0, 0, w, groundY+30, th.Sky1, th.Sky2)
	for i := 0; i < 60; i++ { // stars
		circle(img, rng.Float64()*w, rng.Float64()*groundY*0.7, 0.6+rng.Float64(), alpha(colWhite, 0.3+rng.Float64()*0.5))
	}
	moon := moonOf(idx)
	mx, my := w*0.72, 110.0
	glow(img, mx, my, 170, moon, 0.35)
	radial(img, mx, my, 34, 34, -0.3, -0.3, light(moon, 0.5), shade(moon, 0.75))
	for _, c := range [][3]float64{{-10, -6, 6}, {8, 10, 4}, {12, -12, 3}} {
		disc(img, mx+c[0], my+c[1], c[2], c[2], alpha(shade(moon, 0.7), 0.6))
	}
	for i := 0; i < 3; i++ { // cloud streaks across the moon
		y := 80 + float64(i)*36
		gradRect(img, 0, y, w, 10, clear, alpha(th.Sky2, 0.7))
		gradRect(img, 0, y+10, w, 8, alpha(th.Sky2, 0.7), clear)
	}
	far := ridgePoly(w, groundY-60, 160, seed, groundY+30, idx == 4 || idx == 2)
	shape(img, far, mix(th.Far, th.Sky2, 0.3), th.Far, 0)
	strokePath(img, pathOf(far[2:len(far)-2]), 1.2, alpha(light(moon, 0.2), 0.25))
	gradRect(img, 0, groundY-90, w, 80, clear, alpha(th.Sky2, 0.35))
	near := ridgePoly(w, groundY-8, 60, seed+4, groundY+30, false)
	shape(img, near, th.Near, shade(th.Near, 0.6), 0)
	sil := shade(th.Near, 0.55)
	switch idx {
	case 0: // haunted woods
		for x := 10.0; x < w; x += 50 + rng.Float64()*40 {
			deadTree(img, x, groundY+6, 0.8+rng.Float64()*0.6, sil, rng)
		}
	case 1: // crypt arches
		for x := 20.0; x < w; x += 120 {
			shape(img, []float64{x, groundY + 10, x, groundY - 110, x + 20, groundY - 110, x + 20, groundY + 10}, sil, sil, 0)
			p := []float64{x + 20, groundY - 110}
			for a := 0.0; a <= math.Pi; a += 0.2 {
				p = append(p, x+70-math.Cos(a)*50, groundY-110-math.Sin(a)*40)
			}
			p = append(p, x+120, groundY-110, x+120, groundY-130, x+20, groundY-130)
			shape(img, p, sil, sil, 0)
			glow(img, x+70, groundY-30, 30, colOrange, 0.4)
			disc(img, x+70, groundY-36, 3, 6, rgb(0xff, 0xd0, 0x80))
		}
	case 2: // ashen pass: jagged rocks and lava
		for x := 0.0; x < w; x += 70 + rng.Float64()*50 {
			hh := 40 + rng.Float64()*80
			shape(img, []float64{x - 30, groundY + 10, x - 12, groundY - hh, x, groundY - hh - 20, x + 10, groundY - hh + 10, x + 34, groundY + 10}, sil, sil, 0)
		}
		for i := 0; i < 3; i++ {
			lx := 60 + float64(i)*180
			glow(img, lx, groundY+4, 60, colOrange, 0.5)
			shape(img, []float64{lx - 40, groundY + 2, lx + 40, groundY + 2, lx + 20, groundY + 8, lx - 30, groundY + 8}, rgb(0xff, 0xc0, 0x40), rgb(0xff, 0x50, 0x10), 0)
		}
	case 3: // blood marsh
		for x := 30.0; x < w; x += 90 + rng.Float64()*60 {
			deadTree(img, x, groundY+8, 0.7+rng.Float64()*0.4, sil, rng)
		}
		for i := 0; i < 4; i++ {
			gradRect(img, 0, groundY-40+float64(i)*14, w, 26, clear, alpha(rgb(0x6a, 0x20, 0x30), 0.18))
		}
	case 4: // frozen wastes
		for x := 0.0; x < w; x += 34 + rng.Float64()*30 {
			pine(img, x, groundY+8, 0.8+rng.Float64()*0.5, sil, true)
		}
	case 5: // shadow keep
		for x := 40.0; x < w; x += 110 + rng.Float64()*40 {
			spire(img, x, groundY+10, 30+rng.Float64()*16, 90+rng.Float64()*90, sil, rgb(0xff, 0x6a, 0x30))
		}
	}
	gradRect(img, 0, groundY-30, w, 50, clear, alpha(mix(th.Sky2, colWhite, 0.1), 0.25))
	// stone floor in perspective
	texRect(img, 0, groundY, w, h-groundY, th.Ground1, th.Ground2)
	rows := []float64{groundY}
	for y, step := groundY, 18.0; y < h; step *= 1.24 {
		y += step
		rows = append(rows, y)
	}
	for i := 0; i+1 < len(rows); i++ {
		y0, y1 := rows[i], rows[i+1]
		for j := -10; j <= 10; j++ {
			sp0, sp1 := (w/7)*(0.6+(y0-groundY)/380), (w/7)*(0.6+(y1-groundY)/380)
			off := float64(i%2) * 0.5
			xa0, xa1 := w/2+(float64(j)+off)*sp0, w/2+(float64(j+1)+off)*sp0
			xb0, xb1 := w/2+(float64(j)+off)*sp1, w/2+(float64(j+1)+off)*sp1
			v := 0.75 + rng.Float64()*0.4
			c := shade(mix(th.Ground1, th.Ground2, (y0-groundY)/(h-groundY)), v)
			texPath(img, pathOf([]float64{xa0 + 1, y0 + 1, xa1 - 1, y0 + 1, xb1 - 1, y1 - 1, xb0 + 1, y1 - 1}), light(c, 0.08), shade(c, 0.8))
			if rng.Intn(9) == 0 { // a crack
				cx := (xa0 + xb1) / 2
				line(img, cx-6, y0+4, cx+4, (y0+y1)/2, 1, alpha(colInk, 0.6))
			}
		}
	}
	gradRect(img, 0, groundY, w, 40, alpha(colInk, 0.5), clear)
	radial(img, w*0.5, 520, 280, 140, 0, 0, alpha(light(moon, 0.1), 0.12), clear)
	vignette(img, w, h, 0.75)
	return img
}

// mapBG paints the run's map: dark ground, scattered silhouettes, drifting fog.
func mapBG(r *Run, w, h float64) *ebiten.Image {
	img := ebiten.NewImage(int(w*S), int(h*S))
	th := chapterTheme(r.Chapter)
	idx := (r.Chapter - 1) % len(themes)
	texRect(img, 0, 0, w, h, light(th.Ground1, 0.12), shade(th.Ground2, 0.9))
	rng := rand.New(rand.NewSource(r.Seed))
	for i := 0; i < 30; i++ {
		x, y := rng.Float64()*w, rng.Float64()*h
		radial(img, x, y, 70+rng.Float64()*90, 30+rng.Float64()*40, 0, 0, alpha(light(th.Leaf, 0.15), 0.35), clear)
	}
	for i := 0; i < 50; i++ { // cracks and old roads
		x, y := rng.Float64()*w, rng.Float64()*h
		a := rng.Float64() * math.Pi
		line(img, x, y, x+math.Cos(a)*30, y+math.Sin(a)*30, 1, alpha(colInk, 0.35))
	}
	sil := shade(th.Near, 0.7)
	for i := 0; i < 36; i++ {
		x, y := rng.Float64()*w, rng.Float64()*h
		if x > 50 && x < w-50 && rng.Intn(4) > 0 {
			continue
		}
		s := 0.4 + rng.Float64()*0.3
		shadow(img, x, y, 24*s, 6*s, 0.5)
		switch idx {
		case 4:
			pine(img, x, y, s, sil, true)
		case 1, 5:
			spire(img, x, y, 14*s, 40*s, sil, rgb(0xff, 0x6a, 0x30))
		default:
			deadTree(img, x, y, s, sil, rng)
		}
	}
	for i := 0; i < 8; i++ {
		y := rng.Float64() * h
		gradRect(img, 0, y, w, 40, clear, alpha(th.Sky2, 0.18))
		gradRect(img, 0, y+40, w, 40, alpha(th.Sky2, 0.18), clear)
	}
	vignette(img, w, h, 0.5)
	return img
}

// homeScene paints the home screen: a moonlit cliff over a ruined kingdom.
func homeScene(dst *ebiten.Image, w, h, t float64) {
	gradRect(dst, 0, 0, w, h*0.6, rgb(0x06, 0x06, 0x10), rgb(0x2a, 0x1c, 0x34))
	for i := 0; i < 90; i++ {
		x := math.Mod(float64(i)*137.5, w)
		y := math.Mod(float64(i)*61.7, h*0.5)
		circle(dst, x, y, 0.6+math.Mod(float64(i), 3)*0.4, alpha(colWhite, 0.25+0.35*(0.5+0.5*math.Sin(t*1.5+float64(i)))))
	}
	mx, my := w*0.74, 150.0
	glow(dst, mx, my, 220, rgb(0xff, 0x70, 0x60), 0.3)
	radial(dst, mx, my, 56, 56, -0.3, -0.3, rgb(0xff, 0xd8, 0xc8), rgb(0xd0, 0x50, 0x50))
	for _, c := range [][3]float64{{-16, -10, 9}, {14, 16, 6}, {20, -18, 5}} {
		disc(dst, mx+c[0], my+c[1], c[2], c[2], alpha(rgb(0xa0, 0x30, 0x30), 0.5))
	}
	for i := 0; i < 4; i++ {
		x := math.Mod(float64(i)*190+t*6, w+260) - 130
		y := 200 + float64(i)*60
		radial(dst, x, y, 150, 14, 0, 0, alpha(rgb(0x40, 0x30, 0x50), 0.6), clear)
	}
	// ruined castle on the horizon
	hz := h * 0.52
	far := ridgePoly(w, hz-30, 90, 3, hz+40, true)
	shape(dst, far, rgb(0x1a, 0x12, 0x22), rgb(0x0e, 0x0a, 0x14), 0)
	for i, x := range []float64{w * 0.18, w * 0.27, w * 0.36, w * 0.8} {
		spire(dst, x, hz+10, 22+float64(i%2)*10, 90+float64(i%3)*50, rgb(0x0c, 0x08, 0x12), rgb(0xff, 0x60, 0x30))
	}
	gradRect(dst, 0, hz-20, w, 80, clear, alpha(rgb(0x50, 0x30, 0x50), 0.4))
	gradRect(dst, 0, hz+40, w, h-hz, rgb(0x0c, 0x0a, 0x12), rgb(0x04, 0x04, 0x08))
}

// cliff is the rocky ledge the hero stands on at home, with a brazier.
func cliff(dst *ebiten.Image, x, y, w, t float64) {
	rock := rgb(0x2a, 0x26, 0x30)
	pts := []float64{x - w, y + 4, x - w*0.8, y - 6, x - w*0.4, y - 10, x, y - 8, x + w*0.5, y - 12, x + w*0.9, y - 4, x + w, y + 8, x + w*0.85, y + 60, x + w*0.5, y + 110, x + w*0.1, y + 150, x - w*0.3, y + 120, x - w*0.7, y + 70}
	texPath(dst, pathOf(pts), light(rock, 0.15), shade(rock, 0.3))
	strokePath(dst, pathOf(pts), 2, colInk)
	for i := 0; i < 6; i++ {
		cx := x - w*0.6 + float64(i)*w*0.25
		line(dst, cx, y+10, cx+8, y+50+float64(i%3)*20, 1.2, alpha(colInk, 0.6))
	}
	gradPath(dst, pathOf([]float64{x - w, y + 4, x - w*0.8, y - 6, x - w*0.4, y - 10, x, y - 8, x + w*0.5, y - 12, x + w*0.9, y - 4, x + w, y + 8}), alpha(rgb(0xff, 0x90, 0x70), 0.25), clear)
	// brazier
	bx, by := x+w*0.62, y-12
	glow(dst, bx, by-30, 110, colOrange, 0.35+0.05*math.Sin(t*8))
	shape(dst, []float64{bx - 18, by - 30, bx + 18, by - 30, bx + 10, by - 14, bx + 4, by, bx - 4, by, bx - 10, by - 14}, rgb(0x5a, 0x5a, 0x66), rgb(0x16, 0x16, 0x1e), 1.4)
	flames(dst, bx, by-30, 16, t)
}

func cloud(dst *ebiten.Image, x, y, s float64) {
	radial(dst, x, y, 120*s, 16*s, 0, 0, alpha(rgb(0x40, 0x30, 0x50), 0.6), clear)
}

// merchant draws the hooded shopkeeper with a lantern.
func merchant(dst *ebiten.Image, x, y, t float64) {
	merchantBody.draw(dst, x, y, anim{t: t})
}

// campfire draws a fire with logs, stones and a warm glow.
func campfire(dst *ebiten.Image, x, y, s, t float64) {
	glow(dst, x, y-20*s, 170*s, colOrange, 0.4+0.06*math.Sin(t*7))
	shadow(dst, x, y+6*s, 64*s, 12*s, 0.6)
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		orb(dst, x+math.Cos(a)*46*s, y+math.Sin(a)*12*s, 9*s, 7*s, rgb(0x4a, 0x46, 0x52))
	}
	line(dst, x-40*s, y+6*s, x+40*s, y-8*s, 12*s, colInk)
	line(dst, x-40*s, y+6*s, x+40*s, y-8*s, 9*s, rgb(0x4a, 0x2a, 0x18))
	line(dst, x+40*s, y+6*s, x-40*s, y-8*s, 12*s, colInk)
	line(dst, x+40*s, y+6*s, x-40*s, y-8*s, 9*s, rgb(0x3a, 0x22, 0x14))
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
	kind         int // 0 spark, 1 star, 2 ring
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
		glow(dst, p.x, p.y, p.size*2.4, p.col, 0.6*f)
		circle(dst, p.x, p.y, p.size*f*0.8, alpha(light(p.col, 0.5), f))
	case 1:
		r := p.size * (0.5 + f*0.5)
		shape(dst, []float64{p.x, p.y - r, p.x + r*0.2, p.y - r*0.2, p.x + r, p.y, p.x + r*0.2, p.y + r*0.2, p.x, p.y + r, p.x - r*0.2, p.y + r*0.2, p.x - r, p.y, p.x - r*0.2, p.y - r*0.2}, alpha(colWhite, f), alpha(p.col, f), 0)
	case 2:
		strokeCircle(dst, p.x, p.y, p.size*(1.6-f), 3*f, alpha(p.col, f))
	}
}

// ambience animates over a chapter's baked arena: drifting fog, the moon's
// pulse and the odd flash of distant lightning.
func ambience(dst *ebiten.Image, ch int, w, h, t float64) {
	th := chapterTheme(ch)
	idx := (ch - 1) % len(themes)
	moon := moonOf(idx)
	glow(dst, w*0.72, 110, 120+10*math.Sin(t*0.8), moon, 0.12+0.05*math.Sin(t*0.8))
	if idx == 0 || idx == 3 || idx == 5 {
		if ph := math.Mod(t+3, 13); ph < 0.35 {
			f := 1 - ph/0.35
			if ph > 0.12 && ph < 0.18 {
				f *= 0.3
			}
			gradRect(dst, 0, 0, w, groundY+40, alpha(rgb(0xc8, 0xd0, 0xff), 0.35*f), alpha(rgb(0xc8, 0xd0, 0xff), 0.1*f))
		}
	}
	fog := mix(th.Sky2, colWhite, 0.15)
	for i := 0; i < 5; i++ {
		sp := 8 + float64(i)*5
		x := math.Mod(t*sp+float64(i)*170, w+500) - 250
		y := groundY + 10 + float64(i)*float64(i)*18
		radial(dst, x, y, 240+float64(i)*30, 26+float64(i)*8, 0, 0, alpha(fog, 0.09), clear)
	}
}
