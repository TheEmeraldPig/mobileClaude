package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Drawing cards, charms, hexes, tomes and hearts: smooth shapes with rounded corners
// and soft shadows. A card is drawn once into a cached image (cardImage), so the
// animations can move, tilt and scale it cheaply every frame.

var (
	colPaper   = rgb(0xfff6e6)
	colPaperLo = rgb(0xe8d5b8)
	colRed     = rgb(0xd8382e)
	colInkCard = rgb(0x2b1d2a)
)

func suitColor(s int) color.Color {
	if isRed(s) {
		return colRed
	}
	return colInkCard
}

// suitPath builds a suit shape centred on cx, cy, about size wide (device pixels)
func suitShape(dst *ebiten.Image, s int, cx, cy, size float64, c color.Color) {
	k := uiScale
	cx, cy, size = cx*k, cy*k, size*k
	h := size / 2
	circ := func(x, y, r float64) {
		vector.DrawFilledCircle(dst, float32(x), float32(y), float32(r), c, true)
	}
	poly := func(pts ...float64) {
		var p vector.Path
		p.MoveTo(float32(pts[0]), float32(pts[1]))
		for i := 2; i < len(pts); i += 2 {
			p.LineTo(float32(pts[i]), float32(pts[i+1]))
		}
		p.Close()
		fillDevice(dst, &p, c)
	}
	switch s {
	case Hearts:
		r := h * 0.52
		circ(cx-r*0.95, cy-h*0.28, r)
		circ(cx+r*0.95, cy-h*0.28, r)
		poly(cx-h*0.99, cy-h*0.12, cx+h*0.99, cy-h*0.12, cx, cy+h*0.95)
	case Diamonds:
		poly(cx, cy-h, cx+h*0.74, cy, cx, cy+h, cx-h*0.74, cy)
	case Spades:
		r := h * 0.5
		circ(cx-r*0.95, cy+h*0.12, r)
		circ(cx+r*0.95, cy+h*0.12, r)
		poly(cx-h*0.97, cy, cx+h*0.97, cy, cx, cy-h*0.98)
		poly(cx, cy+h*0.2, cx+h*0.35, cy+h, cx-h*0.35, cy+h)
	case Clubs:
		r := h * 0.43
		circ(cx, cy-h*0.48, r)
		circ(cx-h*0.5, cy+h*0.1, r)
		circ(cx+h*0.5, cy+h*0.1, r)
		poly(cx-h*0.12, cy-h*0.1, cx+h*0.12, cy-h*0.1, cx+h*0.36, cy+h, cx-h*0.36, cy+h)
	}
}

// suitSprite draws a suit from a cached white image (cheap enough for every frame)
var suitCache = map[[2]int]*ebiten.Image{}

