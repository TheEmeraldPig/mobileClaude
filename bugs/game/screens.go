package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The menus under the battlefield: Brigade (level up), Bugs (collection and team)
// and Shop, plus the hatching and welcome-back overlays.

func (g *Game) drawPanel(dst *ebiten.Image) {
	y0 := g.panelY()
	h := g.tabsY() - y0
	gradRect(dst, 0, y0, W, h, rgb(0x46, 0x2a, 0x88), rgb(0x32, 0x1c, 0x66))
	// a faint dotted pattern
	for j := 0; j < int(h/36)+1; j++ {
		for i := 0; i < 16; i++ {
			x := float64(i)*36 + float64(j%2)*18
			fillRect(dst, x, y0+float64(j)*36+10, 3, 3, alpha(colWhite, 0.05))
		}
	}
	area := rect{0, y0, W, h}
	c := clip(dst, area)
	in := easeOut(g.tabT * 4)
	off := (1 - in) * 30
	sy := y0 + 10 - g.scroll[g.tab] + off
	var end float64
	switch g.tab {
	case 0:
		end = g.drawBrigade(c, sy)
	case 1:
		end = g.drawCollection(c, sy)
	case 2:
		end = g.drawShop(c, sy)
	}
	g.scrollMax[g.tab] = math.Max(0, end+g.scroll[g.tab]-off-(y0+h)+14)
	// keep hit boxes inside the visible panel
	var keep []hit
	for _, ht := range g.hits {
		if ht.r.y+ht.r.h > y0 && ht.r.y < y0+h || ht.r.y+ht.r.h <= g.panelY() {
			keep = append(keep, ht)
		}
	}
	g.hits = keep
	gradRect(dst, 0, y0, W, 12, alpha(colInk, 0.5), alpha(colInk, 0))
}

// ---- Brigade ----

func (g *Game) drawBrigade(dst *ebiten.Image, y float64) float64 {
	p := g.p
	sayB(dst, "Brigade damage", 18, y+18, 17, rgb(0xd8, 0xc8, 0xff), 0)
	icon(icSword, 26).draw(dst, 30, y+46, 1, cs1)
	say(dst, num(p.teamDPS())+" /s", 48, y+46, 24, colWhite, 0)
	ml := []string{"Buy x1", "Buy x10", "Buy MAX"}[g.mult]
	g.btn(dst, rect{W - 160, y + 8, 146, 50}, "mult", colBlue, ml, 20, false)
	y += 70
	row := func(id int, name, sub, sub2 string, lvl int, cost float64, n int, drawPic func(cx, cy float64)) {
		r := rect{12, y, W - 24, 96}
		panel(dst, r, 0)
		drawPic(r.x+52, r.y+48)
		ink(dst, name, r.x+104, r.y+24, 21, colInk, 0)
		ink(dst, "Lv "+itoa(lvl), r.x+104+textW(fBody, name, 21)+10, r.y+24, 17, rgb(0x8a, 0x5a, 0xd0), 0)
		ink(dst, sub, r.x+104, r.y+50, 16, rgb(0x5a, 0x40, 0x8a), 0)
		ink(dst, sub2, r.x+104, r.y+73, 14, rgb(0x9a, 0x80, 0xc0), 0)
		br := rect{r.x + r.w - 150, r.y + 16, 138, 64}
		can := cost <= p.Coins
		key := "up:" + itoa(id)
		g.btn(dst, br, key, colGreen, "", 0, !can)
		dy := 0.0
		if g.pressedID == key && g.ptr.down && br.has(g.ptr.x, g.ptr.y) {
			dy = 6
		}
		sayB(dst, "+"+itoa(n)+" Lv", br.cx(), br.y+18+dy, 16, colWhite, 0.5)
		icon(icCoin, 22).draw(dst, br.x+30, br.y+40+dy, 1, cs1)
		say(dst, num(cost), br.x+44, br.y+40+dy, 20, colWhite, 0)
		y += 104
	}
	// tap power
	n := g.buyCount(-1)
	row(-1, "Tap Power", num(p.tapDamage())+" per tap", "Also adds 8% of brigade damage", p.TapLv+1, p.upCost(-1, n), n, func(cx, cy float64) {
		discArt(colOrange).draw(dst, cx, cy, 0.95, cs1)
		icon(icTap, 46).draw(dst, cx, cy, 1+0.05*math.Sin(g.t*5), cs1)
	})
	list := append([]int{}, p.Team...)
	for id := range bugs {
		if p.Bugs[id].Have && !p.inTeam(id) {
			list = append(list, id)
		}
	}
	for _, id := range list {
		b := p.Bugs[id]
		n := g.buyCount(id)
		sub := num(p.dps(id)) + " damage /s"
		if !p.inTeam(id) {
			sub += "  (resting)"
		}
		nm := nextMilestone(b.Level)
		id := id
		row(id, bugs[id].name, sub, "Double damage at Lv "+itoa(nm), b.Level, p.upCost(id, n), n, func(cx, cy float64) {
			discArt(rarityCol[bugs[id].rarity]).draw(dst, cx, cy, 0.95, cs1)
			portrait(dst, id, cx, cy-2, 70, 0, g.t, false)
			for s := 0; s < b.Stars; s++ {
				icon(icStar, 14).draw(dst, cx-24+float64(s)*12, cy+36, 1, cs1)
			}
		})
	}
	return y
}

