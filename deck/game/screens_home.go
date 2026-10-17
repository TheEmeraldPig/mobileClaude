package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- layout ----

func (g *Game) navTab(i int) rect { return rect{W/3*float64(i) + 8, g.h - 80, W/3 - 16, 66} }
func (g *Game) startBtn() rect    { return rect{110, g.h - 214, 320, 90} }
func (g *Game) abandonBtn() rect  { return rect{448, g.h - 200, 64, 64} }
func (g *Game) heroFeet() float64 { return math.Round(600 + (g.h-960)*0.45) }
func (g *Game) heroBtn() rect     { return rect{360, g.heroFeet() - 128, 160, 58} }
func (g *Game) chapCard() rect    { return rect{24, 84, 492, 290} }
func (g *Game) chapArrow(d int) rect {
	if d < 0 {
		return rect{30, 196, 46, 66}
	}
	return rect{W - 76, 196, 46, 66}
}
func (g *Game) shopTile(i int) rect {
	tw := (W - 32 - 14) / 2.0
	return rect{16 + float64(i%2)*(tw+14), 96 + float64(i/2)*186, tw, 172}
}
func (g *Game) slot(i int) rect { return rect{26 + float64(i)*99, 112, 90, 90 * 1.4} }
func (g *Game) colPanel() rect  { return rect{16, g.h - 336, W - 32, 240} }
func (g *Game) colArea() rect   { return rect{16, 300, W - 32, g.colPanel().y - 310} }

const colCW = 86.0

