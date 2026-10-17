package game

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestNumbers(t *testing.T) {
	for in, want := range map[float64]string{5: "5", 999: "999", 1500: "1.50K", 12345: "12.3K", 2.5e6: "2.50M", 1e12: "1.00T"} {
		if got := num(in); got != want {
			t.Errorf("num(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestLevelUp(t *testing.T) {
	p := newProfile(0)
	if err := p.levelUp(0, 1); err == nil {
		t.Fatal("levelled up with no coins")
	}
	p.Coins = 1e6
	d := p.dps(0)
	if err := p.levelUp(0, 9); err != nil {
		t.Fatal(err)
	}
	if p.Bugs[0].Level != 10 || p.dps(0) < d*15 {
		t.Fatalf("level 10 should double damage: level %d dps %v", p.Bugs[0].Level, p.dps(0))
	}
	if p.Coins >= 1e6 {
		t.Fatal("levels were free")
	}
	n := p.maxLevels(-1)
	if p.upCost(-1, n) > p.Coins || p.upCost(-1, n+1) <= p.Coins {
		t.Fatalf("maxLevels %d is wrong", n)
	}
}

func TestStages(t *testing.T) {
	p := newProfile(0)
	for i := 0; i < wavesPerStage+3; i++ {
		p.beatSlug(1)
	}
	if p.Wave != wavesPerStage {
		t.Fatalf("wave %d", p.Wave)
	}
	p.loseBoss()
	if !p.Farming {
		t.Fatal("not farming after losing")
	}
	gems := p.Gems
	for s := 1; s <= 5; s++ {
		egg := p.beatBoss()
		if egg != (s == 5) {
			t.Fatalf("stage %d egg %v", s, egg)
		}
	}
	if p.Stage != 6 || p.Gems <= gems || p.Farming {
		t.Fatalf("after 5 bosses: stage %d gems %d farming %v", p.Stage, p.Gems, p.Farming)
	}
	if p.slots() != 3 {
		t.Fatalf("slots at stage 6 = %d", p.slots())
	}
	if slugHP(10) <= slugHP(9) || bossHP(3) <= slugHP(3) {
		t.Fatal("slugs should get tougher")
	}
}

func TestEggsAndTeam(t *testing.T) {
	p := newProfile(0)
	rng := rand.New(rand.NewSource(3))
	if _, err := p.buy(shopItems[0], rng); err != nil {
		t.Fatal(err)
	}
	if _, err := p.buy(shopItems[0], rng); err != errGems {
		t.Fatal("bought an egg without enough gems")
	}
	seen := map[int]bool{}
	for i := 0; i < 2000; i++ {
		id := rollBug(rng, eggs[1].weights)
		if bugs[id].rarity == rCommon {
			t.Fatal("golden egg gave a common")
		}
		seen[id] = true
	}
	if len(seen) != 6 {
		t.Fatalf("golden eggs gave %d different bugs", len(seen))
	}
	if !p.hatch(3) || p.hatch(3) || p.Bugs[3].Stars != 1 {
		t.Fatal("hatching copies should add stars")
	}
	if err := p.toggleTeam(3); err == nil {
		t.Fatal("team over its slot count")
	}
	p.Stage = 2 // a second slot
	if err := p.toggleTeam(3); err != nil || !p.inTeam(3) {
		t.Fatal("could not add a bug")
	}
	if err := p.toggleTeam(0); err != nil || p.inTeam(0) {
		t.Fatal("could not rest a bug")
	}
	if err := p.toggleTeam(3); err == nil {
		t.Fatal("removed the last bug")
	}
}

func TestOfflineAndSave(t *testing.T) {
	p := newProfile(1000)
	if c, _ := p.offline(1030); c != 0 {
		t.Fatal("paid for 30 seconds away")
	}
	c8, s := p.offline(1000 + 100*3600)
	if s != offlineCap || c8 <= 0 {
		t.Fatalf("offline cap: %v %v", c8, s)
	}
	p.Coins, p.Stage = 123, 9
	b, _ := json.Marshal(p)
	q := &Profile{}
	if err := json.Unmarshal(b, q); err != nil {
		t.Fatal(err)
	}
	q.fix()
	if q.Coins != 123 || q.Stage != 9 || len(q.Bugs) != len(bugs) || len(q.Team) != 1 {
		t.Fatalf("save round trip: %+v", q)
	}
	if _, err := q.freeEgg(1000, rand.New(rand.NewSource(1))); err != nil {
		t.Fatal(err)
	}
	if _, err := q.freeEgg(1001, rand.New(rand.NewSource(1))); err == nil {
		t.Fatal("two free eggs in a row")
	}
}

// TestPacing plays the first stages with a simple player who keeps levelling the
// cheapest thing, and checks the early game moves along but not too fast.
func TestPacing(t *testing.T) {
	p := newProfile(0)
	sec := 0.0
	hp := slugHP(1)
	for sec < 20*60 && p.Stage < 40 {
		taps := 3.0 // taps per second
		dmg := p.teamDPS() + p.tapDamage()*taps
		boss := p.Wave >= wavesPerStage && !p.Farming
		if boss {
			need := bossHP(p.Stage)
			if need/dmg <= bossTime {
				sec += need / dmg
				p.beatBoss()
			} else {
				p.loseBoss()
			}
		} else {
			sec += hp/dmg + 0.5
			p.beatSlug(slugCoins(p.Stage))
		}
		if p.Farming && p.Wave >= wavesPerStage && (p.teamDPS()+p.tapDamage()*taps)*bossTime > bossHP(p.Stage) {
			p.Farming = false
		}
		hp = slugHP(p.Stage)
		for {
			best, cost := -2, 0.0
			for _, id := range append([]int{-1}, p.Team...) {
				if c := p.upCost(id, 1); best == -2 || c < cost {
					best, cost = id, c
				}
			}
			if cost > p.Coins {
				break
			}
			p.levelUp(best, 1)
		}
	}
	if p.Stage < 6 || p.Stage > 35 {
		t.Fatalf("after 20 minutes with one bug the player reached stage %d", p.Stage)
	}
	t.Logf("20 minutes: stage %d, rory lv %d, tap lv %d", p.Stage, p.Bugs[0].Level, p.TapLv)
}
