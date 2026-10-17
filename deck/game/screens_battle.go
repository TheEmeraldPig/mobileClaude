package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const handCW = 104.0

func (g *Game) handCard(i, n int) rect {
	x0, x1 := 12.0, W-12.0
	step := handCW + 6
	if n > 1 {
		step = math.Min(step, (x1-x0-handCW)/float64(n-1))
	}
	total := step*float64(n-1) + handCW
	return rect{x0 + (x1-x0-total)/2 + step*float64(i), g.h - 168, handCW, handCW * 1.4}
}
func (g *Game) endTurnBtn() rect { return rect{356, g.h - 268, 166, 66} }

func (g *Game) clickBattle(x, y float64) {
	r := g.run()
	b := r.B
	if g.endTurnBtn().has(x, y) {
		g.press("endturn")
		return
	}
	for i := len(b.Hand) - 1; i >= 0; i-- {
		hr := g.handCard(i, len(b.Hand))
		if i < len(g.handPos) {
			hr.x, hr.y = g.handPos[i].x, g.handPos[i].y
		}
		if hr.has(x, y) {
			g.press("play:" + itoa(i))
			return
		}
	}
	if y < 110 {
		for i, c := range r.Charms {
			if math.Hypot(x-(W-30-float64(i)*36), y-88) < 16 {
				ch := charmByID(c)
				g.say(ch.Name + ": " + ch.Desc)
			}
		}
		if g.hudDeck().has(x, y) {
			g.press("deck")
		}
	}
}

// enemyHeight is how tall each enemy stands, to place its intent bubble.
var enemyHeight = map[int]float64{shSlime: 96, shGoblin: 122, shBat: 136, shShroom: 118, shSkeleton: 140, shOrc: 160, shGolem: 162, shKing: 214}

func (g *Game) drawBattle(dst *ebiten.Image) {
	r := g.run()
	b := r.B
	t := g.t()
	th := chapterTheme(r.Chapter)
	sx := (g.rng.Float64() - 0.5) * g.shake
	sy := (g.rng.Float64() - 0.5) * g.shake
	dst.DrawImage(g.battleBG(r.Chapter), imgOp(sx, sy, 1))

	// enemy
	kind := r.Nodes[r.Cur].Kind
	es := 1.32
	switch kind {
	case nElite:
		es = 1.38
	case nBoss:
		es = 1.36
	}
	ex, ey := g.foeX()+sx-g.lunge*70, g.foeY()+sy+g.lunge*40
	fade := 1.0
	if b.EHP <= 0 {
		fade = math.Max(0, 1-float64(g.deathT)/30)
	}
	if fade > 0 {
		drawEnemy(g.foeSpr.buf(), b.E.Shape, 180, 330, t, th.Hue)
		squash := 1 + g.enemyHit*0.08
		g.foeSpr.stamp(dst, ex, ey, 180, 330, es, squash, 2-squash, fade, g.enemyHit*0.7)
	}
	// hero
	hx, hy := g.heroX()+sx+g.swing*30, g.heroY()+sy-g.swing*16
	drawHero(g.heroSpr.buf(), 130, 240, t, math.Sin(g.swing*math.Pi))
	g.heroSpr.stamp(dst, hx, hy, 130, 240, 1.45, 1, 1, 1, g.heroHit*0.7)

	// bars and statuses
	hpBar(dst, g.heroX(), g.heroY()+20, 170, r.HP, r.Max, colGreen)
	hpBar(dst, g.foeX(), g.foeY()+22, 200, max(0, b.EHP), b.EMax, colRed)
	blockBadge(dst, g.heroX()-112, g.heroY()+31, b.Block)
	blockBadge(dst, g.foeX()-128, g.foeY()+33, b.EBlock)
	statusRow(dst, g.heroX(), g.heroY()+64, []status{{"Vuln", b.Vuln, colOrange}, {"Weak", b.Weak, colGrey}, {"Str", b.Str, colRed}})
	statusRow(dst, g.foeX(), g.foeY()+66, []status{{"Vuln", b.EVuln, colOrange}, {"Weak", b.EWeak, colGrey}, {"Psn", b.EPois, colGreen}, {"Str", b.EStr, colRed}})
	say(dst, b.E.Name, g.foeX(), g.foeY()+94, 20, colWhite, 0.5)

	// what the enemy plans
	if b.EHP > 0 && !r.Lost {
		iy := math.Max(140, g.foeY()-enemyHeight[b.E.Shape]*es-34) + math.Sin(t*3)*3
		g.drawIntent(dst, b, g.foeX(), iy)
	}
	g.drawParticles(dst)

	// top: floor, coins, charms, deck
	gradRect(dst, 0, 0, W, 112, alpha(rgb(0x10, 0x10, 0x24), 0.7), clear)
	cr := rect{20, 22, 112, 36}
	pill(dst, cr, itoa(r.Coins))
	coinIcon(dst, cr.x+8, cr.cy(), 14)
	label := "FLOOR " + itoa(r.floor())
	switch kind {
	case nElite:
		label = "ELITE"
	case nBoss:
		label = "BOSS"
	}
	say(dst, label, W/2-40, 40, 28, colWhite, 0.5)
	db := g.hudDeck()
	button(dst, db, rgb(0x5a, 0x6a, 0xc8), "", 0, false)
	say(dst, "Deck "+itoa(len(r.Deck)), db.cx(), db.y+(db.h-6)/2, 18, colWhite, 0.5)
	for i, c := range r.Charms {
		charmIcon(dst, c, W-30-float64(i)*36, 88, 14, t)
	}

	// hand area
	gradRect(dst, 0, g.h-330, W, 330, clear, alpha(rgb(0x08, 0x08, 0x18), 0.75))
	ox, oy := 66.0, g.h-238
	glow(dst, ox, oy, 70, colOrange, 0.35+0.1*math.Sin(t*3))
	orb(dst, ox, oy, 42, 42, rgb(0xff, 0x9a, 0x20))
	radial(dst, ox, oy, 32, 32, -0.2, -0.3, rgb(0xff, 0xf0, 0x80), rgb(0xff, 0xa8, 0x20))
	say(dst, itoa(b.Energy)+"/3", ox, oy-2, 30, colWhite, 0.5)
	say(dst, "Draw "+itoa(len(b.Draw))+"   Used "+itoa(len(b.Discard)), ox+68, g.h-200, 14, colCream, 0)
	if g.endT == 0 && b.EHP > 0 {
		button(dst, g.endTurnBtn(), colGreen, "END TURN", 22, false)
	} else {
		button(dst, g.endTurnBtn(), colGrey, "ENEMY", 22, true)
	}
	// cards slide from the draw pile to their places
	for len(g.handPos) < len(b.Hand) {
		g.handPos = append(g.handPos, pt{20, g.h - 230})
	}
	g.handPos = g.handPos[:len(b.Hand)]
	for i, c := range b.Hand {
		tr := g.handCard(i, len(b.Hand))
		p := &g.handPos[i]
		p.x += (tr.x - p.x) * 0.25
		p.y += (tr.y - p.y) * 0.25
		lift := 0.0
		if b.Playable(i) && g.endT == 0 {
			lift = 4 + 2*math.Sin(t*3+float64(i))
		}
		drawCard(dst, c, p.x, p.y-lift, tr.w, !b.Playable(i), t)
	}
	for _, f := range g.flyCards {
		e := 1 - (1-f.t)*(1-f.t)
		w := f.w * (1 - 0.6*e)
		x := f.x + (f.tx-w/2-f.x)*e
		y := f.y + (f.ty-w*0.7-f.y)*e
		drawCard(dst, f.c, x, y, w, false, t)
		glow(dst, x+w/2, y+w*0.7, w, colWhite, 0.4*e)
	}
	for _, f := range g.fl {
		a := math.Min(1, (60-f.t)/18)
		pop := 1 + 0.4*math.Max(0, 1-f.t/8)
		say(dst, f.s, f.x, f.y, f.size*pop, alpha(f.c, a), 0.5)
	}
}

