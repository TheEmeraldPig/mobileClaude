package game

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The skill tree is its own full screen: hexagons in a honeycomb flower around
// Origin on a starry backdrop, with the picked skill's details on the right.

var whitePx = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// fillPoly fills a closed polygon in one colour.
func fillPoly(dst *ebiten.Image, pts [][2]float64, c color.Color) {
	var p vector.Path
	for i, pt := range pts {
		if i == 0 {
			p.MoveTo(float32(pt[0]), float32(pt[1]))
		} else {
			p.LineTo(float32(pt[0]), float32(pt[1]))
		}
	}
	p.Close()
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	r, g, b, a := c.RGBA()
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(r)/0xffff, float32(g)/0xffff, float32(b)/0xffff, float32(a)/0xffff
	}
	dst.DrawTriangles(vs, is, whitePx, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// hexPts are the corners of a pointy-top hexagon of radius s around (x, y).
func hexPts(x, y, s float64) [][2]float64 {
	pts := make([][2]float64, 6)
	for i := range pts {
		a := math.Pi/180*(60*float64(i)) - math.Pi/2
		pts[i] = [2]float64{x + s*math.Cos(a), y + s*math.Sin(a)}
	}
	return pts
}

// treeLayout is where the tree screen puts things this frame.
type treeLayout struct {
	back, unlock, info rect
	centers            [][2]float64 // a centre per node in tree
	s                  float64      // hexagon radius
}

func (g *Game) layTree() treeLayout {
	u := math.Max(g.u, 1)
	wu := g.W / u
	var T treeLayout
	T.back = rect{(wu - 40) * u, 8 * u, 32 * u, 32 * u}
	iw := math.Min(200, wu*0.3)
	T.info = rect{(wu - iw - 12) * u, 50 * u, iw * u, 298 * u}
	T.unlock = rect{T.info.x + 12*u, T.info.y + T.info.h - 56*u, T.info.w - 24*u, 44 * u}
	area := rect{12 * u, 48 * u, T.info.x - 24*u, g.H - 60*u}
	// fit the flower's bounding box (in units of the hexagon radius) into the area
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, n := range tree {
		x, y := math.Sqrt(3)*(float64(n.q)+float64(n.r)/2), 1.5*float64(n.r)
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	bw, bh := maxX-minX+math.Sqrt(3), maxY-minY+2
	T.s = math.Min(area.w/bw, area.h/bh) * 0.98
	ox := area.cx() - (minX+maxX)/2*T.s
	oy := area.cy() - (minY+maxY)/2*T.s
	for _, n := range tree {
		x, y := math.Sqrt(3)*(float64(n.q)+float64(n.r)/2), 1.5*float64(n.r)
		T.centers = append(T.centers, [2]float64{ox + x*T.s, oy + y*T.s})
	}
	return T
}

func (g *Game) tapTree(x, y float64) {
	T := g.layTree()
	switch {
	case T.back.in(x, y):
		g.panel = ""
		g.play("tap")
		return
	case T.unlock.in(x, y):
		g.unlock(g.selN)
		return
	}
	for i, c := range T.centers {
		if math.Hypot(x-c[0], y-c[1]) < T.s*0.9 {
			if g.selN == tree[i].id && g.canUnlock(&tree[i]) {
				g.unlock(tree[i].id) // a second tap unlocks
			} else {
				g.selN = tree[i].id
				g.play("tap")
			}
			return
		}
	}
}

func nodeColor(n *node) color.RGBA {
	if n.branch < 0 {
		return colT
	}
	return rgb(branches[n.branch].col)
}

func (g *Game) drawTree(dst *ebiten.Image) {
	u := g.u
	T := g.layTree()
	// deep space: a dark gradient, slow nebula clouds and stars
	for i := 0; i < 12; i++ {
		y := g.H * float64(i) / 12
		fill(dst, 0, y, g.W, g.H/12+1, mix(rgb(0x05030f), rgb(0x1b1036), float64(i)/12))
	}
	for i, c := range []color.NRGBA{{120, 60, 200, 26}, {40, 90, 200, 22}, {200, 70, 150, 18}} {
		a := g.t*0.05 + float64(i)*2.1
		x, y := g.W*(0.35+0.2*math.Cos(a)), g.H*(0.5+0.25*math.Sin(a*1.3))
		for k := 3; k > 0; k-- {
			vector.DrawFilledCircle(dst, float32(x), float32(y), float32(g.H*0.18*float64(k)), c, true)
		}
	}
	sz := math.Max(1, math.Round(u*1.5))
	for _, st := range g.sky.stars {
		a := 0.35 + 0.65*math.Abs(math.Sin(g.t*1.1+st[2]))
		fill(dst, math.Round(st[0]*g.W), math.Round(st[1]*1.8*g.H), sz, sz, color.NRGBA{230, 220, 255, uint8(a * 255)})
	}

	// links between skills
	for i, n := range tree {
		if n.req == "" {
			continue
		}
		j := 0
		for k := range tree {
			if tree[k].id == n.req {
				j = k
			}
		}
		a, b := T.centers[j], T.centers[i]
		c := color.NRGBA{90, 70, 150, 255}
		w := 3 * u
		if g.has(n.id) {
			cc := nodeColor(&tree[i])
			c, w = color.NRGBA{cc.R, cc.G, cc.B, 255}, 4*u
		}
		vector.StrokeLine(dst, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), float32(w+2*u), colO, true)
		vector.StrokeLine(dst, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), float32(w), c, true)
	}

	// the hexagons
	for i := range tree {
		n := &tree[i]
		x, y := T.centers[i][0], T.centers[i][1]
		s := T.s * 0.92
		bc := nodeColor(n)
		owned, ready := g.has(n.id), g.canUnlock(n)
		body := mul(bc, 0.28) // locked
		edge := mul(bc, 0.45)
		switch {
		case owned:
			body, edge = bc, mix(bc, colW, 0.5)
		case ready && g.st.Points >= n.cost:
			pulse := 0.5 + 0.5*math.Sin(g.t*4+float64(i))
			body, edge = mul(bc, 0.55+0.2*pulse), mix(bc, colW, 0.3)
			vector.DrawFilledCircle(dst, float32(x), float32(y), float32(s*1.15), color.NRGBA{bc.R, bc.G, bc.B, uint8(30 + 40*pulse)}, true)
		case ready:
			body, edge = mul(bc, 0.42), mul(bc, 0.8)
		}
		fillPoly(dst, hexPts(x, y, s), colO)
		fillPoly(dst, hexPts(x, y, s-2*u), edge)
		fillPoly(dst, hexPts(x+0.6*u, y+0.8*u, s-4.5*u), mul(body, 0.8))
		fillPoly(dst, hexPts(x, y, s-5*u), body)
		if g.selN == n.id {
			pts := hexPts(x, y, s+3*u)
			for k := range pts {
				a, b := pts[k], pts[(k+1)%6]
				vector.StrokeLine(dst, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), float32(2*u), colW, true)
			}
		}
		tc := colW
		if !owned && !ready {
			tc = rgb(0x9a90b8)
		}
		fs := math.Max(6, math.Min(9, T.s/u*0.22))
		lines := strings.SplitN(n.name, " ", 2)
		if len(lines) == 2 && textW(n.name, face(fSmall, fs*u)) < s*1.5 {
			lines = []string{n.name}
		}
		ly := y - float64(len(lines)-1)*fs*0.9*u - 3*u
		for _, l := range lines {
			say(dst, l, face(fSmall, fs*u), x, ly, alCenter, tc, math.Round(u))
			ly += fs * 1.8 * u
		}
		cost := itoa(n.cost) + " PT"
		if owned {
			cost = "OWNED"
		}
		say(dst, cost, face(fSmall, fs*0.8*u), x, y+s*0.55, alCenter, mix(tc, colT, 0.6), math.Round(u))
	}

	// header
	say(dst, "SKILL TREE", face(fBig, 24*u), g.W/2, 24*u, alCenter, colT, math.Round(2*u))
	icon(dst, "star", 22*u, 24*u, 16*u)
	say(dst, itoa(g.st.Points)+" SKILL POINTS", face(fBig, 16*u), 34*u, 24*u, alLeft, colT, math.Round(1.5*u))
	bevel(dst, T.back, skOrange, math.Max(1, math.Round(u*1.3)))
	say(dst, "X", face(fBig, 18*u), T.back.cx(), T.back.cy()-u, alCenter, colW, math.Round(1.5*u))

	// details of the picked skill
	p := math.Max(1, math.Round(u*1.3))
	bevel(dst, T.info, skDark, p)
	n := nodeByID(g.selN)
	if n == nil {
		g.wrapSay(dst, "Tap a hexagon to see a skill. Skills cost skill points, which you earn by rebirthing (1, plus 1 for every Outbreak beaten). Each branch starts at Origin in the middle.", T.info.x+12*u, T.info.y+18*u, T.info.w-24*u, 8, rgb(0xd8ccff))
		return
	}
	bc := nodeColor(n)
	bname := "THE ROOT"
	if n.branch >= 0 {
		bname = branches[n.branch].name
	}
	say(dst, bname, face(fSmall, 7*u), T.info.x+12*u, T.info.y+16*u, alLeft, bc, math.Round(u))
	say(dst, n.name, face(fBig, 20*u), T.info.x+12*u, T.info.y+38*u, alLeft, colW, math.Round(1.5*u))
	y := g.wrapSay(dst, n.desc, T.info.x+12*u, T.info.y+64*u, T.info.w-24*u, 9, colT)
	status := ""
	switch {
	case g.has(n.id):
		status = "You have this skill."
	case !g.canUnlock(n):
		status = "Unlock " + nodeByID(n.req).name + " first."
	case g.st.Points < n.cost:
		status = "Needs " + itoa(n.cost) + " skill points. Rebirth to earn more."
	default:
		status = "Costs " + itoa(n.cost) + " skill points."
	}
	g.wrapSay(dst, status, T.info.x+12*u, y+10*u, T.info.w-24*u, 7, rgb(0xd8ccff))
	ok := g.canUnlock(n) && g.st.Points >= n.cost
	sk := skGrey
	label := "LOCKED"
	switch {
	case g.has(n.id):
		label = "OWNED"
	case ok:
		sk, label = skGreen, "UNLOCK"
	}
	bevel(dst, T.unlock, sk, p)
	say(dst, label, face(fBig, 18*u), T.unlock.cx(), T.unlock.cy()-u, alCenter, colW, math.Round(1.5*u))
}

// wrapSay writes text wrapped to width w starting at (x, y) and returns the y below it.
func (g *Game) wrapSay(dst *ebiten.Image, s string, x, y, w, size float64, c color.Color) float64 {
	u := g.u
	f := face(fSmall, size*u)
	line := ""
	step := size * 1.9 * u
	for _, word := range strings.Fields(s) {
		try := strings.TrimSpace(line + " " + word)
		if line != "" && textW(try, f) > w {
			say(dst, line, f, x, y, alLeft, c, math.Round(u))
			y += step
			line = word
		} else {
			line = try
		}
	}
	if line != "" {
		say(dst, line, f, x, y, alLeft, c, math.Round(u))
		y += step
	}
	return y
}
