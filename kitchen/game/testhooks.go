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

// KITCHEN_SCRIPT runs commands for testing without a person, separated by ';':
//
//	wait N        wait N frames
//	tap X Y       tap at layout position X,Y (a press and release)
//	press ID      press a button by id: tab:N, up:N (-1 = tap power), mult, skill:N,
//	              boss, team:N, buy:N, free, collect, collect2, modal, station:N,
//	              build:N, cook:N, serve:N
//	coins N       add N coins
//	gems N        add N gems
//	stage N       jump to stage N
//	wave N        set the slugs beaten on this stage
//	own ID        own chef ID (and put them on the crew if there is room)
//	hire ID       recruit chef ID with the animation
//	level ID N    set chef ID to level N
//	ing N         add N of every ingredient
//	build ST N    set kitchen station ST to level N
//	cooked R Q    put a finished dish R of quality Q in the pantry
//	needle        stop the cooking needle now
//	finish        finish everything that is cooking
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
	v := os.Getenv("KITCHEN_SCRIPT")
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
			g.mon, g.bossOn = nil, false
		case "wave":
			p.Wave = int(num(1))
		case "own":
			id := int(num(1))
			if !p.Chefs[id].Have {
				p.hire(id)
			}
			g.syncTeam()
		case "hire":
			g.hireQueue = append(g.hireQueue, int(num(1)))
		case "level":
			p.Chefs[int(num(1))].Level = int(num(2))
		case "ing":
			for i := range p.Ing {
				p.Ing[i] += int(num(1))
			}
		case "build":
			p.Stations[int(num(1))] = int(num(2))
		case "cooked":
			p.Pantry = append(p.Pantry, dish{R: int(num(1)), Q: num(2)})
		case "needle":
			g.stopNeedle()
		case "finish":
			p.tickKitchen(1e6, false)
		case "charge":
			for i := range g.team {
				g.team[i].charge = 1
			}
		case "away":
			p.LastSeen -= int64(num(1))
			g.arrive(g.now())
		case "scroll":
			g.scroll[g.tab] += num(1)
		case "shot":
			sc.shot = f[1]
		case "quit":
			g.Save()
			return ebiten.Termination
		default:
			return fmt.Errorf("KITCHEN_SCRIPT: unknown command %q", f[0])
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
