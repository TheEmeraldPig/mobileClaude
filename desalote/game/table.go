package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The table: everything during a match (lobby, draft, rounds, results, shop, the end),
// following view.Phase. Overlays (the menu, your deck, picking a hex's target, pricing a
// charm) sit on top.

type overlay int

const (
	ovNone overlay = iota
	ovMenu
	ovDeck
	ovHex   // pick a target for hex hexSlot
	ovPrice // set a price for listing priceCharm
	ovCharms
	ovHexes
	ovPlayer // details of pillPlayer
	ovTomes
	ovTrade
)

var (
	ov         overlay
	hexSlot    int
	priceCharm string
	priceAmt   int
	shopTab    int
	dealAt     int
)

// The scoring animation for the last hand played
type scoreAnim struct {
	res    *ScoreResult
	start  int
	vids   []int // the cards' ids for their motion (visCards)
	flung  bool
	cw, ch float64
}

var (
	anim      *scoreAnim
	dispScore float64
)

func trimFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprint(int(f))
	}
	return fmt.Sprintf("%.1f", f)
}

func animLen(res *ScoreResult) int {
	// the cards light up one by one, then the total shows for a moment
	n := 10 + animStep()*len(res.Scoring) + 40/prefs.Speed
	if res.Special && !prefs.SkipAnim {
		n += 30 / prefs.Speed
	}
	if prefs.SkipAnim {
		n = 10 + len(res.Scoring) + 12
	}
	return n
}

func animating() bool { return anim != nil && frame-anim.start < animLen(anim.res) }

func tableScreen() {
	if view == nil {
		openScreen(scMain)
		return
	}
	p := me()
	if p == nil {
		// a guest waiting to be seated
		r := sheet(420, 160)
		Title("JOINING", r.X+16, r.Y+28)
		Text(netStatus, r.X+16, r.Y+70, 14, colInk, text.AlignStart)
		if Button(Rect{r.X + r.W - 120, r.Y + r.H - 48, 104, 34}, "Leave", styleSlate, true, nil) {
			leaveTable("")
		}
		return
	}
	switch view.Phase {
	case "lobby":
		lobbyScreen()
	case "draft":
		draftScreen(p)
	case "play":
		if view.Mode == ModeRace && p.Stage == "shop" {
			shopScreen(p)
		} else {
			playScreen(p)
		}
	case "results":
		playScreen(p)
		resultsPanel()
	case "shop":
		shopScreen(p)
	case "over":
		overScreen()
	}
	if view == nil || me() == nil { // left the table (or the host closed it) this frame
		return
	}
	if view.Phase != "lobby" && view.Phase != "over" {
		menuButton()
	}
	switch ov {
	case ovMenu:
		menuOverlay()
	case ovDeck:
		deckOverlay(p)
	case ovHex:
		hexOverlay(p)
	case ovPrice:
		priceOverlay(p)
	case ovCharms:
		charmsOverlay(p)
	case ovHexes:
		hexesOverlay(p)
	case ovPlayer:
		playerOverlay()
	case ovTomes:
		tomesOverlay(p)
	case ovTrade:
		tradeOverlay(p)
	}
}

func menuButton() {
	if Button(Rect{logicalW - 44, 8, 36, 32}, "", styleSlate, true, icoPause) {
		if ov == ovMenu {
			ov = ovNone
		} else {
			ov = ovMenu
		}
	}
}

// Escape closes an overlay, or opens the menu
func tableEscape() {
	if ov != ovNone {
		ov = ovNone
	} else {
		ov = ovMenu
	}
}

// ---- The players ----

func playerStatus(p *PlayerState) string {
	switch {
	case p.Gone:
		return "left"
	case p.Out:
		return "out"
	}
	if view.Mode == ModeRace {
		switch p.Stage {
		case "shop":
			return fmt.Sprintf("shop  A%d", p.Ante+1)
		case "won":
			return "won!"
		}
		return fmt.Sprintf("%s  A%d %s", commas(p.Score), p.Ante+1, []string{"S", "B", "Boss"}[p.Blind])
	}
	switch view.Phase {
	case "draft":
		if p.Ready {
			return "ready"
		}
		return "looking"
	case "shop":
		if p.Ready {
			return "done"
		}
		return "shopping"
	case "play", "results":
		s := commas(p.Score)
		if p.Done {
			s += " done"
		}
		return s
	}
	return ""
}

// ---- Lobby ----

