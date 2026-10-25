package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Panels are the pop-up screens: the genome (DNA lab), rebirth, a biote's mutations,
// the world (time, events and everything in effect), settings, the token shop and the
// welcome-back screen. The skill tree and challenge are full screens of their own.

const (
	genRowH  = 56.0 // a genome row, in units
	shopRowH = 46.0
)

func (g *Game) layPanel(L *layout, wu float64) {
	u := math.Max(g.u, 1)
	R := func(x, y, w, h float64) rect { return rect{x * u, y * u, w * u, h * u} }
	pw, ph := math.Min(wu-40, 470), 320.0
	if g.panel == "index" || g.panel == "shop" {
		pw = math.Min(wu-40, 580)
	}
	px, py := wu/2-pw/2, 20.0
	L.panel = R(px, py, pw, ph)
	L.close = R(px+pw-36, py+6, 28, 28)
	// list lays out n rows of height h in a scrolling view from y down to the
	// bottom of the panel, and returns where the first row goes
	list := func(y, h float64, n int) float64 {
		vh := py + ph - 22 - y
		L.view = R(px+12, y, pw-24-14, vh)
		L.bar = R(px+pw-12-10, y, 10, vh)
		L.scrollMax = math.Max(0, float64(n)*(h+6)-6-vh)
		g.scroll = math.Max(0, math.Min(g.scroll, L.scrollMax))
		return y - g.scroll
	}
	switch g.panel {
	case "index":
		// a row per found mutation, with a button for each lab upgrade
		var found []int
		for r := 1; r < nMut; r++ {
			if g.st.Found[r] > 0 {
				found = append(found, r)
			}
		}
		y0 := list(py+46, genRowH, len(found))
		rw := pw - 24 - 14
		bw := (rw - 180 - 2*8 - 8) / 3
		for k, r := range found {
			y := y0 + float64(k)*(genRowH+6)
			L.rows = append(L.rows, R(px+12, y, rw, genRowH))
			for kind := 0; kind < nLab; kind++ {
				L.btns = append(L.btns, R(px+12+180+float64(kind)*(bw+8), y+6, bw, genRowH-12))
			}
			L.rowMut = append(L.rowMut, r)
		}
	case "rebirth":
		if g.st.Money >= rebirthReq(g.st.Rebirths) {
			L.action = R(wu/2-80, py+ph-64, 160, 50)
		} else {
			// can't afford it: rebirth now for $0.99 or tokens
			L.action = R(wu/2-190, py+ph-64, 150, 50)
			L.action2 = R(wu/2-30, py+ph-64, 100, 50)
			L.action3 = R(wu/2+80, py+ph-64, 110, 50)
		}
	case "settings":
		L.action = R(px+pw/2-130, py+54, 260, 38)
		L.action2 = R(px+pw/2-130, py+98, 260, 38)
		L.action4 = R(px+pw/2-130, py+142, 260, 38)
		L.action3 = R(px+pw/2-110, py+ph-60, 220, 44)
		L.btns = []rect{R(px+pw/2-130, py+ph-60, 120, 44), R(px+pw/2+10, py+ph-60, 120, 44)} // YES, NO
	case "shop":
		tw := (pw - 24 - 3*6) / float64(len(shopTabs))
		for i := range shopTabs {
			L.tabs = append(L.tabs, R(px+12+float64(i)*(tw+6), py+44, tw, 24))
		}
		items := shopTab(g.shopTab)
		y0 := list(py+74, shopRowH, len(items))
		rw := pw - 24 - 14
		for k, i := range items {
			y := y0 + float64(k)*(shopRowH+6)
			L.rows = append(L.rows, R(px+12, y, rw, shopRowH))
			L.btns = append(L.btns, R(px+12+rw-8-104-6-72, y+7, 72, 32), R(px+12+rw-8-104, y+7, 104, 32)) // dollars, tokens
			L.rowMut = append(L.rowMut, i)
		}
	case "away", "visitor":
		L.action = R(wu/2-170, py+ph-70, 160, 50)
		L.action2 = R(wu/2+10, py+ph-70, 160, 50)
	}
}

func (g *Game) closePanel() {
	g.panel = ""
}

