package game

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// Painted parts. Each body part is an outline that gets rasterised once into
// a small image: the distance to its edge becomes a height field, the height
// field gives surface normals, and the normals are lit like a little 3D model
// (moonlight from the top left, a cool rim light from behind, specular shine,
// surface grain and a soft dark edge). Parts are then rotated and moved by a
// skeleton every frame, so characters move like cut-out animation.

type material struct {
	base   color.RGBA
	spec   float64 // shine strength
	shine  float64 // shine tightness
	grain  float64 // surface texture
	bulge  float64 // how round the part is
	rim    float64 // rim light strength
	edge   color.RGBA
	glossy bool
}

func mat(base color.RGBA, spec, shine, grain, bulge float64) material {
	return material{base: base, spec: spec, shine: shine, grain: grain, bulge: bulge, rim: 0.7}
}

var (
	mSteel     = mat(rgb(0x8c, 0x94, 0xa8), 0.9, 30, 0.25, 0.75)
	mDarkSteel = mat(rgb(0x4a, 0x4c, 0x5a), 0.8, 26, 0.3, 0.75)
	mBlackIron = mat(rgb(0x2c, 0x2a, 0x36), 0.7, 22, 0.35, 0.8)
	mGold      = mat(rgb(0xc8, 0x96, 0x40), 1.0, 20, 0.2, 0.8)
	mBlade     = mat(rgb(0xc0, 0xc8, 0xd8), 1.2, 40, 0.12, 0.35)
	mCrimson   = mat(rgb(0x7a, 0x16, 0x22), 0.05, 6, 0.55, 0.5)
	mLeather   = mat(rgb(0x5a, 0x3a, 0x26), 0.15, 8, 0.6, 0.7)
	mBone      = mat(rgb(0xd8, 0xcc, 0xae), 0.25, 10, 0.5, 0.9)
	mHorn      = mat(rgb(0xc8, 0xb8, 0x98), 0.4, 14, 0.4, 0.9)
	mStone     = mat(rgb(0x6a, 0x6a, 0x76), 0.05, 4, 1.0, 0.8)
	mRobe      = mat(rgb(0x6a, 0x14, 0x20), 0.05, 4, 0.6, 0.45)
	mGhoulSkin = mat(rgb(0x7a, 0x86, 0x6c), 0.2, 8, 0.6, 0.95)
	mDemonSkin = mat(rgb(0x7a, 0x1c, 0x1e), 0.35, 12, 0.55, 0.95)
	mWing      = mat(rgb(0x4a, 0x12, 0x18), 0.15, 8, 0.5, 0.25)
	mFur       = mat(rgb(0x3e, 0x3a, 0x4a), 0.05, 4, 0.9, 0.85)
	mShroud    = mat(rgb(0x30, 0x34, 0x48), 0.05, 4, 0.5, 0.4)
	mWood      = mat(rgb(0x5a, 0x3a, 0x22), 0.1, 6, 0.8, 0.7)
	mClaw      = mat(rgb(0x2a, 0x24, 0x22), 0.6, 18, 0.2, 0.7)
)

func (m material) tint(h float64) material   { m.base = tint(m.base, h); return m }
func (m material) col(c color.RGBA) material { m.base = c; return m }

// partRes paints parts a little sharper than the screen so they stay crisp when
// characters are scaled up.
const partRes = 1.6

// part is one painted piece. pts outline it around its pivot (0,0) in layout units.
type part struct {
	pts    []float64
	m      material
	sharp  bool                                    // keep corners instead of smoothing
	detail func(dst *ebiten.Image, ox, oy float64) // extra painting on top, in local units shifted by (ox,oy)
	img    *ebiten.Image
	ox, oy float64
	s      float64
}

func newPart(m material, pts ...float64) *part { return &part{pts: pts, m: m} }
func sharpPart(m material, pts ...float64) *part {
	return &part{pts: pts, m: m, sharp: true}
}
func (p *part) with(d func(dst *ebiten.Image, ox, oy float64)) *part { p.detail = d; return p }