func suitSprite(dst *ebiten.Image, st int, cx, cy, size float64, c color.Color) {
	px := int(math.Round(size * uiScale))
	if px < 2 {
		return
	}
	key := [2]int{st, px}
	img := suitCache[key]
	if img == nil {
		img = ebiten.NewImage(px+4, px+4)
		old := uiScale
		uiScale = 1
		suitShape(img, st, float64(px+4)/2, float64(px+4)/2, float64(px), color.White)
		uiScale = old
		suitCache[key] = img
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(cx*uiScale-float64(px+4)/2, cy*uiScale-float64(px+4)/2)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(img, op)
}

// fill a path already in device pixels
func fillDevice(dst *ebiten.Image, p *vector.Path, c color.Color) { fillPath(dst, p, c) }

type cardLook struct {
	faceDown bool
	selected bool
	dim      bool
	glowGold bool // scoring right now
}

// ---- The card image cache ----

type cardKey struct {
	r, s, bonus, glow int
	back              bool
	w, h              int
	scale             float64
}

var cardCache = map[cardKey]*ebiten.Image{}

// A card's picture at w x h logical pixels (with a margin for its rim)
func cardImage(c Card, back bool, w, h float64) *ebiten.Image {
	key := cardKey{c.R, c.S, c.Bonus, c.Glow, back, int(w), int(h), uiScale}
	if img := cardCache[key]; img != nil {
		return img
	}
	if len(cardCache) > 400 {
		for k, img := range cardCache {
			img.Deallocate()
			delete(cardCache, k)
		}
	}
	const m = 3.0 // margin
	img := ebiten.NewImage(int(math.Ceil((w+2*m)*uiScale)), int(math.Ceil((h+2*m)*uiScale)))
	w, h = float64(int(w)), float64(int(h))
	rad := w * 0.11
	if back {
		roundRect(img, m-1.5, m-1.5, w+3, h+3, rad+1.5, colO)
		roundRect(img, m, m, w, h, rad, rgb(0xb8322a))
		roundRect(img, m+w*0.08, m+w*0.08, w*0.84, h-w*0.16, rad*0.6, rgb(0x8f2219))
		// a lattice of little diamonds and a gold centre
		step := w * 0.16
		for yy := m + w*0.16; yy < m+h-w*0.12; yy += step {
			for xx := m + w*0.16; xx < m+w-w*0.12; xx += step {
				suitShape(img, Diamonds, xx, yy, step*0.42, rgb(0xc8483a))
			}
		}
		circle(img, m+w/2, m+h/2, w*0.2, colGoldLo)
		suitShape(img, Diamonds, m+w/2, m+h/2, w*0.26, colGoldHi)
	} else {
		rim := colO
		if c.Glow > 0 {
			rim = colGoldLo
		}
		roundRect(img, m-1.5, m-1.5, w+3, h+3, rad+1.5, rim)
		roundRect(img, m, m, w, h, rad, colPaperLo)
		roundRect(img, m, m, w, h-2.5, rad, colPaper)
		col := suitColor(c.S)
		fs := math.Max(10, w*0.22)
		drawText(img, c.Name(), m+w*0.15, m+fs*0.55+3, textOpt{size: fs, display: true, col: col, align: text.AlignCenter})
		suitShape(img, c.S, m+w*0.15, m+fs*1.3+9, w*0.15, col)
		suitShape(img, c.S, m+w/2, m+h*0.53, w*0.46, col)
		drawText(img, c.Name(), m+w*0.85, m+h-fs*0.55-4, textOpt{size: fs, display: true, col: col, align: text.AlignCenter})
		if c.Bonus > 0 {
			t := fmt.Sprintf("+%d", c.Bonus)
			tw := textWidth(t, 9, true) + 10
			roundRect(img, m+3, m+h-19, tw, 15, 6, colGoldLo)
			drawText(img, t, m+3+tw/2, m+h-12, textOpt{size: 9, display: true, col: colInk, align: text.AlignCenter})
		}
	}
	cardCache[key] = img
	return img
}

// drawCardFX draws a card centred on cx, cy, turned by rot radians and scaled
func drawCardFX(dst *ebiten.Image, c Card, cx, cy, w, h, rot, scale float64, look cardLook) {
	img := cardImage(c, look.faceDown, w, h)
	iw, ih := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	// a soft shadow below, further when lifted
	lift := 0.0
	if look.selected {
		lift = 6
	}
	sh := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sh.GeoM.Translate(-iw/2, -ih/2)
	sh.GeoM.Rotate(rot)
	sh.GeoM.Scale(scale, scale)
	sh.GeoM.Translate((cx+3+lift*0.4)*uiScale, (cy+5+lift)*uiScale)
	sh.ColorScale.Scale(0, 0, 0, 0.35)
	dst.DrawImage(img, sh)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-iw/2, -ih/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx*uiScale, cy*uiScale)
	if look.dim {
		op.ColorScale.Scale(0.55, 0.5, 0.5, 1)
	}
	if look.glowGold {
		// a warm halo behind it
		g := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		g.GeoM.Translate(-iw/2, -ih/2)
		g.GeoM.Rotate(rot)
		g.GeoM.Scale(scale*1.08, scale*1.06)
		g.GeoM.Translate(cx*uiScale, cy*uiScale)
		g.ColorScale.Scale(1, 0.72, 0.2, 0.75)
		g.Blend = ebiten.BlendLighter
		dst.DrawImage(img, g)
	}
	dst.DrawImage(img, op)
}

// drawCard draws a card upright with its top-left corner at x, y
func drawCard(dst *ebiten.Image, c Card, x, y, w, h float64, look cardLook) {
	drawCardFX(dst, c, x+w/2, y+h/2, w, h, 0, 1, look)
}

// ---- Charms, hexes, tomes, hearts ----

