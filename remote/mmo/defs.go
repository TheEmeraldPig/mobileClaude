package mmo

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ---- Rarity ----

const NRarity = 6

type Rarity struct {
	Name string
	Col  [3]int
}

var Rarities = [NRarity]Rarity{
	{"Common", [3]int{126, 239, 109}},
	{"Unusual", [3]int{255, 230, 93}},
	{"Rare", [3]int{77, 82, 227}},
	{"Epic", [3]int{134, 31, 222}},
	{"Legendary", [3]int{222, 31, 31}},
	{"Mythic", [3]int{31, 219, 222}},
}

// Damage and health triple with each rarity (for petals and creatures alike)
func Power(r int) float64 { return math.Pow(3, float64(r)) }

// Healing grows more slowly, so it keeps up with the mote's health
func HealMult(r int) float64 { return math.Pow(1.7, float64(r)) }

// ---- Petals ----

// A petal circles the mote. It hits creatures it touches and loses health when it does;
// when it breaks it grows back after its reload time.
type PetalKind struct {
	ID, Name, Desc string
	Dmg, HP        float64 // at Common
	Reload         float64 // seconds
	Heal           float64 // health per second while it's out (Leaf)
	Mend           float64 // health given back when it's used up on a hurt mote (Heart)
	Light          int     // light it gives off (0-64)
	Burn           float64 // damage per second for 3 seconds (Ember)
	Slow           float64 // seconds a creature is slowed (Frost)
	Col            [3]int
}

var Petals = []*PetalKind{
	{ID: "spark", Name: "Spark", Desc: "A plain, steady petal.", Dmg: 10, HP: 10, Reload: 2.0, Light: 14, Col: [3]int{255, 222, 120}},
	{ID: "pebble", Name: "Pebble", Desc: "Heavy: hits hard, breaks easily.", Dmg: 17, HP: 6, Reload: 3.0, Col: [3]int{150, 156, 170}},
	{ID: "thorn", Name: "Thorn", Desc: "A huge hit, then it snaps.", Dmg: 30, HP: 2, Reload: 3.8, Col: [3]int{196, 92, 88}},
	{ID: "leaf", Name: "Leaf", Desc: "Slowly heals you while it's out.", Dmg: 6, HP: 12, Reload: 2.0, Heal: 1.2, Col: [3]int{98, 200, 96}},
	{ID: "glow", Name: "Glow", Desc: "Lights up dark caves around you.", Dmg: 6, HP: 14, Reload: 2.2, Light: 46, Col: [3]int{140, 236, 255}},
	{ID: "ember", Name: "Ember", Desc: "Sets creatures burning.", Dmg: 7, HP: 7, Reload: 2.4, Burn: 5, Light: 24, Col: [3]int{255, 140, 60}},
	{ID: "frost", Name: "Frost", Desc: "Slows the creatures it hits.", Dmg: 7, HP: 9, Reload: 2.4, Slow: 2, Col: [3]int{176, 214, 255}},
	{ID: "heart", Name: "Heart", Desc: "Used up to heal you when you're hurt.", HP: 5, Reload: 3.5, Mend: 14, Col: [3]int{255, 120, 160}},
}

