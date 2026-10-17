package game

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

const (
	maxEnergy   = 20
	runCost     = 5
	energyEvery = 360 // seconds per energy point
	reviveGems  = 10
	dailyGap    = 20 * 3600
	minDeck     = 8
)

// Profile is everything that is saved between runs.
type Profile struct {
	Gold        int            `json:"gold"`
	Gems        int            `json:"gems"`
	Energy      int            `json:"energy"`
	EnergyAt    int64          `json:"energyAt"`
	Deck        []Card         `json:"deck"`
	HeroLv      int            `json:"heroLv"`
	MaxChapter  int            `json:"maxChapter"`
	Chapter     int            `json:"chapter"`
	Best        map[string]int `json:"best"` // best floor per chapter
	Bought      map[string]int `json:"bought"`
	FreeChestAt int64          `json:"freeChestAt"`
	Runs        int            `json:"runs"`
}

func newProfile(now int64) *Profile {
	p := &Profile{Energy: maxEnergy, EnergyAt: now, Deck: starterDeck()}
	p.fix()
	return p
}

func (p *Profile) fix() {
	if p.Best == nil {
		p.Best = map[string]int{}
	}
	if p.Bought == nil {
		p.Bought = map[string]int{}
	}
	if len(p.Deck) == 0 {
		p.Deck = starterDeck()
	}
	p.MaxChapter = max(1, p.MaxChapter)
	p.Chapter = max(1, min(p.Chapter, p.MaxChapter))
}

func (p *Profile) maxHP() int    { return 50 + 6*p.HeroLv }
func (p *Profile) heroCost() int { return int(100 * math.Pow(1.45, float64(p.HeroLv))) }

func (p *Profile) best(ch int) int { return p.Best[fmt.Sprint(ch)] }

// tickEnergy refills one energy every few minutes, up to the cap.
func (p *Profile) tickEnergy(now int64) {
	if p.Energy >= maxEnergy {
		p.EnergyAt = now
		return
	}
	n := (now - p.EnergyAt) / energyEvery
	if n > 0 {
		p.Energy = min(maxEnergy, p.Energy+int(n))
		p.EnergyAt += n * energyEvery
	}
}

func (p *Profile) nextEnergyIn(now int64) int64 { return energyEvery - (now - p.EnergyAt) }

func (p *Profile) UpgradeHero() error {
	c := p.heroCost()
	if p.Gold < c {
		return fmt.Errorf("Need %d more gold", c-p.Gold)
	}
	p.Gold -= c
	p.HeroLv++
	return nil
}

// Enhance puts an enhancement on deck card i, replacing any it had.
func (p *Profile) Enhance(i, e int) error {
	d := enhDefs[e]
	if p.Deck[i].Enh == e {
		return fmt.Errorf("Already %s", d.Name)
	}
	if p.Gold < d.Gold || p.Gems < d.Gems {
		if d.Gems > 0 {
			return fmt.Errorf("%s costs %d gems", d.Name, d.Gems)
		}
		return fmt.Errorf("Need %d more gold", d.Gold-p.Gold)
	}
	p.Gold -= d.Gold
	p.Gems -= d.Gems
	p.Deck[i].Enh = e
	return nil
}

const removeCost = 75

func (p *Profile) Remove(i int) error {
	if len(p.Deck) <= minDeck {
		return fmt.Errorf("Your deck needs at least %d cards", minDeck)
	}
	if p.Gold < removeCost {
		return fmt.Errorf("Need %d more gold", removeCost-p.Gold)
	}
	p.Gold -= removeCost
	p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
	return nil
}

// sortDeck keeps the Deck screen tidy: by card, then enhancement.
func (p *Profile) sortDeck() {
	idx := map[string]int{}
	for i, c := range cardDefs {
		idx[c.ID] = i
	}
	sort.SliceStable(p.Deck, func(a, b int) bool {
		if idx[p.Deck[a].ID] != idx[p.Deck[b].ID] {
			return idx[p.Deck[a].ID] < idx[p.Deck[b].ID]
		}
		return p.Deck[a].Enh < p.Deck[b].Enh
	})
}

// randomCard picks a non-starter card; rare ones a third as often.
func randomCard(rng *rand.Rand, rareBoost bool) Card {
	var pool []string
	for _, c := range cardDefs {
		n := 0
		switch c.Rarity {
		case 1:
			n = 3
		case 2:
			n = 1
			if rareBoost {
				n = 4
			}
		}
		for k := 0; k < n; k++ {
			pool = append(pool, c.ID)
		}
	}
	return Card{ID: pool[rng.Intn(len(pool))]}
}

// ---- shop ----

type shopItem struct {
	ID, Name, Desc, Price string
	Gold, Gems            int // cost in game currency (0 for real money / free)
	Chest                 int // 0 none, 1 wood, 2 royal
}

var shopItems = []shopItem{
	{"free", "Daily Chest", "A free card every day", "FREE", 0, 0, 1},
	{"wood", "Card Chest", "A random card", "250", 250, 0, 1},
	{"royal", "Royal Chest", "Rare card, enhanced", "40", 0, 40, 2},
	{"energy", "Energy Refill", "Fill energy to 20", "10", 0, 10, 0},
	{"starter", "Starter Pack", "120 gems + 1500 gold", "$1.99", 0, 0, 0},
	{"pouch", "Gem Pouch", "60 gems", "$0.99", 0, 0, 0},
	{"sack", "Gem Sack", "350 gems", "$4.99", 0, 0, 0},
	{"ad", "Watch Ad", "+5 energy", "FREE", 0, 0, 0},
}

// Buy runs a shop item. It returns a card when a chest was opened.
// Real-money items are free placeholders until Google Play Billing is added.
func (p *Profile) Buy(i int, now int64, rng *rand.Rand) (*Card, error) {
	it := shopItems[i]
	if p.Gold < it.Gold {
		return nil, fmt.Errorf("Need %d more gold", it.Gold-p.Gold)
	}
	if p.Gems < it.Gems {
		return nil, fmt.Errorf("Need %d more gems", it.Gems-p.Gems)
	}
	switch it.ID {
	case "free":
		if now-p.FreeChestAt < dailyGap {
			left := dailyGap - (now - p.FreeChestAt)
			return nil, fmt.Errorf("Next free chest in %dh %dm", left/3600, left%3600/60)
		}
		p.FreeChestAt = now
	case "energy":
		if p.Energy >= maxEnergy {
			return nil, fmt.Errorf("Energy is already full")
		}
		p.Energy = maxEnergy
	case "starter":
		if p.Bought["starter"] > 0 {
			return nil, fmt.Errorf("Already bought")
		}
		p.Gems += 120
		p.Gold += 1500
	case "pouch":
		p.Gems += 60
	case "sack":
		p.Gems += 350
	case "ad":
		p.Energy = min(maxEnergy+5, p.Energy+5)
	}
	p.Gold -= it.Gold
	p.Gems -= it.Gems
	p.Bought[it.ID]++
	if it.Chest == 0 {
		return nil, nil
	}
	c := randomCard(rng, it.Chest == 2)
	if it.Chest == 2 {
		c.Enh = 1 + rng.Intn(eCount-2) // any but Echo
	}
	p.Deck = append(p.Deck, c)
	p.sortDeck()
	return &c, nil
}
