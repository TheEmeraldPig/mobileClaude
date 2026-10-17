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

// Drawing kit: everything is drawn in code with shaded shapes. Radial gradients
// come from triangle fans whose centre sits on the highlight, so a circle reads
// as a lit 3D ball. Lilita One font (SIL Open Font License, see FONT_LICENSE.txt).

//go:embed LilitaOne-Regular.ttf
var fontTTF []byte

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

var (
	colInk    = rgb(0x1f, 0x1a, 0x33)
	colWhite  = rgb(0xff, 0xff, 0xff)
	colCream  = rgb(0xff, 0xf4, 0xdc)
	colYellow = rgb(0xff, 0xd2, 0x3a)
	colOrange = rgb(0xff, 0x93, 0x1e)
	colGreen  = rgb(0x62, 0xd8, 0x3a)
	colBlue   = rgb(0x3e, 0x9c, 0xff)
	colRed    = rgb(0xff, 0x4a, 0x4a)
	colPurple = rgb(0xa8, 0x5c, 0xff)
	colGrey   = rgb(0x9a, 0xa4, 0xbd)
	colPanel  = rgb(0x2c, 0x3c, 0x78)
	colGem    = rgb(0xff, 0x4f, 0xa8)
	colGold   = rgb(0xff, 0xc8, 0x2e)
	colSteel  = rgb(0xb8, 0xc6, 0xdc)
	colSkin   = rgb(0xff, 0xd6, 0xb0)
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

// shadow is a soft dark ellipse on the ground.
func shadow(dst *ebiten.Image, cx, cy, rx, ry, a float64) {
	radial(dst, cx, cy, rx, ry, 0, 0, color.RGBA{0, 0, 0, uint8(255 * a)}, clear)
}

func disc(dst *ebiten.Image, cx, cy, rx, ry float64, c color.RGBA) {
	radial(dst, cx, cy, rx, ry, 0, 0, c, c)
}

// orb draws an outlined, lit ball: the basic body part of every character.
func orb(dst *ebiten.Image, cx, cy, rx, ry float64, base color.RGBA) {
	disc(dst, cx, cy, rx+2.5, ry+2.5, colInk)
	radial(dst, cx, cy, rx, ry, -0.35, -0.42, light(base, 0.35), shade(base, 0.55))
	radial(dst, cx+rx*0.15, cy+ry*0.55, rx*0.55, ry*0.3, 0, 0, alpha(light(base, 0.25), 0.35), clear)
	radial(dst, cx-rx*0.38, cy-ry*0.45, rx*0.34, ry*0.26, 0, 0, alpha(colWhite, 0.7), clear)
}

// orbFlat is an orb without the outline, for details on top of other parts.
func orbFlat(dst *ebiten.Image, cx, cy, rx, ry float64, base color.RGBA) {
	radial(dst, cx, cy, rx, ry, -0.35, -0.42, light(base, 0.3), shade(base, 0.6))
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

func fillPath(dst *ebiten.Image, p *vector.Path, c color.RGBA) { gradPath(dst, p, c, c) }

// gradPath fills a path with a top-to-bottom gradient.
func gradPath(dst *ebiten.Image, p *vector.Path, top, bot color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 {
		return
	}
	y0, y1 := float32(math.Inf(1)), float32(math.Inf(-1))
	for i, v := range vs {
		y0, y1 = min(y0, v.DstY), max(y1, v.DstY)
		vs[i].DstX, vs[i].DstY = v.DstX*float32(S), v.DstY*float32(S)
	}
	y0, y1 = y0*float32(S), y1*float32(S)
	for i := range vs {
		t := 0.0
		if y1 > y0 {
			t = float64((vs[i].DstY - y0) / (y1 - y0))
		}
		c := mix(top, bot, t)
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, whiteSub, triAA)
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

// shape fills a polygon with a gradient and outlines it in ink.
func shape(dst *ebiten.Image, pts []float64, top, bot color.RGBA, outline float64) {
	p := pathOf(pts)
	if outline > 0 {
		strokePath(dst, p, outline*2, colInk)
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

// ---- text ----

var faceSrc = func() *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontTTF))
	if err != nil {
		panic(err)
	}
	return s
}()

var faces = map[float64]*text.GoTextFace{}

func face(size float64) *text.GoTextFace {
	size = math.Round(size*2) / 2
	f, ok := faces[size]
	if !ok {
		f = &text.GoTextFace{Source: faceSrc, Size: size}
		faces[size] = f
	}
	return f
}

func drawText(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*S, y*S)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * S * 1.12
	text.Draw(dst, s, face(size*S), op)
}

func textWidth(s string, size float64) float64 {
	w, _ := text.Measure(s, face(size), size*1.12)
	return w
}

// say draws outlined text with a drop shadow: ax is 0 left, 0.5 centre, 1 right; y is the middle.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	o := math.Max(1.4, size/13)
	drawText(dst, s, x, y+o*1.6, size, colInk, ax)
	for _, d := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, -1.2}, {0, 1.3}, {-1.2, 0}, {1.2, 0}} {
		drawText(dst, s, x+d[0]*o, y+d[1]*o, size, colInk, ax)
	}
	drawText(dst, s, x, y, size, c, ax)
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

