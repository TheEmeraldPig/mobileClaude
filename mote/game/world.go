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
	EMPTY   = 0
	SAND    = 1
	WATER   = 2
	STONE   = 3
	BRICK   = 4
	COAL    = 5
	COPPER  = 6
	IRON    = 7
	GOLD    = 8
	CRYSTAL = 9
	TORCH   = 10
)

var blockRGB = map[uint8][3]int{
	SAND:    {226, 194, 107},
	WATER:   {59, 142, 232},
	STONE:   {120, 126, 140},
	BRICK:   {176, 92, 72},
	COAL:    {40, 40, 48},
	COPPER:  {210, 126, 70},
	IRON:    {214, 190, 172},
	GOLD:    {246, 204, 70},
	CRYSTAL: {120, 232, 255},
	TORCH:   {255, 204, 96},
}

// Ores are drawn as specks of their colour in stone
func isOre(t uint8) bool { return t >= COAL && t <= GOLD }

func isSolid(t uint8) bool { return t == SAND || t == STONE || t == BRICK || isOre(t) || t == CRYSTAL }

// Light given off by a block (0-64)
func blockLight(t uint8) int {
	switch t {
	case TORCH:
		return 60
	case CRYSTAL:
		return 22
	}
	return 0
}

// The item you get for mining a block
var blockItem = map[uint8]string{SAND: "sand", STONE: "stone", BRICK: "brick", COAL: "coal", COPPER: "copper",
	IRON: "iron", GOLD: "gold", CRYSTAL: "crystal", TORCH: "torch"}

const (
	CELL = 2   // world px per cell
	CW   = 128 // chunk width in cells
	H    = 600 // world height in cells
)

var (
	NCH       int // chunks in the loaded window
	W         int // loaded window width in cells
	grid      []uint8
	shade     []int8
	dirty     []bool
	saved     = map[int][]uint16{}
	baseChunk int
	seedText  string
	seedN     uint32
	spawnWX   int // world cell x of the spawn point
)

func initWorld(viewCells int) {
	NCH = int(math.Ceil(float64(viewCells)/CW)) + 4
	W = NCH * CW
	grid = make([]uint8, W*H)
	shade = make([]int8, W*H)
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
	// sand over stone
	func(buf []uint8, cx, x0 int) {
		for lx := 0; lx < CW; lx++ {
			x := x0 + lx
			s := surfaceAt(x)
			sandDepth := 6 + int(math.Floor((noise1(float64(x), 40, 4)+1)*6))
			for y := max(0, s); y < H; y++ {
				if y < s+sandDepth {
					buf[y*CW+lx] = SAND
				} else {
					buf[y*CW+lx] = STONE
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

func packChunk(slot int) []uint16 {
	runs := make([]uint16, 0, 256)
	last, n := -1, 0
	for y := 0; y < H; y++ {
		for lx := 0; lx < CW; lx++ {
			v := int(grid[y*W+slot*CW+lx])
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
	dirty[slot] = false
}

func unloadSlot(slot int) {
	if dirty[slot] {
		saved[baseChunk+slot] = packChunk(slot)
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
			if dc > 0 {
				copy(grid[r:r+W-cells], grid[r+cells:r+W])
				copy(shade[r:r+W-cells], shade[r+cells:r+W])
			} else {
				copy(grid[r-cells:r+W], grid[r:r+W+cells])
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
	parts = parts[:0]
	sx := 0
	for d := 0; d < 20000; d++ {
		if surfaceAt(d) < waterLevel {
			sx = d
			break
		}
		if surfaceAt(-d) < waterLevel {
			sx = -d
			break
		}
	}
	baseChunk = floorDiv(sx, CW) - NCH/2
	for i := 0; i < NCH; i++ {
		loadSlot(i)
	}
	player.vx, player.vy, player.energy, player.squash = 0, 0, 1, 0
	player.hp = maxHP
	spawnWX = sx
	ents = ents[:0]
	worldTime = 0.32 // morning
	player.x = float64((sx - baseChunk*CW) * CELL)
	player.y = float64((surfaceAt(sx) - 15) * CELL)
	unstick()
	updatePlayerCells()
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
