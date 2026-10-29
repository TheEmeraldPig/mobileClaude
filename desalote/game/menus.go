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

// Screens and menus, drawn with the pixel widgets in ui.go.

type screen int

const (
	scMain screen = iota
	scSettings
	scJoin
	scHelp
	scTable // the lobby and the whole match (see table.go)
)

var (
	cur            = scMain
	settingsFrom   = scMain
	helpFrom       = scMain
	helpPage       int
	toastText      string
	toastUntil     int
	screenOpenedAt int
)

func openScreen(s screen) {
	if s != cur {
		sfx("open")
	}
	cur = s
	screenOpenedAt = frame
	clearPointers()
}

func toast(s string) { toastText, toastUntil = s, frame+170 }

// sheet returns a centred panel rect
func sheet(w, h float64) Rect {
	w = math.Min(w, logicalW-20)
	h = math.Min(h, logicalH-16)
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

func wrapText(s string, x, y, width, size float64, c color.Color, align text.Align) float64 {
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, w := range strings.Fields(para) {
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
		}
		y += size * 1.35
	}
	return y
}

// How tall wrapped text will be
func wrapHeight(s string, width, size float64) float64 {
	h := 0.0
	for _, para := range strings.Split(s, "\n") {
		line := ""
		n := 1
		for _, w := range strings.Fields(para) {
			try := strings.TrimSpace(line + " " + w)
			if textWidth(try, size, false) > width && line != "" {
				n++
				line = w
			} else {
				line = try
			}
		}
		h += float64(n) * size * 1.35
	}
	return h
}

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

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---- Main menu ----

var menuCards = []Card{{R: 14, S: Spades}, {R: 13, S: Hearts}, {R: 12, S: Clubs}, {R: 11, S: Diamonds}}

// The name, each letter bobbing a little out of step with the others
func drawTitle(dst *ebiten.Image, cx, y, size float64) {
	const name = "DESALOTE"
	w := textWidth(name, size, true)
	x := cx - w/2
	for i, ch := range name {
		l := string(ch)
		lw := textWidth(l, size, true)
		ly := y + math.Sin(float64(frame)/22+float64(i)*0.7)*size*0.05
		lx := x + lw/2
		drawText(dst, l, lx+4, ly+5, textOpt{size: size, display: true, col: colO, align: text.AlignCenter})
		drawText(dst, l, lx+2, ly+2.5, textOpt{size: size, display: true, col: colOrangeL, align: text.AlignCenter})
		drawText(dst, l, lx, ly, textOpt{size: size, display: true, col: colL, align: text.AlignCenter})
		x += lw
	}
}

func mainMenu() {
	// the brand: a fan of cards and the name, on the left
	colW := math.Min(270, logicalW*0.34)
	brandW := math.Min(logicalW*0.5, 440)
	gap := math.Max(24, logicalW*0.05)
	bx := (logicalW-(brandW+gap+colW))/2 + brandW/2
	cx := bx + brandW/2 + gap
	ch := math.Min(logicalH*0.36, 150)
	cw := ch * 0.7
	ts := math.Min(58, logicalH*0.12)
	topY := (logicalH - (ch + ts + 50)) / 2
	ops = append(ops, func(dst *ebiten.Image) {
		for i, c := range menuCards {
			off := float64(i) - 1.5
			x := bx + off*cw*0.62
			y := topY + ch/2 + off*off*ch*0.04 + math.Sin(float64(frame)/40+float64(i))*3
			rot := off*0.13 + math.Sin(float64(frame)/60+float64(i)*1.7)*0.02
			drawCardFX(dst, c, x, y, cw, ch, rot, 1, cardLook{})
		}
		drawTitle(dst, bx, topY+ch+ts*0.6+12, ts)
		drawText(dst, "A CARD GAME FOR 1-4 PLAYERS", bx, topY+ch+ts*1.2+22, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
	bh := 44.0
	if isTouch {
		bh = 40
	}
	n := 5
	if !isTouch {
		n++
	}
	y := (logicalH - (bh+14)*float64(n) + 14) / 2
	if Button(Rect{cx, y, colW, bh}, "Play with bots", styleGold, true, icoPlay) {
		startSolo()
	}
	y += bh + 14
	if Button(Rect{cx, y, colW, bh}, "Host a table", styleViolet, true, icoHome) {
		startHost(false)
	}
	y += bh + 14
	if Button(Rect{cx, y, colW, bh}, "Join a table", styleViolet, true, icoWorld) {
		netStatus = ""
		openScreen(scJoin)
	}
	y += bh + 14
	if Button(Rect{cx, y, colW, bh}, "How to play", styleSlate, true, icoBook) {
		helpFrom, helpPage = scMain, 0
		openScreen(scHelp)
	}
	y += bh + 14
	if Button(Rect{cx, y, colW, bh}, "Settings", styleSlate, true, icoGear) {
		settingsFrom = scMain
		openScreen(scSettings)
	}
	if !isTouch {
		y += bh + 14
		if Button(Rect{cx, y, colW, bh}, "Quit", styleSlate, true, icoX) {
			savePrefs()
			os.Exit(0)
		}
	}
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, "build "+Version, logicalW-10, logicalH-12, textOpt{size: 10, col: colMuted, align: text.AlignEnd})
	})
}

