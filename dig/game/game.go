// Package game is Deep Dig, an idle mining game: tap the rock for coins, hire
// diggers that mine on their own, descend for gems that boost every coin after.
package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
)

// DataDir is where the save goes. Android sets it to the app's files folder.
var DataDir = "."

// Version is set by the build.
var Version = "dev"

const H = 540 // logical screen height; width follows the screen's shape

var (
	colBG     = color.RGBA{0x1d, 0x16, 0x22, 0xff}
	colPanel  = color.RGBA{0x2b, 0x22, 0x33, 0xff}
	colRow    = color.RGBA{0x3a, 0x2e, 0x45, 0xff}
	colEdge   = color.RGBA{0x55, 0x46, 0x66, 0xff}
	colText   = color.RGBA{0xf0, 0xe8, 0xf5, 0xff}
	colDim    = color.RGBA{0xa8, 0x98, 0xb8, 0xff}
	colGold   = color.RGBA{0xf5, 0xc5, 0x42, 0xff}
	colGem    = color.RGBA{0x5e, 0xe6, 0xff, 0xff}
	colGreen  = color.RGBA{0x4c, 0xc2, 0x6a, 0xff}
	colGrey   = color.RGBA{0x55, 0x50, 0x5c, 0xff}
	colRock   = color.RGBA{0x6e, 0x64, 0x72, 0xff}
	colRockHi = color.RGBA{0x8f, 0x86, 0x93, 0xff}
	colRockLo = color.RGBA{0x4a, 0x42, 0x4e, 0xff}
)

type floater struct {
	x, y, vx, vy float64
	life         int
	s            string
	c            color.Color
	size         float64
}

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type Game struct {
	s       *State
	w       float64 // logical width
	tab     int     // 0 upgrades, 1 gems, 2 store
	frame   int
	shake   float64
	fl      []floater
	toast   string
	toastT  int
	popup   string // shown until tapped
	confirm bool   // descend asked once, tap again to confirm
	now     func() int64
	script  *script
}

func NewGame() *Game {
	g := &Game{s: newState(), w: 960, now: func() int64 { return time.Now().Unix() }}
	g.load()
	if c := g.s.Offline(g.now()); c >= 1 {
		g.popup = "Welcome back!\nYour diggers found " + fmtNum(c) + " coins."
	}
	g.s.LastSeen = g.now()
	g.script = loadScript()
	return g
}

func savePath() string { return filepath.Join(DataDir, "dig_save.json") }

func (g *Game) load() {
	b, err := os.ReadFile(savePath())
	if err != nil {
		return
	}
	s := newState()
	if json.Unmarshal(b, s) == nil {
		s.fix()
		g.s = s
	}
}

// Save writes the game to disk. Android calls it when the app is paused.
func (g *Game) Save() {
	g.s.LastSeen = g.now()
	b, _ := json.Marshal(g.s)
	tmp := savePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, savePath())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	g.w = math.Max(720, math.Round(float64(w)*H/float64(max(h, 1))))
	return int(g.w), H
}

// Screen areas.
func (g *Game) leftW() float64        { return math.Round(g.w * 0.44) }
func (g *Game) rockC() (x, y float64) { return g.leftW() / 2, 265 }
func (g *Game) descendBtn() rect      { return rect{14, 470, g.leftW()/2 - 21, 56} }
func (g *Game) dailyBtn() rect        { return rect{g.leftW()/2 + 7, 470, g.leftW()/2 - 21, 56} }
func (g *Game) tabBtn(i int) rect {
	x0, w := g.leftW(), g.w-g.leftW()-12
	return rect{x0 + float64(i)*w/3, 10, w/3 - 6, 46}
}
func (g *Game) row(i int) rect {
	x0 := g.leftW()
	return rect{x0, 66 + float64(i)*58, g.w - x0 - 12, 54}
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 120 }

