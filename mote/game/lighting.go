package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Coloured, smooth lighting.
//
// computeLight (render.go) flood-fills light levels and remembers which kind of source
// each cell's light came from. Here that becomes an RGB light map: each cell gets its
// level (through lightCurve) times the source's colour, blurred so colours blend, and
// drawn over the world with a multiply blend and linear filtering, so light falls off
// in smooth gradients instead of cell by cell.
//
// Things that glow (lava, flames, eyes, the mote...) are drawn on an "emissive" layer
// after the light, so darkness never hides them. Point lights also add a soft coloured
// haze on top, the glow you see around a torch.

const (
	srcNone = iota
	srcSky
	srcTorch
	srcCrystal
	srcLava
	srcCore
	srcFurnace
	srcMote
	srcLantern
	srcWisp
	srcWarden
	srcSun // direct sunlight (or moonlight); srcSky is the softer light from the open sky
	nSrc
)

// Light colours (0..1)
var lightTints = [nSrc][3]float64{
	srcTorch:   {1.0, 0.70, 0.40},
	srcCrystal: {0.50, 0.90, 1.0},
	srcLava:    {1.0, 0.42, 0.18},
	srcCore:    {1.0, 0.50, 0.90},
	srcFurnace: {1.0, 0.58, 0.28},
	srcMote:    {0.86, 0.80, 1.0},
	srcLantern: {1.0, 0.92, 0.74},
	srcWisp:    {0.55, 0.85, 1.0},
	srcWarden:  {1.0, 0.45, 0.85},
}

var (
	shadeCool   = [3]float64{0.48, 0.56, 0.92}  // sky light in the shade
	ambientDark = [3]float64{0.035, 0.03, 0.07} // the darkest it gets
)

// Sunlight is a touch warmer than the sky's overall colour
func sunWarm(c [3]float64) [3]float64 {
	return [3]float64{math.Min(1, c[0]*1.04), c[1], c[2] * 0.9}
}

