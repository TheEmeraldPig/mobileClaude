package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Painted card illustrations, built from lit parts like the characters.
// Coordinates are for a 120-wide card (the art window is about 104x72).

var (
	mScale   = mat(rgb(0x3a, 0x7a, 0x48), 0.5, 16, 0.7, 0.9)
	mToxic   = mat(rgb(0x6a, 0xc0, 0x40), 0.3, 10, 0.6, 1.0)
	mVenom   = mat(rgb(0x8a, 0x4a, 0xd0), 0.3, 10, 0.6, 1.0)
	mBlood   = mat(rgb(0xb0, 0x10, 0x20), 1.0, 30, 0.1, 1.0)
	mPage    = mat(rgb(0xd8, 0xc8, 0xa0), 0.05, 4, 0.5, 0.3)
	mFeather = mat(rgb(0xe8, 0xe4, 0xd8), 0.2, 8, 0.5, 0.5)
	mGlass   = mat(rgb(0x9a, 0xc8, 0xe8), 1.2, 40, 0.05, 0.6)
	mEmber   = mat(rgb(0xff, 0x8a, 0x30), 0.8, 12, 0.3, 1.0)

	pDagger = sharpPart(mBlade, -3, 0, -3, -34, 0, -42, 3, -34, 3, 0)
	pKite   = newPart(mSteel, -26, -26, 0, -32, 26, -26, 24, 4, 0, 32, -24, 4).with(func(d *ebiten.Image, ox, oy float64) {
		fillPath(d, smoothPath(shift(ox, oy, -19, -21, 0, -26, 19, -21, 17, 2, 0, 25, -17, 2)), rgb(0x6a, 0x10, 0x1a))
		fillPath(d, pathOf(shift(ox, oy, -3, -22, 3, -22, 3, 20, -3, 20)), colGold)
		fillPath(d, pathOf(shift(ox, oy, -16, -6, 16, -6, 16, 0, -16, 0)), colGold)
	})
	pBuckler = newPart(mWood, -24, 0, -17, -17, 0, -24, 17, -17, 24, 0, 17, 17, 0, 24, -17, 17).with(func(d *ebiten.Image, ox, oy float64) {
		strokeCircle(d, ox, oy, 21, 3, rgb(0x7a, 0x80, 0x8a))
		disc(d, ox, oy, 7, 7, rgb(0x8a, 0x90, 0x9a))
	})
	pHammerHead = newPart(mSteel, -18, -9, 18, -9, 20, 0, 18, 9, -18, 9, -20, 0)
	pHandle     = sharpPart(mWood, -2.5, 0, 2.5, 0, 2.5, 50, -2.5, 50)
	pAxeBlade   = newPart(mSteel, 0, -14, 16, -20, 30, -10, 32, 10, 18, 18, 0, 10)
	pTower      = newPart(mSteel, -20, -30, 20, -30, 22, 26, 0, 34, -22, 26).with(func(d *ebiten.Image, ox, oy float64) {
		for _, p := range [][2]float64{{-14, -24}, {14, -24}, {-14, 18}, {14, 18}} {
			disc(d, ox+p[0], oy+p[1], 2.4, 2.4, colGoldHi)
		}
		strokePath(d, smoothPath(shift(ox, oy, -16, -26, 16, -26, 18, 23, 0, 30, -18, 23)), 1.4, alpha(colGold, 0.8))
	})
	pHorn = newPart(mHorn, -28, 4, -10, 18, 14, 6, 26, -16, 30, -24, 22, -14, 10, 0, -10, 6, -26, -6).with(func(d *ebiten.Image, ox, oy float64) {
		for _, x := range []float64{-14, 4, 18} {
			detailLine(d, ox, oy, 3, colGold, x, -4+math.Abs(x)*0.2, x+2, 10+x*0.1)
		}
	})
	pCover = newPart(mLeather.col(rgb(0x4a, 0x1a, 0x3a)), -32, -20, 0, -16, 32, -20, 32, 22, 0, 26, -32, 22)
	pPages = newPart(mPage, -29, -18, 0, -13, 29, -18, 29, 18, 0, 22, -29, 18).with(func(d *ebiten.Image, ox, oy float64) {
		detailLine(d, ox, oy, 1.2, alpha(colInk, 0.5), 0, -13, 0, 22)
		for i := 0; i < 4; i++ {
			yy := -9 + float64(i)*7
			detailLine(d, ox, oy, 1.2, alpha(rgb(0x5a, 0x3a, 0x7a), 0.8), -24, yy-2, -5, yy)
			detailLine(d, ox, oy, 1.2, alpha(rgb(0x5a, 0x3a, 0x7a), 0.8), 5, yy, 24, yy-2)
		}
	})
	pGauntlet = newPart(mSteel, -16, 10, -18, -8, -12, -20, 12, -22, 18, -10, 16, 10, 0, 16).with(func(d *ebiten.Image, ox, oy float64) {
		for i := 0; i < 4; i++ {
			detailLine(d, ox, oy, 1.4, alpha(colInk, 0.6), -11+float64(i)*7, -20, -11+float64(i)*7, -8)
		}
	})
	pBlob  = newPart(mToxic, -12, 0, -8, -10, 2, -12, 11, -6, 12, 4, 4, 11, -6, 10)
	pBlobV = newPart(mVenom, -12, 0, -8, -10, 2, -12, 11, -6, 12, 4, 4, 11, -6, 10)
	pSkull = newPart(mBone, -9, 2, -10, -7, -6, -13, 6, -13, 10, -7, 9, 2, 5, 7, -5, 7).with(func(d *ebiten.Image, ox, oy float64) {
		disc(d, ox-4, oy-4, 3, 3.5, colInk)
		disc(d, ox+4, oy-4, 3, 3.5, colInk)
	})
	pFang = newPart(mBone, -5, 0, 5, 0, 3, 14, 0, 24, -3, 14)
	pDrop = newPart(mBlood, 0, -18, 7, -6, 11, 6, 6, 14, -6, 14, -11, 6, -7, -6)
	pRock = newPart(mStone.col(rgb(0x5a, 0x4a, 0x42)), -16, -4, -10, -14, 4, -16, 15, -8, 17, 6, 8, 15, -8, 14, -16, 6).with(func(d *ebiten.Image, ox, oy float64) {
		for _, c := range [][4]float64{{-8, -6, 0, 2}, {4, -10, 8, 4}, {-4, 6, 6, 10}} {
			detailLine(d, ox, oy, 2, rgb(0xff, 0x8a, 0x30), c[0], c[1], c[2], c[3])
		}
	})
	pGoldBlade = sharpPart(mGold.col(rgb(0xf0, 0xd0, 0x80)), -6.5, 0, -6.5, -60, 0, -74, 6.5, -60, 6.5, 0).with(func(d *ebiten.Image, ox, oy float64) {
		detailLine(d, ox, oy, 1, alpha(colWhite, 0.7), 0, -4, 0, -60)
	})
	pHourFrame  = newPart(mGold, -18, -4, 18, -4, 18, 4, -18, 4)
	pGlassTop   = newPart(mGlass, -13, 0, 13, 0, 2, 20, -2, 20)
	pFeather    = newPart(mFeather, 0, -3, 30, -10, 36, -4, 30, 2, 0, 4)
	pGoldShield = newPart(mGold, -18, -18, 0, -22, 18, -18, 16, 4, 0, 22, -16, 4).with(func(d *ebiten.Image, ox, oy float64) {
		disc(d, ox, oy-2, 6, 6, rgb(0xff, 0xf0, 0xb0))
	})
	pDragon = newPart(mScale, -26, -4, -16, -14, 0, -16, 16, -10, 30, -4, 34, 4, 18, 8, 4, 14, -20, 12).with(func(d *ebiten.Image, ox, oy float64) {
		for i := 0; i < 5; i++ {
			detailLine(d, ox, oy, 1, alpha(colInk, 0.4), -20+float64(i)*8, -8, -16+float64(i)*8, 8)
		}
		disc(d, ox+10, oy-6, 3, 2.2, colYellow)
		disc(d, ox+10, oy-6, 1, 2, colInk)
	})
	pDragonJaw   = newPart(mScale, -12, 0, 22, 2, 28, 8, 10, 12, -10, 8)
	pDragonHorn  = newPart(mHorn, 0, 0, -10, -10, -24, -14, -12, -4)
	pDartFeather = newPart(mCrimson, 0, -4, -14, -10, -10, 0, -14, 10, 0, 4)
	pPhWing      = newPart(mEmber, 0, 0, 14, -14, 30, -30, 44, -40, 40, -26, 48, -20, 36, -10, 42, -4, 26, 2, 12, 8)
	pPhWingB     = newPart(mEmber.col(rgb(0xd0, 0x40, 0x20)), mirror([]float64{0, 0, 14, -14, 30, -30, 44, -40, 40, -26, 48, -20, 36, -10, 42, -4, 26, 2, 12, 8}, 0)...)
	pPhBody      = newPart(mEmber.col(rgb(0xff, 0xb0, 0x40)), -6, -10, 0, -14, 6, -10, 7, 6, 0, 14, -7, 6)
	pPhTail      = newPart(mEmber.col(rgb(0xe0, 0x50, 0x20)), -4, 0, 4, 0, 8, 14, 2, 26, 0, 20, -2, 26, -8, 14)
	pPhHead      = newPart(mEmber.col(rgb(0xff, 0xc8, 0x50)), -5, 2, -5, -5, 0, -8, 6, -5, 12, -2, 6, 2).with(func(d *ebiten.Image, ox, oy float64) {
		disc(d, ox+1, oy-3, 1.4, 1.4, colInk)
	})
	pBolt = sharpPart(mGold.col(rgb(0xff, 0xd8, 0x50)), 5, -28, -12, 3, -1, 3, -6, 28, 12, -5, 1, -5)
)