func (g *Game) float(x, y float64, s string, c color.Color, size float64) {
	g.fl = append(g.fl, floater{x, y, (rand.Float64() - 0.5) * 2, -2.2, 50, s, c, size})
}

func (g *Game) tapRock(x, y float64, auto bool) {
	v := g.s.TapValue(g.now())
	g.s.earn(v)
	if !auto {
		g.s.Taps++
		g.shake = 6
		g.float(x, y-20, "+"+fmtNum(v), colGold, 22)
		for i := 0; i < 3; i++ {
			g.fl = append(g.fl, floater{x, y, (rand.Float64() - 0.5) * 7, -rand.Float64()*5 - 1, 30, "■", colRockHi, 12})
		}
	}
}

func (g *Game) click(x, y float64) {
	now := g.now()
	if g.popup != "" {
		g.popup = ""
		return
	}
	cx, cy := g.rockC()
	if math.Hypot(x-cx, y-cy) < 120 {
		g.tapRock(x, y, false)
		return
	}
	if g.descendBtn().has(x, y) {
		n := g.s.DescendGems()
		if n < 1 {
			g.say("Earn " + fmtNum(50000) + " coins this run to descend")
		} else if !g.confirm {
			g.confirm = true
			g.say("Tap again to descend: restart for +" + fmt.Sprint(n) + " gems")
		} else if g.s.Descend() {
			g.confirm = false
			g.popup = fmt.Sprintf("Depth %d!\n+%d gems. Each gem gives +10%% coins.", g.s.Depth, n)
			g.Save()
		}
		return
	}
	g.confirm = false
	if g.dailyBtn().has(x, y) {
		if n := g.s.ClaimDaily(now); n > 0 {
			g.popup = fmt.Sprintf("Daily reward, day %d!\n+%d gems. Come back tomorrow for more.", g.s.DailyStreak, n)
		} else {
			left := dailyGap - (now - g.s.DailyAt)
			g.say(fmt.Sprintf("Next reward in %dh %dm", left/3600, left%3600/60))
		}
		return
	}
	for i := 0; i < 3; i++ {
		if g.tabBtn(i).has(x, y) {
			g.tab = i
			return
		}
	}
	for i := 0; i < 8; i++ {
		if !g.row(i).has(x, y) {
			continue
		}
		ok, name := false, ""
		switch g.tab {
		case 0:
			if i == 0 {
				ok, name = g.s.BuyPick(), "Pickaxe"
			} else if i-1 < len(gens) {
				ok, name = g.s.BuyGen(i-1), gens[i-1].Name
			}
		case 1:
			if i < len(gemShop) {
				ok, name = g.s.BuyGem(i, now), gemShop[i].Name
			}
		case 2:
			if i < len(store) {
				ok, name = g.s.BuyPack(i, now), store[i].Name
			}
		}
		if name == "" {
			return
		}
		r := g.row(i)
		if ok {
			g.float(r.x+r.w-70, r.y+10, "Bought!", colGreen, 18)
			if g.tab > 0 {
				g.Save()
			}
		} else {
			g.say("Not enough for " + name)
		}
		return
	}
}

func (g *Game) Update() error {
	g.frame++
	now := g.now()
	g.s.earn(g.s.CPS(now) / 60)
	if g.s.Bought["auto"] > 0 && g.frame%12 == 0 {
		g.tapRock(0, 0, true)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.click(float64(x), float64(y))
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		g.click(float64(x), float64(y))
	}
	if g.script != nil {
		if err := g.script.step(g); err != nil {
			return err
		}
	}
	g.shake *= 0.75
	if g.toastT > 0 {
		g.toastT--
	}
	k := g.fl[:0]
	for _, f := range g.fl {
		f.x, f.y, f.vy, f.life = f.x+f.vx, f.y+f.vy, f.vy+0.12, f.life-1
		if f.life > 0 {
			k = append(k, f)
		}
	}
	g.fl = k
	if g.frame%300 == 0 {
		g.Save()
	}
	return nil
}

