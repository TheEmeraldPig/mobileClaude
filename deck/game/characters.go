package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

func rectI(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

// Characters are drawn into a buffer first, then stamped onto the screen with
// a scale, a flip, a fade and a white hit-flash.
type sprite struct {
	w, h float64
	s    float64
	img  *ebiten.Image
}

func newSprite(w, h int) *sprite { return &sprite{w: float64(w), h: float64(h)} }

// buf returns the cleared buffer, sized for the current screen scale.
func (s *sprite) buf() *ebiten.Image {
	if s.img == nil || s.s != S {
		if s.img != nil {
			s.img.Deallocate()
		}
		s.img, s.s = ebiten.NewImage(int(s.w*S), int(s.h*S)), S
	}
	s.img.Clear()
	return s.img
}

// stamp draws the buffer so its anchor (ax,ay) lands on (x,y). A negative sx flips it.
func (s *sprite) stamp(dst *ebiten.Image, x, y, ax, ay, scale, sx, sy, fade, flash float64) {
	cm := colorm.ColorM{}
	cm.Scale(1, 1, 1, fade)
	if flash > 0 {
		cm.Translate(flash*fade, flash*fade, flash*fade, 0)
	}
	op := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-ax*S, -ay*S)
	op.GeoM.Scale(scale*sx, scale*sy)
	op.GeoM.Translate(x*S, y*S)
	colorm.DrawImage(dst, s.img, cm, op)
}

func glowEye(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	glow(dst, x, y, r*4, c, 0.7)
	disc(dst, x, y, r*1.3, r*0.8, light(c, 0.3))
	disc(dst, x, y, r*0.6, r*0.4, colWhite)
}

// drawSword is a vector sword for small icons.
func drawSword(dst *ebiten.Image, hx, hy, ang, s float64) {
	R := func(p []float64) []float64 { return rot(p, hx, hy, ang) }
	shape(dst, R([]float64{hx - 5*s, hy - 10*s, hx - 5*s, hy - 100*s, hx, hy - 112*s, hx + 5*s, hy - 100*s, hx + 5*s, hy - 10*s}), rgb(0xd0, 0xd8, 0xe8), rgb(0x3a, 0x40, 0x50), 1.4)
	shape(dst, R([]float64{hx - 18*s, hy - 12*s, hx - 22*s, hy - 18*s, hx, hy - 14*s, hx + 22*s, hy - 18*s, hx + 18*s, hy - 12*s, hx, hy - 6*s}), colGoldHi, colGoldLo, 1.2)
	shape(dst, R([]float64{hx - 3*s, hy - 6*s, hx + 3*s, hy - 6*s, hx + 3*s, hy + 14*s, hx - 3*s, hy + 14*s}), rgb(0x4a, 0x2a, 0x1a), rgb(0x1a, 0x0a, 0x04), 1)
}

// anim is what a character is doing: t is the clock, the rest run 0..1.
type anim struct {
	t    float64
	atk  float64 // attack swing progress
	cast float64 // skill/spell progress
	hurt float64 // recoil, fading from 1 to 0
	dead float64 // collapse progress
}

// shift moves detail points by the part's image origin.
func shift(ox, oy float64, p ...float64) []float64 {
	o := make([]float64, len(p))
	for i := 0; i+1 < len(p); i += 2 {
		o[i], o[i+1] = p[i]+ox, p[i+1]+oy
	}
	return o
}

func detailLine(dst *ebiten.Image, ox, oy, w float64, c color.RGBA, p ...float64) {
	pp := shift(ox, oy, p...)
	for i := 0; i+3 < len(pp); i += 2 {
		line(dst, pp[i], pp[i+1], pp[i+2], pp[i+3], w, c)
	}
}

// attackPose returns lean, lunge and the three arm angles for a weapon swing at progress p.
func attackPose(p float64) (lean, dx, ua, fa, sw float64) {
	if p <= 0 || p >= 1 {
		return 0, 0, 0, 0, 0
	}
	lean = keys(p, 0, 0, 0.3, -0.28, 0.45, 0.32, 0.7, 0.15, 1, 0)
	dx = keys(p, 0, 0, 0.3, -10, 0.45, 50, 0.75, 34, 1, 0)
	ua = keys(p, 0, 0, 0.3, 3.2, 0.45, -0.6, 0.7, -0.4, 1, 0)
	fa = keys(p, 0, 0, 0.3, 4.6, 0.45, 0.4, 0.7, 0.2, 1, 0)
	sw = keys(p, 0, 0, 0.3, -3.0, 0.45, 1.5, 0.7, 1.0, 1, 0)
	return
}

// ---- the hero: a horned dark knight, facing right ----

