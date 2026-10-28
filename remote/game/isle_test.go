package game

import (
	"testing"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// The game draws islands with its own block ids; they must be the shared ones
func TestBlockIDsMatch(t *testing.T) {
	pairs := [][2]int{{EMPTY, mmo.EMPTY}, {SAND, mmo.SAND}, {WATER, mmo.WATER}, {STONE, mmo.STONE}, {BRICK, mmo.BRICK},
		{COAL, mmo.COAL}, {COPPER, mmo.COPPER}, {IRON, mmo.IRON}, {GOLD, mmo.GOLD}, {CRYSTAL, mmo.CRYSTAL},
		{TORCH, mmo.TORCH}, {DIRT, mmo.DIRT}, {GRASS, mmo.GRASS}, {WOOD, mmo.WOOD}, {LEAVES, mmo.LEAVES},
		{PLANK, mmo.PLANK}, {GLASS, mmo.GLASS}, {LAVA, mmo.LAVA}, {FLOWER, mmo.FLOWER}, {BUSH, mmo.BUSH},
		{OBSIDIAN, mmo.OBSIDIAN}, {BEDROCK, mmo.BEDROCK}, {SANDSTONE, mmo.SANDSTONE},
		{WORNSTONE, mmo.WORNSTONE}, {PACKEDDIRT, mmo.PACKEDDIRT}, {DULLCRYSTAL, mmo.DULLCRYSTAL}, {OLDWOOD, mmo.OLDWOOD}}
	for _, p := range pairs {
		if p[0] != p[1] {
			t.Errorf("block id %d in the game is %d in mmo", p[0], p[1])
		}
		if isSolid(uint8(p[0])) != mmo.IsSolid(uint8(p[1])) {
			t.Errorf("block %d: the game and the server disagree on whether it's solid", p[0])
		}
	}
	if W != mmo.GW || H != mmo.GH || CELL != mmo.Cell {
		t.Errorf("world size %dx%d (cell %d) doesn't match the islands", W, H, CELL)
	}
	if float64(sprW*CELL) != mmo.PlayerW || float64(sprH*CELL) != mmo.PlayerH {
		t.Errorf("the character's size doesn't match the server's")
	}
}
