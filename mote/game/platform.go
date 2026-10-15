package game

import (
	"image"
	"image/color"
)

// Version is set at build time with -ldflags "-X .../game.Version=42". "dev" turns the updater off.
var Version = "dev"

// StartFullscreen reports the saved full screen setting.
func StartFullscreen() bool { return prefs.Full }

// SaveNow writes the save game straight away (used when an app is sent to the background).
func SaveNow() { saveGame() }

// WindowIcons draws the mote as the window/taskbar icon in a few sizes.
func WindowIcons() []image.Image {
	var out []image.Image
	for _, size := range []int{16, 32, 48, 64, 128} {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		rows := playerFrame(false, false, "stand", 0)
		px := size / 16
		if px < 1 {
			px = 1
		}
		ox, oy := (size-sprW*px)/2, (size-sprH*px)/2
		for y, r := range rows {
			for x, ch := range r {
				if ch == '.' {
					continue
				}
				c := pcol[ch]
				for i := 0; i < px; i++ {
					for j := 0; j < px; j++ {
						img.Set(ox+x*px+i, oy+y*px+j, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
					}
				}
			}
		}
		out = append(out, img)
	}
	return out
}
