package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Rendering: lighting (fog of war), the day/night sky, the parallax background,
// cave back walls, blocks, the mote, creatures and particles, all in world pixels.

// ---- Day and night ----

const dayFrames = 60 * 60 * 8 // one full day lasts 8 minutes

var worldTime = 0.32 // 0 = midnight, 0.25 = sunrise, 0.5 = noon, 0.75 = sunset

func sunHeight() float64 { return math.Sin(2 * math.Pi * (worldTime - 0.25)) }

func smoothstep(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// 0 at night, 1 in full day
// (it builds up slowly through the morning and fades through the afternoon, so the
// low sun is never as bright as midday)
func daylight() float64 { return smoothstep(-0.15, 0.6, sunHeight()) }

// Sky brightness, in fine light steps (0-1023): smooth through dawn and dusk
func skyFine() float64 { return lightFine * (0.47 + 0.53*daylight()) }

// Direct and ambient daylight (fine steps) and the slant of the light: how many cells
// sideways a ray moves for each cell it falls. The sun rises on the left and sets on
// the right; at night the moon (dimmer) does the same.
func sunLight() (direct, ambient int, slope float64) {
	th := 2 * math.Pi * (worldTime - 0.25)
	bx, by := -math.Cos(th), math.Sin(th) // the sun's position: x -1 left .. 1 right, y up
	if by < 0 {
		bx, by = -bx, -by // the moon is opposite the sun
	}
	slope = math.Max(-2.5, math.Min(2.5, -bx/math.Max(by, 0.2)))
	sky := skyFine()
	// the low sun at dawn and dusk is weaker, and its light goes further sideways
	strength := smoothstep(0, 0.6, by)
	direct = int(sky * (0.62 + 0.38*strength))
	ambient = int(sky * 0.55)
	return
}

func isNight() bool { return sunHeight() < 0 }

// ---- Lighting ----
// Light flows out from the sky, the mote, torches and crystals. It travels easily
// through air and water and only a little way into solid blocks. Unlit = black.
//
// Sources and lightAt() use levels 0-64. Inside, the flood fill works at 16x finer steps
// (0-1023) with diagonal costs close to the true distance, so light spreads in smooth,
// round pools instead of diamonds and steps. A full-strength light carries about 85
// cells through open air and fades gently all the way, like real light does, and about
// 30 cells into solid ground, so the ground darkens gradually below the surface.

const (
	lightMax      = 64   // the scale used by sources and lightAt
	lightFine     = 1023 // the internal scale
	costAir       = 12   // per cell, internal steps (diagonals cost 17, about 12 x 1.41)
	costDiag      = 17
	costSolid     = 34 // light fades about three times as fast inside solid blocks
	costSolidDiag = 48
	lightReach    = lightFine/costAir + 1
)

var (
	light            []uint16 // internal levels 0-1023
	lightSrc         []uint8  // which kind of light reached each cell (see lightTints)
	lightCost        []uint8  // 1 for cells where light travels through something solid
	sunLit           []bool   // cells the sun (or moon) shines on directly
	sunSlope         float64  // cells sideways per cell down that sunlight travels
	lightBase        int      // baseChunk when the light was last worked out
	colTop           []int    // per column of the light window: first block that stops light
	lightW, lightH   int
	lightX0, lightY0 int
	buckets          [lightFine + 1][]int32
)

// Does the sun shine straight onto loaded cell (x, y)?
func sunLitAt(x, y int) bool {
	lx, ly := x-lightX0, y-lightY0
	if lx < 0 || ly < 0 || lx >= lightW || ly >= lightH {
		return false
	}
	return sunLit[ly*lightW+lx]
}

func lightAt(x, y int) int {
	lx, ly := x-lightX0, y-lightY0
	if lx < 0 || ly < 0 || lx >= lightW || ly >= lightH {
		return 0
	}
	return int(light[ly*lightW+lx]) * lightMax / lightFine
}

func playerLightLevel() int {
	if countItem("lantern") > 0 {
		return 60
	}
	return 42
}

func computeLight(ox, oy, vw, vh int) {
	const m = 64 // light from beyond the screen edge can still reach in
	lightX0, lightY0, lightW, lightH = ox-m, oy-m, vw+2*m, vh+2*m
	n := lightW * lightH
	if cap(light) < n {
		light = make([]uint16, n)
		lightSrc = make([]uint8, n)
	}
	light, lightSrc = light[:n], lightSrc[:n]
	if cap(lightCost) < n {
		lightCost = make([]uint8, n)
	}
	lightCost = lightCost[:n]
	clear(light)
	clear(lightSrc)
	for i := range buckets {
		buckets[i] = buckets[i][:0]
	}
	seedFine := func(idx, f int, src uint8) {
		f = min(lightFine, f)
		if f > int(light[idx]) {
			light[idx] = uint16(f)
			lightSrc[idx] = src
			buckets[f] = append(buckets[f], int32(idx))
		}
	}
	seed := func(idx, lvl int, src uint8) { seedFine(idx, lvl*lightFine/lightMax, src) }
	// 1 where light is slowed by a solid block
	for r := 0; r < lightH; r++ {
		wy := lightY0 + r
		row := lightCost[r*lightW : (r+1)*lightW]
		for c := range row {
			wx := lightX0 + c
			if wx < 0 || wx >= W || wy >= H || (wy >= 0 && blocksLight(grid[wy*W+wx])) {
				row[c] = 1
			} else {
				row[c] = 0
			}
		}
	}
	// Daylight comes from where the sun (or at night the moon) is. Direct light follows
	// slanted rays and stops at the first solid block, so hills and cliffs cast shadows
	// that swing round through the day. Open sky everywhere also gives softer, bluer
	// light straight down, so shade is dimmer but never black.
	direct, ambient, slope := sunLight()
	sunSlope = slope
	if cap(sunLit) < n {
		sunLit = make([]bool, n)
	}
	sunLit = sunLit[:n]
	clear(sunLit)
	if cap(colTop) < lightW {
		colTop = make([]int, lightW)
	}
	colTop = colTop[:lightW]
	for c := 0; c < lightW; c++ {
		wx := lightX0 + c
		top, lim := 0, min(H, lightY0+lightH)
		if wx >= 0 && wx < W {
			for top < lim && !blocksLight(grid[top*W+wx]) {
				top++
			}
		}
		colTop[c] = top
	}
	for r := 0; r < lightH; r++ {
		wy := lightY0 + r
		for c := 0; c < lightW; c++ {
			i := r*lightW + c
			if lightCost[i] != 0 {
				continue
			}
			var lit bool
			// where this cell's ray to the sun crosses the row above
			pc := c - int(math.Round(slope*float64(r+1))) + int(math.Round(slope*float64(r)))
			switch {
			case wy < 0:
				lit = true
			case r == 0 || pc < 0 || pc >= lightW:
				lit = wy < colTop[c] // outside the window: straight up
			default:
				lit = sunLit[i-lightW+pc-c]
				// a slanting ray also crosses the cells between, in this row
				for k := min(pc, c) + 1; lit && k < max(pc, c); k++ {
					lit = lightCost[r*lightW+k] == 0
				}
			}
			sunLit[i] = lit
			if lit {
				seedFine(i, direct, srcSun)
			} else if wy < colTop[c] {
				seedFine(i, ambient, srcSky)
			}
		}
	}
	for c := 0; c < lightW; c++ {
		wx := lightX0 + c
		if wx < 0 || wx >= W {
			continue
		}
		for r := 0; r < lightH; r++ {
			wy := lightY0 + r
			if wy < 0 {
				continue
			}
			if wy >= H {
				break
			}
			if t := grid[wy*W+wx]; t != EMPTY {
				if l := blockLight(t); l > 0 {
					seed(r*lightW+c, l, blockLightSrc(t))
				}
			}
		}
	}
	objLights(func(lx, ly, lvl int) {
		c, r := lx-lightX0, ly-lightY0
		if c >= 0 && r >= 0 && c < lightW && r < lightH {
			seed(r*lightW+c, lvl, srcFurnace)
		}
	})
	entityLights(func(lx, ly, lvl int, src uint8) {
		c, r := lx-lightX0, ly-lightY0
		if c >= 0 && r >= 0 && c < lightW && r < lightH {
			seed(r*lightW+c, lvl, src)
		}
	})
	pcx, pcy := int((player.x+player.w/2)/CELL)-lightX0, int((player.y+player.h/2)/CELL)-lightY0
	if pcx >= 0 && pcy >= 0 && pcx < lightW && pcy < lightH {
		src := uint8(srcMote)
		if countItem("lantern") > 0 {
			src = srcLantern
		}
		seed(pcy*lightW+pcx, playerLightLevel(), src)
	}
	type step struct{ dc, dr, air, solid int }
	steps := [8]step{{-1, 0, costAir, costSolid}, {1, 0, costAir, costSolid}, {0, -1, costAir, costSolid}, {0, 1, costAir, costSolid},
		{-1, -1, costDiag, costSolidDiag}, {1, -1, costDiag, costSolidDiag}, {-1, 1, costDiag, costSolidDiag}, {1, 1, costDiag, costSolidDiag}}
	var offs [8]int
	for k, d := range steps {
		offs[k] = d.dr*lightW + d.dc
	}
	// (a cell with no more light than one step costs can't light anything)
	for lvl := lightFine; lvl > costAir; lvl-- {
		for k := 0; k < len(buckets[lvl]); k++ {
			idx := int(buckets[lvl][k])
			if int(light[idx]) != lvl {
				continue
			}
			c, r := idx%lightW, idx/lightW
			src := lightSrc[idx]
			if c > 0 && r > 0 && c < lightW-1 && r < lightH-1 {
				// away from the edges: no bounds checks needed
				for j, d := range steps {
					ni := idx + offs[j]
					nl := lvl - d.air
					if lightCost[ni] != 0 {
						nl = lvl - d.solid
					}
					if nl > int(light[ni]) {
						light[ni] = uint16(nl)
						lightSrc[ni] = src
						buckets[nl] = append(buckets[nl], int32(ni))
					}
				}
				continue
			}
			for _, d := range steps {
				nc, nr := c+d.dc, r+d.dr
				if nc < 0 || nr < 0 || nc >= lightW || nr >= lightH {
					continue
				}
				ni := nr*lightW + nc
				cost := d.air
				if lightCost[ni] != 0 {
					cost = d.solid
				}
				nl := lvl - cost
				if nl > int(light[ni]) {
					light[ni] = uint16(nl)
					lightSrc[ni] = src
					buckets[nl] = append(buckets[nl], int32(ni))
				}
			}
		}
	}
}

// Brightness for an internal light level: a long, even fade that eases in and out at
// both ends (no visible edge where the light runs out, no flat plateau near a source)
var lightCurve = func() [lightFine + 1]float64 {
	var t [lightFine + 1]float64
	for i := range t {
		x := math.Min(1, float64(i)/900)
		t[i] = math.Pow(x*x*(3-2*x), 0.62)
	}
	return t
}()

// ---- Sky and background ----

func mix(a, b [3]int, t float64) [3]int {
	return [3]int{a[0] + int(float64(b[0]-a[0])*t), a[1] + int(float64(b[1]-a[1])*t), a[2] + int(float64(b[2]-a[2])*t)}
}

func shadeRGB(c [3]int, f float64) [3]int {
	return [3]int{int(float64(c[0]) * f), int(float64(c[1]) * f), int(float64(c[2]) * f)}
}

var (
	skyTopDay   = [3]int{92, 136, 196}
	skyLowDay   = [3]int{178, 206, 228}
	skyTopNight = [3]int{10, 12, 32}
	skyLowNight = [3]int{34, 36, 72}
	duskGlow    = [3]int{244, 150, 112}
	farHill     = [3]int{128, 160, 186}
	nearHill    = [3]int{84, 120, 140}
	caveWall    = [3]int{112, 100, 132}
)

// Background pixel for an open-sky cell, in view cell coordinates
// Values the sky needs that only change once a frame
var skyF struct {
	d, sh, dusk                float64
	top, low, cloudHi, cloudLo [3]int
	sunX, sunY, moonX, moonY   float64
}

func prepareSky(vw, vh int) {
	d, sh := daylight(), sunHeight()
	skyF.d, skyF.sh = d, sh
	skyF.dusk = math.Exp(-(sh / 0.2) * (sh / 0.2))
	skyF.top, skyF.low = mix(skyTopNight, skyTopDay, d), mix(skyLowNight, skyLowDay, d)
	skyF.cloudHi = mix([3]int{70, 74, 110}, [3]int{252, 252, 255}, d)
	skyF.cloudLo = mix([3]int{30, 32, 60}, [3]int{150, 172, 208}, d)
	skyF.sunX = float64(vw) * (0.5 + 0.42*math.Cos(2*math.Pi*(worldTime-0.25)+math.Pi))
	skyF.sunY = float64(vh)*0.62 - sh*float64(vh)*0.48
	skyF.moonX = float64(vw) - skyF.sunX
	skyF.moonY = float64(vh)*0.62 + sh*float64(vh)*0.48
}

func skyPixel(vx, vy, vw, vh int, ox, oy int) [3]int {
	d, low := skyF.d, skyF.low
	t := math.Max(0, math.Min(1, float64(vy)/float64(vh)))
	c := mix(skyF.top, low, t)
	if skyF.dusk > 0.02 {
		c = mix(c, duskGlow, skyF.dusk*t*0.7)
	}
	// stars
	if d < 0.6 {
		sx, sy := vx+int(float64(ox)*0.05), vy+int(float64(oy)*0.05)
		if hash32(sx, sy, 77)%997 < 3 {
			tw := 0.6 + 0.4*math.Sin(float64(frame)*0.05+float64(sx))
			c = mix(c, [3]int{240, 240, 255}, (1-d/0.6)*tw)
		}
	}
	// sun and moon
	if dx, dy := float64(vx)-skyF.sunX, float64(vy)-skyF.sunY; dx*dx+dy*dy < 36 {
		c = [3]int{255, 236, 160}
	} else if dx*dx+dy*dy < 64 {
		c = mix(c, [3]int{255, 220, 140}, 0.4)
	}
	if dx, dy := float64(vx)-skyF.moonX, float64(vy)-skyF.moonY; dx*dx+dy*dy < 20 {
		c = [3]int{214, 218, 236}
		if (vx+vy)%5 == 0 {
			c = [3]int{180, 184, 206}
		}
	}
	// big soft clouds, lit from above
	if ca, cl := cloudAt(vx, vy, vw, vh); ca > 0 {
		cloud := mix(skyF.cloudLo, skyF.cloudHi, cl)
		if skyF.dusk > 0.02 {
			cloud = mix(cloud, [3]int{255, 170, 130}, skyF.dusk*cl*0.5)
		}
		c = mix(c, cloud, ca)
	}
	// two layers of distant hills, moving slower than the world
	for li, layer := range hillLayers {
		hTop := hillTop[li][vx]
		if float64(vy) >= hTop {
			// far things fade into the sky, and mist gathers lower down
			mist := 0.3 + 0.35*smoothstep(0, 40, float64(vy)-hTop)
			hc := mix(layer.col, low, mist)
			if float64(vy)-hTop < 1.5 { // a lit rim along the top
				hc = mix(hc, low, 0.35)
			}
			c = shadeRGB(hc, 0.3+0.7*d)
			break
		}
	}
	return c
}

// Distant hills: two layers moving slower than the world. Their outline only depends
// on the column, so it's worked out once per column each frame.
var hillLayers = []struct {
	f    float64
	amp  float64
	col  [3]int
	salt int
}{{0.2, 34, farHill, 51}, {0.45, 24, nearHill, 52}}

var hillTop [2][]float64

func updateHills(vw, vh, ox, oy int, avgSurface float64) {
	for li, layer := range hillLayers {
		if len(hillTop[li]) != vw {
			hillTop[li] = make([]float64, vw)
		}
		horizon := float64(vh)*0.58 + (avgSurface-float64(oy)-float64(vh)*0.58)*layer.f
		for vx := 0; vx < vw; vx++ {
			hx := float64(vx) + float64(ox)*layer.f
			// rolling hills topped with bumpy treetops
			hillTop[li][vx] = horizon - (noise1(hx, 40, layer.salt)*0.5+0.5)*layer.amp - noise1(hx, 9, layer.salt+5)*4 -
				math.Abs(noise1(hx, 3.5, layer.salt+9))*3
		}
	}
}

// Clouds: fractal noise drifting across the upper sky, shaded so their tops catch the
// light and their undersides are blue-grey. Worked out every few frames (they move
// slowly) into a buffer: coverage and light, 0..1.
var (
	cloudBuf          []float32
	cloudN            []float64 // the raw cloud field
	cloudW, cloudH    int
	cloudAt0, cloudOX int
)

func cloudField(x, y float64) float64 {
	return noise2(x, y, 52, 41)*0.58 + noise2(x, y, 21, 42)*0.28 + noise2(x, y, 8, 43)*0.14
}

func updateClouds(vw, vh, ox int) {
	if cloudW != vw || cloudH != vh {
		cloudBuf = make([]float32, vw*vh*2)
		cloudW, cloudH = vw, vh
		cloudAt0 = -100
	}
	if frame-cloudAt0 < 3*lightEvery()*lightEvery() && abs(ox-cloudOX) < 2 {
		return
	}
	cloudAt0, cloudOX = frame, ox
	drift := float64(ox)*0.12 + float64(frame)*0.015
	if len(cloudN) != vw*vh {
		cloudN = make([]float64, vw*vh)
	}
	// the cloud field once per cell...
	for y := 0; y < vh; y++ {
		band := 1 - smoothstep(0.3, 0.62, float64(y)/float64(vh))
		for x := 0; x < vw; x++ {
			if band <= 0 {
				cloudN[y*vw+x] = 0
				continue
			}
			cloudN[y*vw+x] = cloudField(float64(x)+drift, float64(y)*1.15)*band + 0.08*band
		}
	}
	// ...then coverage and shading from it
	for y := 0; y < vh; y++ {
		t := float64(y) / float64(vh)
		band := 1 - smoothstep(0.3, 0.62, t) // clouds stay in the upper sky
		for x := 0; x < vw; x++ {
			k := (y*vw + x) * 2
			cloudBuf[k], cloudBuf[k+1] = 0, 0
			if band <= 0 {
				continue
			}
			n := cloudN[y*vw+x]
			if n < 0.56 {
				continue
			}
			// the field up and to the left, toward the light (looked up, not recomputed)
			up := 0.0
			if ux, uy := x-2, y-3; ux >= 0 && uy >= 0 {
				up = cloudN[uy*vw+ux]
			}
			a := math.Min(1, (n-0.56)*5)
			cloudBuf[k] = float32(a * a * (3 - 2*a) * 0.95)
			cloudBuf[k+1] = float32(math.Max(0, math.Min(1, 0.6+(n-up)*6+(n-0.56)*1.5)))
		}
	}
}

func cloudAt(x, y, vw, vh int) (float64, float64) {
	if x < 0 || y < 0 || x >= cloudW || y >= cloudH {
		return 0, 0
	}
	k := (y*cloudW + x) * 2
	return float64(cloudBuf[k]), float64(cloudBuf[k+1])
}

// ---- Compose the frame ----

func clamp8(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

var skyCells []bool

func drawWorld(screen *ebiten.Image) {
	vw, vh := int(logicalW/zoom/CELL)+2, int(logicalH/zoom/CELL)+2
	viewW, viewH = vw, vh
	if worldImg == nil || worldImg.Bounds().Dx() != vw || worldImg.Bounds().Dy() != vh {
		worldImg = ebiten.NewImage(vw, vh)
		worldPix = make([]byte, vw*vh*4)
	}
	ensureLightBuffers(vw, vh)
	ox, oy := int(math.Floor(cam.x/CELL)), int(math.Floor(cam.y/CELL))
	// the flood fill works in world cells, so on frames it's skipped the last one is
	// still right (unless the loaded chunks just slid along)
	if frame%lightEvery() == 0 || baseChunk != lightBase || lightW == 0 {
		computeLight(ox, oy, vw, vh)
		lightBase = baseChunk
	}
	checkPerformance()
	pix := worldPix
	surf := make([]int, vw)
	avg := 0.0
	for x := 0; x < vw; x++ {
		surf[x] = surfaceAt(baseChunk*CW + ox + x)
		avg += float64(surf[x])
	}
	avg /= float64(vw)
	updateClouds(vw, vh, ox+baseChunk*CW)
	updateHills(vw, vh, ox+baseChunk*CW, oy, avg)
	prepareSky(vw, vh)
	set := func(o int, c [3]int) {
		pix[o], pix[o+1], pix[o+2], pix[o+3] = clamp8(c[0]), clamp8(c[1]), clamp8(c[2]), 255
	}
	if len(skyCells) != vw*vh {
		skyCells = make([]bool, vw*vh)
	}
	sky := skyCells // cells showing the open sky (never darkened)
	clear(sky)
	// The world is drawn at full brightness here; the light map darkens and colours it later.
	for y := 0; y < vh; y++ {
		wy := oy + y
		for x := 0; x < vw; x++ {
			wx := ox + x
			o := (y*vw + x) * 4
			var t uint8 = STONE
			var wl, pt, wpt uint8
			sh := 0
			if wx >= 0 && wx < W && wy >= 0 && wy < H {
				i := wy*W + wx
				t, sh = grid[i], int(shade[i])
				wl, pt, wpt = wall[i], paint[i], wpaint[i]
			} else if wy < 0 {
				t = EMPTY
			}
			openSky := wy < surf[x]
			switch {
			case t == EMPTY || t == WATER || t == TORCH || t == GLASS || t == LAVA || t == DOOROPEN:
				var bg [3]int
				switch {
				case wl != 0:
					bg = wallPixel(wl, wpt, wx, wy, sh)
				case openSky:
					bg = skyPixel(x, y, vw, vh, ox+baseChunk*CW, oy)
					sky[y*vw+x] = t == EMPTY
				default:
					bg = [3]int{caveWall[0] + sh, caveWall[1] + sh, caveWall[2] + sh}
					// the back wall is shadowed where it meets rock
					bg = shadeRGB(bg, caveOcclusion(wx, wy))
				}
				// grass blades poke up from the grass
				if t == EMPTY && wl == 0 {
					if g, ok := grassBlade(wx, wy); ok {
						bg = g
						sky[y*vw+x] = false
					}
				}
				if t == WATER {
					wc := blockRGB[WATER]
					wc = [3]int{wc[0] + sh/2, wc[1] + sh/2, wc[2] + sh/2}
					if openSky && wl == 0 { // under the sky: tint the sky's own colour
						wc = shadeRGB(wc, 0.4+0.6*daylight())
						sky[y*vw+x] = true
					}
					// a bright line on the water's surface
					if wy > 0 && wy < H && wx >= 0 && wx < W && grid[(wy-1)*W+wx] == EMPTY {
						wc = mix(wc, [3]int{200, 230, 255}, 0.35)
					}
					bg = mix(bg, wc, 0.78)
				}
				if t == DOOROPEN {
					bg = doorPixel(t, wx, wy, bg, 1)
				}
				if t == GLASS {
					g := blockRGB[GLASS]
					if (wx+wy)%7 == 0 {
						g = [3]int{240, 250, 255}
					}
					bg = mix(bg, g, 0.4)
				}
				set(o, bg)
				if t == TORCH {
					emit(x, y, vw, vh, torchPixel(wx, wy, bg), 1)
				}
				if t == LAVA { // glowing, slowly churning
					f := 0.85 + 0.15*math.Sin(float64(frame)*0.04+float64(wx)*0.7+float64(wy)*0.3)
					lc := blockRGB[LAVA]
					emit(x, y, vw, vh, shadeRGB([3]int{lc[0], lc[1] + sh*2, lc[2] + sh}, f), 1)
				}
			default:
				c := blockRGB[t]
				if isOre(t) { // ores sparkle a little
					c = shadeRGB(c, 0.9+0.2*float64((sh+10)%3)/2)
				}
				if pt != 0 {
					c = mix(c, paintRGB[pt], 0.7)
				}
				c = [3]int{c[0] + sh, c[1] + sh, c[2] + sh}
				c = blockTexture(c, t, wx, wy)
				if t == DOOR {
					set(o, doorPixel(t, wx, wy, c, 1))
					continue
				}
				if len(mineDmg) > 0 && wx >= 0 && wx < W && wy >= 0 && wy < H {
					if k := crackAt(wy*W + wx); k > 0 && crackPattern(wx+baseChunk*CW, wy) < k {
						c = shadeRGB(c, 0.45) // cracks spread as the block wears down
					}
				}
				// bevels: a lit top edge, shaded undersides and sides
				if isSolid(t) {
					up, up2 := gridAt(wx, wy-1), gridAt(wx, wy-2)
					switch {
					case !isSolid(up):
						if sunLitAt(wx, wy-1) { // the sun catches the top edge
							c = mix(c, [3]int{255, 244, 206}, 0.12+0.22*skyF.d)
						} else if t != GRASS {
							c = mix(c, [3]int{255, 255, 255}, 0.1)
						}
					case !isSolid(up2):
						c = mix(c, [3]int{255, 255, 255}, 0.05)
					}
					// a low sun also lights the faces turned toward it
					if s := sunSlope; math.Abs(s) > 0.5 {
						sx := wx - int(math.Copysign(1, s))
						if !isSolid(gridAt(sx, wy)) && sunLitAt(sx, wy) {
							c = mix(c, [3]int{255, 222, 170}, math.Min(0.3, (math.Abs(s)-0.5)*0.2)*skyF.d)
						}
					}
					if !isSolid(gridAt(wx, wy+1)) {
						c = shadeRGB(c, 0.68)
					} else if !isSolid(gridAt(wx, wy+2)) {
						c = shadeRGB(c, 0.86)
					}
					if !isSolid(gridAt(wx-1, wy)) || !isSolid(gridAt(wx+1, wy)) {
						c = shadeRGB(c, 0.84)
					}
				}
				set(o, c)
				switch t {
				case CORE:
					emit(x, y, vw, vh, mix(c, [3]int{255, 240, 255}, 0.5+0.5*math.Sin(float64(frame)*0.08)), 1)
				case CRYSTAL:
					emit(x, y, vw, vh, mix(c, [3]int{220, 250, 255}, 0.2+0.2*math.Sin(float64(frame)*0.05+float64(wx))), 0.75)
				}
			}
		}
	}
	// emissive drawing: always visible (the mote, eyes, particles, holograms...)
	put := func(x, y int, c [3]int, s int) {
		emit(x, y, vw, vh, [3]int{c[0] + s, c[1] + s, c[2] + s}, 1)
	}
	// lit drawing: part of the world, shaded by the light map
	lit := func(x, y int, c [3]int, s int) {
		if x < 0 || y < 0 || x >= vw || y >= vh {
			return
		}
		o := (y*vw + x) * 4
		pix[o], pix[o+1], pix[o+2] = clamp8(c[0]+s), clamp8(c[1]+s), clamp8(c[2]+s)
		sky[y*vw+x] = false
	}
	drawObjects(lit, ox, oy)
	drawTraders(lit, put, ox, oy)
	drawEntities(lit, put, ox, oy)
	blend := func(x, y int, c [3]int, a float64) { emit(x, y, vw, vh, c, a) }
	drawHologram(blend, ox, oy)
	drawStationHologram(blend, ox, oy)
	drawDoorHologram(blend, ox, oy)
	drawBuildHologram(blend, ox, oy)
	drawPlayer(put, ox, oy)
	drawHeld(put, ox, oy)
	for _, q := range parts {
		put(int(math.Floor(q.x))-ox, int(math.Floor(q.y))-oy, q.c, 0)
	}
	drawEnergy(put, ox, oy)
	if cur == scPlaying {
		drawAim(put, ox, oy)
	}
	buildLightMap(ox, oy, vw, vh, func(x, y int) bool { return sky[y*vw+x] })
	worldImg.WritePixels(pix)
	k := zoom * CELL * uiScale
	var geo ebiten.GeoM
	geo.Scale(k, k)
	geo.Translate((float64(ox*CELL)-cam.x)*zoom*uiScale, (float64(oy*CELL)-cam.y)*zoom*uiScale)
	op := &ebiten.DrawImageOptions{}
	op.GeoM = geo
	screen.DrawImage(worldImg, op)
	composeLight(screen, geo)
	drawVignette(screen)
}

// A background wall: the block's colour, darker, with a faint pattern so it reads as "behind"
// The block at a loaded cell (empty above the world, stone beyond its edges)
func gridAt(wx, wy int) uint8 {
	if wy < 0 {
		return EMPTY
	}
	if wx < 0 || wx >= W || wy >= H {
		return STONE
	}
	return grid[wy*W+wx]
}

// Darker close to solid rock, for open cave cells
func caveOcclusion(wx, wy int) float64 {
	n := 0
	for _, d := range [8][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-2, 0}, {2, 0}, {0, -2}, {0, 2}} {
		if isSolid(gridAt(wx+d[0], wy+d[1])) {
			n++
		}
	}
	return 1 - 0.06*float64(n)
}

// Grass blades in the empty cells just above grass: 1-3 cells tall, swaying in the wind
func grassBlade(wx, wy int) ([3]int, bool) {
	gx := wx + baseChunk*CW
	for k := 1; k <= 3; k++ {
		if gridAt(wx, wy+k) != GRASS {
			continue
		}
		hgt := int(hash32(gx, 0, 131) % 4) // 0 = no blade here
		if k > hgt {
			return [3]int{}, false
		}
		if k < hgt { // a blade's middle can lean with the wind
			lean := math.Sin(float64(frame)*0.03+float64(gx)*0.4) > 0.6
			if lean && hash32(gx, 1, 131)%2 == 0 {
				return [3]int{}, false
			}
		}
		g := blockRGB[GRASS]
		return mix(g, [3]int{200, 236, 120}, 0.15*float64(k)), true
	}
	return [3]int{}, false
}

// Per-material detail on top of the clumpy shading
// Darker pockets inside leaf clumps: a cheap blobby pattern from hashes of 3x3 blocks
func leafPocket(gx, wy int) bool {
	bx, by := floorDiv(gx+int(hash32(wy/3, 0, 137)%3), 3), floorDiv(wy, 3)
	return hash32(bx, by, 136)%4 == 0
}

func blockTexture(c [3]int, t uint8, wx, wy int) [3]int {
	gx := wx + baseChunk*CW
	h := hash32(gx, wy, 133)
	switch t {
	case DIRT:
		if h%29 == 0 {
			c = shadeRGB(c, 0.72) // pebbles
		} else if h%37 == 0 {
			c = mix(c, [3]int{196, 160, 120}, 0.35)
		}
		// grass roots reach down into the dirt below
		if gridAt(wx, wy-1) == GRASS && h%3 != 0 {
			c = mix(c, blockRGB[GRASS], 0.45)
		} else if gridAt(wx, wy-2) == GRASS && h%3 == 0 {
			c = mix(c, blockRGB[GRASS], 0.3)
		}
	case STONE:
		if h%41 == 0 {
			c = shadeRGB(c, 0.75)
		}
	case GRASS:
		c = mix(c, [3]int{170, 226, 96}, 0.25) // bright, sunny tops
	case WOOD:
		// bark: vertical streaks, darker at the trunk's edges
		if (gx+int(hash32(gx, wy/3, 135)%2))%3 == 0 {
			c = shadeRGB(c, 0.8)
		}
		if gridAt(wx-1, wy) != WOOD || gridAt(wx+1, wy) != WOOD {
			c = shadeRGB(c, 0.78)
		}
	case LEAVES:
		// lit from the upper left, shadowed below, with darker pockets inside
		ul, dn := gridAt(wx-1, wy-1), gridAt(wx, wy+1)
		switch {
		case ul != LEAVES && gridAt(wx, wy-1) != LEAVES:
			c = mix(c, [3]int{170, 220, 110}, 0.45)
		case ul != LEAVES:
			c = mix(c, [3]int{150, 206, 100}, 0.25)
		}
		if dn != LEAVES {
			c = shadeRGB(c, 0.72)
		}
		if leafPocket(gx, wy) {
			c = shadeRGB(c, 0.78)
		}
	}
	return c
}

func wallPixel(wl, wpt uint8, wx, wy, sh int) [3]int {
	c := blockRGB[wl]
	if wpt != 0 {
		c = mix(c, paintRGB[wpt], 0.7)
	}
	c = shadeRGB([3]int{c[0] + sh/2, c[1] + sh/2, c[2] + sh/2}, 0.5)
	gx := wx + baseChunk*CW
	switch wl {
	case BRICK, STONE, OBSIDIAN: // mortar lines
		if wy%4 == 0 || (gx+(wy/4)*3)%8 == 0 {
			c = shadeRGB(c, 0.7)
		}
	case PLANK: // boards
		if wy%5 == 0 {
			c = shadeRGB(c, 0.72)
		}
	case GLASS:
		if (gx+wy)%6 == 0 {
			c = mix(c, [3]int{220, 240, 255}, 0.3)
		}
	}
	return c
}

// A placed torch: a little stick with a flickering flame, drawn over the background
func torchPixel(wx, wy int, bg [3]int) [3]int {
	// the torch block itself is one cell; flicker its colour
	f := 0.8 + 0.2*math.Sin(float64(frame)*0.3+float64(wx*7+wy))
	return mix(bg, shadeRGB(blockRGB[TORCH], f), 0.95)
}
