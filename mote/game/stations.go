package game

import (
	"math"
	"sort"
)

// Workstations: the workbench, furnace and anvil. They are background objects, not
// blocks: the mote walks past them, they have a dark outline so they stand out, and
// tapping or right-clicking one opens its crafting menu. Mining one gives back exactly
// one item (holding the pickaxe on it for a moment).
//
// Objects are kept in world cells (wx), so they stay put when the loaded window slides.

type stationDef struct {
	name   string
	sprite []string // one char per cell; '.' is empty
	light  int
}

var stationCol = map[byte][3]int{
	'O': {18, 14, 28}, // outline
	'P': {178, 132, 84}, 'p': {130, 92, 56}, 'h': {96, 66, 40},
	'G': {120, 126, 140}, 'g': {84, 88, 102}, 'k': {60, 62, 74},
	'f': {255, 150, 60}, 'T': {255, 222, 120}, 'r': {200, 70, 40},
	'I': {150, 156, 172}, 'i': {200, 206, 220}, 'm': {96, 100, 116},
	'c': {210, 126, 70}, 'w': {226, 230, 236},
}

var stations = map[string]*stationDef{
	"workbench": {name: "Workbench", sprite: []string{
		"OOOOOOOOOOOO",
		"OPPPPPPPPPPO",
		"OppppppppppO",
		"OhOOOOOOOOhO",
		"OhO.....cOhO",
		"OhO....OOOhO",
		"OOO......OOO",
	}},
	"furnace": {name: "Furnace", light: 34, sprite: []string{
		"..OOOOOO..",
		".OGGGgGGO.",
		"OGgGGGGgGO",
		"OGOOOOOOgO",
		"OgOrfTfOGO",
		"OGOfTTrOgO",
		"OGOOOOOOGO",
		"OGgGGgGGgO",
		"OOOOOOOOOO",
	}},
	"anvil": {name: "Anvil", sprite: []string{
		"OOOOOOOOOOO",
		"OiiiiiiiiIO",
		".OIIIIIIIO.",
		"...OmmmO...",
		"..OmmmmmO..",
		".OOOOOOOOO.",
	}},
}

// Station order for menus and the item list
var stationOrder = []string{"workbench", "furnace", "anvil"}

type Obj struct {
	kind  string
	wx, y int // top-left cell; wx in world cells
	flip  bool
	hits  int // frames being mined
}

type SavedObj struct {
	Kind string `json:"kind"`
	WX   int    `json:"wx"`
	Y    int    `json:"y"`
	Flip bool   `json:"flip,omitempty"`
}

var objs []*Obj

func saveObjs() []SavedObj {
	out := make([]SavedObj, 0, len(objs))
	for _, o := range objs {
		out = append(out, SavedObj{o.kind, o.wx, o.y, o.flip})
	}
	return out
}

func loadObjs(s []SavedObj) {
	objs = objs[:0]
	for _, o := range s {
		if stations[o.Kind] != nil {
			objs = append(objs, &Obj{kind: o.Kind, wx: o.WX, y: o.Y, flip: o.Flip})
		}
	}
}

func (o *Obj) size() (int, int) {
	sp := stations[o.kind].sprite
	return len(sp[0]), len(sp)
}

// Local x of the object's left edge
func (o *Obj) lx() int { return o.wx - baseChunk*CW }

// Sprite cell at (dx, dy) inside the object, with flipping
func (o *Obj) cell(dx, dy int) byte {
	sp := stations[o.kind].sprite
	w := len(sp[0])
	if o.flip {
		dx = w - 1 - dx
	}
	return sp[dy][dx]
}

// The object covering local cell (x, y), counting only its drawn cells
func objAt(x, y int) *Obj {
	for _, o := range objs {
		w, h := o.size()
		dx, dy := x-o.lx(), y-o.y
		if dx >= 0 && dy >= 0 && dx < w && dy < h && o.cell(dx, dy) != '.' {
			return o
		}
	}
	return nil
}

