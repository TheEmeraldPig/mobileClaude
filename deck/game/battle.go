package game

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
)

// Run is one trip through a chapter: 10 fights, a pick of 3 rewards after each.
type Run struct {
	rng     *rand.Rand
	Chapter int
	Floor   int
	HP, Max int
	Deck    []Card // this run's copy; rewards change it, not the saved deck
	Charms  []string
	Gold    int
	Gems    int
	Revived bool
	B       *Battle
	Rewards []reward
	Won     bool
	Lost    bool
}

func newRun(seed int64, p *Profile) *Run {
	r := &Run{rng: rand.New(rand.NewSource(seed)), Chapter: p.Chapter, Floor: 1, HP: p.maxHP(), Max: p.maxHP()}
	r.Deck = slices.Clone(p.Deck)
	r.startFight()
	return r
}

func (r *Run) has(charm string) bool { return slices.Contains(r.Charms, charm) }

func (r *Run) startFight() {
	var d enemyDef
	switch {
	case r.Floor == floorsPerChapter:
		d = boss
	case r.Floor == 5:
		d = elite
	default:
		d = normals[r.rng.Intn(len(normals))]
	}
	r.B = newBattle(r, d)
}

// ---- rewards ----

const (
	rCard = iota
	rCharm
	rEnhance
	rHeal
)

type reward struct {
	Kind  int
	Card  Card   // rCard: the new card; rEnhance: the card after
	Index int    // rEnhance: which deck card
	Charm string // rCharm
	Heal  int
}

func (r *Run) makeRewards() {
	r.Rewards = nil
	var charms []string
	for _, c := range charmDefs {
		if !r.has(c.ID) {
			charms = append(charms, c.ID)
		}
	}
	kinds := []int{rCard, rCard, rEnhance, rEnhance, rHeal}
	if len(r.Charms) < 4 && len(charms) > 0 {
		kinds = append(kinds, rCharm, rCharm)
	}
	used := map[int]bool{}
	for len(r.Rewards) < 3 {
		k := kinds[r.rng.Intn(len(kinds))]
		if (k == rHeal || k == rCharm) && used[k] {
			continue
		}
		used[k] = true
		switch k {
		case rCard:
			r.Rewards = append(r.Rewards, reward{Kind: rCard, Card: randomCard(r.rng, r.Floor >= 5)})
		case rCharm:
			c := charms[r.rng.Intn(len(charms))]
			charms = slices.DeleteFunc(charms, func(s string) bool { return s == c })
			r.Rewards = append(r.Rewards, reward{Kind: rCharm, Charm: c})
			if len(charms) == 0 {
				kinds = slices.DeleteFunc(kinds, func(x int) bool { return x == rCharm })
			}
		case rEnhance:
			i := r.rng.Intn(len(r.Deck))
			c := r.Deck[i]
			c.Enh = 1 + r.rng.Intn(eCount-1)
			if c.Enh == eEcho && r.rng.Intn(2) == 0 {
				c.Enh = eFoil
			}
			if c.Enh == r.Deck[i].Enh {
				continue
			}
			r.Rewards = append(r.Rewards, reward{Kind: rEnhance, Card: c, Index: i})
		case rHeal:
			r.Rewards = append(r.Rewards, reward{Kind: rHeal, Heal: max(8, r.Max*3/10)})
		}
	}
}

func (r *Run) Pick(i int) {
	w := r.Rewards[i]
	switch w.Kind {
	case rCard:
		r.Deck = append(r.Deck, w.Card)
	case rCharm:
		r.Charms = append(r.Charms, w.Charm)
	case rEnhance:
		r.Deck[w.Index] = w.Card
	case rHeal:
		r.HP = min(r.Max, r.HP+w.Heal)
	}
	r.Rewards = nil
	r.Floor++
	r.startFight()
}

