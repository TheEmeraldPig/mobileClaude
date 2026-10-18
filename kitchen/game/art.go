package game

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Drawing kit: colours, the layout scale, cached painted images, outlined text.
// Fonts: Lilita One and Fredoka (SIL Open Font License, see FONT_LICENSE.txt).

//go:embed LilitaOne-Regular.ttf
var titleTTF []byte

//go:embed Fredoka.ttf
var bodyTTF []byte

// W is the layout width; the height follows the screen.
const W = 540.0

// S is the scale from layout units to real screen pixels. Art is painted at the
// screen's own resolution so it stays sharp on any phone.
var S = 1.0

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

var (
	colWhite  = rgb(0xff, 0xff, 0xff)
	colInk    = rgb(0x2a, 0x12, 0x45) // text outline, deep purple
	colCream  = rgb(0xff, 0xf4, 0xd8)
	colGold   = rgb(0xff, 0xc8, 0x2e)
	colGreen  = rgb(0x6c, 0xd8, 0x2c)
	colRed    = rgb(0xf0, 0x40, 0x48)
	colBlue   = rgb(0x3c, 0xa8, 0xf8)
	colPurple = rgb(0x8e, 0x5c, 0xf0)
	colPink   = rgb(0xff, 0x6e, 0xb4)
	colOrange = rgb(0xff, 0x96, 0x28)
	colGrey   = rgb(0x9a, 0x94, 0xb0)
	colPanel  = rgb(0x5a, 0x3a, 0x9c)
	colPanelD = rgb(0x3a, 0x22, 0x6e)
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A)}
}

func shade(c color.RGBA, f float64) color.RGBA {
	m := func(v uint8) uint8 { return uint8(math.Max(0, math.Min(255, float64(v)*f))) }
	return color.RGBA{m(c.R), m(c.G), m(c.B), c.A}
}

func light(c color.RGBA, t float64) color.RGBA { return mix(c, rgb(255, 255, 255), t) }

// alpha sets a straight colour's opacity.
func alpha(c color.RGBA, a float64) color.RGBA {
	c.A = uint8(math.Max(0, math.Min(1, a)) * 255)
	return c
}

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }
func (r rect) cx() float64           { return r.x + r.w/2 }
func (r rect) cy() float64           { return r.y + r.h/2 }

var whiteImg = func() *ebiten.Image {
	i := ebiten.NewImage(3, 3)
	i.Fill(color.White)
	return i
}()

var whiteSub = whiteImg.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(x*S), float32(y*S), float32(w*S), float32(h*S), premul(c), false)
}

func premul(c color.RGBA) color.RGBA {
	a := float64(c.A) / 255
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), c.A}
}

// gradRect fills a rectangle with a vertical gradient.
func gradRect(dst *ebiten.Image, x, y, w, h float64, top, bot color.RGBA) {
	v := func(x, y float64, c color.RGBA) ebiten.Vertex {
		c = premul(c)
		return ebiten.Vertex{DstX: float32(x * S), DstY: float32(y * S), SrcX: 1.5, SrcY: 1.5,
			ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
	}
	vs := []ebiten.Vertex{v(x, y, top), v(x+w, y, top), v(x, y+h, bot), v(x+w, y+h, bot)}
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, whiteSub, nil)
}

// clip returns the part of dst inside r.
func clip(dst *ebiten.Image, r rect) *ebiten.Image {
	return dst.SubImage(image.Rect(int(r.x*S), int(r.y*S), int(math.Ceil((r.x+r.w)*S)), int(math.Ceil((r.y+r.h)*S)))).(*ebiten.Image)
}

// ---- painted image cache ----

// art caches painted images by name. It is cleared when the screen scale changes.
var (
	artCache = map[string]*pimg{}
	artS     = 0.0
)

// pimg is a painted image and the unit position of its top-left corner relative
// to the anchor it was painted around.
type pimg struct {
	img    *ebiten.Image
	ox, oy float64 // anchor offset in units
	k      float64 // pixels per unit it was painted at
}

