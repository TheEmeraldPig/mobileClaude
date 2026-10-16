// Package mobile is the Android entry point, bound into an .aar with:
//
//	ebitenmobile bind -target android -javapkg com.theemeraldpig.hop -o android/app/libs/hop.aar ./mobile
package mobile

import (
	"github.com/TheEmeraldPig/mobileClaude/hop/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

// The game is made on its first frame, after the activity has passed in the data folder.
type lazy struct{ g *game.Game }

func (l *lazy) Update() error {
	if l.g == nil {
		l.g = game.NewGame()
	}
	return l.g.Update()
}

func (l *lazy) Draw(screen *ebiten.Image) {
	if l.g != nil {
		l.g.Draw(screen)
	}
}

func (l *lazy) Layout(w, h int) (int, int) {
	if l.g == nil {
		return w, h
	}
	return l.g.Layout(w, h)
}

var root = &lazy{}

func init() { mobile.SetGame(root) }

// SetDataDir tells the game where to keep its save.
func SetDataDir(dir string) { game.DataDir = dir }

// Save writes the save file; the activity calls it on pause.
func Save() {
	if root.g != nil {
		root.g.Save()
	}
}
