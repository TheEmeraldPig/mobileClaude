package game

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// On Android the game updates before it knows the screen size. With rare blobs on the
// board (they sparkle) that used to divide by zero.
func TestUpdateBeforeLayout(t *testing.T) {
	os.Setenv("BLOB_NOAUDIO", "1")
	os.Setenv("BLOB_FRESH", "1")
	DataDir = t.TempDir()
	g := NewGame()
	g.blobs = nil
	for i := 0; i < 64; i++ {
		g.blobs = append(g.blobs, blob{m: (i * 37) % (1 << nMut) &^ 1, mt: -10, squash: -10})
	}
	for i := 0; i < 120; i++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	g.trySplit(true)
	g.sell()
	g.Draw(ebiten.NewImage(1, 1))
	for _, size := range [][2]int{{0, 0}, {800, 360}, {300, 600}} {
		g.Layout(size[0], size[1])
		g.lay()
		for i := 0; i < 4; i++ {
			g.trySplit(true)
			g.cool = 99
			_ = g.Update()
		}
	}
}
