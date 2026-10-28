package game

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Characters: humanoids with alien touches (a big glossy eye, odd skin tones, antennae
// or horns), drawn in layers from their mmo.Look, with a dark outline like every sprite
// in the game. The body is 10x18 cells (the collision box); features like antennae
// stick out above it.

const (
	sprW    = 10
	sprH    = 18
	bodyTop = 5 // rows above the box for antennae, horns and crests
)

type pose struct {
	face  int    // 1 right, -1 left
	legs  string // "stand", "a", "b", "air"
	blink bool
	swing float64 // < 0: not using a tool; 0..1 through a swing
	aim   float64 // radians, where the tool points
	tool  int
	bob   int
}

type px struct {
	x, y int
	c    [3]int
}

var outlineRGB = [3]int{34, 28, 64}

// The pixels of a character in sprite space (x 0..9, y -bodyTop..17), facing right
func bodyPixels(l mmo.Look, p pose) []px {
	const cw, ch = sprW + 4, sprH + bodyTop + 2
	var canvas [ch][cw][3]int
	var filled [ch][cw]bool
	set := func(x, y int, c [3]int) {
		cx, cy := x+2, y+bodyTop+1
		if cx >= 0 && cy >= 0 && cx < cw && cy < ch {
			canvas[cy][cx], filled[cy][cx] = c, true
		}
	}
	rect := func(x0, y0, x1, y1 int, c [3]int) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				set(x, y, c)
			}
		}
	}
	skin := mmo.SkinTones[l.Skin]
	skinD := shadeRGB(skin, 0.78)
	skinL := mix(skin, [3]int{255, 255, 255}, 0.3)
	shirt := mmo.ClothColours[l.Shirt]
	shirtD := shadeRGB(shirt, 0.72)
	pants := mmo.ClothColours[l.Pants]
	pantsD := shadeRGB(pants, 0.72)
	hair := mmo.HairColours[l.HairCol]
	hairL := mix(hair, [3]int{255, 255, 255}, 0.25)
	boot := [3]int{54, 44, 60}

	// back leg and arm (darker, behind the body)
	switch p.legs {
	case "a":
		rect(2, 13, 3, 15, pantsD)
		rect(1, 16, 2, 16, pantsD)
		rect(0, 17, 2, 17, shadeRGB(boot, 0.8))
	case "b":
		rect(5, 13, 6, 16, pantsD)
		rect(5, 17, 7, 17, shadeRGB(boot, 0.8))
	case "air":
		rect(3, 13, 4, 15, pantsD)
		rect(2, 16, 3, 16, shadeRGB(boot, 0.8))
	default:
		rect(3, 13, 4, 16, pantsD)
		rect(3, 17, 4, 17, shadeRGB(boot, 0.8))
	}
	armSwing := map[string]int{"a": 1, "b": -1}[p.legs]
	rect(3+armSwing, 7, 3, 10, shirtD)
	set(3+armSwing, 11, skinD)

	// body
	rect(3, 7, 6, 11, shirt)
	rect(3, 7, 6, 7, mix(shirt, [3]int{255, 255, 255}, 0.18))
	rect(3, 12, 6, 12, pantsD) // belt
	rect(4, 6, 5, 6, skinD)    // neck

	// front leg
	switch p.legs {
	case "a":
		rect(5, 13, 6, 14, pants)
		rect(6, 15, 7, 16, pants)
		rect(6, 17, 8, 17, boot)
	case "b":
		rect(3, 13, 4, 15, pants)
		rect(2, 16, 3, 16, pants)
		rect(1, 17, 3, 17, boot)
	case "air":
		rect(5, 13, 6, 14, pants)
		rect(6, 15, 7, 15, pants)
		rect(6, 16, 8, 16, boot)
	default:
		rect(5, 13, 6, 16, pants)
		rect(5, 17, 7, 17, boot)
	}

	// head: rounded, with a big glossy alien eye
	rect(3, 0, 6, 0, skin)
	rect(2, 1, 7, 4, skin)
	rect(3, 5, 6, 5, skin)
	set(3, 1, skinL)
	set(2, 4, skinD)
	set(3, 5, skinD)
	if p.blink {
		rect(5, 3, 6, 3, skinD)
	} else {
		rect(5, 2, 6, 3, [3]int{246, 246, 252})
		eye := mmo.EyeColours[l.Eyes]
		rect(6, 2, 6, 3, eye)
		set(7, 2, mix(eye, skin, 0.5)) // the eye wraps round the front of the face
		set(5, 2, [3]int{255, 255, 255})
	}
	set(7, 4, skinD) // mouth

	// hair
	switch mmo.HairStyles[l.Hair] {
	case "Short":
		rect(3, 0, 6, 0, hair)
		rect(2, 1, 4, 1, hair)
		rect(2, 2, 2, 3, hair)
		set(5, 0, hairL)
	case "Long":
		rect(3, 0, 6, 0, hair)
		rect(2, 1, 4, 1, hair)
		rect(1, 2, 2, 7, hair)
		set(5, 0, hairL)
	case "Spiky":
		set(3, -1, hair)
		set(5, -1, hair)
		set(7, 0, hair)
		rect(2, 0, 6, 0, hair)
		rect(2, 1, 5, 1, hair)
		rect(2, 2, 2, 2, hair)
	case "Bun":
		rect(3, 0, 6, 0, hair)
		rect(2, 1, 4, 1, hair)
		rect(2, 2, 2, 3, hair)
		rect(0, 0, 1, 2, hair)
		set(0, 0, hairL)
	case "Crest":
		rect(3, -2, 5, -1, hair)
		rect(3, 0, 6, 0, hair)
		set(4, -2, hairL)
	}

	// alien features
	switch mmo.Features[l.Feature] {
	case "Antennae": // like the mote's
		for _, x := range []int{3, 6} {
			set(x, -1, outlineRGB)
			set(x, -2, outlineRGB)
			tip := tipRGB
			tip[1] = min(255, tip[1]+int(math.Sin(float64(frame)*0.08+float64(x))*20))
			set(x, -3, tip)
		}
	case "Pointed ears":
		set(1, 2, skin)
		set(0, 1, skin)
		set(1, 3, skinD)
	case "Horns":
		bone := [3]int{236, 226, 204}
		set(3, -1, bone)
		set(2, -2, bone)
		set(2, -3, shadeRGB(bone, 0.85))
		set(6, -1, bone)
		set(7, -2, bone)
	case "Head fin":
		fin := mix(skin, hair, 0.5)
		rect(2, -1, 5, -1, fin)
		rect(2, -2, 4, -2, fin)
		set(2, -3, fin)
	}

	// front arm: hanging and swinging, or reaching out with the tool
	if p.swing >= 0 {
		dx, dy := math.Cos(p.aim), math.Sin(p.aim)
		if p.face < 0 {
			dx = -dx // sprite space faces right
		}
		for k := 0.0; k <= 3; k++ {
			set(int(math.Round(5+dx*k)), int(math.Round(8+dy*k)), mix(shirt, outlineRGB, 0.28))
		}
		set(int(math.Round(5+dx*4)), int(math.Round(8+dy*4)), skin)
	} else {
		sleeve := mix(shirt, outlineRGB, 0.28)
		rect(5, 7, 5, 9, sleeve)
		set(5-armSwing, 10, sleeve)
		set(5-armSwing, 11, sleeve)
		set(5-armSwing, 12, skin)
	}

	// outline everything
	out := make([]px, 0, 200)
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			if filled[y][x] {
				out = append(out, px{x - 2, y - bodyTop - 1, canvas[y][x]})
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && ny >= 0 && nx < cw && ny < ch && filled[ny][nx] {
					out = append(out, px{x - 2, y - bodyTop - 1, outlineRGB})
					break
				}
			}
		}
	}
	return out
}

