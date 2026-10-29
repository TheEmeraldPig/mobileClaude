package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Motion and effects: the slowly swirling warm background, cards that glide on springs
// (dealt from the deck, lifted when picked, flung into play, swept away), floating
// score numbers and a little screen shake on big hands.

// ---- Background ----

var bgShaderSrc = []byte(`//kage:unit pixels
package main

var Time float
var Size vec2
var Heat float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	uv := (dst.xy - Size*0.5) / Size.y
	t := Time * 0.04
	r := length(uv)
	a := atan2(uv.y, uv.x)
	// a lazy swirl of warm bands
	sw := a + r*3.2 - t*2.0 + sin(r*5.0 - t*3.0)*0.35
	w1 := sin(sw*3.0 + sin(uv.x*3.0 + t*2.0)*1.2)
	w2 := sin(r*9.0 - t*4.0 + w1*1.4)
	k := clamp(0.5 + 0.3*w1 + 0.2*w2, 0.0, 1.0)
	deep := vec3(0.20, 0.07, 0.06)
	mid := vec3(0.42, 0.13, 0.08)
	hot := vec3(0.62, 0.27, 0.09)
	c := mix(deep, mid, smoothstep(0.15, 0.6, k))
	c = mix(c, hot, smoothstep(0.65, 1.0, k)*(0.55+Heat*0.45))
	// darker toward the edges
	c *= 1.05 - r*0.55
	return vec4(c, 1)
}
`)

var (
	bgShader *ebiten.Shader
	bgHeat   float64 // flares up when a big hand scores
)

func initFX() {
	s, err := ebiten.NewShader(bgShaderSrc)
	if err == nil {
		bgShader = s
	}
}

func drawBackground(dst *ebiten.Image) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	if bgShader == nil {
		dst.Fill(rgb(0x3a1410))
		return
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time": float32(frame) / 60,
		"Size": []float32{float32(w), float32(h)},
		"Heat": float32(bgHeat),
	}
	dst.DrawRectShader(w, h, bgShader, op)
}

// ---- Cards on springs ----

// Where a card is drawn, moving toward where it should be
type cardVis struct {
	x, y, rot, scale     float64
	vx, vy, vrot, vscale float64
	seen                 int // the last frame it was placed
}

var visCards = map[int]*cardVis{}

// placeCard moves card id toward a target and returns where to draw it now. A card
// that wasn't on the table last frame starts from (fromX, fromY).
func placeCard(id int, tx, ty, trot, tscale, fromX, fromY float64) *cardVis {
	v := visCards[id]
	if v == nil || frame-v.seen > 2 {
		v = &cardVis{x: fromX, y: fromY, rot: -0.6 + rand.Float64()*0.3, scale: 0.8}
		visCards[id] = v
	}
	if v.seen != frame {
		v.seen = frame
		const k, damp = 0.16, 0.68
		v.vx = (v.vx + (tx-v.x)*k) * damp
		v.vy = (v.vy + (ty-v.y)*k) * damp
		v.vrot = (v.vrot + (trot-v.rot)*k) * damp
		v.vscale = (v.vscale + (tscale-v.scale)*k) * damp
		v.x += v.vx
		v.y += v.vy
		v.rot += v.vrot
		v.scale += v.vscale
	}
	return v
}

// Give a card a push (when it's picked or scores)
func kickCard(id int, dy, drot, dscale float64) {
	if v := visCards[id]; v != nil {
		v.vy += dy
		v.vrot += drot
		v.vscale += dscale
	}
}

// Cards flying away after being played or discarded
type flyCard struct {
	c            Card
	x, y, rot    float64
	vx, vy, vrot float64
	w, h         float64
	back         bool
	born         int
}

var flying []*flyCard

func flingCard(c Card, v *cardVis, w, h float64, back bool) {
	if v == nil {
		return
	}
	flying = append(flying, &flyCard{c: c, x: v.x, y: v.y, rot: v.rot, vx: 9 + rand.Float64()*5, vy: -6 - rand.Float64()*4,
		vrot: 0.08 + rand.Float64()*0.1, w: w, h: h, back: back, born: frame})
}

func drawFlying() {
	out := flying[:0]
	for _, f := range flying {
		f.vy += 0.6
		f.x += f.vx
		f.y += f.vy
		f.rot += f.vrot
		if f.x > logicalW+f.w || f.y > logicalH+f.h || frame-f.born > 120 {
			continue
		}
		out = append(out, f)
		ff := *f
		ops = append(ops, func(dst *ebiten.Image) {
			drawCardFX(dst, ff.c, ff.x, ff.y, ff.w, ff.h, ff.rot, 1, cardLook{faceDown: ff.back})
		})
	}
	flying = out
}

// ---- Floating numbers ----

type floater struct {
	text string
	x, y float64
	col  color.Color
	born int
	size float64
}

var floaters []*floater

func floatText(s string, x, y, size float64, c color.Color) {
	floaters = append(floaters, &floater{text: s, x: x, y: y, col: c, born: frame, size: size})
}

func drawFloaters() {
	out := floaters[:0]
	for _, f := range floaters {
		age := float64(frame - f.born)
		if age > 55 {
			continue
		}
		out = append(out, f)
		// pop in big, then drift up and fade
		s := f.size * (1 + 0.5*math.Exp(-age/5))
		y := f.y - age*0.6
		a := math.Min(1, (55-age)/15)
		t, c := f.text, withAlpha(f.col, a)
		x := f.x
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, t, x, y, textOpt{size: s, display: true, col: c, align: text.AlignCenter, shadow: a > 0.5})
		})
	}
	floaters = out
}

// ---- Shake ----

var shake float64

func shakeOffset() (float64, float64) {
	if shake < 0.2 {
		return 0, 0
	}
	return (rand.Float64()*2 - 1) * shake, (rand.Float64()*2 - 1) * shake
}

func updateFX() {
	shake *= 0.85
	bgHeat *= 0.97
}
