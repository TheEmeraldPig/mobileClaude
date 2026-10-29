package game

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The look: warm card-room colours, soft rounded panels and buttons with a drop shadow,
// and the Jersey fonts (chunky retro lettering; OFL, see assets/OFL-Jersey.txt).
//
// Immediate mode: menus run once per Update and record what to draw; Draw replays it.
// All sizes are in logical pixels (like CSS px) and scaled by uiScale.

//go:embed assets/Jersey15-Regular.ttf
var bodyTTF []byte

//go:embed assets/Jersey10-Regular.ttf
var displayTTF []byte

var (
	bodyFont    *text.GoTextFaceSource
	displayFont *text.GoTextFaceSource
)

// The Jersey letters are small for their size: text sizes in the code are scaled up
const (
	bodyScale    = 1.45
	displayScale = 1.75
)

func rgb(h uint32) color.RGBA { return color.RGBA{uint8(h >> 16), uint8(h >> 8), uint8(h), 255} }

// The warm palette. (The names are kept from the old style: B is the main button colour,
// T the gold accent, O the dark outline.)
var (
	colNight   = rgb(0x1a0f0f)
	colO       = rgb(0x1c1012) // outlines and text shadows
	colB       = rgb(0x2f8ad8) // blue buttons
	colL       = rgb(0xffe2b8) // light cream (titles)
	colS       = rgb(0x1f5f9c)
	colT       = rgb(0xffb83d) // gold
	colInk     = rgb(0xfff3e0)
	colMuted   = rgb(0xc9a594)
	colPanel   = rgb(0x3b2629)
	colPanelHi = rgb(0x5a3a3c)
	colPanelLo = rgb(0x24161a)
	colSlate   = rgb(0x5b4146)
	colSlateHi = rgb(0x7a575c)
	colSlateLo = rgb(0x2e1e22)
	colDanger  = rgb(0xf0583c)
	colGoldHi  = rgb(0xffe08a)
	colGoldLo  = rgb(0xc77a16)
	colRedBtn  = rgb(0xe0452f)
	colRedLo   = rgb(0x9a2a1c)
	colOrange  = rgb(0xff8a1f)
	colOrangeL = rgb(0xc95f0c)
)

func initFonts() {
	bodyFont, _ = text.NewGoTextFaceSource(bytes.NewReader(bodyTTF))
	displayFont, _ = text.NewGoTextFaceSource(bytes.NewReader(displayTTF))
}

// Rect in logical pixels
type Rect struct{ X, Y, W, H float64 }

func (r Rect) Has(x, y float64) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// ---- Draw list ----

type drawOp func(dst *ebiten.Image)

var ops []drawOp

// A white pixel, stretched and tinted to fill rectangles and shapes; Ebitengine batches these
var whiteImg = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img
}()
var whitePx = whiteImg.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

func fill(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w*uiScale, h*uiScale)
	op.GeoM.Translate(x*uiScale, y*uiScale)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(whitePx, op)
}

// fillPath fills a path given in device pixels
func fillPath(dst *ebiten.Image, p *vector.Path, c color.Color) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	r, g, b, a := c.RGBA()
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}
	dst.DrawTriangles(vs, is, whiteImg, &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero})
}

// A rounded rectangle path in device pixels, from logical units
func roundPath(x, y, w, h, rad float64) *vector.Path {
	s := uiScale
	x, y, w, h = x*s, y*s, w*s, h*s
	rad = math.Max(0, math.Min(rad*s, math.Min(w, h)/2))
	var p vector.Path
	p.MoveTo(float32(x+rad), float32(y))
	p.LineTo(float32(x+w-rad), float32(y))
	p.ArcTo(float32(x+w), float32(y), float32(x+w), float32(y+rad), float32(rad))
	p.LineTo(float32(x+w), float32(y+h-rad))
	p.ArcTo(float32(x+w), float32(y+h), float32(x+w-rad), float32(y+h), float32(rad))
	p.LineTo(float32(x+rad), float32(y+h))
	p.ArcTo(float32(x), float32(y+h), float32(x), float32(y+h-rad), float32(rad))
	p.LineTo(float32(x), float32(y+rad))
	p.ArcTo(float32(x), float32(y), float32(x+rad), float32(y), float32(rad))
	p.Close()
	return &p
}

// Rounded boxes and circles are drawn from small cached white shapes (a circle the size
// of the corner, cut into a nine-slice), tinted and stretched: unlike vector paths these
// batch into a few draw calls.
var discCache = map[int]*ebiten.Image{}

// A white anti-aliased disc of radius r device pixels, in an image 2r+2 wide
func disc(r int) *ebiten.Image {
	if img := discCache[r]; img != nil {
		return img
	}
	n := 2*r + 2
	img := ebiten.NewImage(n, n)
	vector.DrawFilledCircle(img, float32(n)/2, float32(n)/2, float32(r), color.White, true)
	discCache[r] = img
	return img
}

