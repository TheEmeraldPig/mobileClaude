package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Drawing the fight (or the kitchen), the HUD and the crew bar.

func (g *Game) Draw(dst *ebiten.Image) {
	var t0 time.Time
	if timing {
		t0 = time.Now()
	}
	g.hits = g.hits[:0]
	dst.Fill(colPanelD)
	if g.tab == tabKitchen {
		g.drawKitchen(dst)
		g.drawIngredientBar(dst)
	} else {
		g.drawField(dst)
		g.drawSkillBar(dst)
	}
	g.drawHUD(dst)
	g.drawPanel(dst)
	g.drawTabs(dst)
	g.drawOverlays(dst)
	if g.toastT > 0 {
		a := clamp01(g.toastT * 3)
		y := g.tabsY() - 36 - easeOut(math.Min(1, (2.6-g.toastT)*5))*10
		w := textW(fBody, g.toast, 19) + 40
		panel(dst, rect{W/2 - w/2, y - 22, w, 44}, 3)
		sayB(dst, g.toast, W/2, y, 19, alpha(colWhite, a), 0.5)
	}
	if g.script != nil {
		g.script.afterDraw(dst)
	}
	if timing && g.frame%60 == 0 {
		fmt.Printf("draw %.2fms\n", float64(time.Since(t0).Microseconds())/1000)
	}
}

// ---- layers ----

func drawLayer(dst *ebiten.Image, p *pimg, pre, body ebiten.GeoM, cs ebiten.ColorScale, blend ebiten.Blend) {
	if p == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: blend}
	op.GeoM.Scale(1/p.k, 1/p.k)
	op.GeoM.Translate(p.ox, p.oy)
	op.GeoM.Concat(pre)
	op.GeoM.Concat(body)
	op.GeoM.Scale(S, S)
	op.ColorScale = cs
	dst.DrawImage(p.img, op)
}

func bodyM(x, y, sx, sy, rot, skew float64) ebiten.GeoM {
	var m ebiten.GeoM
	m.Scale(sx, sy)
	if skew != 0 {
		m.Skew(skew, 0)
	}
	if rot != 0 {
		m.Rotate(rot)
	}
	m.Translate(x, y)
	return m
}

// drawChef draws chef id's bust with the chin at x,y. face: 0 normal, 1 blink, 2 shout.
func drawChef(dst *ebiten.Image, id int, x, y, sx, sy float64, face int, cs ebiten.ColorScale) {
	z := 0.7
	if sx > 0.75 {
		z = 1.6
	}
	art := chefArt(id, z)
	m := bodyM(x, y, sx, sy, 0, 0)
	var id0 ebiten.GeoM
	drawLayer(dst, art.body, id0, m, cs, ebiten.BlendSourceOver)
	drawLayer(dst, art.face[face], id0, m, cs, ebiten.BlendSourceOver)
}

// portrait draws a chef bust fitted to a round frame size units across.
func portrait(dst *ebiten.Image, id int, cx, cy, size float64, face int, t float64, dark bool) {
	sc := size / 120
	cs := cs1
	if dark {
		cs = ebiten.ColorScale{}
		cs.Scale(0.16, 0.1, 0.3, 1)
	}
	c := clip(dst, rect{cx - size, cy - size*1.3, size * 2, size*1.3 + size*0.45})
	drawChef(c, id, cx, cy+18*sc, sc, sc*(1+0.015*math.Sin(t*3)), face, cs)
}

func shadowAt(dst *ebiten.Image, x, y, w, a float64) {
	var cs ebiten.ColorScale
	cs.Scale(0.12*float32(a), 0.04*float32(a), 0.2*float32(a), 0.45*float32(a))
	glowArt().drawT(dst, x, y, w/100, w/100*0.22, 0, cs)
}

func glowCS(c color.RGBA, a float64) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.Scale(float32(c.R)/255*float32(a), float32(c.G)/255*float32(a), float32(c.B)/255*float32(a), float32(a))
	return cs
}

func tintCS(c color.RGBA, a float64) ebiten.ColorScale { return glowCS(c, a) }

// ---- the fight ----

