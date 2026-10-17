// Package game is Deckbound: a card roguelite. Fight through 10-floor chapters with
// a deck of cards, pick rewards after each fight, and customise your deck for good
// between runs with enhancements, chests and removals.
package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DataDir is where the save goes. Android sets it to the app's files folder.
var DataDir = "."

// Version is set by the build.
var Version = "dev"

const W = 540 // logical screen width (portrait); height follows the screen's shape

type screen int

const (
	scHome screen = iota
	scBattle
	scReward
	scEnd
)

type floater struct {
	fx
	x, y float64
}

type Game struct {
	p                 *Profile
	w, h              float64
	sc                screen
	tab               int // home: 0 shop, 1 battle, 2 deck
	sel               int // deck card picked in the Deck tab, or -1
	r                 *Run
	rng               *rand.Rand
	frame             int
	toast             string
	toastT            int
	popup             string
	popCard           *Card
	fl                []floater
	heroHit, enemyHit float64
	lastHP, lastEHP   int
	now               func() int64
	seed              int64
	script            *script
}

func NewGame() *Game {
	g := &Game{w: W, h: 960, tab: 1, sel: -1, now: func() int64 { return time.Now().Unix() }, seed: time.Now().UnixNano()}
	g.p = newProfile(g.now())
	if b, err := readSave(); err == nil {
		p := newProfile(g.now())
		if json.Unmarshal(b, p) == nil {
			p.fix()
			g.p = p
		}
	}
	g.script = loadScript()
	if g.script != nil {
		g.seed = 1
	}
	g.rng = rand.New(rand.NewSource(g.seed))
	return g
}

// Save writes the profile to disk. Android calls it when the app is paused.
func (g *Game) Save() {
	b, _ := json.Marshal(g.p)
	writeSave(b)
}

func (g *Game) Layout(w, h int) (int, int) {
	g.w = W
	g.h = math.Max(960, math.Round(float64(h)*W/float64(max(w, 1))))
	return W, int(g.h)
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 130 }

func (g *Game) result(err error, ok string) {
	if err != nil {
		g.say(err.Error())
		return
	}
	if ok != "" {
		g.say(ok)
	}
	g.Save()
}

// ---- layout ----

func (g *Game) navTab(i int) rect { return rect{W/3*float64(i) + 8, g.h - 76, W/3 - 16, 62} }
func (g *Game) startBtn() rect    { return rect{110, g.h - 198, 320, 86} }
func (g *Game) heroY() float64    { return math.Round((420+g.h-198)/2 - 40) }
func (g *Game) heroBtn() rect     { return rect{155, g.heroY() + 132, 230, 58} }
func (g *Game) chapPanel() rect   { return rect{24, 76, 492, 330} }
func (g *Game) chapArrow(d int) rect {
	if d < 0 {
		return rect{44, 200, 48, 64}
	}
	return rect{W - 92, 200, 48, 64}
}
func (g *Game) shopTile(i int) rect {
	tw := (W - 32 - 14) / 2.0
	return rect{16 + float64(i%2)*(tw+14), 122 + float64(i/2)*184, tw, 170}
}
func (g *Game) deckPanel() rect { return rect{16, g.h - 330, W - 32, 228} }
func (g *Game) deckGrid() (cw, x0 float64, cols int) {
	gw := W - 32.0
	avail := g.deckPanel().y - 12 - 122
	cw = 92
	for {
		cols = max(1, int((gw+10)/(cw+10)))
		rows := (len(g.p.Deck) + cols - 1) / cols
		if float64(rows)*(cw*1.38+12) <= avail || cw < 40 {
			return cw, 16 + (gw-(float64(cols)*(cw+10)-10))/2, cols
		}
		cw -= 4
	}
}
func (g *Game) deckCard(i int) rect {
	cw, x0, cols := g.deckGrid()
	return rect{x0 + float64(i%cols)*(cw+10), 124 + float64(i/cols)*(cw*1.38+12), cw, cw * 1.38}
}
func (g *Game) enhBtn(i int) rect {
	p := g.deckPanel()
	bw := (p.w - 170 - 10) / 2
	return rect{p.x + 160 + float64(i%2)*(bw+10), p.y + 18 + float64(i/2)*68, bw, 54}
}
func (g *Game) endTurnBtn() rect { return rect{350, g.h - 268, 170, 64} }
func (g *Game) handCard(i, n int) rect {
	cw := 116.0
	x0, x1 := 12.0, W-12.0
	step := cw + 8
	if n > 1 {
		step = math.Min(step, (x1-x0-cw)/float64(n-1))
	}
	total := step*float64(n-1) + cw
	return rect{x0 + (x1-x0-total)/2 + step*float64(i), g.h - 182, cw, cw * 1.38}
}

