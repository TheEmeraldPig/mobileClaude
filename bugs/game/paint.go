package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The painter. All art is painted once on the CPU into images at the screen's real
// resolution, then moved around with transforms. Shapes are signed distance fields
// (negative inside), so every edge is smooth, outlines are exact and shading can
// treat each shape as a soft rounded volume: a toy-like cartoon look with dark
// coloured outlines, wrap-around light, a bounce light on the shaded side and a
// glossy highlight.

// sh is a shape: a signed distance in units and the box it fits in.
type sh struct {
	d              func(x, y float64) float64
	x0, y0, x1, y1 float64
}

func circ(cx, cy, r float64) sh {
	return sh{func(x, y float64) float64 { return math.Hypot(x-cx, y-cy) - r }, cx - r, cy - r, cx + r, cy + r}
}

// ell is an ellipse (a close approximation of the true distance).
func ell(cx, cy, rx, ry float64) sh {
	return sh{func(x, y float64) float64 {
		px, py := (x-cx)/rx, (y-cy)/ry
		k0 := math.Hypot(px, py)
		k1 := math.Hypot(px/rx, py/ry)
		if k1 == 0 {
			return -math.Min(rx, ry)
		}
		return k0 * (k0 - 1) / k1
	}, cx - rx, cy - ry, cx + rx, cy + ry}
}

func rrect(x, y, w, h, r float64) sh {
	cx, cy, hx, hy := x+w/2, y+h/2, w/2-r, h/2-r
	return sh{func(px, py float64) float64 {
		qx, qy := math.Abs(px-cx)-hx, math.Abs(py-cy)-hy
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	}, x, y, x + w, y + h}
}

// capsule is a line from a to b, ra thick at a and rb at b.
func capsule(ax, ay, bx, by, ra, rb float64) sh {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	return sh{func(x, y float64) float64 {
		t := 0.0
		if l2 > 0 {
			t = math.Max(0, math.Min(1, ((x-ax)*dx+(y-ay)*dy)/l2))
		}
		return math.Hypot(x-ax-dx*t, y-ay-dy*t) - (ra + (rb-ra)*t)
	}, math.Min(ax-ra, bx-rb), math.Min(ay-ra, by-rb), math.Max(ax+ra, bx+rb), math.Max(ay+ra, by+rb)}
}

// poly is a closed polygon given as x,y pairs.
func poly(p ...float64) sh {
	n := len(p) / 2
	s := sh{x0: math.Inf(1), y0: math.Inf(1), x1: math.Inf(-1), y1: math.Inf(-1)}
	for i := 0; i < n; i++ {
		s.x0, s.x1 = math.Min(s.x0, p[2*i]), math.Max(s.x1, p[2*i])
		s.y0, s.y1 = math.Min(s.y0, p[2*i+1]), math.Max(s.y1, p[2*i+1])
	}
	s.d = func(x, y float64) float64 {
		d := math.Inf(1)
		in := false
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			ax, ay, bx, by := p[2*j], p[2*j+1], p[2*i], p[2*i+1]
			ex, ey := bx-ax, by-ay
			wx, wy := x-ax, y-ay
			t := 0.0
			if l := ex*ex + ey*ey; l > 0 {
				t = math.Max(0, math.Min(1, (wx*ex+wy*ey)/l))
			}
			d = math.Min(d, math.Hypot(wx-ex*t, wy-ey*t))
			if (ay > y) != (by > y) && x < ax+(y-ay)*ex/ey {
				in = !in
			}
		}
		if in {
			return -d
		}
		return d
	}
	return s
}

// blob is a smooth closed curve through the points (Catmull-Rom).
func blob(p ...float64) sh { return poly(smoothPts(p, true)...) }

