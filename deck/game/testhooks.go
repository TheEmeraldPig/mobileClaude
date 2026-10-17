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

// DECK_SCRIPT runs commands for testing without a person, separated by ';':
//
//	wait N        wait N frames
//	tap X Y       tap at logical position X,Y
//	press ID      press a button by id: start, tab:N, buy:N, sel:N, toggle, enh:N,
//	              node:N, play:N, endturn, pick:N, offer:N, remove, forge, picked:N,
//	              rest, choose:N, continue, deck, revive, ad, home, upgrade
//	card ID       add a card to the collection
//	auto          play out the current fight with a simple bot
//	next          travel to the first reachable map node
//	goto KIND     jump to a spot of that kind: fight elite shop rest event treasure boss
//	gold N        add N gold
//	gems N        add N gems
//	shot FILE     save a screenshot as PNG
//	quit          exit
//
// Runs and chests use a fixed seed while a script is set.
type script struct {
	cmds []string
	wait int
	shot string
}

func loadScript() *script {
	v := os.Getenv("DECK_SCRIPT")
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
		case "press":
			g.press(f[1])
		case "gold":
			g.p.Gold += int(num(1))
		case "card":
			g.p.Cards = append(g.p.Cards, Card{ID: f[1]})
		case "auto":
			autoFight(g)
		case "next":
			if r := g.run(); r != nil && len(r.Reachable()) > 0 {
				g.press("node:" + strconv.Itoa(r.Reachable()[0]))
			}
		case "goto":
			if r := g.run(); r != nil {
				for i, n := range r.Nodes {
					if nodeNames[n.Kind] == map[string]string{"fight": "Fight", "elite": "Elite", "shop": "Merchant", "rest": "Campfire", "event": "Mystery", "treasure": "Treasure", "boss": "Boss"}[f[1]] {
						g.travel(i)
						break
					}
				}
			}
		case "hp":
			if r := g.run(); r != nil {
				r.HP = int(num(1))
			}
		case "coins":
			if r := g.run(); r != nil {
				r.Coins += int(num(1))
			}
		case "evt":
			if r := g.run(); r != nil {
				r.Event = int(num(1))
				g.eventMsg = ""
			}
		case "gems":
			g.p.Gems += int(num(1))
		case "shot":
			sc.shot = f[1]
		case "quit":
			g.Save()
			return ebiten.Termination
		default:
			return fmt.Errorf("DECK_SCRIPT: unknown command %q", f[0])
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

// autoFight plays every affordable card and ends turns until the fight is over.
func autoFight(g *Game) {
	r := g.run()
	for turn := 0; turn < 60 && r.B != nil && r.B.EHP > 0 && !r.Lost; turn++ {
		b := r.B
		for i := 0; i < len(b.Hand); {
			if b.Playable(i) {
				b.Play(i)
				if b.EHP <= 0 {
					break
				}
				continue
			}
			i++
		}
		if b.EHP > 0 {
			b.EndTurn()
		}
	}
	g.handPos = nil
}
