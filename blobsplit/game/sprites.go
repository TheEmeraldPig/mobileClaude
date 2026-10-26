package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const blobPx = 18 // blob sprites are 18x18 pixel art, drawn scaled up

// blobImgs[rarity][0] has open eyes, [1] is blinking.
var blobImgs [][2]*ebiten.Image

func rgb(c uint32) color.RGBA {
	return color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
}

func mul(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, v))) }
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), c.A}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 1) * 6
	i := math.Floor(h)
	f := h - i
	p, q, t := v*(1-s), v*(1-s*f), v*(1-s*(1-f))
	var r, g, b float64
	switch int(i) {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

var outline = rgb(0x1a0f2e)

func initSprites() {
	blobImgs = make([][2]*ebiten.Image, len(rarities))
	for r := range rarities {
		for b := 0; b < 2; b++ {
			blobImgs[r][b] = ebiten.NewImageFromImage(blobSprite(r, b == 1))
		}
	}
	for name, rows := range iconArt {
		icons[name] = ebiten.NewImageFromImage(pixelArt(rows))
	}
	icons["dna"] = ebiten.NewImageFromImage(dnaArt())
}

// blobSprite draws a round blob with googly eyes, shaded like the UI: light top-left
// edge, dark bottom-right edge, dark outline.
func blobSprite(r int, blink bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, blobPx, blobPx))
	const cx, cy, rad = 9.0, 9.5, 8.2
	base := rgb(rarities[r].Body)
	for y := 0; y < blobPx; y++ {
		for x := 0; x < blobPx; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(dx, dy)
			if d > rad {
				continue
			}
			if d > rad-1.15 {
				img.SetRGBA(x, y, outline)
				continue
			}
			c := base
			switch rarities[r].Name {
			case "Rainbow":
				c = hsv(float64(y)/14+float64(x)/40, 0.7, 1)
			case "Toxic":
				if (x*7+y*13)%11 == 0 && d < rad-2.5 {
					c = mul(base, 0.75)
				}
			case "Cosmic":
				if (x*31+y*17)%23 == 0 {
					c = rgb(0xfff3b0)
				} else if (x*11+y*29)%37 == 0 {
					c = rgb(0x9fd8ff)
				} else {
					c = mix(base, rgb(0x4a2a8a), float64(y)/blobPx)
				}
			case "Frost":
				if dy < -4.5 && (x+y)%3 != 0 {
					c = rgb(0xeafcff)
				}
			}
			nx, ny := dx/math.Max(d, 0.01), dy/math.Max(d, 0.01)
			if d > rad-2.6 {
				if nx+ny < -0.5 {
					c = mix(c, color.RGBA{255, 255, 255, 255}, 0.45)
				} else if nx+ny > 0.5 {
					c = mul(c, 0.62)
				}
			} else if dy > 3 {
				c = mul(c, 0.88)
			}
			img.SetRGBA(x, y, c)
		}
	}
	// a shine spot top-left
	img.SetRGBA(4, 5, color.RGBA{255, 255, 255, 255})
	img.SetRGBA(5, 4, color.RGBA{255, 255, 255, 255})

	eye := func(ex, ey float64) {
		if blink {
			for x := int(ex - 2); x <= int(ex+2); x++ {
				img.SetRGBA(x, int(ey), outline)
			}
			return
		}
		for y := int(ey - 4); y <= int(ey+4); y++ {
			for x := int(ex - 4); x <= int(ex+4); x++ {
				d := math.Hypot(float64(x)+0.5-ex, float64(y)+0.5-ey)
				pd := math.Hypot(float64(x)+0.5-ex-0.7, float64(y)+0.5-ey-0.6)
				switch {
				case pd < 1.4:
					img.SetRGBA(x, y, outline)
				case d < 2.75:
					img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
				case d < 3.6:
					img.SetRGBA(x, y, outline)
				}
			}
		}
	}
	eye(5.9, 8.4)
	eye(12.3, 8.4)
	// little mouth
	for _, p := range [][2]int{{8, 13}, {9, 14}, {10, 13}} {
		img.SetRGBA(p[0], p[1], outline)
	}
	if rarities[r].Name == "Heart" {
		heart := []string{
			".o.o.",
			"oHoHo",
			"oHHHo",
			".oHo.",
			"..o..",
		}
		for y, row := range heart {
			for x, ch := range row {
				switch ch {
				case 'o':
					img.SetRGBA(13+x, y, outline)
				case 'H':
					img.SetRGBA(13+x, y, rgb(0xff2a5a))
				}
			}
		}
	}
	return img
}

// Small pixel icons for the UI. Letters pick colours from iconPal.
var icons = map[string]*ebiten.Image{}

var iconPal = map[rune]color.RGBA{
	'o': outline,
	'g': rgb(0x5fe8ff), 'G': rgb(0x2a9fd6), 'w': rgb(0xffffff),
	'y': rgb(0xffc94a), 'Y': rgb(0xd88a1c), 'b': rgb(0x8a6cf0), 'B': rgb(0x5a3fb8),
	'r': rgb(0xe8414f), 'R': rgb(0x9a2233), 'n': rgb(0x7a4a2a), 'N': rgb(0x4e2d18),
	'l': rgb(0x8ef06a), 'L': rgb(0x3fa83a),
}

