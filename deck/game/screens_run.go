package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// click sends a tap to whatever is on screen.
func (g *Game) click(x, y float64) {
	if g.popup != "" {
		g.popup, g.popCard = "", nil
		return
	}
	if g.viewDeck {
		if g.closeBtn().has(x, y) {
			g.press("deck")
		}
		return
	}
	switch g.sc {
	case scHome:
		g.clickHome(x, y)
	case scMap:
		g.clickMap(x, y)
	case scBattle:
		g.clickBattle(x, y)
	case scReward:
		r := g.run()
		for i := range r.Choices {
			if g.rewardCard(i).has(x, y) {
				g.press("pick:" + itoa(i))
				return
			}
		}
		if g.rewardBtn().has(x, y) {
			if len(r.Choices) > 0 {
				g.press("pick:-1")
			} else {
				g.press("continue")
			}
		}
	case scShop:
		r := g.run()
		for i := range r.Shop {
			if g.offerRect(i).has(x, y) {
				g.press("offer:" + itoa(i))
				return
			}
		}
		for i, id := range []string{"remove", "forge", "heal"} {
			if g.serviceBtn(i).has(x, y) {
				g.press(id)
				return
			}
		}
		if g.leaveBtn().has(x, y) {
			g.press("continue")
		}
		g.hudClick(x, y)
	case scRest:
		switch {
		case g.restBtn(0).has(x, y):
			g.press("rest")
		case g.restBtn(1).has(x, y):
			g.press("forge")
		case g.leaveBtn().has(x, y):
			g.press("continue")
		}
		g.hudClick(x, y)
	case scEvent:
		if g.eventMsg != "" {
			if g.eventBtn(0).has(x, y) {
				g.press("continue")
			}
			return
		}
		for i := range events[g.run().Event].Choices {
			if g.eventBtn(i).has(x, y) {
				g.press("choose:" + itoa(i))
			}
		}
	case scPick:
		if g.closeBtn().has(x, y) {
			if g.prevSc == scRest {
				g.sc = scRest
			} else {
				g.press("cancel")
			}
			return
		}
		for i := range g.run().Deck {
			if g.listCard(i).has(x, y) && y > 150 && y < g.h-120 {
				g.press("picked:" + itoa(i))
				return
			}
		}
	case scEnd:
		ids := []string{"home"}
		if g.run().Lost && !g.run().Revived {
			ids = []string{"revive", "ad", "home"}
		}
		for i, r := range g.endBtns() {
			if r.has(x, y) {
				g.press(ids[i])
			}
		}
	}
}

// ---- run HUD (map, shop, campfire) ----

func (g *Game) hudHome() rect { return rect{14, 14, 54, 54} }
func (g *Game) hudDeck() rect { return rect{W - 128, 16, 112, 50} }

func (g *Game) hudClick(x, y float64) {
	switch {
	case g.hudHome().has(x, y):
		g.press("home")
	case g.hudDeck().has(x, y):
		g.press("deck")
	case y > 70 && y < 104:
		for i, c := range g.run().Charms {
			if math.Hypot(x-(W-30-float64(i)*36), y-88) < 16 {
				ch := charmByID(c)
				g.say(ch.Name + ": " + ch.Desc)
			}
		}
	}
}