// The lobby, laid out openly on the table: the seats across the middle, the match
// settings under them, how friends join and Start along the bottom
func lobbyScreen() {
	isHost := host != nil
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, "THE TABLE", 18, 26, textOpt{size: 22, display: true, col: colL, shadow: true})
	})
	if Button(Rect{logicalW - 104, 10, 94, 34}, "Leave", styleRed, true, nil) {
		leaveTable("")
		return
	}
	// seats
	n := view.maxPlayers()
	sw := math.Min(190, (logicalW-40-12*float64(n-1))/float64(n))
	sh := math.Min(108, logicalH*0.24)
	sx := (logicalW - (sw*float64(n) + 12*float64(n-1))) / 2
	sy := 56.0
	for i := 0; i < n; i++ {
		r := Rect{sx + float64(i)*(sw+12), sy, sw, sh}
		if i < len(view.Players) {
			p := view.Players[i]
			tag := "player"
			switch {
			case p.ID == myID:
				tag = "you"
			case p.Bot:
				tag = "bot"
			}
			if p.ID == 0 {
				tag += " - host"
			}
			name, me := p.Name, p.ID == myID
			ops = append(ops, func(dst *ebiten.Image) {
				roundRect(dst, r.X, r.Y+4, r.W, r.H, 16, color.NRGBA{0, 0, 0, 80})
				body := color.Color(color.NRGBA{60, 32, 32, 230})
				if me {
					body = color.NRGBA{96, 52, 38, 235}
				}
				roundRect(dst, r.X, r.Y, r.W, r.H, 16, body)
				drawCardFX(dst, Card{}, r.X+r.W/2, r.Y+r.H*0.34, r.H*0.3, r.H*0.42, 0.08*float64(i%2*2-1), 1, cardLook{faceDown: true})
				drawText(dst, name, r.X+r.W/2, r.Y+r.H*0.66, textOpt{size: 14, col: colInk, align: text.AlignCenter, shadow: true})
				drawText(dst, tag, r.X+r.W/2, r.Y+r.H*0.84, textOpt{size: 11, col: colMuted, align: text.AlignCenter})
			})
			if isHost && p.Bot && Button(Rect{r.X + r.W - 34, r.Y + 6, 28, 28}, "", styleSlate, true, icoX) {
				host.removePlayer(p.ID)
			}
		} else {
			ops = append(ops, func(dst *ebiten.Image) {
				roundRect(dst, r.X, r.Y, r.W, r.H, 16, color.NRGBA{0, 0, 0, 60})
				drawText(dst, "Empty seat", r.X+r.W/2, r.Y+r.H*0.36, textOpt{size: 13, col: colMuted, align: text.AlignCenter})
			})
			if isHost && Button(Rect{r.X + r.W/2 - 50, r.Y + r.H*0.56, 100, 30}, "+ Bot", styleSlate, true, nil) {
				host.addBot()
			}
		}
	}
	// settings, centred under the seats
	y := sy + sh + 22
	row := func(label string, labels []string, active int) int {
		w := 0.0
		for _, l := range labels {
			w += math.Max(38, textWidth(l, 14, false)+22) + 10
		}
		lw := 70.0
		x := (logicalW - (lw + w)) / 2
		Text(label, x, y+15, 12, colT, text.AlignStart)
		c := segWrap(x+lw, y, w, labels, active, isHost)
		y += 40
		return c
	}
	if c := row("MODE", []string{"Duel", "Last Standing", "Co-op", "Race"}, view.Mode); c >= 0 {
		setMode(c)
	}
	y = wrapText(modeBlurbs[view.Mode], logicalW/2, y+4, math.Min(760, logicalW-40), 12, colMuted, text.AlignCenter) + 4
	var ll []string
	for i, n := range lengthNames {
		ll = append(ll, fmt.Sprintf("%s (%d)", n, lengthAntes[i]))
	}
	if c := row("LENGTH", ll, view.Length); c >= 0 {
		host.Length, prefs.Length = c, c
		host.changed()
		savePrefs()
	}
	if c := row("LIVES", []string{"1", "2", "3", "4", "5"}, view.MaxLives-1); c >= 0 {
		host.MaxLives, prefs.Lives = c+1, c+1
		host.changed()
		savePrefs()
	}
	// bottom: how friends join, and Start
	by := logicalH - 50
	info := ""
	switch {
	case !isHost:
		info = "Waiting for the host to start..."
	case solo:
		info = "Playing with bots. Add or remove them in the empty seats."
	case netCode != "" && lanServer == nil:
		info = "Friends join with the online code " + netCode
	case lanServer != nil && myAddress() != "":
		info = "Friends on this Wi-Fi: Join a table, or type " + myAddress()
	case lanServer != nil:
		info = "Connect to Wi-Fi or a hotspot so friends can find you"
	}
	Text(info, 18, by+19, 13, colInk, text.AlignStart)
	if isHost && !solo && netCode == "" && relayURL() != "" && Button(Rect{18, by - 40, 220, 32}, "Use an online code instead", styleSlate, true, nil) {
		startHostingOnline()
	}
	if isHost {
		ok := host.canStart()
		label := "Start the match"
		if !ok {
			label = "Need 2+ players"
		}
		if Button(Rect{logicalW - 210, by, 196, 40}, label, styleGold, ok, nil) {
			host.start()
			sfx("deal")
		}
	}
}

