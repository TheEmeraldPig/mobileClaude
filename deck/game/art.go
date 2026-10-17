package game

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Drawing kit. Everything is drawn in code: gradient-filled paths, radial light,
// a stone/metal grain texture, glows. The look is dark fantasy: near-black
// surfaces, iron and gold trim, ember and arcane light.
// Fonts: Cinzel and Barlow Condensed (SIL Open Font License, see FONT_LICENSE.txt).

//go:embed Cinzel-Bold.ttf
var titleTTF []byte

//go:embed Cinzel-Black.ttf
var heavyTTF []byte

//go:embed BarlowCondensed-SemiBold.ttf
var bodyTTF []byte

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

var (
	colInk    = rgb(0x07, 0x06, 0x0c)
	colWhite  = rgb(0xff, 0xff, 0xff)
	colBone   = rgb(0xe8, 0xdc, 0xc0) // main text
	colCream  = rgb(0xc8, 0xb8, 0x98) // secondary text
	colDim    = rgb(0x8a, 0x80, 0x90)
	colYellow = rgb(0xf0, 0xc0, 0x5a)
	colOrange = rgb(0xff, 0x7a, 0x2a) // ember
	colGreen  = rgb(0x5a, 0xd0, 0x4a) // toxic
	colBlue   = rgb(0x3a, 0xa8, 0xff) // arcane
	colRed    = rgb(0xd8, 0x2a, 0x3a) // blood
	colPurple = rgb(0x9a, 0x5c, 0xff)
	colGrey   = rgb(0x6a, 0x6a, 0x7a)
	colGem    = rgb(0xe0, 0x30, 0x60)
	colGold   = rgb(0xd8, 0xa8, 0x50)
	colGoldHi = rgb(0xff, 0xe8, 0xa8)
	colGoldLo = rgb(0x6a, 0x4a, 0x1e)
	colSteel  = rgb(0xa8, 0xb0, 0xc0)
	colIron   = rgb(0x2a, 0x2a, 0x34)
	colStone  = rgb(0x24, 0x22, 0x2c)
	colSkin   = rgb(0xc8, 0xa8, 0x90)
	clear     = color.RGBA{}
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A)}
}

func shade(c color.RGBA, f float64) color.RGBA {
	m := func(v uint8) uint8 { return uint8(math.Max(0, math.Min(float64(c.A), float64(v)*f))) }
	return color.RGBA{m(c.R), m(c.G), m(c.B), c.A}
}

func light(c color.RGBA, t float64) color.RGBA { return mix(c, alpha(colWhite, float64(c.A)/255), t) }

