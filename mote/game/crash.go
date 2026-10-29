package game

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// If something goes wrong, the game shows a crash screen instead of a black screen,
// and saves the details to crash.txt next to the save game.
//
// The crash screen lists everything (the error, where it happened, and the full stack),
// scrolls (mouse wheel, arrow keys, Page Up/Down, Home/End, or drag on a touch screen),
// and has a bar along the bottom: Ctrl+C copies all of it, Esc closes the game. On
// phones the bar has a Close button instead.
//
// It's drawn with Ebitengine's built-in debug font, so it works even if the game's own
// fonts and UI are what broke.

const crashGame = "Mote"

var (
	crashMsg    string // the error (non-empty once crashed)
	crashReport string // everything, as copied with Ctrl+C
	crashLines  []string
	crashScroll float64 // in lines
	crashMaxTop int     // the furthest the text can scroll (worked out when drawn)
	crashCopied int     // frames left to show "Copied"
	crashDragY  float64
	crashDragID ebiten.TouchID = -1
	crashImg    *ebiten.Image
	crashQuit   bool
)

// CatchCrash must be deferred; it turns a panic into the crash screen.
func CatchCrash() {
	if r := recover(); r != nil {
		ReportCrash(fmt.Sprint(r), string(debug.Stack()))
	}
}

