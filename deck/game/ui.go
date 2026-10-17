package game

import (
	"bytes"
	_ "embed"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Bright, chunky, cartoon UI: thick dark outlines, glossy buttons with a 3D lip,
// outlined white text. Lilita One font (SIL Open Font License, see FONT_LICENSE.txt).

//go:embed LilitaOne-Regular.ttf
var fontTTF []byte

var (
	colInk    = color.RGBA{0x1c, 0x22, 0x3e, 0xff} // outlines and text shadow
	colWhite  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	colCream  = color.RGBA{0xff, 0xf3, 0xd6, 0xff}
	colSky1   = color.RGBA{0x5c, 0xc4, 0xff, 0xff}
	colSky2   = color.RGBA{0x2c, 0x74, 0xd9, 0xff}
	colPanel  = color.RGBA{0x2b, 0x3f, 0x7a, 0xff}
	colPanel2 = color.RGBA{0x3d, 0x57, 0xa3, 0xff}
	colYellow = color.RGBA{0xff, 0xd2, 0x3a, 0xff}
	colOrange = color.RGBA{0xf5, 0x93, 0x1e, 0xff}
	colGreen  = color.RGBA{0x6b, 0xdc, 0x3c, 0xff}
	colGreen2 = color.RGBA{0x2f, 0xa3, 0x2a, 0xff}
	colBlue   = color.RGBA{0x4c, 0xb8, 0xff, 0xff}
	colBlue2  = color.RGBA{0x1f, 0x6f, 0xd6, 0xff}
	colRed    = color.RGBA{0xff, 0x5a, 0x4a, 0xff}
	colRed2   = color.RGBA{0xc6, 0x2a, 0x2a, 0xff}
	colPink   = color.RGBA{0xff, 0x6b, 0xc8, 0xff}
	colPurple = color.RGBA{0xa8, 0x6b, 0xff, 0xff}
	colGrey   = color.RGBA{0x9a, 0xa4, 0xbd, 0xff}
	colGrey2  = color.RGBA{0x5c, 0x66, 0x82, 0xff}
	colGem    = color.RGBA{0xff, 0x4f, 0xa8, 0xff}
	colGold   = color.RGBA{0xff, 0xc8, 0x2e, 0xff}
)

var faceSrc = func() *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontTTF))
	if err != nil {
		panic(err)
	}
	return s
}()

func drawText(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * 1.15
	text.Draw(dst, s, &text.GoTextFace{Source: faceSrc, Size: size}, op)
}

// say draws outlined text: ax is 0 left, 0.5 centre, 1 right; y is the middle.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	o := math.Max(1.5, size/12)
	for _, d := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, -1.2}, {0, 1.6}, {-1.2, 0}, {1.2, 0}} {
		drawText(dst, s, x+d[0]*o, y+d[1]*o, size, colInk, ax)
	}
	drawText(dst, s, x, y, size, c, ax)
}

func shade(c color.RGBA, f float64) color.RGBA {
	m := func(v uint8) uint8 { return uint8(math.Max(0, math.Min(255, float64(v)*f))) }
	return color.RGBA{m(c.R), m(c.G), m(c.B), c.A}
}

func alpha(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

var white = func() *ebiten.Image {
	i := ebiten.NewImage(3, 3)
	i.Fill(color.White)
	return i
}()

func vert(x, y float64, c color.RGBA) ebiten.Vertex {
	return ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: 1, SrcY: 1,
		ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
}

func gradRect(dst *ebiten.Image, x, y, w, h float64, top, bot color.RGBA) {
	vs := []ebiten.Vertex{vert(x, y, top), vert(x+w, y, top), vert(x, y+h, bot), vert(x+w, y+h, bot)}
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, white, nil)
}

func fillPath(dst *ebiten.Image, p *vector.Path, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, white, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }
func (r rect) in(d float64) rect     { return rect{r.x + d, r.y + d, r.w - 2*d, r.h - 2*d} }
func (r rect) cx() float64           { return r.x + r.w/2 }
func (r rect) cy() float64           { return r.y + r.h/2 }

