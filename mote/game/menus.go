package game

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Screens and menus, all drawn with the pixel widgets in ui.go.

type screen int

const (
	scMain screen = iota
	scSeed
	scPlaying
	scPause
	scInv
	scSettings
	scDebug
)

var (
	cur          = scMain
	settingsFrom = scPause
	started      bool
	toastText    string
	toastUntil   int
	hintUntil    int
	invInfo      string
	dbgInfo      string
	seedEntry    string
	listening    string // action waiting for a new key
	dragFrom     = -1
	dragging     bool
	picked       = -1
	copiedUntil  int
)

func paused() bool { return cur != scPlaying }

func openScreen(s screen) {
	if s == scInv {
		picked, dragFrom, dragging = -1, -1, false
		invInfo = "Drag an item to another slot to move it."
		sfx("open")
	}
	if s == scPause && cur == scPlaying {
		sfx("close")
	}
	cur = s
	clearPointers()
}

func toast(s string) { toastText, toastUntil = s, frame+150 }

func play() {
	cur = scPlaying
	if !started {
		started = true
		hintUntil = frame + 480
	}
	clearPointers()
	sfx("open")
}

// sheet returns a centred panel rect
func sheet(w, h float64) Rect {
	w = math.Min(w, logicalW-28)
	h = math.Min(h, logicalH-24)
	r := Rect{(logicalW - w) / 2, (logicalH - h) / 2, w, h}
	Dim()
	Panel(r)
	return r
}

func head(r Rect, title string, back bool) (backClicked, closeClicked bool) {
	x := r.X + 16
	if back {
		backClicked = Button(Rect{r.X + 14, r.Y + 12, 32, 32}, "", styleSlate, true, icoBack)
		x = r.X + 58
	}
	Title(title, x, r.Y+28)
	closeClicked = Button(Rect{r.X + r.W - 46, r.Y + 12, 32, 32}, "", styleSlate, true, icoX)
	return
}

// ---- HUD while playing ----

func hud() {
	hudRects = hudRects[:0]
	// Hotbar
	slot := 48.0
	if isTouch {
		slot = math.Max(36, math.Min(48, logicalH*0.11))
	}
	gap := 8.0
	tw := 5*slot + 4*gap
	x0, y0 := (logicalW-tw)/2, logicalH-slot-16
	bar := Rect{x0 - 8, y0 - 8, tw + 16, slot + 16}
	hudRects = append(hudRects, bar)
	ops = append(ops, func(dst *ebiten.Image) { fill(dst, bar.X, bar.Y, bar.W, bar.H, color.NRGBA{19, 16, 39, 200}) })
	for i := 0; i < 5; i++ {
		r := Rect{x0 + float64(i)*(slot+gap), y0, slot, slot}
		if slotWidget(r, i, i == selected, false, true) {
			if selected == i {
				selected = -1
			} else {
				selected = i
			}
		}
	}
	if isTouch {
		bx := logicalW - 62
		if prefs.BtnSide == "left" {
			bx = 14
		}
		pr := Rect{bx, logicalH/2 - 88, 48, 48}
		ir := Rect{bx, logicalH/2 - 24, 48, 48}
		hudRects = append(hudRects, pr, ir)
		if Button(pr, "", styleSlate, true, icoPause) {
			openScreen(scPause)
		}
		if Button(ir, "", styleSlate, true, icoBag) {
			openScreen(scInv)
		}
		// joystick
		cx, cy := stickCenter()
		ops = append(ops, func(dst *ebiten.Image) { drawStick(dst, cx, cy) })
	}
	if frame < hintUntil {
		hint := "A / D to move, W to levitate, E for inventory, Esc to pause. Scroll or 1-5 to pick a hotbar item, then click the world to use it."
		if isTouch {
			hint = "Push the stick up to levitate. Pick a hotbar slot, then touch the world to use it. With no slot picked, touch to look around."
		}
		wrapText(hint, logicalW-14, 22, math.Min(420, logicalW*0.5), 13, colMuted, text.AlignEnd)
	}
}

