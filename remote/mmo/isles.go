package mmo

import "math"

// Islands. Every island is one shared place on the server; the game and the server
// both build its terrain from its seed, so only creatures and players travel over the
// network.
//
// RULES FOR ISLAND GENERATION (like Mote's world rules):
//  1. Never use math/rand here. Use the seeded hashes in noise.go, so the game and the
//     server build exactly the same island.
//  2. Each step is a function in build(). Give every step its own salt numbers.
//  3. Changing a step changes the island everyone plays on: isles_test.go pins a
//     checksum per island, update it when you mean to.

type Biome int

const (
	BiomeHaven Biome = iota
	BiomeMeadow
	BiomeHollow
	BiomeEmber
)

type IsleDef struct {
	ID, Name, Blurb string
	Seed            uint32
	Level           int // recommended level
	Biome           Biome
	Safe            bool // no creatures
	Ground, Water   int  // mean ground row and sea level
	Hills           float64
	Ocean           int     // cells of sea at each end
	Caves           int     // winding tunnels
	Cavern          float64 // threshold for big caverns (lower opens more; 0 = none)
	CavernTop       int     // how far below the ground caverns start
	CaveTop         int     // how far below the ground tunnels may reach (a few still break through)
	Crystals        float64 // chance of a crystal on a cave wall spot
	Trees           float64 // share of tree spots that get a tree
	Mobs            []MobChance
	MaxMobs         int
	Rarity          [NRarity]float64 // weights for a creature's rarity
	Boss            string           // a boss creature that comes back every few minutes
	MapX, MapY      float64          // where it sits on the travel map (0-1)
}

type MobChance struct {
	Kind   string
	Weight float64
}

var Isles = []*IsleDef{
	{ID: "haven", Name: "Haven", Blurb: "The town everyone shares. Safe: no creatures.",
		Seed: 7001, Level: 1, Biome: BiomeHaven, Safe: true, Ground: 330, Water: 360, Hills: 26, Ocean: 260,
		Trees: 0.55, MapX: 0.5, MapY: 0.55},
	{ID: "meadow", Name: "Greenreach", Blurb: "Grassy hills with skitters, gloomlings and bats.",
		Seed: 7002, Level: 1, Biome: BiomeMeadow, Ground: 320, Water: 360, Hills: 60, Ocean: 220,
		Caves: 26, Cavern: 0.66, CavernTop: 40, CaveTop: 22, Crystals: 0.15, Trees: 0.7,
		Mobs:    []MobChance{{"skitter", 40}, {"gloom", 35}, {"bat", 25}},
		MaxMobs: 70, Rarity: [NRarity]float64{70, 25, 5}, MapX: 0.22, MapY: 0.35},
	{ID: "hollow", Name: "Glimmer Hollow", Blurb: "Dark crystal caves. Bring light.",
		Seed: 7003, Level: 8, Biome: BiomeHollow, Ground: 300, Water: 350, Hills: 70, Ocean: 200,
		Caves: 60, Cavern: 0.56, CavernTop: 10, Crystals: 0.55, Trees: 0.25,
		Mobs:    []MobChance{{"bat", 30}, {"wisp", 30}, {"gloom", 20}, {"brute", 12}},
		MaxMobs: 85, Rarity: [NRarity]float64{20, 45, 28, 6, 1}, MapX: 0.8, MapY: 0.3},
	{ID: "ember", Name: "Cinder Isle", Blurb: "Hot sand and lava. A Warden wakes every few minutes.",
		Seed: 7004, Level: 18, Biome: BiomeEmber, Ground: 310, Water: 355, Hills: 80, Ocean: 200,
		Caves: 45, Cavern: 0.6, CavernTop: 30, CaveTop: 14, Crystals: 0.1,
		Mobs:    []MobChance{{"shade", 35}, {"brute", 30}, {"wisp", 20}, {"skitter", 15}},
		MaxMobs: 90, Rarity: [NRarity]float64{0, 25, 45, 25, 5}, Boss: "warden", MapX: 0.62, MapY: 0.8},
}