func (g *Game) drawIntent(dst *ebiten.Image, b *Battle, x, y float64) {
	m := b.Intent
	ib := rect{x - 50, y - 24, 100, 46}
	roundRect(dst, ib.grow(3), 26, colInk)
	roundGrad(dst, ib, 23, colWhite, rgb(0xe0, 0xe4, 0xf0))
	shape(dst, []float64{x - 8, ib.y + ib.h, x + 8, ib.y + ib.h, x, ib.y + ib.h + 10}, rgb(0xe0, 0xe4, 0xf0), rgb(0xe0, 0xe4, 0xf0), 0)
	switch m.Kind {
	case mAttack:
		swordIcon(dst, ib.x+26, ib.cy(), 16)
		v := itoa(b.IntentDamage())
		if m.Hits > 1 {
			v += "x" + itoa(m.Hits)
		}
		say(dst, v, ib.x+68, ib.cy(), 22, colRed, 0.5)
	case mBlock:
		shieldIcon(dst, ib.x+26, ib.cy(), 15, colBlue)
		say(dst, itoa(m.Val), ib.x+68, ib.cy(), 22, colBlue, 0.5)
	case mBuff:
		shape(dst, []float64{ib.x + 26, ib.y + 8, ib.x + 40, ib.y + 24, ib.x + 31, ib.y + 24, ib.x + 31, ib.y + 38, ib.x + 21, ib.y + 38, ib.x + 21, ib.y + 24, ib.x + 12, ib.y + 24}, colOrange, shade(colOrange, 0.7), 2)
		say(dst, "BUFF", ib.x+66, ib.cy(), 18, colOrange, 0.5)
	case mDebuff:
		cardArt(dst, aCloud, ib.x+26, ib.cy(), 14, g.t(), rEpic)
		say(dst, "CURSE", ib.x+68, ib.cy(), 16, colPurple, 0.5)
	}
}

type status struct {
	name string
	n    int
	c    color.RGBA
}

func statusRow(dst *ebiten.Image, cx, y float64, ss []status) {
	var on []status
	for _, s := range ss {
		if s.n > 0 {
			on = append(on, s)
		}
	}
	x := cx - float64(len(on))*60/2
	for _, s := range on {
		r := rect{x, y - 12, 56, 24}
		roundRect(dst, r.grow(2), 12, colInk)
		roundGrad(dst, r, 10, light(s.c, 0.2), shade(s.c, 0.7))
		drawText(dst, s.name+" "+itoa(s.n), r.cx(), r.cy(), 14, colWhite, 0.5)
		x += 60
	}
}

func blockBadge(dst *ebiten.Image, x, y float64, n int) {
	if n <= 0 {
		return
	}
	shieldIcon(dst, x, y, 20, colBlue)
	say(dst, itoa(n), x, y-2, 18, colWhite, 0.5)
}
