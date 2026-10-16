package game

import (
	"fmt"
	"math"
	"time"
)

// A generator digs coins on its own. Each one bought costs 15% more than the last.
type genDef struct {
	Name string
	Base float64 // cost of the first one
	CPS  float64 // coins per second each
}

var gens = []genDef{
	{"Miner", 15, 0.5},
	{"Mine Cart", 120, 4},
	{"Drill", 1100, 30},
	{"Excavator", 12000, 220},
	{"Mine Shaft", 130000, 1600},
	{"Laser Bore", 1.4e6, 12000},
	{"Core Tap", 2e7, 100000},
}

// Things bought with gems. Gems also give +10% coins each while kept,
// so every gem spent is a choice.
type gemItem struct {
	ID, Name, Desc string
	Cost           int
	Max            int // 0 means no limit
}

var gemShop = []gemItem{
	{"boost", "2x Boost", "Double coins for 15 min", 3, 0},
	{"warp", "Time Warp", "Get 2 hours of coins now", 5, 0},
	{"offline", "Bigger Cart", "+2h offline earnings", 10, 5},
	{"auto", "Auto Tapper", "Taps 5 times a second", 25, 1},
	{"golden", "Golden Pickaxe", "+50% coins forever", 40, 1},
}

// Real-money packs. Purchases are free placeholders until Google Play Billing is added.
type storePack struct {
	ID, Name, Desc, Price string
	Gems                  int
	OneTime               bool
}

var store = []storePack{
	{"starter", "Starter Pack", "60 gems + 2x Boost", "$1.99", 60, true},
	{"pouch", "Gem Pouch", "25 gems", "$0.99", 25, false},
	{"chest", "Gem Chest", "150 gems", "$4.99", 150, false},
	{"vault", "Gem Vault", "400 gems", "$9.99", 400, false},
	{"ad", "Watch Ad", "Free 2x Boost for 5 min", "FREE", 0, false},
}

// State is everything that is saved.
type State struct {
	Coins       float64        `json:"coins"`
	RunCoins    float64        `json:"runCoins"`   // earned since the last descent
	TotalCoins  float64        `json:"totalCoins"` // earned ever
	Pick        int            `json:"pick"`       // pickaxe level
	Owned       []int          `json:"owned"`      // generators owned, by index in gens
	Gems        int            `json:"gems"`
	Depth       int            `json:"depth"` // times descended
	Bought      map[string]int `json:"bought"`
	BoostUntil  int64          `json:"boostUntil"` // unix seconds
	LastSeen    int64          `json:"lastSeen"`
	DailyAt     int64          `json:"dailyAt"`
	DailyStreak int            `json:"dailyStreak"`
	Taps        int64          `json:"taps"`
}

func newState() *State {
	return &State{Owned: make([]int, len(gens)), Bought: map[string]int{}}
}

func (s *State) fix() {
	for len(s.Owned) < len(gens) {
		s.Owned = append(s.Owned, 0)
	}
	if s.Bought == nil {
		s.Bought = map[string]int{}
	}
}

func genCost(i, owned int) float64 { return math.Ceil(gens[i].Base * math.Pow(1.15, float64(owned))) }
func pickCost(lvl int) float64     { return math.Ceil(25 * math.Pow(1.7, float64(lvl))) }

// Multiplier from gems kept, the golden pickaxe and an active boost.
func (s *State) mult(now int64) float64 {
	m := 1 + 0.1*float64(s.Gems)
	if s.Bought["golden"] > 0 {
		m *= 1.5
	}
	if now < s.BoostUntil {
		m *= 2
	}
	return m
}

func (s *State) baseCPS() float64 {
	c := 0.0
	for i, n := range s.Owned {
		c += gens[i].CPS * float64(n)
	}
	return c
}

func (s *State) CPS(now int64) float64 { return s.baseCPS() * s.mult(now) }

