package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Drawing the battlefield: backdrop, bugs, slugs, shots, particles and the HUD.

func (g *Game) Draw(dst *ebiten.Image) {
	var t0 time.Time
	if timing {
		t0 = time.Now()
	}
	g.hits = g.hits[:0]
	dst.Fill(colPanelD)
	g.drawField(dst)
	g.drawHUD(dst)
	g.drawSkillBar(dst)
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

// ---- creatures ----

// drawLayer draws a painted layer with an extra transform in unit space (pre)
// and the body transform (in layout units).
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

// drawBug draws bug id with its feet at x,y. face: 0 open, 1 blink, 2 attack.
func drawBug(dst *ebiten.Image, id int, x, y, sx, sy, rot float64, face int, t float64, cs ebiten.ColorScale) {
	z := 1.0
	if sx > 1.2 {
		z = 2.2
	}
	art := bugArt(id, z)
	m := bodyM(x, y, sx, sy, rot, 0)
	sp := bugs[id]
	var flap ebiten.GeoM
	if art.back != nil {
		f := 0.45 + 0.55*math.Abs(math.Sin(t*26))
		if id == 4 {
			f = 0.8 + 0.2*math.Sin(t*6)
		}
		flap.Translate(-sp.wingX, -sp.wingY)
		flap.Scale(1, f)
		flap.Translate(sp.wingX, sp.wingY)
	}
	drawLayer(dst, art.back, flap, m, cs, ebiten.BlendSourceOver)
	var id0 ebiten.GeoM
	drawLayer(dst, art.body, id0, m, cs, ebiten.BlendSourceOver)
	drawLayer(dst, art.face[face], id0, m, cs, ebiten.BlendSourceOver)
	drawLayer(dst, art.front, flap, m, cs, ebiten.BlendSourceOver)
}

// shadowAt draws a soft contact shadow.
func shadowAt(dst *ebiten.Image, x, y, w, a float64) {
	var cs ebiten.ColorScale
	cs.Scale(0.12*float32(a), 0.04*float32(a), 0.2*float32(a), 0.45*float32(a))
	glowArt().drawT(dst, x, y, w/100, w/100*0.22, 0, cs)
}

func (g *Game) drawTeam(dst *ebiten.Image) {
	order := []int{2, 4, 0, 1, 3} // back row first
	for _, i := range order {
		if i >= len(g.team) {
			continue
		}
		id := g.p.Team[i]
		b := g.team[i]
		x, y, sc := g.bugPos(i)
		sp := bugs[id]
		ph := g.t*3 + float64(id)*1.3
		sx, sy := 1-0.02*math.Sin(ph), 1+0.03*math.Sin(ph)
		rot := 0.0
		dy := 0.0
		face := 0
		if b.blink > 0 {
			face = 1
		}
		groundY := y
		if sp.fly {
			dy += math.Sin(g.t*3.4+float64(id)) * 5
			groundY = y + 34*sc
		}
		if b.atk >= 0 {
			a := b.atk
			face = 2
			switch {
			case a < 0.3:
				k := easeOut(a / 0.3)
				rot -= 0.1 * k
				sx, sy = sx*(1+0.08*k), sy*(1-0.08*k)
				x -= 6 * k
			case a < 0.55:
				k := easeOut((a - 0.3) / 0.25)
				rot += -0.1 + 0.18*k
				sx, sy = sx*(1.08-0.16*k), sy*(0.92+0.18*k)
				x += -6 + 22*k
			default:
				k := easeOut((a - 0.55) / 0.45)
				rot = 0.08 * (1 - k)
				sx, sy = sx*(0.92+0.08*k), sy*(1.1-0.1*k)
				x += 16 * (1 - k)
			}
		}
		if b.hop > 0 {
			h := math.Sin((1 - b.hop) * math.Pi)
			dy -= h * 16
			sy *= 1 + 0.06*h
			sx *= 1 - 0.04*h
		}
		if b.cast > 0 {
			h := math.Sin(b.cast * math.Pi)
			dy -= h * 50
			rot += math.Sin(b.cast*math.Pi*2) * 0.25
			face = 2
			glowArt().draw(dst, x, y+dy-50*sc, 1.2*h, glowCS(bugs[id].shot, 0.7*h))
		}
		pop := backOut(b.pop)
		shadowAt(dst, x, groundY, 110*sc*(1-math.Min(0.4, -dy/120)), 1)
		if g.frenzyT > 0 {
			glowArt().draw(dst, x, y+dy-45*sc, 0.9*sc, glowCS(rgb(0x9c, 0xf0, 0x40), 0.35+0.15*math.Sin(g.t*12)))
		}
		drawBug(dst, id, x, y+dy, sx*sc*pop, sy*sc*pop, rot, face, g.t+float64(id), cs1)
	}
}

func glowCS(c color.RGBA, a float64) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.Scale(float32(c.R)/255*float32(a), float32(c.G)/255*float32(a), float32(c.B)/255*float32(a), float32(a))
	return cs
}

