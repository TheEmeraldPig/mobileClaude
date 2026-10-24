package game

import "strings"

// Ability cards: won from the challenge (one every 10 floors past your best) and
// always active once unlocked. Each has a rarity and a type. Most are made from
// tables: every mutation gets a chance, value and spread card at different rarities,
// and each type has a ladder of general cards.

var cardRarities = []struct {
	name string
	sk   skin
	odds float64 // weight when a card is won
}{
	{"COMMON", skin{rgb(0x5a6a8a), rgb(0x8a9aba), rgb(0x33405a)}, 50},
	{"UNCOMMON", skin{rgb(0x2f9a4a), rgb(0x6fe07a), rgb(0x1a5a2a)}, 28},
	{"RARE", skin{rgb(0x3a6ae0), rgb(0x8ab0ff), rgb(0x203a8a)}, 14},
	{"EPIC", skin{rgb(0x9a3ad8), rgb(0xd08aff), rgb(0x5a1a8a)}, 6},
	{"LEGENDARY", skin{rgb(0xe09a20), rgb(0xffe08a), rgb(0x8a5010)}, 2},
}

const (
	rCommon = iota
	rUncommon
	rRare
	rEpic
	rLegendary
)

// Card types, with the colour of the stripe on the card.
var cardTypes = []struct {
	name string
	col  uint32
}{
	{"LUCK", 0x5fd86a}, {"VALUE", 0xffc94a}, {"SPEED", 0x5fd0ff}, {"GENE", 0xff7ab8},
	{"SPREAD", 0xc0ff60}, {"TIME", 0xb07cff}, {"STORM", 0x9ff0ff}, {"CLIMB", 0x6a8cff},
	{"COMBO", 0xff9a4a}, {"IDLE", 0xd8ccff},
}

const (
	tLuck = iota
	tValue
	tSpeed
	tGene
	tSpread
	tTime
	tStorm
	tClimb
	tCombo
	tIdle
)

type cardDef struct {
	id, name, desc string
	rar, typ       int
	mut            int // mutation shown on the card (0 = a plain biote)
	apply          func(g *Game, m *mods)
}

var cards []cardDef

func init() { cards = buildCards() }

