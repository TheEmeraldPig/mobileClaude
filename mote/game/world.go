package game

// World: seeded, endless left and right, loaded in chunks.
//
// RULES FOR WORLD GENERATION (keep these when adding terrain features):
//  1. Never use math/rand in generation. Use hash01()/noise1() or chunkRng() so the
//     same seed always builds the same world on every device.
//  2. Generation must only depend on the seed and world coordinates, never on screen size,
//     load order or what is already loaded. A chunk must come out identical whenever it is built.
//  3. Add a new feature as a function in genPasses. Each pass fills one chunk; features that
//     cross chunk borders (like caves) must also check the neighbouring chunks' features,
//     as carveCaves does, and only write cells inside the chunk being built.
//  4. Give every pass its own salt number so features don't correlate.
//  5. World height (H) and chunk width (CW) are part of the world format: changing them,
//     or changing a pass, changes every seed's world.
//
// world_test.go pins the output for a few seeds; update it when generation changes on purpose.
//
// The loaded area is a window of NCH chunks held in grid. The window slides as the mote
// travels: chunks leaving are unloaded (saved compressed if they changed) and chunks entering
// are restored from that save or generated fresh from the seed.

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	EMPTY    = 0
	SAND     = 1
	WATER    = 2
	STONE    = 3
	BRICK    = 4
	COAL     = 5
	COPPER   = 6
	IRON     = 7
	GOLD     = 8
	CRYSTAL  = 9
	TORCH    = 10
	DIRT     = 11
	GRASS    = 12
	WOOD     = 13 // tree trunk (you can walk through it)
	LEAVES   = 14
	PLANK    = 15
	GLASS    = 16
	CHEST    = 17
	LAVA     = 18
	SOIL     = 19 // tilled ground for crops
	CROP1    = 20
	CROP2    = 21
	CROP3    = 22 // ripe
	FLOWER   = 23
	BUSH     = 24 // berry bush
	GRAVE    = 25
	CORE     = 26 // the goal at the bottom of the world
	OBSIDIAN = 27
	BEDROCK  = 28
	FURNACE  = 29 // old saves only: furnaces are station objects now
	DOOR     = 30 // closed door (solid)
	DOOROPEN = 31 // open door (walk through it)
)

var blockRGB = map[uint8][3]int{
	SAND:     {226, 194, 107},
	WATER:    {59, 142, 232},
	STONE:    {120, 126, 140},
	BRICK:    {176, 92, 72},
	COAL:     {40, 40, 48},
	COPPER:   {210, 126, 70},
	IRON:     {214, 190, 172},
	GOLD:     {246, 204, 70},
	CRYSTAL:  {120, 232, 255},
	TORCH:    {255, 204, 96},
	DIRT:     {122, 86, 60},
	GRASS:    {88, 168, 74},
	WOOD:     {112, 78, 52},
	LEAVES:   {58, 132, 66},
	PLANK:    {178, 132, 84},
	GLASS:    {190, 226, 240},
	CHEST:    {170, 116, 52},
	LAVA:     {250, 108, 40},
	SOIL:     {84, 58, 40},
	CROP1:    {110, 180, 80},
	CROP2:    {150, 196, 80},
	CROP3:    {232, 200, 90},
	FLOWER:   {236, 120, 170},
	BUSH:     {46, 112, 58},
	GRAVE:    {150, 150, 162},
	CORE:     {255, 120, 220},
	OBSIDIAN: {44, 30, 64},
	BEDROCK:  {26, 24, 32},
	FURNACE:  {96, 90, 104},
	DOOR:     {150, 104, 62},
	DOOROPEN: {150, 104, 62},
}

// Ores are drawn as specks of their colour in stone
func isOre(t uint8) bool { return t >= COAL && t <= GOLD }

func isSolid(t uint8) bool {
	switch t {
	case SAND, STONE, BRICK, CRYSTAL, DIRT, GRASS, PLANK, GLASS, SOIL, GRAVE, CORE, OBSIDIAN, BEDROCK, DOOR:
		return true
	}
	return isOre(t)
}

// Blocks that stop light (glass and plants let it through)
func blocksLight(t uint8) bool { return isSolid(t) && t != GLASS }

func isLiquid(t uint8) bool { return t == WATER || t == LAVA }