func (g *Game) drawMonster(dst *ebiten.Image) {
	m := g.mon
	if m == nil {
		return
	}
	sc := g.monScale(m)
	z := 1.45
	art := monArt(m.kind, m.boss, z)
	gy := g.groundY()
	x := g.monX()
	k := math.Sin(m.phase)
	sx, sy := 1-0.025*k, 1+0.035*k
	rot := math.Sin(m.phase*0.5) * 0.03
	dy := 0.0
	// drop in from above and land with a squash
	if m.spawn < 1 {
		f := m.spawn
		if f < 0.55 {
			dy = -(1 - easeIn(f/0.55)) * 380
		} else {
			q := math.Sin((f - 0.55) / 0.45 * math.Pi)
			sx *= 1 + 0.18*q
			sy *= 1 - 0.18*q
		}
	}
	if m.squish > 0.01 {
		w := math.Sin(g.t*45) * m.squish
		sx += 0.08 * w
		sy -= 0.08 * w
	}
	a := 1.0
	if m.dead > 0 {
		d := easeOut(m.dead)
		if m.gone {
			x += d * 360
			rot += d * 0.4
			a = 1 - d
		} else {
			sy *= 1 - d*0.8
			sx *= 1 + d*0.5
			a = 1 - easeIn(m.dead)
		}
	}
	// a soft spotlight behind the monster so it stands out from the backdrop
	glowArt().drawT(dst, x, gy-100*sc, 2.4*sc, 2.1*sc, 0, glowCS(rgb(0xff, 0xf4, 0xd8), 0.38*a))
	shadowAt(dst, x, gy, 210*sc*sx*(1-math.Min(0.6, -dy/400)), a)
	mm := bodyM(x, gy+dy, sx*sc, sy*sc, rot, 0)
	var id ebiten.GeoM
	drawLayer(dst, art.body, id, mm, csA(a), ebiten.BlendSourceOver)
	face := 0
	if m.hurt > 0 || m.dead > 0 && !m.gone {
		face = 1
	}
	drawLayer(dst, art.face[face], id, mm, csA(a), ebiten.BlendSourceOver)
	if m.hit > 0 {
		var cs ebiten.ColorScale
		f := float32(m.hit * 0.55 * a)
		cs.Scale(f, f, f, f)
		drawLayer(dst, art.body, id, mm, cs, ebiten.BlendLighter)
	}
}

func (g *Game) drawField(dst *ebiten.Image) {
	fh := g.fieldH()
	ar := areaOf(g.p.Stage)
	f := clip(dst, rect{0, 0, W, fh})
	ox, oy := 0.0, 0.0
	if g.shake > 0 {
		ox = math.Sin(g.t*70) * g.shake * 6
		oy = math.Cos(g.t*55) * g.shake * 4
	}
	bg := paintArt("scene"+itoa(ar.kind)+"@"+num(fh), 0, 0, W, fh, 1, func(c *canvas) {
		paintScene(c, ar, W, fh, g.groundY())
	})
	var dim ebiten.ColorScale // the backdrop sits back so the monster pops
	dim.Scale(0.8, 0.78, 0.84, 1)
	bg.draw(f, ox, oy, 1, dim)
	if ar.kind == akGarden {
		cl := paintArt("cloud", -70, -40, 70, 12, 1, paintCloud)
		for i := 0; i < 2; i++ {
			x := math.Mod(g.t*(5+float64(i)*3)+float64(i)*300, W+200) - 100
			cl.draw(f, x+ox, 110+float64(i)*46+oy, 0.8+float64(i)*0.2, cs1)
		}
	}
	g.drawMonster(f)
	for _, s := range g.shots {
		x, y := s.pos()
		k := 0.75 + 0.25*s.t
		foodIcon(s.icon, 40).drawT(f, x, y, s.big*k, s.big*k, s.t*s.spin, cs1)
	}
	g.drawParticles(f, false)
	for _, fl := range g.fl {
		k := fl.t / fl.life
		y := fl.y - easeOut(k)*50
		a := 1 - easeIn(math.Max(0, (k-0.6)/0.4))
		sc := 0.6 + 0.4*backOut(math.Min(1, k*5))
		say(f, fl.s, fl.x, y, fl.size*sc, alpha(fl.col, a), 0.5)
	}
	if g.flash > 0 {
		fillRect(f, 0, 0, W, fh, alpha(colWhite, g.flash*0.6))
	}
	gradRect(f, 0, fh-26, W, 26, alpha(colInk, 0), alpha(colInk, 0.35))
}

