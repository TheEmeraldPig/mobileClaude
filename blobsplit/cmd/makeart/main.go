// makeart writes the app icon and Google Play store graphics:
//
//	go run ./cmd/makeart
package main

import (
	"image"
	"image/png"
	"log"
	"os"

	"github.com/TheEmeraldPig/mobileClaude/blobsplit/game"
)

func write(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func main() {
	write("android/app/src/main/res/mipmap-xxxhdpi/ic_launcher.png", game.IconImage(192))
	write("store/icon-512.png", game.IconImage(512))
	write("store/feature-1024x500.png", game.FeatureImage())
}