var iconArt = map[string][]string{
	"gem": {
		"..oooooo..",
		".owggggGo.",
		"owwgggGGGo",
		"oooooooooo",
		".ogggGGGo.",
		"..oggGGo..",
		"...ogGo...",
		"....oo....",
	},
	"bag": {
		"...oooo...",
		"..oy..yo..",
		"..oy..yo..",
		".oooooooo.",
		"obbbbbbBBo",
		"obbbyybBBo",
		"obbyYYyBBo",
		"obbbyybBBo",
		"obbbbbBBBo",
		".oooooooo.",
	},
	"book": {
		"oooooooooo",
		"orrrrrrrwo",
		"orryyrrrwo",
		"oryrryrrwo",
		"orryyrrrwo",
		"orrrrrrrwo",
		"oRRRRRRRwo",
		"oooooooooo",
	},
	"rebirth": {
		"...oooo...",
		"..olllLo..",
		".olo..oLo.",
		"olo....oo.",
		"olo...olLo",
		"oLo....oo.",
		".oLo..oLo.",
		"..oLLLLo..",
		"...oooo...",
	},
	"sound": {
		"....o.....",
		"...oo..o..",
		"oooyo...o.",
		"oyyyo.o.o.",
		"oyyyo.o.o.",
		"oooyo...o.",
		"...oo..o..",
		"....o.....",
	},
	"mute": {
		"....o.....",
		"...oo.....",
		"oooyo.r.r.",
		"oyyyo..r..",
		"oyyyo.r.r.",
		"oooyo.....",
		"...oo.....",
		"....o.....",
	},
	"token": {
		"..oooo..",
		".oyyyyo.",
		"oyywwyYo",
		"oywyyyYo",
		"oywyyyYo",
		"oyyyyYYo",
		".oYYYYo.",
		"..oooo..",
	},
	"gear": {
		"...oo...",
		".oowwoo.",
		".owwwwo.",
		"owwoowwo",
		"owwoowwo",
		".owwwwo.",
		".oowwoo.",
		"...oo...",
	},
	"tree": {
		"...oooo...",
		"..ollllo..",
		".ollLlllo.",
		"olllllLllo",
		"olLllllllo",
		".ollllLlo.",
		"..oonnoo..",
		"....on....",
		"...onNo...",
		"..oooooo..",
	},
	"star": {
		"....oo....",
		"....oyo...",
		"...oyyo...",
		"oooyyyyooo",
		"oyyyyyyyYo",
		".oyyyyyYo.",
		"..oyyyYo..",
		".oyyooYYo.",
		"oyYo..oYYo",
		"ooo....ooo",
	},
	"heart": {
		".oo.oo.",
		"orroRro",
		"orrrrRo",
		".orrRo.",
		"..oRo..",
		"...o...",
	},
	"note": {
		"....oooo",
		"...oyyyo",
		"...oyooo",
		"...oyo..",
		"...oyo..",
		".oooyo..",
		"oyyyyo..",
		"oyyyyo..",
		".oooo...",
	},
	"nonote": {
		"....oooo",
		"...oyyyo",
		"...oyooo",
		"...oyo..",
		"r..oyo.r",
		".roooyr.",
		"oyrryr..",
		"oyyryro.",
		".rooo..r",
	},
	"quest": {
		".oooooooo.",
		"oyyyyyyyYo",
		"oywwwwwwYo",
		"oyyyyyyyYo",
		"oywwwwyyYo",
		"oyyyyyyyYo",
		"oywwwwwwYo",
		"oyyyyyyyYo",
		"oywwwyyyYo",
		"oYYYYYYYYo",
		".oooooooo.",
	},
	"lock": {
		"..oooo..",
		".o....o.",
		".o....o.",
		"oooooooo",
		"oyyyyyYo",
		"oyyooyYo",
		"oyyyyyYo",
		"oooooooo",
	},
}

func pixelArt(rows []string) *image.RGBA {
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w, len(rows)))
	for y, row := range rows {
		for x, ch := range row {
			if c, ok := iconPal[ch]; ok {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return img
}

// dnaArt is the DNA icon: a double helix, two strands (cyan and pink) twisting round
// each other with gold base-pair rungs between them, the strand in front drawn last.
func dnaArt() *image.RGBA {
	const w, h = 11, 18
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	strand := func(y int, shift float64) (x int, front bool) {
		a := float64(y)*math.Pi/7 + shift
		return int(math.Round(5 + 3.6*math.Sin(a))), math.Cos(a) > 0
	}
	cols := []color.RGBA{rgb(0x5fe8ff), rgb(0xff6fb5)}
	dark := []color.RGBA{rgb(0x2a9fd6), rgb(0xc8407a)}
	// rungs first, between the strands
	for y := 1; y < h-1; y += 2 {
		x0, _ := strand(y, 0)
		x1, _ := strand(y, math.Pi)
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		for x := x0 + 1; x < x1; x++ {
			img.SetRGBA(x, y, rgb(0xffc94a))
		}
	}
	// back halves, then front halves, each with a dark outline
	for pass := 0; pass < 2; pass++ {
		for s := 0; s < 2; s++ {
			for y := 0; y < h; y++ {
				x, front := strand(y, float64(s)*math.Pi)
				if front != (pass == 1) {
					continue
				}
				c := cols[s]
				if !front {
					c = dark[s]
				}
				for _, d := range [][2]int{{-1, 0}, {2, 0}} {
					if xx := x + d[0]; xx >= 0 && xx < w && img.RGBAAt(xx, y).A == 0 {
						img.SetRGBA(xx, y, outline)
					}
				}
				img.SetRGBA(x, y, c)
				if x+1 < w {
					img.SetRGBA(x+1, y, c)
				}
			}
		}
	}
	return img
}
