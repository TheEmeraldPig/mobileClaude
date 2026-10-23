// Blob Split desktop entry point, for playing and testing on a PC.
package main

import (
	"log"

	"github.com/TheEmeraldPig/mobileClaude/blobsplit/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := game.NewGame()
	ebiten.SetWindowTitle("Blob Split")
	ebiten.SetWindowSize(1280, 640)
	if w, h := game.WindowSize(); w > 0 {
		ebiten.SetWindowSize(w, h)
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