func drawStick(dst *ebiten.Image, cx, cy float64) {
	pc := func(x, y, r float64, c color.Color) { // stepped pixel circle
		for dy := -r; dy <= r; dy += 3 {
			w := math.Sqrt(r*r - dy*dy) // half width
			w = math.Floor(w/3) * 3
			fill(dst, x-w, y+dy, w*2, 3, c)
		}
	}
	ring := color.NRGBA{34, 28, 64, 140}
	if stick.moving {
		ring = color.NRGBA{255, 222, 120, 120}
	}
	pc(cx, cy, stickR, ring)
	pc(cx, cy, stickR-6, color.NRGBA{124, 98, 255, 36})
	pc(cx+stick.kx, cy+stick.ky, 26, color.NRGBA{124, 98, 255, 190})
	pc(cx+stick.kx-6, cy+stick.ky-6, 10, color.NRGBA{196, 180, 255, 170})
}

// Slot with an item icon and count. Returns true when tapped/clicked.
func slotWidget(r Rect, i int, sel, pick bool, showKey bool) bool {
	clicked := ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY) && !dragging
	if clicked {
		sfx("click")
	}
	s := inv[i]
	ops = append(ops, func(dst *ebiten.Image) {
		hi, lo := color.Color(color.NRGBA{10, 8, 23, 255}), color.Color(colPanelHi)
		body := color.Color(colPanelLo)
		if sel {
			hi, lo, body = colT, colT, rgb(0x2c2550)
		}
		if pick {
			hi, lo, body = colL, colB, colSlate
		}
		bevel(dst, r, body, hi, lo, false)
		if showKey {
			drawText(dst, fmt.Sprint(i+1), r.X+6, r.Y+9, textOpt{size: 9, display: true, col: colMuted})
		}
		if s != nil && !(dragging && dragFrom == i) {
			isz := r.W * 0.62
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
			op.GeoM.Translate((r.X+(r.W-isz)/2)*uiScale, (r.Y+(r.H-isz)/2)*uiScale)
			dst.DrawImage(items[s.ID].img, op)
			if !items[s.ID].tool {
				drawText(dst, fmt.Sprint(s.N), r.X+r.W-4, r.Y+r.H-8, textOpt{size: 12, align: text.AlignEnd, shadow: true})
			}
		}
	})
	return clicked
}

// Simple word wrap
func wrapText(s string, x, y, width, size float64, c color.Color, align text.Align) float64 {
	words := strings.Fields(s)
	line := ""
	for _, w := range words {
		try := strings.TrimSpace(line + " " + w)
		if textWidth(try, size, false) > width && line != "" {
			Text(line, x, y, size, c, align)
			y += size * 1.35
			line = w
		} else {
			line = try
		}
	}
	if line != "" {
		Text(line, x, y, size, c, align)
		y += size * 1.35
	}
	return y
}

// ---- Main menu ----

// Layout shared by the main menu art and its buttons
func mainLayout() (bx, topY, art, cx, colW float64) {
	art = math.Min(logicalH*0.36, 150)
	colW = math.Min(260, logicalW*0.36)
	gap := math.Max(30, logicalW*0.06)
	brandW := math.Max(art*16/22, 230)
	total := brandW + gap + colW
	bx = (logicalW-total)/2 + brandW/2
	topY = (logicalH - (art + 90)) / 2
	cx = bx + brandW/2 + gap
	return
}

// The mote, title and backdrop of the main menu (also shown behind its sub-menus)
func mainMenuArt() {
	bx, topY, art, _, _ := mainLayout()
	ops = append(ops, func(dst *ebiten.Image) {
		vector := color.NRGBA{14, 10, 32, 180}
		fill(dst, 0, 0, logicalW, logicalH, vector)
	})
	ops = append(ops, func(dst *ebiten.Image) { drawMoteArt(dst, bx, topY, art) })
	ts := math.Min(64, logicalH*0.14)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, "MOTE", bx+8, topY+art+ts*0.55+8, textOpt{size: ts, display: true, col: colO, align: text.AlignCenter})
		drawText(dst, "MOTE", bx+4, topY+art+ts*0.55+4, textOpt{size: ts, display: true, col: colS, align: text.AlignCenter})
		drawText(dst, "MOTE", bx, topY+art+ts*0.55, textOpt{size: ts, display: true, align: text.AlignCenter})
		drawText(dst, "A PIXEL SANDBOX", bx, topY+art+ts*1.2+10, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
}

