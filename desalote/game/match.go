package game

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// The match: modes, rounds, lives, the auction shop, the charm market and hexes.
//
// The host's game runs the match (this file) and sends the whole Match to everyone
// whenever it changes. Each game runs its own player's cards (run.go) and reports
// scores back. Bots are run here by the host.

const (
	ModeDuel = iota
	ModeLastStanding
	ModeCoop
	ModeRace
	numModes
)

var modeNames = [numModes]string{"Duel", "Last Standing", "Co-op Heist", "Race"}
var modeBlurbs = [numModes]string{
	"1 v 1. Small blinds are against the house; Big and Boss blinds are showdowns: the lower score loses a life.",
	"2-4 players. Miss a blind and lose a life; every Boss blind is a showdown where the lowest score loses a life. Last one standing wins.",
	"Team up. Your scores add up against huge blinds and the team shares its lives. Clear every ante to win.",
	"Everyone plays at their own pace. First to beat the final Boss wins. Hex the leaders to slow them down.",
}

var lengthNames = []string{"Short", "Standard", "Long"}
var lengthAntes = []int{3, 5, 8}

var blindNames = [3]string{"Small Blind", "Big Blind", "Boss Blind"}

const botID0 = 10 // bots' player ids start here

// blindTarget is the points needed to beat a blind on its own (ante from 0)
func blindTarget(ante, blind int) int64 {
	base := 800 * math.Pow(1.78, float64(ante))
	t := base * []float64{1, 1.35, 1.8}[blind]
	// round to a nice number
	step := math.Pow(10, math.Floor(math.Log10(t))-1)
	return int64(math.Round(t/step) * step)
}

type ShopItem struct {
	Kind   string   `json:"k"` // charm, hex
	ID     string   `json:"id"`
	Price  int      `json:"p"`
	Bid    int      `json:"b"`
	Bidder int      `json:"w"` // -1: no bids
	Sold   bool     `json:"s"`
	Hand   HandType `json:"h"`
}

type Listing struct {
	ID     int    `json:"id"`
	Seller int    `json:"s"`
	Charm  string `json:"c"`
	Price  int    `json:"p"`
}

type PlayerState struct {
	ID      int              `json:"id"`
	Name    string           `json:"n"`
	Bot     bool             `json:"bot,omitempty"`
	Gone    bool             `json:"gone,omitempty"`
	Lives   int              `json:"l"`
	Money   int              `json:"m"`
	Charms  []string         `json:"c"`
	Hexes   []string         `json:"x"` // owned, not cast yet
	Levels  map[HandType]int `json:"lv"`
	Family  Family           `json:"f"`
	Special HandType         `json:"sp"`
	Offer   []HandType       `json:"of"`
	Picks   []HandType       `json:"pk"`
	Seed    int64            `json:"sd"`
	Score   int64            `json:"sc"` // this round
	Total   int64            `json:"t"`  // the whole match
	Plays   int              `json:"pl"`
	Done    bool             `json:"d"`
	Ready   bool             `json:"r"`
	Out     bool             `json:"o"`
	OnMe    []string         `json:"om"` // hexes waiting for my next round
	Active  []string         `json:"ac"` // hexes on me this round
	Frozen  string           `json:"fz"`
	Cast    int              `json:"hc"` // hexes cast this match
	Tomes   []HandType       `json:"tb"` // tomes bought this shop
	// Race: each player's own progress
	Ante  int        `json:"a"`
	Blind int        `json:"bl"`
	Stage string     `json:"st"` // play, shop, out, won
	Shop  []ShopItem `json:"sh"`
	Round int        `json:"rd"` // counts this player's rounds (a new number = a new deal)
	Tg    int64      `json:"tg"` // this round's target for their own run (0: play every hand)
	Last  string     `json:"ls"` // what happened to them last round
}

