package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The menus under the field: Chefs (level up), Crew (collection), Kitchen
// (stations, recipes, the table) and Shop, plus the recruiting, welcome-back
// and cooking overlays.

var (
	inkDim  = rgb(0x5a, 0x40, 0x8a)
	inkSoft = rgb(0x9a, 0x80, 0xc0)
)

func (g *Game) drawPanel(dst *ebiten.Image) {
	y0 := g.panelY()
	h := g.tabsY() - y0
	gradRect(dst, 0, y0, W, h, rgb(0x46, 0x2a, 0x88), rgb(0x32, 0x1c, 0x66))
	for j := 0; j < int(h/36)+1; j++ {
		for i := 0; i < 16; i++ {
			x := float64(i)*36 + float64(j%2)*18
			fillRect(dst, x, y0+float64(j)*36+10, 3, 3, alpha(colWhite, 0.05))
		}
	}
	c := clip(dst, rect{0, y0, W, h})
	off := (1 - easeOut(g.tabT*4)) * 30
	sy := y0 + 10 - g.scroll[g.tab] + off
	n := len(g.hits)
	var end float64
	switch g.tab {
	case tabChefs:
		end = g.drawChefs(c, sy)
	case tabCrew:
		end = g.drawCrew(c, sy)
	case tabKitchen:
		end = g.drawKitchenPanel(c, sy)
	case tabShop:
		end = g.drawShop(c, sy)
	}
	g.scrollMax[g.tab] = math.Max(0, end+g.scroll[g.tab]-off-(y0+h)+14)
	// only the visible part of the panel takes taps
	keep := g.hits[:n]
	for _, ht := range g.hits[n:] {
		if ht.r.y+ht.r.h > y0 && ht.r.y < y0+h {
			keep = append(keep, ht)
		}
	}
	g.hits = keep
	gradRect(dst, 0, y0, W, 12, alpha(colInk, 0.5), alpha(colInk, 0))
}

func (g *Game) isPressed(id string, r rect) bool {
	return g.pressedID == id && g.ptr.down && r.has(g.ptr.x, g.ptr.y)
}

// ---- Chefs ----

func (g *Game) drawChefs(dst *ebiten.Image, y float64) float64 {
	p := g.p
	sayB(dst, "Crew damage", 18, y+18, 17, rgb(0xd8, 0xc8, 0xff), 0)
	icon(icSword, 26).draw(dst, 30, y+46, 1, cs1)
	say(dst, num(p.teamDPS())+" /s", 48, y+46, 24, colWhite, 0)
	ml := []string{"Buy x1", "Buy x10", "Buy MAX"}[g.mult]
	g.btn(dst, rect{W - 160, y + 8, 146, 50}, "mult", colBlue, ml, 20, false)
	y += 70
	row := func(id int, name, sub, sub2 string, lvl int, cost float64, n int, pic func(cx, cy float64)) {
		r := rect{12, y, W - 24, 96}
		panel(dst, r, 0)
		pic(r.x+52, r.y+48)
		ink(dst, name, r.x+104, r.y+24, 21, colInk, 0)
		ink(dst, "Lv "+itoa(lvl), r.x+104+textW(fBody, name, 21)+10, r.y+24, 17, rgb(0x8a, 0x5a, 0xd0), 0)
		ink(dst, sub, r.x+104, r.y+50, 16, inkDim, 0)
		ink(dst, sub2, r.x+104, r.y+73, 14, inkSoft, 0)
		br := rect{r.x + r.w - 150, r.y + 16, 138, 64}
		key := "up:" + itoa(id)
		g.btn(dst, br, key, colGreen, "", 0, cost > p.Coins)
		dy := 0.0
		if g.isPressed(key, br) {
			dy = 6
		}
		sayB(dst, "+"+itoa(n)+" Lv", br.cx(), br.y+18+dy, 16, colWhite, 0.5)
		icon(icCoin, 22).draw(dst, br.x+30, br.y+40+dy, 1, cs1)
		say(dst, num(cost), br.x+44, br.y+40+dy, 20, colWhite, 0)
		y += 104
	}
	n := g.buyCount(-1)
	row(-1, "Tap Power", num(p.tapDamage())+" per tap", "Also adds 8% of crew damage", p.TapLv+1, p.upCost(-1, n), n, func(cx, cy float64) {
		discArt(colOrange).draw(dst, cx, cy, 0.95, cs1)
		icon(icTap, 46).draw(dst, cx, cy, 1+0.05*math.Sin(g.t*5), cs1)
	})
	list := append([]int{}, p.Team...)
	for id := range chefs {
		if p.Chefs[id].Have && !p.inTeam(id) {
			list = append(list, id)
		}
	}
	for _, id := range list {
		b := p.Chefs[id]
		n := g.buyCount(id)
		sub := num(p.dps(id)) + " damage /s"
		if !p.inTeam(id) {
			sub += "  (off duty)"
		} else if p.favServed(id) {
			sub += "  x2 favourite!"
		}
		id := id
		row(id, chefs[id].name, sub, "Double damage at Lv "+itoa(nextMilestone(b.Level)), b.Level, p.upCost(id, n), n, func(cx, cy float64) {
			discArt(rarityCol[chefs[id].rarity]).draw(dst, cx, cy, 0.95, cs1)
			portrait(dst, id, cx, cy-2, 66, 0, g.t, false)
			for s := 0; s < b.Stars; s++ {
				icon(icStar, 14).draw(dst, cx-24+float64(s)*12, cy+36, 1, cs1)
			}
		})
	}
	return y
}

