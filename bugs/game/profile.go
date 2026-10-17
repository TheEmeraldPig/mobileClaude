package game

import (
	"errors"
	"math"
	"math/rand"
)

// The saved state and the numbers behind the game: bug levels and stars, coins,
// gems, the stage you are on, eggs and the shop.

type owned struct {
	Have   bool `json:"have"`
	Level  int  `json:"lv"`
	Stars  int  `json:"stars"` // extra copies hatched, up to 5
	Shards int  `json:"shards"`
}

type Profile struct {
	Coins    float64  `json:"coins"`
	Gems     int      `json:"gems"`
	Stage    int      `json:"stage"` // 1-based
	Wave     int      `json:"wave"`  // slugs beaten on this stage, 0..wavesPerStage
	Best     int      `json:"best"`
	Bugs     []owned  `json:"bugs"`
	Team     []int    `json:"team"`
	TapLv    int      `json:"tap"`
	Farming  bool     `json:"farm"` // lost to the boss: loop the stage until you try again
	LastSeen int64    `json:"seen"`
	FreeEgg  int64    `json:"free"` // when the next free egg is ready
	Starter  bool     `json:"starter"`
	Bought   []string `json:"bought"`
	Taps     int      `json:"taps"`
	Kills    int      `json:"kills"`
	Hatched  int      `json:"hatched"`
	Sound    bool     `json:"sound"`
}

const (
	wavesPerStage = 10
	bossTime      = 30.0
	maxStars      = 5
	freeEggEvery  = 4 * 3600
	offlineCap    = 8 * 3600
)

func newProfile(now int64) *Profile {
	p := &Profile{Stage: 1, Gems: 150, LastSeen: now, FreeEgg: now}
	p.fix()
	p.Bugs[0] = owned{Have: true, Level: 1}
	p.Team = []int{0}
	return p
}

// fix repairs a loaded save so it matches the current game.
func (p *Profile) fix() {
	for len(p.Bugs) < len(bugs) {
		p.Bugs = append(p.Bugs, owned{})
	}
	p.Bugs = p.Bugs[:len(bugs)]
	if p.Stage < 1 {
		p.Stage = 1
	}
	p.Wave = max(0, min(p.Wave, wavesPerStage))
	var team []int
	seen := map[int]bool{}
	for _, id := range p.Team {
		if id >= 0 && id < len(bugs) && p.Bugs[id].Have && !seen[id] && len(team) < p.slots() {
			team = append(team, id)
			seen[id] = true
		}
	}
	p.Team = team
	if len(p.Team) == 0 {
		for id, b := range p.Bugs {
			if b.Have {
				p.Team = []int{id}
				break
			}
		}
	}
	for i := range p.Bugs {
		if p.Bugs[i].Have && p.Bugs[i].Level < 1 {
			p.Bugs[i].Level = 1
		}
	}
}

// slots is how many bugs can be on the team: more open up as you go.
func (p *Profile) slots() int {
	n := 1
	for _, s := range slotStages {
		if p.Best >= s || p.Stage >= s {
			n++
		}
	}
	return n
}

var slotStages = []int{2, 5, 10, 18}

func nextSlotStage(p *Profile) int {
	for _, s := range slotStages {
		if p.Stage < s && p.Best < s {
			return s
		}
	}
	return 0
}

// ---- damage ----

// bugDPS is a bug's damage per second at a level with some stars.
func bugDPS(id, level, stars int) float64 {
	if level <= 0 {
		return 0
	}
	sp := bugs[id]
	m := math.Pow(2, float64(milestones(level)))
	return sp.dps * float64(level) * m * (1 + 0.5*float64(stars))
}

// milestones counts the levels that double damage: 10, 25, 50, then every 50.
func milestones(level int) int {
	n := 0
	for _, l := range []int{10, 25, 50} {
		if level >= l {
			n++
		}
	}
	if level >= 100 {
		n += level/50 - 1
	}
	return n
}

func nextMilestone(level int) int {
	for _, l := range []int{10, 25, 50} {
		if level < l {
			return l
		}
	}
	return (level/50 + 1) * 50
}

func (p *Profile) dps(id int) float64 { return bugDPS(id, p.Bugs[id].Level, p.Bugs[id].Stars) }

// teamDPS adds up the damage of everyone on the team.
func (p *Profile) teamDPS() float64 {
	t := 0.0
	for _, id := range p.Team {
		t += p.dps(id)
	}
	return t
}

