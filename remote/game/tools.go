package game

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// The sword and the pickaxe. On a PC you aim with the mouse; on a phone the sword goes
// for the nearest creature and the pickaxe for the nearest block you can mine (or where
// you touch the world). The server decides what's hit and what's mined.

var (
	myTool     = mmo.ToolSword
	useSwing   = -1.0 // < 0: not using the tool; 0..1 through a swing
	useAim     float64
	swingCD    int // frames until the sword can swing again
	mineX      = -1
	mineY      = -1
	mineProg   float64 // 0..1 dug into (mineX, mineY)
	noMineWarn int
	touchAimOn bool    // a finger on the world aims the pickaxe (phones)
	touchAimX  float64 // screen px
	touchAimY  float64
)

func playerCenter() (float64, float64) { return player.x + player.w/2, player.y + player.h/2 }

// The world point being aimed at, in world px
func aimPoint() (float64, float64, bool) {
	switch {
	case scriptAimOn:
		cx, cy := playerCenter()
		return cx + scriptAimDX, cy + scriptAimDY, true
	case touchAimOn:
		return cam.x + touchAimX/zoom, cam.y + touchAimY/zoom, true
	case !isTouch:
		return cam.x + hoverX/zoom, cam.y + hoverY/zoom, true
	}
	return 0, 0, false
}

func selectTool(t int) {
	if myTool != t {
		myTool = t
		mineX, mineProg = -1, 0
		sfx("click")
	}
}

// Every frame: use the tool in hand while the use button (or mouse) is held
func updateTool(using bool) {
	if swingCD > 0 {
		swingCD--
	}
	if useSwing >= 0 {
		useSwing += 1.0 / 14
		if useSwing > 1 {
			useSwing = -1
		}
	}
	if cur != scPlaying || me == nil {
		mineX, mineProg = -1, 0
		return
	}
	cx, cy := playerCenter()
	if myTool == mmo.ToolSword {
		mineX = -1
		if !using || swingCD > 0 {
			return
		}
		aim := 0.0
		if player.face < 0 {
			aim = math.Pi
		}
		if ax, ay, ok := aimPoint(); ok {
			aim = math.Atan2(ay-cy, ax-cx)
		} else if e := nearestMob(mmo.SwordReach * 1.6); e != nil {
			aim = math.Atan2(e.y+e.h/2-cy, e.x+e.w/2-cx)
		}
		if math.Cos(aim) < 0 {
			player.face = -1
		} else {
			player.face = 1
		}
		useAim, useSwing, swingCD = aim, 0, int(mmo.SwordEvery*60)
		send(mmo.CMsg{T: "swing", Aim: aim})
		sfx("swing")
		return
	}
	// pickaxe
	if !using {
		mineX, mineProg = -1, 0
		return
	}
	d := curIsle.Def
	if d.Mine == "" {
		if frame-noMineWarn > 240 {
			noMineWarn = frame
			toast(d.Name + " can't be mined. Try a mining island like Stonecrop")
		}
		return
	}
	x, y, ok := mineTarget()
	if !ok {
		mineX, mineProg = -1, 0
		return
	}
	if x != mineX || y != mineY {
		mineX, mineY, mineProg = x, y, 0
	}
	yd, _ := d.YieldAt(grid[y*W+x])
	useAim = math.Atan2(float64(y*CELL+1)-cy, float64(x*CELL+1)-cx)
	if math.Cos(useAim) < 0 {
		player.face = -1
	} else {
		player.face = 1
	}
	if useSwing < 0 {
		useSwing = 0
		sfx("dig")
	}
	mineProg += 1 / (yd.Time * 60)
	if frame%8 == 0 {
		c := blockRGB[grid[y*W+x]]
		burst(2, float64(x)+0.5, float64(y)+0.5, c, 0.8, 0.6, 16)
	}
	if mineProg >= 1 {
		dug(x, y, yd)
	}
}

