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
	}{{"12345", 0, 287081024}, {"12345", -3, 832534417}, {"12345", 17, -197295708}, {"hello", 0, 1212825273}, {"hello", 5, 525584777}}
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

// Villages, sky islands, ruins and their back walls should show up, and walls must be
// repeatable too
func TestStructuresAppear(t *testing.T) {
	seedN = seedFrom("12345")
	doors, sky, walls := 0, 0, 0
	for cx := -40; cx < 40; cx++ {
		b := genChunk(cx)
		for i, v := range b {
			if v == DOOR {
				doors++
			}
			if v == GRASS && i/CW < 130 {
				sky++
			}
		}
		w1, w2 := genWalls(cx), genWalls(cx)
		for i := range w1 {
			if w1[i] != w2[i] {
				t.Fatalf("walls of chunk %d are not repeatable", cx)
			}
			if w1[i] != 0 {
				walls++
			}
		}
	}
	if doors == 0 || sky == 0 || walls == 0 {
		t.Errorf("structures missing: doors %d, sky island grass %d, walls %d", doors, sky, walls)
	}
}

// Sand never generates on top of air or liquid (it would fall as soon as it loads)
func TestNoFloatingSand(t *testing.T) {
	seedN = seedFrom("12345")
	for cx := -20; cx < 20; cx++ {
		b := genChunk(cx)
		for i := 0; i < len(b)-CW; i++ {
			if b[i] == SAND && (b[i+CW] == EMPTY || b[i+CW] == WATER || b[i+CW] == LAVA) {
				t.Fatalf("floating sand in chunk %d at (%d, %d)", cx, i%CW, i/CW)
			}
		}
	}
}

// Each ore clump is one kind of ore
func TestOreClumpsArePure(t *testing.T) {
	seedN = seedFrom("12345")
	for rx := -30; rx < 30; rx++ {
		for ry := 0; ry <= H/oreRegion; ry++ {
			if _, _, r, o, ok := oreClump(rx, ry); ok && (r <= 0 || !isOre(o)) {
				t.Fatalf("bad clump at %d,%d", rx, ry)
			}
		}
	}
}
