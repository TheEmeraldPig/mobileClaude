package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Panels are the pop-up screens: the genome (DNA lab), rebirth, a biote's mutations,
// the world (time, events and everything in effect), settings, the token shop and the
// welcome-back screen. The skill tree and challenge are full screens of their own.

func (g *Game) layPanel(L *layout, wu float64) {
	u := math.Max(g.u, 1)
	R := func(x, y, w, h float64) rect { return rect{x * u, y * u, w * u, h * u} }
	pw, ph := math.Min(wu-40, 470), 320.0
	px, py := wu/2-pw/2, 20.0
	L.panel = R(px, py, pw, ph)
	L.close = R(px+pw-36, py+6, 28, 28)
	switch g.panel {
	case "index":
		// a row per found mutation, with a button for each lab upgrade
		k := 0
		bw := (pw - 24 - 150 - 2*6) / 3
		for r := 1; r < nMut; r++ {
			if g.st.Found[r] == 0 {
				continue
			}
			y := py + 50 + float64(k)*37
			L.rows = append(L.rows, R(px+12, y, pw-24, 34))
			for kind := 0; kind < nLab; kind++ {
				L.btns = append(L.btns, R(px+12+150+float64(kind)*(bw+6), y+3, bw, 28))
			}
			L.rowMut = append(L.rowMut, r)
			k++
		}
	case "rebirth":
		L.action = R(wu/2-80, py+ph-72, 160, 54)
	case "settings":
		L.action = R(px+pw/2-130, py+60, 260, 40)
		L.action2 = R(px+pw/2-130, py+108, 260, 40)
		L.action3 = R(px+pw/2-110, py+ph-60, 220, 44)
		L.btns = []rect{R(px+pw/2-130, py+ph-60, 120, 44), R(px+pw/2+10, py+ph-60, 120, 44)} // YES, NO
	case "shop":
		for i := range shopItems {
			L.rows = append(L.rows, R(px+12, py+70+float64(i)*46, pw-24, 42))
			L.btns = append(L.btns, R(px+pw-12-130, py+74+float64(i)*46, 122, 34))
		}
	case "away":
		L.action = R(wu/2-80, py+ph-72, 160, 50)
	}
}

func (g *Game) closePanel() {
	g.panel = ""
}