func (g *Game) drawHUD(dst *ebiten.Image, home bool) {
	r := g.run()
	texRect(dst, 0, 0, W, 76, rgb(0x1e, 0x1a, 0x24), rgb(0x0c, 0x0a, 0x10))
	gradRect(dst, 0, 74, W, 3, colGold, colGoldLo)
	gradRect(dst, 0, 77, W, 40, alpha(colInk, 0.8), clear)
	if home {
		hb := g.hudHome()
		button(dst, hb, btnStone, "", 0, false)
		cx, cy := hb.cx(), hb.cy()
		shape(dst, []float64{cx - 13, cy + 12, cx - 13, cy - 2, cx, cy - 14, cx + 13, cy - 2, cx + 13, cy + 12, cx + 4, cy + 12, cx + 4, cy + 3, cx - 4, cy + 3, cx - 4, cy + 12}, colGoldHi, colGoldLo, 1.2)
	}
	hr := rect{84, 22, 130, 36}
	pill(dst, hr, itoa(r.HP)+"/"+itoa(r.Max))
	heartIcon(dst, hr.x+10, hr.cy(), 12, colRed)
	cr := rect{232, 22, 112, 36}
	pill(dst, cr, itoa(r.Coins))
	coinIcon(dst, cr.x+8, cr.cy(), 14)
	db := g.hudDeck()
	button(dst, db, btnStone, "", 0, false)
	dk := chamfer(rect{db.x + 13, db.y + 12, 15, 24}, 3)
	strokePath(dst, dk, 2.4, colInk)
	texPath(dst, dk, rgb(0x5a, 0x4a, 0x3a), rgb(0x22, 0x1a, 0x12))
	strokePath(dst, dk, 1, colGold)
	say(dst, "Deck "+itoa(len(r.Deck)), db.x+db.w/2+12, db.y+(db.h-6)/2, 18, colBone, 0.5)
	say(dst, chapterName(r.Chapter), 20, 88, 17, colCream, 0)
	for i, c := range r.Charms {
		charmIcon(dst, c, W-30-float64(i)*36, 88, 14, g.t())
	}
}

// ---- map ----

const (
	mapTop = 200.0
	rowGap = 100.0
)

func (g *Game) nodeY(row int) float64 { return mapTop + float64(mapRows-row)*rowGap }
func (g *Game) mapH() float64         { return mapTop + mapRows*rowGap + 220 }
func (g *Game) clampMap(v float64) float64 {
	return math.Max(0, math.Min(g.mapH()-g.h, v))
}

func (g *Game) nodePos(i int) (float64, float64) {
	n := g.run().Nodes[i]
	if n.Kind == nBoss {
		return W / 2, g.nodeY(n.Row) - 30 - g.mapScroll
	}
	return 70 + float64(n.Col)*100 + n.JX, g.nodeY(n.Row) + n.JY - g.mapScroll
}

func (g *Game) clickMap(x, y float64) {
	if y < 108 {
		g.hudClick(x, y)
		return
	}
	r := g.run()
	for _, i := range r.Reachable() {
		nx, ny := g.nodePos(i)
		rad := 34.0
		if r.Nodes[i].Kind == nBoss {
			rad = 60
		}
		if math.Hypot(x-nx, y-ny) < rad {
			g.press("node:" + itoa(i))
			return
		}
	}
}