var (
	hSteel = mSteel.col(rgb(0x7e, 0x86, 0x9c))
	hDark  = mDarkSteel

	hCapeTop = newPart(mCrimson, -20, -4, -4, -8, 10, -6, 8, 30, -6, 36, -26, 34)
	hCapeLow = newPart(mCrimson, -26, 0, -6, -4, 8, -2, 10, 40, 4, 56, -6, 48, -14, 62, -24, 50, -34, 58, -32, 30)
	hTorso   = newPart(hSteel, -20, 6, -25, -16, -23, -38, -15, -54, 0, -57, 15, -54, 23, -40, 25, -16, 19, 6).with(func(d *ebiten.Image, ox, oy float64) {
		strokePath(d, smoothPath(shift(ox, oy, -15, 0, -19, -18, -17, -36, -10, -48, 10, -48, 17, -36, 19, -18, 15, 0)), 1.6, alpha(colGold, 0.8))
		detailLine(d, ox, oy, 1.2, alpha(colInk, 0.5), 0, -50, 0, 2)
		shape(d, shift(ox, oy, 0, -36, 7, -26, 0, -12, -7, -26), rgb(0xff, 0xa0, 0x40), rgb(0xa0, 0x30, 0x10), 1)
	})
	hBelt = sharpPart(mLeather, -21, -2, 21, -2, 21, 7, -21, 7).with(func(d *ebiten.Image, ox, oy float64) {
		shape(d, shift(ox, oy, -5, -3, 5, -3, 5, 8, -5, 8), colGoldHi, colGoldLo, 1)
	})
	hTasset = newPart(hSteel, -24, 2, 24, 2, 27, 26, 12, 32, 0, 24, -12, 32, -27, 26).with(func(d *ebiten.Image, ox, oy float64) {
		detailLine(d, ox, oy, 1.2, alpha(colInk, 0.55), -8, 6, -10, 28)
		detailLine(d, ox, oy, 1.2, alpha(colInk, 0.55), 8, 6, 10, 28)
	})
	hPauldron = newPart(hSteel, -15, 8, -17, -6, -6, -17, 10, -15, 19, -3, 15, 11, 0, 10).with(func(d *ebiten.Image, ox, oy float64) {
		strokePath(d, smoothPath(shift(ox, oy, -11, 5, -12, -4, -4, -12, 8, -11, 14, -2, 11, 7)), 1.2, alpha(colGold, 0.7))
	})
	hSpike = sharpPart(hSteel, -4, 0, 0, -16, 4, 0)
	hUpper = newPart(hDark, limb(30, 13, 11)...)
	hFore  = newPart(hSteel, limb(28, 12, 10)...)
	hHand  = newPart(hDark, -6, -2, 6, -2, 7, 8, 0, 13, -7, 8)
	hThigh = newPart(hDark, limb(46, 17, 13)...)
	hShin  = newPart(hSteel, limb(46, 13, 10)...).with(func(d *ebiten.Image, ox, oy float64) {
		shape(d, shift(ox, oy, -8, -2, 8, -2, 7, 8, -7, 8), rgb(0xa8, 0xb0, 0xc0), rgb(0x4a, 0x50, 0x60), 1)
	})
	hFoot = newPart(hSteel, -7, -5, 5, -6, 17, 1, 19, 6, -7, 7)
	hHelm = newPart(hSteel, -14, 2, -17, -12, -15, -27, -7, -37, 6, -38, 15, -30, 18, -14, 15, 2, 0, 5).with(func(d *ebiten.Image, ox, oy float64) {
		fillPath(d, pathOf(shift(ox, oy, -11, -19, 17, -20, 17, -14, -11, -13)), rgb(0x08, 0x06, 0x08))
		fillPath(d, pathOf(shift(ox, oy, 3, -14, 8, -14, 8, -2, 3, -2)), rgb(0x08, 0x06, 0x08))
		detailLine(d, ox, oy, 1.2, alpha(colWhite, 0.35), 1, -36, 1, -22)
	})
	hHorn  = newPart(mHorn, -2, -2, 6, -6, 14, -16, 18, -30, 15, -40, 11, -26, 4, -14, -3, -9)
	hBlade = sharpPart(mBlade, -3.6, 0, -3.6, -94, 0, -106, 3.6, -94, 3.6, 0).with(func(d *ebiten.Image, ox, oy float64) {
		detailLine(d, ox, oy, 1, alpha(rgb(0x50, 0x58, 0x6a), 0.9), 0, -4, 0, -92)
	})
	hGuard  = newPart(mGold, -18, -2, -21, -7, -8, -4, 0, -6, 8, -4, 21, -7, 18, -2, 0, 3)
	hGrip   = sharpPart(mLeather, -2.6, 0, 2.6, 0, 2.6, 16, -2.6, 16)
	hPommel = newPart(mGold, -4, 0, 0, -4, 4, 0, 0, 5)
)

// drawHero draws the knight with feet at (x,y), facing right.
func drawHero(dst *ebiten.Image, x, y float64, a anim) {
	t := a.t
	br := math.Sin(t * 2.2)
	lean, dx, ua, fa, sw := attackPose(a.atk)
	cast := math.Sin(clamp01(a.cast) * math.Pi)
	hurt := a.hurt
	dead := easeOut(a.dead)
	dx += -14*hurt - 10*dead
	rootY := y - 94 + br*1.2 + 30*dead
	root := xf{x + dx, rootY, 0, 1}
	torsoA := -0.04 + br*0.015 + lean - 0.3*hurt - 0.1*cast - 1.1*dead
	torso := xf{root.x, root.y, torsoA, 1}
	shadow(dst, x+dx*0.6, y+1, 52, 11, 0.6*(1-dead*0.3))

	// cape: two segments that trail behind the motion
	sway := math.Sin(t*1.7)*0.08 + lean*0.6 - dx*0.004
	ct := torso.j(-6, -50, 0)
	capeT := xf{ct.x, ct.y, 0.08 + sway*0.5 + torsoA*0.4, 1}
	capeL := xf{0, 0, 0.12 + sway + math.Sin(t*2.3+0.8)*0.07, 1}
	capeL.x, capeL.y = capeT.at(-8, 32)
	hCapeLow.draw(dst, capeL)
	hCapeTop.draw(dst, capeT)

	// arms: idle pose plus any swing or spell
	shF := torso.j(13, -44, 0)
	shB := torso.j(-9, -44, 0)
	aUA := -0.45 + math.Sin(t*2.2+1)*0.04 + ua - 1.6*cast
	aFA := -1.75 + math.Sin(t*2.2+1.4)*0.05 + fa - 0.6*cast + 1.4*dead
	aSW := 0.42 + math.Sin(t*2.2+1.8)*0.04 + sw + 0.3*cast + 1.6*dead
	bUA := 0.18 + math.Sin(t*2.2)*0.05 - 0.2*hurt
	bFA := -0.1 + math.Sin(t*2.2+0.5)*0.05

	// back arm
	bu := xf{shB.x, shB.y, bUA, 1}
	bf := xf{0, 0, bFA, 1}
	bf.x, bf.y = bu.at(0, 30)
	hUpper.drawTint(dst, bu, 0.6)
	hFore.drawTint(dst, bf, 0.6)
	bh := bf.j(0, 28, 0)
	hHand.drawTint(dst, bh, 0.6)

	// legs, solved so the feet stay planted
	fFoot := x + 15 + dx*0.85
	bFoot := x - 12 + dx*0.3
	leg := func(hipX, hipY, fx float64, tintF float32) {
		a1, a2 := ik(hipX, hipY, fx, y-6, 46, 46, 1)
		th := xf{hipX, hipY, a1, 1}
		sh := xf{0, 0, a2, 1}
		sh.x, sh.y = th.at(0, 46)
		ft := sh.j(0, 46, -a2)
		hThigh.drawTint(dst, th, tintF)
		hShin.drawTint(dst, sh, tintF)
		hFoot.drawTint(dst, xf{ft.x, ft.y, 0, 1}, tintF)
	}
	hx, hy := root.at(-7, 4)
	leg(hx, hy, bFoot, 0.65)
	hx, hy = root.at(7, 4)
	leg(hx, hy, fFoot, 1)

	hTasset.draw(dst, xf{root.x, root.y + 2, torsoA * 0.4, 1})
	hPauldron.drawTint(dst, xf{shB.x, shB.y, torsoA - 0.2, 1}, 0.7)
	hTorso.draw(dst, torso)
	hBelt.draw(dst, torso.j(0, 0, 0))

	// head
	head := torso.j(1, -55, br*0.02+0.15*hurt-0.3*dead)
	hHorn.drawTint(dst, head.j(-9, -26, -0.5), 0.7)
	hHelm.draw(dst, head)
	hHorn.draw(dst, head.j(8, -27, 0.1))
	vx, vy := head.at(4, -17)
	flick := 0.8 + 0.2*math.Sin(t*7)
	glow(dst, vx, vy, 22, colOrange, 0.45*flick*(1-dead))
	gx, gy := head.at(6, -16)
	disc(dst, gx, gy, 9, 2.2, alpha(rgb(0xff, 0xc0, 0x60), flick*(1-dead)))

	// sword arm and greatsword
	fu := xf{shF.x, shF.y, aUA, 1}
	ff := xf{0, 0, aFA, 1}
	ff.x, ff.y = fu.at(0, 30)
	hUpper.draw(dst, fu)
	hFore.draw(dst, ff)
	hand := ff.j(0, 27, 0)
	swd := xf{hand.x, hand.y, aSW, 1}
	hBlade.draw(dst, swd.j(0, -4, 0))
	hGuard.draw(dst, swd.j(0, -2, 0))
	hGrip.draw(dst, swd.j(0, -2, 0))
	hPommel.draw(dst, swd.j(0, 15, 0))
	hHand.draw(dst, xf{hand.x, hand.y, aSW, 1})
	// runes along the blade
	for i := 0; i < 4; i++ {
		rx, ry := swd.at(0, -22-float64(i)*18)
		glow(dst, rx, ry, 7, colOrange, (0.5+0.5*math.Sin(t*3+float64(i)))*(1-dead))
	}
	if a.atk > 0.3 && a.atk < 0.6 { // swing trail
		for k := 0; k < 6; k++ {
			pp := a.atk - float64(k)*0.025
			_, _, ua2, fa2, sw2 := attackPose(pp)
			u2 := xf{shF.x, shF.y, -0.45 + ua2, 1}
			f2 := xf{0, 0, -1.75 + fa2, 1}
			f2.x, f2.y = u2.at(0, 30)
			h2 := f2.j(0, 27, 0)
			s2 := xf{h2.x, h2.y, 0.42 + sw2, 1}
			tx, ty := s2.at(0, -96)
			glow(dst, tx, ty, 18, colOrange, 0.35*(1-float64(k)/6))
		}
	}
	hPauldron.draw(dst, xf{shF.x, shF.y, torsoA + 0.1, 1})
	hSpike.draw(dst, xf{shF.x + 2, shF.y - 10, torsoA + 0.25, 1})
	if cast > 0 {
		px, py := hand.at(0, 6)
		glow(dst, px, py, 40*cast, colBlue, 0.7*cast)
	}
}