// ---- Joining a table ----

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var (
	joinField = 1 // 0 code, 1 address, 2 name
	codeEntry string
	addrEntry string
)

func joinMenu() {
	startFinding()
	r := sheet(720, 470)
	if back, cl := head(r, "JOIN A TABLE", true); back || cl {
		if netRole == netGuest {
			netShutdown()
		}
		stopFinding()
		openScreen(scMain)
		return
	}
	x, w := r.X+18, r.W-36
	y := r.Y + 62
	busy := netRole == netGuest
	nameField(x, y, w)
	y += 44
	Label("TABLES ON THIS WI-FI", x, y)
	y += 14
	list := foundTables()
	rh := 38.0
	if len(list) == 0 {
		box := Rect{x, y, w, rh}
		Inset(box, colSlateLo)
		Text("Looking for tables"+strings.Repeat(".", (frame/20)%4), box.X+12, box.Y+rh/2, 14, colMuted, text.AlignStart)
		Text("A friend picks Host a table on the same Wi-Fi or hotspot", box.X+box.W-12, box.Y+rh/2, 11, colMuted, text.AlignEnd)
		y += rh + 6
	}
	for i, f := range list {
		if i >= 2 {
			break
		}
		row := Rect{x, y, w, rh}
		Inset(row, colSlateLo)
		Text(f.ad.Host+"'s table", row.X+12, row.Y+13, 14, colInk, text.AlignStart)
		state := "waiting for players"
		if !f.ad.Open {
			state = "match in progress"
		}
		Text(fmt.Sprintf("%s  -  %d of %d players  -  %s", modeNames[clampInt(f.ad.Mode, 0, numModes-1)], f.ad.Players, f.ad.Max, state),
			row.X+12, row.Y+28, 11, colMuted, text.AlignStart)
		if Button(Rect{row.X + row.W - 100, row.Y + 4, 92, rh - 8}, "Join", styleGold, !busy && f.ad.Open && f.ad.Players < f.ad.Max, icoPlay) {
			joinLAN(f.addr)
		}
		y += rh + 6
	}
	y += 6
	// by address, or by online code
	modes := []string{"By address"}
	if relayURL() != "" {
		modes = append(modes, "By online code")
	}
	if joinField == 2 {
		joinField = 1
	}
	if c := seg(x, y, modes, 1-joinField); c >= 0 {
		joinField = 1 - c
	}
	y += 42
	f := Rect{x, y, 250, 34}
	Inset(f, colPanelLo)
	entry, hint := addrEntry, "e.g. 192.168.1.20"
	if joinField == 0 {
		entry, hint = codeEntry, "5 letters, e.g. KQ7XM"
	}
	if entry == "" {
		Text(hint, f.X+10, f.Y+17, 14, colMuted, text.AlignStart)
	}
	caret := ""
	if !isTouch && (frame/30)%2 == 0 && typingTo != 2 {
		caret = "_"
	}
	Text(entry+caret, f.X+10, f.Y+17, 15, colInk, text.AlignStart)
	if ptr.released && f.Has(ptr.x, ptr.y) {
		typingTo = joinField
	}
	ok := (joinField == 1 && addrEntry != "") || (joinField == 0 && len(codeEntry) == 5)
	if Button(Rect{f.X + f.W + 10, y, 100, 34}, "Join", styleGold, ok && !busy, icoPlay) {
		doJoin()
	}
	if busy && Button(Rect{f.X + f.W + 120, y, 100, 34}, "Cancel", styleSlate, true, nil) {
		netShutdown()
		netStatus = ""
	}
	y += 42
	if isTouch {
		y = keypad(x, y, w)
	}
	if netStatus != "" {
		Text(netStatus, x, math.Min(y+10, r.Y+r.H-18), 14, colT, text.AlignStart)
	}
	if !isTouch {
		typeKeys()
	}
}

