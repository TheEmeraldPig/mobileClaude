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

// DIG_SCRIPT runs commands for testing without a person, separated by ';':
//
//	wait N        wait N frames
//	tap X Y       tap at logical position X,Y
//	tab N         switch to tab N (0 upgrades, 1 gems, 2 store)
//	coins N       add N coins
//	gems N        add N gems
//	shot FILE     save a screenshot as PNG
//	quit          exit
type script struct {
	cmds []string
	wait int
	shot string
}

func loadScript() *script {
	v := os.Getenv("DIG_SCRIPT")
	if v == "" {
		return nil
	}
	return &script{cmds: strings.Split(v, ";")}
}

func (sc *script) step(g *Game) error {
	for sc.wait == 0 && sc.shot == "" && len(sc.cmds) > 0 {
		f := strings.Fields(sc.cmds[0])
		sc.cmds = sc.cmds[1:]
		if len(f) == 0 {
			continue
		}
		num := func(i int) float64 {
			if i < len(f) {
				v, _ := strconv.ParseFloat(f[i], 64)
				return v
			}
			return 0
		}
		switch f[0] {
		case "wait":
			sc.wait = int(num(1))
		case "tap":
			g.click(num(1), num(2))
		case "tab":
			g.tab = int(num(1))
		case "coins":
			g.s.earn(num(1))
		case "gems":
			g.s.Gems += int(num(1))
		case "shot":
			sc.shot = f[1]
		case "quit":
			g.Save()
			return ebiten.Termination
		default:
			return fmt.Errorf("DIG_SCRIPT: unknown command %q", f[0])
		}
	}
	if sc.wait > 0 {
		sc.wait--
	}
	return nil
}

func (sc *script) afterDraw(dst *ebiten.Image) {
	if sc.shot == "" {
		return
	}
	b := dst.Bounds()
	img := image.NewRGBA(b)
	dst.ReadPixels(img.Pix)
	if f, err := os.Create(sc.shot); err == nil {
		png.Encode(f, img)
		f.Close()
	}
	sc.shot = ""
}
