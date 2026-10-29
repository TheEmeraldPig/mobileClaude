package game

import (
	"image"
	"image/color"
)

// Version is set at build time with -ldflags "-X .../game.Version=42". "dev" turns the updater off.
var Version = "dev"

// StartFullscreen reports the saved full screen setting.
func StartFullscreen() bool { return prefs.Full }

// SaveNow writes the settings straight away (used when an app is sent to the background).
func SaveNow() { savePrefs() }

// WindowIcons draws a little card (an ace with a gold pip) as the window icon.
func WindowIcons() []image.Image {
	var out []image.Image
	for _, size := range []int{16, 32, 48, 64, 128, 256} {
		out = append(out, iconImage(size))
	}
	return out
}

// The icon at one size: a violet card back with a gold diamond, in the pixel style
func iconImage(size int) *image.RGBA {
	const n = 16
	rows := []string{
		"................",
		"..oooooooooooo..",
		".ollllllllllllo.",
		".olbbbbbbbbbbso.",
		".olbbbbttbbbbso.",
		".olbbbttttbbbso.",
		".olbbttggttbbso.",
		".olbttggggttbso.",
		".olbttggggttbso.",
		".olbbttggttbbso.",
		".olbbbttttbbbso.",
		".olbbbbttbbbbso.",
		".olbbbbbbbbbbso.",
		".ossssssssssssso",
		"..oooooooooooooo",
		"................",
	}
	pal := map[byte]color.RGBA{'o': colO, 'l': colL, 'b': colB, 's': colS, 't': colT, 'g': colGoldHi}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if c, ok := pal[rows[y*n/size][x*n/size]]; ok {
				img.Set(x, y, c)
			}
		}
	}
	return img
}
