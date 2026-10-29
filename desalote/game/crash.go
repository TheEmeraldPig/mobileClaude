package game

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// If something goes wrong, show the error on screen (instead of a black screen)
// and save it to crash.txt next to the save game.

var crashMsg string

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
	crashMsg = msg
	log.Printf("Desalote crashed: %s\n%s", msg, stack)
	os.WriteFile(dataPath("crash.txt"), []byte(Version+"\n"+msg+"\n\n"+stack), 0o644)
	// keep the most useful part of the stack for the screen
	lines := strings.Split(stack, "\n")
	var keep []string
	for _, l := range lines {
		if strings.Contains(l, "/desalote/") {
			keep = append(keep, strings.TrimSpace(l))
		}
		if len(keep) >= 8 {
			break
		}
	}
	crashMsg += "\n\n" + strings.Join(keep, "\n")
}

func Crashed() bool { return crashMsg != "" }

// DrawCrash shows the error in big pixel text.
func DrawCrash(screen *ebiten.Image) {
	screen.Fill(color.RGBA{34, 28, 64, 255})
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	scale := max(1, min(w/480, h/270))
	img := ebiten.NewImage(w/scale, h/scale)
	text := "Desalote ran into a problem (build " + Version + ").\nPlease send a screenshot of this.\n\n" + wrap(crashMsg, w/scale/6-2)
	ebitenutil.DebugPrintAt(img, text, 8, 8)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(img, op)
	img.Deallocate()
}

func wrap(s string, n int) string {
	if n < 10 {
		n = 10
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for len(line) > n {
			out = append(out, line[:n])
			line = line[n:]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
