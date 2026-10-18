package game

import (
	"errors"
	"math"
	"math/rand"
	"sort"
)

// The saved state and the numbers behind the game: chef levels and stars, coins,
// gems, the stage, ingredients, the kitchen and the dishes on the table.

type owned struct {
	Have  bool `json:"have"`
	Level int  `json:"lv"`
	Stars int  `json:"stars"` // extra copies recruited, up to 5
}

type dish struct {
	R int     `json:"r"`
	Q float64 `json:"q"` // strength: cooking quality times station bonus
}

type served struct {
	dish
	Left float64 `json:"left"` // seconds of play left
}

type cooking struct {
	R     int     `json:"r"` // -1 = idle
	Q     float64 `json:"q"`
	Left  float64 `json:"left"`
	Total float64 `json:"total"`
}

type Profile struct {
	Coins    float64              `json:"coins"`
	Gems     int                  `json:"gems"`
	Stage    int                  `json:"stage"` // 1-based
	Wave     int                  `json:"wave"`  // monsters beaten on this stage
	Best     int                  `json:"best"`
	Chefs    []owned              `json:"chefs"`
	Team     []int                `json:"team"`
	TapLv    int                  `json:"tap"`
	Farming  bool                 `json:"farm"`
	LastSeen int64                `json:"seen"`
	FreeBox  int64                `json:"free"`
	Starter  bool                 `json:"starter"`
	Bought   []string             `json:"bought"`
	Taps     int                  `json:"taps"`
	Kills    int                  `json:"kills"`
	Hired    int                  `json:"hired"`
	Ing      [numIngredients]int  `json:"ing"`
	Stations [numStations]int     `json:"stations"` // level, 0 = not built
	Cook     [numStations]cooking `json:"cook"`
	Pantry   []dish               `json:"pantry"`
	Table    []served             `json:"table"`
	Cooked   int                  `json:"cooked"`
}

const (
	wavesPerStage = 10
	bossTime      = 30.0
	maxStars      = 5
	freeBoxEvery  = 4 * 3600
	offlineCap    = 8 * 3600
	plates        = 3
	pantrySize    = 8
	dishTime      = 240.0 // seconds a served dish lasts
)

func newProfile(now int64) *Profile {
	p := &Profile{Stage: 1, Gems: 150, LastSeen: now, FreeBox: now}
	p.fix()
	p.Chefs[0] = owned{Have: true, Level: 1}
	p.Team = []int{0}
	p.Ing[inFlour] = 4
	return p
}

// fix repairs a loaded save so it matches the current game.
func (p *Profile) fix() {
	for len(p.Chefs) < len(chefs) {
		p.Chefs = append(p.Chefs, owned{})
	}
	p.Chefs = p.Chefs[:len(chefs)]
	if p.Stage < 1 {
		p.Stage = 1
	}
	p.Wave = max(0, min(p.Wave, wavesPerStage))
	var team []int
	seen := map[int]bool{}
	for _, id := range p.Team {
		if id >= 0 && id < len(chefs) && p.Chefs[id].Have && !seen[id] && len(team) < p.slots() {
			team = append(team, id)
			seen[id] = true
		}
	}
	p.Team = team
	if len(p.Team) == 0 {
		for id, b := range p.Chefs {
			if b.Have {
				p.Team = []int{id}
				break
			}
		}
	}
	for i := range p.Chefs {
		if p.Chefs[i].Have && p.Chefs[i].Level < 1 {
			p.Chefs[i].Level = 1
		}
	}
	if p.Stations[stCounter] < 1 {
		p.Stations[stCounter] = 1
	}
	for i := range p.Cook {
		if p.Cook[i].Total == 0 {
			p.Cook[i].R = -1
		}
		if p.Cook[i].R >= len(recipes) {
			p.Cook[i] = cooking{R: -1}
		}
	}
	var pan []dish
	for _, d := range p.Pantry {
		if d.R >= 0 && d.R < len(recipes) && len(pan) < pantrySize {
			pan = append(pan, d)
		}
	}
	p.Pantry = pan
	var tab []served
	for _, d := range p.Table {
		if d.R >= 0 && d.R < len(recipes) && d.Left > 0 && len(tab) < plates {
			tab = append(tab, d)
		}
	}
	p.Table = tab
}

// slots is how many chefs can be on the crew: more open up as you go.
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

