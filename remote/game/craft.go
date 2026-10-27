package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The crafting screen, in the spirit of console crafting menus: category tabs along
// the top, a row of big recipe slots, and the selected recipe's details with a large
// Craft button. Works with touch, mouse, or just the keyboard:
// Left/Right (A/D) pick a recipe, Up/Down (W/S) or Tab change the tab, Enter/Space crafts.
//
// craftStation is "" for crafting by hand (from the inventory), or a station id when a
// workbench, furnace or anvil was opened.

var (
	craftStation string
	craftTab     int
	craftSel     int
	craftScroll  float64 // smoothed scroll of the recipe row, in slots
	craftMsg     string
	craftMsgAt   int
	craftFlash   int // frame of the last craft, for a little pop
)

func openStation(kind string) {
	if !prefs.Used[kind] {
		if prefs.Used == nil {
			prefs.Used = map[string]bool{}
		}
		prefs.Used[kind] = true
		savePrefs()
	}
	if kind == "bed" {
		useBed()
		return
	}
	if kind == "storage" {
		openStorage()
		return
	}
	craftStation = kind
	craftMsg = ""
	fixCraftTab()
	openScreen(scCraft)
	sfx("open")
}

var craftOpenedAt int

func openHandCrafting() {
	craftOpenedAt = frame
	craftStation = ""
	craftMsg = ""
	fixCraftTab()
	openScreen(scCraft)
}

func craftTitle() string {
	if craftStation == "" {
		return "CRAFTING"
	}
	return stationTitle(craftStation)
}

func stationTitle(k string) string {
	switch k {
	case "workbench":
		return "WORKBENCH"
	case "furnace":
		return "FURNACE"
	case "anvil":
		return "ANVIL"
	}
	return "CRAFTING"
}

// Recipes made here, by tab
func craftLists() [][]Recipe {
	lists := make([][]Recipe, len(craftCats))
	for _, r := range recipes {
		if r.at == craftStation {
			c := recipeCat(r)
			lists[c] = append(lists[c], r)
		}
	}
	return lists
}

// Tabs that have recipes here
func craftTabs() []int {
	var t []int
	for c, l := range craftLists() {
		if len(l) > 0 {
			t = append(t, c)
		}
	}
	return t
}

// Open on the tab of the station's first recipe (smelting for the furnace, and so on)
func fixCraftTab() {
	for _, r := range recipes {
		if r.at == craftStation {
			craftTab = recipeCat(r)
			break
		}
	}
	craftSel = 0
	craftScroll = 0
}

func stepCraftTab(d int) {
	tabs := craftTabs()
	for i, t := range tabs {
		if t == craftTab {
			craftTab = tabs[(i+d+len(tabs))%len(tabs)]
			break
		}
	}
	craftSel, craftScroll = 0, 0
	sfx("click")
}

func doCraft(r Recipe, times int) {
	made := 0
	msg := ""
	for k := 0; k < times; k++ {
		m := craft(r)
		if m == "" || m == "No room in your inventory." {
			if m != "" {
				msg = m
			}
			break
		}
		made++
	}
	switch {
	case made > 0:
		craftMsg = fmt.Sprintf("Crafted %d %s", made*r.n, items[r.out].name)
		craftFlash = frame
	case msg != "":
		craftMsg = msg
	default:
		craftMsg = "You need more materials"
		sfx("click")
	}
	craftMsgAt = frame
}