// Is the object's whole box free and standing on ground?
func stationFits(kind string, lx, ty int, flip bool) bool {
	sp := stations[kind].sprite
	w, h := len(sp[0]), len(sp)
	if lx < 1 || lx+w >= W-1 || ty < 1 || ty+h >= H {
		return false
	}
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	if math.Hypot(float64(lx+w/2)-pcx, float64(ty+h/2)-pcy) > reach+6 {
		return false
	}
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			t := grid[(ty+dy)*W+lx+dx]
			if isSolid(t) || isLiquid(t) || t == CHEST {
				return false
			}
		}
	}
	// no overlap with other objects
	for _, o := range objs {
		ow, oh := o.size()
		if lx < o.lx()+ow && lx+w > o.lx() && ty < o.y+oh && ty+h > o.y {
			return false
		}
	}
	// most of the bottom must rest on something solid
	support := 0
	for dx := 0; dx < w; dx++ {
		if isSolid(grid[(ty+h)*W+lx+dx]) {
			support++
		}
	}
	return support*5 >= w*2 // at least 40% of the bottom rests on something
}

// Where a held station would go: centred on the aim, resting its bottom on the aimed cell
func stationAnchor(kind string, ax, ay int) (int, int) {
	sp := stations[kind].sprite
	w, h := len(sp[0]), len(sp)
	return ax - w/2, ay - h + 1
}

// Find the nearest spot around (lx, ty) where the station fits and rests on the ground
func settleStation(kind string, lx, ty int, flip bool) (int, int, bool) {
	best, bx, by := 1e9, lx, ty
	for dy := -16; dy <= 16; dy++ {
		for dx := -10; dx <= 10; dx++ {
			d := math.Abs(float64(dy)) + math.Abs(float64(dx))*1.5
			if d < best && stationFits(kind, lx+dx, ty+dy, flip) {
				best, bx, by = d, lx+dx, ty+dy
			}
		}
	}
	return bx, by, best < 1e9
}

func placeStation(it *Item, ax, ay int) bool {
	lx, ty := stationAnchor(it.station, ax, ay)
	lx, ty, ok := settleStation(it.station, lx, ty, buildFlip)
	if !ok {
		toast("Not enough room - it needs flat ground")
		return false
	}
	objs = append(objs, &Obj{kind: it.station, wx: lx + baseChunk*CW, y: ty, flip: buildFlip})
	sort.SliceStable(objs, func(a, b int) bool { return objs[a].wx < objs[b].wx })
	takeSelected(1)
	burst(14, float64(lx+3), float64(ty+3), tipRGB, 1.2, 1.2, 24)
	sfx("build")
	toast(stations[it.station].name + " placed. Tap it to craft.")
	return true
}

func removeObj(o *Obj, give bool) {
	out := objs[:0]
	for _, p := range objs {
		if p != o {
			out = append(out, p)
		}
	}
	objs = out
	w, h := o.size()
	if give {
		dropItem(o.kind, 1, float64((o.lx()+w/2)*CELL), float64((o.y+h/2)*CELL), 0)
	}
	burst(16, float64(o.lx()+w/2), float64(o.y+h/2), stationCol['G'], 1.4, 1.4, 24)
	sfx("dig")
}

// Mining stations: a pickaxe held on one for a short moment knocks it loose
func mineObjects(r int) {
	for _, o := range append([]*Obj(nil), objs...) {
		w, h := o.size()
		cx, cy := o.lx()+w/2, o.y+h/2
		if math.Hypot(float64(cx-aimX), float64(cy-aimY)) > float64(r)+float64(max(w, h))/2 {
			continue
		}
		o.hits++
		if o.hits%6 == 0 {
			burst(2, float64(cx), float64(cy), stationCol['g'], 0.8, 0.8, 14)
		}
		if o.hits > 28 {
			removeObj(o, true)
		}
	}
}

// Stations whose ground is gone pop out as items
func checkObjects() {
	if frame%20 != 0 {
		return
	}
	for _, o := range append([]*Obj(nil), objs...) {
		lx := o.lx()
		w, h := o.size()
		if lx < 0 || lx+w >= W {
			continue
		}
		if o.y+h >= H || !stationFitsGround(lx, o.y+h, w) {
			removeObj(o, true)
		}
	}
	// stop mining progress when the pickaxe moves away
	if !useActive {
		for _, o := range objs {
			o.hits = 0
		}
	}
}

