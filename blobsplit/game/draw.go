package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// layout holds where everything is this frame, in device pixels. Positions are worked
// out in units (the screen is 360 units tall) so phones, tablets and PCs look the same.
type layout struct {
	money, gems          rect
	buy, music, sound    rect
	shop, index, rebirth rect
	world                rect
	cards                [4]rect // three upgrades, then the round
	round                rect
	hand                 []rect // ability card slots
	split, sell          rect
	auto, autosell       rect
	grid                 rect
	cols, grows          int
	cell                 float64
	circle               [][2]float64 // biote centres when they form a circle instead
	panel, close, action rect
	action2, action3     rect
	rows                 []rect // tree nodes, market offers or catalog cells
	p                    float64
}

func (g *Game) lay() layout {
	u := g.u
	if u <= 0 {
		u = 1
	}
	R := func(x, y, w, h float64) rect { return rect{x * u, y * u, w * u, h * u} }
	wu := g.W / u
	var L layout
	L.p = math.Max(1, math.Round(u*1.3))
	leftW := 78.0
	rightW := math.Min(172, wu*0.27)
	rightX := wu - rightW - 10
	L.money = R(10, 6, 140, 34)
	L.gems = R(10, 42, 120, 22)
	for i, r := range []*rect{&L.shop, &L.index, &L.rebirth} {
		*r = R(10, 70+float64(i)*64, leftW-6, 58)
	}
	L.world = R(10, 264, leftW-6, 86)
	L.sound = R(wu-38, 8, 28, 28)
	L.music = R(wu-72, 8, 28, 28)
	L.buy = R(rightX, 8, rightW-72, 28)
	for i := range L.cards {
		L.cards[i] = R(rightX, 44+float64(i)*68, rightW, 62)
	}
	L.round = L.cards[3]
	// ability cards along the top
	slots := g.cardSlots()
	hx0, hx1 := 150.0, rightX-10
	cw := math.Min(28, (hx1-hx0)/float64(slots)-4)
	hx := (hx0+hx1)/2 - (float64(slots)*(cw+4)-4)/2
	for i := 0; i < slots; i++ {
		L.hand = append(L.hand, R(hx+float64(i)*(cw+4), 4, cw, 36))
	}
	cx0, cx1 := 10+leftW+8, rightX-8
	bw := math.Min(170, (cx1-cx0-12)/2)
	bx := (cx0+cx1)/2 - bw - 6
	L.split = R(bx, 360-72, bw, 60)
	L.sell = R(bx+bw+12, 360-72, bw, 60)
	L.auto = R(bx-4, 360-72-16, 46, 20)
	L.autosell = R(bx+bw+12+bw-42, 360-72-16, 46, 20)
	L.grid = R(cx0, 48, cx1-cx0, 360-72-34-48)

	// Biotes are sized to fit how many are on the board, so they only change size when
	// a split makes more of them. They line up in a full rectangle (never a half-empty
	// last row) when one fits the space well, and otherwise form a circle, whichever
	// lets them be bigger.
	n := max(1, len(g.blobs))
	anyBest, rectBest := 0.0, 0.0
	for c := 1; c <= n; c++ {
		r := (n + c - 1) / c
		s := math.Min(L.grid.w/float64(c), L.grid.h/float64(r))
		anyBest = math.Max(anyBest, s)
		if n%c == 0 && s > rectBest {
			rectBest, L.cols, L.grows = s, c, r
		}
	}
	if L.cols == 0 {
		// no room at all (before the screen size is known, e.g. Android's first
		// frame): one row, so positions still work out
		L.cols, L.grows = n, 1
	}
	L.cell = math.Max(0, math.Min(rectBest, 70*u))
	if pts, cell := circleSlots(n, L.grid, 70*u); rectBest < anyBest*0.75 || cell > L.cell {
		L.circle, L.cell = pts, cell
	}
	g.layPanel(&L, wu)
	return L
}