// oy pushes the reward and result screens to the middle of tall screens.
func (g *Game) oy() float64           { return math.Round((g.h-960)/2 + 60) }
func (g *Game) rewardTile(i int) rect { return rect{16 + float64(i)*174, 200 + g.oy(), 160, 290} }
func (g *Game) endBtns() []rect {
	if g.r.Lost && !g.r.Revived {
		return []rect{{40, 430 + g.oy(), 220, 76}, {280, 430 + g.oy(), 220, 76}, {160, 532 + g.oy(), 220, 76}}
	}
	return []rect{{160, 430 + g.oy(), 220, 76}}
}

// ---- actions ----

func (g *Game) startRun() {
	g.p.tickEnergy(g.now())
	if g.p.Energy < runCost {
		g.say("Not enough energy: get more in the Shop")
		return
	}
	g.p.Energy -= runCost
	g.p.Runs++
	g.Save()
	g.seed++
	g.r = newRun(g.seed, g.p)
	g.sc = scBattle
	g.fl = nil
	g.lastHP, g.lastEHP = g.r.HP, g.r.B.EHP
}

// finishRun pays out the run and goes home.
func (g *Game) finishRun() {
	r := g.r
	g.p.Gold += r.Gold
	g.p.Gems += r.Gems
	k := fmt.Sprint(r.Chapter)
	floor := r.Floor
	if r.Lost {
		floor--
	}
	if floor > g.p.Best[k] {
		g.p.Best[k] = floor
	}
	if r.Won && r.Chapter == g.p.MaxChapter {
		g.p.MaxChapter++
		g.p.Chapter = g.p.MaxChapter
	}
	g.r = nil
	g.sc, g.tab = scHome, 1
	g.Save()
}

func (g *Game) press(id string) {
	name, arg, _ := strings.Cut(id, ":")
	n, _ := strconv.Atoi(arg)
	switch name {
	case "tab":
		g.tab, g.sel = n, -1
	case "start":
		g.startRun()
	case "upgrade":
		g.result(g.p.UpgradeHero(), fmt.Sprintf("Hero upgraded: %d HP", g.p.maxHP()))
	case "chapter":
		g.p.Chapter = max(1, min(g.p.MaxChapter, g.p.Chapter+n))
	case "buy":
		c, err := g.p.Buy(n, g.now(), g.rng)
		if err == nil && c != nil {
			g.popup, g.popCard = "NEW CARD!", c
		}
		g.result(err, "")
	case "sel":
		g.sel = n
	case "enh":
		if g.sel >= 0 {
			if n == eCount {
				g.result(g.p.Remove(g.sel), "Card removed")
				g.sel = -1
			} else {
				err := g.p.Enhance(g.sel, n)
				g.result(err, "")
				if err == nil {
					c := g.p.Deck[g.sel]
					g.popup, g.popCard = enhDefs[n].Name+"!", &c
					g.p.sortDeck()
					g.sel = -1
				}
			}
		}
	case "play":
		if g.sc == scBattle && n < len(g.r.B.Hand) {
			if err := g.r.B.Play(n); err != nil {
				g.say(err.Error())
			}
		}
	case "endturn":
		if g.sc == scBattle {
			g.r.B.EndTurn()
		}
	case "pick":
		if g.sc == scReward && n < len(g.r.Rewards) {
			g.r.Pick(n)
			g.sc = scBattle
			g.fl = nil
			g.lastHP, g.lastEHP = g.r.HP, g.r.B.EHP
		}
	case "revive", "ad":
		if name == "revive" {
			if g.p.Gems < reviveGems {
				g.say("Not enough gems: get more in the Shop")
				return
			}
			g.p.Gems -= reviveGems
			g.Save()
		}
		g.r.Revive()
		g.sc = scBattle
		g.fl = nil
		g.lastHP, g.lastEHP = g.r.HP, g.r.B.EHP
	case "home":
		g.finishRun()
	}
}

