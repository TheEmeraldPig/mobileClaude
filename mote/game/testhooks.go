package game

import (
	"fmt"
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
		case "time":
			worldTime, _ = strconv.ParseFloat(arg, 64)
		case "deep": // drop the mote into a cave far below
			teleportDeep()
		case "spawn":
			for k := 0; k < 6; k++ {
				ents = append(ents, &Entity{kind: entGloom, x: player.x + float64(30+k*14), y: player.y - 10, w: 8 * CELL, h: 6 * CELL, hp: 4, face: -1})
			}
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
			addItem(arg, 200)
		case "aim": // aim=sx:sy, start using the selected item at that screen point
			a, b, _ := strings.Cut(arg, ":")
			aimSX, _ = strconv.ParseFloat(a, 64)
			aimSY, _ = strconv.ParseFloat(b, 64)
			useActive, scriptAim = true, true
		case "unaim":
			useActive, scriptAim = false, false
		case "dropone":
			dropSlot(selected, 1)
		case "log":
			n := 0
			for _, e := range ents {
				if e.kind == entItem {
					n++
				}
			}
			fmt.Printf("frame %d inv=%v drops=%d hp=%d player=(%.0f,%.0f) hits=%v\n", frame, invSummary(), n, player.hp, player.x, player.y, hits(player.x, player.y))
		case "bury": // pour sand on the mote
			cx, cy := int(player.x/CELL), int(player.y/CELL)
			for y := cy - 2; y < cy+8; y++ {
				for x := cx - 2; x < cx+12; x++ {
					grid[y*W+x] = SAND
				}
			}
		case "shot":
			shotPath = arg
		case "quit":
			os.Exit(0)
		}
	}
}

var scriptRight, scriptLev, scriptAim bool

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

// Move the mote into the first open cave space well below the surface
func teleportDeep() {
	cx := int((player.x + player.w/2) / CELL)
	for dx := 0; dx < W/2; dx += 3 {
		for _, x := range []int{cx + dx, cx - dx} {
			if x < 8 || x >= W-8 {
				continue
			}
			for y := surfaceAt(baseChunk*CW+x) + 60; y < H-20; y++ {
				if boxFree(x-6, y-8, 12, 16) && y+9 < H && isSolid(grid[(y+9)*W+x]) {
					player.x, player.y = float64((x-5)*CELL), float64((y-6)*CELL)
					player.vx, player.vy = 0, 0
					updatePlayerCells()
					return
				}
			}
		}
	}
}

func invSummary() string {
	var b strings.Builder
	for _, st := range inv {
		if st != nil {
			fmt.Fprintf(&b, "%s:%d ", st.ID, st.N)
		}
	}
	return b.String()
}