// ---- a shared humanoid skeleton for most enemies (authored facing right) ----

type body struct {
	torso, tasset, head, upper, fore, hand, thigh, shin, foot *part
	weapon                                                    []*part // attached to the front hand, pointing up
	shield                                                    *part   // on the back hand
	hipH, torsoL, uaL, faL, thighL, shinL                     float64
	hunch                                                     float64 // forward lean
	stride                                                    float64 // distance between feet
	noLegs                                                    bool    // robed: the tasset covers the legs
	weaponA                                                   float64 // resting weapon angle
	before, after                                             func(dst *ebiten.Image, torso, head, hand xf, a anim)
}

func (b *body) draw(dst *ebiten.Image, x, y float64, a anim) {
	t := a.t
	br := math.Sin(t * 2)
	lean, dx, ua, fa, sw := attackPose(a.atk)
	hurt := a.hurt
	dead := easeOut(a.dead)
	dx += -14*hurt - 6*dead
	root := xf{x + dx, y - b.hipH + br*1.4 + b.hipH*0.3*dead, 0, 1}
	torsoA := b.hunch + br*0.02 + lean*0.8 - 0.32*hurt - 1.0*dead
	torso := xf{root.x, root.y, torsoA, 1}
	shadow(dst, x+dx*0.6, y+1, b.stride*1.6+30, 11, 0.6)
	shF := torso.j(b.torsoL*0.24, -b.torsoL*0.82, 0)
	shB := torso.j(-b.torsoL*0.16, -b.torsoL*0.82, 0)
	head := torso.j(b.torsoL*0.05, -b.torsoL, br*0.03+0.2*hurt-0.4*dead)
	if b.before != nil {
		b.before(dst, torso, head, xf{}, a)
	}
	// back arm (with shield)
	bu := xf{shB.x, shB.y, 0.2 + math.Sin(t*2+0.4)*0.06 - lean*0.4, 1}
	bf := xf{0, 0, -0.5 + math.Sin(t*2)*0.05, 1}
	bf.x, bf.y = bu.at(0, b.uaL)
	b.upper.drawTint(dst, bu, 0.62)
	b.fore.drawTint(dst, bf, 0.62)
	bh := bf.j(0, b.faL, 0)
	b.hand.drawTint(dst, bh, 0.62)
	if b.shield != nil {
		b.shield.drawTint(dst, xf{bh.x, bh.y, 0.1, 1}, 0.85)
	}
	// legs
	if !b.noLegs {
		leg := func(hipX, hipY, fx float64, f float32) {
			a1, a2 := ik(hipX, hipY, fx, y-5, b.thighL, b.shinL, 1)
			th := xf{hipX, hipY, a1, 1}
			sh := xf{0, 0, a2, 1}
			sh.x, sh.y = th.at(0, b.thighL)
			ft := sh.j(0, b.shinL, 0)
			b.thigh.drawTint(dst, th, f)
			b.shin.drawTint(dst, sh, f)
			if b.foot != nil {
				b.foot.drawTint(dst, xf{ft.x, ft.y, 0, 1}, f)
			}
		}
		hx, hy := root.at(-6, 3)
		leg(hx, hy, x-b.stride*0.5+dx*0.3, 0.65)
		hx, hy = root.at(6, 3)
		leg(hx, hy, x+b.stride*0.6+dx*0.85, 1)
	}
	if b.tasset != nil {
		b.tasset.draw(dst, xf{root.x, root.y, torsoA*0.3 + math.Sin(t*2.3)*0.03, 1})
	}
	b.torso.draw(dst, torso)
	if b.head != nil {
		b.head.draw(dst, head)
	}
	// front arm with weapon
	fu := xf{shF.x, shF.y, -0.3 + math.Sin(t*2+1)*0.05 + ua, 1}
	ff := xf{0, 0, -1.4 + math.Sin(t*2+1.3)*0.05 + fa + 1.2*dead, 1}
	ff.x, ff.y = fu.at(0, b.uaL)
	b.upper.draw(dst, fu)
	b.fore.draw(dst, ff)
	hand := ff.j(0, b.faL, 0)
	wa := b.weaponA + math.Sin(t*2+1.6)*0.04 + sw + 1.4*dead
	for _, w := range b.weapon {
		w.draw(dst, xf{hand.x, hand.y, wa, 1})
	}
	b.hand.draw(dst, xf{hand.x, hand.y, wa, 1})
	if b.after != nil {
		b.after(dst, torso, head, xf{hand.x, hand.y, wa, 1}, a)
	}
}

