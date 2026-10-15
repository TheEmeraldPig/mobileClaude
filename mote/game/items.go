package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- Items, inventory, hotbar, crafting ----
// Icons are 8x8 pixel maps drawn with the same palette as the world and the mote.

var iconCol = map[byte][3]int{
	'O': {34, 28, 64}, 'S': {200, 206, 218}, 's': {130, 136, 150}, 'H': {150, 100, 60}, 'h': {110, 70, 40},
	'M': {170, 176, 190}, 'm': {110, 116, 130}, 'W': {59, 142, 232}, 'w': {140, 196, 250},
	'Y': {226, 194, 107}, 'y': {190, 158, 80}, 'G': {120, 126, 140}, 'g': {90, 96, 110},
	'B': {124, 98, 255}, 'L': {196, 180, 255}, 'D': {90, 67, 214}, 'T': {255, 222, 120},
	'K': {60, 58, 70}, 'k': {100, 98, 112}, 'R': {196, 108, 84}, 'r': {140, 70, 56},
}

type Item struct {
	id, name, desc string
	tool, once     bool
	place          uint8
	icon           []string
	img            *ebiten.Image
	give           int
}

var itemOrder = []string{"pickaxe", "bucket", "sand", "stone", "brick", "water", "gem", "bomb"}

var items = map[string]*Item{
	"pickaxe": {name: "Pickaxe", tool: true, desc: "Digs sand, stone and brick", give: 1, icon: []string{
		".SSSSSs.", "S..hH..s", "...hH...", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"bucket": {name: "Bucket", tool: true, desc: "Scoops up water", give: 1, icon: []string{
		"..O..O..", ".O....O.", "OMMMMMMO", "OMwWWWmO", ".MWWWWm.", ".MMMMmm.", "..Mmmm..", "........"}},
	"sand": {name: "Sand", place: SAND, desc: "Place sand", give: 200, icon: []string{
		"OOOOOOOO", "OYYyYYYO", "OYyYYYyO", "OYYYyYYO", "OyYYYYYO", "OYYyYyYO", "OYYYYYyO", "OOOOOOOO"}},
	"stone": {name: "Stone", place: STONE, desc: "Place stone", give: 200, icon: []string{
		"OOOOOOOO", "OGGgGGGO", "OGgGGGgO", "OGGGgGGO", "OgGGGGGO", "OGGgGgGO", "OGGGGGgO", "OOOOOOOO"}},
	"brick": {name: "Brick", place: BRICK, desc: "Place brick (does not fall)", give: 200, icon: []string{
		"OOOOOOOO", "ORRrORRO", "OrrrOrrO", "OOOOOOOO", "ORORRrRO", "OrOrrrrO", "OOOOOOOO", "........"}},
	"water": {name: "Water", place: WATER, desc: "Pour water", give: 200, icon: []string{
		"...OO...", "..OwWO..", "..OwWO..", ".OwWWWO.", "OwWWWWWO", "OWWWWWWO", ".OWWWWO.", "..OOOO.."}},
	"gem": {name: "Energy gem", once: true, desc: "Refills levitation", give: 5, icon: []string{
		"...OO...", "..OLLO..", ".OLBBBO.", "OBLBBBDO", ".OBBBDO.", "..OBDO..", "...OO...", "........"}},
	"bomb": {name: "Bomb", once: true, desc: "Blasts a hole", give: 5, icon: []string{
		".....T..", "....T...", "..OOOh..", ".OKKKKO.", "OKkKKKKO", "OKKKKKKO", ".OKKKKO.", "..OOOO.."}},
}

func initItems() {
	for id, it := range items {
		it.id = id
		img := ebiten.NewImage(8, 8)
		for y, r := range it.icon {
			for x := 0; x < 8; x++ {
				if c, ok := iconCol[r[x]]; ok {
					img.Set(x, y, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
				}
			}
		}
		it.img = img
	}
}

const stackMax = 999

type Stack struct {
	ID string `json:"id"`
	N  int    `json:"n"`
}

// Slots 0-4 are the hotbar, 5-14 the backpack
var (
	inv      = make([]*Stack, 15)
	selected = -1
)

func addItem(id string, n int) int {
	left := n
	if !items[id].tool {
		for _, s := range inv {
			if s != nil && s.ID == id && s.N < stackMax {
				k := min(left, stackMax-s.N)
				s.N += k
				left -= k
				if left == 0 {
					break
				}
			}
		}
	}
	for left > 0 {
		i := -1
		for j, s := range inv {
			if s == nil {
				i = j
				break
			}
		}
		if i < 0 {
			break
		}
		k := min(left, stackMax)
		if items[id].tool {
			k = 1
		}
		inv[i] = &Stack{id, k}
		left -= k
	}
	return n - left
}

func takeSelected(n int) bool {
	if selected < 0 || inv[selected] == nil {
		return false
	}
	inv[selected].N -= n
	if inv[selected].N <= 0 {
		inv[selected] = nil
	}
	return true
}

func countItem(id string) int {
	t := 0
	for _, s := range inv {
		if s != nil && s.ID == id {
			t += s.N
		}
	}
	return t
}

func removeItem(id string, n int) {
	for i := len(inv) - 1; i >= 0 && n > 0; i-- {
		s := inv[i]
		if s == nil || s.ID != id {
			continue
		}
		k := min(n, s.N)
		s.N -= k
		n -= k
		if s.N == 0 {
			inv[i] = nil
		}
	}
}

func starterKit() {
	for i := range inv {
		inv[i] = nil
	}
	addItem("pickaxe", 1)
	addItem("bucket", 1)
	addItem("bomb", 2)
	selected = -1
}

// ---- Crafting ----

type Recipe struct {
	out   string
	n     int
	needs [][2]any // {id, count}
}

var recipes = []Recipe{
	{"brick", 10, [][2]any{{"sand", 10}, {"water", 5}}},
	{"bomb", 1, [][2]any{{"sand", 20}, {"stone", 10}}},
	{"gem", 1, [][2]any{{"stone", 40}, {"water", 20}}},
	{"pickaxe", 1, [][2]any{{"stone", 30}}},
	{"bucket", 1, [][2]any{{"stone", 20}}},
}

func canCraft(r Recipe) bool {
	for _, nd := range r.needs {
		if countItem(nd[0].(string)) < nd[1].(int) {
			return false
		}
	}
	return true
}

func craft(r Recipe) string {
	if !canCraft(r) {
		return ""
	}
	room := false
	for _, s := range inv {
		if s == nil || (s.ID == r.out && !items[r.out].tool && s.N+r.n <= stackMax) {
			room = true
			break
		}
	}
	if !room {
		return "No room in your inventory."
	}
	for _, nd := range r.needs {
		removeItem(nd[0].(string), nd[1].(int))
	}
	addItem(r.out, r.n)
	sfx("give")
	if r.n > 1 {
		return fmt.Sprintf("Crafted %d %s.", r.n, items[r.out].name)
	}
	return "Crafted " + items[r.out].name + "."
}

// ---- Using items: aim is where the pointer is, reach is limited around the mote ----

const reach = 46

var (
	useActive bool
	aimSX     float64
	aimSY     float64
	aimX      int
	aimY      int
	useTick   int
)

func aimCell() bool {
	aimX = int(math.Floor((cam.x + aimSX/zoom) / CELL))
	aimY = int(math.Floor((cam.y + aimSY/zoom) / CELL))
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	return math.Hypot(float64(aimX)-pcx, float64(aimY)-pcy) <= reach
}

func eachInCircle(r int, fn func(x, y, i int)) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r {
				continue
			}
			x, y := aimX+dx, aimY+dy
			if x >= 0 && x < W && y >= 0 && y < H {
				fn(x, y, y*W+x)
			}
		}
	}
}

