package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func testGame(t *testing.T) *Game {
	t.Helper()
	os.Setenv("BLOB_NOAUDIO", "1")
	os.Setenv("BLOB_FRESH", "1")
	DataDir = t.TempDir()
	g := NewGame()
	g.W, g.H, g.u = 800, 360, 1
	g.m = g.computeMods()
	return g
}

// Mutations stack: their multipliers multiply, and the shown chance multiplies too.
func TestStackedValue(t *testing.T) {
	g := testGame(t)
	m := 1<<mGold | 1<<mShadow
	base := g.blobValue(0)
	want := base * rarities[mGold].Mult * rarities[mShadow].Mult
	if got := g.blobValue(m); math.Abs(got-want) > 1e-9 {
		t.Fatalf("gold+shadow value %v, want %v", got, want)
	}
	if comboOdds(m) != rarities[mGold].Odds*rarities[mShadow].Odds {
		t.Fatalf("combo odds %v", comboOdds(m))
	}
	if maskTop(m) != mShadow || maskCount(m) != 2 {
		t.Fatal("mask helpers")
	}
}

// A child always keeps its parent's mutations (except under The Mirror).
func TestChildInherits(t *testing.T) {
	g := testGame(t)
	p := 1<<mFrost | 1<<mGold
	for i := 0; i < 200; i++ {
		if c := g.childMask(p); c&p != p {
			t.Fatalf("child %b lost parent %b's mutations", c, p)
		}
	}
	g.st.Round, g.st.Boss, g.st.Active = 2, 4, true // The Mirror
	g.m = g.computeMods()
	lost := false
	for i := 0; i < 200; i++ {
		if g.childMask(p)&p != p {
			lost = true
		}
	}
	if !lost {
		t.Fatal("The Mirror should stop children inheriting")
	}
}

func TestRounds(t *testing.T) {
	g := testGame(t)
	s := &g.st
	// between rounds, sales don't count
	g.roundSale(roundTarget(1, 0) * 10)
	if s.Round != 0 || s.Market {
		t.Fatal("a sale with no round running should not clear it")
	}
	// failing: run out of sells
	g.startRound()
	for i := 0; i < g.sellsAllowed(); i++ {
		g.roundSale(1)
	}
	if s.RoundSells != 0 || s.RoundEarned != 0 || s.Round != 0 || s.Active {
		t.Fatalf("a failed round should start over: %+v", s.RoundSells)
	}
	// winning Petri opens the market and pays DNA
	g.startRound()
	g.roundSale(roundTarget(1, 0))
	if s.Round != 1 || !s.Market || s.Gems != roundReward(0) || len(s.Offers) != 3 {
		t.Fatalf("after Petri: round %d market %v gems %d", s.Round, s.Market, s.Gems)
	}
	g.closeMarket()
	if s.Active {
		t.Fatal("the next round should wait to be started")
	}
	g.startRound()
	g.roundSale(roundTarget(1, 1))
	g.closeMarket()
	// the Outbreak gives a card and moves to the next ante
	g.startRound()
	g.m = g.computeMods()
	g.roundSale(roundTarget(1, 2))
	if s.Ante != 2 || s.Round != 0 || len(s.Cards) != 1 || s.RunBosses != 1 {
		t.Fatalf("after the Outbreak: ante %d round %d cards %v", s.Ante, s.Round, s.Cards)
	}
}

