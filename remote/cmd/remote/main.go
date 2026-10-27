// Mote: a falling-sand pixel game with a little levitating creature.
// Desktop entry point (Windows, Linux, macOS). Build with:
//
//	go build -ldflags "-H windowsgui -X github.com/TheEmeraldPig/mobileClaude/remote/game.Version=42" -o Remote.exe ./cmd/remote
//
// Add -tags steam for Steam builds (Steam handles updates, so the built-in updater is off).
package main

import (
	"log"

	"github.com/TheEmeraldPig/mobileClaude/remote/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := game.NewGame()
	ebiten.SetWindowTitle("Remote")
	ebiten.SetWindowSize(1280, 720)
	if w, h := game.WindowSizeOverride(); w > 0 {
		ebiten.SetWindowSize(w, h)
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(game.StartFullscreen())
	ebiten.SetWindowIcon(game.WindowIcons())
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
