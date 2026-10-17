package game

import (
	"math"
	"math/rand"
)

// Creatures that live in dark caves, and items lying in the world.
// Positions are in world px (same units as the player), local to the loaded window.

type entKind int

const (
	entGloom   entKind = iota // crawls along cave floors and hops at you
	entBat                    // flutters through dark caves and swoops
	entItem                   // a dropped item
	entShade                  // prowls the surface at night
	entWarden                 // the boss guarding the Core
	entSkitter                // small, quick and skittish
	entWisp                   // a cold glowing spirit of the deep caves; hovers, then dives
	entBrute                  // big, slow and tough, deep underground
)

// What each creature is like. Every speed stays below the mote's, so you can always outrun them.
type kindInfo struct {
	w, h, hp, dmg int
	speed         float64 // fraction of the mote's top speed (always under 1)
	fly           bool
	sense         int // how far along a path (in cells) it notices the mote
}

var kinds = map[entKind]kindInfo{
	entGloom:   {8, 6, 4, 2, 0.55, false, 100},
	entBat:     {7, 5, 2, 1, 0.75, true, 130},
	entShade:   {8, 6, 5, 2, 0.75, false, 140},
	entWarden:  {16, 12, 40, 4, 0.85, true, 240},
	entSkitter: {6, 4, 2, 1, 0.85, false, 90},
	entWisp:    {6, 6, 3, 1, 0.6, true, 140},
	entBrute:   {12, 9, 12, 3, 0.4, false, 110},
}

func foeSpeed(k entKind) float64 { return speed * kinds[k].speed }

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
	aware       int     // frames left chasing after losing track of the mote
	stuck       int     // frames spent barely moving while trying to
	mood        float64 // per-creature random offset so they don't all move alike
}

var ents []*Entity

var fullWarnAt = -1000

const (
	maxCreatures = 3   // creatures alive at once (not counting the Warden), one more on Hard
	spawnEvery   = 420 // frames between spawn attempts
)

func newFoe(k entKind, cx, cy int) *Entity {
	ki := kinds[k]
	return &Entity{kind: k, x: float64(cx * CELL), y: float64(cy * CELL), w: float64(ki.w * CELL), h: float64(ki.h * CELL),
		hp: ki.hp, face: 1, mood: rand.Float64()}
}

// Pick a creature for this depth (in cells below the surface)
func foeForDepth(depth int) entKind {
	type pick struct {
		k entKind
		w float64
	}
	table := []pick{{entGloom, 45}, {entBat, 35}, {entSkitter, 20}}
	if depth > 120 {
		table = append(table, pick{entWisp, 25})
	}
	if depth > 240 {
		table = append(table, pick{entBrute, 22})
	}
	sum := 0.0
	for _, p := range table {
		sum += p.w
	}
	r := rand.Float64() * sum
	for _, p := range table {
		if r -= p.w; r < 0 {
			return p.k
		}
	}
	return entGloom
}

// ---- Spawning ----

