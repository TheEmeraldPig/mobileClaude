package game

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	frame              int
	uiScale            = 1.0
	logicalW, logicalH = 960.0, 540.0
)

type Game struct{}

func (g *Game) Update() error {
	if Crashed() {
		frame++
		runScript()        // (test hooks keep running, to look at the crash screen)
		if CrashUpdate() { // Esc or Close on the crash screen
			return ebiten.Termination
		}
		return nil
	}
	defer CatchCrash()
	if ebiten.IsWindowBeingClosed() {
		savePrefs()
		netShutdown()
		return ebiten.Termination
	}
	frame++
	ops = ops[:0]
	netUpdate()
	runScript()
	readPointers()
	clientUpdate()

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		prefs.Full = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(prefs.Full)
		savePrefs()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch cur {
		case scSettings:
			openScreen(settingsFrom)
		case scHelp:
			openScreen(helpFrom)
		case scJoin:
			if netRole == netGuest {
				netShutdown()
			}
			openScreen(scMain)
		case scTable:
			tableEscape()
		}
	}
	if cur == scTable {
		tableKeys()
	}

	switch cur {
	case scMain:
		mainMenu()
	case scSettings:
		if settingsFrom == scTable {
			behind(tableScreen)
		}
		settingsMenu()
	case scJoin:
		joinMenu()
	case scHelp:
		if helpFrom == scTable {
			behind(tableScreen)
		}
		helpMenu()
	case scTable:
		tableScreen()
	}
	if cur != scJoin {
		stopFinding()
	}
	drawFlying()
	drawFloaters()
	updateFX()
	drawToast()
	checkForUpdate()
	return nil
}

// Keyboard shortcuts at the table: Enter plays, D discards, S uses the special,
// 1-8 pick cards
// Keyboard and mouse shortcuts at the table (the Handy controls; listed in Settings)
func tableKeys() {
	if view == nil || ov != ovNone {
		return
	}
	p := me()
	if p == nil {
		return
	}
	just := inpututil.IsKeyJustPressed
	// any time in a match
	switch {
	case just(ebiten.KeyTab) && myRun != nil:
		ov = ovDeck
	case just(ebiten.KeyH) && len(p.Hexes) > 0 && view.Mode != ModeCoop:
		ov = ovHexes
	case just(ebiten.KeyF):
		cycleSpeed()
	}
	shopping := view.Phase == "shop" || (view.Mode == ModeRace && p.Stage == "shop" && view.Phase == "play")
	if shopping && !p.Out {
		items := len(view.Auction)
		if view.Mode == ModeRace {
			items = len(p.Shop)
		}
		for i := 0; i < items && i < 9; i++ {
			if just(ebiten.Key1 + ebiten.Key(i)) {
				if view.Mode == ModeRace {
					act(action{T: "rbuy", Item: i})
					sfx("coin")
				} else {
					act(action{T: "bid", Item: i})
					sfx("bid")
				}
			}
		}
		switch {
		case just(ebiten.KeyQ):
			ov = ovTomes
		case just(ebiten.KeyW):
			ov = ovTrade
		case just(ebiten.KeyEnter) || just(ebiten.KeySpace):
			if view.Mode == ModeRace {
				act(action{T: "rcont"})
			} else if p.Ready {
				act(action{T: "unready"})
			} else {
				act(action{T: "ready"})
			}
		}
		return
	}
	if view.Phase != "play" || myRun == nil || myRun.Done || animating() {
		return
	}
	for i := 0; i < 9; i++ {
		if just(ebiten.Key1 + ebiten.Key(i)) {
			myRun.Toggle(i)
			sfx("select")
		}
	}
	switch {
	case just(ebiten.KeyEnter) || just(ebiten.KeySpace):
		doPlay()
	case just(ebiten.KeyD):
		doDiscard()
	case just(ebiten.KeyS):
		doSpecial()
	case just(ebiten.KeyR):
		sortHand(false)
	case just(ebiten.KeyT):
		sortHand(true)
	case just(ebiten.KeyC) || (!isTouch && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)):
		deselectAll()
		sfx("unselect")
	}
}

// Game speed: 1x, 2x, 4x
func cycleSpeed() {
	prefs.Speed = map[int]int{1: 2, 2: 4, 4: 1}[prefs.Speed]
	savePrefs()
	toast(fmt.Sprintf("Game speed x%d", prefs.Speed))
}

func (g *Game) Draw(screen *ebiten.Image) {
	if Crashed() {
		DrawCrash(screen)
		takeShot(screen)
		return
	}
	defer CatchCrash()
	drawBackground(screen)
	dx, dy := shakeOffset()
	if dx == 0 && dy == 0 {
		for _, op := range ops {
			op(screen)
		}
	} else {
		// shaking: draw everything off screen, then put it down slightly moved
		b := screen.Bounds()
		if shakeBuf == nil || shakeBuf.Bounds() != b {
			if shakeBuf != nil {
				shakeBuf.Deallocate()
			}
			shakeBuf = ebiten.NewImage(b.Dx(), b.Dy())
		}
		shakeBuf.Clear()
		for _, op := range ops {
			op(shakeBuf)
		}
		o := &ebiten.DrawImageOptions{}
		o.GeoM.Translate(dx*uiScale, dy*uiScale)
		screen.DrawImage(shakeBuf, o)
	}
	takeShot(screen)
}

var shakeBuf *ebiten.Image

// behind draws a screen under a menu without letting it react to the pointer
func behind(f func()) {
	saved, hx, hy := ptr, hoverX, hoverY
	ptr, hoverX, hoverY = uiPointer{}, -1000, -1000
	f()
	ptr, hoverX, hoverY = saved, hx, hy
}

func (g *Game) Layout(outW, outH int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	uiScale = s
	// On big screens, scale the UI up so it stays readable
	if float64(outH) > 760 {
		uiScale = s * float64(outH) / 760
	}
	uiScale *= []float64{0.85, 1, 1.15, 1.3}[prefs.UISize]
	logicalW, logicalH = float64(outW)*s/uiScale, float64(outH)*s/uiScale
	return int(float64(outW) * s), int(float64(outH) * s)
}

// NewGame sets everything up; shared by the desktop main() and the Android wrapper.
func NewGame() *Game {
	loadPrefs()
	ebiten.SetWindowClosingHandled(true)
	initFonts()
	initFX()
	initAudio()
	setVolumes()
	clearPointers()
	return &Game{}
}

func hasArg(a string) bool {
	for _, x := range os.Args[1:] {
		if x == a {
			return true
		}
	}
	return false
}
