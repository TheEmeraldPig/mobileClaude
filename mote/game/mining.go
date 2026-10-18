package game

import (
	"math"
	"math/rand"
)

// Mining: every block has a hardness, and pickaxes have a tier and a power. Holding a
// pickaxe over blocks wears them down a little each frame (cracks show how far along
// they are); harder blocks need more passes, better pickaxes cut faster, and some
// blocks need a strong enough pickaxe at all. Right-click (or the Back button on
// phones) mines background walls instead.

// Pickaxe tiers: 1 stone, 2 copper, 3 iron, 4 gold, 5 core
var pickTier = map[string]int{"pickaxe": 1, "copperpick": 2, "ironpick": 3, "goldpick": 4, "corepick": 5}
var pickPower = map[string]float64{"pickaxe": 1, "copperpick": 1.6, "ironpick": 2.4, "goldpick": 3.4, "corepick": 5}

var tierNames = []string{"", "stone", "copper", "iron", "gold", "core"}

func isPick(id string) bool { return pickTier[id] > 0 }

// How many "hits" a block takes (a stone pickaxe does 1 per frame)
func hardness(t uint8) float64 {
	switch t {
	case LEAVES, FLOWER, BUSH, CROP1, CROP2, CROP3, TORCH:
		return 0.5
	case DIRT, GRASS, SAND, SOIL:
		return 3
	case WOOD, PLANK:
		return 5
	case GLASS, GRAVE:
		return 6
	case SANDSTONE:
		return 8
	case STONE, BRICK:
		return 10
	case COAL:
		return 12
	case COPPER:
		return 16
	case IRON, CRYSTAL:
		return 22
	case GOLD:
		return 28
	case OBSIDIAN:
		return 45
	case CORE:
		return 60
	}
	return 10
}

// The pickaxe tier a block needs
func tierNeed(t uint8) int {
	switch t {
	case IRON, GOLD, CRYSTAL:
		return 2
	case OBSIDIAN, CORE:
		return 3
	}
	return 1
}

// Wear on cells, by grid index (blocks) and grid index + wallKey (walls)
const wallKey = 1 << 30

type wear struct {
	dmg  float64
	last int // frame of the last hit
}

var mineDmg = map[int]*wear{}

var tooHardAt = -1000

// Forget wear when the loaded window slides, and slowly let old cracks heal
func clearWear() { mineDmg = map[int]*wear{} }

func healWear() {
	if frame%30 != 0 {
		return
	}
	for k, w := range mineDmg {
		if frame-w.last > 240 {
			delete(mineDmg, k)
		}
	}
}

// Wear down one cell; true when it breaks
func hitCell(key int, hard, power float64) bool {
	w := mineDmg[key]
	if w == nil {
		w = &wear{}
		mineDmg[key] = w
	}
	w.dmg += power
	w.last = frame
	if w.dmg >= hard {
		delete(mineDmg, key)
		return true
	}
	return false
}

// How cracked a block is, 0..1
func crackAt(i int) float64 {
	w := mineDmg[i]
	if w == nil {
		return 0
	}
	return math.Min(1, w.dmg/hardness(grid[i]))
}

// A fixed noise per cell: lower values crack first, so cracks grow in from the edges
func crackPattern(x, y int) float64 {
	return 0.15 + 0.85*float64(hash32(x, y, 333)%1000)/1000
}

// One frame of mining blocks with a pickaxe around the aim
func mineBlocks(it *Item) {
	tier, power := pickTier[it.id], pickPower[it.id]
	mineObjects(digRadius(it))
	got := map[string]int{}
	chest, core, door := [2]int{-1, -1}, 0, [2]int{-1, -1}
	tooHard := 0
	eachInCircle(digRadius(it), func(x, y, i int) {
		t := grid[i]
		switch t {
		case EMPTY, WATER, LAVA, BEDROCK, DOOROPEN:
			if t == DOOROPEN {
				door = [2]int{x, y}
			}
			return
		case CHEST:
			chest = [2]int{x, y}
			return
		case DOOR:
			door = [2]int{x, y}
			return
		case CORE:
			if !bossDown {
				core = -1000
				return
			}
		}
		if tier < tierNeed(t) {
			tooHard = tierNeed(t)
			return
		}
		// the middle of the circle cuts faster than the edge
		d := math.Hypot(float64(x-aimX), float64(y-aimY)) / float64(digRadius(it)+1)
		if !hitCell(i, hardness(t), power*(1.15-0.5*d)) {
			if rand.Float64() < 0.004*power {
				burst(1, float64(x), float64(y), blockRGB[t], 0.6, 0.6, 12)
			}
			return
		}
		if t == CORE {
			grid[i] = EMPTY
			core++
			return
		}
		name := blockItem[t]
		if name != "" {
			got[name]++
		}
		if t == CROP3 {
			got["seeds"] += 1 + rand.Intn(2)
		}
		if t == LEAVES && rand.Float64() < 0.08 {
			got["seeds"]++
		}
		grid[i] = EMPTY
		paint[i] = 0
		markDirty(x)
		if rand.Float64() < 0.25 {
			burst(1, float64(x), float64(y), blockRGB[t], 0.8, 0.8, 18)
		}
	})
	if got["leaves"] > 0 { // leaves mostly crumble away
		got["leaves"] = (got["leaves"] + 3) / 4
	}
	for name, n := range got {
		giveItem(name, n)
	}
	if (len(got) > 0 && useTick%4 == 0) || useTick%9 == 0 {
		sfx("dig")
	}
	if tooHard > 0 && frame-tooHardAt > 150 {
		tooHardAt = frame
		toast("That needs a " + tierNames[tooHard] + " pickaxe or better")
	}
	if chest[0] >= 0 {
		openChest(chest[0], chest[1])
	}
	if door[0] >= 0 {
		if doorHits++; doorHits > 20 { // hold the pickaxe on a door to take it down
			removeDoor(door[0], door[1])
			doorHits = 0
		}
	} else {
		doorHits = 0
	}
	switch {
	case core > 0:
		coreDust += core
		if coreDust >= 10 {
			giveItem("coreshard", coreDust/10)
			coreDust %= 10
		}
		if useTick%6 == 0 {
			burst(3, float64(aimX), float64(aimY), blockRGB[CORE], 1.4, 1.4, 24)
		}
	case core < 0 && useTick%40 == 1:
		toast("The Warden's power still holds the Core")
	}
	if useTick%20 == 1 {
		hitCreatures(aimX, aimY, digRadius(it), it.dmg)
	}
}

// One frame of mining background walls (right-click, or the Back toggle on phones)
func mineWalls(it *Item) {
	power := pickPower[it.id]
	if it.id == "hammer" {
		power = 2
	}
	n := 0
	eachInCircle(digRadius(it), func(x, y, i int) {
		if wall[i] == 0 {
			return
		}
		if !hitCell(i+wallKey, hardness(wall[i])*0.6, power) {
			return
		}
		giveItem(wallItem[wall[i]], 1)
		wall[i], wpaint[i] = 0, 0
		markDirty(x)
		n++
	})
	if n > 0 || useTick%9 == 0 {
		if useTick%4 == 0 {
			sfx("wall")
		}
	}
}

var (
	mineBack   bool // phones: the Back toggle on the build bar while holding a pickaxe
	backHeld   bool // PC: right mouse button held with a pickaxe
	rightOnObj bool // this right-click went to a station or door
)