func craftMenu() {
	w := math.Min(880, logicalW-28)
	h := math.Min(420, logicalH-24)
	r := sheet(w, h)
	if _, cl := head(r, craftTitle(), false); cl || (craftStation == "" && bindJustPressed("craft") && frame > craftOpenedAt+1) {
		play()
		return
	}
	lists := craftLists()
	tabs := craftTabs()
	if len(tabs) == 0 {
		Text("Nothing to craft here yet.", r.X+18, r.Y+80, 15, colMuted, text.AlignStart)
		return
	}
	list := lists[craftTab]
	compact := r.H < 400

	// keyboard
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		d := 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			d = -1
		}
		stepCraftTab(d)
		list = lists[craftTab]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		stepCraftTab(1)
		list = lists[craftTab]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		stepCraftTab(-1)
		list = lists[craftTab]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		craftSel = min(len(list)-1, craftSel+1)
		sfx("click")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		craftSel = max(0, craftSel-1)
		sfx("click")
	}
	craftSel = max(0, min(len(list)-1, craftSel))
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		n := 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			n = 10
		}
		doCraft(list[craftSel], n)
	}

	// tabs
	tx, ty := r.X+18, r.Y+54
	tabH := 32.0
	if compact {
		tabH = 30
	}
	for _, c := range tabs {
		label := craftCats[c]
		tw := textWidth(label, 14, false) + 26
		st := styleSlate
		if c == craftTab {
			st = styleViolet
		}
		if Button(Rect{tx, ty, tw, tabH}, label, st, true, nil) && c != craftTab {
			craftTab, craftSel, craftScroll = c, 0, 0
			list = lists[craftTab]
		}
		tx += tw + 8
	}
	if !isTouch && !compact {
		Text("Tab / Up / Down: category    Left / Right: recipe    Enter: craft (Shift: x10)", r.X+r.W-18, ty+tabH/2, 11, colMuted, text.AlignEnd)
	}

	// the recipe row
	slot := 64.0
	if compact {
		slot = math.Min(56, (r.H-150)/2.1)
	}
	gap := 10.0
	rowY := ty + tabH + 18
	arrowW := 30.0
	rowX0 := r.X + 18 + arrowW + 8
	rowW := r.W - 36 - 2*(arrowW+8)
	perView := max(1, int((rowW+gap)/(slot+gap)))
	// keep the selection in view (smoothly)
	target := craftScroll
	if float64(craftSel) < target {
		target = float64(craftSel)
	}
	if float64(craftSel) > target+float64(perView)-1 {
		target = float64(craftSel - perView + 1)
	}
	target = math.Max(0, math.Min(target, float64(max(0, len(list)-perView))))
	craftScroll += (target - craftScroll) * 0.35
	if math.Abs(target-craftScroll) < 0.01 {
		craftScroll = target
	}
	if Button(Rect{r.X + 18, rowY + (slot-40)/2, arrowW, 40}, "<", styleSlate, craftSel > 0, nil) {
		craftSel--
	}
	if Button(Rect{r.X + r.W - 18 - arrowW, rowY + (slot-40)/2, arrowW, 40}, ">", styleSlate, craftSel < len(list)-1, nil) {
		craftSel++
	}
	rowRect := Rect{rowX0 - 4, rowY - 6, rowW + 8, slot + 12}
	Inset(rowRect, colPanelLo)
	if _, wy := ebiten.Wheel(); wy != 0 && rowRect.Has(hoverX, hoverY) {
		craftSel = max(0, min(len(list)-1, craftSel-sign(wy)))
	}
	for i, rc := range list {
		x := rowX0 + (float64(i)-craftScroll)*(slot+gap)
		if x < rowX0-slot*0.5 || x > rowX0+rowW-slot*0.5 {
			continue
		}
		sr := Rect{x, rowY, slot, slot}
		sel := i == craftSel
		can := canCraft(rc)
		it := items[rc.out]
		clicked := ptr.released && sr.Has(ptr.x, ptr.y) && sr.Has(ptr.startX, ptr.startY)
		if clicked {
			if craftSel == i && isTouch {
				doCraft(rc, 1) // tapping the selected recipe again crafts it
			}
			craftSel = i
			sfx("click")
		}
		pop := 0.0
		if sel && frame-craftFlash < 10 {
			pop = float64(10-(frame-craftFlash)) * 0.6
		}
		n := rc.n
		ops = append(ops, func(dst *ebiten.Image) {
			body := color.Color(colSlateLo)
			hi, lo := color.Color(colPanelHi), color.Color(color.NRGBA{10, 8, 23, 255})
			if sel {
				body, hi, lo = rgb(0x2c2550), colT, colT
			}
			rr := Rect{sr.X - pop, sr.Y - pop, sr.W + 2*pop, sr.H + 2*pop}
			bevel(dst, rr, body, hi, lo, sel)
			isz := rr.W * 0.6
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
			op.GeoM.Translate((rr.X+(rr.W-isz)/2)*uiScale, (rr.Y+(rr.H-isz)/2)*uiScale)
			if !can {
				op.ColorScale.Scale(0.45, 0.45, 0.55, 1)
			}
			dst.DrawImage(it.img, op)
			if n > 1 {
				drawText(dst, fmt.Sprint(n), rr.X+rr.W-5, rr.Y+rr.H-9, textOpt{size: 12, align: text.AlignEnd, shadow: true})
			}
			if can {
				fill(dst, rr.X+5, rr.Y+5, 6, 6, color.NRGBA{120, 220, 140, 255})
			}
		})
	}

	// details of the selected recipe
	rc := list[craftSel]
	it := items[rc.out]
	dy := rowY + slot + 22
	det := Rect{r.X + 18, dy, r.W - 36, r.Y + r.H - 16 - dy}
	Inset(det, colSlateLo)
	big := math.Min(det.H-24, 72)
	if compact {
		big = math.Min(det.H-20, 52)
	}
	IconAt(it.img, det.X+12, det.Y+12, big)
	nx := det.X + 24 + big
	name := it.name
	if rc.n > 1 {
		name = fmt.Sprintf("%s  x%d", it.name, rc.n)
	}
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, name, nx, det.Y+20, textOpt{size: 18, shadow: true})
	})
	btnW := math.Min(190, det.W*0.28)
	descW := det.W - (nx - det.X) - btnW - 30
	yy := wrapText(it.desc+".", nx, det.Y+44, descW, 12, colMuted, text.AlignStart)
	if have := countItem(rc.out); have > 0 {
		Text(fmt.Sprintf("You have %d", have), nx, yy+2, 12, colMuted, text.AlignStart)
		yy += 18
	}
	// ingredients
	iy := math.Max(yy+6, det.Y+det.H*0.48)
	if compact {
		iy = math.Max(yy+2, det.Y+det.H*0.5)
	}
	Label("NEEDS", nx, iy)
	ix := nx
	iy += 16
	isz := 22.0
	for _, nd := range rc.needs {
		id, need := nd[0].(string), nd[1].(int)
		have := countItem(id)
		col := colInk
		if have < need {
			col = colDanger
		}
		label := fmt.Sprintf("%s %d/%d", items[id].name, min(have, need), need)
		lw := isz + 8 + textWidth(label, 13, false) + 18
		if ix+lw > det.X+det.W-btnW-24 && ix > nx {
			ix = nx
			iy += isz + 8
		}
		chip := Rect{ix, iy, lw - 8, isz + 6}
		Inset(chip, colPanelLo)
		IconAt(items[id].img, ix+4, iy+3, isz)
		Text(label, ix+isz+10, iy+(isz+6)/2, 13, col, text.AlignStart)
		ix += lw
	}

	// craft buttons
	can := canCraft(rc)
	bx := det.X + det.W - btnW - 14
	bh := math.Min(52, det.H*0.4)
	if Button(Rect{bx, det.Y + 14, btnW, bh}, "Craft", styleGold, can, nil) {
		doCraft(rc, 1)
	}
	times := craftableTimes(rc)
	allLabel := "Craft all"
	if times > 1 {
		allLabel = fmt.Sprintf("Craft all (%d)", min(times, 99))
	}
	if !items[rc.out].tool && Button(Rect{bx, det.Y + 24 + bh, btnW, math.Min(36, bh*0.75)}, allLabel, styleViolet, times > 1, nil) {
		doCraft(rc, min(times, 99))
	}
	if craftStation == "" {
		Text("More recipes at the workbench, furnace and anvil.", det.X+14, det.Y+det.H-14, 11, colMuted, text.AlignStart)
	}
	if craftMsg != "" && frame-craftMsgAt < 180 {
		Text(craftMsg, det.X+det.W-14, det.Y+det.H-14, 12, colT, text.AlignEnd)
	}
}
