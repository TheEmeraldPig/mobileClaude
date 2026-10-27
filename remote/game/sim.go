package game

import "math/rand"

// Sand, water and lava simulation, plus crops growing, only around the camera.

func getCell(x, y int) uint8 {
	if x < 0 || x >= W || y < 0 || y >= H {
		return STONE
	}
	return grid[y*W+x]
}

func swapCells(a, b int) {
	grid[a], grid[b] = grid[b], grid[a]
	shade[a], shade[b] = shade[b], shade[a]
}

// Lava next to water cools into obsidian (and the water boils away)
func quench(idx, x, y int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if getCell(x+d[0], y+d[1]) == WATER {
			grid[idx] = OBSIDIAN
			grid[(y+d[1])*W+x+d[0]] = EMPTY
			burst(4, float64(x), float64(y), [3]int{220, 220, 230}, 0.6, 1.2, 20)
			return true
		}
	}
	return false
}

func flow(idx, x, y int, t uint8) {
	if getCell(x, y+1) == EMPTY {
		swapCells(idx, idx+W)
		return
	}
	d := 1
	if rand.Intn(2) == 0 {
		d = -1
	}
	for _, dx := range [2]int{d, -d} {
		if getCell(x+dx, y+1) == EMPTY {
			swapCells(idx, idx+W+dx)
			return
		}
	}
	for _, dx := range [2]int{d, -d} {
		if getCell(x+dx, y) == EMPTY {
			swapCells(idx, idx+dx)
			return
		}
	}
}

func stepSim() {
	const m = 60
	x0 := max(0, int(cam.x/CELL)-m)
	x1 := min(W-1, int(cam.x/CELL)+viewW+m)
	y0 := max(0, int(cam.y/CELL)-m)
	y1 := min(H-2, int(cam.y/CELL)+viewH+m)
	if x1 < x0 || y1 < y0 {
		return
	}
	// simulated chunks get saved on unload
	for c := x0 / CW; c <= x1/CW && c < NCH; c++ {
		dirty[c] = true
	}
	for y := y1; y >= y0; y-- {
		ltr := rand.Intn(2) == 0
		for i := x0; i <= x1; i++ {
			x := i
			if !ltr {
				x = x1 - (i - x0)
			}
			idx := y*W + x
			switch t := grid[idx]; t {
			case SAND:
				below := getCell(x, y+1)
				if below == EMPTY || isLiquid(below) {
					swapCells(idx, idx+W)
					continue
				}
				d := 1
				if rand.Intn(2) == 0 {
					d = -1
				}
				for _, dx := range [2]int{d, -d} {
					n := getCell(x+dx, y+1)
					if n == EMPTY || isLiquid(n) {
						swapCells(idx, idx+W+dx)
						break
					}
				}
			case WATER:
				flow(idx, x, y, t)
			case LAVA:
				if quench(idx, x, y) {
					continue
				}
				if frame%3 == 0 { // lava is thick and slow
					flow(idx, x, y, t)
				}
			case CROP1, CROP2:
				// crops grow slowly (about a minute per stage) on tilled soil
				if getCell(x, y+1) == SOIL && rand.Intn(3600) == 0 {
					grid[idx]++
				}
			}
		}
	}
}
