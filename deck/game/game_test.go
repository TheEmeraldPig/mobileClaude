package game

import (
	"math/rand"
	"testing"
)

func TestEnhancements(t *testing.T) {
	s := Card{ID: "strike"}
	if s.stats().Dmg != 6 {
		t.Fatal("strike should deal 6")
	}
	if (Card{ID: "strike", Enh: eFoil}).stats().Dmg != 9 || (Card{ID: "strike", Enh: eHolo}).stats().Dmg != 9 {
		t.Fatal("foil and holo strike should deal 9")
	}
	if (Card{ID: "bash", Enh: eLight}).stats().Cost != 1 {
		t.Fatal("light bash should cost 1")
	}
}

func TestProfile(t *testing.T) {
	p := newProfile(1000)
	if p.Enhance(0, eFoil) == nil {
		t.Fatal("enhanced with no gold")
	}
	p.Gold = 1000
	if p.Enhance(0, eFoil) != nil || p.Deck[0].Enh != eFoil || p.Gold != 850 {
		t.Fatal("foil")
	}
	for len(p.Deck) > minDeck {
		if err := p.Remove(0); err != nil {
			t.Fatal(err)
		}
	}
	if p.Remove(0) == nil {
		t.Fatal("removed below the minimum deck size")
	}
	rng := rand.New(rand.NewSource(1))
	if _, err := p.Buy(0, 1000+dailyGap, rng); err != nil || len(p.Deck) != minDeck+1 {
		t.Fatalf("free chest: %v", err)
	}
	if _, err := p.Buy(0, 1000+dailyGap+5, rng); err == nil {
		t.Fatal("second free chest the same day")
	}
	p.Energy = 0
	p.tickEnergy(p.EnergyAt + 3*energyEvery + 10)
	if p.Energy != 3 {
		t.Fatalf("energy regen: %d", p.Energy)
	}
}

// A simple bot plays every card it can afford each turn. With the starter deck it
// should beat the first floor, and the rewards and next fight should follow.
func TestRun(t *testing.T) {
	p := newProfile(0)
	r := newRun(7, p)
	for turn := 0; turn < 40 && r.Rewards == nil && !r.Lost; turn++ {
		b := r.B
		for i := 0; i < len(b.Hand); {
			if b.Play(i) != nil {
				i++
			}
			if r.Rewards != nil {
				break
			}
		}
		if r.Rewards == nil {
			b.EndTurn()
		}
	}
	if r.Lost || len(r.Rewards) != 3 || r.Gold <= 0 {
		t.Fatalf("floor 1: lost=%v rewards=%d gold=%d hp=%d", r.Lost, len(r.Rewards), r.Gold, r.HP)
	}
	r.Pick(0)
	if r.Floor != 2 || r.B == nil || r.B.EHP <= 0 || len(r.B.Hand) != 5 {
		t.Fatal("next fight should start with a fresh hand")
	}
	r.HP = 1
	r.B.Intent = move{mAttack, 99, 1}
	r.B.Hand = nil
	r.B.EndTurn()
	if !r.Lost {
		t.Fatal("should lose at 0 HP")
	}
	r.Revive()
	if r.Lost || r.HP != r.Max {
		t.Fatal("revive")
	}
}