// Revive refills health and restarts the current fight.
func (r *Run) Revive() {
	r.HP, r.Lost, r.Revived = r.Max, false, true
	r.startFight()
}

// ---- battle ----

type Battle struct {
	run                       *Run
	E                         enemyDef
	EHP, EMax, EBlock         int
	EStr, EVuln, EWeak, EPois int
	moveI                     int
	Intent                    move

	Block, Vuln, Weak   int
	Energy, Turn        int
	Draw, Hand, Discard []Card
	played              int // cards played this turn
	fx                  []fx
}

// fx is a floating number or word for the screen.
type fx struct {
	S     string
	Enemy bool // over the enemy (else over the hero)
	Kind  int  // 0 damage, 1 block, 2 heal, 3 info
	T     int
}

func scale(v, ch int, f float64) int { return int(math.Round(float64(v) * (1 + f*float64(ch-1)))) }

func newBattle(r *Run, d enemyDef) *Battle {
	b := &Battle{run: r, E: d}
	b.EMax = scale(d.HP, r.Chapter, 0.55)
	b.EHP = b.EMax
	b.Draw = slices.Clone(r.Deck)
	r.rng.Shuffle(len(b.Draw), func(i, j int) { b.Draw[i], b.Draw[j] = b.Draw[j], b.Draw[i] })
	b.moveI = r.rng.Intn(len(d.Moves))
	b.Intent = b.nextIntent()
	if r.has("ironskin") {
		b.Block = 8
	}
	b.startTurn()
	return b
}

func (b *Battle) nextIntent() move {
	m := b.E.Moves[b.moveI%len(b.E.Moves)]
	b.moveI++
	if m.Kind == mAttack || m.Kind == mBlock {
		m.Val = scale(m.Val, b.run.Chapter, 0.3)
	}
	return m
}

// IntentDamage is what the enemy's attack will do per hit, after strength and weak.
func (b *Battle) IntentDamage() int {
	d := b.Intent.Val + b.EStr
	if b.EWeak > 0 {
		d = d * 3 / 4
	}
	if b.Vuln > 0 {
		d = d * 3 / 2
	}
	return max(0, d)
}

func (b *Battle) drawCards(n int) {
	for ; n > 0; n-- {
		if len(b.Draw) == 0 {
			if len(b.Discard) == 0 {
				return
			}
			b.Draw, b.Discard = b.Discard, nil
			b.run.rng.Shuffle(len(b.Draw), func(i, j int) { b.Draw[i], b.Draw[j] = b.Draw[j], b.Draw[i] })
		}
		if len(b.Hand) >= 10 {
			return
		}
		b.Hand = append(b.Hand, b.Draw[len(b.Draw)-1])
		b.Draw = b.Draw[:len(b.Draw)-1]
	}
}

func (b *Battle) startTurn() {
	b.Turn++
	if b.Turn > 1 || !b.run.has("ironskin") {
		b.Block = 0
	}
	b.Energy = 3
	if b.Turn == 1 && b.run.has("battery") {
		b.Energy++
	}
	b.played = 0
	n := 5
	if b.run.has("hands") {
		n++
	}
	b.drawCards(n)
}

func (b *Battle) say(s string, enemy bool, kind int) {
	b.fx = append(b.fx, fx{S: s, Enemy: enemy, Kind: kind})
}

func (b *Battle) hitEnemy(d int) {
	if b.EVuln > 0 {
		d = d * 3 / 2
	}
	if b.Weak > 0 {
		d = d * 3 / 4
	}
	taken := max(0, d-b.EBlock)
	b.EBlock = max(0, b.EBlock-d)
	b.EHP -= taken
	b.say(fmt.Sprint(taken), true, 0)
}

