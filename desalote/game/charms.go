package game

import (
	"fmt"
	"math/rand"
)

// Charms: what you collect to make your hands score more (up to maxCharms at once).
// Hexes: bought to curse a rival's next round. Tomes: level up one of your picked hands.

const maxCharms = 5

type scoreCtx struct {
	run         *Run
	hand        HandType
	cards       []Card // the played cards (after a special hand's change)
	scoring     []int
	base, mult  float64
	xmult       float64
	special     bool
	firstPlay   bool
	lastPlay    bool
	usedDiscard bool
	notes       []string
}

func (c *scoreCtx) note(f string, a ...any) { c.notes = append(c.notes, fmt.Sprintf(f, a...)) }

type charmDef struct {
	name   string
	desc   string
	price  int
	rarity int // 0 common, 1 uncommon, 2 rare
	// passive changes to each round
	handSize, plays, discards int
	// while scoring a hand
	score func(c *scoreCtx)
	// money at the end of each round
	roundEnd int
	art      [3]uint32 // colours: body, light, dark
}

var charms = map[string]*charmDef{
	"kin": {name: "Kin Knot", desc: "+4 Mult for each of your picked hands in your own family", price: 6, rarity: 1,
		score: func(c *scoreCtx) {
			n := 0
			for _, h := range c.run.Picks {
				if hands[h].fam == c.run.Family {
					n++
				}
			}
			c.mult += float64(4 * n)
		}, art: [3]uint32{0x6fd38a, 0xc2f5cf, 0x2f8a52}},
	"outlier": {name: "Outlier", desc: "+40 points when the hand is a pick outside your family", price: 5,
		score: func(c *scoreCtx) {
			if f := hands[c.hand].fam; f >= 0 && f != c.run.Family {
				c.base += 40
			}
		}, art: [3]uint32{0xf09a5a, 0xffd2a8, 0xa65a24}},
	"lodestone": {name: "Lodestone", desc: "+1 Mult for each card in your deck of the scoring cards' main suit", price: 5,
		score: func(c *scoreCtx) {
			var n [4]int
			for _, i := range c.scoring {
				n[c.cards[i].S]++
			}
			best := 0
			for s := 1; s < 4; s++ {
				if n[s] > n[best] {
					best = s
				}
			}
			c.mult += float64(suitCounts(c.run.Deck)[best])
		}, art: [3]uint32{0x8c96b4, 0xd6dcf0, 0x4a5272}},
	"thirteen": {name: "Thirteen", desc: "x2 Mult if the scoring cards add up to a multiple of 13", price: 6, rarity: 1,
		score: func(c *scoreCtx) {
			sum := 0
			for _, i := range c.scoring {
				sum += c.cards[i].Value()
			}
			if sum > 0 && sum%13 == 0 {
				c.xmult *= 2
				c.note("Thirteen!")
			}
		}, art: [3]uint32{0xe05a7a, 0xffb4c6, 0x8c2440}},
	"sparkplug": {name: "Spark Plug", desc: "Use your special hand twice each round", price: 7, rarity: 2,
		art: [3]uint32{0xffde78, 0xfff3c4, 0xd9a93a}},
	"afterglow": {name: "Afterglow", desc: "Special hands buff their cards twice as much", price: 5,
		art: [3]uint32{0xff9ad6, 0xffd6f0, 0xb24a8c}},
	"pocketwatch": {name: "Pocket Watch", desc: "+1 play each round", price: 7, rarity: 2, plays: 1,
		art: [3]uint32{0xd9b25a, 0xfff0b4, 0x8c6a1e}},
	"sieve": {name: "Sieve", desc: "+2 discards each round", price: 4, discards: 2,
		art: [3]uint32{0x7ab4e0, 0xc8e6ff, 0x3a6a96}},
	"widegrip": {name: "Wide Grip", desc: "+1 card in your hand", price: 5, handSize: 1,
		art: [3]uint32{0xb48cf0, 0xe2d2ff, 0x6a46aa}},
	"satchel": {name: "Satchel", desc: "+2 cards in your hand", price: 9, rarity: 2, handSize: 2,
		art: [3]uint32{0x9a6a3a, 0xe0b88a, 0x5a3a1a}},
	"sleeve": {name: "Card Sleeve", desc: "+1 card in your hand and +1 discard", price: 7, rarity: 1, handSize: 1, discards: 1,
		art: [3]uint32{0x5a8ae0, 0xb0d0ff, 0x2a4a8a}},
	"tally": {name: "Tally Stone", desc: "+2 points for every hand you've played this match", price: 5,
		score: func(c *scoreCtx) { c.base += float64(2 * c.run.HandsPlayed) },
		art:   [3]uint32{0xa0a0a0, 0xe6e6e6, 0x5a5a5a}},
	"underdog": {name: "Underdog", desc: "x1.5 Mult while you have fewer lives than every rival", price: 6, rarity: 1,
		score: func(c *scoreCtx) {
			if c.run.Table.Underdog {
				c.xmult *= 1.5
			}
		}, art: [3]uint32{0x9a7a5a, 0xdcc0a0, 0x5a3e24}},
	"spite": {name: "Spite", desc: "+3 Mult for each hex you've cast this match", price: 5,
		score: func(c *scoreCtx) { c.mult += float64(3 * c.run.Table.HexesCast) },
		art:   [3]uint32{0x9a4ad6, 0xd6a8ff, 0x52207e}},
	"mirror": {name: "Mirror Ward", desc: "The next hex cast on you goes back to its caster, then this breaks", price: 5,
		art: [3]uint32{0xbfe6f0, 0xffffff, 0x6aa0b4}},
	"thorn": {name: "Thorn Ward", desc: "When someone hexes you, they pay you $3", price: 4,
		art: [3]uint32{0x5a9a4a, 0xa8dc96, 0x2a5a22}},
	"magpie": {name: "Magpie", desc: "+$1 whenever a rival wins an auction", price: 4,
		art: [3]uint32{0x3a3a52, 0x8a8ab4, 0x1a1a2a}},
	"haggler": {name: "Haggler", desc: "Auctions you win cost $2 less", price: 4,
		art: [3]uint32{0xd6a05a, 0xffdca8, 0x8a5a1e}},
	"purse": {name: "Coin Purse", desc: "+$2 at the end of each round", price: 4, roundEnd: 2,
		art: [3]uint32{0xc9a050, 0xffe6a0, 0x7a5a1e}},
	"lastgasp": {name: "Last Gasp", desc: "x2 Mult on a play after you've used all your discards", price: 6, rarity: 1,
		score: func(c *scoreCtx) {
			if c.usedDiscard && c.run.Discards == 0 && !c.special {
				c.xmult *= 2
			}
		}, art: [3]uint32{0xd04a3a, 0xff9a8a, 0x7a1a12}},
	"firstlight": {name: "First Light", desc: "+50 points on your first play of a round", price: 4,
		score: func(c *scoreCtx) {
			if c.firstPlay {
				c.base += 50
			}
		}, art: [3]uint32{0xfff0a0, 0xffffff, 0xd6b44a}},
	"lone": {name: "Lone Wolf", desc: "+20 Mult if you play exactly one card", price: 5,
		score: func(c *scoreCtx) {
			if len(c.cards) == 1 {
				c.mult += 20
			}
		}, art: [3]uint32{0x8a8aa0, 0xd0d0e6, 0x46465a}},
	"fullhand": {name: "Full Hand", desc: "+5 Mult when you play 5 cards", price: 5,
		score: func(c *scoreCtx) {
			if len(c.cards) == 5 {
				c.mult += 5
			}
		}, art: [3]uint32{0x5ac8b4, 0xb4f0e6, 0x22786a}},
	"lowtide": {name: "Low Tide", desc: "Scoring 2s, 3s, 4s and 5s score triple points", price: 5,
		art: [3]uint32{0x4a7ad6, 0x9ac0ff, 0x203c80}},
	"echo": {name: "Echo", desc: "The first scoring card scores twice", price: 5,
		art: [3]uint32{0xc0a0ff, 0xeee0ff, 0x7050c0}},
	"stacked": {name: "Stacked Deck", desc: "+2 Mult for each special-hand buff on your cards", price: 5, rarity: 1,
		score: func(c *scoreCtx) {
			n := 0
			for _, d := range c.run.Deck {
				n += d.Glow
			}
			c.mult += float64(2 * n)
		}, art: [3]uint32{0xe0c060, 0xfff0c0, 0x907020}},
	"rally": {name: "Rally Flag", desc: "+3 Mult for each other player still in the match", price: 5,
		score: func(c *scoreCtx) { c.mult += float64(3 * c.run.Table.Others) },
		art:   [3]uint32{0xe05050, 0xffb0b0, 0x902020}},
	"coin": {name: "Gambler's Coin", desc: "1 in 3: x3 Mult. Otherwise x0.75", price: 5,
		score: func(c *scoreCtx) {
			if c.run.rng.Intn(3) == 0 {
				c.xmult *= 3
				c.note("Heads!")
			} else {
				c.xmult *= 0.75
			}
		}, art: [3]uint32{0xd6c27a, 0xfff4c8, 0x8a7430}},
	"bloodhound": {name: "Bloodhound", desc: "+10 points for each life your rivals have lost", price: 5,
		score: func(c *scoreCtx) { c.base += float64(10 * c.run.Table.RivalLivesLost) },
		art:   [3]uint32{0x8a4a2a, 0xd09a7a, 0x4a2010}},
	"synergist": {name: "Synergist", desc: "Your synergy bonus is doubled", price: 8, rarity: 2,
		art: [3]uint32{0x6ae0e0, 0xd0ffff, 0x208a8a}},
}