func IsleByID(id string) *IsleDef {
	for _, d := range Isles {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// Isle is an island's terrain, GW x GH cells
type Isle struct {
	Def            *IsleDef
	Cells          []uint8
	Surf           []int16 // the ground row of each column (from the terrain shape, ignoring caves)
	SpawnX, SpawnY int     // the spawn point: column, and the ground row under it
	LavaRow        int     // lava fills open cells from here down (GH = none)
}

// At is the block at a cell; outside the island is solid on the sides and below, open above
func (is *Isle) At(x, y int) uint8 {
	if x < 0 || x >= GW || y >= GH {
		return BEDROCK
	}
	if y < 0 {
		return EMPTY
	}
	return is.Cells[y*GW+x]
}

func (is *Isle) Solid(x, y int) bool { return IsSolid(is.At(x, y)) }

// The column's ground row (for columns outside, the nearest edge's)
func (is *Isle) SurfAt(x int) int {
	return int(is.Surf[max(0, min(GW-1, x))])
}

// BoxFree reports whether a box in world px touches no solid cell
func (is *Isle) BoxFree(x, y, w, h float64) bool {
	x0, x1 := int(math.Floor(x/Cell)), int(math.Floor((x+w-0.01)/Cell))
	y0, y1 := int(math.Floor(y/Cell)), int(math.Floor((y+h-0.01)/Cell))
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			if is.Solid(cx, cy) {
				return false
			}
		}
	}
	return true
}

// SpawnPos is where a mote appears, in world px (its top-left corner)
func (is *Isle) SpawnPos() (float64, float64) {
	return float64(is.SpawnX*Cell) - MoteW/2, float64(is.SpawnY*Cell) - MoteH - 2
}

// BuildIsle generates an island from its definition (about a quarter of a second)
func BuildIsle(d *IsleDef) *Isle {
	g := gen{seed: d.Seed}
	is := &Isle{Def: d, Cells: make([]uint8, GW*GH), Surf: make([]int16, GW), LavaRow: GH}
	for x := 0; x < GW; x++ {
		is.Surf[x] = int16(g.surface(d, x))
	}
	g.fill(is)
	g.caves(is)
	g.caverns(is)
	g.water(is)
	g.ores(is)
	g.crystals(is)
	if d.Biome == BiomeEmber {
		// lava pools on the cave floors deep down
		is.LavaRow = d.Ground + 150
		for x := 0; x < GW; x++ {
			for y := GH - 4; y >= is.LavaRow; y-- {
				if is.Cells[y*GW+x] != EMPTY {
					continue
				}
				for k := 1; k <= 4; k++ {
					if b := is.Cells[(y+k)*GW+x]; IsSolid(b) {
						is.Cells[y*GW+x] = LAVA
						break
					} else if b != LAVA {
						break
					}
				}
			}
		}
	}
	g.trees(is)
	g.flowers(is)
	for y := GH - 3; y < GH; y++ {
		for x := 0; x < GW; x++ {
			is.Cells[y*GW+x] = BEDROCK
		}
	}
	g.findSpawn(is)
	if d.Biome == BiomeHaven {
		g.plaza(is)
	} else {
		g.shrine(is)
	}
	return is
}

