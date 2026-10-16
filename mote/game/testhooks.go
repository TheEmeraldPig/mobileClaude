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
//   MOTE_SCRIPT="30:play,90:inv,150:shot=/tmp/a.png,160:quit"
// runs actions at given frames: play, pause, inv, settings, debug, main, seed, right, lev, stop, select=N, shot=PATH, quit.

type scriptStep struct {
	frame int
	act   string
}

var (
	script   []scriptStep
	shotPath string
)

func init() {
	if os.Getenv("MOTE_TOUCH") == "1" {
		isTouch = true
	}
	for _, part := range strings.Split(os.Getenv("MOTE_SCRIPT"), ",") {
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
		case "play":
			play()
		case "pause":
			openScreen(scPause)
		case "inv":
			openScreen(scInv)
		case "settings":
			openScreen(scSettings)
		case "time":
			worldTime, _ = strconv.ParseFloat(arg, 64)
		case "deep": // drop the mote into a cave far below
			teleportDeep()
		case "spawn":
			for k := 0; k < 6; k++ {
				ents = append(ents, &Entity{kind: entGloom, x: player.x + float64(30+k*14), y: player.y - 10, w: 8 * CELL, h: 6 * CELL, hp: 4, face: -1})
			}
		case "main":
			openScreen(scMain)
		case "seed":
			openScreen(scSeed)
		case "right":
			scriptRight = true
		case "lev":
			scriptLev = true
		case "stop":
			scriptRight, scriptLev = false, false
		case "select":
			selected, _ = strconv.Atoi(arg)
		case "give":
			addItem(arg, 200)
		case "aim": // aim=sx:sy, start using the selected item at that screen point
			a, b, _ := strings.Cut(arg, ":")
			aimSX, _ = strconv.ParseFloat(a, 64)
			aimSY, _ = strconv.ParseFloat(b, 64)
			useActive, scriptAim = true, true
		case "unaim":
			useActive, scriptAim = false, false
		case "dropone":
			dropSlot(selected, 1)
		case "log":
			n := 0
			for _, e := range ents {
				if e.kind == entItem {
					n++
				}
			}
			fmt.Printf("frame %d inv=%v drops=%d hp=%d player=(%.0f,%.0f) hits=%v\n", frame, invSummary(), n, player.hp, player.x, player.y, hits(player.x, player.y))
		case "bury": // pour sand on the mote
			cx, cy := int(player.x/CELL), int(player.y/CELL)
			for y := cy - 2; y < cy+8; y++ {
				for x := cx - 2; x < cx+12; x++ {
					grid[y*W+x] = SAND
				}
			}
		case "hurt":
			n, _ := strconv.Atoi(arg)
			player.invuln = 0
			hurtPlayer(n, 1)
		case "essence":
			n, _ := strconv.Atoi(arg)
			meta.Essence += n
		case "upgrades":
			upgradesFrom = scMain
			openScreen(scUpgrades)
		case "tpchest": // move next to the nearest chest (graves first, dungeons with arg=deep)
			teleportTo(func(t uint8, y int) bool { return t == CHEST && (arg != "deep" || y > H*55/100) })
		case "tpcore":
			shiftWindow(floorDiv(coreX(), CW) - NCH/2 - baseChunk)
			teleportTo(func(t uint8, y int) bool { return t == CORE })
		case "selectid":
			for i := 0; i < hotbarN; i++ {
				if inv[i] != nil && inv[i].ID == arg {
					selected = i
				}
			}
		case "openchest":
			for i, t := range grid {
				if t == CHEST {
					openChest(i%W, i/W)
					break
				}
			}
		case "aimcore":
			aimSX = (float64((coreX()-baseChunk*CW)*CELL) - cam.x) * zoom
			aimSY = (float64(coreY*CELL) - cam.y) * zoom
			useActive, scriptAim = true, true
		case "soil":
			for i, t := range grid {
				if t == SOIL {
					fmt.Printf("soil at %d,%d above=%d player cells x %d-%d y %d-%d aim=%d,%d\n", i%W, i/W, grid[i-W], px0, px1, py0, py1, aimX, aimY)
				}
			}
		case "aimfeet": // aim at the cell arg rows below the mote's feet, a few cells in front
			dy, _ := strconv.Atoi(arg)
			wx := player.x + player.w/2 + float64(player.face*16)
			wy := player.y + player.h + float64(dy*CELL) - 1
			aimSX, aimSY = (wx-cam.x)*zoom, (wy-cam.y)*zoom
			useActive, scriptAim = true, true
		case "around":
			aimCell()
			for y := aimY - 3; y <= aimY+3; y++ {
				row := ""
				for x := aimX - 6; x <= aimX+6; x++ {
					row += fmt.Sprintf("%3d", grid[y*W+x])
				}
				fmt.Println("around", y, row)
			}
		case "useonce":
			useOnce()
		case "grow": // ripen every crop in the loaded window
			for i, t := range grid {
				if t == CROP1 || t == CROP2 {
					grid[i] = CROP3
				}
			}
		case "count":
			c := map[uint8]int{}
			for _, t := range grid {
				c[t]++
			}
			fmt.Printf("frame %d blocks: plank=%d brick=%d glass=%d soil=%d crop1=%d crop3=%d furnace=%d chest=%d core=%d\n", frame, c[PLANK], c[BRICK], c[GLASS], c[SOIL], c[CROP1], c[CROP3], c[FURNACE], c[CHEST], c[CORE])
		case "spawninfo":
			x := int((player.x + player.w/2) / CELL)
			fmt.Printf("spawninfo x=%d top=%d safe=%v px=%.0f surface=%d\n", x, groundTop(x), safeSpawn(x), player.x/CELL, surfaceAt(x+baseChunk*CW)-0)
		case "ents": // list creatures near the mote
			for _, e := range ents {
				if e.kind != entItem {
					fmt.Printf("ent kind=%d at (%.0f,%.0f) hp=%d\n", e.kind, e.x/CELL, e.y/CELL, e.hp)
				}
			}
		case "state":
			fmt.Printf("frame %d screen=%d run=%+v essence=%d hp=%d pos=%.0f,%.0f\n", frame, cur, run, meta.Essence, player.hp, player.x/CELL, player.y/CELL)
		case "newrun":
			startRun(arg)
		case "shot":
			shotPath = arg
		case "quit":
			os.Exit(0)
		}
	}
}

