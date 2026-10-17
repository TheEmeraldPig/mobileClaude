package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- layout ----

func (g *Game) navTab(i int) rect { return rect{W/3*float64(i) + 10, g.h - 82, W/3 - 20, 68} }
func (g *Game) startBtn() rect    { return rect{100, g.h - 216, 340, 86} }
func (g *Game) abandonBtn() rect  { return rect{456, g.h - 202, 58, 58} }
func (g *Game) heroFeet() float64 { return math.Round(624 + (g.h-960)*0.45) }
func (g *Game) heroBtn() rect     { return rect{366, g.heroFeet() - 120, 150, 54} }
func (g *Game) chapCard() rect    { return rect{26, 92, 488, 300} }
func (g *Game) chapArrow(d int) rect {
	if d < 0 {
		return rect{30, 210, 44, 64}
	}
	return rect{W - 74, 210, 44, 64}
}
func (g *Game) shopTile(i int) rect {
	tw := (W - 36 - 16) / 2.0
	return rect{18 + float64(i%2)*(tw+16), 104 + float64(i/2)*186, tw, 168}
}
func (g *Game) slot(i int) rect { return rect{28 + float64(i)*98, 120, 88, 88 * 1.4} }
func (g *Game) colPanel() rect  { return rect{18, g.h - 340, W - 36, 236} }
func (g *Game) colArea() rect   { return rect{18, 316, W - 36, g.colPanel().y - 330} }

const colCW = 86.0

func (g *Game) colCard(i int) rect {
	a := g.colArea()
	return rect{38 + float64(i%5)*(colCW+8), a.y + 34 + float64(i/5)*(colCW*1.4+14) - g.colScroll, colCW, colCW * 1.4}
}
func (g *Game) colMax() float64 {
	rows := (len(g.p.Cards) + 4) / 5
	return math.Max(0, float64(rows)*(colCW*1.4+14)+42-g.colArea().h)
}
func (g *Game) colBtn(i int) rect {
	p := g.colPanel()
	bw := (p.w - 172 - 12) / 2
	return rect{p.x + 162 + float64(i%2)*(bw+12), p.y + 18 + float64(i/2)*70, bw, 54}
}

func (g *Game) clickHome(x, y float64) {
	for i := 0; i < 3; i++ {
		if g.navTab(i).has(x, y) {
			g.press("tab:" + itoa(i))
			return
		}
	}
	switch g.tab {
	case 0:
		for i := range shopItems {
			if g.shopTile(i).has(x, y) {
				g.press("buy:" + itoa(i))
			}
		}
	case 1:
		switch {
		case g.run() != nil && g.abandonBtn().has(x, y):
			g.press("abandon")
		case g.startBtn().has(x, y):
			g.press("start")
		case g.heroBtn().has(x, y):
			g.press("upgrade")
		case g.chapArrow(-1).has(x, y):
			g.press("chapter:-1")
		case g.chapArrow(1).has(x, y):
			g.press("chapter:1")
		}
	case 2:
		if g.sel >= 0 && g.colPanel().has(x, y) {
			for i := 0; i < 6; i++ {
				if g.colBtn(i).has(x, y) {
					if i == 0 {
						g.press("toggle")
					} else {
						g.press("enh:" + itoa(i))
					}
				}
			}
			return
		}
		n := 0
		for i, c := range g.p.Cards {
			if c.In {
				if g.slot(n).has(x, y) {
					g.press("sel:" + itoa(i))
					return
				}
				n++
			}
		}
		if g.colArea().has(x, y) {
			for i := range g.p.Cards {
				if g.colCard(i).has(x, y) {
					g.press("sel:" + itoa(i))
					return
				}
			}
		}
		g.sel = -1
	}
}

// ---- drawing ----

var romans = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

func roman(n int) string {
	if n >= 1 && n <= len(romans) {
		return romans[n-1]
	}
	return itoa(n)
}

