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

// BUGS_SCRIPT runs commands for testing without a person, separated by ';':
//
//	wait N        wait N frames
//	tap X Y       tap at layout position X,Y (a press and release)
//	press ID      press a button by id: tab:N, up:N (-1 = tap power), mult, skill:N,
//	              boss, team:N, buy:N, free, collect, collect2, modal
//	coins N       add N coins
//	gems N        add N gems
//	stage N       jump to stage N
//	wave N        set the slugs beaten on this stage
//	own ID        own bug ID (and put it on the team if there is room)
//	hatch ID      hatch bug ID with the animation
//	level ID N    set bug ID to level N
//	charge        fill every skill meter
//	away SECS     pretend the game was closed for SECS seconds
//	scroll N      scroll the menu down by N
//	shot FILE     save a screenshot as PNG
//	quit          exit
//
// Eggs use a fixed seed while a script is set.
type script struct {
	cmds []string
	wait int
	shot string
}

func loadScript() *script {
	v := os.Getenv("BUGS_SCRIPT")
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
		p := g.p
		switch f[0] {
		case "wait":
			sc.wait = int(num(1))
		case "tap":
			x, y := num(1), num(2)
			if !g.pointerDown(x, y) {
				g.click(x, y)
			}
		case "press":
			g.press(f[1])
		case "coins":
			p.Coins += num(1)
		case "gems":
			p.Gems += int(num(1))
		case "stage":
			p.Stage = int(num(1))
			p.Best = max(p.Best, p.Stage-1)
			p.Wave = 0
			g.slugs, g.bossOn = nil, false
		case "wave":
			p.Wave = int(num(1))
		case "own":
			id := int(num(1))
			if !p.Bugs[id].Have {
				p.hatch(id)
			}
			g.syncTeam()
		case "hatch":
			g.hatchQueue = append(g.hatchQueue, int(num(1)))
		case "level":
			p.Bugs[int(num(1))].Level = int(num(2))
		case "charge":
			for i := range g.team {
				g.team[i].charge = 1
			}
		case "away":
			p.LastSeen -= int64(num(1))
			if c, secs := p.offline(g.now()); c > 0 {
				g.welcome, g.welcomeS, g.welcomeT = c, secs, 0
			}
		case "scroll":
			g.scroll[g.tab] += num(1)
		case "shot":
			sc.shot = f[1]
		case "quit":
			g.Save()
			return ebiten.Termination
		default:
			return fmt.Errorf("BUGS_SCRIPT: unknown command %q", f[0])
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
	img := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(img.Pix)
	if f, err := os.Create(sc.shot); err == nil {
		png.Encode(f, img)
		f.Close()
	}
	sc.shot = ""
}
