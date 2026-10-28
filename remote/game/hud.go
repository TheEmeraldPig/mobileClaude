package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The HUD while playing: health, level and coins at the top left, the island at the
// top, buttons at the top right, the petal bar at the bottom, and on phones the
// joystick and the attack and defend buttons.

var emotesOpen bool

var (
	icoMap    = pixIcon("........", "#.##.#..", "##.##.#.", "#.#.##.#", "##.#.##.", "#.##.#..", "........", "........")
	icoTalk   = pixIcon(".######.", "########", "#o#o#o##", "########", ".######.", "..##....", ".#......", "........")
	icoSword  = pixIcon("......##", ".....###", "....###.", "#..###..", ".####...", "..##....", ".#.#....", "#.......")
	icoShield = pixIcon("########", "#oooooo#", "#oooooo#", "#oooooo#", ".#oooo#.", ".#oooo#.", "..#oo#..", "...##...")
)

func hud() {
	hudRects = hudRects[:0]
	for k := range holdRects {
		delete(holdRects, k)
	}
	if me == nil {
		return
	}
	// health, level, coins
	bw := math.Min(220, logicalW*0.24)
	hp, mhp := math.Max(0, myHP), math.Max(1, myMaxHP)
	lvl, xp, need := me.Level, me.XP, mmo.XPNeed(me.Level)
	coins := me.Coins
	ops = append(ops, func(dst *ebiten.Image) {
		x, y := 14.0, 14.0
		bevel(dst, Rect{x, y, bw, 18}, colPanelLo, colO, colPanelHi, false)
		fill(dst, x+3, y+3, (bw-6)*hp/mhp, 12, color.NRGBA{240, 86, 112, 255})
		fill(dst, x+3, y+3, (bw-6)*hp/mhp, 3, color.NRGBA{255, 160, 176, 255})
		drawText(dst, fmt.Sprintf("%d / %d", int(math.Ceil(hp)), int(mhp)), x+bw/2, y+9, textOpt{size: 11, align: text.AlignCenter, shadow: true})
		y += 28
		drawText(dst, fmt.Sprintf("LV %d", lvl), x, y+5, textOpt{size: 10, display: true, col: colT, shadow: true})
		lw := 48.0
		fill(dst, x+lw, y+2, bw-lw, 7, colO)
		if need > 0 {
			fill(dst, x+lw+1, y+3, (bw-lw-2)*math.Min(1, float64(xp)/float64(need)), 5, colT)
		}
		y += 18
		coinIcon(dst, x, y)
		drawText(dst, mmo.FmtNum(coins), x+16, y+6, textOpt{size: 12, shadow: true, col: colGoldHi})
	})
	// the island
	if d := mmo.IsleByID(isleID); d != nil {
		name := d.Name
		sub := fmt.Sprintf("%d here", len(others)+1)
		if netSolo {
			sub = "Solo"
		}
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, name, logicalW/2, 18, textOpt{size: 11, display: true, col: colL, align: text.AlignCenter, shadow: true})
			drawText(dst, sub, logicalW/2, 34, textOpt{size: 11, col: colMuted, align: text.AlignCenter, shadow: true})
		})
	}
	// buttons at the top right
	bs := 40.0
	if isTouch {
		bs = 44
	}
	bx := logicalW - 14 - bs
	top := []struct {
		ico *ebiten.Image
		do  func()
	}{
		{icoPause, func() { openScreen(scPause) }},
		{icoBag, func() { openScreen(scBag) }},
		{icoMap, func() { openScreen(scTravel) }},
		{icoTalk, func() { emotesOpen = !emotesOpen }},
	}
	for _, b := range top {
		r := Rect{bx, 14, bs, bs}
		hudRects = append(hudRects, r)
		if Button(r, "", styleSlate, true, b.ico) {
			b.do()
		}
		bx -= bs + 10
	}
	if emotesOpen {
		ex := logicalW - 14
		for i := len(mmo.Emotes) - 1; i >= 0; i-- {
			w := textWidth(mmo.Emotes[i], 14, false) + 22
			ex -= w
			r := Rect{ex, 14 + bs + 12, w, 32}
			hudRects = append(hudRects, r)
			if Button(r, mmo.Emotes[i], styleViolet, true, nil) {
				sendEmote(i)
			}
			ex -= 8
		}
	}
	toolBar()
	if isTouch {
		useButtons()
		cx, cy := stickCenter()
		ops = append(ops, func(dst *ebiten.Image) { drawStick(dst, cx, cy) })
	}
	if frame < hintUntil && prefs.Tips {
		hint := "A / D to move, W or Space to jump (hold it in the air to hover). Hold the left mouse button to use your sword or pickaxe. 1 / 2 or Q: switch tools. E: bag, M: travel, 3-8: emotes."
		if isTouch {
			hint = "Push the stick up to jump, and keep it up in the air to hover. Hold the big button to use your tool; the small one switches between sword and pickaxe."
		}
		hy := 62.0
		if frame < toastUntil {
			hy = 98 // under the message
		}
		wrapText(hint, logicalW/2, hy, math.Min(520, logicalW*0.5), 13, colInk, text.AlignCenter)
	}
}

