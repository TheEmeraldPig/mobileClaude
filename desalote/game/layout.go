package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The open table layout: small floating pieces instead of panels down the sides.
//   - the players as pills along the top (tap one for details)
//   - a compact score card at the top left
//   - your charms as a row of coins at the top right (tap for details); hexes,
//     tomes, trading and your deck open in their own menus

// ---- Players ----

var pillPlayer int // the player shown in ovPlayer

// playerPills draws the players along the top and returns the y below them
func playerPills(y float64) float64 {
	ps := view.Players
	n := float64(len(ps))
	gap := 8.0
	avail := logicalW - 16 - 52
	pw := math.Min(200, (avail-gap*(n-1))/n)
	x0 := 8 + (avail-(pw*n+gap*(n-1)))/2
	h := 36.0
	for i, p := range ps {
		r := Rect{x0 + float64(i)*(pw+gap), y, pw, h}
		if ptr.released && !ptr.consumed && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY) {
			ptr.consumed = true
			pillPlayer, ov = p.ID, ovPlayer
			sfx("open")
		}
		me := p.ID == myID
		name := p.Name
		for textWidth(name, 12, false) > pw-62 && len(name) > 3 {
			name = name[:len(name)-1]
		}
		nameCol := color.Color(colInk)
		switch {
		case me:
			nameCol = colT
		case p.Out || p.Gone:
			nameCol = colMuted
		}
		showLives := view.Mode != ModeCoop || i == 0
		lives, maxLives := p.Lives, view.MaxLives
		status := playerStatus(p)
		money := fmt.Sprintf("$%d", p.Money)
		hexed := len(p.OnMe) > 0 || len(p.Active) > 0
		ops = append(ops, func(dst *ebiten.Image) {
			body := color.Color(color.NRGBA{40, 22, 24, 200})
			if me {
				body = color.NRGBA{80, 46, 40, 220}
			}
			roundRect(dst, r.X, r.Y+3, r.W, r.H, h/2, color.NRGBA{0, 0, 0, 70})
			roundRect(dst, r.X, r.Y, r.W, r.H, h/2, body)
			drawText(dst, name, r.X+14, r.Y+11, textOpt{size: 12, col: nameCol})
			drawText(dst, money, r.X+14, r.Y+26, textOpt{size: 11, display: true, col: colT})
			drawText(dst, status, r.X+r.W-12, r.Y+26, textOpt{size: 11, col: colMuted, align: text.AlignEnd})
			if showLives {
				for l := 0; l < maxLives && l < 5; l++ {
					drawHeart(dst, r.X+r.W-14-float64(l+1)*11, r.Y+5, 1.35, l < lives)
				}
			}
			if hexed {
				circle(dst, r.X+6, r.Y+h/2, 3, rgb(0xc04aa0))
			}
		})
	}
	return y + h
}

// ---- Score card ----