// alpha scales a (premultiplied) colour's opacity.
func alpha(c color.RGBA, a float64) color.RGBA {
	a = math.Max(0, math.Min(1, a))
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

func hue(h float64) color.RGBA {
	h = math.Mod(math.Mod(h, 1)+1, 1) * 6
	f := func(n float64) uint8 {
		k := math.Mod(n+h, 6)
		return uint8(255 * (1 - math.Max(0, math.Min(math.Min(k, 4-k), 1))*0.7))
	}
	return color.RGBA{f(5), f(3), f(1), 0xff}
}

// tint rotates a colour's hue by h turns.
func tint(c color.RGBA, h float64) color.RGBA {
	if h == 0 {
		return c
	}
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	v, d := mx, mx-mn
	if d == 0 {
		return c
	}
	s := d / mx
	var hh float64
	switch mx {
	case r:
		hh = math.Mod((g-b)/d, 6)
	case g:
		hh = (b-r)/d + 2
	default:
		hh = (r-g)/d + 4
	}
	hh = math.Mod(hh/6+h+1, 1) * 6
	x := v * s * (1 - math.Abs(math.Mod(hh, 2)-1))
	m := v - v*s
	var rr, gg, bb float64
	switch int(hh) {
	case 0:
		rr, gg, bb = v*s, x, 0
	case 1:
		rr, gg, bb = x, v*s, 0
	case 2:
		rr, gg, bb = 0, v*s, x
	case 3:
		rr, gg, bb = 0, x, v*s
	case 4:
		rr, gg, bb = x, 0, v*s
	default:
		rr, gg, bb = v*s, 0, x
	}
	return color.RGBA{uint8((rr + m) * 255), uint8((gg + m) * 255), uint8((bb + m) * 255), c.A}
}

var whiteImg = func() *ebiten.Image {
	i := ebiten.NewImage(3, 3)
	i.Fill(color.White)
	return i
}()

var whiteSub = whiteImg.SubImage(rectI(1, 1, 2, 2)).(*ebiten.Image)

// noiseTex is a tileable grain used to give surfaces a stone or metal feel.
var noiseTex = func() *ebiten.Image {
	const n = 256
	val := func(grid, x, y int, seed uint32) float64 {
		h := uint32((x%grid)*73856093) ^ uint32((y%grid)*19349663) ^ seed
		h ^= h >> 13
		h *= 0x5bd1e995
		h ^= h >> 15
		return float64(h&0xffff) / 65535
	}
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := 0.0
			amp, tot := 1.0, 0.0
			for o, grid := range []int{8, 16, 32, 64, 128} {
				cell := float64(n) / float64(grid)
				fx, fy := float64(x)/cell, float64(y)/cell
				x0, y0 := int(fx), int(fy)
				tx, ty := fx-float64(x0), fy-float64(y0)
				tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
				s := uint32(o * 7919)
				a := val(grid, x0, y0, s)*(1-tx) + val(grid, x0+1, y0, s)*tx
				b := val(grid, x0, y0+1, s)*(1-tx) + val(grid, x0+1, y0+1, s)*tx
				v += (a*(1-ty) + b*ty) * amp
				tot += amp
				amp *= 0.6
			}
			v /= tot
			g := byte(255 * (0.6 + 0.4*v))
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = g, g, g, 255
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}()

func vtx(x, y float64, c color.RGBA) ebiten.Vertex {
	return ebiten.Vertex{DstX: float32(x * S), DstY: float32(y * S), SrcX: 1.5, SrcY: 1.5,
		ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
}

// S is the scale from layout units (540 wide) to real screen pixels. Everything is
// drawn at the screen's own resolution, so shapes and text stay sharp without the
// GPU cost of anti-aliasing.
var S = 1.0

var triAA = &ebiten.DrawTrianglesOptions{}
var triAdd = &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter}
var triTex = &ebiten.DrawTrianglesOptions{Address: ebiten.AddressRepeat}

// clip returns the part of dst inside r, for drawing that must not spill out.
func clip(dst *ebiten.Image, r rect) *ebiten.Image {
	return dst.SubImage(image.Rect(int(r.x*S), int(r.y*S), int(math.Ceil((r.x+r.w)*S)), int(math.Ceil((r.y+r.h)*S)))).(*ebiten.Image)
}

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(x*S), float32(y*S), float32(w*S), float32(h*S), c, false)
}

func strokeCircle(dst *ebiten.Image, x, y, r, w float64, c color.RGBA) {
	vector.StrokeCircle(dst, float32(x*S), float32(y*S), float32(r*S), float32(w*S), c, false)
}

// imgOp places an image drawn at screen resolution at layout position (x,y) with a scale.
func imgOp(x, y, scale float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x*S, y*S)
	return op
}

// radial fills an ellipse with a gradient from cin at the highlight point
// (offset hx,hy as a fraction of the radii) to cout at the rim.
func radial(dst *ebiten.Image, cx, cy, rx, ry, hx, hy float64, cin, cout color.RGBA) {
	radialOpt(dst, cx, cy, rx, ry, hx, hy, cin, cout, triAA)
}