var typingTo = 1 // what the keyboard types into: 0 code, 1 address, 2 name

func doJoin() {
	if joinField == 1 && addrEntry != "" {
		joinLAN(addrEntry)
	} else if joinField == 0 && len(codeEntry) == 5 {
		joinCode(codeEntry)
	}
}

// The player's name, with a button for a new random one
func nameField(x, y, w float64) {
	Label("YOUR NAME", x, y+6)
	nf := Rect{x + 110, y - 8, math.Min(260, w-110-110), 32}
	Inset(nf, colPanelLo)
	caret := ""
	if typingTo == 2 && !isTouch && (frame/30)%2 == 0 {
		caret = "_"
	}
	Text(playerName()+caret, nf.X+10, nf.Y+16, 15, colInk, text.AlignStart)
	if ptr.released && nf.Has(ptr.x, ptr.y) {
		typingTo = 2
	}
	if Button(Rect{nf.X + nf.W + 10, nf.Y, 94, 32}, "Shuffle", styleSlate, true, nil) {
		prefs.Name = randomPlayerName()
		savePrefs()
	}
}

// Phones: an on-screen pad for the address or the code
func keypad(x, y, w float64) float64 {
	keys := "1234567890.<"
	if joinField == 0 {
		keys = codeAlphabet + "<"
	}
	per := 12
	if joinField == 0 {
		per = 11
	}
	kw := (w - float64(per-1)*5) / float64(per)
	for i, ch := range keys {
		kx := x + float64(i%per)*(kw+5)
		ky := y + float64(i/per)*34
		if Button(Rect{kx, ky, kw, 29}, string(ch), styleSlate, true, nil) {
			typeChar(ch)
		}
	}
	return y + float64((len(keys)+per-1)/per)*34
}

func typeChar(ch rune) {
	switch {
	case ch == '<':
		if joinField == 0 && len(codeEntry) > 0 {
			codeEntry = codeEntry[:len(codeEntry)-1]
		} else if joinField == 1 && len(addrEntry) > 0 {
			addrEntry = addrEntry[:len(addrEntry)-1]
		}
	case joinField == 0:
		c := strings.ToUpper(string(ch))
		if strings.Contains(codeAlphabet, c) && len(codeEntry) < 5 {
			codeEntry += c
		}
	default:
		if strings.ContainsRune("0123456789.:abcdefABCDEF", ch) && len(addrEntry) < 40 {
			addrEntry += string(ch)
		}
	}
}