func (g *Game) click(x, y float64) {
	if g.popup != "" {
		g.popup, g.popCard = "", nil
		return
	}
	switch g.sc {
	case scHome:
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
			if g.sel >= 0 {
				for i := 1; i <= eCount; i++ {
					if g.enhBtn(i-1).has(x, y) {
						g.press("enh:" + itoa(i))
						return
					}
				}
			}
			for i := range g.p.Deck {
				if g.deckCard(i).has(x, y) {
					g.press("sel:" + itoa(i))
					return
				}
			}
			g.sel = -1
		}
	case scBattle:
		b := g.r.B
		if g.endTurnBtn().has(x, y) {
			g.press("endturn")
			return
		}
		for i := len(b.Hand) - 1; i >= 0; i-- {
			if g.handCard(i, len(b.Hand)).has(x, y) {
				g.press("play:" + itoa(i))
				return
			}
		}
		for i, c := range g.r.Charms {
			if math.Hypot(x-(W-34-float64(i)*44), y-98) < 20 {
				ch := charmByID(c)
				g.say(ch.Name + ": " + ch.Desc)
			}
		}
	case scReward:
		for i := range g.r.Rewards {
			if g.rewardTile(i).has(x, y) {
				g.press("pick:" + itoa(i))
			}
		}
	case scEnd:
		ids := []string{"home"}
		if g.r.Lost && !g.r.Revived {
			ids = []string{"revive", "ad", "home"}
		}
		for i, r := range g.endBtns() {
			if r.has(x, y) {
				g.press(ids[i])
			}
		}
	}
}

func (g *Game) Update() error {
	g.frame++
	if g.frame%60 == 0 {
		g.p.tickEnergy(g.now())
	}
	if g.script != nil {
		if err := g.script.step(g); err != nil {
			return err
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.click(float64(x), float64(y))
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		g.click(float64(x), float64(y))
	}
	if g.sc == scBattle && inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.press("endturn")
	}
	if g.r != nil && (g.sc == scBattle || g.sc == scReward || g.sc == scEnd) {
		b := g.r.B
		if g.r.HP < g.lastHP {
			g.heroHit = 1
		}
		if b.EHP < g.lastEHP {
			g.enemyHit = 1
		}
		g.lastHP, g.lastEHP = g.r.HP, b.EHP
		for _, f := range b.fx {
			x, y := 150.0, 470.0
			if f.Enemy {
				x, y = 300, 290
			}
			k := 0 // stack numbers that appear together
			for _, o := range g.fl {
				if o.Enemy == f.Enemy && o.T < 25 {
					k++
				}
			}
			g.fl = append(g.fl, floater{f, x + float64(k%2)*30 - 15, y - float64(k)*30})
		}
		b.fx = nil
		if g.sc == scBattle {
			switch {
			case g.r.Lost, g.r.Won:
				g.sc, g.toastT = scEnd, 0
			case g.r.Rewards != nil:
				g.sc, g.toastT = scReward, 0
			}
		}
	}
	k := g.fl[:0]
	for _, f := range g.fl {
		f.T++
		f.y -= 1.2
		if f.T < 55 {
			k = append(k, f)
		}
	}
	g.fl = k
	g.heroHit = math.Max(0, g.heroHit-0.06)
	g.enemyHit = math.Max(0, g.enemyHit-0.08)
	if g.toastT > 0 {
		g.toastT--
	}
	return nil
}

// ---- drawing ----

func (g *Game) t() float64 { return float64(g.frame) / 60 }

func (g *Game) Draw(dst *ebiten.Image) {
	switch g.sc {
	case scHome:
		gradRect(dst, 0, 0, W, g.h, colSky1, colSky2)
		g.drawClouds(dst)
		switch g.tab {
		case 0:
			g.drawShop(dst)
		case 1:
			g.drawHome(dst)
		case 2:
			g.drawDeck(dst)
		}
		g.drawNav(dst)
		g.drawTopBar(dst, []string{"SHOP", "", "DECK"}[g.tab])
	default:
		g.drawBattle(dst)
		switch g.sc {
		case scReward:
			g.drawReward(dst)
		case scEnd:
			g.drawEnd(dst)
		}
	}
	if g.toastT > 0 {
		a := math.Min(1, float64(g.toastT)/20)
		r := rect{20, 112, W - 40, 44}
		roundRect(dst, r, 22, alpha(color.RGBA{0x10, 0x14, 0x2a, 0xff}, 0.85*a))
		say(dst, g.toast, W/2, r.cy(), 18, alpha(colWhite, a), 0.5)
	}
	if g.popup != "" {
		vector.DrawFilledRect(dst, 0, 0, W, float32(g.h), color.RGBA{0, 0, 0, 0xb4}, false)
		cx, cy := W/2.0, g.h/2
		for i := 0; i < 12; i++ { // light rays
			a := g.t()*0.6 + float64(i)*math.Pi/6
			poly(dst, []float64{cx, cy, cx + math.Cos(a)*700, cy + math.Sin(a)*700, cx + math.Cos(a+0.18)*700, cy + math.Sin(a+0.18)*700}, alpha(colYellow, 0.12))
		}
		say(dst, g.popup, cx, cy-210, 44, colYellow, 0.5)
		if g.popCard != nil {
			drawCard(dst, *g.popCard, cx-100, cy-150, 200, false, g.t())
		}
		say(dst, "Tap to continue", cx, cy+190, 20, colWhite, 0.5)
	}
	if g.script != nil {
		g.script.afterDraw(dst)
	}
}

