package game

import "math"

// Big natural structures: sky islands, underground ruins, desert pyramids and surface
// villages (where traders live). Like every generation pass they follow the rules at the
// top of world.go: everything comes from the seed and world position, each structure is
// anchored in a "slot" of world columns, and a chunk builds every structure that might
// reach into it but only writes its own cells (set() clips to the chunk).

// ---- Background walls ----
// Structures also give themselves back walls. genWalls runs the structure passes again
// with genWall set, on a scratch copy of the blocks, and keeps only the walls.

var genWall []uint8

func setW(x0, x, y int, t uint8) {
	if genWall != nil {
		set(genWall, x0, x, y, t)
	}
}

func genWalls(cx int) []uint8 {
	genWall = make([]uint8, CW*H)
	scratch := make([]uint8, CW*H)
	x0 := cx * CW
	for _, f := range []genPass{placeDungeons, placeRuins, placePyramids, placeSkyIslands, placeVillages} {
		f(scratch, cx, x0)
	}
	w := genWall
	genWall = nil
	return w
}

// ---- Sky islands ----

const skySlot = 520

type skyIsland struct {
	cx, top, w, h int // centre column, top row of the grass, width, depth
	house         bool
}

func skyIslandAt(sl int) (skyIsland, bool) {
	if hash01(sl, 0, 201) > 0.5 {
		return skyIsland{}, false
	}
	w := 70 + int(hash01(sl, 1, 201)*70)
	is := skyIsland{
		cx:    sl*skySlot + 80 + int(hash01(sl, 2, 201)*float64(skySlot-160)),
		top:   36 + int(hash01(sl, 3, 201)*50),
		w:     w,
		h:     16 + int(hash01(sl, 4, 201)*14),
		house: hash01(sl, 5, 201) < 0.7,
	}
	// keep clear of tall mountains
	if surfaceAt(is.cx) < is.top+is.h+40 {
		return skyIsland{}, false
	}
	return is, true
}

func placeSkyIslands(buf []uint8, cx, x0 int) {
	for sl := floorDiv(x0-200, skySlot); sl <= floorDiv(x0+CW+200, skySlot); sl++ {
		is, ok := skyIslandAt(sl)
		if !ok {
			continue
		}
		half := is.w / 2
		for x := is.cx - half; x <= is.cx+half; x++ {
			if x < x0 || x >= x0+CW {
				continue
			}
			u := float64(x-is.cx) / float64(half) // -1..1
			// a rounded top and a craggy, tapering underside
			top := is.top + int(math.Round(u*u*4+noise1(float64(x), 9, 202)*1.5))
			depth := float64(is.h) * (1 - u*u) * (0.75 + 0.35*(noise1(float64(x), 7, 203)+1)/2)
			bottom := is.top + int(depth)
			for y := top; y <= bottom; y++ {
				switch {
				case y == top:
					set(buf, x0, x, y, GRASS)
				case y < top+4:
					set(buf, x0, x, y, DIRT)
				default:
					t := uint8(STONE)
					if hash01(x, y, 204) < 0.05 {
						t = GOLD
					} else if hash01(x, y, 205) < 0.04 {
						t = CRYSTAL
					}
					set(buf, x0, x, y, t)
				}
			}
		}
		// a few trees
		for k := 0; k < 3; k++ {
			tx := is.cx - half/2 + int(hash01(sl, 10+k, 206)*float64(half))
			if is.house && math.Abs(float64(tx-is.cx)) < 18 {
				continue
			}
			s := is.top + int(math.Round(math.Pow(float64(tx-is.cx)/float64(half), 2)*4))
			for y := s - 9; y < s; y++ {
				set(buf, x0, tx, y, WOOD)
			}
			for dy := -4; dy <= 4; dy++ {
				for dx := -5; dx <= 5; dx++ {
					if dx*dx+dy*dy*2 <= 18 && get(buf, x0, tx+dx, s-9+dy) == EMPTY {
						set(buf, x0, tx+dx, s-9+dy, LEAVES)
					}
				}
			}
		}
		if is.house { // a little shrine with a chest of sky loot
			hx, hy := is.cx-11, is.top
			for y := hy - 18; y < hy; y++ {
				for x := hx; x < hx+22; x++ {
					edge := x < hx+2 || x >= hx+20 || y < hy-16
					switch {
					case y < hy-16 && (x-hx+(hy-y))%3 == 0:
						set(buf, x0, x, y, BRICK)
					case edge && y >= hy-12 && y < hy-1 && (x < hx+2 || x >= hx+20):
						set(buf, x0, x, y, EMPTY) // open sides
					case edge:
						set(buf, x0, x, y, GLASS)
					default:
						set(buf, x0, x, y, EMPTY)
						setW(x0, x, y, PLANK)
					}
				}
			}
			for x := hx - 1; x < hx+23; x++ { // roof
				set(buf, x0, x, hy-19, BRICK)
				set(buf, x0, x, hy-18, BRICK)
			}
			set(buf, x0, hx+6, hy-14, TORCH)
			set(buf, x0, hx+15, hy-14, TORCH)
			chestAt(buf, x0, hx+10, hy-2)
		}
	}
}