func smoothPts(p []float64, closed bool) []float64 {
	n := len(p) / 2
	at := func(i int) (float64, float64) {
		if closed {
			i = (i%n + n) % n
		} else {
			i = max(0, min(n-1, i))
		}
		return p[2*i], p[2*i+1]
	}
	var out []float64
	segs := n
	if !closed {
		segs = n - 1
	}
	for i := 0; i < segs; i++ {
		x0, y0 := at(i - 1)
		x1, y1 := at(i)
		x2, y2 := at(i + 1)
		x3, y3 := at(i + 2)
		for k := 0; k < 8; k++ {
			t := float64(k) / 8
			t2, t3 := t*t, t*t*t
			f := func(a, b, c, d float64) float64 {
				return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
			}
			out = append(out, f(x0, x1, x2, x3), f(y0, y1, y2, y3))
		}
	}
	if !closed {
		out = append(out, p[len(p)-2], p[len(p)-1])
	}
	return out
}

// line is a smooth open curve through the points with a thickness going from w0 to w1.
func line(w0, w1 float64, p ...float64) sh {
	q := p
	if len(p) > 4 {
		q = smoothPts(p, false)
	}
	n := len(q)/2 - 1
	var s sh
	for i := 0; i < n; i++ {
		a, b := float64(i)/float64(n), float64(i+1)/float64(n)
		c := capsule(q[2*i], q[2*i+1], q[2*i+2], q[2*i+3], w0+(w1-w0)*a, w0+(w1-w0)*b)
		if i == 0 {
			s = c
		} else {
			s = s.or(c)
		}
	}
	return s
}

// arc is a stroke along part of an ellipse, angles in radians (0 = right, clockwise).
func arc(cx, cy, rx, ry, a0, a1, w float64) sh {
	var p []float64
	for i := 0; i <= 10; i++ {
		a := a0 + (a1-a0)*float64(i)/10
		p = append(p, cx+math.Cos(a)*rx, cy+math.Sin(a)*ry)
	}
	return line(w, w, p...)
}

func (a sh) or(b sh) sh {
	return sh{func(x, y float64) float64 { return math.Min(a.d(x, y), b.d(x, y)) },
		math.Min(a.x0, b.x0), math.Min(a.y0, b.y0), math.Max(a.x1, b.x1), math.Max(a.y1, b.y1)}
}

// melt joins two shapes with a soft fillet of size k.
func (a sh) melt(b sh, k float64) sh {
	return sh{func(x, y float64) float64 {
		da, db := a.d(x, y), b.d(x, y)
		h := math.Max(0, math.Min(1, 0.5+0.5*(db-da)/k))
		return db*(1-h) + da*h - k*h*(1-h)
	}, math.Min(a.x0, b.x0), math.Min(a.y0, b.y0), math.Max(a.x1, b.x1), math.Max(a.y1, b.y1)}
}

func (a sh) and(b sh) sh {
	return sh{func(x, y float64) float64 { return math.Max(a.d(x, y), b.d(x, y)) },
		math.Max(a.x0, b.x0), math.Max(a.y0, b.y0), math.Min(a.x1, b.x1), math.Min(a.y1, b.y1)}
}

func (a sh) minus(b sh) sh {
	return sh{func(x, y float64) float64 { return math.Max(a.d(x, y), -b.d(x, y)) }, a.x0, a.y0, a.x1, a.y1}
}

func (a sh) grow(r float64) sh {
	return sh{func(x, y float64) float64 { return a.d(x, y) - r }, a.x0 - r, a.y0 - r, a.x1 + r, a.y1 + r}
}

// ring turns a shape into its outline, w thick.
func (a sh) ring(w float64) sh {
	return sh{func(x, y float64) float64 { return math.Abs(a.d(x, y)) - w/2 }, a.x0 - w, a.y0 - w, a.x1 + w, a.y1 + w}
}

func (a sh) move(dx, dy float64) sh {
	return sh{func(x, y float64) float64 { return a.d(x-dx, y-dy) }, a.x0 + dx, a.y0 + dy, a.x1 + dx, a.y1 + dy}
}

// rot turns a shape by an angle (radians) around cx,cy.
func (a sh) rot(cx, cy, ang float64) sh {
	c, s := math.Cos(ang), math.Sin(ang)
	r := 0.0
	for _, p := range [][2]float64{{a.x0, a.y0}, {a.x1, a.y0}, {a.x0, a.y1}, {a.x1, a.y1}} {
		r = math.Max(r, math.Hypot(p[0]-cx, p[1]-cy))
	}
	return sh{func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		return a.d(cx+dx*c+dy*s, cy-dx*s+dy*c)
	}, cx - r, cy - r, cx + r, cy + r}
}