func (g *Game) drawClouds(dst *ebiten.Image) {
	for i := 0; i < 5; i++ {
		x := math.Mod(float64(i)*260+g.t()*8, W+300) - 150
		y := 150 + float64(i)*150
		c := alpha(colWhite, 0.18)
		ellipse(dst, x, y, 70, 22, c)
		ellipse(dst, x+30, y-14, 40, 22, c)
	}
}

func pill(dst *ebiten.Image, r rect, val string) {
	roundRect(dst, rect{r.x - 3, r.y - 3, r.w + 6, r.h + 6}, r.h/2+3, colInk)
	roundRect(dst, r, r.h/2, color.RGBA{0x14, 0x1c, 0x3c, 0xe0})
	say(dst, val, r.x+r.w-12, r.cy(), 18, colWhite, 1)
}

func (g *Game) drawTopBar(dst *ebiten.Image, title string) {
	now := g.now()
	er := rect{16, 14, 150, 34}
	pill(dst, er, fmt.Sprintf("%d/%d", g.p.Energy, maxEnergy))
	boltIcon(dst, er.x+6, er.cy(), 14, colGreen)
	if g.p.Energy < maxEnergy {
		s := g.p.nextEnergyIn(now)
		drawText(dst, fmt.Sprintf("+1 in %d:%02d", s/60, s%60), er.cx()+6, er.y+er.h+12, 13, colWhite, 0.5)
	}
	gr := rect{186, 14, 160, 34}
	pill(dst, gr, itoa(g.p.Gold))
	coinIcon(dst, gr.x+6, gr.cy(), 15)
	mr := rect{366, 14, 158, 34}
	pill(dst, mr, itoa(g.p.Gems))
	gemIcon(dst, mr.x+6, mr.cy(), 13)
	if title != "" {
		say(dst, title, 24, 94, 36, colWhite, 0)
	}
}

func (g *Game) drawNav(dst *ebiten.Image) {
	gradRect(dst, 0, g.h-88, W, 88, color.RGBA{0x1a, 0x24, 0x4c, 0xf0}, color.RGBA{0x0e, 0x14, 0x2e, 0xff})
	vector.DrawFilledRect(dst, 0, float32(g.h-88), W, 3, colInk, false)
	for i, name := range []string{"SHOP", "BATTLE", "DECK"} {
		r := g.navTab(i)
		col := colPanel2
		if i == g.tab {
			r.y -= 14
			r.h += 10
			col = colYellow
		}
		button(dst, r, col, "", 0, false)
		ix, iy := r.cx(), r.y+22
		switch i {
		case 0:
			chestIcon(dst, ix, iy, 0.3, 1)
		case 1:
			icon(dst, iSword, ix, iy-2, 28)
		case 2:
			icon(dst, iCards, ix+4, iy-2, 28)
		}
		say(dst, name, r.cx(), r.y+r.h-20, 17, colWhite, 0.5)
	}
}

