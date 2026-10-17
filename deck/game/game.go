// Package game is Deckbound: a portrait card roguelite. Build a 5-card starting
// deck from your collection, climb a branching map of fights, shops, campfires
// and mysteries, and grow the deck as you go.
package game

import (
	"encoding/json"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// DataDir is where the save goes. Android sets it to the app's files folder.
var DataDir = "."

// Version is set by the build.
var Version = "dev"

var timing = os.Getenv("DECK_TIMING") != ""

// DECK_SCALE forces the render scale (for screenshots).
var scaleOverride, _ = strconv.ParseFloat(os.Getenv("DECK_SCALE"), 64)
var drawTime time.Duration

const W = 540 // logical screen width (portrait); height follows the screen's shape

type screen int

const (
	scHome screen = iota
	scMap
	scBattle
	scReward
	scShop
	scRest
	scEvent
	scPick
	scEnd
)

const (
	pickRemove = iota
	pickForge
)

type pt struct{ x, y float64 }

type floater struct {
	s       string
	x, y    float64
	c       color.RGBA
	size, t float64
}

type flying struct {
	c       Card
	x, y, w float64
	tx, ty  float64
	t       float64
}

type pointer struct {
	down, moved, touch bool
	id                 ebiten.TouchID
	sx, sy, x, y       float64
	lastY              float64
}

type Game struct {
	p     *Profile
	w, h  float64
	sc    screen
	tab   int // home: 0 shop, 1 battle, 2 deck
	sel   int // collection card picked in the Deck tab, or -1
	rng   *rand.Rand
	frame int
	now   func() int64
	seed  int64

	toast    string
	toastT   int
	popup    string
	popCard  *Card
	eventMsg string
	pickWhy  int
	prevSc   screen
	viewDeck bool

	ptr                                    pointer
	mapScroll, colScroll, listScroll       float64
	parts                                  []particle
	fl                                     []floater
	flyCards                               []flying
	handPos                                []pt
	heroHit, enemyHit, swing, lunge, shake float64
	endT, deathT                           int
	lastHP, lastEHP                        int

	bgs             map[[2]int]*ebiten.Image
	mapImg          *ebiten.Image
	mapKey          int64
	mapS            float64
	heroSpr, foeSpr *sprite
	script          *script
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
	g.bgs = map[[2]int]*ebiten.Image{}
	g.heroSpr = newSprite(260, 260)
	g.foeSpr = newSprite(360, 360)
	return g
}

func (g *Game) run() *Run { return g.p.Run }

// Save writes the profile (and any run in progress) to disk.
func (g *Game) Save() {
	b, _ := json.Marshal(g.p)
	writeSave(b)
}

func (g *Game) Layout(w, h int) (int, int) {
	fw, fh := g.LayoutF(float64(w), float64(h))
	return int(fw), int(fh)
}

// LayoutF renders at the screen's real pixel size; S maps layout units to pixels.
func (g *Game) LayoutF(w, h float64) (float64, float64) {
	g.w = W
	g.h = math.Max(960, math.Round(h*W/math.Max(w, 1)))
	dsf := ebiten.Monitor().DeviceScaleFactor()
	if scaleOverride > 0 {
		dsf = scaleOverride * W / math.Max(w, 1)
	}
	S = math.Max(0.5, math.Min(3, w*dsf/W))
	return W * S, g.h * S
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 140 }

func (g *Game) result(err error, ok string) bool {
	if err != nil {
		g.say(err.Error())
		return false
	}
	if ok != "" {
		g.say(ok)
	}
	g.Save()
	return true
}

func (g *Game) t() float64 { return float64(g.frame) / 60 }

func (g *Game) battleBG(ch int) *ebiten.Image {
	k := [2]int{ch, int(g.h * 1000 * S)}
	if g.bgs[k] == nil {
		g.bgs[k] = battleBG(ch, W, g.h)
	}
	return g.bgs[k]
}

// ---- moving between screens ----

func (g *Game) startRun() {
	g.p.tickEnergy(g.now())
	if g.p.Energy < runCost {
		g.say("Not enough energy: get more in the Shop")
		return
	}
	g.p.Energy -= runCost
	g.p.Runs++
	g.seed++
	g.p.Run = newRun(g.seed, g.p)
	g.p.Run.restore()
	g.Save()
	g.openMap()
}

func (g *Game) openMap() {
	r := g.run()
	g.sc = scMap
	g.viewDeck = false
	row := 0
	if r.Cur >= 0 {
		row = r.Nodes[r.Cur].Row
	}
	g.mapScroll = g.clampMap(g.nodeY(row) - (g.h - 280))
}

// continueRun picks up a saved run where it left off.
func (g *Game) continueRun() {
	r := g.run()
	r.Resume()
	if r.Cur >= 0 && !r.Nodes[r.Cur].Done {
		g.enterScreen()
		return
	}
	g.openMap()
}

func (g *Game) travel(i int) {
	r := g.run()
	r.Enter(i)
	g.eventMsg = ""
	g.Save()
	g.enterScreen()
}

func (g *Game) enterScreen() {
	r := g.run()
	switch r.Nodes[r.Cur].Kind {
	case nFight, nElite, nBoss:
		g.sc = scBattle
		g.fl, g.parts, g.flyCards, g.handPos = nil, nil, nil, nil
		g.endT, g.deathT = 0, 0
		g.lastHP, g.lastEHP = r.HP, r.B.EHP
	case nShop:
		g.sc = scShop
	case nRest:
		g.sc = scRest
	case nEvent:
		g.sc = scEvent
	case nTreasure:
		g.sc = scReward
	}
}

func (g *Game) leaveNode() {
	g.run().Leave()
	g.Save()
	g.openMap()
}

// finishRun pays out the run and goes home.
func (g *Game) finishRun() {
	r := g.run()
	g.p.Gold += r.Gold
	g.p.Gems += r.Gems
	k := strconv.Itoa(r.Chapter)
	f := r.floor()
	if r.Lost {
		f--
	}
	if r.Won {
		f = mapRows + 1
		if r.Chapter == g.p.MaxChapter {
			g.p.MaxChapter++
			g.p.Chapter = g.p.MaxChapter
		}
	}
	if f > g.p.Best[k] {
		g.p.Best[k] = f
	}
	g.p.Run = nil
	g.sc, g.tab = scHome, 1
	g.Save()
}

// ---- input ----

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
	g.updatePointer()
	if g.sc == scBattle && inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.press("endturn")
	}
	g.updateBattle()
	g.stepEffects()
	if g.toastT > 0 {
		g.toastT--
	}
	return nil
}