// paintArt returns the cached image for key, painting it with fn over the unit box
// x0..x1, y0..y1 (anchor at 0,0) at scale z (units drawn per layout unit).
func paintArt(key string, x0, y0, x1, y1, z float64, fn func(c *canvas)) *pimg {
	if artS != S {
		for _, p := range artCache {
			p.img.Deallocate()
		}
		artCache, artS = map[string]*pimg{}, S
	}
	if p, ok := artCache[key]; ok {
		return p
	}
	k := S * z
	c := newCanvas(x0, y0, x1, y1, k)
	fn(c)
	p := &pimg{img: c.image(), ox: x0, oy: y0, k: k}
	artCache[key] = p
	return p
}

// draw places the image with its anchor at layout x,y, scaled by sc.
func (p *pimg) draw(dst *ebiten.Image, x, y, sc float64, cs ebiten.ColorScale) {
	p.drawT(dst, x, y, sc, sc, 0, cs)
}

// drawT places the image with its anchor at x,y, scaled sx,sy and turned by a (radians).
func (p *pimg) drawT(dst *ebiten.Image, x, y, sx, sy, a float64, cs ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	u := S / p.k // pixels of this image per screen pixel at scale 1
	op.GeoM.Translate(p.ox*p.k, p.oy*p.k)
	op.GeoM.Scale(sx*u, sy*u)
	if a != 0 {
		op.GeoM.Rotate(a)
	}
	op.GeoM.Translate(x*S, y*S)
	op.ColorScale = cs
	dst.DrawImage(p.img, op)
}

// drawSkew is drawT with a horizontal shear (for wobbling slime).
func (p *pimg) drawSkew(dst *ebiten.Image, x, y, sx, sy, skew float64, cs ebiten.ColorScale, blend ebiten.Blend) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: blend}
	u := S / p.k
	op.GeoM.Translate(p.ox*p.k, p.oy*p.k)
	op.GeoM.Scale(sx*u, sy*u)
	op.GeoM.Skew(skew, 0)
	op.GeoM.Translate(x*S, y*S)
	op.ColorScale = cs
	dst.DrawImage(p.img, op)
}

var cs1 ebiten.ColorScale

func csA(a float64) ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleAlpha(float32(a))
	return c
}

// slice3 draws a painted bar image stretched to width w, keeping its round ends:
// the image is h tall and 2h wide; the middle column is stretched.
func slice3(dst *ebiten.Image, p *pimg, x, y, w float64, cs ebiten.ColorScale) {
	img := p.img
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	half := iw / 2
	sc := S / p.k
	capW := float64(half) / p.k
	if w < capW*2 {
		capW = w / 2
	}
	draw := func(sx0, sx1 int, dx, dw float64) {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(dw*S/float64(sx1-sx0), sc)
		op.GeoM.Translate(dx*S, y*S)
		op.ColorScale = cs
		dst.DrawImage(img.SubImage(image.Rect(sx0, 0, sx1, ih)).(*ebiten.Image), op)
	}
	cw := int(capW * p.k)
	draw(0, cw, x, capW)
	draw(half, half+1, x+capW, w-capW*2)
	draw(iw-cw, iw, x+w-capW, capW)
}

// slice9 draws a painted panel image (corner size cr units) stretched to w x h.
func slice9(dst *ebiten.Image, p *pimg, r rect, cr float64, cs ebiten.ColorScale) {
	img := p.img
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	c := int(cr * p.k)
	xs := []int{0, c, iw - c, iw}
	ys := []int{0, c, ih - c, ih}
	dx := []float64{r.x, r.x + cr, r.x + r.w - cr, r.x + r.w}
	dy := []float64{r.y, r.y + cr, r.y + r.h - cr, r.y + r.h}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			if xs[i+1] <= xs[i] || ys[j+1] <= ys[j] || dx[i+1] <= dx[i] || dy[j+1] <= dy[j] {
				continue
			}
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale((dx[i+1]-dx[i])*S/float64(xs[i+1]-xs[i]), (dy[j+1]-dy[j])*S/float64(ys[j+1]-ys[j]))
			op.GeoM.Translate(dx[i]*S, dy[j]*S)
			op.ColorScale = cs
			dst.DrawImage(img.SubImage(image.Rect(xs[i], ys[j], xs[i+1], ys[j+1])).(*ebiten.Image), op)
		}
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
	bodySrc  = mustFace(bodyTTF)
	faces    = map[[2]float64]*text.GoTextFace{}
)

