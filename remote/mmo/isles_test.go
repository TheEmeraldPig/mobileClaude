package mmo

import (
	"hash/fnv"
	"testing"
)

// The game and the server must build the same islands. These checksums pin each
// island's terrain: if one changes on purpose, update it here (every player's island
// changes too).
var isleSums = map[string]uint32{"haven": 131437615, "meadow": 1672584056, "quarry": 849878547, "hollow": 4043564861, "ember": 1146150333}

func TestIslesAreStable(t *testing.T) {
	for _, d := range Isles {
		a, b := BuildIsle(d), BuildIsle(d)
		ha, hb := fnv.New32a(), fnv.New32a()
		ha.Write(a.Cells)
		hb.Write(b.Cells)
		if ha.Sum32() != hb.Sum32() {
			t.Fatalf("%s: two builds differ", d.ID)
		}
		if want, ok := isleSums[d.ID]; ok && want != ha.Sum32() {
			t.Errorf("%s: checksum %d, want %d (did island generation change?)", d.ID, ha.Sum32(), want)
		}
		t.Logf("%s: %d", d.ID, ha.Sum32())
		// the spawn point stands on solid ground with room for a mote above
		x, y := a.SpawnPos()
		if !a.BoxFree(x, y, PlayerW, PlayerH) {
			t.Errorf("%s: no room at the spawn point", d.ID)
		}
		if !a.Solid(a.SpawnX, a.SpawnY) {
			t.Errorf("%s: the spawn point isn't on the ground", d.ID)
		}
	}
}
