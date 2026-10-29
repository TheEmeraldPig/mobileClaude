package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Drawing cards, charms and hexes in the pixel style of the UI: dark outlines, a light
// top-left edge and a shaded bottom-right edge, 3px "pixels".

var (
	colPaper   = rgb(0xf4eedc)
	colPaperHi = rgb(0xffffff)
	colPaperLo = rgb(0xcfc4a6)
	colRed     = rgb(0xd8434e)
	colRedLo   = rgb(0x8c1f2c)
	colInkCard = rgb(0x2a2344)
	colFelt    = rgb(0x1b2a2a)
	colFeltHi  = rgb(0x22363a)
)

// 7x7 suit pips
var suitPips = [4][]string{
	{"...#...", "..###..", ".#####.", "#######", "#######", "..#.#..", ".#####."}, // spade
	{".##.##.", "#######", "#######", "#######", ".#####.", "..###..", "...#..."}, // heart
	{"..###..", "..###..", "##.#.##", "#######", "##.#.##", "...#...", ".#####."}, // club
	{"...#...", "..###..", ".#####.", "#######", ".#####.", "..###..", "...#..."}, // diamond
}

func suitColor(s int) color.Color {
	if isRed(s) {
		return colRed
	}
	return colInkCard
}

func drawPip(dst *ebiten.Image, s int, x, y, px float64, c color.Color) {
	for j, row := range suitPips[s] {
		for i := 0; i < len(row); i++ {
			if row[i] == '#' {
				fill(dst, x+float64(i)*px, y+float64(j)*px, px, px, c)
			}
		}
	}
}

type cardLook struct {
	faceDown bool
	selected bool
	dim      bool
	glowGold bool // scoring right now
}

// drawCard draws a playing card with its top-left corner at x, y
func drawCard(dst *ebiten.Image, c Card, x, y, w, h float64, look cardLook) {
	if look.faceDown {
		drawCardBack(dst, x, y, w, h)
		return
	}
	body, hi, lo := color.Color(colPaper), color.Color(colPaperHi), color.Color(colPaperLo)
	if c.Glow > 0 {
		hi = colGoldHi
	}
	bevel(dst, Rect{x, y, w, h}, body, hi, lo, true)
	if look.glowGold {
		fill(dst, x-6, y-6, w+12, 3, colT)
		fill(dst, x-6, y+h+3, w+12, 3, colT)
		fill(dst, x-6, y-3, 3, h+6, colT)
		fill(dst, x+w+3, y-3, 3, h+6, colT)
	}
	col := suitColor(c.S)
	fs := max(11, w*0.24)
	drawText(dst, c.Name(), x+5, y+fs*0.62+3, textOpt{size: fs, display: true, col: col})
	small := max(1.2, w/52)
	drawPip(dst, c.S, x+5, y+fs*1.2+4, small, col)
	// the big pip in the middle
	px := max(2, w/14)
	drawPip(dst, c.S, x+w/2-3.5*px, y+h/2-3.5*px+px*0.5, px, col)
	// rank again, bottom right
	drawText(dst, c.Name(), x+w-5, y+h-fs*0.62-3, textOpt{size: fs, display: true, col: col, align: text.AlignEnd})
	if c.Bonus > 0 {
		t := fmt.Sprintf("+%d", c.Bonus)
		tw := textWidth(t, 9, true) + 8
		fill(dst, x+3, y+h-17, tw, 14, colGoldLo)
		drawText(dst, t, x+7, y+h-10, textOpt{size: 9, display: true, col: colO})
	}
	if look.dim {
		fill(dst, x, y, w, h, color.NRGBA{20, 16, 40, 140})
	}
}

func drawCardBack(dst *ebiten.Image, x, y, w, h float64) {
	bevel(dst, Rect{x, y, w, h}, colB, colL, colS, true)
	// a diamond lattice
	p := max(3, w/16)
	for j := 0.0; j+p <= h-12; j += p * 2 {
		for i := 0.0; i+p <= w-12; i += p * 2 {
			off := 0.0
			if int(j/(p*2))%2 == 1 {
				off = p
			}
			if i+off+p <= w-12 {
				fill(dst, x+6+i+off, y+6+j, p, p, colS)
			}
		}
	}
	fill(dst, x+w/2-p*1.5, y+h/2-p*1.5, p*3, p*3, colT)
	fill(dst, x+w/2-p*0.5, y+h/2-p*0.5, p, p, colGoldHi)
}

