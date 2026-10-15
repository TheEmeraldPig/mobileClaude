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

// The same seed must build the same chunks as the web version.
func TestSeedMatchesWeb(t *testing.T) {
	// Checksums taken from the web version (sandbox/index.html) for the same seeds and chunks
	want := []struct {
		seed string
		cx   int
		sum  int32
	}{{"12345", 0, 1354437337}, {"12345", -3, 1455508183}, {"12345", 17, 1036798767}, {"hello", 0, 319047322}, {"hello", 5, -1985747448}}
	for _, w := range want {
		seedN = seedFrom(w.seed)
		if got := chunkSum(w.cx); got != w.sum {
			t.Errorf("seed %q chunk %d: got %d, want %d", w.seed, w.cx, got, w.sum)
		}
	}
}