// ---- enemies ----

var skeleton = func() *body {
	bone := mBone
	return &body{
		torso: newPart(bone, -12, 4, -15, -20, -14, -40, -6, -50, 6, -50, 14, -40, 15, -20, 12, 4, 0, 6).with(func(d *ebiten.Image, ox, oy float64) {
			for i := 0; i < 4; i++ {
				yy := -40 + float64(i)*9
				curve(d, ox-13, oy+yy+4, ox, oy+yy-3, ox+13, oy+yy+4, 3.2, rgb(0x10, 0x0c, 0x0a))
			}
			detailLine(d, ox, oy, 3, rgb(0xb0, 0xa4, 0x88), 0, -48, 0, 2)
		}),
		tasset: newPart(bone, -13, -2, 13, -2, 15, 8, 0, 12, -15, 8),
		head: newPart(bone, -13, 0, -16, -14, -12, -28, 0, -33, 12, -28, 16, -14, 12, 0, 8, 8, -8, 8).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, smoothPath(shift(ox, oy, -10, -16, -3, -18, -3, -10, -10, -9)), rgb(0x06, 0x04, 0x04))
			fillPath(d, smoothPath(shift(ox, oy, 3, -18, 11, -16, 11, -9, 3, -10)), rgb(0x06, 0x04, 0x04))
			fillPath(d, pathOf(shift(ox, oy, -1, -6, 2, -6, 0.5, -1)), rgb(0x06, 0x04, 0x04))
			for i := 0; i < 5; i++ {
				detailLine(d, ox, oy, 1.2, rgb(0x20, 0x18, 0x10), -6+float64(i)*3, 1, -6+float64(i)*3, 7)
			}
		}),
		upper: newPart(bone, limb(28, 7, 6)...), fore: newPart(bone, limb(26, 6, 5)...),
		hand:  newPart(bone, -4, -1, 4, -1, 5, 6, 0, 9, -5, 6),
		thigh: newPart(bone, limb(34, 8, 6)...), shin: newPart(bone, limb(34, 7, 5)...),
		foot: newPart(bone, -5, -3, 4, -4, 12, 1, 12, 4, -5, 4),
		weapon: []*part{
			sharpPart(mBlade.col(rgb(0x8a, 0x78, 0x68)), -3.5, -2, -3.5, -70, 0, -80, 3.5, -66, 2.5, -48, 4, -36, 3.5, -2),
			sharpPart(mDarkSteel.col(rgb(0x5a, 0x4a, 0x3a)), -12, -3, 12, -3, 12, 2, -12, 2),
		},
		shield: newPart(mWood, -20, -14, 0, -24, 20, -14, 22, 6, 0, 24, -22, 6).with(func(d *ebiten.Image, ox, oy float64) {
			strokePath(d, smoothPath(shift(ox, oy, -17, -12, 0, -21, 17, -12, 19, 5, 0, 21, -19, 5)), 2.5, rgb(0x6a, 0x5a, 0x4a))
			disc(d, ox, oy, 5, 5, rgb(0x6a, 0x5a, 0x4a))
		}),
		hipH: 70, torsoL: 50, uaL: 28, faL: 26, thighL: 34, shinL: 34, stride: 26, weaponA: 0.5,
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, ex := range []float64{-6.5, 7} {
				x, y := head.at(ex, -13)
				glowEye(dst, x, y, 2.4, colRed)
			}
		},
	}
}()

var cultist = func() *body {
	robe := mRobe
	return &body{
		torso: newPart(robe, -16, 6, -20, -18, -16, -42, -6, -52, 8, -52, 16, -42, 20, -18, 16, 6).with(func(d *ebiten.Image, ox, oy float64) {
			detailLine(d, ox, oy, 2, alpha(colGold, 0.7), 0, -48, 0, 6)
			for i := 0; i < 3; i++ {
				shape(d, shift(ox, oy, -3, -30+float64(i)*10, 0, -34+float64(i)*10, 3, -30+float64(i)*10, 0, -26+float64(i)*10), colGoldHi, colGoldLo, 0)
			}
		}),
		tasset: newPart(robe, -18, -2, 18, -2, 26, 40, 30, 68, 16, 62, 4, 72, -8, 62, -22, 70, -30, 60, -24, 30),
		head: newPart(robe.col(rgb(0x5a, 0x10, 0x1a)), -16, 4, -18, -18, -8, -36, 6, -36, 14, -22, 16, 4).with(func(d *ebiten.Image, ox, oy float64) {
			shape(d, shift(ox, oy, -9, -24, 9, -24, 10, -10, 1, 2, -8, -6), rgb(0xe8, 0xe0, 0xc8), rgb(0x8a, 0x80, 0x68), 1)
			fillPath(d, smoothPath(shift(ox, oy, -6, -17, -1, -18, -1, -14, -6, -13)), rgb(0x04, 0x02, 0x02))
			fillPath(d, smoothPath(shift(ox, oy, 2, -18, 7, -17, 7, -13, 2, -14)), rgb(0x04, 0x02, 0x02))
		}),
		upper: newPart(robe, limb(28, 12, 12)...), fore: newPart(robe, limb(26, 12, 14)...),
		hand: newPart(mBone, -4, -1, 4, -1, 5, 6, 0, 9, -5, 6),
		weapon: []*part{
			sharpPart(mWood, -2.5, 60, 2.5, 60, 2.5, -110, -2.5, -110),
			newPart(mBone, -9, -110, -10, -122, 0, -130, 10, -122, 9, -110, 0, -104),
		},
		hipH: 70, torsoL: 52, uaL: 28, faL: 26, thighL: 34, shinL: 34, stride: 20, noLegs: true, weaponA: 0.05,
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, ex := range []float64{-3.5, 4.5} {
				x, y := head.at(ex, -15.5)
				glowEye(dst, x, y, 2, rgb(0x7a, 0xff, 0x50))
			}
			fx, fy := hand.at(0, -134)
			glow(dst, fx, fy, 40, rgb(0x7a, 0xff, 0x50), 0.55+0.15*math.Sin(a.t*9))
			flames(dst, fx, fy+8, 9, a.t)
		},
	}
}()