func (g *Game) drawMap(dst *ebiten.Image) {
	r := g.run()
	t := g.t()
	if g.mapImg == nil || g.mapKey != r.Seed || g.mapS != S {
		g.mapImg = mapBG(r, W, g.mapH())
		g.mapKey, g.mapS = r.Seed, S
	}
	dst.DrawImage(g.mapImg, imgOp(0, -g.mapScroll, 1))
	reach := r.Reachable()
	// paths
	for i, n := range r.Nodes {
		x0, y0 := g.nodePos(i)
		for _, j := range n.Next {
			x1, y1 := g.nodePos(j)
			done := n.Done && (r.Cur == j || r.Nodes[j].Done)
			c := alpha(colInk, 0.55)
			if done {
				c = colYellow
			}
			d := math.Hypot(x1-x0, y1-y0)
			for s := 30.0; s < d-30; s += 13 {
				f := s / d
				circle(dst, x0+(x1-x0)*f, y0+(y1-y0)*f, 3.2, c)
			}
		}
	}
	// nodes
	for i, n := range r.Nodes {
		x, y := g.nodePos(i)
		if y < -60 || y > g.h+60 {
			continue
		}
		rad := 28.0
		if n.Kind == nBoss {
			rad = 54
		}
		can := containsInt(reach, i)
		if can {
			p := 0.5 + 0.5*math.Sin(t*5)
			glow(dst, x, y, rad*2.2, colYellow, 0.4+0.3*p)
			rad *= 1 + 0.08*p
		}
		base := []color.RGBA{rgb(0x8a, 0x2a, 0x22), rgb(0x5a, 0x1a, 0x7a), rgb(0x8a, 0x6a, 0x20), rgb(0x9a, 0x4a, 0x14), rgb(0x1e, 0x4a, 0x7a), rgb(0x7a, 0x5a, 0x1a), rgb(0x9a, 0x10, 0x18)}[n.Kind]
		old := n.Done || (!can && r.Cur >= 0 && n.Row <= r.Nodes[r.Cur].Row)
		if old {
			base = mix(base, rgb(0x20, 0x1e, 0x24), 0.75)
		}
		shadow(dst, x, y+rad*0.7, rad*1.2, rad*0.4, 0.7)
		medallion(dst, x, y, rad, base, n.Kind == nBoss)
		nodeIcon(dst, n.Kind, x, y, rad*0.82, t)
		if old {
			disc(dst, x, y, rad*0.86, rad*0.86, alpha(colInk, 0.5))
		}
		if n.Done {
			shape(dst, []float64{x + rad*0.2, y + rad*0.5, x + rad*0.9, y - rad*0.3, x + rad*1.05, y - rad*0.1, x + rad*0.25, y + rad*0.85, x - rad*0.2, y + rad*0.35, x - rad*0.05, y + rad*0.2}, colGreen, shade(colGreen, 0.7), 2)
		}
		if i == r.Cur {
			drawHero(g.heroSpr.buf(), 150, 260, t, 0)
			g.heroSpr.stamp(dst, x, y-rad*0.4, 150, 260, 0.42, 1, 1, 1, 0)
		}
	}
	if r.Cur < 0 {
		say(dst, "Choose your path", W/2, g.h-60, 24, colBone, 0.5)
	}
	g.drawParticles(dst)
	g.drawHUD(dst, true)
}

// ---- reward ----

func (g *Game) rewardCard(i int) rect { return rect{22 + float64(i)*170, g.rewardY(), 156, 156 * 1.4} }
func (g *Game) lootY() float64 {
	if r := g.run(); r.Nodes[r.Cur].Kind == nTreasure {
		return 350
	}
	return 230
}
func (g *Game) rewardY() float64 {
	r := g.run()
	n := 1
	if r.RewardCharm != "" {
		n++
	}
	if r.RewardGold > 0 {
		n++
	}
	return g.lootY() + float64(n)*50 + 10 + 80
}
func (g *Game) rewardBtn() rect {
	if len(g.run().Choices) == 0 {
		return rect{160, g.rewardY() + 20, 220, 70}
	}
	return rect{180, g.rewardY() + 240, 180, 62}
}

