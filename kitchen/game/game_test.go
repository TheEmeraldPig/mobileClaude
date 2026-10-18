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
	if p.Chefs[0].Level != 10 || p.dps(0) < d*15 {
		t.Fatalf("level 10 should double damage: level %d dps %v", p.Chefs[0].Level, p.dps(0))
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
		p.beatMonster(1)
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
	if monHP(10) <= monHP(9) || bossHP(3) <= monHP(3) {
		t.Fatal("slugs should get tougher")
	}
}

func TestRecruitAndTeam(t *testing.T) {
	p := newProfile(0)
	rng := rand.New(rand.NewSource(3))
	if _, err := p.buy(shopItems[0], rng); err != nil {
		t.Fatal(err)
	}
	if _, err := p.buy(shopItems[0], rng); err != errGems {
		t.Fatal("bought a box without enough gems")
	}
	seen := map[int]bool{}
	for i := 0; i < 2000; i++ {
		id := rollChef(rng, boxes[1].weights)
		if chefs[id].rarity == rCommon {
			t.Fatal("star recruit gave a common")
		}
		seen[id] = true
	}
	if len(seen) != 6 {
		t.Fatalf("star recruits gave %d different chefs", len(seen))
	}
	if !p.hire(3) || p.hire(3) || p.Chefs[3].Stars != 1 {
		t.Fatal("recruiting copies should add stars")
	}
	if err := p.toggleTeam(3); err == nil {
		t.Fatal("team over its slot count")
	}
	p.Stage = 2 // a second slot
	if err := p.toggleTeam(3); err != nil || !p.inTeam(3) {
		t.Fatal("could not add a chef")
	}
	if err := p.toggleTeam(0); err != nil || p.inTeam(0) {
		t.Fatal("could not take a chef off duty")
	}
	if err := p.toggleTeam(3); err == nil {
		t.Fatal("removed the last chef")
	}
}

func TestOfflineAndSave(t *testing.T) {
	p := newProfile(1000)
	if a := p.offline(1030); a.coins != 0 {
		t.Fatal("paid for 30 seconds away")
	}
	a := p.offline(1000 + 100*3600)
	if a.secs != offlineCap || a.coins <= 0 || a.total() <= 0 {
		t.Fatalf("offline: %+v", a)
	}
	p.Coins, p.Stage = 123, 9
	b, _ := json.Marshal(p)
	q := &Profile{}
	if err := json.Unmarshal(b, q); err != nil {
		t.Fatal(err)
	}
	q.fix()
	if q.Coins != 123 || q.Stage != 9 || len(q.Chefs) != len(chefs) || len(q.Team) != 1 {
		t.Fatalf("save round trip: %+v", q)
	}
	if _, err := q.freeBox(1000, rand.New(rand.NewSource(1))); err != nil {
		t.Fatal(err)
	}
	if _, err := q.freeBox(1001, rand.New(rand.NewSource(1))); err == nil {
		t.Fatal("two free boxes in a row")
	}
}

// TestPacing plays the first stages with a simple player who keeps levelling the
// cheapest thing, and checks the early game moves along but not too fast.
func TestPacing(t *testing.T) {
	p := newProfile(0)
	sec := 0.0
	hp := monHP(1)
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
			p.beatMonster(monCoins(p.Stage))
		}
		if p.Farming && p.Wave >= wavesPerStage && (p.teamDPS()+p.tapDamage()*taps)*bossTime > bossHP(p.Stage) {
			p.Farming = false
		}
		hp = monHP(p.Stage)
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
		t.Fatalf("after 20 minutes with one chef the player reached stage %d", p.Stage)
	}
	t.Logf("20 minutes: stage %d, rory lv %d, tap lv %d", p.Stage, p.Chefs[0].Level, p.TapLv)
}

func TestKitchen(t *testing.T) {
	p := newProfile(0)
	if err := p.canCook(dPizza); err == nil {
		t.Fatal("cooked pizza without an oven")
	}
	if err := p.startCook(dToast, 1.5); err != nil {
		t.Fatal(err)
	}
	if p.Ing[inFlour] != 2 {
		t.Fatalf("toast should use 2 flour, %d left", p.Ing[inFlour])
	}
	if err := p.startCook(dToast, 1); err == nil {
		t.Fatal("the counter cooked two things at once")
	}
	if _, err := p.collect(stCounter); err == nil {
		t.Fatal("collected before it was done")
	}
	p.tickKitchen(100, true)
	if _, err := p.collect(stCounter); err != nil || len(p.Pantry) != 1 {
		t.Fatal("could not collect the toast")
	}
	tap := p.tapDamage()
	if err := p.serve(0); err != nil {
		t.Fatal(err)
	}
	if p.tapDamage() <= tap*1.3 {
		t.Fatalf("perfect toast should add 37%% tap damage: %v -> %v", tap, p.tapDamage())
	}
	// a full course and a combo
	p.Table = append(p.Table, served{dish{dOmelette, 1}, dishTime}, served{dish{dPancakes, 1}, dishTime})
	if !p.fullCourse() || len(p.activeCombos()) != 1 {
		t.Fatalf("full course %v combos %v", p.fullCourse(), p.activeCombos())
	}
	e := p.effects()
	if e[efTap] < (25*1.5+0)*1.5+60-0.01 || e[efCrit] != 6*1.5 {
		t.Fatalf("effects %v", e)
	}
	if err := p.serve(0); err == nil {
		t.Fatal("served onto a full table")
	}
	p.tickKitchen(dishTime+1, true)
	if len(p.Table) != 0 {
		t.Fatal("dishes did not run out")
	}
	// stations
	p.Coins = 1e9
	if err := p.upgradeStation(stOven); err != nil || p.Stations[stOven] != 1 {
		t.Fatal("could not build the oven")
	}
	before := p.cookTime(dPizza)
	p.upgradeStation(stOven)
	if p.cookTime(dPizza) >= before || p.stationBonus(stOven) <= 1 {
		t.Fatal("upgrading the oven did nothing")
	}
	// a favourite dish doubles that chef
	d := p.dps(0)
	p.Table = []served{{dish{dToast, 1}, dishTime}}
	if p.dps(0) < d*1.99 {
		t.Fatal("Pip's favourite did not double his damage")
	}
}

func TestDrops(t *testing.T) {
	p := newProfile(0)
	if p.dropCount(false) != 1 || p.dropCount(true) < 4 {
		t.Fatal("drop counts")
	}
	p.Table = []served{{dish{dSundae, 1}, dishTime}}
	if p.dropCount(false) != 2 {
		t.Fatal("the sundae should double drops")
	}
	for _, fam := range areaMonsters {
		for _, m := range fam {
			if len(monsters[m].drop) == 0 {
				t.Fatal("a monster drops nothing")
			}
		}
	}
	// every ingredient can be found somewhere and every recipe can be made
	var found [numIngredients]bool
	for _, m := range monsters {
		for _, d := range m.drop {
			found[d] = true
		}
	}
	for i, f := range found {
		if !f {
			t.Fatalf("no monster drops %s", ingName[i])
		}
	}
}
