package game

import (
	"fmt"
	"math"
	"math/bits"
	"strings"
)

// Rarity is a mutation a biote can carry. rarities[0] is the plain biote; the others
// are mutations, and a biote can carry several at once (a bit mask, bit r for
// mutation r): their multipliers multiply. Without luck the chance that a new biote
// gains any mutation is 1 in mutationOdds; which one follows the shown odds.
type Rarity struct {
	Name  string
	Odds  float64 // 1 in Odds
	Mult  float64 // sell value multiplier
	Body  uint32  // body colour
	Label uint32  // colour of its "1/N" label
}

var rarities = []Rarity{
	{"Biote", 1, 1, 0xe8414f, 0xffffff},
	{"Gold", 25, 3, 0xf5c02c, 0xffd95a},
	{"Shadow", 60, 6, 0x3b3550, 0xb9a8ff},
	{"Frost", 150, 15, 0x6fd6f2, 0x9ff0ff},
	{"Heart", 400, 40, 0xff6fb5, 0xff9fd0},
	{"Toxic", 1000, 100, 0x7be04a, 0xb6ff7a},
	{"Rainbow", 3000, 300, 0xffffff, 0xffffff},
	{"Cosmic", 10000, 1000, 0x1d1240, 0xd9c6ff},
}

// Mutation ids, for cards and events.
const (
	mGold = 1 + iota
	mShadow
	mFrost
	mHeart
	mToxic
	mRainbow
	mCosmic
	nMut = 8 // len(rarities)
)

const mutationOdds = 15

// mutationChance is the base chance (luck x1) that a new biote gains mutation r.
func mutationChance(r int) float64 {
	sum := 0.0
	for _, x := range rarities[1:] {
		sum += 1 / x.Odds
	}
	return (1 / rarities[r].Odds) / sum / mutationOdds
}

// Mutation masks.
func maskHas(m, r int) bool { return m&(1<<r) != 0 }
func maskCount(m int) int   { return bits.OnesCount(uint(m)) }

// maskTop is the rarest mutation in m (0 for a plain biote); it picks the sprite.
func maskTop(m int) int {
	if m == 0 {
		return 0
	}
	return bits.Len(uint(m)) - 1
}

func maskList(m int) []int {
	var out []int
	for r := 1; r < nMut; r++ {
		if maskHas(m, r) {
			out = append(out, r)
		}
	}
	return out
}

// comboOdds is how rare a mask is: its mutations' odds multiplied together.
func comboOdds(m int) float64 {
	o := 1.0
	for _, r := range maskList(m) {
		o *= rarities[r].Odds
	}
	return o
}

// Upgrades bought with money: biote value, split speed and capacity. They reset on
// rebirth. Luck comes from the skill tree, cards, the time of day and events.
const (
	maxCapLv   = 4 // capacity 4, 8, 16, 32, 64
	maxSpeedLv = 5 // 3.0s down to 2.0s
)

func capOf(l int) int            { return 4 << l }
func capCost(l int) float64      { return growCost(10, 1.2, 0.25, l) }
func speedCost(l int) float64    { return growCost(15, 1.0, 0.15, l) }
func valueOf(l int) float64      { return (1 + float64(l)) * math.Pow(1.07, float64(l)) }
func valueCost(l int) float64    { return growCost(3, 0.07, 0.0015, l) }
func baseCooldown(l int) float64 { return 3.0 - 0.2*float64(l) }

// growCost is base * e^(a*l + b*l*l) + l: each level costs only a little more than
// the last while levels are low (e^a times, and always at least $1 more), and the
// steps get steeper as levels go up.
func growCost(base, a, b float64, l int) float64 {
	x := float64(l)
	return math.Round(base*math.Exp(a*x+b*x*x) + x)
}

// Rebirth: needs money, resets the run (money, upgrades, rounds) and doubles biote
// value for good. It pays skill points: one, plus one for every Outbreak beaten.
func rebirthReq(r int) float64     { return 25000 * math.Pow(10, float64(r)) }
func rebirthMult(r int) float64    { return math.Pow(2, float64(r)) }
func rebirthPoints(bosses int) int { return 1 + bosses }

// Rounds, three per ante, like Balatro's blinds: earn the target in a limited
// number of sells. The third is a boss with a twist.
var roundNames = []string{"PETRI", "COLONY", "OUTBREAK"}

func roundTarget(ante, round int) float64 {
	return 30 * math.Pow(6, float64(ante-1)) * []float64{1, 2, 3.5}[round]
}

func roundReward(round int) int { return []int{2, 3, 5}[round] }

const baseSells = 4

var suffixes = []string{"", "K", "M", "B", "T", "Qa", "Qi", "Sx", "Sp", "Oc", "No", "Dc"}

// fmtNum writes big numbers short: 1234 -> 1.23K, 1.6e9 -> 1.6B.
func fmtNum(f float64) string {
	if f < 1000 {
		if f == math.Floor(f) {
			return fmt.Sprintf("%.0f", f)
		}
		if f < 10 {
			return trimZeros(fmt.Sprintf("%.2f", f))
		}
		return fmt.Sprintf("%.0f", math.Floor(f))
	}
	e := int(math.Floor(math.Log10(f) / 3))
	if e >= len(suffixes) {
		return fmt.Sprintf("%.2e", f)
	}
	v := f / math.Pow(1000, float64(e))
	var s string
	switch {
	case v >= 100:
		s = fmt.Sprintf("%.0f", math.Floor(v))
	case v >= 10:
		s = trimZeros(fmt.Sprintf("%.1f", math.Floor(v*10)/10))
	default:
		s = trimZeros(fmt.Sprintf("%.2f", math.Floor(v*100)/100))
	}
	return s + suffixes[e]
}

func fmtFixed(f float64, d int) string { return fmt.Sprintf("%.*f", d, f) }

func sprintf1(f float64) string { return trimZeros(fmtFixed(f, 1)) }

func trimZeros(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// fmtTime writes seconds as m:ss.
func fmtTime(s float64) string {
	n := int(math.Max(0, s))
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}
