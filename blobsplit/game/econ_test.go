package game

import "testing"

func TestFmtNum(t *testing.T) {
	for f, want := range map[float64]string{0: "0", 7: "7", 999: "999", 1234: "1.23K", 15600: "15.6K", 1.6e9: "1.6B", 2.5: "2.5"} {
		if got := fmtNum(f); got != want {
			t.Errorf("fmtNum(%v) = %q, want %q", f, got, want)
		}
	}
}

// Upgrades must always get more expensive, and the board must never be bigger than
// the 64 blobs the layout is made for.
func TestUpgradeCurves(t *testing.T) {
	for l := 0; l < 50; l++ {
		if valueCost(l+1) <= valueCost(l) || valueOf(l+1) <= valueOf(l) {
			t.Fatalf("blob value level %d does not grow", l)
		}
	}
	if capOf(maxCapLv) != 64 {
		t.Fatalf("max capacity is %d, want 64", capOf(maxCapLv))
	}
	if baseCooldown(maxSpeedLv) < 1.9 {
		t.Fatalf("split cooldown at max level is %v", baseCooldown(maxSpeedLv))
	}
	for i := 1; i < len(rarities); i++ {
		if rarities[i].Odds <= rarities[i-1].Odds || rarities[i].Mult <= rarities[i-1].Mult {
			t.Fatalf("rarity %s is not rarer and worth more than the one before", rarities[i].Name)
		}
	}
}

// Without luck, a plain blob mutates into something 1 time in 15.
func TestMutationChance(t *testing.T) {
	sum := 0.0
	for r := 1; r < len(rarities); r++ {
		sum += mutationChance(r)
	}
	if sum < 1.0/15-1e-9 || sum > 1.0/15+1e-9 {
		t.Fatalf("chance of any mutation is 1/%.2f, want 1/15", 1/sum)
	}
}
