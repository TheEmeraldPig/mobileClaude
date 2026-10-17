package game

import (
	"image/color"
	"math"
)

// The bug brigade: who they are and how they are painted. Each bug is painted
// facing right with its feet at 0,0 and about 100 units tall, in layers: wings
// behind, body, face (open, blink, attack), wings in front.

const (
	rCommon = iota
	rRare
	rEpic
	rLegend
)

var rarityName = []string{"Common", "Rare", "Epic", "Legendary"}
var rarityCol = []color.RGBA{rgb(0x8c, 0xd8, 0x4a), rgb(0x3c, 0xb0, 0xff), rgb(0xb0, 0x62, 0xff), rgb(0xff, 0xb4, 0x1e)}

// Skill effects, on top of the skill's direct damage.
const (
	skHit    = iota // big hit on the front slug
	skAll           // hits every slug on the field
	skFrenzy        // the whole team attacks twice as fast for a while
	skCrit          // taps crit for a while
	skTime          // adds time to the boss timer
	skGold          // slugs drop triple coins for a while
)

type species struct {
	name, title string
	rarity      int
	dps         float64 // damage per second at level 1
	period      float64 // seconds between attacks
	shot        color.RGBA
	skill       string
	skillText   string
	skKind      int
	skDmg       float64 // skill damage in seconds of the bug's own damage
	cool        float64 // seconds to charge the skill
	fly         bool
	// face layout
	e1x, e1y, e1r, e2x, e2y, e2r float64
	mx, my                       float64
	skin                         color.RGBA // face colour, for closed eyes
	wingX, wingY                 float64    // wing pivot
}

var bugs = []species{
	{name: "Rory", title: "Ladybug", rarity: rCommon, dps: 1, period: 1.1, shot: rgb(0xff, 0x50, 0x50),
		skill: "Spot Storm", skillText: "Hurls a storm of spots at the slug", skKind: skHit, skDmg: 30, cool: 16,
		e1x: 30, e1y: -58, e1r: 12.5, e2x: 55, e2y: -57, e2r: 10.5, mx: 46, my: -42, skin: rgb(0x3e, 0x38, 0x66)},
	{name: "Hopper", title: "Grasshopper", rarity: rCommon, dps: 0.9, period: 0.6, shot: rgb(0x9c, 0xf0, 0x40),
		skill: "Leg Day", skillText: "The whole brigade attacks twice as fast", skKind: skFrenzy, skDmg: 8, cool: 22,
		e1x: 28, e1y: -66, e1r: 12, e2x: 50, e2y: -65, e2r: 10, mx: 44, my: -50, skin: rgb(0x9a, 0xe0, 0x50)},
	{name: "Buzz", title: "Bumblebee", rarity: rRare, dps: 2.6, period: 0.9, shot: rgb(0xff, 0xd0, 0x30),
		skill: "Sting Spree", skillText: "Your taps crit for 8 seconds", skKind: skCrit, skDmg: 12, cool: 20, fly: true,
		e1x: 30, e1y: -62, e1r: 12, e2x: 52, e2y: -61, e2r: 10, mx: 44, my: -46, skin: rgb(0xff, 0xd6, 0x4a), wingX: -6, wingY: -78},
	{name: "Dozer", title: "Rhino Beetle", rarity: rRare, dps: 3.2, period: 1.4, shot: rgb(0x50, 0x90, 0xff),
		skill: "Horn Charge", skillText: "A huge charge, triple damage to bosses", skKind: skHit, skDmg: 45, cool: 24,
		e1x: 30, e1y: -52, e1r: 10.5, e2x: 50, e2y: -51, e2r: 9, mx: 44, my: -38, skin: rgb(0x2c, 0x4a, 0xb0)},
	{name: "Luna", title: "Luna Moth", rarity: rEpic, dps: 6.5, period: 1.2, shot: rgb(0xd0, 0x9a, 0xff),
		skill: "Moon Dust", skillText: "Glittering dust hits every slug", skKind: skAll, skDmg: 25, cool: 20, fly: true,
		e1x: 22, e1y: -68, e1r: 11, e2x: 42, e2y: -67, e2r: 9.5, mx: 34, my: -54, skin: rgb(0xd8, 0xc0, 0xff), wingX: -10, wingY: -70},
	{name: "Glim", title: "Firefly", rarity: rEpic, dps: 6, period: 1.0, shot: rgb(0xff, 0xf0, 0x70),
		skill: "Night Light", skillText: "Dazzles the boss: +6 seconds", skKind: skTime, skDmg: 20, cool: 26, fly: true,
		e1x: 28, e1y: -60, e1r: 11.5, e2x: 49, e2y: -59, e2r: 9.5, mx: 42, my: -45, skin: rgb(0x4a, 0x2a, 0x5a), wingX: -8, wingY: -74},
	{name: "Webby", title: "Jumping Spider", rarity: rEpic, dps: 7, period: 1.1, shot: rgb(0xff, 0x7a, 0xd8),
		skill: "Sticky Loot", skillText: "Slugs drop triple coins for 15 seconds", skKind: skGold, skDmg: 15, cool: 28,
		e1x: 20, e1y: -48, e1r: 12.5, e2x: 42, e2y: -47, e2r: 11, mx: 33, my: -32, skin: rgb(0x8a, 0x4a, 0xd8)},
	{name: "Sir Stag", title: "Stag Beetle", rarity: rLegend, dps: 18, period: 1.3, shot: rgb(0xff, 0xc0, 0x30),
		skill: "Royal Chomp", skillText: "A crushing chomp for massive damage", skKind: skHit, skDmg: 70, cool: 24,
		e1x: 26, e1y: -54, e1r: 10.5, e2x: 45, e2y: -53, e2r: 9, mx: 40, my: -40, skin: rgb(0xc8, 0x7a, 0x2a)},
}