func (g *Game) drawHome(dst *ebiten.Image) {
	t := g.t()
	// chapter card
	ch := g.p.Chapter
	pr := g.chapPanel()
	box(dst, pr, colPanel, 18)
	art := rect{pr.x + 14, pr.y + 54, pr.w - 28, 210}
	hc := hue(float64(ch)*0.17 + 0.25)
	gradRect(dst, art.x, art.y, art.w, art.h, shade(hc, 0.9), shade(hc, 0.45))
	ellipse(dst, art.cx(), art.y+art.h-14, art.w*0.36, 18, alpha(colInk, 0.35))
	drawEnemy(dst, boss, art.cx(), art.y+art.h-66, 0.95, t, 0)
	roundRect(dst, rect{art.x, art.y, art.w, 4}, 2, colInk)
	say(dst, fmt.Sprintf("CHAPTER %d", ch), pr.cx(), pr.y+28, 32, colYellow, 0.5)
	say(dst, chapterName(ch), art.cx(), art.y+24, 26, colWhite, 0.5)
	best := g.p.best(ch)
	label := fmt.Sprintf("Best: floor %d / %d", best, floorsPerChapter)
	if best >= floorsPerChapter {
		label = "Cleared!"
	}
	say(dst, label, pr.cx(), pr.y+pr.h-34, 20, colCream, 0.5)
	if ch > 1 {
		button(dst, g.chapArrow(-1), colBlue, "<", 30, false)
	}
	if ch < g.p.MaxChapter {
		button(dst, g.chapArrow(1), colBlue, ">", 30, false)
	}

	// hero on a stone pedestal
	hx, hy := W/2.0, g.heroY()
	ellipse(dst, hx, hy+76, 105, 30, colInk)
	ellipse(dst, hx, hy+72, 100, 26, color.RGBA{0x8a, 0x94, 0xb0, 0xff})
	ellipse(dst, hx, hy+66, 92, 20, color.RGBA{0xb6, 0xc0, 0xd8, 0xff})
	drawHero(dst, hx, hy, 1.4, t, 0)
	say(dst, fmt.Sprintf("Hero Lv %d   %d HP", g.p.HeroLv+1, g.p.maxHP()), hx, hy+114, 20, colWhite, 0.5)
	hb := g.heroBtn()
	button(dst, hb, colGreen, "", 0, g.p.Gold < g.p.heroCost())
	say(dst, "+6 HP", hb.x+18, hb.y+(hb.h-6)/2, 20, colWhite, 0)
	coinIcon(dst, hb.x+hb.w-90, hb.y+(hb.h-6)/2, 12)
	say(dst, itoa(g.p.heroCost()), hb.x+hb.w-16, hb.y+(hb.h-6)/2, 20, colWhite, 1)

	sb := g.startBtn()
	button(dst, sb, colYellow, "", 0, g.p.Energy < runCost)
	say(dst, "START", sb.cx()-24, sb.y+(sb.h-6)/2, 38, colWhite, 0.5)
	boltIcon(dst, sb.x+sb.w-62, sb.y+(sb.h-6)/2, 14, colGreen)
	say(dst, itoa(runCost), sb.x+sb.w-26, sb.y+(sb.h-6)/2, 26, colWhite, 0.5)
}