// ---- UI pieces ----

// panel is a rounded box with an ink outline, a gradient body and a lit top edge.
func panel(dst *ebiten.Image, r rect, top, bot color.RGBA, rad float64) {
	roundRect(dst, rect{r.x - 3, r.y - 1, r.w + 6, r.h + 8}, rad+3, alpha(colInk, 0.55))
	roundRect(dst, r.grow(3), rad+3, colInk)
	roundGrad(dst, r, rad, top, bot)
	roundGrad(dst, rect{r.x + 4, r.y + 3, r.w - 8, math.Min(18, r.h*0.3)}, rad-2, alpha(colWhite, 0.22), clear)
}

// button draws a glossy game button with a 3D lip.
func button(dst *ebiten.Image, r rect, c color.RGBA, label string, size float64, off bool) {
	if off {
		c = rgb(0x8c, 0x94, 0xaa)
	}
	roundRect(dst, rect{r.x - 3, r.y - 3, r.w + 6, r.h + 7}, 17, colInk)
	roundRect(dst, r, 14, shade(c, 0.62))
	roundGrad(dst, rect{r.x, r.y, r.w, r.h - 6}, 14, light(c, 0.22), shade(c, 0.92))
	roundGrad(dst, rect{r.x + 6, r.y + 3, r.w - 12, (r.h - 6) * 0.42}, 10, alpha(colWhite, 0.45), alpha(colWhite, 0.08))
	if label != "" {
		say(dst, label, r.cx(), r.y+(r.h-6)/2, size, colWhite, 0.5)
	}
}

func pill(dst *ebiten.Image, r rect, val string) {
	roundRect(dst, r.grow(3), r.h/2+3, colInk)
	roundGrad(dst, r, r.h/2, rgb(0x2a, 0x30, 0x58), rgb(0x16, 0x1a, 0x34))
	say(dst, val, r.x+r.w-14, r.cy(), 19, colWhite, 1)
}

func coinIcon(dst *ebiten.Image, x, y, r float64) {
	disc(dst, x, y+1.5, r+2.5, r+2.5, colInk)
	radial(dst, x, y, r, r, -0.3, -0.35, rgb(0xff, 0xf0, 0x90), rgb(0xd8, 0x8a, 0x10))
	radial(dst, x, y, r*0.66, r*0.66, 0.3, 0.35, rgb(0xff, 0xe0, 0x60), rgb(0xe8, 0xa0, 0x18))
	drawText(dst, "$", x, y, r*1.1, alpha(rgb(0xb0, 0x6a, 0x00), 0.8), 0.5)
	radial(dst, x-r*0.35, y-r*0.4, r*0.3, r*0.22, 0, 0, alpha(colWhite, 0.8), clear)
}