func (g *Game) tapPanel(L layout, x, y float64) {
	if L.close.in(x, y) || !L.panel.in(x, y) {
		if g.panel == "visitor" && g.offer.gift {
			g.giveBuff(g.offer.buff, 1) // closing a gift still takes it
		}
		g.closePanel()
		g.play("tap")
		return
	}
	inList := func(r rect) bool { return r.in(x, y) && (L.view.w == 0 || L.view.in(x, y)) }
	switch g.panel {
	case "rebirth":
		switch {
		case L.action.in(x, y):
			g.rebirth(false)
		case L.action2.in(x, y):
			g.buyItem("rebirth", true)
		case L.action3.in(x, y):
			g.buyItem("rebirth", false)
		}
	case "index":
		for i, b := range L.btns {
			if inList(b) {
				g.buyLab(i%nLab, L.rowMut[i/nLab])
				g.burst(x, y, 8, rgb(0x9ff0ff))
			}
		}
	case "settings":
		switch {
		case g.confirm && L.btns[0].in(x, y):
			g.resetSave()
		case g.confirm && L.btns[1].in(x, y):
			g.confirm = false
			g.play("tap")
		case L.action.in(x, y):
			g.st.NoMusic = !g.st.NoMusic
			g.updateMusic()
			g.play("tap")
		case L.action2.in(x, y):
			g.st.Sound = !g.st.Sound
			g.play("tap")
		case L.action4.in(x, y):
			g.st.Tut = 0 // replay the tutorial
			g.closePanel()
			g.play("tap")
		case !g.confirm && L.action3.in(x, y):
			g.confirm = true // it asks again before anything is deleted
			g.play("deny")
		}
	case "shop":
		for i, t := range L.tabs {
			if t.in(x, y) && i != g.shopTab {
				g.shopTab, g.scroll = i, 0
				g.play("tap")
			}
		}
		for i, b := range L.btns {
			if inList(b) {
				g.buyItem(shopItems[L.rowMut[i/2]].id, i%2 == 0)
			}
		}
	case "away":
		switch {
		case L.action.in(x, y):
			g.closePanel()
			g.play("tap")
		case L.action2.in(x, y) && !g.awayDoubled:
			gain := g.awayGain
			g.watchAd(func() {
				g.st.Money += gain
				g.st.Earned += gain
				g.awayDoubled = true
				g.soldT = g.t
				g.confetti(50)
				g.floats = append(g.floats, floater{s: "x2!  +$" + fmtNum(gain), x: g.W / 2, y: g.H * 0.45, c: colG, size: 30})
				g.closePanel()
			})
		}
	case "visitor":
		o := g.offer
		switch {
		case L.action.in(x, y) && o.gift: // take it
			g.giveBuff(o.buff, 1)
			g.closePanel()
		case L.action2.in(x, y) && o.gift: // double it
			g.closePanel()
			g.watchAd(func() { g.giveBuff(o.buff, 2) })
		case L.action.in(x, y): // watch for it
			g.closePanel()
			g.watchAd(func() { g.giveBuff(o.buff, 1) })
		case L.action2.in(x, y):
			g.closePanel()
			g.play("tap")
		}
	}
}

