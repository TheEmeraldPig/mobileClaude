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
func daylight() float64 { return smoothstep(-0.15, 0.25, sunHeight()) }

func skyLevel() int { return 20 + int(math.Round(44*daylight())) }

func isNight() bool { return daylight() < 0.3 }

// ---- Lighting ----
// Light flows out from the sky, the mote, torches and crystals. It travels easily
// through air and water and only a little way into solid blocks. Unlit = black.

const lightMax = 64 // light falls by 2 per cell through air (3 diagonally), 6 through solid blocks

var (
	light            []uint8
	lightW, lightH   int
	lightX0, lightY0 int
	buckets          [lightMax + 1][]int32
)

func lightAt(x, y int) int {
	lx, ly := x-lightX0, y-lightY0
	if lx < 0 || ly < 0 || lx >= lightW || ly >= lightH {
		return 0
	}
	return int(light[ly*lightW+lx])
}

func playerLightLevel() int {
	if countItem("lantern") > 0 {
		return 64
	}
	return 48
}

func computeLight(ox, oy, vw, vh int) {
	const m = lightMax / 2
	lightX0, lightY0, lightW, lightH = ox-m, oy-m, vw+2*m, vh+2*m
	n := lightW * lightH
	if cap(light) < n {
		light = make([]uint8, n)
	}
	light = light[:n]
	for i := range light {
		light[i] = 0
	}
	for i := range buckets {
		buckets[i] = buckets[i][:0]
	}
	seed := func(idx, lvl int) {
		if lvl > lightMax {
			lvl = lightMax
		}
		if lvl > int(light[idx]) {
			light[idx] = uint8(lvl)
			buckets[lvl] = append(buckets[lvl], int32(idx))
		}
	}
	sky := skyLevel()
	for c := 0; c < lightW; c++ {
		wx := lightX0 + c
		if wx < 0 || wx >= W {
			continue
		}
		// sky light reaches down until the first solid block in this column
		top := 0
		for top < H && !blocksLight(grid[top*W+wx]) {
			top++
		}
		for r := 0; r < lightH; r++ {
			wy := lightY0 + r
			if wy < 0 {
				seed(r*lightW+c, sky)
				continue
			}
			if wy >= H {
				break
			}
			if wy < top {
				seed(r*lightW+c, sky)
			}
			if l := blockLight(grid[wy*W+wx]); l > 0 {
				seed(r*lightW+c, l)
			}
		}
	}
	pcx, pcy := int((player.x+player.w/2)/CELL)-lightX0, int((player.y+player.h/2)/CELL)-lightY0
	if pcx >= 0 && pcy >= 0 && pcx < lightW && pcy < lightH {
		seed(pcy*lightW+pcx, playerLightLevel())
	}
	for lvl := lightMax; lvl > 0; lvl-- {
		for k := 0; k < len(buckets[lvl]); k++ {
			idx := int(buckets[lvl][k])
			if int(light[idx]) != lvl {
				continue
			}
			c, r := idx%lightW, idx/lightW
			try := func(nc, nr, cost int) {
				if nc < 0 || nr < 0 || nc >= lightW || nr >= lightH {
					return
				}
				wx, wy := lightX0+nc, lightY0+nr
				if wx < 0 || wx >= W || wy >= H || (wy >= 0 && blocksLight(grid[wy*W+wx])) {
					cost *= 3
				}
				nl := lvl - cost
				ni := nr*lightW + nc
				if nl > int(light[ni]) {
					light[ni] = uint8(nl)
					if nl > 0 {
						buckets[nl] = append(buckets[nl], int32(ni))
					}
				}
			}
			try(c-1, r, 2)
			try(c+1, r, 2)
			try(c, r-1, 2)
			try(c, r+1, 2)
			try(c-1, r-1, 3)
			try(c+1, r-1, 3)
			try(c-1, r+1, 3)
			try(c+1, r+1, 3)
		}
	}
}