func ReportCrash(msg, stack string) {
	if crashMsg != "" {
		return
	}
	if msg == "" {
		msg = "unknown error"
	}
	crashMsg = msg
	log.Printf("%s crashed: %s\n%s", crashGame, msg, stack)
	// the lines from the game's own code are the useful ones: list them first
	// (a stack lists each call as a function line, then a file:line line)
	var where []string
	sl := strings.Split(stack, "\n")
	for i, l := range sl {
		l = strings.TrimSpace(l)
		if i == 0 || !strings.Contains(l, "/mote/") || !strings.Contains(l, ".go:") || strings.Contains(l, "/crash.go:") {
			continue
		}
		file := l[strings.LastIndex(l, "/mote/")+len("/mote/"):]
		if sp := strings.Index(file, " +0x"); sp > 0 {
			file = file[:sp]
		}
		fn := strings.TrimSpace(sl[i-1])
		if p := strings.LastIndex(fn, "/"); p >= 0 {
			fn = fn[p+1:]
		}
		if p := strings.LastIndex(fn, "("); p > 0 && strings.HasSuffix(fn, ")") {
			fn = fn[:p] // drop the argument values
		}
		where = append(where, fn+"  ("+file+")")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s crashed (build %s, %s/%s)\n\n", crashGame, Version, runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "Error: %s\n", msg)
	if len(where) > 0 {
		b.WriteString("\nWhere (the game's own code, innermost first):\n")
		for _, l := range where {
			b.WriteString("  " + l + "\n")
		}
	}
	b.WriteString("\nFull details:\n")
	b.WriteString(strings.TrimRight(stack, "\n"))
	b.WriteString("\n")
	crashReport = b.String()
	saveCrashFile(crashReport)
}

func Crashed() bool { return crashMsg != "" }

// CrashUpdate handles the crash screen's keys and scrolling; it reports true when the
// player wants to close the game. Call it from Update while Crashed().
func CrashUpdate() bool {
	if crashCopied > 0 {
		crashCopied--
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if crashClipboard(crashReport) {
			crashCopied = 150
		} else {
			crashCopied = -150 // couldn't
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || crashQuit {
		return true
	}
	// scrolling
	_, wy := ebiten.Wheel()
	crashScroll -= wy * 3
	step := func(k ebiten.Key) bool {
		d := inpututil.KeyPressDuration(k)
		return d == 1 || (d > 20 && d%3 == 0)
	}
	if step(ebiten.KeyDown) {
		crashScroll++
	}
	if step(ebiten.KeyUp) {
		crashScroll--
	}
	if step(ebiten.KeyPageDown) || step(ebiten.KeySpace) {
		crashScroll += 15
	}
	if step(ebiten.KeyPageUp) {
		crashScroll -= 15
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		crashScroll = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		crashScroll = float64(crashMaxTop)
	}
	// touch: drag to scroll, tap Close in the bottom bar
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		_, y := ebiten.TouchPosition(id)
		crashDragID, crashDragY = id, float64(y)
	}
	if crashDragID >= 0 {
		if inpututil.IsTouchJustReleased(crashDragID) {
			x, y := inpututil.TouchPositionInPreviousTick(crashDragID)
			if crashOnClose(x, y) {
				crashQuit = true
			}
			crashDragID = -1
		} else {
			_, y := ebiten.TouchPosition(crashDragID)
			crashScroll -= (float64(y) - crashDragY) / crashLinePx()
			crashDragY = float64(y)
		}
	}
	// a click on Close works with a mouse too
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if x, y := ebiten.CursorPosition(); crashOnClose(x, y) {
			crashQuit = true
		}
	}
	crashScroll = max(0, min(crashScroll, float64(crashMaxTop)))
	return crashQuit
}

// Screen pixels per line of crash text (and where the Close button was last drawn)
var (
	crashScale   = 1
	crashCloseAt [4]int // x, y, w, h in screen pixels
)

func crashLinePx() float64 { return float64(16 * crashScale) }

func crashOnClose(x, y int) bool {
	c := crashCloseAt
	return c[2] > 0 && x >= c[0] && x < c[0]+c[2] && y >= c[1] && y < c[1]+c[3]
}

func crashTouch() bool { return runtime.GOOS == "android" || runtime.GOOS == "ios" }

// DrawCrash shows the crash screen.
func DrawCrash(screen *ebiten.Image) {
	screen.Fill(color.RGBA{34, 28, 64, 255})
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w < 1 || h < 1 {
		return
	}
	scale := max(1, min(w/480, h/270))
	crashScale = scale
	lw, lh := max(1, w/scale), max(1, h/scale)
	if crashImg == nil || crashImg.Bounds().Dx() != lw || crashImg.Bounds().Dy() != lh {
		if crashImg != nil {
			crashImg.Deallocate()
		}
		crashImg = ebiten.NewImage(lw, lh)
	}
	img := crashImg
	img.Clear()
	// the text, wrapped to the screen
	cols := max(20, lw/6-3)
	crashLines = crashLines[:0]
	head := crashGame + " ran into a problem (build " + Version + ")."
	body := crashReport // the report without its first line (the title replaces it)
	if i := strings.Index(body, "\n"); i >= 0 {
		body = body[i+1:]
	}
	for _, l := range strings.Split(head+"\n"+body, "\n") {
		l = strings.ReplaceAll(l, "\t", "    ")
		for len(l) > cols {
			crashLines = append(crashLines, l[:cols])
			l = "  " + l[cols:]
		}
		crashLines = append(crashLines, l)
	}
	const lineH, top = 16, 6
	barH := 22
	rows := max(1, (lh-top-barH-4)/lineH)
	crashMaxTop = max(0, len(crashLines)-rows)
	first := int(max(0, min(crashScroll, float64(crashMaxTop))))
	for i := 0; i < rows && first+i < len(crashLines); i++ {
		ebitenutil.DebugPrintAt(img, crashLines[first+i], 8, top+i*lineH)
	}
	// a scroll bar when it doesn't all fit
	if crashMaxTop > 0 {
		trackH := float32(lh - top - barH - 8)
		thumb := max(8, trackH*float32(rows)/float32(len(crashLines)))
		y := float32(top) + (trackH-thumb)*float32(first)/float32(crashMaxTop)
		crashFill(img, float32(lw-5), float32(top), 3, trackH, color.RGBA{60, 52, 100, 255})
		crashFill(img, float32(lw-5), y, 3, thumb, color.RGBA{196, 180, 255, 255})
	}
	// the bar along the bottom
	by := lh - barH
	crashFill(img, 0, float32(by), float32(lw), float32(barH), color.RGBA{18, 14, 38, 255})
	crashFill(img, 0, float32(by), float32(lw), 1, color.RGBA{124, 98, 255, 255})
	var hint string
	switch {
	case crashCopied > 0:
		hint = "Copied! Paste it to whoever is fixing the game."
	case crashCopied < 0:
		hint = "Couldn't copy here. It's saved in crash.txt."
	case crashTouch():
		hint = "Drag to scroll. Take a screenshot to share it."
	default:
		hint = "Ctrl+C: copy all crash info    Esc: close the game    Wheel/arrows: scroll"
	}
	ebitenutil.DebugPrintAt(img, hint, 8, by+3)
	// a Close button on the right (for touch, and mouse users)
	cw := 6*5 + 12
	cx := lw - cw - 6
	if crashMaxTop > 0 {
		cx -= 4
	}
	crashFill(img, float32(cx), float32(by+3), float32(cw), float32(barH-6), color.RGBA{124, 98, 255, 255})
	ebitenutil.DebugPrintAt(img, "Close", cx+6, by+2)
	crashCloseAt = [4]int{cx * scale, (by + 1) * scale, cw * scale, (barH - 2) * scale}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(img, op)
}

var crashPx *ebiten.Image

func crashFill(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	if crashPx == nil {
		crashPx = ebiten.NewImage(1, 1)
		crashPx.Fill(color.White)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(crashPx, op)
}

func saveCrashFile(report string) {
	os.WriteFile(dataPath("crash.txt"), []byte(report), 0o644)
}
