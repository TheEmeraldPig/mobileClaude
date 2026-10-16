package game

import "testing"

func TestEconomy(t *testing.T) {
	s := newState()
	if s.BuyGen(0) {
		t.Fatal("bought a miner with no coins")
	}
	s.earn(15)
	if !s.BuyGen(0) || s.Coins != 0 || genCost(0, 1) != 18 {
		t.Fatalf("miner buy: coins %v next cost %v", s.Coins, genCost(0, 1))
	}
	if s.DescendGems() != 0 || s.Descend() {
		t.Fatal("descended too early")
	}
	s.earn(50000 * 16)
	if s.DescendGems() != 4 || !s.Descend() || s.Gems != 4 || s.Owned[0] != 0 {
		t.Fatalf("descend: gems %d owned %v", s.Gems, s.Owned)
	}
	if m := s.mult(0); m < 1.39 || m > 1.41 {
		t.Fatalf("4 gems should give 1.4x, got %v", m)
	}
	if !s.BuyGem(0, 1000) || s.BoostUntil != 1000+900 {
		t.Fatalf("boost: %v", s.BoostUntil)
	}
	if s.ClaimDaily(100000) != 1 || s.ClaimDaily(100001) != 0 || s.ClaimDaily(100000+dailyGap) != 2 {
		t.Fatal("daily streak")
	}
	s.Owned[0] = 10
	s.LastSeen = 0
	if s.Offline(500) != 0 {
		t.Fatal("offline with no last-seen time")
	}
	s.LastSeen = 1
	s.BoostUntil = 0
	if c := s.Offline(1 + 100000); c > 5*s.mult(0)*7200*0.5+1 {
		t.Fatalf("offline not capped: %v", c)
	}
}