func (g *Game) drawHome(dst *ebiten.Image) {
	t := g.t()
	homeScene(dst, W, g.h, t)
	switch g.tab {
	case 0:
		texRect(dst, 0, 0, W, g.h, alpha(rgb(0x14, 0x10, 0x1a), 0.85), alpha(rgb(0x06, 0x05, 0x0a), 0.95))
		g.drawHomeShop(dst, t)
	case 1:
		g.drawHomeBattle(dst, t)
	case 2:
		texRect(dst, 0, 0, W, g.h, alpha(rgb(0x14, 0x10, 0x1a), 0.85), alpha(rgb(0x06, 0x05, 0x0a), 0.95))
		g.drawCollection(dst, t)
	}
	g.drawParticles(dst)
	g.drawNav(dst, t)
	g.topBar(dst)
}

func (g *Game) drawNav(dst *ebiten.Image, t float64) {
	y := g.h - 100
	texRect(dst, 0, y, W, 100, rgb(0x22, 0x1e, 0x28), rgb(0x08, 0x07, 0x0c))
	gradRect(dst, 0, y-2, W, 4, colGoldLo, colGold)
	gradRect(dst, 0, y+2, W, 10, alpha(colInk, 0.6), clear)
	for i := 1; i < 3; i++ {
		x := W / 3 * float64(i)
		gradRect(dst, x-1, y+12, 2, 80, alpha(colGold, 0.4), clear)
	}
	for i, name := range []string{"SHOP", "BATTLE", "DECK"} {
		r := g.navTab(i)
		on := i == g.tab
		if on {
			r = rect{r.x - 4, r.y - 18, r.w + 8, r.h + 14}
			glow(dst, r.cx(), r.cy(), 90, colOrange, 0.25)
			frame(dst, r, rgb(0x4a, 0x2a, 0x22), colOrange, true)
		}
		ix, iy := r.cx(), r.y+26
		a := 0.55
		if on {
			a = 1
		}
		switch i {
		case 0:
			chestIcon(dst, ix, iy+6, 0.3, false)
		case 1:
			drawSword(dst, ix-6, iy+16, -0.6, 0.3)
			drawSword(dst, ix+6, iy+16, 0.6, 0.3)
		case 2:
			for k := 0; k < 3; k++ {
				cr := rect{ix - 16 + float64(k)*8, iy - 14 - float64(k%2)*2, 16, 24}
				p := chamfer(cr, 3)
				strokePath(dst, p, 2, colInk)
				texPath(dst, p, rgb(0x5a, 0x4a, 0x3a), rgb(0x22, 0x1a, 0x12))
				strokePath(dst, p, 1, colGold)
			}
		}
		if !on {
			fillRect(dst, r.x, r.y, r.w, r.h-20, alpha(rgb(0x08, 0x07, 0x0c), 1-a))
		}
		c := alpha(colBone, a)
		if on {
			c = colGoldHi
		}
		say(dst, name, r.cx(), r.y+r.h-18, 16, c, 0.5)
	}
}