// circleSlots packs n biotes into a circle in the middle of area: a ring when there are
// a few, a sunflower spiral when there are more. It returns their centres and the
// biote size.
func circleSlots(n int, area rect, maxCell float64) ([][2]float64, float64) {
	pts := make([][2]float64, n)
	switch {
	case n == 1:
	case n <= 7:
		for i := range pts {
			a := 2*math.Pi*float64(i)/float64(n) - math.Pi/2
			pts[i] = [2]float64{math.Cos(a), math.Sin(a)}
		}
	default:
		golden := math.Pi * (3 - math.Sqrt(5))
		for i := range pts {
			r, a := math.Sqrt(float64(i)+0.5), float64(i)*golden
			pts[i] = [2]float64{r * math.Cos(a), r * math.Sin(a)}
		}
	}
	// scale so the closest two biotes just touch and the outer ones stay inside
	near, far := math.Inf(1), 0.0
	for i := range pts {
		far = math.Max(far, math.Hypot(pts[i][0], pts[i][1]))
		for j := i + 1; j < n; j++ {
			near = math.Min(near, math.Hypot(pts[i][0]-pts[j][0], pts[i][1]-pts[j][1]))
		}
	}
	R := math.Min(area.w, area.h) / 2
	cell := math.Min(R*2, maxCell)
	if n > 1 {
		cell = math.Min(maxCell, near*R/(far+near/2))
	}
	s := 0.0
	if n > 1 {
		s = cell / near
	}
	for i := range pts {
		pts[i] = [2]float64{area.cx() + pts[i][0]*s, area.cy() + pts[i][1]*s}
	}
	return pts, cell
}

// spriteScale is how much a biote sprite is scaled up to fill a cell of size s: whole
// pixels where it can, so the pixel art stays crisp.
func spriteScale(s float64) float64 {
	scale := s * 0.86 / blobPx
	if scale >= 2 {
		scale = math.Round(scale)
		if scale*blobPx > s*0.98 {
			scale--
		}
	}
	return scale
}

// blobPos is the centre and cell size of biote i, including its pop-in animation.
func (g *Game) blobPos(L layout, i int) (x, y, s float64) {
	slot := func(i int) (float64, float64) {
		if i < len(L.circle) {
			return L.circle[i][0], L.circle[i][1]
		}
		cols := max(1, L.cols)
		c, r := i%cols, i/cols
		ox := L.grid.x + (L.grid.w-float64(cols)*L.cell)/2
		oy := L.grid.y + (L.grid.h-float64(L.grows)*L.cell)/2
		return ox + (float64(c)+0.5)*L.cell, oy + (float64(r)+0.5)*L.cell
	}
	x, y = slot(i)
	if i < len(g.blobs) {
		b := g.blobs[i]
		if a := g.t - b.mt; a < 0.3 {
			k := 1 - math.Pow(1-a/0.3, 3)
			x, y = b.fx+(x-b.fx)*k, b.fy+(y-b.fy)*k
		}
		y += math.Sin(g.t*2.2+b.phase) * L.cell * 0.025
	}
	return x, y, g.cellNow(L)
}

// cellNow is the biote size, easing from its old size for a moment after a split.
func (g *Game) cellNow(L layout) float64 {
	if a := g.t - g.cellT; a < 0.3 && g.cellA > 0 {
		k := 1 - math.Pow(1-a/0.3, 3)
		return g.cellA + (L.cell-g.cellA)*k
	}
	return L.cell
}

func (g *Game) Draw(screen *ebiten.Image) {
	// On Android the first frame can come before Layout has told us the screen size
	if b := screen.Bounds(); g.W != float64(b.Dx()) || g.H != float64(b.Dy()) {
		g.W, g.H = float64(b.Dx()), float64(b.Dy())
		g.u = g.H / 360
	}
	if g.W < 1 || g.H < 1 {
		return
	}
	L := g.lay()
	g.drawSky(screen)
	g.drawBoard(screen, L)
	g.drawParticles(screen, false)
	g.drawWeather(screen)
	g.drawHUD(screen, L)
	g.drawParticles(screen, true)
	for _, f := range g.floats {
		a := math.Min(1, 1.6*(1-f.t/1.3))
		c := f.c
		sz := f.size
		if sz == 0 {
			sz = 20
		}
		// big ones pop in, then all drift up and fade
		sz *= 1 + 0.4*math.Max(0, 1-f.t/0.15)
		say(screen, f.s, face(fBig, sz*g.u), f.x, f.y-f.t*34*g.u, alCenter, color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}, math.Round(sz/10*g.u*a))
	}
	g.drawBanner(screen)
	if g.panel != "" {
		g.drawPanel(screen, L)
	}
	g.hookDraw(screen)
}

