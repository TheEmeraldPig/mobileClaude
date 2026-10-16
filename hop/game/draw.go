package game

import (
	"bytes"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomonobold"
)

// Neon space: deep navy, glowing cyan and magenta, gold for coins.
var (
	colBG     = color.RGBA{0x07, 0x0b, 0x1a, 0xff}
	colPanel  = color.RGBA{0x10, 0x17, 0x33, 0xff}
	colPanel2 = color.RGBA{0x18, 0x22, 0x47, 0xff}
	colText   = color.RGBA{0xe6, 0xf1, 0xff, 0xff}
	colDim    = color.RGBA{0x7d, 0x8b, 0xb3, 0xff}
	colCyan   = color.RGBA{0x3e, 0xf0, 0xff, 0xff}
	colPink   = color.RGBA{0xff, 0x4f, 0xd8, 0xff}
	colGold   = color.RGBA{0xff, 0xd8, 0x4a, 0xff}
	colRed    = color.RGBA{0xff, 0x3d, 0x5a, 0xff}
	colGreen  = color.RGBA{0x4d, 0xff, 0xa0, 0xff}
	colRock   = color.RGBA{0x8a, 0x7f, 0xa8, 0xff}
)

var planetCols = [][2]color.RGBA{
	{{0x2e, 0xc4, 0xb6, 0xff}, {0x1a, 0x6e, 0x8a, 0xff}},
	{{0xff, 0x8a, 0x3d, 0xff}, {0xa8, 0x3a, 0x4a, 0xff}},
	{{0xa3, 0x5b, 0xff, 0xff}, {0x4a, 0x2a, 0x9a, 0xff}},
	{{0xff, 0x5f, 0x9e, 0xff}, {0x8a, 0x2a, 0x6a, 0xff}},
	{{0x9b, 0xe0, 0x4a, 0xff}, {0x3a, 0x7a, 0x4a, 0xff}},
}

var faceSrc = func() *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF))
	if err != nil {
		panic(err)
	}
	return s
}()

// say draws text; ax is 0 left, 0.5 centre, 1 right; y is the text's middle.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * 1.3
	text.Draw(dst, s, &text.GoTextFace{Source: faceSrc, Size: size}, op)
}

func fade(c color.RGBA, a float64) color.RGBA {
	a = math.Max(0, math.Min(1, a))
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
}

var white = func() *ebiten.Image {
	i := ebiten.NewImage(3, 3)
	i.Fill(color.White)
	return i
}()

func fillPath(dst *ebiten.Image, p *vector.Path, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, white, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

func roundPath(r rect, rad float64) *vector.Path {
	p := &vector.Path{}
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

// panel draws a rounded box with a glowing edge in the given colour.
func panel(dst *ebiten.Image, r rect, edge color.RGBA, fill color.RGBA) {
	fillPath(dst, roundPath(rect{r.x - 3, r.y - 3, r.w + 6, r.h + 6}, 15), fade(edge, 0.18))
	fillPath(dst, roundPath(r, 12), edge)
	fillPath(dst, roundPath(rect{r.x + 2, r.y + 2, r.w - 4, r.h - 4}, 10), fill)
}

func glow(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	for i, a := range []float64{0.06, 0.1, 0.16} {
		vector.DrawFilledCircle(dst, float32(x), float32(y), float32(r*(1.9-0.3*float64(i))), fade(c, a), true)
	}
}

func drawShip(dst *ebiten.Image, s shipDef, x, y, ang float64) {
	glow(dst, x, y, 10, s.Body)
	c, sn := math.Cos(ang), math.Sin(ang)
	tri := func(nose, back, wing float64) {
		p := &vector.Path{}
		p.MoveTo(float32(x+c*nose), float32(y+sn*nose))
		p.LineTo(float32(x-c*back-sn*wing), float32(y-sn*back+c*wing))
		p.LineTo(float32(x-c*back*0.4), float32(y-sn*back*0.4))
		p.LineTo(float32(x-c*back+sn*wing), float32(y-sn*back-c*wing))
		p.Close()
		fillPath(dst, p, s.Body)
	}
	switch s.Shape {
	case 0:
		tri(14, 9, 9)
	case 1:
		tri(18, 10, 5)
	case 2:
		vector.DrawFilledCircle(dst, float32(x), float32(y), 10, fade(s.Body, 0.5), true)
		vector.StrokeCircle(dst, float32(x), float32(y), 10, 2, s.Body, true)
		vector.DrawFilledCircle(dst, float32(x), float32(y), 5, colText, true)
	case 3:
		vector.DrawFilledCircle(dst, float32(x), float32(y), 8, s.Body, true)
		vector.DrawFilledCircle(dst, float32(x+c*2), float32(y+sn*2), 4, colText, true)
	}
}

func drawPlanet(dst *ebiten.Image, x, y, r float64, col int, ring bool) {
	pc := planetCols[col%len(planetCols)]
	glow(dst, x, y, r, pc[0])
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(r), pc[1], true)
	vector.DrawFilledCircle(dst, float32(x-r*0.18), float32(y-r*0.18), float32(r*0.8), pc[0], true)
	vector.DrawFilledCircle(dst, float32(x-r*0.4), float32(y-r*0.4), float32(r*0.22), fade(colText, 0.35), true)
	if ring {
		o := r + 34
		for i := 0; i < 28; i++ { // dashed orbit line
			a := float64(i) / 28 * 2 * math.Pi
			vector.StrokeLine(dst, float32(x+math.Cos(a)*o), float32(y+math.Sin(a)*o),
				float32(x+math.Cos(a+0.12)*o), float32(y+math.Sin(a+0.12)*o), 1.5, fade(pc[0], 0.45), true)
		}
	}
}

// A fixed starfield that scrolls slower than the world.
var bgStars = func() []pt {
	s := make([]pt, 140)
	x := uint32(12345)
	next := func() float64 { x = x*1664525 + 1013904223; return float64(x>>8) / (1 << 24) }
	for i := range s {
		s[i] = pt{next() * 2000, next() * H}
	}
	return s
}()

func drawStars(dst *ebiten.Image, camX, w, t float64) {
	for i, s := range bgStars {
		depth := 0.1 + float64(i%3)*0.12
		x := math.Mod(s.x-camX*depth, 2000)
		if x < 0 {
			x += 2000
		}
		if x > w {
			continue
		}
		tw := 0.4 + 0.3*math.Sin(t*2+float64(i))
		vector.DrawFilledRect(dst, float32(x), float32(s.y), 2, 2, fade(colText, tw*(0.5+depth)), false)
	}
}
