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
	if dusk := math.Exp(-(sh / 0.22) * (sh / 0.22)); dusk > 0.02 {
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

// Fill the light map for the view. skyCell reports cells that show the open sky
// (they are never darkened: the sky already has its own day and night colours).
func buildLightMap(ox, oy, vw, vh int, skyCell func(x, y int) bool) {
	sky := skyTint()
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
			f := lightCurve[lvl]
			tint := lightTints[src]
			if src == srcSky {
				tint = sky
			}
			for c := 0; c < 3; c++ {
				lightBuf[k+c] = float32(f * tint[c])
			}
			// haze: only from point lights, and stronger close to them
			if src != srcSky && src != srcNone {
				h := float32(math.Max(0, f-0.55) / 0.45)
				h *= h
				for c := 0; c < 3; c++ {
					hazeBuf[k+c] = h * float32(tint[c])
				}
			} else {
				hazeBuf[k], hazeBuf[k+1], hazeBuf[k+2] = 0, 0, 0
			}
		}
	}
	blur3(lightBuf, vw, vh, 2)
	blur3(hazeBuf, vw, vh, 2)
	for i, j := 0, 0; i < vw*vh*3; i, j = i+3, j+4 {
		lightPix[j], lightPix[j+1], lightPix[j+2], lightPix[j+3] = toByte(lightBuf[i]), toByte(lightBuf[i+1]), toByte(lightBuf[i+2]), 255
		hazePix[j], hazePix[j+1], hazePix[j+2], hazePix[j+3] = toByte(hazeBuf[i]*0.16), toByte(hazeBuf[i+1]*0.16), toByte(hazeBuf[i+2]*0.16), 255
	}
	lightImg.WritePixels(lightPix)
	hazeImg.WritePixels(hazePix)
}

// A small separable [1 2 1] blur on interleaved RGB, repeated
func blur3(b []float32, w, h, passes int) {
	t := tmpBuf
	row := w * 3
	for p := 0; p < passes; p++ {
		for y := 0; y < h; y++ {
			r := b[y*row : y*row+row]
			o := t[y*row : y*row+row]
			o[0], o[1], o[2] = (3*r[0]+r[3])*0.25, (3*r[1]+r[4])*0.25, (3*r[2]+r[5])*0.25
			for i := 3; i < row-3; i++ {
				o[i] = (r[i-3] + 2*r[i] + r[i+3]) * 0.25
			}
			n := row - 3
			o[n], o[n+1], o[n+2] = (r[n-3]+3*r[n])*0.25, (r[n-2]+3*r[n+1])*0.25, (r[n-1]+3*r[n+2])*0.25
		}
		copy(b[:row], t[:row])
		copy(b[(h-1)*row:], t[(h-1)*row:])
		for y := 1; y < h-1; y++ {
			u, c, d := t[(y-1)*row:y*row], t[y*row:(y+1)*row], t[(y+1)*row:(y+2)*row]
			o := b[y*row : (y+1)*row]
			for i := range o {
				o[i] = (u[i] + 2*c[i] + d[i]) * 0.25
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
	if prefs.Simple {
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
	if prefs.Simple {
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
