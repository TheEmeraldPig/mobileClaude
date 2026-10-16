// Command dig runs Deep Dig on a computer.
package main

import (
	"log"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/dig/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if d := os.Getenv("DIG_DATA"); d != "" {
		game.DataDir = d
	} else if d, err := os.UserConfigDir(); err == nil {
		game.DataDir = d + "/DeepDig"
		os.MkdirAll(game.DataDir, 0o755)
	}
	g := game.NewGame()
	ebiten.SetWindowTitle("Deep Dig")
	ebiten.SetWindowSize(1100, 540)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(g)
	g.Save()
	if err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
