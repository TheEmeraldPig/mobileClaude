package game

import (
	"math"
	"strings"
)

// Blueprints build ready-made structures. While one is held, a see-through hologram
// shows where it will go: green when it fits, red when it doesn't.
//
// Template characters: B brick, P plank, G glass, T torch, S soil, W water, O stone,
// D dirt, R door (closed), space = cleared to air, . = left as it is.

var bpTemplates = map[string][]string{}

var bpBlock = map[byte]uint8{'B': BRICK, 'P': PLANK, 'G': GLASS, 'T': TORCH, 'S': SOIL, 'W': WATER, 'O': STONE, 'D': DIRT, 'R': DOOR, ' ': EMPTY}

func grid2(w, h int, fill byte) [][]byte {
	g := make([][]byte, h)
	for y := range g {
		g[y] = []byte(strings.Repeat(string(fill), w))
	}
	return g
}

func toRows(g [][]byte) []string {
	out := make([]string, len(g))
	for i, r := range g {
		out[i] = string(r)
	}
	return out
}

func initBlueprints() {
	// Hut: plank roof, brick walls, a doorway on the left, windows on the right, a torch inside
	{
		w, h := 24, 24
		g := grid2(w, h, ' ')
		for y := 0; y < 5; y++ { // roof
			for x := 0; x < w; x++ {
				if x >= 5-y-1 && x < w-(5-y-1) {
					g[y][x] = 'P'
				} else {
					g[y][x] = '.'
				}
			}
		}
		for y := 5; y < h-1; y++ {
			g[y][0], g[y][1], g[y][w-2], g[y][w-1] = 'B', 'B', 'B', 'B'
		}
		for y := h - 16; y < h-1; y++ { // doorway with a door
			g[y][0], g[y][1] = 'R', 'R'
		}
		for y := 8; y < 11; y++ {
			g[y][w-2], g[y][w-1] = 'G', 'G'
		}
		for x := 0; x < w; x++ {
			g[h-1][x] = 'B'
		}
		g[6][w/2] = 'T'
		bpTemplates["hut"] = toRows(g)
	}
	// Tower: three floors joined by gaps you levitate through
	{
		w, h := 18, 48
		g := grid2(w, h, ' ')
		for y := 0; y < h; y++ {
			g[y][0], g[y][1], g[y][w-2], g[y][w-1] = 'B', 'B', 'B', 'B'
		}
		for x := 0; x < w; x++ {
			g[0][x], g[1][x] = 'B', 'B'
			g[h-1][x] = 'B'
			if x%4 == 1 {
				g[0][x] = '.'
			}
		}
		for _, fy := range []int{16, 32} {
			for x := 2; x < w-2; x++ {
				if x < 4 || x > 9 { // leave a gap to fly up through
					g[fy][x] = 'P'
				}
			}
		}
		for _, ty := range []int{4, 20, 36} {
			g[ty][w-4] = 'T'
			for y := ty; y < ty+3; y++ {
				g[y][0], g[y][1] = 'G', 'G'
			}
		}
		for y := h - 16; y < h-1; y++ {
			g[y][w-2], g[y][w-1] = 'R', 'R'
		}
		bpTemplates["tower"] = toRows(g)
	}
	// Bridge: a long deck with posts
	{
		w, h := 52, 5
		g := grid2(w, h, '.')
		for x := 0; x < w; x++ {
			g[3][x], g[4][x] = 'P', 'P'
			if x%8 == 0 || x == w-1 {
				g[1][x], g[2][x] = 'P', 'P'
				g[0][x] = 'T'
			}
		}
		bpTemplates["bridge"] = toRows(g)
	}
	// Farm: tilled soil with a water channel down the middle, in a brick bed
	{
		w, h := 32, 6
		g := grid2(w, h, ' ')
		for x := 0; x < w; x++ {
			g[h-1][x] = 'B'
			switch {
			case x == 0 || x == w-1:
				g[h-2][x] = 'B'
			case x == w/2 || x == w/2-1:
				g[h-2][x] = 'W'
			default:
				g[h-2][x] = 'S'
			}
		}
		for y := 0; y < h-2; y++ {
			g[y][0], g[y][w-1] = 'P', 'P'
		}
		g[0][0], g[0][w-1] = 'T', 'T'
		bpTemplates["farm"] = toRows(g)
	}
	// Bunker: thick stone walls, a slit window and a light
	{
		w, h := 28, 22
		g := grid2(w, h, ' ')
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < 3 || x >= w-3 || y < 3 || y >= h-2 {
					g[y][x] = 'O'
				}
			}
		}
		for x := w/2 - 3; x < w/2+3; x++ {
			g[1][x] = 'G'
		}
		for y := h - 17; y < h-2; y++ {
			for x := 0; x < 3; x++ {
				g[y][x] = 'R'
			}
		}
		g[4][w/2] = 'T'
		bpTemplates["bunker"] = toRows(g)
	}
}