func (g *Game) drawPanel(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	fill(dst, 0, 0, g.W, g.H, color.NRGBA{10, 5, 25, 150})
	bevel(dst, L.panel, skDark, p)
	title := map[string]string{"index": "GENOME", "rebirth": "REBIRTH", "blob": "BIOTE", "world": "THE WORLD",
		"settings": "SETTINGS", "shop": "SHOP", "away": "WELCOME BACK", "visitor": "A VISITOR!"}[g.panel]
	say(dst, title, face(fBig, 22*u), L.panel.cx(), L.panel.y+22*u, alCenter, colT, math.Round(2*u))
	bevel(dst, L.close, skOrange, p)
	say(dst, "X", face(fBig, 16*u), L.close.cx(), L.close.cy()-u, alCenter, colW, math.Round(1.5*u))
	lx := L.panel.x + 16*u // left edge for text
	soft := rgb(0xd8ccff)

	switch g.panel {
	case "index":
		// the DNA lab: every found mutation, with its value, chance and spread upgrades
		icon(dst, "dna", L.panel.x+20*u, L.panel.y+22*u, 18*u)
		say(dst, itoa(g.st.Gems)+" DNA", face(fBig, 16*u), L.panel.x+32*u, L.panel.y+22*u, alLeft, rgb(0x9ff0ff), math.Round(1.5*u))
		clip := g.clipTo(dst, L.view)
		for k, row := range L.rows {
			if row.y > L.view.y+L.view.h || row.y+row.h < L.view.y {
				continue
			}
			r := L.rowMut[k]
			bevel(clip, row, skViolet, p)
			s := math.Max(1, math.Floor(44*u/blobPx))
			g.drawBlob(clip, 1<<r, row.x+28*u, row.cy(), s, s, 0)
			say(clip, rarities[r].Name, face(fBig, 20*u), row.x+54*u, row.y+18*u, alLeft, colW, math.Round(1.5*u))
			ch := g.gainChance(r)
			line := "x" + fmtNum(g.mutMult(r)) + " VALUE"
			if ch > 0 {
				line = "1/" + fmtNum(math.Round(1/ch)) + "   " + line
			}
			say(clip, line, face(fSmall, 8*u), row.x+54*u, row.y+40*u, alLeft, g.mutColor(r), math.Round(u))
			for kind := 0; kind < nLab; kind++ {
				b := L.btns[k*nLab+kind]
				lv := g.lab(kind, r)
				cost := labCost(r, lv)
				sk := skGrey
				if lv < labMax && g.st.Gems >= cost {
					sk = skGreen
				}
				b = g.pop(b)
				bevel(clip, b, sk, math.Max(1, p-1))
				say(clip, fmt.Sprintf("%s %d", labNames[kind], lv), face(fSmall, 8*u), b.cx(), b.y+13*u, alCenter, colW, math.Round(u))
				price := itoa(cost) + " DNA"
				if lv >= labMax {
					price = "MAX"
				}
				say(clip, price, face(fSmall, 8*u), b.cx(), b.y+b.h-12*u, alCenter, rgb(0x9ff0ff), math.Round(u))
			}
		}
		g.drawScrollBar(dst, L)
		if len(L.rows) == 0 {
			say(dst, "SPLIT TO FIND MUTATIONS!", face(fBig, 16*u), L.panel.cx(), L.panel.y+150*u, alCenter, soft, math.Round(1.5*u))
		}
		say(dst, fmt.Sprintf("FOUND %d/%d.  VALUE +25%%, CHANCE +20%%, SPREAD +50%% A LEVEL.  DNA COMES FROM REBIRTHS.", len(L.rows), nMut-1), face(fSmall, 6*u), L.panel.cx(), L.panel.y+L.panel.h-11*u, alCenter, soft, math.Round(u))

	case "rebirth":
		cx, y := L.panel.cx(), L.panel.y+58*u
		r := g.st.Rebirths
		pts, dna, tokens := g.rebirthRewards()
		say(dst, "x"+fmtNum(rebirthMult(r))+"  >  x"+fmtNum(rebirthMult(r+1)), face(fBig, 28*u), cx, y+8*u, alCenter, colT, math.Round(2*u))
		say(dst, "BIOTE VALUE MULTIPLIER", face(fSmall, 8*u), cx, y+32*u, alCenter, colW, math.Round(u))
		col := L.panel.w / 3
		for i, it := range []struct {
			ic, label string
			n         int
			c         color.RGBA
		}{{"star", "SKILL POINT", pts, colT}, {"dna", "DNA", dna, rgb(0x9ff0ff)}, {"token", "TOKENS", tokens, colT}} {
			x := L.panel.x + col*(float64(i)+0.5)
			icon(dst, it.ic, x, y+60*u, 18*u)
			say(dst, "+"+itoa(it.n), face(fBig, 18*u), x, y+82*u, alCenter, it.c, math.Round(1.5*u))
			say(dst, it.label, face(fSmall, 6*u), x, y+97*u, alCenter, soft, math.Round(u))
		}
		say(dst, "DNA: 1 PER FLOOR CLEARED THIS RUN.  TOKENS: 1 EACH TIME YOUR MONEY DOUBLES THE NEED.", face(fSmall, 6*u), cx, y+114*u, alCenter, soft, math.Round(u))
		say(dst, "NEEDS $"+fmtNum(rebirthReq(r)), face(fBig, 16*u), cx, y+134*u, alCenter, colG, math.Round(1.5*u))
		say(dst, "RESETS MONEY, BIOTES, UPGRADES AND YOUR FLOOR. KEEPS CARDS, DNA, TOKENS AND SKILLS.", face(fSmall, 6*u), cx, y+152*u, alCenter, soft, math.Round(u))
		ok := g.st.Money >= rebirthReq(r)
		sk := skGrey
		if ok {
			sk = skGold
		}
		ab := g.pop(L.action)
		bevel(dst, ab, sk, p)
		say(dst, "REBIRTH", face(fBig, 22*u), ab.cx(), ab.cy()-2*u, alCenter, colW, math.Round(2*u))
		if !ok {
			k := math.Min(1, g.st.Money/rebirthReq(r))
			bar := rect{L.action.x, L.action.y - 24*u, L.action3.x + L.action3.w - L.action.x, 8 * u}
			fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
			fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
			it := shopByID("rebirth")
			g.drawPriceButtons(dst, it, g.pop(L.action2), g.pop(L.action3), p, "REBIRTH NOW")
		}

	case "settings":
		onoff := func(b bool) string {
			if b {
				return "ON"
			}
			return "OFF"
		}
		for _, it := range []struct {
			r     rect
			label string
			on    bool
		}{{L.action, "MUSIC", !g.st.NoMusic}, {L.action2, "SOUND", g.st.Sound}} {
			sk := skGrey
			if it.on {
				sk = skGreen
			}
			bevel(dst, it.r, sk, p)
			say(dst, it.label+": "+onoff(it.on), face(fBig, 16*u), it.r.cx(), it.r.cy()-u, alCenter, colW, math.Round(1.5*u))
		}
		bevel(dst, g.pop(L.action4), skViolet, p)
		say(dst, "REPLAY TUTORIAL", face(fBig, 16*u), L.action4.cx(), L.action4.cy()-u, alCenter, colW, math.Round(1.5*u))
		say(dst, "VERSION "+Version, face(fSmall, 6*u), L.panel.cx(), L.action4.y+L.action4.h+14*u, alCenter, soft, math.Round(u))
		danger := skin{rgb(0xc8303a), rgb(0xff7a7a), rgb(0x701820)}
		if g.confirm {
			say(dst, "DELETE YOUR WHOLE SAVE? THIS CAN'T BE UNDONE.", face(fSmall, 8*u), L.panel.cx(), L.btns[0].y-16*u, alCenter, rgb(0xff8a8a), math.Round(u))
			bevel(dst, L.btns[0], danger, p)
			say(dst, "YES, RESET", face(fBig, 15*u), L.btns[0].cx(), L.btns[0].cy()-u, alCenter, colW, math.Round(1.5*u))
			bevel(dst, L.btns[1], skGrey, p)
			say(dst, "NO, KEEP IT", face(fBig, 15*u), L.btns[1].cx(), L.btns[1].cy()-u, alCenter, colW, math.Round(1.5*u))
		} else {
			bevel(dst, L.action3, danger, p)
			say(dst, "RESET SAVE", face(fBig, 16*u), L.action3.cx(), L.action3.cy()-u, alCenter, colW, math.Round(1.5*u))
		}

	case "shop":
		icon(dst, "token", L.panel.x+20*u, L.panel.y+22*u, 16*u)
		say(dst, itoa(g.st.Tokens)+" TOKENS", face(fBig, 16*u), L.panel.x+32*u, L.panel.y+22*u, alLeft, colT, math.Round(1.5*u))
		for i, t := range L.tabs {
			sk := skGrey
			if i == g.shopTab {
				sk = skGold
			}
			bevel(dst, t, sk, math.Max(1, p-1))
			say(dst, shopTabs[i], face(fSmall, 8*u), t.cx(), t.cy(), alCenter, colW, math.Round(u))
		}
		clip := g.clipTo(dst, L.view)
		for k, row := range L.rows {
			if row.y > L.view.y+L.view.h || row.y+row.h < L.view.y {
				continue
			}
			it := &shopItems[L.rowMut[k]]
			sk := skViolet
			if it.tab == tabBundles {
				sk = skin{rgb(0x8a3fb8), rgb(0xd88aff), rgb(0x4a1a6a)}
			}
			bevel(clip, row, sk, p)
			say(clip, it.name, face(fBig, 15*u), row.x+10*u, row.y+14*u, alLeft, colW, math.Round(1.5*u))
			say(clip, it.desc, face(fSmall, 6*u), row.x+10*u, row.y+32*u, alLeft, soft, math.Round(u))
			if it.tab == tabBundles && len(it.perks) > 1 {
				g.shimmer(clip, row, float64(k)*0.5)
			}
			b1, b2 := L.btns[2*k], L.btns[2*k+1]
			if g.owned(it) {
				bevel(clip, rect{b1.x, b1.y, b2.x + b2.w - b1.x, b1.h}, skDark, math.Max(1, p-1))
				say(clip, "OWNED", face(fBig, 14*u), (b1.x+b2.x+b2.w)/2, b1.cy()-u, alCenter, colT, math.Round(1.5*u))
				continue
			}
			g.drawPriceButtons(clip, it, g.pop(b1), g.pop(b2), math.Max(1, p-1), "")
		}
		g.drawScrollBar(dst, L)
		say(dst, "1000 TOKENS = $1.  EARN TOKENS BY REBIRTHING WITH MORE MONEY THAN YOU NEED.", face(fSmall, 6*u), L.panel.cx(), L.panel.y+L.panel.h-11*u, alCenter, soft, math.Round(u))

	case "away":
		cx := L.panel.cx()
		say(dst, "YOU WERE AWAY FOR "+fmtDur(g.away), face(fBig, 18*u), cx, L.panel.y+76*u, alCenter, colW, math.Round(1.5*u))
		bump := 1 + 0.06*math.Sin(g.t*4)
		say(dst, "+$"+fmtNum(g.awayGain), face(fBig, 34*u*bump), cx, L.panel.y+124*u, alCenter, colG, math.Round(2.5*u))
		why := "YOUR BIOTES KEPT WORKING WHILE YOU WERE AWAY (UP TO 8 HOURS)"
		if g.st.Rate > 0 {
			why = fmt.Sprintf("YOUR BIOTES KEPT WORKING AT %d%% OF YOUR RECENT INCOME (UP TO 8 HOURS)", int(math.Round(g.idleRate()/g.st.Rate*100)))
		}
		say(dst, why, face(fSmall, 7*u), cx, L.panel.y+166*u, alCenter, soft, math.Round(u))
		ab := g.pop(L.action)
		bevel(dst, ab, skGreen, p)
		say(dst, "COLLECT", face(fBig, 20*u), ab.cx(), ab.cy()-2*u, alCenter, colW, math.Round(2*u))
		ab = g.pop(L.action2)
		if !g.awayDoubled {
			ab = grow(ab, 1+0.04*math.Abs(math.Sin(g.t*4)))
		}
		bevel(dst, ab, skGold, p)
		say(dst, "x2  +$"+fmtNum(g.awayGain), face(fBig, 17*u), ab.cx(), ab.cy()-2*u, alCenter, colW, math.Round(1.5*u))
		g.adTag(dst, ab.cx(), ab.y)

	case "visitor":
		o := g.offer
		b := buffs[o.buff]
		cx := L.panel.cx()
		c := rgb(b.col)
		g.drawOrb(dst, o.buff, cx, L.panel.y+86*u, 2)
		say(dst, b.name, face(fBig, 26*u), cx, L.panel.y+136*u, alCenter, c, math.Round(2*u))
		what := b.desc
		if b.secs > 0 {
			what += " FOR " + fmtTime(b.secs)
		}
		say(dst, what, face(fSmall, 9*u), cx, L.panel.y+164*u, alCenter, colW, math.Round(u))
		hint := "IT'S A GIFT! TAKE IT, OR WATCH AN AD TO DOUBLE IT."
		l1, l2 := "TAKE IT", "x2 IT"
		if !o.gift {
			hint, l1, l2 = "WATCH AN AD TO GET IT.", "GET IT", "NO THANKS"
		}
		say(dst, hint, face(fSmall, 7*u), cx, L.panel.y+188*u, alCenter, soft, math.Round(u))
		a1, a2 := g.pop(L.action), g.pop(L.action2)
		if o.gift {
			bevel(dst, a1, skGreen, p)
			bevel(dst, grow(a2, 1+0.04*math.Abs(math.Sin(g.t*4))), skGold, p)
			g.adTag(dst, a2.cx(), a2.y)
		} else {
			bevel(dst, grow(a1, 1+0.04*math.Abs(math.Sin(g.t*4))), skGold, p)
			g.adTag(dst, a1.cx(), a1.y)
			bevel(dst, a2, skGrey, p)
		}
		say(dst, l1, face(fBig, 18*u), a1.cx(), a1.cy()-2*u, alCenter, colW, math.Round(1.5*u))
		say(dst, l2, face(fBig, 18*u), a2.cx(), a2.cy()-2*u, alCenter, colW, math.Round(1.5*u))

	case "blob":
		if g.sel >= len(g.blobs) {
			break
		}
		b := g.blobs[g.sel]
		s := math.Floor(90 * u / blobPx)
		cx, cy := L.panel.x+70*u, L.panel.y+110*u
		top := maskTop(b.m)
		if top > 0 {
			c := g.mutColor(top)
			vector.DrawFilledCircle(dst, float32(cx), float32(cy), float32(blobPx*s*0.62), color.NRGBA{c.R, c.G, c.B, 60}, true)
		}
		g.drawBlob(dst, b.m, cx, cy, s, s, 0)
		g.drawPips(dst, b.m, cx, cy+blobPx*s/2+4*u, math.Max(1, s/2))
		say(dst, "1/"+fmtNum(comboOdds(b.m)), face(fBig, 18*u), cx, L.panel.y+186*u, alCenter, g.mutColor(top), math.Round(1.5*u))
		say(dst, "ALL ITS MUTATIONS TOGETHER", face(fSmall, 6*u), cx, L.panel.y+200*u, alCenter, soft, math.Round(u))
		x0 := L.panel.x + 140*u
		x1 := L.panel.x + L.panel.w - 16*u
		y := L.panel.y + 56*u
		say(dst, "BASE VALUE", face(fSmall, 8*u), x0, y, alLeft, colW, math.Round(u))
		say(dst, "$"+fmtNum(g.baseValueAt(g.st.ValueLv)), face(fSmall, 8*u), x1, y, alRight, colG, math.Round(u))
		y += 22 * u
		list := maskList(b.m)
		if len(list) == 0 {
			say(dst, "NO MUTATIONS YET", face(fSmall, 8*u), x0, y, alLeft, soft, math.Round(u))
			y += 20 * u
		}
		for _, r := range list {
			say(dst, rarities[r].Name, face(fSmall, 8*u), x0, y, alLeft, g.mutColor(r), math.Round(u))
			say(dst, "1/"+fmtNum(rarities[r].Odds), face(fSmall, 8*u), x0+(x1-x0)*0.5, y, alCenter, colW, math.Round(u))
			say(dst, "x"+fmtNum(g.mutMult(r)), face(fSmall, 8*u), x1, y, alRight, colT, math.Round(u))
			y += 20 * u
		}
		if len(list) >= 2 && g.m.hybrid != 1 {
			say(dst, "HYBRID CARD", face(fSmall, 8*u), x0, y, alLeft, colW, math.Round(u))
			say(dst, "x"+fmtNum(g.m.hybrid), face(fSmall, 8*u), x1, y, alRight, colT, math.Round(u))
			y += 20 * u
		}
		fill(dst, x0, y-6*u, x1-x0, math.Max(1, u), soft)
		y += 6 * u
		say(dst, "VALUE", face(fBig, 16*u), x0, y, alLeft, colW, math.Round(1.5*u))
		say(dst, "$"+fmtNum(g.blobValue(b.m)), face(fBig, 16*u), x1, y, alRight, colG, math.Round(1.5*u))

	case "world":
		y := L.panel.y + 50*u
		x1 := L.panel.x + L.panel.w - 16*u
		for _, n := range g.m.notes {
			say(dst, n, face(fSmall, 8*u), lx, y, alLeft, colT, math.Round(u))
			y += 16 * u
		}
		say(dst, "LUCK x"+sprintf1(g.m.luck), face(fSmall, 8*u), x1, L.panel.y+50*u, alRight, colG, math.Round(u))
		y += 6 * u
		say(dst, "MUTATION", face(fSmall, 6*u), lx, y, alLeft, soft, math.Round(u))
		say(dst, "CHANCE NOW", face(fSmall, 6*u), lx+(x1-lx)*0.5, y, alCenter, soft, math.Round(u))
		say(dst, "VALUE", face(fSmall, 6*u), x1, y, alRight, soft, math.Round(u))
		y += 15 * u
		hidden := 0
		for r := 1; r < nMut; r++ {
			name := rarities[r].Name
			if g.st.Found[r] == 0 {
				hidden++ // like the catalog, only found mutations are shown
				continue
			}
			ch := g.gainChance(r)
			chs := "NONE"
			if ch > 0 {
				chs = "1/" + fmtNum(math.Round(1/ch))
			}
			say(dst, name, face(fSmall, 7*u), lx, y, alLeft, g.mutColor(r), math.Round(u))
			say(dst, chs, face(fSmall, 7*u), lx+(x1-lx)*0.5, y, alCenter, colW, math.Round(u))
			say(dst, "x"+fmtNum(g.mutMult(r)), face(fSmall, 7*u), x1, y, alRight, colT, math.Round(u))
			y += 15 * u
		}
		if hidden > 0 {
			say(dst, fmt.Sprintf("+%d MUTATIONS NOT FOUND YET", hidden), face(fSmall, 7*u), lx, y, alLeft, soft, math.Round(u))
		}
		say(dst, "DAY: GOLD & HEART UP.  NIGHT: SHADOW & COSMIC UP.  EVENTS COME EVERY FEW MINUTES.", face(fSmall, 6*u), L.panel.cx(), L.panel.y+L.panel.h-14*u, alCenter, soft, math.Round(u))

	}
}

