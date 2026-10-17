package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Interface animation: screen fades, how long the current screen has been up
// (for staggered entrances), counters that count, health bars that drain,
// the hero walking the map, and the pressed-button state.

var (
	pressX, pressY float64 // where the finger is, in layout units
	pressOn        bool
	animT          float64 // seconds since start, for idle shimmer
)

type uiAnim struct {
	fadeT, scT       float64
	lastSc           screen
	lastTab          int
	lastView         bool
	dGold, dGems     float64
	dCoins, dHP      float64
	trailHP, trailEH float64
	popT             float64
	travelT          float64
	travelTo         int
	navX             float64
	intentPop        float64
	lastIntent       move
	handDelay        []int
	cardSpr          *sprite
}

func approach(v *float64, target, rate float64) {
	if math.Abs(target-*v) < 0.5 {
		*v = target
		return
	}
	*v += (target - *v) * rate
}

func (g *Game) tickAnim() {
	animT = g.t()
	pressX, pressY, pressOn = g.ptr.x, g.ptr.y, g.ptr.down && !g.ptr.moved
	if g.sc != g.lastSc || g.tab != g.lastTab || g.viewDeck != g.lastView {
		if g.sc != g.lastSc || g.viewDeck != g.lastView {
			g.fadeT = 1
		}
		g.scT = 0
		g.lastSc, g.lastTab, g.lastView = g.sc, g.tab, g.viewDeck
	}
	g.scT += 1.0 / 60
	g.fadeT = math.Max(0, g.fadeT-0.07)
	approach(&g.dGold, float64(g.p.Gold), 0.15)
	approach(&g.dGems, float64(g.p.Gems), 0.15)
	approach(&g.navX, float64(g.tab), 0.25)
	if g.popup != "" {
		g.popT = math.Min(1, g.popT+1.0/40)
	} else {
		g.popT = 0
	}
	g.intentPop = math.Max(0, g.intentPop-0.06)
	if r := g.run(); r != nil {
		approach(&g.dCoins, float64(r.Coins), 0.15)
		approach(&g.dHP, float64(r.HP), 0.2)
		if float64(r.HP) > g.trailHP {
			g.trailHP = float64(r.HP)
		} else if g.heroHit < 0.5 {
			g.trailHP = math.Max(float64(r.HP), g.trailHP-0.6)
		}
		if r.B != nil {
			if float64(r.B.EHP) > g.trailEH {
				g.trailEH = float64(r.B.EHP)
			} else if g.enemyHit < 0.5 {
				g.trailEH = math.Max(float64(max(0, r.B.EHP)), g.trailEH-0.6)
			}
			if r.B.Intent != g.lastIntent {
				g.lastIntent, g.intentPop = r.B.Intent, 1
			}
		}
		if g.travelT > 0 {
			g.travelT += 1.0 / 40
			if g.travelT >= 1 {
				g.travelT = 0
				g.travel(g.travelTo)
			}
		}
	}
}

// drawFade darkens the screen just after it changes.
func (g *Game) drawFade(dst *ebiten.Image) {
	if g.fadeT > 0 {
		fillRect(dst, 0, 0, W, g.h, color.RGBA{0, 0, 0, uint8(255 * easeIn(g.fadeT))})
	}
}

// enter returns 0..1 for an element appearing delay seconds after the screen opened.
func (g *Game) enter(delay, dur float64) float64 { return clamp01((g.scT - delay) / dur) }

// drawCardFx draws a card that can flip (sx from -1 to 1; negative shows its back),
// scale and fade. (x,y) is the card's centre.
func (g *Game) drawCardFx(dst *ebiten.Image, c Card, x, y, w, sx, fade, t float64) {
	if g.cardSpr == nil {
		g.cardSpr = newSprite(280, 380)
	}
	buf := g.cardSpr.buf()
	h := w * cardH / cardW
	if sx < 0 {
		drawCardBack(buf, 20, 20, w, t)
	} else {
		drawCard(buf, c, 20, 20, w, false, t)
	}
	g.cardSpr.stamp(dst, x, y, 20+w/2, 20+h/2, 1, math.Abs(sx)+0.001, 1, fade, 0)
}

// drawCardBack is the back of a card: dark iron, gold filigree, a red eye.
func drawCardBack(dst *ebiten.Image, x, y, w, t float64) {
	s := w / cardW
	h := cardH * s
	r := rect{x, y, w, h}
	fillPath(dst, chamfer(r.grow(3.5*s), 12*s), colInk)
	goldEdge(dst, chamfer(r.grow(2.5*s), 11*s), chamfer(r.grow(1*s), 10*s))
	texPath(dst, chamfer(r, 9*s), rgb(0x2a, 0x16, 0x1e), rgb(0x0c, 0x06, 0x0a))
	strokePath(dst, chamfer(r.grow(-7*s), 6*s), 1.2*s, alpha(colGold, 0.6))
	runeCircle(dst, r.cx(), r.cy(), 34*s, t, colGold, 0.5)
	shape(dst, []float64{r.cx() - 22*s, r.cy(), r.cx(), r.cy() - 12*s, r.cx() + 22*s, r.cy(), r.cx(), r.cy() + 12*s}, colGoldHi, colGoldLo, 1.2*s)
	glow(dst, r.cx(), r.cy(), 18*s, colRed, 0.7)
	disc(dst, r.cx(), r.cy(), 6*s, 6*s, rgb(0xff, 0x40, 0x30))
	disc(dst, r.cx(), r.cy(), 2*s, 5*s, colInk)
}