func (g *Game) drawHomeBattle(dst *ebiten.Image, t float64) {
	ch := g.p.Chapter
	if r := g.run(); r != nil {
		ch = r.Chapter
	}
	// chapter card with a look at the arena and its lord
	pr := g.chapCard()
	frame(dst, pr, rgb(0x2a, 0x22, 0x2e), colRed, true)
	art := rect{pr.x + 14, pr.y + 56, pr.w - 28, 196}
	fillPath(dst, chamfer(art.grow(2), 8), colInk)
	sub := clip(dst, art)
	k := art.w / W
	sub.DrawImage(g.battleBG(ch), imgOp(art.x, art.y-(groundY-160)*k, k))
	drawEnemy(g.foeSpr.buf(), shDemon, 180, 330, t, chapterTheme(ch).Hue)
	g.foeSpr.stamp(sub, art.cx(), art.y+art.h-6, 180, 330, 0.56, 1, 1, 1, 0)
	gradRect(sub, art.x, art.y+art.h-60, art.w, 60, clear, alpha(colInk, 0.85))
	gradRect(sub, art.x, art.y, art.w, 50, alpha(colInk, 0.7), clear)
	strokePath(dst, chamfer(art.grow(1), 8), 1.5, colGold)
	say(dst, "CHAPTER "+roman(ch), pr.cx(), pr.y+30, 26, colGoldHi, 0.5)
	divider(dst, pr.cx(), pr.y+50, 260)
	say(dst, chapterName(ch), art.cx(), art.y+art.h-24, 26, colBone, 0.5)
	best := g.p.best(ch)
	label := "Deepest floor  " + itoa(best) + " / " + itoa(mapRows+1)
	if best > mapRows {
		label = "Conquered"
	}
	drawText(dst, label, pr.cx(), pr.y+pr.h-22, 17, colCream, 0.5)
	if g.run() == nil {
		if ch > 1 {
			button(dst, g.chapArrow(-1), rgb(0x3a, 0x34, 0x44), "<", 26, false)
		}
		if ch < g.p.MaxChapter {
			button(dst, g.chapArrow(1), rgb(0x3a, 0x34, 0x44), ">", 26, false)
		}
	}
	// the knight on a cliff
	hx, hf := W/2.0-50, g.heroFeet()
	cliff(dst, hx+30, hf, 170, t)
	glow(dst, hx, hf-100, 120, rgb(0xff, 0x60, 0x50), 0.12)
	drawHero(g.heroSpr.buf(), 150, 260, t, 0)
	g.heroSpr.stamp(dst, hx, hf, 150, 260, 1.2, 1, 1, 1, 0)
	// hero plate
	hb := g.heroBtn()
	plate := rect{hb.x - 6, hb.y - 92, hb.w + 12, 78}
	frame(dst, plate, rgb(0x22, 0x1e, 0x2a), colRed, false)
	say(dst, "KNIGHT", plate.cx(), plate.y+18, 15, colGoldHi, 0.5)
	drawText(dst, "Level "+itoa(g.p.HeroLv+1), plate.cx(), plate.y+38, 15, colCream, 0.5)
	heartIcon(dst, plate.cx()-26, plate.y+59, 8, colRed)
	say(dst, itoa(g.p.maxHP())+" HP", plate.cx()+10, plate.y+59, 15, colBone, 0.5)
	button(dst, hb, rgb(0x2e, 0x5a, 0x34), "", 0, g.p.Gold < g.p.heroCost())
	say(dst, "+6", hb.x+16, hb.cy(), 17, colBone, 0)
	coinIcon(dst, hb.x+hb.w-66, hb.cy(), 10)
	say(dst, itoa(g.p.heroCost()), hb.x+hb.w-12, hb.cy(), 17, colBone, 1)
	// descend or continue
	sb := g.startBtn()
	glow(dst, sb.cx(), sb.cy(), 200, colOrange, 0.18+0.06*math.Sin(t*2.5))
	if r := g.run(); r != nil {
		button(dst, sb, rgb(0x8a, 0x3a, 0x1a), "CONTINUE", 30, false)
		button(dst, g.abandonBtn(), rgb(0x6a, 0x14, 0x1c), "X", 22, false)
		drawText(dst, "Floor "+itoa(r.floor())+" of "+itoa(mapRows+1), sb.cx(), sb.y-16, 16, colCream, 0.5)
		return
	}
	button(dst, sb, rgb(0x9a, 0x40, 0x1a), "", 0, g.p.Energy < runCost)
	say(dst, "DESCEND", sb.cx()-24, sb.cy(), 30, colBone, 0.5)
	boltIcon(dst, sb.x+sb.w-64, sb.cy(), 14, colBlue)
	say(dst, itoa(runCost), sb.x+sb.w-30, sb.cy(), 24, colBone, 0.5)
}