// chefDPS is a chef's base damage per second at a level with some stars.
func chefDPS(id, level, stars int) float64 {
	if level <= 0 {
		return 0
	}
	m := math.Pow(2, float64(milestones(level)))
	return chefs[id].dps * float64(level) * m * (1 + 0.5*float64(stars))
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

// dps is a chef's damage per second with the dishes on the table.
func (p *Profile) dps(id int) float64 {
	e := p.effects()
	d := chefDPS(id, p.Chefs[id].Level, p.Chefs[id].Stars) * (1 + (e[efChef]+e[efAll])/100)
	if p.favServed(id) {
		d *= 2
	}
	return d
}

// teamDPS adds up the damage of everyone on the crew.
func (p *Profile) teamDPS() float64 {
	t := 0.0
	for _, id := range p.Team {
		t += p.dps(id)
	}
	return t
}

// tapDamage is what one tap does: its own level plus a share of the crew's damage.
func (p *Profile) tapDamage() float64 {
	l := float64(p.TapLv + 1)
	e := p.effects()
	return (l*math.Pow(2, float64(milestones(p.TapLv+1))) + p.teamDPS()*0.08) * (1 + (e[efTap]+e[efAll])/100)
}

// ---- costs ----

func levelCost(id, level int) float64 {
	r := []float64{5, 40, 260, 1400}[chefs[id].rarity]
	return math.Floor(r * math.Pow(1.075, float64(level-1)) * (1 + float64(level)*0.02))
}

func tapCost(lv int) float64 { return math.Floor(6 * math.Pow(1.08, float64(lv))) }

// upCost is the price of n levels for chef id (or the tap power if id < 0).
func (p *Profile) upCost(id, n int) float64 {
	t := 0.0
	for i := 0; i < n; i++ {
		if id < 0 {
			t += tapCost(p.TapLv + i)
		} else {
			t += levelCost(id, p.Chefs[id].Level+i)
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
			c = levelCost(id, p.Chefs[id].Level+n)
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
		p.Chefs[id].Level += n
	}
	return nil
}

// ---- monsters and stages ----

func monHP(stage int) float64 {
	s := float64(stage - 1)
	return math.Floor(10 * math.Pow(1.38, s) * (1 + s*0.1))
}

func bossHP(stage int) float64 { return monHP(stage) * 9 }

func monCoins(stage int) float64 {
	s := float64(stage - 1)
	return math.Max(1, math.Floor(2*math.Pow(1.33, s)*(1+s*0.08)))
}

func bossCoins(stage int) float64 { return monCoins(stage) * 12 }

func bossGems(stage int) int { return 3 + stage/5 }

// area is which place a stage is in (5 stages each).
func area(stage int) int { return (stage - 1) / 5 }

// coinMult is the coin bonus from the table.
func (p *Profile) coinMult() float64 { return 1 + p.effects()[efCoins]/100 }

// dropCount is how many ingredients a monster drops on this stage.
func (p *Profile) dropCount(boss bool) int {
	n := 1 + p.Stage/8
	if boss {
		n = 4 + p.Stage/4
	}
	return int(math.Round(float64(n) * (1 + p.effects()[efDrops]/100)))
}

func (p *Profile) beatMonster(coins float64) {
	p.Coins += coins
	p.Kills++
	if !p.Farming && p.Wave < wavesPerStage {
		p.Wave++
	}
}

// beatBoss moves on to the next stage. It returns true if this stage gives a box.
func (p *Profile) beatBoss() (box bool) {
	p.Coins += bossCoins(p.Stage) * p.coinMult()
	p.Gems += bossGems(p.Stage)
	p.Kills++
	box = p.Stage%5 == 0 && p.Stage > p.Best
	p.Stage++
	p.Best = max(p.Best, p.Stage-1)
	p.Wave = 0
	p.Farming = false
	return box
}

func (p *Profile) loseBoss() {
	p.Farming = true
	p.Wave = wavesPerStage
}

// ---- recruiting ----

type boxKind struct {
	name    string
	gems    int
	weights [4]float64 // by rarity
}

var boxes = []boxKind{
	{"Recruit Box", 120, [4]float64{62, 30, 7.5, 0.5}},
	{"Star Recruit", 400, [4]float64{0, 62, 33, 5}},
}

func rollChef(rng *rand.Rand, w [4]float64) int {
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
	for id, c := range chefs {
		if c.rarity == r {
			pool = append(pool, id)
		}
	}
	return pool[rng.Intn(len(pool))]
}

// hire adds a chef. It returns whether it is new.
func (p *Profile) hire(id int) (isNew bool) {
	b := &p.Chefs[id]
	p.Hired++
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

func (p *Profile) toggleTeam(id int) error {
	if !p.Chefs[id].Have {
		return errors.New("Recruit this chef from a box first")
	}
	for i, t := range p.Team {
		if t == id {
			if len(p.Team) == 1 {
				return errors.New("Your crew needs at least one chef")
			}
			p.Team = append(p.Team[:i:i], p.Team[i+1:]...)
			return nil
		}
	}
	if len(p.Team) >= p.slots() {
		if s := nextSlotStage(p); s > 0 {
			return errors.New("Crew is full. A new spot opens at stage " + itoa(s))
		}
		return errors.New("Crew is full")
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

// ---- the kitchen ----

// stationPrice is what building (level 0) or upgrading a station costs.
func (p *Profile) stationPrice(st int) float64 {
	lv := p.Stations[st]
	base := math.Max(stationCost[st], 80)
	if lv == 0 {
		return stationCost[st]
	}
	return math.Floor(base * 2.5 * math.Pow(3, float64(lv-1)))
}

func (p *Profile) upgradeStation(st int) error {
	c := p.stationPrice(st)
	if c > p.Coins {
		return errCoins
	}
	p.Coins -= c
	p.Stations[st]++
	return nil
}

// cookTime is how long recipe r takes at its station's level.
func (p *Profile) cookTime(r int) float64 {
	rc := recipes[r]
	return rc.secs * math.Pow(0.85, float64(max(1, p.Stations[rc.station])-1))
}

// stationBonus makes dishes from better stations stronger.
func (p *Profile) stationBonus(st int) float64 { return 1 + 0.2*float64(max(1, p.Stations[st])-1) }

func (p *Profile) canCook(r int) error {
	rc := recipes[r]
	if p.Stations[rc.station] == 0 {
		return errors.New("Build the " + stationName[rc.station] + " first")
	}
	if p.Cook[rc.station].R >= 0 {
		return errors.New("The " + stationName[rc.station] + " is busy")
	}
	for i, n := range rc.need {
		if p.Ing[i] < n {
			return errors.New("Need more " + ingName[i])
		}
	}
	if len(p.Pantry) >= pantrySize {
		return errors.New("The pantry is full: serve some dishes")
	}
	return nil
}

// startCook spends the ingredients and puts recipe r on its station.
func (p *Profile) startCook(r int, quality float64) error {
	if err := p.canCook(r); err != nil {
		return err
	}
	rc := recipes[r]
	for i, n := range rc.need {
		p.Ing[i] -= n
	}
	t := p.cookTime(r)
	p.Cook[rc.station] = cooking{R: r, Q: quality * p.stationBonus(rc.station), Left: t, Total: t}
	return nil
}

// tickKitchen moves cooking along; served dishes only count down while playing.
func (p *Profile) tickKitchen(secs float64, playing bool) {
	for i := range p.Cook {
		if p.Cook[i].R >= 0 && p.Cook[i].Left > 0 {
			p.Cook[i].Left = math.Max(0, p.Cook[i].Left-secs)
		}
	}
	if !playing {
		return
	}
	keep := p.Table[:0]
	for _, d := range p.Table {
		d.Left -= secs
		if d.Left > 0 {
			keep = append(keep, d)
		}
	}
	p.Table = keep
}

// collect takes a finished dish off a station into the pantry.
func (p *Profile) collect(st int) (dish, error) {
	c := p.Cook[st]
	if c.R < 0 || c.Left > 0 {
		return dish{}, errors.New("Not ready yet")
	}
	if len(p.Pantry) >= pantrySize {
		return dish{}, errors.New("The pantry is full: serve some dishes")
	}
	d := dish{R: c.R, Q: c.Q}
	p.Pantry = append(p.Pantry, d)
	p.Cook[st] = cooking{R: -1}
	p.Cooked++
	return d, nil
}

// serve puts pantry dish i on the table.
func (p *Profile) serve(i int) error {
	if i < 0 || i >= len(p.Pantry) {
		return errors.New("No such dish")
	}
	if len(p.Table) >= plates {
		return errors.New("The table is full")
	}
	d := p.Pantry[i]
	p.Pantry = append(p.Pantry[:i:i], p.Pantry[i+1:]...)
	p.Table = append(p.Table, served{dish: d, Left: dishTime})
	return nil
}

func (p *Profile) onTable(r int) bool {
	for _, d := range p.Table {
		if d.R == r {
			return true
		}
	}
	return false
}

// fullCourse is true when a starter, a main and a dessert are on the table.
func (p *Profile) fullCourse() bool {
	var have [3]bool
	for _, d := range p.Table {
		have[recipes[d.R].course] = true
	}
	return have[0] && have[1] && have[2]
}

// activeCombos lists the combos on the table.
func (p *Profile) activeCombos() []int {
	var out []int
	for i, c := range combos {
		ok := true
		for _, r := range c.dishes {
			if !p.onTable(r) {
				ok = false
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func (p *Profile) favServed(id int) bool { return p.onTable(chefs[id].fav) }

// effects adds up every bonus from the dishes on the table.
func (p *Profile) effects() (e [numEffects]float64) {
	k := 1.0
	if p.fullCourse() {
		k = fullCourse
	}
	for _, d := range p.Table {
		rc := recipes[d.R]
		e[rc.eff] += rc.val * d.Q * k
	}
	for _, i := range p.activeCombos() {
		e[combos[i].eff] += combos[i].val
	}
	return e
}

// ---- offline and the shop ----

// killRate is roughly how many monsters a second the crew beats on this stage.
func (p *Profile) killRate() float64 {
	return 1 / (monHP(p.Stage)/math.Max(p.teamDPS(), 0.01) + 1.2)
}

func (p *Profile) coinRate() float64 { return monCoins(p.Stage) * p.killRate() }

type away struct {
	coins float64
	secs  int64
	ing   [numIngredients]int
}

// offline works out what was earned while away (half rate, capped at 8 hours)
// and moves cooking along.
func (p *Profile) offline(now int64) (a away) {
	secs := now - p.LastSeen
	if secs < 60 {
		return
	}
	p.tickKitchen(float64(secs), false)
	if secs > offlineCap {
		secs = offlineCap
	}
	a.secs = secs
	a.coins = math.Floor(p.coinRate() * float64(secs) * 0.5)
	kills := p.killRate() * float64(secs) * 0.5
	fam := areaMonsters[area(p.Stage)%len(areaMonsters)]
	per := kills * float64(1+p.Stage/8) / float64(len(fam))
	for _, m := range fam {
		ds := monsters[m].drop
		for _, ing := range ds {
			a.ing[ing] += int(per / float64(len(ds)))
		}
	}
	return a
}

func (a away) total() int {
	t := 0
	for _, n := range a.ing {
		t += n
	}
	return t
}

type shopItem struct {
	id, name, desc string
	gems           int    // price in gems, or
	price          string // real money (placeholder until Google Play Billing)
	give           int    // gems given
	icon           int
}

const (
	iconBox = iota
	iconGoldBox
	iconCoins
	iconGems
	iconChest
	iconStarter
	iconIngredients
)

var shopItems = []shopItem{
	{id: "box", name: "Recruit Box", desc: "A random chef. Copies add stars.", gems: 120, icon: iconBox},
	{id: "gold", name: "Star Recruit", desc: "Rare or better, 5% Legendary", gems: 400, icon: iconGoldBox},
	{id: "coins", name: "Coin Sack", desc: "1 hour of coins, right now", gems: 40, icon: iconCoins},
	{id: "groceries", name: "Grocery Crate", desc: "5 of every ingredient", gems: 60, icon: iconIngredients},
	{id: "starter", name: "Starter Pack", desc: "Star Recruit + 500 gems. Once only!", price: "$2.99", give: 500, icon: iconStarter},
	{id: "g1", name: "Handful of Gems", desc: "150 gems", price: "$0.99", give: 150, icon: iconGems},
	{id: "g2", name: "Pouch of Gems", desc: "900 gems (+20%)", price: "$4.99", give: 900, icon: iconGems},
	{id: "g3", name: "Chest of Gems", desc: "2,000 gems (+33%)", price: "$9.99", give: 2000, icon: iconChest},
}

// buy pays for a shop item. It returns a chef to recruit (or -1 for none).
func (p *Profile) buy(it shopItem, rng *rand.Rand) (chef int, err error) {
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
	case "box":
		return rollChef(rng, boxes[0].weights), nil
	case "gold":
		return rollChef(rng, boxes[1].weights), nil
	case "starter":
		p.Starter = true
		return rollChef(rng, boxes[1].weights), nil
	case "coins":
		p.Coins += math.Max(100, p.coinRate()*3600)
	case "groceries":
		for i := range p.Ing {
			p.Ing[i] += 5
		}
	}
	return -1, nil
}

func (p *Profile) freeBox(now int64, rng *rand.Rand) (int, error) {
	if now < p.FreeBox {
		return -1, errors.New("Not ready yet")
	}
	p.FreeBox = now + freeBoxEvery
	return rollChef(rng, boxes[0].weights), nil
}

// effectLines describes the table's bonuses, biggest first.
func (p *Profile) effectLines() []string {
	e := p.effects()
	type kv struct {
		k int
		v float64
	}
	var l []kv
	for k, v := range e {
		if v > 0 {
			l = append(l, kv{k, v})
		}
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
	var out []string
	for _, x := range l {
		out = append(out, effectText(x.k, x.v))
	}
	return out
}