func (g *Game) drawShop(dst *ebiten.Image) {
	now := g.now()
	for i, it := range shopItems {
		r := g.shopTile(i)
		c := colPanel2
		if it.Price[0] == '$' {
			c = color.RGBA{0x7a, 0x4c, 0xc8, 0xff}
		}
		box(dst, r, c, 16)
		say(dst, it.Name, r.cx(), r.y+20, 20, colWhite, 0.5)
		cy := r.y + 64
		switch {
		case it.Chest > 0:
			chestIcon(dst, r.cx(), cy+6, 0.7, it.Chest)
		case it.ID == "energy" || it.ID == "ad":
			boltIcon(dst, r.cx(), cy, 30, colGreen)
		default:
			gemIcon(dst, r.cx()-18, cy+8, 18)
			gemIcon(dst, r.cx()+18, cy+8, 18)
			gemIcon(dst, r.cx(), cy-12, 22)
		}
		drawText(dst, it.Desc, r.cx(), r.y+110, 14, colCream, 0.5)
		br := rect{r.x + 14, r.y + r.h - 48, r.w - 28, 40}
		price := it.Price
		off := false
		switch it.ID {
		case "free":
			if left := dailyGap - (now - g.p.FreeChestAt); left > 0 {
				price, off = fmt.Sprintf("%dh %dm", left/3600, left%3600/60), true
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
	drawText(dst, "Test build: real-money items are free\nuntil Google Play billing is added", 140, 94, 13, colWhite, 0)
}

func (g *Game) drawDeck(dst *ebiten.Image) {
	t := g.t()
	for i, c := range g.p.Deck {
		r := g.deckCard(i)
		lift := 0.0
		if i == g.sel {
			lift = 8
		}
		drawCard(dst, c, r.x, r.y-lift, r.w, g.sel >= 0 && i != g.sel, t)
	}
	say(dst, fmt.Sprintf("%d cards", len(g.p.Deck)), 146, 96, 20, colCream, 0)
	p := g.deckPanel()
	box(dst, p, colPanel, 18)
	if g.sel < 0 || g.sel >= len(g.p.Deck) {
		say(dst, "CUSTOMIZE", p.cx(), p.y+28, 28, colYellow, 0.5)
		drawText(dst, "Tap a card to enhance or remove it. Enhancements stay for every run.", p.cx(), p.y+60, 14, colCream, 0.5)
		for i := 1; i < eCount; i++ {
			e := enhDefs[i]
			x := p.x + 16 + float64((i-1)/3)*244
			y := p.y + 84 + float64((i-1)%3)*44
			roundRect(dst, rect{x, y, 66, 30}, 8, colInk)
			roundRect(dst, rect{x + 2, y + 2, 62, 26}, 6, e.Col)
			drawText(dst, e.Name, x+33, y+15, 15, colInk, 0.5)
			drawText(dst, e.Desc, x+74, y+15, 14, colWhite, 0)
		}
		return
	}
	drawCard(dst, g.p.Deck[g.sel], p.x+14, p.y+18, 136, false, t)
	for i := 1; i <= eCount; i++ {
		r := g.enhBtn(i - 1)
		ty := r.y + (r.h-6)/2
		if i == eCount {
			button(dst, r, colRed, "", 0, len(g.p.Deck) <= minDeck)
			say(dst, "Remove", r.x+14, ty, 18, colWhite, 0)
			coinIcon(dst, r.x+r.w-62, ty, 10)
			say(dst, itoa(removeCost), r.x+r.w-12, ty, 18, colWhite, 1)
			continue
		}
		e := enhDefs[i]
		on := g.p.Deck[g.sel].Enh == i
		button(dst, r, shade(e.Col, 0.85), "", 0, on)
		name := e.Name
		if on {
			name += " (on)"
		}
		say(dst, name, r.x+14, ty, 18, colWhite, 0)
		if e.Gems > 0 {
			gemIcon(dst, r.x+r.w-50, ty, 9)
			say(dst, itoa(e.Gems), r.x+r.w-12, ty, 18, colWhite, 1)
		} else {
			coinIcon(dst, r.x+r.w-62, ty, 10)
			say(dst, itoa(e.Gold), r.x+r.w-12, ty, 18, colWhite, 1)
		}
	}
}

// ---- battle ----

func (g *Game) drawBattle(dst *ebiten.Image) {
	t := g.t()
	r := g.r
	b := r.B
	w := g.w
	// arena: sky, back wall, tiled floor
	h := g.h
	hc := hue(float64(r.Chapter)*0.17 + 0.25)
	gradRect(dst, 0, 0, w, 250, shade(hc, 1.0), shade(hc, 0.7))
	gradRect(dst, 0, 250, w, h-250, color.RGBA{0x9c, 0x8a, 0x70, 0xff}, color.RGBA{0x6c, 0x5c, 0x4a, 0xff})
	for x := -40.0; x < w+40; x += 70 {
		vector.StrokeLine(dst, float32(x), 250, float32(w/2+(x-w/2)*2.2), float32(h), 2, alpha(colInk, 0.18), true)
	}
	for _, y := range []float64{285, 340, 420, 530, 680} {
		vector.StrokeLine(dst, 0, float32(y), float32(w), float32(y), 2, alpha(colInk, 0.18), true)
	}
	vector.DrawFilledRect(dst, 0, 246, float32(w), 6, alpha(colInk, 0.5), false)

	hx, hy := 150.0, 570.0
	ex, ey := 300.0, 330.0
	drawHero(dst, hx, hy, 1.15, t, g.heroHit)
	es := 1.3
	switch r.Floor {
	case 5:
		es = 1.45
	case floorsPerChapter:
		es = 1.65
	}
	if b.EHP > 0 {
		drawEnemy(dst, b.E, ex, ey, es, t, g.enemyHit)
	}
	// health, block and statuses
	hpBar(dst, hx, 636, 170, r.HP, r.Max, colGreen)
	hpBar(dst, ex, 405, 200, max(0, b.EHP), b.EMax, colRed)
	blockBadge(dst, hx-112, 647, b.Block)
	blockBadge(dst, ex-128, 416, b.EBlock)
	statusRow(dst, hx, 678, []status{{"Vulnerable", b.Vuln, colOrange}, {"Weak", b.Weak, colGrey2}})
	statusRow(dst, ex, 446, []status{{"Vulnerable", b.EVuln, colOrange}, {"Weak", b.EWeak, colGrey2}, {"Poison", b.EPois, colGreen2}, {"Strength", b.EStr, colRed2}})
	say(dst, b.E.Name, ex, 474, 20, colWhite, 0.5)
	// what the enemy will do next
	if b.EHP > 0 && !r.Lost {
		iy := ey - 95*es
		m := b.Intent
		ib := rect{ex - 46, iy - 22, 92, 40}
		roundRect(dst, rect{ib.x - 3, ib.y - 3, ib.w + 6, ib.h + 6}, 22, colInk)
		roundRect(dst, ib, 20, colCream)
		switch m.Kind {
		case mAttack:
			icon(dst, iSword, ib.x+22, ib.cy(), 26)
			v := itoa(b.IntentDamage())
			if m.Hits > 1 {
				v += "x" + itoa(m.Hits)
			}
			say(dst, v, ib.x+66, ib.cy(), 20, colRed, 0.5)
		case mBlock:
			icon(dst, iShield, ib.x+22, ib.cy(), 24)
			say(dst, itoa(m.Val), ib.x+66, ib.cy(), 20, colBlue, 0.5)
		case mBuff:
			say(dst, "BUFF", ib.cx(), ib.cy(), 18, colOrange, 0.5)
		case mDebuff:
			icon(dst, iSkull, ib.x+22, ib.cy(), 24)
			say(dst, "HEX", ib.x+62, ib.cy(), 18, colPurple, 0.5)
		}
	}
	// top bar: run gold, floor dots, charms
	gr := rect{16, 14, 120, 34}
	pill(dst, gr, itoa(r.Gold))
	coinIcon(dst, gr.x+6, gr.cy(), 15)
	say(dst, fmt.Sprintf("FLOOR %d", r.Floor), w/2, 26, 24, colWhite, 0.5)
	for i := 1; i <= floorsPerChapter; i++ {
		x := w/2 + (float64(i)-5.5)*22
		c := alpha(colInk, 0.5)
		switch {
		case i < r.Floor:
			c = colYellow
		case i == r.Floor:
			c = colWhite
		}
		rad := 6.0
		if i == 5 || i == floorsPerChapter {
			rad = 8
		}
		circle(dst, x, 52, rad+2, colInk)
		circle(dst, x, 52, rad, c)
	}
	for i, c := range r.Charms {
		x := w - 34 - float64(i)*44
		ball(dst, x, 98, 18, hue(float64(len(c))*0.13+float64(c[0])*0.05))
		say(dst, string(c[0]-32), x, 97, 20, colWhite, 0.5)
	}
	// hand area
	gradRect(dst, 0, h-320, w, 320, color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 0, 0x90})
	ball(dst, 432, 600, 44, colOrange)
	circle(dst, 432, 600, 34, colYellow)
	say(dst, fmt.Sprintf("%d/3", b.Energy), 432, 598, 30, colWhite, 0.5)
	drawText(dst, fmt.Sprintf("Draw %d   Used %d", len(b.Draw), len(b.Discard)), 432, 664, 14, colWhite, 0.5)
	for i, c := range b.Hand {
		hr := g.handCard(i, len(b.Hand))
		drawCard(dst, c, hr.x, hr.y, hr.w, c.stats().Cost > b.Energy, t)
	}
	button(dst, g.endTurnBtn(), colGreen, "END TURN", 22, false)
	// floating numbers
	for _, f := range g.fl {
		a := math.Min(1, float64(55-f.T)/15)
		c := colWhite
		size := 34.0
		switch f.Kind {
		case 0:
			if !f.Enemy {
				c = colRed
			}
		case 1:
			c = colBlue
		case 2:
			c = colGreen
		case 3:
			c, size = colYellow, 24
		}
		say(dst, f.S, f.x, f.y, size*(1+0.3*math.Max(0, 1-float64(f.T)/8)), alpha(c, a), 0.5)
	}
}

type status struct {
	name string
	n    int
	c    color.RGBA
}

func statusRow(dst *ebiten.Image, cx, y float64, ss []status) {
	var on []status
	for _, s := range ss {
		if s.n > 0 {
			on = append(on, s)
		}
	}
	x := cx - float64(len(on))*58/2
	for _, s := range on {
		roundRect(dst, rect{x, y - 11, 54, 22}, 10, colInk)
		roundRect(dst, rect{x + 2, y - 9, 50, 18}, 8, s.c)
		drawText(dst, s.name[:4]+" "+itoa(s.n), x+27, y, 13, colWhite, 0.5)
		x += 58
	}
}

func blockBadge(dst *ebiten.Image, x, y float64, n int) {
	if n <= 0 {
		return
	}
	icon(dst, iShield, x, y, 34)
	say(dst, itoa(n), x, y-2, 18, colWhite, 0.5)
}

func (g *Game) drawReward(dst *ebiten.Image) {
	t := g.t()
	vector.DrawFilledRect(dst, 0, 0, W, float32(g.h), color.RGBA{0, 0, 0, 0xa8}, false)
	ribbon(dst, W/2, 110+g.oy(), 360, "VICTORY!", colYellow)
	say(dst, "Choose one", W/2, 164+g.oy(), 24, colWhite, 0.5)
	for i, w := range g.r.Rewards {
		r := g.rewardTile(i)
		box(dst, r, colPanel2, 18)
		title, desc := "", ""
		switch w.Kind {
		case rCard:
			title = "New Card"
			drawCard(dst, w.Card, r.cx()-60, r.y+50, 120, false, t)
		case rEnhance:
			title = enhDefs[w.Card.Enh].Name + " Upgrade"
			drawCard(dst, w.Card, r.cx()-60, r.y+50, 120, false, t)
		case rCharm:
			ch := charmByID(w.Charm)
			title, desc = "Charm", ch.Desc
			ball(dst, r.cx(), r.y+120, 46, hue(float64(len(ch.ID))*0.13+float64(ch.ID[0])*0.05))
			say(dst, string(ch.ID[0]-32), r.cx(), r.y+118, 50, colWhite, 0.5)
			say(dst, ch.Name, r.cx(), r.y+196, 22, colYellow, 0.5)
		case rHeal:
			title, desc = "Rest", fmt.Sprintf("Heal %d HP", w.Heal)
			icon(dst, iHeart, r.cx(), r.y+130, 90)
		}
		say(dst, title, r.cx(), r.y+24, 22, colWhite, 0.5)
		if desc != "" {
			drawText(dst, wrap(desc, 16), r.cx(), r.y+244, 16, colCream, 0.5)
		}
	}
}

func ribbon(dst *ebiten.Image, cx, cy, w float64, s string, c color.RGBA) {
	for _, d := range []float64{-1, 1} {
		poly(dst, []float64{cx + d*w/2, cy - 18, cx + d*(w/2+40), cy - 18, cx + d*(w/2+24), cy + 4, cx + d*(w/2+40), cy + 26, cx + d*w/2, cy + 26}, colInk)
		poly(dst, []float64{cx + d*w/2, cy - 14, cx + d*(w/2+34), cy - 14, cx + d*(w/2+20), cy + 4, cx + d*(w/2+34), cy + 22, cx + d*w/2, cy + 22}, shade(c, 0.7))
	}
	box(dst, rect{cx - w/2, cy - 28, w, 56}, c, 10)
	say(dst, s, cx, cy-2, 34, colWhite, 0.5)
}

func (g *Game) drawEnd(dst *ebiten.Image) {
	r := g.r
	vector.DrawFilledRect(dst, 0, 0, W, float32(g.h), color.RGBA{0, 0, 0, 0xb0}, false)
	if r.Won {
		ribbon(dst, W/2, 150+g.oy(), 400, "CHAPTER CLEAR!", colYellow)
	} else {
		ribbon(dst, W/2, 150+g.oy(), 360, "DEFEATED", colRed)
	}
	floor := r.Floor
	if r.Lost {
		floor--
	}
	say(dst, fmt.Sprintf("Floors cleared: %d / %d", floor, floorsPerChapter), W/2, 228+g.oy(), 24, colWhite, 0.5)
	pr := rect{W/2 - 170, 270 + g.oy(), 340, 120}
	box(dst, pr, colPanel, 16)
	say(dst, "REWARDS", pr.cx(), pr.y+24, 20, colYellow, 0.5)
	coinIcon(dst, pr.cx()-90, pr.y+72, 18)
	say(dst, itoa(r.Gold), pr.cx()-64, pr.y+72, 28, colWhite, 0)
	gemIcon(dst, pr.cx()+50, pr.y+72, 16)
	say(dst, itoa(r.Gems), pr.cx()+74, pr.y+72, 28, colWhite, 0)
	bs := g.endBtns()
	if r.Lost && !r.Revived {
		button(dst, bs[0], colBlue, "", 0, g.p.Gems < reviveGems)
		say(dst, "REVIVE", bs[0].cx(), bs[0].y+20, 22, colWhite, 0.5)
		gemIcon(dst, bs[0].cx()-16, bs[0].y+46, 9)
		say(dst, itoa(reviveGems), bs[0].cx()+10, bs[0].y+46, 18, colWhite, 0.5)
		button(dst, bs[1], colGreen, "", 0, false)
		say(dst, "REVIVE", bs[1].cx(), bs[1].y+20, 22, colWhite, 0.5)
		say(dst, "watch ad", bs[1].cx(), bs[1].y+46, 16, colWhite, 0.5)
		button(dst, bs[2], colYellow, "HOME", 26, false)
		return
	}
	button(dst, bs[0], colYellow, "HOME", 28, false)
}
