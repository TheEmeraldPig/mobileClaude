package game

import (
	"math"
	"math/rand"
)

// mods is everything that changes the rules right now: skill tree nodes, ability
// cards, day or night, the current event and the boss. It is worked out once a frame.
type mods struct {
	luck    float64       // multiplies every mutation chance
	chance  [nMut]float64 // multiplies one mutation's chance
	worth   [nMut]float64 // multiplies one mutation's value multiplier
	value   float64       // multiplies all biote value
	cool    float64       // multiplies the split time
	sells   int           // extra sells per round
	jackpot float64       // chance a sale pays x10
	crowd   float64       // +value per biote on the board, as a fraction
	hybrid  float64       // value multiplier for biotes with 2+ mutations
	rolls   int           // times each new biote rolls for mutations
	inherit bool          // children copy their parent's mutations
	genes   float64       // multiplies every mutation's value multiplier
	plain   float64       // multiplies the value of biotes with no mutations
	notes   []string      // what is in effect, for the world panel
}

func newMods() mods {
	m := mods{luck: 1, value: 1, cool: 1, hybrid: 1, rolls: 1, inherit: true, genes: 1, plain: 1}
	for i := range m.chance {
		m.chance[i], m.worth[i] = 1, 1
	}
	return m
}

// ---- ability cards ----

var cardRarities = []struct {
	name  string
	sk    skin
	price int
	odds  float64 // weight in the market and for boss rewards
}{
	{"COMMON", skin{rgb(0x5a6a8a), rgb(0x8a9aba), rgb(0x33405a)}, 3, 60},
	{"UNCOMMON", skin{rgb(0x2f9a4a), rgb(0x6fe07a), rgb(0x1a5a2a)}, 6, 28},
	{"RARE", skin{rgb(0x3a6ae0), rgb(0x8ab0ff), rgb(0x203a8a)}, 10, 10},
	{"LEGENDARY", skin{rgb(0xe09a20), rgb(0xffe08a), rgb(0x8a5010)}, 18, 2},
}

type cardDef struct {
	id, name, desc string
	rar            int
	mut            int // mutation shown on the card (0 = a plain biote)
	apply          func(g *Game, m *mods)
}

var cards = []cardDef{
	{"clover", "Clover", "+30% luck", 0, 0, func(g *Game, m *mods) { m.luck *= 1.3 }},
	{"fert", "Fertilizer", "-10% split time", 0, 0, func(g *Game, m *mods) { m.cool *= 0.9 }},
	{"purse", "Coin Purse", "+25% sale value", 0, mGold, func(g *Game, m *mods) { m.value *= 1.25 }},
	{"harvest", "Extra Harvest", "+1 sell each round", 0, 0, func(g *Game, m *mods) { m.sells++ }},
	{"tooth", "Gold Tooth", "Gold x2 chance", 0, mGold, func(g *Game, m *mods) { m.chance[mGold] *= 2 }},
	{"sunscreen", "Sunscreen", "+50% value by day", 0, 0, func(g *Game, m *mods) {
		if !g.night() {
			m.value *= 1.5
		}
	}},
	{"owl", "Night Owl", "x2 luck at night", 1, mShadow, func(g *Game, m *mods) {
		if g.night() {
			m.luck *= 2
		}
	}},
	{"shade", "Shade", "Shadow x3 chance, x2 value", 1, mShadow, func(g *Game, m *mods) {
		m.chance[mShadow] *= 3
		m.worth[mShadow] *= 2
	}},
	{"frostbite", "Frostbite", "Frost x3 chance", 1, mFrost, func(g *Game, m *mods) { m.chance[mFrost] *= 3 }},
	{"crowd", "Crowd", "+1% value per biote", 1, 0, func(g *Game, m *mods) { m.crowd += 0.01 }},
	{"mitosis", "Mitosis", "-25% split time", 1, 0, func(g *Game, m *mods) { m.cool *= 0.75 }},
	{"storm", "Storm Chaser", "Events twice as strong", 1, mFrost, nil}, // read in eventMods
	{"heart", "Heartthrob", "Heart x3 chance, x2 value", 2, mHeart, func(g *Game, m *mods) {
		m.chance[mHeart] *= 3
		m.worth[mHeart] *= 2
	}},
	{"toxic", "Toxic Waste", "Toxic x4 chance", 2, mToxic, func(g *Game, m *mods) { m.chance[mToxic] *= 4 }},
	{"jackpot", "Jackpot", "5% chance a sale pays x10", 2, mGold, func(g *Game, m *mods) { m.jackpot += 0.05 }},
	{"hybrid", "Hybrid", "x3 value with 2+ mutations", 2, mRainbow, func(g *Game, m *mods) { m.hybrid *= 3 }},
	{"prism", "Prism", "Rainbow x4 chance, x2 value", 3, mRainbow, func(g *Game, m *mods) {
		m.chance[mRainbow] *= 4
		m.worth[mRainbow] *= 2
	}},
	{"stargazer", "Stargazer", "Cosmic x5 chance at night", 3, mCosmic, func(g *Game, m *mods) {
		if g.night() {
			m.chance[mCosmic] *= 5
		}
	}},
	{"midas", "Midas", "x3 all value", 3, mGold, func(g *Game, m *mods) { m.value *= 3 }},
	{"mutagen", "Mutagen", "x3 luck", 3, mToxic, func(g *Game, m *mods) { m.luck *= 3 }},
}