// Cells you can build into or that falling sand can replace
func isReplaceable(t uint8) bool { return t == EMPTY || t == WATER || t == FLOWER || t == LEAVES }

// Light given off by a block (0-64)
func blockLight(t uint8) int {
	switch t {
	case TORCH:
		return 60
	case CRYSTAL:
		return 22
	case LAVA:
		return 40
	case CORE:
		return 64
	}
	return 0
}

// The item you get for mining a block
// (BEDROCK, CHEST and CORE are handled specially)
var blockItem = map[uint8]string{SAND: "sand", STONE: "stone", BRICK: "brick", COAL: "coal", COPPER: "copper",
	IRON: "iron", GOLD: "gold", CRYSTAL: "crystal", TORCH: "torch", DIRT: "dirt", GRASS: "dirt", WOOD: "wood",
	LEAVES: "leaves", PLANK: "plank", GLASS: "glass", SOIL: "dirt", CROP1: "seeds", CROP2: "seeds", CROP3: "wheat",
	FLOWER: "seeds", BUSH: "berries", GRAVE: "stone", OBSIDIAN: "obsidian"}

const (
	CELL = 2   // world px per cell
	CW   = 128 // chunk width in cells
	H    = 600 // world height in cells
)

var (
	NCH    int // chunks in the loaded window
	W      int // loaded window width in cells
	grid   []uint8
	shade  []int8
	wall   []uint8 // background walls (a block id, 0 = none)
	paint  []uint8 // paint on blocks (index into paintRGB, 0 = none)
	wpaint []uint8 // paint on walls
	dirty  []bool
	saved  = map[int][]uint16{}
	// the extra layers, saved per chunk only when they have something in them
	savedWalls  = map[int][]uint16{}
	savedPaint  = map[int][]uint16{}
	savedWPaint = map[int][]uint16{}
	baseChunk   int
	seedText    string
	seedN       uint32
	spawnWX     int // world cell x of the spawn point
	spawnY      int // cell y of a bed used as the spawn point (0 = the surface)
)

func initWorld(viewCells int) {
	NCH = int(math.Ceil(float64(viewCells)/CW)) + 4
	W = NCH * CW
	grid = make([]uint8, W*H)
	shade = make([]int8, W*H)
	wall = make([]uint8, W*H)
	paint = make([]uint8, W*H)
	wpaint = make([]uint8, W*H)
	dirty = make([]bool, NCH)
}

// ---- Seeded hashing ----