func setMode(c int) {
	host.Mode = c
	prefs.Mode = c
	for len(host.Players) > host.maxPlayers() {
		// a duel has two seats: remove bots first
		removed := false
		for i := len(host.Players) - 1; i > 0; i-- {
			if host.Players[i].Bot {
				host.removePlayer(host.Players[i].ID)
				removed = true
				break
			}
		}
		if !removed {
			host.Mode = ModeLastStanding
			toast("Too many players for a duel")
			break
		}
	}
	host.changed()
	savePrefs()
}

// Segmented buttons that wrap onto more rows; read-only for guests
func segWrap(x, y, w float64, labels []string, active int, enabled bool) int {
	clicked := -1
	cx, cy := x, y
	for i, l := range labels {
		bw := math.Max(38, textWidth(l, 14, false)+22)
		if cx+bw > x+w && cx > x {
			cx, cy = x, cy+38
		}
		st := styleSlate
		if i == active {
			st = styleViolet
		}
		if Button(Rect{cx, cy, bw, 30}, l, st, enabled || i == active, nil) && enabled {
			clicked = i
		}
		cx += bw + 8
	}
	return clicked
}

// ---- The deal: your cards and hands for the match ----

func draftScreen(p *PlayerState) {
	top := playerPills(8) + 12
	if myRun == nil {
		return
	}
	titleY := top + 12
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, "YOUR DEAL", 18, titleY, textOpt{size: 20, display: true, col: colL, shadow: true})
	})
	// your 13 cards, fanned across the table
	deck := myRun.Deck
	ch := math.Min(110, logicalH*0.24)
	cw := ch * 0.7
	// everything below sits in the middle of the free space
	by := logicalH - 50
	content := 34 + ch + 46 + 34 + math.Min(86, logicalH*0.22)
	top = math.Max(top, top+(by-10-top-content)/2)
	step := math.Min(cw*0.78, (logicalW-60-cw)/12)
	x0 := (logicalW - (step*12 + cw)) / 2
	cy := top + 34 + ch/2
	for i, c := range deck {
		o := float64(i) - 6
		cc := c
		x := x0 + cw/2 + float64(i)*step
		y := cy + o*o*0.9 + math.Sin(float64(frame)/50+float64(i)*0.8)*1.5
		rot := o * 0.025
		ops = append(ops, func(dst *ebiten.Image) { drawCardFX(dst, cc, x, y, cw, ch, rot, 1, cardLook{}) })
	}
	y := cy + ch/2 + 46
	// facts about the deal, as a line of pills
	fc := familyColors[p.Family]
	facts := []struct {
		k, v string
		c    color.Color
	}{
		{"Family", familyNames[p.Family], rgb(fc[1])},
		{"Special", hands[p.Special].name, colT},
		{"Flush", fmt.Sprintf("%d of a suit", myRun.Rules.flush), colInk},
		{"Straight", fmt.Sprintf("%d in a row", myRun.Rules.straight), colInk},
	}
	pw := math.Min(200, (logicalW-40-30)/4)
	px := (logicalW - (pw*4 + 30)) / 2
	for i, f := range facts {
		r := Rect{px + float64(i)*(pw+10), y - 20, pw, 40}
		ff := f
		ops = append(ops, func(dst *ebiten.Image) {
			roundRect(dst, r.X, r.Y, r.W, r.H, 20, color.NRGBA{40, 22, 24, 215})
			drawText(dst, ff.k, r.X+r.W/2, r.Y+12, textOpt{size: 10, display: true, col: colMuted, align: text.AlignCenter})
			drawText(dst, ff.v, r.X+r.W/2, r.Y+28, textOpt{size: 13, col: ff.c, align: text.AlignCenter})
		})
	}
	y += 34
	// the three hands you were dealt
	hw := math.Min(230, (logicalW-40-20)/3)
	hh := math.Min(86, logicalH*0.22)
	hx := (logicalW - (hw*3 + 20)) / 2
	for i, h := range p.Picks {
		r := Rect{hx + float64(i)*(hw+10), y, hw, hh}
		hh2 := h
		f := hands[h].fam
		lv := p.Levels[h]
		b, m := handLevel(h, lv+1)
		how := handHow(h, myRun.Rules)
		ops = append(ops, func(dst *ebiten.Image) {
			col := familyColors[f]
			roundRect(dst, r.X, r.Y+4, r.W, r.H, 14, color.NRGBA{0, 0, 0, 80})
			roundRect(dst, r.X, r.Y, r.W, r.H, 14, color.NRGBA{60, 32, 32, 235})
			roundRect(dst, r.X+8, r.Y+8, 6, r.H-16, 3, rgb(col[0]))
			drawText(dst, hands[hh2].name, r.X+22, r.Y+r.H*0.25, textOpt{size: 14, col: colInk, shadow: true})
			drawText(dst, how, r.X+22, r.Y+r.H*0.52, textOpt{size: 10, col: colL})
			lvS := fmt.Sprintf("Lv %d   %d x %d", lv+1, b, m)
			drawText(dst, lvS, r.X+22, r.Y+r.H*0.78, textOpt{size: 12, display: true, col: colT})
		})
	}
	// the synergy roll
	_, sm := synergyOf(p.Picks)
	syn := fmt.Sprintf("Synergy roll: all 3 %s hands, x%.1f", familyNames[p.Family], sm)
	switch p.Synergy {
	case 2:
		syn = fmt.Sprintf("Synergy roll: 2 %s hands, x%.1f. The other hand starts a level up.", familyNames[p.Family], sm)
	case 1:
		syn = "Synergy roll: no synergy. Your hands from other families start a level up."
	}
	Text(syn, 18, by+20, 13, colT, text.AlignStart)
	if p.Ready {
		Text("Waiting for the others...", logicalW-230, by+20, 13, colMuted, text.AlignEnd)
		if Button(Rect{logicalW - 210, by, 196, 40}, "Not yet", styleSlate, true, nil) {
			act(action{T: "undealt"})
		}
		return
	}
	if Button(Rect{logicalW - 210, by, 196, 40}, "Ready", styleGold, true, nil) {
		act(action{T: "dealt"})
		sfx("score")
	}
}

