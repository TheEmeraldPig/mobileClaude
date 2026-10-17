package game

import (
	"math"
	"math/rand"
	"sort"
)

// The mote: a little one-eyed levitating creature drawn with world pixels.

const (
	sprW        = 10
	sprH        = 13
	levSeconds  = 5.0
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
	lifting           bool
	walk, squash      float64
	blink, nextBlink  int
	idle              int
	emptyWarned       bool
	hp                int // in half-hearts
	maxHP             int
	lavaTimer         int
	invuln            int
	regen             int
}

var player = Player{w: sprW * CELL, h: sprH * CELL, face: 1, energy: 1, nextBlink: 120, hp: 10, maxHP: 10}

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
	if x0 < 0 || x1 >= W || y1 >= H {
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

func hurtPlayer(dmg, dir int) {
	p := &player
	if p.invuln > 0 || p.hp <= 0 {
		return
	}
	dmg = max(1, int(math.Ceil(float64(dmg)*modeDamage[world.Mode]))-armorLevel())
	p.hp -= dmg
	p.invuln = 60
	shake = math.Max(shake, 5)
	p.regen = 0
	if dir == 0 {
		dir = -p.face
	}
	p.vx, p.vy = float64(-dir)*4.5, -4
	burst(10, (p.x+p.w/2)/CELL, (p.y+p.h/2)/CELL, [3]int{240, 122, 106}, 1.4, 1.4, 22)
	sfx("hurt")
	if p.hp <= 0 {
		die()
	}
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
	// lava burns
	if cellAt(p.x+p.w/2, p.y+p.h-2) == LAVA || cellAt(p.x+p.w/2, p.y+p.h/2) == LAVA {
		if p.lavaTimer++; p.lavaTimer%30 == 1 {
			hurtPlayer(2, 0)
			p.vy = -6
		}
	} else {
		p.lavaTimer = 0
	}
	if p.hp < p.maxHP {
		if p.regen++; p.regen >= 60*8 { // a half heart every 8 seconds
			p.hp++
			p.regen = 0
		}
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

	// Levitation: a full bar lasts levSeconds; refills fast on the ground and slowly in the air
	wantLift := input.lev || stick.up
	p.lifting = wantLift && p.energy > 0
	if p.lifting {
		p.energy = math.Max(0, p.energy-1/(levSeconds*60))
		p.vy += (-2.2 - p.vy) * 0.15
		p.emptyWarned = false
		if rand.Float64() < 0.6 {
			c := glowRGB
			if rand.Float64() < 0.5 {
				c = tipRGB
			}
			burst(1, p.x/CELL+2+rand.Float64()*6, (p.y+p.h)/CELL, c, 0.4, -0.3, 22)
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
		if wantLift && !p.emptyWarned {
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

// ---- Sprite ----

// Facing right. O outline, B body, L highlight, A antenna, T antenna tip. The eye is added per frame.
var moteBody = []string{
	"....T.....",
	"....A.....",
	"..OOOOOO..",
	".OBBBBBBO.",
	"OBLLBBBBBO",
	"OBLBBBBBBO",
	"OBBBBBBBBO",
	"OBBBBBBBBO",
	"OBBBBBBBBO",
	"OBBBBBBBBO",
	".OBBBBBBO.",
	"..OOOOOO..",
}

var moteFeet = map[string]string{"stand": "..OO..OO..", "a": ".OO....OO.", "b": "...OOOO...", "tuck": ".........."}

var (
	glowRGB = [3]int{201, 167, 255}
	tipRGB  = [3]int{255, 222, 120}
	pcol    = map[byte][3]int{'O': {34, 28, 64}, 'B': {124, 98, 255}, 'L': {196, 180, 255}, 'A': {34, 28, 64},
		'W': {246, 246, 252}, 'E': {18, 16, 30}, 'T': tipRGB}
	pShade = func() []int {
		s := make([]int, sprW*sprH)
		for i := range s {
			s[i] = rand.Intn(12) - 6
		}
		return s
	}()
)

func playerFrame(blink, lookUp bool, feet string, sway int) [][]byte {
	rows := make([][]byte, 0, sprH)
	for _, r := range moteBody {
		rows = append(rows, []byte(r))
	}
	if sway != 0 {
		rows[0][4] = '.'
		rows[0][4+sway] = 'T'
	}
	if blink {
		for x := 5; x <= 7; x++ {
			rows[6][x] = 'O'
		}
	} else {
		for y := 5; y <= 7; y++ {
			for x := 5; x <= 7; x++ {
				rows[y][x] = 'W'
			}
		}
		py := 6
		if lookUp {
			py = 5
		}
		for y := py; y <= py+1; y++ {
			for x := 6; x <= 7; x++ {
				rows[y][x] = 'E'
			}
		}
	}
	return append(rows, []byte(moteFeet[feet]))
}

func drawPlayer(put func(x, y int, c [3]int, s int), ox, oy int) {
	p := &player
	stretch := 0.0
	if !p.onGround && !p.lifting && p.vy > 3 {
		stretch = math.Min(0.2, (p.vy-3)*0.04)
	}
	breathe := 0
	if p.onGround && p.idle > 20 && frame%90 < 25 {
		breathe = 1
	}
	tw := max(6, int(math.Round(sprW*(1+p.squash*0.7-stretch*0.5))))
	th := max(8, int(math.Round(sprH*(1-p.squash+stretch)))-breathe)
	bob := 0
	if p.lifting {
		bob = int(math.Round(math.Sin(float64(frame) * 0.2)))
	}
	bx := int(math.Round(p.x/CELL)) - ox + (sprW-tw)/2
	by := int(math.Round(p.y/CELL)) - oy + (sprH - th) + bob
	sway := 0
	if math.Abs(p.vx) > 0.5 {
		sway = -1
	} else if p.idle > 60 && (frame>>4)%4 == 0 {
		sway = 1
	}
	feet := "stand"
	if p.lifting || (!p.onGround && !p.inWater) {
		feet = "tuck"
	} else if math.Abs(p.vx) > 0.3 {
		if int(p.walk)%2 == 1 {
			feet = "a"
		} else {
			feet = "b"
		}
	}
	if p.invuln > 0 && (p.invuln/4)%2 == 0 {
		return
	}
	rows := playerFrame(p.blink > 0, p.lifting, feet, sway)
	tipPulse := int(math.Round(math.Sin(float64(frame)*0.08)*20)) - 10
	if p.lifting {
		tipPulse = 30
	}
	for ty := 0; ty < th; ty++ {
		sy := min(sprH-1, int((float64(ty)+0.5)*sprH/float64(th)))
		for tx := 0; tx < tw; tx++ {
			sxr := min(sprW-1, int((float64(tx)+0.5)*sprW/float64(tw)))
			sx := sxr
			if p.face < 0 {
				sx = sprW - 1 - sxr
			}
			ch := rows[sy][sx]
			if ch == '.' {
				continue
			}
			s := pShade[sy*sprW+sx]
			if ch == 'T' {
				s = tipPulse
			} else if ch == 'W' || ch == 'E' {
				s = 0
			}
			put(bx+tx, by+ty, pcol[ch], s)
		}
	}
}

// Small levitation bar drawn in world pixels just under the mote
func drawEnergy(put func(x, y int, c [3]int, s int), ox, oy int) {
	p := &player
	const w = 12
	x0 := int(math.Round(p.x/CELL)) - ox + (sprW-w)/2
	y0 := int(math.Round(p.y/CELL)) - oy + sprH + 2
	fill := int(math.Round(p.energy * (w - 2)))
	col := pcol['B']
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
				c = pcol['O']
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