// spline turns control points into a dense closed outline (Catmull-Rom).
func spline(pts []float64, sharp bool) []float64 {
	n := len(pts) / 2
	if sharp || n < 3 {
		return pts
	}
	at := func(i int) (float64, float64) { i = (i%n + n) % n; return pts[2*i], pts[2*i+1] }
	var out []float64
	for i := 0; i < n; i++ {
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
	return out
}

// hash noise for surface grain
func grainAt(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return float64(h&0xffff) / 65535
}

func smoothGrain(x, y float64) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	a := grainAt(x0, y0)*(1-fx) + grainAt(x0+1, y0)*fx
	b := grainAt(x0, y0+1)*(1-fx) + grainAt(x0+1, y0+1)*fx
	return a*(1-fy) + b*fy
}

var (
	lightDir = norm3(-0.5, -0.7, 0.62)
	rimDir   = norm3(0.85, 0.15, 0.25)
	halfDir  = norm3(lightDir[0], lightDir[1], lightDir[2]+1)
	rimCol   = [3]float64{0.55, 0.62, 0.85}
)

func norm3(x, y, z float64) [3]float64 {
	l := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / l, y / l, z / l}
}

// build paints the part at the current screen scale.
func (p *part) build() {
	if p.img != nil && p.s == S {
		return
	}
	if p.img != nil {
		p.img.Deallocate()
	}
	p.s = S
	poly := spline(p.pts, p.sharp)
	kk := S * partRes
	minx, miny, maxx, maxy := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i := 0; i < len(poly); i += 2 {
		minx, maxx = math.Min(minx, poly[i]), math.Max(maxx, poly[i])
		miny, maxy = math.Min(miny, poly[i+1]), math.Max(maxy, poly[i+1])
	}
	const pad = 3.0
	p.ox, p.oy = -minx+pad, -miny+pad
	k := kk
	w := int(math.Ceil((maxx - minx + 2*pad) * k))
	h := int(math.Ceil((maxy - miny + 2*pad) * k))
	px := make([]float64, len(poly))
	for i := 0; i < len(poly); i += 2 {
		px[i], px[i+1] = (poly[i]+p.ox)*k, (poly[i+1]+p.oy)*k
	}
	// coverage by scanline with 4 sub-rows per pixel
	cov := make([]float32, w*h)
	n := len(px) / 2
	var xs []float64
	for row := 0; row < h; row++ {
		for sub := 0; sub < 4; sub++ {
			sy := float64(row) + (float64(sub)+0.5)/4
			xs = xs[:0]
			for i := 0; i < n; i++ {
				x0, y0 := px[2*i], px[2*i+1]
				j := (i + 1) % n
				x1, y1 := px[2*j], px[2*j+1]
				if (y0 <= sy && y1 > sy) || (y1 <= sy && y0 > sy) {
					xs = append(xs, x0+(sy-y0)/(y1-y0)*(x1-x0))
				}
			}
			sort.Float64s(xs)
			for i := 0; i+1 < len(xs); i += 2 {
				a, b := math.Max(0, xs[i]), math.Min(float64(w), xs[i+1])
				if b <= a {
					continue
				}
				ia, ib := int(a), int(b)
				base := row * w
				if ia == ib {
					cov[base+ia] += float32((b - a) / 4)
					continue
				}
				cov[base+ia] += float32((float64(ia+1) - a) / 4)
				for x := ia + 1; x < ib && x < w; x++ {
					cov[base+x] += 0.25
				}
				if ib < w {
					cov[base+ib] += float32((b - float64(ib)) / 4)
				}
			}
		}
	}
	// distance to the edge (two-pass chamfer)
	const big = 1e9
	dist := make([]float64, w*h)
	for i, c := range cov {
		if c > 0.5 {
			dist[i] = big
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if dist[i] == 0 {
				continue
			}
			d := dist[i]
			if x > 0 {
				d = math.Min(d, dist[i-1]+1)
			} else {
				d = 1
			}
			if y > 0 {
				d = math.Min(d, dist[i-w]+1)
				if x > 0 {
					d = math.Min(d, dist[i-w-1]+1.414)
				}
				if x < w-1 {
					d = math.Min(d, dist[i-w+1]+1.414)
				}
			} else {
				d = 1
			}
			dist[i] = d
		}
	}
	dmax := 1.0
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			if dist[i] == 0 {
				continue
			}
			d := dist[i]
			if x < w-1 {
				d = math.Min(d, dist[i+1]+1)
			} else {
				d = 1
			}
			if y < h-1 {
				d = math.Min(d, dist[i+w]+1)
				if x < w-1 {
					d = math.Min(d, dist[i+w+1]+1.414)
				}
				if x > 0 {
					d = math.Min(d, dist[i+w-1]+1.414)
				}
			} else {
				d = 1
			}
			dist[i] = d
			dmax = math.Max(dmax, d)
		}
	}
	// height field: a rounded dome over the distance
	ht := make([]float64, w*h)
	reach := math.Min(dmax, 14*k) * 0.95
	for i, d := range dist {
		if d > 0 {
			u := math.Min(1, d/reach)
			ht[i] = math.Sqrt(1-(1-u)*(1-u)) * reach * p.m.bulge
		}
	}
	m := p.m
	edge := m.edge
	if edge.A == 0 {
		edge = shade(m.base, 0.18)
	}
	br, bg, bb := float64(m.base.R)/255, float64(m.base.G)/255, float64(m.base.B)/255
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			c := float64(cov[i])
			if c <= 0.01 {
				continue
			}
			c = math.Min(1, c)
			hl := func(xx, yy int) float64 {
				if xx < 0 || yy < 0 || xx >= w || yy >= h {
					return 0
				}
				return ht[yy*w+xx]
			}
			dx := hl(x+1, y) - hl(x-1, y)
			dy := hl(x, y+1) - hl(x, y-1)
			nn := norm3(-dx, -dy, 2)
			diff := math.Max(0, nn[0]*lightDir[0]+nn[1]*lightDir[1]+nn[2]*lightDir[2])
			spec := m.spec * math.Pow(math.Max(0, nn[0]*halfDir[0]+nn[1]*halfDir[1]+nn[2]*halfDir[2]), m.shine)
			rim := m.rim * math.Pow(1-nn[2], 2.2) * math.Max(0, nn[0]*rimDir[0]+nn[1]*rimDir[1]+0.3)
			g := 1 + (smoothGrain(float64(x)/(1.6*k), float64(y)/(1.6*k))*0.7+grainAt(x, y)*0.3-0.5)*m.grain*0.6
			// soft occlusion toward the bottom-right and near edges
			occ := 0.75 + 0.25*math.Min(1, dist[i]/(5*k))
			lit := (0.2 + 0.95*diff) * g * occ
			r := br*lit + spec*0.95 + rim*rimCol[0]*0.6
			gg := bg*lit + spec*0.95 + rim*rimCol[1]*0.6
			b := bb*lit + spec + rim*rimCol[2]*0.6
			// painted dark edge
			if e := 1.3 * k; dist[i] < e {
				f := 1 - dist[i]/e
				f *= f
				r = r*(1-f) + float64(edge.R)/255*f
				gg = gg*(1-f) + float64(edge.G)/255*f
				b = b*(1-f) + float64(edge.B)/255*f
			}
			o := i * 4
			pix[o] = byte(math.Min(1, r) * 255 * c)
			pix[o+1] = byte(math.Min(1, gg) * 255 * c)
			pix[o+2] = byte(math.Min(1, b) * 255 * c)
			pix[o+3] = byte(255 * c)
		}
	}
	p.img = ebiten.NewImage(w, h)
	p.img.WritePixels(pix)
	if p.detail != nil {
		saved := S
		S = kk
		p.detail(p.img, p.ox, p.oy)
		S = saved
	}
}