func buildCards() []cardDef {
	var out []cardDef
	add := func(id, name, desc string, rar, typ, mut int, f func(g *Game, m *mods)) {
		out = append(out, cardDef{id, name, desc, rar, typ, mut, f})
	}
	// every mutation: two chance cards, two value cards and a spread card
	for r := 1; r < nMut; r++ {
		r := r
		n := rarities[r].Name
		lo := strings.ToLower(n)
		add(lo+"_charm", n+" Charm", n+" x1.5 chance", rCommon, tGene, r, func(g *Game, m *mods) { m.chance[r] *= 1.5 })
		add(lo+"_lure", n+" Lure", n+" x3 chance", rRare, tGene, r, func(g *Game, m *mods) { m.chance[r] *= 3 })
		add(lo+"_polish", n+" Polish", n+" x1.5 value", rUncommon, tGene, r, func(g *Game, m *mods) { m.worth[r] *= 1.5 })
		add(lo+"_crown", n+" Crown", n+" x3 value", rEpic, tGene, r, func(g *Game, m *mods) { m.worth[r] *= 3 })
		add(lo+"_spores", n+" Spores", n+" spreads x2", rUncommon, tSpread, r, func(g *Game, m *mods) { m.spreadMut[r] *= 2 })
	}
	ladder := func(typ int, items []struct {
		id, name, desc string
		rar            int
		mut            int
		f              func(g *Game, m *mods)
	}) {
		for _, it := range items {
			add(it.id, it.name, it.desc, it.rar, typ, it.mut, it.f)
		}
	}
	type item = struct {
		id, name, desc string
		rar            int
		mut            int
		f              func(g *Game, m *mods)
	}
	ladder(tLuck, []item{
		{"clover", "Clover", "x1.2 luck", rCommon, 0, func(g *Game, m *mods) { m.luck *= 1.2 }},
		{"horseshoe", "Horseshoe", "x1.4 luck", rUncommon, mGold, func(g *Game, m *mods) { m.luck *= 1.4 }},
		{"fourleaf", "Four Leaf", "x1.7 luck", rRare, 0, func(g *Game, m *mods) { m.luck *= 1.7 }},
		{"rabbit", "Rabbit Foot", "x2.2 luck", rEpic, mHeart, func(g *Game, m *mods) { m.luck *= 2.2 }},
		{"mutagen", "Mutagen", "x3 luck", rLegendary, mToxic, func(g *Game, m *mods) { m.luck *= 3 }},
	})
	ladder(tValue, []item{
		{"purse", "Coin Purse", "+20% value", rCommon, mGold, func(g *Game, m *mods) { m.value *= 1.2 }},
		{"piggy", "Piggy Bank", "+50% value", rUncommon, 0, func(g *Game, m *mods) { m.value *= 1.5 }},
		{"vault", "Vault", "x2 value", rRare, mGold, func(g *Game, m *mods) { m.value *= 2 }},
		{"treasury", "Treasury", "x3 value", rEpic, mGold, func(g *Game, m *mods) { m.value *= 3 }},
		{"midas", "Midas", "x5 value", rLegendary, mGold, func(g *Game, m *mods) { m.value *= 5 }},
		{"plainjane", "Plain Jane", "Plain biotes x3", rUncommon, 0, func(g *Game, m *mods) { m.plain *= 3 }},
	})
	ladder(tSpeed, []item{
		{"fert", "Fertilizer", "-5% split time", rCommon, 0, func(g *Game, m *mods) { m.cool *= 0.95 }},
		{"serum", "Growth Serum", "-10% split time", rUncommon, mToxic, func(g *Game, m *mods) { m.cool *= 0.9 }},
		{"mitosis", "Mitosis", "-15% split time", rRare, 0, func(g *Game, m *mods) { m.cool *= 0.85 }},
		{"overclock", "Overclock", "-20% split time", rEpic, mFrost, func(g *Game, m *mods) { m.cool *= 0.8 }},
	})
	ladder(tTime, []item{
		{"sunscreen", "Sunscreen", "+30% value by day", rCommon, 0, func(g *Game, m *mods) {
			if !g.night() {
				m.value *= 1.3
			}
		}},
		{"owl", "Night Owl", "x1.5 luck at night", rUncommon, mShadow, func(g *Game, m *mods) {
			if g.night() {
				m.luck *= 1.5
			}
		}},
		{"solar", "Solar Panel", "x1.6 luck by day", rRare, mGold, func(g *Game, m *mods) {
			if !g.night() {
				m.luck *= 1.6
			}
		}},
		{"moonstone", "Moonstone", "x2 value at night", rRare, mShadow, func(g *Game, m *mods) {
			if g.night() {
				m.value *= 2
			}
		}},
		{"eclipseheart", "Eclipse Heart", "x2 luck and value, day and night", rEpic, mCosmic, func(g *Game, m *mods) {
			m.luck *= 2
			m.value *= 2
		}},
	})
	ladder(tStorm, []item{
		{"vane", "Weather Vane", "Events last 25% longer", rCommon, 0, func(g *Game, m *mods) { m.evLen *= 1.25 }},
		{"storm", "Storm Chaser", "Events twice as strong", rUncommon, mFrost, func(g *Game, m *mods) { m.evPow *= 2 }},
		{"rod", "Lightning Rod", "Events come 50% more often", rRare, mFrost, func(g *Game, m *mods) { m.evFreq *= 1.5 }},
		{"eye", "Eye of the Storm", "x2 value during events", rEpic, mRainbow, func(g *Game, m *mods) {
			if g.st.Event != "" {
				m.value *= 2
			}
		}},
	})
	ladder(tClimb, []item{
		{"harvest", "Extra Harvest", "+1 sell in every room", rCommon, 0, func(g *Game, m *mods) { m.sells++ }},
		{"secondwind", "Second Wind", "+1 sell in every room", rUncommon, 0, func(g *Game, m *mods) { m.sells++ }},
		{"shortcut", "Shortcut", "Room money targets -10%", rRare, 0, func(g *Game, m *mods) { m.target *= 0.9 }},
		{"siege", "Siege Engine", "Bosses have no twist", rEpic, mShadow, func(g *Game, m *mods) { m.noBoss = true }},
		{"ladder", "Golden Ladder", "Room money targets -25%", rLegendary, mGold, func(g *Game, m *mods) { m.target *= 0.75 }},
	})
	ladder(tCombo, []item{
		{"crowd", "Crowd", "+1% value per biote", rUncommon, 0, func(g *Game, m *mods) { m.crowd += 0.01 }},
		{"hybrid", "Hybrid", "x3 value with 2+ mutations", rRare, mRainbow, func(g *Game, m *mods) { m.hybrid *= 3 }},
		{"jackpot", "Jackpot", "5% chance a sale pays x10", rRare, mGold, func(g *Game, m *mods) { m.jackpot += 0.05 }},
		{"chimera", "Chimera", "x2 more value for 3+ mutations", rEpic, mCosmic, func(g *Game, m *mods) { m.chimera *= 2 }},
		{"megajackpot", "Mega Jackpot", "10% chance a sale pays x10", rLegendary, mGold, func(g *Game, m *mods) { m.jackpot += 0.1 }},
	})
	ladder(tSpread, []item{
		{"contagion", "Contagion", "All mutations spread x2", rRare, mToxic, func(g *Game, m *mods) { m.spread *= 2 }},
		{"pandemic", "Pandemic", "All mutations spread x3", rLegendary, mToxic, func(g *Game, m *mods) { m.spread *= 3 }},
	})
	ladder(tIdle, []item{
		{"nightshift", "Night Shift", "+25% gains while away", rUncommon, mShadow, func(g *Game, m *mods) { m.offline *= 1.25 }},
		{"dreamer", "Dreamer", "+50% gains while away", rRare, mCosmic, func(g *Game, m *mods) { m.offline *= 1.5 }},
	})
	return out
}

// cardsOf lists the cards of one rarity, in table order.
func cardsOf(rar int) []int {
	var out []int
	for i, c := range cards {
		if c.rar == rar {
			out = append(out, i)
		}
	}
	return out
}

func cardByID(id string) *cardDef {
	for i := range cards {
		if cards[i].id == id {
			return &cards[i]
		}
	}
	return nil
}

// oldCards maps cards from before the big card update to their new versions.
var oldCards = map[string]string{
	"tooth": "gold_charm", "shade": "shadow_lure", "frostbite": "frost_lure", "heart": "heart_lure",
	"toxic": "toxic_lure", "prism": "rainbow_lure", "stargazer": "cosmic_lure",
}
