package game

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// If the game panics, the error is shown on screen (and saved to crash.txt) instead
// of leaving a black screen.
var crashMsg string

// CatchCrash is deferred around the game's Update and Draw.
func CatchCrash() {
	if r := recover(); r != nil {
		if crashMsg != "" {
			return
		}
		stack := string(debug.Stack())
		crashMsg = fmt.Sprint(r)
		log.Printf("Blob Split crashed: %s\n%s", crashMsg, stack)
		_ = os.MkdirAll(dataDir(), 0o755)
		_ = os.WriteFile(filepath.Join(dataDir(), "crash.txt"), []byte(Version+"\n"+crashMsg+"\n\n"+stack), 0o644)
		var keep []string
		for _, l := range strings.Split(stack, "\n") {
			if strings.Contains(l, "blobsplit/game") {
				keep = append(keep, strings.TrimSpace(l))
			}
		}
		if len(keep) > 6 {
			keep = keep[:6]
		}
		crashMsg += "\n\n" + strings.Join(keep, "\n")
	}
}

func Crashed() bool { return crashMsg != "" }

// DrawCrash shows the error in big pixel text.
func DrawCrash(screen *ebiten.Image) {
	screen.Fill(color.RGBA{34, 28, 64, 255})
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w < 1 || h < 1 {
		return
	}
	scale := max(1, min(w/480, h/270))
	img := ebiten.NewImage(max(1, w/scale), max(1, h/scale))
	n := max(20, w/scale/6-2)
	var lines []string
	for _, l := range strings.Split(crashMsg, "\n") {
		for len(l) > n {
			lines = append(lines, l[:n])
			l = l[n:]
		}
		lines = append(lines, l)
	}
	ebitenutil.DebugPrintAt(img, "Blob Split ran into a problem (build "+Version+").\nPlease send a screenshot of this.\n\n"+strings.Join(lines, "\n"), 8, 8)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(img, op)
	img.Deallocate()
}
