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
)

// Profile is everything that is saved between runs, plus the run in progress.
type Profile struct {
	Gold        int            `json:"gold"`
	Gems        int            `json:"gems"`
	Energy      int            `json:"energy"`
	EnergyAt    int64          `json:"energyAt"`
	Cards       []Card         `json:"cards"` // your collection; In marks the starting deck
	HeroLv      int            `json:"heroLv"`
	MaxChapter  int            `json:"maxChapter"`
	Chapter     int            `json:"chapter"`
	Best        map[string]int `json:"best"` // best floor per chapter
	Bought      map[string]int `json:"bought"`
	FreeChestAt int64          `json:"freeChestAt"`
	Runs        int            `json:"runs"`
	Run         *Run           `json:"run,omitempty"`
}

func newProfile(now int64) *Profile {
	p := &Profile{Energy: maxEnergy, EnergyAt: now, Cards: starterCollection()}
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
	if len(p.Cards) == 0 {
		p.Cards = starterCollection()
	}
	if checkLoadout(p.Loadout()) != nil { // an old or broken save: fall back to the first legal deck
		for i := range p.Cards {
			p.Cards[i].In = false
		}
		for i := range p.Cards {
			p.Cards[i].In = true
			if checkLoadout(p.Loadout()) != nil {
				p.Cards[i].In = false
			}
		}
	}
	p.MaxChapter = max(1, p.MaxChapter)
	p.Chapter = max(1, min(p.Chapter, p.MaxChapter))
	if p.Run != nil {
		p.Run.restore()
	}
}

// Loadout is the starting deck.
func (p *Profile) Loadout() []Card {
	var d []Card
	for _, c := range p.Cards {
		if c.In {
			c.In = false
			d = append(d, c)
		}
	}
	return d
}

// Toggle puts collection card i into the starting deck, or takes it out.
func (p *Profile) Toggle(i int) error {
	if p.Cards[i].In {
		p.Cards[i].In = false
		if len(p.Loadout()) == 0 {
			p.Cards[i].In = true
			return fmt.Errorf("Your deck needs at least 1 card")
		}
		return nil
	}
	p.Cards[i].In = true
	if err := checkLoadout(p.Loadout()); err != nil {
		p.Cards[i].In = false
		return err
	}
	return nil
}

func (p *Profile) maxHP() int      { return 50 + 6*p.HeroLv }
func (p *Profile) heroCost() int   { return int(100 * math.Pow(1.45, float64(p.HeroLv))) }
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

// Enhance puts an enhancement on collection card i for good, replacing any it had.
func (p *Profile) Enhance(i, e int) error {
	d := enhDefs[e]
	if p.Cards[i].Enh == e {
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
	p.Cards[i].Enh = e
	return nil
}

// sortCards orders the collection: starting deck first, then rarity, card and enhancement.
func (p *Profile) sortCards() {
	sort.SliceStable(p.Cards, func(a, b int) bool {
		x, y := p.Cards[a], p.Cards[b]
		if x.In != y.In {
			return x.In
		}
		if x.def().Rarity != y.def().Rarity {
			return x.def().Rarity > y.def().Rarity
		}
		if cardIndex[x.ID] != cardIndex[y.ID] {
			return cardIndex[x.ID] < cardIndex[y.ID]
		}
		return x.Enh > y.Enh
	})
}

// rollRarity picks a rarity from weights (common..legendary).
func rollRarity(rng *rand.Rand, w [rarityCount]int) int {
	t := 0
	for _, v := range w {
		t += v
	}
	n := rng.Intn(t)
	for i, v := range w {
		if n < v {
			return i
		}
		n -= v
	}
	return rCommon
}

func randomCardOf(rng *rand.Rand, rarity int) Card {
	ids := cardsOfRarity(rarity)
	return Card{ID: ids[rng.Intn(len(ids))]}
}

// ---- home shop ----

type shopItem struct {
	ID, Name, Desc, Price string
	Gold, Gems            int // cost in game currency (0 for real money / free)
	Chest                 int // 0 none, 1 wood, 2 royal
	Odds                  [rarityCount]int
}

var shopItems = []shopItem{
	{"free", "Daily Chest", "A free card every day", "FREE", 0, 0, 1, [5]int{60, 30, 10, 0, 0}},
	{"wood", "Card Chest", "Common to Epic", "300", 300, 0, 1, [5]int{45, 35, 16, 4, 0}},
	{"royal", "Royal Chest", "Rare to Legendary", "50", 0, 50, 2, [5]int{0, 0, 62, 30, 8}},
	{"energy", "Energy Refill", "Fill energy to 20", "10", 0, 10, 0, [5]int{}},
	{"starter", "Starter Pack", "120 gems + 1500 gold", "$1.99", 0, 0, 0, [5]int{}},
	{"pouch", "Gem Pouch", "60 gems", "$0.99", 0, 0, 0, [5]int{}},
	{"sack", "Gem Sack", "350 gems", "$4.99", 0, 0, 0, [5]int{}},
	{"ad", "Watch Ad", "+5 energy", "FREE", 0, 0, 0, [5]int{}},
}

// Buy runs a shop item and returns the card when a chest was opened.
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
	c := randomCardOf(rng, rollRarity(rng, it.Odds))
	if it.Chest == 2 && rng.Intn(3) == 0 {
		c.Enh = 1 + rng.Intn(eCount-2) // any but Echo
	}
	p.Cards = append(p.Cards, c)
	return &c, nil
}
