package game

import (
	"image"
	"image/color"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Version is set at build time with -ldflags "-X .../game.Version=42". "dev" turns the updater off.
var Version = "dev"

// StartFullscreen reports the saved full screen setting.
func StartFullscreen() bool { return prefs.Full }

// SaveNow writes the settings straight away (used when an app is sent to the background).
// Characters are saved by the server.
func SaveNow() { savePrefs() }

// WindowIcons draws the default character's head as the window/taskbar icon in a few sizes.
func WindowIcons() []image.Image {
	var out []image.Image
	pix := bodyPixels(mmo.DefaultLook, pose{face: 1, legs: "stand", swing: -1})
	for _, size := range []int{16, 32, 48, 64, 128} {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		// the head and antennae: rows -4..6 of the sprite, 11 rows
		k := max(1, size/12)
		ox, oy := (size-sprW*k)/2, (size-11*k)/2
		for _, q := range pix {
			if q.y < -4 || q.y > 6 {
				continue
			}
			for i := 0; i < k; i++ {
				for j := 0; j < k; j++ {
					img.Set(ox+q.x*k+i, oy+(q.y+4)*k+j, color.RGBA{uint8(q.c[0]), uint8(q.c[1]), uint8(q.c[2]), 255})
				}
			}
		}
		out = append(out, img)
	}
	return out
}
