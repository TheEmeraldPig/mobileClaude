package game

import "sort"

// Poker hands, grouped in three families. Every player can always score High Card and
// Pair. At the start of a match each player is given a family and a special hand, and
// picks three hands to play for the whole match (at least one from their own family);
// any other hand they play scores as the best of the hands they do have. The more of
// the three picks share a family, the more all three score (synergy). High Card and
// Pair never count toward synergy.

type Family int

const (
	FamSame     Family = iota // ranks that match
	FamCounting               // ranks in a row
	FamSuits                  // suits
)

var familyNames = [3]string{"Same", "Counting", "Suits"}
var familyBlurb = [3]string{
	"Matching ranks. Your deck always has at least 3 cards of one rank.",
	"Ranks in a row. Straights are as long as your deck's average run.",
	"Matching suits. Flushes are as big as your deck's average suit.",
}

type HandType int

const (
	HighCard HandType = iota
	Pair
	TwoPair
	ThreeKind
	FullHouse
	FourKind
	Straight
	SkipStraight
	Ladder
	Flush
	TwoTone
	Rainbow
	FiveKind // specials
	StraightFlush
	FlushHouse
	numHands
)

type handInfo struct {
	name       string
	fam        Family // -1: everyone's (High Card, Pair) or a special
	base, mult int    // at level 1
	lvBase     int    // per level after the first
	lvMult     int
	how        string
}

var hands = [numHands]handInfo{
	HighCard:      {"High Card", -1, 5, 1, 10, 1, "Your best card"},
	Pair:          {"Pair", -1, 10, 2, 15, 1, "2 cards of one rank"},
	TwoPair:       {"Two Pair", FamSame, 20, 2, 20, 1, "2 pairs"},
	ThreeKind:     {"Three of a Kind", FamSame, 30, 3, 20, 2, "3 cards of one rank"},
	FullHouse:     {"Full House", FamSame, 40, 4, 25, 2, "3 of one rank and 2 of another"},
	FourKind:      {"Four of a Kind", FamSame, 60, 7, 30, 3, "4 cards of one rank"},
	Straight:      {"Straight", FamCounting, 30, 4, 30, 3, "%S ranks in a row"},
	SkipStraight:  {"Skip Straight", FamCounting, 35, 4, 30, 3, "%S ranks stepping by 2 (3-5-7)"},
	Ladder:        {"Ladder", FamCounting, 30, 3, 25, 2, "Pairs of 2 ranks in a row (7 7 8 8)"},
	Flush:         {"Flush", FamSuits, 35, 4, 15, 2, "%F cards of one suit"},
	TwoTone:       {"Two Tone", FamSuits, 20, 2, 15, 1, "4+ cards of exactly 2 suits, 2+ of each"},
	Rainbow:       {"Rainbow", FamSuits, 15, 2, 15, 1, "4 cards of 4 different suits"},
	FiveKind:      {"Five of a Kind", -1, 45, 5, 0, 0, "5 cards become your deck's most common rank"},
	StraightFlush: {"Straight Flush", -1, 60, 6, 0, 0, "5 cards become a straight in one suit"},
	FlushHouse:    {"Flush House", -1, 65, 6, 0, 0, "5 cards become a full house in one suit"},
}

var specialHands = []HandType{FiveKind, StraightFlush, FlushHouse}

func familyHands(f Family) []HandType {
	var out []HandType
	for h := TwoPair; h < FiveKind; h++ {
		if hands[h].fam == f {
			out = append(out, h)
		}
	}
	return out
}

// Synergy: the most of the three picked hands that share a family -> a multiplier for
// all of them
func synergyOf(picks []HandType) (group int, mult float64) {
	var n [3]int
	for _, h := range picks {
		if f := hands[h].fam; f >= 0 {
			n[f]++
		}
	}
	group = max(n[0], max(n[1], n[2]))
	switch group {
	case 3:
		return 3, 1.5
	case 2:
		return 2, 1.2
	}
	return group, 1
}

// The rules a deck plays by
type handRules struct {
	flush, straight int
}

// A recognised hand among the played cards: which cards score
type madeHand struct {
	h     HandType
	cards []int // indices into the played cards
}