var abyssKnight = func() *body {
	ar := mBlackIron
	void := rgb(0xa0, 0x50, 0xff)
	runes := func(d *ebiten.Image, ox, oy float64, p ...float64) {
		detailLine(d, ox, oy, 1.8, light(void, 0.4), p...)
	}
	return &body{
		torso: newPart(ar, -24, 6, -30, -20, -28, -46, -18, -64, 0, -68, 18, -64, 28, -46, 30, -20, 24, 6).with(func(d *ebiten.Image, ox, oy float64) {
			runes(d, ox, oy, -14, -50, -4, -32, -10, -14)
			runes(d, ox, oy, 12, -48, 4, -30, 10, -10)
			strokePath(d, smoothPath(shift(ox, oy, -19, 0, -24, -22, -22, -44, -13, -58, 13, -58, 22, -44, 24, -22, 19, 0)), 1.4, alpha(void, 0.5))
		}),
		tasset: newPart(ar, -28, 0, 28, 0, 32, 34, 14, 40, 0, 30, -14, 40, -32, 34),
		head: newPart(ar, -16, 2, -20, -16, -17, -34, -8, -46, 8, -46, 17, -34, 20, -16, 16, 2).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, pathOf(shift(ox, oy, -13, -24, 19, -25, 19, -19, -13, -18)), rgb(0x06, 0x02, 0x0c))
		}),
		upper: newPart(ar, limb(34, 16, 13)...), fore: newPart(ar, limb(32, 14, 12)...),
		hand:  newPart(ar, -7, -2, 7, -2, 8, 9, 0, 14, -8, 9),
		thigh: newPart(ar, limb(44, 18, 15)...), shin: newPart(ar, limb(44, 15, 12)...),
		foot: newPart(ar, -8, -5, 6, -6, 20, 1, 22, 7, -8, 8),
		weapon: []*part{
			sharpPart(mDarkSteel, -6, -4, -6, -130, 0, -146, 6, -130, 6, -4),
			newPart(ar, -24, -2, -28, -9, 0, -6, 28, -9, 24, -2, 0, 4),
			sharpPart(mLeather, -3, 0, 3, 0, 3, 20, -3, 20),
		},
		hipH: 92, torsoL: 66, uaL: 34, faL: 32, thighL: 44, shinL: 44, stride: 34, weaponA: 0.35,
		before: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			cx, cy := torso.at(-10, -60)
			cape := []float64{cx - 10, cy, cx + 24, cy - 2, cx + 30 + math.Sin(a.t*1.6)*6, cy + 120, cx - 30 + math.Sin(a.t*1.9)*8, cy + 132}
			blob(dst, cape, rgb(0x2a, 0x10, 0x40), rgb(0x0a, 0x04, 0x12), 1.4)
		},
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			x, y := head.at(3, -21.5)
			glow(dst, x, y, 24, void, 0.6)
			disc(dst, x, y, 13, 2.2, light(void, 0.5))
			for _, h := range []float64{-1, 1} {
				hx, hy := head.at(h*12, -36)
				blob(dst, []float64{hx, hy, hx + h*16, hy - 8, hx + h*26, hy - 28, hx + h*22, hy - 42, hx + h*18, hy - 26, hx + h*6, hy - 14}, rgb(0x2a, 0x26, 0x34), rgb(0x08, 0x06, 0x0c), 1.2)
			}
			for i := 0; i < 5; i++ {
				rx, ry := hand.at(0, -26-float64(i)*22)
				glow(dst, rx, ry, 9, void, 0.4+0.3*math.Sin(a.t*3+float64(i)))
			}
		},
	}
}()

var ghoul = func() *body {
	sk := mGhoulSkin
	return &body{
		torso: newPart(sk, -16, 6, -20, -16, -14, -40, 0, -50, 14, -42, 18, -16, 14, 6).with(func(d *ebiten.Image, ox, oy float64) {
			for i := 0; i < 3; i++ {
				curve(d, ox-12, oy-32+float64(i)*8, ox, oy-26+float64(i)*8, ox+12, oy-32+float64(i)*8, 1.4, alpha(colInk, 0.5))
			}
		}),
		tasset: newPart(mLeather.col(rgb(0x4a, 0x3a, 0x30)), -18, -2, 18, -2, 20, 18, 8, 14, 0, 22, -10, 14, -20, 20),
		head: newPart(sk, -14, 4, -18, -8, -12, -22, 4, -26, 18, -16, 22, -2, 14, 8, 0, 10).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, smoothPath(shift(ox, oy, 2, 0, 20, 0, 18, 8, 4, 8)), rgb(0x1a, 0x06, 0x06))
			for i := 0; i < 4; i++ {
				shape(d, shift(ox, oy, 4+float64(i)*4, 0, 7+float64(i)*4, 0, 5.5+float64(i)*4, 5), colBone, colCream, 0)
			}
		}),
		upper: newPart(sk, limb(34, 10, 8)...), fore: newPart(sk, limb(36, 8, 7)...),
		hand:  newPart(mClaw, -5, -2, 5, -2, 7, 10, 3, 20, 0, 12, -3, 20, -7, 10),
		thigh: newPart(sk, limb(30, 11, 9)...), shin: newPart(sk, limb(32, 9, 7)...),
		foot: newPart(sk, -5, -3, 4, -4, 14, 2, 12, 5, -5, 5),
		hipH: 58, torsoL: 48, uaL: 34, faL: 36, thighL: 30, shinL: 32, stride: 30, hunch: 0.65, weaponA: 1.2,
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, ex := range []float64{2, 12} {
				x, y := head.at(ex, -10)
				glowEye(dst, x, y, 3, rgb(0xd0, 0xff, 0x60))
			}
		},
	}
}()