// ---- skeleton transforms ----

// xf is a bone: a position, an angle and a scale, in layout units.
type xf struct{ x, y, a, k float64 }

func (t xf) at(lx, ly float64) (float64, float64) {
	c, s := math.Cos(t.a), math.Sin(t.a)
	return t.x + (lx*c-ly*s)*t.k, t.y + (lx*s+ly*c)*t.k
}

// j is a child joint at local (lx,ly), turned by da.
func (t xf) j(lx, ly, da float64) xf {
	x, y := t.at(lx, ly)
	return xf{x, y, t.a + da, t.k}
}

// draw stamps the part at bone t.
func (p *part) draw(dst *ebiten.Image, t xf) {
	p.build()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-p.ox*S*partRes, -p.oy*S*partRes)
	op.GeoM.Scale(t.k/partRes, t.k/partRes)
	op.GeoM.Rotate(t.a)
	op.GeoM.Translate(t.x*S, t.y*S)
	dst.DrawImage(p.img, op)
}

// drawTint stamps the part with a colour multiplier (for dimming the far limbs).
func (p *part) drawTint(dst *ebiten.Image, t xf, f float32) {
	p.build()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-p.ox*S*partRes, -p.oy*S*partRes)
	op.GeoM.Scale(t.k/partRes, t.k/partRes)
	op.GeoM.Rotate(t.a)
	op.GeoM.Translate(t.x*S, t.y*S)
	op.ColorScale.Scale(f, f, f, 1)
	dst.DrawImage(p.img, op)
}