func handHow(h HandType, rules handRules) string {
	s := hands[h].how
	s = strings.ReplaceAll(s, "%S", fmt.Sprint(rules.straight))
	s = strings.ReplaceAll(s, "%F", fmt.Sprint(rules.flush))
	return s
}

// ---- Playing a round ----

func playScreen(p *PlayerState) {
	top := playerPills(8) + 10
	r := myRun
	if r == nil {
		return
	}
	card := scoreCard(p, r, top)
	charmRow(p, top)

	// the hand, the buttons, and the last hand scored, across the whole table
	mx := 8.0
	mw := logicalW - 16
	bh := 40.0
	by := logicalH - bh - 12
	// the hand fits under the score card (raised cards may overlap it a little)
	ch := math.Min(150, math.Min((by-top)*0.45, by-16-(card.Y+card.H)-6))
	cw := ch * 0.7
	handCW, handCH = cw, ch
	n := len(r.Hand)
	step := cw * 0.92
	if n > 1 && step*float64(n-1)+cw > mw-cw*0.9 {
		step = (mw - cw*0.9 - cw) / float64(n-1)
	}
	hy := by - 16 - ch/2 // the hand's middle line
	deckX, deckY := mx+mw-cw*0.45, hy+ch*0.05
	busy := animating()
	if !r.Done {
		// the deck pile the cards are dealt from
		pile := len(r.Pile)
		for k := min(pile, 4) - 1; k >= 0; k-- {
			kx, ky := deckX+float64(k)*1.5, deckY-float64(k)*1.5
			ops = append(ops, func(dst *ebiten.Image) {
				drawCardFX(dst, Card{}, kx, ky, cw*0.8, ch*0.8, 0.06, 1, cardLook{faceDown: true})
			})
		}
		Text(fmt.Sprint(pile), deckX, deckY+ch*0.47, 12, colMuted, text.AlignCenter)
		if dr := (Rect{deckX - cw*0.45, deckY - ch*0.45, cw * 0.9, ch * 0.9}); ptr.released && !ptr.consumed && dr.Has(ptr.x, ptr.y) {
			ptr.consumed = true
			ov = ovDeck
			sfx("open")
		}
		c0 := float64(n-1) / 2
		hovered := -1
		for i := 0; i < n; i++ {
			o := float64(i) - c0
			hit := Rect{mx + (mw-cw*0.9)/2 + o*step - step/2, hy - ch/2 - ch*0.2, step, ch * 1.2}
			if i == 0 || i == n-1 {
				hit.W = cw * 0.8
				if i == 0 {
					hit.X = hit.X + step - hit.W
				}
			}
			if !isTouch && hit.Has(hoverX, hoverY) && !ptr.down {
				hovered = i
			}
			if busy || ov != ovNone {
				continue
			}
			// press a card to pick it (or put it back); keep holding and sweep across
			// the others to do the same to them (Handy's drag select)
			switch {
			case ptr.pressed && hit.Has(ptr.x, ptr.y):
				cardDrag, cardDragVal = true, !r.Sel[i]
				setPick(r, i, cardDragVal)
			case cardDrag && ptr.down && prefs.DragSelect && hit.Has(ptr.x, ptr.y) && r.Sel[i] != cardDragVal:
				setPick(r, i, cardDragVal)
			case !isTouch && prefs.HoverSelect && !ptr.down && ebiten.IsKeyPressed(ebiten.KeyShift) && hit.Has(hoverX, hoverY) && !r.Sel[i]:
				setPick(r, i, true) // Shift + point: pick without clicking
			}
		}
		if cardDrag && ptr.released {
			cardDrag = false
			ptr.consumed = true
		}
		for i := 0; i < n; i++ {
			d := r.Hand[i]
			o := float64(i) - c0
			tx := mx + (mw-cw*0.9)/2 + o*step
			ty := hy + o*o*1.6
			trot := o*0.045 + math.Sin(float64(frame)/55+float64(i)*1.3)*0.012
			ty += math.Sin(float64(frame)/45+float64(i)*0.9) * 1.6
			sc := 1.0
			if r.Sel[i] {
				ty -= ch * 0.22
			}
			if i == hovered {
				ty -= 8
				sc = 1.06
			}
			v := placeCard(d, tx, ty, trot, sc, deckX, deckY)
			c, fd, sel := r.Deck[d], r.FaceDown[d], r.Sel[i]
			vx, vy, vr, vs := v.x, v.y, v.rot, v.scale
			ops = append(ops, func(dst *ebiten.Image) {
				drawCardFX(dst, c, vx, vy, cw, ch, vr, vs, cardLook{faceDown: fd, selected: sel})
			})
		}
		bw := math.Min(150, (mw-cw-150)/3)
		bx := mx + (mw-cw*0.9-(bw*3+20))/2
		sortLabel := "Sort: rank"
		if r.SortSuit {
			sortLabel = "Sort: suit"
		}
		if Button(Rect{mx + 4, by + 4, math.Min(110, bx-mx-14), bh - 8}, sortLabel, styleSlate, true, nil) {
			sortHand(!r.SortSuit)
		}
		if Button(Rect{bx, by, bw, bh}, fmt.Sprintf("Play hand  %d", r.Plays), styleViolet, r.CanPlay() && !busy, nil) {
			doPlay()
		}
		if Button(Rect{bx + bw + 10, by, bw, bh}, fmt.Sprintf("Discard  %d", r.Discards), styleRed, r.CanDiscard() && !busy, nil) {
			doDiscard()
		}
		spLabel := "Special"
		if r.SpecialLeft > 1 {
			spLabel = fmt.Sprintf("Special x%d", r.SpecialLeft)
		}
		if Button(Rect{bx + 2*(bw+10), by, bw, bh}, spLabel, styleGold, r.CanSpecial() && !busy, nil) {
			doSpecial()
		}
		if r.Plays == 0 && !r.Done && !busy {
			Text("Out of plays: use your special, or", mx+mw/2, top+40, 12, colMuted, text.AlignCenter)
			if Button(Rect{mx + mw/2 - 60, top + 54, 120, 32}, "Finish", styleSlate, true, nil) {
				r.Finish()
				reportRun()
			}
		}
		if anim == nil && len(r.Selected()) == 0 && frame-dealAt > 40 {
			hint := "Pick up to 5 cards, then Play hand. Discard swaps them for new ones."
			if !isTouch {
				hint = "Click up to 5 cards (or keys 1-8), then Play (Enter). D discards, S uses your special."
			}
			Text(hint, (card.X+card.W+logicalW)/2, top+(hy-ch/2-top)*0.5, 12, colMuted, text.AlignCenter)
		}
	} else if !animating() {
		msg := "Round done. Waiting for the others..."
		switch {
		case p.Out:
			msg = "You're out. Watching the rest of the match."
		case view.Mode == ModeRace:
			msg = "Round done..."
		case view.Phase == "results":
			msg = ""
		}
		Text(msg, mx+mw/2, (top+by)/2+40, 15, colMuted, text.AlignCenter)
	}
	// the last hand, scoring
	if anim != nil {
		ax := card.X + card.W + 12
		drawAnim(ax, top+40, logicalW-ax-12, hy-ch/2-top-50)
		if ptr.released && !ptr.consumed && animating() {
			anim.start = frame - animLen(anim.res) // tap to skip
		}
	}
}