func gemIcon(dst *ebiten.Image, x, y, r float64) {
	out := []float64{x - r, y - r*0.35, x - r*0.5, y - r, x + r*0.5, y - r, x + r, y - r*0.35, x, y + r}
	shape(dst, out, rgb(0xff, 0x8a, 0xc8), rgb(0xc8, 0x1a, 0x6a), 2.2)
	shape(dst, []float64{x - r*0.5, y - r, x + r*0.5, y - r, x + r*0.3, y - r*0.35, x - r*0.3, y - r*0.35}, rgb(0xff, 0xd0, 0xe8), rgb(0xff, 0x9a, 0xd0), 0)
	shape(dst, []float64{x - r*0.3, y - r*0.35, x + r*0.3, y - r*0.35, x, y + r}, rgb(0xff, 0x6a, 0xb4), rgb(0xd8, 0x2a, 0x7a), 0)
	glow(dst, x-r*0.3, y-r*0.55, r*0.5, colWhite, 0.5)
}

func boltIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	pts := []float64{x + 0.25*r, y - r, x - 0.6*r, y + 0.15*r, x - 0.05*r, y + 0.15*r, x - 0.3*r, y + r, x + 0.62*r, y - 0.22*r, x + 0.08*r, y - 0.22*r}
	shape(dst, pts, light(c, 0.5), shade(c, 0.75), 2.2)
}

func heartIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	p := &vector.Path{}
	p.MoveTo(float32(x), float32(y+r*0.9))
	p.CubicTo(float32(x-r*1.4), float32(y-r*0.1), float32(x-r*0.7), float32(y-r*1.2), float32(x), float32(y-r*0.45))
	p.CubicTo(float32(x+r*0.7), float32(y-r*1.2), float32(x+r*1.4), float32(y-r*0.1), float32(x), float32(y+r*0.9))
	p.Close()
	strokePath(dst, p, 4, colInk)
	gradPath(dst, p, light(c, 0.35), shade(c, 0.7))
	radial(dst, x-r*0.4, y-r*0.35, r*0.25, r*0.18, 0, 0, alpha(colWhite, 0.8), clear)
}

func shieldIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	pts := []float64{x - r*0.8, y - r*0.75, x, y - r, x + r*0.8, y - r*0.75, x + r*0.72, y + r*0.15, x, y + r, x - r*0.72, y + r*0.15}
	shape(dst, pts, light(c, 0.35), shade(c, 0.7), 2.4)
	shape(dst, []float64{x - r*0.5, y - r*0.55, x, y - r*0.72, x + r*0.5, y - r*0.55, x + r*0.44, y + r*0.05, x, y + r*0.62, x - r*0.44, y + r*0.05}, alpha(colWhite, 0.25), alpha(colWhite, 0.02), 0)
}

func swordIcon(dst *ebiten.Image, x, y, r float64) {
	blade := rot([]float64{x - r*0.12, y + r*0.35, x - r*0.12, y - r*0.8, x, y - r, x + r*0.12, y - r*0.8, x + r*0.12, y + r*0.35}, x, y, math.Pi/4)
	shape(dst, blade, colWhite, colSteel, 2)
	guard := rot([]float64{x - r*0.42, y + r*0.32, x + r*0.42, y + r*0.32, x + r*0.42, y + r*0.48, x - r*0.42, y + r*0.48}, x, y, math.Pi/4)
	shape(dst, guard, colYellow, colOrange, 2)
	grip := rot([]float64{x - r*0.08, y + r*0.48, x + r*0.08, y + r*0.48, x + r*0.08, y + r*0.85, x - r*0.08, y + r*0.85}, x, y, math.Pi/4)
	shape(dst, grip, rgb(0x9a, 0x5a, 0x30), rgb(0x5a, 0x30, 0x18), 2)
}

