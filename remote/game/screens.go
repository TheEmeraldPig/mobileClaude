package game

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---- Main menu ----

func mainMenu() {
	mainMenuArt()
	_, _, _, cx, colW := mainLayout()
	bh := 46.0
	if isTouch {
		bh = 40
	}
	n := 4.0
	if !isTouch {
		n++
	}
	y := (logicalH - (bh+16)*n - 40) / 2
	if netState != netOff {
		Text(netStatus, cx+colW/2, y+bh/2, 14, colT, text.AlignCenter)
		y += bh + 16
		if Button(Rect{cx, y, colW, bh}, "Cancel", styleSlate, true, icoX) {
			leaveGame("")
		}
		return
	}
	online := serverURL() != ""
	st := styleGold
	if !online {
		st = styleSlate
	}
	if Button(Rect{cx, y, colW, bh}, "Play online", st, true, icoWorld) {
		if prefs.Name == "" {
			openName(true)
		} else {
			playOnline()
		}
	}
	y += bh + 16
	solo := styleSlate
	if !online {
		solo = styleGold
	}
	if Button(Rect{cx, y, colW, bh}, "Play solo", solo, true, icoPlay) {
		playSolo()
	}
	y += bh + 16
	name := prefs.Name
	if name == "" {
		name = "Pick a name"
	}
	if Button(Rect{cx, y, colW, bh}, "Name: "+name, styleSlate, true, icoBag) {
		openName(false)
	}
	y += bh + 16
	if Button(Rect{cx, y, colW, bh}, "Settings", styleSlate, true, icoGear) {
		settingsFrom = scMain
		openScreen(scSettings)
	}
	if !isTouch {
		y += bh + 16
		if Button(Rect{cx, y, colW, bh}, "Quit", styleSlate, true, icoX) {
			savePrefs()
			os.Exit(0)
		}
	}
	if netStatus != "" {
		wrapText(netStatus, cx+colW/2, y+bh+28, colW+40, 13, colT, text.AlignCenter)
	}
}

// ---- Name ----

var (
	nameEntry    string
	nameThenPlay bool
)

func openName(thenPlay bool) {
	nameEntry, nameThenPlay = prefs.Name, thenPlay
	if nameEntry == "" {
		nameEntry = randomName()
	}
	openScreen(scName)
}

var nameParts = [][]string{
	{"Pip", "Bram", "Fizz", "Nib", "Tuft", "Moss", "Dot", "Wisp", "Glim", "Puck", "Lumen", "Zip", "Rue", "Sol"},
	{"bean", "fern", "spark", "pebble", "drop", "bloom", "mote", "flick", "nook", "berry", "kin", "wick"},
}

func randomName() string {
	a, b := nameParts[0], nameParts[1]
	return a[frame*7%len(a)] + b[(frame*13+len(prefs.Key))%len(b)]
}