// flipX mirrors a shape around the vertical line x = cx.
func (a sh) flipX(cx float64) sh {
	return sh{func(x, y float64) float64 { return a.d(2*cx-x, y) }, 2*cx - a.x1, a.y0, 2*cx - a.x0, a.y1}
}

// fuzz roughens the edge, for fur and fluff.
func (a sh) fuzz(amp, freq float64) sh {
	return sh{func(x, y float64) float64 {
		return a.d(x, y) + amp*(vnoise(x*freq, y*freq)-0.5)*2
	}, a.x0 - amp, a.y0 - amp, a.x1 + amp, a.y1 + amp}
}

// vnoise is smooth value noise in 0..1.
func vnoise(x, y float64) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	tx, ty := x-x0, y-y0
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	h := func(i, j float64) float64 {
		n := uint32(int32(i)*374761393) ^ uint32(int32(j)*668265263)
		n = (n ^ n>>13) * 1274126177
		return float64(n&0xffff) / 65535
	}
	a := h(x0, y0)*(1-tx) + h(x0+1, y0)*tx
	b := h(x0, y0+1)*(1-tx) + h(x0+1, y0+1)*tx
	return a*(1-ty) + b*ty
}

// ---- canvas ----

type canvas struct {
	w, h   int
	k      float64 // pixels per unit
	ox, oy float64 // unit at pixel 0,0
	pix    []float32
}

// newCanvas makes a canvas showing units x0..x1, y0..y1 at k pixels per unit.
func newCanvas(x0, y0, x1, y1, k float64) *canvas {
	w, h := int(math.Ceil((x1-x0)*k)), int(math.Ceil((y1-y0)*k))
	w, h = max(w, 1), max(h, 1)
	return &canvas{w: w, h: h, k: k, ox: x0, oy: y0, pix: make([]float32, w*h*4)}
}

func (c *canvas) image() *ebiten.Image {
	b := make([]byte, len(c.pix))
	for i, v := range c.pix {
		b[i] = byte(math.Max(0, math.Min(255, float64(v)*255+0.5)))
	}
	img := ebiten.NewImage(c.w, c.h)
	img.WritePixels(b)
	return img
}

// each calls fn for every pixel near shape s with the unit position and distance in pixels.
func (c *canvas) each(s sh, fn func(i int, x, y, d float64)) {
	pad := 2 / c.k
	px0 := max(0, int((s.x0-pad-c.ox)*c.k))
	py0 := max(0, int((s.y0-pad-c.oy)*c.k))
	px1 := min(c.w, int(math.Ceil((s.x1+pad-c.ox)*c.k)))
	py1 := min(c.h, int(math.Ceil((s.y1+pad-c.oy)*c.k)))
	// Skip empty blocks: test the block centre against its radius first.
	const B = 8
	for by := py0; by < py1; by += B {
		for bx := px0; bx < px1; bx += B {
			ex, ey := min(bx+B, px1), min(by+B, py1)
			cx := c.ox + float64(bx+ex)/2/c.k
			cy := c.oy + float64(by+ey)/2/c.k
			if s.d(cx, cy)*c.k > float64(B)*0.75+1.5 {
				continue
			}
			for py := by; py < ey; py++ {
				y := c.oy + (float64(py)+0.5)/c.k
				for px := bx; px < ex; px++ {
					x := c.ox + (float64(px)+0.5)/c.k
					d := s.d(x, y) * c.k
					if d < 1 {
						fn((py*c.w+px)*4, x, y, d)
					}
				}
			}
		}
	}
}

// put blends a straight colour with coverage a over pixel i.
func (c *canvas) put(i int, r, g, b, a float64) {
	if a <= 0 {
		return
	}
	a = math.Min(a, 1)
	ia := float32(1 - a)
	p := c.pix[i : i+4 : i+4]
	p[0] = float32(r*a) + p[0]*ia
	p[1] = float32(g*a) + p[1]*ia
	p[2] = float32(b*a) + p[2]*ia
	p[3] = float32(a) + p[3]*ia
}