func PetalByID(id string) *PetalKind {
	for _, p := range Petals {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// A petal in an inventory: its kind and rarity, written "spark:2"
type PetalID string

func MakePetal(kind string, r int) PetalID { return PetalID(kind + ":" + strconv.Itoa(r)) }

func (p PetalID) Split() (*PetalKind, int, bool) {
	k, rs, ok := strings.Cut(string(p), ":")
	if !ok {
		return nil, 0, false
	}
	r, err := strconv.Atoi(rs)
	kind := PetalByID(k)
	if err != nil || kind == nil || r < 0 || r >= NRarity {
		return nil, 0, false
	}
	return kind, r, true
}

func (p PetalID) Valid() bool { _, _, ok := p.Split(); return ok }

func (p PetalID) Name() string {
	k, r, ok := p.Split()
	if !ok {
		return string(p)
	}
	return Rarities[r].Name + " " + k.Name
}

// ---- Creatures ----

type MobKind struct {
	ID, Name   string
	HP, Dmg    float64 // at Common
	Speed      float64 // px per second (the mote walks at 150)
	Fly        bool
	W, H       int // cells
	Sense      float64
	XP, Coins  float64
	Drops      []string // petal kinds it can drop
	Light      int
	Hop        bool // jumps at you when close
	DropChance float64
}

var Mobs = []*MobKind{
	{ID: "skitter", Name: "Skitter", HP: 20, Dmg: 6, Speed: 110, W: 6, H: 4, Sense: 220, XP: 4, Coins: 1, Drops: []string{"spark", "pebble"}, DropChance: 0.35},
	{ID: "gloom", Name: "Gloomling", HP: 35, Dmg: 10, Speed: 75, W: 8, H: 6, Sense: 200, XP: 6, Coins: 2, Drops: []string{"leaf", "spark", "heart"}, Hop: true, DropChance: 0.4},
	{ID: "bat", Name: "Bat", HP: 18, Dmg: 8, Speed: 100, Fly: true, W: 7, H: 5, Sense: 260, XP: 5, Coins: 1, Drops: []string{"spark", "thorn"}, DropChance: 0.35},
	{ID: "wisp", Name: "Wisp", HP: 40, Dmg: 12, Speed: 80, Fly: true, W: 6, H: 6, Sense: 280, XP: 9, Coins: 3, Drops: []string{"glow", "frost"}, Light: 30, DropChance: 0.4},
	{ID: "brute", Name: "Brute", HP: 120, Dmg: 22, Speed: 50, W: 12, H: 9, Sense: 220, XP: 20, Coins: 6, Drops: []string{"pebble", "thorn", "heart"}, DropChance: 0.5},
	{ID: "shade", Name: "Shade", HP: 60, Dmg: 16, Speed: 105, W: 8, H: 6, Sense: 300, XP: 12, Coins: 4, Drops: []string{"ember", "thorn"}, Hop: true, DropChance: 0.4},
	{ID: "warden", Name: "Warden", HP: 900, Dmg: 40, Speed: 85, Fly: true, W: 16, H: 12, Sense: 420, XP: 300, Coins: 120, Drops: []string{"ember", "glow", "frost", "heart"}, Light: 40, DropChance: 1},
}

func MobByID(id string) *MobKind {
	for _, m := range Mobs {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// Creature health, contact damage and rewards at a rarity
func (m *MobKind) MaxHP(r int) float64        { return m.HP * Power(r) }
func (m *MobKind) Damage(r int) float64       { return m.Dmg * math.Pow(2, float64(r)) }
func (m *MobKind) XPFor(r int) int64          { return int64(math.Round(m.XP * math.Pow(2.5, float64(r)))) }
func (m *MobKind) CoinsFor(r int) int64       { return int64(math.Round(m.Coins * math.Pow(2, float64(r)))) }
func (m *MobKind) SizePx() (float64, float64) { return float64(m.W * Cell), float64(m.H * Cell) }

// ---- Levels ----

const MaxLevel = 60

// XP needed to go from level l to l+1
func XPNeed(l int) int64 { return int64(30 * math.Pow(float64(l), 1.8)) }

func MaxHPAt(l int) float64 { return 100 + 12*float64(l-1) }

// Petal slots: 5, then one more at levels 8, 16 and 25
func SlotsAt(l int) int {
	n := 5
	for _, at := range []int{8, 16, 25} {
		if l >= at {
			n++
		}
	}
	return n
}

// Coins to combine five petals of rarity r into one of rarity r+1
func CombineCost(r int) int64 { return int64(5 * math.Pow(3, float64(r))) }

const CombineN = 5

// Petal orbit, in world px from the mote's centre
const (
	OrbitNormal = 30.0
	OrbitAttack = 62.0
	OrbitDefend = 18.0
	PetalR      = 5.0 // a petal's hit radius
	PetalSpin   = 2.6 // radians per second
)

func OrbitFor(mode int) float64 {
	switch mode {
	case ModeAttack:
		return OrbitAttack
	case ModeDefend:
		return OrbitDefend
	}
	return OrbitNormal
}

const (
	ModeNormal = 0
	ModeAttack = 1
	ModeDefend = 2
)

// Where petal i of n is around a mote at time t (server seconds)
func PetalPos(cx, cy, radius, t float64, i, n int) (float64, float64) {
	a := t*PetalSpin + 2*math.Pi*float64(i)/float64(max(1, n))
	return cx + math.Cos(a)*radius, cy + math.Sin(a)*radius
}

func FmtNum(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 10000 {
		return s
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(n)/1e6)
}