func (g *Game) drawReward(dst *ebiten.Image) {
	r := g.run()
	t := g.t()
	if r.Nodes[r.Cur].Kind == nTreasure {
		g.drawMapUnder(dst)
	} else {
		dst.DrawImage(g.battleBG(r.Chapter), nil)
	}
	fillRect(dst, 0, 0, W, g.h, color.RGBA{0, 0, 0, 0xa8})
	title := "VICTORY!"
	if r.Nodes[r.Cur].Kind == nTreasure {
		title = "TREASURE!"
		glow(dst, W/2, 260, 140, colYellow, 0.4)
		chestIcon(dst, W/2, 270, 0.9, true)
	}
	ribbon(dst, W/2, 150, 360, title, rgb(0x6a, 0x14, 0x1c))
	lp := rect{60, g.lootY(), 420, 0}
	rows := []func(float64){func(yy float64) {
		coinIcon(dst, lp.x+34, yy, 14)
		say(dst, "+"+itoa(r.RewardCoins)+" coins", lp.x+60, yy, 22, colBone, 0)
	}}
	if r.RewardGold > 0 {
		rows = append(rows, func(yy float64) {
			coinIcon(dst, lp.x+34, yy, 14)
			say(dst, "+"+itoa(r.RewardGold)+" gold (after the run)", lp.x+60, yy, 18, colCream, 0)
		})
	}
	if r.RewardCharm != "" {
		ch := charmByID(r.RewardCharm)
		rows = append(rows, func(yy float64) {
			charmIcon(dst, r.RewardCharm, lp.x+34, yy, 16, t)
			say(dst, ch.Name, lp.x+60, yy-8, 20, colGoldHi, 0)
			drawText(dst, ch.Desc, lp.x+60, yy+13, 14, colCream, 0)
		})
	}
	lp.h = float64(len(rows))*50 + 10
	panel(dst, lp, rgb(0x3a, 0x4e, 0x96), rgb(0x22, 0x2c, 0x5c), 14)
	for i, f := range rows {
		f(lp.y + 30 + float64(i)*50)
	}
	if len(r.Choices) > 0 {
		say(dst, "Add a card to your deck", W/2, g.rewardY()-30, 22, colBone, 0.5)
		for i, c := range r.Choices {
			cr := g.rewardCard(i)
			glow(dst, cr.cx(), cr.cy(), 110, rarityCol[c.def().Rarity], 0.3)
			drawCard(dst, c, cr.x, cr.y, cr.w, false, t)
		}
		button(dst, g.rewardBtn(), btnStone, "Skip", 24, false)
	} else {
		button(dst, g.rewardBtn(), btnGo, "Continue", 28, false)
	}
	g.drawParticles(dst)
}

func (g *Game) drawMapUnder(dst *ebiten.Image) {
	r := g.run()
	if g.mapImg == nil || g.mapKey != r.Seed || g.mapS != S {
		g.mapImg = mapBG(r, W, g.mapH())
		g.mapKey, g.mapS = r.Seed, S
	}
	dst.DrawImage(g.mapImg, imgOp(0, -g.mapScroll, 1))
}

// ---- merchant ----

func (g *Game) offerRect(i int) rect {
	o := g.run().Shop[i]
	if o.Charm != "" {
		j := i - 5
		return rect{360, 512 + float64(j)*96, 164, 86}
	}
	if i < 3 {
		return rect{36 + float64(i)*178, 300, 112, 112 * 1.4}
	}
	return rect{36 + float64(i-3)*178, 512, 112, 112 * 1.4}
}
func (g *Game) serviceBtn(i int) rect { return rect{18 + float64(i)*170, 730, 164, 78} }
func (g *Game) leaveBtn() rect        { return rect{170, g.h - 104, 200, 66} }