func radialOpt(dst *ebiten.Image, cx, cy, rx, ry, hx, hy float64, cin, cout color.RGBA, op *ebiten.DrawTrianglesOptions) {
	const seg = 40
	vs := make([]ebiten.Vertex, 0, seg+2)
	is := make([]uint16, 0, seg*3)
	vs = append(vs, vtx(cx+hx*rx, cy+hy*ry, cin))
	for i := 0; i <= seg; i++ {
		a := float64(i) / seg * 2 * math.Pi
		vs = append(vs, vtx(cx+math.Cos(a)*rx, cy+math.Sin(a)*ry, cout))
		if i > 0 {
			is = append(is, 0, uint16(i), uint16(i+1))
		}
	}
	dst.DrawTriangles(vs, is, whiteSub, op)
}

// glow adds soft light around a point.
func glow(dst *ebiten.Image, cx, cy, r float64, c color.RGBA, a float64) {
	radialOpt(dst, cx, cy, r, r, 0, 0, alpha(c, a), clear, triAdd)
}

// shadow is a soft dark ellipse.
func shadow(dst *ebiten.Image, cx, cy, rx, ry, a float64) {
	radial(dst, cx, cy, rx, ry, 0, 0, color.RGBA{0, 0, 0, uint8(255 * a)}, clear)
}

func disc(dst *ebiten.Image, cx, cy, rx, ry float64, c color.RGBA) {
	radial(dst, cx, cy, rx, ry, 0, 0, c, c)
}

// orb draws a lit ball with a thin dark rim.
func orb(dst *ebiten.Image, cx, cy, rx, ry float64, base color.RGBA) {
	disc(dst, cx, cy, rx+1.6, ry+1.6, colInk)
	radial(dst, cx, cy, rx, ry, -0.35, -0.42, light(base, 0.3), shade(base, 0.35))
	radial(dst, cx-rx*0.38, cy-ry*0.45, rx*0.3, ry*0.22, 0, 0, alpha(colWhite, 0.5), clear)
}

func circle(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	vector.DrawFilledCircle(dst, float32(x*S), float32(y*S), float32(r*S), c, false)
}

// ---- paths ----

func pathOf(pts []float64) *vector.Path {
	p := &vector.Path{}
	for i := 0; i+1 < len(pts); i += 2 {
		if i == 0 {
			p.MoveTo(float32(pts[0]), float32(pts[1]))
		} else {
			p.LineTo(float32(pts[i]), float32(pts[i+1]))
		}
	}
	p.Close()
	return p
}

// smoothPath runs a closed curve through the points (midpoint quadratic smoothing).
func smoothPath(pts []float64) *vector.Path {
	p := &vector.Path{}
	n := len(pts) / 2
	if n < 3 {
		return pathOf(pts)
	}
	at := func(i int) (float64, float64) { i = (i%n + n) % n; return pts[2*i], pts[2*i+1] }
	x0, y0 := at(0)
	x1, y1 := at(1)
	p.MoveTo(float32((x0+x1)/2), float32((y0+y1)/2))
	for i := 1; i <= n; i++ {
		cx, cy := at(i)
		nx, ny := at(i + 1)
		p.QuadTo(float32(cx), float32(cy), float32((cx+nx)/2), float32((cy+ny)/2))
	}
	p.Close()
	return p
}

func fillPath(dst *ebiten.Image, p *vector.Path, c color.RGBA) { gradPath(dst, p, c, c) }