// add brightens pixel i (for glows), keeping alpha.
func (c *canvas) add(i int, r, g, b, a float64) {
	p := c.pix[i : i+4 : i+4]
	p[3] = float32(math.Min(1, float64(p[3])+a))
	p[0] = min(p[3], p[0]+float32(r*a))
	p[1] = min(p[3], p[1]+float32(g*a))
	p[2] = min(p[3], p[2]+float32(b*a))
}

func cov(d float64) float64 { return math.Max(0, math.Min(1, 0.5-d)) }

func f3(col color.RGBA) (float64, float64, float64) {
	return float64(col.R) / 255, float64(col.G) / 255, float64(col.B) / 255
}

// fill paints a shape in a flat colour (a scales opacity via col.A).
func (c *canvas) fill(s sh, col color.RGBA) {
	r, g, b := f3(col)
	o := float64(col.A) / 255
	c.each(s, func(i int, _, _, d float64) { c.put(i, r, g, b, cov(d)*o) })
}

// grad fills a shape with a vertical gradient between y0 and y1.
func (c *canvas) grad(s sh, top, bot color.RGBA, y0, y1 float64) {
	r0, g0, b0 := f3(top)
	r1, g1, b1 := f3(bot)
	a0, a1 := float64(top.A)/255, float64(bot.A)/255
	c.each(s, func(i int, _, y, d float64) {
		t := math.Max(0, math.Min(1, (y-y0)/(y1-y0)))
		c.put(i, r0+(r1-r0)*t, g0+(g1-g0)*t, b0+(b1-b0)*t, cov(d)*(a0+(a1-a0)*t))
	})
}

// soft paints a blurred shape: full colour inside, fading over w units outside.
func (c *canvas) soft(s sh, col color.RGBA, w float64) {
	r, g, b := f3(col)
	o := float64(col.A) / 255
	c.each(s.grow(w), func(i int, x, y, d float64) {
		dd := s.d(x, y)
		t := math.Max(0, math.Min(1, 1-dd/w))
		c.put(i, r, g, b, t*t*(3-2*t)*o)
	})
}

// glowIn adds light that fades from the shape's middle outwards over w units.
func (c *canvas) glowIn(s sh, col color.RGBA, w, a float64) {
	r, g, b := f3(col)
	c.each(s.grow(w), func(i int, x, y, d float64) {
		dd := s.d(x, y)
		t := math.Max(0, math.Min(1, 1-dd/w))
		c.add(i, r, g, b, t*t*a)
	})
}

// mat describes how a volume is shaded.
type mat struct {
	base   color.RGBA
	dark   color.RGBA // shadow side; zero picks a cool darker shade of base
	gloss  float64    // highlight strength
	shine  float64    // highlight tightness (bigger = smaller spot)
	rim    float64    // bounce light on the shaded edge
	flat   float64    // 0 = fully rounded, 1 = flat colour
	depth  float64    // how far in the rounding goes, in units; 0 = whole shape
	tex    float64    // grain amount
	texF   float64    // grain frequency
	stripe func(x, y float64) float64
}

func darkOf(c color.RGBA) color.RGBA {
	d := mix(shade(c, 0.52), rgb(0x3a, 0x1c, 0x6a), 0.28)
	return d
}

// outlineOf is the cartoon outline colour for a base colour: a deep version of it.
func outlineOf(c color.RGBA) color.RGBA {
	return mix(shade(c, 0.28), rgb(0x24, 0x10, 0x3a), 0.5)
}

var lightDir = func() [3]float64 {
	x, y, z := -0.48, -0.66, 0.58
	l := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / l, y / l, z / l}
}()

// vol paints a shaded rounded volume.
func (c *canvas) vol(s sh, m mat) { c.volOn(s, s, m) }

