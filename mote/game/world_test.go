package game

import "testing"

func chunkSum(cx int) int32 {
	b := genChunk(cx)
	var h int32
	for _, v := range b {
		h = h*31 + int32(v)
	}
	return h
}

// A seed must always build exactly the same chunks. If you change world generation on
// purpose, update these numbers (every seed's world changes with it).
func TestSeedsAreStable(t *testing.T) {
	want := []struct {
		seed string
		cx   int
		sum  int32
	}{{"12345", 0, -628427592}, {"12345", -3, 1492155978}, {"12345", 17, -1171153157}, {"hello", 0, 958037013}, {"hello", 5, -1773515645}}
	for _, w := range want {
		seedN = seedFrom(w.seed)
		a, b := chunkSum(w.cx), chunkSum(w.cx)
		if a != b {
			t.Fatalf("seed %q chunk %d is not repeatable", w.seed, w.cx)
		}
		if a != w.sum {
			t.Errorf("seed %q chunk %d: got %d, want %d", w.seed, w.cx, a, w.sum)
		}
	}
}

// Ores and caves should actually appear
func TestWorldHasOresAndCaves(t *testing.T) {
	seedN = seedFrom("12345")
	count := map[uint8]int{}
	for cx := -4; cx < 4; cx++ {
		for _, v := range genChunk(cx) {
			count[v]++
		}
	}
	for _, b := range []uint8{COAL, COPPER, IRON, GOLD, CRYSTAL, LAVA, CHEST, CORE, BEDROCK} {
		if count[b] == 0 {
			t.Errorf("no block %d generated", b)
		}
	}
	t.Logf("counts: %v", count)
}