const (
	fBody = iota
	fTitle
)

func face(src int, size float64) *text.GoTextFace {
	size = math.Round(size*2) / 2
	k := [2]float64{float64(src), size}
	f, ok := faces[k]
	if !ok {
		f = &text.GoTextFace{Source: []*text.GoTextFaceSource{bodySrc, titleSrc}[src], Size: size}
		if src == fBody {
			f.SetVariation(text.MustParseTag("wght"), 600)
		}
		faces[k] = f
	}
	return f
}

func drawFont(dst *ebiten.Image, src int, s string, x, y, size float64, c color.RGBA, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*S, y*S)
	op.ColorScale.ScaleWithColor(premul(c))
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * S * 1.12
	text.Draw(dst, s, face(src, size*S), op)
}

var ring8 = func() [][2]float64 {
	var r [][2]float64
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		r = append(r, [2]float64{math.Cos(a), math.Sin(a)})
	}
	return r
}()

// say draws chunky title text with a dark outline and a drop shadow: the cartoon
// game label. ax is 0 left, 0.5 centre, 1 right; y is the middle.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.RGBA, ax float64) {
	sayOut(dst, fTitle, s, x, y, size, c, colInk, ax)
}

// sayB is body text with an outline, for small labels.
func sayB(dst *ebiten.Image, s string, x, y, size float64, c color.RGBA, ax float64) {
	sayOut(dst, fBody, s, x, y, size, c, colInk, ax)
}

func sayOut(dst *ebiten.Image, src int, s string, x, y, size float64, c, out color.RGBA, ax float64) {
	o := math.Max(1.2, size/11)
	a := float64(c.A) / 255
	drawFont(dst, src, s, x, y+o*1.1, size, alpha(out, a), ax)
	for _, d := range ring8 {
		drawFont(dst, src, s, x+d[0]*o, y+d[1]*o, size, alpha(out, a), ax)
	}
	drawFont(dst, src, s, x, y, size, c, ax)
}

// ink is plain body text without an outline, for text on light cards.
func ink(dst *ebiten.Image, s string, x, y, size float64, c color.RGBA, ax float64) {
	drawFont(dst, fBody, s, x, y, size, c, ax)
}

func textW(src int, s string, size float64) float64 {
	w, _ := text.Measure(s, face(src, size), size*1.12)
	return w
}

// num formats big numbers the idle-game way: 950, 9.5K, 12.3M, 4.1B ...
func num(v float64) string {
	if v < 0 {
		return "-" + num(-v)
	}
	if v < 1000 {
		if v < 10 && v != math.Floor(v) {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%d", int64(v))
	}
	units := []string{"K", "M", "B", "T", "aa", "ab", "ac", "ad", "ae", "af", "ag", "ah"}
	i := -1
	for v >= 1000 && i < len(units)-1 {
		v /= 1000
		i++
	}
	switch {
	case v >= 100:
		return fmt.Sprintf("%d%s", int64(v), units[i])
	case v >= 10:
		return fmt.Sprintf("%.1f%s", math.Floor(v*10)/10, units[i])
	}
	return fmt.Sprintf("%.2f%s", math.Floor(v*100)/100, units[i])
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// ---- easing ----

func easeOut(t float64) float64 { t = clamp01(t); return 1 - (1-t)*(1-t)*(1-t) }
func easeIn(t float64) float64  { t = clamp01(t); return t * t * t }
func backOut(t float64) float64 {
	t = clamp01(t)
	const c = 1.9
	return 1 + (c+1)*math.Pow(t-1, 3) + c*math.Pow(t-1, 2)
}
func clamp01(t float64) float64 { return math.Max(0, math.Min(1, t)) }

// elastic overshoots and settles, for things popping in.
func elastic(t float64) float64 {
	t = clamp01(t)
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*2*math.Pi/3) + 1
}
