package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// The DNA lab, the token shop, idle gains and resetting the save.

// ---- DNA lab: spend DNA to make each mutation worth more, appear more often and
// spread further. It lives in the GENOME screen (the old catalog). ----

const (
	labValue  = iota // +25% value a level
	labChance        // +20% chance a level
	labSpread        // +50% spread a level
	nLab
)

var labNames = []string{"VALUE", "CHANCE", "SPREAD"}

const labMax = 20

// spreadBase is the chance a neighbour's mutation reaches a new biote, before mods.
const spreadBase = 0.03

func (g *Game) lab(kind, r int) int {
	if kind < len(g.st.Lab) && r < len(g.st.Lab[kind]) {
		return g.st.Lab[kind][r]
	}
	return 0
}

// labCost is the DNA for the next level: rarer mutations cost more.
func labCost(r, lv int) int { return (1 + (r-1)/2) * (lv + 1) }

func (g *Game) buyLab(kind, r int) {
	lv := g.lab(kind, r)
	c := labCost(r, lv)
	if lv >= labMax || g.st.Gems < c || g.st.Found[r] == 0 {
		g.play("deny")
		return
	}
	g.st.Gems -= c
	g.st.Lab[kind][r]++
	g.play("buy")
	g.save()
}

// ---- idle gains: while the game is closed you earn a share of your recent income ----

const (
	idleShare = 0.25     // of what you were earning per second
	idleMax   = 8 * 3600 // seconds counted at most
)

// noteSale keeps a running average of income per second, for idle gains.
func (g *Game) noteSale(v float64) {
	if g.lastSale > 0 {
		gap := math.Max(1, g.t-g.lastSale)
		g.st.Rate = g.st.Rate*0.85 + (v/gap)*0.15
	}
	g.lastSale = g.t
}

// idleRate is the money per second earned while away.
func (g *Game) idleRate() float64 {
	r := g.st.Rate * idleShare * g.m.offline
	if g.st.Perks["offline2"] {
		r *= 2
	}
	return r
}

// giveIdle pays sec seconds of idle gains and shows the welcome-back screen.
func (g *Game) giveIdle(sec float64) {
	sec = math.Min(sec, idleMax)
	gain := g.idleRate() * sec
	g.st.Money += gain
	g.st.Earned += gain
	g.away, g.awayGain, g.awayDoubled = sec, gain, false
	g.panel = "away"
}

// incomeRate is money per second while playing: your recent income, or before
// there is any, a guess from the board you can fill. Time skips pay this.
func (g *Game) incomeRate() float64 {
	c := float64(g.capacity())
	guess := c * g.baseValueAt(g.st.ValueLv) / (g.cooldown() * (math.Log2(c) + 1))
	return math.Max(g.st.Rate, guess)
}

// checkAway runs once at start: pays for the time since the game was last saved.
func (g *Game) checkAway() {
	if g.st.LastSeen == 0 {
		return
	}
	if sec := float64(time.Now().Unix() - g.st.LastSeen); sec > 60 && g.st.Rate > 0 {
		g.giveIdle(sec)
	}
}

func fmtDur(sec float64) string {
	s := int(sec)
	if s >= 3600 {
		return fmt.Sprintf("%dh %dm", s/3600, s%3600/60)
	}
	return fmt.Sprintf("%dm %ds", s/60, s%60)
}

// ---- reset ----

func (g *Game) resetSave() {
	keepSound, keepMusic := g.st.Sound, g.st.NoMusic
	g.st = newState()
	g.st.Sound, g.st.NoMusic = keepSound, keepMusic
	g.blobs = []blob{{m: 0, born: g.t, mt: -10, squash: -10, phase: rand.Float64() * 7}}
	g.shown, g.cool, g.confirm = 0, 99, false
	g.panel = ""
	g.save()
	g.play("rebirth")
}
