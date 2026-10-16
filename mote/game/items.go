package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strings"

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
	'c': {210, 126, 70}, 'C': {150, 84, 46}, 'i': {214, 190, 172}, 'I': {150, 134, 124},
	'u': {246, 204, 70}, 'U': {190, 150, 40}, 'x': {120, 232, 255}, 'X': {210, 250, 255}, 'z': {60, 150, 200},
	'f': {255, 150, 60},
}

type Item struct {
	id, name, desc string
	tool, once     bool
	place          uint8
	icon           []string
	img            *ebiten.Image
	reach          int // tool radius in cells
	dmg            int // damage to creatures
}

// A rough lump of ore in two shades
func oreIcon(a, d byte) []string {
	rows := []string{"........", "..OOOO..", ".OaAaaO.", "OaaAaAaO", "OAaaaAaO", ".OaAaaO.", "..OOOO..", "........"}
	for i, r := range rows {
		rows[i] = strings.NewReplacer("a", string(a), "A", string(d)).Replace(r)
	}
	return rows
}

var itemOrder = []string{"pickaxe", "ironpick", "bucket", "coppersword", "ironsword", "lantern", "torch", "sand", "stone",
	"brick", "water", "coal", "copper", "iron", "gold", "crystal", "gem", "bomb"}

var items = map[string]*Item{
	"pickaxe": {name: "Pickaxe", tool: true, desc: "Digs sand, stone and ore", reach: 4, dmg: 1, icon: []string{
		".SSSSSs.", "S..hH..s", "...hH...", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"ironpick": {name: "Iron pickaxe", tool: true, desc: "Digs a much bigger area", reach: 6, dmg: 2, icon: []string{
		".iiiiiI.", "i..hH..I", "i..hH..I", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"bucket": {name: "Bucket", tool: true, desc: "Scoops up water", reach: 5, icon: []string{
		"..O..O..", ".O....O.", "OMMMMMMO", "OMwWWWmO", ".MWWWWm.", ".MMMMmm.", "..Mmmm..", "........"}},
	"coppersword": {name: "Copper sword", tool: true, desc: "Hits creatures", reach: 7, dmg: 2, icon: []string{
		"......Oc", ".....Occ", "....Occ.", "...Occ..", "h.Occ...", ".hcc....", ".hh.....", "h..h...."}},
	"ironsword": {name: "Iron sword", tool: true, desc: "Hits creatures hard", reach: 8, dmg: 4, icon: []string{
		"......Oi", ".....Oii", "....OiI.", "...OiI..", "T.OiI...", ".TiI....", ".hT.....", "h..T...."}},
	"lantern": {name: "Lantern", tool: true, desc: "Lights up more around you while you carry it", icon: []string{
		"...OO...", "..O..O..", ".OOOOOO.", ".OxXxxO.", ".OXxxxO.", ".OxxxzO.", ".OOOOOO.", "........"}},
	"torch": {name: "Torch", once: true, place: TORCH, desc: "Place a light that never goes out", icon: []string{
		"...f....", "..fTf...", "..TTT...", "...h....", "...h....", "...h....", "...h....", "........"}},
	"sand": {name: "Sand", place: SAND, desc: "Falls when nothing holds it up", icon: []string{
		"OOOOOOOO", "OYYyYYYO", "OYyYYYyO", "OYYYyYYO", "OyYYYYYO", "OYYyYyYO", "OYYYYYyO", "OOOOOOOO"}},
	"stone": {name: "Stone", place: STONE, desc: "A solid building block", icon: []string{
		"OOOOOOOO", "OGGgGGGO", "OGgGGGgO", "OGGGgGGO", "OgGGGGGO", "OGGgGgGO", "OGGGGGgO", "OOOOOOOO"}},
	"brick": {name: "Brick", place: BRICK, desc: "A sturdy building block", icon: []string{
		"OOOOOOOO", "ORRrORRO", "OrrrOrrO", "OOOOOOOO", "ORORRrRO", "OrOrrrrO", "OOOOOOOO", "........"}},
	"water": {name: "Water", place: WATER, desc: "Pour water", icon: []string{
		"...OO...", "..OwWO..", "..OwWO..", ".OwWWWO.", "OwWWWWWO", "OWWWWWWO", ".OWWWWO.", "..OOOO.."}},
	"coal":    {name: "Coal", place: COAL, desc: "Burns well. Used for torches", icon: oreIcon('K', 'k')},
	"copper":  {name: "Copper", place: COPPER, desc: "Soft orange metal", icon: oreIcon('c', 'C')},
	"iron":    {name: "Iron", place: IRON, desc: "Strong metal found deeper down", icon: oreIcon('i', 'I')},
	"gold":    {name: "Gold", place: GOLD, desc: "Rare, found very deep", icon: oreIcon('u', 'U')},
	"crystal": {name: "Crystal", place: CRYSTAL, desc: "Glows softly. Grows in caves", icon: oreIcon('x', 'z')},
	"gem": {name: "Energy gem", once: true, desc: "Refills levitation", icon: []string{
		"...OO...", "..OLLO..", ".OLBBBO.", "OBLBBBDO", ".OBBBDO.", "..OBDO..", "...OO...", "........"}},
	"bomb": {name: "Bomb", once: true, desc: "Blasts a hole", icon: []string{
		".....T..", "....T...", "..OOOh..", ".OKKKKO.", "OKkKKKKO", "OKKKKKKO", ".OKKKKO.", "..OOOO.."}},
}

func initItems() {
	for id, it := range items {
		it.id = id
		img := ebiten.NewImage(8, 8)
		for y, r := range it.icon {
			for x := 0; x < 8 && x < len(r); x++ {
				if c, ok := iconCol[r[x]]; ok {
					img.Set(x, y, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
				}
			}
		}
		it.img = img
	}
}

const (
	stackMax = 999
	hotbarN  = 8
	invSize  = hotbarN + 32 // hotbar plus a 4x8 backpack
)

type Stack struct {
	ID string `json:"id"`
	N  int    `json:"n"`
}

var (
	inv      = make([]*Stack, invSize)
	selected = -1
)

func addItem(id string, n int) int {
	if n <= 0 || items[id] == nil {
		return 0
	}
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
	addItem("torch", 8)
	addItem("bomb", 2)
	selected = -1
}

// Drop n items from a slot into the world in front of the mote
func dropSlot(i, n int) {
	s := inv[i]
	if s == nil {
		return
	}
	n = min(n, s.N)
	throwItem(s.ID, n)
	s.N -= n
	if s.N <= 0 {
		inv[i] = nil
	}
}

// Move a slot between the hotbar and the backpack (merging stacks where possible)
func quickMove(i int) {
	s := inv[i]
	if s == nil {
		return
	}
	from, to := hotbarN, invSize
	if i >= hotbarN {
		from, to = 0, hotbarN
	}
	if !items[s.ID].tool {
		for j := from; j < to && s.N > 0; j++ {
			if t := inv[j]; t != nil && t.ID == s.ID && t.N < stackMax {
				k := min(s.N, stackMax-t.N)
				t.N += k
				s.N -= k
			}
		}
	}
	if s.N == 0 {
		inv[i] = nil
		return
	}
	for j := from; j < to; j++ {
		if inv[j] == nil {
			inv[j], inv[i] = s, nil
			return
		}
	}
}

// Split half of a stack into an empty slot
func splitStack(i int) {
	s := inv[i]
	if s == nil || s.N < 2 {
		return
	}
	for j := hotbarN; j < invSize; j++ {
		if inv[j] == nil {
			half := s.N / 2
			inv[j] = &Stack{s.ID, half}
			s.N -= half
			return
		}
	}
}

// Merge stacks and sort the backpack in item order
func sortBackpack() {
	totals := map[string]int{}
	var tools []string
	for i := hotbarN; i < invSize; i++ {
		if s := inv[i]; s != nil {
			if items[s.ID].tool {
				tools = append(tools, s.ID)
			} else {
				totals[s.ID] += s.N
			}
			inv[i] = nil
		}
	}
	rank := map[string]int{}
	for k, id := range itemOrder {
		rank[id] = k
	}
	sort.Slice(tools, func(a, b int) bool { return rank[tools[a]] < rank[tools[b]] })
	j := hotbarN
	for _, id := range tools {
		inv[j] = &Stack{id, 1}
		j++
	}
	for _, id := range itemOrder {
		for n := totals[id]; n > 0 && j < invSize; j++ {
			k := min(n, stackMax)
			inv[j] = &Stack{id, k}
			n -= k
		}
	}
}

// ---- Crafting ----

type Recipe struct {
	out   string
	n     int
	needs [][2]any // {id, count}
}

var recipes = []Recipe{
	{"torch", 4, [][2]any{{"coal", 1}, {"stone", 1}}},
	{"brick", 10, [][2]any{{"sand", 10}, {"water", 5}}},
	{"bomb", 1, [][2]any{{"sand", 20}, {"coal", 2}}},
	{"coppersword", 1, [][2]any{{"copper", 12}, {"stone", 4}}},
	{"ironsword", 1, [][2]any{{"iron", 12}, {"coal", 4}}},
	{"ironpick", 1, [][2]any{{"iron", 15}, {"stone", 10}}},
	{"lantern", 1, [][2]any{{"crystal", 6}, {"copper", 6}}},
	{"gem", 1, [][2]any{{"crystal", 2}, {"gold", 1}}},
	{"pickaxe", 1, [][2]any{{"stone", 30}}},
	{"bucket", 1, [][2]any{{"copper", 8}}},
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
	id := inv[selected].ID
	it := items[id]
	if it.once || !aimCell() {
		return
	}
	useTick++
	switch {
	case id == "pickaxe" || id == "ironpick":
		got := map[uint8]int{}
		eachInCircle(it.reach, func(x, y, i int) {
			t := grid[i]
			if blockItem[t] == "" {
				return
			}
			got[t]++
			grid[i] = EMPTY
			if rand.Float64() < 0.12 {
				burst(1, float64(x), float64(y), blockRGB[t], 0.8, 0.8, 18)
			}
		})
		for t, n := range got {
			addItem(blockItem[t], n)
		}
		if len(got) > 0 && useTick%4 == 0 {
			sfx("dig")
		}
		if useTick%20 == 1 {
			hitCreatures(aimX, aimY, it.reach, it.dmg)
		}
	case id == "bucket":
		n := 0
		eachInCircle(it.reach, func(x, y, i int) {
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
	case it.dmg > 0: // swords: swing every 18 frames while held
		if useTick%18 == 1 {
			swing(it)
		}
	case it.place != 0:
		// building is as fast as mining: fill every empty cell in the circle at once
		n := 0
		eachInCircle(4, func(x, y, i int) {
			if (grid[i] != EMPTY && !(grid[i] == WATER && it.place != WATER)) || inv[selected] == nil {
				return
			}
			grid[i] = it.place
			shade[i] = int8(rand.Intn(21) - 10)
			n++
			takeSelected(1)
		})
		if n > 0 && useTick%4 == 1 {
			sfx("place")
		}
	}
}

func swing(it *Item) {
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	// swing toward the aim, but never further than the sword reaches
	dx, dy := float64(aimX)-pcx, float64(aimY)-pcy
	d := math.Hypot(dx, dy)
	r := float64(it.reach + 6)
	if d > r {
		dx, dy = dx/d*r, dy/d*r
	}
	if dx != 0 {
		player.face = sign(dx)
	}
	cx, cy := int(pcx+dx), int(pcy+dy)
	for k := 0; k < 10; k++ {
		a := math.Atan2(dy, dx) + (float64(k)-5)*0.15
		burst(1, pcx+math.Cos(a)*r*0.8, pcy+math.Sin(a)*r*0.8, [3]int{230, 230, 250}, 0.2, 0.2, 8)
	}
	sfx("swing")
	hitCreatures(cx, cy, it.reach, it.dmg)
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
	case "torch":
		if !aimCell() || aimX < 0 || aimX >= W || aimY < 0 || aimY >= H {
			return
		}
		i := aimY*W + aimX
		if grid[i] != EMPTY {
			return
		}
		grid[i] = TORCH
		shade[i] = 0
		takeSelected(1)
		sfx("place")
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
	hitCreatures(cx, cy, r, 6)
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
	if !useActive || selected < 0 || inv[selected] == nil {
		return
	}
	id := inv[selected].ID
	if id == "gem" || id == "lantern" {
		return
	}
	ok := aimCell()
	r := items[id].reach
	switch {
	case id == "bomb":
		r = 12
	case id == "torch":
		r = 1
	case items[id].place != 0:
		r = 4
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
