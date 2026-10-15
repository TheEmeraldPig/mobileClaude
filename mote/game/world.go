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
// The hashing and generation match the web version bit for bit, so a seed gives the same
// world in both.
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
	EMPTY = 0
	SAND  = 1
	WATER = 2
	STONE = 3
	BRICK = 4
)

var blockRGB = map[uint8][3]int{
	SAND:  {226, 194, 107},
	WATER: {59, 142, 232},
	STONE: {120, 126, 140},
	BRICK: {176, 92, 72},
}

func isSolid(t uint8) bool { return t == SAND || t == STONE || t == BRICK }

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
}

// Worms that may start in nearby chunks
func carveCaves(buf []uint8, cx, x0 int) {
	for c := cx - 5; c <= cx+5; c++ {
		r := chunkRng(c, 5)
		worms := 1
		if r() < 0.45 {
			worms = 2
		}
		for k := 0; k < worms; k++ {
			x := float64(c*CW) + r()*CW
			y := H*0.55 + r()*H*0.4
			a := r() * math.Pi * 2
			length := 80 + r()*200
			for st := 0.0; st < length; st++ {
				a += (r() - 0.5) * 0.5
				x += math.Cos(a) * 2
				y += math.Sin(a) * 1.2
				rad := 4 + r()*5
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