func useHeld() {
	if !useActive || selected < 0 || inv[selected] == nil {
		return
	}
	it := items[inv[selected].ID]
	if it.once || !aimCell() {
		return
	}
	useTick++
	switch it.id {
	case "pickaxe":
		got := map[uint8]int{}
		eachInCircle(4, func(x, y, i int) {
			t := grid[i]
			if t != SAND && t != STONE && t != BRICK {
				return
			}
			got[t]++
			grid[i] = EMPTY
			if rand.Float64() < 0.12 {
				burst(1, float64(x), float64(y), blockRGB[t], 0.8, 0.8, 18)
			}
		})
		addItem("sand", got[SAND])
		addItem("stone", got[STONE])
		addItem("brick", got[BRICK])
		if len(got) > 0 && useTick%4 == 0 {
			sfx("dig")
		}
	case "bucket":
		n := 0
		eachInCircle(5, func(x, y, i int) {
			if grid[i] == WATER {
				grid[i] = EMPTY
				n++
			}
		})
		if n > 0 {
			addItem("water", n)
			if useTick%5 == 0 {
				sfx("slurp")
			}
		}
	default:
		if it.place == 0 {
			return
		}
		n := 0
		eachInCircle(2, func(x, y, i int) {
			if grid[i] != EMPTY || inPlayer(x, y) || inv[selected] == nil || rand.Float64() > 0.5 {
				return
			}
			grid[i] = it.place
			shade[i] = int8(rand.Intn(21) - 10)
			n++
			takeSelected(1)
		})
		if n > 0 && useTick%4 == 0 {
			sfx("place")
		}
	}
}

func useOnce() {
	if selected < 0 || inv[selected] == nil {
		return
	}
	switch inv[selected].ID {
	case "gem":
		takeSelected(1)
		player.energy = 1
		sfx("gem")
		burst(14, (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL, glowRGB, 1.2, 1.2, 30)
	case "bomb":
		if !aimCell() {
			return
		}
		takeSelected(1)
		explode(aimX, aimY, 12)
	}
}

func explode(cx, cy, r int) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := cx+dx, cy+dy
			if dx*dx+dy*dy > r*r || x < 0 || x >= W || y < 0 || y >= H {
				continue
			}
			i := y*W + x
			if grid[i] != EMPTY && rand.Float64() < 0.15 {
				burst(1, float64(x), float64(y), blockRGB[grid[i]], 2.4, 2.4, 40)
			}
			grid[i] = EMPTY
		}
	}
	burst(30, float64(cx), float64(cy), tipRGB, 3, 3, 26)
	burst(20, float64(cx), float64(cy), [3]int{240, 122, 106}, 2.5, 2.5, 22)
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	dist := math.Hypot(pcx-float64(cx), pcy-float64(cy))
	if rr := float64(r) * 2.5; dist < rr {
		f := (1 - dist/rr) * 9
		player.vy -= f
		d := sign(pcx - float64(cx))
		if d == 0 {
			d = 1
		}
		player.vx += float64(d) * f * 0.6
	}
	sfx("boom")
}

func drawAim(put func(x, y int, c [3]int, s int), ox, oy int) {
	if !useActive || selected < 0 || inv[selected] == nil || inv[selected].ID == "gem" {
		return
	}
	ok := aimCell()
	r := map[string]int{"bomb": 12, "pickaxe": 4, "bucket": 5}[inv[selected].ID]
	if r == 0 {
		r = 2
	}
	c := tipRGB
	if !ok {
		c = [3]int{240, 122, 106}
	}
	for a := 0; a < 40; a++ {
		t := float64(a) / 40 * math.Pi * 2
		put(int(math.Round(float64(aimX)+math.Cos(t)*float64(r+1)))-ox, int(math.Round(float64(aimY)+math.Sin(t)*float64(r+1)))-oy, c, 0)
	}
}
