package game

import (
	"sort"
	"testing"
)

func cards(spec ...[2]int) []Card {
	var out []Card
	for _, s := range spec {
		out = append(out, Card{R: s[0], S: s[1]})
	}
	return out
}

func TestDeckIsThirteenUniqueCards(t *testing.T) {
	for seed := int64(0); seed < 500; seed++ {
		for fam := FamSame; fam <= FamSuits; fam++ {
			d := DealDeck(seed, fam)
			if len(d) != 13 {
				t.Fatalf("seed %d: %d cards", seed, len(d))
			}
			seen := map[[2]int]bool{}
			for _, c := range d {
				k := [2]int{c.R, c.S}
				if seen[k] {
					t.Fatalf("seed %d: duplicate %v", seed, k)
				}
				seen[k] = true
			}
			if fam == FamSame {
				n := rankCounts(d)
				best := 0
				for _, k := range n {
					best = max(best, k)
				}
				if best < 3 {
					t.Fatalf("seed %d: Same deck has no 3 of a rank", seed)
				}
			}
		}
	}
}

func TestSameSeedSameDeck(t *testing.T) {
	a, b := DealDeck(42, FamSuits), DealDeck(42, FamSuits)
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("decks differ")
		}
	}
}

func TestFlushSizeExamples(t *testing.T) {
	// the examples from the design: 3 clubs + 3 hearts -> 3
	d := cards([2]int{2, Clubs}, [2]int{3, Clubs}, [2]int{4, Clubs}, [2]int{2, Hearts}, [2]int{3, Hearts}, [2]int{4, Hearts})
	if FlushSize(d) != 3 {
		t.Fatalf("got %d", FlushSize(d))
	}
	// 2 diamonds, 5 clubs, 3 hearts -> 3.33 -> 3
	d = cards([2]int{2, Diamonds}, [2]int{3, Diamonds}, [2]int{2, Clubs}, [2]int{3, Clubs}, [2]int{4, Clubs}, [2]int{5, Clubs},
		[2]int{6, Clubs}, [2]int{2, Hearts}, [2]int{3, Hearts}, [2]int{4, Hearts})
	if FlushSize(d) != 3 {
		t.Fatalf("got %d", FlushSize(d))
	}
	// 13 cards in 2 suits -> 6.5 -> capped at 5
	var big []Card
	for r := 2; r <= 8; r++ {
		big = append(big, Card{R: r, S: Spades})
	}
	for r := 2; r <= 7; r++ {
		big = append(big, Card{R: r, S: Hearts})
	}
	if FlushSize(big) != 5 {
		t.Fatalf("got %d", FlushSize(big))
	}
}

func TestStraightSize(t *testing.T) {
	// runs 2-3-4-5 (4) and 9-10 (2): average 3 -> 3
	d := cards([2]int{2, 0}, [2]int{3, 0}, [2]int{4, 0}, [2]int{5, 1}, [2]int{9, 0}, [2]int{10, 2})
	if got := StraightSize(d); got != 3 {
		t.Fatalf("got %d (runs %v)", got, rankRuns(d))
	}
	// runs 2-3-4-5 (4) and 9-10-J (3): 3.5 rounds up to 4
	d = append(d, Card{R: 11, S: 3})
	if got := StraightSize(d); got != 4 {
		t.Fatalf("got %d (runs %v)", got, rankRuns(d))
	}
	// an ace extends a run that starts at 2
	d = cards([2]int{14, 0}, [2]int{2, 0}, [2]int{3, 1})
	if r := rankRuns(d); len(r) != 1 || r[0] != 3 {
		t.Fatalf("runs %v", r)
	}
}

