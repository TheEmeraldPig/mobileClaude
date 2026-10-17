package game

import (
	"math"
	"math/rand"
	"os"

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
	autosaveAt         int
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
	frame++
	ops = ops[:0]
	runScript()
	readPointers(cur != scPlaying)

	// keyboard shortcuts
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		prefs.Full = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(prefs.Full)
		savePrefs()
	}
	if listening == "" && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch cur {
		case scPlaying:
			openScreen(scPause)
		case scPause, scInv, scChest:
			play()
		case scSettings:
			if settingsFrom == scMain {
				openScreen(scMain)
			} else {
				openScreen(scPause)
			}
		case scSeed, scWorlds:
			openScreen(scMain)
		case scCraft:
			if craftStation == "" {
				openScreen(scInv)
			} else {
				play()
			}
		}
	} else if bindJustPressed("inv") && (cur == scPlaying || cur == scInv || cur == scCraft || cur == scChest) && listening == "" {
		if cur == scPlaying {
			openScreen(scInv)
		} else {
			play()
		}
	}
	if cur == scPlaying {
		input.left, input.right, input.lev = bindDown("left"), bindDown("right") || scriptRight, bindDown("lev") || scriptLev
		for i := 0; i < hotbarN; i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				if selected == i {
					selected = -1
				} else {
					selected = i
				}
				showItemName()
				sfx("click")
			}
		}
		// Q drops one of the selected item, Ctrl+Q the whole stack
		if bindJustPressed("drop") && selected >= 0 && inv[selected] != nil {
			n := 1
			if ebiten.IsKeyPressed(ebiten.KeyControl) {
				n = inv[selected].N
			}
			dropSlot(selected, n)
		}
		if _, wy := ebiten.Wheel(); wy != 0 {
			d := 1
			if wy > 0 {
				d = -1
			}
			if selected < 0 {
				if d > 0 {
					selected = 0
				} else {
					selected = hotbarN - 1
				}
			} else {
				selected = (selected + d + hotbarN) % hotbarN
			}
			showItemName()
			sfx("click")
		}
		// the mouse also aims while held
		if useActive && !isTouch && !scriptAim {
			aimSX, aimSY = hoverX, hoverY
			updateShape()
		}
		// right-click opens a station
		if !isTouch && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			x, y := int(math.Floor((cam.x+hoverX/zoom)/CELL)), int(math.Floor((cam.y+hoverY/zoom)/CELL))
			if x >= 0 && x < W && y >= 0 && y < H && isDoor(grid[y*W+x]) {
				if cellInBuildReach(x, y) {
					toggleDoor(x, y)
				}
			} else if o := objAt(x, y); o != nil {
				if objInReach(o) {
					openStation(o.kind)
				} else {
					toast("Get closer to use the " + stations[o.kind].name)
				}
			}
		}
		buildKeys()
		if bindJustPressed("craft") && heldBuildKind() != buildPaint {
			openHandCrafting()
		}
	} else {
		input = gameInput{}
	}

	if !paused() {
		worldTime = math.Mod(worldTime+1.0/dayFrames, 1)
		updatePlayer()
		followPlayer()
		useHeld()
		updateEntities()
		checkObjects()
		updateParticles()
	}
	setHum(player.lifting && !paused(), player.energy)
	peekSX += (peekX - peekSX) * 0.04
	peekSY += (peekY - peekSY) * 0.04
	placeCamera()
	if !paused() {
		stepSim()
	}

	// UI for this frame
	switch cur {
	case scMain:
		mainMenu()
	case scSeed:
		mainMenuArt()
		seedMenu()
	case scPlaying:
		hud()
	case scPause:
		pauseMenu()
	case scInv:
		invMenu()
	case scSettings:
		if settingsFrom == scMain {
			mainMenuArt()
		}
		settingsMenu()
	case scDead:
		deathMenu()
	case scCraft:
		craftMenu()
	case scWorlds:
		mainMenuArt()
		worldsMenu()
	case scChest:
		storageMenu()
	}
	shake *= 0.86
	drawToast()

	if frame >= autosaveAt {
		autosaveAt = frame + 60*20
		saveGame()
	}
	checkForUpdate()
	return nil
}

// screen shake strength in world px, set by explosions and hits
var shake float64

func placeCamera() {
	cam.x = player.x + player.w/2 + peekSX/zoom - logicalW/2/zoom
	cam.y = player.y + player.h/2 + peekSY/zoom - logicalH/2/zoom
	if shake > 0.3 && !paused() {
		cam.x += (rand.Float64()*2 - 1) * shake
		cam.y += (rand.Float64()*2 - 1) * shake
	}
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
	logicalW, logicalH = float64(outW)*s/uiScale, float64(outH)*s/uiScale
	return int(float64(outW) * s), int(float64(outH) * s)
}

// NewGame sets everything up; shared by the desktop main() and the Android wrapper.
func NewGame() *Game {
	loadPrefs()
	initFonts()
	initItems()
	initAudio()
	setVolumes()
	applyFOV()
	// enough loaded width for the widest field of view plus looking around
	initWorld(2844)
	loadWorldList()
	if last := lastWorld(); last != nil && openWorld(*last) {
		cur = scMain
		if os.Getenv("MOTE_RESUME") == "1" || hasArg("--resume") {
			play()
			toast("Updated to the newest version")
		}
	} else {
		// a world to show behind the main menu; it isn't saved
		createWorld(prefs.Seed)
		world = WorldInfo{}
		player.maxHP, player.hp = 10, 10
	}
	clearPointers()
	placeCamera()
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
