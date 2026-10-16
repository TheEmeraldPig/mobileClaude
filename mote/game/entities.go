package game

import (
	"math"
	"math/rand"
)

// Creatures that live in dark caves, and items lying in the world.
// Positions are in world px (same units as the player), local to the loaded window.

type entKind int

const (
	entGloom  entKind = iota // crawls along cave floors and hops at you
	entBat                   // flutters through dark caves
	entItem                  // a dropped item
	entShade                 // prowls the surface at night
	entWarden                // the boss guarding the Core
)

type Entity struct {
	kind        entKind
	x, y, w, h  float64
	vx, vy      float64
	hp          int
	face        int
	t           int // age in frames
	hurt        int // flash timer
	onGround    bool
	item        string
	n           int
	pickupDelay int
	attackTimer int
}

var ents []*Entity

const maxCreatures = 10

// ---- Spawning ----

func spawnCreatures() {
	if frame%45 != 0 {
		return
	}
	n := 0
	for _, e := range ents {
		if e.kind != entItem {
			n++
		}
	}
	if n >= maxCreatures {
		return
	}
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	// the Warden wakes when you come near the Core
	if !run.BossDown {
		kx, ky := float64(coreX()-baseChunk*CW), float64(coreY)
		near := math.Hypot(kx-pcx, ky-pcy) < 90
		awake := false
		for _, e := range ents {
			awake = awake || e.kind == entWarden
		}
		if near && !awake {
			ents = append(ents, &Entity{kind: entWarden, x: kx * CELL, y: (ky - 20) * CELL, w: 16 * CELL, h: 12 * CELL, hp: 40, face: 1})
			toast("The Warden of the Core awakens!")
			sfx("boom")
			shake = 8
			return
		}
	}
	// shades creep over the surface at night
	if isNight() && rand.Float64() < 0.5 {
		cx := int(pcx) + []int{-1, 1}[rand.Intn(2)]*(55+rand.Intn(40))
		if cx > 8 && cx < W-8 {
			cy := surfaceAt(baseChunk*CW+cx) - 6
			if cy > 8 && boxFree(cx-4, cy-4, 8, 6) && lightAt(cx, cy) < 36 {
				ents = append(ents, &Entity{kind: entShade, x: float64(cx * CELL), y: float64((cy - 4) * CELL), w: 8 * CELL, h: 6 * CELL, hp: 5, face: 1})
				return
			}
		}
	}
	for try := 0; try < 12; try++ {
		ang := rand.Float64() * 2 * math.Pi
		dist := 45 + rand.Float64()*70
		cx, cy := int(pcx+math.Cos(ang)*dist), int(pcy+math.Sin(ang)*dist*0.6)
		if cx < 4 || cx >= W-4 || cy < 4 || cy >= H-4 {
			continue
		}
		if cy < surfaceAt(baseChunk*CW+cx)+18 || lightAt(cx, cy) > 8 { // only deep and in the dark
			continue
		}
		if !boxFree(cx-3, cy-3, 7, 7) {
			continue
		}
		e := &Entity{x: float64(cx * CELL), y: float64(cy * CELL), face: 1}
		if rand.Float64() < 0.55 {
			e.kind, e.w, e.h, e.hp = entGloom, 8*CELL, 6*CELL, 4
			// drop it to the floor
			for i := 0; i < 80 && !entHits(e, e.x, e.y+CELL); i++ {
				e.y += CELL
			}
		} else {
			e.kind, e.w, e.h, e.hp = entBat, 7*CELL, 5*CELL, 2
		}
		ents = append(ents, e)
		return
	}
}

func boxFree(x, y, w, h int) bool {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if i < 0 || i >= W || j < 0 || j >= H || grid[j*W+i] != EMPTY {
				return false
			}
		}
	}
	return true
}