func seedFrom(text string) uint32 {
	t := strings.TrimSpace(text)
	if len(t) >= 1 && len(t) <= 9 && isDigits(t) {
		n, _ := strconv.ParseUint(t, 10, 32)
		return uint32(n)
	}
	h := uint32(2166136261)
	for _, c := range utf16.Encode([]rune(t)) {
		h = (h ^ uint32(c)) * 16777619
	}
	return h
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func hash32(a, b, c int) uint32 {
	h := seedN ^ (uint32(int32(a)) * 0x27d4eb2d) ^ (uint32(int32(b)) * 0x165667b1) ^ (uint32(int32(c)) * 0x1b873593)
	h = (h ^ (h >> 15)) * 0x85ebca6b
	h = (h ^ (h >> 13)) * 0xc2b2ae35
	return h ^ (h >> 16)
}

func hash01(a, b, c int) float64 { return float64(hash32(a, b, c)) / 4294967296 }

// Smooth 1D value noise in -1..1, endless in both directions
func noise1(x float64, scale float64, salt int) float64 {
	f := x / scale
	i := math.Floor(f)
	t := (1 - math.Cos((f-i)*math.Pi)) / 2
	ii := int(i)
	return (hash01(ii, salt, 7)*2-1)*(1-t) + (hash01(ii+1, salt, 7)*2-1)*t
}

// A repeatable random sequence for one chunk and one feature (mulberry32)
func chunkRng(cx, salt int) func() float64 {
	a := hash32(cx, salt, 1234567)
	return func() float64 {
		a += 0x6D2B79F5
		t := (a ^ (a >> 15)) * (1 | a)
		t = (t + (t^(t>>7))*(61|t)) ^ t
		return float64(t^(t>>14)) / 4294967296
	}
}

// ---- Terrain ----

var waterLevel = int(math.Floor(H * 0.5))

func surfaceAt(x int) int {
	fx := float64(x)
	return int(math.Floor(H*0.45 + noise1(fx, 220, 1)*H*0.2 + noise1(fx, 60, 2)*H*0.05 + noise1(fx, 14, 3)*H*0.012))
}

type genPass func(buf []uint8, cx, x0 int)

var genPasses = []genPass{
	// grass and dirt over stone; sand in deserts and on beaches
	func(buf []uint8, cx, x0 int) {
		for lx := 0; lx < CW; lx++ {
			x := x0 + lx
			s := surfaceAt(x)
			topDepth := 6 + int(math.Floor((noise1(float64(x), 40, 4)+1)*6))
			sandy := isDesert(x) || s >= waterLevel-3
			for y := max(0, s); y < H; y++ {
				switch {
				case y >= s+topDepth:
					buf[y*CW+lx] = STONE
				case sandy:
					buf[y*CW+lx] = SAND
				case y == s:
					buf[y*CW+lx] = GRASS
				default:
					buf[y*CW+lx] = DIRT
				}
			}
		}
	},
	carveCaves,
	carveCaverns,
	// water in the valleys
	func(buf []uint8, cx, x0 int) {
		for lx := 0; lx < CW; lx++ {
			s := surfaceAt(x0 + lx)
			for y := waterLevel; y < s && y < H; y++ {
				if buf[y*CW+lx] == EMPTY {
					buf[y*CW+lx] = WATER
				}
			}
		}
	},
	placeOres,
	placeCrystals,
	fillLava,
	plantTrees,
	plantFlowers,
	placeGraves,
	placeDungeons,
	placeCore,
	// unbreakable floor at the very bottom
	func(buf []uint8, cx, x0 int) {
		for y := H - 3; y < H; y++ {
			for lx := 0; lx < CW; lx++ {
				buf[y*CW+lx] = BEDROCK
			}
		}
	},
}

// Deserts: long sandy stretches with no trees
func isDesert(x int) bool { return noise1(float64(x), 900, 61) > 0.5 }

const lavaLevel = H - 70

// Lava pools the deepest caves
func fillLava(buf []uint8, cx, x0 int) {
	for y := lavaLevel; y < H; y++ {
		for lx := 0; lx < CW; lx++ {
			if buf[y*CW+lx] == EMPTY {
				buf[y*CW+lx] = LAVA
			}
		}
	}
}

// set writes a cell only if it falls inside the chunk being built
func set(buf []uint8, x0, x, y int, t uint8) {
	lx := x - x0
	if lx >= 0 && lx < CW && y >= 0 && y < H {
		buf[y*CW+lx] = t
	}
}

func get(buf []uint8, x0, x, y int) uint8 {
	lx := x - x0
	if lx >= 0 && lx < CW && y >= 0 && y < H {
		return buf[y*CW+lx]
	}
	return EMPTY
}

// Trees: one may grow in each 11-cell-wide slot of grassland. Canopies cross chunk
// borders, so slots from the neighbouring chunks are checked too.
func plantTrees(buf []uint8, cx, x0 int) {
	const slot = 11
	for sl := floorDiv(x0, slot) - 1; sl <= floorDiv(x0+CW, slot)+1; sl++ {
		if hash01(sl, 0, 71) > 0.55 {
			continue
		}
		x := sl*slot + 2 + int(hash01(sl, 1, 71)*float64(slot-4))
		s := surfaceAt(x)
		if isDesert(x) || s >= waterLevel-3 {
			continue
		}
		height := 9 + int(hash01(sl, 2, 71)*8)
		top := s - height
		for y := top; y < s; y++ {
			set(buf, x0, x, y, WOOD)
			if hash01(x, y, 72) < 0.15 && y < s-3 { // a little branch
				set(buf, x0, x+[]int{-1, 1}[int(hash01(x, y, 73)*2)], y, WOOD)
			}
		}
		r := 4 + int(hash01(sl, 3, 71)*2)
		for dy := -r; dy <= r; dy++ {
			for dx := -r - 1; dx <= r+1; dx++ {
				if dx*dx+dy*dy*2 <= r*r+2 && get(buf, x0, x+dx, top+dy) == EMPTY {
					set(buf, x0, x+dx, top+dy, LEAVES)
				}
			}
		}
	}
}

// Flowers and berry bushes on the grass
func plantFlowers(buf []uint8, cx, x0 int) {
	for lx := 0; lx < CW; lx++ {
		x := x0 + lx
		s := surfaceAt(x)
		if s < 1 || buf[s*CW+lx] != GRASS || buf[(s-1)*CW+lx] != EMPTY {
			continue
		}
		switch h := hash01(x, 0, 81); {
		case h < 0.05:
			buf[(s-1)*CW+lx] = FLOWER
		case h < 0.08:
			buf[(s-1)*CW+lx] = BUSH
			if s >= 2 && buf[(s-2)*CW+lx] == EMPTY {
				buf[(s-2)*CW+lx] = BUSH
			}
		}
	}
}

// Graves on the surface, each with a chest buried beneath it
func placeGraves(buf []uint8, cx, x0 int) {
	const slot = 90
	for sl := floorDiv(x0, slot) - 1; sl <= floorDiv(x0+CW, slot)+1; sl++ {
		if hash01(sl, 0, 91) > 0.35 {
			continue
		}
		x := sl*slot + 8 + int(hash01(sl, 1, 91)*float64(slot-16))
		s := surfaceAt(x)
		if s >= waterLevel-3 {
			continue
		}
		// headstone, 3 wide and 5 tall, with a cross cut in
		for y := s - 5; y < s; y++ {
			for dx := -1; dx <= 1; dx++ {
				set(buf, x0, x+dx, y, GRAVE)
			}
		}
		set(buf, x0, x-1, s-5, EMPTY)
		set(buf, x0, x+1, s-5, EMPTY)
		chestAt(buf, x0, x-1, s+4)
	}
}

// A 3x2 chest with its top-left at (x, y)
func chestAt(buf []uint8, x0, x, y int) {
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 3; dx++ {
			set(buf, x0, x+dx, y+dy, CHEST)
		}
	}
}

