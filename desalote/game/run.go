package game

import (
	"math"
	"math/rand"
)

// A player's own run: their locked 13-card deck, the hand they hold, and playing a
// round (blind). Each game runs its own player's run (and the host runs its bots');
// only scores and choices travel over the network.

const (
	baseHandSize = 7
	basePlays    = 4
	baseDiscards = 3
	specialBuff  = 3 // permanent points a card gets each time a special hand uses it
)

// What the run knows about the rest of the table (for charms)
type TableInfo struct {
	Underdog       bool
	HexesCast      int
	Others         int
	RivalLivesLost int
}

type Run struct {
	Deck    []Card
	Family  Family
	Special HandType
	Picks   []HandType
	Rules   handRules

	// set by the match
	Charms      []string
	Levels      map[HandType]int
	Hexes       []string // hexes on this player this round
	Frozen      string   // a charm switched off by Freeze
	Table       TableInfo
	HandsPlayed int

	// this round
	Pile, Hand, Used []int // indices into Deck
	Sel              []bool
	FaceDown         map[int]bool
	Plays, Discards  int
	HandSize         int
	SpecialLeft      int
	PlayNo           int
	Score            int64
	Target           int64 // 0: a showdown, play every hand
	Done             bool
	Last             *ScoreResult

	rng *rand.Rand
}

type ScoreResult struct {
	Hand    HandType
	Cards   []Card
	Scoring []int
	Base    int
	Mult    float64
	Total   int64
	Notes   []string
	Special bool
}

func NewRun(seed int64, fam Family, special HandType) *Run {
	r := &Run{Family: fam, Special: special, Levels: map[HandType]int{}, rng: rand.New(rand.NewSource(seed ^ 0x5eed))}
	r.Deck = DealDeck(seed, fam)
	r.Rules = handRules{flush: FlushSize(r.Deck), straight: StraightSize(r.Deck)}
	return r
}

func (r *Run) has(id string) bool {
	if id == r.Frozen {
		return false
	}
	for _, c := range r.Charms {
		if c == id {
			return true
		}
	}
	return false
}

func (r *Run) hexed(id string) bool {
	for _, h := range r.Hexes {
		if h == id {
			return true
		}
	}
	return false
}

// The hands this player can score
func (r *Run) Known() []HandType {
	return append([]HandType{HighCard, Pair}, r.Picks...)
}

// StartRound deals a fresh round against a target (0 for a showdown)
func (r *Run) StartRound(target int64) {
	r.Target, r.Score, r.Done, r.PlayNo, r.Last = target, 0, false, 0, nil
	r.HandSize, r.Plays, r.Discards, r.SpecialLeft = baseHandSize, basePlays, baseDiscards, 1
	for _, id := range r.Charms {
		if c := charms[id]; c != nil && r.has(id) {
			r.HandSize += c.handSize
			r.Plays += c.plays
			r.Discards += c.discards
		}
	}
	if r.has("sparkplug") {
		r.SpecialLeft = 2
	}
	if r.hexed("squeeze") {
		r.HandSize--
	}
	if r.hexed("shackle") {
		r.Plays = max(1, r.Plays-1)
	}
	if r.hexed("jinx") {
		r.SpecialLeft = 0
	}
	r.Pile, r.Hand, r.Used = r.rng.Perm(len(r.Deck)), nil, nil
	r.FaceDown = map[int]bool{}
	r.draw()
	if r.hexed("fog") {
		for i := 0; i < 3 && i < len(r.Hand); i++ {
			r.FaceDown[r.Hand[i]] = true
		}
	}
}

// Fill the hand up; when the pile runs out the used cards are shuffled back in (the deck
// cycles)
func (r *Run) draw() {
	for len(r.Hand) < r.HandSize {
		if len(r.Pile) == 0 {
			if len(r.Used) == 0 {
				break
			}
			r.Pile = r.Used
			r.Used = nil
			r.rng.Shuffle(len(r.Pile), func(i, j int) { r.Pile[i], r.Pile[j] = r.Pile[j], r.Pile[i] })
		}
		r.Hand = append(r.Hand, r.Pile[0])
		r.Pile = r.Pile[1:]
	}
	r.Sel = make([]bool, len(r.Hand))
	r.sortHand()
}

func (r *Run) sortHand() {
	sel := map[int]bool{}
	for i, d := range r.Hand {
		if i < len(r.Sel) && r.Sel[i] {
			sel[d] = true
		}
	}
	h := r.Hand
	for i := 1; i < len(h); i++ {
		for j := i; j > 0 && cardLess(r.Deck[h[j-1]], r.Deck[h[j]]); j-- {
			h[j], h[j-1] = h[j-1], h[j]
		}
	}
	r.Sel = make([]bool, len(h))
	for i, d := range h {
		r.Sel[i] = sel[d]
	}
}