type Match struct {
	Mode     int            `json:"mode"`
	Length   int            `json:"len"`
	MaxLives int            `json:"maxl"`
	Phase    string         `json:"ph"` // lobby, draft, play, results, shop, over
	Ante     int            `json:"a"`
	Blind    int            `json:"b"`
	Round    int            `json:"rd"`
	Target   int64          `json:"tg"`
	Showdown bool           `json:"sd"`
	Secs     int            `json:"secs"`
	Players  []*PlayerState `json:"pl"`
	Auction  []ShopItem     `json:"au"`
	Market   []Listing      `json:"mk"`
	Log      []string       `json:"log"`
	Results  []string       `json:"res"`
	Winner   string         `json:"win"`
	Seed     int64          `json:"seed"`
	Team     int            `json:"team"` // co-op: the team's lives
	Rev      int            `json:"rev"`

	timer    int // frames
	nextList int
	rng      *rand.Rand
	bots     map[int]*Run
	botWait  map[int]int
}

func (m *Match) player(id int) *PlayerState {
	for _, p := range m.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (m *Match) alive() []*PlayerState {
	var out []*PlayerState
	for _, p := range m.Players {
		if !p.Out && !p.Gone {
			out = append(out, p)
		}
	}
	return out
}

func (m *Match) logf(f string, a ...any) {
	m.Log = append(m.Log, fmt.Sprintf(f, a...))
	if len(m.Log) > 8 {
		m.Log = m.Log[len(m.Log)-8:]
	}
}

func (m *Match) changed() { m.Rev++ }

// ---- Host: setting up ----

func newLobby(hostName string, seed int64) *Match {
	m := &Match{Mode: ModeLastStanding, Length: 1, MaxLives: 3, Phase: "lobby", Seed: seed,
		rng: rand.New(rand.NewSource(seed)), bots: map[int]*Run{}, botWait: map[int]int{}}
	m.Players = []*PlayerState{{ID: 0, Name: hostName}}
	return m
}

func (m *Match) addPlayer(id int, name string, bot bool) {
	if p := m.player(id); p != nil {
		p.Name, p.Gone = name, false
		return
	}
	m.Players = append(m.Players, &PlayerState{ID: id, Name: name, Bot: bot})
	m.changed()
}

func (m *Match) removePlayer(id int) {
	if m.Phase == "lobby" {
		for i, p := range m.Players {
			if p.ID == id {
				m.Players = append(m.Players[:i], m.Players[i+1:]...)
				break
			}
		}
	} else if p := m.player(id); p != nil {
		p.Gone, p.Done, p.Ready = true, true, true
		m.logf("%s left", p.Name)
	}
	m.changed()
}

var botNames = []string{"Vesper", "Quill", "Marrow", "Tansy", "Bramble", "Cobalt", "Wren", "Juniper"}

func (m *Match) addBot() {
	id := botID0
	for m.player(id) != nil {
		id++
	}
	m.addPlayer(id, botNames[(id-botID0)%len(botNames)], true)
}

func (m *Match) maxPlayers() int {
	if m.Mode == ModeDuel {
		return 2
	}
	return 4
}

func (m *Match) canStart() bool {
	n := len(m.Players)
	return n >= 2 && n <= m.maxPlayers()
}

// Start: deal each player a family, a special hand, a deck and hands to pick from
func (m *Match) start() {
	m.Phase, m.Ante, m.Blind, m.Round = "draft", 0, 0, 0
	m.Team = m.MaxLives
	fams := m.rng.Perm(3)
	sps := m.rng.Perm(3)
	for i, p := range m.Players {
		p.Lives, p.Money, p.Charms, p.Hexes = m.MaxLives, 4, nil, nil
		p.Levels = map[HandType]int{}
		p.Family = Family(fams[i%3])
		p.Special = specialHands[sps[i%3]]
		p.Seed = m.Seed*31 + int64(p.ID)*7919 + 1
		p.Total, p.Out, p.Ready, p.Done, p.Cast = 0, false, false, false, 0
		p.Ante, p.Blind, p.Stage = 0, 0, "play"
		// offer three hands of their family and two from the others
		own := familyHands(p.Family)
		m.rng.Shuffle(len(own), func(a, b int) { own[a], own[b] = own[b], own[a] })
		var other []HandType
		for f := FamSame; f <= FamSuits; f++ {
			if f != p.Family {
				other = append(other, familyHands(f)...)
			}
		}
		m.rng.Shuffle(len(other), func(a, b int) { other[a], other[b] = other[b], other[a] })
		p.Offer = append(append([]HandType{}, own[:3]...), other[:2]...)
		p.Picks = nil
		if p.Bot {
			// bots stay in their family (most synergy)
			p.Picks = append([]HandType{}, own[:3]...)
			p.Ready = true
			r := NewRun(p.Seed, p.Family, p.Special)
			r.Picks = p.Picks
			m.bots[p.ID] = r
		}
	}
	m.Log = nil
	m.logf("The match begins: %s, %s", modeNames[m.Mode], lengthNames[m.Length])
	m.changed()
}

// validPicks: three different hands from the offer, at least one from your family
func validPicks(p *PlayerState, picks []HandType) bool {
	if len(picks) != 3 {
		return false
	}
	own := false
	seen := map[HandType]bool{}
	for _, h := range picks {
		ok := false
		for _, o := range p.Offer {
			ok = ok || o == h
		}
		if !ok || seen[h] {
			return false
		}
		seen[h] = true
		own = own || hands[h].fam == p.Family
	}
	return own
}

// ---- Host: rounds ----

func (m *Match) isShowdown(blind int) bool {
	switch m.Mode {
	case ModeDuel:
		return blind >= 1
	case ModeLastStanding:
		return blind == 2
	}
	return false
}

func (m *Match) targetFor(ante, blind int) int64 {
	t := blindTarget(ante, blind)
	if m.Mode == ModeCoop {
		n := int64(len(m.alive()))
		t = t * n * 13 / 10
	}
	return t
}

func (m *Match) startBlind() {
	m.Phase = "play"
	m.Round++
	m.Showdown = m.isShowdown(m.Blind)
	m.Target = m.targetFor(m.Ante, m.Blind)
	if m.Showdown {
		m.Target = 0
	}
	for _, p := range m.alive() {
		m.beginPlayerRound(p)
	}
	m.Results = nil
	m.changed()
}

// A player's next round: their waiting hexes take effect
func (m *Match) beginPlayerRound(p *PlayerState) {
	p.Score, p.Done, p.Ready, p.Plays = 0, false, false, basePlays
	p.Round++
	p.Active, p.OnMe, p.Frozen = p.OnMe, nil, ""
	for _, h := range p.Active {
		if h == "freeze" && len(p.Charms) > 0 {
			p.Frozen = p.Charms[m.rng.Intn(len(p.Charms))]
		}
	}
	p.Tg = m.Target
	if m.Mode == ModeRace {
		p.Tg = m.targetFor(p.Ante, p.Blind)
	} else if m.Mode == ModeCoop {
		p.Tg = 0 // in co-op everyone plays every hand for the team
	}
	if r := m.bots[p.ID]; r != nil {
		m.syncRun(r, p)
		r.StartRound(p.Tg)
		m.botWait[p.ID] = 50 + m.rng.Intn(40)
	}
}

// Give a run what the match knows about its player
func (m *Match) syncRun(r *Run, p *PlayerState) {
	r.Charms = append([]string{}, p.Charms...)
	r.Levels = map[HandType]int{}
	for k, v := range p.Levels {
		r.Levels[k] = v
	}
	r.Hexes = append([]string{}, p.Active...)
	r.Frozen = p.Frozen
	r.Picks = p.Picks
	r.Table = m.tableInfo(p)
}

func (m *Match) tableInfo(p *PlayerState) TableInfo {
	t := TableInfo{HexesCast: p.Cast}
	under := true
	others := 0
	for _, q := range m.Players {
		if q.ID == p.ID {
			continue
		}
		t.RivalLivesLost += m.MaxLives - max(0, q.Lives)
		if !q.Out && !q.Gone {
			others++
			if q.Lives <= p.Lives {
				under = false
			}
		}
	}
	t.Others = others
	t.Underdog = under && others > 0 && m.Mode != ModeCoop
	return t
}

// A player's score report
func (m *Match) report(id int, round int, score int64, plays int, done bool) {
	p := m.player(id)
	if p == nil || p.Out || round != p.Round {
		return
	}
	if m.Mode == ModeRace {
		if p.Stage != "play" {
			return
		}
	} else if m.Phase != "play" {
		return
	}
	p.Score, p.Plays = score, plays
	if done && !p.Done {
		p.Done = true
		if m.Mode == ModeRace {
			m.raceFinishBlind(p)
		}
	}
	m.changed()
}

// Everyone is done: who passed, who loses lives, who gets paid
func (m *Match) resolveBlind() {
	m.Results = nil
	alive := m.alive()
	pay := func(p *PlayerState, n int, why string) {
		if hasCharm(p, "purse") {
			n += 2
		}
		p.Money += n
		p.Last = fmt.Sprintf("%s  +$%d", why, n)
	}
	for _, p := range alive {
		p.Total += p.Score
	}
	switch {
	case m.Mode == ModeCoop:
		var sum int64
		for _, p := range alive {
			sum += p.Score
		}
		if sum >= m.Target {
			m.Results = append(m.Results, fmt.Sprintf("The team scored %s of %s: blind beaten!", commas(sum), commas(m.Target)))
			for _, p := range alive {
				pay(p, 4+m.Blind, "Team win")
			}
		} else {
			m.Team--
			m.Results = append(m.Results, fmt.Sprintf("The team scored %s of %s: lost a life!", commas(sum), commas(m.Target)))
			for _, p := range alive {
				pay(p, 2, "Team loss")
			}
		}
		for _, p := range m.Players {
			p.Lives = m.Team
		}
	case m.Showdown:
		low := int64(math.MaxInt64)
		high := int64(-1)
		for _, p := range alive {
			low = min(low, p.Score)
			high = max(high, p.Score)
		}
		for _, p := range alive {
			switch {
			case low == high:
				pay(p, 4, "Tie")
			case p.Score == low:
				p.Lives--
				pay(p, 3, "Lost the showdown")
				m.Results = append(m.Results, fmt.Sprintf("%s had the lowest score (%s) and loses a life", p.Name, commas(p.Score)))
			case p.Score == high:
				pay(p, 6, "Won the showdown")
				m.Results = append(m.Results, fmt.Sprintf("%s wins the showdown with %s", p.Name, commas(p.Score)))
			default:
				pay(p, 4, "Survived")
			}
		}
		if low == high {
			m.Results = append(m.Results, "A tie: nobody loses a life")
		}
	default:
		for _, p := range alive {
			if p.Score >= m.Target {
				pay(p, 3+m.Blind+max(0, p.Plays), "Blind beaten")
			} else {
				p.Lives--
				pay(p, 2, "Missed the blind")
				m.Results = append(m.Results, fmt.Sprintf("%s missed the blind (%s of %s) and loses a life", p.Name, commas(p.Score), commas(m.Target)))
			}
		}
		if len(m.Results) == 0 {
			m.Results = append(m.Results, "Everyone beat the blind")
		}
	}
	for _, p := range alive {
		if p.Lives <= 0 && !p.Out {
			p.Out = true
			m.Results = append(m.Results, p.Name+" is out!")
			m.logf("%s is out", p.Name)
		}
	}
	m.Phase = "results"
	m.timer = 60 * 5
	m.checkOver(true)
	m.changed()
}

// Is the match over? (after a blind; last = the blind just played was checked)
func (m *Match) checkOver(afterBlind bool) {
	if m.Phase == "over" {
		return
	}
	lastBlind := afterBlind && m.Ante == lengthAntes[m.Length]-1 && m.Blind == 2
	switch m.Mode {
	case ModeCoop:
		if m.Team <= 0 {
			m.end("The house wins. The team ran out of lives.")
		} else if lastBlind {
			m.end("The heist is done: the team wins!")
		}
	case ModeRace:
		// handled per player
	default:
		alive := m.alive()
		humans := 0
		for _, p := range m.Players {
			if !p.Gone {
				humans++
			}
		}
		if len(alive) == 1 && humans > 1 {
			m.end(alive[0].Name + " is the last one standing!")
		} else if len(alive) == 0 {
			best := m.leader(m.Players)
			m.end(best.Name + " wins on points")
		} else if lastBlind {
			best := m.leader(alive)
			m.end(best.Name + " wins with the most lives left!")
		}
	}
}

// The player with the most lives, then the most points
func (m *Match) leader(ps []*PlayerState) *PlayerState {
	s := append([]*PlayerState{}, ps...)
	sort.SliceStable(s, func(a, b int) bool {
		if s[a].Lives != s[b].Lives {
			return s[a].Lives > s[b].Lives
		}
		return s[a].Total > s[b].Total
	})
	return s[0]
}

func (m *Match) end(msg string) {
	m.Phase = "over"
	m.Winner = msg
	m.logf("%s", msg)
	m.changed()
}

// After the results: on to the shop (and the next blind)
func (m *Match) nextBlind() {
	m.Blind++
	if m.Blind > 2 {
		m.Blind = 0
		m.Ante++
	}
}

// ---- Host: the shop ----

func (m *Match) openShop() {
	m.Phase = "shop"
	m.timer = 60 * 30
	m.Auction = nil
	skip := map[string]bool{}
	for i := 0; i < 3; i++ {
		id := randomCharm(m.rng, skip)
		skip[id] = true
		m.Auction = append(m.Auction, ShopItem{Kind: "charm", ID: id, Price: charms[id].price, Bidder: -1})
	}
	if m.Mode == ModeCoop {
		id := randomCharm(m.rng, skip)
		m.Auction = append(m.Auction, ShopItem{Kind: "charm", ID: id, Price: charms[id].price, Bidder: -1})
	} else {
		id := hexOrder[m.rng.Intn(len(hexOrder))]
		m.Auction = append(m.Auction, ShopItem{Kind: "hex", ID: id, Price: hexes[id].price, Bidder: -1})
	}
	for _, p := range m.Players {
		p.Ready, p.Tomes = p.Out || p.Gone, nil
		if p.Bot && !p.Out {
			m.botWait[p.ID] = 90 + m.rng.Intn(120)
		}
	}
	m.changed()
}

// Money a player has promised in bids they're winning
func (m *Match) committed(id int, except int) int {
	n := 0
	for i, it := range m.Auction {
		if i != except && !it.Sold && it.Bidder == id {
			n += it.Bid
		}
	}
	return n
}

func (m *Match) bid(id, item, amount int) bool {
	p := m.player(id)
	if m.Phase != "shop" || p == nil || p.Out || item < 0 || item >= len(m.Auction) {
		return false
	}
	it := &m.Auction[item]
	if it.Sold || it.Bidder == id {
		return false
	}
	need := max(it.Price, it.Bid+1)
	if amount < need {
		amount = need
	}
	if amount > p.Money-m.committed(id, item) {
		return false
	}
	if it.Kind == "charm" && len(p.Charms) >= maxCharms {
		return false
	}
	if prev := m.player(it.Bidder); prev != nil && !prev.Bot {
		prev.Ready = false // outbid: they get to answer
	}
	it.Bid, it.Bidder = amount, id
	p.Ready = false
	// a late bid gives the others a moment to answer
	if m.timer < 60*5 {
		m.timer = 60 * 5
	}
	m.changed()
	return true
}

func (m *Match) closeAuction() {
	for i := range m.Auction {
		it := &m.Auction[i]
		if it.Sold || it.Bidder < 0 {
			continue
		}
		p := m.player(it.Bidder)
		if p == nil {
			continue
		}
		cost := it.Bid
		if hasCharm(p, "haggler") {
			cost = max(1, cost-2)
		}
		if p.Money < cost || (it.Kind == "charm" && len(p.Charms) >= maxCharms) {
			continue
		}
		p.Money -= cost
		it.Sold = true
		switch it.Kind {
		case "charm":
			p.Charms = append(p.Charms, it.ID)
			m.logf("%s won %s for $%d", p.Name, charms[it.ID].name, cost)
		case "hex":
			p.Hexes = append(p.Hexes, it.ID)
			m.logf("%s won the hex %s for $%d", p.Name, hexes[it.ID].name, cost)
		}
		for _, q := range m.Players {
			if q.ID != p.ID && hasCharm(q, "magpie") {
				q.Money++
			}
		}
	}
	m.changed()
}

// A tome levels up one of your picked hands, at a set price (once per hand per shop)
func tomePrice(p *PlayerState, h HandType) int { return 3 + p.Levels[h] }

func (m *Match) buyTome(id int, h HandType) bool {
	p := m.player(id)
	if p == nil || p.Out {
		return false
	}
	if m.Mode == ModeRace {
		if p.Stage != "shop" {
			return false
		}
	} else if m.Phase != "shop" {
		return false
	}
	picked := false
	for _, x := range p.Picks {
		picked = picked || x == h
	}
	for _, x := range p.Tomes {
		if x == h {
			return false
		}
	}
	price := tomePrice(p, h)
	if !picked || p.Money-m.committed(id, -1) < price {
		return false
	}
	p.Money -= price
	p.Levels[h]++
	p.Tomes = append(p.Tomes, h)
	m.changed()
	return true
}

func hasCharm(p *PlayerState, id string) bool {
	for _, c := range p.Charms {
		if c == id {
			return true
		}
	}
	return false
}

func removeCharm(p *PlayerState, id string) bool {
	for i, c := range p.Charms {
		if c == id {
			p.Charms = append(p.Charms[:i], p.Charms[i+1:]...)
			return true
		}
	}
	return false
}

func sellPrice(id string) int { return max(1, charms[id].price/2) }

// Sell a charm back to the bank for half its price
func (m *Match) sell(id int, charm string) {
	p := m.player(id)
	if p == nil || charms[charm] == nil || !removeCharm(p, charm) {
		return
	}
	p.Money += sellPrice(charm)
	m.unlistCharm(id, charm)
	m.changed()
}

// Put a charm on the market for other players to buy
func (m *Match) list(id int, charm string, price int) {
	p := m.player(id)
	if p == nil || !hasCharm(p, charm) || price < 1 || price > 99 {
		return
	}
	m.unlistCharm(id, charm)
	m.nextList++
	m.Market = append(m.Market, Listing{ID: m.nextList, Seller: id, Charm: charm, Price: price})
	m.logf("%s is selling %s for $%d", p.Name, charms[charm].name, price)
	m.changed()
}

func (m *Match) unlistCharm(seller int, charm string) {
	for i, l := range m.Market {
		if l.Seller == seller && l.Charm == charm {
			m.Market = append(m.Market[:i], m.Market[i+1:]...)
			m.changed()
			return
		}
	}
}

func (m *Match) buyListing(id, listing int) bool {
	b := m.player(id)
	if b == nil || b.Out {
		return false
	}
	for i, l := range m.Market {
		if l.ID != listing {
			continue
		}
		s := m.player(l.Seller)
		if s == nil || l.Seller == id || len(b.Charms) >= maxCharms || b.Money-m.committed(id, -1) < l.Price || !hasCharm(s, l.Charm) {
			return false
		}
		removeCharm(s, l.Charm)
		b.Charms = append(b.Charms, l.Charm)
		b.Money -= l.Price
		s.Money += l.Price
		m.Market = append(m.Market[:i], m.Market[i+1:]...)
		m.logf("%s bought %s from %s for $%d", b.Name, charms[l.Charm].name, s.Name, l.Price)
		m.changed()
		return true
	}
	return false
}

// ---- Host: hexes ----

// castHex: the hex cast, and whether a Mirror Ward sent it back
func (m *Match) castHex(id, slot, target int) (hex string, bounced, ok bool) {
	p, t := m.player(id), m.player(target)
	if p == nil || t == nil || id == target || t.Out || t.Gone || slot < 0 || slot >= len(p.Hexes) || m.Mode == ModeCoop || m.Phase == "over" {
		return "", false, false
	}
	h := p.Hexes[slot]
	p.Hexes = append(p.Hexes[:slot], p.Hexes[slot+1:]...)
	p.Cast++
	if hasCharm(t, "mirror") {
		removeCharm(t, "mirror")
		m.logf("%s's Mirror Ward sent %s back at %s!", t.Name, hexes[h].name, p.Name)
		m.applyHex(t, p, h)
		m.changed()
		return h, true, true
	}
	if hasCharm(t, "thorn") {
		n := min(3, p.Money)
		p.Money -= n
		t.Money += n
	}
	m.applyHex(p, t, h)
	m.logf("%s hexed %s with %s", p.Name, t.Name, hexes[h].name)
	m.changed()
	return h, false, true
}

func (m *Match) applyHex(from, to *PlayerState, h string) {
	if h == "pickpocket" {
		n := min(3, to.Money)
		to.Money -= n
		from.Money += n
		return
	}
	to.OnMe = append(to.OnMe, h)
}

// ---- Host: race mode, where everyone moves at their own pace ----

func (m *Match) raceFinishBlind(p *PlayerState) {
	target := m.targetFor(p.Ante, p.Blind)
	p.Total += p.Score
	if p.Score >= target {
		n := 3 + p.Blind + max(0, p.Plays)
		if hasCharm(p, "purse") {
			n += 2
		}
		p.Money += n
		p.Last = fmt.Sprintf("Beat the %s  +$%d", blindNames[p.Blind], n)
		if p.Ante == lengthAntes[m.Length]-1 && p.Blind == 2 {
			p.Stage = "won"
			m.end(p.Name + " wins the race!")
			return
		}
		p.Blind++
		if p.Blind > 2 {
			p.Blind = 0
			p.Ante++
		}
		m.logf("%s reached %s, ante %d", p.Name, blindNames[p.Blind], p.Ante+1)
	} else {
		p.Lives--
		p.Money += 2
		p.Last = fmt.Sprintf("Missed the blind: lost a life  +$2")
		m.logf("%s missed a blind and lost a life", p.Name)
		if p.Lives <= 0 {
			p.Out, p.Stage = true, "out"
			if len(m.alive()) == 0 {
				m.end(m.raceLeader().Name + " got furthest and wins!")
			}
			return
		}
	}
	m.openRaceShop(p)
}

func (m *Match) raceLeader() *PlayerState {
	s := append([]*PlayerState{}, m.Players...)
	sort.SliceStable(s, func(a, b int) bool {
		if s[a].Ante*3+s[a].Blind != s[b].Ante*3+s[b].Blind {
			return s[a].Ante*3+s[a].Blind > s[b].Ante*3+s[b].Blind
		}
		return s[a].Total > s[b].Total
	})
	return s[0]
}

func (m *Match) openRaceShop(p *PlayerState) {
	p.Stage, p.Tomes = "shop", nil
	p.Shop = nil
	skip := map[string]bool{}
	for _, c := range p.Charms {
		skip[c] = true
	}
	for i := 0; i < 3; i++ {
		id := randomCharm(m.rng, skip)
		skip[id] = true
		p.Shop = append(p.Shop, ShopItem{Kind: "charm", ID: id, Price: charms[id].price, Bidder: -1})
	}
	id := hexOrder[m.rng.Intn(len(hexOrder))]
	p.Shop = append(p.Shop, ShopItem{Kind: "hex", ID: id, Price: hexes[id].price, Bidder: -1})
	if p.Bot {
		m.botWait[p.ID] = 120 + m.rng.Intn(120)
	}
	m.changed()
}

func (m *Match) raceBuy(id, item int) bool {
	p := m.player(id)
	if p == nil || p.Stage != "shop" || item < 0 || item >= len(p.Shop) {
		return false
	}
	it := &p.Shop[item]
	if it.Sold || p.Money < it.Price || (it.Kind == "charm" && len(p.Charms) >= maxCharms) {
		return false
	}
	p.Money -= it.Price
	it.Sold = true
	if it.Kind == "charm" {
		p.Charms = append(p.Charms, it.ID)
	} else {
		p.Hexes = append(p.Hexes, it.ID)
	}
	m.changed()
	return true
}

func (m *Match) raceContinue(id int) {
	p := m.player(id)
	if p == nil || p.Stage != "shop" || m.Phase == "over" {
		return
	}
	p.Stage = "play"
	m.beginPlayerRound(p)
	m.changed()
}

// ---- Host: each frame ----

func (m *Match) tick() {
	if m.timer > 0 {
		m.timer--
		if s := (m.timer + 59) / 60; s != m.Secs {
			m.Secs = s
			m.changed()
		}
	}
	m.tickBots()
	switch m.Phase {
	case "draft":
		all := true
		for _, p := range m.Players {
			all = all && (p.Ready || p.Gone)
		}
		if all {
			if m.Mode == ModeRace {
				m.Phase = "play"
				for _, p := range m.alive() {
					p.Stage = "play"
					m.beginPlayerRound(p)
				}
				m.changed()
			} else {
				m.startBlind()
			}
		}
	case "play":
		if m.Mode == ModeRace {
			return
		}
		all := true
		for _, p := range m.alive() {
			all = all && p.Done
		}
		if all {
			m.resolveBlind()
		}
	case "results":
		if m.timer == 0 {
			m.nextBlind()
			m.openShop()
		}
	case "shop":
		all := true
		for _, p := range m.alive() {
			all = all && (p.Ready || p.Gone)
		}
		if m.timer == 0 || all {
			m.closeAuction()
			m.startBlind()
		}
	}
}

// Bots: play a move now and then, bid in the shop, cast hexes at the leader
func (m *Match) tickBots() {
	for _, p := range m.Players {
		if !p.Bot || p.Out {
			continue
		}
		if m.botWait[p.ID] > 0 {
			m.botWait[p.ID]--
			continue
		}
		r := m.bots[p.ID]
		playing := (m.Mode == ModeRace && p.Stage == "play" && m.Phase == "play") || (m.Mode != ModeRace && m.Phase == "play")
		shopping := (m.Mode == ModeRace && p.Stage == "shop") || (m.Mode != ModeRace && m.Phase == "shop" && !p.Ready)
		switch {
		case playing && r != nil && !p.Done:
			r.botTurn()
			m.report(p.ID, p.Round, r.Score, r.Plays, r.Done)
			m.botWait[p.ID] = 45 + m.rng.Intn(50)
		case shopping:
			m.botShop(p)
			m.botWait[p.ID] = 40 + m.rng.Intn(60)
		default:
			m.botWait[p.ID] = 30
		}
		if len(p.Hexes) > 0 && m.Mode != ModeCoop && m.Phase != "over" {
			if t := m.hexTarget(p); t != nil {
				m.castHex(p.ID, 0, t.ID)
			}
		}
	}
}

func (m *Match) hexTarget(p *PlayerState) *PlayerState {
	var best *PlayerState
	for _, q := range m.alive() {
		if q.ID == p.ID {
			continue
		}
		if best == nil || q.Lives > best.Lives || (q.Lives == best.Lives && q.Total > best.Total) {
			best = q
		}
	}
	return best
}

func (m *Match) botShop(p *PlayerState) {
	if m.Mode == ModeRace {
		for i, it := range p.Shop {
			if !it.Sold && p.Money >= it.Price+2 && m.rng.Intn(2) == 0 {
				m.raceBuy(p.ID, i)
				return
			}
		}
		for _, h := range p.Picks {
			if m.rng.Intn(3) == 0 && m.buyTome(p.ID, h) {
				return
			}
		}
		m.raceContinue(p.ID)
		return
	}
	// raise a bid on something it likes, if it can afford it
	for i, it := range m.Auction {
		if it.Sold || it.Bidder == p.ID {
			continue
		}
		worth := it.Price + 2 + m.rng.Intn(3)
		if it.Kind == "charm" {
			worth += charms[it.ID].rarity * 2
		}
		if max(it.Price, it.Bid+1) <= worth && m.rng.Intn(3) > 0 && m.bid(p.ID, i, 0) {
			return
		}
	}
	for _, h := range p.Picks {
		if p.Money-m.committed(p.ID, -1) >= tomePrice(p, h)+3 && m.rng.Intn(2) == 0 && m.buyTome(p.ID, h) {
			return
		}
	}
	p.Ready = true
	m.changed()
}

// ---- Helpers ----

func commas(n int64) string {
	s := fmt.Sprint(n)
	neg := n < 0
	if neg {
		s = s[1:]
	}
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	if neg {
		out = "-" + out
	}
	return out
}
