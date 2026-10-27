package game

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/hajimehoshi/ebiten/v2"
)

// Test hooks for automated checks (unused in normal play):
//
//	REMOTE_SCRIPT="30:solo,200:travel=meadow,400:shot=/tmp/a.png,410:quit"
//
// runs actions at given frames: solo, online, travel=ID, right, left, lev, stop, attack,
// defend, calm, tpmob (next to the nearest creature), emote=N, petals, map, pause,
// settings, name, main, dead, away, state, fps, shot=PATH, quit.

type scriptStep struct {
	frame int
	act   string
}

var (
	script                               []scriptStep
	shotPath                             string
	scriptRight, scriptLev, scriptAttack bool
	scriptLeft                           bool
)

func init() {
	if os.Getenv("REMOTE_TOUCH") == "1" {
		isTouch = true
	}
	for _, part := range strings.Split(os.Getenv("REMOTE_SCRIPT"), ",") {
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
		case "solo":
			if prefs.Name == "" {
				prefs.Name = "Tester"
			}
			playSolo()
		case "online":
			playOnline()
		case "server": // server=URL
			prefs.Server = arg
		case "travel":
			send(mmo.CMsg{T: "travel", Isle: arg})
		case "right":
			scriptRight = true
		case "lev":
			scriptLev = true
		case "stop":
			scriptRight, scriptLev, scriptAttack = false, false, false
		case "attack":
			scriptAttack = true
		case "calm":
			scriptAttack = false
		case "tpmob": // stand beside the nearest creature
			best := math.Inf(1)
			var tgt *Entity
			for _, e := range ents {
				if d := math.Hypot(e.x-player.x, e.y-player.y); d < best {
					best, tgt = d, e
				}
			}
			if tgt != nil {
				player.x, player.y = tgt.x-40, tgt.y-6
				unstick()
				updatePlayerCells()
			}
		case "emote":
			n, _ := strconv.Atoi(arg)
			sendEmote(n)
		case "petals":
			openScreen(scPetals)
		case "map":
			openScreen(scTravel)
		case "pause":
			openScreen(scPause)
		case "settings":
			openScreen(scSettings)
		case "name":
			openName(false)
		case "main":
			openScreen(scMain)
		case "play":
			play()
		case "dead":
			openScreen(scDead)
		case "away": // show the away screen with made-up numbers
			awayNews = &mmo.Away{Secs: 3*3600 + 700, Isle: "meadow", Kills: 214, XP: 1040, Coins: 260, Levels: 2,
				Petals: map[mmo.PetalID]int{"spark:0": 12, "leaf:1": 3, "thorn:0": 2}}
			openScreen(scAway)
		case "pickbag": // pick the first bag petal on the petals screen
			if me != nil {
				if b := me.Sorted(); len(b) > 0 {
					pickBag = b[0]
				}
			}
		case "state":
			lvl, xp, coins, kills := 0, int64(0), int64(0), int64(0)
			if me != nil {
				lvl, xp, coins, kills = me.Level, me.XP, me.Coins, me.Kills
			}
			fmt.Printf("frame %d net=%d isle=%s hp=%.0f/%.0f lvl=%d xp=%d coins=%d kills=%d mobs=%d others=%d player=(%.0f,%.0f) status=%q\n",
				frame, netState, isleID, myHP, myMaxHP, lvl, xp, coins, kills, len(ents), len(others), player.x, player.y, netStatus)
			for _, o := range others {
				name := "?"
				if o.info != nil {
					name = o.info.Name
				}
				fmt.Printf("  other %d %s seen=%d at (%.0f,%.0f)\n", o.id, name, o.seen, o.x, o.y)
			}
		case "fps":
			fmt.Printf("frame %d fps=%.1f tps=%.1f\n", frame, ebiten.ActualFPS(), ebiten.ActualTPS())
		case "shot":
			shotPath = arg
		case "quit":
			os.Exit(0)
		}
	}
}

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

// WindowSizeOverride lets tests pick a window size with REMOTE_SIZE=800x370.
func WindowSizeOverride() (int, int) {
	w, h, ok := strings.Cut(os.Getenv("REMOTE_SIZE"), "x")
	if !ok {
		return 0, 0
	}
	a, _ := strconv.Atoi(w)
	b, _ := strconv.Atoi(h)
	return a, b
}