// Brick dungeon rooms underground, with a chest and a torch-less dark inside
func placeDungeons(buf []uint8, cx, x0 int) {
	for c := cx - 1; c <= cx+1; c++ {
		r := chunkRng(c, 101)
		if r() > 0.4 {
			continue
		}
		w, h := 26+int(r()*14), 16+int(r()*6)
		rx := c*CW + int(r()*float64(CW-w))
		ry := int(H*0.62 + r()*H*0.22)
		for y := ry; y < ry+h; y++ {
			for x := rx; x < rx+w; x++ {
				wall := x < rx+2 || x >= rx+w-2 || y < ry+2 || y >= ry+h-2
				if wall {
					// cracked walls: a few gaps so you can get in
					if hash01(x, y, 102) < 0.08 {
						set(buf, x0, x, y, EMPTY)
					} else {
						set(buf, x0, x, y, BRICK)
					}
				} else {
					set(buf, x0, x, y, EMPTY)
				}
			}
		}
		// doorways on both sides
		for y := ry + h - 2 - 14; y < ry+h-2; y++ {
			set(buf, x0, rx, y, EMPTY)
			set(buf, x0, rx+1, y, EMPTY)
			set(buf, x0, rx+w-1, y, EMPTY)
			set(buf, x0, rx+w-2, y, EMPTY)
		}
		chestAt(buf, x0, rx+w/2-1, ry+h-4)
		if r() < 0.6 {
			chestAt(buf, x0, rx+4, ry+h-4)
		}
	}
}

// The Core: one per world, sealed in obsidian near the bottom, close to where you start
func coreX() int { return int(hash01(0, 0, 111)*float64(CW*6)) - CW*3 }

const coreY = H - 40

func placeCore(buf []uint8, cx, x0 int) {
	x := coreX()
	if x < x0-30 || x > x0+CW+30 {
		return
	}
	// a round chamber with an obsidian shell and a flat floor to stand on
	for dy := -22; dy <= 22; dy++ {
		for dx := -24; dx <= 24; dx++ {
			d := dx*dx + dy*dy
			switch {
			case d <= 17*17 && dy < 9:
				set(buf, x0, x+dx, coreY+dy, EMPTY)
			case d <= 21*21:
				set(buf, x0, x+dx, coreY+dy, OBSIDIAN)
			}
		}
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx*dx+dy*dy <= 5 {
				set(buf, x0, x+dx, coreY+dy, CORE)
			}
		}
	}
}