// ---- Bugs (collection) ----

func (g *Game) drawCollection(dst *ebiten.Image, y float64) float64 {
	p := g.p
	have := 0
	for _, b := range p.Bugs {
		if b.Have {
			have++
		}
	}
	say(dst, "Your bugs  "+itoa(have)+"/"+itoa(len(bugs)), 18, y+20, 24, colWhite, 0)
	sayB(dst, fmt.Sprintf("Team %d/%d · tap a bug to add or rest it", len(p.Team), p.slots()), 18, y+48, 15, rgb(0xd8, 0xc8, 0xff), 0)
	y += 70
	cw, ch := (W-24-20)/3, 214.0
	for i := range bugs {
		col, rowi := i%3, i/3
		r := rect{12 + float64(col)*(cw+10), y + float64(rowi)*(ch+10), cw, ch}
		b := p.Bugs[i]
		sp := bugs[i]
		inT := p.inTeam(i)
		style := 0
		if inT {
			style = 2
		}
		k := 1.0
		if pt := g.pressT["team:"+itoa(i)]; pt > 0 {
			k = 1 - 0.05*math.Sin(pt*math.Pi)
		}
		rr := rect{r.cx() - r.w*k/2, r.cy() - r.h*k/2, r.w * k, r.h * k}
		panel(dst, rr, style)
		rc := rarityCol[sp.rarity]
		slice3(dst, barArt(22, rc, false), rr.x+10, rr.y+10, rr.w-20, cs1)
		sayB(dst, rarityName[sp.rarity], rr.cx(), rr.y+21, 13, colWhite, 0.5)
		face := 0
		if b.Have && math.Mod(g.t+float64(i)*0.7, 4) < 0.15 {
			face = 1
		}
		portrait(dst, i, rr.cx(), rr.y+100, 118*k, face, g.t+float64(i), !b.Have)
		if b.Have {
			ink(dst, sp.name, rr.cx(), rr.y+164, 19, colInk, 0.5)
			for s := 0; s < maxStars; s++ {
				ic := icStarOff
				if s < b.Stars {
					ic = icStar
				}
				icon(ic, 18).draw(dst, rr.cx()-34+float64(s)*17, rr.y+190, 1, cs1)
			}
			if inT {
				icon(icCheck, 30).draw(dst, rr.x+rr.w-20, rr.y+44, 1, cs1)
			}
		} else {
			ink(dst, "???", rr.cx(), rr.y+164, 19, rgb(0x8a, 0x70, 0xb0), 0.5)
			ink(dst, "Hatch from eggs", rr.cx(), rr.y+190, 13, rgb(0x9a, 0x80, 0xc0), 0.5)
		}
		g.hits = append(g.hits, hit{r, "team:" + itoa(i)})
	}
	rows := (len(bugs) + 2) / 3
	y += float64(rows) * (ch + 10)
	// skill guide for the team
	sayB(dst, "Skills: tap a bug's portrait above when it says TAP!", W/2, y+16, 15, rgb(0xd8, 0xc8, 0xff), 0.5)
	return y + 34
}

// ---- Shop ----

