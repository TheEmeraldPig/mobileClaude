package game

import (
	"math"
	"math/rand"
	"sort"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Your character: walking, jumping, and a short alien hover (hold jump in the air).
// It's drawn in body.go.

const (
	jumpSpeed   = 6.2 // px per frame: a jump clears a bit more than the character's height
	levSeconds  = 2.5 // a full hover bar lasts this long
	speed       = 2.5
	refillGnd   = 1.0 / 120 // full in ~2s on the ground
	refillAir   = 1.0 / 900 // full in ~15s while airborne
	stepUpLimit = 8
)

type Player struct {
	x, y, w, h        float64
	vx, vy            float64
	onGround, inWater bool
	face              int
	energy            float64
	lifting           bool // hovering
	jumpHeld          bool
	walk, squash      float64
	blink, nextBlink  int
	idle              int
	emptyWarned       bool
	lavaTimer         int
	invuln            int // frames left flashing after a hit
}

var player = Player{w: sprW * CELL, h: sprH * CELL, face: 1, energy: 1, nextBlink: 120}

var px0, px1, py0, py1 int

func updatePlayerCells() {
	px0 = int(math.Floor(player.x / CELL))
	px1 = int(math.Floor((player.x + player.w - 0.01) / CELL))
	py0 = int(math.Floor(player.y / CELL))
	py1 = int(math.Floor((player.y + player.h - 0.01) / CELL))
}

func inPlayer(x, y int) bool { return x >= px0 && x <= px1 && y >= py0 && y <= py1 }

func hits(x, y float64) bool {
	x0, x1 := int(math.Floor(x/CELL)), int(math.Floor((x+player.w-0.01)/CELL))
	y0, y1 := int(math.Floor(y/CELL)), int(math.Floor((y+player.h-0.01)/CELL))
	if x0 < 0 || x1 >= W || y1 >= H { // the island's ends are walls
		return true
	}
	for cy := max(0, y0); cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			if isSolid(grid[cy*W+cx]) {
				return true
			}
		}
	}
	return false
}

// Blocks may fall or be built into the mote; it slides out toward the nearest free spot,
// a little each frame. It is never thrown far: if there is no room close by, it stays
// stuck until it digs itself out.
const pushRange = 16.0 // px, a bit over half the mote's height

var pushOffsets = func() [][2]float64 {
	var o [][2]float64
	for dy := -pushRange; dy <= pushRange; dy++ {
		for dx := -pushRange; dx <= pushRange; dx++ {
			if (dx != 0 || dy != 0) && math.Hypot(dx, dy) <= pushRange {
				o = append(o, [2]float64{dx, dy})
			}
		}
	}
	// nearest first; on a tie prefer up, then sideways
	sort.SliceStable(o, func(i, j int) bool {
		di, dj := math.Hypot(o[i][0], o[i][1]), math.Hypot(o[j][0], o[j][1])
		if di != dj {
			return di < dj
		}
		return o[i][1] < o[j][1]
	})
	return o
}()

func pushOut() {
	p := &player
	if !hits(p.x, p.y) {
		return
	}
	for _, o := range pushOffsets {
		if hits(p.x+o[0], p.y+o[1]) {
			continue
		}
		d := math.Hypot(o[0], o[1])
		step := math.Min(1.5, d) / d
		p.x += o[0] * step
		p.y += o[1] * step
		p.vx *= 0.5
		if o[1] < 0 {
			p.vy = math.Min(p.vy, 0)
		}
		return
	}
	// buried: stay put
	p.vx, p.vy = 0, 0
}

func unstick() {
	for i := 0; i < H*CELL && hits(player.x, player.y); i++ {
		player.y--
	}
}

// ---- Particles (world cells) ----

type Particle struct {
	x, y, vx, vy, life, g float64
	c                     [3]int
}

var parts []Particle

func burst(n int, x, y float64, c [3]int, spread, up, life float64) {
	for i := 0; i < n; i++ {
		parts = append(parts, Particle{x: x, y: y, vx: (rand.Float64() - 0.5) * spread, vy: -rand.Float64() * up,
			life: life * (0.6 + rand.Float64()*0.6), c: c, g: 0.08})
	}
}

