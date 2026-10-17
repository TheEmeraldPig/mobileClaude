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
		case "arena": // a closed room with a wall between the mote and its right side, and a tunnel under the wall
			px, py := int(player.x/CELL), int((player.y+player.h)/CELL)
			for y := py - 60; y < py+40; y++ {
				for x := px - 30; x < px+110; x++ {
					t := uint8(STONE)
					if y < py && x > px-28 && x < px+108 && y > py-58 {
						t = EMPTY
					}
					grid[y*W+x] = t
				}
			}
			for y := py - 45; y < py; y++ { // the wall
				for x := px + 40; x < px+46; x++ {
					grid[y*W+x] = STONE
				}
			}
			dig := func(x0, y0, w, h int) {
				for y := y0; y < y0+h; y++ {
					for x := x0; x < x0+w; x++ {
						grid[y*W+x] = EMPTY
					}
				}
			}
			dig(px+56, py, 12, 8) // shaft down on the right
			dig(px+18, py+8, 50, 8)
			dig(px+18, py, 12, 8) // shaft up on the left
			for i := range dirty {
				dirty[i] = true
			}
		case "foe": // foe=kind, spawned to the right of the mote
			k, _ := strconv.Atoi(arg)
			e := newFoe(entKind(k), int(player.x/CELL)+80, int(player.y/CELL)-4)
			ents = append(ents, e)
		case "spawn":
			for k := 0; k < 6; k++ {
				ents = append(ents, &Entity{kind: entGloom, x: player.x + float64(30+k*14), y: player.y - 10, w: 8 * CELL, h: 6 * CELL, hp: 4, face: -1})
			}
		case "main":
			openScreen(scMain)
		case "seed":
			startNewWorldScreen()
		case "right":
			scriptRight = true
		case "lev":
			scriptLev = true
		case "stop":
			scriptRight, scriptLev = false, false
		case "select":
			selected, _ = strconv.Atoi(arg)
		case "give":
			if items[arg] != nil && items[arg].tool {
				addItem(arg, 1)
			} else {
				addItem(arg, 200)
			}
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
		case "blockme": // fill the lower half of the mote with stone
			updatePlayerCells()
			for y := (py0 + py1) / 2; y <= py1; y++ {
				for x := px0; x <= px1; x++ {
					grid[y*W+x] = STONE
				}
			}
		case "god": // lots of health, for watching creatures
			player.maxHP, player.hp = 999, 999
		case "hurt":
			n, _ := strconv.Atoi(arg)
			player.invuln = 0
			hurtPlayer(n, 1)
		case "craft": // craft=STATION (or "hand") opens that crafting menu
			if arg == "hand" {
				openHandCrafting()
			} else {
				openStation(arg)
			}
		case "worlds":
			openScreen(scWorlds)
		case "die":
			player.invuln = 0
			hurtPlayer(999, 1)
		case "respawn":
			respawn()
		case "mode": // mode=N for the current world
			world.Mode, _ = strconv.Atoi(arg)
		case "station": // station=KIND placed where the mote stands
			lx := int((player.x+player.w)/CELL) + 4
			ty := int((player.y+player.h)/CELL) - len(stations[arg].sprite)
			objs = append(objs, &Obj{kind: arg, wx: lx + baseChunk*CW, y: ty})
		case "shape": // shape=N
			buildShape, _ = strconv.Atoi(arg)
		case "drag": // drag=x0:y0:x1:y1 in screen px, building the held item's shape
			var v [4]float64
			fmt.Sscanf(strings.ReplaceAll(arg, ":", " "), "%f %f %f %f", &v[0], &v[1], &v[2], &v[3])
			aimSX, aimSY = v[0], v[1]
			beginAction()
			startShape()
			aimSX, aimSY = v[2], v[3]
			finishShape()
			endAction()
		case "paint": // paint=N colour
			paintIdx, _ = strconv.Atoi(arg)
		case "undo":
			undoBuild()
		case "rot":
			buildRot = (buildRot + 1) % 4
		case "tpchest": // move next to the nearest chest (graves first, dungeons with arg=deep)
			teleportNear(func(t uint8, y int) bool { return t == CHEST && (arg != "deep" || y > H*55/100) })
		case "tpcore":
			shiftWindow(floorDiv(coreX(), CW) - NCH/2 - baseChunk)
			teleportNear(func(t uint8, y int) bool { return t == CORE })
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
		case "fitdbg":
			px, py := int((player.x+player.w/2)/CELL), int((player.y+player.h)/CELL)
			sp := stations["workbench"].sprite
			w, h := len(sp[0]), len(sp)
			for _, dx := range []int{-30, -15, 10, 20} {
				lx, ty := px+dx, py-h
				free, sup := true, 0
				for yy := 0; yy < h; yy++ {
					for xx := 0; xx < w; xx++ {
						if t := grid[(ty+yy)*W+lx+xx]; isSolid(t) || isLiquid(t) {
							free = false
						}
					}
				}
				for xx := 0; xx < w; xx++ {
					if isSolid(grid[(ty+h)*W+lx+xx]) {
						sup++
					}
				}
				_, _, ok := settleStation("workbench", lx, ty, false)
				fmt.Printf("fit dx=%d free=%v sup=%d/%d fits=%v settle=%v\n", dx, free, sup, w, stationFits("workbench", lx, ty, false), ok)
			}
		case "tapat": // tapat=sx:sy, a touch/click on the world at that screen point
			a, b, _ := strings.Cut(arg, ":")
			x, _ := strconv.ParseFloat(a, 64)
			y, _ := strconv.ParseFloat(b, 64)
			startWorldPointer(77, x, y)
			endWorldPointer(77)
		case "lightdbg":
			pcx, pcy := int((player.x+player.w/2)/CELL), int((player.y+player.h/2)/CELL)
			for dy := -12; dy <= 12; dy += 3 {
				row := ""
				for dx := -12; dx <= 12; dx += 3 {
					x, y := pcx+dx, pcy+dy
					row += fmt.Sprintf("%2d/%d/%2d ", lightAt(x, y), grid[y*W+x], lightSrc[(y-lightY0)*lightW+x-lightX0])
				}
				fmt.Println("L", row)
			}
			if lightImg != nil {
				vw := lightImg.Bounds().Dx()
				vh := lightImg.Bounds().Dy()
				for dy := -12; dy <= 12; dy += 6 {
					row := ""
					for dx := -12; dx <= 12; dx += 6 {
						o := ((vh/2+dy)*vw + vw/2 + dx) * 4
						row += fmt.Sprintf("%3d,%3d,%3d|%3d,%3d,%3d,%3d ", lightPix[o], lightPix[o+1], lightPix[o+2], worldPix[o], worldPix[o+1], worldPix[o+2], emitPix[o+3])
					}
					fmt.Println("P", row)
				}
			}
		case "tpstruct": // tpstruct=village|sky|ruin|pyramid: go to the nearest one
			wx0 := baseChunk*CW + int(player.x/CELL)
			for d := 0; d < 40; d++ {
				for _, s := range []int{d, -d - 1} {
					switch arg {
					case "village":
						if hs := villageHouses(floorDiv(wx0, villageSlot) + s); hs != nil {
							teleportTo(hs[0].x + hs[0].w/2)
							return
						}
					case "sky":
						if is, ok := skyIslandAt(floorDiv(wx0, skySlot) + s); ok {
							teleportTo(is.cx)
							player.y = float64((is.top - 14) * CELL)
							player.x = float64((is.cx-12-baseChunk*CW)*CELL) - player.w/2
							return
						}
					case "ruin":
						sl := floorDiv(wx0, ruinSlot) + s
						if hash01(sl, 0, 211) <= 0.55 {
							w := 90 + int(hash01(sl, 1, 211)*70)
							h := 44 + int(hash01(sl, 2, 211)*22)
							rx := sl*ruinSlot + 40 + int(hash01(sl, 3, 211)*float64(ruinSlot-80-w))
							ry := int(H*0.52 + hash01(sl, 4, 211)*H*0.18)
							teleportTo(rx + w/2)
							player.y = float64((ry + h - 3 - 14) * CELL)
							player.x = float64((rx+w/2-baseChunk*CW)*CELL)
							return
						}
					case "pyramid":
						sl := floorDiv(wx0, pyramidSlot) + s
						px := sl*pyramidSlot + 100 + int(hash01(sl, 1, 221)*float64(pyramidSlot-200))
						if hash01(sl, 0, 221) <= 0.7 && isDesert(px) && isDesert(px-50) && isDesert(px+50) {
							teleportTo(px - 70)
							return
						}
					}
				}
			}
		case "shop":
			k, _ := strconv.Atoi(arg)
			openShop(k)
		case "useonce":
			useOnce()
			fmt.Printf("useonce sel=%d toast=%q objs=%d aim=%.0f,%.0f\n", selected, toastText, len(objs), aimSX, aimSY)
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
					ki := kinds[e.kind]
					px, py, steps, ok := pathDir(!ki.fly, (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL, 999)
					fmt.Printf("ent kind=%d at (%.0f,%.0f) v=(%.2f,%.2f) hp=%d aware=%d path=(%.2f,%.2f) steps=%d ok=%v mote=(%.0f,%.0f) mhp=%d\n", e.kind, e.x/CELL, e.y/CELL, e.vx, e.vy, e.hp, e.aware, px, py, steps, ok, player.x/CELL, player.y/CELL, player.hp)
				}
			}
		case "state":
			fmt.Printf("frame %d screen=%d world=%q mode=%d hp=%d/%d pos=%.0f,%.0f objs=%d inv=%s\n", frame, cur, world.Name, world.Mode, player.hp, player.maxHP, player.x/CELL, player.y/CELL, len(objs), invSummary())
		case "newrun", "newworld": // newworld=SEED[:MODE]
			seed, m, _ := strings.Cut(arg, ":")
			mode := modeNormal
			if m != "" {
				mode, _ = strconv.Atoi(m)
			}
			createNewWorld("Test world", seed, mode)
		case "shot":
			shotPath = arg
		case "save":
			saveGame()
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
func teleportNear(match func(t uint8, y int) bool) {
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