// A tap gives the pickaxe's power plus 5% of coins per second, so taps stay worth doing.
func (s *State) TapValue(now int64) float64 {
	return (math.Pow(float64(s.Pick+1), 1.6) + 0.05*s.baseCPS()) * s.mult(now)
}

func (s *State) earn(c float64) {
	s.Coins += c
	s.RunCoins += c
	s.TotalCoins += c
}

func (s *State) BuyGen(i int) bool {
	c := genCost(i, s.Owned[i])
	if s.Coins < c {
		return false
	}
	s.Coins -= c
	s.Owned[i]++
	return true
}

func (s *State) BuyPick() bool {
	c := pickCost(s.Pick)
	if s.Coins < c {
		return false
	}
	s.Coins -= c
	s.Pick++
	return true
}

// Gems for descending now: grows with the square root of coins earned this run.
func (s *State) DescendGems() int { return int(math.Sqrt(s.RunCoins / 50000)) }

// Descend starts the mine over, keeping gems and gem-shop buys.
func (s *State) Descend() bool {
	g := s.DescendGems()
	if g < 1 {
		return false
	}
	s.Gems += g
	s.Depth++
	s.Coins, s.RunCoins, s.Pick = 0, 0, 0
	s.Owned = make([]int, len(gens))
	return true
}

func (s *State) addBoost(now int64, d time.Duration) {
	if s.BoostUntil < now {
		s.BoostUntil = now
	}
	s.BoostUntil += int64(d.Seconds())
}

func (s *State) BuyGem(i int, now int64) bool {
	it := gemShop[i]
	if s.Gems < it.Cost || (it.Max > 0 && s.Bought[it.ID] >= it.Max) {
		return false
	}
	s.Gems -= it.Cost
	s.Bought[it.ID]++
	switch it.ID {
	case "boost":
		s.addBoost(now, 15*time.Minute)
	case "warp":
		s.earn(s.CPS(now) * 7200)
	}
	return true
}

func (s *State) BuyPack(i int, now int64) bool {
	p := store[i]
	if p.OneTime && s.Bought[p.ID] > 0 {
		return false
	}
	s.Bought[p.ID]++
	s.Gems += p.Gems
	switch p.ID {
	case "starter":
		s.addBoost(now, 15*time.Minute)
	case "ad":
		s.addBoost(now, 5*time.Minute)
	}
	return true
}

func (s *State) offlineCap() int64 { return int64(2+2*s.Bought["offline"]) * 3600 }

// Offline gives half rate for the time away, up to the cap. Returns the coins given.
func (s *State) Offline(now int64) float64 {
	if s.LastSeen == 0 || now <= s.LastSeen {
		return 0
	}
	d := now - s.LastSeen
	if d > s.offlineCap() {
		d = s.offlineCap()
	}
	c := s.baseCPS() * s.mult(s.LastSeen) * float64(d) * 0.5
	s.earn(c)
	return c
}

const dailyGap = 20 * 3600

func (s *State) DailyReady(now int64) bool { return now-s.DailyAt >= dailyGap }

// Daily reward: 1 to 7 gems by streak. Missing two days resets the streak.
func (s *State) ClaimDaily(now int64) int {
	if !s.DailyReady(now) {
		return 0
	}
	if now-s.DailyAt > 2*dailyGap+8*3600 {
		s.DailyStreak = 0
	}
	s.DailyStreak++
	g := min(s.DailyStreak, 7)
	s.Gems += g
	s.DailyAt = now
	return g
}

var suffixes = []string{"", "K", "M", "B", "T", "Qa", "Qi", "Sx", "Sp", "Oc", "No", "Dc"}

func fmtNum(v float64) string {
	if v < 1000 {
		if v < 10 && v != math.Floor(v) {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.0f", math.Floor(v))
	}
	i := 0
	for v >= 1000 && i < len(suffixes)-1 {
		v /= 1000
		i++
	}
	return fmt.Sprintf("%.2f%s", v, suffixes[i])
}