func chestIcon(dst *ebiten.Image, x, y, s float64, royal bool) {
	body, trim := rgb(0xc0, 0x74, 0x34), rgb(0xff, 0xd0, 0x48)
	if royal {
		body, trim = rgb(0x8a, 0x4c, 0xe8), rgb(0xff, 0xd8, 0x50)
	}
	shadow(dst, x, y+40*s, 46*s, 10*s, 0.35)
	roundRect(dst, rect{x - 43*s, y - 7*s, 86 * s, 47 * s}, 7*s, colInk)
	roundGrad(dst, rect{x - 40*s, y - 4*s, 80 * s, 41 * s}, 6*s, body, shade(body, 0.6))
	p := &vector.Path{}
	p.MoveTo(float32(x-43*s), float32(y-2*s))
	p.LineTo(float32(x-43*s), float32(y-18*s))
	p.Arc(float32(x), float32(y-18*s), float32(43*s), math.Pi, 2*math.Pi, vector.Clockwise)
	p.LineTo(float32(x+43*s), float32(y-2*s))
	p.Close()
	strokePath(dst, p, 5*s, colInk)
	gradPath(dst, p, light(body, 0.3), shade(body, 0.8))
	for _, dx := range []float64{-30, 30} {
		roundGrad(dst, rect{x + dx*s - 5*s, y - 58*s, 10 * s, 96 * s}, 3*s, light(trim, 0.3), shade(trim, 0.7))
	}
	roundGrad(dst, rect{x - 46*s, y - 7*s, 92 * s, 9 * s}, 3*s, light(trim, 0.3), shade(trim, 0.7))
	roundRect(dst, rect{x - 10*s, y - 12*s, 20 * s, 22 * s}, 4*s, colInk)
	roundGrad(dst, rect{x - 8*s, y - 10*s, 16 * s, 18 * s}, 3*s, light(trim, 0.4), shade(trim, 0.7))
	circle(dst, x, y-2*s, 2.6*s, colInk)
	if royal {
		glow(dst, x, y-20*s, 60*s, colYellow, 0.25)
	}
}

// hpBar draws a health bar with the numbers on it.
func hpBar(dst *ebiten.Image, x, y, w float64, hp, maxHP int, c color.RGBA) {
	r := rect{x - w/2, y, w, 22}
	roundRect(dst, r.grow(3), 10, colInk)
	roundGrad(dst, r, 8, shade(c, 0.25), shade(c, 0.4))
	f := math.Max(0, math.Min(1, float64(hp)/float64(max(1, maxHP))))
	if f > 0 {
		fr := rect{r.x, r.y, math.Max(16, r.w*f), r.h}
		roundGrad(dst, fr, 8, light(c, 0.3), shade(c, 0.8))
		roundGrad(dst, rect{fr.x + 3, fr.y + 2, fr.w - 6, 7}, 4, alpha(colWhite, 0.5), alpha(colWhite, 0.1))
	}
	say(dst, itoa(max(0, hp))+" / "+itoa(maxHP), x, y+11, 15, colWhite, 0.5)
}

// ribbon is a banner with folded ends.
func ribbon(dst *ebiten.Image, cx, cy, w float64, s string, c color.RGBA) {
	for _, d := range []float64{-1, 1} {
		shape(dst, []float64{cx + d*(w/2-10), cy - 14, cx + d*(w/2+38), cy - 14, cx + d*(w/2+22), cy + 6, cx + d*(w/2+38), cy + 26, cx + d*(w/2-10), cy + 26}, shade(c, 0.75), shade(c, 0.5), 2.5)
	}
	panel(dst, rect{cx - w/2, cy - 30, w, 58}, light(c, 0.15), shade(c, 0.8), 12)
	say(dst, s, cx, cy-2, 34, colWhite, 0.5)
}