func (g *Game) updatePointer() {
	p := &g.ptr
	if !p.down {
		if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
			x, y := ebiten.TouchPosition(ids[0])
			fx, fy := float64(x)/S, float64(y)/S
			*p = pointer{down: true, touch: true, id: ids[0], sx: fx, sy: fy, x: fx, y: fy, lastY: fy}
		} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			fx, fy := float64(x)/S, float64(y)/S
			*p = pointer{down: true, sx: fx, sy: fy, x: fx, y: fy, lastY: fy}
		}
		if _, wy := ebiten.Wheel(); wy != 0 {
			g.scrollBy(wy * 40)
		}
		return
	}
	released := false
	if p.touch {
		if inpututil.IsTouchJustReleased(p.id) {
			released = true
		} else {
			x, y := ebiten.TouchPosition(p.id)
			p.x, p.y = float64(x)/S, float64(y)/S
		}
	} else {
		x, y := ebiten.CursorPosition()
		p.x, p.y = float64(x)/S, float64(y)/S
		released = !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	}
	if math.Hypot(p.x-p.sx, p.y-p.sy) > 14 {
		p.moved = true
	}
	if p.moved {
		g.scrollBy(p.y - p.lastY)
	}
	p.lastY = p.y
	if released {
		p.down = false
		if !p.moved {
			g.click(p.sx, p.sy)
		}
	}
}

