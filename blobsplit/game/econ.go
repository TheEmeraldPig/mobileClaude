package game

import (
	"fmt"
	"math"
	"strings"
)

// Rarity is a kind of blob. Every new blob made by a split rolls one: the rarest kind
// whose chance (luck / Odds) comes up wins, otherwise it is a plain blob.
type Rarity struct {
	Name  string
	Odds  float64 // 1 in Odds
	Mult  float64 // sell value multiplier
	Body  uint32  // body colour
	Label uint32  // colour of its "1/N" label
}

var rarities = []Rarity{
	{"Blob", 1, 1, 0xe8414f, 0xffffff},
	{"Gold", 100, 5, 0xf5c02c, 0xffd95a},
	{"Shadow", 250, 12, 0x3b3550, 0xb9a8ff},
	{"Frost", 600, 30, 0x6fd6f2, 0x9ff0ff},
	{"Heart", 1500, 75, 0xff6fb5, 0xff9fd0},
	{"Toxic", 4000, 200, 0x7be04a, 0xb6ff7a},
	{"Rainbow", 12000, 700, 0xffffff, 0xffffff},
	{"Cosmic", 50000, 3000, 0x1d1240, 0xd9c6ff},
}

// Upgrades bought with money. They reset on rebirth.
const (
	maxCapLv   = 4  // capacity 4, 8, 16, 32, 64
	maxSpeedLv = 5  // 3.0s down to 2.0s
	maxLuckLv  = 20 // luck x1.0 up to x3.0
)

func capOf(l int) int         { return 4 << l }
func capCost(l int) float64   { return growCost(10, 1.2, 0.25, l) }
func speedCost(l int) float64 { return growCost(15, 1.0, 0.15, l) }
func valueOf(l int) float64   { return (1 + float64(l)) * math.Pow(1.07, float64(l)) }
func valueCost(l int) float64 { return growCost(3, 0.07, 0.0015, l) }
func luckOf(l int) float64    { return 1 + 0.1*float64(l) }
func luckCost(l int) float64  { return growCost(50, 0.25, 0.02, l) }

// growCost is base * e^(a*l + b*l*l) + l: each level costs only a little more than
// the last while levels are low (e^a times, and always at least $1 more), and the
// steps get steeper as levels go up.
func growCost(base, a, b float64, l int) float64 {
	x := float64(l)
	return math.Round(base*math.Exp(a*x+b*x*x) + x)
}
func baseCooldown(l int) float64 { return 3.0 - 0.2*float64(l) }

// Rebirth: needs money, resets the run and doubles all blob value; pays gems.
func rebirthReq(r int) float64  { return 25000 * math.Pow(10, float64(r)) }
func rebirthMult(r int) float64 { return math.Pow(2, float64(r)) }
func rebirthGems(r int) int     { return 3 + 2*r }

// Gem upgrades are permanent.
type gemUp struct {
	id, name, desc string
	max            int
	base           int // cost of level l is base * 2^l
}

var gemUps = []gemUp{
	{"auto", "Auto Split", "Splits for you", 1, 2},
	{"autosell", "Auto Sell", "Sells when full", 1, 5},
	{"golden", "Golden Touch", "+50% blob value", 10, 2},
	{"lucky", "Lucky Charm", "+50% luck", 10, 3},
	{"quick", "Quick Hands", "-10% split time", 5, 4},
}

func gemCost(u gemUp, l int) int { return u.base << l }

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

func trimZeros(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}
