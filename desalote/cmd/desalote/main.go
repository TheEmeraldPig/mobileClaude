// Desalote: a multiplayer poker-hand scoring card game.
// Desktop entry point (Windows, Linux, macOS). Build with:
//
//	go build -ldflags "-H windowsgui -X github.com/TheEmeraldPig/mobileClaude/desalote/game.Version=42" -o Desalote.exe ./cmd/desalote
//
// Add -tags steam for Steam builds (Steam handles updates, so the built-in updater is off).
package main

import (
	"log"

	"github.com/TheEmeraldPig/mobileClaude/desalote/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := game.NewGame()
	ebiten.SetWindowTitle("Desalote")
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