func sw(dst *ebiten.Image, x, y, a, k float64) {
	b := xf{x, y, a, k}
	hBlade.draw(dst, b.j(0, -4, 0))
	hGuard.draw(dst, b.j(0, -2, 0))
	hGrip.draw(dst, b.j(0, -2, 0))
	hPommel.draw(dst, b.j(0, 15, 0))
}

// litArt draws painted art for card art a centred at (x,y), scale k. It
// reports false for arts that stay as vector magic effects.
func litArt(dst *ebiten.Image, a int, x, y, k, t float64, rarity int) bool {
	P := func(p *part, lx, ly, ang float64) { p.draw(dst, xf{x + lx*k, y + ly*k, ang, k}) }
	switch a {
	case aSword:
		sw(dst, x-18*k, y+26*k, 0.75, 0.72*k)
		sparkle(dst, x+22*k, y-20*k, 6*k, t)
	case aDagger:
		P(pDagger, -8, 18, 0.9)
		P(hGuard, -8, 18, 0.9)
		for i := 0; i < 3; i++ {
			yy := y + float64(i-1)*10*k
			line(dst, x-46*k, yy+8*k, x-28*k, yy+4*k, 2*k, alpha(colWhite, 0.5))
		}
	case aTwin:
		sw(dst, x-18*k, y+24*k, 0.75, 0.62*k)
		sw(dst, x+18*k, y+24*k, -0.75, 0.62*k)
		glow(dst, x, y-6*k, 22*k, colOrange, 0.7)
		sparkle(dst, x, y-6*k, 8*k, t)
	case aShield:
		P(pKite, 0, 0, 0)
	case aBrace:
		P(pBuckler, 0, 0, math.Sin(t)*0.1)
		for _, d := range []float64{-1, 1} {
			curve(dst, x+d*32*k, y-14*k, x+d*40*k, y, x+d*32*k, y+14*k, 2.5*k, alpha(colBone, 0.6))
		}
	case aHammer:
		P(pHandle, -10, -16, -0.55)
		P(pHammerHead, -10, -16, -0.55)
		burst(dst, x-24*k, y-24*k, 12*k, colOrange, t)
	case aAxe:
		P(pHandle, 8, -24, 0.45)
		P(pAxeBlade, 6, -18, 0.45)
	case aTower:
		P(pTower, 0, 0, 0)
		if rarity >= rRare {
			glow(dst, x, y, 34*k, colBlue, 0.35)
		}
	case aHorn:
		P(pHorn, -4, 4, 0)
		for i := 1; i <= 3; i++ {
			r := float64(i) * 7 * k
			curve(dst, x+30*k+r*0.5, y-26*k-r*0.5, x+36*k+r, y-12*k, x+30*k+r*0.5, y+r*0.4-2*k, 2*k, alpha(colBone, 0.8-float64(i)*0.2))
		}
	case aBook:
		glow(dst, x, y-6*k, 34*k, colPurple, 0.6)
		P(pCover, 0, 2, 0)
		P(pPages, 0, 0, 0)
		sparkle(dst, x-14*k, y-24*k, 5*k, t)
		sparkle(dst, x+16*k, y-28*k, 4*k, t+1)
	case aFist:
		flames(dst, x, y-10*k, 22*k, t)
		P(pGauntlet, 0, 6, 0)
	case aCloud:
		bp := pBlob
		if rarity >= rEpic {
			bp = pBlobV
		}
		for i, b := range [][3]float64{{-18, 4, 1.1}, {16, 4, 1.15}, {0, -8, 1.3}, {-4, 10, 1}} {
			sc := b[2] * (1 + 0.05*math.Sin(t*3+float64(i)))
			bp.draw(dst, xf{x + b[0]*k, y + b[1]*k, 0, k * sc})
		}
		P(pSkull, 0, 0, 0)
	case aFang:
		P(pDrop, 0, 8+math.Mod(t*8, 6), 0)
		P(pFang, -14, -24, 0.15)
		P(pFang, 14, -24, -0.15)
	case aMeteor:
		for i := 0; i < 3; i++ {
			o := float64(i) * 7 * k
			shape(dst, []float64{x - 4*k + o, y - 8*k - o, x + 34*k + o, y - 36*k - o, x + 8*k + o, y + 6*k - o}, alpha(colOrange, 0.8-float64(i)*0.2), alpha(colYellow, 0.2), 0)
		}
		glow(dst, x-4*k, y+6*k, 34*k, colOrange, 0.7)
		P(pRock, -6, 8, t*0.5)
	case aExcalibur:
		for i := 0; i < 10; i++ {
			an := float64(i)*math.Pi/5 + t*0.6
			shape(dst, []float64{x, y - 4*k, x + math.Cos(an)*60*k, y - 4*k + math.Sin(an)*60*k, x + math.Cos(an+0.15)*60*k, y - 4*k + math.Sin(an+0.15)*60*k}, alpha(colYellow, 0.3), clear, 0)
		}
		glow(dst, x, y-4*k, 32*k, colYellow, 0.8)
		P(pGoldBlade, 0, 28, 0)
		P(hGuard, 0, 30, 0)
		sparkle(dst, x, y-30*k, 7*k, t)
	case aHourglass:
		glow(dst, x, y, 30*k, colBlue, 0.4)
		P(pGlassTop, 0, -24, 0)
		P(pGlassTop, 0, 24, math.Pi)
		f := math.Mod(t*0.3, 1)
		shape(dst, []float64{x - 10*k*(1-f), y - 20*k + 16*k*f, x + 10*k*(1-f), y - 20*k + 16*k*f, x, y - 2*k}, rgb(0xff, 0xd8, 0x70), rgb(0xe8, 0xa0, 0x30), 0)
		shape(dst, []float64{x - 10*k*f, y + 22*k - 12*k*f, x + 10*k*f, y + 22*k - 12*k*f, x + 11*k, y + 22*k, x - 11*k, y + 22*k}, rgb(0xff, 0xd8, 0x70), rgb(0xe8, 0xa0, 0x30), 0)
		P(pHourFrame, 0, -26, 0)
		P(pHourFrame, 0, 26, 0)
	case aAegis:
		glow(dst, x, y, 40*k, colYellow, 0.6)
		for i := 0; i < 3; i++ {
			fl := math.Sin(t*2) * 0.08
			P(pFeather, 14, -6+float64(i)*7, -0.5+float64(i)*0.35+fl)
			pFeather.draw(dst, xf{x - 14*k, y + (-6+float64(i)*7)*k, math.Pi + 0.5 - float64(i)*0.35 - fl, k})
		}
		P(pGoldShield, 0, 2, 0)
	case aPhoenix:
		glow(dst, x, y, 44*k, colOrange, 0.7)
		fl := math.Sin(t*5) * 0.15
		P(pPhWingB, -4, -2, 0.2-fl)
		P(pPhTail, 0, 10, 0)
		P(pPhBody, 0, 0, 0)
		P(pPhWing, 4, -2, -0.2+fl)
		P(pPhHead, 0, -14, 0)
		for i := 0; i < 3; i++ {
			fy := y + 30*k - math.Mod(t*24+float64(i)*10, 30)*k
			glow(dst, x-10*k+float64(i)*10*k, fy, 6*k, colYellow, 0.6)
		}
	case aDragon:
		shape(dst, []float64{x + 18*k, y, x + 52*k, y - 20*k, x + 56*k, y + 16*k, x + 20*k, y + 10*k}, alpha(colOrange, 0.9), alpha(colYellow, 0.3), 0)
		glow(dst, x+34*k, y, 26*k, colOrange, 0.8)
		P(pDragonHorn, -12, -10, 0)
		P(pDragonJaw, -8, 6, 0.15+0.1*math.Sin(t*4))
		P(pDragon, -10, 0, 0)
	case aDart:
		P(pDartFeather, -20, 10, -0.6)
		P(pDagger, -14, 6, 0.95)
		glow(dst, x+16*k, y-16*k, 10*k, colGreen, 0.8)
		for i := 0; i < 2; i++ {
			dy := math.Mod(t*20+float64(i)*12, 24) * k
			drop(dst, x+16*k+float64(i)*6*k, y-8*k+dy, 3*k, colGreen)
		}
	case aBolt:
		glow(dst, x, y, 36*k, colYellow, 0.7)
		P(pBolt, 0, 0, 0)
		sparkle(dst, x+16*k, y-20*k, 6*k, t)
	default:
		return false
	}
	return true
}
