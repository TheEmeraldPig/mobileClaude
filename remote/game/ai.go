package game

import "math"

// Path finding for creatures. Every few frames a breadth-first search spreads out from
// the mote over a coarse grid of 4x4-cell blocks, giving each block its distance to the
// mote. A creature then steps toward the neighbouring block that is closer, so it walks
// around walls and through tunnels instead of pushing straight at the mote.
//
// Fliers can use any open block. Walkers can only climb onto blocks with ground a short
// jump below them, but can always drop down.

const (
	fbs        = 4  // block size in cells
	fgW        = 72 // field size in blocks (288 x 208 cells around the mote)
	fgH        = 52
	fieldEach  = 15 // frames between updates
	jumpBlocks = 4  // how high (in blocks) a walker can hop
)

var (
	flyDist, walkDist   = make([]int16, fgW*fgH), make([]int16, fgW*fgH)
	blkOpen, blkRoomy   = make([]bool, fgW*fgH), make([]bool, fgW*fgH)
	blkGround           = make([]bool, fgW*fgH)
	blkHeight           = make([]int, fgW*fgH)
	fieldX0, fieldY0    int // top-left of the field in local cells
	fieldBase, fieldAge = math.MinInt, 0
	bfsQueue            = make([]int32, 0, fgW*fgH)
)

// Recompute the fields when they are old or the loaded window has moved
func updateFields() {
	fieldAge++
	if fieldAge < fieldEach && fieldBase == baseChunk {
		return
	}
	fieldAge, fieldBase = 0, baseChunk
	pcx, pcy := int((player.x+player.w/2)/CELL), int((player.y+player.h/2)/CELL)
	fieldX0 = (pcx/fbs - fgW/2) * fbs
	fieldY0 = (pcy/fbs - fgH/2) * fbs
	for by := 0; by < fgH; by++ {
		for bx := 0; bx < fgW; bx++ {
			blkOpen[by*fgW+bx] = blockOpen(fieldX0+bx*fbs, fieldY0+by*fbs)
		}
	}
	open := func(bx, by int) bool {
		return bx >= 0 && by >= 0 && bx < fgW && by < fgH && blkOpen[by*fgW+bx]
	}
	for i := range blkRoomy {
		bx, by := i%fgW, i/fgW
		// room for a creature two blocks across: part of an open 2x2 square
		blkRoomy[i] = false
		for _, d := range [4][2]int{{0, 0}, {-1, 0}, {0, -1}, {-1, -1}} {
			x, y := bx+d[0], by+d[1]
			if open(x, y) && open(x+1, y) && open(x, y+1) && open(x+1, y+1) {
				blkRoomy[i] = true
				break
			}
		}
	}
	// how many blocks above something to stand on (bottom row up)
	for by := fgH - 1; by >= 0; by-- {
		for bx := 0; bx < fgW; bx++ {
			i := by*fgW + bx
			switch {
			case !blkOpen[i]:
				blkHeight[i] = 99
			case !open(bx, by+1):
				blkHeight[i] = 0
			default:
				blkHeight[i] = min(99, blkHeight[i+fgW]+1)
			}
		}
	}
	// within a jump of ground below
	for i := range blkGround {
		blkGround[i] = blkRoomy[i] && blkHeight[i] <= jumpBlocks
	}
	pbx, pby := (pcx-fieldX0)/fbs, (pcy-fieldY0)/fbs
	bfs(flyDist, pbx, pby, false)
	bfs(walkDist, pbx, pby, true)
}

// A block is open when none of its cells are solid or lava
func blockOpen(x0, y0 int) bool {
	for y := y0; y < y0+fbs; y++ {
		for x := x0; x < x0+fbs; x++ {
			if x < 0 || y < 0 || x >= W || y >= H {
				return false
			}
			if t := grid[y*W+x]; isSolid(t) || t == LAVA {
				return false
			}
		}
	}
	return true
}

func bfs(dist []int16, pbx, pby int, walk bool) {
	for i := range dist {
		dist[i] = -1
	}
	q := bfsQueue[:0]
	seed := func(bx, by int, d int16) {
		if bx >= 0 && by >= 0 && bx < fgW && by < fgH && dist[by*fgW+bx] < 0 {
			dist[by*fgW+bx] = d
			q = append(q, int32(by*fgW+bx))
		}
	}
	seed(pbx, pby, 0)
	if walk { // walkers aim for the ground under a levitating mote
		for by, d := pby+1, int16(1); by < fgH && by-pby < 20 && blkOpen[by*fgW+pbx]; by, d = by+1, d+1 {
			seed(pbx, by, d)
		}
	}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for h := 0; h < len(q); h++ {
		b := int(q[h])
		bx, by := b%fgW, b/fgW
		for _, d := range dirs {
			// a creature at A moves into B; we found B, so look for A
			ax, ay := bx+d[0], by+d[1]
			if ax < 0 || ay < 0 || ax >= fgW || ay >= fgH {
				continue
			}
			a := ay*fgW + ax
			if dist[a] >= 0 || !blkRoomy[a] {
				continue
			}
			if walk {
				switch {
				case by > ay: // dropping down is always possible
				case by == ay: // walking needs ground under A
					if !blkGround[a] {
						continue
					}
				default: // jumping up needs ground under both
					if !blkGround[a] || !blkGround[b] {
						continue
					}
				}
			}
			dist[a] = dist[b] + 1
			q = append(q, int32(a))
		}
	}
	bfsQueue = q
}

// Direction (unit vector) along the path toward the mote, from a point in local cells.
// ok is false when the mote is out of reach or too far along the path.
func pathDir(walk bool, cx, cy float64, maxSteps int) (dx, dy float64, steps int, ok bool) {
	dist := flyDist
	if walk {
		dist = walkDist
	}
	at := func(bx, by int) int {
		if bx < 0 || by < 0 || bx >= fgW || by >= fgH {
			return -1
		}
		return int(dist[by*fgW+bx])
	}
	bx, by := int(math.Floor((cx-float64(fieldX0))/fbs)), int(math.Floor((cy-float64(fieldY0))/fbs))
	own := at(bx, by)
	if own < 0 { // the centre may be just inside a wall; use the best block next to it
		best := -1
		for _, n := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if d := at(bx+n[0], by+n[1]); d >= 0 && (best < 0 || d < best) {
				best, bx, by = d, bx+n[0], by+n[1]
			}
		}
		own = best
	}
	if own < 0 || own > maxSteps {
		return 0, 0, own, false
	}
	tx, ty := bx, by
	bestD := own
	for _, n := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		d := at(bx+n[0], by+n[1])
		if d < 0 || d >= bestD {
			continue
		}
		if n[0] != 0 && n[1] != 0 && (at(bx+n[0], by) < 0 || at(bx, by+n[1]) < 0) {
			continue // don't cut corners
		}
		bestD, tx, ty = d, bx+n[0], by+n[1]
	}
	gx := float64(fieldX0+tx*fbs) + fbs/2
	gy := float64(fieldY0+ty*fbs) + fbs/2
	if bestD == 0 && own == 0 { // same block as the mote: go straight at it
		gx, gy = (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	}
	vx, vy := gx-cx, gy-cy
	l := math.Hypot(vx, vy)
	if l < 0.01 {
		return 0, 0, own, true
	}
	return vx / l, vy / l, own, true
}