// A charm: a pendant-shaped token in the charm's colours, with its initial
func drawCharm(dst *ebiten.Image, id string, x, y, size float64, frozen bool) {
	d := charms[id]
	if d == nil {
		return
	}
	body, light, dark := rgb(d.art[0]), rgb(d.art[1]), rgb(d.art[2])
	p := size / 12
	// the loop it hangs from
	fill(dst, x+size/2-p, y, p*2, p, colO)
	fill(dst, x+size/2-p*2, y+p, p, p*2, colO)
	fill(dst, x+size/2+p, y+p, p, p*2, colO)
	shape := charmShapes[charmShape(id)]
	for j, row := range shape {
		for i := 0; i < len(row); i++ {
			var c color.Color
			switch row[i] {
			case 'o':
				c = colO
			case 'b':
				c = body
			case 'l':
				c = light
			case 's':
				c = dark
			case 't':
				c = colT
			default:
				continue
			}
			fill(dst, x+float64(i)*p, y+float64(j+2)*p, p, p, c)
		}
	}
	if frozen {
		fill(dst, x, y+p*2, size, size-p*2, color.NRGBA{150, 200, 255, 130})
	}
}

// Charm shapes, 12 wide x 10 tall (below the 2-row loop)
var charmShapes = [][]string{
	{ // round amulet
		"...oooooo...",
		"..ollllbbo..",
		".ollbbbbbso.",
		".olbbttbbso.",
		".olbtbbtbso.",
		".olbbttbbso.",
		".obbbbbbsso.",
		"..obbbssso..",
		"...oooooo...",
		"............",
	},
	{ // gem
		"....oooo....",
		"...ollbbo...",
		"..ollbbbso..",
		".ollbbbbsso.",
		".olbbttbbso.",
		"..obbbbbso..",
		"...obbbso...",
		"....obso....",
		".....oo.....",
		"............",
	},
	{ // shield
		".oooooooooo.",
		".ollllbbbbo.",
		".olbbbbbbso.",
		".olbbttbbso.",
		".olbbttbbso.",
		".obbbbbbbso.",
		"..obbbbbso..",
		"...obbsso...",
		"....osso....",
		".....oo.....",
	},
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
	bevel(dst, Rect{x, y, w, h}, rgb(0x2a1840), rgb(0x5a3a80), rgb(0x150a24), true)
	p := max(2, w/12)
	eye := []string{
		"..#####..",
		".#.....#.",
		"#..ooo..#",
		"#.ooxoo.#",
		"#..ooo..#",
		".#.....#.",
		"..#####..",
	}
	ox, oy := x+w/2-4.5*p, y+h/2-3.5*p
	for j, row := range eye {
		for i := 0; i < len(row); i++ {
			switch row[i] {
			case '#':
				fill(dst, ox+float64(i)*p, oy+float64(j)*p, p, p, rgb(0xb070ff))
			case 'o':
				fill(dst, ox+float64(i)*p, oy+float64(j)*p, p, p, rgb(0xe8d0ff))
			case 'x':
				fill(dst, ox+float64(i)*p, oy+float64(j)*p, p, p, colDanger)
			}
		}
	}
}

// A tome: a little book in the hand's family colour
func drawTome(dst *ebiten.Image, f Family, x, y, w, h float64) {
	c := familyColors[max(0, int(f))]
	bevel(dst, Rect{x, y, w, h}, rgb(c[0]), rgb(c[1]), rgb(c[2]), true)
	fill(dst, x+6, y+5, 4, h-10, rgb(c[2]))
	p := max(2, w/14)
	fill(dst, x+w/2-p, y+h/2-p*3, p*2, p*6, colT)
	fill(dst, x+w/2-p*3, y+h/2-p, p*6, p*2, colT)
}

var familyColors = [3][3]uint32{
	{0xe07a5a, 0xffc0a8, 0x8c3a22}, // Same
	{0x5aa0e0, 0xb0d8ff, 0x22508c}, // Counting
	{0x6ac87a, 0xc0f0c8, 0x2a7a3a}, // Suits
}

// Pixel heart for lives
func drawHeart(dst *ebiten.Image, x, y, px float64, full bool) {
	rows := []string{".##.##.", "#######", "#######", ".#####.", "..###..", "...#..."}
	c := color.Color(colDanger)
	if !full {
		c = colSlate
	}
	for j, r := range rows {
		for i := 0; i < len(r); i++ {
			if r[i] == '#' {
				fill(dst, x+float64(i)*px, y+float64(j)*px, px, px, c)
			}
		}
	}
}
