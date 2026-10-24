package game

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The sky: pixel-art backdrops for day, dusk and night blended by the time of day, a
// sun and moon that cross it, drifting clouds, stars, shooting stars, fireflies, and
// weather for events (snow, hearts, gold, aurora, heat, eclipse).

type mote struct{ x, y, vx, vy, t, life float64 }

type sky struct {
	bgs        [3]*ebiten.Image // night, dusk, day
	bgKey      [2]int
	q          float64 // backdrop pixel size
	sun, moon  *ebiten.Image
	clouds     []*ebiten.Image
	cloudPos   [][3]float64 // x (0-1.3), y (0-1), speed, as fractions of the screen
	stars      [][3]float64
	shooting   []mote
	fireflies  []mote
	weather    []mote
	weatherFor string
}

func (s *sky) init() {
	for i := 0; i < 70; i++ {
		s.stars = append(s.stars, [3]float64{rand.Float64(), rand.Float64() * 0.55, rand.Float64() * 7})
	}
	for i := 0; i < 3; i++ {
		s.clouds = append(s.clouds, ebiten.NewImageFromImage(cloudArt(i)))
	}
	for i := 0; i < 5; i++ {
		s.cloudPos = append(s.cloudPos, [3]float64{rand.Float64() * 1.3, 0.05 + rand.Float64()*0.3, 0.004 + rand.Float64()*0.006})
	}
	for i := 0; i < 16; i++ {
		s.fireflies = append(s.fireflies, mote{x: rand.Float64(), y: 0.7 + rand.Float64()*0.25, t: rand.Float64() * 7})
	}
	s.sun = ebiten.NewImageFromImage(discArt(15, false))
	s.moon = ebiten.NewImageFromImage(discArt(13, true))
}

// discArt is a pixel sun (with a glow ring) or moon (with craters).
func discArt(r int, moon bool) *image.RGBA {
	n := r*2 + 7
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	c := float64(n) / 2
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)+0.5-c, float64(y)+0.5-c)
			switch {
			case d < float64(r) && moon:
				col := rgb(0xf4ecd0)
				for _, k := range [][3]float64{{-0.35, -0.2, 0.22}, {0.3, 0.35, 0.16}, {0.4, -0.35, 0.1}} {
					if math.Hypot(float64(x)+0.5-c-k[0]*float64(r), float64(y)+0.5-c-k[1]*float64(r)) < k[2]*float64(r) {
						col = rgb(0xcfc6a8)
					}
				}
				if float64(x)-c+float64(y)-c > float64(r)*0.9 {
					col = mul(col, 0.85)
				}
				img.SetRGBA(x, y, col)
			case d < float64(r):
				col := rgb(0xffe066)
				if d < float64(r)*0.6 {
					col = rgb(0xfff4b0)
				}
				img.SetRGBA(x, y, col)
			case d < float64(r)+3 && !moon:
				img.SetRGBA(x, y, color.RGBA{255, 200, 80, 90})
			}
		}
	}
	return img
}

func cloudArt(v int) *image.RGBA {
	w, h := 34+v*8, 12+v*2
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(int64(v) + 3))
	type blob struct{ x, y, r float64 }
	var bs []blob
	for i := 0; i < 5+v; i++ {
		bs = append(bs, blob{4 + rng.Float64()*float64(w-8), float64(h) * (0.45 + rng.Float64()*0.2), 3 + rng.Float64()*float64(h)*0.35})
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for _, b := range bs {
				if math.Hypot(float64(x)-b.x, float64(y)-b.y) < b.r && y < h-1 {
					c := color.RGBA{255, 255, 255, 255}
					if float64(y) > b.y+b.r*0.3 {
						c = rgb(0xdfe6f5)
					}
					img.SetRGBA(x, y, c)
					break
				}
			}
		}
	}
	return img
}

// backdrop draws one sky (0 night, 1 dusk, 2 day) at low resolution, with hills.
func backdrop(kind, lw, lh int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, lw, lh))
	cols := [3][5]uint32{
		{0x0c0820, 0x1e1240, 0x3a1a50, 0x1a1030, 0x0e0818}, // sky top, mid, horizon, far hill, near hill
		{0x1b1036, 0x4a2160, 0xe0703a, 0x4a1f52, 0x2a1238},
		{0x3a78d8, 0x7ec4f0, 0xffe2b0, 0x5aa060, 0x3a7a40},
	}[kind]
	top, mid, low := rgb(cols[0]), rgb(cols[1]), rgb(cols[2])
	for y := 0; y < lh; y++ {
		t := float64(y) / float64(lh)
		var c color.RGBA
		if t < 0.5 {
			c = mix(top, mid, t/0.5)
		} else {
			c = mix(mid, low, math.Min(1, (t-0.5)/0.35))
		}
		band := math.Floor(t*14) / 14 // banding, for a pixel-art look
		c = mix(c, mix(top, low, band), 0.25)
		for x := 0; x < lw; x++ {
			cc := c
			fx := float64(x) / float64(lw)
			h1 := 0.70 + 0.06*math.Sin(fx*7+1) + 0.03*math.Sin(fx*17)
			h2 := 0.80 + 0.05*math.Sin(fx*5+3) + 0.02*math.Sin(fx*23+1)
			if t > h2 {
				cc = mix(rgb(cols[4]), mul(rgb(cols[4]), 0.7), (t-h2)*3)
			} else if t > h1 {
				cc = rgb(cols[3])
			}
			img.SetRGBA(x, y, cc)
		}
	}
	return img
}