// ---- Crew ----

func (g *Game) drawCrew(dst *ebiten.Image, y float64) float64 {
	p := g.p
	have := 0
	for _, b := range p.Chefs {
		if b.Have {
			have++
		}
	}
	say(dst, "Your chefs  "+itoa(have)+"/"+itoa(len(chefs)), 18, y+20, 24, colWhite, 0)
	sayB(dst, fmt.Sprintf("Crew %d/%d · tap a chef to put them on or off duty", len(p.Team), p.slots()), 18, y+48, 15, rgb(0xd8, 0xc8, 0xff), 0)
	y += 70
	cw, ch := (W-24-20)/3, 236.0
	for i := range chefs {
		col, rowi := i%3, i/3
		r := rect{12 + float64(col)*(cw+10), y + float64(rowi)*(ch+10), cw, ch}
		b := p.Chefs[i]
		c := chefs[i]
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
		slice3(dst, barArt(22, rarityCol[c.rarity], false), rr.x+10, rr.y+10, rr.w-20, cs1)
		sayB(dst, rarityName[c.rarity], rr.cx(), rr.y+21, 13, colWhite, 0.5)
		face := 0
		if b.Have && math.Mod(g.t+float64(i)*0.7, 4) < 0.15 {
			face = 1
		}
		drawChef(dst, i, rr.cx(), rr.y+112, 0.56*k, 0.56*k, face, darkCS(!b.Have))
		if b.Have {
			ink(dst, c.name, rr.cx(), rr.y+168, 18, colInk, 0.5)
			ink(dst, c.title, rr.cx(), rr.y+188, 12, inkSoft, 0.5)
			for s := 0; s < maxStars; s++ {
				ic := icStarOff
				if s < b.Stars {
					ic = icStar
				}
				icon(ic, 17).draw(dst, rr.cx()-34+float64(s)*17, rr.y+206, 1, cs1)
			}
			foodIcon(fiDish+c.fav, 26).draw(dst, rr.x+22, rr.y+48, 1, cs1)
			if inT {
				icon(icCheck, 28).draw(dst, rr.x+rr.w-20, rr.y+46, 1, cs1)
			}
		} else {
			ink(dst, "???", rr.cx(), rr.y+168, 19, inkSoft, 0.5)
			ink(dst, "Recruit from boxes", rr.cx(), rr.y+196, 13, inkSoft, 0.5)
		}
		g.hits = append(g.hits, hit{r, "team:" + itoa(i)})
	}
	y += float64((len(chefs)+2)/3) * (ch + 10)
	sayB(dst, "The dish in the corner is each chef's favourite:", W/2, y+14, 15, rgb(0xd8, 0xc8, 0xff), 0.5)
	sayB(dst, "serve it and that chef does double damage.", W/2, y+34, 15, rgb(0xd8, 0xc8, 0xff), 0.5)
	return y + 50
}

