package game

import "math"

// Doors: a 3x14 block group. Closed doors are solid (creatures can't get through);
// tap or right-click to open or close one. Holding a pickaxe on a door takes it down
// as a single item.

const doorW, doorH = 3, 14

var doorHits int

func isDoor(t uint8) bool { return t == DOOR || t == DOOROPEN }

// All cells of the door touching (x, y)
func doorCells(x, y int) [][2]int {
	var out [][2]int
	seen := map[[2]int]bool{{x, y}: true}
	stack := [][2]int{{x, y}}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, c)
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if n[0] >= 0 && n[0] < W && n[1] >= 0 && n[1] < H && !seen[n] && isDoor(grid[n[1]*W+n[0]]) {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	return out
}

func toggleDoor(x, y int) {
	cells := doorCells(x, y)
	open := grid[y*W+x] == DOOR
	if !open { // don't close a door on the mote
		updatePlayerCells()
		for _, c := range cells {
			if inPlayer(c[0], c[1]) {
				return
			}
		}
	}
	for _, c := range cells {
		if open {
			grid[c[1]*W+c[0]] = DOOROPEN
		} else {
			grid[c[1]*W+c[0]] = DOOR
		}
		markDirty(c[0])
	}
	sfx("door")
}

func removeDoor(x, y int) {
	cells := doorCells(x, y)
	for _, c := range cells {
		grid[c[1]*W+c[0]] = EMPTY
		markDirty(c[0])
	}
	giveItem("door", 1)
	burst(10, float64(x), float64(y), blockRGB[DOOR], 1, 1, 20)
	sfx("dig")
}

// Where a door would go for an aim at (ax, ay): bottom-centre near the aim, on the ground
func doorSpot(ax, ay int) (int, int, bool) {
	best, bx, by := 1e9, 0, 0
	for dy := -10; dy <= 10; dy++ {
		for dx := -3; dx <= 3; dx++ {
			x0, y0 := ax-doorW/2+dx, ay-doorH+1+dy
			if d := math.Abs(float64(dy)) + math.Abs(float64(dx))*1.5; d < best && doorFits(x0, y0) {
				best, bx, by = d, x0, y0
			}
		}
	}
	return bx, by, best < 1e9
}

func doorFits(x0, y0 int) bool {
	if x0 < 1 || x0+doorW >= W-1 || y0 < 1 || y0+doorH >= H {
		return false
	}
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	if math.Hypot(float64(x0+1)-pcx, float64(y0+doorH/2)-pcy) > reach+6 {
		return false
	}
	updatePlayerCells()
	for y := y0; y < y0+doorH; y++ {
		for x := x0; x < x0+doorW; x++ {
			if !isReplaceable(grid[y*W+x]) || inPlayer(x, y) || objAt(x, y) != nil {
				return false
			}
		}
	}
	for x := x0; x < x0+doorW; x++ {
		if !isSolid(grid[(y0+doorH)*W+x]) {
			return false
		}
	}
	return true
}

func placeDoor(ax, ay int) bool {
	x0, y0, ok := doorSpot(ax, ay)
	if !ok {
		toast("A door needs a gap with a solid floor")
		return false
	}
	for y := y0; y < y0+doorH; y++ {
		for x := x0; x < x0+doorW; x++ {
			grid[y*W+x] = DOOR
			paint[y*W+x] = 0
			markDirty(x)
		}
	}
	takeSelected(1)
	sfx("build")
	return true
}

func drawDoorHologram(blend func(x, y int, c [3]int, a float64), ox, oy int) {
	if cur != scPlaying || selected < 0 || inv[selected] == nil || inv[selected].ID != "door" {
		return
	}
	ax, ay, _ := holoAim()
	x0, y0, ok := doorSpot(ax, ay)
	tint := [3]int{120, 255, 160}
	if !ok {
		x0, y0 = ax-doorW/2, ay-doorH+1
		tint = [3]int{255, 110, 110}
	}
	a := 0.45 + 0.12*math.Sin(float64(frame)*0.15)
	for y := y0; y < y0+doorH; y++ {
		for x := x0; x < x0+doorW; x++ {
			blend(x-ox, y-oy, mix(blockRGB[DOOR], tint, 0.45), a)
		}
	}
}

// Door looks: closed has boards, a frame and a handle; open is a faint frame
func doorPixel(t uint8, wx, wy int, bg [3]int, lf float64) [3]int {
	c := blockRGB[DOOR]
	left := wx > 0 && !isDoor(grid[wy*W+wx-1])
	right := wx < W-1 && !isDoor(grid[wy*W+wx+1])
	top := wy > 0 && !isDoor(grid[(wy-1)*W+wx])
	bottom := wy < H-1 && !isDoor(grid[(wy+1)*W+wx])
	edge := left || right || top || bottom
	if t == DOOROPEN {
		if edge {
			return mix(bg, shadeRGB(c, 0.6*lf), 0.7)
		}
		return mix(bg, shadeRGB(c, lf), 0.18)
	}
	switch {
	case edge:
		c = shadeRGB(c, 0.6)
	case wy%4 == 0:
		c = shadeRGB(c, 0.8)
	}
	// the handle, two-thirds of the way down on the right
	if !edge && wx+2 < W && !isDoor(grid[wy*W+wx+2]) && wy%14 == 8 {
		c = [3]int{255, 222, 120}
	}
	return shadeRGB(c, lf)
}