func (g *Game) drawShop(dst *ebiten.Image, y float64) float64 {
	p := g.p
	now := g.now()
	// free egg
	r := rect{12, y, W - 24, 120}
	panel(dst, r, 2)
	ready := now >= p.FreeEgg
	bob := 0.0
	if ready {
		bob = math.Sin(g.t*5) * 0.12
		glowArt().draw(dst, r.x+64, r.y+60, 0.9, glowCS(rgb(0xff, 0xe0, 0x80), 0.6))
	}
	icon(icEgg, 80).drawT(dst, r.x+64, r.y+62, 1, 1, bob, cs1)
	ink(dst, "Free Egg", r.x+124, r.y+36, 24, rgb(0x6a, 0x30, 0x10), 0)
	if ready {
		ink(dst, "Ready to hatch!", r.x+124, r.y+70, 17, rgb(0x9a, 0x5a, 0x20), 0)
		g.btn(dst, rect{r.x + r.w - 150, r.y + 30, 136, 62}, "free", colGreen, "Hatch!", 24, false)
	} else {
		left := p.FreeEgg - now
		ink(dst, "A new one every 4 hours", r.x+124, r.y+70, 17, rgb(0x9a, 0x5a, 0x20), 0)
		icon(icClock, 26).draw(dst, r.x+r.w-120, r.y+60, 1, cs1)
		say(dst, fmt.Sprintf("%d:%02d:%02d", left/3600, left/60%60, left%60), r.x+r.w-100, r.y+60, 22, colWhite, 0)
	}
	y += 132
	// egg pair
	cw := (W - 24 - 10) / 2
	for i := 0; i < 2; i++ {
		it := shopItems[i]
		r := rect{12 + float64(i)*(cw+10), y, cw, 250}
		panel(dst, r, 0)
		ic := icEgg
		if i == 1 {
			ic = icGoldEgg
			raysArt().drawT(dst, r.cx(), r.y+80, 0.35, 0.35, g.t*0.3, csA(0.5))
		}
		icon(ic, 96).drawT(dst, r.cx(), r.y+80, 1, 1, math.Sin(g.t*2+float64(i))*0.06, cs1)
		ink(dst, it.name, r.cx(), r.y+148, 21, colInk, 0.5)
		ink(dst, it.desc, r.cx(), r.y+174, 13, rgb(0x6a, 0x50, 0x9a), 0.5)
		g.gemButton(dst, rect{r.x + 18, r.y + 188, r.w - 36, 52}, "buy:"+itoa(i), it.gems)
	}
	y += 262
	// rows
	for i := 2; i < len(shopItems); i++ {
		it := shopItems[i]
		if it.id == "starter" && p.Starter {
			continue
		}
		style := 0
		if it.id == "starter" {
			style = 2
		}
		r := rect{12, y, W - 24, 100}
		panel(dst, r, style)
		ic := map[int]int{iconCoins: icSack, iconGems: icGem, iconChest: icChest, iconStarter: icGift}[it.icon]
		icon(ic, 62).draw(dst, r.x+52, r.y+50, 1+0.04*math.Sin(g.t*3+float64(i)), cs1)
		ink(dst, it.name, r.x+100, r.y+36, 21, colInk, 0)
		desc := it.desc
		if it.id == "coins" {
			desc = "+" + num(math.Max(100, p.coinRate()*3600)) + " coins right now"
		}
		ink(dst, desc, r.x+100, r.y+66, 14, rgb(0x6a, 0x50, 0x9a), 0)
		br := rect{r.x + r.w - 140, r.y + 22, 126, 56}
		if it.price != "" {
			g.btn(dst, br, "buy:"+itoa(i), colPink, it.price, 22, false)
		} else {
			g.gemButton(dst, br, "buy:"+itoa(i), it.gems)
		}
		y += 108
	}
	sayB(dst, "Real-money items are free test purchases for now.", W/2, y+12, 13, rgb(0xb8, 0xa8, 0xe0), 0.5)
	return y + 30
}

func (g *Game) gemButton(dst *ebiten.Image, r rect, id string, gems int) {
	g.btn(dst, r, id, colGreen, "", 0, g.p.Gems < gems)
	dy := -r.h * btnLip / 2
	if g.pressedID == id && g.ptr.down && r.has(g.ptr.x, g.ptr.y) {
		dy = r.h * btnLip * 0.3
	}
	w := textW(fTitle, itoa(gems), 22) + 30
	icon(icGem, 26).draw(dst, r.cx()-w/2+12, r.cy()+dy, 1, cs1)
	say(dst, itoa(gems), r.cx()-w/2+30, r.cy()+dy, 22, colWhite, 0)
}