// ---- drawing ----

var faceSrc = func() *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
	return s
}()

// say draws text; ax is 0 left, 0.5 centre, 1 right; y is the text's middle.
func say(dst *ebiten.Image, s string, x, y, size float64, c color.Color, ax float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = text.Align(int(ax * 2))
	op.SecondaryAlign = text.AlignCenter
	op.LineSpacing = size * 1.3
	text.Draw(dst, s, &text.GoTextFace{Source: faceSrc, Size: size}, op)
}

func box(dst *ebiten.Image, r rect, fill, edge color.Color) {
	vector.DrawFilledRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), edge, false)
	vector.DrawFilledRect(dst, float32(r.x+3), float32(r.y+3), float32(r.w-6), float32(r.h-6), fill, false)
}

func (g *Game) drawRock(dst *ebiten.Image) {
	cx, cy := g.rockC()
	cx += (rand.Float64() - 0.5) * g.shake
	cy += (rand.Float64() - 0.5) * g.shake
	r := float32(110 - g.shake)
	x, y := float32(cx), float32(cy)
	vector.DrawFilledCircle(dst, x+6, y+8, r, color.RGBA{0, 0, 0, 0x60}, true)
	vector.DrawFilledCircle(dst, x, y, r, colRockLo, true)
	vector.DrawFilledCircle(dst, x-6, y-6, r-8, colRock, true)
	vector.DrawFilledCircle(dst, x-30, y-34, r*0.35, colRockHi, true)
	// ore specks: gold, more gems the deeper you go
	for i := 0; i < 9; i++ {
		a := float64(i) * 2.39
		d := float32(25 + (i*37)%70)
		c := colGold
		if i < g.s.Depth%10 {
			c = colGem
		}
		vector.DrawFilledCircle(dst, x+d*float32(math.Cos(a)), y+d*float32(math.Sin(a)), 7, c, true)
	}
}

