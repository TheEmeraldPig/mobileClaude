package game

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
)

// ---- the map ----

const (
	nFight = iota
	nElite
	nShop
	nRest
	nEvent
	nTreasure
	nBoss
)

var nodeNames = []string{"Fight", "Elite", "Merchant", "Campfire", "Mystery", "Treasure", "Boss"}

const (
	mapRows = 10 // rows of choices; the boss is one more row on top
	mapCols = 5
)

type Node struct {
	Row, Col int
	JX, JY   float64 // small random offset so the map looks hand-drawn
	Kind     int
	Next     []int
	Done     bool
}

func genMap(rng *rand.Rand) []Node {
	var nodes []Node
	at := map[[2]int]int{}
	get := func(r, c int) int {
		if i, ok := at[[2]int{r, c}]; ok {
			return i
		}
		nodes = append(nodes, Node{Row: r, Col: c, JX: rng.Float64()*36 - 18, JY: rng.Float64()*26 - 13})
		at[[2]int{r, c}] = len(nodes) - 1
		return len(nodes) - 1
	}
	type edge struct{ r, a, b int }
	var edges []edge
	crosses := func(r, a, b int) bool {
		for _, e := range edges {
			if e.r == r && ((a < e.a && b > e.b) || (a > e.a && b < e.b)) {
				return true
			}
		}
		return false
	}
	starts := rng.Perm(mapCols)[:3]
	starts = append(starts, rng.Intn(mapCols))
	for _, c := range starts {
		for r := 0; r < mapRows-1; r++ {
			opts := []int{c - 1, c, c + 1}
			rng.Shuffle(3, func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
			nc := -1
			for _, o := range opts {
				if o >= 0 && o < mapCols && !crosses(r, c, o) {
					nc = o
					break
				}
			}
			if nc < 0 {
				nc = c
			}
			from, to := get(r, c), get(r+1, nc)
			if !slices.Contains(nodes[from].Next, to) {
				nodes[from].Next = append(nodes[from].Next, to)
				edges = append(edges, edge{r, c, nc})
			}
			c = nc
		}
	}
	bossI := len(nodes)
	nodes = append(nodes, Node{Row: mapRows, Col: 2, Kind: nBoss})
	for i := range nodes {
		if nodes[i].Row == mapRows-1 {
			nodes[i].Next = []int{bossI}
		}
	}
	// what each spot holds
	parents := make([][]int, len(nodes))
	for i, n := range nodes {
		for _, j := range n.Next {
			parents[j] = append(parents[j], i)
		}
	}
	order := make([]int, 0, len(nodes))
	for r := 0; r <= mapRows; r++ {
		for i, n := range nodes {
			if n.Row == r {
				order = append(order, i)
			}
		}
	}
	shops := 0
	for _, i := range order {
		n := &nodes[i]
		switch {
		case n.Kind == nBoss:
			continue
		case n.Row == 0:
			n.Kind = nFight
		case n.Row == 5:
			n.Kind = nTreasure
		case n.Row == mapRows-1:
			n.Kind = nRest
		default:
			for try := 0; try < 10; try++ {
				k := pickKind(rng, n.Row)
				clash := false
				if k == nElite || k == nShop || k == nRest {
					for _, p := range parents[i] {
						clash = clash || nodes[p].Kind == k
					}
				}
				n.Kind = k
				if !clash {
					break
				}
			}
		}
		if n.Kind == nShop {
			shops++
		}
	}
	if shops == 0 { // always at least one merchant
		for _, i := range order {
			if nodes[i].Row >= 3 && nodes[i].Row <= 7 && nodes[i].Kind == nFight {
				nodes[i].Kind = nShop
				break
			}
		}
	}
	return nodes
}

func pickKind(rng *rand.Rand, row int) int {
	type kw struct{ k, w int }
	ws := []kw{{nFight, 46}, {nEvent, 22}}
	if row >= 2 {
		ws = append(ws, kw{nShop, 12})
	}
	if row >= 3 {
		ws = append(ws, kw{nElite, 13}, kw{nRest, 8})
	}
	t := 0
	for _, w := range ws {
		t += w.w
	}
	n := rng.Intn(t)
	for _, w := range ws {
		if n < w.w {
			return w.k
		}
		n -= w.w
	}
	return nFight
}

// ---- a run ----

type shopOffer struct {
	Card  Card
	Charm string
	Price int
	Sold  bool
}

// Run is one trip up a chapter's map. All of it is saved, except a fight in progress.
type Run struct {
	Seed    int64
	Chapter int
	Nodes   []Node
	Cur     int // node being visited or last visited; -1 before the first
	HP, Max int
	Deck    []Card
	Charms  []string
	Coins   int // spent in the merchant's shop during the run
	Gold    int // paid out to your profile when the run ends
	Gems    int
	Fights  int
	Revived bool
	Won     bool
	Lost    bool

	// what the current spot holds
	Shop        []shopOffer
	ShopRemoved bool
	ShopForged  bool
	ShopHealed  bool
	ForgeEnh    int
	Event       int
	Choices     []Card
	RewardCharm string
	RewardCoins int
	RewardGold  int

	rng *rand.Rand
	B   *Battle
}

func newRun(seed int64, p *Profile) *Run {
	r := &Run{Seed: seed, Chapter: p.Chapter, Cur: -1, HP: p.maxHP(), Max: p.maxHP(), Deck: p.Loadout()}
	r.Nodes = genMap(rand.New(rand.NewSource(seed)))
	return r
}

// restore rebuilds what isn't saved after loading.
func (r *Run) restore() {
	r.rng = rand.New(rand.NewSource(r.Seed*31 + int64(r.Cur)))
}

func (r *Run) has(charm string) bool { return slices.Contains(r.Charms, charm) }

func (r *Run) floor() int {
	if r.Cur < 0 {
		return 0
	}
	return r.Nodes[r.Cur].Row + 1
}

// Reachable lists the nodes you can travel to next.
func (r *Run) Reachable() []int {
	if r.Cur < 0 {
		var out []int
		for i, n := range r.Nodes {
			if n.Row == 0 {
				out = append(out, i)
			}
		}
		return out
	}
	if !r.Nodes[r.Cur].Done {
		return nil
	}
	return r.Nodes[r.Cur].Next
}

// Enter travels to node i and sets up what is there.
func (r *Run) Enter(i int) {
	r.Cur = i
	r.restore()
	r.Shop, r.Choices, r.RewardCharm, r.RewardCoins, r.RewardGold = nil, nil, "", 0, 0
	r.ShopRemoved, r.ShopForged, r.ShopHealed = false, false, false
	switch r.Nodes[i].Kind {
	case nFight:
		r.B = newBattle(r, normals[r.rng.Intn(len(normals))], 1.0)
	case nElite:
		r.B = newBattle(r, elites[r.rng.Intn(len(elites))], 1.0)
	case nBoss:
		r.B = newBattle(r, boss, 1.0)
	case nShop:
		r.makeShop()
	case nRest:
		r.ForgeEnh = r.randomEnh()
	case nEvent:
		r.Event = r.rng.Intn(len(events))
	case nTreasure:
		r.RewardCharm = r.randomCharm()
		r.RewardCoins = 25 + r.rng.Intn(20)
		r.takeCharm(r.RewardCharm)
		r.Coins += r.RewardCoins
	}
}

// Resume re-enters a spot that was not finished when the game closed.
func (r *Run) Resume() {
	if r.Cur < 0 || r.Nodes[r.Cur].Done {
		return
	}
	switch r.Nodes[r.Cur].Kind {
	case nFight, nElite, nBoss:
		r.Enter(r.Cur)
	}
}

// Leave finishes the current spot and returns to the map.
func (r *Run) Leave() {
	if r.Cur >= 0 {
		r.Nodes[r.Cur].Done = true
	}
	r.B = nil
	r.Shop, r.Choices, r.RewardCharm = nil, nil, ""
}

func (r *Run) randomEnh() int {
	if r.rng.Intn(8) == 0 {
		return eEcho
	}
	return 1 + r.rng.Intn(eGold)
}

func (r *Run) randomCharm() string {
	var free []string
	for _, c := range charmDefs {
		if !r.has(c.ID) {
			free = append(free, c.ID)
		}
	}
	if len(free) == 0 {
		return ""
	}
	return free[r.rng.Intn(len(free))]
}

func (r *Run) takeCharm(id string) {
	if id == "" || r.has(id) {
		return
	}
	r.Charms = append(r.Charms, id)
	if id == "heart" {
		r.Max += 12
		r.HP += 12
	}
}

func scaleCh(v, ch int, f float64) int { return int(float64(v)*(1+f*float64(ch-1)) + 0.5) }

// winFight pays out a won fight and sets up the card choice.
func (r *Run) winFight() {
	k := r.Nodes[r.Cur].Kind
	r.Fights++
	coins := 10 + r.rng.Intn(9)
	gold := 15
	odds := [rarityCount]int{55, 30, 12, 3, 0}
	switch k {
	case nElite:
		coins, gold = 28+r.rng.Intn(13), 40
		odds = [rarityCount]int{12, 40, 32, 13, 3}
		r.RewardCharm = r.randomCharm()
		r.takeCharm(r.RewardCharm)
	case nBoss:
		r.Gold += scaleCh(150, r.Chapter, 0.5)
		r.Gems += 5
		r.Won = true
		return
	}
	if r.has("coin") {
		coins = coins * 3 / 2
	}
	r.RewardCoins, r.RewardGold = coins, scaleCh(gold, r.Chapter, 0.5)
	r.Coins += coins
	r.Gold += r.RewardGold
	r.Choices = nil
	for len(r.Choices) < 3 {
		c := randomCardOf(r.rng, rollRarity(r.rng, odds))
		if !slices.ContainsFunc(r.Choices, func(o Card) bool { return o.ID == c.ID }) {
			r.Choices = append(r.Choices, c)
		}
	}
}

// PickCard takes reward card i (or none when i < 0).
func (r *Run) PickCard(i int) {
	if i >= 0 && i < len(r.Choices) {
		r.Deck = append(r.Deck, r.Choices[i])
	}
	r.Choices = nil
}

// Revive refills health and restarts the current fight.
func (r *Run) Revive() {
	r.HP, r.Lost, r.Revived = r.Max, false, true
	r.Enter(r.Cur)
}

// ---- merchant ----

var rarityPrice = []int{45, 70, 110, 170, 260}

const (
	removePrice = 60
	forgePrice  = 90
	healPrice   = 45
)

func (r *Run) makeShop() {
	slots := [][rarityCount]int{{60, 40, 0, 0, 0}, {60, 40, 0, 0, 0}, {0, 55, 40, 5, 0}, {0, 55, 40, 5, 0}, {0, 0, 60, 32, 8}}
	for _, w := range slots {
		c := randomCardOf(r.rng, rollRarity(r.rng, w))
		price := rarityPrice[c.def().Rarity] * (90 + r.rng.Intn(21)) / 100
		if r.rng.Intn(5) == 0 {
			c.Enh = 1 + r.rng.Intn(eGold)
			price = price * 7 / 5
		}
		r.Shop = append(r.Shop, shopOffer{Card: c, Price: price})
	}
	for i := 0; i < 2; i++ {
		ch := r.randomCharm()
		if ch == "" || slices.ContainsFunc(r.Shop, func(o shopOffer) bool { return o.Charm == ch }) {
			continue
		}
		r.Shop = append(r.Shop, shopOffer{Charm: ch, Price: 130 + r.rng.Intn(41)})
	}
	r.ForgeEnh = r.randomEnh()
}

func (r *Run) BuyOffer(i int) error {
	o := &r.Shop[i]
	if o.Sold {
		return fmt.Errorf("Sold out")
	}
	if r.Coins < o.Price {
		return fmt.Errorf("Need %d more coins", o.Price-r.Coins)
	}
	r.Coins -= o.Price
	o.Sold = true
	if o.Charm != "" {
		r.takeCharm(o.Charm)
	} else {
		r.Deck = append(r.Deck, o.Card)
	}
	return nil
}

func (r *Run) ShopHeal() error {
	if r.ShopHealed {
		return fmt.Errorf("Already used")
	}
	if r.Coins < healPrice {
		return fmt.Errorf("Need %d more coins", healPrice-r.Coins)
	}
	r.Coins -= healPrice
	r.ShopHealed = true
	r.HP = min(r.Max, r.HP+r.Max*3/10)
	return nil
}

// RemoveCard and ForgeCard are the merchant's and campfire's services.
func (r *Run) RemoveCard(i int) {
	r.Deck = append(r.Deck[:i], r.Deck[i+1:]...)
}

func (r *Run) ForgeCard(i int) {
	r.Deck[i].Enh = r.ForgeEnh
}

func (r *Run) restHeal() int { return max(10, r.Max*3/10) }

// ---- mystery events ----

type eventDef struct {
	Title, Text string
	Art         int
	Choices     []string
}

var events = []eventDef{
	{"Wishing Well", "Coins glitter at the bottom of an old well.", aHourglass, []string{"Toss 25 coins: maybe a Rare card", "Walk away"}},
	{"Blood Altar", "A dark altar hums. It wants something of you.", aFang, []string{"Lose 10 HP: gain an Epic card", "Walk away"}},
	{"Traveling Trader", "\"I'll swap one of yours for something better.\"", aBook, []string{"Trade a random card up a rarity", "No thanks"}},
	{"Fairy Spring", "The water glows softly.", aAegis, []string{"Drink: heal 15 HP", "Fill your purse: +35 coins"}},
	{"Gambler", "A grinning goblin shuffles cups. \"Double or nothing?\"", aDagger, []string{"Bet half your coins", "Walk away"}},
	{"Old Forge", "An abandoned anvil, still warm.", aHammer, []string{"Foil 2 random cards", "Walk away"}},
}

// Choose resolves an event choice and says what happened.
func (r *Run) Choose(c int) (string, error) {
	rng := r.rng
	switch r.Event*10 + c {
	case 0:
		if r.Coins < 25 {
			return "", fmt.Errorf("You need 25 coins")
		}
		r.Coins -= 25
		if rng.Intn(100) < 55 {
			card := randomCardOf(rng, rRare)
			r.Deck = append(r.Deck, card)
			return "A " + card.def().Name + " floats up!", nil
		}
		return "Splash. Nothing happens.", nil
	case 10:
		if r.HP <= 10 {
			return "", fmt.Errorf("That would kill you")
		}
		r.HP -= 10
		card := randomCardOf(rng, rEpic)
		r.Deck = append(r.Deck, card)
		return "The altar gives you " + card.def().Name + ".", nil
	case 20:
		i := rng.Intn(len(r.Deck))
		old := r.Deck[i]
		rar := min(rLegendary, old.def().Rarity+1)
		card := randomCardOf(rng, rar)
		card.Enh = old.Enh
		r.Deck[i] = card
		return old.def().Name + " became " + card.def().Name + "!", nil
	case 30:
		r.HP = min(r.Max, r.HP+15)
		return "You feel refreshed.", nil
	case 31:
		r.Coins += 35
		return "+35 coins.", nil
	case 40:
		bet := r.Coins / 2
		if bet == 0 {
			return "", fmt.Errorf("You have nothing to bet")
		}
		if rng.Intn(2) == 0 {
			r.Coins += bet
			return fmt.Sprintf("You win %d coins!", bet), nil
		}
		r.Coins -= bet
		return fmt.Sprintf("You lose %d coins.", bet), nil
	case 50:
		idx := rng.Perm(len(r.Deck))
		var names []string
		for _, i := range idx[:min(2, len(idx))] {
			r.Deck[i].Enh = eFoil
			names = append(names, r.Deck[i].def().Name)
		}
		return "Foiled: " + strings.Join(names, " and "), nil
	}
	return "You move on.", nil
}