func colorVerts(vs []ebiten.Vertex, top, bot color.RGBA, tex bool) {
	y0, y1 := float32(math.Inf(1)), float32(math.Inf(-1))
	for _, v := range vs {
		y0, y1 = min(y0, v.DstY), max(y1, v.DstY)
	}
	for i := range vs {
		t := 0.0
		if y1 > y0 {
			t = float64((vs[i].DstY - y0) / (y1 - y0))
		}
		c := mix(top, bot, t)
		if tex {
			vs[i].SrcX, vs[i].SrcY = vs[i].DstX*0.8, vs[i].DstY*0.8
		} else {
			vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
		}
		vs[i].DstX, vs[i].DstY = vs[i].DstX*float32(S), vs[i].DstY*float32(S)
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
}

// gradPath fills a path with a top-to-bottom gradient.
func gradPath(dst *ebiten.Image, p *vector.Path, top, bot color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 {
		return
	}
	colorVerts(vs, top, bot, false)
	dst.DrawTriangles(vs, is, whiteSub, triAA)
}

// texPath fills a path with the grain texture shaded by a gradient.
func texPath(dst *ebiten.Image, p *vector.Path, top, bot color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 {
		return
	}
	colorVerts(vs, top, bot, true)
	dst.DrawTriangles(vs, is, noiseTex, triTex)
}

func strokePath(dst *ebiten.Image, p *vector.Path, w float64, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(w), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	for i := range vs {
		vs[i].DstX, vs[i].DstY = vs[i].DstX*float32(S), vs[i].DstY*float32(S)
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, whiteSub, triAA)
}

// shape fills a polygon with a gradient and a thin dark outline.
func shape(dst *ebiten.Image, pts []float64, top, bot color.RGBA, outline float64) {
	p := pathOf(pts)
	if outline > 0 {
		strokePath(dst, p, outline*1.4, colInk)
	}
	gradPath(dst, p, top, bot)
}

// blob is shape with smooth curves through the points.
func blob(dst *ebiten.Image, pts []float64, top, bot color.RGBA, outline float64) {
	p := smoothPath(pts)
	if outline > 0 {
		strokePath(dst, p, outline*1.4, colInk)
	}
	gradPath(dst, p, top, bot)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.RGBA) {
	p := &vector.Path{}
	p.MoveTo(float32(x0), float32(y0))
	p.LineTo(float32(x1), float32(y1))
	strokePath(dst, p, w, c)
}

// curve draws a quadratic curve stroke.
func curve(dst *ebiten.Image, x0, y0, cx, cy, x1, y1, w float64, c color.RGBA) {
	p := &vector.Path{}
	p.MoveTo(float32(x0), float32(y0))
	p.QuadTo(float32(cx), float32(cy), float32(x1), float32(y1))
	strokePath(dst, p, w, c)
}

// rot rotates points around (cx,cy).
func rot(pts []float64, cx, cy, a float64) []float64 {
	o := make([]float64, len(pts))
	c, s := math.Cos(a), math.Sin(a)
	for i := 0; i+1 < len(pts); i += 2 {
		x, y := pts[i]-cx, pts[i+1]-cy
		o[i], o[i+1] = cx+x*c-y*s, cy+x*s+y*c
	}
	return o
}

// mirror flips points around x = cx.
func mirror(pts []float64, cx float64) []float64 {
	o := make([]float64, len(pts))
	for i := 0; i+1 < len(pts); i += 2 {
		o[i], o[i+1] = 2*cx-pts[i], pts[i+1]
	}
	return o
}

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }
func (r rect) cx() float64           { return r.x + r.w/2 }
func (r rect) cy() float64           { return r.y + r.h/2 }
func (r rect) grow(d float64) rect   { return rect{r.x - d, r.y - d, r.w + 2*d, r.h + 2*d} }