// scrollBy moves whatever list is on screen (positive dy = finger moving down).
func (g *Game) scrollBy(dy float64) {
	switch {
	case g.viewDeck || g.sc == scPick:
		g.listScroll = math.Max(0, math.Min(g.listMax(), g.listScroll-dy))
	case g.sc == scMap:
		g.mapScroll = g.clampMap(g.mapScroll - dy)
	case g.sc == scHome && g.tab == 2:
		g.colScroll = math.Max(0, math.Min(g.colMax(), g.colScroll-dy))
	}
}

func (g *Game) press(id string) {
	name, arg, _ := strings.Cut(id, ":")
	n, _ := strconv.Atoi(arg)
	r := g.run()
	switch name {
	case "tab":
		g.tab, g.sel = n, -1
	case "start":
		if r != nil {
			g.continueRun()
		} else {
			g.startRun()
		}
	case "abandon":
		if r != nil {
			g.p.Run = nil
			g.Save()
			g.say("Run abandoned")
		}
	case "upgrade":
		g.result(g.p.UpgradeHero(), "Hero upgraded: "+itoa(g.p.maxHP())+" HP")
	case "chapter":
		if r == nil {
			g.p.Chapter = max(1, min(g.p.MaxChapter, g.p.Chapter+n))
		}
	case "buy":
		c, err := g.p.Buy(n, g.now(), g.rng)
		if g.result(err, "") && c != nil {
			g.popup, g.popCard = "NEW CARD!", c
		}
	case "sel":
		g.sel = n
	case "toggle":
		if g.sel >= 0 && g.sel < len(g.p.Cards) {
			if g.result(g.p.Toggle(g.sel), "") {
				c := g.p.Cards[g.sel]
				g.p.sortCards()
				g.sel = g.findCard(c)
				g.Save()
			}
		}
	case "enh":
		if g.sel >= 0 && g.result(g.p.Enhance(g.sel, n), "") {
			c := g.p.Cards[g.sel]
			g.popup, g.popCard = enhDefs[n].Name+"!", &c
		}
	case "node":
		if r != nil && containsInt(r.Reachable(), n) {
			g.travel(n)
		}
	case "home":
		if r != nil && (r.Won || r.Lost) {
			g.finishRun()
		} else {
			g.sc, g.tab = scHome, 1
		}
	case "deck":
		g.viewDeck = !g.viewDeck
		g.listScroll = 0
	case "play":
		g.playCard(n)
	case "endturn":
		if g.sc == scBattle && g.endT == 0 && g.deathT == 0 && r.B.EHP > 0 {
			g.endT = 1
		}
	case "pick":
		r.PickCard(n)
		g.leaveNode()
	case "continue":
		g.leaveNode()
	case "offer":
		g.result(r.BuyOffer(n), "Bought!")
	case "heal":
		g.result(r.ShopHeal(), "Healed")
	case "remove", "forge":
		inShop := g.sc == scShop
		if name == "remove" {
			switch {
			case r.ShopRemoved:
				g.say("Already used")
				return
			case len(r.Deck) <= 1:
				g.say("You need at least 1 card")
				return
			case inShop && r.Coins < removePrice:
				g.say("Need " + itoa(removePrice-r.Coins) + " more coins")
				return
			}
			g.pickWhy = pickRemove
		} else {
			switch {
			case inShop && r.ShopForged:
				g.say("Already used")
				return
			case inShop && r.Coins < forgePrice:
				g.say("Need " + itoa(forgePrice-r.Coins) + " more coins")
				return
			}
			g.pickWhy = pickForge
		}
		g.prevSc = g.sc
		g.sc, g.listScroll = scPick, 0
	case "picked":
		g.pickCard(n)
	case "cancel":
		g.sc = g.prevSc
	case "rest":
		r.HP = min(r.Max, r.HP+r.restHeal())
		g.leaveNode()
	case "choose":
		msg, err := r.Choose(n)
		if g.result(err, "") {
			g.eventMsg = msg
		}
	case "revive", "ad":
		if name == "revive" {
			if g.p.Gems < reviveGems {
				g.say("Not enough gems: get more in the Shop")
				return
			}
			g.p.Gems -= reviveGems
		}
		r.Revive()
		g.Save()
		g.enterScreen()
	}
}