func entHits(e *Entity, x, y float64) bool {
	x0, x1 := int(math.Floor(x/CELL)), int(math.Floor((x+e.w-0.01)/CELL))
	y0, y1 := int(math.Floor(y/CELL)), int(math.Floor((y+e.h-0.01)/CELL))
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

// Move with collisions; returns whether it bumped sideways
func entMove(e *Entity) bool {
	bumped := false
	dx := e.vx
	for math.Abs(dx) > 0 {
		s := float64(sign(dx)) * math.Min(1, math.Abs(dx))
		if !entHits(e, e.x+s, e.y) {
			e.x += s
		} else if !entHits(e, e.x+s, e.y-CELL*2) && e.onGround {
			e.x += s
			e.y -= CELL * 2
		} else {
			e.vx = 0
			bumped = true
			break
		}
		dx -= s
	}
	dy := e.vy
	e.onGround = false
	for math.Abs(dy) > 0 {
		s := float64(sign(dy)) * math.Min(1, math.Abs(dy))
		if !entHits(e, e.x, e.y+s) {
			e.y += s
		} else {
			if s > 0 {
				e.onGround = true
			}
			e.vy = 0
			break
		}
		dy -= s
	}
	return bumped
}

// ---- Update ----

func updateEntities() {
	spawnCreatures()
	pcx, pcy := player.x+player.w/2, player.y+player.h/2
	out := ents[:0]
	for _, e := range ents {
		e.t++
		if e.hurt > 0 {
			e.hurt--
		}
		ex, ey := e.x+e.w/2, e.y+e.h/2
		dx, dy := pcx-ex, pcy-ey
		dist := math.Hypot(dx, dy) / CELL
		switch e.kind {
		case entGloom:
			e.vy = math.Min(e.vy+0.4, 8)
			if dist < 70 && math.Abs(dy)/CELL < 30 {
				e.face = sign(dx)
				e.vx += (float64(e.face)*1.3 - e.vx) * 0.1
				if e.onGround && (dy < -CELL*6 || rand.Float64() < 0.01) {
					e.vy = -5.5
				}
			} else {
				if e.t%120 == 0 {
					e.face = []int{-1, 1}[rand.Intn(2)]
				}
				e.vx += (float64(e.face)*0.5 - e.vx) * 0.1
			}
			if entMove(e) && e.onGround {
				e.vy = -5
			}
		case entShade:
			e.vy = math.Min(e.vy+0.4, 8)
			e.face = sign(dx)
			e.vx += (float64(e.face)*1.7 - e.vx) * 0.1
			if e.onGround && (dy < -CELL*6 || rand.Float64() < 0.015) {
				e.vy = -6
			}
			if entMove(e) && e.onGround {
				e.vy = -6
			}
			// shades fade away in daylight
			if !isNight() && rand.Float64() < 0.01 {
				e.hp = 0
			}
		case entWarden:
			e.face = sign(dx)
			if e.t%200 < 40 && e.t > 100 { // charge
				e.vx += (dx/(dist*CELL+1)*5 - e.vx) * 0.2
				e.vy += (dy/(dist*CELL+1)*5 - e.vy) * 0.2
			} else {
				e.vx += (dx/(dist*CELL+1)*1.4 - e.vx) * 0.05
				e.vy += (dy/(dist*CELL+1)*1.4 + math.Sin(float64(e.t)*0.05)*0.8 - e.vy) * 0.05
			}
			entMove(e)
		case entBat:
			wob := math.Sin(float64(e.t) * 0.15)
			if dist < 80 {
				e.vx += (dx/(dist*CELL+1)*1.8 - e.vx) * 0.05
				e.vy += (dy/(dist*CELL+1)*1.8 + wob*0.6 - e.vy) * 0.05
			} else {
				e.vx += (math.Cos(float64(e.t)*0.01)*0.8 - e.vx) * 0.05
				e.vy += (wob*0.5 - e.vy) * 0.05
			}
			// bats shy away from bright light
			if lightAt(int(ex/CELL), int(ey/CELL)) > 44 {
				e.vx -= float64(sign(dx)) * 0.3
				e.vy -= 0.2
			}
			e.face = sign(e.vx)
			entMove(e)
		case entItem:
			e.vy = math.Min(e.vy+0.35, 6)
			e.vx *= 0.92
			if e.pickupDelay > 0 {
				e.pickupDelay--
			} else if dist < 14 { // drift toward the mote when close
				e.vx += dx / (dist*CELL + 1) * 0.6
				e.vy += dy / (dist*CELL + 1) * 0.6
			}
			entMove(e)
			if e.pickupDelay == 0 && overlap(e) {
				got := addItem(e.item, e.n)
				if got > 0 {
					e.n -= got
					sfx("pickup")
				}
				if e.n <= 0 {
					continue
				}
			}
		}
		// creatures hurt the mote on contact
		if e.kind != entItem && e.hp > 0 && overlap(e) {
			hurtPlayer(map[entKind]int{entGloom: 2, entBat: 1, entShade: 2, entWarden: 4}[e.kind], sign(dx))
		}
		if e.hp <= 0 {
			killEntity(e)
			continue
		}
		// forget things far away
		// creatures far away are forgotten; dropped items stay where they are
		if e.y > float64(H*CELL) || (dist > 260 && e.kind != entWarden && e.kind != entItem) {
			continue
		}
		out = append(out, e)
	}
	ents = out
}

func overlap(e *Entity) bool {
	return e.x < player.x+player.w && e.x+e.w > player.x && e.y < player.y+player.h && e.y+e.h > player.y
}

func killEntity(e *Entity) {
	cx, cy := (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL
	burst(18, cx, cy, [3]int{150, 90, 220}, 1.6, 1.6, 30)
	sfx("die")
	run.Kills++
	switch e.kind {
	case entGloom:
		dropItem("coal", 1+rand.Intn(3), e.x, e.y, 0)
		if rand.Float64() < 0.2 {
			dropItem("crystal", 1, e.x, e.y, 0)
		}
	case entShade:
		if rand.Float64() < 0.5 {
			dropItem("seeds", 1+rand.Intn(3), e.x, e.y, 0)
		}
	case entWarden:
		run.BossDown = true
		burst(60, cx, cy, [3]int{255, 120, 220}, 3, 3, 50)
		shake = 12
		dropItem("heart", 2, e.x, e.y, 0)
		dropItem("goldbar", 6, e.x+20, e.y, 0)
		dropItem("crystal", 6, e.x-20, e.y, 0)
		toast("The Warden falls! Mine the Core to win the run")
	}
}

// Damage creatures within r cells of a world cell; returns whether anything was hit
func hitCreatures(cx, cy, r, dmg int) bool {
	hit := false
	for _, e := range ents {
		if e.kind == entItem || e.hurt > 0 {
			continue
		}
		ex, ey := (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL
		if math.Hypot(ex-float64(cx), ey-float64(cy)) <= float64(r)+3 {
			e.hp -= dmg
			e.hurt = 12
			kb := float64(sign(ex - (player.x+player.w/2)/CELL))
			e.vx, e.vy = kb*4, -3
			burst(5, ex, ey, [3]int{240, 122, 106}, 1, 1, 14)
			hit = true
		}
	}
	if hit {
		sfx("hit")
	}
	return hit
}

func dropItem(id string, n int, x, y float64, dir int) {
	e := &Entity{kind: entItem, item: id, n: n, x: x, y: y, w: 4 * CELL, h: 4 * CELL, hp: 1,
		vx: float64(dir) * 3, vy: -2.5, pickupDelay: 20}
	if dir != 0 {
		e.pickupDelay = 70
	}
	ents = append(ents, e)
}

// Throw items out in front of the mote
func throwItem(id string, n int) {
	x := player.x + player.w/2 - 2*CELL + float64(player.face)*player.w/2
	dropItem(id, n, x, player.y+4, player.face)
	sfx("place")
}

// ---- Drawing ----

var gloomSprite = []string{
	"..OOOO..",
	".OGGGGO.",
	"OGGGGGGO",
	"OGeGGeGO",
	"OGGGGGGO",
	".O.O.O..",
}

var batSprite = [][]string{{
	"O.....O",
	"OO.O.OO",
	"OOOOOOO",
	".OeOeO.",
	"..O.O..",
}, {
	".......",
	"..O.O..",
	"OOOOOOO",
	"OOeOeOO",
	"O.O.O.O",
}}

var entCol = map[byte][3]int{'O': {30, 22, 46}, 'G': {82, 64, 120}, 'S': {40, 34, 70}, 'P': {150, 60, 140}, 'K': {255, 120, 220}}

var wardenSprite = []string{
	"..OO........OO..",
	".OPPO......OPPO.",
	"OPPPPOOOOOOPPPPO",
	"OPPPPPPPPPPPPPPO",
	".OPPeePPPPeePPO.",
	".OPPeePKKPeePPO.",
	"..OPPPPKKPPPPO..",
	"..OPPPPPPPPPPO..",
	"...OPOPPPPOPO...",
	"...OO.OPPO.OO...",
	"......OPPO......",
	".......OO.......",
}

func drawEntities(lit, put func(x, y int, c [3]int, s int), ox, oy int) {
	for _, e := range ents {
		bx, by := int(math.Round(e.x/CELL))-ox, int(math.Round(e.y/CELL))-oy
		switch e.kind {
		case entItem:
			it := items[e.item]
			if it == nil {
				continue
			}
			bob := int(math.Round(math.Sin(float64(e.t)*0.08) * 0.6))
			for y, r := range it.icon {
				for x := 0; x < 8; x += 2 { // icons shrink to 4x4 world pixels
					ch := r[x]
					if y%2 == 1 || ch == '.' {
						continue
					}
					if c, ok := iconCol[ch]; ok {
						lit(bx+x/2, by+y/2+bob, c, 0)
					}
				}
			}
		default:
			rows := gloomSprite
			switch e.kind {
			case entBat:
				rows = batSprite[(e.t/6)%2]
			case entWarden:
				rows = wardenSprite
			}
			for y, r := range rows {
				for x := 0; x < len(r); x++ {
					sx := x
					if e.face < 0 {
						sx = len(r) - 1 - x
					}
					ch := r[sx]
					switch {
					case ch == '.':
					case ch == 'e':
						put(bx+x, by+y, [3]int{255, 90, 90}, 0) // glowing eyes, visible in the dark
					case e.hurt > 0:
						put(bx+x, by+y, [3]int{255, 255, 255}, 0)
					case e.kind == entShade && ch == 'G':
						lit(bx+x, by+y, entCol['S'], 0)
					case e.kind == entWarden:
						put(bx+x, by+y, entCol[ch], 0) // the Warden glows with the Core's light
					default:
						lit(bx+x, by+y, entCol[ch], 0)
					}
				}
			}
		}
	}
}