func mainMenu() {
	mainMenuArt()
	_, _, _, cx, colW := mainLayout()
	bh := 46.0
	if isTouch {
		bh = 40
	}
	y := (logicalH - (bh+18)*4) / 2
	label := "Play"
	if started {
		label = "Continue"
	}
	if Button(Rect{cx, y, colW, bh}, label, styleGold, true, icoPlay) {
		play()
	}
	y += bh + 18
	if Button(Rect{cx, y, colW, bh}, "New world", styleViolet, true, icoWorld) {
		seedEntry = ""
		openScreen(scSeed)
	}
	y += bh + 18
	if Button(Rect{cx, y, colW, bh}, "Settings", styleSlate, true, icoGear) {
		settingsFrom = scMain
		openScreen(scSettings)
	}
	if !isTouch {
		y += bh + 18
		if Button(Rect{cx, y, colW, bh}, "Quit", styleSlate, true, icoX) {
			saveGame()
			os.Exit(0)
		}
	}
}

func drawMoteArt(dst *ebiten.Image, cx, top, h float64) {
	px := h / 22
	x0 := cx - 8*px
	blink := frame%200 < 7
	rows := playerFrame(blink, true, "tuck", 0)
	bob := math.Round(math.Sin(float64(frame)*0.07) * 1.2)
	for y, r := range rows[:12] {
		for x, ch := range r {
			if ch == '.' {
				continue
			}
			c := pcol[ch]
			if ch == 'T' {
				c = [3]int{255, 222 + int(math.Sin(float64(frame)*0.1)*20), 120}
				c[1] = min(255, c[1])
			}
			fill(dst, x0+float64(3+x)*px, top+float64(2+y)*px+bob*px, px, px, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
		}
	}
	// falling sparkles
	for k := 0; k < 4; k++ {
		ph := float64((frame+k*17)%60) / 60
		sx := x0 + float64(4+(k*5)%8)*px
		sy := top + (15+ph*6+bob)*px
		c := tipRGB
		if k%2 == 0 {
			c = glowRGB
		}
		fill(dst, sx, sy, px, px, color.NRGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), uint8(255 * (1 - ph))})
	}
}

// ---- New world (seed) ----

func seedMenu() {
	r := sheet(520, 250)
	if isTouch {
		r = sheet(560, 280)
	}
	if _, cl := head(r, "NEW WORLD", false); cl {
		openScreen(scMain)
		return
	}
	Label("SEED", r.X+18, r.Y+66)
	field := Rect{r.X + 18, r.Y + 80, r.W - 36 - 110, 38}
	shown := seedEntry
	if shown == "" {
		shown = "Empty = random world"
	}
	caret := ""
	if (frame/30)%2 == 0 {
		caret = "_"
	}
	Inset(field, colPanelLo)
	c := colInk
	if seedEntry == "" {
		c = colMuted
		caret = ""
	}
	Text(shown+caret, field.X+12, field.Y+field.H/2, 16, c, text.AlignStart)
	if Button(Rect{field.X + field.W + 16, field.Y, 94, 38}, "Random", styleSlate, true, nil) {
		seedEntry = randomSeed()
	}
	// typing on a keyboard
	for _, ch := range ebiten.AppendInputChars(nil) {
		if len(seedEntry) < 32 && ch >= 32 {
			seedEntry += string(ch)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(seedEntry) > 0 {
		seedEntry = seedEntry[:len(seedEntry)-1]
	}
	y := field.Y + field.H + 14
	if isTouch { // number pad for phones
		kw := (r.W - 36 - 11*6) / 12
		keys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "<", "C"}
		for i, k := range keys {
			kr := Rect{r.X + 18 + float64(i)*(kw+6), y, kw, 34}
			if Button(kr, k, styleSlate, true, nil) {
				switch k {
				case "<":
					if len(seedEntry) > 0 {
						seedEntry = seedEntry[:len(seedEntry)-1]
					}
				case "C":
					seedEntry = ""
				default:
					if len(seedEntry) < 9 {
						seedEntry += k
					}
				}
			}
		}
		y += 46
	} else {
		Text("Type a seed (numbers or words). The same seed always makes the same world.", r.X+18, y+8, 13, colMuted, text.AlignStart)
		y += 30
	}
	if Button(Rect{r.X + 18, y + 4, r.W - 36, 42}, "Create world", styleGold, true, nil) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		createWorld(seedEntry)
		starterKit()
		started = false
		play()
		saveGame()
	}
}

// ---- Pause ----