// ---- tabs ----

func (g *Game) drawTabs(dst *ebiten.Image) {
	y := g.tabsY()
	gradRect(dst, 0, y, W, g.h-y, rgb(0x2e, 0x18, 0x5a), rgb(0x1e, 0x0e, 0x3e))
	fillRect(dst, 0, y, W, 3, rgb(0x6a, 0x4a, 0xb0))
	names := []string{"Brigade", "Bugs", "Shop"}
	cols := []ebiten.ColorScale{}
	_ = cols
	tw := W / 3
	for i, n := range names {
		sel := g.tab == i
		r := rect{float64(i)*tw + 8, y + 10, tw - 16, 68}
		col := colPanel
		if sel {
			col = rgb(0xff, 0xb4, 0x2a)
			r.y -= 8 * easeOut(g.tabT*5)
		}
		button(dst, r, col, "", 0, false, false)
		cx := r.cx()
		switch i {
		case 0:
			icon(icSword, 34).draw(dst, cx-textW(fTitle, n, 20)/2-16, r.y+30, 1, cs1)
		case 1:
			portrait(dst, 0, cx-textW(fTitle, n, 20)/2-18, r.y+30, 44, 0, g.t, false)
		case 2:
			icon(icGem, 32).draw(dst, cx-textW(fTitle, n, 20)/2-16, r.y+30, 1, cs1)
		}
		say(dst, n, cx+14, r.y+30, 20, colWhite, 0.5)
		// little "!" when there is something to do
		badge := false
		switch i {
		case 0:
			badge = g.canUpgrade()
		case 2:
			badge = g.now() >= g.p.FreeEgg
		}
		if badge && !sel {
			bx, by := r.x+r.w-10, r.y+6
			k := 1 + 0.12*math.Sin(g.t*6)
			discArt(colRed).draw(dst, bx, by, 0.3*k, cs1)
			say(dst, "!", bx, by, 18, colWhite, 0.5)
		}
		g.hits = append(g.hits, hit{rect{float64(i) * tw, y, tw, g.h - y}, "tab:" + itoa(i)})
	}
}

func (g *Game) canUpgrade() bool {
	p := g.p
	if p.upCost(-1, 1) <= p.Coins {
		return true
	}
	for _, id := range p.Team {
		if levelCost(id, p.Bugs[id].Level) <= p.Coins {
			return true
		}
	}
	return false
}

// ---- overlays ----

func (g *Game) drawOverlays(dst *ebiten.Image) {
	if g.welcome > 0 {
		a := easeOut(g.welcomeT * 3)
		fillRect(dst, 0, 0, W, g.h, alpha(colInk, 0.7*a))
		k := backOut(math.Min(1, g.welcomeT*2.5))
		w, h := 440*k, 380*k
		r := rect{W/2 - w/2, g.h/2 - h/2, w, h}
		panel(dst, r, 0)
		if k < 0.98 {
			return
		}
		say(dst, "Welcome back!", W/2, r.y+44, 34, rgb(0xff, 0xd0, 0x3a), 0.5)
		hrs := g.welcomeS / 3600
		mins := g.welcomeS / 60 % 60
		ink(dst, fmt.Sprintf("Your brigade kept fighting for %dh %02dm", hrs, mins), W/2, r.y+90, 17, rgb(0x5a, 0x40, 0x8a), 0.5)
		raysArt().drawT(dst, W/2, r.y+170, 0.4, 0.4, g.t*0.4, csA(0.5))
		icon(icSack, 90).draw(dst, W/2, r.y+160, 1+0.04*math.Sin(g.t*4), cs1)
		icon(icCoin, 30).draw(dst, W/2-textW(fTitle, num(g.welcome), 34)/2-12, r.y+232, 1, cs1)
		say(dst, num(g.welcome), W/2+10, r.y+232, 34, colWhite, 0.5)
		g.btn(dst, rect{r.x + 24, r.y + 276, 180, 70}, "collect", colGreen, "Collect", 24, false)
		r2 := rect{r.x + r.w - 204, r.y + 276, 180, 70}
		g.btn(dst, r2, "collect2", colPink, "", 0, g.p.Gems < 30)
		say(dst, "x2", r2.x+44, r2.y+30, 26, colWhite, 0.5)
		icon(icGem, 24).draw(dst, r2.x+96, r2.y+30, 1, cs1)
		say(dst, "30", r2.x+112, r2.y+30, 22, colWhite, 0)
		return
	}
	if g.hatchID >= 0 {
		g.drawHatch(dst)
	}
}

