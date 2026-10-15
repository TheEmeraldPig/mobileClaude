package game

import "math/rand"

// Sand and water simulation, only around the camera.

func getCell(x, y int) uint8 {
	if x < 0 || x >= W || y < 0 || y >= H || inPlayer(x, y) {
		return STONE
	}
	return grid[y*W+x]
}

func swapCells(a, b int) {
	grid[a], grid[b] = grid[b], grid[a]
	shade[a], shade[b] = shade[b], shade[a]
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
			switch grid[idx] {
			case SAND:
				below := getCell(x, y+1)
				if below == EMPTY || below == WATER {
					swapCells(idx, idx+W)
					continue
				}
				d := 1
				if rand.Intn(2) == 0 {
					d = -1
				}
				for _, dx := range [2]int{d, -d} {
					n := getCell(x+dx, y+1)
					if n == EMPTY || n == WATER {
						swapCells(idx, idx+W+dx)
						break
					}
				}
			case WATER:
				if getCell(x, y+1) == EMPTY {
					swapCells(idx, idx+W)
					continue
				}
				d := 1
				if rand.Intn(2) == 0 {
					d = -1
				}
				moved := false
				for _, dx := range [2]int{d, -d} {
					if getCell(x+dx, y+1) == EMPTY {
						swapCells(idx, idx+W+dx)
						moved = true
						break
					}
				}
				if moved {
					continue
				}
				for _, dx := range [2]int{d, -d} {
					if getCell(x+dx, y) == EMPTY {
						swapCells(idx, idx+dx)
						break
					}
				}
			}
		}
	}
}