func (g *Game) drawParticles(dst *ebiten.Image, steam bool) {
	for _, q := range g.parts {
		if (q.kind == ptSteam) != steam {
			continue
		}
		k := q.t / q.life
		a := 1 - easeIn(k)
		switch q.kind {
		case ptPuff:
			puffArt().draw(dst, q.x, q.y, q.size*(0.6+0.8*easeOut(k)), csA(a*0.95))
		case ptSpark:
			glowArt().draw(dst, q.x, q.y, 0.2*q.size*(1-k*0.5), glowCS(q.col, a))
		case ptStar:
			sparkArt().drawT(dst, q.x, q.y, 0.55*q.size*(1-k), 0.55*q.size*(1-k), q.rot+k*3, glowCS(light(q.col, 0.5), 1))
		case ptCoin:
			icon(icCoin, 24).drawT(dst, q.x, q.y, math.Cos(q.t*12)*q.size, q.size, 0, cs1)
		case ptIng:
			foodIcon(q.icon, 30).drawT(dst, q.x, q.y, q.size, q.size, math.Sin(q.t*8)*0.3, cs1)
		case ptGoo:
			dropArt().draw(dst, q.x, q.y, q.size*(1-k*0.5), tintCS(q.col, a))
		case ptSteam:
			puffArt().draw(dst, q.x, q.y, q.size*(0.4+0.8*k), csA(a*0.5))
		case ptConfetti:
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-0.5, -0.5)
			op.GeoM.Scale(10*S, 5*S*math.Abs(math.Cos(q.t*8+q.rot)))
			op.GeoM.Rotate(q.rot)
			op.GeoM.Translate(q.x*S, q.y*S)
			op.ColorScale = tintCS(q.col, a)
			dst.DrawImage(whiteSub, op)
		}
	}
}

func puffArt() *pimg {
	return paintArt("puff", -24, -24, 24, 24, 1, func(c *canvas) {
		s := circ(0, 0, 14).melt(circ(-9, 4, 10), 4).melt(circ(9, 3, 11), 4).melt(circ(2, -9, 10), 4)
		c.vol(s, mat{base: colWhite, dark: rgb(0xd0, 0xc8, 0xf0), flat: 0.3})
	})
}

func dropArt() *pimg {
	return paintArt("drop", -12, -12, 12, 12, 1, func(c *canvas) {
		c.vol(circ(0, 0, 9), mat{base: colWhite, dark: rgb(0x90, 0x90, 0x90), gloss: 1, shine: 20})
	})
}

// ---- HUD ----

func (g *Game) drawHUD(dst *ebiten.Image) {
	p := g.p
	cp := 1 + 0.15*g.coinPulse
	panel(dst, rect{10, 12, 176, 42}, 3)
	icon(icCoin, 34).draw(dst, 30, 33, cp, cs1)
	say(dst, num(g.shownCoin), 54, 33, 24, colWhite, 0)
	gp := 1 + 0.15*g.gemPulse
	panel(dst, rect{W - 156, 12, 146, 42}, 3)
	icon(icGem, 32).draw(dst, W-136, 33, gp, cs1)
	say(dst, itoa(p.Gems), W-112, 33, 24, colWhite, 0)
	g.hits = append(g.hits, hit{rect{W - 156, 12, 146, 42}, "tab:" + itoa(tabShop)})
	ar := areaOf(p.Stage)
	say(dst, "Stage "+itoa(p.Stage), W/2+4, 30, 28, colWhite, 0.5)
	sayB(dst, ar.name, W/2+4, 56, 15, rgb(0xff, 0xf0, 0xb0), 0.5)
	if g.tab != tabKitchen {
		g.drawMonsterBar(dst)
	}
	g.drawTable(dst)
	if p.Farming && !g.bossOn && g.tab != tabKitchen {
		r := rect{W - 164, 64, 154, 50}
		k := 1 + 0.05*math.Sin(g.t*6)
		g.btn(dst, rect{r.cx() - r.w*k/2, r.cy() - r.h*k/2, r.w * k, r.h * k}, "boss", colOrange, "Fight Boss!", 20, false)
	}
	// active skill effects, right side
	y := 120.0
	chip := func(ic int, label string, tleft float64, col color.RGBA) {
		panel(dst, rect{W - 134, y, 124, 34}, 3)
		icon(ic, 26).draw(dst, W-116, y+17, 1, cs1)
		sayB(dst, label, W-98, y+17, 15, col, 0)
		sayB(dst, fmt.Sprintf("%.0fs", math.Ceil(tleft)), W-18, y+17, 15, colWhite, 1)
		y += 38
	}
	if g.frenzyT > 0 {
		chip(icTap, "Frenzy", g.frenzyT, rgb(0xb0, 0xff, 0x60))
	}
	if g.critT > 0 {
		chip(icStar, "Crits", g.critT, rgb(0xff, 0xe0, 0x60))
	}
	if g.goldT > 0 {
		chip(icCoin, "Coins x3", g.goldT, rgb(0xff, 0xd0, 0x40))
	}
	if g.bannerT > 0 {
		k := 2.2 - g.bannerT
		in := elastic(math.Min(1, k*1.6))
		a := clamp01(g.bannerT * 2.5)
		cy := g.fieldH() * 0.44
		gradRect(dst, 0, cy-44, W, 44, alpha(colInk, 0), alpha(colInk, 0.55*a))
		gradRect(dst, 0, cy, W, 44, alpha(colInk, 0.55*a), alpha(colInk, 0))
		sayOut(dst, fTitle, g.banner, W/2, cy-6, 40*in, alpha(rgb(0xff, 0xe0, 0x50), a), colInk, 0.5)
		if g.bannerSub != "" {
			sayB(dst, g.bannerSub, W/2, cy+26, 19, alpha(colWhite, a), 0.5)
		}
	}
}