// Draw a character whose box's top-left is at world px (x, y)
func drawBody(put func(x, y int, c [3]int, s int), ox, oy int, x, y float64, l mmo.Look, p pose) {
	bx := int(math.Round(x/CELL)) - ox
	by := int(math.Round(y/CELL)) - oy + p.bob
	for _, q := range bodyPixels(l, p) {
		sx := q.x
		if p.face < 0 {
			sx = sprW - 1 - q.x
		}
		put(bx+sx, by+q.y, q.c, 0)
	}
	if p.swing >= 0 {
		drawTool(put, bx, by, p)
	}
}

// The tool in hand, from the hand along the aim (swinging through an arc for the sword)
func drawTool(put func(x, y int, c [3]int, s int), bx, by int, p pose) {
	hx, hy := float64(bx)+5+math.Cos(p.aim)*4, float64(by)+8+math.Sin(p.aim)*4
	if p.face < 0 {
		hx = float64(bx) + sprW - 1 - 5 + math.Cos(p.aim)*4
	}
	a := p.aim
	if p.tool == mmo.ToolSword {
		a += (-1.3 + 2.6*easeOut(p.swing)) * float64(p.face)
		blade, edge, hilt := [3]int{214, 224, 240}, [3]int{255, 255, 255}, [3]int{255, 222, 120}
		for k := 0.0; k <= 10; k += 0.5 {
			c := blade
			switch {
			case k < 2:
				c = [3]int{110, 78, 52}
			case k < 3:
				c = hilt
			case int(k*2)%3 == 0:
				c = edge
			}
			put(int(math.Round(hx+math.Cos(a)*k)), int(math.Round(hy+math.Sin(a)*k)), c, 0)
		}
		return
	}
	// pickaxe: a handle, and a head across its end
	a += math.Sin(p.swing*math.Pi*2) * 0.6 * float64(p.face)
	for k := 0.0; k <= 7; k += 0.5 {
		put(int(math.Round(hx+math.Cos(a)*k)), int(math.Round(hy+math.Sin(a)*k)), [3]int{140, 98, 60}, 0)
	}
	ex, ey := hx+math.Cos(a)*7, hy+math.Sin(a)*7
	for k := -3.0; k <= 3; k += 0.5 {
		put(int(math.Round(ex+math.Cos(a+math.Pi/2)*k)), int(math.Round(ey+math.Sin(a+math.Pi/2)*k)), [3]int{176, 182, 196}, 0)
	}
}

func easeOut(t float64) float64 { return 1 - (1-t)*(1-t) }
