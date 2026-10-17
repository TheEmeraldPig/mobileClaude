package game

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func cards(ids ...string) []Card {
	var c []Card
	for _, id := range ids {
		c = append(c, Card{ID: id})
	}
	return c
}

func TestLoadoutRules(t *testing.T) {
	ok := [][]Card{
		cards("strike", "defend", "jab", "twin", "brace"),   // commons
		cards("bash", "heavy", "wall", "warcry", "insight"), // uncommons only
		cards("excalibur", "strike", "strike", "defend", "defend"),
		cards("meteor", "phoenix", "strike", "defend", "jab"),
		cards("fireball", "plague", "focus", "bash", "strike"),
	}
	for _, d := range ok {
		if err := checkLoadout(d); err != nil {
			t.Errorf("%v should be allowed: %v", d, err)
		}
	}
	bad := [][]Card{
		cards("strike", "strike", "strike", "strike", "strike", "strike"), // 6 cards
		cards("excalibur", "aegis"),                                       // 2 legendaries
		cards("excalibur", "bash"),                                        // legendary + uncommon
		cards("meteor", "phoenix", "toxic"),                               // 3 epics
		cards("meteor", "bash"),                                           // epic + uncommon
		cards("meteor", "fireball"),                                       // epic + rare
		cards("fireball", "plague", "focus", "fortress"),                  // 4 rares
		cards("fireball", "bash", "heavy"),                                // rare + 2 uncommons
	}
	for _, d := range bad {
		if checkLoadout(d) == nil {
			t.Errorf("%v should be refused", d)
		}
	}
}

func TestEnhancements(t *testing.T) {
	if (Card{ID: "strike", Enh: eFoil}).stats().Dmg != 9 || (Card{ID: "strike", Enh: eHolo}).stats().Dmg != 9 {
		t.Fatal("foil and holo strike should deal 9")
	}
	if (Card{ID: "bash", Enh: eLight}).stats().Cost != 1 {
		t.Fatal("light bash should cost 1")
	}
}

func TestProfile(t *testing.T) {
	p := newProfile(1000)
	if len(p.Loadout()) != 5 {
		t.Fatalf("starter deck has %d cards", len(p.Loadout()))
	}
	// a 6th card does not fit
	for i, c := range p.Cards {
		if !c.In {
			if p.Toggle(i) == nil {
				t.Fatal("added a 6th card")
			}
			break
		}
	}
	p.Gold = 1000
	if p.Enhance(0, eFoil) != nil || p.Cards[0].Enh != eFoil || p.Gold != 850 {
		t.Fatal("foil")
	}
	rng := rand.New(rand.NewSource(1))
	n := len(p.Cards)
	if _, err := p.Buy(0, 1000+dailyGap, rng); err != nil || len(p.Cards) != n+1 {
		t.Fatalf("free chest: %v", err)
	}
	if _, err := p.Buy(0, 1000+dailyGap+5, rng); err == nil {
		t.Fatal("second free chest the same day")
	}
}

// Every map node must lead on to the boss, and the first row must offer a choice.
func TestMap(t *testing.T) {
	for seed := int64(1); seed < 60; seed++ {
		nodes := genMap(rand.New(rand.NewSource(seed)))
		starts, shops := 0, 0
		boss := -1
		for i, n := range nodes {
			if n.Row == 0 {
				starts++
			}
			if n.Kind == nShop {
				shops++
			}
			if n.Kind == nBoss {
				boss = i
			}
			if n.Kind != nBoss && len(n.Next) == 0 {
				t.Fatalf("seed %d: node %d is a dead end", seed, i)
			}
			for _, j := range n.Next {
				if nodes[j].Row != n.Row+1 {
					t.Fatalf("seed %d: path skips a row", seed)
				}
			}
		}
		if starts < 2 || boss < 0 || shops == 0 {
			t.Fatalf("seed %d: starts %d boss %d shops %d", seed, starts, boss, shops)
		}
	}
}

func TestRun(t *testing.T) {
	p := newProfile(0)
	r := newRun(7, p)
	r.restore()
	// first fight: a bot plays everything it can
	r.Enter(r.Reachable()[0])
	for turn := 0; turn < 40 && r.Choices == nil && !r.Lost; turn++ {
		b := r.B
		for i := 0; i < len(b.Hand); {
			if b.Playable(i) {
				b.Play(i)
				continue
			}
			i++
		}
		if r.Choices == nil {
			b.EndTurn()
		}
	}
	if r.Lost || len(r.Choices) != 3 || r.Coins <= 0 || r.Gold <= 0 {
		t.Fatalf("floor 1: lost=%v choices=%d coins=%d hp=%d", r.Lost, len(r.Choices), r.Coins, r.HP)
	}
	n := len(r.Deck)
	r.PickCard(0)
	r.Leave()
	if len(r.Deck) != n+1 || len(r.Reachable()) == 0 {
		t.Fatal("picking a card and moving on")
	}
	// the merchant
	r.Coins = 1000
	r.Cur = 0
	r.makeShop()
	if err := r.BuyOffer(0); err != nil || !r.Shop[0].Sold || r.BuyOffer(0) == nil {
		t.Fatalf("shop: %v", err)
	}
	// saving and loading keeps the run
	b, _ := json.Marshal(p2(p, r))
	var q Profile
	if err := json.Unmarshal(b, &q); err != nil || q.Run == nil || len(q.Run.Deck) != len(r.Deck) || len(q.Run.Nodes) != len(r.Nodes) {
		t.Fatalf("save round trip: %v", err)
	}
	// losing and reviving
	r.Enter(r.Reachable()[0])
	for r.B == nil { // not a fight: try the next spot
		r.Leave()
		r.Enter(r.Reachable()[0])
	}
	r.HP = 1
	r.B.Intent = move{mAttack, 99, 1}
	r.B.EndTurn()
	if !r.Lost {
		t.Fatal("should lose at 0 HP")
	}
	r.Revive()
	if r.Lost || r.HP != r.Max || r.B == nil {
		t.Fatal("revive")
	}
}

func p2(p *Profile, r *Run) *Profile {
	c := *p
	c.Run = r
	return &c
}

// Every event choice resolves without crashing.
func TestEvents(t *testing.T) {
	for e := range events {
		for c := range events[e].Choices {
			r := newRun(3, newProfile(0))
			r.restore()
			r.Coins = 100
			r.Event = e
			if _, err := r.Choose(c); err != nil {
				t.Errorf("event %d choice %d: %v", e, c, err)
			}
		}
	}
}