func (g *Game) drawHatch(dst *ebiten.Image) {
	t := g.hatchT
	id := g.hatchID
	sp := bugs[id]
	a := easeOut(t * 3)
	fillRect(dst, 0, 0, W, g.h, alpha(rgb(0x14, 0x08, 0x2a), 0.85*a))
	cx, cy := W/2, g.h*0.42
	const crack = 1.7
	if t < crack {
		// the egg drops in, then wobbles harder and harder
		drop := easeOut(math.Min(1, t*2.5))
		wob := math.Sin(t*22) * 0.25 * easeIn(t/crack)
		y := cy - (1-drop)*300
		glowArt().draw(dst, cx, cy, 2.6*easeIn(t/crack), glowCS(rarityCol[sp.rarity], 0.8))
		ic := icEgg
		if sp.rarity >= rEpic {
			ic = icGoldEgg
		}
		icon(ic, 200).drawT(dst, cx, y, 1+0.06*math.Sin(t*30)*easeIn(t/crack), 1, wob, cs1)
		sayB(dst, "Hatching...", cx, cy+170, 22, alpha(colWhite, a), 0.5)
		return
	}
	k := t - crack
	if k < 0.25 {
		fillRect(dst, 0, 0, W, g.h, alpha(colWhite, 1-k*4))
	}
	raysArt().drawT(dst, cx, cy, 1.4, 1.4, t*0.4, glowCS(rarityCol[sp.rarity], 0.9))
	raysArt().drawT(dst, cx, cy, 1.0, 1.0, -t*0.25, csA(0.5))
	glowArt().draw(dst, cx, cy, 2.4, glowCS(light(rarityCol[sp.rarity], 0.4), 0.7))
	pop := elastic(math.Min(1, k*1.4))
	face := 0
	if math.Mod(t, 3) > 2.85 {
		face = 1
	}
	sc := 2.2 * pop
	drawBug(dst, id, cx-8*sc/1.5, cy+70*sc/1.6+math.Sin(t*3)*4, sc, sc*(1+0.02*math.Sin(t*4)), 0, face, t, cs1)
	if k > 0.3 {
		ta := easeOut((k - 0.3) * 3)
		title := "NEW BUG!"
		if !g.hatchNew {
			title = "+1 STAR!"
			if g.p.Bugs[id].Stars >= maxStars {
				title = "MAX STARS! +COINS"
			}
		}
		sayOut(dst, fTitle, title, cx, cy-190, 44*backOut(math.Min(1, (k-0.3)*3)), alpha(rgb(0xff, 0xe0, 0x50), ta), colInk, 0.5)
		y := cy + 170
		slice3(dst, barArt(34, rarityCol[sp.rarity], false), cx-90, y-17, 180, csA(ta))
		say(dst, rarityName[sp.rarity], cx, y-2, 20, alpha(colWhite, ta), 0.5)
		say(dst, sp.name+" the "+sp.title, cx, y+46, 30, alpha(colWhite, ta), 0.5)
		sayB(dst, "Skill: "+sp.skill+" - "+sp.skillText, cx, y+82, 15, alpha(rgb(0xd8, 0xc8, 0xff), ta), 0.5)
		if k > 0.9 {
			sayB(dst, "Tap to continue", cx, g.h-120+math.Sin(t*4)*3, 19, alpha(colWhite, 0.6+0.4*math.Sin(t*4)), 0.5)
		}
	}
	// confetti on the reveal
	if k < 0.05 {
		for i := 0; i < 50; i++ {
			an := g.rng.Float64() * 2 * math.Pi
			s := 4 + g.rng.Float64()*8
			g.parts = append(g.parts, particle{x: cx, y: cy, vx: math.Cos(an) * s, vy: math.Sin(an)*s - 4, g: 0.25, life: 2, size: 1, kind: ptConfetti,
				col: []color.RGBA{colGold, colPink, colBlue, colGreen, colPurple}[i%5], rot: an, vr: 0.2})
		}
	}
	g.drawParticles(dst)
}