// mutColor is a mutation's label colour (Rainbow cycles).
func (g *Game) mutColor(r int) color.RGBA {
	if rarities[r].Name == "Rainbow" {
		return hsv(g.t*0.5, 0.6, 1)
	}
	return rgb(rarities[r].Label)
}

func (g *Game) drawBlob(dst *ebiten.Image, mask int, x, y, sx, sy float64, blink int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-blobPx/2, -blobPx/2)
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	dst.DrawImage(blobImgs[maskTop(mask)][blink], op)
}

// pips are small squares under a biote, one for each mutation besides the top one.
func (g *Game) drawPips(dst *ebiten.Image, mask int, x, y, scale float64) {
	list := maskList(mask)
	if len(list) < 2 {
		return
	}
	list = list[:len(list)-1]
	ps := math.Max(2, math.Round(scale*2.2))
	w := float64(len(list))*(ps+math.Round(scale)) - math.Round(scale)
	px := math.Round(x - w/2)
	for _, r := range list {
		fill(dst, px-1, y-1, ps+2, ps+2, colO)
		fill(dst, px, y, ps, ps, g.mutColor(r))
		px += ps + math.Round(scale)
	}
}

func (g *Game) drawBoard(dst *ebiten.Image, L layout) {
	for i, b := range g.blobs {
		x, y, s := g.blobPos(L, i)
		size := s * 0.86
		scale := spriteScale(s)
		if g.t-g.cellT < 0.3 {
			scale = s * 0.86 / blobPx // smooth while growing or shrinking
		}
		sx, sy := scale, scale
		if a := g.t - b.born; a < 0.3 {
			k := a / 0.3
			e := 1 + 2.7*math.Pow(k-1, 3) + 1.7*math.Pow(k-1, 2) // ease out back
			sx, sy = scale*(0.3+0.7*e), scale*(0.3+0.7*e)
		}
		if a := g.t - b.squash; a < 0.3 {
			w := 0.18 * (1 - a/0.3) * math.Cos(a*30)
			sx, sy = sx*(1+w), sy*(1-w)
		}
		top := maskTop(b.m)
		if top > 0 {
			c := g.mutColor(top)
			pulse := 0.5 + 0.5*math.Sin(g.t*3+b.phase)
			glow := 35 + 30*pulse + 15*float64(maskCount(b.m)-1)
			vector.DrawFilledCircle(dst, float32(x), float32(y), float32(size*(0.55+0.08*pulse)), color.NRGBA{c.R, c.G, c.B, uint8(math.Min(150, glow))}, true)
		}
		blink := 0
		if math.Mod(g.t+b.phase*5, 4.3) < 0.13 {
			blink = 1
		}
		g.drawBlob(dst, b.m, x, y, sx, sy, blink)
		if top > 0 {
			// the chance of all its mutations together, above it
			say(dst, "1/"+fmtNum(comboOdds(b.m)), face(fSmall, math.Max(7, math.Min(10, s/g.u*0.22))*g.u), x, y-blobPx*scale/2-4*g.u, alCenter, g.mutColor(top), math.Round(g.u))
			g.drawPips(dst, b.m, x, y+blobPx*scale/2-scale, math.Max(1, scale/2))
		}
	}
}

