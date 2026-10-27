package game

import (
	"math"
	"math/rand"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	frame              int
	uiScale            = 1.0
	logicalW, logicalH = 960.0, 540.0
	zoom               = 1.5
	viewW, viewH       int // visible area in cells
	cam                struct{ x, y float64 }
	worldImg           *ebiten.Image
	worldPix           []byte
)

var bgRGB = [3]int{16, 18, 22}

func applyFOV() {
	zoom = 1.5 * 3 / float64(prefs.FOV)
}

type Game struct{}

func (g *Game) Update() error {
	if Crashed() {
		return nil
	}
	defer CatchCrash()
	if ebiten.IsWindowBeingClosed() {
		savePrefs()
		return ebiten.Termination
	}
	frame++
	ops = ops[:0]
	netUpdate()
	runScript()
	readPointers(cur != scPlaying)

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		prefs.Full = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(prefs.Full)
		savePrefs()
	}
	if listening == "" && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch cur {
		case scPlaying:
			openScreen(scPause)
		case scPause, scPetals, scTravel:
			play()
		case scSettings:
			if settingsFrom == scMain {
				openScreen(scMain)
			} else {
				openScreen(scPause)
			}
		case scName:
			openScreen(scMain)
		}
	} else if cur == scPlaying || cur == scPetals || cur == scTravel {
		if bindJustPressed("petals") {
			toggleScreen(scPetals)
		} else if bindJustPressed("map") {
			toggleScreen(scTravel)
		}
	}
	myMode = mmo.ModeNormal
	if cur == scPlaying {
		input.left, input.right, input.lev = bindDown("left"), bindDown("right") || scriptRight, bindDown("lev") || scriptLev
		attack := mouseAttack || held("attack") || scriptAttack
		defend := held("defend") || (!isTouch && ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight))
		switch {
		case attack:
			myMode = mmo.ModeAttack
		case defend:
			myMode = mmo.ModeDefend
		}
		for i := range mmo.Emotes {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				sendEmote(i)
			}
		}
	} else {
		input = gameInput{}
	}

	if netState == netOnline && curIsle != nil && cur != scDead {
		worldTime = math.Mod(worldTime+1.0/dayFrames, 1)
		updatePlayer()
	} else if cur == scMain {
		worldTime = math.Mod(worldTime+1.0/dayFrames, 1)
	}
	updateRemote()
	updateParticles()
	setHum(player.lifting && cur != scMain, player.energy)
	peekSX += (peekX - peekSX) * 0.04
	peekSY += (peekY - peekSY) * 0.04
	if cur == scMain && netState == netOff {
		menuCamera()
	} else {
		placeCamera()
	}

	switch cur {
	case scMain:
		mainMenu()
	case scName:
		mainMenuArt()
		nameMenu()
	case scPlaying:
		worldLabels()
		drawPopups()
		hud()
	case scPause:
		pauseMenu()
	case scPetals:
		petalsMenu()
	case scTravel:
		travelMenu()
	case scSettings:
		if settingsFrom == scMain {
			mainMenuArt()
		}
		settingsMenu()
	case scDead:
		worldLabels()
		deadMenu()
	case scAway:
		awayMenu()
	}
	shake *= 0.86
	drawToast()
	checkForUpdate()
	return nil
}

func toggleScreen(s screen) {
	if cur == s {
		play()
	} else {
		openScreen(s)
	}
}

// screen shake strength in world px
var shake float64

func placeCamera() {
	cam.x = player.x + player.w/2 + peekSX/zoom - logicalW/2/zoom
	cam.y = player.y + player.h/2 + peekSY/zoom - logicalH/2/zoom
	if shake > 0.3 {
		cam.x += (rand.Float64()*2 - 1) * shake
		cam.y += (rand.Float64()*2 - 1) * shake
	}
	clampCamera()
}

// Behind the main menu: Haven, drifting slowly along
func menuCamera() {
	if curIsle == nil {
		loadIsle("haven")
		isleID = "haven"
	}
	sx, sy := curIsle.SpawnPos()
	cam.x = sx - logicalW/2/zoom + math.Sin(float64(frame)*0.0015)*500
	cam.y = sy - logicalH/2/zoom - 30
	clampCamera()
}

func (g *Game) Draw(screen *ebiten.Image) {
	if Crashed() {
		DrawCrash(screen)
		return
	}
	defer CatchCrash()
	screen.Fill(colNight)
	drawWorld(screen)
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
	applyFOV()
	initWorld()
	loadIsle("haven")
	isleID = "haven"
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