var (
	handCW, handCH = 70.0, 100.0
	cardDrag       bool // a press on a card is being swept across the others
	cardDragVal    bool // picking (true) or putting back
)

// Pick or put back card i of the hand, with a sound and a little hop
func setPick(r *Run, i int, on bool) {
	if r.Sel[i] == on {
		return
	}
	r.Toggle(i)
	if r.Sel[i] {
		sfx("select")
		kickCard(r.Hand[i], -4, 0, 0.08)
	} else if !on {
		sfx("unselect")
	}
}

// Sort the hand by suit or rank (and remember it)
func sortHand(suit bool) {
	if myRun == nil {
		return
	}
	myRun.SortBy(suit)
	prefs.SortSuit = suit
	savePrefs()
	sfx("deal")
}

func deselectAll() {
	if myRun == nil {
		return
	}
	for i := range myRun.Sel {
		myRun.Sel[i] = false
	}
}

// The hand's actions, shared by the buttons and the keys
func doPlay() {
	r := myRun
	if r == nil || !r.CanPlay() || animating() {
		return
	}
	var ids []int
	for _, i := range r.Selected() {
		ids = append(ids, r.Hand[i])
	}
	if res := r.Play(); res != nil {
		startAnim(res, ids)
		sfx("play")
		reportRun()
		dealAt = frame
	}
}

