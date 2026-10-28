package game

// The world is one island at a time (see isle.go). Its blocks come from the shared
// generator in remote/mmo, so the game and the server see the same terrain.

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

const (
	EMPTY     = 0
	SAND      = 1
	WATER     = 2
	STONE     = 3
	BRICK     = 4
	COAL      = 5
	COPPER    = 6
	IRON      = 7
	GOLD      = 8
	CRYSTAL   = 9
	TORCH     = 10
	DIRT      = 11
	GRASS     = 12
	WOOD      = 13 // tree trunk (you can walk through it)
	LEAVES    = 14
	PLANK     = 15
	GLASS     = 16
	CHEST     = 17 // (Mote ids kept so the numbers match the shared mmo ones)
	LAVA      = 18
	SOIL      = 19 // tilled ground for crops
	CROP1     = 20
	CROP2     = 21
	CROP3     = 22 // ripe
	FLOWER    = 23
	BUSH      = 24 // berry bush
	GRAVE     = 25
	CORE      = 26 // the goal at the bottom of the world
	OBSIDIAN  = 27
	BEDROCK   = 28
	FURNACE   = 29 // old saves only: furnaces are station objects now
	DOOR      = 30 // closed door (solid)
	DOOROPEN  = 31 // open door (walk through it)
	SANDSTONE = 32 // sand packed solid: it doesn't fall
	// mined blocks turn into these (see mmo/mine.go)
	WORNSTONE   = 33
	PACKEDDIRT  = 34
	DULLCRYSTAL = 35
	OLDWOOD     = 36
)

var blockRGB = map[uint8][3]int{
	SAND:        {226, 194, 107},
	WATER:       {59, 142, 232},
	STONE:       {120, 126, 140},
	BRICK:       {176, 92, 72},
	COAL:        {40, 40, 48},
	COPPER:      {210, 126, 70},
	IRON:        {214, 190, 172},
	GOLD:        {246, 204, 70},
	CRYSTAL:     {120, 232, 255},
	TORCH:       {255, 204, 96},
	DIRT:        {122, 86, 60},
	GRASS:       {88, 168, 74},
	WOOD:        {112, 78, 52},
	LEAVES:      {58, 132, 66},
	PLANK:       {178, 132, 84},
	GLASS:       {190, 226, 240},
	CHEST:       {170, 116, 52},
	LAVA:        {250, 108, 40},
	SOIL:        {84, 58, 40},
	CROP1:       {110, 180, 80},
	CROP2:       {150, 196, 80},
	CROP3:       {232, 200, 90},
	FLOWER:      {236, 120, 170},
	BUSH:        {46, 112, 58},
	GRAVE:       {150, 150, 162},
	CORE:        {255, 120, 220},
	OBSIDIAN:    {44, 30, 64},
	BEDROCK:     {26, 24, 32},
	FURNACE:     {96, 90, 104},
	DOOR:        {150, 104, 62},
	SANDSTONE:   {204, 168, 96},
	DOOROPEN:    {150, 104, 62},
	WORNSTONE:   {92, 94, 108},
	PACKEDDIRT:  {90, 66, 50},
	DULLCRYSTAL: {84, 118, 136},
	OLDWOOD:     {82, 64, 52},
}

// Ores are drawn as specks of their colour in stone
func isOre(t uint8) bool { return t >= COAL && t <= GOLD }

func isSolid(t uint8) bool {
	switch t {
	case SAND, STONE, BRICK, CRYSTAL, DIRT, GRASS, PLANK, GLASS, SOIL, GRAVE, CORE, OBSIDIAN, BEDROCK, DOOR, SANDSTONE,
		WORNSTONE, PACKEDDIRT, DULLCRYSTAL:
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

const (
	CELL = mmo.Cell // world px per cell
	H    = mmo.GH   // island height in cells
	topH = 600      // the sky's colours and the distant hills are laid out for this height
)

var (
	W         = mmo.GW // island width in cells
	grid      []uint8
	shade     []int8
	baseChunk = 0 // left over from Mote's sliding window: the island always starts at 0
	CW        = 128
	seedN     uint32 // the island's seed, for block textures
)

func initWorld() {
	grid = make([]uint8, W*H)
	shade = make([]int8, W*H)
}

// ---- Seeded hashing (block textures) ----

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

// Smooth 2D value noise in 0..1
func noise2(x, y, scale float64, salt int) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-ix, fy-iy
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty) // smooth, like a cosine ease but cheaper
	i, j := int(ix), int(iy)
	a, b := hash01(i, j, salt), hash01(i+1, j, salt)
	c, d := hash01(i, j+1, salt), hash01(i+1, j+1, salt)
	return (a*(1-tx)+b*tx)*(1-ty) + (c*(1-tx)+d*tx)*ty
}

// A block's texture: soft clumps of light and dark with a little fine grain on top,
// so surfaces look painted rather than noisy
func texShade(x, y int) int8 {
	clump := noise2(float64(x), float64(y), 3.2, 98)*16 - 8
	grain := float64(hash32(x, y, 99)%7) - 3
	return int8(clump + grain)
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