// limb outlines a tapered limb hanging down from its pivot.
func limb(l, w0, w1 float64) []float64 {
	return []float64{-w0 * 0.5, -w0 * 0.3, 0, -w0 * 0.55, w0 * 0.5, -w0 * 0.3, w0*0.45 + (w1-w0)*0.2, l * 0.5, w1 * 0.5, l, 0, l + w1*0.4, -w1 * 0.5, l, -w0*0.45 - (w1-w0)*0.2, l * 0.5}
}

// ik solves a two-bone limb: from the root at (x0,y0) reach (x1,y1) with bone
// lengths l1,l2. bend picks which way the middle joint points. It returns the
// world angles of both bones (0 = pointing down).
func ik(x0, y0, x1, y1, l1, l2, bend float64) (float64, float64) {
	dx, dy := x1-x0, y1-y0
	d := math.Max(0.01, math.Min(l1+l2-0.01, math.Hypot(dx, dy)))
	base := math.Atan2(dy, dx)
	a1 := math.Acos(math.Max(-1, math.Min(1, (l1*l1+d*d-l2*l2)/(2*l1*d))))
	a2 := math.Acos(math.Max(-1, math.Min(1, (l1*l1+l2*l2-d*d)/(2*l1*l2))))
	upper := base - bend*a1
	lower := upper + bend*(math.Pi-a2)
	return upper - math.Pi/2, lower - math.Pi/2
}

// ---- easing ----

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
func easeOut(t float64) float64 { t = clamp01(t); return 1 - (1-t)*(1-t)*(1-t) }
func easeIn(t float64) float64  { t = clamp01(t); return t * t * t }
func easeInOut(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}
func backOut(t float64) float64 {
	t = clamp01(t)
	c := 1.70158
	return 1 + (c+1)*math.Pow(t-1, 3) + c*math.Pow(t-1, 2)
}
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// keys interpolates keyframes (time, value pairs, times rising) at t with smooth easing.
func keys(t float64, kv ...float64) float64 {
	if t <= kv[0] {
		return kv[1]
	}
	for i := 2; i+1 < len(kv); i += 2 {
		if t <= kv[i] {
			f := (t - kv[i-2]) / (kv[i] - kv[i-2])
			return lerp(kv[i-1], kv[i+1], easeInOut(f))
		}
	}
	return kv[len(kv)-1]
}
