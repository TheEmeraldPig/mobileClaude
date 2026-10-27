package game

import "testing"

func loadTestWorld(seed string) {
	initWorld(400)
	seedN = seedFrom(seed)
	baseChunk = -NCH / 2
	for i := 0; i < NCH; i++ {
		loadSlot(i)
	}
	player.x, player.y = float64(W/2*CELL), float64(200*CELL)
}

// The light a world cell gets must not depend on where the camera is (otherwise the
// lighting flickers as you move)
func TestLightIgnoresCamera(t *testing.T) {
	loadTestWorld("12345")
	worldTime = 0.3
	const vw, vh = 200, 120
	ox, oy := W/2-100, surfaceAt(baseChunk*CW+W/2)-60
	snap := func(dx, dy int) map[[2]int]uint16 {
		computeLight(ox+dx, oy+dy, vw, vh)
		m := map[[2]int]uint16{}
		for y := oy + 10; y < oy+vh-10; y++ {
			for x := ox + 10; x < ox+vw-10; x++ {
				m[[2]int{x, y}] = light[(y-lightY0)*lightW+x-lightX0]
			}
		}
		return m
	}
	a := snap(0, 0)
	for _, d := range [][2]int{{0, 1}, {1, 0}, {3, -2}} {
		b := snap(d[0], d[1])
		diff := 0
		for k, v := range a {
			if b[k] != v {
				diff++
			}
		}
		if diff > 0 {
			t.Errorf("camera moved by %v: %d of %d cells changed their light", d, diff, len(a))
		}
	}
}

// As the sun moves a tiny bit, the light should only change a tiny bit (no cells
// snapping between sun and shade)
func TestSunMovesSmoothly(t *testing.T) {
	loadTestWorld("12345")
	const vw, vh = 200, 120
	ox, oy := W/2-100, surfaceAt(baseChunk*CW+W/2)-60
	worldTime = 0.4
	computeLight(ox, oy, vw, vh)
	a := append([]uint16(nil), light...)
	worldTime = 0.401
	computeLight(ox, oy, vw, vh)
	worst := 0
	for i := range a {
		worst = max(worst, abs(int(light[i])-int(a[i])))
	}
	t.Logf("worst change %d", worst)
	if worst > 80 {
		t.Errorf("a tiny step of the sun changed a cell's light by %d of %d", worst, lightFine)
	}
}

// A creature that dies leaves its loot behind (it used to vanish with it)
func TestCreaturesDropLoot(t *testing.T) {
	loadTestWorld("12345")
	cx := W / 2
	cy := surfaceAt(baseChunk*CW+cx) - 20
	player.x, player.y = float64((cx+40)*CELL), float64(cy*CELL)
	e := newFoe(entGloom, cx, cy)
	e.hp = 0
	ents = []*Entity{e}
	updateEntities()
	coins := 0
	for _, it := range ents {
		if it.kind == entItem && it.item == "coin" {
			coins += it.n
		}
	}
	if coins == 0 {
		t.Errorf("a gloomling died and dropped no coins (ents: %d)", len(ents))
	}
	ents = nil
}