func (g *Game) drawShop(dst *ebiten.Image) {
	r := g.run()
	t := g.t()
	g.drawMapUnder(dst)
	gradRect(dst, 0, 0, W, g.h, alpha(rgb(0x10, 0x08, 0x08), 0.8), alpha(rgb(0x04, 0x02, 0x06), 0.95))
	glow(dst, W/2-60, 200, 220, colOrange, 0.2)
	// beam with hanging banners
	texRect(dst, 0, 100, W, 22, rgb(0x4a, 0x30, 0x1e), rgb(0x1e, 0x12, 0x0a))
	gradRect(dst, 0, 122, W, 3, colInk, colInk)
	for i := 0; i < 4; i++ {
		x := 40 + float64(i)*140
		c := []color.RGBA{rgb(0x6a, 0x14, 0x1c), rgb(0x2a, 0x1a, 0x3a)}[i%2]
		shape(dst, []float64{x, 122, x + 50, 122, x + 50, 186, x + 25, 172, x, 186}, shade(c, 1.1), shade(c, 0.4), 1.4)
		strokePath(dst, pathOf([]float64{x + 4, 126, x + 46, 126, x + 46, 178, x + 25, 166, x + 4, 178}), 1, alpha(colGold, 0.6))
		setGem(dst, x+25, 144, 4, colOrange)
	}
	merchant(dst, W/2, 262, t)
	roundRect(dst, rect{-4, 250, W + 8, 34}, 6, colInk)
	texRect(dst, 0, 253, W, 28, rgb(0x5a, 0x3a, 0x22), rgb(0x22, 0x14, 0x0a))
	gradRect(dst, 0, 253, W, 2, colGold, colGold)
	for i, o := range r.Shop {
		rr := g.offerRect(i)
		var tag rect
		if o.Charm != "" {
			ch := charmByID(o.Charm)
			frame(dst, rr, rgb(0x2a, 0x1a, 0x30), colPurple, false)
			charmIcon(dst, o.Charm, rr.x+32, rr.y+34, 20, t)
			drawText(dst, wrap(ch.Name, 12), rr.x+100, rr.y+30, 15, colBone, 0.5)
			tag = rect{rr.x + 60, rr.y + 54, 96, 26}
		} else {
			glow(dst, rr.cx(), rr.cy(), 80, rarityCol[o.Card.def().Rarity], 0.2)
			drawCard(dst, o.Card, rr.x, rr.y, rr.w, o.Sold, t)
			tag = rect{rr.x + 8, rr.y + rr.h + 10, rr.w - 16, 28}
		}
		if o.Sold {
			say(dst, "SOLD", rr.cx(), rr.cy(), 28, colRed, 0.5)
			continue
		}
		roundRect(dst, tag.grow(2), 10, colInk)
		fillPath(dst, chamfer(tag, 6), rgb(0x14, 0x10, 0x0c))
		strokePath(dst, chamfer(tag, 6), 1, colGold)
		coinIcon(dst, tag.x+14, tag.cy(), 9)
		c := colGoldHi
		if r.Coins < o.Price {
			c = rgb(0xd0, 0x40, 0x40)
		}
		drawText(dst, itoa(o.Price), tag.cx()+8, tag.cy(), 17, c, 0.5)
	}
	labels := []string{"Remove", enhDefs[r.ForgeEnh].Name, "Heal"}
	subs := []string{"a card", "a card", "30% HP"}
	prices := []int{removePrice, forgePrice, healPrice}
	used := []bool{r.ShopRemoved, r.ShopForged, r.ShopHealed}
	cols := []color.RGBA{colRed, enhDefs[r.ForgeEnh].Col, colGreen}
	for i := range labels {
		b := g.serviceBtn(i)
		button(dst, b, shade(cols[i], 0.9), "", 0, used[i])
		say(dst, labels[i], b.cx(), b.y+18, 20, colBone, 0.5)
		drawText(dst, subs[i], b.cx(), b.y+38, 13, colBone, 0.5)
		if used[i] {
			say(dst, "used", b.cx(), b.y+56, 15, colBone, 0.5)
		} else {
			coinIcon(dst, b.cx()-18, b.y+57, 8)
			say(dst, itoa(prices[i]), b.cx()+8, b.y+57, 16, colBone, 0.5)
		}
	}
	button(dst, g.leaveBtn(), btnArcane, "Leave", 28, false)
	g.drawHUD(dst, false)
}

// ---- campfire ----

func (g *Game) restBtn(i int) rect { return rect{60, 640 + float64(i)*104, 420, 88} }