func nameMenu() {
	r := sheet(700, 380)
	if back, cl := head(r, "YOUR NAME", true); back || cl {
		openScreen(scMain)
		return
	}
	x, y := r.X+18, r.Y+62
	field := Rect{x, y, r.W - 36 - 110, 36}
	Inset(field, colPanelLo)
	caret := ""
	if (frame/30)%2 == 0 {
		caret = "_"
	}
	Text(nameEntry+caret, field.X+12, field.Y+18, 16, colInk, text.AlignStart)
	if Button(Rect{field.X + field.W + 14, y, 96, 36}, "Shuffle", styleSlate, true, nil) {
		nameEntry = randomName()
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if ch >= 32 && len([]rune(nameEntry)) < 16 {
			nameEntry += string(ch)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(nameEntry) > 0 {
		rs := []rune(nameEntry)
		nameEntry = string(rs[:len(rs)-1])
	}
	y += 52
	// letters on screen, for phones (and mice)
	rows := []string{"QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM"}
	// the widest row is the last: half a key of indent per row, 7 letters, Space and <
	kw := math.Min(52, math.Min((r.W-36-9*6)/10, (r.W-36-8*6)/11))
	for i, row := range rows {
		rx := x + float64(i)*kw*0.5
		for _, ch := range row {
			if Button(Rect{rx, y, kw, 34}, string(ch), styleSlate, true, nil) && len([]rune(nameEntry)) < 16 {
				if nameEntry == "" || strings.HasSuffix(nameEntry, " ") {
					nameEntry += string(ch)
				} else {
					nameEntry += strings.ToLower(string(ch))
				}
			}
			rx += kw + 6
		}
		if i == 2 {
			if Button(Rect{rx, y, kw * 1.6, 34}, "Space", styleSlate, true, nil) && len([]rune(nameEntry)) < 15 {
				nameEntry += " "
			}
			rx += kw*1.6 + 6
			if Button(Rect{rx, y, kw * 1.4, 34}, "<", styleSlate, true, nil) && len(nameEntry) > 0 {
				rs := []rune(nameEntry)
				nameEntry = string(rs[:len(rs)-1])
			}
		}
		y += 40
	}
	if Button(Rect{x, r.Y + r.H - 56, r.W - 36, 42}, "Done", styleGold, true, icoPlay) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		prefs.Name = mmo.CleanName(nameEntry)
		savePrefs()
		if netState == netOnline {
			send(mmo.CMsg{T: "name", Name: prefs.Name})
			play()
			return
		}
		openScreen(scMain)
		if nameThenPlay {
			playOnline()
		}
	}
}

// ---- Pause ----

func pauseMenu() {
	r := sheet(760, 400)
	if _, cl := head(r, "MENU", false); cl {
		play()
		return
	}
	lw := math.Min(260, r.W*0.4)
	x, y := r.X+18, r.Y+62
	bh, gap := 36.0, 11.0
	if Button(Rect{x, y, lw, bh}, "Resume", styleGold, true, icoPlay) {
		play()
		return
	}
	y += bh + gap
	if Button(Rect{x, y, lw, bh}, "Petals", styleViolet, true, icoBag) {
		openScreen(scPetals)
		return
	}
	y += bh + gap
	if Button(Rect{x, y, lw, bh}, "Travel", styleViolet, true, icoMap) {
		openScreen(scTravel)
		return
	}
	y += bh + gap
	if Button(Rect{x, y, lw, bh}, "Settings", styleSlate, true, icoGear) {
		settingsFrom = scPause
		openScreen(scSettings)
		return
	}
	y += bh + gap
	if Button(Rect{x, y, lw, bh}, "Leave", styleSlate, true, icoHome) {
		leaveGame("")
		return
	}
	y += bh + gap + 8
	if me != nil {
		Text(fmt.Sprintf("%s  -  level %d  -  %s kills", me.Name, me.Level, mmo.FmtNum(me.Kills)), x, y, 13, colMuted, text.AlignStart)
	}
	kr := Rect{x + lw + 22, r.Y + 66, r.W - lw - 58, r.H - 88}
	Inset(kr, colPanelLo)
	var rows [][2]string
	b := prefs.Binds
	if isTouch {
		Label("TOUCH CONTROLS", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", "Joystick left / right"}, {"Levitate", "Push the joystick up"}, {"Attack", "Hold the sword"},
			{"Defend", "Hold the shield"}, {"Petals", "Bag button, or tap the petal bar"}, {"Travel", "Map button"},
			{"Emotes", "Speech button"}, {"Look around", "Touch the world"}}
	} else {
		Label("KEYBOARD AND MOUSE", kr.X+12, kr.Y+16)
		rows = [][2]string{{"Move", keyLabel(b["left"]) + " / " + keyLabel(b["right"]) + " or arrows"}, {"Levitate", keyLabel(b["lev"]) + ", Up or Space"},
			{"Attack", "Hold the left mouse button"}, {"Defend", "Hold the right mouse button"}, {"Petals", keyLabel(b["petals"])},
			{"Travel", keyLabel(b["map"])}, {"Emotes", "1-6"}, {"Menu", "Esc"}, {"Full screen", "F11"}}
	}
	ry := kr.Y + 40
	step := math.Min(26, (kr.H-48)/float64(len(rows)))
	for _, rw := range rows {
		Text(rw[0], kr.X+12, ry, 13, colMuted, text.AlignStart)
		Text(rw[1], kr.X+kr.W-12, ry, 13, colInk, text.AlignEnd)
		ry += step
	}
}

// ---- Petals: loadout, bag, combining ----

var (
	pickSlot  = -1        // a loadout slot picked
	pickBag   mmo.PetalID // a bag petal picked
	bagScroll int
)

func petalsMenu() {
	r := sheet(900, 470)
	if _, cl := head(r, "PETALS", false); cl {
		play()
		return
	}
	if me == nil {
		return
	}
	infoW := math.Min(270, r.W*0.34)
	x, y := r.X+18, r.Y+60
	lw := r.W - 36 - infoW - 18
	Label("EQUIPPED", x, y+4)
	y += 16
	n := len(me.Load)
	ss := math.Min(52, (lw-float64(n-1)*8)/float64(n))
	for i, p := range me.Load {
		sr := Rect{x + float64(i)*(ss+8), y, ss, ss}
		if petalSlot(sr, p, pickSlot == i, false) {
			switch {
			case pickBag != "": // put the picked bag petal here
				send(mmo.CMsg{T: "equip", Slot: i, Petal: pickBag})
				pickBag, pickSlot = "", -1
			case pickSlot >= 0 && pickSlot != i:
				send(mmo.CMsg{T: "swap", Slot: pickSlot, Slot2: i})
				pickSlot = -1
			case pickSlot == i:
				pickSlot = -1
			default:
				pickSlot = i
			}
		}
	}
	y += ss + 18
	Label(fmt.Sprintf("BAG  -  %s COINS", mmo.FmtNum(me.Coins)), x, y+4)
	y += 16
	bag := me.Sorted()
	bs := ss
	cols := int((lw + 8) / (bs + 8))
	rowsFit := int((r.Y + r.H - 16 - y + 8) / (bs + 8))
	maxScroll := max(0, (len(bag)+cols-1)/cols-rowsFit)
	bagScroll = max(0, min(bagScroll, maxScroll))
	if _, wy := ebiten.Wheel(); wy != 0 {
		bagScroll = max(0, min(maxScroll, bagScroll-int(math.Copysign(1, wy))))
	}
	if len(bag) == 0 {
		Text("Your bag is empty. Defeat creatures to find petals.", x, y+20, 13, colMuted, text.AlignStart)
	}
	for i := bagScroll * cols; i < len(bag) && i < (bagScroll+rowsFit)*cols; i++ {
		k := i - bagScroll*cols
		br := Rect{x + float64(k%cols)*(bs+8), y + float64(k/cols)*(bs+8), bs, bs}
		p := bag[i]
		if petalSlot(br, p, pickBag == p, false) {
			if pickSlot >= 0 {
				send(mmo.CMsg{T: "equip", Slot: pickSlot, Petal: p})
				pickSlot, pickBag = -1, ""
			} else if pickBag == p {
				pickBag = ""
			} else {
				pickBag = p
			}
		}
		cnt := me.Inv[p]
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, fmt.Sprint(cnt), br.X+br.W-4, br.Y+9, textOpt{size: 11, align: text.AlignEnd, shadow: true})
		})
	}
	if maxScroll > 0 {
		if Button(Rect{x + lw - 30, r.Y + r.H - 46, 30, 30}, "v", styleSlate, bagScroll < maxScroll, nil) {
			bagScroll++
		}
		if Button(Rect{x + lw - 68, r.Y + r.H - 46, 30, 30}, "^", styleSlate, bagScroll > 0, nil) {
			bagScroll--
		}
	}
	// details of what's picked
	ir := Rect{r.X + r.W - 18 - infoW, r.Y + 60, infoW, r.H - 78}
	Inset(ir, colPanelLo)
	var p mmo.PetalID
	inSlot := false
	switch {
	case pickBag != "":
		p = pickBag
	case pickSlot >= 0 && pickSlot < len(me.Load):
		p, inSlot = me.Load[pickSlot], true
	}
	k, rar, ok := p.Split()
	if !ok {
		hint := "Tap a petal in your bag, then a slot to equip it. Tap two slots to swap them."
		if !isTouch {
			hint = "Click a petal in your bag, then a slot to equip it. Click two slots to swap them."
		}
		wrapText(hint, ir.X+12, ir.Y+20, ir.W-24, 13, colMuted, text.AlignStart)
		wrapText("Five of the same petal combine into one of the next rarity.", ir.X+12, ir.Y+100, ir.W-24, 13, colMuted, text.AlignStart)
		return
	}
	rc := mmo.Rarities[rar].Col
	Text(k.Name, ir.X+12, ir.Y+20, 18, colInk, text.AlignStart)
	Text(mmo.Rarities[rar].Name, ir.X+12, ir.Y+42, 14, color.NRGBA{uint8(rc[0]), uint8(rc[1]), uint8(rc[2]), 255}, text.AlignStart)
	ty := wrapText(k.Desc, ir.X+12, ir.Y+66, ir.W-24, 13, colMuted, text.AlignStart) + 6
	pow := mmo.Power(rar)
	stats := [][2]string{}
	if k.Dmg > 0 {
		stats = append(stats, [2]string{"Damage", mmo.FmtNum(int64(k.Dmg * pow))})
	}
	stats = append(stats, [2]string{"Health", mmo.FmtNum(int64(k.HP * pow))}, [2]string{"Grows back", fmt.Sprintf("%.1fs", k.Reload)})
	if k.Heal > 0 {
		stats = append(stats, [2]string{"Heals", fmt.Sprintf("%.1f/s", k.Heal*mmo.HealMult(rar))})
	}
	if k.Mend > 0 {
		stats = append(stats, [2]string{"Heals", fmt.Sprintf("%.0f", k.Mend*mmo.HealMult(rar))})
	}
	if k.Burn > 0 {
		stats = append(stats, [2]string{"Burn", fmt.Sprintf("%s/s", mmo.FmtNum(int64(k.Burn*pow)))})
	}
	if k.Slow > 0 {
		stats = append(stats, [2]string{"Slows", fmt.Sprintf("%.2fs", k.Slow+0.25*float64(rar))})
	}
	for _, s := range stats {
		Text(s[0], ir.X+12, ty, 13, colMuted, text.AlignStart)
		Text(s[1], ir.X+ir.W-12, ty, 13, colInk, text.AlignEnd)
		ty += 20
	}
	by := ir.Y + ir.H - 44
	if inSlot {
		if Button(Rect{ir.X + 10, by, ir.W - 20, 34}, "Take off", styleSlate, true, nil) {
			send(mmo.CMsg{T: "equip", Slot: pickSlot, Petal: ""})
			pickSlot = -1
		}
		return
	}
	if rar < mmo.NRarity-1 {
		cost := mmo.CombineCost(rar)
		have := me.Inv[p]
		can := have >= mmo.CombineN && me.Coins >= cost
		label := fmt.Sprintf("Combine 5 (%s coins)", mmo.FmtNum(cost))
		if Button(Rect{ir.X + 10, by, ir.W - 20, 34}, label, styleGold, can, nil) {
			send(mmo.CMsg{T: "combine", Petal: p})
		}
		Text(fmt.Sprintf("You have %d", have), ir.X+12, by-14, 12, colMuted, text.AlignStart)
	}
	if Button(Rect{ir.X + 10, by - 76, ir.W - 20, 34}, "Equip", styleViolet, true, nil) {
		slot := -1
		for i, l := range me.Load {
			if l == "" {
				slot = i
				break
			}
		}
		if slot < 0 {
			toast("Pick a slot to swap it into")
			return
		}
		send(mmo.CMsg{T: "equip", Slot: slot, Petal: p})
		pickBag = ""
	}
}