func (s *sky) update(g *Game) {
	night := g.night()
	for i := range s.cloudPos {
		c := &s.cloudPos[i]
		c[0] -= c[2] * dt
		if c[0] < -0.3 {
			c[0], c[1] = 1.1+rand.Float64()*0.2, 0.05+rand.Float64()*0.3
		}
	}
	// shooting stars at night, lots during a meteor shower
	rate := 0.0
	if night {
		rate = 0.004
	}
	if g.st.Event == "meteor" {
		rate = 0.09
	}
	if rand.Float64() < rate {
		s.shooting = append(s.shooting, mote{x: 0.3 + rand.Float64()*0.8, y: rand.Float64() * 0.35, vx: -(0.5 + rand.Float64()*0.4), vy: 0.25 + rand.Float64()*0.2, life: 0.7 + rand.Float64()*0.5})
	}
	live := s.shooting[:0]
	for _, m := range s.shooting {
		m.t += dt
		m.x += m.vx * dt
		m.y += m.vy * dt
		if m.t < m.life {
			live = append(live, m)
		}
	}
	s.shooting = live
	for i := range s.fireflies {
		f := &s.fireflies[i]
		f.t += dt
		f.x += math.Sin(f.t*0.7+float64(i))*0.0006 + 0.0001
		f.y += math.Cos(f.t*0.9+float64(i)*2) * 0.0004
		if f.x > 1.02 {
			f.x = -0.02
		}
	}
	// weather for the current event
	if s.weatherFor != g.st.Event {
		s.weatherFor, s.weather = g.st.Event, nil
	}
	want := map[string]int{"blizzard": 140, "bloom": 26, "goldrush": 50}[g.st.Event]
	for len(s.weather) < want {
		// spread over the screen when the event starts; after that each one wraps
		// back to the top (or the bottom, for hearts)
		s.weather = append(s.weather, mote{x: rand.Float64(), y: rand.Float64(), t: rand.Float64() * 7})
	}
	for i := range s.weather {
		m := &s.weather[i]
		m.t += dt
		switch g.st.Event {
		case "blizzard":
			m.y += 0.12 * dt
			m.x += (math.Sin(m.t*1.5)*0.04 - 0.05) * dt
		case "bloom":
			m.y -= 0.06 * dt
			m.x += math.Sin(m.t*2) * 0.02 * dt
		case "goldrush":
			m.y += 0.1 * dt
		}
		if m.y > 1.05 || m.y < -0.25 || m.x < -0.05 || m.x > 1.05 {
			m.x = rand.Float64()
			if g.st.Event == "bloom" {
				m.y = 1.05
			} else {
				m.y = -0.05
			}
		}
	}
}

