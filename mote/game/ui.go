package game

import (
	"bytes"
	_ "embed"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Pixel UI in the style of the mote: dark outline, violet body, light top-left edge,
// shaded bottom-right edge, gold antenna-tip accent, 3px "pixels".
//
// Immediate mode: menus run once per Update and record what to draw; Draw replays it.
// All sizes are in logical pixels (like CSS px) and scaled by uiScale.

//go:embed assets/PixelifySans.ttf
var pixelifyTTF []byte

//go:embed assets/Silkscreen-Regular.ttf
var silkTTF []byte

var (
	bodyFont    *text.GoTextFaceSource
	displayFont *text.GoTextFaceSource
)

func rgb(h uint32) color.RGBA { return color.RGBA{uint8(h >> 16), uint8(h >> 8), uint8(h), 255} }

var (
	colNight   = rgb(0x101216)
	colO       = rgb(0x221c40)
	colB       = rgb(0x7c62ff)
	colL       = rgb(0xc4b4ff)
	colS       = rgb(0x5a43d6)
	colT       = rgb(0xffde78)
	colInk     = rgb(0xf6f6fc)
	colMuted   = rgb(0xa7a0d6)
	colPanel   = rgb(0x1d1939)
	colPanelHi = rgb(0x2f2962)
	colPanelLo = rgb(0x131027)
	colSlate   = rgb(0x2e2a58)
	colSlateHi = rgb(0x463f80)
	colSlateLo = rgb(0x1d1a3d)
	colDanger  = rgb(0xf07a6a)
	colGoldHi  = rgb(0xfff3c4)
	colGoldLo  = rgb(0xd9a93a)
)

func initFonts() {
	bodyFont, _ = text.NewGoTextFaceSource(bytes.NewReader(pixelifyTTF))
	displayFont, _ = text.NewGoTextFaceSource(bytes.NewReader(silkTTF))
}

// Rect in logical pixels
type Rect struct{ X, Y, W, H float64 }

func (r Rect) Has(x, y float64) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// ---- Draw list ----

type drawOp func(dst *ebiten.Image)

var ops []drawOp

func fill(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.DrawFilledRect(dst, float32(x*uiScale), float32(y*uiScale), float32(w*uiScale), float32(h*uiScale), c, false)
}

// Bevelled box with a notched pixel outline (and an optional drop below it)
func bevel(dst *ebiten.Image, r Rect, body, hi, lo color.Color, drop bool) {
	const p = 3.0
	if drop {
		fill(dst, r.X, r.Y+r.H+p, r.W, p, colO)
	}
	fill(dst, r.X, r.Y-p, r.W, p, colO)
	fill(dst, r.X, r.Y+r.H, r.W, p, colO)
	fill(dst, r.X-p, r.Y, p, r.H, colO)
	fill(dst, r.X+r.W, r.Y, p, r.H, colO)
	fill(dst, r.X, r.Y, r.W, r.H, body)
	fill(dst, r.X, r.Y, r.W, p, hi)
	fill(dst, r.X, r.Y, p, r.H, hi)
	fill(dst, r.X, r.Y+r.H-p, r.W, p, lo)
	fill(dst, r.X+r.W-p, r.Y, p, r.H, lo)
}

type textOpt struct {
	size    float64
	display bool
	col     color.Color
	align   text.Align
	shadow  bool
}

func drawText(dst *ebiten.Image, s string, x, y float64, o textOpt) {
	src := bodyFont
	if o.display {
		src = displayFont
	}
	face := &text.GoTextFace{Source: src, Size: o.size * uiScale}
	op := &text.DrawOptions{}
	op.PrimaryAlign = o.align
	op.SecondaryAlign = text.AlignCenter
	if o.shadow {
		op.GeoM.Translate((x+2)*uiScale, (y+2)*uiScale)
		op.ColorScale.ScaleWithColor(colO)
		text.Draw(dst, s, face, op)
		op = &text.DrawOptions{}
		op.PrimaryAlign = o.align
		op.SecondaryAlign = text.AlignCenter
	}
	op.GeoM.Translate(x*uiScale, y*uiScale)
	c := o.col
	if c == nil {
		c = colInk
	}
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

func textWidth(s string, size float64, display bool) float64 {
	src := bodyFont
	if display {
		src = displayFont
	}
	w, _ := text.Measure(s, &text.GoTextFace{Source: src, Size: size}, 0)
	return w
}

// ---- Widgets ----

type btnStyle int

const (
	styleViolet btnStyle = iota
	styleSlate
	styleGold
)

func btnColors(st btnStyle) (body, hi, lo color.Color, txt color.Color) {
	switch st {
	case styleSlate:
		return colSlate, colSlateHi, colSlateLo, colInk
	case styleGold:
		return colT, colGoldHi, colGoldLo, colO
	}
	return colB, colL, colS, colInk
}

// Button records a button and reports whether it was clicked this frame.
func Button(r Rect, label string, st btnStyle, enabled bool, icon *ebiten.Image) bool {
	hover := enabled && ptr.down && r.Has(ptr.x, ptr.y)
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
			hi, lo = lo, hi
		}
		bevel(dst, rr, body, hi, lo, !hover)
		tx := rr.X + rr.W/2
		align := text.AlignCenter
		if icon != nil {
			isz := min(rr.H-14, 22.0)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
			op.GeoM.Translate((rr.X+10)*uiScale, (rr.Y+(rr.H-isz)/2)*uiScale)
			dst.DrawImage(icon, op)
			tx, align = rr.X+18+isz, text.AlignStart
		}
		drawText(dst, label, tx, rr.Y+rr.H/2, textOpt{size: size, col: txt, align: align, shadow: st != styleGold})
		if !enabled {
			fill(dst, rr.X, rr.Y, rr.W, rr.H, color.NRGBA{16, 12, 32, 150})
		}
	})
	if clicked {
		sfx("click")
	}
	return clicked
}