func roundRect(dst *ebiten.Image, x, y, w, h, rad float64, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	s := uiScale
	X, Y, W, H := x*s, y*s, w*s, h*s
	R := math.Round(math.Max(0, math.Min(rad*s, math.Min(W, H)/2)))
	if R < 1 {
		fill(dst, x, y, w, h, c)
		return
	}
	img := disc(int(R))
	c0 := float64(int(R) + 1) // the disc's centre in its image
	piece := func(sx0, sy0, sx1, sy1 int, dx, dy, dw, dh float64) {
		if dw <= 0 || dh <= 0 {
			return
		}
		sub := img.SubImage(image.Rect(sx0, sy0, sx1, sy1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(dw/float64(sx1-sx0), dh/float64(sy1-sy0))
		op.GeoM.Translate(dx, dy)
		op.ColorScale.ScaleWithColor(c)
		dst.DrawImage(sub, op)
	}
	ci := int(c0)
	n := 2*int(R) + 2
	// corners
	piece(0, 0, ci, ci, X-(c0-R), Y-(c0-R), c0, c0)
	piece(ci, 0, n, ci, X+W-R, Y-(c0-R), c0, c0)
	piece(0, ci, ci, n, X-(c0-R), Y+H-R, c0, c0)
	piece(ci, ci, n, n, X+W-R, Y+H-R, c0, c0)
	// edges and middle, stretched from the disc's centre row and column
	piece(ci-1, 0, ci, ci, X+R, Y-(c0-R), W-2*R, c0)
	piece(ci-1, ci, ci, n, X+R, Y+H-R, W-2*R, c0)
	piece(0, ci-1, ci, ci, X-(c0-R), Y+R, c0, H-2*R)
	piece(ci, ci-1, n, ci, X+W-R, Y+R, c0, H-2*R)
	piece(ci-1, ci-1, ci, ci, X+R, Y+R, W-2*R, H-2*R)
}

func circle(dst *ebiten.Image, cx, cy, r float64, c color.Color) {
	R := math.Round(r * uiScale)
	if R < 1 {
		return
	}
	img := disc(int(R))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(cx*uiScale-R-1, cy*uiScale-R-1)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(img, op)
}

func withAlpha(c color.Color, a float64) color.Color {
	r, g, b, al := c.RGBA()
	k := math.Max(0, math.Min(1, a))
	return color.RGBA64{uint16(float64(r) * k), uint16(float64(g) * k), uint16(float64(b) * k), uint16(float64(al) * k)}
}

func boxRadius(r Rect) float64 { return math.Min(12, math.Min(r.W, r.H)*0.3) }

// bevel draws a soft rounded box: a drop shadow, a dark rim, the body, a lighter band
// along the top and a darker one along the bottom. (Named after the old pixel bevel.)
func bevel(dst *ebiten.Image, r Rect, body, hi, lo color.Color, drop bool) {
	rad := boxRadius(r)
	if drop {
		roundRect(dst, r.X, r.Y+4, r.W, r.H, rad, color.NRGBA{0, 0, 0, 90})
	}
	roundRect(dst, r.X-2, r.Y-2, r.W+4, r.H+4, rad+2, colO)
	roundRect(dst, r.X, r.Y, r.W, r.H, rad, lo)
	roundRect(dst, r.X, r.Y, r.W, r.H-3, rad, body)
	band := math.Min(r.H*0.42, 14)
	if band > 4 {
		roundRect(dst, r.X+3, r.Y+2, r.W-6, band, math.Max(2, rad-3), withAlpha(hi, 0.35))
	}
}

type textOpt struct {
	size    float64
	display bool
	col     color.Color
	align   text.Align
	shadow  bool
}

func face(size float64, display bool) *text.GoTextFace {
	if display {
		return &text.GoTextFace{Source: displayFont, Size: size * displayScale}
	}
	return &text.GoTextFace{Source: bodyFont, Size: size * bodyScale}
}

func drawText(dst *ebiten.Image, s string, x, y float64, o textOpt) {
	f := face(o.size*uiScale, o.display)
	draw := func(dx, dy float64, c color.Color) {
		op := &text.DrawOptions{}
		op.PrimaryAlign = o.align
		op.SecondaryAlign = text.AlignCenter
		op.GeoM.Translate((x+dx)*uiScale, (y+dy)*uiScale)
		op.ColorScale.ScaleWithColor(c)
		text.Draw(dst, s, f, op)
	}
	if o.shadow {
		draw(0, math.Max(1.5, o.size*0.12), colO)
	}
	c := o.col
	if c == nil {
		c = colInk
	}
	draw(0, 0, c)
}

func textWidth(s string, size float64, display bool) float64 {
	w, _ := text.Measure(s, face(size, display), 0)
	return w
}

// ---- Widgets ----

type btnStyle int

const (
	styleViolet btnStyle = iota // blue: the main choice in a menu
	styleSlate                  // muted
	styleGold                   // orange: the big action
	styleRed                    // red: discard, leave
)

func btnColors(st btnStyle) (body, hi, lo color.Color, txt color.Color) {
	switch st {
	case styleSlate:
		return colSlate, colSlateHi, colSlateLo, colInk
	case styleGold:
		return colOrange, colGoldHi, colOrangeL, colInk
	case styleRed:
		return colRedBtn, rgb(0xff9a80), colRedLo, colInk
	}
	return colB, rgb(0x9fd2ff), colS, colInk
}

// Button records a button and reports whether it was clicked this frame. Icons only
// matter for buttons without a label (close, back, menu).
func Button(r Rect, label string, st btnStyle, enabled bool, icon *ebiten.Image) bool {
	hover := enabled && ptr.down && r.Has(ptr.x, ptr.y)
	mouseOver := enabled && !isTouch && !ptr.down && r.Has(hoverX, hoverY)
	clicked := enabled && ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY)
	if clicked {
		ptr.consumed = true
	}
	size := 15.0
	if isTouch {
		size = 14
	}
	ops = append(ops, func(dst *ebiten.Image) {
		body, hi, lo, txt := btnColors(st)
		rr := r
		if hover {
			rr.Y += 3
		} else if mouseOver {
			rr.Y -= 1
			body = lighten(body, 0.12)
		}
		bevel(dst, rr, body, hi, lo, !hover)
		cx, cy := rr.X+rr.W/2, rr.Y+rr.H/2-1
		if label == "" {
			drawGlyph(dst, icon, cx, cy, math.Min(rr.W, rr.H)*0.42, txt)
		} else {
			drawText(dst, label, cx, cy, textOpt{size: size, col: txt, align: text.AlignCenter, shadow: true})
		}
		if !enabled {
			roundRect(dst, rr.X-2, rr.Y-2, rr.W+4, rr.H+4, boxRadius(rr)+2, color.NRGBA{20, 10, 12, 150})
		}
	})
	if clicked {
		sfx("click")
	}
	return clicked
}

func lighten(c color.Color, k float64) color.Color {
	r, g, b, a := c.RGBA()
	f := func(v uint32) uint8 { return uint8(math.Min(255, float64(v>>8)+(255-float64(v>>8))*k)) }
	return color.RGBA{f(r), f(g), f(b), uint8(a >> 8)}
}

// Smooth glyphs for the buttons that have no label
func drawGlyph(dst *ebiten.Image, icon *ebiten.Image, cx, cy, s float64, c color.Color) {
	line := func(x0, y0, x1, y1, w float64) {
		vector.StrokeLine(dst, float32(x0*uiScale), float32(y0*uiScale), float32(x1*uiScale), float32(y1*uiScale), float32(w*uiScale), c, true)
	}
	h := s / 2
	switch icon {
	case icoX:
		line(cx-h, cy-h, cx+h, cy+h, 3)
		line(cx-h, cy+h, cx+h, cy-h, 3)
	case icoBack:
		line(cx-h, cy, cx+h, cy, 3)
		line(cx-h, cy, cx-h*0.2, cy-h*0.8, 3)
		line(cx-h, cy, cx-h*0.2, cy+h*0.8, 3)
	case icoPause:
		for _, dy := range []float64{-h * 0.7, 0, h * 0.7} {
			line(cx-h, cy+dy, cx+h, cy+dy, 2.6)
		}
	}
}

func Panel(r Rect) {
	ops = append(ops, func(dst *ebiten.Image) { bevel(dst, r, colPanel, colPanelHi, colPanelLo, true) })
}

func Inset(r Rect, body color.Color) {
	ops = append(ops, func(dst *ebiten.Image) {
		rad := math.Min(8, math.Min(r.W, r.H)*0.3)
		roundRect(dst, r.X, r.Y, r.W, r.H, rad, colPanelLo)
		roundRect(dst, r.X, r.Y+2, r.W, r.H-2, rad, body)
	})
}

func Label(s string, x, y float64) {
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, s, x, y, textOpt{size: 10, display: true, col: colT, shadow: true})
	})
}