var charmOrder = func() []string {
	ids := make([]string, 0, len(charms))
	for id := range charms {
		ids = append(ids, id)
	}
	sortStrings(ids)
	return ids
}()

// A random charm id, rarer ones less often, avoiding ids in skip
func randomCharm(rng *rand.Rand, skip map[string]bool) string {
	weights := [3]int{10, 5, 2}
	total := 0
	for _, id := range charmOrder {
		if !skip[id] {
			total += weights[charms[id].rarity]
		}
	}
	if total == 0 {
		return charmOrder[rng.Intn(len(charmOrder))]
	}
	n := rng.Intn(total)
	for _, id := range charmOrder {
		if skip[id] {
			continue
		}
		n -= weights[charms[id].rarity]
		if n < 0 {
			return id
		}
	}
	return charmOrder[0]
}

// ---- Hexes ----

type hexDef struct {
	name, desc string
	price      int
	now        bool // happens straight away instead of in their next round
}

var hexes = map[string]*hexDef{
	"fog":        {name: "Fog", desc: "Their first 2 cards are dealt face down next round", price: 3},
	"rust":       {name: "Rust", desc: "Their aces and face cards score 0 points next round", price: 4},
	"squeeze":    {name: "Squeeze", desc: "1 fewer card in their hand next round", price: 4},
	"muddle":     {name: "Muddle", desc: "Their picked hands get -3 Mult next round", price: 4},
	"jinx":       {name: "Jinx", desc: "They can't use their special hand next round", price: 3},
	"freeze":     {name: "Freeze", desc: "One of their charms does nothing next round", price: 3},
	"shackle":    {name: "Shackle", desc: "1 fewer discard next round", price: 4},
	"pickpocket": {name: "Pickpocket", desc: "Take $3 from them now", price: 3, now: true},
}

var hexOrder = []string{"fog", "rust", "squeeze", "muddle", "jinx", "freeze", "shackle", "pickpocket"}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