// drawMonsterBar shows the stage progress, the monster's name and health, and the boss timer.
func (g *Game) drawMonsterBar(dst *ebiten.Image) {
	p := g.p
	if !g.bossOn {
		n := wavesPerStage
		w := 18.0
		x0 := W/2 - float64(n)*w/2 + w/2
		for i := 0; i < n; i++ {
			c := rgb(0x3a, 0x22, 0x6e)
			if i < p.Wave {
				c = rgb(0xff, 0xd0, 0x3a)
			}
			glowArt().draw(dst, x0+float64(i)*w, 80, 0.07, tintCS(colInk, 0.7))
			dropArt().draw(dst, x0+float64(i)*w, 80, 0.6, tintCS(c, 1))
		}
		icon(icCrown, 24).draw(dst, x0+float64(n)*w+4, 79, 1, cs1)
	}
	m := g.mon
	if m == nil || m.dead > 0 && m.gone {
		return
	}
	x, y := W/2-130, 98.0
	if g.bossOn {
		y = 72
	}
	bw := 260.0
	sayB(dst, monOf(m.kind, m.boss).name, W/2, y+8, 17, rgb(0xff, 0xe0, 0xe8), 0.5)
	slice3(dst, barArt(16, colRed, true), x, y+20, bw, cs1)
	if m.trail > m.hp {
		slice3(dst, barArt(16, rgb(0xff, 0xf0, 0xc0), false), x, y+20, math.Max(16, bw*m.trail/m.max), cs1)
	}
	if m.hp > 0 {
		slice3(dst, barArt(16, colRed, false), x, y+20, math.Max(16, bw*m.hp/m.max), cs1)
	}
	sayB(dst, num(m.hp), W/2, y+28, 13, colWhite, 0.5)
	if g.bossOn {
		tf := clamp01(g.bossT / (bossTime + p.effects()[efBossTime]))
		col := colBlue
		pulse := 1.0
		if g.bossT < 8 {
			col = colOrange
			pulse = 1 + 0.12*math.Abs(math.Sin(g.t*8))
		}
		bar(dst, x+28, y+42, bw-28, 12, tf, col)
		icon(icClock, 24).draw(dst, x+10, y+48, pulse, cs1)
	}
}

// drawTable shows the dishes on the table, left side.
func (g *Game) drawTable(dst *ebiten.Image) {
	p := g.p
	if len(p.Table) == 0 {
		return
	}
	y := 64.0
	for _, d := range p.Table {
		panel(dst, rect{10, y, 64, 64}, 3)
		foodIcon(fiPlate, 52).draw(dst, 42, y+38, 1, cs1)
		foodIcon(fiDish+d.R, 38).draw(dst, 42, y+30+math.Sin(g.t*3+float64(d.R))*1.5, 1, cs1)
		bar(dst, 16, y+52, 52, 8, d.Left/dishTime, colGreen)
		y += 70
	}
	if p.fullCourse() {
		sayB(dst, "Full course x1.5", 12, y+10, 13, rgb(0xff, 0xe0, 0x60), 0)
		y += 20
	}
	for _, i := range p.activeCombos() {
		sayB(dst, combos[i].name, 12, y+10, 13, rgb(0xb0, 0xff, 0x80), 0)
		y += 20
	}
}

func (g *Game) btn(dst *ebiten.Image, r rect, id string, col color.RGBA, label string, size float64, off bool) {
	pressed := g.pressedID == id && g.ptr.down && r.has(g.ptr.x, g.ptr.y)
	button(dst, r, col, label, size, pressed, off)
	g.hits = append(g.hits, hit{r, id})
}