// ---- Placement ----

func heldBlueprint() string {
	if selected < 0 || inv[selected] == nil {
		return ""
	}
	return items[inv[selected].ID].blueprint
}

// The template turned by buildRot quarter turns and mirrored by buildFlip
func bpRows(name string) []string {
	t := bpTemplates[name]
	g := make([][]byte, len(t))
	for y, r := range t {
		g[y] = []byte(r)
	}
	if buildFlip {
		for _, r := range g {
			for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
				r[i], r[j] = r[j], r[i]
			}
		}
	}
	for k := 0; k < buildRot; k++ { // a quarter turn clockwise
		h, w := len(g), len(g[0])
		n := make([][]byte, w)
		for x := 0; x < w; x++ {
			n[x] = make([]byte, h)
			for y := 0; y < h; y++ {
				n[x][h-1-y] = g[y][x]
			}
		}
		g = n
	}
	return toRows(g)
}

func bpFits(t []string, ax, ay int) bool {
	h, w := len(t), len(t[0])
	x0, y0 := ax-w/2, ay-h+1
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	if math.Hypot(float64(ax)-pcx, float64(ay-h/2)-pcy) > 80 {
		return false
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if t[y][x] == '.' {
				continue
			}
			gx, gy := x0+x, y0+y
			if gx < 0 || gx >= W || gy < 0 || gy >= H {
				return false
			}
			switch grid[gy*W+gx] {
			case BEDROCK, CORE, CHEST, LAVA:
				return false
			}
		}
	}
	return true
}

func placeBlueprint(it *Item, tx, ty int) {
	t := bpRows(it.blueprint)
	if !bpFits(t, tx, ty) {
		toast("It doesn't fit there")
		return
	}
	h, w := len(t), len(t[0])
	x0, y0 := tx-w/2, ty-h+1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b, ok := bpBlock[t[y][x]]
			if !ok {
				continue
			}
			i := (y0+y)*W + x0 + x
			grid[i] = b
			paint[i] = 0
			shade[i] = int8(int(hash32(x0+x, y0+y, 5)%11) - 5)
			// houses get a back wall, so they keep creatures out
			if it.blueprint == "hut" || it.blueprint == "tower" || it.blueprint == "bunker" {
				if wall[i] == 0 {
					wall[i] = PLANK
					if it.blueprint == "bunker" {
						wall[i] = STONE
					}
				}
			}
			markDirty(x0 + x)
		}
	}
	burst(30, float64(tx), float64(ty-h/2), tipRGB, 2, 2, 30)
	takeSelected(1)
	holoSet = false
	sfx("build")
	toast(it.name[:len(it.name)-len(" blueprint")] + " built!")
	shake = math.Max(shake, 3)
}

// Draw the hologram over the world pixels
func drawHologram(blend func(x, y int, c [3]int, a float64), ox, oy int) {
	name := heldBlueprint()
	if name == "" || cur != scPlaying {
		return
	}
	ax, ay, _ := holoAim()
	t := bpRows(name)
	ok := bpFits(t, ax, ay)
	tint := [3]int{120, 255, 160}
	if !ok {
		tint = [3]int{255, 110, 110}
	}
	pulse := 0.45 + 0.15*math.Sin(float64(frame)*0.15)
	h, w := len(t), len(t[0])
	x0, y0 := ax-w/2, ay-h+1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := t[y][x]
			if ch == '.' {
				continue
			}
			c := tint
			a := 0.12
			if b := bpBlock[ch]; b != EMPTY {
				c = mix(blockRGB[b], tint, 0.45)
				a = pulse
			}
			blend(x0+x-ox, y0+y-oy, c, a)
		}
	}
}
