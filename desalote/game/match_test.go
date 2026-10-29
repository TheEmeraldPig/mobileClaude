package game

import "testing"

// Whole matches between bots, in every mode, must finish
func TestBotMatchesFinish(t *testing.T) {
	for mode := 0; mode < numModes; mode++ {
		for length := 0; length < len(lengthAntes); length++ {
			for seed := int64(1); seed <= 4; seed++ {
				m := newLobby("host", seed)
				m.Mode, m.Length = mode, length
				m.Players[0].Bot = true
				n := 4
				if mode == ModeDuel {
					n = 2
				}
				for len(m.Players) < n {
					m.addBot()
				}
				m.start()
				f := 0
				for ; f < 60*60*120 && m.Phase != "over"; f++ {
					m.tick()
				}
				if m.Phase != "over" {
					t.Fatalf("%s %s seed %d: still %q at ante %d after %d frames", modeNames[mode], lengthNames[length], seed, m.Phase, m.Ante, f)
				}
				t.Logf("%-13s %-8s seed %d: %s (ante %d, %d min)", modeNames[mode], lengthNames[length], seed, m.Winner, m.Ante+1, f/3600)
			}
		}
	}
}

func testMatch(mode int) *Match {
	m := newLobby("a", 3)
	m.Mode = mode
	m.addPlayer(1, "b", false)
	m.addPlayer(2, "c", false)
	m.start()
	for _, p := range m.Players {
		p.Picks = p.Offer[:3]
		p.Ready = true
	}
	return m
}

func TestHexes(t *testing.T) {
	m := testMatch(ModeLastStanding)
	a, b := m.player(0), m.player(1)
	a.Hexes = []string{"squeeze", "pickpocket", "rust"}
	if h, bounced, ok := m.castHex(0, 0, 1); !ok || bounced || h != "squeeze" {
		t.Fatal("cast failed")
	}
	if len(b.OnMe) != 1 || b.OnMe[0] != "squeeze" {
		t.Fatalf("squeeze not waiting on b: %v", b.OnMe)
	}
	// pickpocket happens now
	b.Money, a.Money = 5, 0
	m.castHex(0, 0, 1)
	if b.Money != 2 || a.Money != 3 {
		t.Fatalf("pickpocket: a $%d b $%d", a.Money, b.Money)
	}
	// a mirror ward bounces the next hex and breaks
	b.Charms = []string{"mirror"}
	if _, bounced, _ := m.castHex(0, 0, 1); !bounced || len(a.OnMe) != 1 || hasCharm(b, "mirror") {
		t.Fatalf("mirror: bounced %v, a.OnMe %v", bounced, a.OnMe)
	}
	// hexes take effect in the next round, then are gone
	m.tick()
	if m.Phase != "play" {
		t.Fatalf("phase %s", m.Phase)
	}
	if len(b.Active) != 1 || len(b.OnMe) != 0 {
		t.Fatalf("b active %v on %v", b.Active, b.OnMe)
	}
	r := NewRun(b.Seed, b.Family, b.Special)
	m.syncRun(r, b)
	r.StartRound(100)
	if r.HandSize != baseHandSize-1 {
		t.Fatalf("squeeze: hand size %d", r.HandSize)
	}
	// no hexes in co-op
	c := testMatch(ModeCoop)
	c.player(0).Hexes = []string{"fog"}
	if _, _, ok := c.castHex(0, 0, 1); ok {
		t.Fatal("hexed a teammate")
	}
}

func TestAuctionAndMarket(t *testing.T) {
	m := testMatch(ModeLastStanding)
	m.tick() // into the first blind
	m.openShop()
	a, b := m.player(0), m.player(1)
	a.Money, b.Money = 10, 10
	m.Auction[0] = ShopItem{Kind: "charm", ID: "sieve", Price: 4, Bidder: -1}
	if !m.bid(0, 0, 0) || m.Auction[0].Bid != 4 {
		t.Fatal("opening bid")
	}
	if m.bid(0, 0, 0) {
		t.Fatal("raised own bid")
	}
	if !m.bid(1, 0, 0) || m.Auction[0].Bid != 5 || m.Auction[0].Bidder != 1 {
		t.Fatal("outbid")
	}
	// can't promise more than you have across items
	m.Auction[1] = ShopItem{Kind: "charm", ID: "echo", Price: 6, Bidder: -1}
	if m.bid(1, 1, 0) {
		t.Fatal("bid beyond money")
	}
	b.Charms = []string{"haggler"}
	m.closeAuction()
	if !hasCharm(b, "sieve") || b.Money != 7 { // $5 - $2 haggler
		t.Fatalf("b: %v $%d", b.Charms, b.Money)
	}
	// market
	m.list(1, "sieve", 6)
	if len(m.Market) != 1 {
		t.Fatal("not listed")
	}
	if !m.buyListing(0, m.Market[0].ID) || !hasCharm(a, "sieve") || hasCharm(b, "sieve") || a.Money != 4 || b.Money != 13 {
		t.Fatalf("trade: a %v $%d, b %v $%d", a.Charms, a.Money, b.Charms, b.Money)
	}
	// tomes
	a.Money = 10
	h := a.Picks[0]
	if !m.buyTome(0, h) || a.Levels[h] != 1 || m.buyTome(0, h) {
		t.Fatal("tome")
	}
}

func TestShowdownLosesLowest(t *testing.T) {
	m := testMatch(ModeDuel)
	m.Players = m.Players[:2]
	m.tick()
	m.Showdown, m.Target = true, 0
	m.player(0).Score, m.player(1).Score = 500, 900
	for _, p := range m.Players {
		p.Done = true
	}
	m.resolveBlind()
	if m.player(0).Lives != m.MaxLives-1 || m.player(1).Lives != m.MaxLives {
		t.Fatalf("lives %d %d", m.player(0).Lives, m.player(1).Lives)
	}
}
