// Package game is Orbit Hop: circle a planet, tap to fling yourself to the next one,
// stay ahead of the void. Coins from each run buy upgrades and ships.
package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DataDir is where the save goes. Android sets it to the app's files folder.
var DataDir = "."

// Version is set by the build.
var Version = "dev"

const H = 540 // logical screen height; width follows the screen's shape

type screen int

const (
	scMenu screen = iota
	scPlay
	scOver
	scShop
)

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type btn struct {
	r          rect
	id         string
	label, sub string
	col        color.RGBA
	off        bool
}

type Game struct {
	p        *Profile
	w        float64
	sc       screen
	tab      int // shop: 0 upgrades, 1 ships, 2 store
	r        *run
	earned   int
	newBest  bool
	prevBest int
	frame    int
	toast    string
	toastT   int
	popup    string
	now      func() int64
	seed     int64
	script   *script
}

func NewGame() *Game {
	g := &Game{p: newProfile(), w: 960, now: func() int64 { return time.Now().Unix() }, seed: time.Now().UnixNano()}
	if b, err := readSave(); err == nil {
		p := newProfile()
		if json.Unmarshal(b, p) == nil {
			p.fix()
			g.p = p
		}
	}
	g.script = loadScript()
	if g.script != nil {
		g.seed = 1
	}
	return g
}

// Save writes the profile to disk. Android calls it when the app is paused.
func (g *Game) Save() {
	b, _ := json.Marshal(g.p)
	writeSave(b)
}

func (g *Game) Layout(w, h int) (int, int) {
	g.w = math.Max(720, math.Round(float64(w)*H/float64(max(h, 1))))
	return int(g.w), H
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 120 }

func (g *Game) startRun() {
	g.seed++
	g.r = newRun(g.seed, g.p, g.w)
	g.sc = scPlay
	g.p.Runs++
}

func (g *Game) endRun() {
	g.earned = g.p.runCoins(g.r.score, g.r.starsN)
	g.p.Coins += g.earned
	g.prevBest = g.p.Best
	g.newBest = g.r.score > g.p.Best
	if g.newBest {
		g.p.Best = g.r.score
	}
	g.Save()
}

// buttons lists the buttons on the current screen; Update and Draw both use it.
func (g *Game) buttons() []btn {
	w := g.w
	switch g.sc {
	case scMenu:
		ready := g.p.DailyReady(g.now())
		daily := btn{rect{w/2 + 5, 372, 145, 56}, "daily", "DAILY", "", colGreen, !ready}
		if ready {
			daily.sub = "ready!"
		} else {
			left := dailyGap - (g.now() - g.p.DailyAt)
			daily.sub = fmt.Sprintf("%dh %dm", left/3600, left%3600/60)
		}
		return []btn{
			{rect{w/2 - 150, 284, 300, 74}, "play", "PLAY", "", colGold, false},
			{rect{w/2 - 150, 372, 145, 56}, "shop", "SHOP", "", colCyan, false},
			daily,
		}
	case scOver:
		var b []btn
		if !g.r.revived {
			b = append(b,
				btn{rect{w/2 - 230, 300, 225, 62}, "revive", "CONTINUE", fmt.Sprintf("%d gems", reviveGems), colCyan, g.p.Gems < reviveGems},
				btn{rect{w/2 + 5, 300, 225, 62}, "ad", "CONTINUE", "watch ad", colGreen, false})
		}
		return append(b,
			btn{rect{w/2 - 230, 380, 225, 62}, "again", "AGAIN", "", colGold, false},
			btn{rect{w/2 + 5, 380, 225, 62}, "menu", "MENU", "", colPink, false})
	case scShop:
		b := []btn{{rect{16, 14, 120, 46}, "back", "< BACK", "", colPink, false}}
		x0 := 156.0
		tw := (w - x0 - 190) / 3 // room for coins and gems on the right
		for i, n := range []string{"UPGRADES", "SHIPS", "STORE"} {
			c := colDim
			if i == g.tab {
				c = colGold
			}
			b = append(b, btn{rect{x0 + float64(i)*tw, 14, tw - 10, 46}, fmt.Sprint("tab", i), n, "", c, false})
		}
		return b
	}
	return nil
}

func (g *Game) row(i int) rect { return rect{16, 80 + float64(i)*72, g.w - 32, 64} }

func (g *Game) shipCard(i int) rect {
	cw := (g.w - 32 - 3*14) / 4
	return rect{16 + float64(i)*(cw+14), 90, cw, 300}
}