func cardByID(id string) *cardDef {
	for i := range cards {
		if cards[i].id == id {
			return &cards[i]
		}
	}
	return nil
}

func (g *Game) hasCard(id string) bool {
	for _, c := range g.st.Cards {
		if c == id {
			return true
		}
	}
	return false
}

// randomCard picks a card: first a rarity by weight (the Better Market node shifts
// the odds to rarer), then a card of that rarity.
func (g *Game) randomCard() string {
	w := make([]float64, len(cardRarities))
	total := 0.0
	for i, r := range cardRarities {
		w[i] = r.odds
		if g.has("market") && i > 0 {
			w[i] *= 2
		}
		total += w[i]
	}
	x := rand.Float64() * total
	rar := 0
	for i := range w {
		if x -= w[i]; x < 0 {
			rar = i
			break
		}
	}
	var pool []string
	for _, c := range cards {
		if c.rar == rar {
			pool = append(pool, c.id)
		}
	}
	return pool[rand.Intn(len(pool))]
}

func (g *Game) cardSlots() int {
	n := 3
	if g.has("slot1") {
		n++
	}
	if g.has("slot2") {
		n++
	}
	return n
}

// ---- skill tree: bought with points from rebirths, kept forever ----

// Skills sit on a hexagon grid (axial coordinates q, r) in a flower around Origin:
// six branches, one per direction, each bending outwards after two steps.
type node struct {
	id, name, desc string
	cost           int
	req            string
	q, r           int
	branch         int // index into branches; -1 for Origin
}

var branches = []struct {
	name string
	col  uint32
}{
	{"LUCK", 0x5fd86a}, {"NATURE", 0xb07cff}, {"SPEED", 0x5fd0ff},
	{"GENES", 0xff7ab8}, {"CARDS", 0x6a8cff}, {"WEALTH", 0xffc94a},
}

