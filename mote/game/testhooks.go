package game

import (
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Test hooks for automated checks (unused in normal play):
//   MOTE_SCRIPT="30:play,90:inv,150:shot=/tmp/a.png,160:quit"
// runs actions at given frames: play, pause, inv, settings, debug, main, seed, right, lev, stop, select=N, shot=PATH, quit.

type scriptStep struct {
	frame int
	act   string
}

var (
	script   []scriptStep
	shotPath string
)

func init() {
	if os.Getenv("MOTE_TOUCH") == "1" {
		isTouch = true
	}
	for _, part := range strings.Split(os.Getenv("MOTE_SCRIPT"), ",") {
		f, a, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		n, _ := strconv.Atoi(f)
		script = append(script, scriptStep{n, a})
	}
}

func runScript() {
	for _, s := range script {
		if s.frame != frame {
			continue
		}
		act, arg, _ := strings.Cut(s.act, "=")
		switch act {
		case "play":
			play()
		case "pause":
			openScreen(scPause)
		case "inv":
			openScreen(scInv)
		case "settings":
			openScreen(scSettings)
		case "debug":
			openScreen(scDebug)
		case "main":
			openScreen(scMain)
		case "seed":
			openScreen(scSeed)
		case "right":
			scriptRight = true
		case "lev":
			scriptLev = true
		case "stop":
			scriptRight, scriptLev = false, false
		case "select":
			selected, _ = strconv.Atoi(arg)
		case "give":
			addItem(arg, items[arg].give)
		case "shot":
			shotPath = arg
		case "quit":
			os.Exit(0)
		}
	}
}

var scriptRight, scriptLev bool

func takeShot(screen *ebiten.Image) {
	if shotPath == "" {
		return
	}
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	if f, err := os.Create(shotPath); err == nil {
		png.Encode(f, img)
		f.Close()
	}
	shotPath = ""
}

// WindowSizeOverride lets tests pick a window size with MOTE_SIZE=800x370.
func WindowSizeOverride() (int, int) {
	w, h, ok := strings.Cut(os.Getenv("MOTE_SIZE"), "x")
	if !ok {
		return 0, 0
	}
	a, _ := strconv.Atoi(w)
	b, _ := strconv.Atoi(h)
	return a, b
}