func pauseMenu() {
	r := sheet(760, 360)
	if _, cl := head(r, "PAUSED", false); cl {
		play()
		return
	}
	lw := math.Min(270, r.W*0.4)
	x, y := r.X+18, r.Y+66
	bh := 36.0
	if Button(Rect{x, y, lw, bh}, "Resume", styleGold, true, icoPlay) {
		play()
		return
	}
	y += bh + 16
	if Button(Rect{x, y, lw, bh}, "Settings", styleViolet, true, icoGear) {
		settingsFrom = scPause
		openScreen(scSettings)
		return
	}
	y += bh + 16
	Label("SEED", x, y+12)
	Text(seedText, x+48, y+12, 14, colInk, text.AlignStart)
	cl := "Copy"
	if frame < copiedUntil {
		cl = "Copied"
	}
	if canCopy() && Button(Rect{x + lw - 76, y, 76, 26}, cl, styleSlate, true, nil) {
		if copyText(seedText) {
			copiedUntil = frame + 90
		}
	}
	y += 26 + 16
	if Button(Rect{x, y, lw, bh}, "Debug", styleSlate, true, icoBug) {
		dbgInfo = "Tap an item to add it to your inventory."
		openScreen(scDebug)
		return
	}
	y += bh + 16
	if Button(Rect{x, y, lw, bh}, "Main menu", styleSlate, true, icoHome) {
		saveGame()
		openScreen(scMain)
		return
	}
	// controls for this device
	kr := Rect{x + lw + 22, r.Y + 66, r.W - lw - 58, r.H - 88}
	Inset(kr, colPanelLo)
	var rows [][2]string
	b := prefs.Binds
	if isTouch {
		Label("TOUCH CONTROLS", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", "Joystick left / right"}, {"Levitate", "Push the joystick up"}, {"Inventory", "Bag button"},
			{"Pause", "Pause button"}, {"Pick item", "Tap a hotbar slot"}, {"Use item", "Touch the world"},
			{"Look around", "Touch with no slot picked"}, {"Move joystick", "Hold it still, then drag"}}
	} else {
		Label("KEYBOARD AND MOUSE", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", keyLabel(b["left"]) + " / " + keyLabel(b["right"]) + " or arrows"}, {"Levitate", keyLabel(b["lev"]) + ", Up or Space"},
			{"Inventory", keyLabel(b["inv"])}, {"Pause", "Esc"}, {"Pick item", "1-5 or scroll"}, {"Use item", "Click the world"},
			{"Look around", "Hold click, no slot picked"}, {"Full screen", "F11"}}
	}
	ry := kr.Y + 40
	step := math.Min(24, (kr.H-48)/float64(len(rows)))
	for _, rw := range rows {
		Text(rw[0], kr.X+12, ry, 13, colMuted, text.AlignStart)
		Text(rw[1], kr.X+kr.W-12, ry, 13, colInk, text.AlignEnd)
		ry += step
	}
}

// ---- Inventory and crafting ----

func invMenu() {
	r := sheet(800, 380)
	if _, cl := head(r, "INVENTORY", false); cl {
		play()
		return
	}
	slot := 46.0
	if isTouch {
		slot = math.Max(32, math.Min(46, logicalH*0.105))
	}
	gap := 9.0
	x0, y := r.X+18, r.Y+66
	slotRects := make([]Rect, 15)
	Label("HOTBAR", x0, y)
	y += 12
	for i := 0; i < 5; i++ {
		slotRects[i] = Rect{x0 + float64(i)*(slot+gap), y, slot, slot}
	}
	y += slot + 22
	Label("BACKPACK", x0, y)
	y += 12
	for i := 5; i < 15; i++ {
		k := i - 5
		slotRects[i] = Rect{x0 + float64(k%5)*(slot+gap), y + float64(k/5)*(slot+gap), slot, slot}
	}
	infoY := y + 2*(slot+gap) + 10

	// dragging and tap-to-move
	if ptr.pressed {
		for i, sr := range slotRects {
			if sr.Has(ptr.x, ptr.y) && inv[i] != nil {
				dragFrom = i
			}
		}
	}
	if ptr.down && dragFrom >= 0 && math.Hypot(ptr.x-ptr.startX, ptr.y-ptr.startY) > 8 {
		dragging = true
		picked = -1
	}
	for i, sr := range slotRects {
		if slotWidget(sr, i, false, i == picked || (dragging && i == dragFrom), i < 5) {
			if picked < 0 {
				if inv[i] != nil {
					picked = i
					it := items[inv[i].ID]
					invInfo = it.name + ": " + it.desc + ". Tap another slot to move it."
				}
			} else {
				if picked != i {
					inv[picked], inv[i] = inv[i], inv[picked]
					sfx("place")
				}
				picked = -1
				invInfo = "Drag an item to another slot to move it."
			}
		}
	}
	if ptr.released {
		if dragging {
			for i, sr := range slotRects {
				if sr.Has(ptr.x, ptr.y) && i != dragFrom {
					inv[dragFrom], inv[i] = inv[i], inv[dragFrom]
					sfx("place")
				}
			}
		}
		dragging, dragFrom = false, -1
	}
	if dragging && dragFrom >= 0 && inv[dragFrom] != nil {
		img, gx, gy := items[inv[dragFrom].ID].img, ptr.x-20, ptr.y-20
		ops = append(ops, func(dst *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(40.0/8*uiScale, 40.0/8*uiScale)
			op.GeoM.Translate(gx*uiScale, gy*uiScale)
			dst.DrawImage(img, op)
		})
	}
	leftW := 5*slot + 4*gap
	wrapText(invInfo, x0, infoY, leftW, 12, colMuted, text.AlignStart)

	// crafting list
	cx := x0 + leftW + 26
	cw := r.X + r.W - 18 - cx
	Label("CRAFTING", cx, r.Y+66)
	ry := r.Y + 80
	rh := math.Min(52, (r.H-100)/float64(len(recipes))-8)
	for _, rc := range recipes {
		row := Rect{cx, ry, cw, rh}
		Inset(row, colSlateLo)
		it := items[rc.out]
		IconAt(it.img, row.X+8, row.Y+(rh-24)/2, 24)
		name := it.name
		if rc.n > 1 {
			name += fmt.Sprintf(" x%d", rc.n)
		}
		Text(name, row.X+42, row.Y+rh*0.32, 13, colInk, text.AlignStart)
		nx := row.X + 42
		for _, nd := range rc.needs {
			id, n := nd[0].(string), nd[1].(int)
			have := countItem(id)
			col := colMuted
			if have < n {
				col = colDanger
			}
			IconAt(items[id].img, nx, row.Y+rh*0.72-6, 12)
			s := fmt.Sprintf("%d/%d", min(have, n), n)
			Text(s, nx+15, row.Y+rh*0.72, 11, col, text.AlignStart)
			nx += 22 + textWidth(s, 11, false) + 8
		}
		if Button(Rect{row.X + row.W - 78, row.Y + (rh-28)/2, 68, 28}, "Craft", styleViolet, canCraft(rc), nil) {
			if msg := craft(rc); msg != "" {
				invInfo = msg
			}
		}
		ry += rh + 8
	}
}

// ---- Settings ----

func seg(x, y float64, labels []string, active int) int {
	clicked := -1
	for i, l := range labels {
		w := math.Max(38, textWidth(l, 14, false)+22)
		st := styleSlate
		if i == active {
			st = styleViolet
		}
		if Button(Rect{x, y, w, 30}, l, st, true, nil) {
			clicked = i
		}
		x += w + 10
	}
	return clicked
}

func settingsMenu() {
	r := sheet(760, 340)
	back, cl := head(r, "SETTINGS", true)
	if back || cl {
		listening = ""
		if settingsFrom == scMain || cl && settingsFrom == scMain {
			openScreen(scMain)
		} else if cl {
			play()
		} else {
			openScreen(scPause)
		}
		return
	}
	x, y := r.X+18, r.Y+66
	Label("FIELD OF VIEW", x, y)
	if c := seg(x, y+14, []string{"1", "2", "3", "4", "5"}, prefs.FOV-1); c >= 0 {
		prefs.FOV = c + 1
		applyFOV()
		savePrefs()
	}
	Text("Closer        Normal is 3        Wider", x, y+60, 11, colMuted, text.AlignStart)
	y += 82
	Label("MUSIC", x, y)
	if c := seg(x, y+14, []string{"On", "Off"}, b2i(!prefs.Music)); c >= 0 {
		prefs.Music = c == 0
		setVolumes()
		savePrefs()
	}
	y += 58
	Label("SOUND EFFECTS", x, y)
	if c := seg(x, y+14, []string{"On", "Off"}, b2i(!prefs.Sound)); c >= 0 {
		prefs.Sound = c == 0
		setVolumes()
		savePrefs()
	}
	if !isTouch {
		y += 58
		Label("FULL SCREEN", x, y)
		if c := seg(x, y+14, []string{"On", "Off"}, b2i(!prefs.Full)); c >= 0 {
			prefs.Full = c == 0
			ebiten.SetFullscreen(prefs.Full)
			savePrefs()
		}
	}

	cx := r.X + r.W/2 + 10
	cw := r.X + r.W - 18 - cx
	y = r.Y + 66
	Label("CONTROLS", cx, y)
	y += 16
	if isTouch {
		Label("JOYSTICK SIDE", cx, y+6)
		side := 0
		if prefs.StickX > 0.5 {
			side = 1
		}
		if c := seg(cx, y+20, []string{"Left", "Right"}, side); c >= 0 {
			prefs.StickX, prefs.StickY = []float64{0.14, 0.86}[c], 0.75
			savePrefs()
		}
		y += 64
		wrapText("You can also hold the joystick still for a moment and drag it anywhere.", cx, y, cw, 12, colMuted, text.AlignStart)
		y += 40
		Label("PAUSE AND BAG BUTTONS SIDE", cx, y)
		bs := 1
		if prefs.BtnSide == "left" {
			bs = 0
		}
		if c := seg(cx, y+14, []string{"Left", "Right"}, bs); c >= 0 {
			prefs.BtnSide = []string{"left", "right"}[c]
			savePrefs()
		}
	} else {
		acts := [][2]string{{"left", "Move left"}, {"right", "Move right"}, {"lev", "Levitate"}, {"inv", "Open inventory"}}
		for _, a := range acts {
			Text(a[1], cx, y+15, 14, colInk, text.AlignStart)
			l := keyLabel(prefs.Binds[a[0]])
			st := styleSlate
			if listening == a[0] {
				l, st = "Press...", styleGold
			}
			if Button(Rect{cx + cw - 90, y, 90, 30}, l, st, true, nil) {
				listening = a[0]
			}
			y += 42
		}
		wrapText("Click a key box, then press the new key (Esc cancels). Arrow keys and Space always work too.", cx, y+4, cw, 12, colMuted, text.AlignStart)
		if listening != "" {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				if k != ebiten.KeyEscape {
					prefs.Binds[listening] = k.String()
					savePrefs()
				}
				listening = ""
				break
			}
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---- Debug ----

func debugMenu() {
	r := sheet(720, 300)
	back, cl := head(r, "DEBUG", true)
	if back {
		openScreen(scPause)
		return
	}
	if cl {
		play()
		return
	}
	Label("GIVE ITEM", r.X+18, r.Y+66)
	cols := 4
	if r.W < 560 {
		cols = 3
	}
	bw := (r.W - 36 - float64(cols-1)*14) / float64(cols)
	for i, id := range itemOrder {
		it := items[id]
		br := Rect{r.X + 18 + float64(i%cols)*(bw+14), r.Y + 82 + float64(i/cols)*52, bw, 40}
		if Button(br, fmt.Sprintf("%s +%d", it.name, it.give), styleSlate, true, it.img) {
			got := addItem(id, it.give)
			sfx("give")
			if got > 0 {
				dbgInfo = fmt.Sprintf("Gave %d %s.", got, it.name)
			} else {
				dbgInfo = "Inventory is full."
			}
		}
	}
	y := r.Y + 82 + float64((len(itemOrder)+cols-1)/cols)*52 + 6
	hw := (r.W - 36 - 14) / 2
	if Button(Rect{r.X + 18, y, hw, 36}, "Fill energy", styleSlate, true, nil) {
		player.energy = 1
		sfx("gem")
		dbgInfo = "Levitation energy filled."
	}
	if Button(Rect{r.X + 32 + hw, y, hw, 36}, "Clear inventory", styleSlate, true, nil) {
		for i := range inv {
			inv[i] = nil
		}
		selected = -1
		dbgInfo = "Inventory cleared."
	}
	Text(dbgInfo, r.X+18, y+58, 13, colMuted, text.AlignStart)
}

// ---- Toast ----

func drawToast() {
	if frame >= toastUntil {
		return
	}
	w := textWidth(toastText, 11, true) + 28
	r := Rect{(logicalW - w) / 2, 14, w, 30}
	Panel(r)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, toastText, r.X+r.W/2, r.Y+r.H/2, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
}