func (g *Game) Draw(dst *ebiten.Image) {
	now := g.now()
	s := g.s
	dst.Fill(colBG)
	lw := g.leftW()

	// left: coins, rock, buttons
	say(dst, fmtNum(s.Coins)+" coins", lw/2, 34, 34, colGold, 0.5)
	say(dst, fmtNum(s.CPS(now))+" per second", lw/2, 70, 18, colDim, 0.5)
	say(dst, fmt.Sprintf("%d gems  (+%d%%)", s.Gems, s.Gems*10), 14, 100, 16, colGem, 0)
	say(dst, fmt.Sprintf("Depth %d", s.Depth), lw-14, 100, 16, colDim, 1)
	g.drawRock(dst)
	say(dst, "Tap: +"+fmtNum(s.TapValue(now)), lw/2, 400, 18, colText, 0.5)
	if now < s.BoostUntil {
		left := s.BoostUntil - now
		say(dst, fmt.Sprintf("2x BOOST %d:%02d", left/60, left%60), lw/2, 430, 18, colGreen, 0.5)
	}
	n := s.DescendGems()
	dc := colGrey
	if n >= 1 {
		dc = colGem
	}
	db := g.descendBtn()
	box(dst, db, colRow, dc)
	say(dst, "Descend", db.x+db.w/2, db.y+18, 18, colText, 0.5)
	say(dst, fmt.Sprintf("+%d gems", n), db.x+db.w/2, db.y+39, 14, dc, 0.5)
	yb := g.dailyBtn()
	ready := s.DailyReady(now)
	ec := colGrey
	if ready && g.frame/20%2 == 0 {
		ec = colGold
	} else if ready {
		ec = colGreen
	}
	box(dst, yb, colRow, ec)
	say(dst, "Daily", yb.x+yb.w/2, yb.y+18, 18, colText, 0.5)
	if ready {
		say(dst, "Ready!", yb.x+yb.w/2, yb.y+39, 14, colGold, 0.5)
	} else {
		left := dailyGap - (now - s.DailyAt)
		say(dst, fmt.Sprintf("%dh %dm", left/3600, left%3600/60), yb.x+yb.w/2, yb.y+39, 14, colDim, 0.5)
	}

	// right: tabs and the list
	vector.DrawFilledRect(dst, float32(lw-6), 0, float32(g.w-lw+6), H, colPanel, false)
	for i, name := range []string{"Upgrades", "Gem Shop", "Store"} {
		r := g.tabBtn(i)
		e := colEdge
		if i == g.tab {
			e = colGold
		}
		box(dst, r, colRow, e)
		say(dst, name, r.x+r.w/2, r.y+r.h/2, 18, colText, 0.5)
	}
	type line struct {
		name, sub, cost string
		ok, done        bool
		costC           color.RGBA
	}
	var lines []line
	switch g.tab {
	case 0:
		lines = append(lines, line{fmt.Sprintf("Pickaxe  Lv %d", s.Pick), "More coins per tap", fmtNum(pickCost(s.Pick)), s.Coins >= pickCost(s.Pick), false, colGold})
		for i, gd := range gens {
			c := genCost(i, s.Owned[i])
			lines = append(lines, line{fmt.Sprintf("%s  x%d", gd.Name, s.Owned[i]), fmtNum(gd.CPS) + "/s each", fmtNum(c), s.Coins >= c, false, colGold})
		}
	case 1:
		for _, it := range gemShop {
			done := it.Max > 0 && s.Bought[it.ID] >= it.Max
			cost := fmt.Sprintf("%d gems", it.Cost)
			if done {
				cost = "Owned"
			}
			lines = append(lines, line{it.Name, it.Desc, cost, s.Gems >= it.Cost, done, colGem})
		}
	case 2:
		for _, p := range store {
			done := p.OneTime && s.Bought[p.ID] > 0
			cost := p.Price
			if done {
				cost = "Owned"
			}
			lines = append(lines, line{p.Name, p.Desc, cost, true, done, colGreen})
		}
	}
	for i, l := range lines {
		r := g.row(i)
		e := colEdge
		if l.ok && !l.done {
			e = l.costC
		}
		box(dst, r, colRow, e)
		say(dst, l.name, r.x+14, r.y+19, 17, colText, 0)
		say(dst, l.sub, r.x+14, r.y+38, 13, colDim, 0)
		cc := l.costC
		if !l.ok || l.done {
			cc = colDim
		}
		say(dst, l.cost, r.x+r.w-14, r.y+r.h/2, 17, cc, 1)
	}
	if g.tab == 2 {
		say(dst, "Test build: purchases are free until Google Play billing is added", lw+(g.w-lw)/2, H-22, 12, colDim, 0.5)
	}

	for _, f := range g.fl {
		a := math.Min(1, float64(f.life)/20)
		r, gg, b, _ := f.c.RGBA()
		say(dst, f.s, f.x, f.y, f.size, color.NRGBA{uint8(r >> 8), uint8(gg >> 8), uint8(b >> 8), uint8(a * 255)}, 0.5)
	}
	if g.toastT > 0 {
		r := rect{g.w/2 - 300, 116, 600, 40}
		box(dst, r, colPanel, colGold)
		say(dst, g.toast, g.w/2, r.y+20, 16, colText, 0.5)
	}
	if g.popup != "" {
		vector.DrawFilledRect(dst, 0, 0, float32(g.w), H, color.RGBA{0, 0, 0, 0xa0}, false)
		r := rect{g.w/2 - 260, H/2 - 90, 520, 180}
		box(dst, r, colPanel, colGold)
		say(dst, g.popup, g.w/2, r.y+75, 20, colText, 0.5)
		say(dst, "Tap to continue", g.w/2, r.y+r.h-22, 14, colDim, 0.5)
	}
	if g.script != nil {
		g.script.afterDraw(dst)
	}
}