// scoreCard draws the blind, target, round score and points x mult at the top left
// and returns its rect
func scoreCard(p *PlayerState, r *Run, top float64) Rect {
	w := math.Min(220, logicalW*0.25)
	x, y := 10.0, top
	ante, blind := view.Ante, view.Blind
	if view.Mode == ModeRace {
		ante, blind = p.Ante, p.Blind
	}
	bname := blindNames[blind]
	if view.Showdown && view.Mode != ModeRace {
		bname = "Showdown"
	}
	// the target line
	tLabel, tVal := "Target", ""
	switch {
	case view.Mode == ModeCoop:
		var sum int64
		for _, q := range view.alive() {
			sum += q.Score
		}
		tLabel, tVal = "Team "+commas(sum)+" /", commas(view.Target)
	case r.Target > 0:
		tVal = commas(r.Target)
	default:
		tLabel = "Play every hand"
	}
	// the score rolls up to the real one once the hand has scored
	shown := r.Score
	if anim != nil && animating() {
		shown -= anim.res.Total
	}
	dispScore += (float64(shown) - dispScore) * 0.12
	if math.Abs(float64(shown)-dispScore) < 1 {
		dispScore = float64(shown)
	}
	score := commas(int64(math.Round(dispScore)))
	// the hand: its name, and its points x mult
	hname, pts, mlt := "", "", ""
	if anim != nil && animating() {
		b, m, done := animNumbers()
		hname, pts, mlt = hands[anim.res.Hand].name, fmt.Sprint(b), trimFloat(m)
		if done {
			hname = "= " + commas(anim.res.Total)
		}
	} else if h, ok := r.Preview(); ok && !r.Done {
		b, m := handLevel(h, r.Levels[h]+1)
		hname, pts, mlt = hands[h].name, fmt.Sprint(b), fmt.Sprint(m)
	}
	hexLine := ""
	if len(r.Hexes) > 0 {
		var names []string
		for _, h := range r.Hexes {
			names = append(names, hexes[h].name)
		}
		hexLine = "Hexed: " + strings.Join(names, ", ")
	}
	h := 150.0
	if hexLine != "" {
		h += 18
	}
	card := Rect{x, y, w, h}
	ops = append(ops, func(dst *ebiten.Image) {
		roundRect(dst, card.X, card.Y+4, card.W, card.H, 14, color.NRGBA{0, 0, 0, 80})
		roundRect(dst, card.X, card.Y, card.W, card.H, 14, color.NRGBA{40, 22, 24, 215})
		cx := card.X + card.W/2
		drawText(dst, fmt.Sprintf("ANTE %d/%d  -  %s", ante+1, lengthAntes[view.Length], strings.ToUpper(bname)), cx, card.Y+14,
			textOpt{size: 10, display: true, col: colT, align: text.AlignCenter, shadow: true})
		if tVal != "" {
			drawText(dst, tLabel+" "+tVal, cx, card.Y+33, textOpt{size: 13, col: colDanger, align: text.AlignCenter})
		} else {
			drawText(dst, tLabel, cx, card.Y+33, textOpt{size: 12, col: colMuted, align: text.AlignCenter})
		}
		roundRect(dst, card.X+10, card.Y+46, card.W-20, 34, 10, colPanelLo)
		drawText(dst, score, cx, card.Y+62, textOpt{size: 21, display: true, align: text.AlignCenter, shadow: true})
		drawText(dst, hname, cx, card.Y+94, textOpt{size: 12, col: colInk, align: text.AlignCenter})
		bw := (card.W - 20 - 18) / 2
		by := card.Y + 106
		bevel(dst, Rect{card.X + 10, by, bw, 32}, colB, rgb(0x9fd2ff), colS, false)
		bevel(dst, Rect{card.X + 28 + bw, by, bw, 32}, colRedBtn, rgb(0xff9a80), colRedLo, false)
		drawText(dst, pts, card.X+10+bw/2, by+15, textOpt{size: 17, display: true, align: text.AlignCenter, shadow: true})
		drawText(dst, "X", cx, by+15, textOpt{size: 14, display: true, col: colDanger, align: text.AlignCenter, shadow: true})
		drawText(dst, mlt, card.X+28+bw+bw/2, by+15, textOpt{size: 17, display: true, align: text.AlignCenter, shadow: true})
		if hexLine != "" {
			drawText(dst, hexLine, cx, card.Y+h-12, textOpt{size: 11, col: rgb(0xd070c0), align: text.AlignCenter})
		}
	})
	return card
}

// ---- Charms and hexes ----

// charmRow draws your charms as coins at the top right (with a Hexes button when you
// have some) and returns its height. Tapping a coin opens the charms menu; with quick
// actions on, Ctrl+click sells it straight away when selling is allowed.
func charmRow(p *PlayerState, top float64) float64 {
	sz := 34.0
	x := logicalW - 12 - float64(maxCharms)*(sz+6)
	for i := 0; i < maxCharms; i++ {
		cx := x + float64(i)*(sz+6)
		r := Rect{cx, top, sz, sz}
		if i >= len(p.Charms) {
			ops = append(ops, func(dst *ebiten.Image) { circle(dst, r.X+sz/2, r.Y+sz/2, sz/2-2, color.NRGBA{0, 0, 0, 60}) })
			continue
		}
		id := p.Charms[i]
		frozen := p.Frozen == id
		bob := math.Sin(float64(frame)/50+float64(i)*1.1) * 1.5
		ops = append(ops, func(dst *ebiten.Image) { drawCharm(dst, id, r.X, r.Y+bob, sz, frozen) })
		if ptr.released && !ptr.consumed && r.Has(ptr.x, ptr.y) {
			ptr.consumed = true
			if quickMod() && canTrade(view, p) {
				act(action{T: "sell", Charm: id})
				sfx("coin")
			} else {
				ov = ovCharms
				sfx("open")
			}
		} else if !isTouch && r.Has(hoverX, hoverY) && ov == ovNone {
			tip(charms[id].name, charms[id].desc)
		}
	}
	if len(p.Hexes) > 0 && view.Mode != ModeCoop && !p.Out {
		bw := 104.0
		if Button(Rect{x - bw - 10, top + 2, bw, 30}, fmt.Sprintf("Hexes (%d)", len(p.Hexes)), styleViolet, true, nil) {
			ov = ovHexes
		}
	}
	return sz
}

// Ctrl (or Cmd) held: the quick actions of the Handy controls
func quickMod() bool {
	return prefs.QuickAct && !isTouch && (ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta))
}

// ---- Menus ----

