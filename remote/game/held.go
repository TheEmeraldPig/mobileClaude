package game

import "math"

// The item in the mote's hand, drawn while it's being used: pickaxes and other tools
// swing back and forth toward the aim, swords slash through an arc, and blocks and
// other things bob toward where they go.

var (
	usedAt   = -1000 // last frame an item was in use
	swingAt  = -1000 // last sword swing
	swingAng float64
)

const swingFrames = 10

func drawHeld(put func(x, y int, c [3]int, s int), ox, oy int) {
	p := &player
	if p.invuln > 0 && (p.invuln/4)%2 == 0 {
		return
	}
	var it *Item
	switch {
	case handMining && useActive:
		it = items["hands"]
	case selected >= 0 && inv[selected] != nil:
		it = items[inv[selected].ID]
	}
	sword := it != nil && it.dmg > 0 && !isPick(it.id)
	if it == nil || (frame-usedAt > 6 && !(sword && frame-swingAt < swingFrames+4)) {
		return
	}
	// the hand: front of the mote, a little below the middle
	pcx, pcy := p.x/CELL+p.w/CELL/2, p.y/CELL+p.h/CELL/2
	aim := math.Atan2(float64(aimY)+0.5-pcy, float64(aimX)+0.5-pcx)
	left := math.Cos(aim) < 0
	side := 4.5
	if left {
		side = -4.5
	}
	hx, hy := pcx-float64(ox)+side, pcy-float64(oy)+2
	t := float64(frame - usedAt + useTick)

	switch {
	case sword:
		k := math.Min(1, float64(frame-swingAt)/swingFrames)
		k = 1 - (1-k)*(1-k) // fast start, eases out
		dir := 1.0
		if left {
			dir = -1
		}
		th := swingAng + dir*(-1.5+2.6*k)
		drawRotIcon(put, it.icon, hx, hy, th, -math.Pi/4, 1.5, 6.5, left, 1.0)
	case isPick(it.id) && it.id != "hands", it.id == "hoe", it.id == "hammer":
		th := aim + 0.65*math.Sin(t*0.4) - 0.2
		if left {
			th = aim - 0.65*math.Sin(t*0.4) + 0.2
		}
		drawRotIcon(put, it.icon, hx, hy, th, -math.Pi/2, 4, 7.5, left, 0.85)
	default:
		// blocks, hands, food...: held out and pushed toward the aim
		push := 1 + 1.5*math.Abs(math.Sin(t*0.3))
		x0 := int(math.Round(hx+math.Cos(aim)*push)) - 2
		y0 := int(math.Round(hy+math.Sin(aim)*push)) - 2
		for y, r := range it.icon {
			for x := 0; x < 8; x += 2 {
				if y%2 == 1 || x >= len(r) {
					continue
				}
				if c, ok := iconCol[r[x]]; ok {
					put(x0+x/2, y0+y/2, c, 0)
				}
			}
		}
	}
}

// Draw an 8x8 icon rotated about its handle (u0, v0). base is the angle the icon
// points at as drawn; th is the angle to point at. mirror flips it for the left side.
func drawRotIcon(put func(x, y int, c [3]int, s int), icon []string, px, py, th, base, u0, v0 float64, mirror bool, scale float64) {
	if mirror {
		u0 = 8 - u0
		base = math.Pi - base
	}
	rot := th - base
	cs, sn := math.Cos(-rot), math.Sin(-rot)
	const R = 10
	for y := int(py) - R; y <= int(py)+R; y++ {
		for x := int(px) - R; x <= int(px)+R; x++ {
			dx, dy := float64(x)+0.5-px, float64(y)+0.5-py
			u := u0 + (dx*cs-dy*sn)/scale
			v := v0 + (dx*sn+dy*cs)/scale
			if u < 0 || v < 0 || u >= 8 || v >= 8 {
				continue
			}
			ui, vi := int(u), int(v)
			if mirror {
				ui = 7 - ui
			}
			r := icon[vi]
			if ui >= len(r) {
				continue
			}
			if c, ok := iconCol[r[ui]]; ok {
				put(x, y, c, 0)
			}
		}
	}
}