func sendEmote(i int) {
	send(mmo.CMsg{T: "emote", Emote: i})
	emotesOpen = false
}

func coinIcon(dst *ebiten.Image, x, y float64) {
	const p = 2.0
	rows := []string{".OOOO.", "OTTTTO", "OTLTTO", "OTTTTO", "OTTTSO", ".OOOO."}
	for r, row := range rows {
		for c, ch := range row {
			var col color.Color
			switch ch {
			case 'O':
				col = colO
			case 'T':
				col = colT
			case 'L':
				col = colGoldHi
			case 'S':
				col = colGoldLo
			default:
				continue
			}
			fill(dst, x+float64(c)*p, y+float64(r)*p, p, p, col)
		}
	}
}

// The equipped petals along the bottom; greyed out while they grow back
func petalBar() {
	n := len(me.Load)
	slot := 34.0
	if isTouch {
		slot = 38
	}
	gap := 6.0
	tw := float64(n)*slot + float64(n-1)*gap
	x0, y0 := (logicalW-tw)/2, logicalH-slot-12
	bar := Rect{x0 - 6, y0 - 6, tw + 12, slot + 12}
	hudRects = append(hudRects, bar)
	ops = append(ops, func(dst *ebiten.Image) { fill(dst, bar.X, bar.Y, bar.W, bar.H, color.NRGBA{19, 16, 39, 190}) })
	eqIdx := 0
	for i, p := range me.Load {
		r := Rect{x0 + float64(i)*(slot+gap), y0, slot, slot}
		out := true
		if p.Valid() {
			out = myAlive&(1<<eqIdx) != 0
			eqIdx++
		}
		petalSlot(r, p, false, !out)
	}
	if ptr.released && bar.Has(ptr.x, ptr.y) && bar.Has(ptr.startX, ptr.startY) {
		openScreen(scPetals)
	}
}

// A slot with a petal in it, on its rarity's colour
func petalSlot(r Rect, p mmo.PetalID, sel, dim bool) bool {
	clicked := ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY)
	ops = append(ops, func(dst *ebiten.Image) {
		body := color.Color(colPanelLo)
		hi, lo := color.Color(color.NRGBA{10, 8, 23, 255}), color.Color(colPanelHi)
		if k, rar, ok := p.Split(); ok {
			rc := mmo.Rarities[rar].Col
			body = color.NRGBA{uint8(rc[0] * 2 / 3), uint8(rc[1] * 2 / 3), uint8(rc[2] * 2 / 3), 255}
			hi = color.NRGBA{uint8(min(255, rc[0]+40)), uint8(min(255, rc[1]+40)), uint8(min(255, rc[2]+40)), 255}
			lo = color.NRGBA{uint8(rc[0] / 2), uint8(rc[1] / 2), uint8(rc[2] / 2), 255}
			bevel(dst, r, body, hi, lo, false)
			petalIcon(dst, r.X+r.W/2, r.Y+r.H/2-3, r.W*0.5, k)
			if r.W >= 40 {
				drawText(dst, k.Name, r.X+r.W/2, r.Y+r.H-8, textOpt{size: 9, align: text.AlignCenter, shadow: true})
			}
		} else {
			bevel(dst, r, body, hi, lo, false)
		}
		if sel {
			for _, e := range []Rect{{r.X - 3, r.Y - 6, r.W + 6, 3}, {r.X - 3, r.Y + r.H + 3, r.W + 6, 3}, {r.X - 6, r.Y - 3, 3, r.H + 6}, {r.X + r.W + 3, r.Y - 3, 3, r.H + 6}} {
				fill(dst, e.X, e.Y, e.W, e.H, colT)
			}
		}
		if dim {
			fill(dst, r.X, r.Y, r.W, r.H, color.NRGBA{16, 12, 32, 150})
		}
	})
	if clicked {
		sfx("click")
	}
	return clicked
}