func doDiscard() {
	r := myRun
	if r == nil || !r.CanDiscard() || animating() {
		return
	}
	for _, i := range r.Selected() {
		d := r.Hand[i]
		flingCard(r.Deck[d], visCards[d], handCW, handCH, false)
		delete(visCards, d)
	}
	r.Discard()
	sfx("discard")
	dealAt = frame
}

func doSpecial() {
	r := myRun
	if r == nil || !r.CanSpecial() || animating() {
		return
	}
	before := append([]int{}, r.Hand...)
	if res := r.UseSpecial(); res != nil {
		still := map[int]bool{}
		for _, d := range r.Hand {
			still[d] = true
		}
		var ids []int
		for _, d := range before {
			if !still[d] && len(ids) < len(res.Cards) {
				ids = append(ids, d)
			}
		}
		startAnim(res, ids)
		sfx("special")
		bgHeat = 1
		reportRun()
		dealAt = frame
	}
}

// The played cards leave the hand and become the animation's cards, starting where
// they were
func startAnim(res *ScoreResult, ids []int) {
	if anim != nil {
		endAnim()
	}
	anim = &scoreAnim{res: res, start: frame}
	for i := range res.Cards {
		vid := 1000 + anim.start%1000*8 + i
		anim.vids = append(anim.vids, vid)
		if i < len(ids) {
			if v := visCards[ids[i]]; v != nil {
				cp := *v
				cp.seen = frame - 1
				visCards[vid] = &cp
			}
			delete(visCards, ids[i])
		}
	}
}

// The scored cards are swept off the table
func endAnim() {
	if anim == nil || anim.flung {
		return
	}
	anim.flung = true
	for i, c := range anim.res.Cards {
		if i < len(anim.vids) {
			flingCard(c, visCards[anim.vids[i]], anim.cw, anim.ch, false)
			delete(visCards, anim.vids[i])
		}
	}
}