func (g *Game) findCard(c Card) int {
	for i, o := range g.p.Cards {
		if o == c {
			return i
		}
	}
	return -1
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// pickCard finishes a remove or forge on run deck card i.
func (g *Game) pickCard(i int) {
	r := g.run()
	if i < 0 || i >= len(r.Deck) {
		return
	}
	c := r.Deck[i]
	switch g.pickWhy {
	case pickRemove:
		r.RemoveCard(i)
		if g.prevSc == scShop {
			r.Coins -= removePrice
			r.ShopRemoved = true
		}
		g.say(c.def().Name + " removed")
	case pickForge:
		r.ForgeCard(i)
		c = r.Deck[i]
		g.popup, g.popCard = enhDefs[r.ForgeEnh].Name+"!", &c
		if g.prevSc == scShop {
			r.Coins -= forgePrice
			r.ShopForged = true
		}
	}
	g.Save()
	if g.prevSc == scRest {
		g.leaveNode()
		return
	}
	g.sc = g.prevSc
}

// ---- battle flow and effects ----

func (g *Game) heroX() float64 { return 150 }
func (g *Game) heroY() float64 { return g.h - 330 }
func (g *Game) foeX() float64  { return 350 }
func (g *Game) foeY() float64  { return g.h - 500 }

func (g *Game) playCard(i int) {
	r := g.run()
	if g.sc != scBattle || g.endT > 0 || g.deathT > 0 || i >= len(r.B.Hand) {
		return
	}
	b := r.B
	if !b.Playable(i) {
		g.say("Not enough energy")
		return
	}
	c := b.Hand[i]
	hr := g.handCard(i, len(b.Hand))
	if i < len(g.handPos) {
		hr.x, hr.y = g.handPos[i].x, g.handPos[i].y
		g.handPos = append(g.handPos[:i], g.handPos[i+1:]...)
	}
	tx, ty := g.foeX(), g.foeY()-90
	if c.def().Kind == kSkill {
		tx, ty = g.heroX(), g.heroY()-70
	} else {
		g.swing = 1
	}
	g.flyCards = append(g.flyCards, flying{c: c, x: hr.x, y: hr.y, w: hr.w, tx: tx, ty: ty})
	b.Play(i)
}

func (g *Game) updateBattle() {
	r := g.run()
	if r == nil || r.B == nil || (g.sc != scBattle && g.sc != scEnd) {
		return
	}
	b := r.B
	if g.endT > 0 {
		g.endT++
		g.lunge = math.Sin(math.Min(1, float64(g.endT)/24) * math.Pi)
		if g.endT == 12 {
			b.EndTurn()
			g.handPos = nil
		}
		if g.endT >= 26 {
			g.endT, g.lunge = 0, 0
		}
	}
	for _, f := range b.fx {
		g.effect(f)
	}
	b.fx = nil
	if r.HP < g.lastHP {
		g.heroHit = 1
	}
	if b.EHP < g.lastEHP {
		g.enemyHit = 1
	}
	g.lastHP, g.lastEHP = r.HP, b.EHP
	if g.sc != scBattle {
		return
	}
	switch {
	case r.Lost:
		g.sc, g.toastT = scEnd, 0
		g.Save()
	case b.EHP <= 0:
		if g.deathT == 0 {
			g.deathT = 1
			g.burst(g.foeX(), g.foeY()-70, colYellow, 40)
			g.burst(g.foeX(), g.foeY()-70, colWhite, 20)
		}
		g.deathT++
		if g.deathT > 50 {
			g.deathT, g.toastT = 0, 0
			if r.Won {
				g.sc = scEnd
			} else {
				g.sc = scReward
			}
			g.Save()
		}
	}
}

func (g *Game) effect(f fx) {
	x, y := g.heroX(), g.heroY()-110
	if f.Enemy {
		x, y = g.foeX(), g.foeY()-130
	}
	k := 0.0
	for _, o := range g.fl {
		if math.Abs(o.x-x) < 60 && o.t < 20 {
			k++
		}
	}
	c, size := colWhite, 40.0
	switch f.Kind {
	case fxDamage:
		if f.Enemy {
			g.slash(x, y+50)
		} else {
			c = colRed
			g.shake = 8
			g.burst(x, y+60, colRed, 14)
		}
		if n, _ := strconv.Atoi(f.S); n >= 12 {
			g.shake = 10
			size = 52
		}
	case fxBlock:
		c = colBlue
		g.ring(x, y+60, colBlue)
	case fxHeal:
		c = colGreen
		g.burst(x, y+60, colGreen, 12)
	case fxPoison:
		c, size = rgb(0x8a, 0xff, 0x50), 28
		g.burst(x, y+60, colGreen, 10)
	case fxBuff:
		c, size = colOrange, 28
		g.ring(x, y+60, colOrange)
	default:
		c, size = colYellow, 28
	}
	g.fl = append(g.fl, floater{s: f.S, x: x + (k-1)*14, y: y - k*34, c: c, size: size})
}

func (g *Game) burst(x, y float64, c color.RGBA, n int) {
	for i := 0; i < n; i++ {
		a := g.rng.Float64() * 2 * math.Pi
		v := 2 + g.rng.Float64()*5
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 2, life: 30 + g.rng.Float64()*20, max: 50, size: 3 + g.rng.Float64()*4, col: c, grav: 0.15, kind: g.rng.Intn(2)})
	}
}