func smooth(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// The ground's shape: hills in the middle sloping down to the sea floor at both ends
func (g gen) surface(d *IsleDef, x int) int {
	fx := float64(x)
	s := float64(d.Ground) + g.noise1(fx, 220, 1)*d.Hills*0.8 + g.noise1(fx, 60, 2)*d.Hills*0.3 + g.noise1(fx, 14, 3)*d.Hills*0.06
	if d.Biome == BiomeHaven { // a flat town square in the middle
		mid := math.Abs(fx - GW/2)
		s += (float64(d.Ground) - s) * (1 - smooth(110, 170, mid))
	}
	// keep the land above the sea, then slope it down to the sea floor at the ends
	s = math.Min(s, float64(d.Water-8))
	edge := math.Min(fx, GW-1-fx)
	t := smooth(0, float64(d.Ocean), edge)
	floor := float64(d.Water) + 46 + g.noise1(fx, 30, 4)*6
	return int(math.Floor(floor + (s-floor)*t))
}

func (g gen) fill(is *Isle) {
	d := is.Def
	for x := 0; x < GW; x++ {
		s := int(is.Surf[x])
		top := 6 + int(math.Floor((g.noise1(float64(x), 40, 5)+1)*6))
		beach := s >= d.Water-4
		for y := max(0, s); y < GH; y++ {
			dep := y - s
			var t uint8
			switch {
			case dep >= top:
				t = STONE
			case d.Biome == BiomeEmber:
				t = SAND
				if dep >= 3 {
					t = SANDSTONE
				}
			case beach:
				t = SAND
				if dep >= 4 {
					t = SANDSTONE
				}
			case dep == 0:
				t = GRASS
			default:
				t = DIRT
			}
			is.Cells[y*GW+x] = t
		}
	}
}

// Winding tunnels
func (g gen) caves(is *Isle) {
	d := is.Def
	for k := 0; k < d.Caves; k++ {
		r := g.rng(k, 5)
		x := float64(d.Ocean) + r()*float64(GW-2*d.Ocean)
		y := float64(is.Surf[int(x)]) + 6 + r()*260
		a := r() * math.Pi * 2
		length := 140 + r()*260
		base := 5 + r()*4
		top := d.CaveTop
		if k%6 == 0 {
			top = 0 // an entrance from the surface
		}
		for st := 0.0; st < length; st++ {
			a += (r() - 0.5) * 0.45
			x += math.Cos(a) * 2.2
			y += math.Sin(a) * 1.2
			rad := base + r()*4
			for dy := -rad; dy <= rad; dy++ {
				for dx := -rad; dx <= rad; dx++ {
					cx, cy := int(math.Floor(x+dx)), int(math.Floor(y+dy))
					if cx >= 0 && cx < GW && cy < int(is.Surf[cx])+top {
						continue
					}
					if dx*dx+dy*dy <= rad*rad && cx >= 0 && cx < GW && cy >= 0 && cy < GH-4 {
						is.Cells[cy*GW+cx] = EMPTY
					}
				}
			}
		}
	}
}

// Big open caverns
func (g gen) caverns(is *Isle) {
	d := is.Def
	if d.Cavern == 0 {
		return
	}
	for x := 0; x < GW; x++ {
		fx := float64(x)
		edge := smooth(float64(d.Ocean)*0.6, float64(d.Ocean), math.Min(fx, GW-1-fx)) // none under the sea
		if edge == 0 {
			continue
		}
		top := int(is.Surf[x]) + d.CavernTop
		for y := max(top, 0); y < min(GH-6, top+420); y++ {
			fade := math.Min(1, float64(y-top)/30) * edge
			n := g.noise2(fx, float64(y), 56, 11) * 0.7
			if (n+0.3)*fade <= d.Cavern {
				continue
			}
			n += g.noise2(fx, float64(y), 18, 12) * 0.3
			if n*fade > d.Cavern {
				is.Cells[y*GW+x] = EMPTY
			}
		}
	}
}

// The sea, and lakes in low valleys
func (g gen) water(is *Isle) {
	w := is.Def.Water
	for x := 0; x < GW; x++ {
		for y := w; y < int(is.Surf[x]) && y < GH; y++ {
			if is.Cells[y*GW+x] == EMPTY {
				is.Cells[y*GW+x] = WATER
			}
		}
	}
}

// Ore clumps in the stone (just for looks: Remote has no mining)
func (g gen) ores(is *Isle) {
	const region = 18
	kinds := []uint8{COAL, COPPER, IRON, GOLD}
	for rx := 0; rx <= GW/region; rx++ {
		for ry := 0; ry <= GH/region; ry++ {
			if g.hash01(rx, ry, 41) > 0.3 {
				continue
			}
			cx := float64(rx*region) + 2 + g.hash01(rx, ry, 42)*(region-4)
			cy := float64(ry*region) + 2 + g.hash01(rx, ry, 43)*(region-4)
			dep := int(cy) - is.SurfAt(int(cx))
			n := 1 + min(3, max(0, dep/90))
			t := kinds[int(g.hash01(rx, ry, 44)*float64(n))]
			r := 2.2 + g.hash01(rx, ry, 45)*2.2
			ri := int(r) + 2
			for y := int(cy) - ri; y <= int(cy)+ri; y++ {
				for x := int(cx) - ri; x <= int(cx)+ri; x++ {
					if x < 0 || x >= GW || y < 0 || y >= GH {
						continue
					}
					rr := r * (0.8 + 0.4*g.hash01(x, y, 46))
					if math.Hypot(float64(x)+0.5-cx, (float64(y)+0.5-cy)*1.2) > rr {
						continue
					}
					if i := y*GW + x; is.Cells[i] == STONE {
						is.Cells[i] = t
					}
				}
			}
		}
	}
}

// Glowing crystals on cave walls
func (g gen) crystals(is *Isle) {
	p := is.Def.Crystals
	if p == 0 {
		return
	}
	for x := 1; x < GW-1; x++ {
		for y := int(is.Surf[x]) + 12; y < GH-4; y++ {
			i := y*GW + x
			if is.Cells[i] != STONE {
				continue
			}
			c := is.Cells
			if c[i-GW] != EMPTY && c[i+GW] != EMPTY && c[i-1] != EMPTY && c[i+1] != EMPTY {
				continue
			}
			if g.noise2(float64(x), float64(y), 7, 31) > 0.72 && g.hash01(x, y, 32) < p {
				c[i] = CRYSTAL
			}
		}
	}
}

func (is *Isle) set(x, y int, t uint8) {
	if x >= 0 && x < GW && y >= 0 && y < GH {
		is.Cells[y*GW+x] = t
	}
}

func (is *Isle) get(x, y int) uint8 {
	if x >= 0 && x < GW && y >= 0 && y < GH {
		return is.Cells[y*GW+x]
	}
	return BEDROCK
}

// Big Kyora-style trees (ported from Mote's world.go)
func (g gen) trees(is *Isle) {
	d := is.Def
	if d.Trees == 0 {
		return
	}
	const slot = 34
	for sl := 0; sl < GW/slot; sl++ {
		if g.hash01(sl, 0, 71) > d.Trees {
			continue
		}
		x := sl*slot + 4 + int(g.hash01(sl, 1, 71)*float64(slot-8))
		s := is.SurfAt(x)
		if s >= d.Water-4 || absI(is.SurfAt(x-4)-is.SurfAt(x+4)) > 5 || is.get(x, s) != GRASS {
			continue
		}
		if d.Biome == BiomeHaven && absI(x-GW/2) < 130 {
			continue // the town square stays open
		}
		size := 0.6 + g.hash01(sl, 4, 71)*0.35
		if g.hash01(sl, 5, 71) < 0.45 {
			size = 1.2 + g.hash01(sl, 6, 71)*0.6
		}
		g.growTree(is, x, s, sl, size)
	}
}

func (g gen) growTree(is *Isle, x, s, sl int, size float64) {
	r := func(k int) float64 { return g.hash01(sl, 10+k, 72) }
	height := int(size * (34 + r(0)*16))
	base := 1 + size*2.2
	sway := (r(1) - 0.5) * 8 * size
	wood := func(xx, yy int) {
		if t := is.get(xx, yy); t == EMPTY || t == LEAVES || t == FLOWER || t == BUSH {
			is.set(xx, yy, WOOD)
		}
	}
	trunkX := func(h int) float64 {
		f := float64(h) / float64(height)
		return float64(x) + math.Sin(f*math.Pi*0.9)*sway
	}
	for h := -1; h <= height; h++ {
		f := float64(h) / float64(height)
		hw := base*(1-f*0.75) + 0.4
		if h < 4 {
			hw += float64(4-h) * 0.8 * size
		}
		c := trunkX(h)
		for xx := int(math.Floor(c - hw)); xx <= int(math.Ceil(c+hw)); xx++ {
			wood(xx, s-h)
		}
	}
	n := 2 + int(r(2)*2.99*size)
	for b := 0; b < n; b++ {
		dir := 1.0
		if (b+int(r(3)*2))%2 == 0 {
			dir = -1
		}
		h0 := int(float64(height) * (0.45 + 0.4*g.hash01(sl, 30+b, 72)))
		bx, by := trunkX(h0), float64(s-h0)
		ang := -0.35 - 0.5*g.hash01(sl, 40+b, 72)
		length := size * (10 + 10*g.hash01(sl, 50+b, 72))
		thick := 1.2 * size
		for st := 0.0; st < length; st += 0.6 {
			bx += math.Cos(ang) * 0.6 * dir
			by += math.Sin(ang) * 0.6
			ang -= 0.012
			t := thick * (1 - st/length*0.6)
			for dy := -t; dy <= t; dy++ {
				wood(int(math.Round(bx)), int(math.Round(by+dy)))
			}
		}
		g.leafClump(is, bx, by-2, size*(5+3*g.hash01(sl, 60+b, 72)), sl*7+b)
	}
	tx, ty := trunkX(height), float64(s-height)
	cr := size * (8 + 4*r(5))
	g.leafClump(is, tx, ty-cr*0.3, cr, sl*7+5)
	g.leafClump(is, tx-cr*0.8, ty+cr*0.2, cr*0.65, sl*7+6)
	g.leafClump(is, tx+cr*0.8, ty+cr*0.15, cr*0.65, sl*7+7)
}

func (g gen) leafClump(is *Isle, cx, cy, rad float64, salt int) {
	ri := int(rad*1.5) + 2
	for dy := -ri; dy <= ri; dy++ {
		for dx := -ri; dx <= ri; dx++ {
			xx, yy := int(cx)+dx, int(cy)+dy
			d := math.Hypot(float64(dx)/1.3, float64(dy))
			edge := rad * (0.72 + 0.42*g.noise2(float64(xx), float64(yy), 4, 74+salt%5))
			if d <= edge && is.get(xx, yy) == EMPTY {
				is.set(xx, yy, LEAVES)
			}
		}
	}
}

func (g gen) flowers(is *Isle) {
	for x := 0; x < GW; x++ {
		s := int(is.Surf[x])
		if s < 2 || is.get(x, s) != GRASS || is.get(x, s-1) != EMPTY {
			continue
		}
		switch h := g.hash01(x, 0, 81); {
		case h < 0.05:
			is.set(x, s-1, FLOWER)
		case h < 0.08:
			is.set(x, s-1, BUSH)
			if is.get(x, s-2) == EMPTY {
				is.set(x, s-2, BUSH)
			}
		}
	}
}

// Dry, fairly flat ground with solid rock under it, as close to the middle as possible
func (g gen) findSpawn(is *Isle) {
	ok := func(x int) bool {
		s := is.SurfAt(x)
		if s >= is.Def.Water-4 {
			return false
		}
		for c := x - 8; c <= x+8; c++ {
			if absI(is.SurfAt(c)-s) > 3 {
				return false
			}
			for y := is.SurfAt(c); y < is.SurfAt(c)+10; y++ {
				if !is.Solid(c, y) {
					return false
				}
			}
		}
		return true
	}
	mid := GW / 2
	for d := 0; d < GW/2-20; d++ {
		for _, x := range []int{mid + d, mid - d} {
			if ok(x) {
				is.SpawnX, is.SpawnY = x, is.SurfAt(x)
				return
			}
		}
	}
	is.SpawnX, is.SpawnY = mid, is.SurfAt(mid)
}

// The travel shrine at every island's spawn point: an obsidian arch with a crystal on top,
// and a stone floor under it
func (g gen) shrine(is *Isle) {
	x0, s := is.SpawnX, is.SpawnY
	for x := x0 - 16; x <= x0+16; x++ {
		for y := s - 40; y < s; y++ {
			if t := is.get(x, y); t != EMPTY && t != WATER {
				is.set(x, y, EMPTY) // clear leaves and flowers
			}
		}
		is.set(x, s, BRICK)
	}
	g.arch(is, x0, s, 12, 26)
}

func (g gen) arch(is *Isle, x0, s, rx, ry int) {
	for a := 0.0; a <= math.Pi; a += 0.01 {
		for th := 0.0; th < 2.5; th += 0.5 {
			x := x0 + int(math.Round(math.Cos(a)*(float64(rx)+th)))
			y := s - int(math.Round(math.Sin(a)*(float64(ry)+th)))
			is.set(x, y, OBSIDIAN)
		}
	}
	for y := s - 3; y < s; y++ { // feet
		for _, x := range []int{x0 - rx - 2, x0 - rx - 1, x0 - rx, x0 + rx, x0 + rx + 1, x0 + rx + 2} {
			is.set(x, y, OBSIDIAN)
		}
	}
	for dy := 0; dy < 3; dy++ {
		for dx := -1 + dy/2; dx <= 1-dy/2; dx++ {
			is.set(x0+dx, s-ry-4-dy, CRYSTAL)
		}
	}
}

// Haven's town square: a brick floor, the big travel arch, and torches on posts
func (g gen) plaza(is *Isle) {
	x0, s := is.SpawnX, is.SpawnY
	for x := x0 - 120; x <= x0+120; x++ {
		for y := s - 60; y < s; y++ {
			if t := is.get(x, y); t != EMPTY && t != WATER {
				is.set(x, y, EMPTY)
			}
		}
		for y := s; y < s+3; y++ {
			is.set(x, y, BRICK)
		}
	}
	g.arch(is, x0, s, 18, 36)
	for _, dx := range []int{-110, -80, -50, 50, 80, 110} {
		x := x0 + dx
		for y := s - 9; y < s; y++ {
			is.set(x, y, PLANK)
		}
		is.set(x, s-10, TORCH)
	}
	// low walls at the edges of the square
	for _, dx := range []int{-120, 120} {
		for y := s - 4; y < s; y++ {
			for x := x0 + dx - 2; x <= x0+dx+2; x++ {
				is.set(x, y, BRICK)
			}
		}
	}
}

func absI(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