func (g *Game) drawHomeShop(dst *ebiten.Image, t float64) {
	now := g.now()
	say(dst, "BLACK MARKET", W/2, 86, 22, colGoldHi, 0.5)
	divider(dst, W/2, 100, 300)
	for i, it := range shopItems {
		r := g.shopTile(i)
		fill, acc := rgb(0x2a, 0x24, 0x30), colOrange
		if it.Price[0] == '$' {
			fill, acc = rgb(0x30, 0x16, 0x2a), colGem
		}
		frame(dst, r, fill, acc, true)
		say(dst, it.Name, r.cx(), r.y+22, 16, colGoldHi, 0.5)
		cy := r.y + 66
		runeCircle(dst, r.cx(), cy, 34, t, acc, 0.25)
		switch {
		case it.Chest > 0:
			chestIcon(dst, r.cx(), cy+8, 0.5, it.Chest == 2)
		case it.ID == "energy" || it.ID == "ad":
			boltIcon(dst, r.cx(), cy, 26, colBlue)
		default:
			gemIcon(dst, r.cx()-20, cy+8, 13)
			gemIcon(dst, r.cx()+20, cy+8, 13)
			gemIcon(dst, r.cx(), cy-8, 17)
		}
		drawText(dst, it.Desc, r.cx(), r.y+110, 14, colCream, 0.5)
		br := rect{r.x + 18, r.y + r.h - 44, r.w - 36, 34}
		price, off := it.Price, false
		switch it.ID {
		case "free":
			if left := dailyGap - (now - g.p.FreeChestAt); left > 0 {
				price, off = itoa(int(left/3600))+"h "+itoa(int(left%3600/60))+"m", true
			}
		case "starter":
			if g.p.Bought["starter"] > 0 {
				price, off = "OWNED", true
			}
		}
		button(dst, br, rgb(0x2e, 0x5a, 0x34), "", 0, off)
		switch {
		case it.Gold > 0:
			coinIcon(dst, br.cx()-28, br.cy(), 9)
			say(dst, price, br.cx()+10, br.cy(), 17, colBone, 0.5)
		case it.Gems > 0:
			gemIcon(dst, br.cx()-26, br.cy(), 8)
			say(dst, price, br.cx()+10, br.cy(), 17, colBone, 0.5)
		default:
			say(dst, price, br.cx(), br.cy(), 17, colBone, 0.5)
		}
	}
	drawText(dst, "Real-money items are free in this test build", W/2, g.h-118, 13, colDim, 0.5)
}