var scriptRight, scriptLev, scriptAim bool

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

// WindowSizeOverride lets tests pick a window size with MOTE_SIZE=800x370.
func WindowSizeOverride() (int, int) {
	w, h, ok := strings.Cut(os.Getenv("MOTE_SIZE"), "x")
	if !ok {
		return 0, 0
	}
	a, _ := strconv.Atoi(w)
	b, _ := strconv.Atoi(h)
	return a, b
}

// Move the mote into the first open cave space well below the surface
func teleportDeep() {
	cx := int((player.x + player.w/2) / CELL)
	for dx := 0; dx < W/2; dx += 3 {
		for _, x := range []int{cx + dx, cx - dx} {
			if x < 8 || x >= W-8 {
				continue
			}
			for y := surfaceAt(baseChunk*CW+x) + 60; y < H-20; y++ {
				if boxFree(x-6, y-8, 12, 16) && y+9 < H && isSolid(grid[(y+9)*W+x]) {
					player.x, player.y = float64((x-5)*CELL), float64((y-6)*CELL)
					player.vx, player.vy = 0, 0
					updatePlayerCells()
					return
				}
			}
		}
	}
}

func invSummary() string {
	var b strings.Builder
	for _, st := range inv {
		if st != nil {
			fmt.Fprintf(&b, "%s:%d ", st.ID, st.N)
		}
	}
	return b.String()
}

// Put the mote in open space next to the first matching block in the loaded window
func teleportTo(match func(t uint8, y int) bool) {
	for y := 0; y < H; y++ {
		for x := 20; x < W-20; x++ {
			if !match(grid[y*W+x], y) {
				continue
			}
			for dx := 0; dx < 40; dx++ {
				for _, sx := range []int{x - dx, x + dx} {
					for sy := y - 16; sy < y+6; sy++ {
						if sx > 6 && sx < W-12 && sy > 0 && boxFree(sx, sy, 11, 14) {
							player.x, player.y = float64(sx*CELL), float64(sy*CELL)
							player.vx, player.vy = 0, 0
							updatePlayerCells()
							return
						}
					}
				}
			}
			return
		}
	}
}