func (g *Game) ring(x, y float64, c color.RGBA) {
	g.parts = append(g.parts, particle{x: x, y: y, life: 26, max: 26, size: 60, col: c, kind: 2})
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		g.parts = append(g.parts, particle{x: x + math.Cos(a)*40, y: y + math.Sin(a)*40, vx: math.Cos(a), vy: -1.5, life: 30, max: 30, size: 5, col: c, kind: 1})
	}
}

func (g *Game) slash(x, y float64) {
	for i := 0; i < 14; i++ {
		f := float64(i)/13 - 0.5
		g.parts = append(g.parts, particle{x: x + f*120, y: y - f*90, vx: 3, vy: -2, life: 14, max: 14, size: 6, col: colWhite})
	}
	g.burst(x, y, colYellow, 12)
}

func (g *Game) stepEffects() {
	k := g.parts[:0]
	for _, p := range g.parts {
		if p.step() {
			k = append(k, p)
		}
	}
	g.parts = k
	f := g.fl[:0]
	for _, o := range g.fl {
		o.t++
		o.y -= 1.1
		if o.t < 60 {
			f = append(f, o)
		}
	}
	g.fl = f
	c := g.flyCards[:0]
	for _, o := range g.flyCards {
		o.t += 1.0 / 16
		if o.t < 1 {
			c = append(c, o)
		}
	}
	g.flyCards = c
	g.heroHit = math.Max(0, g.heroHit-0.07)
	g.enemyHit = math.Max(0, g.enemyHit-0.08)
	g.swing = math.Max(0, g.swing-0.06)
	g.shake *= 0.85
	// drifting motes for atmosphere
	if (g.sc == scBattle || g.sc == scMap || g.sc == scHome) && g.frame%9 == 0 && len(g.parts) < 120 {
		c := rgb(0xff, 0xff, 0xe0)
		if r := g.run(); r != nil && g.sc != scHome {
			switch (r.Chapter - 1) % len(themes) {
			case 1:
				c = rgb(0xb0, 0x9a, 0xff)
			case 3:
				c = rgb(0xc8, 0xff, 0x60)
			case 5:
				c = rgb(0xff, 0x8a, 0x40)
			}
		}
		g.parts = append(g.parts, particle{x: g.rng.Float64() * W, y: g.h * (0.2 + g.rng.Float64()*0.6), vx: (g.rng.Float64() - 0.5) * 0.6, vy: -0.3 - g.rng.Float64()*0.5, life: 160, max: 160, size: 1.5 + g.rng.Float64()*1.5, col: c})
	}
}