// Smooth 2D value noise in 0..1
func noise2(x, y, scale float64, salt int) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := math.Floor(fx), math.Floor(fy)
	tx, ty := (1-math.Cos((fx-ix)*math.Pi))/2, (1-math.Cos((fy-iy)*math.Pi))/2
	i, j := int(ix), int(iy)
	a, b := hash01(i, j, salt), hash01(i+1, j, salt)
	c, d := hash01(i, j+1, salt), hash01(i+1, j+1, salt)
	return (a*(1-tx)+b*tx)*(1-ty) + (c*(1-tx)+d*tx)*ty
}

// Winding tunnels that may start in nearby chunks
func carveCaves(buf []uint8, cx, x0 int) {
	for c := cx - 6; c <= cx+6; c++ {
		r := chunkRng(c, 5)
		worms := 1
		if r() < 0.6 {
			worms = 2
		}
		if r() < 0.25 {
			worms++
		}
		for k := 0; k < worms; k++ {
			x := float64(c*CW) + r()*CW
			y := H*0.5 + r()*H*0.45
			a := r() * math.Pi * 2
			length := 120 + r()*260
			base := 5 + r()*4
			for st := 0.0; st < length; st++ {
				a += (r() - 0.5) * 0.45
				x += math.Cos(a) * 2.2
				y += math.Sin(a) * 1.2
				rad := base + r()*4
				if x+rad < float64(x0) || x-rad >= float64(x0+CW) {
					continue
				}
				for dy := -rad; dy <= rad; dy++ {
					for dx := -rad; dx <= rad; dx++ {
						lx := int(math.Floor(x+dx)) - x0
						yy := int(math.Floor(y + dy))
						if dx*dx+dy*dy <= rad*rad && lx >= 0 && lx < CW && yy >= 0 && yy < H {
							buf[yy*CW+lx] = EMPTY
						}
					}
				}
			}
		}
	}
}

// Big open caverns deeper down
func carveCaverns(buf []uint8, cx, x0 int) {
	for lx := 0; lx < CW; lx++ {
		x := float64(x0 + lx)
		top := surfaceAt(x0+lx) + 30
		for y := max(top, 0); y < H-4; y++ {
			n := noise2(x, float64(y), 56, 11)*0.7 + noise2(x, float64(y), 18, 12)*0.3
			fade := math.Min(1, float64(y-top)/40) // caverns open up gradually below the surface
			if n*fade > 0.63 {
				buf[y*CW+lx] = EMPTY
			}
		}
	}
}

// Ore veins in stone, rarer and richer deeper down
func placeOres(buf []uint8, cx, x0 int) {
	type ore struct {
		t      uint8
		depth  int
		scale  float64
		thresh float64
		salt   int
	}
	ores := []ore{{COAL, 4, 6, 0.85, 21}, {COPPER, 25, 5, 0.865, 22}, {IRON, 70, 5, 0.88, 23}, {GOLD, 140, 4, 0.9, 24}}
	for lx := 0; lx < CW; lx++ {
		x := x0 + lx
		s := surfaceAt(x)
		for y := max(s, 0); y < H; y++ {
			i := y*CW + lx
			if buf[i] != STONE {
				continue
			}
			for _, o := range ores {
				if y-s >= o.depth && noise2(float64(x), float64(y)*1.3, o.scale, o.salt) > o.thresh {
					buf[i] = o.t
					break
				}
			}
		}
	}
}

// Glowing crystals growing on cave walls
func placeCrystals(buf []uint8, cx, x0 int) {
	for lx := 1; lx < CW-1; lx++ {
		x := x0 + lx
		s := surfaceAt(x)
		for y := max(s+50, 1); y < H-1; y++ {
			i := y*CW + lx
			if buf[i] != STONE {
				continue
			}
			if buf[i-CW] != EMPTY && buf[i+CW] != EMPTY && buf[i-1] != EMPTY && buf[i+1] != EMPTY {
				continue
			}
			if noise2(float64(x), float64(y), 7, 31) > 0.74 && hash01(x, y, 32) < 0.3 {
				buf[i] = CRYSTAL
			}
		}
	}
}