// ---- Underground ruins ----

const ruinSlot = 640

func placeRuins(buf []uint8, cx, x0 int) {
	for sl := floorDiv(x0-220, ruinSlot); sl <= floorDiv(x0+CW+220, ruinSlot); sl++ {
		if hash01(sl, 0, 211) > 0.55 {
			continue
		}
		w := 90 + int(hash01(sl, 1, 211)*70)
		h := 44 + int(hash01(sl, 2, 211)*22)
		rx := sl*ruinSlot + 40 + int(hash01(sl, 3, 211)*float64(ruinSlot-80-w))
		ry := int(H*0.52 + hash01(sl, 4, 211)*H*0.18)
		if rx+w < x0-2 || rx > x0+CW+2 {
			continue
		}
		floors := []int{ry + h/3, ry + 2*h/3}
		for y := ry; y < ry+h; y++ {
			for x := rx; x < rx+w; x++ {
				shell := x < rx+3 || x >= rx+w-3 || y < ry+3 || y >= ry+h-3
				floor := false
				for _, f := range floors {
					// floors with a gap to climb through, offset per floor
					gap := rx + w/4 + (f-ry)*7%(w/2)
					if y >= f && y < f+2 && (x < gap || x > gap+10) {
						floor = true
					}
				}
				pillar := (x-rx)%22 < 2 && x > rx+6 && x < rx+w-6
				decay := hash01(x, y, 212) < 0.06 || noise2(float64(x), float64(y), 9, 213) > 0.78
				switch {
				case (shell || floor || pillar) && !decay:
					t := uint8(BRICK)
					if hash01(x, y, 214) < 0.3 {
						t = STONE // old, patched with stone
					}
					set(buf, x0, x, y, t)
				default:
					set(buf, x0, x, y, EMPTY)
					if !shell && hash01(x, y, 217) > 0.08 && noise2(float64(x), float64(y), 12, 218) < 0.72 {
						setW(x0, x, y, BRICK) // crumbling back walls
					}
				}
			}
		}
		// entrances on both sides of the ground floor
		for y := ry + h - 3 - 15; y < ry+h-3; y++ {
			for d := 0; d < 3; d++ {
				set(buf, x0, rx+d, y, EMPTY)
				set(buf, x0, rx+w-1-d, y, EMPTY)
			}
		}
		// torches and chests on each level
		levels := []int{floors[0] - 1, floors[1] - 1, ry + h - 4}
		for k, ly := range levels {
			for j := 0; j < 3; j++ {
				tx := rx + 10 + int(hash01(sl, 20+k*3+j, 215)*float64(w-20))
				set(buf, x0, tx, ly-8, TORCH)
			}
			if hash01(sl, 30+k, 216) < 0.75 {
				chx := rx + 8 + int(hash01(sl, 40+k, 216)*float64(w-20))
				chestAt(buf, x0, chx, ly-1)
			}
		}
	}
}

// ---- Desert pyramids ----

const pyramidSlot = 700

func placePyramids(buf []uint8, cx, x0 int) {
	for sl := floorDiv(x0-120, pyramidSlot); sl <= floorDiv(x0+CW+120, pyramidSlot); sl++ {
		px := sl*pyramidSlot + 100 + int(hash01(sl, 1, 221)*float64(pyramidSlot-200))
		if hash01(sl, 0, 221) > 0.7 || !isDesert(px) || !isDesert(px-50) || !isDesert(px+50) {
			continue
		}
		base := surfaceAt(px) + 4
		half := 50
		if px+half < x0 || px-half > x0+CW {
			continue
		}
		for dy := 0; dy <= half; dy++ {
			y := base - dy
			for x := px - half + dy; x <= px+half-dy; x++ {
				set(buf, x0, x, y, SAND)
				if (x+y)%6 == 0 || dy%5 == 0 {
					set(buf, x0, x, y, BRICK)
				}
			}
			// fill the ground below the base edge so it doesn't float
			if dy == 0 {
				for x := px - half; x <= px+half; x++ {
					for y2 := base + 1; y2 < surfaceAt(x)+8; y2++ {
						set(buf, x0, x, y2, SAND)
					}
				}
			}
		}
		// a corridor in from the left and a treasure room
		for y := base - 14; y < base-2; y++ {
			for x := px - half; x < px+6; x++ {
				set(buf, x0, x, y, EMPTY)
			}
		}
		for y := base - 24; y < base-2; y++ {
			for x := px - 10; x < px+12; x++ {
				set(buf, x0, x, y, EMPTY)
				setW(x0, x, y, BRICK)
			}
		}
		for y := base - 14; y < base-2; y++ {
			for x := px - half + 8; x < px-10; x++ {
				setW(x0, x, y, BRICK)
			}
		}
		set(buf, x0, px-6, base-18, TORCH)
		set(buf, x0, px+8, base-18, TORCH)
		chestAt(buf, x0, px-1, base-4)
		chestAt(buf, x0, px+5, base-4)
	}
}