// tapDamage is what one tap does: its own level plus a share of the team's damage.
func (p *Profile) tapDamage() float64 {
	l := float64(p.TapLv + 1)
	return l*math.Pow(2, float64(milestones(p.TapLv+1))) + p.teamDPS()*0.08
}

// ---- costs ----

func levelCost(id, level int) float64 {
	r := []float64{5, 40, 260, 1400}[bugs[id].rarity]
	return math.Floor(r * math.Pow(1.075, float64(level-1)) * (1 + float64(level)*0.02))
}

func tapCost(lv int) float64 { return math.Floor(6 * math.Pow(1.08, float64(lv))) }

// upCost is the price of n levels for bug id (or the tap power if id < 0).
func (p *Profile) upCost(id, n int) float64 {
	t := 0.0
	for i := 0; i < n; i++ {
		if id < 0 {
			t += tapCost(p.TapLv + i)
		} else {
			t += levelCost(id, p.Bugs[id].Level+i)
		}
	}
	return t
}

// maxLevels is how many levels you can afford now (at least 1 to show a price).
func (p *Profile) maxLevels(id int) int {
	n, t := 0, 0.0
	for n < 1000 {
		var c float64
		if id < 0 {
			c = tapCost(p.TapLv + n)
		} else {
			c = levelCost(id, p.Bugs[id].Level+n)
		}
		if t+c > p.Coins {
			break
		}
		t += c
		n++
	}
	return max(1, n)
}

var errCoins = errors.New("Not enough coins")
var errGems = errors.New("Not enough gems")

func (p *Profile) levelUp(id, n int) error {
	c := p.upCost(id, n)
	if c > p.Coins {
		return errCoins
	}
	p.Coins -= c
	if id < 0 {
		p.TapLv += n
	} else {
		p.Bugs[id].Level += n
	}
	return nil
}

// ---- slugs and stages ----

// slugHP is how tough a regular slug is on a stage.
func slugHP(stage int) float64 {
	s := float64(stage - 1)
	return math.Floor(10 * math.Pow(1.38, s) * (1 + s*0.1))
}

func bossHP(stage int) float64 { return slugHP(stage) * 9 }

// slugCoins is what a regular slug drops.
func slugCoins(stage int) float64 {
	s := float64(stage - 1)
	return math.Max(1, math.Floor(2*math.Pow(1.33, s)*(1+s*0.08)))
}

func bossCoins(stage int) float64 { return slugCoins(stage) * 12 }

func bossGems(stage int) int { return 3 + stage/5 }

// area is which place a stage is in (5 stages each).
func area(stage int) int { return (stage - 1) / 5 }

// beatSlug counts a slug on the stage and pays out.
func (p *Profile) beatSlug(coins float64) {
	p.Coins += coins
	p.Kills++
	if !p.Farming && p.Wave < wavesPerStage {
		p.Wave++
	}
}

// beatBoss moves on to the next stage. It returns true if this stage gives an egg.
func (p *Profile) beatBoss() (egg bool) {
	p.Coins += bossCoins(p.Stage)
	p.Gems += bossGems(p.Stage)
	p.Kills++
	egg = p.Stage%5 == 0 && p.Stage > p.Best
	p.Stage++
	p.Best = max(p.Best, p.Stage-1)
	p.Wave = 0
	p.Farming = false
	return egg
}

func (p *Profile) loseBoss() {
	p.Farming = true
	p.Wave = wavesPerStage
}

// ---- eggs ----

type eggKind struct {
	name    string
	gems    int
	weights [4]float64 // by rarity
}

var eggs = []eggKind{
	{"Bug Egg", 120, [4]float64{62, 30, 7.5, 0.5}},
	{"Golden Egg", 400, [4]float64{0, 62, 33, 5}},
}

// rollBug picks a bug from an egg.
func rollBug(rng *rand.Rand, w [4]float64) int {
	tot := 0.0
	for _, v := range w {
		tot += v
	}
	x := rng.Float64() * tot
	r := 0
	for r = 0; r < 3; r++ {
		if x < w[r] {
			break
		}
		x -= w[r]
	}
	var pool []int
	for id, sp := range bugs {
		if sp.rarity == r {
			pool = append(pool, id)
		}
	}
	return pool[rng.Intn(len(pool))]
}