func roundPath(r rect, rad float64) *vector.Path {
	p := &vector.Path{}
	rad = math.Max(0.5, math.Min(rad, math.Min(r.w, r.h)/2))
	x, y, w, h, k := float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(rad)
	p.MoveTo(x+k, y)
	p.LineTo(x+w-k, y)
	p.Arc(x+w-k, y+k, k, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(x+w, y+h-k)
	p.Arc(x+w-k, y+h-k, k, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(x+k, y+h)
	p.Arc(x+k, y+h-k, k, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(x, y+k)
	p.Arc(x+k, y+k, k, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
	return p
}

// chamfer is a rectangle with its corners cut off: the basic frame shape.
func chamfer(r rect, c float64) *vector.Path {
	c = math.Min(c, math.Min(r.w, r.h)/2)
	return pathOf([]float64{r.x + c, r.y, r.x + r.w - c, r.y, r.x + r.w, r.y + c, r.x + r.w, r.y + r.h - c, r.x + r.w - c, r.y + r.h, r.x + c, r.y + r.h, r.x, r.y + r.h - c, r.x, r.y + c})
}

func roundRect(dst *ebiten.Image, r rect, rad float64, c color.RGBA) {
	fillPath(dst, roundPath(r, rad), c)
}
func roundGrad(dst *ebiten.Image, r rect, rad float64, top, bot color.RGBA) {
	gradPath(dst, roundPath(r, rad), top, bot)
}

func gradRect(dst *ebiten.Image, x, y, w, h float64, top, bot color.RGBA) {
	vs := []ebiten.Vertex{vtx(x, y, top), vtx(x+w, y, top), vtx(x, y+h, bot), vtx(x+w, y+h, bot)}
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, whiteSub, nil)
}

// texRect fills a rectangle with the grain texture shaded by a gradient.
func texRect(dst *ebiten.Image, x, y, w, h float64, top, bot color.RGBA) {
	texPath(dst, pathOf([]float64{x, y, x + w, y, x + w, y + h, x, y + h}), top, bot)
}

// vignette darkens the edges of the screen.
func vignette(dst *ebiten.Image, w, h, a float64) {
	gradRect(dst, 0, 0, w, h*0.18, alpha(colInk, a), clear)
	gradRect(dst, 0, h*0.75, w, h*0.25, clear, alpha(colInk, a*1.2))
	for _, d := range []float64{0, 1} {
		vs := []ebiten.Vertex{vtx(d*w, 0, alpha(colInk, a*0.8)), vtx(d*w+(0.5-d)*w*0.4, 0, clear), vtx(d*w, h, alpha(colInk, a*0.8)), vtx(d*w+(0.5-d)*w*0.4, h, clear)}
		dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, whiteSub, nil)
	}
}

// ---- text ----

func mustFace(b []byte) *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return s
}

var (
	titleSrc = mustFace(titleTTF)
	heavySrc = mustFace(heavyTTF)
	bodySrc  = mustFace(bodyTTF)
	faces    = map[[2]float64]*text.GoTextFace{}
)

const (
	fBody = iota
	fTitle
	fHeavy
)

func face(src int, size float64) *text.GoTextFace {
	size = math.Round(size*2) / 2
	k := [2]float64{float64(src), size}
	f, ok := faces[k]
	if !ok {
		f = &text.GoTextFace{Source: []*text.GoTextFaceSource{bodySrc, titleSrc, heavySrc}[src], Size: size}
		faces[k] = f
	}
	return f
}

func drawFont(dst *ebiten.Image, src int, s string, x, y, size float64, c color.Color, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*S, y*S)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * S * 1.15
	text.Draw(dst, s, face(src, size*S), op)
}

// drawText draws body text (condensed sans): ax is 0 left, 0.5 centre, 1 right; y is the middle.
func drawText(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	drawFont(dst, fBody, s, x, y, size, c, ax)
}

func textWidth(s string, size float64) float64 {
	w, _ := text.Measure(s, face(fBody, size), size*1.15)
	return w
}

func titleWidth(s string, size float64) float64 {
	w, _ := text.Measure(s, face(fTitle, size), size*1.15)
	return w
}

// say draws a title-font label with a soft dark shadow.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	src := fTitle
	if size >= 30 {
		src = fHeavy
	}
	o := math.Max(1, size/16)
	sh := color.RGBA{0, 0, 0, 0xb0}
	for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {0, 2}, {1, 2}, {-1, 2}} {
		drawFont(dst, src, s, x+d[0]*o, y+d[1]*o, size, sh, ax)
	}
	drawFont(dst, src, s, x, y, size, c, ax)
}

// sayBody is body text with the same shadow, for small labels on busy art.
func sayBody(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	sh := color.RGBA{0, 0, 0, 0xc0}
	for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, 1.5}} {
		drawFont(dst, fBody, s, x+d[0], y+d[1], size, sh, ax)
	}
	drawFont(dst, fBody, s, x, y, size, c, ax)
}

func wrap(s string, n int) string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		for len(ln) > n {
			cut := strings.LastIndex(ln[:n], " ")
			if cut <= 0 {
				break
			}
			out = append(out, ln[:cut])
			ln = ln[cut+1:]
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
