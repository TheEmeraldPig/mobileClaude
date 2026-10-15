package game

import (
	"math"
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
		case scPause, scInv, scDebug:
			play()
		case scSettings:
			if settingsFrom == scMain {
				openScreen(scMain)
			} else {
				openScreen(scPause)
			}
		case scSeed:
			openScreen(scMain)
		}
	} else if bindJustPressed("inv") && (cur == scPlaying || cur == scInv) && listening == "" {
		if cur == scInv {
			play()
		} else {
			openScreen(scInv)
		}
	}
	if cur == scPlaying {
		input.left, input.right, input.lev = bindDown("left"), bindDown("right") || scriptRight, bindDown("lev") || scriptLev
		for i := 0; i < 5; i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				if selected == i {
					selected = -1
				} else {
					selected = i
				}
				sfx("click")
			}
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
					selected = 4
				}
			} else {
				selected = (selected + d + 5) % 5
			}
			sfx("click")
		}
		// the mouse also aims while held
		if useActive && !isTouch {
			aimSX, aimSY = hoverX, hoverY
		}
	} else {
		input = gameInput{}
	}

	if !paused() {
		updatePlayer()
		followPlayer()
		useHeld()
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
	case scDebug:
		debugMenu()
	}
	drawToast()

	if frame >= autosaveAt {
		autosaveAt = frame + 60*20
		saveGame()
	}
	checkForUpdate()
	return nil
}

func placeCamera() {
	cam.x = player.x + player.w/2 + peekSX/zoom - logicalW/2/zoom
	cam.y = player.y + player.h/2 + peekSY/zoom - logicalH/2/zoom
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colNight)
	drawWorld(screen)
	for _, op := range ops {
		op(screen)
	}
	takeShot(screen)
}

// Render the visible cells into a small image (one pixel per cell), then scale it up.
func drawWorld(screen *ebiten.Image) {
	vw, vh := int(logicalW/zoom/CELL)+2, int(logicalH/zoom/CELL)+2
	viewW, viewH = vw, vh
	if worldImg == nil || worldImg.Bounds().Dx() != vw || worldImg.Bounds().Dy() != vh {
		worldImg = ebiten.NewImage(vw, vh)
		worldPix = make([]byte, vw*vh*4)
	}
	ox, oy := int(math.Floor(cam.x/CELL)), int(math.Floor(cam.y/CELL))
	pix := worldPix
	for y := 0; y < vh; y++ {
		wy := oy + y
		for x := 0; x < vw; x++ {
			wx := ox + x
			o := (y*vw + x) * 4
			c, s := bgRGB, 0
			if wx >= 0 && wx < W && wy >= 0 && wy < H {
				i := wy*W + wx
				if t := grid[i]; t != EMPTY {
					c, s = blockRGB[t], int(shade[i])
				}
			} else if wy >= 0 {
				c = blockRGB[STONE]
			}
			pix[o] = clamp8(c[0] + s)
			pix[o+1] = clamp8(c[1] + s)
			pix[o+2] = clamp8(c[2] + s)
			pix[o+3] = 255
		}
	}
	put := func(x, y int, c [3]int, s int) {
		if x < 0 || y < 0 || x >= vw || y >= vh {
			return
		}
		o := (y*vw + x) * 4
		pix[o], pix[o+1], pix[o+2] = clamp8(c[0]+s), clamp8(c[1]+s), clamp8(c[2]+s)
	}
	drawPlayer(put, ox, oy)
	for _, q := range parts {
		put(int(math.Floor(q.x))-ox, int(math.Floor(q.y))-oy, q.c, 0)
	}
	drawEnergy(put, ox, oy)
	if cur == scPlaying {
		drawAim(put, ox, oy)
	}
	worldImg.WritePixels(pix)
	k := zoom * CELL * uiScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate((float64(ox*CELL)-cam.x)*zoom*uiScale, (float64(oy*CELL)-cam.y)*zoom*uiScale)
	screen.DrawImage(worldImg, op)
}

func clamp8(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
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
	if st := loadSave(); st != nil && restoreState(st) {
		if os.Getenv("MOTE_RESUME") == "1" || hasArg("--resume") {
			play()
			toast("Updated to the newest version")
		}
	} else {
		createWorld(prefs.Seed)
		starterKit()
	}
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