func updateParticles() {
	out := parts[:0]
	for _, q := range parts {
		q.vy += q.g
		q.x += q.vx
		q.y += q.vy
		q.life--
		cx, cy := int(q.x), int(q.y)
		if q.life <= 0 || (cx >= 0 && cx < W && cy >= 0 && cy < H && isSolid(grid[cy*W+cx]) && q.vy > 0) {
			continue
		}
		out = append(out, q)
	}
	parts = out
}

// ---- Physics ----

func updatePlayer() {
	p := &player
	pushOut()
	if p.invuln > 0 {
		p.invuln--
	}
	// lava throws the mote up (the server works out the burns)
	if cellAt(p.x+p.w/2, p.y+p.h-2) == LAVA || cellAt(p.x+p.w/2, p.y+p.h/2) == LAVA {
		if p.lavaTimer++; p.lavaTimer%30 == 1 {
			p.vy = -6
		}
	} else {
		p.lavaTimer = 0
	}
	wasInWater, wasOnGround := p.inWater, p.onGround
	p.inWater = cellAt(p.x+p.w/2, p.y+p.h/2) == WATER
	dir := 0.0
	if input.right {
		dir++
	}
	if input.left {
		dir--
	}
	dir = math.Max(-1, math.Min(1, dir+stick.x))
	if dir != 0 {
		p.face = sign(dir)
	}
	sp := speed
	if p.inWater {
		sp *= 0.6
	}
	p.vx += (dir*sp - p.vx) * 0.35

	// Jumping, and hovering: holding jump on the way down slows the fall to a gentle
	// glide (and a slight rise) while the hover bar lasts
	wantLift := input.lev || stick.up
	if wantLift && !p.jumpHeld && (p.onGround || p.inWater) {
		p.vy = -jumpSpeed
		if p.inWater {
			p.vy = -jumpSpeed * 0.6
		}
		p.jumpHeld = true
		sfx("step")
	}
	if !wantLift {
		p.jumpHeld = false
	}
	p.lifting = wantLift && !p.onGround && p.vy > -1.5 && p.energy > 0
	if p.lifting {
		p.energy = math.Max(0, p.energy-1/(levSeconds*60))
		p.vy += (-0.9 - p.vy) * 0.12
		p.emptyWarned = false
		if rand.Float64() < 0.6 {
			c := glowRGB
			if rand.Float64() < 0.5 {
				c = tipRGB
			}
			burst(1, p.x/CELL+3+rand.Float64()*4, (p.y+p.h)/CELL, c, 0.4, -0.3, 22)
		}
	} else {
		if p.inWater {
			p.vy += 0.15
			p.vy = math.Min(p.vy, 2)
		} else {
			p.vy += 0.45
			p.vy = math.Min(p.vy, 10)
		}
		if p.onGround {
			p.energy = math.Min(1, p.energy+refillGnd)
		} else {
			p.energy = math.Min(1, p.energy+refillAir)
		}
		if wantLift && !p.onGround && p.vy > 0 && p.energy == 0 && !p.emptyWarned {
			sfx("empty")
			p.emptyWarned = true
		}
	}
	if !wantLift {
		p.emptyWarned = false
	}

	dx := p.vx
	for math.Abs(dx) > 0 {
		s := float64(sign(dx)) * math.Min(1, math.Abs(dx))
		if !hits(p.x+s, p.y) {
			p.x += s
		} else {
			climbed := false
			for up := 1.0; up <= stepUpLimit; up++ {
				if !hits(p.x+s, p.y-up) {
					p.x += s
					p.y -= up
					climbed = true
					break
				}
			}
			if !climbed {
				p.vx = 0
				break
			}
		}
		dx -= s
	}
	dy, impact := p.vy, 0.0
	p.onGround = false
	for math.Abs(dy) > 0 {
		s := float64(sign(dy)) * math.Min(1, math.Abs(dy))
		if !hits(p.x, p.y+s) {
			p.y += s
		} else {
			if s > 0 {
				p.onGround = true
				impact = p.vy
			}
			p.vy = 0
			break
		}
		dy -= s
	}
	if p.y < -p.h*4 {
		p.y = -p.h * 4
		p.vy = 0
	}

	// Landing: squash, dust and a thump
	if p.onGround && !wasOnGround && impact > 2.5 {
		p.squash = math.Min(0.35, impact*0.035)
		under := cellAt(p.x+p.w/2, p.y+p.h+1)
		c, ok := blockRGB[under]
		if !ok {
			c = blockRGB[SAND]
		}
		burst(int(math.Min(14, impact*1.5)), (p.x+p.w/2)/CELL, (p.y+p.h)/CELL-1, c, 1.6, 0.8, 26)
		sfxLand(impact)
	}
	// Splash on entering water
	if p.inWater && !wasInWater && p.vy > 1 {
		burst(16, (p.x+p.w/2)/CELL, p.y/CELL+sprH/2, blockRGB[WATER], 1.4, 1.8, 35)
		sfx("splash")
	}
	// Footsteps
	if p.onGround && math.Abs(p.vx) > 0.3 {
		before := int(p.walk)
		p.walk += math.Abs(p.vx) * 0.09
		if int(p.walk) != before {
			sfx("step")
			under := cellAt(p.x+p.w/2, p.y+p.h+1)
			c, ok := blockRGB[under]
			if !ok {
				c = blockRGB[SAND]
			}
			burst(2, (p.x+p.w/2)/CELL-float64(p.face*3), (p.y+p.h)/CELL-1, c, 0.6, 0.5, 16)
		}
		p.idle = 0
	} else {
		p.idle++
	}

	p.squash *= 0.8
	p.nextBlink--
	if p.nextBlink <= 0 {
		p.blink = 7
		p.nextBlink = 120 + rand.Intn(200)
	}
	if p.blink > 0 {
		p.blink--
	}
	updatePlayerCells()
}