func genChunk(cx int) []uint8 {
	buf := make([]uint8, CW*H)
	for _, pass := range genPasses {
		pass(buf, cx, cx*CW)
	}
	return buf
}

// ---- Loading and unloading ----

func packChunk(slot int) []uint16 { return packLayer(grid, slot) }

func packLayer(layer []uint8, slot int) []uint16 {
	runs := make([]uint16, 0, 256)
	last, n := -1, 0
	for y := 0; y < H; y++ {
		for lx := 0; lx < CW; lx++ {
			v := int(layer[y*W+slot*CW+lx])
			if v == last && n < 65535 {
				n++
			} else {
				if n > 0 {
					runs = append(runs, uint16(last), uint16(n))
				}
				last, n = v, 1
			}
		}
	}
	return append(runs, uint16(last), uint16(n))
}

func unpackChunk(runs []uint16) []uint8 {
	buf := make([]uint8, CW*H)
	o := 0
	for i := 0; i+1 < len(runs); i += 2 {
		v, n := uint8(runs[i]), int(runs[i+1])
		for j := 0; j < n && o < len(buf); j++ {
			buf[o] = v
			o++
		}
	}
	return buf
}

func loadSlot(slot int) {
	cx := baseChunk + slot
	var buf []uint8
	if r, ok := saved[cx]; ok {
		buf = unpackChunk(r)
	} else {
		buf = genChunk(cx)
	}
	for y := 0; y < H; y++ {
		row := y*W + slot*CW
		copy(grid[row:row+CW], buf[y*CW:y*CW+CW])
		for lx := 0; lx < CW; lx++ {
			shade[row+lx] = int8(int(hash32(cx*CW+lx, y, 99)%21) - 10)
		}
	}
	loadLayer(wall, savedWalls, slot)
	loadLayer(paint, savedPaint, slot)
	loadLayer(wpaint, savedWPaint, slot)
	dirty[slot] = false
	convertOldFurnaces(slot)
}

// Restore one of the extra layers for a chunk (all zero when nothing was saved)
func loadLayer(layer []uint8, from map[int][]uint16, slot int) {
	r, ok := from[baseChunk+slot]
	var buf []uint8
	if ok {
		buf = unpackChunk(r)
	}
	for y := 0; y < H; y++ {
		row := y*W + slot*CW
		if buf == nil {
			clear(layer[row : row+CW])
		} else {
			copy(layer[row:row+CW], buf[y*CW:y*CW+CW])
		}
	}
}

// Save one of the extra layers for a chunk, or forget it when it's empty
func saveLayer(layer []uint8, to map[int][]uint16, slot int) {
	for y := 0; y < H; y++ {
		row := y*W + slot*CW
		for _, v := range layer[row : row+CW] {
			if v != 0 {
				to[baseChunk+slot] = packLayer(layer, slot)
				return
			}
		}
	}
	delete(to, baseChunk+slot)
}

func saveLayers(slot int) {
	saveLayer(wall, savedWalls, slot)
	saveLayer(paint, savedPaint, slot)
	saveLayer(wpaint, savedWPaint, slot)
}

func unloadSlot(slot int) {
	if dirty[slot] {
		saved[baseChunk+slot] = packChunk(slot)
		saveLayers(slot)
	}
}