func (g *Game) slugScale(s *slug) float64 {
	if s.boss {
		return 1.45
	}
	return 1.0
}

func (g *Game) drawSlugs(dst *ebiten.Image) {
	gy := g.groundY() + 8
	for i := len(g.slugs) - 1; i >= 0; i-- {
		s := g.slugs[i]
		sc := g.slugScale(s)
		z := 1.0
		if s.boss {
			z = 1.4
		}
		art := slugArt(s.kind, s.boss, z)
		k := math.Sin(s.phase)
		sx, sy := 1+0.06*k, 1-0.05*k
		skew := math.Sin(s.phase*0.5) * 0.05
		if s.squish > 0.01 {
			w := math.Sin(g.t*45) * s.squish
			sx += 0.1 * w
			sy -= 0.1 * w
		}
		pop := backOut(s.spawn)
		a := 1.0
		x := s.x
		if s.dead > 0 {
			d := easeOut(s.dead)
			if s.gone { // the boss crawls away
				x += d * 300
				a = 1 - d
			} else {
				sy *= 1 - d*0.75
				sx *= 1 + d*0.5
				a = 1 - easeIn(s.dead)
			}
		}
		shadowAt(dst, x, gy, 130*sc*sx, a)
		m := bodyM(x, gy, sx*sc*pop, sy*sc*pop, 0, skew)
		var id ebiten.GeoM
		drawLayer(dst, art.body, id, m, csA(a), ebiten.BlendSourceOver)
		face := 0
		if s.hurt > 0 || s.dead > 0 && !s.gone {
			face = 1
		}
		// the eye stalks sway a little on their own
		var sway ebiten.GeoM
		sway.Translate(0, 0)
		sway.Skew(math.Sin(g.t*2.2+float64(i))*0.06, 0)
		sway.Translate(math.Sin(g.t*2.2+float64(i))*0.06*96, 0)
		drawLayer(dst, art.face[face], sway, m, csA(a), ebiten.BlendSourceOver)
		if s.hit > 0 {
			var cs ebiten.ColorScale
			f := float32(s.hit * 0.6 * a)
			cs.Scale(f, f, f, f)
			drawLayer(dst, art.body, id, m, cs, ebiten.BlendLighter)
		}
		// small health bar over hurt regular slugs
		if !s.boss && s.dead == 0 && s.hp < s.max {
			bw := 70.0
			by := gy - 152*sc
			bar(dst, x-bw/2, by, bw, 12, s.hp/s.max, colRed)
		}
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
	bg.draw(f, ox, oy, 1, cs1)
	// clouds drift
	cl := paintArt("cloud"+fmt.Sprint(ar.night), -70, -40, 70, 12, 1, func(c *canvas) { paintCloud(c, ar.night) })
	for i := 0; i < 3; i++ {
		sp := 6 + float64(i)*3
		x := math.Mod(g.t*sp+float64(i)*230, W+200) - 100
		y := 120 + float64(i%2)*50 + float64(i)*8
		cl.draw(f, x+ox, y+oy, 0.8+float64(i%2)*0.3, csA(0.92))
	}
	g.drawSlugs(f)
	g.drawTeam(f)
	// shots
	for _, s := range g.shots {
		x, y := s.pos()
		glowArt().draw(f, x, y, 0.32*s.big, glowCS(s.col, 0.9))
		sparkArt().drawT(f, x, y, 0.5*s.big, 0.5*s.big, g.t*8, csA(1))
		glowArt().draw(f, x, y, 0.1*s.big, glowCS(colWhite, 1))
	}
	g.drawParticles(f)
	for _, fl := range g.fl {
		k := fl.t / fl.life
		y := fl.y - easeOut(k)*50
		a := 1 - easeIn(math.Max(0, (k-0.6)/0.4))
		sc := 0.6 + 0.4*backOut(math.Min(1, k*5))
		say(f, fl.s, fl.x, y, fl.size*sc, alpha(fl.col, a), 0.5)
	}
	// foreground grass sways in front of everything
	tuft := paintArt("tuft"+itoa(ar.kind), -40, -100, 40, 8, 1, func(c *canvas) { paintTuft(c, shade(ar.ground, 0.9), int64(ar.kind)) })
	for i, x := range []float64{-6, 548, 300} {
		sw := math.Sin(g.t*1.6+float64(i)) * 0.06
		sc := 1.0
		if i == 2 {
			sc = 0.55
		}
		tuft.drawSkew(f, x, fh+4, sc, sc, sw, cs1, ebiten.BlendSourceOver)
	}
	if g.flash > 0 {
		fillRect(f, 0, 0, W, fh, alpha(colWhite, g.flash*0.6))
	}
	// a soft shade where the field meets the bar
	gradRect(f, 0, fh-30, W, 30, alpha(colInk, 0), alpha(colInk, 0.35))
}

func (g *Game) drawParticles(dst *ebiten.Image) {
	for _, q := range g.parts {
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
		case ptGoo:
			dropArt().draw(dst, q.x, q.y, q.size*(1-k*0.5), tintCS(q.col, a))
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

func tintCS(c color.RGBA, a float64) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.Scale(float32(c.R)/255*float32(a), float32(c.G)/255*float32(a), float32(c.B)/255*float32(a), float32(a))
	return cs
}

func puffArt() *pimg {
	return paintArt("puff", -24, -24, 24, 24, 1, func(c *canvas) {
		s := circ(0, 0, 14).melt(circ(-9, 4, 10), 4).melt(circ(9, 3, 11), 4).melt(circ(2, -9, 10), 4)
		c.vol(s, mat{base: colWhite, dark: rgb(0xd0, 0xc8, 0xf0), flat: 0.3, rim: 0.4})
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
	// coins
	cp := 1 + 0.15*g.coinPulse
	panel(dst, rect{10, 12, 176, 42}, 3)
	icon(icCoin, 34).draw(dst, 30, 33, cp, cs1)
	say(dst, num(g.shownCoin), 54, 33, 24, colWhite, 0)
	// gems
	gp := 1 + 0.15*g.gemPulse
	panel(dst, rect{W - 156, 12, 146, 42}, 3)
	icon(icGem, 32).draw(dst, W-136, 33, gp, cs1)
	say(dst, itoa(p.Gems), W-112, 33, 24, colWhite, 0)
	g.hits = append(g.hits, hit{rect{W - 156, 12, 146, 42}, "tab:2"})
	// stage
	ar := areaOf(p.Stage)
	say(dst, "Stage "+itoa(p.Stage), W/2+4, 30, 28, colWhite, 0.5)
	sayB(dst, ar.name, W/2+4, 56, 15, rgb(0xff, 0xf0, 0xb0), 0.5)
	if g.bossOn {
		b := g.front()
		x := W/2 - 150
		y := 76.0
		if b != nil && b.boss {
			panel(dst, rect{x - 10, y - 4, 320, 66}, 3)
			sayB(dst, slugName(b.kind, true), x, y+12, 17, rgb(0xff, 0xd0, 0xe0), 0)
			sayB(dst, num(b.hp), x+300, y+12, 17, colWhite, 1)
			slice3(dst, barArt(18, colRed, true), x, y+24, 300, cs1)
			if b.trail > b.hp {
				slice3(dst, barArt(18, rgb(0xff, 0xf0, 0xc0), false), x, y+24, math.Max(18, 300*b.trail/b.max), cs1)
			}
			if b.hp > 0 {
				slice3(dst, barArt(18, colRed, false), x, y+24, math.Max(18, 300*b.hp/b.max), cs1)
			}
			// timer
			tf := clamp01(g.bossT / bossTime)
			col := colBlue
			if g.bossT < 8 {
				col = colOrange
			}
			bar(dst, x+30, y+46, 270, 12, tf, col)
			pulse := 1.0
			if g.bossT < 8 {
				pulse = 1 + 0.12*math.Abs(math.Sin(g.t*8))
			}
			icon(icClock, 26).draw(dst, x+12, y+52, pulse, cs1)
		}
	} else {
		// progress pips to the boss
		n := wavesPerStage
		w := 18.0
		x0 := W/2 - float64(n)*w/2 + w/2
		for i := 0; i < n; i++ {
			on := i < p.Wave
			c := rgb(0x3a, 0x22, 0x6e)
			if on {
				c = rgb(0xff, 0xd0, 0x3a)
			}
			glowArt().draw(dst, x0+float64(i)*w, 80, 0.07, tintCS(colInk, 0.7))
			dropArt().draw(dst, x0+float64(i)*w, 80, 0.6, tintCS(c, 1))
		}
		icon(icCrown, 24).draw(dst, x0+float64(n)*w+4, 79, 1, cs1)
	}
	if p.Farming && !g.bossOn {
		r := rect{W - 164, 64, 154, 50}
		k := 1 + 0.05*math.Sin(g.t*6)
		rr := rect{r.cx() - r.w*k/2, r.cy() - r.h*k/2, r.w * k, r.h * k}
		g.btn(dst, rr, "boss", colOrange, "Fight Boss!", 20, false)
	}
	// active skill effects
	y := 64.0
	chip := func(ic int, label string, tleft float64, col color.RGBA) {
		panel(dst, rect{10, y, 124, 34}, 3)
		icon(ic, 26).draw(dst, 28, y+17, 1, cs1)
		sayB(dst, label, 46, y+17, 15, col, 0)
		sayB(dst, fmt.Sprintf("%.0fs", math.Ceil(tleft)), 126, y+17, 15, colWhite, 1)
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
	// banner
	if g.bannerT > 0 {
		k := 2.2 - g.bannerT
		in := elastic(math.Min(1, k*1.6))
		a := clamp01(g.bannerT * 2.5)
		cy := 190.0
		gradRect(dst, 0, cy-44, W, 44, alpha(colInk, 0), alpha(colInk, 0.55*a))
		gradRect(dst, 0, cy, W, 44, alpha(colInk, 0.55*a), alpha(colInk, 0))
		sayOut(dst, fTitle, g.banner, W/2, cy-6, 46*in, alpha(rgb(0xff, 0xe0, 0x50), a), colInk, 0.5)
		if g.bannerSub != "" {
			sayB(dst, g.bannerSub, W/2, cy+28, 20, alpha(colWhite, a), 0.5)
		}
	}
}

// btn draws a button and registers its hit box.
func (g *Game) btn(dst *ebiten.Image, r rect, id string, col color.RGBA, label string, size float64, off bool) {
	pressed := g.pressedID == id && g.ptr.down && r.has(g.ptr.x, g.ptr.y)
	button(dst, r, col, label, size, pressed, off)
	g.hits = append(g.hits, hit{r, id})
}

// ---- skill bar ----

func (g *Game) drawSkillBar(dst *ebiten.Image) {
	y0 := g.skillY()
	gradRect(dst, 0, y0, W, 96, rgb(0x7a, 0x52, 0xd0), rgb(0x4a, 0x2c, 0x96))
	fillRect(dst, 0, y0, W, 3, rgb(0xb8, 0x9a, 0xff))
	fillRect(dst, 0, y0+93, W, 3, rgb(0x24, 0x12, 0x48))
	p := g.p
	for i := 0; i < 5; i++ {
		cx := 62 + float64(i)*104
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
			g.hits = append(g.hits, hit{rect{cx - 40, cy - 40, 80, 80}, "tab:1"})
			continue
		}
		b := g.team[i]
		id := p.Team[i]
		ready := b.charge >= 1
		if ready {
			pulse := 0.5 + 0.5*math.Sin(g.t*6)
			glowArt().draw(dst, cx, cy, 0.95+0.1*pulse, glowCS(rgb(0xff, 0xe0, 0x60), 0.6+0.3*pulse))
		}
		discArt(rarityCol[bugs[id].rarity]).draw(dst, cx, cy, 1, cs1)
		k := 1.0
		if pt := g.pressT["skill:"+itoa(i)]; pt > 0 {
			k = 1 + 0.15*math.Sin(pt*math.Pi)
		}
		face := 0
		if ready {
			face = 2
		}
		portrait(dst, id, cx, cy-4, 76*k, face, g.t, false)
		bar(dst, cx-36, cy+26, 72, 14, b.charge, rgb(0xff, 0xc8, 0x30))
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

// discArt is the round frame behind a portrait.
func discArt(col color.RGBA) *pimg {
	return paintArt("disc"+itoa(int(col.R)<<16|int(col.G)<<8|int(col.B)), -42, -42, 42, 42, 1, func(c *canvas) {
		c.fill(circ(0, 0, 40), rgb(0x24, 0x12, 0x48))
		c.vol(circ(0, 0, 37), mat{base: col, flat: 0.5, depth: 10, gloss: 0.3, rim: 0.3})
		c.vol(circ(0, 0, 31), mat{base: mix(rgb(0x3a, 0x22, 0x70), col, 0.25), dark: rgb(0x2a, 0x16, 0x50), flat: 0.6, depth: 31})
	})
}

// portrait draws a bug fitted into a box size units across, centred at cx,cy.
func portrait(dst *ebiten.Image, id int, cx, cy, size float64, face int, t float64, dark bool) {
	sc := size / 150
	cs := cs1
	if dark {
		cs = ebiten.ColorScale{}
		cs.Scale(0.16, 0.1, 0.3, 1)
	}
	drawBug(dst, id, cx-8*sc, cy+62*sc, sc, sc*(1+0.02*math.Sin(t*3)), 0, face, t, cs)
}