func (g *Game) drawParticles(dst *ebiten.Image, blobs bool) {
	for _, p := range g.parts {
		if (p.r >= 0) != blobs {
			continue
		}
		if p.age < 0 {
			// a sold biote waiting its turn: it squashes down, ready to jump
			if p.r >= 0 {
				w := 0.25 * math.Max(0, 1+p.age/0.15) // squashes over the last 0.15s
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-blobPx/2, -blobPx)
				op.GeoM.Scale(p.s0*(1+w*0.5), p.s0*(1-w*0.6))
				op.GeoM.Translate(p.x, p.y+blobPx/2*p.s0)
				dst.DrawImage(blobImgs[p.r][0], op)
			}
			continue
		}
		if p.r >= 0 {
			k := p.age / p.life
			e := k * k
			x := p.x + (p.tx-p.x)*e
			y := p.y + (p.ty-p.y)*e - math.Sin(k*math.Pi)*40*g.u
			end := g.u * 1.6
			s := math.Max(1, p.s0+(end-p.s0)*math.Sqrt(k))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-blobPx/2, -blobPx/2)
			op.GeoM.Scale(s, s)
			op.GeoM.Rotate(k * 6)
			op.GeoM.Translate(x, y)
			dst.DrawImage(blobImgs[p.r][0], op)
			continue
		}
		if p.r == -2 {
			k := p.age / p.life
			vector.StrokeCircle(dst, float32(p.x), float32(p.y), float32(p.size*(0.6+1.2*k)), float32(math.Max(1, 3*g.u*(1-k))), color.NRGBA{p.c.R, p.c.G, p.c.B, uint8(220 * (1 - k))}, true)
			continue
		}
		a := 1 - p.age/p.life
		fill(dst, math.Round(p.x), math.Round(p.y), p.size, p.size, color.NRGBA{p.c.R, p.c.G, p.c.B, uint8(255 * a)})
	}
}

// drawCardFace draws an ability card in r: its rarity colour, its mutation's sprite
// and, if there's room, its name.
func (g *Game) drawCardFace(dst *ebiten.Image, r rect, c *cardDef, p float64, named bool) {
	sk := cardRarities[c.rar].sk
	bevel(dst, r, sk, p)
	inner := rect{r.x + 2*p, r.y + 2*p, r.w - 4*p, r.h * 0.55}
	fill(dst, inner.x, inner.y, inner.w, inner.h, mul(sk.b, 0.55))
	s := math.Max(1, math.Floor(math.Min(inner.w, inner.h)*0.9/blobPx))
	g.drawBlob(dst, 1<<c.mut&^1, inner.cx(), inner.cy(), s, s, 0)
	if named {
		say(dst, c.name, face(fBig, 12*g.u), r.cx(), r.y+r.h*0.72, alCenter, colW, math.Round(1.5*g.u))
	} else {
		say(dst, string([]rune(c.name)[:1]), face(fBig, 12*g.u), r.cx(), r.y+r.h*0.78, alCenter, colW, math.Round(1.5*g.u))
	}
}