// Slide the window by dc chunks (positive = to the right), keeping the mote's world position
func shiftWindow(dc int) {
	if dc == 0 {
		return
	}
	if abs(dc) >= NCH {
		for i := 0; i < NCH; i++ {
			unloadSlot(i)
		}
		baseChunk += dc
		for i := 0; i < NCH; i++ {
			loadSlot(i)
		}
	} else {
		keep := NCH - abs(dc)
		cells := dc * CW
		if dc > 0 {
			for i := 0; i < dc; i++ {
				unloadSlot(i)
			}
		} else {
			for i := NCH + dc; i < NCH; i++ {
				unloadSlot(i)
			}
		}
		for y := 0; y < H; y++ {
			r := y * W
			for _, layer := range [][]uint8{grid, wall, paint, wpaint} {
				if dc > 0 {
					copy(layer[r:r+W-cells], layer[r+cells:r+W])
				} else {
					copy(layer[r-cells:r+W], layer[r:r+W+cells])
				}
			}
			if dc > 0 {
				copy(shade[r:r+W-cells], shade[r+cells:r+W])
			} else {
				copy(shade[r-cells:r+W], shade[r:r+W+cells])
			}
		}
		if dc > 0 {
			copy(dirty, dirty[dc:])
		} else {
			copy(dirty[-dc:], dirty[:keep])
		}
		baseChunk += dc
		if dc > 0 {
			for i := keep; i < NCH; i++ {
				loadSlot(i)
			}
		} else {
			for i := 0; i < -dc; i++ {
				loadSlot(i)
			}
		}
	}
	shift := float64(dc * CW * CELL)
	player.x -= shift
	cam.x -= shift
	for i := range parts {
		parts[i].x -= float64(dc * CW)
	}
	for _, e := range ents {
		e.x -= shift
	}
	updatePlayerCells()
}

// Keep the mote in the middle chunks of the window
func followPlayer() {
	slot := int(math.Floor(player.x / CELL / CW))
	mid := NCH / 2
	if abs(slot-mid) >= 2 {
		shiftWindow(slot - mid)
	}
}

// Build a world from a seed and drop the mote on dry ground near world x = 0
func createWorld(text string) {
	seedText = strings.TrimSpace(text)
	if seedText == "" {
		seedText = randomSeed()
	}
	seedN = seedFrom(seedText)
	prefs.Seed = seedText
	savePrefs()
	saved = map[int][]uint16{}
	savedWalls, savedPaint, savedWPaint = map[int][]uint16{}, map[int][]uint16{}, map[int][]uint16{}
	objs = objs[:0]
	parts = parts[:0]
	sx := 0
	for d := 0; d < 20000; d++ {
		if surfaceAt(d) < waterLevel-4 && !isDesert(d) {
			sx = d
			break
		}
		if surfaceAt(-d) < waterLevel-4 && !isDesert(-d) {
			sx = -d
			break
		}
	}
	baseChunk = floorDiv(sx, CW) - NCH/2
	for i := 0; i < NCH; i++ {
		loadSlot(i)
	}
	player.vx, player.vy, player.energy, player.squash = 0, 0, 1, 0
	player.hp = player.maxHP
	// the noise surface ignores lakes and caves, so look at the real blocks for a safe spot
	lx := sx - baseChunk*CW
	for d := 0; d < W/2-40; d++ {
		if safeSpawn(lx + d) {
			lx += d
			break
		}
		if safeSpawn(lx - d) {
			lx -= d
			break
		}
	}
	sx = lx + baseChunk*CW
	spawnWX, spawnY = sx, 0
	ents = ents[:0]
	worldTime = 0.32 // morning
	player.x = float64(lx * CELL)
	player.y = float64((groundTop(lx) - 15) * CELL)
	unstick()
	updatePlayerCells()
}

// The first solid or liquid cell from the top of a loaded column
func groundTop(x int) int {
	for y := 0; y < H; y++ {
		if t := grid[y*W+x]; isSolid(t) || isLiquid(t) {
			return y
		}
	}
	return H - 1
}

// Dry, fairly flat ground with no cave right under it and no lake or lava nearby
func safeSpawn(x int) bool {
	if x < 40 || x >= W-40 {
		return false
	}
	top := groundTop(x)
	for c := x - 30; c <= x+30; c++ {
		t := groundTop(c)
		if isLiquid(grid[t*W+c]) {
			return false
		}
		if c < x-6 || c > x+6 {
			continue
		}
		if abs(t-top) > 4 || abs(t-surfaceAt(c+baseChunk*CW)) > 6 { // not down in a ravine
			return false
		}
		for y := t; y < t+12; y++ {
			if !isSolid(grid[y*W+c]) {
				return false
			}
		}
	}
	return true
}

func cellAt(px, py float64) uint8 {
	cx, cy := int(math.Floor(px/CELL)), int(math.Floor(py/CELL))
	if cx >= 0 && cx < W && cy >= 0 && cy < H {
		return grid[cy*W+cx]
	}
	return EMPTY
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
