package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Panels are the pop-up screens: skill tree, catalog, rebirth, market, a biote's
// mutations, a card, the world (time, events and everything in effect) and the round.

func (g *Game) layPanel(L *layout, wu float64) {
	u := math.Max(g.u, 1)
	R := func(x, y, w, h float64) rect { return rect{x * u, y * u, w * u, h * u} }
	pw, ph := math.Min(wu-40, 470), 320.0
	px, py := wu/2-pw/2, 20.0
	L.panel = R(px, py, pw, ph)
	L.close = R(px+pw-36, py+6, 28, 28)
	switch g.panel {
	case "index":
		cw, ch := (pw-24-3*8)/4, (ph-56-12-8)/2
		for i := range rarities {
			L.rows = append(L.rows, R(px+12+float64(i%4)*(cw+8), py+50+float64(i/4)*(ch+8), cw, ch))
		}
	case "rebirth":
		L.action = R(wu/2-80, py+ph-72, 160, 54)
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
	}
}

func (g *Game) drawPanel(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	fill(dst, 0, 0, g.W, g.H, color.NRGBA{10, 5, 25, 150})
	bevel(dst, L.panel, skDark, p)
	title := map[string]string{"tree": "SKILL TREE", "index": "CATALOG", "rebirth": "REBIRTH",
		"blob": "BIOTE", "world": "THE WORLD"}[g.panel]
	say(dst, title, face(fBig, 22*u), L.panel.cx(), L.panel.y+22*u, alCenter, colT, math.Round(2*u))
	bevel(dst, L.close, skOrange, p)
	say(dst, "X", face(fBig, 16*u), L.close.cx(), L.close.cy()-u, alCenter, colW, math.Round(1.5*u))
	lx := L.panel.x + 16*u // left edge for text
	soft := rgb(0xd8ccff)

	switch g.panel {
	case "index":
		found := 0
		k := 0
		for i := range rarities {
			if i > 0 && g.st.Found[i] == 0 {
				continue // only mutations you have found are shown
			}
			if i > 0 {
				found++
			}
			r := L.rows[k]
			k++
			ri := rarities[i]
			bevel(dst, r, skViolet, p)
			s := math.Floor(math.Min(r.w*0.5, r.h*0.45) / blobPx)
			g.drawBlob(dst, 1<<i&^1, r.cx(), r.y+6*u+blobPx*s/2, s, s, 0)
			ty := r.y + 10*u + blobPx*s
			say(dst, ri.Name, face(fBig, 13*u), r.cx(), ty, alCenter, colW, math.Round(1.5*u))
			odds := "PLAIN  x1"
			if i > 0 {
				odds = "1/" + fmtNum(ri.Odds) + "  x" + fmtNum(g.mutMult(i))
			}
			say(dst, odds, face(fSmall, 7*u), r.cx(), ty+14*u, alCenter, g.mutColor(i), math.Round(u))
			if i > 0 {
				say(dst, "GAINED "+fmtNum(float64(g.st.Found[i])), face(fSmall, 6*u), r.cx(), ty+25*u, alCenter, soft, math.Round(u))
			}
		}
		say(dst, fmt.Sprintf("FOUND %d/%d", found, nMut-1), face(fSmall, 8*u), lx, L.panel.y+22*u, alLeft, soft, math.Round(u))
		if found == 0 {
			say(dst, "SPLIT TO FIND MUTATIONS!", face(fBig, 16*u), L.panel.cx(), L.panel.y+250*u, alCenter, soft, math.Round(1.5*u))
		}

	case "rebirth":
		cx, y := L.panel.cx(), L.panel.y+62*u
		r := g.st.Rebirths
		say(dst, "x"+fmtNum(rebirthMult(r))+"  >  x"+fmtNum(rebirthMult(r+1)), face(fBig, 30*u), cx, y+10*u, alCenter, colT, math.Round(2*u))
		say(dst, "BIOTE VALUE MULTIPLIER", face(fSmall, 8*u), cx, y+36*u, alCenter, colW, math.Round(u))
		icon(dst, "star", cx-60*u, y+66*u, 14*u)
		say(dst, fmt.Sprintf("+%d SKILL POINTS", rebirthPoints(g.st.RunFloors)), face(fBig, 18*u), cx-48*u, y+66*u, alLeft, colT, math.Round(1.5*u))
		say(dst, "+1, AND +1 FOR EVERY FLOOR CLEARED THIS RUN", face(fSmall, 6*u), cx, y+84*u, alCenter, soft, math.Round(u))
		say(dst, "NEEDS $"+fmtNum(rebirthReq(r)), face(fBig, 16*u), cx, y+108*u, alCenter, colG, math.Round(1.5*u))
		say(dst, "RESETS MONEY, BIOTES, UPGRADES AND YOUR FLOOR. KEEPS CARDS, DNA AND SKILLS.", face(fSmall, 6*u), cx, y+128*u, alCenter, soft, math.Round(u))
		ok := g.st.Money >= rebirthReq(r)
		sk := skGrey
		if ok {
			sk = skGold
		}
		bevel(dst, L.action, sk, p)
		say(dst, "REBIRTH", face(fBig, 22*u), L.action.cx(), L.action.cy()-2*u, alCenter, colW, math.Round(2*u))
		if !ok {
			k := math.Min(1, g.st.Money/rebirthReq(r))
			bar := rect{L.action.x, L.action.y - 14*u, L.action.w, 8 * u}
			fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
			fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
		}

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
