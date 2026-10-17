// Command deck runs Deckbound on a computer.
package main

import (
	"log"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/deck/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if d := os.Getenv("DECK_DATA"); d != "" {
		game.DataDir = d
	} else if d, err := os.UserConfigDir(); err == nil {
		game.DataDir = d + "/Deckbound"
		os.MkdirAll(game.DataDir, 0o755)
	}
	g := game.NewGame()
	ebiten.SetWindowTitle("Deckbound")
	ebiten.SetWindowSize(450, 900)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(g)
	g.Save()
	if err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