// ---- painting helpers ----

var colLeg = rgb(0x36, 0x2a, 0x52)

// leg paints a little bent leg ending in a round foot.
func leg(c *canvas, col color.RGBA, w float64, p ...float64) {
	l := line(w, w*0.8, p...)
	n := len(p)
	foot := ell(p[n-2]+w*0.4, p[n-1]-w*0.5, w*1.5, w*1.0)
	s := l.or(foot)
	c.part(s, mat{base: col, gloss: 0.4, rim: 0.3}, 2.2)
}

// antenna paints a curved feeler with a ball on the end.
func antenna(c *canvas, col color.RGBA, tip color.RGBA, p ...float64) {
	l := line(2.4, 2.0, p...)
	c.fill(l.grow(2), outlineOf(col))
	c.vol(l, mat{base: col, gloss: 0.3})
	n := len(p)
	b := circ(p[n-2], p[n-1], 5.5)
	c.part(b, mat{base: tip, gloss: 0.9, rim: 0.4}, 2.2)
}

func blush(c *canvas, x, y float64) {
	c.soft(ell(x, y, 5, 3), alpha(rgb(0xff, 0x6a, 0x9a), 0.55), 3)
}

// seam paints the line down the middle of a beetle's shell.
func seam(c *canvas, shell sh, col color.RGBA, p ...float64) {
	c.fill(line(1.6, 1.2, p...).and(shell.grow(-2)), alpha(outlineOf(col), 0.85))
}

// ---- the bodies ----

