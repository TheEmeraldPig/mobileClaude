package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// layout holds where everything is this frame, in device pixels. Positions are worked
// out in units (the screen is 360 units tall) so phones, tablets and PCs look the same.
type layout struct {
	money, gems, chip, sound rect
	shop, index, rebirth     rect
	cards                    [4]rect
	split, sell, auto        rect
	grid                     rect
	cols, grows              int
	cell                     float64
	panel, close, action     rect
	rows                     []rect // shop rows or index cells
	p                        float64
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
	L.money = R(10, 6, 200, 34)
	L.gems = R(10, 42, 120, 22)
	for i, r := range []*rect{&L.shop, &L.index, &L.rebirth} {
		*r = R(10, 74+float64(i)*66, leftW-6, 60)
	}
	L.sound = R(wu-38, 8, 28, 28)
	L.chip = R(rightX, 8, rightW-34, 28)
	for i := range L.cards {
		L.cards[i] = R(rightX, 44+float64(i)*68, rightW, 62)
	}
	cx0, cx1 := 10+leftW+8, rightX-8
	bw := math.Min(170, (cx1-cx0-12)/2)
	bx := (cx0+cx1)/2 - bw - 6
	L.split = R(bx, 360-72, bw, 60)
	L.sell = R(bx+bw+12, 360-72, bw, 60)
	L.auto = R(bx-4, 360-72-16, 54, 20)
	L.grid = R(cx0, 46, cx1-cx0, 360-72-34-46)

	// fit the board's capacity into the grid area with the biggest cells
	n := g.capacity()
	best := 0.0
	for c := 1; c <= n; c++ {
		r := (n + c - 1) / c
		s := math.Min(L.grid.w/float64(c), L.grid.h/float64(r))
		if s > best {
			best, L.cols, L.grows = s, c, r
		}
	}
	L.cell = math.Min(best, 70*u)

	pw, ph := math.Min(wu-40, 470), 320.0
	L.panel = R(wu/2-pw/2, 20, pw, ph)
	L.close = R(wu/2+pw/2-36, 26, 28, 28)
	px, py := wu/2-pw/2, 20.0
	switch g.panel {
	case "shop":
		for i := range gemUps {
			L.rows = append(L.rows, R(px+12, py+44+float64(i)*53, pw-24, 48))
		}
	case "index":
		cw, ch := (pw-24-3*8)/4, (ph-56-12-8)/2
		for i := range rarities {
			L.rows = append(L.rows, R(px+12+float64(i%4)*(cw+8), py+50+float64(i/4)*(ch+8), cw, ch))
		}
	case "rebirth":
		L.action = R(wu/2-80, py+ph-72, 160, 54)
	}
	return L
}

// blobPos is the centre and cell size of blob i, including its pop-in animation.
func (g *Game) blobPos(L layout, i int) (x, y, s float64) {
	slot := func(i int) (float64, float64) {
		c, r := i%L.cols, i/L.cols
		ox := L.grid.x + (L.grid.w-float64(L.cols)*L.cell)/2
		oy := L.grid.y + (L.grid.h-float64(L.grows)*L.cell)/2
		return ox + (float64(c)+0.5)*L.cell, oy + (float64(r)+0.5)*L.cell
	}
	x, y = slot(i)
	if i < len(g.blobs) {
		b := g.blobs[i]
		if a := g.t - b.born; b.from >= 0 && a < 0.3 {
			fx, fy := slot(b.from)
			k := 1 - math.Pow(1-a/0.3, 3)
			x, y = fx+(x-fx)*k, fy+(y-fy)*k
		}
		y += math.Sin(g.t*2.2+b.phase) * L.cell * 0.025
	}
	return x, y, L.cell
}

func (g *Game) Draw(screen *ebiten.Image) {
	L := g.lay()
	g.drawBackground(screen)
	g.drawBoard(screen, L)
	g.drawParticles(screen, false)
	g.drawHUD(screen, L)
	g.drawParticles(screen, true)
	for _, f := range g.floats {
		a := 1 - f.t/1.1
		c := f.c
		say(screen, f.s, face(fBig, 20*g.u), f.x, f.y-f.t*34*g.u, alCenter, color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}, math.Round(1.5*g.u*a))
	}
	g.drawBanner(screen)
	if g.panel != "" {
		g.drawPanel(screen, L)
	}
	g.hookDraw(screen)
}