var gargoyle = func() *body {
	st := mStone
	gw := []float64{0, 0, 30, -40, 80, -62, 74, -30, 90, -6, 66, -4, 66, 20, 40, 10, 20, 30}
	wing := newPart(st.col(rgb(0x50, 0x50, 0x5c)), gw...).with(wingBones(gw))
	wingB := newPart(st.col(rgb(0x50, 0x50, 0x5c)), mirror(gw, 0)...).with(wingBones(mirror(gw, 0)))
	return &body{
		torso: newPart(st, -20, 6, -24, -20, -18, -46, 0, -54, 18, -46, 24, -20, 20, 6).with(func(d *ebiten.Image, ox, oy float64) {
			detailLine(d, ox, oy, 1.6, alpha(colOrange, 0.8), -4, -46, 4, -30, -2, -14)
		}),
		head: newPart(st, -14, 4, -16, -12, -8, -24, 8, -24, 16, -12, 18, 2, 8, 10, -8, 10).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, smoothPath(shift(ox, oy, -2, 2, 16, 2, 14, 10, 0, 9)), rgb(0x10, 0x08, 0x06))
			for i := 0; i < 3; i++ {
				shape(d, shift(ox, oy, 1+float64(i)*5, 2, 4+float64(i)*5, 2, 2.5+float64(i)*5, 7), colBone, colCream, 0)
			}
		}),
		upper: newPart(st, limb(30, 13, 11)...), fore: newPart(st, limb(30, 11, 10)...),
		hand:  newPart(mClaw, -6, -2, 6, -2, 8, 10, 3, 20, 0, 12, -3, 20, -8, 10),
		thigh: newPart(st, limb(32, 15, 12)...), shin: newPart(st, limb(32, 12, 10)...),
		foot: newPart(st, -6, -4, 5, -5, 18, 2, 16, 6, -6, 6),
		hipH: 50, torsoL: 54, uaL: 30, faL: 30, thighL: 32, shinL: 32, stride: 40, hunch: 0.25, weaponA: 1.1,
		before: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			flap := math.Sin(a.t*2.2) * 0.18
			wx, wy := torso.at(-6, -44)
			wingB.drawTint(dst, xf{wx - 4, wy, 0.3 + flap, 1}, 0.6)
			wing.draw(dst, xf{wx + 6, wy, -0.3 - flap*0.8, 1})
		},
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, h := range []float64{-6, 6} {
				hx, hy := head.at(h, -20)
				blob(dst, []float64{hx, hy, hx + h*1.4, hy - 10, hx + h*2.6, hy - 26, hx + h*2.2, hy - 34, hx + h*1.6, hy - 20, hx + h*0.4, hy - 8}, rgb(0x4a, 0x48, 0x52), rgb(0x10, 0x10, 0x16), 1.2)
			}
			for _, ex := range []float64{3, 12} {
				x, y := head.at(ex, -11)
				glowEye(dst, x, y, 3, colOrange)
			}
		},
	}
}()

var demon = func() *body {
	sk := mDemonSkin
	lava := rgb(0xff, 0x8a, 0x20)
	wpts := []float64{0, 0, 40, -60, 110, -96, 150, -110, 140, -60, 158, -20, 120, -22, 126, 14, 90, 6, 82, 36, 30, 20}
	wing := newPart(mWing, wpts...).with(wingBones(wpts))
	wingB := newPart(mWing, mirror(wpts, 0)...).with(wingBones(mirror(wpts, 0)))
	return &body{
		torso: newPart(sk, -16, 8, -20, -24, -32, -56, -44, -80, -24, -94, 0, -96, 24, -94, 44, -80, 32, -56, 20, -24, 16, 8).with(func(d *ebiten.Image, ox, oy float64) {
			for _, c := range [][]float64{{-18, -70, -6, -48, -14, -24}, {16, -70, 6, -46, 14, -22}, {-2, -36, 4, -10}} {
				detailLine(d, ox, oy, 2.6, lava, c...)
			}
			curve(d, ox-28, oy-64, ox-12, oy-52, ox, oy-62, 1.6, alpha(colInk, 0.6))
			curve(d, ox+28, oy-64, ox+12, oy-52, ox, oy-62, 1.6, alpha(colInk, 0.6))
		}),
		tasset: newPart(mLeather.col(rgb(0x2a, 0x10, 0x10)), -32, -2, 32, -2, 36, 30, 16, 26, 0, 40, -16, 26, -36, 30),
		head: newPart(sk, -20, 6, -24, -14, -16, -32, 0, -36, 16, -32, 24, -14, 20, 6, 8, 16, -8, 16).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, smoothPath(shift(ox, oy, -12, 4, 12, 4, 9, 12, -9, 12)), rgb(0x20, 0x02, 0x02))
			for i := 0; i < 5; i++ {
				shape(d, shift(ox, oy, -10+float64(i)*4.4, 4, -7+float64(i)*4.4, 4, -8.5+float64(i)*4.4, 10), colBone, colCream, 0)
			}
			fillPath(d, pathOf(shift(ox, oy, -16, -18, -3, -12, -4, -9, -16, -13)), colInk)
			fillPath(d, pathOf(shift(ox, oy, 16, -18, 3, -12, 4, -9, 16, -13)), colInk)
		}),
		upper: newPart(sk, limb(42, 20, 16)...), fore: newPart(sk, limb(42, 16, 14)...),
		hand:  newPart(mClaw, -8, -2, 8, -2, 11, 12, 5, 26, 0, 14, -5, 26, -11, 12),
		thigh: newPart(sk, limb(46, 28, 20)...), shin: newPart(sk, limb(46, 20, 15)...),
		foot: newPart(mClaw, -9, -5, 8, -6, 24, 2, 22, 8, -9, 8),
		hipH: 98, torsoL: 90, uaL: 42, faL: 42, thighL: 46, shinL: 46, stride: 48, hunch: 0.05, weaponA: 1.2,
		before: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			flap := math.Sin(a.t*1.6) * 0.16
			wx, wy := torso.at(-8, -76)
			wingB.drawTint(dst, xf{wx - 6, wy, 0.35 + flap, 1}, 0.6)
			wing.draw(dst, xf{wx + 10, wy, -0.45 - flap, 1})
		},
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, c := range [][2]float64{{-12, -60}, {12, -58}, {0, -24}} {
				x, y := torso.at(c[0], c[1])
				glow(dst, x, y, 26, lava, 0.3+0.15*math.Sin(a.t*3+c[0]))
			}
			for _, h := range []float64{-1, 1} {
				hx, hy := head.at(h*14, -26)
				blob(dst, []float64{hx, hy, hx + h*22, hy - 10, hx + h*36, hy - 40, hx + h*30, hy - 66, hx + h*26, hy - 40, hx + h*12, hy - 20}, rgb(0x2e, 0x26, 0x24), rgb(0x0a, 0x06, 0x06), 1.4)
			}
			for i := 0; i < 5; i++ {
				fx, fy := head.at(-16+float64(i)*8, -34)
				fh := 16 + 10*math.Sin(a.t*7+float64(i)*1.7)
				blob(dst, []float64{fx - 5, fy, fx, fy - fh, fx + 5, fy}, rgb(0xff, 0xe0, 0x80), lava, 0)
				glow(dst, fx, fy-fh*0.5, 14, lava, 0.3)
			}
			for _, ex := range []float64{-9, 9} {
				x, y := head.at(ex, -14)
				glowEye(dst, x, y, 4, rgb(0xff, 0xd0, 0x40))
			}
		},
	}
}()