// ---- Villages ----

const villageSlot = 820

type vHouse struct {
	x, floor, w, h int // left column, floor row (the row the trader stands on), size
	kind           int // trader: 0 merchant, 1 smith, 2 farmer
	id             int
}

// The houses of the village in slot sl (empty if there is none)
func villageHouses(sl int) []vHouse {
	if hash01(sl, 0, 231) > 0.42 {
		return nil
	}
	start := sl*villageSlot + 60 + int(hash01(sl, 1, 231)*float64(villageSlot-260))
	n := 3 + int(hash01(sl, 2, 231)*2)
	var hs []vHouse
	x := start
	for k := 0; k < n; k++ {
		w := 26 + int(hash01(sl, 10+k, 232)*8)
		mid := x + w/2
		s := surfaceAt(mid)
		if isDesert(mid) || s >= waterLevel-4 {
			return nil
		}
		hs = append(hs, vHouse{x: x, floor: s - 1, w: w, h: 22, kind: k % 3, id: sl*8 + k})
		x += w + 10 + int(hash01(sl, 20+k, 232)*14)
	}
	return hs
}

func placeVillages(buf []uint8, cx, x0 int) {
	for sl := floorDiv(x0-260, villageSlot); sl <= floorDiv(x0+CW+260, villageSlot); sl++ {
		for _, hs := range villageHouses(sl) {
			if hs.x+hs.w+6 < x0 || hs.x-6 > x0+CW {
				continue
			}
			buildHouse(buf, x0, hs)
		}
	}
}

func buildHouse(buf []uint8, x0 int, hs vHouse) {
	fl := hs.floor
	wallT, roofT := uint8(PLANK), uint8(BRICK)
	if hs.kind == 1 {
		wallT = STONE // the smith's house is stone
	}
	// flatten: solid ground under the house, clear air above it
	for x := hs.x - 3; x < hs.x+hs.w+3; x++ {
		for y := fl + 1; y < surfaceAt(x)+6; y++ {
			t := uint8(DIRT)
			if y == fl+1 && (x < hs.x || x >= hs.x+hs.w) {
				t = GRASS
			}
			set(buf, x0, x, y, t)
		}
		for y := fl - hs.h - 8; y <= fl; y++ {
			set(buf, x0, x, y, EMPTY)
		}
	}
	top := fl - hs.h
	for y := top; y <= fl+1; y++ {
		for x := hs.x; x < hs.x+hs.w; x++ {
			wallCell := x < hs.x+2 || x >= hs.x+hs.w-2
			switch {
			case y == fl+1:
				set(buf, x0, x, y, roofT) // floor
			case y <= top+1:
				set(buf, x0, x, y, roofT)
			case wallCell && y > fl-15 && (x < hs.x+2) == (hs.id%2 == 0):
				set(buf, x0, x, y, DOOR) // a door on one side
			case wallCell && y > top+5 && y < top+9:
				set(buf, x0, x, y, GLASS)
			case wallCell:
				set(buf, x0, x, y, wallT)
			}
			if y > top+1 && y <= fl {
				setW(x0, x, y, wallT) // back wall
			}
		}
	}
	// a pitched roof
	for k := 1; k <= 6; k++ {
		for x := hs.x - 1 + k; x < hs.x+hs.w+1-k; x++ {
			set(buf, x0, x, top-k+1, roofT)
		}
	}
	set(buf, x0, hs.x+hs.w/2, top+4, TORCH)
}

// Traders standing in village houses inside the loaded window
func loadedHouses() []vHouse {
	var out []vHouse
	wx0, wx1 := baseChunk*CW, (baseChunk+NCH)*CW
	for sl := floorDiv(wx0-260, villageSlot); sl <= floorDiv(wx1+260, villageSlot); sl++ {
		for _, hs := range villageHouses(sl) {
			if hs.x >= wx0 && hs.x+hs.w <= wx1 {
				out = append(out, hs)
			}
		}
	}
	return out
}