func paintBugBody(c *canvas, id int) {
	switch id {
	case 0: // Rory the ladybug
		red := rgb(0xf2, 0x36, 0x3c)
		head := rgb(0x3e, 0x38, 0x66)
		leg(c, shade(colLeg, 0.8), 4.2, -26, -16, -34, -6, -36, 0)
		leg(c, shade(colLeg, 0.8), 4.2, 14, -16, 22, -6, 24, 0)
		shell := ell(-8, -50, 50, 40)
		c.part(shell, mat{base: red, gloss: 0.95, shine: 28, rim: 0.55}, 2.6)
		spot := mat{base: rgb(0x30, 0x22, 0x40), gloss: 0.5, shine: 28}
		for _, s := range [][3]float64{{-30, -64, 10}, {-8, -38, 9}, {-40, -36, 7.5}, {4, -72, 7}, {-18, -82, 5.5}} {
			c.volOn(circ(s[0], s[1], s[2]).and(shell), shell, spot)
		}
		seam(c, shell, red, 18, -86, -12, -74, -40, -52, -58, -40)
		leg(c, colLeg, 4.4, -12, -14, -16, -4, -18, 0)
		leg(c, colLeg, 4.4, 28, -18, 34, -6, 38, 0)
		antenna(c, head, head, 30, -78, 26, -92, 16, -104)
		antenna(c, head, head, 50, -80, 56, -94, 64, -104)
		hd := circ(40, -52, 30)
		c.part(hd, mat{base: head, gloss: 0.8, shine: 30, rim: 0.6}, 2.6)
		blush(c, 24, -42)
		blush(c, 62, -42)
	case 1: // Hopper the grasshopper
		g := rgb(0x7e, 0xd6, 0x3a)
		dg := rgb(0x4e, 0xa8, 0x2a)
		// far hind leg
		c.part(capsule(-18, -36, -46, -64, 10, 6), mat{base: shade(dg, 0.8), gloss: 0.3}, 2.2)
		leg(c, shade(dg, 0.75), 3.6, -46, -64, -58, -30, -62, 0)
		leg(c, shade(dg, 0.8), 3.4, 14, -26, 10, -10, 8, 0)
		body := capsule(-50, -32, 10, -42, 18, 24).melt(ell(-20, -30, 30, 18), 10)
		c.part(body, mat{base: g, gloss: 0.7, rim: 0.5}, 2.6)
		belly := ell(-18, -22, 34, 10).and(body)
		c.volOn(belly, body, mat{base: rgb(0xd6, 0xf0, 0x6a), gloss: 0.5})
		for i := 0; i < 4; i++ {
			x := -44 + float64(i)*12
			c.fill(arc(x, -34, 6, 14, math.Pi*0.6, math.Pi*1.4, 1.1).and(body.grow(-2)), alpha(outlineOf(g), 0.5))
		}
		wing := blob(-56, -44, -30, -60, 0, -58, 4, -50, -30, -44)
		c.part(wing, mat{base: dg, gloss: 0.6, flat: 0.3}, 2)
		// near hind leg: thigh up and back, shin down
		thigh := capsule(-14, -34, -44, -70, 12, 6.5)
		c.part(thigh, mat{base: g, gloss: 0.6, rim: 0.5}, 2.4)
		c.fill(line(1, 1, -20, -42, -40, -64).and(thigh.grow(-2)), alpha(outlineOf(g), 0.5))
		leg(c, dg, 3.8, -44, -70, -54, -34, -56, 0)
		leg(c, dg, 3.6, 24, -30, 26, -12, 26, 0)
		antenna(c, g, dg, 26, -88, 10, -112, -14, -124)
		antenna(c, g, dg, 44, -88, 46, -114, 34, -130)
		hd := circ(36, -62, 28).melt(ell(44, -50, 18, 14), 8)
		c.part(hd, mat{base: rgb(0x9a, 0xe4, 0x52), gloss: 0.8, rim: 0.6}, 2.6)
		blush(c, 22, -50)
		blush(c, 60, -50)
	case 2: // Buzz the bumblebee
		y := rgb(0xff, 0xc8, 0x22)
		dk := rgb(0x3a, 0x26, 0x30)
		leg(c, dk, 3.2, -14, -22, -18, -10, -14, -4)
		leg(c, dk, 3.2, 4, -22, 4, -10, 8, -4)
		sting := poly(-52, -54, -76, -46, -52, -38)
		c.part(sting, mat{base: dk, gloss: 0.5}, 2)
		body := ell(-12, -50, 42, 36).fuzz(1.6, 0.4)
		stripes := func(x, yy float64) float64 {
			v := math.Sin((x*0.95 - yy*0.28) * 0.16)
			return 0.22 + 0.78*smooth(-0.15, 0.15, v)
		}
		c.fill(body.grow(2.6), outlineOf(y))
		c.volOn(body, ell(-12, -50, 42, 36), mat{base: y, gloss: 0.55, rim: 0.5, tex: 0.05, texF: 0.5, stripe: stripes})
		leg(c, dk, 3.4, -4, -18, -6, -6, -2, -2)
		leg(c, dk, 3.4, 14, -20, 18, -8, 22, -4)
		antenna(c, dk, dk, 26, -80, 18, -96, 10, -104)
		antenna(c, dk, dk, 44, -82, 50, -98, 60, -104)
		hd := circ(38, -58, 27).fuzz(1, 0.5)
		c.fill(hd.grow(2.6), outlineOf(y))
		c.volOn(hd, circ(38, -58, 27), mat{base: rgb(0xff, 0xd8, 0x4a), gloss: 0.7, rim: 0.5, tex: 0.08})
		blush(c, 22, -46)
		blush(c, 60, -46)
	case 3: // Dozer the rhino beetle
		b := rgb(0x3a, 0x6a, 0xf0)
		hdc := rgb(0x2c, 0x4a, 0xb0)
		leg(c, shade(colLeg, 0.8), 5, -32, -14, -42, -4, -44, 0)
		leg(c, shade(colLeg, 0.8), 5, 8, -14, 16, -4, 18, 0)
		shell := ell(-14, -46, 50, 38)
		c.part(shell, mat{base: b, gloss: 1.3, shine: 40, rim: 0.7}, 2.6)
		seam(c, shell, b, 14, -82, -16, -70, -44, -50, -62, -38)
		c.shine(ell(-30, -70, 14, 6).rot(-30, -70, -0.4), 0.35)
		leg(c, colLeg, 5.2, -16, -12, -22, -4, -24, 0)
		leg(c, colLeg, 5.2, 24, -14, 32, -4, 36, 0)
		horn := line(10, 3.2, 52, -60, 66, -82, 64, -110)
		c.part(horn, mat{base: rgb(0x34, 0x40, 0x8a), gloss: 1, shine: 36, rim: 0.5}, 2.4)
		hd := ell(34, -46, 30, 26)
		c.part(hd, mat{base: hdc, gloss: 1, shine: 34, rim: 0.6}, 2.6)
		c.part(line(5, 2, 30, -70, 34, -84, 30, -92), mat{base: rgb(0x34, 0x40, 0x8a), gloss: 1}, 2)
		blush(c, 22, -34)
		blush(c, 58, -34)
	case 4: // Luna the moth
		l := rgb(0xc8, 0xa8, 0xff)
		leg(c, rgb(0x8a, 0x6a, 0xc8), 2.8, -10, -22, -12, -10, -8, -6)
		body := ell(-8, -48, 26, 30).fuzz(2.4, 0.35)
		c.fill(body.grow(2.6), outlineOf(l))
		c.volOn(body, ell(-8, -48, 26, 30), mat{base: l, gloss: 0.3, rim: 0.5, tex: 0.15, texF: 0.45,
			stripe: func(x, y float64) float64 { return 0.82 + 0.18*smooth(-0.3, 0.3, math.Sin(y*0.3)) }})
		leg(c, rgb(0x8a, 0x6a, 0xc8), 2.8, 6, -22, 8, -10, 12, -6)
		collar := ell(16, -58, 20, 16).fuzz(2.6, 0.4)
		c.fill(collar.grow(2.2), outlineOf(l))
		c.volOn(collar, ell(16, -58, 20, 16), mat{base: rgb(0xfa, 0xf2, 0xff), gloss: 0.2, rim: 0.4, tex: 0.12})
		gold := rgb(0xf0, 0xc0, 0x5a)
		for _, s := range []float64{-1, 1} {
			bx := 24 + s*6
			f := blob(bx, -84, bx-8+s*4, -100, bx-6+s*10, -122, bx+4+s*10, -118, bx+6+s*4, -98)
			c.part(f, mat{base: gold, gloss: 0.4, flat: 0.3}, 2)
			c.fill(line(0.9, 0.7, bx, -86, bx+s*8, -120).and(f), alpha(outlineOf(gold), 0.6))
		}
		hd := circ(32, -66, 23)
		c.part(hd, mat{base: rgb(0xe0, 0xcc, 0xff), gloss: 0.6, rim: 0.5}, 2.6)
		blush(c, 18, -56)
		blush(c, 50, -56)
	case 5: // Glim the firefly
		dk := rgb(0x4a, 0x2a, 0x5a)
		leg(c, colLeg, 3, -6, -24, -8, -12, -4, -8)
		lamp := ell(-46, -46, 22, 20)
		c.glowIn(lamp, rgb(0xff, 0xf0, 0x60), 24, 0.6)
		c.fill(lamp.grow(2.4), rgb(0x7a, 0x44, 0x24))
		c.vol(lamp, mat{base: rgb(0xff, 0xf2, 0x7a), dark: rgb(0xf0, 0xb0, 0x30), gloss: 0.9, flat: 0.25, rim: 0.3})
		for i := 0; i < 3; i++ {
			x := -40 + float64(i)*-6
			c.fill(arc(x, -46, 5, 16, -math.Pi*0.4, math.Pi*0.4, 1).and(lamp.grow(-2)), alpha(rgb(0xe0, 0xa0, 0x30), 0.6))
		}
		body := capsule(-30, -48, 8, -54, 18, 21)
		c.part(body, mat{base: dk, gloss: 0.8, shine: 30, rim: 0.6}, 2.6)
		cover := blob(-36, -62, -6, -76, 12, -66, 0, -50, -30, -44)
		c.part(cover, mat{base: rgb(0xe8, 0x50, 0x44), gloss: 0.9, shine: 30, rim: 0.5}, 2.2)
		seam(c, cover, rgb(0xe8, 0x50, 0x44), 10, -66, -14, -62, -34, -52)
		leg(c, colLeg, 3.2, 12, -26, 14, -12, 18, -8)
		antenna(c, dk, rgb(0xff, 0xe0, 0x60), 26, -80, 20, -98, 8, -106)
		antenna(c, dk, rgb(0xff, 0xe0, 0x60), 42, -82, 50, -98, 60, -104)
		hd := circ(36, -58, 25)
		c.part(hd, mat{base: dk, gloss: 0.8, shine: 30, rim: 0.6}, 2.6)
		cap := ell(36, -76, 20, 9).and(hd)
		c.volOn(cap, hd, mat{base: rgb(0xf0, 0x5a, 0x48), gloss: 0.8})
		blush(c, 22, -46)
		blush(c, 58, -46)
	case 6: // Webby the jumping spider
		p := rgb(0x8a, 0x4a, 0xd8)
		dl := rgb(0x4a, 0x26, 0x86)
		spiderLegs := func(col color.RGBA, dx float64) {
			for i, f := range []float64{-34, -8, 44, 70} {
				hx, hy := 18+dx+float64(i)*6, -36.0
				kx := (hx+f)/2 + (f-hx)*0.25
				ky := -66 + math.Abs(float64(i)-1.5)*5
				leg(c, col, 3.8, hx, hy, kx, ky, f+dx, -2)
				c.part(circ(kx, ky, 4.4), mat{base: col, gloss: 0.5}, 1.8)
			}
		}
		spiderLegs(shade(dl, 0.75), -8)
		abd := ell(-24, -48, 36, 34).fuzz(1.2, 0.4)
		c.fill(abd.grow(2.6), outlineOf(p))
		c.volOn(abd, ell(-24, -48, 36, 34), mat{base: p, gloss: 0.5, rim: 0.5, tex: 0.08})
		heart := blob(-24, -60, -14, -70, -6, -60, -24, -38, -42, -60, -34, -70)
		c.volOn(heart.and(abd), abd, mat{base: rgb(0xff, 0x7a, 0xc8), gloss: 0.6})
		ceph := circ(26, -42, 28)
		spiderLegs(dl, 4)
		c.part(ceph, mat{base: rgb(0x9a, 0x5a, 0xe4), gloss: 0.8, rim: 0.6}, 2.6)
		for _, e := range [][3]float64{{14, -66, 4}, {34, -66, 4}} {
			c.part(circ(e[0], e[1], e[2]), mat{base: rgb(0x24, 0x1c, 0x3a), gloss: 1}, 1.4)
		}
		blush(c, 10, -32)
		blush(c, 50, -32)
	case 7: // Sir Stag the stag beetle
		gd := rgb(0xff, 0xb4, 0x2a)
		br := rgb(0xc0, 0x6c, 0x24)
		leg(c, shade(colLeg, 0.8), 4.6, -34, -14, -44, -4, -46, 0)
		leg(c, shade(colLeg, 0.8), 4.6, 6, -14, 14, -4, 16, 0)
		shell := ell(-16, -48, 50, 38)
		c.part(shell, mat{base: gd, gloss: 1.4, shine: 44, rim: 0.8}, 2.6)
		seam(c, shell, gd, 14, -84, -18, -72, -46, -52, -64, -40)
		c.shine(ell(-32, -72, 14, 6).rot(-32, -72, -0.4), 0.4)
		leg(c, colLeg, 4.8, -18, -12, -24, -4, -26, 0)
		leg(c, colLeg, 4.8, 22, -14, 30, -4, 34, 0)
		jaw := mat{base: rgb(0x8a, 0x44, 0x22), gloss: 1.1, shine: 40, rim: 0.5}
		upper := line(7, 3, 46, -60, 66, -82, 84, -76).or(capsule(66, -80, 70, -92, 3, 2))
		lower := line(6.5, 3, 50, -40, 72, -44, 86, -58).or(capsule(72, -46, 78, -36, 3, 2))
		c.part(lower, jaw, 2.2)
		hd := ell(32, -48, 26, 22)
		c.part(hd, mat{base: br, gloss: 1, shine: 34, rim: 0.6}, 2.6)
		c.part(upper, jaw, 2.2)
		crown := poly(18, -66, 16, -84, 24, -76, 30, -90, 36, -76, 44, -84, 42, -66)
		c.part(crown, mat{base: rgb(0xff, 0xd8, 0x40), gloss: 1.2, shine: 30, flat: 0.3}, 2)
		c.part(circ(30, -72, 3), mat{base: rgb(0xff, 0x40, 0x7a), gloss: 1}, 1.2)
		blush(c, 20, -38)
		blush(c, 52, -38)
	}
}