// The background is a pixel-art dusk: purple sky to orange, a big moon and hills. It is
// drawn at low resolution once per screen size and scaled up, so it looks pixelated.
func (g *Game) drawBackground(dst *ebiten.Image) {
	q := math.Max(2, math.Round(g.u*2))
	lw, lh := int(math.Ceil(g.W/q)), int(math.Ceil(g.H/q))
	if g.bg == nil || g.bgKey != [2]int{lw, lh} {
		g.bgKey = [2]int{lw, lh}
		img := image.NewRGBA(image.Rect(0, 0, lw, lh))
		top, mid, low := rgb(0x1b1036), rgb(0x4a2160), rgb(0xe0703a)
		mcx, mcy, mr := float64(lw)*0.3, float64(lh)*0.36, float64(lh)*0.24
		for y := 0; y < lh; y++ {
			t := float64(y) / float64(lh)
			var c color.RGBA
			if t < 0.5 {
				c = mix(top, mid, t/0.5)
			} else {
				c = mix(mid, low, math.Min(1, (t-0.5)/0.35))
			}
			// banding, for a pixel-art look
			band := math.Floor(t*14) / 14
			c = mix(c, mix(top, low, band), 0.25)
			for x := 0; x < lw; x++ {
				cc := c
				d := math.Hypot(float64(x)-mcx, float64(y)-mcy)
				if d < mr {
					cc = rgb(0xffcf6a)
					if math.Hypot(float64(x)-mcx+mr*0.35, float64(y)-mcy+mr*0.2) < mr*0.18 ||
						math.Hypot(float64(x)-mcx-mr*0.3, float64(y)-mcy-mr*0.35) < mr*0.12 ||
						math.Hypot(float64(x)-mcx-mr*0.45, float64(y)-mcy+mr*0.3) < mr*0.08 {
						cc = rgb(0xf0a84a)
					}
					if float64(x)-mcx+float64(y)-mcy > mr*0.9 {
						cc = mul(cc, 0.9)
					}
				} else if d < mr*1.25 {
					cc = mix(cc, rgb(0xffcf6a), 0.18*(1-(d-mr)/(mr*0.25)))
				}
				// two layers of hills
				fx := float64(x) / float64(lw)
				h1 := 0.70 + 0.06*math.Sin(fx*7+1) + 0.03*math.Sin(fx*17)
				h2 := 0.80 + 0.05*math.Sin(fx*5+3) + 0.02*math.Sin(fx*23+1)
				if t > h2 {
					cc = mix(rgb(0x2a1238), rgb(0x1a0b24), (t-h2)*3)
				} else if t > h1 {
					cc = rgb(0x4a1f52)
				}
				img.SetRGBA(x, y, cc)
			}
		}
		g.bg = ebiten.NewImageFromImage(img)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(q, q)
	dst.DrawImage(g.bg, op)
	for _, s := range g.stars {
		a := 0.4 + 0.6*math.Abs(math.Sin(g.t*1.3+s[2]))
		sz := math.Max(1, math.Round(g.u*1.5))
		fill(dst, math.Round(s[0]*g.W), math.Round(s[1]*g.H), sz, sz, color.NRGBA{255, 240, 220, uint8(a * 255)})
	}
}

func (g *Game) drawBoard(dst *ebiten.Image, L layout) {
	for i, b := range g.blobs {
		x, y, s := g.blobPos(L, i)
		size := s * 0.86
		scale := size / blobPx
		if scale >= 2 {
			scale = math.Round(scale)
			if scale*blobPx > s*0.98 {
				scale--
			}
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
		if b.r > 0 {
			c := rgb(rarities[b.r].Label)
			if rarities[b.r].Name == "Rainbow" {
				c = hsv(g.t*0.5, 0.6, 1)
			}
			pulse := 0.5 + 0.5*math.Sin(g.t*3+b.phase)
			vector.DrawFilledCircle(dst, float32(x), float32(y), float32(size*(0.55+0.08*pulse)), color.NRGBA{c.R, c.G, c.B, uint8(35 + 30*pulse)}, true)
		}
		blink := 0
		if math.Mod(g.t+b.phase*5, 4.3) < 0.13 {
			blink = 1
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-blobPx/2, -blobPx/2)
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate(math.Round(x), math.Round(y))
		dst.DrawImage(blobImgs[b.r][blink], op)
		if b.r > 0 {
			c := rgb(rarities[b.r].Label)
			if rarities[b.r].Name == "Rainbow" {
				c = hsv(g.t*0.5, 0.6, 1)
			}
			say(dst, "1/"+fmtNum(rarities[b.r].Odds), face(fSmall, math.Max(7, math.Min(10, s/g.u*0.22))*g.u), x, y-blobPx*scale/2-4*g.u, alCenter, c, math.Round(g.u))
		}
	}
}

func (g *Game) drawParticles(dst *ebiten.Image, blobs bool) {
	for _, p := range g.parts {
		if (p.r >= 0) != blobs || p.age < 0 {
			continue
		}
		if p.r >= 0 {
			k := p.age / p.life
			e := k * k
			x := p.x + (p.tx-p.x)*e
			y := p.y + (p.ty-p.y)*e - math.Sin(k*math.Pi)*40*g.u
			s := math.Max(1, (g.u*2.2)*(1-0.6*k))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-blobPx/2, -blobPx/2)
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(x, y)
			dst.DrawImage(blobImgs[p.r][0], op)
			continue
		}
		a := 1 - p.age/p.life
		fill(dst, math.Round(p.x), math.Round(p.y), p.size, p.size, color.NRGBA{p.c.R, p.c.G, p.c.B, uint8(255 * a)})
	}
}

func (g *Game) drawHUD(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	// money and gems
	say(dst, "$", face(fBig, 34*u), L.money.x+2*u, L.money.cy(), alLeft, colG, math.Round(2*u))
	say(dst, fmtNum(g.shown), face(fBig, 32*u), L.money.x+24*u, L.money.cy(), alLeft, colW, math.Round(2*u))
	icon(dst, "gem", L.gems.x+10*u, L.gems.cy(), 14*u)
	say(dst, fmtNum(float64(g.st.Gems)), face(fBig, 20*u), L.gems.x+24*u, L.gems.cy(), alLeft, rgb(0x9ff0ff), math.Round(1.5*u))

	// left menu
	canRebirth := g.st.Money >= rebirthReq(g.st.Rebirths)
	for _, b := range []struct {
		r         rect
		ic, label string
		sk        skin
	}{{L.shop, "bag", "SHOP", skViolet}, {L.index, "book", "INDEX", skViolet}, {L.rebirth, "rebirth", "REBIRTH", skViolet}} {
		sk := b.sk
		if b.ic == "rebirth" && canRebirth {
			sk = skGold
		}
		bevel(dst, b.r, sk, p)
		icon(dst, b.ic, b.r.cx(), b.r.y+24*u, 26*u)
		say(dst, b.label, face(fSmall, 8*u), b.r.cx(), b.r.y+b.r.h-12*u, alCenter, colW, math.Round(u))
	}
	if canRebirth {
		badge(dst, L.rebirth.x+L.rebirth.w-6*u, L.rebirth.y+4*u, u, g.t)
	}
	if g.st.Gems >= cheapestGem(g) {
		badge(dst, L.shop.x+L.shop.w-6*u, L.shop.y+4*u, u, g.t)
	}

	// top right: rebirth multiplier and sound
	bevel(dst, L.chip, skDark, p)
	icon(dst, "rebirth", L.chip.x+14*u, L.chip.cy(), 14*u)
	say(dst, "x"+fmtNum(rebirthMult(g.st.Rebirths))+" REBIRTH", face(fSmall, 8*u), L.chip.x+26*u, L.chip.cy(), alLeft, colT, math.Round(u))
	bevel(dst, L.sound, skDark, p)
	ic := "sound"
	if !g.st.Sound {
		ic = "mute"
	}
	icon(dst, ic, L.sound.cx(), L.sound.cy(), 14*u)

	// upgrade cards
	for i, r := range L.cards {
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
	lbl := "SPLIT"
	if full {
		lbl = "FULL"
	}
	say(dst, lbl, face(fBig, 26*u), L.split.cx(), L.split.cy()-3*u, alCenter, colW, math.Round(2*u))
	say(dst, fmt.Sprintf("x%d/%d", n, c), face(fBig, 18*u), L.split.cx()+14*u, L.split.y-6*u, alCenter, colW, math.Round(1.5*u))

	ask := skGrey
	if g.gem("auto") > 0 && g.st.Auto {
		ask = skGreen
	}
	bevel(dst, L.auto, ask, p)
	if g.gem("auto") == 0 {
		icon(dst, "lock", L.auto.x+11*u, L.auto.cy(), 9*u)
		say(dst, "AUTO", face(fSmall, 7*u), L.auto.x+32*u, L.auto.cy(), alCenter, colW, math.Round(u))
	} else {
		say(dst, "AUTO", face(fSmall, 8*u), L.auto.cx(), L.auto.cy(), alCenter, colW, math.Round(u))
	}

	sk := skOrange
	if n < 2 {
		sk = skGrey
	}
	bevel(dst, L.sell, sk, p)
	say(dst, "SELL", face(fBig, 26*u), L.sell.cx(), L.sell.cy()-6*u, alCenter, colW, math.Round(2*u))
	say(dst, "+$"+fmtNum(g.boardValue()), face(fSmall, 8*u), L.sell.cx(), L.sell.y+L.sell.h-14*u, alCenter, colG, math.Round(u))
}

func cheapestGem(g *Game) int {
	m := math.MaxInt
	for _, u := range gemUps {
		if l := g.gem(u.id); l < u.max {
			m = min(m, gemCost(u, l))
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
	a := g.t - g.banner.t
	if g.banner.r < 0 || a > 2.8 {
		return
	}
	u := g.u
	k := math.Min(1, a/0.25)
	if a > 2.4 {
		k = 1 - (a-2.4)/0.4
	}
	w, h := 240*u, 54*u
	r := rect{g.W/2 - w/2, -h + (h+8*u)*k, w, h}
	bevel(dst, r, skGold, math.Max(1, math.Round(u*1.3)))
	ri := rarities[g.banner.r]
	op := &ebiten.DrawImageOptions{}
	s := math.Floor(40 * u / blobPx)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(r.x+10*u, r.cy()-blobPx*s/2)
	dst.DrawImage(blobImgs[g.banner.r][0], op)
	say(dst, "NEW BLOB!", face(fSmall, 9*u), r.x+60*u, r.y+16*u, alLeft, colO, 0)
	say(dst, ri.Name+"  1/"+fmtNum(ri.Odds), face(fBig, 18*u), r.x+60*u, r.y+35*u, alLeft, colW, math.Round(1.5*u))
}

func (g *Game) drawPanel(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	fill(dst, 0, 0, g.W, g.H, color.NRGBA{10, 5, 25, 150})
	bevel(dst, L.panel, skDark, p)
	title := map[string]string{"shop": "GEM SHOP", "index": "BLOB INDEX", "rebirth": "REBIRTH"}[g.panel]
	say(dst, title, face(fBig, 22*u), L.panel.cx(), L.panel.y+22*u, alCenter, colT, math.Round(2*u))
	bevel(dst, L.close, skOrange, p)
	say(dst, "X", face(fBig, 16*u), L.close.cx(), L.close.cy()-u, alCenter, colW, math.Round(1.5*u))

	switch g.panel {
	case "shop":
		icon(dst, "gem", L.panel.x+20*u, L.panel.y+22*u, 12*u)
		say(dst, fmtNum(float64(g.st.Gems)), face(fBig, 16*u), L.panel.x+32*u, L.panel.y+22*u, alLeft, rgb(0x9ff0ff), math.Round(1.5*u))
		for i, r := range L.rows {
			gu := gemUps[i]
			lv := g.gem(gu.id)
			bevel(dst, r, skViolet, p)
			say(dst, gu.name, face(fBig, 15*u), r.x+10*u, r.y+16*u, alLeft, colW, math.Round(1.5*u))
			desc := gu.desc
			if gu.max > 1 {
				desc += fmt.Sprintf("   LV %d/%d", lv, gu.max)
			}
			say(dst, desc, face(fSmall, 7*u), r.x+10*u, r.y+34*u, alLeft, rgb(0xd8ccff), math.Round(u))
			br := rect{r.x + r.w - 92*u, r.y + 7*u, 84 * u, r.h - 14*u}
			switch {
			case lv >= gu.max && gu.max == 1:
				on := (gu.id == "auto" && g.st.Auto) || (gu.id == "autosell" && g.st.AutoSell)
				sk, t := skGrey, "OFF"
				if on {
					sk, t = skGreen, "ON"
				}
				bevel(dst, br, sk, p)
				say(dst, t, face(fSmall, 9*u), br.cx(), br.cy(), alCenter, colW, math.Round(u))
			case lv >= gu.max:
				bevel(dst, br, skDark, p)
				say(dst, "MAXED", face(fSmall, 8*u), br.cx(), br.cy(), alCenter, colT, math.Round(u))
			default:
				c := gemCost(gu, lv)
				sk := skGrey
				if g.st.Gems >= c {
					sk = skGreen
				}
				bevel(dst, br, sk, p)
				icon(dst, "gem", br.x+22*u, br.cy(), 11*u)
				say(dst, fmt.Sprint(c), face(fBig, 15*u), br.x+34*u, br.cy()-u, alLeft, colW, math.Round(1.5*u))
			}
		}
	case "index":
		for i, r := range L.rows {
			ri := rarities[i]
			found := g.st.Found[i] > 0 || i == 0
			bevel(dst, r, skViolet, p)
			s := math.Floor(math.Min(r.w*0.5, r.h*0.45) / blobPx)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(r.cx()-blobPx*s/2, r.y+6*u)
			if !found {
				op.ColorScale.Scale(0, 0, 0, 0.6)
			}
			dst.DrawImage(blobImgs[i][0], op)
			name := ri.Name
			if !found {
				name = "???"
			}
			ty := r.y + 10*u + blobPx*s
			say(dst, name, face(fBig, 13*u), r.cx(), ty, alCenter, colW, math.Round(1.5*u))
			odds := "COMMON"
			if i > 0 {
				odds = "1/" + fmtNum(ri.Odds)
			}
			say(dst, odds+"  x"+fmtNum(ri.Mult), face(fSmall, 7*u), r.cx(), ty+14*u, alCenter, rgb(ri.Label), math.Round(u))
			if found && i > 0 {
				say(dst, "MADE "+fmtNum(float64(g.st.Found[i])), face(fSmall, 6*u), r.cx(), ty+25*u, alCenter, rgb(0xd8ccff), math.Round(u))
			}
		}
	case "rebirth":
		cx, y := L.panel.cx(), L.panel.y+62*u
		r := g.st.Rebirths
		say(dst, "x"+fmtNum(rebirthMult(r))+"  >  x"+fmtNum(rebirthMult(r+1)), face(fBig, 30*u), cx, y+10*u, alCenter, colT, math.Round(2*u))
		say(dst, "BLOB VALUE MULTIPLIER", face(fSmall, 8*u), cx, y+36*u, alCenter, colW, math.Round(u))
		icon(dst, "gem", cx-50*u, y+66*u, 14*u)
		say(dst, fmt.Sprintf("+%d GEMS", rebirthGems(r)), face(fBig, 18*u), cx-36*u, y+66*u, alLeft, rgb(0x9ff0ff), math.Round(1.5*u))
		say(dst, "NEEDS $"+fmtNum(rebirthReq(r)), face(fBig, 16*u), cx, y+102*u, alCenter, colG, math.Round(1.5*u))
		say(dst, "RESETS YOUR MONEY, BLOBS AND UPGRADES", face(fSmall, 7*u), cx, y+126*u, alCenter, rgb(0xd8ccff), math.Round(u))
		ok := g.st.Money >= rebirthReq(r)
		sk := skGrey
		if ok {
			sk = skGold
		}
		bevel(dst, L.action, sk, p)
		say(dst, "REBIRTH", face(fBig, 22*u), L.action.cx(), L.action.cy()-2*u, alCenter, colW, math.Round(2*u))
		if !ok {
			k := math.Min(1, g.st.Money/rebirthReq(r))
			bar := rect{L.action.x, L.action.y - 14*u, L.action.w, 8 * u}
			fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
			fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
		}
	}
}