// ---- crew bar ----

func (g *Game) drawSkillBar(dst *ebiten.Image) {
	y0 := g.skillY()
	gradRect(dst, 0, y0, W, 96, rgb(0x7a, 0x52, 0xd0), rgb(0x4a, 0x2c, 0x96))
	fillRect(dst, 0, y0, W, 3, rgb(0xb8, 0x9a, 0xff))
	fillRect(dst, 0, y0+93, W, 3, rgb(0x24, 0x12, 0x48))
	p := g.p
	for i := 0; i < 5; i++ {
		cx := slotX(i)
		cy := y0 + 46
		if i >= p.slots() {
			discArt(rgb(0x3a, 0x24, 0x6e)).draw(dst, cx, cy, 1, cs1)
			icon(icLock, 26).draw(dst, cx, cy-6, 1, csA(0.8))
			if s := slotStageFor(i); s > 0 {
				sayB(dst, "Stage "+itoa(s), cx, cy+20, 13, rgb(0xd0, 0xc0, 0xff), 0.5)
			}
			continue
		}
		if i >= len(p.Team) {
			discArt(rgb(0x5a, 0x3c, 0x9c)).draw(dst, cx, cy, 1, cs1)
			icon(icPlus, 26).draw(dst, cx, cy, 1+0.06*math.Sin(g.t*4), cs1)
			g.hits = append(g.hits, hit{rect{cx - 40, cy - 40, 80, 80}, "tab:" + itoa(tabCrew)})
			continue
		}
		b := g.team[i]
		id := p.Team[i]
		ready := b.charge >= 1
		if ready {
			pulse := 0.5 + 0.5*math.Sin(g.t*6)
			glowArt().draw(dst, cx, cy, 0.95+0.1*pulse, glowCS(rgb(0xff, 0xe0, 0x60), 0.6+0.3*pulse))
		}
		discArt(rarityCol[chefs[id].rarity]).draw(dst, cx, cy, 1, cs1)
		k := 1.0
		dy := 0.0
		if b.atk >= 0 { // the throw: a quick hop
			dy = -math.Sin(b.atk*math.Pi) * 8
			k = 1 + 0.06*math.Sin(b.atk*math.Pi)
		}
		if b.cast > 0 {
			dy -= math.Sin(b.cast*math.Pi) * 14
			k += 0.12 * math.Sin(b.cast*math.Pi)
		}
		face := 0
		if b.blink > 0 {
			face = 1
		}
		if ready || b.atk >= 0 && b.atk < 0.6 || b.cast > 0 {
			face = 2
		}
		portrait(dst, id, cx, cy-2+dy, 70*k, face, g.t, false)
		bar(dst, cx-36, cy+26, 72, 14, b.charge, rgb(0xff, 0xc8, 0x30))
		if p.favServed(id) {
			foodIcon(fiHeart, 22).draw(dst, cx+30, cy-30, 1+0.1*math.Sin(g.t*5), cs1)
		}
		if ready {
			bob := math.Abs(math.Sin(g.t * 5))
			say(dst, "TAP!", cx, cy-40-bob*5, 18, rgb(0xff, 0xf0, 0x60), 0.5)
		}
		g.hits = append(g.hits, hit{rect{cx - 46, cy - 46, 92, 92}, "skill:" + itoa(i)})
	}
}

func slotStageFor(i int) int {
	if i-1 >= 0 && i-1 < len(slotStages) {
		return slotStages[i-1]
	}
	return 0
}

func discArt(col color.RGBA) *pimg {
	return paintArt("disc"+itoa(int(col.R)<<16|int(col.G)<<8|int(col.B)), -42, -42, 42, 42, 1, func(c *canvas) {
		c.fill(circ(0, 0, 40), rgb(0x1a, 0x0e, 0x20))
		c.vol(circ(0, 0, 36.5), mat{base: col, flat: 0.5, depth: 10, gloss: 0.3})
		c.vol(circ(0, 0, 30.5), mat{base: mix(rgb(0xf0, 0xe8, 0xff), col, 0.25), dark: mix(rgb(0xc8, 0xb8, 0xf0), col, 0.25), flat: 0.6, depth: 31})
	})
}

// ---- the kitchen ----

