package game

import (
	"bytes"
	_ "embed"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/PixelifySans.ttf
var pixelifyTTF []byte

//go:embed assets/Silkscreen-Regular.ttf
var silkTTF []byte

var (
	pixelifySrc, silkSrc *text.GoTextFaceSource
	faces                = map[[2]int]*text.GoTextFace{}
)

// Palette, the same as Mote's UI: dark outline, violet body, light top-left edge,
// shaded bottom-right edge, gold accent.
var (
	colO = outline
	colB = rgb(0x6b4fc8)
	colL = rgb(0xa58bff)
	colS = rgb(0x3e2a85)
	colT = rgb(0xffc94a)
	colW = color.RGBA{255, 255, 255, 255}
	colG = rgb(0x6dff7a) // money green
)

// A button colour set: body, light edge, shade edge.
type skin struct{ b, l, s color.RGBA }

var (
	skViolet = skin{colB, colL, colS}
	skOrange = skin{rgb(0xf0862a), rgb(0xffc56b), rgb(0xa8501a)}
	skDark   = skin{rgb(0x2b1f55), rgb(0x45358a), rgb(0x17102f)}
	skGreen  = skin{rgb(0x3fbf4a), rgb(0x8ef06a), rgb(0x1f7a2a)}
	skGrey   = skin{rgb(0x4a4560), rgb(0x6a6584), rgb(0x2a2638)}
	skGold   = skin{rgb(0xf5b52c), rgb(0xffe08a), rgb(0xb0701a)}
)

func initFonts() {
	var err error
	if pixelifySrc, err = text.NewGoTextFaceSource(bytes.NewReader(pixelifyTTF)); err != nil {
		panic(err)
	}
	if silkSrc, err = text.NewGoTextFaceSource(bytes.NewReader(silkTTF)); err != nil {
		panic(err)
	}
}

const (
	fBig   = 0 // Pixelify Sans
	fSmall = 1 // Silkscreen
)

func face(kind int, size float64) *text.GoTextFace {
	k := [2]int{kind, int(math.Round(size))}
	if f, ok := faces[k]; ok {
		return f
	}
	src := pixelifySrc
	if kind == fSmall {
		src = silkSrc
	}
	f := &text.GoTextFace{Source: src, Size: float64(k[1])}
	faces[k] = f
	return f
}

type rect struct{ x, y, w, h float64 }

func (r rect) in(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }
func (r rect) cx() float64          { return r.x + r.w/2 }
func (r rect) cy() float64          { return r.y + r.h/2 }

func fill(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

// bevel draws a pixel box with clipped corners: outline, body, light top-left edge and
// shaded bottom-right edge. p is the size of one "pixel".
func bevel(dst *ebiten.Image, r rect, sk skin, p float64) {
	x, y, w, h := math.Round(r.x), math.Round(r.y), math.Round(r.w), math.Round(r.h)
	fill(dst, x+p, y, w-2*p, h, colO)
	fill(dst, x, y+p, w, h-2*p, colO)
	fill(dst, x+p, y+p, w-2*p, h-2*p, sk.b)
	e := p
	fill(dst, x+p, y+p, w-2*p, e, sk.l)
	fill(dst, x+p, y+p, e, h-2*p, sk.l)
	fill(dst, x+p, y+h-p-e, w-2*p, e, sk.s)
	fill(dst, x+w-p-e, y+p+e, e, h-2*p-e, sk.s)
}

const (
	alLeft   = 0
	alCenter = 1
	alRight  = 2
)

// say draws text with a dark outline, vertically centred on y.
func say(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, al int, c color.Color, o float64) {
	op := &text.DrawOptions{}
	op.LayoutOptions.PrimaryAlign = text.Align(al)
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	_, _, _, ca := c.RGBA()
	if o > 0 && ca > 0 {
		for _, d := range [][2]float64{{-o, 0}, {o, 0}, {0, -o}, {0, o}, {o, o}, {-o, o}, {o, -o}, {-o, -o}, {o * 0.5, o * 2}} {
			op.GeoM.Reset()
			op.GeoM.Translate(x+d[0], y+d[1])
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(colO)
			op.ColorScale.ScaleAlpha(float32(math.Pow(float64(ca)/0xffff, 4))) // the 9 overlapping copies fade with the text
			text.Draw(dst, s, f, op)
		}
	}
	op.GeoM.Reset()
	op.GeoM.Translate(x, y)
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

func textW(s string, f *text.GoTextFace) float64 {
	w, _ := text.Measure(s, f, 0)
	return w
}

// icon draws a pixel icon scaled to height h, centred on (cx, cy).
func icon(dst *ebiten.Image, name string, cx, cy, h float64) {
	img := icons[name]
	if img == nil {
		return
	}
	b := img.Bounds()
	s := h / float64(b.Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(math.Round(cx-float64(b.Dx())*s/2), math.Round(cy-h/2))
	dst.DrawImage(img, op)
}