// The scoring animation: the played cards fly to the middle and light up one by one
// while the points add up, then they're swept away
func drawAnim(x, y, w, h float64) {
	a := anim
	res := a.res
	t := frame - a.start
	total := animLen(res)
	speed := animStep()
	if t > total+25 {
		endAnim()
	}
	n := len(res.Cards)
	ch := math.Min(h-46, handCH)
	if ch < 40 {
		return
	}
	cw := ch * 0.7
	a.cw, a.ch = cw, ch
	step := cw + 10
	cx0 := x + (w-(step*float64(n-1)+cw))/2 + cw/2
	cy := y + 40 + ch/2
	lit := min(len(res.Scoring), max(0, (t-10)/speed+1))
	scoringSet := map[int]bool{}
	for k := 0; k < lit; k++ {
		scoringSet[res.Scoring[k]] = true
	}
	// a card lighting up this frame pops and shows its points
	if t >= 10 && (t-10)%speed == 0 && (t-10)/speed < len(res.Scoring) {
		k := (t - 10) / speed
		i := res.Scoring[k]
		sfx("tick")
		if i < len(a.vids) {
			kickCard(a.vids[i], -7, (rand.Float64()-0.5)*0.25, 0.3)
		}
		floatText(fmt.Sprintf("+%d", res.Cards[i].Value()), cx0+float64(i)*step, cy+ch/2+14, 15, rgb(0x6ac4ff))
	}
	if t == total-10 {
		sfx("score")
		shake = math.Min(7, 1.5+math.Log10(float64(max(1, res.Total)))*1.2)
		bgHeat = math.Max(bgHeat, math.Min(1, float64(res.Total)/3000))
	}
	if !a.flung {
		for i, c := range res.Cards {
			in := false
			for _, s := range res.Scoring {
				in = in || s == i
			}
			ty := cy
			if scoringSet[i] {
				ty -= 8
			}
			trot := (float64(i) - float64(n-1)/2) * 0.03
			v := placeCard(a.vids[i], cx0+float64(i)*step, ty, trot, 1, cx0, logicalH)
			cc, vx, vy, vr, vs := c, v.x, v.y, v.rot, v.scale
			look := cardLook{glowGold: scoringSet[i], dim: !in && t > 10}
			ops = append(ops, func(dst *ebiten.Image) { drawCardFX(dst, cc, vx, vy, cw, ch, vr, vs, look) })
		}
	}
	name := hands[res.Hand].name
	if res.Special {
		name = "Special: " + name
	}
	pop := 1 + 0.4*math.Exp(-float64(t)/6)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, name, x+w/2, y+14, textOpt{size: 17 * pop, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
	if len(res.Notes) > 0 && t > 10 {
		Text(strings.Join(res.Notes, "   "), x+w/2, cy+ch/2+16, 12, colL, text.AlignCenter)
	}
}

// Frames between scoring cards lighting up
func animStep() int {
	if prefs.SkipAnim {
		return 1
	}
	return max(2, 14/prefs.Speed)
}

// What the points and mult boxes show during the animation: the hand's base, then
// each scoring card added as it lights, then the final numbers with charms
func animNumbers() (base int, mult float64, done bool) {
	a := anim
	res := a.res
	t := frame - a.start
	total := animLen(res)
	if t >= total-10 {
		return res.Base, res.Mult, true
	}
	speed := animStep()
	lit := min(len(res.Scoring), max(0, (t-10)/speed+1))
	b, m := handLevel(res.Hand, myRun.Levels[res.Hand]+1)
	for k := 0; k < lit; k++ {
		b += res.Cards[res.Scoring[k]].Value()
	}
	return b, float64(m), false
}

// A tooltip by the mouse
func tip(title, body string) {
	w := math.Min(260, logicalW*0.3)
	h := wrapHeight(body, w-16, 11) + 30
	x := math.Min(hoverX+16, logicalW-w-6)
	y := math.Min(hoverY+12, logicalH-h-6)
	ops = append(ops, func(dst *ebiten.Image) { bevel(dst, Rect{x, y, w, h}, colPanelHi, colSlateHi, colPanelLo, false) })
	Text(title, x+8, y+12, 13, colT, text.AlignStart)
	wrapText(body, x+8, y+30, w-16, 11, colInk, text.AlignStart)
}

// ---- After a round ----

func resultsPanel() {
	r := sheet(520, 260)
	Title("ROUND OVER", r.X+16, r.Y+28)
	Text(fmt.Sprintf("Shop opens in %d", view.Secs), r.X+r.W-60, r.Y+28, 12, colMuted, text.AlignEnd)
	y := r.Y + 60
	for _, l := range view.Results {
		y = wrapText(l, r.X+16, y, r.W-32, 13, colInk, text.AlignStart)
	}
	if p := me(); p != nil && p.Last != "" {
		Text(p.Last, r.X+16, r.Y+r.H-26, 14, colT, text.AlignStart)
	}
}

func overScreen() {
	r := sheet(560, 380)
	Title("MATCH OVER", r.X+16, r.Y+28)
	y := wrapText(view.Winner, r.X+16, r.Y+64, r.W-32, 17, colT, text.AlignStart) + 6
	Label("STANDINGS", r.X+16, y)
	y += 16
	ps := append([]*PlayerState{}, view.Players...)
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && (ps[j].Lives > ps[j-1].Lives || (ps[j].Lives == ps[j-1].Lives && ps[j].Total > ps[j-1].Total)); j-- {
			ps[j], ps[j-1] = ps[j-1], ps[j]
		}
	}
	for _, p := range ps {
		row := Rect{r.X + 16, y, r.W - 32, 30}
		Inset(row, colSlateLo)
		c := color.Color(colInk)
		if p.ID == myID {
			c = colT
		}
		Text(p.Name, row.X+10, row.Y+15, 14, c, text.AlignStart)
		info := fmt.Sprintf("%s points   %d charms", commas(p.Total), len(p.Charms))
		if view.Mode != ModeCoop {
			info = fmt.Sprintf("%d lives   ", max(0, p.Lives)) + info
		}
		Text(info, row.X+row.W-10, row.Y+15, 12, colMuted, text.AlignEnd)
		y += 34
	}
	if host != nil && Button(Rect{r.X + r.W - 300, r.Y + r.H - 50, 140, 36}, "Play again", styleGold, true, icoPlay) {
		rematch()
	}
	if Button(Rect{r.X + r.W - 150, r.Y + r.H - 50, 134, 36}, "Leave", styleSlate, true, nil) {
		leaveTable("")
	}
}

// ---- Overlays ----

