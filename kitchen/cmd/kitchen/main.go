// Command kitchen runs Kitchen Clash on a computer.
package main

import (
	"log"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/kitchen/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if d := os.Getenv("KITCHEN_DATA"); d != "" {
		game.DataDir = d
	} else if d, err := os.UserConfigDir(); err == nil {
		game.DataDir = d + "/KitchenClash"
		os.MkdirAll(game.DataDir, 0o755)
	}
	g := game.NewGame()
	ebiten.SetWindowTitle("Kitchen Clash")
	ebiten.SetWindowSize(450, 900)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(g)
	g.Save()
	if err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