// paintBugWings paints the back (front=false) or front wings around the wing pivot.
func paintBugWings(c *canvas, id int, front bool) {
	sp := bugs[id]
	px, py := sp.wingX, sp.wingY
	clearWing := mat{base: rgb(0xd8, 0xf4, 0xff), dark: rgb(0x9c, 0xc8, 0xf0), gloss: 1.2, shine: 20, flat: 0.5}
	wing := func(s sh, m mat) {
		c.fill(s.grow(2.2), alpha(outlineOf(m.base), 0.9))
		c.vol(s, m)
	}
	switch id {
	case 2, 5: // bee and firefly: clear wings
		if !front {
			w := ell(px-14, py-24, 14, 26).rot(px, py, -0.5)
			wing(w, clearWing)
		} else {
			w := ell(px+2, py-28, 15, 30).rot(px, py, 0.25)
			wing(w, clearWing)
			c.fill(line(0.9, 0.7, px+2, py-6, px+4, py-46).and(w), rgb(0x9c, 0xc8, 0xf0))
		}
	case 4: // Luna: big pastel wings with eye spots, both behind the body
		if front {
			return
		}
		w := blob(px+4, py, px+14, py-40, px+2, py-70, px-24, py-66, px-26, py-30, px-8, py+4)
		wing(w, mat{base: rgb(0xff, 0xb4, 0xe0), gloss: 0.4, flat: 0.4})
		w = blob(px+2, py, px+2, py-40, px-16, py-80, px-46, py-76, px-54, py-42, px-26, py-6)
		wing(w, mat{base: rgb(0xb8, 0xf0, 0xdc), gloss: 0.5, flat: 0.35})
		c.volOn(ell(px-24, py-44, 19, 25).and(w).minus(ell(px-24, py-44, 15, 21)), w, mat{base: rgb(0x8a, 0xd8, 0xc0), flat: 0.4})
		c.part(circ(px-24, py-46, 9.5), mat{base: rgb(0xff, 0xd0, 0x40), gloss: 0.6}, 1.8)
		c.part(circ(px-24, py-46, 4.8), mat{base: rgb(0x8a, 0x40, 0xc0), gloss: 0.8}, 1)
	}
}

