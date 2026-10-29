package game

import (
	"math/rand"
	"time"
)

// The client side of a match: what this game shows and does for its own player.
// When this game hosts (or plays alone with bots) it also runs the match (host).

var (
	host   *Match // the match this game runs, when hosting or playing solo
	view   *Match // the match as this game sees it (== host when hosting)
	myID   int
	myRun  *Run
	mySeed int64 // the seed myRun was dealt from
	myRnd  int   // the round number myRun is playing
	solo   bool  // playing alone with bots (no network)
)

type action struct {
	T      string     `json:"t"`
	Picks  []HandType `json:"pk,omitempty"`
	Round  int        `json:"rd,omitempty"`
	Score  int64      `json:"sc,omitempty"`
	Plays  int        `json:"pl,omitempty"`
	Done   bool       `json:"d,omitempty"`
	Item   int        `json:"i,omitempty"`
	Target int        `json:"tg,omitempty"`
	Price  int        `json:"p,omitempty"`
	Charm  string     `json:"c,omitempty"`
	Hand   HandType   `json:"h,omitempty"`
}

// Do something: the host handles it straight away, a guest sends it
func act(a action) {
	if host != nil {
		hostAct(myID, a)
	} else {
		netSend(0, mAct, a)
	}
}

// The host handles a player's action
func hostAct(from int, a action) {
	m := host
	p := m.player(from)
	if p == nil {
		return
	}
	switch a.T {
	case "pick":
		if m.Phase == "draft" && validPicks(p, a.Picks) {
			p.Picks, p.Ready = a.Picks, true
			m.changed()
		}
	case "unpick":
		if m.Phase == "draft" {
			p.Ready = false
			m.changed()
		}
	case "report":
		m.report(from, a.Round, a.Score, a.Plays, a.Done)
	case "bid":
		if a.Item >= 0 && a.Item < len(m.Auction) {
			prev := m.Auction[a.Item].Bidder
			if m.bid(from, a.Item, 0) {
				if prev >= 0 && prev != from {
					tell(prev, p.Name+" outbid you!", "outbid")
				}
			} else {
				tell(from, "You can't bid on that", "empty")
			}
		}
	case "ready":
		if m.Phase == "shop" {
			p.Ready = true
			m.changed()
		}
	case "unready":
		if m.Phase == "shop" {
			p.Ready = false
			m.changed()
		}
	case "tome":
		if !m.buyTome(from, a.Hand) {
			tell(from, "You can't buy that", "empty")
		}
	case "sell":
		if canTrade(m, p) {
			m.sell(from, a.Charm)
		}
	case "list":
		if canTrade(m, p) {
			m.list(from, a.Charm, a.Price)
		}
	case "unlist":
		m.unlistCharm(from, a.Charm)
	case "buyl":
		if !canTrade(m, p) || !m.buyListing(from, a.Item) {
			tell(from, "You can't buy that", "empty")
		}
	case "cast":
		if h, bounced, ok := m.castHex(from, a.Item, a.Target); ok {
			if bounced {
				tell(from, "Their Mirror Ward sent "+hexes[h].name+" back at you!", "hex")
				tell(a.Target, "Your Mirror Ward bounced "+hexes[h].name+" back!", "hex")
			} else {
				tell(from, "You cast "+hexes[h].name, "hex")
				tell(a.Target, p.Name+" hexed you with "+hexes[h].name+"!", "hex")
			}
		}
	case "rbuy":
		if !m.raceBuy(from, a.Item) {
			tell(from, "You can't buy that", "empty")
		}
	case "rcont":
		m.raceContinue(from)
	}
}

// Trading and selling happen between rounds, so a round's charms don't change mid-play
func canTrade(m *Match, p *PlayerState) bool {
	if m.Mode == ModeRace {
		return p.Stage == "shop"
	}
	return m.Phase == "shop" || m.Phase == "results"
}

func me() *PlayerState {
	if view == nil {
		return nil
	}
	return view.player(myID)
}

func inMatch() bool { return view != nil && view.Phase != "lobby" && view.Phase != "over" }

// Each frame: run the match if hosting, and keep this player's cards in step with it
func clientUpdate() {
	if host != nil {
		host.tick()
		view = host
	}
	p := me()
	if view == nil || p == nil {
		return
	}
	if p.Seed != 0 && (myRun == nil || mySeed != p.Seed) {
		myRun = NewRun(p.Seed, p.Family, p.Special)
		myRun.SortSuit = prefs.SortSuit
		mySeed, myRnd = p.Seed, 0
		draftPicks = nil
	}
	playing := view.Phase == "play" && !p.Out
	if view.Mode == ModeRace {
		playing = playing && p.Stage == "play"
	}
	if playing && myRun != nil && p.Round != myRnd {
		myRnd = p.Round
		syncMyRun(p)
		myRun.StartRound(p.Tg)
		anim = nil
		dealAt = frame
		sfx("deal")
	}
	// the table info charms use changes during a round (lives, hexes cast)
	if myRun != nil {
		myRun.Table = view.tableInfo(p)
	}
}

func syncMyRun(p *PlayerState) {
	r := myRun
	r.Charms = append([]string{}, p.Charms...)
	r.Levels = map[HandType]int{}
	for k, v := range p.Levels {
		r.Levels[k] = v
	}
	r.Hexes = append([]string{}, p.Active...)
	r.Frozen = p.Frozen
	r.Picks = append([]HandType{}, p.Picks...)
	r.Table = view.tableInfo(p)
}

// Tell the host how this round is going
func reportRun() {
	if myRun == nil {
		return
	}
	act(action{T: "report", Round: myRnd, Score: myRun.Score, Plays: myRun.Plays, Done: myRun.Done})
}

// ---- Starting and leaving ----

func newSeed() int64 { return time.Now().UnixNano()&0x7fffffffffff + rand.Int63n(1000) }

// Play alone against bots
func startSolo() {
	netShutdown()
	solo = true
	host = newLobby(playerName(), newSeed())
	applyLobbyPrefs()
	for len(host.Players) < 2 {
		host.addBot()
	}
	view, myID, myRun = host, 0, nil
	openScreen(scTable)
}

// Host a table for friends (bots can fill empty seats)
func startHost(online bool) {
	solo = false
	host = newLobby(playerName(), newSeed())
	applyLobbyPrefs()
	view, myID, myRun = host, 0, nil
	if online {
		startHostingOnline()
	} else {
		startHostingLAN()
	}
	openScreen(scTable)
}

func applyLobbyPrefs() {
	host.Mode, host.Length, host.MaxLives = prefs.Mode, prefs.Length, prefs.Lives
}

func leaveTable(reason string) {
	netShutdown()
	host, view, myRun, solo = nil, nil, nil, false
	anim, ov, netStatus = nil, ovNone, ""
	if reason != "" {
		toast(reason)
	}
	openScreen(scMain)
}

// Back to the lobby with the same players, for another match
func rematch() {
	if host == nil {
		return
	}
	m := host
	m.Phase = "lobby"
	m.Seed = newSeed()
	m.rng = rand.New(rand.NewSource(m.Seed))
	m.bots = map[int]*Run{}
	m.Winner, m.Results, m.Auction, m.Market = "", nil, nil, nil
	var keep []*PlayerState
	for _, p := range m.Players {
		if !p.Gone {
			keep = append(keep, &PlayerState{ID: p.ID, Name: p.Name, Bot: p.Bot})
		}
	}
	m.Players = keep
	m.changed()
}