func (g *Game) drawHUD(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	// money, DNA and points
	bump := 1 + 0.2*math.Max(0, 1-(g.t-g.soldT)/0.2)
	say(dst, "$", face(fBig, 34*u*bump), L.money.x+2*u, L.money.cy(), alLeft, colG, math.Round(2*u))
	mc := colW
	if bump > 1 {
		mc = mix(colW, colG, (bump-1)*4)
	}
	say(dst, fmtNum(g.shown), face(fBig, 32*u*bump), L.money.x+24*u*bump, L.money.cy(), alLeft, mc, math.Round(2*u))
	icon(dst, "gem", L.gems.x+10*u, L.gems.cy(), 14*u)
	say(dst, itoa(g.st.Gems), face(fBig, 20*u), L.gems.x+22*u, L.gems.cy(), alLeft, rgb(0x9ff0ff), math.Round(1.5*u))
	if g.st.Points > 0 {
		icon(dst, "star", L.gems.x+56*u, L.gems.cy(), 13*u)
		say(dst, itoa(g.st.Points), face(fBig, 20*u), L.gems.x+66*u, L.gems.cy(), alLeft, colT, math.Round(1.5*u))
	}

	// left menu
	canRebirth := g.st.Money >= rebirthReq(g.st.Rebirths)
	for _, b := range []struct {
		r         rect
		ic, label string
	}{{L.shop, "tree", "SKILLS"}, {L.index, "book", "CATALOG"}, {L.rebirth, "rebirth", "REBIRTH"}} {
		sk := skViolet
		if b.ic == "rebirth" && canRebirth {
			sk = skGold
		}
		bevel(dst, b.r, sk, p)
		if b.ic == "rebirth" {
			// with the current rebirth multiplier next to the icon
			icon(dst, b.ic, b.r.cx()-13*u, b.r.y+22*u, 24*u)
			say(dst, "x"+fmtNum(rebirthMult(g.st.Rebirths)), face(fBig, 16*u), b.r.cx()+1*u, b.r.y+22*u, alLeft, rgb(0x8ef06a), math.Round(1.5*u))
		} else {
			icon(dst, b.ic, b.r.cx(), b.r.y+22*u, 24*u)
		}
		say(dst, b.label, face(fSmall, 8*u), b.r.cx(), b.r.y+b.r.h-11*u, alCenter, colW, math.Round(u))
	}
	if canRebirth {
		badge(dst, L.rebirth.x+L.rebirth.w-6*u, L.rebirth.y+4*u, u, g.t)
	}
	if g.st.Points > 0 && g.cheapestNode() <= g.st.Points {
		badge(dst, L.shop.x+L.shop.w-6*u, L.shop.y+4*u, u, g.t)
	}
	g.drawWorldChip(dst, L.world, p)

	// ability cards
	for i, r := range L.hand {
		if i < len(g.st.Cards) {
			if c := cardByID(g.st.Cards[i]); c != nil {
				g.drawCardFace(dst, r, c, math.Max(1, p-1), false)
				continue
			}
		}
		fill(dst, r.x, r.y, r.w, r.h, color.NRGBA{10, 5, 25, 90})
		vector.StrokeRect(dst, float32(r.x+u), float32(r.y+u), float32(r.w-2*u), float32(r.h-2*u), float32(math.Max(1, u)), color.NRGBA{200, 180, 255, 70}, false)
	}

	// top right: buy amount, music and sound
	bevel(dst, L.buy, skGreen, p)
	say(dst, "BUY "+buyModes[g.st.BuyMode], face(fBig, 14*u), L.buy.cx(), L.buy.cy()-u, alCenter, colW, math.Round(1.5*u))
	bevel(dst, L.music, skDark, p)
	ic := "note"
	if g.st.NoMusic {
		ic = "nonote"
	}
	icon(dst, ic, L.music.cx(), L.music.cy(), 14*u)
	bevel(dst, L.sound, skDark, p)
	ic = "sound"
	if !g.st.Sound {
		ic = "mute"
	}
	icon(dst, ic, L.sound.cx(), L.sound.cy(), 14*u)

	// upgrade cards
	for i, r := range L.cards[:nUps] {
		title, from, to, cost, maxed := g.cardInfo(i)
		afford := !maxed && g.st.Money >= cost
		sk := skDark
		if afford {
			sk = skOrange
		}
		bevel(dst, r, sk, p)
		say(dst, title, face(fSmall, 8*u), r.x+9*u, r.y+15*u, alLeft, colW, math.Round(u))
		if maxed {
			say(dst, "MAXED", face(fSmall, 8*u), r.x+r.w-9*u, r.y+15*u, alRight, colT, math.Round(u))
			say(dst, from, face(fSmall, 9*u), r.x+9*u, r.y+40*u, alLeft, colW, math.Round(u))
		} else {
			cc := colG
			if !afford {
				cc = rgb(0xff8a8a)
			}
			say(dst, "$"+fmtNum(cost), face(fSmall, 8*u), r.x+r.w-9*u, r.y+15*u, alRight, cc, math.Round(u))
			say(dst, from+" > "+to, face(fSmall, 9*u), r.x+9*u, r.y+40*u, alLeft, colW, math.Round(u))
		}
	}
	g.drawRoundCard(dst, L.round, p)

	// split, auto and sell
	n, c := len(g.blobs), g.capacity()
	full := n >= c
	bevel(dst, L.split, skViolet, p)
	if !full {
		k := math.Min(1, g.cool/g.cooldown())
		in := rect{L.split.x + 2*p, L.split.y + L.split.h - 2*p - 5*u, (L.split.w - 4*p) * k, 4 * u}
		fill(dst, in.x, in.y, in.w, in.h, colL)
		if k >= 1 {
			fill(dst, L.split.x+2*p, L.split.y+2*p, L.split.w-4*p, L.split.h-4*p, color.NRGBA{255, 255, 255, uint8(25 + 20*math.Sin(g.t*6))})
		}
	}
	lbl, sub, subc := "SPLIT", "READY!", colT
	if full {
		lbl, sub, subc = "FULL", "SELL TO SPLIT", rgb(0xd8ccff)
	} else if left := g.cooldown() - g.cool; left > 0 {
		sub, subc = fmt.Sprintf("%.1fs", left), colW
	}
	say(dst, lbl, face(fBig, 26*u), L.split.cx(), L.split.cy()-6*u, alCenter, colW, math.Round(2*u))
	say(dst, sub, face(fSmall, 8*u), L.split.cx(), L.split.y+L.split.h-15*u, alCenter, subc, math.Round(u))
	say(dst, fmt.Sprintf("x%d/%d", n, c), face(fBig, 15*u), L.split.x+L.split.w-2*u, L.split.y-8*u, alRight, colW, math.Round(1.5*u))
	g.drawToggle(dst, L.auto, "AUTO", g.has("auto"), g.st.Auto, p)

	sk := skOrange
	if n < 2 {
		sk = skGrey
	}
	bevel(dst, L.sell, sk, p)
	say(dst, "SELL", face(fBig, 26*u), L.sell.cx(), L.sell.cy()-6*u, alCenter, colW, math.Round(2*u))
	say(dst, "+$"+fmtNum(g.boardValue()), face(fSmall, 8*u), L.sell.cx(), L.sell.y+L.sell.h-14*u, alCenter, colG, math.Round(u))
	g.drawToggle(dst, L.autosell, "AUTO", g.has("autosell"), g.st.AutoSell, p)
}