var tree = []node{
	{"origin", "Origin", "+25% biote value", 1, "", 0, 0, -1},
	// luck: up and right
	{"luck1", "Lucky I", "+25% luck", 1, "origin", 1, -1, 0},
	{"luck2", "Lucky II", "+50% luck", 2, "luck1", 2, -2, 0},
	{"luck3", "Lucky III", "x2 luck", 3, "luck2", 3, -2, 0},
	{"helix", "Double Helix", "New biotes roll mutations twice", 5, "luck3", 4, -2, 0},
	// nature: up and left
	{"nightv", "Night Vision", "Night bonuses x2", 1, "origin", 0, -1, 1},
	{"magnet", "Storm Magnet", "Events come twice as often", 2, "nightv", 0, -2, 1},
	{"longev", "Long Events", "Events last twice as long", 2, "magnet", -1, -2, 1},
	{"sun", "Sun Worship", "Day bonuses x2", 3, "longev", -2, -2, 1},
	// speed: left
	{"quick1", "Quick I", "-10% split time", 1, "origin", -1, 0, 2},
	{"auto", "Auto Split", "Splits for you (AUTO button)", 2, "quick1", -2, 0, 2},
	{"quick2", "Quick II", "-15% split time", 2, "auto", -3, 0, 2},
	{"autosell", "Auto Sell", "Sells when full (AUTO on SELL)", 3, "quick2", -4, 0, 2},
	// genes: down and left
	{"genes1", "Strong Genes", "Mutation multipliers x1.25", 1, "origin", -1, 1, 3},
	{"bank", "DNA Bank", "+1 DNA for every round cleared", 2, "genes1", -2, 2, 3},
	{"pure", "Purebred", "Plain biotes worth x3", 2, "bank", -3, 2, 3},
	{"genes2", "Dominant Genes", "Mutation multipliers x1.5", 4, "pure", -4, 2, 3},
	// cards: down and right
	{"slot1", "Card Slot", "+1 card slot", 1, "origin", 0, 1, 4},
	{"market", "Better Market", "Rarer cards in the market", 2, "slot1", 0, 2, 4},
	{"slot2", "Card Slot II", "+1 card slot", 3, "market", 1, 2, 4},
	{"reroll", "Free Reroll", "First market reroll is free", 2, "slot2", 2, 2, 4},
	// wealth: right
	{"value1", "Value I", "x1.5 biote value", 1, "origin", 1, 0, 5},
	{"value2", "Value II", "x2 biote value", 2, "value1", 2, 0, 5},
	{"harvest", "Harvest", "+1 sell each round", 2, "value2", 3, 0, 5},
	{"value3", "Value III", "x3 biote value", 4, "harvest", 4, 0, 5},
}

func (g *Game) has(id string) bool { return g.st.Tree[id] }

func nodeByID(id string) *node {
	for i := range tree {
		if tree[i].id == id {
			return &tree[i]
		}
	}
	return nil
}

// canUnlock says whether node n is next in its branch.
func (g *Game) canUnlock(n *node) bool {
	return !g.has(n.id) && (n.req == "" || g.has(n.req))
}

// ---- day and night ----

const dayLen = 240.0 // seconds for a whole day and night

// daylight is 1 at midday and 0 at night, with a short dawn and dusk between.
func (g *Game) daylight() float64 {
	s := math.Sin(2 * math.Pi * g.st.Clock / dayLen)
	return math.Max(0, math.Min(1, 0.5+1.6*s))
}

func (g *Game) night() bool { return g.daylight() < 0.5 }

// ---- events ----

type eventDef struct {
	id, name, desc string
	when           int // 0 any time, 1 day only, 2 night only
	apply          func(m *mods, k float64)
}

// k is the event's strength: 1, or 2 with the Storm Chaser card. A "x4" becomes x7.
func boost(x, k float64) float64 { return 1 + (x-1)*k }

var events = []eventDef{
	{"meteor", "Meteor Shower", "Cosmic x5", 2, func(m *mods, k float64) { m.chance[mCosmic] *= boost(5, k) }},
	{"blizzard", "Blizzard", "Frost x5", 0, func(m *mods, k float64) { m.chance[mFrost] *= boost(5, k) }},
	{"goldrush", "Gold Rush", "Gold x4, value x1.5", 1, func(m *mods, k float64) {
		m.chance[mGold] *= boost(4, k)
		m.value *= boost(1.5, k)
	}},
	{"aurora", "Aurora", "Rainbow x4", 2, func(m *mods, k float64) { m.chance[mRainbow] *= boost(4, k) }},
	{"bloom", "Love Bloom", "Heart x4", 1, func(m *mods, k float64) { m.chance[mHeart] *= boost(4, k) }},
	{"heatwave", "Heatwave", "Toxic x4, no Frost", 1, func(m *mods, k float64) {
		m.chance[mToxic] *= boost(4, k)
		m.chance[mFrost] = 0
	}},
	{"eclipse", "Eclipse", "All mutations x2", 0, func(m *mods, k float64) { m.luck *= boost(2, k) }},
}

func eventByID(id string) *eventDef {
	for i := range events {
		if events[i].id == id {
			return &events[i]
		}
	}
	return nil
}