func cardLess(a, b Card) bool {
	if a.R != b.R {
		return a.R < b.R
	}
	return a.S > b.S
}

func (r *Run) Selected() []int {
	var out []int
	for i, s := range r.Sel {
		if s {
			out = append(out, i)
		}
	}
	return out
}

func (r *Run) Toggle(i int) {
	if i < 0 || i >= len(r.Sel) {
		return
	}
	if !r.Sel[i] && len(r.Selected()) >= 5 {
		return
	}
	r.Sel[i] = !r.Sel[i]
}

func (r *Run) selectedCards() []Card {
	var cs []Card
	for _, i := range r.Selected() {
		cs = append(cs, r.Deck[r.Hand[i]])
	}
	return cs
}

// Preview: what the selected cards would make
func (r *Run) Preview() (HandType, bool) {
	cs := r.selectedCards()
	if len(cs) == 0 {
		return HighCard, false
	}
	return bestHand(cs, r.Known(), r.Levels, r.Rules).h, true
}

func (r *Run) CanPlay() bool    { return !r.Done && r.Plays > 0 && len(r.Selected()) > 0 }
func (r *Run) CanDiscard() bool { return !r.Done && r.Discards > 0 && len(r.Selected()) > 0 }
func (r *Run) CanSpecial() bool { return !r.Done && r.SpecialLeft > 0 && len(r.Hand) >= 5 }

// Play the selected cards
func (r *Run) Play() *ScoreResult {
	if !r.CanPlay() {
		return nil
	}
	cs := r.selectedCards()
	f := bestHand(cs, r.Known(), r.Levels, r.Rules)
	r.Plays--
	r.PlayNo++
	res := r.score(f.h, cs, f.cards, false)
	r.removeSelected()
	r.finishPlay(res)
	return res
}

func (r *Run) Discard() {
	if !r.CanDiscard() {
		return
	}
	r.Discards--
	r.removeSelected()
	r.draw()
}

func (r *Run) removeSelected() {
	var keep []int
	for i, d := range r.Hand {
		if r.Sel[i] {
			r.Used = append(r.Used, d)
			delete(r.FaceDown, d)
		} else {
			keep = append(keep, d)
		}
	}
	r.Hand = keep
	r.Sel = make([]bool, len(keep))
}

// Special: five cards from the hand, chosen by the game, briefly become the special hand
// and score it. It doesn't use a play. Afterwards they go back to normal, a little
// stronger for good.
func (r *Run) UseSpecial() *ScoreResult {
	if !r.CanSpecial() {
		return nil
	}
	r.SpecialLeft--
	pick := r.rng.Perm(len(r.Hand))[:5]
	var cs []Card
	for _, i := range pick {
		cs = append(cs, r.Deck[r.Hand[i]])
	}
	changed := r.transform(cs)
	res := r.score(r.Special, changed, allIdx(5), true)
	buff := specialBuff
	if r.has("afterglow") {
		buff *= 2
	}
	for i := range r.Sel {
		r.Sel[i] = false
	}
	for _, i := range pick {
		d := r.Hand[i]
		r.Deck[d].Bonus += buff
		r.Deck[d].Glow++
		r.Sel[i] = true
	}
	r.removeSelected()
	r.finishPlay(res)
	return res
}

// The five cards as the special hand (suits or ranks changed for this play only)
func (r *Run) transform(cs []Card) []Card {
	out := append([]Card(nil), cs...)
	mainSuit := func() int {
		n := suitCounts(cs)
		best := 0
		for s := 1; s < 4; s++ {
			if n[s] > n[best] {
				best = s
			}
		}
		return best
	}
	switch r.Special {
	case FiveKind:
		rank := mostCommonRank(r.Deck, r.rng)
		for i := range out {
			out[i].R = rank
		}
	case StraightFlush:
		s := mainSuit()
		top := 0
		for _, c := range cs {
			top = max(top, c.R)
		}
		top = max(top, 6)
		for i := range out {
			out[i].R, out[i].S = top-i, s
		}
	case FlushHouse:
		s := mainSuit()
		n := rankCounts(cs)
		a, b := 0, 0
		for rk := 14; rk >= 2; rk-- {
			if n[rk] > n[a] {
				a = rk
			}
		}
		for rk := 14; rk >= 2; rk-- {
			if rk != a && n[rk] > n[b] {
				b = rk
			}
		}
		if b == 0 {
			b = a - 1
			if b < 2 {
				b = 13
			}
		}
		for i := range out {
			out[i].S = s
			if i < 3 {
				out[i].R = a
			} else {
				out[i].R = b
			}
		}
	}
	return out
}

