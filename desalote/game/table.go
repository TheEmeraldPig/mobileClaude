package game

import (
	"fmt"
	"image/color"
	"math"
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
)

var (
	ov         overlay
	hexSlot    int
	priceCharm string
	priceAmt   int
	shopTab    int
	draftPicks []HandType
	dealAt     int
)

// The scoring animation for the last hand played
type scoreAnim struct {
	res   *ScoreResult
	start int
}

var anim *scoreAnim

func animLen(res *ScoreResult) int {
	n := 50 + 14*len(res.Scoring)
	if res.Special {
		n += 30
	}
	if prefs.Fast {
		n /= 2
	}
	return n
}

func animating() bool { return anim != nil && frame-anim.start < animLen(anim.res) }

func tableScreen() {
	ops = append(ops, drawFelt)
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

// ---- The players along the top ----

func topStrip(y float64) float64 {
	ps := view.Players
	n := float64(len(ps))
	x0, w := 8.0, logicalW-8-52
	gap := 8.0
	cw := (w - gap*(n-1)) / n
	h := 46.0
	for i, p := range ps {
		r := Rect{x0 + float64(i)*(cw+gap), y, cw, h}
		body := color.Color(colPanel)
		if p.ID == myID {
			body = colPanelHi
		}
		pp := p
		ops = append(ops, func(dst *ebiten.Image) {
			bevel(dst, r, body, colSlateHi, colPanelLo, false)
			nameCol := color.Color(colInk)
			if pp.ID == myID {
				nameCol = colT
			}
			if pp.Out || pp.Gone {
				nameCol = colMuted
			}
			name := pp.Name
			for textWidth(name, 13, false) > r.W-70 && len(name) > 3 {
				name = name[:len(name)-1]
			}
			drawText(dst, name, r.X+8, r.Y+12, textOpt{size: 13, col: nameCol})
			// lives
			if view.Mode != ModeCoop || pp == view.Players[0] {
				for l := 0; l < view.MaxLives && l < 5; l++ {
					drawHeart(dst, r.X+r.W-12-float64(l+1)*13, r.Y+6, 1.6, l < pp.Lives)
				}
			}
			drawText(dst, fmt.Sprintf("$%d", pp.Money), r.X+8, r.Y+32, textOpt{size: 12, display: true, col: colT})
			drawText(dst, playerStatus(pp), r.X+r.W-8, r.Y+32, textOpt{size: 12, col: colMuted, align: text.AlignEnd})
			if len(pp.OnMe) > 0 || len(pp.Active) > 0 {
				drawText(dst, "HEXED", r.X+r.W/2, r.Y+32, textOpt{size: 9, display: true, col: rgb(0xb070ff), align: text.AlignCenter})
			}
		})
	}
	return y + h
}

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
		return "picking"
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

func lobbyScreen() {
	r := sheet(780, 470)
	Title("TABLE", r.X+16, r.Y+28)
	isHost := host != nil
	x, y := r.X+18, r.Y+60
	// seats
	lw := math.Min(300, r.W*0.4)
	Label("PLAYERS", x, y)
	y += 14
	for i := 0; i < view.maxPlayers(); i++ {
		row := Rect{x, y, lw, 34}
		Inset(row, colSlateLo)
		if i < len(view.Players) {
			p := view.Players[i]
			who := p.Name
			if p.Bot {
				who += "  (bot)"
			}
			if p.ID == 0 {
				who += "  (host)"
			}
			c := color.Color(colInk)
			if p.ID == myID {
				c = colT
			}
			Text(who, row.X+10, row.Y+17, 14, c, text.AlignStart)
			if isHost && p.Bot && Button(Rect{row.X + row.W - 74, row.Y + 3, 70, 28}, "Remove", styleSlate, true, nil) {
				host.removePlayer(p.ID)
			}
		} else {
			Text("Empty seat", row.X+10, row.Y+17, 13, colMuted, text.AlignStart)
			if isHost && Button(Rect{row.X + row.W - 74, row.Y + 3, 70, 28}, "+ Bot", styleSlate, true, nil) {
				host.addBot()
			}
		}
		y += 40
	}
	// how friends join
	y += 6
	if isHost && !solo {
		Label("FRIENDS JOIN", x, y)
		y += 18
		if lanServer != nil {
			if a := myAddress(); a != "" {
				wrapText("On the same Wi-Fi: Join a table, or type "+a, x, y, lw, 12, colInk, text.AlignStart)
			} else {
				wrapText("Connect to Wi-Fi or a hotspot so friends can find you", x, y, lw, 12, colMuted, text.AlignStart)
			}
			y += 36
		}
		if netCode != "" && lanServer == nil {
			Text("Online code: "+netCode, x, y+6, 16, colT, text.AlignStart)
			y += 26
		} else if relayURL() != "" && Button(Rect{x, y, lw, 30}, "Use an online code instead", styleSlate, true, nil) {
			startHostingOnline()
		}
		if netStatus != "" {
			Text(netStatus, x, r.Y+r.H-22, 11, colMuted, text.AlignStart)
		}
	} else if !isHost {
		Text("Waiting for the host to start...", x, y+8, 13, colMuted, text.AlignStart)
	}
	// match settings
	sx := x + lw + 30
	sw := r.X + r.W - 18 - sx
	sy := r.Y + 60
	Label("MODE", sx, sy)
	modeLabels := []string{"Duel", "Last Standing", "Co-op", "Race"}
	if c := segWrap(sx, sy+14, sw, modeLabels, view.Mode, isHost); c >= 0 {
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
	by := wrapText(modeBlurbs[view.Mode], sx, sy+92, sw, 12, colMuted, text.AlignStart)
	sy = math.Max(by+4, sy+150)
	Label("LENGTH", sx, sy)
	ll := []string{}
	for i, n := range lengthNames {
		ll = append(ll, fmt.Sprintf("%s (%d)", n, lengthAntes[i]))
	}
	if c := segWrap(sx, sy+14, sw, ll, view.Length, isHost); c >= 0 {
		host.Length, prefs.Length = c, c
		host.changed()
		savePrefs()
	}
	sy += 60
	Label("LIVES", sx, sy)
	if c := segWrap(sx, sy+14, sw, []string{"1", "2", "3", "4", "5"}, view.MaxLives-1, isHost); c >= 0 {
		host.MaxLives, prefs.Lives = c+1, c+1
		host.changed()
		savePrefs()
	}
	if isHost {
		ok := host.canStart()
		label := "Start the match"
		if !ok {
			label = "Need 2+ players"
		}
		if Button(Rect{r.X + r.W - 200, r.Y + r.H - 52, 184, 38}, label, styleGold, ok, icoPlay) {
			host.start()
			sfx("deal")
		}
	}
	if Button(Rect{r.X + r.W - 46, r.Y + 12, 32, 32}, "", styleSlate, true, icoX) {
		leaveTable("")
	}
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

// ---- Draft: pick your three hands ----

func draftScreen(p *PlayerState) {
	y := topStrip(8) + 10
	if myRun == nil {
		return
	}
	// left: your cards
	lw := math.Min(logicalW*0.46, 430)
	left := Rect{10, y, lw, logicalH - y - 10}
	Panel(left)
	Label("YOUR 13 CARDS", left.X+12, left.Y+16)
	cw := math.Min((lw-24-6*5)/7, (left.H-130)/2/1.4)
	ch := cw * 1.4
	deck := myRun.Deck
	for i, c := range deck {
		col, row := i%7, i/7
		cx := left.X + 12 + float64(col)*(cw+5)
		cy := left.Y + 32 + float64(row)*(ch+8)
		cc := c
		ops = append(ops, func(dst *ebiten.Image) { drawCard(dst, cc, cx, cy, cw, ch, cardLook{}) })
	}
	ty := left.Y + 32 + 2*(ch+8) + 8
	Text(fmt.Sprintf("Flush: %d cards of one suit", myRun.Rules.flush), left.X+12, ty, 13, colInk, text.AlignStart)
	Text(fmt.Sprintf("Straight: %d ranks in a row", myRun.Rules.straight), left.X+12, ty+18, 13, colInk, text.AlignStart)
	wrapText("Your deck is locked for the whole match.", left.X+12, ty+38, lw-24, 11, colMuted, text.AlignStart)

	// right: family, special, and the hands to pick
	rx := left.X + lw + 10
	right := Rect{rx, y, logicalW - rx - 10, logicalH - y - 10}
	Panel(right)
	x, w := right.X+12, right.W-24
	fc := familyColors[p.Family]
	Label("YOUR FAMILY", x, right.Y+16)
	Text(familyNames[p.Family], x, right.Y+36, 16, rgb(fc[1]), text.AlignStart)
	Label("YOUR SPECIAL", x+w*0.5, right.Y+16)
	Text(hands[p.Special].name, x+w*0.5, right.Y+36, 16, colT, text.AlignStart)
	yy := wrapText(familyBlurb[p.Family], x, right.Y+56, w, 11, colMuted, text.AlignStart)
	if p.Ready {
		Label("YOUR HANDS", x, yy+8)
		hy := yy + 26
		for _, h := range p.Picks {
			Text(hands[h].name, x, hy, 14, colInk, text.AlignStart)
			hy += 20
		}
		Text("Waiting for the others...", x, right.Y+right.H-30, 13, colMuted, text.AlignStart)
		if Button(Rect{right.X + right.W - 112, right.Y + right.H - 46, 100, 34}, "Change", styleSlate, true, nil) {
			draftPicks = append([]HandType{}, p.Picks...)
			act(action{T: "unpick"})
		}
		return
	}
	Label("PICK 3 HANDS (1+ FROM YOUR FAMILY)", x, yy+8)
	by := yy + 22
	bh := math.Max(30, math.Min(40, (right.Y+right.H-50-by)/5-6))
	for _, h := range p.Offer {
		on := false
		for _, d := range draftPicks {
			on = on || d == h
		}
		r := Rect{x, by, w, bh}
		st := styleSlate
		if on {
			st = styleViolet
		}
		if Button(r, "", st, true, nil) {
			if on {
				var keep []HandType
				for _, d := range draftPicks {
					if d != h {
						keep = append(keep, d)
					}
				}
				draftPicks = keep
			} else if len(draftPicks) < 3 {
				draftPicks = append(draftPicks, h)
			}
		}
		hh := h
		f := hands[h].fam
		ops = append(ops, func(dst *ebiten.Image) {
			fcol := familyColors[f]
			fill(dst, r.X+6, r.Y+6, 6, r.H-12, rgb(fcol[0]))
			drawText(dst, hands[hh].name, r.X+20, r.Y+r.H/2-5, textOpt{size: 14, shadow: true})
			drawText(dst, handHow(hh, myRun.Rules), r.X+20, r.Y+r.H/2+9, textOpt{size: 10, col: colL})
			b, m := handLevel(hh, 1)
			drawText(dst, fmt.Sprintf("%d x %d", b, m), r.X+r.W-10, r.Y+r.H/2, textOpt{size: 12, display: true, col: colT, align: text.AlignEnd})
		})
		by += bh + 6
	}
	g, m := synergyOf(draftPicks)
	syn := "Synergy: none yet"
	if m > 1 {
		syn = fmt.Sprintf("Synergy: x%.1f (%d %s hands)", m, g, familyNames[majorFamily(draftPicks)])
	}
	Text(syn, x, right.Y+right.H-28, 13, colT, text.AlignStart)
	if Button(Rect{right.X + right.W - 112, right.Y + right.H - 46, 100, 34}, "Ready", styleGold, validPicks(p, draftPicks), icoPlay) {
		act(action{T: "pick", Picks: draftPicks})
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
	top := topStrip(8) + 8
	r := myRun
	if r == nil {
		return
	}
	lw := math.Min(210, logicalW*0.22)
	rw := math.Min(180, logicalW*0.19)
	// left: the blind and your score
	left := Rect{8, top, lw, logicalH - top - 8}
	Panel(left)
	x := left.X + 10
	y := left.Y + 14
	ante, blind := view.Ante, view.Blind
	if view.Mode == ModeRace {
		ante, blind = p.Ante, p.Blind
	}
	Label(fmt.Sprintf("ANTE %d / %d", ante+1, lengthAntes[view.Length]), x, y)
	bname := blindNames[blind]
	if view.Showdown && view.Mode != ModeRace {
		bname = "SHOWDOWN"
	}
	Text(bname, x, y+20, 15, colInk, text.AlignStart)
	y += 40
	switch {
	case view.Mode == ModeCoop:
		var sum int64
		for _, q := range view.alive() {
			sum += q.Score
		}
		Text("Team target", x, y, 11, colMuted, text.AlignStart)
		Text(commas(view.Target), x, y+16, 16, colDanger, text.AlignStart)
		Text("Team so far: "+commas(sum), x, y+34, 12, colInk, text.AlignStart)
		y += 50
	case r.Target > 0:
		Text("Target", x, y, 11, colMuted, text.AlignStart)
		Text(commas(r.Target), x, y+16, 16, colDanger, text.AlignStart)
		y += 36
	default:
		wrapText("Play all your hands. Lowest total loses a life.", x, y, lw-20, 11, colMuted, text.AlignStart)
		y += 36
	}
	Text("Your score", x, y, 11, colMuted, text.AlignStart)
	Text(commas(r.Score), x, y+18, 22, colT, text.AlignStart)
	y += 44
	// what the selection makes
	if h, ok := r.Preview(); ok && !r.Done {
		b, m := handLevel(h, r.Levels[h]+1)
		Inset(Rect{x, y, lw - 20, 40}, colPanelLo)
		Text(hands[h].name, x+8, y+12, 13, colInk, text.AlignStart)
		Text(fmt.Sprintf("%d x %d", b, m), x+8, y+28, 12, colT, text.AlignStart)
	}
	y += 50
	Text(fmt.Sprintf("Plays %d", r.Plays), x, y, 13, rgb(0x7ab4e0), text.AlignStart)
	Text(fmt.Sprintf("Discards %d", r.Discards), x+lw*0.48, y, 13, colDanger, text.AlignStart)
	y += 20
	if len(r.Hexes) > 0 {
		var names []string
		for _, h := range r.Hexes {
			names = append(names, hexes[h].name)
		}
		wrapText("Hexed: "+strings.Join(names, ", "), x, y, lw-20, 11, rgb(0xb070ff), text.AlignStart)
	}
	if left.Y+left.H-44 > y+18 && Button(Rect{x, left.Y + left.H - 42, lw - 20, 32}, "Your deck", styleSlate, true, nil) {
		ov = ovDeck
	}

	// right: charms and hexes
	right := Rect{logicalW - rw - 8, top, rw, logicalH - top - 8}
	charmPanel(right, p)

	// middle: the hand, the buttons, and the last hand scored
	mx := left.X + left.W + 10
	mw := right.X - 10 - mx
	bh := 38.0
	by := logicalH - bh - 12
	ch := math.Min(150, (by-top)*0.42)
	cw := ch * 0.7
	n := len(r.Hand)
	step := cw + 8
	if n > 1 && step*float64(n-1)+cw > mw-10 {
		step = (mw - 10 - cw) / float64(n-1)
	}
	hx := mx + (mw-(step*float64(max(0, n-1))+cw))/2
	hy := by - 14 - ch
	busy := animating()
	if !r.Done {
		for i := 0; i < n; i++ {
			d := r.Hand[i]
			cr := Rect{hx + float64(i)*step, hy, cw, ch}
			hit := cr
			if i < n-1 {
				hit.W = step
			}
			sel := r.Sel[i]
			if sel {
				cr.Y -= ch * 0.18
			}
			// cards slide in when dealt
			if age := frame - dealAt; age < 20+i*3 {
				t := math.Max(0, float64(age-i*3)) / 20
				cr.Y += (1 - t) * (1 - t) * 50
			}
			if !busy && ptr.released && hit.Has(ptr.x, ptr.y) && hit.Has(ptr.startX, ptr.startY) {
				ptr.consumed = true
				r.Toggle(i)
				if r.Sel[i] {
					sfx("select")
				} else {
					sfx("unselect")
				}
			}
			c, fd := r.Deck[d], r.FaceDown[d]
			ops = append(ops, func(dst *ebiten.Image) { drawCard(dst, c, cr.X, cr.Y, cw, ch, cardLook{faceDown: fd, selected: sel}) })
		}
		bw := math.Min(130, (mw-20)/3)
		bx := mx + (mw-(bw*3+20))/2
		if Button(Rect{bx, by, bw, bh}, "Play", styleGold, r.CanPlay() && !busy, icoPlay) {
			if res := r.Play(); res != nil {
				anim = &scoreAnim{res: res, start: frame}
				sfx("play")
				reportRun()
				dealAt = frame
			}
		}
		if Button(Rect{bx + bw + 10, by, bw, bh}, fmt.Sprintf("Discard"), styleSlate, r.CanDiscard() && !busy, nil) {
			r.Discard()
			sfx("discard")
			dealAt = frame
		}
		spLabel := "Special"
		if r.SpecialLeft > 1 {
			spLabel = fmt.Sprintf("Special x%d", r.SpecialLeft)
		}
		if Button(Rect{bx + 2*(bw+10), by, bw, bh}, spLabel, styleViolet, r.CanSpecial() && !busy, nil) {
			if res := r.UseSpecial(); res != nil {
				anim = &scoreAnim{res: res, start: frame}
				sfx("special")
				reportRun()
				dealAt = frame
			}
		}
		if r.Plays == 0 && !r.Done && !busy {
			Text("Out of plays: use your special, or", mx+mw/2, hy-24, 12, colMuted, text.AlignCenter)
			if Button(Rect{mx + mw/2 - 60, hy - 12 - 52, 120, 30}, "Finish", styleSlate, true, nil) {
				r.Finish()
				reportRun()
			}
		}
		if !isTouch && !busy {
			Text(hands[myRun.Special].name+": "+hands[myRun.Special].how, mx+mw/2, by+bh+6, 10, colMuted, text.AlignCenter)
		}
		if anim == nil && len(r.Selected()) == 0 && frame-dealAt > 40 {
			hint := "Pick up to 5 cards, then Play. Discard swaps them for new ones."
			if !isTouch {
				hint = "Click up to 5 cards (or keys 1-8), then Play (Enter). D discards, S uses your special."
			}
			Text(hint, mx+mw/2, (top+hy)/2, 12, colMuted, text.AlignCenter)
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
		drawAnim(mx, top+6, mw, hy-top-30)
		if ptr.released && !ptr.consumed && animating() {
			anim.start = frame - animLen(anim.res) // tap to skip
		}
	}
}

// The scoring animation: the played cards light up one by one while the points add up
func drawAnim(x, y, w, h float64) {
	a := anim
	res := a.res
	t := frame - a.start
	total := animLen(res)
	speed := 14
	if prefs.Fast {
		speed = 7
	}
	n := len(res.Cards)
	ch := math.Min(h-50, 110)
	if ch < 40 {
		return
	}
	cw := ch * 0.7
	step := cw + 8
	cx := x + (w-(step*float64(n-1)+cw))/2
	lit := min(len(res.Scoring), max(0, (t-10)/speed+1))
	scoringSet := map[int]bool{}
	for k := 0; k < lit; k++ {
		scoringSet[res.Scoring[k]] = true
	}
	fade := 1.0
	if t > total+60 {
		fade = math.Max(0, 1-float64(t-total-60)/30)
	}
	if fade <= 0 {
		return
	}
	for i, c := range res.Cards {
		cc := c
		ix := cx + float64(i)*step
		iy := y + 34
		if scoringSet[i] {
			iy -= 6
		}
		in := false
		for _, s := range res.Scoring {
			in = in || s == i
		}
		look := cardLook{glowGold: scoringSet[i], dim: !in}
		ops = append(ops, func(dst *ebiten.Image) { drawCard(dst, cc, ix, iy, cw, ch, look) })
	}
	if t%speed == 10%speed && t >= 10 && (t-10)/speed < len(res.Scoring) {
		sfx("tick")
	}
	if t == total-10 {
		sfx("score")
	}
	name := hands[res.Hand].name
	if res.Special {
		name = "SPECIAL: " + name
	}
	ShadowText(name, x+w/2-textWidth(name, 16, false)/2, y+12, 16, colT)
	// points x mult = total
	ty := y + 34 + ch + 22
	var line string
	if t < total-10 {
		line = fmt.Sprintf("%d points  x  %.1f mult", res.Base, res.Mult)
	} else {
		line = fmt.Sprintf("%d x %.1f = %s", res.Base, res.Mult, commas(res.Total))
	}
	Text(line, x+w/2, ty, 15, colInk, text.AlignCenter)
	if len(res.Notes) > 0 {
		Text(strings.Join(res.Notes, "   "), x+w/2, ty+20, 11, colL, text.AlignCenter)
	}
}

// The charms you hold and the hexes you can cast
func charmPanel(r Rect, p *PlayerState) {
	Panel(r)
	x, y := r.X+10, r.Y+14
	Label(fmt.Sprintf("CHARMS %d/%d", len(p.Charms), maxCharms), x, y)
	y += 14
	sz := math.Min(30, (r.H-140)/float64(maxCharms)-8)
	for _, id := range p.Charms {
		cid := id
		cy := y
		frozen := p.Frozen == id
		ops = append(ops, func(dst *ebiten.Image) { drawCharm(dst, cid, x, cy, sz, frozen) })
		Text(charms[id].name, x+sz+8, y+sz*0.45, 12, colInk, text.AlignStart)
		hit := Rect{x, y, r.W - 20, sz + 4}
		if ptr.released && hit.Has(ptr.x, ptr.y) && !ptr.consumed {
			toast(charms[id].name + ": " + charms[id].desc)
		} else if !isTouch && hit.Has(hoverX, hoverY) {
			tip(charms[id].name, charms[id].desc)
		}
		y += sz + 6
	}
	if len(p.Charms) == 0 {
		wrapText("None yet. Win them at the auction.", x, y+6, r.W-20, 11, colMuted, text.AlignStart)
		y += 34
	}
	if view.Mode == ModeCoop {
		return
	}
	y = math.Max(y+6, r.Y+r.H-30-float64(max(1, len(p.Hexes)))*34-14)
	Label("HEXES", x, y)
	y += 12
	if len(p.Hexes) == 0 {
		Text("None", x, y+12, 11, colMuted, text.AlignStart)
	}
	for i, h := range p.Hexes {
		if Button(Rect{x, y, r.W - 20, 30}, "Cast "+hexes[h].name, styleViolet, inMatch() && !p.Out, nil) {
			ov, hexSlot = ovHex, i
		}
		y += 34
	}
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
	r := sheet(320, 290)
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