// paintFace paints eyes and mouth: state 0 open, 1 blink, 2 attack, 3 hurt.
func paintFace(c *canvas, sp species, state int) {
	eyes := [][3]float64{{sp.e1x, sp.e1y, sp.e1r}, {sp.e2x, sp.e2y, sp.e2r}}
	for _, e := range eyes {
		paintEye(c, e[0], e[1], e[2], 1, state, sp.skin)
	}
	mx, my := sp.mx, sp.my
	mo := outlineOf(sp.skin)
	if int(sp.skin.R)+int(sp.skin.G)+int(sp.skin.B) < 300 {
		mo = rgb(0xff, 0xd0, 0xe0)
	}
	switch state {
	case 2, 3:
		m := ell(mx, my+1, 6.5, 6)
		c.fill(m.grow(1.8), outlineOf(sp.skin))
		c.fill(m, rgb(0x6a, 0x1c, 0x3a))
		c.fill(ell(mx, my+5, 4.5, 3).and(m), rgb(0xff, 0x7a, 0x9a))
	default:
		c.fill(arc(mx, my-3, 6, 5, math.Pi*0.15, math.Pi*0.85, 1.7), mo)
	}
}

// paintEye paints one big cartoon eye looking dir (+1 right, -1 left).
func paintEye(c *canvas, x, y, r, dir float64, state int, skin color.RGBA) {
	out := rgb(0x24, 0x14, 0x36)
	switch state {
	case 1: // blink: a happy closed curve
		c.fill(circ(x, y, r+1), skin)
		c.fill(arc(x, y-r*0.15, r*0.8, r*0.55, math.Pi*0.1, math.Pi*0.9, r*0.13), out)
		return
	case 3: // hurt: squeezed shut
		c.fill(circ(x, y, r+1), skin)
		k := r * 0.7
		c.fill(line(r*0.13, r*0.13, x-k*dir, y-k*0.7, x+k*0.6*dir, y, x-k*dir, y+k*0.7), out)
		return
	}
	c.fill(circ(x, y, r+2.4), out)
	c.vol(circ(x, y, r), mat{base: colWhite, dark: rgb(0xc4, 0xc0, 0xe6), gloss: 0.3, rim: 0.3})
	px, py := x+dir*r*0.26, y+r*0.06
	pr := r * 0.64
	if state == 2 {
		pr = r * 0.58
	}
	c.vol(circ(px, py, pr), mat{base: rgb(0x2a, 0x1e, 0x46), dark: rgb(0x0e, 0x08, 0x1a), gloss: 0.8, shine: 40})
	c.fill(circ(px-pr*0.34, py-pr*0.38, pr*0.36), colWhite)
	c.fill(circ(px+pr*0.36, py+pr*0.34, pr*0.15), colWhite)
	if state == 2 { // determined brow
		c.fill(line(r*0.16, r*0.12, x-r*0.9*dir, y-r*1.25, x+r*0.8*dir, y-r*0.9), out)
	}
}

