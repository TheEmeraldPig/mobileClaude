// Package mmo is the part of Remote shared by the game and the server: island
// generation, petals, creatures, characters and the messages between them. It has no
// graphics, so the server can run it on its own.
package mmo

// Block ids. They are the same numbers as the game's (game/world.go); world_test.go in
// the game checks that they match.
const (
	EMPTY     = 0
	SAND      = 1
	WATER     = 2
	STONE     = 3
	BRICK     = 4
	COAL      = 5
	COPPER    = 6
	IRON      = 7
	GOLD      = 8
	CRYSTAL   = 9
	TORCH     = 10
	DIRT      = 11
	GRASS     = 12
	WOOD      = 13 // tree trunk (you can walk through it)
	LEAVES    = 14
	PLANK     = 15
	GLASS     = 16
	LAVA      = 18
	FLOWER    = 23
	BUSH      = 24
	OBSIDIAN  = 27
	BEDROCK   = 28
	SANDSTONE = 32
	// What mined blocks turn into: they stay put and can't be mined again (until they grow back)
	WORNSTONE   = 33
	PACKEDDIRT  = 34
	DULLCRYSTAL = 35
	OLDWOOD     = 36 // a mined tree trunk (you can walk through it)
)

// Solid blocks stop motes and creatures
func IsSolid(t uint8) bool {
	switch t {
	case SAND, STONE, BRICK, CRYSTAL, DIRT, GRASS, PLANK, GLASS, OBSIDIAN, BEDROCK, SANDSTONE,
		COAL, COPPER, IRON, GOLD, WORNSTONE, PACKEDDIRT, DULLCRYSTAL:
		return true
	}
	return false
}

const (
	Cell = 2    // world px per cell
	GW   = 1792 // island width in cells
	GH   = 1600 // island height in cells
)

// A player's size in world px (10x18 cells)
const (
	PlayerW = 20.0
	PlayerH = 36.0
)