// drawToggle is a small AUTO switch: locked until its skill is unlocked.
func (g *Game) drawToggle(dst *ebiten.Image, r rect, label string, owned, on bool, p float64) {
	u := g.u
	sk := skGrey
	if owned && on {
		sk = skGreen
	}
	bevel(dst, r, sk, p)
	if !owned {
		icon(dst, "lock", r.x+9*u, r.cy(), 9*u)
		say(dst, label, face(fSmall, 7*u), r.x+28*u, r.cy(), alCenter, colW, math.Round(u))
	} else {
		say(dst, label, face(fSmall, 8*u), r.cx(), r.cy(), alCenter, colW, math.Round(u))
	}
}

// drawRoundCard shows the current round: its name, the target, sells left.
func (g *Game) drawRoundCard(dst *ebiten.Image, r rect, p float64) {
	u, s := g.u, &g.st
	sk := skin{rgb(0x3a2a6a), rgb(0x6a4fc8), rgb(0x1f1440)}
	if s.Round == 2 {
		sk = skin{rgb(0x6a1f3a), rgb(0xc84f6a), rgb(0x3a1020)}
	}
	bevel(dst, r, sk, p)
	title := roundNames[s.Round]
	say(dst, title, face(fSmall, 8*u), r.x+9*u, r.y+13*u, alLeft, colT, math.Round(u))
	say(dst, "ANTE "+itoa(s.Ante), face(fSmall, 8*u), r.x+r.w-9*u, r.y+13*u, alRight, colW, math.Round(u))
	target := roundTarget(s.Ante, s.Round)
	say(dst, "$"+fmtNum(s.RoundEarned)+" / $"+fmtNum(target), face(fSmall, 8*u), r.x+9*u, r.y+30*u, alLeft, colW, math.Round(u))
	bar := rect{r.x + 9*u, r.y + 40*u, r.w - 18*u, 6 * u}
	fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
	fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*math.Min(1, s.RoundEarned/target), bar.h-2*u, colG)
	left := g.sellsAllowed() - s.RoundSells
	sub := fmt.Sprintf("%d SELLS LEFT", left)
	if b := g.boss(); b != nil {
		sub = b.name + " - " + sub
	}
	say(dst, sub, face(fSmall, 6*u), r.x+9*u, r.y+53*u, alLeft, rgb(0xd8ccff), math.Round(u))
}

