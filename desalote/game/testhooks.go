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
//
//	DESALOTE_SCRIPT="10:solo,20:start,60:autopick,200:shot=/tmp/a.png,210:quit"
//
// runs actions at given frames. DESALOTE_TOUCH=1 uses the phone layout,
// DESALOTE_SIZE=1200x560 sets the window size, DESALOTE_NOAUDIO=1 skips sound.

type scriptStep struct {
	frame int
	act   string
}

var (
	script    []scriptStep
	shotPath  string
	scriptTap struct {
		on, down, hold bool
		x, y           float64
	}
)

func init() {
	if os.Getenv("DESALOTE_TOUCH") == "1" {
		isTouch = true
	}
	for _, part := range strings.Split(os.Getenv("DESALOTE_SCRIPT"), ",") {
		f, a, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		n, _ := strconv.Atoi(f)
		script = append(script, scriptStep{n, a})
	}
}

func runScript() {
	// a scripted tap is down for one frame and up the next (unless held)
	if scriptTap.on && !scriptTap.hold {
		if scriptTap.down {
			scriptTap.down = false
		} else {
			scriptTap.on = false
		}
	}
	for _, s := range script {
		if s.frame != frame {
			continue
		}
		cmd, arg, _ := strings.Cut(s.act, "=")
		switch cmd {
		case "solo":
			startSolo()
		case "host":
			startHost(false)
		case "joinlan": // joinlan=ADDRESS
			openScreen(scJoin)
			joinLAN(arg)
		case "joinfound": // the first table heard on the network
			if l := foundTables(); len(l) > 0 {
				joinLAN(l[0].addr)
			}
		case "join":
			openScreen(scJoin)
		case "mode": // mode=N (host)
			if host != nil {
				host.Mode, _ = strconv.Atoi(arg)
			}
		case "length":
			if host != nil {
				host.Length, _ = strconv.Atoi(arg)
			}
		case "bots": // bots=N more bots
			n, _ := strconv.Atoi(arg)
			for i := 0; i < n && host != nil; i++ {
				host.addBot()
			}
		case "start":
			if host != nil && host.canStart() {
				host.start()
			}
		case "autopick": // ready after the deal
			act(action{T: "dealt"})
		case "select": // select=0:2:3 cards in my hand
			if myRun != nil {
				for i := range myRun.Sel {
					myRun.Sel[i] = false
				}
				for _, k := range strings.Split(arg, ":") {
					i, _ := strconv.Atoi(k)
					myRun.Toggle(i)
				}
			}
		case "botmove": // let a bot brain make my next move
			if myRun != nil {
				if res := myRun.botTurn(); res != nil {
					startAnim(res, nil)
				}
				reportRun()
			}
		case "money": // money=N for me (host only)
			if host != nil {
				if p := host.player(myID); p != nil {
					p.Money, _ = strconv.Atoi(arg)
				}
			}
		case "givecharm":
			if host != nil {
				if p := host.player(myID); p != nil && charms[arg] != nil {
					p.Charms = append(p.Charms, arg)
				}
			}
		case "givehex":
			if host != nil {
				if p := host.player(myID); p != nil && hexes[arg] != nil {
					p.Hexes = append(p.Hexes, arg)
				}
			}
		case "hexme": // a hex on me next round
			if host != nil {
				if p := host.player(myID); p != nil {
					p.OnMe = append(p.OnMe, arg)
				}
			}
		case "shop": // jump to the shop (host)
			if host != nil {
				host.openShop()
			}
		case "end": // end the match now (host)
			if host != nil {
				host.end(playerName() + " wins on a test")
			}
		case "tab":
			shopTab, _ = strconv.Atoi(arg)
		case "ov":
			n, _ := strconv.Atoi(arg)
			ov = overlay(n)
		case "help":
			helpFrom = cur
			helpPage, _ = strconv.Atoi(arg)
			openScreen(scHelp)
		case "settings":
			openScreen(scSettings)
		case "ready":
			act(action{T: "ready"})
		case "rcont":
			act(action{T: "rcont"})
		case "skipshop": // end the shop timer now (host)
			if host != nil && host.Phase == "shop" {
				host.timer = 1
			}
		case "tap": // tap=x:y in logical px
			xs, ys, _ := strings.Cut(arg, ":")
			scriptTap.x, _ = strconv.ParseFloat(xs, 64)
			scriptTap.y, _ = strconv.ParseFloat(ys, 64)
			scriptTap.on, scriptTap.down = true, true
		case "press": // press=x:y and keep holding (move=x:y, release)
			xs, ys, _ := strings.Cut(arg, ":")
			scriptTap.x, _ = strconv.ParseFloat(xs, 64)
			scriptTap.y, _ = strconv.ParseFloat(ys, 64)
			scriptTap.on, scriptTap.down, scriptTap.hold = true, true, true
		case "move":
			xs, ys, _ := strings.Cut(arg, ":")
			scriptTap.x, _ = strconv.ParseFloat(xs, 64)
			scriptTap.y, _ = strconv.ParseFloat(ys, 64)
		case "release":
			scriptTap.down, scriptTap.hold = false, false
		case "handy": // the Handy settings tab
			settingsFrom, settingsTab = cur, 1
			openScreen(scSettings)
		case "state":
			if view != nil {
				p := me()
				fmt.Printf("frame %d: phase %s ante %d blind %d round %d", frame, view.Phase, view.Ante, view.Blind, view.Round)
				if p != nil {
					fmt.Printf(" | me: lives %d money %d score %d done %v charms %v", p.Lives, p.Money, p.Score, p.Done, p.Charms)
				}
				fmt.Println()
				for _, q := range view.Players {
					fmt.Printf("  %d %s lives %d $%d score %d total %d out %v\n", q.ID, q.Name, q.Lives, q.Money, q.Score, q.Total, q.Out)
				}
			} else {
				fmt.Printf("frame %d: no match (screen %d) %s\n", frame, cur, netStatus)
			}
		case "shot":
			shotPath = arg
		case "fps":
			fmt.Printf("fps %.1f\n", ebiten.ActualFPS())
		case "crash": // make the game crash, to see the crash screen
			var m map[string]int
			m["boom"]++ // writing to a nil map panics
		case "crashscroll": // crashscroll=N lines down
			n, _ := strconv.Atoi(arg)
			crashScroll = float64(n)
		case "crashclose": // press Close on the crash screen
			crashQuit = true
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

// WindowSizeOverride lets tests pick a window size with DESALOTE_SIZE=800x370.
func WindowSizeOverride() (int, int) {
	w, h, ok := strings.Cut(os.Getenv("DESALOTE_SIZE"), "x")
	if !ok {
		return 0, 0
	}
	a, _ := strconv.Atoi(w)
	b, _ := strconv.Atoi(h)
	return a, b
}