// ---- the dire wolf: a four-legged rig ----

var (
	wFur  = mFur
	wBody = newPart(wFur, -50, 0, -46, -22, -20, -34, 10, -36, 40, -30, 56, -16, 52, 6, 30, 16, 0, 14, -30, 14)
	wNeck = newPart(wFur, -14, 10, -10, -16, 8, -24, 22, -14, 20, 10)
	wHead = newPart(wFur, -14, 8, -16, -10, -4, -20, 14, -16, 34, -8, 44, -2, 40, 6, 20, 10).with(func(d *ebiten.Image, ox, oy float64) {
		fillPath(d, smoothPath(shift(ox, oy, 39, -6, 46, -3, 43, 2, 38, 0)), rgb(0x06, 0x04, 0x06))
	})
	wJaw   = newPart(wFur, -10, -2, 30, -2, 38, 2, 28, 8, -6, 6)
	wEar   = sharpPart(wFur, -6, 0, 0, -18, 7, 0)
	wThigh = newPart(wFur, limb(28, 22, 15)...)
	wShin  = newPart(wFur, limb(30, 14, 10)...)
	wPaw   = newPart(mClaw, -6, -3, 6, -4, 14, 2, 12, 5, -6, 5)
	wTail  = newPart(wFur, -6, 0, 6, 0, 8, 20, 4, 40, 0, 48, -4, 40, -8, 20)
)

func drawWolf(dst *ebiten.Image, x, y float64, a anim) {
	t := a.t
	br := math.Sin(t * 2.6)
	p := clamp01(a.atk)
	pounce := 0.0
	if p > 0 && p < 1 {
		pounce = math.Sin(math.Pi * clamp01((p-0.25)/0.5))
	}
	dx := keys(p, 0, 0, 0.25, -12, 0.5, 70, 0.8, 30, 1, 0) - 16*a.hurt
	dead := easeOut(a.dead)
	root := xf{x + dx, y - 56 + br*1.2 - pounce*30 + dead*26, -0.08*pounce + 0.1*a.hurt + 0.5*dead, 1}
	shadow(dst, x+dx*0.6, y+1, 80, 12, 0.6)
	tail := root.j(-48, -14, -2.2+math.Sin(t*3)*0.25)
	wTail.draw(dst, tail)
	legs := func(off, phase float64, f float32, front bool) {
		hx, hy := root.at(off, 2)
		fx := hx + 6 + math.Sin(t*2+phase)*2 + pounce*22
		if !front {
			fx = hx - 4 - pounce*18
		}
		a1, a2 := ik(hx, hy, fx, y-5+(-pounce*24), 28, 30, -1)
		if front {
			a1, a2 = ik(hx, hy, fx, y-5+(-pounce*24), 28, 30, 1)
		}
		th := xf{hx, hy, a1, 1}
		sh := xf{0, 0, a2, 1}
		sh.x, sh.y = th.at(0, 28)
		ft := sh.j(0, 30, 0)
		wThigh.drawTint(dst, th, f)
		wShin.drawTint(dst, sh, f)
		wPaw.drawTint(dst, xf{ft.x, ft.y, 0, 1}, f)
	}
	legs(-34, 0, 0.6, false)
	legs(34, 1.5, 0.6, true)
	wBody.draw(dst, root)
	legs(-26, 0.7, 1, false)
	neck := root.j(44, -18, -0.2+br*0.03)
	wNeck.draw(dst, neck)
	head := neck.j(12, -10, 0.15+math.Sin(t*1.3)*0.05-0.2*pounce)
	open := 0.12 + 0.08*math.Sin(t*5) + 0.5*pounce
	wJaw.draw(dst, head.j(4, 6, open))
	wEar.drawTint(dst, head.j(0, -14, -0.3), 0.7)
	wHead.draw(dst, head)
	wEar.draw(dst, head.j(6, -14, -0.1))
	ex, ey := head.at(12, -8)
	glowEye(dst, ex, ey, 3.4, colRed)
	for i := 0; i < 4; i++ {
		tx, ty := head.at(16+float64(i)*6, 6)
		shape(dst, []float64{tx, ty, tx + 3, ty, tx + 1.5, ty + 5}, colBone, colCream, 0)
	}
	legs(40, 2.6, 1, true)
}

// ---- the wraith: floating cloth ----

var (
	wrHood = newPart(mShroud, -20, 10, -24, -14, -14, -36, 4, -44, 18, -32, 24, -8, 20, 10, 0, 6).with(func(d *ebiten.Image, ox, oy float64) {
		fillPath(d, smoothPath(shift(ox, oy, -10, 6, -12, -14, 2, -26, 14, -14, 14, 6)), rgb(0x02, 0x02, 0x06))
	})
	wrRobe = newPart(mShroud, -24, 0, 24, 0, 30, 50, 10, 46, -10, 52, -30, 48)
	wrTail = newPart(mShroud.col(rgb(0x24, 0x28, 0x3a)), -28, 0, 28, 0, 22, 40, 10, 32, 2, 56, -8, 34, -20, 48)
	wrArm  = newPart(mShroud, limb(46, 12, 9)...)
	wrClaw = newPart(mBone.col(rgb(0xc8, 0xd8, 0xd0)), -5, -2, 5, -2, 7, 12, 3, 24, 0, 14, -3, 24, -7, 12)
)

