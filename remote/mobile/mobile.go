// Package mobile is the Android entry point. It is bound into an .aar with:
//
//	ebitenmobile bind -target android -javapkg com.theemeraldpig.remote -o android/app/libs/remote.aar ./mobile
//
// The Android activity calls Mobile.setDataDir(getFilesDir()) before the game view starts.
package mobile

import (
	"os"

	"github.com/TheEmeraldPig/mobileClaude/remote/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

// The game is created on its first frame, after the activity has passed in the data folder.
type lazy struct{ g ebiten.Game }

func (l *lazy) Update() error {
	if game.Crashed() {
		if game.CrashUpdate() { // Close on the crash screen
			os.Exit(0)
		}
		return nil
	}
	if l.g == nil {
		func() {
			defer game.CatchCrash()
			l.g = game.NewGame()
		}()
		if l.g == nil {
			return nil
		}
	}
	return l.g.Update()
}

func (l *lazy) Draw(screen *ebiten.Image) {
	if game.Crashed() {
		game.DrawCrash(screen)
		return
	}
	if l.g != nil {
		l.g.Draw(screen)
	}
}

func (l *lazy) Layout(w, h int) (int, int) {
	if l.g == nil || game.Crashed() {
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