func spawnCreatures() {
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	// the Warden wakes when you come near the Core
	if !bossDown && frame%30 == 0 {
		kx, ky := float64(coreX()-baseChunk*CW), float64(coreY)
		near := math.Hypot(kx-pcx, ky-pcy) < 90
		awake := false
		for _, e := range ents {
			awake = awake || e.kind == entWarden
		}
		if near && !awake {
			e := newFoe(entWarden, int(kx), int(ky)-20)
			ents = append(ents, e)
			toast("The Warden of the Core awakens!")
			sfx("boom")
			shake = 8
			return
		}
	}
	if frame%spawnEvery != 0 || rand.Float64() > 0.4 {
		return
	}
	n := 0
	for _, e := range ents {
		if e.kind != entItem && e.kind != entWarden {
			n++
		}
	}
	if n >= maxCreatures+b2i(world.Mode >= modeHard) {
		return
	}
	// shades creep over the surface at night
	if isNight() && rand.Float64() < 0.5 {
		cx := int(pcx) + []int{-1, 1}[rand.Intn(2)]*(60+rand.Intn(40))
		if cx > 8 && cx < W-8 {
			cy := surfaceAt(baseChunk*CW+cx) - 6
			if cy > 8 && boxFree(cx-4, cy-4, 8, 6) && lightAt(cx, cy) < 36 && wall[cy*W+cx] == 0 {
				ents = append(ents, newFoe(entShade, cx-4, cy-4))
				return
			}
		}
	}
	for try := 0; try < 12; try++ {
		ang := rand.Float64() * 2 * math.Pi
		dist := 55 + rand.Float64()*70
		cx, cy := int(pcx+math.Cos(ang)*dist), int(pcy+math.Sin(ang)*dist*0.6)
		if cx < 8 || cx >= W-8 || cy < 8 || cy >= H-8 {
			continue
		}
		depth := cy - surfaceAt(baseChunk*CW+cx)
		if depth < 18 || lightAt(cx, cy) > 8 || wall[cy*W+cx] != 0 { // only deep, in the dark, and not in front of walls
			continue
		}
		k := foeForDepth(depth)
		ki := kinds[k]
		if !boxFree(cx-1, cy-1, ki.w+2, ki.h+2) {
			continue
		}
		e := newFoe(k, cx, cy)
		if !ki.fly { // drop it to the floor
			for i := 0; i < 80 && !entHits(e, e.x, e.y+CELL); i++ {
				e.y += CELL
			}
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
	for _, e := range ents {
		if e.kind != entItem {
			updateFields()
			break
		}
	}
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
		default:
			think(e, dx, dy, dist)
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
					popItem(e.item, got)
				} else if frame-fullWarnAt > 240 {
					fullWarnAt = frame
					toast("Your inventory is full")
				}
				if e.n <= 0 {
					continue
				}
			}
		}
		// creatures hurt the mote on contact
		if e.kind != entItem && e.hp > 0 && overlap(e) {
			hurtPlayer(kinds[e.kind].dmg, sign(dx))
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

// Creature behaviour: follow the path to the mote when they've noticed it, each in its own way
func think(e *Entity, dx, dy, dist float64) {
	ki := kinds[e.kind]
	ex, ey := (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL
	top := foeSpeed(e.kind)
	if cellAt(e.x+e.w/2, e.y+e.h/2) == WATER {
		top *= 0.55 // the mote is slowed in water too, but less
	}
	// notice the mote when it is near along a real path; keep chasing a while after
	reach := ki.sense / fbs
	if e.aware > 0 {
		reach *= 2
	}
	px, py, steps, chasing := pathDir(!ki.fly, ex, ey, reach)
	if chasing {
		e.aware = 240
	} else if e.aware > 0 {
		e.aware--
	}
	if dist < 9 { // right next to the mote: go straight for it
		px, py, chasing = dx/(dist*CELL+0.01), dy/(dist*CELL+0.01), true
	}
	phase := (e.t + int(e.mood*600)) % 600

	if ki.fly {
		wob := math.Sin(float64(e.t)*0.12 + e.mood*6)
		wx, wy := 0.0, 0.0
		switch {
		case !chasing:
			// drift about
			wx = math.Cos(float64(e.t)*0.01+e.mood*9) * 0.5 * top
			wy = wob * 0.4 * top
		case e.kind == entBat && phase%200 < 45 && dist < 30:
			// swoop away, then come back around
			wx, wy = -px*top+(-py)*0.6*top, -py*top-0.5*top
		case e.kind == entWisp && phase < 300 && steps*fbs < 40:
			// hover at a distance, circling the mote, before diving
			ox, oy := -py, px // sideways
			keep := (float64(steps*fbs) - 22) / 22
			wx = (ox*0.8+px*keep)*top + wob*0.2
			wy = (oy*0.8+py*keep)*top + wob*0.2
		case e.kind == entWarden && e.t%240 < 60 && e.t > 120:
			wx, wy = px*top, py*top // charge
		default:
			slow := 1.0
			if e.kind == entWarden {
				slow = 0.55
			}
			wx = (px - py*wob*0.35) * top * slow
			wy = (py + px*wob*0.35) * top * slow
		}
		// bats shy away from torches, lanterns and daylight (but not the mote's own glow)
		if e.kind == entBat && lightAt(int(ex), int(ey)) > 52 {
			wx -= float64(sign(dx)) * top
			wy -= 0.5 * top
		}
		e.vx += (wx - e.vx) * 0.08
		e.vy += (wy - e.vy) * 0.08
		if e.hurt == 0 {
			if l := math.Hypot(e.vx, e.vy); l > top {
				e.vx, e.vy = e.vx/l*top, e.vy/l*top
			}
		}
		if e.vx != 0 {
			e.face = sign(e.vx)
		}
		x0, y0 := e.x, e.y
		entMove(e)
		unstickFoe(e, x0, y0, top)
		return
	}

	// walkers
	e.vy = math.Min(e.vy+0.4, 8)
	want := 0.0
	jump := false
	switch {
	case chasing:
		if math.Abs(px) > 0.15 {
			e.face = sign(px)
		}
		want = float64(e.face) * top
		if math.Abs(px) < 0.3 && py < -0.5 { // the way on is straight up: hop
			want *= 0.3
			jump = true
		} else if py < -0.45 {
			jump = true
		}
		// skitters lose their nerve now and then and scuttle back
		if e.kind == entSkitter && phase%240 < 50 {
			want = -want * 0.8
		}
	default:
		// wander, turning now and then and at walls
		if e.t%150 == 0 && rand.Float64() < 0.5 {
			e.face = -e.face
		}
		want = float64(e.face) * top * 0.35
		if e.kind == entShade {
			want = float64(sign(dx)) * top * 0.5 // shades drift toward the mote even unseen
		}
	}
	e.vx += (want - e.vx) * 0.12
	if e.hurt == 0 {
		e.vx = math.Max(-top, math.Min(top, e.vx))
	}
	jv := map[entKind]float64{entGloom: 5.6, entShade: 6, entSkitter: 5.4, entBrute: 5.4}[e.kind] // all clear 4 blocks
	if jump && e.onGround {
		e.vy = -jv
	}
	x0, y0 := e.x, e.y
	bumped := entMove(e)
	unstickFoe(e, x0, y0, top)
	if bumped && e.onGround {
		if chasing {
			e.vy = -jv
		} else {
			e.face = -e.face
		}
	}
	// shades fade away in daylight
	if e.kind == entShade && !isNight() && rand.Float64() < 0.01 {
		e.hp = 0
	}
}

// A creature that keeps trying to move but goes nowhere gets a shove in a random direction
func unstickFoe(e *Entity, x0, y0, top float64) {
	if math.Hypot(e.x-x0, e.y-y0) < 0.05 && e.hurt == 0 {
		e.stuck++
	} else {
		e.stuck = 0
	}
	if e.stuck > 25 {
		a := rand.Float64() * 2 * math.Pi
		e.vx, e.vy = math.Cos(a)*top, math.Sin(a)*top
		if !kinds[e.kind].fly && e.onGround {
			e.vy = -4
		}
		e.stuck = 0
	}
}

func overlap(e *Entity) bool {
	return e.x < player.x+player.w && e.x+e.w > player.x && e.y < player.y+player.h && e.y+e.h > player.y
}

func killEntity(e *Entity) {
	cx, cy := (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL
	burst(18, cx, cy, [3]int{150, 90, 220}, 1.6, 1.6, 30)
	sfx("die")
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
	case entSkitter:
		if rand.Float64() < 0.6 {
			dropItem("copper", 1+rand.Intn(2), e.x, e.y, 0)
		}
	case entWisp:
		dropItem("crystal", 1+rand.Intn(2), e.x, e.y, 0)
		if rand.Float64() < 0.15 {
			dropItem("gem", 1, e.x, e.y, 0)
		}
	case entBrute:
		dropItem("iron", 2+rand.Intn(3), e.x, e.y, 0)
		if rand.Float64() < 0.3 {
			dropItem("heart", 1, e.x, e.y, 0)
		}
	case entWarden:
		bossDown = true
		burst(60, cx, cy, [3]int{255, 120, 220}, 3, 3, 50)
		shake = 12
		dropItem("heart", 2, e.x, e.y, 0)
		dropItem("goldbar", 6, e.x+20, e.y, 0)
		dropItem("crystal", 6, e.x-20, e.y, 0)
		toast("The Warden falls! The Core can be mined now")
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
			popDamage(dmg, e.x+e.w/2, e.y, false)
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

var skitterSprite = [][]string{{
	".O..O.",
	"ORRRRO",
	"OeRReO",
	"O.OO.O",
}, {
	"O....O",
	"ORRRRO",
	"OeRReO",
	".O..O.",
}}

var wispSprite = [][]string{{
	"..WW..",
	".WVVW.",
	"WVeeVW",
	"WVVVVW",
	".WVVW.",
	"..W.W.",
}, {
	"..WW..",
	".WVVW.",
	"WVeeVW",
	"WVVVVW",
	".WVVW.",
	".W.W..",
}}

var bruteSprite = []string{
	"...OOOOOO...",
	"..OBBBBBBO..",
	".OBBBBBBBBO.",
	"OBBeBBBBeBBO",
	"OBBBBBBBBBBO",
	"OBMBBBBBBMBO",
	".OBBBBBBBBO.",
	".OBO....OBO.",
	".OO......OO.",
}

var entCol = map[byte][3]int{'O': {30, 22, 46}, 'G': {82, 64, 120}, 'S': {40, 34, 70}, 'P': {150, 60, 140}, 'K': {255, 120, 220},
	'R': {128, 74, 54}, 'W': {120, 200, 255}, 'V': {200, 240, 255}, 'B': {84, 78, 96}, 'M': {70, 110, 66}}

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
			case entSkitter:
				rows = skitterSprite[(e.t/4)%2]
			case entWisp:
				rows = wispSprite[(e.t/10)%2]
			case entBrute:
				rows = bruteSprite
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
					case e.kind == entWarden || e.kind == entWisp:
						put(bx+x, by+y, entCol[ch], 0) // these glow, visible in the dark
					default:
						lit(bx+x, by+y, entCol[ch], 0)
					}
				}
			}
		}
	}
}
