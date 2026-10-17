package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Little floating texts in the world: "+3 Stone" when you pick things up, and damage
// numbers. Positions are world px in the loaded window (shifted with it).

type popup struct {
	text string
	id   string // item id for merging pickups
	n    int
	x, y float64
	born int
	col  color.Color
}

var popups []*popup

// Show what was picked up over the mote, merging repeated pickups of the same item
func popItem(id string, n int) {
	if n <= 0 || items[id] == nil {
		return
	}
	lanes := 0
	for _, p := range popups {
		if p.id == id && frame-p.born < 50 {
			p.n += n
			p.text = fmt.Sprintf("+%d %s", p.n, items[id].name)
			p.born = frame
			p.x = player.x + player.w/2
			return
		}
		if p.id != "" && frame-p.born < 70 {
			lanes++
		}
	}
	popups = append(popups, &popup{text: fmt.Sprintf("+%d %s", n, items[id].name), id: id, n: n,
		x: player.x + player.w/2, y: player.y - 4 - float64(lanes%4)*10, born: frame, col: colInk})
}

// A damage number at a world point
func popDamage(n int, x, y float64, hurtsMote bool) {
	c := color.Color(colT)
	if hurtsMote {
		c = colDanger
	}
	popups = append(popups, &popup{text: fmt.Sprint(n), x: x, y: y, born: frame, col: c})
}

func shiftPopups(dx float64) {
	for _, p := range popups {
		p.x -= dx
	}
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