func menuOverlay() {
	r := sheet(320, 340)
	if _, cl := head(r, "MENU", false); cl {
		ov = ovNone
		return
	}
	x, w := r.X+18, r.W-36
	y := r.Y + 62
	if Button(Rect{x, y, w, 38}, "Back to the table", styleGold, true, icoPlay) {
		ov = ovNone
	}
	y += 48
	if myRun != nil && Button(Rect{x, y, w, 38}, "Your deck and hands", styleSlate, true, nil) {
		ov = ovDeck
	}
	y += 48
	if Button(Rect{x, y, w, 38}, "How to play", styleSlate, true, icoBook) {
		ov = ovNone
		helpFrom, helpPage = scTable, 0
		openScreen(scHelp)
	}
	y += 48
	if Button(Rect{x, y, w, 38}, "Settings", styleSlate, true, icoGear) {
		ov = ovNone
		settingsFrom = scTable
		openScreen(scSettings)
	}
	y += 48
	label := "Leave the table"
	if host != nil && !solo {
		label = "Close the table"
	}
	if Button(Rect{x, y, w, 38}, label, styleSlate, true, icoX) {
		ov = ovNone
		leaveTable("")
	}
}

func deckOverlay(p *PlayerState) {
	r := sheet(700, 440)
	if _, cl := head(r, "YOUR DECK", false); cl || myRun == nil {
		ov = ovNone
		return
	}
	deck := myRun.Deck
	cw := math.Min((r.W-36-6*6)/7, (r.H-235)/2/1.4)
	ch := cw * 1.4
	for i, c := range deck {
		cx := r.X + 18 + float64(i%7)*(cw+6)
		cy := r.Y + 56 + float64(i/7)*(ch+8)
		cc := c
		ops = append(ops, func(dst *ebiten.Image) { drawCard(dst, cc, cx, cy, cw, ch, cardLook{}) })
	}
	y := r.Y + 56 + 2*(ch+8) + 10
	x := r.X + 18
	Text(fmt.Sprintf("Flush %d   Straight %d   Special: %s", myRun.Rules.flush, myRun.Rules.straight, hands[p.Special].name), x, y, 13, colInk, text.AlignStart)
	y += 22
	g, m := synergyOf(p.Picks)
	for _, h := range myRun.Known() {
		b, mu := handLevel(h, p.Levels[h]+1)
		t := fmt.Sprintf("%s  Lv %d:  %d x %d", hands[h].name, p.Levels[h]+1, b, mu)
		Text(t, x, y, 12, colInk, text.AlignStart)
		Text(handHow(h, myRun.Rules), x+260, y, 11, colMuted, text.AlignStart)
		y += 17
	}
	if m > 1 {
		Text(fmt.Sprintf("Synergy x%.1f (%d %s)", m, g, familyNames[majorFamily(p.Picks)]), x, y+4, 12, colT, text.AlignStart)
	}
}

func hexOverlay(p *PlayerState) {
	if hexSlot >= len(p.Hexes) {
		ov = ovNone
		return
	}
	h := hexes[p.Hexes[hexSlot]]
	r := sheet(420, 300)
	if _, cl := head(r, "CAST "+strings.ToUpper(h.name), false); cl {
		ov = ovNone
		return
	}
	y := wrapText(h.desc, r.X+18, r.Y+64, r.W-36, 13, colInk, text.AlignStart) + 6
	Label("ON WHO?", r.X+18, y)
	y += 14
	for _, q := range view.alive() {
		if q.ID == myID {
			continue
		}
		if Button(Rect{r.X + 18, y, r.W - 36, 34}, q.Name, styleViolet, true, nil) {
			act(action{T: "cast", Item: hexSlot, Target: q.ID})
			ov = ovNone
			return
		}
		y += 42
	}
}

func priceOverlay(p *PlayerState) {
	if !hasCharm(p, priceCharm) {
		ov = ovNone
		return
	}
	r := sheet(380, 220)
	if _, cl := head(r, "SELL TO PLAYERS", false); cl {
		ov = ovNone
		return
	}
	x, y := r.X+18, r.Y+64
	Text(charms[priceCharm].name, x, y, 15, colT, text.AlignStart)
	y += 30
	if Button(Rect{x, y, 40, 34}, "-", styleSlate, priceAmt > 1, nil) {
		priceAmt--
	}
	Text(fmt.Sprintf("$%d", priceAmt), x+80, y+17, 18, colInk, text.AlignCenter)
	if Button(Rect{x + 120, y, 40, 34}, "+", styleSlate, priceAmt < 30, nil) {
		priceAmt++
	}
	if Button(Rect{r.X + r.W - 140, r.Y + r.H - 50, 124, 36}, "List it", styleGold, true, nil) {
		act(action{T: "list", Charm: priceCharm, Price: priceAmt})
		ov = ovNone
	}
}
