package game

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Screens and menus, all drawn with the pixel widgets in ui.go. The game's own screens
// (main menu, petals, travel...) are in screens.go; the HUD is in hud.go.

type screen int

const (
	scMain screen = iota
	scPlaying
	scPause
	scPetals
	scTravel
	scSettings
	scDead
	scAway
	scName
)

var (
	cur            = scMain
	settingsFrom   = scPause
	settingsTab    int
	started        bool
	toastText      string
	toastUntil     int
	hintUntil      int
	listening      string // action waiting for a new key
	screenOpenedAt int
)

// The world never pauses: it's shared with everyone else
func paused() bool { return false }

func openScreen(s screen) {
	if s == scPetals || s == scTravel {
		pickSlot, pickBag = -1, ""
		sfx("open")
	}
	if s == scPause && cur == scPlaying {
		sfx("close")
	}
	cur = s
	screenOpenedAt = frame
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
		drawText(dst, "REMOTE", bx+8, topY+art+ts*0.55+8, textOpt{size: ts, display: true, col: colO, align: text.AlignCenter})
		drawText(dst, "REMOTE", bx+4, topY+art+ts*0.55+4, textOpt{size: ts, display: true, col: colS, align: text.AlignCenter})
		drawText(dst, "REMOTE", bx, topY+art+ts*0.55, textOpt{size: ts, display: true, align: text.AlignCenter})
		drawText(dst, "A PIXEL MMO", bx, topY+art+ts*1.2+10, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
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
	tabs := []string{"Display", "Sound", "Controls", "Server"}
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
		Text("1 is closest, 5 shows the most. 3 is normal.", x, y+58, 12, colMuted, text.AlignStart)
		y += 80
		Label("INTERFACE SIZE", x, y)
		if c := seg(x, y+14, []string{"Small", "Normal", "Large", "Larger"}, prefs.UISize); c >= 0 {
			prefs.UISize = c
			savePrefs()
		}
		Label("GRAPHICS", x+380, y)
		gq := 0 // Auto: lighter when the game runs slowly
		if prefs.Fancy {
			gq = 1
		} else if prefs.Simple {
			gq = 2
		}
		if c := seg(x+380, y+14, []string{"Auto", "Fancy", "Fast"}, gq); c >= 0 {
			prefs.Fancy, prefs.Simple = c == 1, c == 2
			savePrefs()
		}
		y += 70
		Label("TIPS", x, y)
		if c := seg(x, y+14, []string{"On", "Off"}, b2i(!prefs.Tips)); c >= 0 {
			prefs.Tips = c == 0
			savePrefs()
		}
		if !isTouch {
			Label("FULL SCREEN", x+220, y)
			if c := seg(x+220, y+14, []string{"On", "Off"}, b2i(!prefs.Full)); c >= 0 {
				prefs.Full = c == 0
				ebiten.SetFullscreen(prefs.Full)
				savePrefs()
			}
			Text("F11 also switches full screen.", x+360, y+30, 12, colMuted, text.AlignStart)
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
			Label("ATTACK BUTTONS SIDE", x, y)
			bs := 1
			if prefs.BtnSide == "left" {
				bs = 0
			}
			if c := seg(x, y+14, []string{"Left", "Right"}, bs); c >= 0 {
				prefs.BtnSide = []string{"left", "right"}[c]
				savePrefs()
			}
		} else {
			acts := [][2]string{{"left", "Move left"}, {"right", "Move right"}, {"lev", "Levitate"}, {"petals", "Petals"}, {"map", "Travel map"}}
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
	case 3:
		serverSettings(x, y, r)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func drawToast() {
	if frame >= toastUntil {
		return
	}
	w := textWidth(toastText, 11, true) + 28
	r := Rect{(logicalW - w) / 2, 52, w, 30} // under the island's name
	if cur != scPlaying {                    // menus have their titles at the top: show it at the bottom instead
		if toastUntil-150 < screenOpenedAt { // only messages from this menu
			return
		}
		r.Y = logicalH - 40
	}
	Panel(r)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, toastText, r.X+r.W/2, r.Y+r.H/2, textOpt{size: 11, display: true, col: colT, align: text.AlignCenter, shadow: true})
	})
}
