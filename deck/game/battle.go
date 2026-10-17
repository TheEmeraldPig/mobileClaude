package game

import (
	"fmt"
	"slices"
)

type Battle struct {
	run                       *Run
	E                         enemyDef
	EHP, EMax, EBlock         int
	EStr, EVuln, EWeak, EPois int
	moveI                     int
	Intent                    move

	Block, Vuln, Weak, Str int
	Energy, Turn           int
	Draw, Hand, Discard    []Card
	played                 int // cards played this turn
	fx                     []fx
}

// fx is something that happened, for the screen to show.
type fx struct {
	S     string
	Enemy bool // on the enemy (else on the hero)
	Kind  int
}

const (
	fxDamage = iota
	fxBlock
	fxHeal
	fxPoison
	fxInfo
	fxBuff
)

func newBattle(r *Run, d enemyDef, mult float64) *Battle {
	b := &Battle{run: r, E: d}
	b.EMax = int(float64(scaleCh(d.HP, r.Chapter, 0.6)) * mult)
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
		m.Val = scaleCh(m.Val, b.run.Chapter, 0.3)
	}
	return m
}

// IntentDamage is what the enemy's attack will do per hit.
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
	if b.Turn > 1 && !b.run.has("tower") {
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
	b.say(itoa(taken), true, fxDamage)
}

// Playable says whether hand card i can be played now.
func (b *Battle) Playable(i int) bool {
	return b.EHP > 0 && !b.run.Lost && b.Hand[i].stats().Cost <= b.Energy
}

// Play plays hand card i.
func (b *Battle) Play(i int) error {
	c := b.Hand[i]
	d := c.stats()
	if b.EHP <= 0 || b.run.Lost {
		return fmt.Errorf("The fight is over")
	}
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
		b.say("COMBO!", false, fxInfo)
	}
	r := b.run
	for t := 0; t < times; t++ {
		if d.Dmg > 0 {
			dmg := d.Dmg + b.Str
			if r.has("whetstone") {
				dmg += 2
			}
			for h := 0; h < max(1, d.Hits); h++ {
				b.hitEnemy(dmg * mult)
			}
		}
		if d.Block > 0 {
			b.Block += d.Block * mult
			b.say("+"+itoa(d.Block*mult), false, fxBlock)
		}
		if d.Poison > 0 {
			p := d.Poison
			if r.has("venom") {
				p += 2
			}
			b.EPois += p * mult
			b.say(fmt.Sprintf("+%d poison", p*mult), true, fxPoison)
		}
		b.EVuln += d.Vuln
		b.EWeak += d.Weak
		if d.Str > 0 {
			b.Str += d.Str * mult
			b.say(fmt.Sprintf("+%d STR", d.Str*mult), false, fxBuff)
		}
		if d.Draw > 0 {
			b.drawCards(d.Draw)
		}
		b.Energy += d.Gain
		if d.Heal > 0 {
			r.HP = min(r.Max, r.HP+d.Heal*mult)
			b.say("+"+itoa(d.Heal*mult), false, fxHeal)
		}
		if c.Enh == eGold {
			r.Coins += 6
		}
	}
	if d.Kind == kAttack && r.has("fang") {
		r.HP = min(r.Max, r.HP+2)
	}
	if !d.Exhaust {
		b.Discard = append(b.Discard, c)
	}
	if b.EHP <= 0 {
		r.winFight()
	}
	return nil
}

// EndTurn discards the hand and lets the enemy act.
func (b *Battle) EndTurn() {
	if b.EHP <= 0 || b.run.Lost {
		return
	}
	b.Discard = append(b.Discard, b.Hand...)
	b.Hand = nil
	r := b.run
	b.EBlock = 0
	if b.EPois > 0 {
		b.EHP -= b.EPois
		b.say(itoa(b.EPois), true, fxPoison)
		b.EPois--
		if b.EHP <= 0 {
			r.winFight()
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
			b.say(itoa(taken), false, fxDamage)
			if r.has("thorns") {
				b.EHP -= 3
			}
		}
	case mBlock:
		b.EBlock += m.Val
		b.say("+"+itoa(m.Val), true, fxBlock)
	case mBuff:
		b.EStr += m.Val
		b.say("STRONGER", true, fxBuff)
	case mDebuff:
		b.Weak += m.Val + 1
		b.Vuln += m.Val
		b.say("CURSED", false, fxInfo)
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
		r.winFight()
		return
	}
	b.Intent = b.nextIntent()
	b.startTurn()
}