// volOn paints shape s coloured by m but shaped (lit) like the volume n, so markings
// such as spots and stripes follow the body they sit on.
func (c *canvas) volOn(s, n sh, m mat) {
	depth := m.depth * c.k
	if depth == 0 {
		// The deepest point of n, found on a coarse grid.
		md := 0.0
		for i := 0; i <= 16; i++ {
			for j := 0; j <= 16; j++ {
				x := n.x0 + (n.x1-n.x0)*float64(i)/16
				y := n.y0 + (n.y1-n.y0)*float64(j)/16
				md = math.Min(md, n.d(x, y))
			}
		}
		depth = math.Max(-md*c.k, 1)
	}
	dark := m.dark
	if dark.A == 0 {
		dark = darkOf(m.base)
	}
	br, bg, bb := f3(m.base)
	dr, dg, db := f3(dark)
	hr, hg, hb := f3(light(m.base, 0.35))
	shine := m.shine
	if shine == 0 {
		shine = 24
	}
	L := lightDir
	hx, hy, hz := L[0], L[1], L[2]+1
	hl := math.Sqrt(hx*hx + hy*hy + hz*hz)
	hx, hy, hz = hx/hl, hy/hl, hz/hl
	e := 0.6 / c.k
	c.each(s, func(i int, x, y, d float64) {
		a := cov(d)
		if a <= 0 {
			return
		}
		nd := n.d(x, y) * c.k
		gx := n.d(x+e, y) - n.d(x-e, y)
		gy := n.d(x, y+e) - n.d(x, y-e)
		gl := math.Hypot(gx, gy)
		if gl > 0 {
			gx, gy = gx/gl, gy/gl
		}
		t := math.Max(0, math.Min(1, -nd/depth))
		ed := 1 - t
		ed = 1 - (1-ed)*(1-ed)*0.15 - (1-ed)*0.85 // keep a soft dome all the way in
		ed *= 1 - m.flat
		nz := math.Sqrt(math.Max(0, 1-ed*ed))
		nx, ny := gx*ed, gy*ed
		diff := nx*L[0] + ny*L[1] + nz*L[2]
		lam := diff*0.5 + 0.5
		k := smooth(0.18, 0.78, lam)
		r := dr + (br-dr)*k
		g := dg + (bg-dg)*k
		b := db + (bb-db)*k
		if hi := smooth(0.82, 1.0, lam) * 0.55; hi > 0 {
			r, g, b = r+(hr-r)*hi, g+(hg-g)*hi, b+(hb-b)*hi
		}
		if m.rim > 0 {
			rim := math.Pow(ed, 3) * math.Max(0, gx*0.55+gy*0.8) * m.rim
			r, g, b = r+(hr-r)*rim, g+(hg-g)*rim, b+(hb-b)*rim
		}
		if m.stripe != nil {
			v := m.stripe(x, y)
			r, g, b = r*v, g*v, b*v
		}
		if m.tex > 0 {
			f := m.texF
			if f == 0 {
				f = 0.6
			}
			v := 1 + (vnoise(x*f, y*f)-0.5)*m.tex + (vnoise(x*f*3.1, y*f*3.1)-0.5)*m.tex*0.6
			r, g, b = r*v, g*v, b*v
		}
		if m.gloss > 0 {
			sp := math.Max(0, nx*hx+ny*hy+nz*hz)
			s := math.Pow(sp, shine) * m.gloss
			s = math.Min(1, s*1.6)
			r, g, b = r+(1-r)*s, g+(1-g)*s, b+(1-b)*s
		}
		c.put(i, r, g, b, a)
	})
}

func smooth(e0, e1, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

// part paints a volume with its cartoon outline underneath (ow units wide).
func (c *canvas) part(s sh, m mat, ow float64) {
	c.fill(s.grow(ow), outlineOf(m.base))
	c.vol(s, m)
}

// shine paints a soft white highlight blob.
func (c *canvas) shine(s sh, a float64) {
	c.each(s.grow(1.5/c.k), func(i int, x, y, d float64) {
		c.put(i, 1, 1, 1, cov(d)*a)
	})
}

// rgba converts the canvas to a Go image (for tests and the app icon).
func (c *canvas) rgba() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.w, c.h))
	for i, v := range c.pix {
		img.Pix[i] = byte(math.Max(0, math.Min(255, float64(v)*255+0.5)))
	}
	return img
}