func (g *Game) click(x, y float64) {
	if g.popup != "" {
		g.popup = ""
		return
	}
	if g.sc == scPlay {
		g.r.Launch()
		return
	}
	for _, b := range g.buttons() {
		if b.r.has(x, y) {
			g.press(b.id)
			return
		}
	}
	if g.sc != scShop {
		return
	}
	switch g.tab {
	case 0:
		for i := range upgrades {
			if g.row(i).has(x, y) {
				g.result(g.p.BuyUpg(i), upgrades[i].Name+" upgraded")
			}
		}
	case 1:
		for i := range ships {
			if g.shipCard(i).has(x, y) {
				g.result(g.p.TapShip(i), "Flying the "+ships[i].Name)
			}
		}
	case 2:
		for i := range store {
			if g.row(i).has(x, y) {
				g.result(g.p.BuyPack(i), store[i].Name+": thanks!")
			}
		}
	}
}

func (g *Game) result(err error, ok string) {
	if err != nil {
		g.say(err.Error())
		return
	}
	g.say(ok)
	g.Save()
}

func (g *Game) press(id string) {
	switch id {
	case "play", "again":
		g.startRun()
	case "shop":
		g.sc, g.tab = scShop, 0
	case "back", "menu":
		g.sc = scMenu
	case "tab0", "tab1", "tab2":
		g.tab = int(id[3] - '0')
	case "daily":
		if n := g.p.ClaimDaily(g.now()); n > 0 {
			g.popup = fmt.Sprintf("Daily reward, day %d!\n+%d gems\nCome back tomorrow for more.", g.p.DailyStreak, n)
			g.Save()
		} else {
			g.say("Already claimed today")
		}
	case "revive", "ad":
		if id == "revive" {
			if g.p.Gems < reviveGems {
				g.say("Not enough gems: get more in the Store")
				return
			}
			g.p.Gems -= reviveGems
		}
		// take back this run's coins; they are paid again when the run ends
		g.p.Coins -= g.earned
		g.p.Best = g.prevBest
		g.r.Revive()
		g.sc = scPlay
	}
}

func (g *Game) Update() error {
	g.frame++
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
	if g.sc == scPlay && (inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter)) {
		g.r.Launch()
	}
	if g.sc == scPlay {
		g.r.step(1.0/60, g.w)
		if g.r.dead {
			g.sc = scOver
			g.endRun()
		}
	}
	if g.toastT > 0 {
		g.toastT--
	}
	return nil
}

func (g *Game) Draw(dst *ebiten.Image) {
	dst.Fill(colBG)
	t := float64(g.frame) / 60
	switch g.sc {
	case scMenu:
		g.drawMenu(dst, t)
	case scPlay, scOver:
		g.drawRun(dst, t)
		if g.sc == scOver {
			g.drawOver(dst)
		}
	case scShop:
		drawStars(dst, t*20, g.w, t)
		g.drawShop(dst)
	}
	for _, b := range g.buttons() {
		drawBtn(dst, b)
	}
	g.drawTop(dst)
	if g.toastT > 0 {
		r := rect{g.w/2 - 280, H - 64, 560, 44}
		panel(dst, r, colGold, colPanel)
		say(dst, g.toast, g.w/2, r.y+22, 16, colText, 0.5)
	}
	if g.popup != "" {
		vector.DrawFilledRect(dst, 0, 0, float32(g.w), H, color.RGBA{0, 0, 0, 0xb0}, false)
		r := rect{g.w/2 - 250, H/2 - 100, 500, 200}
		panel(dst, r, colGold, colPanel)
		say(dst, g.popup, g.w/2, r.y+85, 20, colText, 0.5)
		say(dst, "tap to continue", g.w/2, r.y+r.h-24, 14, colDim, 0.5)
	}
	if g.script != nil {
		g.script.afterDraw(dst)
	}
}

func drawBtn(dst *ebiten.Image, b btn) {
	c := b.col
	if b.off {
		c = fade(c, 0.45)
	}
	panel(dst, b.r, c, colPanel)
	cy := b.r.y + b.r.h/2
	size := math.Min(28, b.r.h*0.4)
	if b.sub != "" {
		say(dst, b.label, b.r.x+b.r.w/2, cy-10, size*0.8, colText, 0.5)
		say(dst, b.sub, b.r.x+b.r.w/2, cy+13, 14, c, 0.5)
	} else {
		say(dst, b.label, b.r.x+b.r.w/2, cy, size, c, 0.5)
	}
}

// coins and gems, top right
func (g *Game) drawTop(dst *ebiten.Image) {
	if g.sc == scPlay {
		return
	}
	say(dst, fmt.Sprintf("%d", g.p.Coins), g.w-120, 37, 20, colGold, 1)
	vector.DrawFilledCircle(dst, float32(g.w-104), 37, 8, colGold, true)
	say(dst, fmt.Sprintf("%d", g.p.Gems), g.w-38, 37, 20, colCyan, 1)
	gem(dst, g.w-22, 37, 9)
}