func charmsOverlay(p *PlayerState) {
	r := sheet(560, 380)
	if _, cl := head(r, fmt.Sprintf("YOUR CHARMS %d/%d", len(p.Charms), maxCharms), false); cl {
		ov = ovNone
		return
	}
	x, y, w := r.X+18, r.Y+60, r.W-36
	if len(p.Charms) == 0 {
		wrapText("No charms yet. Win them at the auction between rounds, or buy them from other players.", x, y+10, w, 14, colMuted, text.AlignStart)
		return
	}
	trade := canTrade(view, p)
	rh := math.Min(52, (r.H-80)/float64(maxCharms)-6)
	for _, id := range p.Charms {
		d := charms[id]
		cid, yy := id, y
		frozen := p.Frozen == id
		ops = append(ops, func(dst *ebiten.Image) { drawCharm(dst, cid, x, yy+(rh-36)/2, 36, frozen) })
		Text(d.name, x+46, y+rh*0.3, 14, colT, text.AlignStart)
		desc := d.desc
		if frozen {
			desc = "Frozen this round. " + desc
		}
		Text(desc, x+46, y+rh*0.72, 11, colInk, text.AlignStart)
		if trade && Button(Rect{x + w - 84, y + (rh-30)/2, 84, 30}, fmt.Sprintf("Sell $%d", sellPrice(id)), styleRed, true, nil) {
			act(action{T: "sell", Charm: id})
			sfx("coin")
		}
		y += rh + 6
	}
	if !trade {
		Text("You can sell charms between rounds.", x, r.Y+r.H-20, 11, colMuted, text.AlignStart)
	}
}

func hexesOverlay(p *PlayerState) {
	r := sheet(520, 320)
	if _, cl := head(r, "YOUR HEXES", false); cl {
		ov = ovNone
		return
	}
	x, y, w := r.X+18, r.Y+60, r.W-36
	if len(p.Hexes) == 0 {
		Text("No hexes. Win them at the auction.", x, y+10, 14, colMuted, text.AlignStart)
		return
	}
	for i, h := range p.Hexes {
		d := hexes[h]
		yy := y
		ops = append(ops, func(dst *ebiten.Image) { drawHex(dst, x, yy, 30, 40) })
		Text(d.name, x+42, y+12, 14, colT, text.AlignStart)
		Text(d.desc, x+42, y+30, 11, colInk, text.AlignStart)
		if Button(Rect{x + w - 90, y + 5, 90, 30}, "Cast", styleViolet, inMatch() && !p.Out, nil) {
			ov, hexSlot = ovHex, i
		}
		y += 50
	}
}

func playerOverlay() {
	q := view.player(pillPlayer)
	if q == nil {
		ov = ovNone
		return
	}
	r := sheet(460, 330)
	if _, cl := head(r, strings.ToUpper(q.Name), false); cl {
		ov = ovNone
		return
	}
	x, y := r.X+18, r.Y+66
	line := func(k, v string) {
		Text(k, x, y, 13, colMuted, text.AlignStart)
		Text(v, x+120, y, 14, colInk, text.AlignStart)
		y += 24
	}
	state := "playing"
	switch {
	case q.Gone:
		state = "left the table"
	case q.Out:
		state = "out"
	}
	line("Status", state)
	if view.Mode != ModeCoop {
		line("Lives", fmt.Sprintf("%d of %d", max(0, q.Lives), view.MaxLives))
	}
	line("Money", fmt.Sprintf("$%d", q.Money))
	line("This round", commas(q.Score))
	line("Whole match", commas(q.Total))
	if view.Mode == ModeRace {
		line("Progress", fmt.Sprintf("ante %d, %s", q.Ante+1, blindNames[q.Blind]))
	}
	var names []string
	for _, c := range q.Charms {
		names = append(names, charms[c].name)
	}
	if len(names) == 0 {
		names = []string{"none"}
	}
	Text("Charms", x, y, 13, colMuted, text.AlignStart)
	wrapText(strings.Join(names, ", "), x+120, y, r.W-156, 14, colInk, text.AlignStart)
	if len(q.OnMe)+len(q.Active) > 0 {
		Text("Hexed!", r.X+r.W-20, r.Y+r.H-22, 13, rgb(0xd070c0), text.AlignEnd)
	}
}

func tomesOverlay(p *PlayerState) {
	r := sheet(620, 330)
	if _, cl := head(r, "TOMES", false); cl {
		ov = ovNone
		return
	}
	tomes(p, Rect{r.X + 18, r.Y + 56, r.W - 36, r.H - 70})
}

func tradeOverlay(p *PlayerState) {
	r := sheet(680, 360)
	if _, cl := head(r, "TRADE CHARMS", false); cl {
		ov = ovNone
		return
	}
	trade(p, Rect{r.X + 18, r.Y + 56, r.W - 36, r.H - 70})
}