func darkCS(dark bool) ebiten.ColorScale {
	if !dark {
		return cs1
	}
	var cs ebiten.ColorScale
	cs.Scale(0.16, 0.1, 0.3, 1)
	return cs
}

// ---- Kitchen ----

func (g *Game) drawKitchenPanel(dst *ebiten.Image, y float64) float64 {
	p := g.p
	// the table
	say(dst, "The Table", 18, y+18, 24, colWhite, 0)
	sayB(dst, "Served dishes power up the fight", 150, y+19, 14, rgb(0xd8, 0xc8, 0xff), 0)
	y += 40
	pw := (W - 24 - 20) / 3
	for i := 0; i < plates; i++ {
		r := rect{12 + float64(i)*(pw+10), y, pw, 118}
		panel(dst, r, 0)
		foodIcon(fiPlate, 70).draw(dst, r.cx(), r.y+52, 1, cs1)
		if i < len(p.Table) {
			d := p.Table[i]
			rc := recipes[d.R]
			foodIcon(fiDish+d.R, 52).draw(dst, r.cx(), r.y+40+math.Sin(g.t*3+float64(i))*2, 1, cs1)
			ink(dst, effectText(rc.eff, rc.val*d.Q), r.cx(), r.y+84, 12, inkDim, 0.5)
			bar(dst, r.x+12, r.y+98, r.w-24, 10, d.Left/dishTime, colGreen)
		} else {
			ink(dst, "empty plate", r.cx(), r.y+92, 13, inkSoft, 0.5)
		}
	}
	y += 128
	lines := p.effectLines()
	if p.fullCourse() {
		sayB(dst, "FULL COURSE: every dish x1.5", 18, y+10, 16, rgb(0xff, 0xe0, 0x60), 0)
		y += 24
	}
	for _, ci := range p.activeCombos() {
		cb := combos[ci]
		sayB(dst, "COMBO "+cb.name+": "+effectText(cb.eff, cb.val), 18, y+10, 16, rgb(0xb0, 0xff, 0x80), 0)
		y += 24
	}
	for _, id := range p.Team {
		if p.favServed(id) {
			sayB(dst, chefs[id].name+" loves this: double damage", 18, y+10, 15, rgb(0xff, 0xa0, 0xc0), 0)
			y += 22
		}
	}
	if len(lines) > 0 {
		for i, l := range lines {
			sayB(dst, l, 18+float64(i%2)*258, y+10+float64(i/2)*22, 15, colWhite, 0)
		}
		y += float64((len(lines)+1)/2)*22 + 6
	}
	// the pantry
	if len(p.Pantry) > 0 {
		say(dst, "Ready to serve", 18, y+18, 22, colWhite, 0)
		y += 38
		cw := (W - 24 - 30) / 4
		for i, d := range p.Pantry {
			r := rect{12 + float64(i%4)*(cw+10), y + float64(i/4)*136, cw, 128}
			panel(dst, r, 2)
			foodIcon(fiDish+d.R, 50).draw(dst, r.cx(), r.y+36, 1, cs1)
			ink(dst, courseName[recipes[d.R].course], r.cx(), r.y+70, 12, rgb(0x9a, 0x5a, 0x20), 0.5)
			stars := 1
			if d.Q >= 1.5 {
				stars = 3
			} else if d.Q >= 1 {
				stars = 2
			}
			for s := 0; s < 3; s++ {
				ic := icStarOff
				if s < stars {
					ic = icStar
				}
				icon(ic, 13).draw(dst, r.cx()-14+float64(s)*14, r.y+84, 1, cs1)
			}
			g.btn(dst, rect{r.x + 8, r.y + 94, r.w - 16, 28}, "serve:"+itoa(i), colOrange, "Serve", 15, len(p.Table) >= plates)
		}
		y += float64((len(p.Pantry)+3)/4)*136 + 6
	}
	// the selected station
	st := g.station
	lv := p.Stations[st]
	r := rect{12, y, W - 24, 78}
	panel(dst, r, 1)
	stationArt(st).draw(dst, r.x+44, r.y+72, 0.36, cs1)
	title := stationName[st]
	if lv > 0 {
		title += "  Lv " + itoa(lv)
	}
	say(dst, title, r.x+84, r.y+26, 22, colWhite, 0)
	sub := "Tap a station above to see its recipes"
	if lv > 0 {
		sub = fmt.Sprintf("Cooks %.0f%% faster, dishes %.0f%% stronger", (1-math.Pow(0.85, float64(lv-1)))*100, (p.stationBonus(st)-1)*100)
		if lv == 1 {
			sub = "Upgrade: cooks faster, dishes stronger"
		}
	} else {
		sub = "Build it to cook its recipes"
	}
	sayB(dst, sub, r.x+84, r.y+54, 13, rgb(0xd8, 0xc8, 0xff), 0)
	br := rect{r.x + r.w - 136, r.y + 12, 124, 54}
	label := "Upgrade"
	if lv == 0 {
		label = "Build"
	}
	key := "build:" + itoa(st)
	price := p.stationPrice(st)
	g.btn(dst, br, key, colGreen, "", 0, price > p.Coins)
	dy := 0.0
	if g.isPressed(key, br) {
		dy = 5
	}
	sayB(dst, label, br.cx(), br.y+15+dy, 14, colWhite, 0.5)
	icon(icCoin, 18).draw(dst, br.x+26, br.y+33+dy, 1, cs1)
	say(dst, num(price), br.x+38, br.y+33+dy, 17, colWhite, 0)
	y += 88
	for ri, rc := range recipes {
		if rc.station != st {
			continue
		}
		y = g.recipeRow(dst, ri, y)
	}
	// combos guide
	y += 6
	say(dst, "Combos", 18, y+18, 22, colWhite, 0)
	sayB(dst, "Starter + main + dessert = full course, every dish x1.5", 18, y+44, 13, rgb(0xff, 0xe0, 0x60), 0)
	y += 56
	for _, cb := range combos {
		x := 22.0
		for k, d := range cb.dishes {
			if k > 0 {
				sayB(dst, "+", x+2, y+18, 16, colWhite, 0)
				x += 16
			}
			foodIcon(fiDish+d, 30).draw(dst, x+14, y+18, 1, cs1)
			x += 32
		}
		sayB(dst, cb.name, x+10, y+10, 15, rgb(0xb0, 0xff, 0x80), 0)
		sayB(dst, effectText(cb.eff, cb.val), x+10, y+28, 13, colWhite, 0)
		y += 42
	}
	return y + 10
}