func stationFitsGround(lx, y, w int) bool {
	for dx := 0; dx < w; dx++ {
		if isSolid(grid[y*W+lx+dx]) {
			return true
		}
	}
	return false
}

// Is the mote close enough to use this station?
func objInReach(o *Obj) bool {
	w, h := o.size()
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	return math.Hypot(float64(o.lx()+w/2)-pcx, float64(o.y+h/2)-pcy) <= reach+8
}

// Light from stations (the furnace glows)
func objLights(seed func(lx, ly, lvl int)) {
	for _, o := range objs {
		if l := stations[o.kind].light; l > 0 {
			w, h := o.size()
			seed(o.lx()+w/2, o.y+h/2+1, l)
		}
	}
}

// Draw stations behind blocks; only cells that aren't covered by a solid block show
func drawObjects(lit func(x, y int, c [3]int, s int), ox, oy int) {
	for _, o := range objs {
		w, h := o.size()
		lx := o.lx()
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				ch := o.cell(dx, dy)
				if ch == '.' {
					continue
				}
				x, y := lx+dx, o.y+dy
				if x >= 0 && x < W && y >= 0 && y < H && isSolid(grid[y*W+x]) {
					continue
				}
				c := stationCol[ch]
				if o.kind == "furnace" && (ch == 'f' || ch == 'T' || ch == 'r') {
					f := 0.8 + 0.2*math.Sin(float64(frame)*0.25+float64(dx*3+dy))
					c = shadeRGB(c, f)
					lit(x-ox, y-oy, c, 40) // the fire is bright even in the dark
					continue
				}
				if o.hits > 0 && (dx+dy+frame/3)%3 == 0 {
					c = mix(c, [3]int{255, 255, 255}, 0.3)
				}
				lit(x-ox, y-oy, c, 0)
			}
		}
	}
}

// Ghost of the held station where it would go
func drawStationHologram(blend func(x, y int, c [3]int, a float64), ox, oy int) {
	if cur != scPlaying || selected < 0 || inv[selected] == nil {
		return
	}
	it := items[inv[selected].ID]
	if it.station == "" {
		return
	}
	ax, ay, ok := holoAim()
	if !ok {
		return
	}
	lx, ty := stationAnchor(it.station, ax, ay)
	lx, ty, fits := settleStation(it.station, lx, ty, buildFlip)
	tint := [3]int{120, 255, 160}
	if !fits {
		tint = [3]int{255, 110, 110}
	}
	pulse := 0.5 + 0.15*math.Sin(float64(frame)*0.15)
	tmp := &Obj{kind: it.station, flip: buildFlip}
	sp := stations[it.station].sprite
	for dy := range sp {
		for dx := range sp[dy] {
			ch := tmp.cell(dx, dy)
			if ch == '.' {
				continue
			}
			blend(lx+dx-ox, ty+dy-oy, mix(stationCol[ch], tint, 0.5), pulse)
		}
	}
}

// Older saves had furnaces as 4x3 blocks: turn each one back into a furnace item
func convertOldFurnaces(slot int) {
	x0 := slot * CW
	for y := 0; y < H; y++ {
		for x := x0; x < x0+CW; x++ {
			if grid[y*W+x] != FURNACE {
				continue
			}
			// clear the whole group
			stack := [][2]int{{x, y}}
			for len(stack) > 0 {
				c := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if c[0] < 0 || c[0] >= W || c[1] < 0 || c[1] >= H || grid[c[1]*W+c[0]] != FURNACE {
					continue
				}
				grid[c[1]*W+c[0]] = EMPTY
				stack = append(stack, [2]int{c[0] + 1, c[1]}, [2]int{c[0] - 1, c[1]}, [2]int{c[0], c[1] + 1}, [2]int{c[0], c[1] - 1})
			}
			dropItem("furnace", 1, float64(x*CELL), float64(y*CELL), 0)
			dirty[slot] = true
		}
	}
}
