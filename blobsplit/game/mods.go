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
	target  float64       // multiplies room money targets
	notes   []string      // what is in effect, for the world panel

	spread    float64       // multiplies how often neighbours pass mutations on
	spreadMut [nMut]float64 // the same, for one mutation
	chimera   float64       // value multiplier for biotes with 3+ mutations
	offline   float64       // multiplies gains while the game is closed
	evLen     float64       // multiplies how long events last
	evFreq    float64       // multiplies how often events come
	evPow     float64       // how strong events are (1 normal)
	noBoss    bool          // bosses have no twist
}

func newMods() mods {
	m := mods{luck: 1, value: 1, cool: 1, hybrid: 1, rolls: 1, inherit: true, genes: 1, plain: 1, target: 1,
		spread: 1, chimera: 1, offline: 1, evLen: 1, evFreq: 1, evPow: 1}
	for i := range m.chance {
		m.chance[i], m.worth[i], m.spreadMut[i] = 1, 1, 1
	}
	return m
}

// ---- ability cards ----

func (g *Game) hasCard(id string) bool {
	for _, c := range g.st.Cards {
		if c == id {
			return true
		}
	}
	return false
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
	{"GENES", 0xff7ab8}, {"CHALLENGE", 0x6a8cff}, {"WEALTH", 0xffc94a},
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
	{"auto", "Fast Auto Split", "AUTO splits as fast as you can (x2 slower without)", 2, "quick1", -2, 0, 2},
	{"quick2", "Quick II", "-15% split time", 2, "auto", -3, 0, 2},
	{"autosell", "Fast Auto Sell", "AUTO sells as soon as it's full (x2 slower without)", 3, "quick2", -4, 0, 2},
	// genes: down and left
	{"genes1", "Strong Genes", "Mutation multipliers x1.25", 1, "origin", -1, 1, 3},
	{"bank", "Gene Bank", "+50% DNA from rebirths", 2, "genes1", -2, 2, 3},
	{"pure", "Purebred", "Plain biotes worth x3", 2, "bank", -3, 2, 3},
	{"genes2", "Dominant Genes", "Mutation multipliers x1.5", 4, "pure", -4, 2, 3},
	// cards: down and right
	{"slot1", "Climber", "Room money targets -15%", 1, "origin", 0, 1, 4},
	{"market", "Lucky Draw", "Rarer cards from floors", 2, "slot1", 0, 2, 4},
	{"slot2", "Endurance", "+1 sell in every room", 3, "market", 1, 2, 4},
	{"reroll", "Bounty", "+1 token every rebirth", 2, "slot2", 2, 2, 4},
	// wealth: right
	{"value1", "Value I", "x1.5 biote value", 1, "origin", 1, 0, 5},
	{"value2", "Value II", "x2 biote value", 2, "value1", 2, 0, 5},
	{"harvest", "Harvest", "+1 sell in every room", 2, "value2", 3, 0, 5},
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
// visible says whether a skill shows in the tree: once you own it, or own the skill
// it needs, so each branch grows outwards as you unlock it.
func (g *Game) visible(n *node) bool {
	return n.req == "" || g.has(n.id) || g.has(n.req)
}

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
	l := 45 * g.m.evLen
	if g.has("longev") {
		l *= 2
	}
	return l
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
	speed := g.m.evFreq
	if g.has("magnet") {
		speed *= 2
	}
	if s.NextEvent -= dt * speed; s.NextEvent > 0 {
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

// boss is the twist in play now: only in a floor's last room.
func (g *Game) boss() *bossDef {
	if g.st.Room != roomsPerFloor-1 {
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
	if g.has("slot1") {
		m.target *= 0.85
	}
	if g.has("slot2") {
		m.sells++
	}
	// the DNA lab: each mutation's value, how common it is, how far it spreads
	for r := 1; r < nMut; r++ {
		m.worth[r] *= 1 + 0.25*float64(g.lab(labValue, r))
		m.chance[r] *= 1 + 0.2*float64(g.lab(labChance, r))
		m.spreadMut[r] *= 1 + 0.5*float64(g.lab(labSpread, r))
	}
	// cards
	for _, id := range g.st.Cards {
		if c := cardByID(id); c != nil && c.apply != nil {
			c.apply(g, &m)
		}
	}
	// shop boosts and visitors' buffs
	if g.st.Perks["xluck"] {
		m.luck *= 1.5
	}
	if g.buff("cash") {
		m.value *= 2
		add("DOUBLE CASH: biote value x2")
	}
	if g.buff("luck") {
		m.luck *= 3
		add("LUCKY STAR: mutation chance x3")
	}
	if g.buff("rush") {
		m.cool *= 0.5
		add("SPLIT RUSH: splits 2x faster")
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
		e.apply(&m, m.evPow)
		add(e.name + ": " + e.desc)
	}
	// the boss
	if b := g.boss(); b != nil && !m.noBoss {
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
