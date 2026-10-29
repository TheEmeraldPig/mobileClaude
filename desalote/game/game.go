package game

import (
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
		settingsMenu()
	case scJoin:
		joinMenu()
	case scHelp:
		if helpFrom == scTable {
			tableScreen()
		}
		helpMenu()
	case scTable:
		tableScreen()
	}
	if cur != scJoin {
		stopFinding()
	}
	drawToast()
	checkForUpdate()
	return nil
}

// Keyboard shortcuts at the table: Enter plays, D discards, S uses the special,
// 1-8 pick cards
func tableKeys() {
	if view == nil || view.Phase != "play" || myRun == nil || myRun.Done || animating() || ov != ovNone {
		return
	}
	for i := 0; i < 9; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			myRun.Toggle(i)
			sfx("select")
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if res := myRun.Play(); res != nil {
			anim = &scoreAnim{res: res, start: frame}
			sfx("play")
			reportRun()
			dealAt = frame
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyD):
		if myRun.CanDiscard() {
			myRun.Discard()
			sfx("discard")
			dealAt = frame
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		if res := myRun.UseSpecial(); res != nil {
			anim = &scoreAnim{res: res, start: frame}
			sfx("special")
			reportRun()
			dealAt = frame
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if Crashed() {
		DrawCrash(screen)
		return
	}
	defer CatchCrash()
	screen.Fill(colNight)
	for _, op := range ops {
		op(screen)
	}
	takeShot(screen)
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