func (g *Game) drawRest(dst *ebiten.Image) {
	r := g.run()
	t := g.t()
	g.drawMapUnder(dst)
	gradRect(dst, 0, 0, W, g.h, alpha(rgb(0x04, 0x04, 0x0c), 0.88), alpha(rgb(0x08, 0x04, 0x06), 0.95))
	for i := 0; i < 40; i++ {
		x := math.Mod(float64(i)*137.5, W)
		y := math.Mod(float64(i)*71.3, 300) + 110
		circle(dst, x, y, 1.2+math.Mod(float64(i), 3)*0.5, alpha(colWhite, 0.4+0.4*math.Sin(t*2+float64(i))))
	}
	ribbon(dst, W/2, 170, 320, "CAMPFIRE", rgb(0x7a, 0x30, 0x10))
	campfire(dst, W/2+70, 500, 1.5, t)
	drawHero(g.heroSpr.buf(), 150, 260, t, 0)
	g.heroSpr.stamp(dst, W/2-130, 520, 150, 260, 1.2, 1, 1, 1, 0)
	b0 := g.restBtn(0)
	button(dst, b0, btnGo, "", 0, false)
	say(dst, "Rest", b0.x+30, b0.y+28, 28, colBone, 0)
	drawText(dst, "Heal "+itoa(r.restHeal())+" HP", b0.x+30, b0.y+58, 16, colBone, 0)
	heartIcon(dst, b0.x+b0.w-50, b0.y+40, 20, colRed)
	b1 := g.restBtn(1)
	e := enhDefs[r.ForgeEnh]
	button(dst, b1, shade(e.Col, 0.85), "", 0, false)
	say(dst, "Forge", b1.x+30, b1.y+28, 28, colBone, 0)
	drawText(dst, "Make one card "+e.Name+": "+e.Desc, b1.x+30, b1.y+58, 15, colBone, 0)
	cardArt(dst, aHammer, b1.x+b1.w-50, b1.y+40, 22, t, 0)
	drawText(dst, "Skip", g.leaveBtn().cx(), g.leaveBtn().cy(), 20, alpha(colWhite, 0.7), 0.5)
	g.drawParticles(dst)
	g.drawHUD(dst, false)
}

// ---- mystery event ----

func (g *Game) eventBtn(i int) rect { return rect{44, 600 + float64(i)*92, 452, 76} }

func (g *Game) drawEvent(dst *ebiten.Image) {
	r := g.run()
	t := g.t()
	ev := events[r.Event]
	g.drawMapUnder(dst)
	fillRect(dst, 0, 0, W, g.h, color.RGBA{0, 0, 0, 0x90})
	p := rect{24, 124, 492, 720}
	panel(dst, p, rgb(0x3a, 0x4e, 0x96), rgb(0x1e, 0x26, 0x54), 20)
	art := rect{44, 144, 452, 230}
	roundRect(dst, art.grow(3), 12, colInk)
	sub := clip(dst, art)
	radial(sub, art.cx(), art.cy(), art.w*0.7, art.h, 0, -0.1, rgb(0x3a, 0x2a, 0x4a), rgb(0x06, 0x04, 0x0a))
	runeCircle(sub, art.cx(), art.cy(), 100, t, colPurple, 0.3)
	for i := 0; i < 12; i++ {
		a := float64(i)*math.Pi/6 + t*0.1
		shape(sub, []float64{art.cx(), art.cy(), art.cx() + math.Cos(a)*500, art.cy() + math.Sin(a)*500, art.cx() + math.Cos(a+0.18)*500, art.cy() + math.Sin(a+0.18)*500}, alpha(colWhite, 0.08), clear, 0)
	}
	eventArt(sub, r.Event, art.cx(), art.cy(), t)
	say(dst, ev.Title, W/2, 410, 32, colGoldHi, 0.5)
	drawText(dst, wrap(ev.Text, 40), W/2, 470, 19, colCream, 0.5)
	if g.eventMsg != "" {
		roundRect(dst, rect{44, 520, 452, 60}, 12, alpha(colInk, 0.5))
		say(dst, wrap(g.eventMsg, 36), W/2, 550, 19, colBone, 0.5)
		button(dst, g.eventBtn(0), btnGo, "Continue", 26, false)
		return
	}
	for i, ch := range ev.Choices {
		c := colBlue
		if i == len(ev.Choices)-1 && len(ev.Choices) > 1 && (ch == "Walk away" || ch == "No thanks") {
			c = colGrey
		}
		button(dst, g.eventBtn(i), c, ch, 22, false)
	}
}

// ---- card lists: pick a card, or look at the deck ----

const listCW = 112.0

func (g *Game) listCard(i int) rect {
	return rect{25 + float64(i%4)*(listCW+14), 166 + float64(i/4)*(listCW*1.4+16) - g.listScroll, listCW, listCW * 1.4}
}
func (g *Game) listMax() float64 {
	r := g.run()
	if r == nil {
		return 0
	}
	rows := (len(r.Deck) + 3) / 4
	return math.Max(0, float64(rows)*(listCW*1.4+16)+40-(g.h-290))
}
func (g *Game) closeBtn() rect { return rect{170, g.h - 100, 200, 64} }

