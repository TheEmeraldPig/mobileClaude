package game

import (
	"fmt"
	"image/color"
	"math"
)

// Upgrades bought with coins. Each level costs more than the last.
type upgDef struct {
	ID, Name, Desc string
	Base           float64
	Max            int
}

var upgrades = []upgDef{
	{"magnet", "Star Magnet", "Pick up stars from further away", 40, 5},
	{"gravity", "Gravity Well", "Planets catch you from further away", 60, 5},
	{"shield", "Shield", "Survive an asteroid hit each run", 150, 3},
	{"value", "Star Value", "+1 coin per star", 80, 5},
	{"slow", "Slow Void", "The void chases you more slowly", 100, 5},
}

func upgCost(u upgDef, lvl int) int { return int(u.Base * math.Pow(2.2, float64(lvl))) }

// Ships change how you look. Some cost coins, the best ones gems.
type shipDef struct {
	ID, Name    string
	Coins, Gems int
	Body, Trail color.RGBA
	Shape       int // 0 dart, 1 needle, 2 saucer, 3 comet
}

var ships = []shipDef{
	{"dart", "Dart", 0, 0, color.RGBA{0x3e, 0xf0, 0xff, 0xff}, color.RGBA{0x3e, 0xf0, 0xff, 0xff}, 0},
	{"needle", "Needle", 400, 0, color.RGBA{0x4d, 0xff, 0xa0, 0xff}, color.RGBA{0x2f, 0xc8, 0xff, 0xff}, 1},
	{"saucer", "Saucer", 1200, 0, color.RGBA{0xff, 0xd8, 0x4a, 0xff}, color.RGBA{0xff, 0x8a, 0x3d, 0xff}, 2},
	{"comet", "Comet", 0, 30, color.RGBA{0xff, 0x4f, 0xd8, 0xff}, color.RGBA{0xa3, 0x5b, 0xff, 0xff}, 3},
}

// Real-money packs. Purchases are free placeholders until Google Play Billing is added.
type storePack struct {
	ID, Name, Desc, Price string
	Gems, Coins           int
	OneTime               bool
}

var store = []storePack{
	{"starter", "Starter Pack", "40 gems + 1000 coins", "$1.99", 40, 1000, true},
	{"pouch", "Gem Pouch", "25 gems", "$0.99", 25, 0, false},
	{"chest", "Gem Chest", "150 gems", "$4.99", 150, 0, false},
	{"doubler", "Coin Doubler", "2x coins from every run, forever", "$2.99", 0, 0, true},
	{"ad", "Watch Ad", "Get 100 free coins", "FREE", 0, 100, false},
}

const reviveGems = 5

// Profile is everything that is saved.
type Profile struct {
	Coins       int             `json:"coins"`
	Gems        int             `json:"gems"`
	Best        int             `json:"best"`
	Runs        int             `json:"runs"`
	Upg         map[string]int  `json:"upg"`
	Ships       map[string]bool `json:"ships"`
	Ship        string          `json:"ship"`
	Bought      map[string]int  `json:"bought"`
	DailyAt     int64           `json:"dailyAt"`
	DailyStreak int             `json:"dailyStreak"`
}

func newProfile() *Profile {
	p := &Profile{}
	p.fix()
	return p
}

func (p *Profile) fix() {
	if p.Upg == nil {
		p.Upg = map[string]int{}
	}
	if p.Ships == nil {
		p.Ships = map[string]bool{}
	}
	if p.Bought == nil {
		p.Bought = map[string]int{}
	}
	p.Ships["dart"] = true
	if !p.Ships[p.Ship] {
		p.Ship = "dart"
	}
}

func (p *Profile) ship() shipDef {
	for _, s := range ships {
		if s.ID == p.Ship {
			return s
		}
	}
	return ships[0]
}

func (p *Profile) BuyUpg(i int) error {
	u := upgrades[i]
	lvl := p.Upg[u.ID]
	if lvl >= u.Max {
		return fmt.Errorf("%s is maxed", u.Name)
	}
	c := upgCost(u, lvl)
	if p.Coins < c {
		return fmt.Errorf("Need %d more coins", c-p.Coins)
	}
	p.Coins -= c
	p.Upg[u.ID]++
	return nil
}

// TapShip buys a ship if needed, then flies it.
func (p *Profile) TapShip(i int) error {
	s := ships[i]
	if !p.Ships[s.ID] {
		if p.Coins < s.Coins || p.Gems < s.Gems {
			if s.Gems > 0 {
				return fmt.Errorf("%s costs %d gems", s.Name, s.Gems)
			}
			return fmt.Errorf("%s costs %d coins", s.Name, s.Coins)
		}
		p.Coins -= s.Coins
		p.Gems -= s.Gems
		p.Ships[s.ID] = true
	}
	p.Ship = s.ID
	return nil
}

func (p *Profile) BuyPack(i int) error {
	s := store[i]
	if s.OneTime && p.Bought[s.ID] > 0 {
		return fmt.Errorf("Already owned")
	}
	p.Bought[s.ID]++
	p.Gems += s.Gems
	p.Coins += s.Coins
	return nil
}

// Coins for a run: one per planet plus the stars, doubled with the Coin Doubler.
func (p *Profile) runCoins(score, stars int) int {
	c := score + stars*(1+p.Upg["value"])
	if p.Bought["doubler"] > 0 {
		c *= 2
	}
	return c
}

const dailyGap = 20 * 3600

func (p *Profile) DailyReady(now int64) bool { return now-p.DailyAt >= dailyGap }

// Daily reward: 1 to 7 gems by streak. Missing a day resets the streak.
func (p *Profile) ClaimDaily(now int64) int {
	if !p.DailyReady(now) {
		return 0
	}
	if now-p.DailyAt > 2*dailyGap+8*3600 {
		p.DailyStreak = 0
	}
	p.DailyStreak++
	g := min(p.DailyStreak, 7)
	p.Gems += g
	p.DailyAt = now
	return g
}