func mixF(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

// The colour a light fades to at the edge of its glow: deeper and redder for warm
// lights, deeper for cool ones
func warmEdge(c [3]float64) [3]float64 {
	if c[0] > c[2] {
		return [3]float64{c[0], c[1] * 0.62, c[2] * 0.45}
	}
	return [3]float64{c[0] * 0.7, c[1] * 0.8, c[2]}
}

func blockLightSrc(t uint8) uint8 {
	switch t {
	case TORCH:
		return srcTorch
	case CRYSTAL:
		return srcCrystal
	case LAVA:
		return srcLava
	case CORE:
		return srcCore
	}
	return srcTorch
}

// Sunlight in the day, blue moonlight at night, orange at dawn and dusk
func skyTint() [3]float64 {
	d := daylight()
	night := [3]float64{0.55, 0.64, 1.0}
	day := [3]float64{1.0, 0.98, 0.93}
	c := [3]float64{}
	for k := range c {
		c[k] = night[k] + (day[k]-night[k])*d
	}
	sh := sunHeight()
	if dusk := math.Exp(-(sh / 0.3) * (sh / 0.3)); dusk > 0.02 {
		dc := [3]float64{1.0, 0.62, 0.42}
		for k := range c {
			c[k] += (dc[k] - c[k]) * dusk * 0.55
		}
	}
	return c
}

// Creatures that glow light up the caves around them
func entityLights(seed func(lx, ly, lvl int, src uint8)) {
	for _, e := range ents {
		cx, cy := int((e.x+e.w/2)/CELL), int((e.y+e.h/2)/CELL)
		switch e.kind {
		case entWisp:
			seed(cx, cy, 30, srcWisp)
		case entWarden:
			seed(cx, cy, 40, srcWarden)
		}
	}
}

var (
	lightImg, hazeImg, emitImg *ebiten.Image
	lightBuf, hazeBuf          []float32 // RGB per cell
	lightPix, hazePix, emitPix []byte
	tmpBuf                     []float32
)

var multiplyBlend = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

func ensureLightBuffers(vw, vh int) {
	if lightImg == nil || lightImg.Bounds().Dx() != vw || lightImg.Bounds().Dy() != vh {
		lightImg = ebiten.NewImage(vw, vh)
		hazeImg = ebiten.NewImage(vw, vh)
		emitImg = ebiten.NewImage(vw, vh)
		lightBuf = make([]float32, vw*vh*3)
		hazeBuf = make([]float32, vw*vh*3)
		tmpBuf = make([]float32, vw*vh*3)
		lightPix = make([]byte, vw*vh*4)
		hazePix = make([]byte, vw*vh*4)
		emitPix = make([]byte, vw*vh*4)
	}
	clear(emitPix)
}

// The colour of the light map for each kind of light at each level, worked out ahead
// instead of for every cell. Sun and sky change with the time of day (redone each
// frame); the rest never change.
var (
	lightRGB, hazeRGB [nSrc][lightFine + 1][3]float32
	lightRGBReady     bool
)

func lightColour(src uint8, lvl int, sky [3]float64) (l, h [3]float32) {
	f := lightCurve[lvl]
	tint := lightTints[src]
	switch {
	case src == srcSun:
		// direct sun is warm, cooling to blue as it fades
		tint = mixF(shadeCool, sunWarm(sky), math.Pow(f, 0.7))
	case src == srcSky:
		// shade under open sky is cooler, lit by the blue sky rather than the sun
		tint = mixF(shadeCool, sky, math.Pow(f, 0.7)*0.75)
	case src != srcNone:
		// firelight reddens toward the edge of its glow
		tint = mixF(warmEdge(tint), tint, math.Pow(f, 0.5))
	}
	for c := 0; c < 3; c++ {
		// never fully black: darkness is a deep violet-blue
		l[c] = float32(math.Max(f*tint[c], ambientDark[c]))
	}
	// haze: only from point lights, and stronger close to them
	if src != srcSky && src != srcSun && src != srcNone {
		hh := float32(math.Max(0, f-0.5) / 0.5)
		hh *= hh
		for c := 0; c < 3; c++ {
			h[c] = hh * float32(tint[c])
		}
	}
	return
}

func prepareLightColours() {
	sky := skyTint()
	for src := uint8(0); src < nSrc; src++ {
		if lightRGBReady && src != srcSun && src != srcSky {
			continue
		}
		for lvl := 0; lvl <= lightFine; lvl++ {
			lightRGB[src][lvl], hazeRGB[src][lvl] = lightColour(src, lvl, sky)
		}
	}
	lightRGBReady = true
}

// Fill the light map for the view. skyCell reports cells that show the open sky
// (they are never darkened: the sky already has its own day and night colours).
func buildLightMap(ox, oy, vw, vh int, skyCell func(x, y int) bool) {
	prepareLightColours()
	for y := 0; y < vh; y++ {
		for x := 0; x < vw; x++ {
			k := (y*vw + x) * 3
			if skyCell(x, y) {
				lightBuf[k], lightBuf[k+1], lightBuf[k+2] = 1, 1, 1
				hazeBuf[k], hazeBuf[k+1], hazeBuf[k+2] = 0, 0, 0
				continue
			}
			lx, ly := ox+x-lightX0, oy+y-lightY0
			lvl, src := 0, uint8(0)
			if lx >= 0 && ly >= 0 && lx < lightW && ly < lightH {
				lvl, src = int(light[ly*lightW+lx]), lightSrc[ly*lightW+lx]
			}
			lc, hc := &lightRGB[src][lvl], &hazeRGB[src][lvl]
			lightBuf[k], lightBuf[k+1], lightBuf[k+2] = lc[0], lc[1], lc[2]
			hazeBuf[k], hazeBuf[k+1], hazeBuf[k+2] = hc[0], hc[1], hc[2]
		}
	}
	blur3(lightBuf, vw, vh, 4-lightEvery())
	for i, j := 0, 0; i < vw*vh*3; i, j = i+3, j+4 {
		lightPix[j], lightPix[j+1], lightPix[j+2], lightPix[j+3] = toByte(lightBuf[i]), toByte(lightBuf[i+1]), toByte(lightBuf[i+2]), 255
	}
	lightImg.WritePixels(lightPix)
	if simpleFX() {
		return // no haze in the lighter mode
	}
	blur3(hazeBuf, vw, vh, 3) // a wide, soft bloom (drawn scaled up with smoothing)
	for i, j := 0, 0; i < vw*vh*3; i, j = i+3, j+4 {
		hazePix[j], hazePix[j+1], hazePix[j+2], hazePix[j+3] = toByte(hazeBuf[i]*0.24), toByte(hazeBuf[i+1]*0.24), toByte(hazeBuf[i+2]*0.24), 255
	}
	hazeImg.WritePixels(hazePix)
}

// A separable binomial blur on interleaved RGB: the same as running a [1 2 1] blur
// `passes` times (1-3), done as one wider pass across and one down
var binom = [4][]float32{
	nil,
	{1. / 4, 2. / 4, 1. / 4},
	{1. / 16, 4. / 16, 6. / 16, 4. / 16, 1. / 16},
	{1. / 64, 6. / 64, 15. / 64, 20. / 64, 15. / 64, 6. / 64, 1. / 64},
}

func blur3(b []float32, w, h, passes int) {
	k := binom[min(passes, 3)]
	r := len(k) / 2
	t := tmpBuf
	row := w * 3
	// across: b -> t
	for y := 0; y < h; y++ {
		src := b[y*row : (y+1)*row]
		dst := t[y*row : (y+1)*row]
		for x := 0; x < w; x++ {
			var s0, s1, s2 float32
			if x >= r && x < w-r {
				o := (x - r) * 3
				for i, kv := range k {
					s0 += src[o+i*3] * kv
					s1 += src[o+i*3+1] * kv
					s2 += src[o+i*3+2] * kv
				}
			} else {
				for i, kv := range k {
					xx := min(max(x+i-r, 0), w-1) * 3
					s0 += src[xx] * kv
					s1 += src[xx+1] * kv
					s2 += src[xx+2] * kv
				}
			}
			dst[x*3], dst[x*3+1], dst[x*3+2] = s0, s1, s2
		}
	}
	// down: t -> b, a whole row at a time
	for y := 0; y < h; y++ {
		dst := b[y*row : (y+1)*row]
		clear(dst)
		for i, kv := range k {
			yy := min(max(y+i-r, 0), h-1)
			src := t[yy*row : (yy+1)*row]
			for j := range dst {
				dst[j] += src[j] * kv
			}
		}
	}
}

func toByte(v float32) byte {
	if v >= 1 {
		return 255
	}
	if v <= 0 {
		return 0
	}
	return byte(v * 255)
}

// Put a glowing pixel on the emissive layer (a = 0..1)
func emit(x, y, vw, vh int, c [3]int, a float64) {
	if x < 0 || y < 0 || x >= vw || y >= vh {
		return
	}
	o := (y*vw + x) * 4
	pa := float64(emitPix[o+3]) / 255
	na := a + pa*(1-a)
	if na <= 0 {
		return
	}
	for k := 0; k < 3; k++ {
		emitPix[o+k] = clamp8(int((float64(c[k])*a + float64(emitPix[o+k])*pa*(1-a)) / na))
	}
	emitPix[o+3] = clamp8(int(na * 255))
}

// Light the world image that was just drawn to the screen, then add glow and haze
func composeLight(screen *ebiten.Image, geo ebiten.GeoM) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: multiplyBlend}
	op.GeoM = geo
	screen.DrawImage(lightImg, op)
	// emissive things on top, unaffected by darkness (premultiplied alpha)
	for i := 0; i < len(emitPix); i += 4 {
		a := uint16(emitPix[i+3])
		emitPix[i] = byte(uint16(emitPix[i]) * a / 255)
		emitPix[i+1] = byte(uint16(emitPix[i+1]) * a / 255)
		emitPix[i+2] = byte(uint16(emitPix[i+2]) * a / 255)
	}
	emitImg.WritePixels(emitPix)
	op = &ebiten.DrawImageOptions{}
	op.GeoM = geo
	screen.DrawImage(emitImg, op)
	if simpleFX() {
		return
	}
	// coloured haze around lights
	op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	op.GeoM = geo
	screen.DrawImage(hazeImg, op)
}