func (g *Game) drawCardList(dst *ebiten.Image, title string) {
	r := g.run()
	t := g.t()
	fillRect(dst, 0, 0, W, g.h, color.RGBA{0x08, 0x0a, 0x1c, 0xe8})
	say(dst, title, W/2, 110, 26, colGoldHi, 0.5)
	area := rect{0, 150, W, g.h - 270}
	sub := clip(dst, area)
	for i, c := range r.Deck {
		cr := g.listCard(i)
		if cr.y > area.y+area.h || cr.y+cr.h < area.y {
			continue
		}
		drawCard(sub, c, cr.x, cr.y, cr.w, false, t)
	}
}

func (g *Game) drawPick(dst *ebiten.Image) {
	r := g.run()
	title := "Choose a card to remove"
	if g.pickWhy == pickForge {
		title = "Choose a card to make " + enhDefs[r.ForgeEnh].Name
	}
	g.drawCardList(dst, title)
	button(dst, g.closeBtn(), btnStone, "Back", 26, false)
}

func (g *Game) drawDeckView(dst *ebiten.Image) {
	g.drawCardList(dst, "Your deck: "+itoa(len(g.run().Deck))+" cards")
	button(dst, g.closeBtn(), btnArcane, "Close", 26, false)
}

// ---- end of a run ----

func (g *Game) endBtns() []rect {
	y := g.h/2 + 60
	if g.run().Lost && !g.run().Revived {
		return []rect{{40, y, 220, 80}, {280, y, 220, 80}, {160, y + 100, 220, 76}}
	}
	return []rect{{160, y, 220, 80}}
}

func (g *Game) drawEnd(dst *ebiten.Image) {
	r := g.run()
	fillRect(dst, 0, 0, W, g.h, color.RGBA{0, 0, 0, 0xb4})
	cy := g.h/2 - 200
	if r.Won {
		glow(dst, W/2, cy, 260, colYellow, 0.4)
		ribbon(dst, W/2, cy, 400, "CONQUERED", rgb(0x6a, 0x4a, 0x14))
	} else {
		ribbon(dst, W/2, cy, 340, "YOU DIED", rgb(0x4a, 0x08, 0x10))
	}
	say(dst, "Reached floor "+itoa(r.floor())+" of "+itoa(mapRows+1), W/2, cy+76, 24, colBone, 0.5)
	pr := rect{90, cy + 110, 360, 120}
	panel(dst, pr, rgb(0x3a, 0x4e, 0x96), rgb(0x22, 0x2c, 0x5c), 16)
	say(dst, "REWARDS", pr.cx(), pr.y+24, 20, colGoldHi, 0.5)
	coinIcon(dst, pr.cx()-100, pr.y+76, 18)
	say(dst, itoa(r.Gold), pr.cx()-74, pr.y+76, 30, colBone, 0)
	gemIcon(dst, pr.cx()+52, pr.y+76, 16)
	say(dst, itoa(r.Gems), pr.cx()+76, pr.y+76, 30, colBone, 0)
	bs := g.endBtns()
	if r.Lost && !r.Revived {
		button(dst, bs[0], btnArcane, "", 0, g.p.Gems < reviveGems)
		say(dst, "REVIVE", bs[0].cx(), bs[0].y+22, 24, colBone, 0.5)
		gemIcon(dst, bs[0].cx()-16, bs[0].y+50, 9)
		say(dst, itoa(reviveGems), bs[0].cx()+10, bs[0].y+50, 18, colBone, 0.5)
		button(dst, bs[1], btnGo, "", 0, false)
		say(dst, "REVIVE", bs[1].cx(), bs[1].y+22, 24, colBone, 0.5)
		say(dst, "watch ad", bs[1].cx(), bs[1].y+50, 16, colBone, 0.5)
		button(dst, bs[2], btnFire, "HOME", 28, false)
		return
	}
	button(dst, bs[0], btnFire, "HOME", 30, false)
}