func (g *Game) eventLength() float64 {
	if g.has("longev") {
		return 90
	}
	return 45
}

// tickEvents runs the event clock: when one ends, the next comes 2-4 minutes later
// (half that with Storm Magnet), picked from the events that fit the time of day.
func (g *Game) tickEvents() {
	s := &g.st
	s.Clock += dt
	if s.Event != "" {
		if s.EventLeft -= dt; s.EventLeft <= 0 {
			s.Event = ""
			s.NextEvent = 120 + rand.Float64()*120
		}
		return
	}
	if g.has("magnet") {
		s.NextEvent -= dt
	}
	if s.NextEvent -= dt; s.NextEvent > 0 {
		return
	}
	var pool []string
	for _, e := range events {
		if e.when == 0 || (e.when == 1) == !g.night() {
			pool = append(pool, e.id)
		}
	}
	g.startEvent(pool[rand.Intn(len(pool))])
}

func (g *Game) startEvent(id string) {
	g.st.Event, g.st.EventLeft = id, g.eventLength()
	g.eventT = g.t
	g.play("rare")
}

// ---- bosses: the Outbreak round's twist ----

type bossDef struct{ id, name, desc string }

var bosses = []bossDef{
	{"parasite", "The Parasite", "Biote value halved"},
	{"drought", "The Drought", "No new mutations"},
	{"freeze", "The Freeze", "Split time doubled"},
	{"leech", "The Leech", "Only 2 sells"},
	{"mirror", "The Mirror", "Children don't inherit mutations"},
}

// boss is the twist in play now (nil outside an Outbreak round).
func (g *Game) boss() *bossDef {
	if !g.st.Active || g.st.Round != 2 {
		return nil
	}
	return &bosses[g.st.Boss%len(bosses)]
}

// ---- putting it together ----

func (g *Game) computeMods() mods {
	m := newMods()
	add := func(s string) { m.notes = append(m.notes, s) }
	// skill tree
	if g.has("origin") {
		m.value *= 1.25
	}
	if g.has("luck1") {
		m.luck *= 1.25
	}
	if g.has("luck2") {
		m.luck *= 1.5
	}
	if g.has("luck3") {
		m.luck *= 2
	}
	if g.has("helix") {
		m.rolls = 2
	}
	if g.has("quick1") {
		m.cool *= 0.9
	}
	if g.has("quick2") {
		m.cool *= 0.85
	}
	for id, x := range map[string]float64{"value1": 1.5, "value2": 2, "value3": 3} {
		if g.has(id) {
			m.value *= x
		}
	}
	if g.has("harvest") {
		m.sells++
	}
	if g.has("genes1") {
		m.genes *= 1.25
	}
	if g.has("genes2") {
		m.genes *= 1.5
	}
	if g.has("pure") {
		m.plain *= 3
	}
	// cards
	for _, id := range g.st.Cards {
		if c := cardByID(id); c != nil && c.apply != nil {
			c.apply(g, &m)
		}
	}
	// day and night
	if g.night() {
		k := 1.0
		if g.has("nightv") {
			k = 2
		}
		m.chance[mShadow] *= boost(2, k)
		m.chance[mCosmic] *= boost(2, k)
		add("NIGHT: Shadow & Cosmic x" + sprintf1(boost(2, k)))
	} else {
		k := 1.0
		if g.has("sun") {
			k = 2
		}
		m.chance[mGold] *= boost(1.5, k)
		m.chance[mHeart] *= boost(1.5, k)
		add("DAY: Gold & Heart x" + sprintf1(boost(1.5, k)))
	}
	// the current event
	if e := eventByID(g.st.Event); e != nil {
		k := 1.0
		if g.hasCard("storm") {
			k = 2
		}
		e.apply(&m, k)
		add(e.name + ": " + e.desc)
	}
	// the boss
	if b := g.boss(); b != nil {
		switch b.id {
		case "parasite":
			m.value *= 0.5
		case "drought":
			m.luck = 0
		case "freeze":
			m.cool *= 2
		case "leech":
			m.sells -= 2
		case "mirror":
			m.inherit = false
		}
		add(b.name + ": " + b.desc)
	}
	return m
}
