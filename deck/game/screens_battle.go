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
var enemyHeight = map[int]float64{shGhoul: 130, shWolf: 110, shWraith: 200, shCultist: 175, shSkeleton: 178, shKnight: 240, shGargoyle: 196, shDemon: 300}

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
	es := 1.38
	switch kind {
	case nElite:
		es = 1.2
	case nBoss:
		es = 1.04
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
	drawHero(g.heroSpr.buf(), 150, 260, t, math.Sin(g.swing*math.Pi))
	g.heroSpr.stamp(dst, hx, hy, 150, 260, 1.3, 1, 1, 1, g.heroHit*0.7)

	// bars and statuses
	hpBar(dst, g.heroX(), g.heroY()+20, 170, r.HP, r.Max, colGreen)
	hpBar(dst, g.foeX(), g.foeY()+22, 200, max(0, b.EHP), b.EMax, colRed)
	blockBadge(dst, g.heroX()-112, g.heroY()+31, b.Block)
	blockBadge(dst, g.foeX()-128, g.foeY()+33, b.EBlock)
	statusRow(dst, g.heroX(), g.heroY()+64, []status{{"Vuln", b.Vuln, colOrange}, {"Weak", b.Weak, colGrey}, {"Str", b.Str, colRed}})
	statusRow(dst, g.foeX(), g.foeY()+66, []status{{"Vuln", b.EVuln, colOrange}, {"Weak", b.EWeak, colGrey}, {"Psn", b.EPois, colGreen}, {"Str", b.EStr, colRed}})
	say(dst, b.E.Name, g.foeX(), g.foeY()+94, 20, colBone, 0.5)

	// what the enemy plans
	if b.EHP > 0 && !r.Lost {
		iy := math.Max(140, g.foeY()-enemyHeight[b.E.Shape]*es-34) + math.Sin(t*3)*3
		g.drawIntent(dst, b, g.foeX(), iy)
	}
	g.drawParticles(dst)

	// top: floor, coins, charms, deck
	texRect(dst, 0, 0, W, 76, rgb(0x1e, 0x1a, 0x24), rgb(0x0c, 0x0a, 0x10))
	gradRect(dst, 0, 74, W, 3, colGold, colGoldLo)
	gradRect(dst, 0, 77, W, 40, alpha(colInk, 0.8), clear)
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
	say(dst, label, W/2-40, 40, 28, colBone, 0.5)
	db := g.hudDeck()
	button(dst, db, btnStone, "", 0, false)
	say(dst, "Deck "+itoa(len(r.Deck)), db.cx(), db.y+(db.h-6)/2, 18, colBone, 0.5)
	for i, c := range r.Charms {
		charmIcon(dst, c, W-30-float64(i)*36, 88, 14, t)
	}

	// hand area
	gradRect(dst, 0, g.h-340, W, 340, clear, alpha(colInk, 0.85))
	ox, oy := 66.0, g.h-238
	// energy: a soul flame in a gold ring
	glow(dst, ox, oy, 80, colBlue, 0.3+0.1*math.Sin(t*3))
	for i := 0; i < 8; i++ {
		a := float64(i)*math.Pi/4 + t*0.3
		shape(dst, []float64{ox + math.Cos(a-0.12)*42, oy + math.Sin(a-0.12)*42, ox + math.Cos(a)*52, oy + math.Sin(a)*52, ox + math.Cos(a+0.12)*42, oy + math.Sin(a+0.12)*42}, colGoldHi, colGoldLo, 1)
	}
	disc(dst, ox, oy, 45, 45, colInk)
	radial(dst, ox, oy, 43, 43, -0.3, -0.4, colGoldHi, colGoldLo)
	radial(dst, ox, oy, 36, 36, 0, 0.2, rgb(0x0a, 0x1a, 0x2e), rgb(0x02, 0x06, 0x10))
	for i := 0; i < 3; i++ {
		f := 26 - float64(i)*7
		w := math.Sin(t*6+float64(i)) * 3
		blob(dst, []float64{ox - f*0.6, oy + 14, ox - f*0.3 + w, oy - f*0.4, ox + w*1.5, oy - f - 4, ox + f*0.35 - w, oy - f*0.3, ox + f*0.6, oy + 14}, alpha(light(colBlue, 0.3+float64(i)*0.3), 0.8), alpha(colBlue, 0.6), 0)
	}
	say(dst, itoa(b.Energy)+"/3", ox, oy+4, 26, colWhite, 0.5)
	sayBody(dst, "Draw "+itoa(len(b.Draw))+"   Used "+itoa(len(b.Discard)), ox+68, g.h-200, 15, colCream, 0)
	if g.endT == 0 && b.EHP > 0 {
		button(dst, g.endTurnBtn(), btnFire, "END TURN", 19, false)
	} else {
		button(dst, g.endTurnBtn(), btnStone, "ENEMY TURN", 17, true)
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
	ib := rect{x - 52, y - 22, 104, 44}
	shape(dst, []float64{x - 9, ib.y + ib.h, x + 9, ib.y + ib.h, x, ib.y + ib.h + 11}, colGoldHi, colGoldLo, 1.2)
	fillPath(dst, chamfer(ib.grow(3), 13), colInk)
	goldEdge(dst, chamfer(ib.grow(2), 12), chamfer(ib.grow(0.8), 11))
	texPath(dst, chamfer(ib, 10), rgb(0x2a, 0x24, 0x30), rgb(0x0c, 0x0a, 0x10))
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
		fillPath(dst, chamfer(r.grow(2), 7), colInk)
		texPath(dst, chamfer(r, 6), shade(s.c, 0.7), shade(s.c, 0.25))
		strokePath(dst, chamfer(r, 6), 1, alpha(light(s.c, 0.3), 0.9))
		drawText(dst, s.name+" "+itoa(s.n), r.cx(), r.cy(), 14, colBone, 0.5)
		x += 60
	}
}

func blockBadge(dst *ebiten.Image, x, y float64, n int) {
	if n <= 0 {
		return
	}
	shieldIcon(dst, x, y, 20, btnArcane)
	say(dst, itoa(n), x, y-1, 17, colWhite, 0.5)
}