// hatch adds a bug. It returns whether it is new and its star count after.
func (p *Profile) hatch(id int) (isNew bool) {
	b := &p.Bugs[id]
	p.Hatched++
	if !b.Have {
		*b = owned{Have: true, Level: 1}
		if len(p.Team) < p.slots() {
			p.Team = append(p.Team, id)
		}
		return true
	}
	if b.Stars < maxStars {
		b.Stars++
	} else {
		p.Coins += levelCost(id, b.Level) * 10
	}
	return false
}

// toggleTeam puts a bug on or off the team.
func (p *Profile) toggleTeam(id int) error {
	if !p.Bugs[id].Have {
		return errors.New("Hatch this bug from an egg first")
	}
	for i, t := range p.Team {
		if t == id {
			if len(p.Team) == 1 {
				return errors.New("Your brigade needs at least one bug")
			}
			p.Team = append(p.Team[:i:i], p.Team[i+1:]...)
			return nil
		}
	}
	if len(p.Team) >= p.slots() {
		if s := nextSlotStage(p); s > 0 {
			return errors.New("Team is full. A new slot opens at stage " + itoa(s))
		}
		return errors.New("Team is full")
	}
	p.Team = append(p.Team, id)
	return nil
}

func (p *Profile) inTeam(id int) bool {
	for _, t := range p.Team {
		if t == id {
			return true
		}
	}
	return false
}

// ---- offline and the shop ----

// coinRate is roughly how many coins a second the team earns on the current stage.
func (p *Profile) coinRate() float64 {
	hp := slugHP(p.Stage)
	perKill := hp/math.Max(p.teamDPS(), 0.01) + 1.2 // walking in takes a moment
	return slugCoins(p.Stage) / perKill
}

// offline works out the coins earned while away (half rate, capped at 8 hours).
func (p *Profile) offline(now int64) (coins float64, secs int64) {
	secs = now - p.LastSeen
	if secs < 60 {
		return 0, 0
	}
	if secs > offlineCap {
		secs = offlineCap
	}
	return math.Floor(p.coinRate() * float64(secs) * 0.5), secs
}

type shopItem struct {
	id, name, desc string
	gems           int    // price in gems, or
	price          string // real money (placeholder until Google Play Billing)
	give           int    // gems given
	icon           int
}

const (
	iconEgg = iota
	iconGoldEgg
	iconCoins
	iconGems
	iconChest
	iconStarter
)

var shopItems = []shopItem{
	{id: "egg", name: "Bug Egg", desc: "A random bug. Copies add stars.", gems: 120, icon: iconEgg},
	{id: "gold", name: "Golden Egg", desc: "Rare or better, 5% Legendary", gems: 400, icon: iconGoldEgg},
	{id: "coins", name: "Coin Sack", desc: "1 hour of coins, right now", gems: 40, icon: iconCoins},
	{id: "starter", name: "Starter Pack", desc: "Golden Egg + 500 gems. Once only!", price: "$2.99", give: 500, icon: iconStarter},
	{id: "g1", name: "Handful of Gems", desc: "150 gems", price: "$0.99", give: 150, icon: iconGems},
	{id: "g2", name: "Pouch of Gems", desc: "900 gems (+20%)", price: "$4.99", give: 900, icon: iconGems},
	{id: "g3", name: "Chest of Gems", desc: "2,000 gems (+33%)", price: "$9.99", give: 2000, icon: iconChest},
}

// buy pays for a shop item. It returns an egg to hatch (or -1 for none).
func (p *Profile) buy(it shopItem, rng *rand.Rand) (egg int, err error) {
	if it.gems > p.Gems {
		return -1, errGems
	}
	if it.id == "starter" && p.Starter {
		return -1, errors.New("Already bought")
	}
	p.Gems -= it.gems
	if it.price != "" {
		p.Bought = append(p.Bought, it.id)
		p.Gems += it.give
	}
	switch it.id {
	case "egg":
		return rollBug(rng, eggs[0].weights), nil
	case "gold":
		return rollBug(rng, eggs[1].weights), nil
	case "starter":
		p.Starter = true
		return rollBug(rng, eggs[1].weights), nil
	case "coins":
		p.Coins += math.Max(100, p.coinRate()*3600)
	}
	return -1, nil
}

// freeEgg hatches the free egg if it is ready.
func (p *Profile) freeEgg(now int64, rng *rand.Rand) (int, error) {
	if now < p.FreeEgg {
		return -1, errors.New("Not ready yet")
	}
	p.FreeEgg = now + freeEggEvery
	return rollBug(rng, eggs[0].weights), nil
}
