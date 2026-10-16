package game

import (
	"math"
	"testing"
)

func TestShop(t *testing.T) {
	p := newProfile()
	if p.BuyUpg(0) == nil {
		t.Fatal("bought an upgrade with no coins")
	}
	p.Coins = 1000
	if p.BuyUpg(0) != nil || p.Upg["magnet"] != 1 || p.Coins != 960 {
		t.Fatalf("upgrade: %v %v", p.Upg, p.Coins)
	}
	if p.TapShip(3) == nil {
		t.Fatal("bought a gem ship with no gems")
	}
	if p.TapShip(1) != nil || p.Ship != "needle" || p.Coins != 560 {
		t.Fatalf("ship: %v %v", p.Ship, p.Coins)
	}
	if p.BuyPack(0) != nil || p.BuyPack(0) == nil || p.Gems != 40 {
		t.Fatal("starter pack should be one-time")
	}
	if p.runCoins(10, 5) != 10+5*1 {
		t.Fatalf("run coins %d", p.runCoins(10, 5))
	}
	if p.ClaimDaily(100000) != 1 || p.ClaimDaily(100001) != 0 || p.ClaimDaily(100000+dailyGap) != 2 {
		t.Fatal("daily streak")
	}
}

// A ship launched at the right moment reaches the next planet; one that never
// launches is caught by the void.
func TestRun(t *testing.T) {
	p := newProfile()
	r := newRun(1, p, 960)
	for i := 0; i < 60*120 && !r.dead; i++ {
		r.step(1.0/60, 960)
	}
	if !r.dead || r.why != "The void caught you" {
		t.Fatalf("idle run should end in the void, got dead=%v %q", r.dead, r.why)
	}

	r = newRun(1, p, 960)
	next := r.planets[1]
	for i := 0; i < 60*20 && r.score == 0 && !r.dead; i++ {
		if r.on == 0 {
			// launch when the ship's heading points at the next planet
			hx, hy := -math.Sin(r.ang)*r.dir, math.Cos(r.ang)*r.dir
			dx, dy := next.x-r.x, next.y-r.y
			if (hx*dx+hy*dy)/math.Hypot(dx, dy) > 0.995 {
				r.Launch()
			}
		}
		r.step(1.0/60, 960)
	}
	if r.score < 1 || r.on != 1 {
		t.Fatalf("aimed launch should reach planet 1: score %d on %d dead %v %q", r.score, r.on, r.dead, r.why)
	}
	r.hit("test")
	r.Revive()
	if r.dead || r.on != 1 {
		t.Fatal("revive should put the ship back in orbit")
	}
}