// A charm: a round medallion in the charm's colours with its initial
func drawCharm(dst *ebiten.Image, id string, x, y, size float64, frozen bool) {
	d := charms[id]
	if d == nil {
		return
	}
	body, light, dark := rgb(d.art[0]), rgb(d.art[1]), rgb(d.art[2])
	cx, cy, r := x+size/2, y+size/2, size/2
	circle(dst, cx+1.5, cy+2.5, r, color.NRGBA{0, 0, 0, 90})
	circle(dst, cx, cy, r, colO)
	circle(dst, cx, cy, r-2, dark)
	circle(dst, cx-r*0.08, cy-r*0.1, r-4, body)
	circle(dst, cx-r*0.3, cy-r*0.35, r*0.28, withAlpha(light, 0.8))
	switch charmShape(id) {
	case 0:
		suitSprite(dst, Diamonds, cx, cy+r*0.05, r*0.75, light)
	case 1:
		circle(dst, cx, cy+r*0.05, r*0.3, light)
		circle(dst, cx, cy+r*0.05, r*0.15, dark)
	case 2:
		suitSprite(dst, Hearts, cx, cy+r*0.05, r*0.7, light)
	}
	if frozen {
		circle(dst, cx, cy, r, color.NRGBA{150, 200, 255, 140})
	}
}

func charmShape(id string) int {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	return ((h % 3) + 3) % 3
}

// A hex: a dark card with a single staring eye
func drawHex(dst *ebiten.Image, x, y, w, h float64) {
	roundRect(dst, x+2, y+4, w, h, w*0.12, color.NRGBA{0, 0, 0, 90})
	roundRect(dst, x-1.5, y-1.5, w+3, h+3, w*0.12+1.5, colO)
	roundRect(dst, x, y, w, h, w*0.12, rgb(0x3a1838))
	roundRect(dst, x+w*0.1, y+w*0.1, w*0.8, h-w*0.2, w*0.08, rgb(0x55224f))
	cx, cy := x+w/2, y+h/2
	ew := w * 0.34
	var p vector.Path
	s := uiScale
	p.MoveTo(float32((cx-ew)*s), float32(cy*s))
	p.QuadTo(float32(cx*s), float32((cy-ew*1.1)*s), float32((cx+ew)*s), float32(cy*s))
	p.QuadTo(float32(cx*s), float32((cy+ew*1.1)*s), float32((cx-ew)*s), float32(cy*s))
	fillPath(dst, &p, rgb(0xf0d8ff))
	circle(dst, cx, cy, ew*0.38, rgb(0xc04aa0))
	circle(dst, cx, cy, ew*0.17, colO)
}

// A tome: a little book in the hand's family colour
func drawTome(dst *ebiten.Image, f Family, x, y, w, h float64) {
	c := familyColors[max(0, int(f))]
	roundRect(dst, x+2, y+4, w, h, 5, color.NRGBA{0, 0, 0, 90})
	roundRect(dst, x-1.5, y-1.5, w+3, h+3, 6, colO)
	roundRect(dst, x, y, w, h, 5, rgb(c[2]))
	roundRect(dst, x+w*0.18, y, w*0.82, h-3, 5, rgb(c[0]))
	roundRect(dst, x+w*0.3, y+h*0.15, w*0.6, h*0.12, 3, withAlpha(rgb(c[1]), 0.7))
	cx, cy := x+w*0.59, y+h*0.6
	l := math.Min(w, h) * 0.22
	vector.StrokeLine(dst, float32((cx-l)*uiScale), float32(cy*uiScale), float32((cx+l)*uiScale), float32(cy*uiScale), float32(3*uiScale), colGoldHi, true)
	vector.StrokeLine(dst, float32(cx*uiScale), float32((cy-l)*uiScale), float32(cx*uiScale), float32((cy+l)*uiScale), float32(3*uiScale), colGoldHi, true)
}

var familyColors = [3][3]uint32{
	{0xe8703c, 0xffc8a0, 0x8c3412}, // Same
	{0x3c9ad8, 0xb0dcff, 0x1c5288}, // Counting
	{0x5cb85a, 0xc0f0b8, 0x2a6a2a}, // Suits
}

// A heart for lives; px sets the size (the heart is 7*px wide)
func drawHeart(dst *ebiten.Image, x, y, px float64, full bool) {
	c := color.Color(colDanger)
	if !full {
		c = colSlate
	}
	suitSprite(dst, Hearts, x+3.5*px, y+3*px, 6.6*px, c)
}
