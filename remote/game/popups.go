package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Little floating texts in the world: damage numbers and rewards. Positions are world px.

type popup struct {
	text string
	id   string // "" for damage numbers
	x, y float64
	born int
	col  color.Color
}

var popups []*popup

// A floating line of text at a world point (rewards, petal names)
func popText(t string, x, y float64, c color.Color) {
	popups = append(popups, &popup{text: t, id: "text", x: x, y: y, born: frame, col: c})
}

// A damage number at a world point
func popDamage(n int, x, y float64, hurtsMote bool) {
	c := color.Color(colT)
	if hurtsMote {
		c = colDanger
	}
	popups = append(popups, &popup{text: fmt.Sprint(n), x: x, y: y, born: frame, col: c})
}

func drawPopups() {
	out := popups[:0]
	for _, p := range popups {
		age := frame - p.born
		if age > 70 {
			continue
		}
		out = append(out, p)
		sx := (p.x - cam.x) * zoom
		sy := (p.y-cam.y)*zoom - float64(min(age, 30))*0.5
		a := 1.0
		if age > 45 {
			a = 1 - float64(age-45)/25
		}
		t, c := p.text, p.col
		size := 13.0
		if p.id == "" {
			size = 15
		}
		ops = append(ops, func(dst *ebiten.Image) {
			r, g, b, _ := c.RGBA()
			col := color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(255 * a)}
			drawText(dst, t, sx, sy, textOpt{size: size, col: col, align: text.AlignCenter, shadow: a > 0.6})
		})
	}
	popups = out
}
