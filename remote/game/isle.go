package game

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// The island you're on. Its terrain is built from its seed by the same code the
// server uses (remote/mmo), so only creatures and players come over the network.

var (
	curIsle   *mmo.Isle
	isleCache = map[string]*mmo.Isle{}
	skyTop    []int // per column: where the open sky ends (the first block that stops light, or the ground line if lower)
)

// Build (or reuse) an island and copy it into the grid the renderer draws
func loadIsle(id string) {
	d := mmo.IsleByID(id)
	if d == nil {
		d = mmo.Isles[0]
	}
	is := isleCache[d.ID]
	if is == nil {
		is = mmo.BuildIsle(d)
		isleCache[d.ID] = is
	}
	curIsle = is
	seedN = d.Seed
	copy(grid, is.Cells)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if grid[y*W+x] != EMPTY {
				shade[y*W+x] = texShade(x, y)
			} else {
				shade[y*W+x] = 0
			}
		}
	}
	skyTop = make([]int, W)
	for x := 0; x < W; x++ {
		y := 0
		for y < H && !blocksLight(grid[y*W+x]) && grid[y*W+x] != WATER {
			y++
		}
		skyTop[x] = max(y, is.SurfAt(x)) // and arches and overhangs on the ground keep their sky
	}
	parts = parts[:0]
	popups = popups[:0]
	ents = ents[:0]
}

// The row where the open sky ends at column x (caves begin under the first solid block)
func surfaceAt(x int) int {
	if skyTop == nil {
		return 300
	}
	return skyTop[max(0, min(W-1, x))]
}

func spawnPlayerAt(x, y float64) {
	player.x, player.y = x, y
	player.vx, player.vy = 0, 0
	player.energy, player.squash = 1, 0
	unstick()
	updatePlayerCells()
	placeCamera()
}

// Keep the view on the island: its ends are at the screen's edges, never beyond
func clampCamera() {
	vw := logicalW / zoom
	maxX := float64(W*CELL) - vw
	cam.x = math.Max(0, math.Min(maxX, cam.x))
	cam.y = math.Min(float64(H*CELL)-logicalH/zoom, cam.y)
}