// A soft dark edge around the screen
var vignetteImg *ebiten.Image

func drawVignette(screen *ebiten.Image) {
	if simpleFX() {
		return
	}
	const n = 64
	if vignetteImg == nil {
		vignetteImg = ebiten.NewImage(n, n)
		pix := make([]byte, n*n*4)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
				d := math.Max(0, math.Hypot(dx*0.9, dy)-0.55) / 0.75
				a := math.Min(1, d*d) * 0.55
				pix[(y*n+x)*4+3] = byte(a * 255)
			}
		}
		vignetteImg.WritePixels(pix)
	}
	b := screen.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(b.Dx())/n, float64(b.Dy())/n)
	op.ColorScale.ScaleWithColor(color.NRGBA{8, 6, 18, 255})
	screen.DrawImage(vignetteImg, op)
}

// ---- Performance ----
// On slow devices the game switches to a lighter mode by itself: a lighter blur,
// clouds update less often, and the haze and vignette are left out. The Graphics setting can force either way.

var lowSpec bool // the automatic light mode is on

func simpleFX() bool { return prefs.Simple || (lowSpec && !prefs.Fancy) }

// 2 in the lighter mode (it slows cloud updates and lightens the blur). The light
// itself is worked out every frame: skipping frames made it lag and stutter.
func lightEvery() int {
	if lowSpec && !prefs.Fancy || prefs.Simple {
		return 2
	}
	return 1
}

var slowChecks int

// Called once a frame: after a few seconds of play, a frame rate well under 60
// turns the lighter mode on for the rest of the session
func checkPerformance() {
	if lowSpec || prefs.Fancy || cur != scPlaying || frame < 120 || frame%30 != 0 {
		return
	}
	if ebiten.ActualFPS() < 42 {
		slowChecks++
	} else {
		slowChecks = 0
	}
	if slowChecks >= 2 {
		lowSpec = true
	}
}