func TestMarketAndTree(t *testing.T) {
	g := testGame(t)
	s := &g.st
	s.Gems = 100
	s.Offers = []string{"clover", "midas", "harvest"}
	g.buyOffer(0)
	if len(s.Cards) != 1 || s.Gems != 100-cardRarities[0].price || s.Offers[0] != "" {
		t.Fatalf("buying: %v %d", s.Cards, s.Gems)
	}
	g.m = g.computeMods()
	if g.m.luck != 1.3 {
		t.Fatalf("clover luck %v", g.m.luck)
	}
	// the tree: nodes need the one above
	s.Points = 10
	g.unlock("luck1")
	if g.has("luck1") {
		t.Fatal("luck1 needs origin first")
	}
	g.unlock("origin")
	g.unlock("luck1")
	if !g.has("origin") || !g.has("luck1") || s.Points != 8 {
		t.Fatalf("tree: %v points %d", s.Tree, s.Points)
	}
	for _, n := range tree {
		if n.req != "" && nodeByID(n.req) == nil {
			t.Fatalf("%s needs unknown node %s", n.id, n.req)
		}
	}
	for _, c := range cards {
		if c.rar < 0 || c.rar >= len(cardRarities) {
			t.Fatalf("card %s rarity", c.id)
		}
	}
}

// Rebirth pays a point plus one per Outbreak beaten, and keeps cards and skills.
func TestRebirth(t *testing.T) {
	g := testGame(t)
	s := &g.st
	s.Money, s.RunBosses, s.Cards, s.Ante = rebirthReq(0), 2, []string{"midas"}, 3
	s.Tree["origin"] = true
	g.rebirth()
	if s.Points != 3 || s.Rebirths != 1 || s.Ante != 1 || len(s.Cards) != 1 || !g.has("origin") {
		t.Fatalf("rebirth: points %d ante %d cards %v", s.Points, s.Ante, s.Cards)
	}
}

// Saves from Blob Split (single rarities, gem shop) load into Biote.
func TestOldSave(t *testing.T) {
	DataDir = t.TempDir()
	os.Unsetenv("BLOB_FRESH")
	defer os.Setenv("BLOB_FRESH", "1")
	old := map[string]any{"money": 500, "gems": 7, "rebirths": 2, "blobs": []int{0, 1, 3, 0}, "gemLv": map[string]int{"auto": 1}, "auto": true, "found": []int{5, 3, 0, 1, 0, 0, 0, 0}}
	b, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(DataDir, "save.json"), b, 0o644)
	st := loadState()
	if len(st.Board) != 4 || st.Board[1] != 1<<1 || st.Board[2] != 1<<3 || st.Gems != 7 || st.Points != 4 || st.Auto || st.Ver != stateVer {
		t.Fatalf("migrated: %+v", st)
	}
}

// Events and day/night change the chances.
func TestWorldMods(t *testing.T) {
	g := testGame(t)
	g.st.Clock = dayLen * 0.75 // midnight
	g.m = g.computeMods()
	night := g.gainChance(mShadow)
	g.st.Clock = dayLen * 0.25 // midday
	g.m = g.computeMods()
	if day := g.gainChance(mShadow); !(night > day) {
		t.Fatalf("shadow should be likelier at night: %v vs %v", night, day)
	}
	before := g.gainChance(mFrost)
	g.st.Event = "blizzard"
	g.m = g.computeMods()
	if g.gainChance(mFrost) <= before {
		t.Fatal("blizzard should raise frost")
	}
	g.st.Event = "heatwave"
	g.m = g.computeMods()
	if g.gainChance(mFrost) != 0 {
		t.Fatal("heatwave should stop frost")
	}
}

// Every skill sits on its own hexagon, right next to the skill it needs.
func TestTreeHexes(t *testing.T) {
	seen := map[[2]int]string{}
	for _, n := range tree {
		k := [2]int{n.q, n.r}
		if o, ok := seen[k]; ok {
			t.Fatalf("%s and %s share a hexagon", n.id, o)
		}
		seen[k] = n.id
		if n.req == "" {
			continue
		}
		p := nodeByID(n.req)
		dq, dr := n.q-p.q, n.r-p.r
		if !((dq == 1 && dr == 0) || (dq == -1 && dr == 0) || (dq == 0 && dr == 1) || (dq == 0 && dr == -1) || (dq == 1 && dr == -1) || (dq == -1 && dr == 1)) {
			t.Fatalf("%s is not next to %s", n.id, n.req)
		}
	}
}
