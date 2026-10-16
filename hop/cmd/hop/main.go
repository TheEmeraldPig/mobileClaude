// Command hop runs Orbit Hop on a computer.
package main

import (
	"log"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/hop/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if d := os.Getenv("HOP_DATA"); d != "" {
		game.DataDir = d
	} else if d, err := os.UserConfigDir(); err == nil {
		game.DataDir = d + "/OrbitHop"
		os.MkdirAll(game.DataDir, 0o755)
	}
	g := game.NewGame()
	ebiten.SetWindowTitle("Orbit Hop")
	ebiten.SetWindowSize(1100, 540)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(g)
	g.Save()
	if err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