func (r *Run) finishPlay(res *ScoreResult) {
	r.HandsPlayed++
	r.Score += res.Total
	r.Last = res
	if r.Target > 0 && r.Score >= r.Target {
		r.Done = true
	}
	if r.Plays == 0 && !r.CanSpecial() {
		r.Done = true
	}
	if !r.Done {
		r.draw()
	}
}

// Stop playing this round (only once out of plays, leaving a special unused)
func (r *Run) Finish() { r.Done = true }

// score works out a hand's total with charms, hexes, levels and synergy
func (r *Run) score(h HandType, cs []Card, scoring []int, special bool) *ScoreResult {
	base, mult := handLevel(h, r.Levels[h]+1)
	c := &scoreCtx{run: r, hand: h, cards: cs, scoring: scoring, mult: float64(mult), xmult: 1,
		special: special, firstPlay: r.PlayNo <= 1 && !special, lastPlay: r.Plays == 0 && !special}
	c.base = float64(base)
	for k, i := range scoring {
		v := cs[i].Value()
		if r.hexed("rust") && cs[i].R > 10 {
			v = cs[i].Bonus
		}
		if r.has("lowtide") && cs[i].R <= 5 {
			v *= 3
		}
		c.base += float64(v)
		if k == 0 && r.has("echo") {
			c.base += float64(v)
		}
	}
	if r.hexed("muddle") && hands[h].fam >= 0 {
		c.mult = math.Max(1, c.mult-3)
		c.note("Muddled")
	}
	for _, id := range r.Charms {
		if d := charms[id]; d != nil && d.score != nil && r.has(id) {
			d.score(c)
		}
	}
	// synergy for the three picked hands
	if hands[h].fam >= 0 {
		g, m := synergyOf(r.Picks)
		if r.has("synergist") {
			m = 1 + (m-1)*2
		}
		if m > 1 {
			c.xmult *= m
			c.note("Synergy x%.1f (%d %s)", m, g, familyNames[majorFamily(r.Picks)])
		}
	}
	mlt := c.mult * c.xmult
	total := int64(math.Floor(c.base * mlt))
	return &ScoreResult{Hand: h, Cards: cs, Scoring: scoring, Base: int(c.base), Mult: mlt, Total: total, Notes: c.notes, Special: special}
}

func majorFamily(picks []HandType) Family {
	var n [3]int
	for _, h := range picks {
		if f := hands[h].fam; f >= 0 {
			n[f]++
		}
	}
	best := 0
	for f := 1; f < 3; f++ {
		if n[f] > n[best] {
			best = f
		}
	}
	return Family(best)
}

// ---- A simple player for bots (and tests) ----

// botTurn makes one move: uses the special first, then plays the best hand it can find,
// discarding when the best hand is weak and discards are left
func (r *Run) botTurn() *ScoreResult {
	if r.Done {
		return nil
	}
	if r.CanSpecial() && (r.PlayNo == 0 || r.Plays == 0) {
		return r.UseSpecial()
	}
	if r.Plays == 0 {
		r.Finish()
		return nil
	}
	best, bestSel := int64(-1), []int(nil)
	var bestH HandType
	n := len(r.Hand)
	// try every choice of up to 5 cards (7-8 cards: at most a few hundred)
	for mask := 1; mask < 1<<n; mask++ {
		if bitsSet(mask) > 5 {
			continue
		}
		var cs []Card
		var sel []int
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				cs = append(cs, r.Deck[r.Hand[i]])
				sel = append(sel, i)
			}
		}
		f := bestHand(cs, r.Known(), r.Levels, r.Rules)
		base, mult := handLevel(f.h, r.Levels[f.h]+1)
		for _, i := range f.cards {
			base += cs[i].Value()
		}
		s := int64(base * mult)
		if s > best {
			best, bestSel, bestH = s, sel, f.h
		}
	}
	for i := range r.Sel {
		r.Sel[i] = false
	}
	if bestH <= Pair && r.Discards > 0 && r.Plays > 1 {
		// throw away the cards that aren't helping
		k := 0
		keep := map[int]bool{}
		for _, i := range bestSel {
			keep[i] = true
		}
		for i := range r.Hand {
			if !keep[i] && k < 5 {
				r.Sel[i] = true
				k++
			}
		}
		if k > 0 {
			r.Discard()
			return nil
		}
	}
	for _, i := range bestSel {
		r.Sel[i] = true
	}
	return r.Play()
}

func bitsSet(m int) int {
	n := 0
	for ; m > 0; m &= m - 1 {
		n++
	}
	return n
}
