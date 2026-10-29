package game

import (
	"math"
	"math/rand"
	"sort"
)

// Cards and the 13-card deck.
//
// Every player gets 13 different cards (no two with the same rank and suit), dealt at
// random from the 52 when the match starts and locked for the whole match: nothing adds,
// removes or changes cards, except the small permanent bonus a card gets from a special
// hand. Because decks differ, the sizes of flushes and straights adapt to each deck
// (FlushSize, StraightSize).

const (
	Spades = iota
	Hearts
	Clubs
	Diamonds
)

var suitNames = [4]string{"Spades", "Hearts", "Clubs", "Diamonds"}

type Card struct {
	R     int `json:"r"` // 2-14 (11 J, 12 Q, 13 K, 14 A)
	S     int `json:"s"` // suit
	Bonus int `json:"b"` // permanent extra base points, from special hands
	Glow  int `json:"g"` // times buffed (for the look)
}

func (c Card) Value() int {
	v := c.R
	switch {
	case c.R == 14:
		v = 11
	case c.R > 10:
		v = 10
	}
	return v + c.Bonus
}

func rankName(r int) string {
	switch r {
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	case 14, 1:
		return "A"
	case 10:
		return "10"
	}
	return string(rune('0' + r))
}

// The card's name, like "10" or "Q"
func (c Card) Name() string { return rankName(c.R) }

func isRed(s int) bool { return s == Hearts || s == Diamonds }

// DealDeck deals 13 different cards for one player. Players of the "Same" family always
// get at least three cards of one rank.
func DealDeck(seed int64, fam Family) []Card {
	rng := rand.New(rand.NewSource(seed))
	all := make([]Card, 0, 52)
	for s := 0; s < 4; s++ {
		for r := 2; r <= 14; r++ {
			all = append(all, Card{R: r, S: s})
		}
	}
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	deck := append([]Card(nil), all[:13]...)
	if fam == FamSame {
		counts := rankCounts(deck)
		best := 0
		for r := 2; r <= 14; r++ {
			if counts[r] > counts[best] {
				best = r
			}
		}
		if counts[best] < 3 {
			// swap in the missing suits of the most common rank, replacing cards of ranks
			// that appear only once
			for s := 0; s < 4 && counts[best] < 3; s++ {
				if hasCard(deck, best, s) {
					continue
				}
				for i := range deck {
					if deck[i].R != best && counts[deck[i].R] == 1 {
						counts[deck[i].R]--
						deck[i] = Card{R: best, S: s}
						counts[best]++
						break
					}
				}
			}
		}
	}
	sortCards(deck)
	return deck
}

func hasCard(deck []Card, r, s int) bool {
	for _, c := range deck {
		if c.R == r && c.S == s {
			return true
		}
	}
	return false
}

func rankCounts(cards []Card) [15]int {
	var n [15]int
	for _, c := range cards {
		n[c.R]++
	}
	return n
}

func suitCounts(cards []Card) [4]int {
	var n [4]int
	for _, c := range cards {
		n[c.S]++
	}
	return n
}

func sortCards(c []Card) {
	sort.SliceStable(c, func(a, b int) bool {
		if c[a].R != c[b].R {
			return c[a].R > c[b].R
		}
		return c[a].S < c[b].S
	})
}

// FlushSize: how many cards of one suit make a flush for this deck. It is the average
// number of cards per suit the deck has, rounded down (3 clubs + 3 hearts -> 3;
// 2 diamonds, 5 clubs, 3 hearts -> 3.33 -> 3). At most 5, since a play is 5 cards.
func FlushSize(deck []Card) int {
	n := suitCounts(deck)
	total, suits := 0, 0
	for _, k := range n {
		if k > 0 {
			total += k
			suits++
		}
	}
	if suits == 0 {
		return 5
	}
	return clampInt(total/suits, 1, 5)
}

// StraightSize: how many ranks in a row make a straight for this deck. It is the
// average length of the runs of ranks in a row the deck has, rounded up (at most 5,
// since a play is 5 cards). An ace counts both low and high.
func StraightSize(deck []Card) int {
	runs := rankRuns(deck)
	if len(runs) == 0 {
		return 5
	}
	sum := 0
	for _, l := range runs {
		sum += l
	}
	return clampInt(int(math.Ceil(float64(sum)/float64(len(runs)))), 1, 5)
}

// The lengths of the runs of ranks in a row in a deck. An ace also sits below the 2:
// it lengthens a run that starts at 2 (unless that run already reaches the ace).
func rankRuns(deck []Card) []int {
	var has [15]bool
	for _, c := range deck {
		has[c.R] = true
	}
	var runs []int
	start, l := 0, 0
	lowRun, topRun := -1, -1 // the runs that start at 2 and end at the ace
	for r := 2; r <= 15; r++ {
		if r <= 14 && has[r] {
			if l == 0 {
				start = r
			}
			l++
			continue
		}
		if l > 0 {
			if start == 2 {
				lowRun = len(runs)
			}
			if r == 15 {
				topRun = len(runs)
			}
			runs = append(runs, l)
		}
		l = 0
	}
	if has[14] && lowRun >= 0 && runs[lowRun] < 13 {
		runs[lowRun]++
		if !has[13] && topRun >= 0 && topRun != lowRun { // a lone ace at the top is that same ace
			runs = append(runs[:topRun], runs[topRun+1:]...)
		}
	}
	return runs
}

// The deck's most common rank, ties broken at random (for Five of a Kind)
func mostCommonRank(deck []Card, rng *rand.Rand) int {
	n := rankCounts(deck)
	best := 0
	var ties []int
	for r := 2; r <= 14; r++ {
		switch {
		case n[r] > best:
			best, ties = n[r], []int{r}
		case n[r] == best && best > 0:
			ties = append(ties, r)
		}
	}
	if len(ties) == 0 {
		return 14
	}
	return ties[rng.Intn(len(ties))]
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }
