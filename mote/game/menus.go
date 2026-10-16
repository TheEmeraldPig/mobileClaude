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
)

var (
	cur          = scMain
	settingsFrom = scPause
	settingsTab  int
	started      bool
	toastText    string
	toastUntil   int
	hintUntil    int
	invInfo      string
	seedEntry    string
	listening    string // action waiting for a new key
	dragFrom     = -1
	dragging     bool
	picked       = -1
	copiedUntil  int
	nameUntil    int
	recipeScroll int
	lastTapSlot  = -1
	lastTapFrame int
)

func paused() bool { return cur != scPlaying }

func openScreen(s screen) {
	if s == scInv {
		picked, dragFrom, dragging = -1, -1, false
		invInfo = defaultInvInfo()
		sfx("open")
	}
	if s == scPause && cur == scPlaying {
		sfx("close")
	}
	cur = s
	clearPointers()
}

func defaultInvInfo() string {
	if isTouch {
		return "Drag items to move them. Double-tap to send to the hotbar or backpack. Drag out of this window to drop."
	}
	return "Drag items to move them. Shift-click: hotbar <-> backpack. Right-click: split a stack. Drag out to drop."
}

func toast(s string) { toastText, toastUntil = s, frame+150 }

func showItemName() { nameUntil = frame + 90 }

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

func hotbarSlotSize() float64 {
	if isTouch {
		return math.Max(36, math.Min(46, logicalH*0.11))
	}
	return 52
}

var heartRows = []string{".XX.XX.", "XXXXXXX", "XXXXXXX", ".XXXXX.", "..XXX..", "...X..."}

func drawHearts(dst *ebiten.Image, x, y float64) {
	const px = 3.0
	for h := 0; h < maxHP/2; h++ {
		hx := x + float64(h)*(8*px)
		fullness := player.hp - h*2 // 2 full, 1 half, <=0 empty
		for r, row := range heartRows {
			for c, ch := range row {
				if ch != 'X' {
					continue
				}
				col := color.Color(color.NRGBA{40, 30, 60, 220})
				if fullness >= 2 || (fullness == 1 && c < 4) {
					col = color.NRGBA{240, 86, 112, 255}
					if r == 1 && c <= 2 {
						col = color.NRGBA{255, 170, 180, 255}
					}
				}
				fill(dst, hx+float64(c)*px, y+float64(r)*px, px, px, col)
			}
		}
	}
}

func hud() {
	hudRects = hudRects[:0]
	ops = append(ops, func(dst *ebiten.Image) { drawHearts(dst, 14, 12) })
	// Hotbar
	slot := hotbarSlotSize()
	gap := 7.0
	tw := hotbarN*slot + (hotbarN-1)*gap
	x0, y0 := (logicalW-tw)/2, logicalH-slot-14
	bar := Rect{x0 - 7, y0 - 7, tw + 14, slot + 14}
	hudRects = append(hudRects, bar)
	ops = append(ops, func(dst *ebiten.Image) { fill(dst, bar.X, bar.Y, bar.W, bar.H, color.NRGBA{19, 16, 39, 200}) })
	for i := 0; i < hotbarN; i++ {
		r := Rect{x0 + float64(i)*(slot+gap), y0, slot, slot}
		if slotWidget(r, i, i == selected, false, true) {
			if selected == i {
				selected = -1
			} else {
				selected = i
			}
			showItemName()
		}
	}
	if frame < nameUntil && selected >= 0 && inv[selected] != nil {
		name := items[inv[selected].ID].name
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, name, logicalW/2, y0-22, textOpt{size: 14, align: text.AlignCenter, shadow: true})
		})
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
		cx, cy := stickCenter()
		ops = append(ops, func(dst *ebiten.Image) { drawStick(dst, cx, cy) })
	}
	if frame < hintUntil {
		hint := "A / D to move, W to levitate, E for inventory, Q to drop, Esc to pause. Scroll or 1-8 to pick an item, then click the world to use it."
		if isTouch {
			hint = "Push the stick up to levitate. Pick a hotbar slot, then touch the world to use it. With no slot picked, touch to look around."
		}
		wrapText(hint, logicalW-14, 22, math.Min(420, logicalW*0.45), 13, colMuted, text.AlignEnd)
	}
}

func drawStick(dst *ebiten.Image, cx, cy float64) {
	pc := func(x, y, r float64, c color.Color) { // stepped pixel circle
		for dy := -r; dy <= r; dy += 3 {
			w := math.Floor(math.Sqrt(r*r-dy*dy)/3) * 3
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
	hover := !isTouch && r.Has(hoverX, hoverY)
	ops = append(ops, func(dst *ebiten.Image) {
		hi, lo := color.Color(color.NRGBA{10, 8, 23, 255}), color.Color(colPanelHi)
		body := color.Color(colPanelLo)
		if hover {
			body = rgb(0x1c1838)
		}
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
	r := sheet(780, 360)
	if _, cl := head(r, "PAUSED", false); cl {
		play()
		return
	}
	lw := math.Min(270, r.W*0.4)
	x, y := r.X+18, r.Y+66
	bh := 38.0
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
	if Button(Rect{x, y, lw, bh}, "Main menu", styleSlate, true, icoHome) {
		saveGame()
		openScreen(scMain)
		return
	}
	y += bh + 20
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
	y += 40
	day := "Day"
	if isNight() {
		day = "Night"
	}
	Text(fmt.Sprintf("%s  -  %02d:%02d", day, int(worldTime*24), int(math.Mod(worldTime*24*60, 60))), x, y+6, 13, colMuted, text.AlignStart)

	// controls for this device
	kr := Rect{x + lw + 22, r.Y + 66, r.W - lw - 58, r.H - 88}
	Inset(kr, colPanelLo)
	var rows [][2]string
	b := prefs.Binds
	if isTouch {
		Label("TOUCH CONTROLS", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", "Joystick left / right"}, {"Levitate", "Push the joystick up"}, {"Inventory", "Bag button"},
			{"Pause", "Pause button"}, {"Pick item", "Tap a hotbar slot"}, {"Use item", "Touch the world"},
			{"Look around", "Touch with no slot picked"}, {"Drop", "Drag out of the inventory"}}
	} else {
		Label("KEYBOARD AND MOUSE", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", keyLabel(b["left"]) + " / " + keyLabel(b["right"]) + " or arrows"}, {"Levitate", keyLabel(b["lev"]) + ", Up or Space"},
			{"Inventory", keyLabel(b["inv"])}, {"Drop item", keyLabel(b["drop"]) + " (Ctrl for the stack)"}, {"Pause", "Esc"},
			{"Pick item", "1-8 or scroll"}, {"Use item", "Click the world"}, {"Look around", "Hold click, no slot picked"}, {"Full screen", "F11"}}
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
	r := sheet(940, 470)
	if _, cl := head(r, "INVENTORY", false); cl {
		play()
		return
	}
	gap := 7.0
	labels, top := 2*22.0, 60.0
	if isTouch { // phones: no section labels, so the slots can be bigger
		labels, top = 10, 52
	}
	availH := r.H - top - labels - 50
	slot := math.Min(52, (availH-5*gap)/5)
	slot = math.Min(slot, (r.W*0.56-7*gap)/8)
	x0, y := r.X+18, r.Y+top
	slotRects := make([]Rect, invSize)
	if !isTouch {
		Label("HOTBAR", x0, y+4)
		y += 16
	}
	for i := 0; i < hotbarN; i++ {
		slotRects[i] = Rect{x0 + float64(i)*(slot+gap), y, slot, slot}
	}
	if isTouch {
		y += slot + 10
	} else {
		y += slot + 14
		Label("BACKPACK", x0, y+4)
		y += 16
	}
	for i := hotbarN; i < invSize; i++ {
		k := i - hotbarN
		slotRects[i] = Rect{x0 + float64(k%8)*(slot+gap), y + float64(k/8)*(slot+gap), slot, slot}
	}
	leftW := 8*slot + 7*gap
	by := y + 4*(slot+gap) + 6

	// hover info and desktop shortcuts
	hoverSlot := -1
	for i, sr := range slotRects {
		if !isTouch && sr.Has(hoverX, hoverY) {
			hoverSlot = i
		}
	}
	if hoverSlot >= 0 && inv[hoverSlot] != nil && !dragging {
		it := items[inv[hoverSlot].ID]
		invInfo = it.name + ": " + it.desc + "."
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			splitStack(hoverSlot)
			sfx("place")
		}
	}

	// dragging, tap-to-move, double-tap / shift-click quick move
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
		if !slotWidget(sr, i, false, i == picked || (dragging && i == dragFrom), i < hotbarN) {
			continue
		}
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		double := isTouch && lastTapSlot == i && frame-lastTapFrame < 20
		lastTapSlot, lastTapFrame = i, frame
		switch {
		case (shift || double) && inv[i] != nil:
			quickMove(i)
			picked = -1
			sfx("place")
		case picked < 0:
			if inv[i] != nil {
				picked = i
				it := items[inv[i].ID]
				invInfo = it.name + ": " + it.desc + ". Tap another slot to move it."
			}
		default:
			if picked != i {
				inv[picked], inv[i] = inv[i], inv[picked]
				sfx("place")
			}
			picked = -1
		}
	}
	if ptr.released {
		if dragging && dragFrom >= 0 && inv[dragFrom] != nil {
			moved := false
			for i, sr := range slotRects {
				if sr.Has(ptr.x, ptr.y) {
					if i != dragFrom {
						inv[dragFrom], inv[i] = inv[i], inv[dragFrom]
						sfx("place")
					}
					moved = true
				}
			}
			if !moved && !r.Has(ptr.x, ptr.y) { // dragged out of the window: drop it
				it := items[inv[dragFrom].ID]
				n := inv[dragFrom].N
				dropSlot(dragFrom, n)
				invInfo = fmt.Sprintf("Dropped %d %s.", n, it.name)
			}
		}
		dragging, dragFrom = false, -1
	}
	if dragging && dragFrom >= 0 && inv[dragFrom] != nil {
		img, gx, gy := items[inv[dragFrom].ID].img, ptr.x-22, ptr.y-22
		ops = append(ops, func(dst *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(44.0/8*uiScale, 44.0/8*uiScale)
			op.GeoM.Translate(gx*uiScale, gy*uiScale)
			dst.DrawImage(img, op)
		})
	}

	// buttons under the backpack
	bw := 92.0
	if Button(Rect{x0, by, bw, 30}, "Sort", styleSlate, true, nil) {
		sortBackpack()
		picked = -1
		invInfo = "Backpack sorted."
	}
	canDrop := picked >= 0 && inv[picked] != nil
	if Button(Rect{x0 + bw + 12, by, bw, 30}, "Drop", styleSlate, canDrop, nil) {
		n := inv[picked].N
		name := items[inv[picked].ID].name
		dropSlot(picked, n)
		invInfo = fmt.Sprintf("Dropped %d %s.", n, name)
		picked = -1
	}
	info := invInfo
	if isTouch && picked < 0 && !strings.HasPrefix(info, "Crafted") && !strings.HasPrefix(info, "Dropped") && !strings.HasPrefix(info, "No room") && info != "Backpack sorted." {
		info = "Double-tap: hotbar/backpack. Drag out: drop."
	} else if isTouch && picked >= 0 && inv[picked] != nil {
		info = items[inv[picked].ID].name + ": tap a slot to move it"
	}
	wrapText(info, x0+2*bw+30, by+8, leftW-2*bw-30, 12, colMuted, text.AlignStart)

	// crafting list, scrollable
	cx := x0 + leftW + 24
	cw := r.X + r.W - 18 - cx
	Label("CRAFTING", cx, r.Y+64)
	listTop, listBottom := r.Y+82, r.Y+r.H-16
	rh := 46.0
	if isTouch {
		rh = 40
	}
	visible := int((listBottom - listTop) / (rh + 7))
	maxScroll := max(0, len(recipes)-visible)
	list := Rect{cx, listTop, cw, listBottom - listTop}
	if _, wy := ebiten.Wheel(); wy != 0 && list.Has(hoverX, hoverY) {
		recipeScroll -= sign(wy)
	}
	if maxScroll > 0 {
		if Button(Rect{cx + cw - 64, r.Y + 52, 28, 22}, "<", styleSlate, recipeScroll > 0, nil) {
			recipeScroll--
		}
		if Button(Rect{cx + cw - 30, r.Y + 52, 28, 22}, ">", styleSlate, recipeScroll < maxScroll, nil) {
			recipeScroll++
		}
	}
	recipeScroll = max(0, min(maxScroll, recipeScroll))
	ry := listTop
	for k := recipeScroll; k < len(recipes) && k < recipeScroll+visible; k++ {
		rc := recipes[k]
		row := Rect{cx, ry, cw, rh}
		Inset(row, colSlateLo)
		it := items[rc.out]
		IconAt(it.img, row.X+8, row.Y+(rh-24)/2, 24)
		name := it.name
		if rc.n > 1 {
			name += fmt.Sprintf(" x%d", rc.n)
		}
		Text(name, row.X+42, row.Y+rh*0.3, 13, colInk, text.AlignStart)
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
		if Button(Rect{row.X + row.W - 74, row.Y + (rh-26)/2, 64, 26}, "Craft", styleViolet, canCraft(rc), nil) {
			if msg := craft(rc); msg != "" {
				invInfo = msg
			}
		}
		ry += rh + 7
	}
}

// ---- Settings (tabs: display, sound, controls) ----

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
	r := sheet(700, 380)
	back, cl := head(r, "SETTINGS", true)
	if back || cl {
		listening = ""
		switch {
		case settingsFrom == scMain:
			openScreen(scMain)
		case cl:
			play()
		default:
			openScreen(scPause)
		}
		return
	}
	tabs := []string{"Display", "Sound", "Controls"}
	if c := seg(r.X+18, r.Y+58, tabs, settingsTab); c >= 0 {
		settingsTab = c
		listening = ""
	}
	ops = append(ops, func(dst *ebiten.Image) { fill(dst, r.X+18, r.Y+100, r.W-36, 3, colPanelHi) })
	x, y := r.X+18, r.Y+120
	lvls := []string{"Off", "1", "2", "3", "4", "5"}
	switch settingsTab {
	case 0:
		Label("FIELD OF VIEW", x, y)
		if c := seg(x, y+14, []string{"1", "2", "3", "4", "5"}, prefs.FOV-1); c >= 0 {
			prefs.FOV = c + 1
			applyFOV()
			savePrefs()
		}
		Text("1 is closest, 5 shows the most. 3 is normal.", x, y+62, 12, colMuted, text.AlignStart)
		if !isTouch {
			y += 88
			Label("FULL SCREEN", x, y)
			if c := seg(x, y+14, []string{"On", "Off"}, b2i(!prefs.Full)); c >= 0 {
				prefs.Full = c == 0
				ebiten.SetFullscreen(prefs.Full)
				savePrefs()
			}
			Text("F11 also switches full screen.", x, y+62, 12, colMuted, text.AlignStart)
		}
	case 1:
		Label("MUSIC VOLUME", x, y)
		if c := seg(x, y+14, lvls, prefs.MusicVol); c >= 0 {
			prefs.MusicVol = c
			setVolumes()
			savePrefs()
		}
		y += 70
		Label("SOUND EFFECTS VOLUME", x, y)
		if c := seg(x, y+14, lvls, prefs.SfxVol); c >= 0 {
			prefs.SfxVol = c
			setVolumes()
			savePrefs()
		}
	case 2:
		if isTouch {
			Label("JOYSTICK SIDE", x, y)
			side := 0
			if prefs.StickX > 0.5 {
				side = 1
			}
			if c := seg(x, y+14, []string{"Left", "Right"}, side); c >= 0 {
				prefs.StickX, prefs.StickY = []float64{0.12, 0.88}[c], 0.62
				savePrefs()
			}
			wrapText("You can also hold the joystick still for a moment and drag it anywhere.", x+200, y+20, r.W-240, 12, colMuted, text.AlignStart)
			y += 70
			Label("PAUSE AND BAG BUTTONS SIDE", x, y)
			bs := 1
			if prefs.BtnSide == "left" {
				bs = 0
			}
			if c := seg(x, y+14, []string{"Left", "Right"}, bs); c >= 0 {
				prefs.BtnSide = []string{"left", "right"}[c]
				savePrefs()
			}
		} else {
			acts := [][2]string{{"left", "Move left"}, {"right", "Move right"}, {"lev", "Levitate"}, {"inv", "Open inventory"}, {"drop", "Drop item"}}
			colW := (r.W - 36 - 30) / 2
			for k, a := range acts {
				ax := x + float64(k%2)*(colW+30)
				ay := y + float64(k/2)*44
				Text(a[1], ax, ay+15, 14, colInk, text.AlignStart)
				l := keyLabel(prefs.Binds[a[0]])
				st := styleSlate
				if listening == a[0] {
					l, st = "Press...", styleGold
				}
				if Button(Rect{ax + colW - 96, ay, 96, 30}, l, st, true, nil) {
					listening = a[0]
				}
			}
			wrapText("Click a key box, then press the new key (Esc cancels). Arrow keys and Space always work too.", x, y+3*44+4, r.W-36, 12, colMuted, text.AlignStart)
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
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
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