// Play plays hand card i. It returns an error when the card can't be played.
func (b *Battle) Play(i int) error {
	c := b.Hand[i]
	d := c.stats()
	if d.Cost > b.Energy {
		return fmt.Errorf("Not enough energy")
	}
	b.Energy -= d.Cost
	b.Hand = append(b.Hand[:i], b.Hand[i+1:]...)
	b.played++
	times := 1
	if c.Enh == eEcho {
		times = 2
	}
	mult := 1
	if b.run.has("combo") && b.played%3 == 0 {
		mult = 2
		b.say("COMBO!", false, 3)
	}
	r := b.run
	for t := 0; t < times; t++ {
		if d.Dmg > 0 {
			dmg := d.Dmg
			if r.has("whetstone") {
				dmg += 2
			}
			for h := 0; h < max(1, d.Hits); h++ {
				b.hitEnemy((dmg + 0) * mult)
			}
		}
		if d.Block > 0 {
			b.Block += d.Block * mult
			b.say(fmt.Sprintf("+%d", d.Block*mult), false, 1)
		}
		b.EPois += d.Poison * mult
		b.EVuln += d.Vuln
		b.EWeak += d.Weak
		if d.Draw > 0 {
			b.drawCards(d.Draw)
		}
		b.Energy += d.Gain
		if d.Heal > 0 {
			r.HP = min(r.Max, r.HP+d.Heal*mult)
			b.say(fmt.Sprintf("+%d", d.Heal*mult), false, 2)
		}
		if c.Enh == eGold {
			r.Gold += 6
		}
	}
	if d.Kind == kAttack && r.has("fang") {
		r.HP = min(r.Max, r.HP+2)
	}
	if !d.Exhaust {
		b.Discard = append(b.Discard, c)
	}
	b.checkWin()
	return nil
}

func (b *Battle) checkWin() {
	if b.EHP > 0 {
		return
	}
	r := b.run
	gold := 12 + r.rng.Intn(9)
	switch r.Floor {
	case 5:
		gold = 40
	case floorsPerChapter:
		gold = 120
		r.Gems += 5
	}
	gold = scale(gold, r.Chapter, 0.5)
	if r.has("coin") {
		gold = gold * 3 / 2
	}
	r.Gold += gold
	if r.Floor == floorsPerChapter {
		r.Won = true
		return
	}
	r.makeRewards()
}

// EndTurn discards the hand and lets the enemy act.
func (b *Battle) EndTurn() {
	if b.EHP <= 0 || b.run.Lost {
		return
	}
	b.Discard = append(b.Discard, b.Hand...)
	b.Hand = nil
	r := b.run
	// enemy turn
	b.EBlock = 0
	if b.EPois > 0 {
		b.EHP -= b.EPois
		b.say(fmt.Sprint(b.EPois), true, 0)
		b.EPois--
		if b.EHP <= 0 {
			b.checkWin()
			return
		}
	}
	m := b.Intent
	switch m.Kind {
	case mAttack:
		for h := 0; h < max(1, m.Hits); h++ {
			d := b.IntentDamage()
			taken := max(0, d-b.Block)
			b.Block = max(0, b.Block-d)
			r.HP -= taken
			b.say(fmt.Sprint(taken), false, 0)
			if r.has("thorns") {
				b.EHP -= 3
			}
		}
	case mBlock:
		b.EBlock += m.Val
		b.say(fmt.Sprintf("+%d", m.Val), true, 1)
	case mBuff:
		b.EStr += m.Val
		b.say("STRONGER", true, 3)
	case mDebuff:
		b.Weak += m.Val + 1
		b.Vuln += m.Val
		b.say("WEAKENED", false, 3)
	}
	b.EVuln = max(0, b.EVuln-1)
	b.EWeak = max(0, b.EWeak-1)
	b.Vuln = max(0, b.Vuln-1)
	b.Weak = max(0, b.Weak-1)
	if r.HP <= 0 {
		r.HP = 0
		r.Lost = true
		return
	}
	if b.EHP <= 0 {
		b.checkWin()
		return
	}
	b.Intent = b.nextIntent()
	b.startTurn()
}