// A petal drawn big, centred on (cx, cy)
func petalIcon(dst *ebiten.Image, cx, cy, size float64, k *mmo.PetalKind) {
	p := size / 5
	for y, row := range petalRows {
		for x := 0; x < len(row); x++ {
			var c [3]int
			switch row[x] {
			case 'o':
				c = [3]int{34, 28, 64}
			case 'B':
				c = k.Col
			case 'L':
				c = mix(k.Col, [3]int{255, 255, 255}, 0.55)
			case 'S':
				c = shadeRGB(k.Col, 0.75)
			default:
				continue
			}
			fill(dst, cx-size/2+float64(x)*p, cy-size/2+float64(y)*p, p, p, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
		}
	}
}

// Phones: hold the sword to attack, the shield to defend
func attackButtons() {
	big, small := 78.0, 54.0
	x := logicalW - 20 - big
	sx := x - small - 16
	if prefs.BtnSide == "left" {
		x, sx = 20, 20+big+16
	}
	ar := Rect{x, logicalH - 20 - big - 64, big, big}
	dr := Rect{sx, logicalH - 20 - small - 64, small, small}
	holdRects["attack"], holdRects["defend"] = ar, dr
	a, d := held("attack"), held("defend")
	for _, b := range []struct {
		r   Rect
		on  bool
		ico *ebiten.Image
	}{{ar, a, icoSword}, {dr, d, icoShield}} {
		b := b
		ops = append(ops, func(dst *ebiten.Image) {
			body, hi, lo := color.Color(color.NRGBA{46, 40, 92, 170}), color.Color(color.NRGBA{80, 70, 150, 170}), color.Color(color.NRGBA{24, 20, 52, 170})
			rr := b.r
			if b.on {
				body, hi, lo = colB, colS, colL
				rr.Y += 3
			}
			bevel(dst, rr, body, hi, lo, !b.on)
			isz := rr.W * 0.45
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
			op.GeoM.Translate((rr.X+(rr.W-isz)/2)*uiScale, (rr.Y+(rr.H-isz)/2)*uiScale)
			dst.DrawImage(b.ico, op)
		})
	}
}

// ---- Tools ----

var (
	icoPick  = pixIcon(".####...", "#....#..", "...#..#.", "..#....#", ".#......", "#.......", "........", "........")
	toolIco  = []*ebiten.Image{icoSword, icoPick}
	toolName = []string{"Sword", "Pickaxe"}
)

// The two tools along the bottom: tap one to hold it
func toolBar() {
	slot := 40.0
	if isTouch {
		slot = 44
	}
	gap := 8.0
	x0, y0 := (logicalW-2*slot-gap)/2, logicalH-slot-12
	bar := Rect{x0 - 6, y0 - 6, 2*slot + gap + 12, slot + 12}
	hudRects = append(hudRects, bar)
	ops = append(ops, func(dst *ebiten.Image) { fill(dst, bar.X, bar.Y, bar.W, bar.H, color.NRGBA{19, 16, 39, 190}) })
	for t := 0; t < 2; t++ {
		r := Rect{x0 + float64(t)*(slot+gap), y0, slot, slot}
		sel := myTool == t
		if ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY) {
			selectTool(t)
		}
		ico, key := toolIco[t], fmt.Sprint(t+1)
		ops = append(ops, func(dst *ebiten.Image) {
			hi, lo, body := color.Color(color.NRGBA{10, 8, 23, 255}), color.Color(colPanelHi), color.Color(colPanelLo)
			if sel {
				hi, lo, body = colT, colGoldLo, rgb(0x2c2550)
			}
			bevel(dst, r, body, hi, lo, false)
			isz := r.W * 0.55
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
			op.GeoM.Translate((r.X+(r.W-isz)/2)*uiScale, (r.Y+(r.H-isz)/2)*uiScale)
			dst.DrawImage(ico, op)
			if !isTouch {
				drawText(dst, key, r.X+6, r.Y+9, textOpt{size: 9, display: true, col: colMuted})
			}
		})
	}
	if mineX >= 0 && mineProg > 0 { // digging progress over the bar
		p := mineProg
		ops = append(ops, func(dst *ebiten.Image) {
			fill(dst, bar.X, bar.Y-8, bar.W, 4, colO)
			fill(dst, bar.X, bar.Y-8, bar.W*p, 4, colT)
		})
	}
}

// Phones: hold the big button to use the tool in hand; the small one switches tools
func useButtons() {
	big, small := 78.0, 54.0
	x := logicalW - 20 - big
	sx := x - small - 16
	if prefs.BtnSide == "left" {
		x, sx = 20, 20+big+16
	}
	ur := Rect{x, logicalH - 20 - big - 64, big, big}
	sr := Rect{sx, logicalH - 20 - small - 64, small, small}
	holdRects["use"] = ur
	hudRects = append(hudRects, sr)
	on := held("use")
	ico := toolIco[myTool]
	ops = append(ops, func(dst *ebiten.Image) {
		body, hi, lo := color.Color(color.NRGBA{46, 40, 92, 170}), color.Color(color.NRGBA{80, 70, 150, 170}), color.Color(color.NRGBA{24, 20, 52, 170})
		rr := ur
		if on {
			body, hi, lo = colB, colS, colL
			rr.Y += 3
		}
		bevel(dst, rr, body, hi, lo, !on)
		isz := rr.W * 0.45
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
		op.GeoM.Translate((rr.X+(rr.W-isz)/2)*uiScale, (rr.Y+(rr.H-isz)/2)*uiScale)
		dst.DrawImage(ico, op)
	})
	if Button(sr, "", styleSlate, true, toolIco[1-myTool]) {
		selectTool(1 - myTool)
	}
}