// ---- Travel ----

var travelPick string

func travelMenu() {
	r := sheet(900, 470)
	if _, cl := head(r, "TRAVEL", false); cl {
		play()
		return
	}
	infoW := math.Min(270, r.W*0.34)
	mr := Rect{r.X + 18, r.Y + 60, r.W - 54 - infoW, r.H - 78}
	ops = append(ops, func(dst *ebiten.Image) {
		bevel(dst, mr, rgb(0x1d3a6e), rgb(0x2c5aa0), rgb(0x142a52), false)
		// a few waves
		for i := 0; i < 26; i++ {
			wx := mr.X + math.Mod(float64(i*97+frame/3), mr.W-20) + 6
			wy := mr.Y + math.Mod(float64(i*53), mr.H-20) + 10
			fill(dst, wx, wy, 9, 3, rgb(0x3a6ab8))
		}
	})
	if travelPick == "" {
		travelPick = isleID
	}
	for _, d := range mmo.Isles {
		cx, cy := mr.X+mr.W*d.MapX, mr.Y+mr.H*d.MapY
		land, sand := rgb(0x58a84a), rgb(0xe2c26b)
		switch d.Biome {
		case mmo.BiomeHollow:
			land = rgb(0x4a6a8a)
		case mmo.BiomeEmber:
			land, sand = rgb(0xc0603a), rgb(0xcca860)
		}
		ops = append(ops, func(dst *ebiten.Image) { // a little pixel island under the name
			const p = 4.0
			for dy := -7.0; dy <= 7; dy++ {
				hw := math.Floor(math.Sqrt(1-dy*dy/56)*20) * p
				fill(dst, cx-hw-p, cy+dy*p, hw*2+2*p, p, sand)
				if dy > -6 && dy < 6 {
					fill(dst, cx-hw+p, cy+dy*p, hw*2-2*p, p, land)
				}
			}
		})
		w := math.Max(110, textWidth(d.Name, 14, false)+30)
		br := Rect{cx - w/2, cy - 20, w, 40}
		st := styleSlate
		if d.ID == travelPick {
			st = styleViolet
		}
		if d.ID == isleID {
			st = styleGold
		}
		if Button(br, d.Name, st, true, nil) {
			travelPick = d.ID
		}
		lbl := fmt.Sprintf("Level %d+", d.Level)
		if d.Safe {
			lbl = "Town"
		}
		Text(lbl, cx, br.Y+br.H+16, 12, colInk, text.AlignCenter)
	}
	ir := Rect{r.X + r.W - 18 - infoW, r.Y + 60, infoW, r.H - 78}
	Inset(ir, colPanelLo)
	d := mmo.IsleByID(travelPick)
	if d == nil {
		return
	}
	Text(d.Name, ir.X+12, ir.Y+20, 18, colInk, text.AlignStart)
	sub := fmt.Sprintf("Recommended level %d", d.Level)
	if d.Safe {
		sub = "Safe town"
	}
	Text(sub, ir.X+12, ir.Y+42, 13, colT, text.AlignStart)
	ty := wrapText(d.Blurb, ir.X+12, ir.Y+66, ir.W-24, 13, colMuted, text.AlignStart) + 8
	if len(d.Mobs) > 0 {
		names := []string{}
		for _, m := range d.Mobs {
			names = append(names, mmo.MobByID(m.Kind).Name)
		}
		wrapText("Creatures: "+strings.Join(names, ", "), ir.X+12, ty, ir.W-24, 12, colInk, text.AlignStart)
	}
	if !d.Safe {
		wrapText("Stay here when you log off and your mote keeps fighting while you're away (up to 12 hours).", ir.X+12, ir.Y+ir.H-110, ir.W-24, 12, colMuted, text.AlignStart)
	}
	if d.ID == isleID {
		Text("You're here", ir.X+ir.W/2, ir.Y+ir.H-28, 14, colT, text.AlignCenter)
		return
	}
	if Button(Rect{ir.X + 10, ir.Y + ir.H - 46, ir.W - 20, 36}, "Travel here", styleGold, true, icoPlay) {
		send(mmo.CMsg{T: "travel", Isle: d.ID})
		toast("Travelling to " + d.Name + "...")
	}
}