func (g *Game) recipeRow(dst *ebiten.Image, ri int, y float64) float64 {
	p := g.p
	rc := recipes[ri]
	r := rect{12, y, W - 24, 104}
	panel(dst, r, 0)
	foodIcon(fiDish+ri, 58).draw(dst, r.x+44, r.y+48, 1, cs1)
	ink(dst, rc.name, r.x+84, r.y+20, 19, colInk, 0)
	ink(dst, courseName[rc.course], r.x+84+textW(fBody, rc.name, 19)+8, r.y+21, 13, rgb(0x8a, 0x5a, 0xd0), 0)
	ink(dst, effectText(rc.eff, rc.val*p.stationBonus(rc.station))+" for 4 min", r.x+84, r.y+44, 14, inkDim, 0)
	x := r.x + 84
	for i, n := range rc.need {
		if n == 0 {
			continue
		}
		foodIcon(i, 26).draw(dst, x+12, r.y+76, 1, cs1)
		col := rgb(0x3a, 0x22, 0x6e)
		if p.Ing[i] < n {
			col = colRed
		}
		ink(dst, itoa(p.Ing[i])+"/"+itoa(n), x+26, r.y+77, 13, col, 0)
		x += 66
	}
	br := rect{r.x + r.w - 118, r.y + 22, 106, 58}
	err := p.canCook(ri)
	key := "cook:" + itoa(ri)
	g.btn(dst, br, key, colOrange, "", 0, err != nil)
	dy := 0.0
	if g.isPressed(key, br) {
		dy = 5
	}
	sayB(dst, "Cook", br.cx(), br.y+18+dy, 18, colWhite, 0.5)
	sayB(dst, fmt.Sprintf("%ds", int(math.Ceil(p.cookTime(ri)))), br.cx(), br.y+38+dy, 13, colWhite, 0.5)
	return y + 112
}