func drawWraith(dst *ebiten.Image, x, y float64, a anim) {
	t := a.t
	spirit := rgb(0x50, 0xd0, 0xd0)
	p := clamp01(a.atk)
	reach := 0.0
	if p > 0 && p < 1 {
		reach = math.Sin(math.Pi * p)
	}
	dx := 60*reach - 14*a.hurt
	dead := easeOut(a.dead)
	fy := math.Sin(t*1.8) * 8
	root := xf{x + dx, y - 110 + fy, 0.05 + 0.1*math.Sin(t*1.3) + 0.2*reach - 0.2*a.hurt, 1}
	shadow(dst, x+dx, y+1, 46*(1-fy/40), 9, 0.4)
	glow(dst, root.x, root.y-10, 110, spirit, 0.22*(1-dead))
	tail := root.j(0, 46, math.Sin(t*2.2)*0.2)
	wrTail.draw(dst, tail)
	wrTail.draw(dst, xf{tail.x, tail.y, tail.a, 1}.j(0, 34, math.Sin(t*2.6+1)*0.3))
	sh := root.j(-12, -2, 0)
	wrArm.drawTint(dst, xf{sh.x, sh.y, -0.4 + math.Sin(t*2)*0.15 - reach*0.9, 1}, 0.6)
	wrRobe.draw(dst, root)
	head := root.j(2, -2, 0.1*math.Sin(t*1.6))
	wrHood.draw(dst, head)
	for _, ex := range []float64{-3, 8} {
		px, py := head.at(ex, -10)
		glowEye(dst, px, py, 3.2, spirit)
	}
	sf := root.j(14, -2, 0)
	arm := xf{sf.x, sf.y, -0.8 + math.Sin(t*2+1)*0.15 - reach*0.9, 1}
	wrArm.draw(dst, arm)
	wrClaw.draw(dst, arm.j(0, 44, 0))
	for i := 0; i < 4; i++ {
		wy := root.y + 60 - math.Mod(t*30+float64(i)*24, 90)
		glow(dst, root.x-20+float64(i)*14, wy, 9, spirit, 0.3)
	}
}

// drawEnemy draws an enemy with its feet at (x,y), authored facing right
// (the battle screen flips it to face the hero).
func drawEnemy(dst *ebiten.Image, shapeID int, x, y float64, a anim) {
	switch shapeID {
	case shGhoul:
		ghoul.draw(dst, x, y, a)
	case shWolf:
		drawWolf(dst, x, y, a)
	case shWraith:
		drawWraith(dst, x, y, a)
	case shCultist:
		cultist.draw(dst, x, y, a)
	case shSkeleton:
		skeleton.draw(dst, x, y, a)
	case shKnight:
		abyssKnight.draw(dst, x, y, a)
	case shGargoyle:
		gargoyle.draw(dst, x, y, a)
	case shDemon:
		demon.draw(dst, x, y, a)
	}
}

// wingBones paints the finger bones of a membrane wing from its root to each outer point.
func wingBones(pts []float64) func(d *ebiten.Image, ox, oy float64) {
	return func(d *ebiten.Image, ox, oy float64) {
		for i := 4; i+1 < len(pts)-2; i += 4 {
			line(d, ox, oy, ox+pts[i], oy+pts[i+1], 2.4, alpha(colInk, 0.55))
		}
	}
}

// the merchant: a hooded peddler with a lantern
var merchantBody = func() *body {
	robe := mRobe.col(rgb(0x4a, 0x34, 0x2a))
	return &body{
		torso: newPart(robe, -18, 6, -22, -18, -18, -42, -6, -52, 8, -52, 18, -42, 22, -18, 18, 6).with(func(d *ebiten.Image, ox, oy float64) {
			detailLine(d, ox, oy, 2.4, rgb(0x2a, 0x1a, 0x10), -14, -20, 14, -28)
			for i := 0; i < 3; i++ {
				disc(d, ox-6+float64(i)*6, oy-23-float64(i)*2.6, 2.4, 2.4, colGold)
			}
		}),
		tasset: newPart(robe, -20, -2, 20, -2, 26, 40, 30, 70, 14, 64, 2, 72, -12, 64, -26, 70, -30, 40),
		head: newPart(robe.col(rgb(0x3a, 0x28, 0x20)), -18, 4, -20, -18, -8, -38, 8, -38, 16, -22, 18, 4).with(func(d *ebiten.Image, ox, oy float64) {
			fillPath(d, smoothPath(shift(ox, oy, -10, 2, -10, -20, 4, -28, 14, -14, 12, 2)), rgb(0x04, 0x02, 0x04))
		}),
		upper: newPart(robe, limb(26, 12, 12)...), fore: newPart(robe, limb(24, 12, 13)...),
		hand: newPart(mSkinDark(), -4, -1, 4, -1, 5, 6, 0, 9, -5, 6),
		weapon: []*part{
			sharpPart(mDarkSteel, -1.5, 0, 1.5, 0, 1.5, 18, -1.5, 18),
			newPart(mGlass.col(rgb(0xff, 0xc0, 0x60)), -9, 18, 9, 18, 11, 40, -11, 40),
		},
		hipH: 72, torsoL: 52, uaL: 26, faL: 24, thighL: 34, shinL: 34, stride: 22, noLegs: true, weaponA: math.Pi,
		before: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			px, py := torso.at(-20, -40)
			pack := []float64{px - 30, py, px - 4, py - 16, px + 8, py + 10, px + 4, py + 60, px - 34, py + 56}
			blob(dst, pack, rgb(0x6a, 0x4a, 0x2a), rgb(0x24, 0x18, 0x0c), 1.4)
		},
		after: func(dst *ebiten.Image, torso, head, hand xf, a anim) {
			for _, ex := range []float64{0, 8} {
				x, y := head.at(ex, -12)
				glowEye(dst, x, y, 2.2, colYellow)
			}
			lx, ly := hand.at(0, -30)
			glow(dst, lx, ly, 70, colOrange, 0.45+0.08*math.Sin(a.t*7))
		},
	}
}()

func mSkinDark() material { return mat(rgb(0x8a, 0x6a, 0x5a), 0.2, 8, 0.5, 0.9) }