func Panel(r Rect) {
	ops = append(ops, func(dst *ebiten.Image) { bevel(dst, r, colPanel, colPanelHi, colPanelLo, false) })
}

func Inset(r Rect, body color.Color) {
	ops = append(ops, func(dst *ebiten.Image) { bevel(dst, r, body, color.NRGBA{10, 8, 23, 255}, colPanelHi, false) })
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

func Dim() {
	ops = append(ops, func(dst *ebiten.Image) {
		vector.DrawFilledRect(dst, 0, 0, float32(dst.Bounds().Dx()), float32(dst.Bounds().Dy()), color.NRGBA{14, 10, 32, 158}, false)
	})
}

func IconAt(img *ebiten.Image, x, y, size float64) {
	ops = append(ops, func(dst *ebiten.Image) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(size/8*uiScale, size/8*uiScale)
		op.GeoM.Translate(x*uiScale, y*uiScale)
		dst.DrawImage(img, op)
	})
}

// Pixel icons for buttons (8x8, ink coloured)
func pixIcon(rows ...string) *ebiten.Image {
	img := ebiten.NewImage(8, 8)
	for y, r := range rows {
		for x := 0; x < len(r) && x < 8; x++ {
			switch r[x] {
			case '#':
				img.Set(x, y, colInk)
			case 'o':
				img.Set(x, y, colO)
			}
		}
	}
	return img
}

var (
	icoX     = pixIcon("#.....#.", ".#...#..", "..#.#...", "...#....", "..#.#...", ".#...#..", "#.....#.", "........")
	icoBack  = pixIcon("..#.....", ".#......", "#######.", "#######.", ".#......", "..#.....", "........", "........")
	icoPlay  = pixIcon(".#......", ".##.....", ".###....", ".####...", ".###....", ".##.....", ".#......", "........")
	icoGear  = pixIcon("...#....", ".#####..", ".##o##..", "###o###.", ".##o##..", ".#####..", "...#....", "........")
	icoHome  = pixIcon("...##...", "..####..", ".######.", "########", ".######.", ".##oo##.", ".##oo##.", "........")
	icoWorld = pixIcon("........", "....#...", "...###..", "..#####.", "#.######", "########", "########", "........")
	icoBug   = pixIcon(".#...#..", "..###...", "#######.", ".#ooo#..", "#######.", ".#####..", "#.....#.", "........")
	icoCraft = pixIcon(".###....", "####....", ".####...", "..#.#...", ".....#..", "......#.", ".......#", "........")
	icoPause = pixIcon("........", ".##.##..", ".##.##..", ".##.##..", ".##.##..", ".##.##..", ".##.##..", "........")
	icoBag   = func() *ebiten.Image {
		img := ebiten.NewImage(10, 10)
		set := func(x, y, w, h int, c color.Color) {
			for i := x; i < x+w; i++ {
				for j := y; j < y+h; j++ {
					img.Set(i, j, c)
				}
			}
		}
		set(3, 0, 4, 1, colO)
		set(2, 1, 1, 2, colO)
		set(7, 1, 1, 2, colO)
		set(1, 3, 8, 7, colO)
		set(2, 3, 6, 6, colL)
		set(3, 4, 5, 5, colB)
		set(2, 8, 6, 1, colS)
		set(4, 5, 2, 2, colT)
		return img
	}()
)