// cardText writes a card's rarity and what it does, under its face.
func (g *Game) cardText(dst *ebiten.Image, c *cardDef, cx, y, w float64) {
	u := g.u
	say(dst, cardRarities[c.rar].name, face(fSmall, 6*u), cx, y, alCenter, cardRarities[c.rar].sk.l, math.Round(u))
	say(dst, c.desc, face(fSmall, 6*u), cx, y+14*u, alCenter, colW, math.Round(u))
}

// drawPriceButtons draws an item's two buy buttons: its dollar price and its token
// price (green when you have the tokens). above, if set, is written over them.
func (g *Game) drawPriceButtons(dst *ebiten.Image, it *shopItem, real, tok rect, p float64, above string) {
	u := g.u
	bevel(dst, real, skGold, p)
	say(dst, it.price(), face(fBig, 15*u), real.cx(), real.cy()-u, alCenter, colW, math.Round(1.5*u))
	sk := skGrey
	if g.st.Tokens >= it.tokens() {
		sk = skGreen
	}
	bevel(dst, tok, sk, p)
	icon(dst, "token", tok.x+11*u, tok.cy(), 12*u)
	say(dst, itoa(it.tokens()), face(fBig, 14*u), tok.x+20*u, tok.cy()-u, alLeft, colW, math.Round(1.5*u))
	if above != "" {
		say(dst, above, face(fSmall, 7*u), (real.x+tok.x+tok.w)/2, real.y-7*u, alCenter, colT, math.Round(u))
	}
}

// clipTo is dst cut down to r: drawing outside r is dropped.
func (g *Game) clipTo(dst *ebiten.Image, r rect) *ebiten.Image {
	return dst.SubImage(image.Rect(int(r.x), int(r.y), int(math.Ceil(r.x+r.w)), int(math.Ceil(r.y+r.h)))).(*ebiten.Image)
}

// drawScrollBar shows how far the panel's list is scrolled; it can be dragged.
func (g *Game) drawScrollBar(dst *ebiten.Image, L layout) {
	if L.scrollMax <= 0 {
		return
	}
	b := L.bar
	fill(dst, b.x, b.y, b.w, b.h, colO)
	fill(dst, b.x+g.u, b.y+g.u, b.w-2*g.u, b.h-2*g.u, rgb(0x17102f))
	k := L.view.h / (L.view.h + L.scrollMax*g.u)
	th := math.Max(16*g.u, b.h*k)
	ty := b.y + (b.h-th)*(g.scroll/L.scrollMax)
	bevel(dst, rect{b.x, ty, b.w, th}, skViolet, math.Max(1, math.Round(g.u)))
}
