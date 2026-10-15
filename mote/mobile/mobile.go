// Package mobile is the Android entry point. It is bound into an .aar with:
//
//	ebitenmobile bind -target android -javapkg com.theemeraldpig.mote -o android/app/libs/mote.aar ./mobile
//
// The Android activity calls Mobile.setDataDir(getFilesDir()) before the game view starts.
package mobile

import (
	"github.com/TheEmeraldPig/mobileClaude/mote/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

// The game is created on its first frame, after the activity has passed in the data folder.
type lazy struct{ g ebiten.Game }

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

func init() { mobile.SetGame(&lazy{}) }

// SetDataDir tells the game where to keep its settings and save.
func SetDataDir(dir string) { game.DataDir = dir }

// Save is called when the app goes to the background.
func Save() { game.SaveNow() }

// Dummy is needed so gomobile binds this package even without other exported calls.
func Dummy() {}