var lightCurve = func() [lightMax + 1]float64 {
	var t [lightMax + 1]float64
	for i := range t {
		t[i] = math.Pow(math.Min(1, float64(i)/36), 0.85) // bright near the source, then a soft edge
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
	skyTopDay   = [3]int{88, 120, 212}
	skyLowDay   = [3]int{178, 202, 240}
	skyTopNight = [3]int{8, 8, 24}
	skyLowNight = [3]int{30, 26, 64}
	duskGlow    = [3]int{244, 150, 112}
	farHill     = [3]int{120, 112, 190}
	nearHill    = [3]int{70, 62, 132}
	caveWall    = [3]int{74, 66, 92}
)

// Background pixel for an open-sky cell, in view cell coordinates
func skyPixel(vx, vy, vw, vh int, ox, oy int, avgSurface float64) [3]int {
	d := daylight()
	sh := sunHeight()
	t := math.Max(0, math.Min(1, float64(vy)/float64(vh)))
	top, low := mix(skyTopNight, skyTopDay, d), mix(skyLowNight, skyLowDay, d)
	c := mix(top, low, t)
	if dusk := math.Exp(-(sh / 0.2) * (sh / 0.2)); dusk > 0.02 {
		c = mix(c, duskGlow, dusk*t*0.7)
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
	sunX := float64(vw) * (0.5 + 0.42*math.Cos(2*math.Pi*(worldTime-0.25)+math.Pi))
	sunY := float64(vh)*0.62 - sh*float64(vh)*0.48
	moonX := float64(vw) - sunX
	moonY := float64(vh)*0.62 + sh*float64(vh)*0.48
	if dx, dy := float64(vx)-sunX, float64(vy)-sunY; dx*dx+dy*dy < 36 {
		c = [3]int{255, 236, 160}
	} else if dx*dx+dy*dy < 64 {
		c = mix(c, [3]int{255, 220, 140}, 0.4)
	}
	if dx, dy := float64(vx)-moonX, float64(vy)-moonY; dx*dx+dy*dy < 20 {
		c = [3]int{214, 218, 236}
		if (vx+vy)%5 == 0 {
			c = [3]int{180, 184, 206}
		}
	}
	// clouds drifting across the upper sky
	if t < 0.55 {
		cn := noise2(float64(vx)+float64(ox)*0.15+float64(frame)*0.02, float64(vy)*2.2, 22, 41)
		if cn > 0.7 {
			cloud := mix([3]int{60, 56, 96}, [3]int{244, 244, 252}, d)
			c = mix(c, cloud, math.Min(1, (cn-0.7)*8)*0.85)
		}
	}
	// two layers of distant hills, moving slower than the world
	for _, layer := range []struct {
		f    float64
		amp  float64
		col  [3]int
		salt int
	}{{0.2, 34, farHill, 51}, {0.45, 24, nearHill, 52}} {
		hx := float64(vx) + float64(ox)*layer.f
		horizon := float64(vh)*0.58 + (avgSurface-float64(oy)-float64(vh)*0.58)*layer.f
		hTop := horizon - (noise1(hx, 40, layer.salt)*0.5+0.5)*layer.amp - noise1(hx, 9, layer.salt+5)*4
		if float64(vy) >= hTop {
			hc := mix(layer.col, low, 0.35)
			c = shadeRGB(hc, 0.35+0.65*d)
			break
		}
	}
	return c
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

func drawWorld(screen *ebiten.Image) {
	vw, vh := int(logicalW/zoom/CELL)+2, int(logicalH/zoom/CELL)+2
	viewW, viewH = vw, vh
	if worldImg == nil || worldImg.Bounds().Dx() != vw || worldImg.Bounds().Dy() != vh {
		worldImg = ebiten.NewImage(vw, vh)
		worldPix = make([]byte, vw*vh*4)
	}
	ox, oy := int(math.Floor(cam.x/CELL)), int(math.Floor(cam.y/CELL))
	computeLight(ox, oy, vw, vh)
	pix := worldPix
	surf := make([]int, vw)
	avg := 0.0
	for x := 0; x < vw; x++ {
		surf[x] = surfaceAt(baseChunk*CW + ox + x)
		avg += float64(surf[x])
	}
	avg /= float64(vw)
	set := func(o int, c [3]int) {
		pix[o], pix[o+1], pix[o+2], pix[o+3] = clamp8(c[0]), clamp8(c[1]), clamp8(c[2]), 255
	}
	for y := 0; y < vh; y++ {
		wy := oy + y
		for x := 0; x < vw; x++ {
			wx := ox + x
			o := (y*vw + x) * 4
			lf := lightCurve[lightAt(wx, wy)]
			var t uint8 = STONE
			sh := 0
			if wx >= 0 && wx < W && wy >= 0 && wy < H {
				i := wy*W + wx
				t, sh = grid[i], int(shade[i])
			} else if wy < 0 {
				t = EMPTY
			}
			openSky := wy < surf[x]
			switch {
			case t == EMPTY || t == WATER || t == TORCH || t == GLASS || t == LAVA:
				var bg [3]int
				if openSky {
					bg = skyPixel(x, y, vw, vh, ox+baseChunk*CW, oy, avg)
				} else {
					bg = shadeRGB([3]int{caveWall[0] + sh, caveWall[1] + sh, caveWall[2] + sh}, lf)
				}
				if t == WATER {
					wc := blockRGB[WATER]
					wc = [3]int{wc[0] + sh/2, wc[1] + sh/2, wc[2] + sh/2}
					if !openSky {
						wc = shadeRGB(wc, lf)
					} else {
						wc = shadeRGB(wc, 0.35+0.65*lf)
					}
					bg = mix(bg, wc, 0.78)
				}
				if t == TORCH {
					bg = torchPixel(wx, wy, bg)
				}
				if t == GLASS {
					g := shadeRGB(blockRGB[GLASS], 0.3+0.7*lf)
					if (wx+wy)%7 == 0 {
						g = [3]int{240, 250, 255}
					}
					bg = mix(bg, g, 0.4)
				}
				if t == LAVA { // glowing, slowly churning
					f := 0.85 + 0.15*math.Sin(float64(frame)*0.04+float64(wx)*0.7+float64(wy)*0.3)
					lc := blockRGB[LAVA]
					bg = shadeRGB([3]int{lc[0], lc[1] + sh*2, lc[2] + sh}, f)
				}
				set(o, bg)
			default:
				c := blockRGB[t]
				if isOre(t) && (sh+10)%3 == 0 {
					c = blockRGB[STONE]
				}
				c = [3]int{c[0] + sh, c[1] + sh, c[2] + sh}
				if t == CORE {
					c = mix(c, [3]int{255, 240, 255}, 0.5+0.5*math.Sin(float64(frame)*0.08))
					lf = 1
				}
				if t == FURNACE && (wx+wy)%3 == 0 {
					c = [3]int{255, 150, 60}
					lf = math.Max(lf, 0.9)
				}
				if t == CRYSTAL {
					c = mix(c, [3]int{220, 250, 255}, 0.2+0.2*math.Sin(float64(frame)*0.05+float64(wx)))
					lf = math.Max(lf, 0.85)
				}
				set(o, shadeRGB(c, lf))
			}
		}
	}
	put := func(x, y int, c [3]int, s int) {
		if x < 0 || y < 0 || x >= vw || y >= vh {
			return
		}
		o := (y*vw + x) * 4
		pix[o], pix[o+1], pix[o+2] = clamp8(c[0]+s), clamp8(c[1]+s), clamp8(c[2]+s)
	}
	// creatures are only visible where there is light (their eyes always glow)
	lit := func(x, y int, c [3]int, s int) {
		f := lightCurve[lightAt(x+ox, y+oy)]
		put(x, y, shadeRGB([3]int{c[0] + s, c[1] + s, c[2] + s}, f), 0)
	}
	drawEntities(lit, put, ox, oy)
	blend := func(x, y int, c [3]int, a float64) {
		if x < 0 || y < 0 || x >= vw || y >= vh {
			return
		}
		o := (y*vw + x) * 4
		for k := 0; k < 3; k++ {
			pix[o+k] = clamp8(int(float64(pix[o+k])*(1-a) + float64(c[k])*a))
		}
	}
	drawHologram(blend, ox, oy)
	drawPlayer(put, ox, oy)
	for _, q := range parts {
		put(int(math.Floor(q.x))-ox, int(math.Floor(q.y))-oy, q.c, 0)
	}
	drawEnergy(put, ox, oy)
	if cur == scPlaying {
		drawAim(put, ox, oy)
	}
	worldImg.WritePixels(pix)
	k := zoom * CELL * uiScale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate((float64(ox*CELL)-cam.x)*zoom*uiScale, (float64(oy*CELL)-cam.y)*zoom*uiScale)
	screen.DrawImage(worldImg, op)
}

// A placed torch: a little stick with a flickering flame, drawn over the background
func torchPixel(wx, wy int, bg [3]int) [3]int {
	// the torch block itself is one cell; flicker its colour
	f := 0.8 + 0.2*math.Sin(float64(frame)*0.3+float64(wx*7+wy))
	return mix(bg, shadeRGB(blockRGB[TORCH], f), 0.95)
}