// drawWorldChip shows day or night, and the current event.
func (g *Game) drawWorldChip(dst *ebiten.Image, r rect, p float64) {
	u := g.u
	bevel(dst, r, skDark, p)
	night := g.night()
	// time until the sky turns
	ph := math.Mod(g.st.Clock, dayLen) / dayLen
	left := 0.0
	if night {
		left = (1 - ph) * dayLen
	} else {
		left = (0.5 - ph) * dayLen
	}
	name, c := "DAY", colT
	if night {
		name, c = "NIGHT", rgb(0xb9a8ff)
	}
	g.drawSunMoon(dst, r.x+13*u, r.y+15*u, 7*u, night)
	say(dst, name, face(fSmall, 8*u), r.x+25*u, r.y+11*u, alLeft, c, math.Round(u))
	say(dst, fmtTime(left), face(fSmall, 7*u), r.x+25*u, r.y+22*u, alLeft, colW, math.Round(u))
	if e := eventByID(g.st.Event); e != nil {
		flash := 0.75 + 0.25*math.Sin(g.t*5)
		say(dst, e.name, face(fSmall, 7*u), r.cx(), r.y+42*u, alCenter, mix(colW, colT, flash), math.Round(u))
		short, _, _ := strings.Cut(e.desc, ",") // the box is narrow: the main effect only
		say(dst, short, face(fSmall, 6*u), r.cx(), r.y+55*u, alCenter, rgb(0xd8ccff), math.Round(u))
		say(dst, fmtTime(g.st.EventLeft), face(fSmall, 6*u), r.cx(), r.y+68*u, alCenter, colW, math.Round(u))
	} else {
		say(dst, "NEXT EVENT", face(fSmall, 6*u), r.cx(), r.y+46*u, alCenter, rgb(0xd8ccff), math.Round(u))
		next := g.st.NextEvent
		if g.has("magnet") {
			next /= 2
		}
		say(dst, fmtTime(next), face(fSmall, 7*u), r.cx(), r.y+60*u, alCenter, colW, math.Round(u))
	}
}

func (g *Game) cheapestNode() int {
	m := math.MaxInt
	for i := range tree {
		if g.canUnlock(&tree[i]) {
			m = min(m, tree[i].cost)
		}
	}
	return m
}

// badge is a bouncing "!" that marks something new.
func badge(dst *ebiten.Image, x, y, u, t float64) {
	y += math.Sin(t*6) * 2 * u
	r := rect{x - 7*u, y - 7*u, 14 * u, 14 * u}
	bevel(dst, r, skGold, math.Max(1, math.Round(u)))
	say(dst, "!", face(fBig, 12*u), r.cx(), r.cy()-u, alCenter, colO, 0)
}

func (g *Game) drawBanner(dst *ebiten.Image) {
	u := g.u
	show := func(a float64, title, line string, sprite int) {
		k := math.Min(1, a/0.25)
		if a > 2.4 {
			k = 1 - (a-2.4)/0.4
		}
		w, h := 260*u, 54*u
		r := rect{g.W/2 - w/2, -h + (h+8*u)*k, w, h}
		bevel(dst, r, skGold, math.Max(1, math.Round(u*1.3)))
		s := math.Floor(40 * u / blobPx)
		g.drawBlob(dst, 1<<sprite&^1, r.x+10*u+blobPx*s/2, r.cy(), s, s, 0)
		say(dst, title, face(fSmall, 9*u), r.x+60*u, r.y+16*u, alLeft, colO, 0)
		say(dst, line, face(fBig, 17*u), r.x+60*u, r.y+35*u, alLeft, colW, math.Round(1.5*u))
	}
	if a := g.t - g.banner.t; g.banner.r >= 0 && a < 2.8 {
		ri := rarities[g.banner.r]
		line := ri.Name + "  1/" + fmtNum(ri.Odds)
		if g.banner.text != "NEW MUTATION!" {
			line = g.banner.text
			show(a, "BOSS REWARD", line, g.banner.r)
			return
		}
		show(a, g.banner.text, line, g.banner.r)
		return
	}
	if e := eventByID(g.st.Event); e != nil {
		if a := g.t - g.eventT; a < 2.8 {
			top := map[string]int{"meteor": mCosmic, "blizzard": mFrost, "goldrush": mGold, "aurora": mRainbow, "bloom": mHeart, "heatwave": mToxic, "eclipse": mShadow}[e.id]
			show(a, "EVENT!", e.name+": "+e.desc, top)
		}
	}
}