func gem(dst *ebiten.Image, x, y, r float64) {
	p := &vector.Path{}
	p.MoveTo(float32(x), float32(y-r))
	p.LineTo(float32(x+r*0.8), float32(y))
	p.LineTo(float32(x), float32(y+r))
	p.LineTo(float32(x-r*0.8), float32(y))
	p.Close()
	fillPath(dst, p, colCyan)
}

func (g *Game) drawMenu(dst *ebiten.Image, t float64) {
	drawStars(dst, t*30, g.w, t)
	// a planet with the player's ship circling it
	px, py := g.w*0.84, 150.0
	drawPlanet(dst, px, py, 46, 2, true)
	a := t * 2.4
	drawShip(dst, g.p.ship(), px+math.Cos(a)*80, py+math.Sin(a)*80, a+math.Pi/2)
	drawPlanet(dst, g.w*0.12, 430, 28, 1, false)
	drawPlanet(dst, g.w*0.2, 110, 16, 4, false)
	say(dst, "ORBIT HOP", g.w/2+3, 143, 76, fade(colPink, 0.8), 0.5)
	say(dst, "ORBIT HOP", g.w/2, 140, 76, colCyan, 0.5)
	say(dst, "tap to leave orbit  -  reach the next planet  -  outrun the void", g.w/2, 205, 15, colDim, 0.5)
	if g.p.Best > 0 {
		say(dst, fmt.Sprintf("BEST %d", g.p.Best), g.w/2, 245, 20, colGold, 0.5)
	}
}

func (g *Game) drawRun(dst *ebiten.Image, t float64) {
	r := g.r
	cx := r.camX
	drawStars(dst, cx, g.w, t)
	// the void
	vx := r.void - cx
	if vx > -60 {
		for i := 0; i < 6; i++ {
			w := float32(vx + 14 - float64(i)*10)
			if w > 0 {
				vector.DrawFilledRect(dst, 0, 0, w, H, fade(colRed, 0.12), false)
			}
		}
		for y := 0.0; y < H; y += 12 {
			wx := vx + math.Sin(y*0.05+t*5)*6
			vector.DrawFilledCircle(dst, float32(wx), float32(y), 7, fade(colRed, 0.8), true)
		}
	}
	for i, pl := range r.planets {
		if pl.x-cx > -120 && pl.x-cx < g.w+120 {
			drawPlanet(dst, pl.x-cx, pl.y, pl.r, pl.col, i > r.from || i == r.on)
		}
	}
	for _, s := range r.stars {
		if !s.got {
			sz := 6 + math.Sin(t*6+s.x)*1.5
			glow(dst, s.x-cx, s.y, 5, colGold)
			p := &vector.Path{}
			x, y := float32(s.x-cx), float32(s.y)
			p.MoveTo(x, y-float32(sz))
			p.LineTo(x+float32(sz)*0.6, y)
			p.LineTo(x, y+float32(sz))
			p.LineTo(x-float32(sz)*0.6, y)
			p.Close()
			fillPath(dst, p, colGold)
		}
	}
	for _, k := range r.rocks {
		if k.r > 0 {
			y := k.y(r.t)
			vector.DrawFilledCircle(dst, float32(k.x-cx), float32(y), float32(k.r), colRock, true)
			vector.DrawFilledCircle(dst, float32(k.x-cx-k.r*0.3), float32(y-k.r*0.3), float32(k.r*0.3), fade(colBG, 0.4), true)
		}
	}
	sh := g.p.ship()
	for i, p := range r.trail {
		f := float64(i) / float64(len(r.trail))
		rad := 2 + f*4
		if sh.Shape == 3 {
			rad *= 1.8
		}
		vector.DrawFilledCircle(dst, float32(p.x-cx), float32(p.y), float32(rad), fade(sh.Trail, f*0.6), true)
	}
	if !r.dead {
		ang := math.Atan2(r.vy, r.vx)
		if r.on >= 0 {
			ang = r.ang + r.dir*math.Pi/2
		}
		drawShip(dst, sh, r.x-cx, r.y, ang)
		for i := 0; i < r.shields; i++ {
			vector.StrokeCircle(dst, float32(r.x-cx), float32(r.y), float32(16+i*4), 1.5, fade(colCyan, 0.6), true)
		}
	}
	// HUD
	say(dst, fmt.Sprint(r.score), g.w/2, 44, 48, colText, 0.5)
	say(dst, fmt.Sprintf("%d", r.starsN), 44, 36, 22, colGold, 0)
	vector.DrawFilledCircle(dst, 26, 36, 8, colGold, true)
	if r.perfect > 0 {
		say(dst, "PERFECT +1", g.w/2, 96, 22, fade(colGreen, float64(r.perfect)/25), 0.5)
	}
	if r.score == 0 && r.on == 0 && g.sc == scPlay {
		say(dst, "TAP to leave orbit", g.w/2, H-60, 22, fade(colText, 0.6+0.4*math.Sin(t*5)), 0.5)
	}
}