func (g *Game) drawCollection(dst *ebiten.Image, t float64) {
	say(dst, "STARTING DECK", 28, 94, 18, colGoldHi, 0)
	deck := g.p.Loadout()
	say(dst, itoa(len(deck))+" / "+itoa(maxLoadout), W-28, 94, 18, colBone, 1)
	divider(dst, W/2, 108, W-60)
	n := 0
	for i, c := range g.p.Cards {
		if c.In {
			r := g.slot(n)
			lift := 0.0
			if i == g.sel {
				lift = 6
				glow(dst, r.cx(), r.cy(), 70, colOrange, 0.3)
			}
			drawCard(dst, Card{ID: c.ID, Enh: c.Enh}, r.x, r.y-lift, r.w, false, t)
			n++
		}
	}
	for ; n < maxLoadout; n++ {
		r := g.slot(n)
		fillPath(dst, chamfer(r, 8), alpha(colInk, 0.7))
		for k := 0.0; k < 1; k += 0.08 { // dashed gold outline
			strokePath(dst, pathOf([]float64{r.x + r.w*k, r.y, r.x + r.w*(k+0.04), r.y}), 1.2, alpha(colGold, 0.6))
			strokePath(dst, pathOf([]float64{r.x + r.w*k, r.y + r.h, r.x + r.w*(k+0.04), r.y + r.h}), 1.2, alpha(colGold, 0.6))
		}
		say(dst, "+", r.cx(), r.cy(), 30, alpha(colGold, 0.6), 0.5)
	}
	rr := rect{24, 256, W - 48, 44}
	fillPath(dst, chamfer(rr, 8), alpha(colInk, 0.7))
	strokePath(dst, chamfer(rr, 8), 1, alpha(colGold, 0.5))
	drawText(dst, "1 Legendary, or 2 Epics, or 3 Rares + 1 Uncommon, plus Commons.\nOr any mix of Commons and Uncommons.", W/2, rr.cy(), 14, colCream, 0.5)
	// collection
	a := g.colArea()
	frame(dst, a, rgb(0x1e, 0x1a, 0x24), colOrange, true)
	say(dst, "COLLECTION  "+itoa(len(g.p.Cards)), a.x+18, a.y+16, 13, colGold, 0)
	sub := clip(dst, rect{a.x, a.y + 28, a.w, a.h - 32})
	for i, c := range g.p.Cards {
		r := g.colCard(i)
		if r.y > a.y+a.h || r.y+r.h < a.y {
			continue
		}
		drawCard(sub, Card{ID: c.ID, Enh: c.Enh}, r.x, r.y, r.w, g.sel >= 0 && i != g.sel, t)
		if c.In {
			b := rect{r.x + r.w - 24, r.y + r.h - 26, 22, 18}
			fillPath(sub, chamfer(b.grow(1.5), 5), colInk)
			gradPath(sub, chamfer(b, 4), rgb(0x3a, 0x8a, 0x3a), rgb(0x14, 0x3a, 0x14))
			drawFont(sub, fTitle, "IN", b.cx(), b.cy(), 9, colBone, 0.5)
		}
	}
	if g.colMax() > 0 {
		f := g.colScroll / g.colMax()
		bh := a.h * a.h / (a.h + g.colMax())
		fillRect(dst, a.x+a.w-8, a.y+(a.h-bh)*f, 4, bh, alpha(colGold, 0.6))
	}
	// selected card
	p := g.colPanel()
	frame(dst, p, rgb(0x26, 0x20, 0x2c), colRed, true)
	if g.sel < 0 || g.sel >= len(g.p.Cards) {
		say(dst, "THE FORGE", p.cx(), p.y+28, 20, colGoldHi, 0.5)
		divider(dst, p.cx(), p.y+46, 240)
		drawText(dst, "Pick a card to take it into the depths or reforge it.\nEnhancements stay on the card for every run.", p.cx(), p.y+74, 14, colCream, 0.5)
		for i := 1; i < eCount; i++ {
			e := enhDefs[i]
			x := p.x + 20 + float64((i-1)/3)*244
			y := p.y + 104 + float64((i-1)%3)*40
			tr := rect{x, y, 64, 26}
			fillPath(dst, chamfer(tr.grow(1.5), 6), colInk)
			gradPath(dst, chamfer(tr, 5), shade(e.Col, 0.55), shade(e.Col, 0.22))
			strokePath(dst, chamfer(tr, 5), 1, e.Col)
			drawFont(dst, fTitle, e.Name, tr.cx(), tr.cy(), 12, light(e.Col, 0.4), 0.5)
			drawText(dst, e.Desc, x+74, tr.cy(), 14, colBone, 0)
		}
		return
	}
	c := g.p.Cards[g.sel]
	glow(dst, p.x+80, p.y+110, 80, rarityCol[c.def().Rarity], 0.25)
	drawCard(dst, Card{ID: c.ID, Enh: c.Enh}, p.x+18, p.y+20, 124, false, t)
	for i := 0; i < 6; i++ {
		r := g.colBtn(i)
		if i == 0 {
			if c.In {
				button(dst, r, rgb(0x6a, 0x16, 0x20), "Take out", 17, false)
			} else {
				button(dst, r, rgb(0x2e, 0x5a, 0x34), "Add to deck", 17, false)
			}
			continue
		}
		e := enhDefs[i]
		on := c.Enh == i
		button(dst, r, shade(e.Col, 0.55), "", 0, on)
		name := e.Name
		if on {
			name += " (on)"
		}
		say(dst, name, r.x+14, r.cy(), 15, light(e.Col, 0.5), 0)
		if e.Gems > 0 {
			gemIcon(dst, r.x+r.w-44, r.cy(), 7)
			say(dst, itoa(e.Gems), r.x+r.w-12, r.cy(), 15, colBone, 1)
		} else {
			coinIcon(dst, r.x+r.w-56, r.cy(), 8)
			say(dst, itoa(e.Gold), r.x+r.w-12, r.cy(), 15, colBone, 1)
		}
	}
	rc := rarityCol[c.def().Rarity]
	say(dst, rarityNames[c.def().Rarity], p.x+80, p.y+p.h-14, 13, light(rc, 0.3), 0.5)
}