// Keyboard typing on computers
func typeKeys() {
	for _, ch := range ebiten.AppendInputChars(nil) {
		if ch < 32 {
			continue
		}
		if typingTo == 2 {
			if len(prefs.Name) < 16 {
				prefs.Name += string(ch)
				savePrefs()
			}
			continue
		}
		joinField = typingTo
		typeChar(ch)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if typingTo == 2 {
			if len(prefs.Name) > 0 {
				prefs.Name = prefs.Name[:len(prefs.Name)-1]
				savePrefs()
			}
		} else {
			typeChar('<')
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && netRole != netGuest && typingTo != 2 {
		doJoin()
	}
}

// ---- Settings ----

func settingsMenu() {
	r := sheet(640, 400)
	if back, cl := head(r, "SETTINGS", true); back || cl {
		openScreen(settingsFrom)
		return
	}
	x, y := r.X+18, r.Y+70
	lvls := []string{"Off", "1", "2", "3", "4", "5"}
	Label("MUSIC VOLUME", x, y)
	if c := seg(x, y+14, lvls, prefs.MusicVol); c >= 0 {
		prefs.MusicVol = c
		setVolumes()
		savePrefs()
	}
	y += 64
	Label("SOUND EFFECTS VOLUME", x, y)
	if c := seg(x, y+14, lvls, prefs.SfxVol); c >= 0 {
		prefs.SfxVol = c
		setVolumes()
		savePrefs()
		sfx("coin")
	}
	y += 64
	Label("INTERFACE SIZE", x, y)
	if c := seg(x, y+14, []string{"Small", "Normal", "Large", "Larger"}, prefs.UISize); c >= 0 {
		prefs.UISize = c
		savePrefs()
	}
	y += 64
	Label("SCORING ANIMATION", x, y)
	if c := seg(x, y+14, []string{"Normal", "Fast"}, b2i(prefs.Fast)); c >= 0 {
		prefs.Fast = c == 1
		savePrefs()
	}
	if !isTouch {
		Label("FULL SCREEN", x+260, y)
		if c := seg(x+260, y+14, []string{"On", "Off"}, b2i(!prefs.Full)); c >= 0 {
			prefs.Full = c == 0
			ebiten.SetFullscreen(prefs.Full)
			savePrefs()
		}
	}
	y += 64
	if settingsFrom == scMain {
		nameField(x, y+8, r.W-36)
		if !isTouch {
			typeKeys()
		}
	}
}

// ---- How to play ----

var helpPages = []struct{ title, body string }{
	{"THE IDEA", "Desalote is a poker-hand scoring game for 1 to 4 players. Each round you play hands of cards to score points: a hand's points (its base plus the value of its scoring cards) times its Mult.\n" +
		"Beat the round's target, or beat your rivals in a showdown, to keep your lives. Between rounds, bid against each other for charms, buy tomes to level up your hands, trade charms, and buy hexes to curse your rivals."},
	{"YOUR 13 CARDS", "You don't get a full deck. At the start of a match you're dealt 13 different cards at random, and they're yours, unchanged, for the whole match: nothing adds or removes cards.\n" +
		"Because every deck is different, flushes and straights adapt to yours. A flush needs as many cards of one suit as your deck's average per suit (rounded down). A straight needs as many ranks in a row as your deck's average run (rounded up). Neither is ever more than 5, since you play at most 5 cards.\n" +
		"Each round you hold 7 cards, play up to 5 at a time, and have 4 plays and 3 discards. When your pile runs out, your played cards are shuffled back in."},
	{"YOUR HANDS", "Everyone can always score High Card and Pair. You're also given a family: Same (matching ranks), Counting (ranks in a row) or Suits. From five hands offered to you, pick three to play all match, at least one from your own family. Any other hand only counts as a Pair or High Card.\n" +
		"Synergy: when two of your picks share a family, all three score x1.2. When all three do, x1.5.\n" +
		"A Same family deck always has at least 3 cards of one rank."},
	{"YOUR SPECIAL HAND", "You also get a special hand: Five of a Kind, Straight Flush or Flush House. Once a round, press Special: the game picks 5 cards from your hand (you don't choose) and briefly turns them into that hand. It scores big and doesn't use a play.\n" +
		"Afterwards the cards return to normal, each with +3 points for good.\n" +
		"Five of a Kind always uses your deck's most common rank (ties are picked at random), so it's a little predictable and scores a little less than the other two."},
	{"MODES", "Duel: 1 v 1. Small blinds are against the house; Big and Boss blinds are showdowns, where the lower score loses a life.\n" +
		"Last Standing: up to 4. Miss a blind's target and lose a life; each Boss blind is a showdown where the lowest score loses a life.\n" +
		"Co-op Heist: everyone's scores add up against big targets, and the team shares its lives.\n" +
		"Race: everyone plays at their own pace with their own shop. First to beat the last Boss blind wins.\n" +
		"The host picks the length: Short (3 antes), Standard (5) or Long (8), and the lives."},
	{"THE SHOP", "Auction: charms (and a hex) go to the highest bidder. Tap Bid to raise by $1; a bid near the end adds a few seconds. You can't promise more money than you have.\n" +
		"Tomes level up one of your picked hands, at a set price.\n" +
		"Market: sell a charm back for half its price, or list it for other players to buy at the price you set.\n" +
		"Hexes: cast one on a rival to curse their next round (Fog, Rust, Squeeze and more). A Mirror Ward bounces one back."},
}

func helpMenu() {
	if helpFrom == scMain {
	}
	r := sheet(720, 440)
	if back, cl := head(r, "HOW TO PLAY", true); back || cl {
		openScreen(helpFrom)
		return
	}
	x := r.X + 18
	y := r.Y + 62
	if c := seg(x, y, []string{"1", "2", "3", "4", "5", "6"}, helpPage); c >= 0 {
		helpPage = c
	}
	pg := helpPages[helpPage]
	Label(pg.title, x, y+50)
	size := 14.0
	if wrapHeight(pg.body, r.W-36, size) > r.H-170 {
		size = 12.5
	}
	wrapText(pg.body, x, y+74, r.W-36, size, colInk, text.AlignStart)
	if helpPage < len(helpPages)-1 && Button(Rect{r.X + r.W - 120, r.Y + r.H - 48, 104, 34}, "Next", styleViolet, true, nil) {
		helpPage++
	}
}

// ---- Toast ----

func drawToast() {
	if frame >= toastUntil {
		return
	}
	w := textWidth(toastText, 11, true) + 28
	r := Rect{(logicalW - w) / 2, logicalH * 0.42, w, 30}
	Panel(r)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, toastText, r.X+r.W/2, r.Y+r.H/2, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
}