// ---- sprites ----

type bugSprite struct {
	back, body, front *pimg
	face              [3]*pimg
}

// bugArt returns the painted layers of bug id at scale z.
func bugArt(id int, z float64) *bugSprite {
	key := "bug" + itoa(id) + "@" + num(z*100)
	sp := bugs[id]
	s := &bugSprite{}
	s.body = paintArt(key, -90, -140, 100, 10, z, func(c *canvas) { paintBugBody(c, id) })
	for st := 0; st < 3; st++ {
		st := st
		s.face[st] = paintArt(key+"f"+itoa(st), -10, -90, 80, -20, z, func(c *canvas) { paintFace(c, sp, st) })
	}
	if id == 2 || id == 4 || id == 5 {
		front := id != 4
		// wings are painted around their pivot so they can flap
		x0, y0, x1, y1 := -70.0, -80.0, 30.0, 20.0
		if id == 4 {
			x0, y0, x1, y1 = -70, -95, 30, 20
		}
		s.back = paintArt(key+"wb", x0+sp.wingX, y0+sp.wingY, x1+sp.wingX, y1+sp.wingY, z, func(c *canvas) { paintBugWings(c, id, false) })
		if front {
			s.front = paintArt(key+"wf", x0+sp.wingX, y0+sp.wingY, x1+sp.wingX, y1+sp.wingY, z, func(c *canvas) { paintBugWings(c, id, true) })
		}
	}
	return s
}