func (g *Game) colCard(i int) rect {
	a := g.colArea()
	return rect{37 + float64(i%5)*(colCW+9), a.y + 8 + float64(i/5)*(colCW*1.4+12) - g.colScroll, colCW, colCW * 1.4}
}
func (g *Game) colMax() float64 {
	rows := (len(g.p.Cards) + 4) / 5
	return math.Max(0, float64(rows)*(colCW*1.4+12)+16-g.colArea().h)
}
func (g *Game) colBtn(i int) rect {
	p := g.colPanel()
	bw := (p.w - 168 - 10) / 2
	return rect{p.x + 158 + float64(i%2)*(bw+10), p.y + 16 + float64(i/2)*72, bw, 58}
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

func (g *Game) drawHome(dst *ebiten.Image) {
	t := g.t()
	homeSky(dst, W, g.h, t)
	switch g.tab {
	case 0:
		g.drawHomeShop(dst)
	case 1:
		g.drawHomeBattle(dst, t)
	case 2:
		g.drawCollection(dst, t)
	}
	g.drawNav(dst)
	g.topBar(dst)
}

func (g *Game) drawNav(dst *ebiten.Image) {
	gradRect(dst, 0, g.h-96, W, 96, rgb(0x24, 0x2c, 0x5c), rgb(0x10, 0x14, 0x30))
	fillRect(dst, 0, g.h-96, W, 3, colInk)
	for i, name := range []string{"SHOP", "BATTLE", "DECK"} {
		r := g.navTab(i)
		col := rgb(0x3d, 0x52, 0x9c)
		if i == g.tab {
			r = rect{r.x - 4, r.y - 16, r.w + 8, r.h + 12}
			col = colYellow
		}
		button(dst, r, col, "", 0, false)
		ix, iy := r.cx(), r.y+24
		switch i {
		case 0:
			chestIcon(dst, ix, iy+4, 0.3, false)
		case 1:
			swordArt(dst, ix-10, iy, 14, -0.75, colSteel)
			swordArt(dst, ix+10, iy, 14, 0.75, colSteel)
		case 2:
			roundRect(dst, rect{ix - 16, iy - 14, 20, 28}, 4, colInk)
			roundGrad(dst, rect{ix - 14, iy - 12, 16, 24}, 3, colBlue, shade(colBlue, 0.6))
			roundRect(dst, rect{ix - 4, iy - 12, 20, 28}, 4, colInk)
			roundGrad(dst, rect{ix - 2, iy - 10, 16, 24}, 3, colCream, rgb(0xe0, 0xc8, 0x98))
		}
		say(dst, name, r.cx(), r.y+r.h-20, 17, colWhite, 0.5)
	}
}

func (g *Game) drawHomeBattle(dst *ebiten.Image, t float64) {
	ch := g.p.Chapter
	if r := g.run(); r != nil {
		ch = r.Chapter
	}
	// chapter card with a peek at the arena
	pr := g.chapCard()
	panel(dst, pr, rgb(0x3a, 0x4e, 0x96), rgb(0x22, 0x2c, 0x5c), 20)
	art := rect{pr.x + 12, pr.y + 50, pr.w - 24, 190}
	bg := g.battleBG(ch)
	sub := clip(dst, art)
	k := art.w / W
	sub.DrawImage(bg, imgOp(art.x, art.y-(groundY-150)*k, k))
	drawEnemy(g.foeSpr.buf(), shKing, 180, 330, t, chapterTheme(ch).Hue)
	g.foeSpr.stamp(sub, art.cx(), art.y+art.h-8, 180, 330, 0.78, 1, 1, 1, 0)
	gradRect(sub, art.x, art.y+art.h-40, art.w, 40, clear, alpha(colInk, 0.5))
	strokePath(dst, roundPath(art, 6), 4, colInk)
	say(dst, "CHAPTER "+itoa(ch), pr.cx(), pr.y+26, 32, colYellow, 0.5)
	say(dst, chapterName(ch), art.cx(), art.y+22, 26, colWhite, 0.5)
	best := g.p.best(ch)
	label := "Best: floor " + itoa(best) + " / " + itoa(mapRows+1)
	if best > mapRows {
		label = "Cleared!"
	}
	say(dst, label, pr.cx(), pr.y+pr.h-26, 19, colCream, 0.5)
	if g.run() == nil {
		if ch > 1 {
			button(dst, g.chapArrow(-1), colBlue, "<", 30, false)
		}
		if ch < g.p.MaxChapter {
			button(dst, g.chapArrow(1), colBlue, ">", 30, false)
		}
	}
	// hero on a floating island
	hx, hf := W/2.0-30, g.heroFeet()
	island(dst, hx, hf, 130, t)
	drawHero(g.heroSpr.buf(), 130, 240, t, 0)
	g.heroSpr.stamp(dst, hx, hf+math.Sin(t*1.2)*4, 130, 240, 1.25, 1, 1, 1, 0)
	hb := g.heroBtn()
	say(dst, "Hero Lv "+itoa(g.p.HeroLv+1), hb.cx(), hb.y-40, 20, colWhite, 0.5)
	heartIcon(dst, hb.cx()-30, hb.y-14, 10, colRed)
	say(dst, itoa(g.p.maxHP())+" HP", hb.cx()+12, hb.y-14, 18, colWhite, 0.5)
	button(dst, hb, colGreen, "", 0, g.p.Gold < g.p.heroCost())
	say(dst, "+6", hb.x+16, hb.y+(hb.h-6)/2, 20, colWhite, 0)
	coinIcon(dst, hb.x+hb.w-72, hb.y+(hb.h-6)/2, 11)
	say(dst, itoa(g.p.heroCost()), hb.x+hb.w-12, hb.y+(hb.h-6)/2, 20, colWhite, 1)
	// start or continue
	sb := g.startBtn()
	if g.run() != nil {
		button(dst, sb, colGreen, "CONTINUE", 36, false)
		ab := g.abandonBtn()
		button(dst, ab, colRed, "X", 28, false)
		say(dst, "Floor "+itoa(g.run().floor()), sb.cx(), sb.y-18, 18, colWhite, 0.5)
		return
	}
	button(dst, sb, colYellow, "", 0, g.p.Energy < runCost)
	say(dst, "START", sb.cx()-26, sb.y+(sb.h-6)/2, 40, colWhite, 0.5)
	boltIcon(dst, sb.x+sb.w-66, sb.y+(sb.h-6)/2, 16, colGreen)
	say(dst, itoa(runCost), sb.x+sb.w-28, sb.y+(sb.h-6)/2, 28, colWhite, 0.5)
}

func (g *Game) drawHomeShop(dst *ebiten.Image) {
	now := g.now()
	for i, it := range shopItems {
		r := g.shopTile(i)
		top, bot := rgb(0x4a, 0x62, 0xb8), rgb(0x26, 0x34, 0x74)
		if it.Price[0] == '$' {
			top, bot = rgb(0x9a, 0x5a, 0xe8), rgb(0x4a, 0x24, 0x8a)
		}
		panel(dst, r, top, bot, 16)
		say(dst, it.Name, r.cx(), r.y+20, 20, colWhite, 0.5)
		cy := r.y + 70
		switch {
		case it.Chest > 0:
			glow(dst, r.cx(), cy, 60, colYellow, 0.25)
			chestIcon(dst, r.cx(), cy+4, 0.62, it.Chest == 2)
		case it.ID == "energy" || it.ID == "ad":
			glow(dst, r.cx(), cy, 50, colGreen, 0.4)
			boltIcon(dst, r.cx(), cy, 28, colGreen)
		default:
			glow(dst, r.cx(), cy, 50, colGem, 0.35)
			gemIcon(dst, r.cx()-20, cy+8, 16)
			gemIcon(dst, r.cx()+20, cy+8, 16)
			gemIcon(dst, r.cx(), cy-10, 20)
		}
		drawText(dst, it.Desc, r.cx(), r.y+112, 14, colCream, 0.5)
		br := rect{r.x + 14, r.y + r.h - 48, r.w - 28, 40}
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
		button(dst, br, colGreen, "", 0, off)
		by := br.y + (br.h-6)/2
		switch {
		case it.Gold > 0:
			coinIcon(dst, br.cx()-30, by, 11)
			say(dst, price, br.cx()+12, by, 20, colWhite, 0.5)
		case it.Gems > 0:
			gemIcon(dst, br.cx()-26, by, 10)
			say(dst, price, br.cx()+12, by, 20, colWhite, 0.5)
		default:
			say(dst, price, br.cx(), by, 20, colWhite, 0.5)
		}
	}
	say(dst, "Real-money items are free in this test build", W/2, g.h-112, 13, colWhite, 0.5)
}

func (g *Game) drawCollection(dst *ebiten.Image, t float64) {
	// starting deck
	say(dst, "STARTING DECK", 26, 90, 22, colWhite, 0)
	deck := g.p.Loadout()
	say(dst, itoa(len(deck))+" / "+itoa(maxLoadout), W-26, 90, 20, colYellow, 1)
	n := 0
	for i, c := range g.p.Cards {
		if c.In {
			r := g.slot(n)
			lift := 0.0
			if i == g.sel {
				lift = 6
			}
			drawCard(dst, Card{ID: c.ID, Enh: c.Enh}, r.x, r.y-lift, r.w, false, t)
			n++
		}
	}
	for ; n < maxLoadout; n++ {
		r := g.slot(n)
		roundRect(dst, r.grow(2), 12, alpha(colInk, 0.6))
		strokePath(dst, roundPath(r, 10), 2.5, alpha(colWhite, 0.5))
		say(dst, "+", r.cx(), r.cy(), 40, alpha(colWhite, 0.6), 0.5)
	}
	roundRect(dst, rect{20, 248, W - 40, 40}, 12, alpha(colInk, 0.55))
	drawText(dst, "Fits 1 Legendary, or 2 Epics, or 3 Rares + 1 Uncommon,\nplus any Commons. Or any mix of Commons and Uncommons.", W/2, 268, 13, colWhite, 0.5)
	// collection
	a := g.colArea()
	panel(dst, rect{a.x, a.y - 4, a.w, a.h + 8}, rgb(0x2e, 0x3c, 0x7a), rgb(0x1c, 0x24, 0x50), 16)
	sub := clip(dst, a)
	for i, c := range g.p.Cards {
		r := g.colCard(i)
		if r.y > a.y+a.h || r.y+r.h < a.y {
			continue
		}
		drawCard(sub, Card{ID: c.ID, Enh: c.Enh}, r.x, r.y, r.w, g.sel >= 0 && i != g.sel, t)
		if c.In {
			roundRect(sub, rect{r.x + r.w - 26, r.y + r.h - 30, 24, 24}, 12, colInk)
			roundGrad(sub, rect{r.x + r.w - 24, r.y + r.h - 28, 20, 20}, 10, colGreen, shade(colGreen, 0.6))
			drawText(sub, "IN", r.x+r.w-14, r.y+r.h-18, 10, colWhite, 0.5)
		}
	}
	if g.colMax() > 0 {
		f := g.colScroll / g.colMax()
		bh := a.h * a.h / (a.h + g.colMax())
		roundRect(dst, rect{a.x + a.w - 7, a.y + (a.h-bh)*f, 5, bh}, 3, alpha(colWhite, 0.5))
	}
	// selected card panel
	p := g.colPanel()
	panel(dst, p, rgb(0x3a, 0x4e, 0x96), rgb(0x22, 0x2c, 0x5c), 18)
	if g.sel < 0 || g.sel >= len(g.p.Cards) {
		say(dst, "COLLECTION", p.cx(), p.y+30, 26, colYellow, 0.5)
		drawText(dst, "Tap a card to add it to your starting deck or enhance it.\nEnhancements stay on the card for every run.", p.cx(), p.y+70, 14, colCream, 0.5)
		for i := 1; i < eCount; i++ {
			e := enhDefs[i]
			x := p.x + 18 + float64((i-1)/3)*244
			y := p.y + 102 + float64((i-1)%3)*42
			roundRect(dst, rect{x, y, 68, 30}, 9, colInk)
			roundGrad(dst, rect{x + 2, y + 2, 64, 26}, 7, light(e.Col, 0.3), shade(e.Col, 0.8))
			drawText(dst, e.Name, x+34, y+15, 15, colInk, 0.5)
			drawText(dst, e.Desc, x+76, y+15, 14, colWhite, 0)
		}
		return
	}
	c := g.p.Cards[g.sel]
	drawCard(dst, Card{ID: c.ID, Enh: c.Enh}, p.x+16, p.y+22, 128, false, t)
	for i := 0; i < 6; i++ {
		r := g.colBtn(i)
		ty := r.y + (r.h-6)/2
		if i == 0 {
			if c.In {
				button(dst, r, colRed, "Take out", 20, false)
			} else {
				button(dst, r, colGreen, "Add to deck", 20, false)
			}
			continue
		}
		e := enhDefs[i]
		on := c.Enh == i
		button(dst, r, shade(e.Col, 0.85), "", 0, on)
		name := e.Name
		if on {
			name += " (on)"
		}
		say(dst, name, r.x+14, ty, 18, colWhite, 0)
		if e.Gems > 0 {
			gemIcon(dst, r.x+r.w-46, ty, 9)
			say(dst, itoa(e.Gems), r.x+r.w-12, ty, 18, colWhite, 1)
		} else {
			coinIcon(dst, r.x+r.w-58, ty, 10)
			say(dst, itoa(e.Gold), r.x+r.w-12, ty, 18, colWhite, 1)
		}
	}
	rc := rarityCol[c.def().Rarity]
	drawText(dst, rarityNames[c.def().Rarity], p.x+80, p.y+p.h-14, 14, light(rc, 0.4), 0.5)
}