func (g *Game) drawSky(dst *ebiten.Image) {
	s := &g.sky
	q := math.Max(2, math.Round(g.u*2))
	lw, lh := int(math.Ceil(g.W/q)), int(math.Ceil(g.H/q))
	if s.bgs[0] == nil || s.bgKey != [2]int{lw, lh} {
		s.bgKey, s.q = [2]int{lw, lh}, q
		for k := range s.bgs {
			s.bgs[k] = ebiten.NewImageFromImage(backdrop(k, lw, lh))
		}
	}
	d := g.daylight()
	// blend night -> dusk -> day
	a, b, k := 0, 1, d*2
	if d > 0.5 {
		a, b, k = 1, 2, (d-0.5)*2
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(q, q)
	dst.DrawImage(s.bgs[a], op)
	op.ColorScale.ScaleAlpha(float32(k))
	dst.DrawImage(s.bgs[b], op)

	// stars and shooting stars come out as it gets dark
	night := 1 - d
	sz := math.Max(1, math.Round(g.u*1.5))
	for _, st := range s.stars {
		a := (0.4 + 0.6*math.Abs(math.Sin(g.t*1.3+st[2]))) * night
		if a > 0.02 {
			fill(dst, math.Round(st[0]*g.W), math.Round(st[1]*g.H), sz, sz, color.NRGBA{255, 240, 220, uint8(a * 255)})
		}
	}
	for _, m := range s.shooting {
		fade := 1 - m.t/m.life
		x, y := m.x*g.W, m.y*g.H
		tx, ty := x-m.vx*g.W*0.12, y-m.vy*g.H*0.12
		vector.StrokeLine(dst, float32(x), float32(y), float32(tx), float32(ty), float32(math.Max(1, 1.5*g.u)), color.NRGBA{255, 240, 200, uint8(200 * fade)}, true)
		fill(dst, math.Round(x)-sz/2, math.Round(y)-sz/2, sz*1.5, sz*1.5, color.NRGBA{255, 255, 255, uint8(255 * fade)})
	}
	if g.st.Event == "aurora" {
		for band := 0; band < 3; band++ {
			c := []color.NRGBA{{80, 255, 160, 40}, {120, 160, 255, 34}, {200, 120, 255, 30}}[band]
			for x := 0.0; x < g.W; x += q * 2 {
				y := g.H * (0.12 + 0.05*float64(band) + 0.04*math.Sin(x/g.W*6+g.t*0.6+float64(band)))
				fill(dst, x, y, q*2, g.H*0.08, c)
			}
		}
	}

	// sun by day, moon by night, crossing the sky
	ph := math.Mod(g.st.Clock, dayLen) / dayLen
	img, k2 := s.sun, ph/0.5
	if ph >= 0.5 {
		img, k2 = s.moon, (ph-0.5)/0.5
	}
	bx, by := g.W*(0.06+0.6*k2), g.H*(0.66-0.5*math.Sin(math.Pi*k2))
	op = &ebiten.DrawImageOptions{}
	n := float64(img.Bounds().Dx())
	op.GeoM.Translate(-n/2, -n/2)
	op.GeoM.Scale(q, q)
	op.GeoM.Translate(math.Round(bx), math.Round(by))
	if g.st.Event == "eclipse" && img == s.sun {
		op.ColorScale.Scale(0.15, 0.1, 0.2, 1)
	}
	dst.DrawImage(img, op)

	// clouds: white by day, dim and purple at night
	for i, c := range s.cloudPos {
		img := s.clouds[i%len(s.clouds)]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(q, q)
		op.GeoM.Translate(math.Round(c[0]*g.W), math.Round(c[1]*g.H))
		op.ColorScale.Scale(float32(0.35+0.65*d), float32(0.3+0.7*d), float32(0.5+0.5*d), float32(0.45+0.45*d))
		dst.DrawImage(img, op)
	}
	// fireflies over the hills at night
	if night > 0.3 {
		for _, f := range s.fireflies {
			a := night * (0.5 + 0.5*math.Sin(f.t*3))
			fill(dst, math.Round(f.x*g.W), math.Round(f.y*g.H), sz*1.3, sz*1.3, color.NRGBA{220, 255, 120, uint8(a * 230)})
		}
	}
}

// drawWeather draws the event's weather over the board.
func (g *Game) drawWeather(dst *ebiten.Image) {
	s := &g.sky
	u := g.u
	switch g.st.Event {
	case "blizzard":
		for _, m := range s.weather {
			sz := math.Max(1, math.Round(u*(1.5+math.Mod(m.t, 1.5))))
			fill(dst, math.Round(m.x*g.W), math.Round(m.y*g.H), sz, sz, color.NRGBA{240, 250, 255, 220})
		}
		fill(dst, 0, 0, g.W, g.H, color.NRGBA{200, 230, 255, 18})
	case "bloom":
		for _, m := range s.weather {
			icon(dst, "heart", m.x*g.W, m.y*g.H, 9*u)
		}
	case "goldrush":
		for _, m := range s.weather {
			a := 0.5 + 0.5*math.Sin(m.t*8)
			sz := math.Max(1, math.Round(u*2))
			fill(dst, math.Round(m.x*g.W), math.Round(m.y*g.H), sz, sz, color.NRGBA{255, 210, 80, uint8(120 + 120*a)})
		}
	case "heatwave":
		fill(dst, 0, 0, g.W, g.H, color.NRGBA{255, 120, 40, uint8(22 + 10*math.Sin(g.t*2))})
	case "eclipse":
		fill(dst, 0, 0, g.W, g.H, color.NRGBA{10, 0, 20, 70})
	}
}

// drawSunMoon is the little sun or moon in the world chip.
func (g *Game) drawSunMoon(dst *ebiten.Image, x, y, r float64, night bool) {
	img := g.sky.sun
	if night {
		img = g.sky.moon
	}
	if img == nil {
		return
	}
	n := float64(img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-n/2, -n/2)
	op.GeoM.Scale(2*r/n, 2*r/n)
	op.GeoM.Translate(x, y)
	dst.DrawImage(img, op)
}