// ---- Drawing ----

var (
	glowRGB = [3]int{201, 167, 255}
	tipRGB  = [3]int{255, 222, 120}
)

// The look your character has (from the server once online, else from Settings)
func myLook() mmo.Look {
	if me != nil {
		return me.Look
	}
	return *prefs.Look
}

func legFrame(onGround, lifting bool, vx, walk float64) string {
	switch {
	case !onGround || lifting:
		return "air"
	case math.Abs(vx) > 0.3:
		if int(walk)%2 == 1 {
			return "a"
		}
		return "b"
	}
	return "stand"
}

func drawPlayer(put func(x, y int, c [3]int, s int), ox, oy int) {
	p := &player
	if p.invuln > 0 && (p.invuln/4)%2 == 0 {
		return
	}
	ps := pose{face: p.face, legs: legFrame(p.onGround, p.lifting, p.vx, p.walk), blink: p.blink > 0, swing: -1, tool: myTool}
	if useSwing >= 0 {
		ps.swing, ps.aim = useSwing, useAim
	}
	if p.onGround && p.idle > 20 && frame%90 < 25 {
		ps.bob = 1 // breathing
	}
	drawBody(put, ox, oy, p.x, p.y, myLook(), ps)
}

// The small hover bar under the character (only while it isn't full)
func drawEnergy(put func(x, y int, c [3]int, s int), ox, oy int) {
	p := &player
	if p.energy >= 1 {
		return
	}
	const w = 12
	x0 := int(math.Round(p.x/CELL)) - ox + (sprW-w)/2
	y0 := int(math.Round(p.y/CELL)) - oy + sprH + 2
	fill := int(math.Round(p.energy * (w - 2)))
	col := glowRGB
	if p.energy < 0.2 {
		col = [3]int{240, 122, 106}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < 3; y++ {
			edge := y == 0 || y == 2 || x == 0 || x == w-1
			if edge && (x == 0 || x == w-1) && (y == 0 || y == 2) {
				continue
			}
			c := [3]int{19, 16, 39}
			if edge {
				c = outlineRGB
			} else if x-1 < fill {
				c = col
			}
			put(x0+x, y0+y, c, 0)
		}
	}
}

func sign(v float64) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