func (g *Game) drawOver(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, 0, 0, float32(g.w), H, color.RGBA{0, 0, 0, 0xa0}, false)
	r := g.r
	say(dst, "LOST IN SPACE", g.w/2, 70, 44, colPink, 0.5)
	say(dst, r.why, g.w/2, 115, 16, colDim, 0.5)
	say(dst, fmt.Sprint(r.score), g.w/2, 180, 64, colText, 0.5)
	best := fmt.Sprintf("best %d", g.p.Best)
	if g.newBest {
		best = "NEW BEST!"
	}
	say(dst, best, g.w/2, 230, 18, colGold, 0.5)
	say(dst, fmt.Sprintf("+%d coins", g.earned), g.w/2, 264, 20, colGold, 0.5)
}

var shipBuf = ebiten.NewImage(64, 64)

func (g *Game) drawShop(dst *ebiten.Image) {
	switch g.tab {
	case 0:
		for i, u := range upgrades {
			r := g.row(i)
			lvl := g.p.Upg[u.ID]
			c := colCyan
			cost := fmt.Sprintf("%d", upgCost(u, lvl))
			if lvl >= u.Max {
				c, cost = colDim, "MAX"
			} else if g.p.Coins < upgCost(u, lvl) {
				c = fade(colCyan, 0.4)
			}
			panel(dst, r, c, colPanel)
			say(dst, u.Name, r.x+18, r.y+22, 19, colText, 0)
			say(dst, u.Desc, r.x+18, r.y+45, 13, colDim, 0)
			for k := 0; k < u.Max; k++ { // level pips
				pc := fade(colCyan, 0.25)
				if k < lvl {
					pc = colCyan
				}
				vector.DrawFilledRect(dst, float32(r.x+r.w-260+float64(k)*18), float32(r.y+26), 12, 12, pc, false)
			}
			say(dst, cost, r.x+r.w-40, r.y+r.h/2, 20, colGold, 1)
			if cost != "MAX" {
				vector.DrawFilledCircle(dst, float32(r.x+r.w-24), float32(r.y+r.h/2), 7, colGold, true)
			}
		}
	case 1:
		for i, s := range ships {
			r := g.shipCard(i)
			c := fade(s.Body, 0.5)
			if g.p.Ship == s.ID {
				c = s.Body
			}
			panel(dst, r, c, colPanel)
			t := float64(g.frame) / 60
			shipBuf.Clear()
			drawShip(shipBuf, s, 32, 32, -math.Pi/2)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Translate(-32, -32)
			op.GeoM.Scale(3, 3)
			op.GeoM.Translate(r.x+r.w/2, r.y+110+math.Sin(t*3+float64(i))*6)
			dst.DrawImage(shipBuf, op)
			say(dst, s.Name, r.x+r.w/2, r.y+200, 20, colText, 0.5)
			label, lc := "", colGold
			switch {
			case g.p.Ship == s.ID:
				label, lc = "FLYING", s.Body
			case g.p.Ships[s.ID]:
				label, lc = "tap to fly", colDim
			case s.Gems > 0:
				label, lc = fmt.Sprintf("%d gems", s.Gems), colCyan
			default:
				label = fmt.Sprintf("%d coins", s.Coins)
			}
			say(dst, label, r.x+r.w/2, r.y+250, 16, lc, 0.5)
		}
	case 2:
		for i, s := range store {
			r := g.row(i)
			owned := s.OneTime && g.p.Bought[s.ID] > 0
			c := colGreen
			price := s.Price
			if owned {
				c, price = colDim, "OWNED"
			}
			panel(dst, r, c, colPanel)
			say(dst, s.Name, r.x+18, r.y+22, 19, colText, 0)
			say(dst, s.Desc, r.x+18, r.y+45, 13, colDim, 0)
			say(dst, price, r.x+r.w-20, r.y+r.h/2, 20, c, 1)
		}
		say(dst, "Test build: purchases are free until Google Play billing is added", g.w/2, H-14, 12, colDim, 0.5)
	}
}