// A block was dug out: it turns to its worn block straight away here (the server tells
// everyone, and puts it back if it disagrees)
func dug(x, y int, yd mmo.Yield) {
	send(mmo.CMsg{T: "mine", CX: x, CY: y})
	c := blockRGB[grid[y*W+x]]
	mmo.Brush(func(dx, dy int) {
		bx, by := x+dx, y+dy
		if bx >= 0 && by >= 0 && bx < W && by < H {
			if b, ok := curIsle.Def.YieldAt(grid[by*W+bx]); ok {
				setCell(bx, by, b.Becomes)
			}
		}
	})
	burst(10, float64(x)+0.5, float64(y)+0.5, c, 1.4, 1.2, 24)
	if r := mmo.ResourceByID(yd.Item); r != nil {
		popText("+1 "+r.Name, float64(x*CELL), float64(y*CELL)-6, colInk)
	}
	sfx("pickup")
	mineX, mineProg = -1, 0
}

// The block to mine: the one aimed at, if it's in reach and minable; on a phone with no
// finger on the world, the nearest minable block in front of the character
func mineTarget() (int, int, bool) {
	cx, cy := playerCenter()
	d := curIsle.Def
	ok := func(x, y int) bool {
		if x < 0 || y < 0 || x >= W || y >= H {
			return false
		}
		if _, can := d.YieldAt(grid[y*W+x]); !can {
			return false
		}
		return math.Hypot(float64(x*CELL+1)-cx, float64(y*CELL+1)-cy) <= mmo.MineReach
	}
	if ax, ay, aimed := aimPoint(); aimed {
		x, y := int(math.Floor(ax/CELL)), int(math.Floor(ay/CELL))
		return x, y, ok(x, y)
	}
	// nearest exposed block, preferring the side the character faces and its feet
	best, bx, by := math.Inf(1), 0, 0
	px, py := int(cx/CELL), int(cy/CELL)
	r := int(mmo.MineReach / CELL)
	for y := py - r; y <= py+r; y++ {
		for x := px - r; x <= px+r; x++ {
			if !ok(x, y) || !exposed(x, y) {
				continue
			}
			dx, dy := float64(x-px), float64(y-py-4)
			score := math.Hypot(dx, dy)
			if (dx > 0) != (player.face > 0) {
				score += 12
			}
			if score < best {
				best, bx, by = score, x, y
			}
		}
	}
	return bx, by, !math.IsInf(best, 1)
}

// A block with open space beside it (you can't dig into the middle of a wall)
func exposed(x, y int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if !isSolid(gridAt(x+d[0], y+d[1])) {
			return true
		}
	}
	return false
}

func nearestMob(reach float64) *Entity {
	cx, cy := playerCenter()
	var best *Entity
	bd := reach
	for _, e := range ents {
		if d := math.Hypot(e.x+e.w/2-cx, e.y+e.h/2-cy); d < bd {
			best, bd = e, d
		}
	}
	return best
}

// Change a block on the island we're drawing
func setCell(x, y int, t uint8) {
	if x < 0 || y < 0 || x >= W || y >= H {
		return
	}
	grid[y*W+x] = t
	shade[y*W+x] = 0
	if t != EMPTY {
		shade[y*W+x] = texShade(x, y)
	}
}

func applyEdits(es []mmo.Edit) {
	for _, e := range es {
		setCell(e.X, e.Y, e.T)
		if e.X == mineX && e.Y == mineY {
			mineX, mineProg = -1, 0
		}
	}
}

// The block being dug shows cracks; the block under the aim gets a faint frame
func drawMineTarget(put func(x, y int, c [3]int, s int), blend func(x, y int, c [3]int, a float64), ox, oy int) {
	if cur != scPlaying || myTool != mmo.ToolPickaxe || curIsle == nil || curIsle.Def.Mine == "" {
		return
	}
	x, y, ok := mineTarget()
	if !ok {
		return
	}
	mmo.Brush(func(dx, dy int) {
		a := 0.18
		if x == mineX && y == mineY && mineProg > 0 {
			a = 0.2 + 0.6*mineProg // cracking open
			blend(x-ox+dx, y-oy+dy, [3]int{20, 16, 30}, a)
			return
		}
		blend(x-ox+dx, y-oy+dy, [3]int{255, 244, 200}, a)
	})
}