func Title(s string, x, y float64) {
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, s, x, y, textOpt{size: 17, display: true, col: colL, shadow: true})
	})
}

func Text(s string, x, y, size float64, c color.Color, align text.Align) {
	ops = append(ops, func(dst *ebiten.Image) { drawText(dst, s, x, y, textOpt{size: size, col: c, align: align}) })
}

// Text with a dark drop shadow
func ShadowText(s string, x, y, size float64, c color.Color) {
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, s, x, y, textOpt{size: size, col: c, shadow: true})
	})
}

func Dim() {
	ops = append(ops, func(dst *ebiten.Image) {
		vector.DrawFilledRect(dst, 0, 0, float32(dst.Bounds().Dx()), float32(dst.Bounds().Dy()), color.NRGBA{20, 8, 8, 160}, false)
	})
}

// Markers for the glyphs drawn on unlabelled buttons (drawGlyph); the images themselves
// aren't drawn
var (
	icoX     = ebiten.NewImage(1, 1)
	icoBack  = ebiten.NewImage(1, 1)
	icoPause = ebiten.NewImage(1, 1)
	icoPlay  = ebiten.NewImage(1, 1)
	icoGear  = ebiten.NewImage(1, 1)
	icoHome  = ebiten.NewImage(1, 1)
	icoWorld = ebiten.NewImage(1, 1)
	icoBook  = ebiten.NewImage(1, 1)
)