// ---- Shop ----

func (g *Game) drawShop(dst *ebiten.Image, y float64) float64 {
	p := g.p
	now := g.now()
	r := rect{12, y, W - 24, 120}
	panel(dst, r, 2)
	ready := now >= p.FreeBox
	bob := 0.0
	if ready {
		bob = math.Sin(g.t*5) * 0.12
		glowArt().draw(dst, r.x+64, r.y+60, 0.9, glowCS(rgb(0xff, 0xe0, 0x80), 0.6))
	}
	foodIcon(fiBox, 80).drawT(dst, r.x+64, r.y+62, 1, 1, bob, cs1)
	ink(dst, "Free Recruit", r.x+124, r.y+36, 24, rgb(0x6a, 0x30, 0x10), 0)
	if ready {
		ink(dst, "A new chef is waiting!", r.x+124, r.y+70, 16, rgb(0x9a, 0x5a, 0x20), 0)
		g.btn(dst, rect{r.x + r.w - 150, r.y + 30, 136, 62}, "free", colGreen, "Open!", 24, false)
	} else {
		left := p.FreeBox - now
		ink(dst, "A new one every 4 hours", r.x+124, r.y+70, 16, rgb(0x9a, 0x5a, 0x20), 0)
		icon(icClock, 26).draw(dst, r.x+r.w-120, r.y+60, 1, cs1)
		say(dst, fmt.Sprintf("%d:%02d:%02d", left/3600, left/60%60, left%60), r.x+r.w-100, r.y+60, 22, colWhite, 0)
	}
	y += 132
	cw := (W - 24 - 10) / 2
	for i := 0; i < 2; i++ {
		it := shopItems[i]
		r := rect{12 + float64(i)*(cw+10), y, cw, 250}
		panel(dst, r, 0)
		ic := fiBox
		if i == 1 {
			ic = fiGoldBox
			raysArt().drawT(dst, r.cx(), r.y+80, 0.35, 0.35, g.t*0.3, csA(0.5))
		}
		foodIcon(ic, 96).drawT(dst, r.cx(), r.y+80, 1, 1, math.Sin(g.t*2+float64(i))*0.06, cs1)
		ink(dst, it.name, r.cx(), r.y+148, 21, colInk, 0.5)
		ink(dst, it.desc, r.cx(), r.y+174, 13, inkDim, 0.5)
		g.gemButton(dst, rect{r.x + 18, r.y + 188, r.w - 36, 52}, "buy:"+itoa(i), it.gems)
	}
	y += 262
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
		switch it.icon {
		case iconIngredients:
			for k, f := range []int{inTomato, inCheese, inMeat} {
				drawFood(dst, f, r.x+36+float64(k)*16, r.y+56-float64(k%2)*12, 36)
			}
		default:
			ic := map[int]int{iconCoins: icSack, iconGems: icGem, iconChest: icChest, iconStarter: icGift}[it.icon]
			icon(ic, 62).draw(dst, r.x+52, r.y+50, 1+0.04*math.Sin(g.t*3+float64(i)), cs1)
		}
		ink(dst, it.name, r.x+100, r.y+36, 21, colInk, 0)
		desc := it.desc
		if it.id == "coins" {
			desc = "+" + num(math.Max(100, p.coinRate()*3600)) + " coins right now"
		}
		ink(dst, desc, r.x+100, r.y+66, 14, inkDim, 0)
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
	if g.isPressed(id, r) {
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
	names := []string{"Chefs", "Crew", "Kitchen", "Shop"}
	tw := W / numTabs
	for i, n := range names {
		sel := g.tab == i
		r := rect{float64(i)*tw + 6, y + 10, tw - 12, 68}
		col := colPanel
		if sel {
			col = rgb(0xff, 0xb4, 0x2a)
			r.y -= 8 * easeOut(g.tabT*5)
		}
		button(dst, r, col, "", 0, false, false)
		cx := r.cx()
		switch i {
		case tabChefs:
			icon(icSword, 30).draw(dst, cx, r.y+24, 1, cs1)
		case tabCrew:
			portrait(dst, 0, cx, r.y+26, 34, 0, g.t, false)
		case tabKitchen:
			drawFood(dst, fiDish+dPizza, cx, r.y+24, 34*(1+0.15*g.ingPulse))
		case tabShop:
			icon(icGem, 30).draw(dst, cx, r.y+24, 1, cs1)
		}
		say(dst, n, cx, r.y+50, 16, colWhite, 0.5)
		badge := false
		switch i {
		case tabChefs:
			badge = g.canUpgrade()
		case tabKitchen:
			badge = g.kitchenNews()
		case tabShop:
			badge = g.now() >= g.p.FreeBox
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
		if levelCost(id, p.Chefs[id].Level) <= p.Coins {
			return true
		}
	}
	return false
}

// kitchenNews is true when a dish is ready, or a plate is free and a dish waits.
func (g *Game) kitchenNews() bool {
	p := g.p
	for _, c := range p.Cook {
		if c.R >= 0 && c.Left <= 0 {
			return true
		}
	}
	return len(p.Pantry) > 0 && len(p.Table) < plates
}

// ---- overlays ----

func (g *Game) drawOverlays(dst *ebiten.Image) {
	switch {
	case g.welcome.secs > 0:
		g.drawWelcome(dst)
	case g.cookR >= 0:
		g.drawCookGame(dst)
	case g.hireID >= 0:
		g.drawHire(dst)
	}
}

func (g *Game) drawWelcome(dst *ebiten.Image) {
	a := easeOut(g.welcomeT * 3)
	fillRect(dst, 0, 0, W, g.h, alpha(colInk, 0.7*a))
	k := backOut(math.Min(1, g.welcomeT*2.5))
	w, h := 460*k, 450*k
	r := rect{W/2 - w/2, g.h/2 - h/2, w, h}
	panel(dst, r, 0)
	if k < 0.98 {
		return
	}
	wl := g.welcome
	say(dst, "Welcome back!", W/2, r.y+44, 34, rgb(0xff, 0xd0, 0x3a), 0.5)
	ink(dst, fmt.Sprintf("Your crew kept cooking for %dh %02dm", wl.secs/3600, wl.secs/60%60), W/2, r.y+88, 17, inkDim, 0.5)
	raysArt().drawT(dst, W/2, r.y+160, 0.4, 0.4, g.t*0.4, csA(0.5))
	icon(icSack, 84).draw(dst, W/2, r.y+152, 1+0.04*math.Sin(g.t*4), cs1)
	icon(icCoin, 30).draw(dst, W/2-textW(fTitle, num(wl.coins), 34)/2-12, r.y+222, 1, cs1)
	say(dst, num(wl.coins), W/2+10, r.y+222, 34, colWhite, 0.5)
	n := 0
	for _, c := range wl.ing {
		if c > 0 {
			n++
		}
	}
	x := W/2 - float64(n)*60/2 + 30
	for i, c := range wl.ing {
		if c == 0 {
			continue
		}
		drawFood(dst, i, x, r.y+272, 36)
		ink(dst, "+"+itoa(c), x, r.y+304, 15, inkDim, 0.5)
		x += 60
	}
	g.btn(dst, rect{r.x + 24, r.y + 346, 190, 70}, "collect", colGreen, "Collect", 24, false)
	r2 := rect{r.x + r.w - 214, r.y + 346, 190, 70}
	g.btn(dst, r2, "collect2", colPink, "", 0, g.p.Gems < 30)
	say(dst, "x2", r2.x+48, r2.y+30, 26, colWhite, 0.5)
	icon(icGem, 24).draw(dst, r2.x+100, r2.y+30, 1, cs1)
	say(dst, "30", r2.x+116, r2.y+30, 22, colWhite, 0)
}

// drawCookGame is the cooking minigame: stop the needle in the green.
func (g *Game) drawCookGame(dst *ebiten.Image) {
	t := g.cookT
	a := easeOut(t * 4)
	fillRect(dst, 0, 0, W, g.h, alpha(colInk, 0.75*a))
	rc := recipes[g.cookR]
	cy := g.h * 0.42
	k := backOut(math.Min(1, t*3))
	say(dst, "Cooking: "+rc.name, W/2, cy-200, 30*k, colWhite, 0.5)
	// a sizzling pan with the dish in it
	sz := math.Sin(t*30) * 2
	glowArt().drawT(dst, W/2, cy-40, 2, 0.8, 0, glowCS(rgb(0xff, 0x90, 0x30), 0.4))
	panArt().draw(dst, W/2, cy-40+sz*0.3, 1.6*k, cs1)
	foodIcon(fiDish+g.cookR, 90).drawT(dst, W/2, cy-60+sz, k, k, math.Sin(t*6)*0.08, cs1)
	if g.frame%5 == 0 {
		g.parts = append(g.parts, particle{x: W/2 + (g.rng.Float64()-0.5)*100, y: cy - 70, vy: -1.2, life: 1, size: 0.8, kind: ptSteam})
	}
	g.drawParticles(dst, true)
	// the meter
	mx, my, mw := W/2-200, cy+60, 400.0
	slice3(dst, barArt(40, colRed, false), mx, my, mw, cs1)
	slice3(dst, barArt(40, colGold, false), mx+mw*0.28, my, mw*0.44, cs1)
	slice3(dst, barArt(40, colGreen, false), mx+mw*0.43, my, mw*0.14, cs1)
	pos := needle(t)
	if g.cookDone > 0 {
		pos = g.cookPos
	}
	nx := mx + 20 + (mw-40)*pos
	icon(icTap, 34).draw(dst, nx, my-14, 1, cs1)
	fillRect(dst, nx-2.5, my-4, 5, 48, colInk)
	fillRect(dst, nx-1, my-2, 2, 44, colWhite)
	if g.cookDone > 0 {
		w, col := cookWord(g.cookQ)
		q := backOut(math.Min(1, (t-g.cookDone)*4))
		sayOut(dst, fTitle, w, W/2, my+100, 48*q, col, colInk, 0.5)
	} else {
		pulse := 0.6 + 0.4*math.Sin(t*6)
		sayB(dst, "Tap when the needle is in the green!", W/2, my+80, 20, alpha(colWhite, pulse), 0.5)
		sayB(dst, "Perfect makes the dish 50% stronger", W/2, my+110, 15, rgb(0xd8, 0xc8, 0xff), 0.5)
	}
}

func panArt() *pimg {
	return paintArt("pan", -110, -40, 110, 40, 1, func(c *canvas) {
		c.part(rrect(40, -10, 66, 16, 7), mat{base: rgb(0x3a, 0x2a, 0x24), gloss: 0.6}, 2.4)
		pan := ell(-10, 0, 60, 26)
		c.part(pan, mat{base: rgb(0x3a, 0x3a, 0x48), gloss: 1, shine: 26}, 2.4)
		c.volOn(ell(-10, -4, 50, 18), pan, mat{base: rgb(0x22, 0x22, 0x2c), flat: 0.4})
	})
}

func (g *Game) drawHire(dst *ebiten.Image) {
	t := g.hireT
	id := g.hireID
	c := chefs[id]
	a := easeOut(t * 3)
	fillRect(dst, 0, 0, W, g.h, alpha(rgb(0x14, 0x08, 0x2a), 0.85*a))
	cx, cy := W/2, g.h*0.42
	const open = 1.7
	if t < open {
		drop := easeOut(math.Min(1, t*2.5))
		wob := math.Sin(t*22) * 0.25 * easeIn(t/open)
		y := cy - (1-drop)*300
		glowArt().draw(dst, cx, cy, 2.6*easeIn(t/open), glowCS(rarityCol[c.rarity], 0.8))
		ic := fiBox
		if c.rarity >= rEpic {
			ic = fiGoldBox
		}
		foodIcon(ic, 200).drawT(dst, cx, y, 1+0.06*math.Sin(t*30)*easeIn(t/open), 1, wob, cs1)
		sayB(dst, "Opening...", cx, cy+170, 22, alpha(colWhite, a), 0.5)
		return
	}
	k := t - open
	if k < 0.25 {
		fillRect(dst, 0, 0, W, g.h, alpha(colWhite, 1-k*4))
	}
	raysArt().drawT(dst, cx, cy, 1.4, 1.4, t*0.4, glowCS(rarityCol[c.rarity], 0.9))
	raysArt().drawT(dst, cx, cy, 1.0, 1.0, -t*0.25, csA(0.5))
	glowArt().draw(dst, cx, cy, 2.4, glowCS(light(rarityCol[c.rarity], 0.4), 0.7))
	pop := elastic(math.Min(1, k*1.4))
	face := 2
	if math.Mod(t, 3) > 2.85 {
		face = 1
	}
	sc := 1.5 * pop
	drawChef(dst, id, cx, cy+40*sc+math.Sin(t*3)*4, sc, sc*(1+0.02*math.Sin(t*4)), face, cs1)
	if k > 0.3 {
		ta := easeOut((k - 0.3) * 3)
		title := "NEW CHEF!"
		if !g.hireNew {
			title = "+1 STAR!"
			if g.p.Chefs[id].Stars >= maxStars {
				title = "MAX STARS! +COINS"
			}
		}
		sayOut(dst, fTitle, title, cx, cy-200, 44*backOut(math.Min(1, (k-0.3)*3)), alpha(rgb(0xff, 0xe0, 0x50), ta), colInk, 0.5)
		y := cy + 170
		slice3(dst, barArt(34, rarityCol[c.rarity], false), cx-90, y-17, 180, csA(ta))
		say(dst, rarityName[c.rarity], cx, y-2, 20, alpha(colWhite, ta), 0.5)
		say(dst, c.name+", "+c.title, cx, y+46, 28, alpha(colWhite, ta), 0.5)
		sayB(dst, "Skill: "+c.skill+" - "+c.skillText, cx, y+82, 15, alpha(rgb(0xd8, 0xc8, 0xff), ta), 0.5)
		sayB(dst, "Favourite dish: "+recipes[c.fav].name, cx, y+106, 15, alpha(rgb(0xff, 0xb0, 0xd0), ta), 0.5)
		if k > 0.9 {
			sayB(dst, "Tap to continue", cx, g.h-110+math.Sin(t*4)*3, 19, alpha(colWhite, 0.6+0.4*math.Sin(t*4)), 0.5)
		}
	}
	if k < 0.05 {
		for i := 0; i < 50; i++ {
			an := g.rng.Float64() * 2 * math.Pi
			s := 4 + g.rng.Float64()*8
			g.parts = append(g.parts, particle{x: cx, y: cy, vx: math.Cos(an) * s, vy: math.Sin(an)*s - 4, g: 0.25, life: 2, size: 1, kind: ptConfetti,
				col: []color.RGBA{colGold, colPink, colBlue, colGreen, colPurple}[i%5], rot: an, vr: 0.2})
		}
	}
	g.drawParticles(dst, false)
}