func (g *Game) drawKitchen(dst *ebiten.Image) {
	fh := g.fieldH()
	f := clip(dst, rect{0, 0, W, fh})
	paintArt("kitchen@"+num(fh), 0, 0, W, fh, 1, func(c *canvas) { paintKitchen(c, W, fh) }).draw(f, 0, 0, 1, cs1)
	p := g.p
	for st := 0; st < numStations; st++ {
		x, y := g.stationPos(st)
		lv := p.Stations[st]
		cook := p.Cook[st]
		sel := g.station == st
		if sel {
			glowArt().drawT(f, x, y-80, 1.0, 1.6, 0, glowCS(rgb(0xff, 0xf0, 0x90), 0.55+0.15*math.Sin(g.t*4)))
		}
		sx, sy := 1.0, 1.0
		busy := cook.R >= 0 && cook.Left > 0
		if busy {
			w := math.Sin(g.t*14) * 0.012
			sx, sy = 1+w, 1-w
		}
		if pt := g.pressT["station:"+itoa(st)]; pt > 0 {
			q := math.Sin(pt * math.Pi)
			sx, sy = sx*(1+0.06*q), sy*(1-0.06*q)
		}
		cs := cs1
		if lv == 0 {
			cs = ebiten.ColorScale{}
			cs.Scale(0.45, 0.4, 0.55, 1)
		}
		shadowAt(f, x, y, 120, 0.8)
		stationArt(st).drawT(f, x, y, sx, sy, 0, cs)
		if st == stOven && busy {
			glowArt().drawT(f, x, y-80, 0.5, 0.35, 0, glowCS(rgb(0xff, 0x8a, 0x30), 0.6+0.2*math.Sin(g.t*6)))
		}
		top := y - []float64{124, 160, 156, 140, 196}[st]
		switch {
		case lv == 0:
			icon(icLock, 30).draw(f, x, top-60, 1, cs1)
			panel(f, rect{x - 48, top - 36, 96, 28}, 3)
			icon(icCoin, 18).draw(f, x-30, top-22, 1, cs1)
			sayB(f, num(p.stationPrice(st)), x-18, top-22, 15, colWhite, 0)
		case busy:
			bar(f, x-40, top-22, 80, 14, 1-cook.Left/cook.Total, colGreen)
			foodIcon(fiDish+cook.R, 30).draw(f, x, top-44, 1, cs1)
			sayB(f, fmt.Sprintf("%ds", int(math.Ceil(cook.Left))), x, top-62, 14, colWhite, 0.5)
			if g.frame%9 == st {
				g.parts = append(g.parts, particle{x: x + (g.rng.Float64()-0.5)*30, y: top + 10, vy: -0.8, life: 1.2, size: 0.6, kind: ptSteam})
			}
		case cook.R >= 0:
			b := math.Abs(math.Sin(g.t * 5))
			glowArt().draw(f, x, top-40-b*8, 0.5, glowCS(rgb(0xff, 0xf0, 0x80), 0.7))
			foodIcon(fiDish+cook.R, 44).draw(f, x, top-40-b*8, 1, cs1)
			say(f, "Ready!", x, top-74-b*8, 16, rgb(0xb0, 0xff, 0x60), 0.5)
		}
		sayB(f, stationName[st], x, y+14, 14, colWhite, 0.5)
		if lv > 1 {
			sayB(f, "Lv "+itoa(lv), x+40, y-12, 12, rgb(0xff, 0xe0, 0x60), 1)
		}
		g.hits = append(g.hits, hit{rect{x - 52, y - 200, 104, 220}, "station:" + itoa(st)})
	}
	g.drawParticles(f, true)
	gradRect(f, 0, fh-20, W, 20, alpha(colInk, 0), alpha(colInk, 0.35))
}

// drawIngredientBar shows the ingredients you have, under the kitchen.
func (g *Game) drawIngredientBar(dst *ebiten.Image) {
	y0 := g.skillY()
	gradRect(dst, 0, y0, W, 96, rgb(0xb0, 0x70, 0x40), rgb(0x8a, 0x50, 0x2c))
	fillRect(dst, 0, y0, W, 3, rgb(0xe0, 0xa8, 0x70))
	fillRect(dst, 0, y0+93, W, 3, rgb(0x4a, 0x24, 0x10))
	w := W / numIngredients
	for i := 0; i < numIngredients; i++ {
		cx := w*float64(i) + w/2
		k := 1 + 0.12*g.ingPulse
		foodIcon(i, 40).draw(dst, cx, y0+38, k, cs1)
		say(dst, itoa(g.p.Ing[i]), cx, y0+74, 18, colWhite, 0.5)
	}
}