// ---- Knocked out ----

func deadMenu() {
	r := sheet(460, 230)
	Title("KNOCKED OUT", r.X+r.W/2-textWidth("KNOCKED OUT", 17, true)/2, r.Y+32)
	wrapText("You keep everything. Get back up here, or go back to Haven.", r.X+r.W/2, r.Y+70, r.W-40, 13, colMuted, text.AlignCenter)
	bw := (r.W - 46) / 2
	if Button(Rect{r.X + 18, r.Y + r.H - 60, bw, 42}, "Get up", styleGold, true, icoPlay) {
		send(mmo.CMsg{T: "respawn"})
	}
	if Button(Rect{r.X + 28 + bw, r.Y + r.H - 60, bw, 42}, "Haven", styleSlate, true, icoHome) {
		send(mmo.CMsg{T: "respawn", Isle: "haven"})
	}
}

// ---- While you were away ----

func awayMenu() {
	a := awayNews
	if a == nil {
		play()
		return
	}
	r := sheet(560, 320)
	Title("WHILE YOU WERE AWAY", r.X+18, r.Y+30)
	h, m := a.Secs/3600, (a.Secs%3600)/60
	where := a.Isle
	if d := mmo.IsleByID(a.Isle); d != nil {
		where = d.Name
	}
	x, y := r.X+18, r.Y+64
	Text(fmt.Sprintf("Your mote kept fighting on %s for %dh %02dm.", where, h, m), x, y, 14, colInk, text.AlignStart)
	y += 30
	rows := [][2]string{{"Creatures beaten", mmo.FmtNum(a.Kills)}, {"Experience", "+" + mmo.FmtNum(a.XP)}, {"Coins", "+" + mmo.FmtNum(a.Coins)}}
	if a.Levels > 0 {
		rows = append(rows, [2]string{"Levels", fmt.Sprintf("+%d", a.Levels)})
	}
	for _, rw := range rows {
		Text(rw[0], x, y, 14, colMuted, text.AlignStart)
		Text(rw[1], r.X+r.W-18, y, 14, colT, text.AlignEnd)
		y += 22
	}
	px := x
	for p, n := range a.Petals {
		k, rar, ok := p.Split()
		if !ok || px > r.X+r.W-60 {
			continue
		}
		pr := Rect{px, y + 4, 40, 40}
		petalSlot(pr, p, false, false)
		_ = k
		_ = rar
		cnt := n
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, fmt.Sprint(cnt), pr.X+pr.W-4, pr.Y+9, textOpt{size: 11, align: text.AlignEnd, shadow: true})
		})
		px += 48
	}
	if Button(Rect{r.X + 18, r.Y + r.H - 56, r.W - 36, 42}, "Nice!", styleGold, true, icoPlay) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		awayNews = nil
		play()
	}
}

// ---- Settings: server ----

func serverSettings(x, y float64, r Rect) {
	Label("ONLINE SERVER", x, y)
	shown := prefs.Server
	if shown == "" {
		shown = ServerURL
		if shown == "" {
			shown = "(none built in)"
		}
	}
	field := Rect{x, y + 14, r.W - 36 - 110, 32}
	Inset(field, colPanelLo)
	Text(shown, field.X+10, field.Y+16, 14, colInk, text.AlignStart)
	if Button(Rect{field.X + field.W + 14, field.Y, 96, 32}, "Default", styleSlate, prefs.Server != "", nil) {
		prefs.Server = ""
		savePrefs()
	}
	if !isTouch {
		for _, ch := range ebiten.AppendInputChars(nil) {
			if ch > 32 && len(prefs.Server) < 120 {
				prefs.Server += string(ch)
				savePrefs()
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(prefs.Server) > 0 {
			prefs.Server = prefs.Server[:len(prefs.Server)-1]
			savePrefs()
		}
		wrapText("Type to change the address (for your own server). Default goes back to the official one.", x, y+64, r.W-36, 12, colMuted, text.AlignStart)
	}
}