func roundPath(r rect, rad float64) *vector.Path {
	p := &vector.Path{}
	rad = math.Min(rad, math.Min(r.w, r.h)/2)
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

// box is an outlined rounded panel with a darker lip under it.
func box(dst *ebiten.Image, r rect, c color.RGBA, rad float64) {
	roundRect(dst, rect{r.x - 3, r.y - 3, r.w + 6, r.h + 9}, rad+3, colInk)
	roundRect(dst, rect{r.x, r.y + 2, r.w, r.h}, rad, shade(c, 0.7))
	roundRect(dst, rect{r.x, r.y, r.w, r.h - 3}, rad, c)
}

// button draws a glossy game button: lip, body, shine.
func button(dst *ebiten.Image, r rect, c color.RGBA, label string, size float64, off bool) {
	if off {
		c = colGrey
	}
	roundRect(dst, rect{r.x - 3, r.y - 3, r.w + 6, r.h + 6}, 16, colInk)
	roundRect(dst, r, 13, shade(c, 0.68))
	roundRect(dst, rect{r.x, r.y, r.w, r.h - 6}, 13, c)
	roundRect(dst, rect{r.x + 6, r.y + 4, r.w - 12, (r.h - 6) * 0.38}, 9, alpha(colWhite, 0.32))
	if label != "" {
		say(dst, label, r.cx(), r.y+(r.h-6)/2, size, colWhite, 0.5)
	}
}

func ellipse(dst *ebiten.Image, cx, cy, rx, ry float64, c color.RGBA) {
	p := &vector.Path{}
	for i := 0; i <= 28; i++ {
		a := float64(i) / 28 * 2 * math.Pi
		x, y := float32(cx+math.Cos(a)*rx), float32(cy+math.Sin(a)*ry)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	fillPath(dst, p, c)
}

func circle(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(r), c, true)
}

// inked circle: outline then fill
func ball(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	circle(dst, x, y, r+3, colInk)
	circle(dst, x, y, r, c)
	circle(dst, x-r*0.35, y-r*0.35, r*0.3, alpha(colWhite, 0.4))
}

func poly(dst *ebiten.Image, pts []float64, c color.RGBA) {
	p := &vector.Path{}
	for i := 0; i < len(pts); i += 2 {
		if i == 0 {
			p.MoveTo(float32(pts[0]), float32(pts[1]))
		} else {
			p.LineTo(float32(pts[i]), float32(pts[i+1]))
		}
	}
	p.Close()
	fillPath(dst, p, c)
}

// ---- icons ----

func coinIcon(dst *ebiten.Image, x, y, r float64) {
	circle(dst, x, y, r+2, colInk)
	circle(dst, x, y, r, colGold)
	circle(dst, x, y, r*0.6, shade(colGold, 0.8))
	circle(dst, x-r*0.3, y-r*0.3, r*0.25, alpha(colWhite, 0.6))
}

func gemIcon(dst *ebiten.Image, x, y, r float64) {
	poly(dst, []float64{x, y - r - 3, x + r + 3, y - r*0.2, x, y + r + 3, x - r - 3, y - r*0.2}, colInk)
	poly(dst, []float64{x, y - r, x + r, y - r*0.2, x, y + r, x - r, y - r*0.2}, colGem)
	poly(dst, []float64{x, y - r, x + r*0.5, y - r*0.2, x - r*0.5, y - r*0.2}, alpha(colWhite, 0.5))
}

func boltIcon(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	pts := func(s float64) []float64 {
		return []float64{x + 0.2*s, y - s, x - 0.6*s, y + 0.15*s, x - 0.05*s, y + 0.15*s, x - 0.25*s, y + s, x + 0.6*s, y - 0.2*s, x + 0.05*s, y - 0.2*s}
	}
	poly(dst, pts(r+4), colInk)
	poly(dst, pts(r), c)
}

func icon(dst *ebiten.Image, kind int, x, y, s float64) {
	switch kind {
	case iSword:
		poly(dst, []float64{x - s*0.15, y + s*0.45, x + s*0.6, y - s*0.75, x + s*0.75, y - s*0.6, x - s*0.0, y + s*0.6}, colInk)
		poly(dst, []float64{x - s*0.08, y + s*0.4, x + s*0.6, y - s*0.68, x + s*0.68, y - s*0.6, x - s*0.0, y + s*0.48}, colCream)
		poly(dst, []float64{x - s*0.5, y + s*0.1, x - s*0.4, y, x + s*0.1, y + s*0.5, x, y + s*0.6}, colYellow)
		ball(dst, x-s*0.3, y+s*0.55, s*0.12, colOrange)
	case iShield:
		sh := func(k float64, c color.RGBA) {
			poly(dst, []float64{x - s*0.6*k, y - s*0.6*k, x + s*0.6*k, y - s*0.6*k, x + s*0.55*k, y + s*0.1*k, x, y + s*0.7*k, x - s*0.55*k, y + s*0.1*k}, c)
		}
		sh(1.15, colInk)
		sh(1, colBlue)
		sh(0.6, colCream)
	case iFlame:
		ball(dst, x, y+s*0.2, s*0.45, colOrange)
		poly(dst, []float64{x - s*0.35, y, x, y - s*0.8, x + s*0.35, y}, colOrange)
		circle(dst, x, y+s*0.28, s*0.25, colYellow)
	case iDrop:
		poly(dst, []float64{x - s*0.42, y + s*0.1, x, y - s*0.75, x + s*0.42, y + s*0.1}, colInk)
		ball(dst, x, y+s*0.25, s*0.42, colGreen)
		poly(dst, []float64{x - s*0.3, y + s*0.1, x, y - s*0.6, x + s*0.3, y + s*0.1}, colGreen)
	case iBolt:
		boltIcon(dst, x, y, s*0.7, colYellow)
	case iHeart:
		circle(dst, x-s*0.25, y-s*0.15, s*0.33, colInk)
		circle(dst, x+s*0.25, y-s*0.15, s*0.33, colInk)
		poly(dst, []float64{x - s*0.58, y - s*0.05, x + s*0.58, y - s*0.05, x, y + s*0.68}, colInk)
		circle(dst, x-s*0.25, y-s*0.15, s*0.27, colRed)
		circle(dst, x+s*0.25, y-s*0.15, s*0.27, colRed)
		poly(dst, []float64{x - s*0.5, y - s*0.05, x + s*0.5, y - s*0.05, x, y + s*0.58}, colRed)
	case iCards:
		box(dst, rect{x - s*0.55, y - s*0.5, s * 0.6, s * 0.85}, colBlue, 5)
		box(dst, rect{x - s*0.1, y - s*0.4, s * 0.6, s * 0.85}, colCream, 5)
	case iSkull:
		ball(dst, x, y-s*0.1, s*0.45, colGrey)
		circle(dst, x-s*0.17, y-s*0.1, s*0.12, colInk)
		circle(dst, x+s*0.17, y-s*0.1, s*0.12, colInk)
		vector.DrawFilledRect(dst, float32(x-s*0.2), float32(y+s*0.25), float32(s*0.4), float32(s*0.15), colGrey, true)
	}
}

// ---- cards ----

var kindCol = []color.RGBA{{0xf0, 0x5a, 0x3c, 0xff}, {0x3c, 0x8c, 0xf0, 0xff}}

func wrap(s string, n int) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for len(line) > n {
			cut := strings.LastIndex(line[:n], " ")
			if cut <= 0 {
				break
			}
			out = append(out, line[:cut])
			line = line[cut+1:]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// drawCard draws a card with its top-left at x,y and width w (height is 1.38w).
func drawCard(dst *ebiten.Image, c Card, x, y, w float64, dim bool, t float64) {
	h := w * 1.38
	s := w / 120 // sizes below are for a 120-wide card
	d := c.stats()
	base := c.def()
	body := kindCol[d.Kind]
	if c.Enh != eNone {
		ec := enhDefs[c.Enh].Col
		if c.Enh == eHolo {
			ec = hue(t*0.5 + x*0.002)
		}
		roundRect(dst, rect{x - 7*s, y - 7*s, w + 14*s, h + 14*s}, 18*s, alpha(ec, 0.55+0.25*math.Sin(t*4)))
	}
	roundRect(dst, rect{x - 3*s, y - 3*s, w + 6*s, h + 6*s}, 14*s, colInk)
	roundRect(dst, rect{x, y, w, h}, 12*s, body)
	roundRect(dst, rect{x + 6*s, y + 28*s, w - 12*s, 62 * s}, 8*s, shade(body, 1.35))
	icon(dst, base.Icon, x+w/2, y+60*s, 34*s)
	say(dst, base.Name, x+w/2, y+16*s, 15*s, colWhite, 0.5)
	roundRect(dst, rect{x + 6*s, y + 96*s, w - 12*s, h - 102*s}, 8*s, colCream)
	drawText(dst, wrap(c.text(), 14), x+w/2, y+96*s+(h-102*s)/2, 13*s, colInk, 0.5)
	// cost orb
	cc := colYellow
	if d.Cost < base.Cost {
		cc = colGreen
	}
	ball(dst, x+4*s, y+4*s, 14*s, cc)
	say(dst, itoa(d.Cost), x+4*s, y+3*s, 18*s, colWhite, 0.5)
	if c.Enh != eNone {
		ed := enhDefs[c.Enh]
		tw := 46 * s
		roundRect(dst, rect{x + w - tw - 2*s, y + h - 12*s, tw + 6*s, 20 * s}, 8*s, colInk)
		roundRect(dst, rect{x + w - tw, y + h - 10*s, tw + 2*s, 16 * s}, 6*s, ed.Col)
		drawText(dst, ed.Name, x+w-tw/2+1*s, y+h-2*s, 12*s, colInk, 0.5)
	}
	if dim {
		roundRect(dst, rect{x - 3*s, y - 3*s, w + 6*s, h + 6*s}, 14*s, color.RGBA{0, 0, 0, 0x70})
	}
}

func hue(h float64) color.RGBA {
	h = math.Mod(h, 1) * 6
	f := func(n float64) uint8 {
		k := math.Mod(n+h, 6)
		return uint8(255 * (1 - math.Max(0, math.Min(math.Min(k, 4-k), 1))*0.65))
	}
	return color.RGBA{f(5), f(3), f(1), 0xff}
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

// ---- characters ----

func drawHero(dst *ebiten.Image, x, y, s, t float64, hit float64) {
	bob := math.Sin(t*3) * 3 * s
	y += bob
	ellipse(dst, x, y+48*s-bob, 40*s, 10*s, color.RGBA{0, 0, 0, 0x40})
	// sword behind
	poly(dst, []float64{x + 30*s, y + 10*s, x + 70*s, y - 50*s, x + 78*s, y - 44*s, x + 38*s, y + 16*s}, colInk)
	poly(dst, []float64{x + 33*s, y + 9*s, x + 70*s, y - 45*s, x + 74*s, y - 42*s, x + 37*s, y + 12*s}, colCream)
	// body
	ball(dst, x, y+18*s, 30*s, colBlue2)
	ellipse(dst, x, y+28*s, 22*s, 10*s, colYellow)
	// head
	ball(dst, x, y-22*s, 26*s, color.RGBA{0xff, 0xd6, 0xa8, 0xff})
	// helmet
	p := &vector.Path{}
	p.Arc(float32(x), float32(y-24*s), float32(29*s), math.Pi, 2*math.Pi, vector.Clockwise)
	p.Close()
	fillPath(dst, p, colInk)
	p = &vector.Path{}
	p.Arc(float32(x), float32(y-24*s), float32(26*s), math.Pi, 2*math.Pi, vector.Clockwise)
	p.Close()
	fillPath(dst, p, colGrey)
	ellipse(dst, x+4*s, y-56*s, 8*s, 14*s, colRed)
	// eyes
	circle(dst, x+6*s, y-16*s, 4.5*s, colInk)
	circle(dst, x+18*s, y-16*s, 4.5*s, colInk)
	circle(dst, x+7*s, y-17.5*s, 1.6*s, colWhite)
	circle(dst, x+19*s, y-17.5*s, 1.6*s, colWhite)
	ball(dst, x+30*s, y+14*s, 9*s, color.RGBA{0xff, 0xd6, 0xa8, 0xff})
	if hit > 0 {
		circle(dst, x, y, 50*s, alpha(colRed, hit*0.5))
	}
}

func drawEnemy(dst *ebiten.Image, e enemyDef, x, y, s, t, hit float64) {
	c := e.Col
	bob := math.Sin(t*2.5+1) * 4 * s
	ellipse(dst, x, y+48*s, 46*s, 11*s, color.RGBA{0, 0, 0, 0x40})
	eyes := func(ex, ey, r float64, angry bool) {
		for _, dx := range []float64{-1, 1} {
			circle(dst, ex+dx*r*1.5, ey, r+2*s, colInk)
			circle(dst, ex+dx*r*1.5, ey, r, colWhite)
			circle(dst, ex+dx*r*1.5-r*0.25, ey+r*0.1, r*0.5, colInk)
			if angry {
				poly(dst, []float64{ex + dx*r*0.4, ey - r*1.2, ex + dx*r*2.7, ey - r*1.9, ex + dx*r*2.7, ey - r*1.3, ex + dx*r*0.4, ey - r*0.7}, colInk)
			}
		}
	}
	switch e.Shape {
	case shSlime:
		sq := 1 + math.Sin(t*4)*0.06
		ellipse(dst, x, y+20*s, 47*s*sq, 37*s/sq, colInk)
		ellipse(dst, x, y+20*s, 44*s*sq, 34*s/sq, c)
		ellipse(dst, x-16*s, y+4*s, 12*s, 7*s, alpha(colWhite, 0.45))
		eyes(x-6*s, y+16*s, 8*s, false)
	case shGoblin:
		y += bob
		ball(dst, x, y+26*s, 26*s, color.RGBA{0x6a, 0x48, 0x30, 0xff})
		poly(dst, []float64{x - 30*s, y - 14*s, x - 64*s, y - 34*s, x - 26*s, y - 2*s}, colInk)
		poly(dst, []float64{x + 30*s, y - 14*s, x + 64*s, y - 34*s, x + 26*s, y - 2*s}, colInk)
		poly(dst, []float64{x - 28*s, y - 13*s, x - 58*s, y - 30*s, x - 26*s, y - 5*s}, c)
		poly(dst, []float64{x + 28*s, y - 13*s, x + 58*s, y - 30*s, x + 26*s, y - 5*s}, c)
		ball(dst, x, y-12*s, 30*s, c)
		eyes(x-6*s, y-14*s, 7*s, true)
	case shBat:
		y += bob*2 - 30*s
		flap := math.Sin(t*12) * 14 * s
		for _, dx := range []float64{-1, 1} {
			poly(dst, []float64{x + dx*14*s, y - 6*s, x + dx*70*s, y - 30*s - flap, x + dx*58*s, y + 10*s, x + dx*40*s, y + 2*s, x + dx*14*s, y + 16*s}, colInk)
			poly(dst, []float64{x + dx*16*s, y - 3*s, x + dx*64*s, y - 26*s - flap, x + dx*54*s, y + 6*s, x + dx*38*s, y, x + dx*16*s, y + 12*s}, shade(c, 0.75))
		}
		ball(dst, x, y, 26*s, c)
		eyes(x-6*s, y-2*s, 6*s, true)
	case shOrc, shKing:
		y += bob
		ball(dst, x, y+20*s, 42*s, shade(c, 0.8))
		ball(dst, x, y-26*s, 34*s, c)
		poly(dst, []float64{x - 14*s, y - 6*s, x - 8*s, y - 22*s, x - 2*s, y - 6*s}, colCream)
		poly(dst, []float64{x + 14*s, y - 6*s, x + 8*s, y - 22*s, x + 2*s, y - 6*s}, colCream)
		eyes(x-6*s, y-32*s, 7*s, true)
		if e.Shape == shKing {
			pts := []float64{x - 30*s, y - 52*s, x - 34*s, y - 86*s, x - 16*s, y - 66*s, x, y - 92*s, x + 16*s, y - 66*s, x + 34*s, y - 86*s, x + 30*s, y - 52*s}
			poly(dst, scalePts(pts, x, y-70*s, 1.12), colInk)
			poly(dst, pts, colYellow)
		}
	}
	if hit > 0 {
		circle(dst, x, y, 60*s, alpha(colWhite, hit*0.6))
	}
}

func scalePts(p []float64, cx, cy, k float64) []float64 {
	o := make([]float64, len(p))
	for i := 0; i < len(p); i += 2 {
		o[i], o[i+1] = cx+(p[i]-cx)*k, cy+(p[i+1]-cy)*k
	}
	return o
}

// hpBar draws a health bar with the number in it.
func hpBar(dst *ebiten.Image, x, y, w float64, hp, maxHP int, c color.RGBA) {
	r := rect{x - w/2, y, w, 22}
	roundRect(dst, rect{r.x - 3, r.y - 3, r.w + 6, r.h + 6}, 9, colInk)
	roundRect(dst, r, 7, shade(c, 0.35))
	f := math.Max(0, math.Min(1, float64(hp)/float64(max(1, maxHP))))
	if f > 0 {
		roundRect(dst, rect{r.x, r.y, math.Max(14, r.w*f), r.h}, 7, c)
		roundRect(dst, rect{r.x + 3, r.y + 3, math.Max(8, r.w*f-6), 6}, 3, alpha(colWhite, 0.35))
	}
	say(dst, itoa(hp)+"/"+itoa(maxHP), x, y+10, 15, colWhite, 0.5)
}

func chestIcon(dst *ebiten.Image, x, y, s float64, kind int) {
	body := color.RGBA{0xb0, 0x6a, 0x30, 0xff}
	trim := colYellow
	if kind == 2 {
		body, trim = colPurple, colGold
	}
	box(dst, rect{x - 40*s, y - 6*s, 80 * s, 44 * s}, body, 6*s)
	box(dst, rect{x - 42*s, y - 34*s, 84 * s, 30 * s}, shade(body, 1.15), 10*s)
	vector.DrawFilledRect(dst, float32(x-6*s), float32(y-34*s), float32(12*s), float32(72*s), trim, true)
	ball(dst, x, y-4*s, 7*s, trim)
}
