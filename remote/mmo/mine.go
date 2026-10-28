package mmo

import "math"

// Mining. Most islands can't be mined at all. On an island that can, a mined block
// isn't removed: it turns into a worn block that can't be mined again, and the player
// gets its resource. Islands stay the same shape for everyone, and worn blocks grow
// back after the island's Regrow time.
//
// Each island names a rule set (IsleDef.Mine), so different kinds of island can give
// different things, take longer, or turn blocks into something else.

// What mining a block gives
type Yield struct {
	Item    string  // resource id (see Resources)
	Becomes uint8   // the block left behind
	Time    float64 // seconds of digging with the basic pickaxe
}

var MineRules = map[string]map[uint8]Yield{
	// a rocky island: stone, ores, crystals, dirt and trees
	"quarry": {
		STONE:     {"stone", WORNSTONE, 0.45},
		COAL:      {"coal", WORNSTONE, 0.6},
		COPPER:    {"copper", WORNSTONE, 0.8},
		IRON:      {"iron", WORNSTONE, 1.1},
		GOLD:      {"gold", WORNSTONE, 1.5},
		CRYSTAL:   {"crystal", DULLCRYSTAL, 1.0},
		DIRT:      {"dirt", PACKEDDIRT, 0.3},
		GRASS:     {"dirt", PACKEDDIRT, 0.3},
		SAND:      {"sand", PACKEDDIRT, 0.3},
		SANDSTONE: {"sand", WORNSTONE, 0.45},
		WOOD:      {"wood", OLDWOOD, 0.5},
	},
}

// YieldAt is what mining block t on this island gives, if it can be mined
func (d *IsleDef) YieldAt(t uint8) (Yield, bool) {
	rules := MineRules[d.Mine]
	if rules == nil {
		return Yield{}, false
	}
	y, ok := rules[t]
	return y, ok
}

// Blocks are mined within this many px of a player's centre
const MineReach = 64.0

// One dig wears a small round patch of blocks (the aimed one sets the time and the
// resource, one per dig)
const MineRadius float64 = 2.2

func Brush(fn func(dx, dy int)) {
	r := int(math.Floor(MineRadius))
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if float64(dx*dx+dy*dy) <= MineRadius*MineRadius {
				fn(dx, dy)
			}
		}
	}
}

// A resource in the bag
type Resource struct {
	ID, Name string
	Col      [3]int
}

var Resources = []Resource{
	{"stone", "Stone", [3]int{120, 126, 140}},
	{"dirt", "Dirt", [3]int{122, 86, 60}},
	{"sand", "Sand", [3]int{226, 194, 107}},
	{"wood", "Wood", [3]int{112, 78, 52}},
	{"coal", "Coal", [3]int{50, 50, 60}},
	{"copper", "Copper", [3]int{210, 126, 70}},
	{"iron", "Iron", [3]int{214, 190, 172}},
	{"gold", "Gold", [3]int{246, 204, 70}},
	{"crystal", "Crystal", [3]int{120, 232, 255}},
}

func ResourceByID(id string) *Resource {
	for i := range Resources {
		if Resources[i].ID == id {
			return &Resources[i]
		}
	}
	return nil
}

// A changed block on an island (sent to everyone there)
type Edit struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T uint8 `json:"t"`
}

// ---- The sword ----

const (
	SwordReach = 52.0 // px from the player's centre
	SwordArc   = 1.2  // radians either side of the aim
	SwordEvery = 0.4  // seconds between swings
)

// Sword damage grows with level
func SwordDamage(level int) float64 { return 10 + 3*float64(level-1) }

// PetalsOn switches the petal system back on (orbiting petals instead of the sword).
// It's kept for later; while it's off, petals are neither drawn nor used in fights.
const PetalsOn = false

// Damage per second a character can deal, for away gains
func FightPower(c *Char) float64 {
	if PetalsOn {
		return LoadoutPower(c.Load)
	}
	return SwordDamage(c.Level) / SwordEvery
}

// Angle difference in -pi..pi
func AngleDiff(a, b float64) float64 {
	d := math.Mod(a-b+math.Pi, 2*math.Pi)
	if d < 0 {
		d += 2 * math.Pi
	}
	return d - math.Pi
}