// ---- drawing ----

func (g *Game) Draw(dst *ebiten.Image) {
	if timing {
		t0 := time.Now()
		defer func() {
			drawTime += time.Since(t0)
			if g.frame%120 == 0 {
				println("draw ms/frame:", drawTime.Milliseconds()/120, "screen", int(g.sc))
				drawTime = 0
			}
		}()
	}
	switch g.sc {
	case scHome:
		g.drawHome(dst)
	case scMap:
		g.drawMap(dst)
	case scBattle, scEnd:
		g.drawBattle(dst)
		if g.sc == scEnd {
			g.drawEnd(dst)
		}
	case scReward:
		g.drawReward(dst)
	case scShop:
		g.drawShop(dst)
	case scRest:
		g.drawRest(dst)
	case scEvent:
		g.drawEvent(dst)
	case scPick:
		g.drawPick(dst)
	}
	if g.viewDeck {
		g.drawDeckView(dst)
	}
	if g.toastT > 0 {
		a := math.Min(1, float64(g.toastT)/20)
		r := rect{20, 120, W - 40, 46}
		roundRect(dst, r, 23, alpha(rgb(0x14, 0x12, 0x28), 0.88*a))
		say(dst, g.toast, W/2, r.cy(), 18, alpha(colWhite, a), 0.5)
	}
	if g.popup != "" {
		g.drawPopup(dst)
	}
	if g.script != nil {
		g.script.afterDraw(dst)
	}
}

func (g *Game) drawPopup(dst *ebiten.Image) {
	fillRect(dst, 0, 0, W, g.h, color.RGBA{0, 0, 0, 0xc0})
	cx, cy := W/2.0, g.h/2
	for i := 0; i < 14; i++ {
		a := g.t()*0.5 + float64(i)*math.Pi/7
		shape(dst, []float64{cx, cy, cx + math.Cos(a)*800, cy + math.Sin(a)*800, cx + math.Cos(a+0.14)*800, cy + math.Sin(a+0.14)*800}, alpha(colYellow, 0.16), clear, 0)
	}
	glow(dst, cx, cy, 220, colYellow, 0.4)
	say(dst, g.popup, cx, cy-230, 46, colYellow, 0.5)
	if g.popCard != nil {
		drawCard(dst, *g.popCard, cx-105, cy-160, 210, false, g.t())
		r := g.popCard.def().Rarity
		say(dst, rarityNames[r], cx, cy+158, 26, light(rarityCol[r], 0.3), 0.5)
	}
	say(dst, "Tap to continue", cx, cy+215, 20, colWhite, 0.5)
}

func (g *Game) drawParticles(dst *ebiten.Image) {
	for i := range g.parts {
		g.parts[i].draw(dst)
	}
}

// topBar shows energy, gold and gems on the home screens.
func (g *Game) topBar(dst *ebiten.Image) {
	now := g.now()
	er := rect{18, 16, 148, 36}
	pill(dst, er, itoa(g.p.Energy)+"/"+itoa(maxEnergy))
	boltIcon(dst, er.x+8, er.cy(), 15, colGreen)
	if g.p.Energy < maxEnergy {
		s := g.p.nextEnergyIn(now)
		say(dst, "+1 in "+itoa(int(s/60))+":"+pad2(int(s%60)), er.cx()+6, er.y+er.h+14, 13, colWhite, 0.5)
	}
	gr := rect{186, 16, 160, 36}
	pill(dst, gr, itoa(g.p.Gold))
	coinIcon(dst, gr.x+8, gr.cy(), 15)
	mr := rect{366, 16, 156, 36}
	pill(dst, mr, itoa(g.p.Gems))
	gemIcon(dst, mr.x+8, mr.cy(), 13)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