func TestDetect(t *testing.T) {
	rules := handRules{flush: 3, straight: 3}
	cases := []struct {
		h  HandType
		cs []Card
		ok bool
	}{
		{Pair, cards([2]int{5, 0}, [2]int{5, 1}), true},
		{TwoPair, cards([2]int{5, 0}, [2]int{5, 1}, [2]int{9, 0}, [2]int{9, 2}), true},
		{FullHouse, cards([2]int{5, 0}, [2]int{5, 1}, [2]int{5, 2}, [2]int{9, 0}, [2]int{9, 2}), true},
		{FourKind, cards([2]int{5, 0}, [2]int{5, 1}, [2]int{5, 2}, [2]int{5, 3}), true},
		{Straight, cards([2]int{5, 0}, [2]int{6, 1}, [2]int{7, 2}), true},
		{Straight, cards([2]int{14, 0}, [2]int{2, 1}, [2]int{3, 2}), true},
		{Straight, cards([2]int{5, 0}, [2]int{6, 1}, [2]int{8, 2}), false},
		{SkipStraight, cards([2]int{3, 0}, [2]int{5, 1}, [2]int{7, 2}), true},
		{Ladder, cards([2]int{7, 0}, [2]int{7, 1}, [2]int{8, 0}, [2]int{8, 2}), true},
		{Ladder, cards([2]int{7, 0}, [2]int{7, 1}, [2]int{9, 0}, [2]int{9, 2}), false},
		{Flush, cards([2]int{2, 1}, [2]int{9, 1}, [2]int{12, 1}), true},
		{TwoTone, cards([2]int{2, 1}, [2]int{9, 1}, [2]int{12, 0}, [2]int{4, 0}), true},
		{TwoTone, cards([2]int{2, 1}, [2]int{9, 1}, [2]int{12, 0}, [2]int{4, 2}), false},
		{Rainbow, cards([2]int{2, 0}, [2]int{9, 1}, [2]int{12, 2}, [2]int{4, 3}), true},
	}
	for i, c := range cases {
		if _, ok := detect(c.h, c.cs, rules); ok != c.ok {
			t.Errorf("case %d (%s): got %v", i, hands[c.h].name, ok)
		}
	}
}

func TestSpecialHandsScoreAsTheirHand(t *testing.T) {
	for _, sp := range specialHands {
		r := NewRun(7, FamSame, sp)
		r.Picks = []HandType{TwoPair, ThreeKind, FullHouse}
		r.StartRound(1 << 40)
		before := 0
		for _, c := range r.Deck {
			before += c.Bonus
		}
		res := r.UseSpecial()
		if res == nil || res.Hand != sp || !res.Special {
			t.Fatalf("%s: %+v", hands[sp].name, res)
		}
		// the changed cards really make that hand
		if sp == FiveKind {
			for _, c := range res.Cards {
				if c.R != res.Cards[0].R {
					t.Fatal("five of a kind ranks differ")
				}
			}
		}
		if sp == StraightFlush {
			if _, ok := detect(Straight, res.Cards, handRules{straight: len(res.Cards)}); !ok {
				t.Fatalf("not a straight: %v", res.Cards)
			}
		}
		after := 0
		for _, c := range r.Deck {
			after += c.Bonus
		}
		if after-before != len(res.Cards)*specialBuff || len(res.Cards) != baseHandSize {
			t.Fatalf("%s: buff %d", hands[sp].name, after-before)
		}
		if r.Plays != basePlays {
			t.Fatal("special used a play")
		}
	}
	// five of a kind scores less than the other two
	if hands[FiveKind].base >= hands[StraightFlush].base || hands[FiveKind].mult >= hands[StraightFlush].mult {
		t.Fatal("five of a kind should be the weakest special")
	}
}

func TestSynergy(t *testing.T) {
	if _, m := synergyOf([]HandType{TwoPair, ThreeKind, FullHouse}); m != 1.5 {
		t.Fatal(m)
	}
	if _, m := synergyOf([]HandType{TwoPair, ThreeKind, Flush}); m != 1.2 {
		t.Fatal(m)
	}
	if _, m := synergyOf([]HandType{TwoPair, Straight, Flush}); m != 1 {
		t.Fatal(m)
	}
}

// How much a plain bot scores in a round, to set the blind targets (go test -v -run Bot)
func TestBotRoundScores(t *testing.T) {
	var all []int64
	for seed := int64(1); seed <= 300; seed++ {
		fam := Family(seed % 3)
		r := NewRun(seed, fam, specialHands[seed%3])
		r.Picks = familyHands(fam)[:3]
		r.StartRound(0)
		for i := 0; i < 50 && !r.Done; i++ {
			r.botTurn()
		}
		if !r.Done {
			t.Fatalf("seed %d: bot never finished", seed)
		}
		all = append(all, r.Score)
	}
	sort.Slice(all, func(a, b int) bool { return all[a] < all[b] })
	t.Logf("round totals: 10%% %d, median %d, 90%% %d", all[len(all)/10], all[len(all)/2], all[len(all)*9/10])
	if all[len(all)/2] < blindTarget(0, 0) {
		t.Errorf("a plain bot's median round (%d) should beat the first blind (%d)", all[len(all)/2], blindTarget(0, 0))
	}
}