// detect reports which cards make hand h, or ok=false
func detect(h HandType, cs []Card, rules handRules) ([]int, bool) {
	byRank := map[int][]int{}
	bySuit := map[int][]int{}
	for i, c := range cs {
		byRank[c.R] = append(byRank[c.R], i)
		bySuit[c.S] = append(bySuit[c.S], i)
	}
	// ranks with at least n cards, highest first
	ranksWith := func(n int) []int {
		var out []int
		for r, ix := range byRank {
			if len(ix) >= n {
				out = append(out, r)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(out)))
		return out
	}
	switch h {
	case HighCard:
		if len(cs) == 0 {
			return nil, false
		}
		best := 0
		for i, c := range cs {
			if c.R > cs[best].R {
				best = i
			}
		}
		return []int{best}, true
	case Pair:
		if r := ranksWith(2); len(r) > 0 {
			return byRank[r[0]][:2], true
		}
	case TwoPair:
		if r := ranksWith(2); len(r) >= 2 {
			return append(append([]int{}, byRank[r[0]][:2]...), byRank[r[1]][:2]...), true
		}
	case ThreeKind:
		if r := ranksWith(3); len(r) > 0 {
			return byRank[r[0]][:3], true
		}
	case FourKind:
		if r := ranksWith(4); len(r) > 0 {
			return byRank[r[0]][:4], true
		}
	case FullHouse:
		for _, t := range ranksWith(3) {
			for _, p := range ranksWith(2) {
				if p != t {
					return append(append([]int{}, byRank[t][:3]...), byRank[p][:2]...), true
				}
			}
		}
	case Flush:
		for s := 0; s < 4; s++ {
			if len(bySuit[s]) >= rules.flush {
				return bySuit[s], true
			}
		}
	case Straight, SkipStraight:
		step := 1
		if h == SkipStraight {
			step = 2
		}
		return findRun(cs, byRank, step, rules.straight)
	case Ladder:
		for _, r := range ranksWith(2) {
			if len(byRank[r+1]) >= 2 {
				return append(append([]int{}, byRank[r+1][:2]...), byRank[r][:2]...), true
			}
			if r == 14 && len(byRank[2]) >= 2 { // A A 2 2
				return append(append([]int{}, byRank[2][:2]...), byRank[14][:2]...), true
			}
		}
	case TwoTone:
		if len(cs) < 4 {
			return nil, false
		}
		used := 0
		for s := 0; s < 4; s++ {
			if n := len(bySuit[s]); n > 0 {
				if n < 2 {
					return nil, false
				}
				used++
			}
		}
		if used == 2 {
			return allIdx(len(cs)), true
		}
	case Rainbow:
		if len(bySuit) == 4 {
			return []int{bySuit[0][0], bySuit[1][0], bySuit[2][0], bySuit[3][0]}, true
		}
	}
	return nil, false
}

// Look for n ranks stepping by step (1 for a straight); an ace can sit below the 2
func findRun(cs []Card, byRank map[int][]int, step, n int) ([]int, bool) {
	at := func(r int) []int {
		if r == 1 {
			return byRank[14]
		}
		return byRank[r]
	}
	for top := 14; top >= 1+(n-1)*step; top-- {
		var ix []int
		for k := 0; k < n; k++ {
			l := at(top - k*step)
			if len(l) == 0 {
				ix = nil
				break
			}
			ix = append(ix, l[0])
		}
		if ix != nil {
			return ix, true
		}
	}
	return nil, false
}

func allIdx(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// The hand's points and mult at a level
func handLevel(h HandType, level int) (base, mult int) {
	hi := hands[h]
	l := max(0, level-1)
	return hi.base + hi.lvBase*l, hi.mult + hi.lvMult*l
}

// bestHand picks the known hand that scores most for the played cards (before charms)
func bestHand(cs []Card, known []HandType, levels map[HandType]int, rules handRules) madeHand {
	best := madeHand{h: HighCard}
	bestScore := -1
	for _, h := range known {
		ix, ok := detect(h, cs, rules)
		if !ok {
			continue
		}
		base, mult := handLevel(h, levels[h]+1)
		for _, i := range ix {
			base += cs[i].Value()
		}
		if s := base * mult; s > bestScore {
			best, bestScore = madeHand{h: h, cards: ix}, s
		}
	}
	return best
}