func (g *Game) tapPanel(L layout, x, y float64) {
	if L.close.in(x, y) || !L.panel.in(x, y) {
		g.closePanel()
		g.play("tap")
		return
	}
	switch g.panel {
	case "rebirth":
		if L.action.in(x, y) {
			g.rebirth()
		}
	case "index":
		for i, b := range L.btns {
			if b.in(x, y) {
				g.buyLab(i%nLab, L.rowMut[i/nLab])
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
		case !g.confirm && L.action3.in(x, y):
			g.confirm = true // it asks again before anything is deleted
			g.play("deny")
		}
	case "shop":
		for i, b := range L.btns {
			if b.in(x, y) {
				g.buyShop(i)
			}
		}
	case "away":
		if L.action.in(x, y) {
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
		"settings": "SETTINGS", "shop": "TOKEN SHOP", "away": "WELCOME BACK"}[g.panel]
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
		for k, row := range L.rows {
			r := L.rowMut[k]
			bevel(dst, row, skViolet, p)
			s := math.Max(1, math.Floor(26*u/blobPx))
			g.drawBlob(dst, 1<<r, row.x+18*u, row.cy(), s, s, 0)
			say(dst, rarities[r].Name, face(fBig, 13*u), row.x+34*u, row.y+11*u, alLeft, colW, math.Round(1.5*u))
			ch := g.gainChance(r)
			line := "x" + fmtNum(g.mutMult(r))
			if ch > 0 {
				line = "1/" + fmtNum(math.Round(1/ch)) + "  " + line
			}
			say(dst, line, face(fSmall, 6*u), row.x+34*u, row.y+25*u, alLeft, g.mutColor(r), math.Round(u))
			for kind := 0; kind < nLab; kind++ {
				b := L.btns[k*nLab+kind]
				lv := g.lab(kind, r)
				cost := labCost(r, lv)
				sk := skGrey
				if lv < labMax && g.st.Gems >= cost {
					sk = skGreen
				}
				bevel(dst, b, sk, math.Max(1, p-1))
				say(dst, fmt.Sprintf("%s %d", labNames[kind], lv), face(fSmall, 6*u), b.cx(), b.y+9*u, alCenter, colW, math.Round(u))
				price := itoa(cost) + " DNA"
				if lv >= labMax {
					price = "MAX"
				}
				say(dst, price, face(fSmall, 6*u), b.cx(), b.y+20*u, alCenter, rgb(0x9ff0ff), math.Round(u))
			}
		}
		if len(L.rows) == 0 {
			say(dst, "SPLIT TO FIND MUTATIONS!", face(fBig, 16*u), L.panel.cx(), L.panel.y+150*u, alCenter, soft, math.Round(1.5*u))
		}
		say(dst, fmt.Sprintf("FOUND %d/%d.  VALUE +25%%, CHANCE +20%%, SPREAD +50%% A LEVEL.  DNA COMES FROM REBIRTHS.", len(L.rows), nMut-1), face(fSmall, 6*u), L.panel.cx(), L.panel.y+L.panel.h-12*u, alCenter, soft, math.Round(u))

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
		bevel(dst, L.action, sk, p)
		say(dst, "REBIRTH", face(fBig, 22*u), L.action.cx(), L.action.cy()-2*u, alCenter, colW, math.Round(2*u))
		if !ok {
			k := math.Min(1, g.st.Money/rebirthReq(r))
			bar := rect{L.action.x, L.action.y - 12*u, L.action.w, 8 * u}
			fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
			fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
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
		say(dst, "VERSION "+Version, face(fSmall, 6*u), L.panel.cx(), L.action2.y+L.action2.h+16*u, alCenter, soft, math.Round(u))
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
		say(dst, "1000 TOKENS = $1.  EARN TOKENS BY REBIRTHING WITH MORE MONEY THAN YOU NEED.", face(fSmall, 6*u), L.panel.cx(), L.panel.y+48*u, alCenter, soft, math.Round(u))
		for i, it := range shopItems {
			r := L.rows[i]
			bevel(dst, r, skViolet, p)
			say(dst, it.name, face(fBig, 14*u), r.x+10*u, r.y+13*u, alLeft, colW, math.Round(1.5*u))
			say(dst, it.desc, face(fSmall, 6*u), r.x+10*u, r.y+29*u, alLeft, soft, math.Round(u))
			b := L.btns[i]
			owned := it.once && g.st.Perks[it.id]
			sk := skGrey
			if !owned && g.st.Tokens >= it.tokens() {
				sk = skGreen
			}
			bevel(dst, b, sk, math.Max(1, p-1))
			if owned {
				say(dst, "OWNED", face(fSmall, 8*u), b.cx(), b.cy(), alCenter, colT, math.Round(u))
			} else {
				say(dst, it.price()+" / "+itoa(it.tokens()), face(fSmall, 7*u), b.cx()+6*u, b.cy(), alCenter, colW, math.Round(u))
				icon(dst, "token", b.x+9*u, b.cy(), 11*u)
			}
		}
		say(dst, "REAL-MONEY PAYMENT ARRIVES WITH THE GOOGLE PLAY RELEASE; FOR NOW ITEMS ARE BOUGHT WITH TOKENS.", face(fSmall, 6*u), L.panel.cx(), L.panel.y+L.panel.h-12*u, alCenter, soft, math.Round(u))

	case "away":
		cx := L.panel.cx()
		say(dst, "YOU WERE AWAY FOR "+fmtDur(g.away), face(fBig, 18*u), cx, L.panel.y+80*u, alCenter, colW, math.Round(1.5*u))
		say(dst, "+$"+fmtNum(g.awayGain), face(fBig, 34*u), cx, L.panel.y+130*u, alCenter, colG, math.Round(2.5*u))
		say(dst, fmt.Sprintf("YOUR BIOTES KEPT WORKING AT %d%% OF YOUR RECENT INCOME (UP TO 8 HOURS)", int(math.Round(g.idleRate()/math.Max(g.st.Rate, 1e-9)*100))), face(fSmall, 7*u), cx, L.panel.y+172*u, alCenter, soft, math.Round(u))
		bevel(dst, L.action, skGold, p)
		say(dst, "COLLECT", face(fBig, 20*u), L.action.cx(), L.action.cy()-2*u, alCenter, colW, math.Round(2*u))

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
