package game

import (
	"image/color"
	"math"
)

// The chefs: who they are and their inked bust portraits. A bust is painted with
// the chin near 0,0 (head centre at 0,-30), in two layers: the bust, and the face
// (eyes, brows, mouth, moustache) in three states: normal, blink and shouting.

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
	skHit    = iota // big hit
	skMulti         // a flurry of hits
	skFrenzy        // the whole crew attacks twice as fast for a while
	skCrit          // taps crit for a while
	skTime          // adds time to the boss timer
	skGold          // monsters drop triple coins for a while
)

type chef struct {
	name, title string
	rarity      int
	dps         float64 // damage per second at level 1
	period      float64 // seconds between throws
	shot        int     // food icon they throw
	glow        color.RGBA
	skill       string
	skillText   string
	skKind      int
	skDmg       float64 // skill damage in seconds of the chef's own damage
	cool        float64 // seconds to charge the skill
	fav         int     // favourite dish: on the table, this chef does double damage
	look        look
}

type look struct {
	skin, hair, jacket, trim color.RGBA
	hat                      int
	hairStyle                int
	stache                   int // 0 none, 1 thin, 2 big, 3 grand curled
	brow                     int // 0 friendly, 1 stern
	freckles, beauty, beard  bool
	robot                    bool
}

const (
	hatToque = iota
	hatTallToque
	hatHeadband
	hatBandana
	hatWitch
	hatSombrero
	hatNinja
	hatNone
)

var (
	skinLight = rgb(0xff, 0xd8, 0xb8)
	skinTan   = rgb(0xe8, 0xb0, 0x80)
	skinBrown = rgb(0xb8, 0x7a, 0x50)
)

var chefs = []chef{
	{name: "Pip", title: "Rookie Cook", rarity: rCommon, dps: 1, period: 1.0, shot: inEgg, glow: rgb(0xff, 0xf0, 0xb0),
		skill: "Egg Barrage", skillText: "Hurls a whole carton of eggs", skKind: skHit, skDmg: 30, cool: 16, fav: dToast,
		look: look{skin: skinLight, hair: rgb(0xa8, 0x5a, 0x2a), jacket: rgb(0xf8, 0xf8, 0xff), trim: rgb(0xf0, 0x40, 0x40), hat: hatToque, hairStyle: 0, freckles: true}},
	{name: "Tony", title: "Pizza Ninja", rarity: rCommon, dps: 0.9, period: 0.6, shot: fiDish + dPizza, glow: rgb(0xff, 0x80, 0x40),
		skill: "Slice Storm", skillText: "The whole crew throws twice as fast", skKind: skFrenzy, skDmg: 8, cool: 22, fav: dPizza,
		look: look{skin: skinTan, hair: rgb(0x2a, 0x22, 0x30), jacket: rgb(0x3a, 0x34, 0x48), trim: rgb(0xf0, 0x40, 0x30), hat: hatNinja, brow: 1}},
	{name: "Kenji", title: "Sushi Master", rarity: rRare, dps: 2.6, period: 0.9, shot: fiKnife, glow: rgb(0xd0, 0xe8, 0xff),
		skill: "Thousand Cuts", skillText: "Your taps crit for 8 seconds", skKind: skCrit, skDmg: 12, cool: 20, fav: dOmelette,
		look: look{skin: skinLight, hair: rgb(0x22, 0x1c, 0x28), jacket: rgb(0x2c, 0x48, 0x9a), trim: colWhite, hat: hatHeadband, hairStyle: 2, stache: 1, brow: 1}},
	{name: "Bertha", title: "Grill Master", rarity: rRare, dps: 3.2, period: 1.4, shot: inMeat, glow: rgb(0xff, 0xa0, 0x50),
		skill: "Flame Grill", skillText: "A blazing hit, triple damage to bosses", skKind: skHit, skDmg: 45, cool: 24, fav: dBurger,
		look: look{skin: skinLight, hair: rgb(0xe0, 0x50, 0x20), jacket: rgb(0xf8, 0xf8, 0xff), trim: rgb(0x8a, 0x50, 0x30), hat: hatBandana, hairStyle: 3, freckles: true}},
	{name: "Madame Creme", title: "Pastry Witch", rarity: rEpic, dps: 6.5, period: 1.2, shot: fiDish + dCake, glow: rgb(0xff, 0x9a, 0xe0),
		skill: "Sugar Hex", skillText: "A flurry of cursed cupcakes", skKind: skMulti, skDmg: 26, cool: 20, fav: dCake,
		look: look{skin: rgb(0xf8, 0xe0, 0xe8), hair: rgb(0x9a, 0x5a, 0xe0), jacket: rgb(0xff, 0x9a, 0xc8), trim: rgb(0xff, 0xf0, 0xf8), hat: hatWitch, hairStyle: 4, beauty: true}},
	{name: "El Picante", title: "Taco Bandit", rarity: rEpic, dps: 6, period: 1.0, shot: fiDish + dTaco, glow: rgb(0xff, 0xd0, 0x40),
		skill: "Salsa Heist", skillText: "Monsters drop triple coins for 15 seconds", skKind: skGold, skDmg: 15, cool: 28, fav: dTaco,
		look: look{skin: skinBrown, hair: rgb(0x22, 0x18, 0x18), jacket: rgb(0x3a, 0xa0, 0x58), trim: rgb(0xf0, 0x40, 0x40), hat: hatSombrero, stache: 2}},
	{name: "Scoop-3000", title: "Ice Cream Robot", rarity: rEpic, dps: 7, period: 1.1, shot: fiDish + dSundae, glow: rgb(0x80, 0xf0, 0xff),
		skill: "Brain Freeze", skillText: "Freezes the boss timer: +6 seconds", skKind: skTime, skDmg: 20, cool: 26, fav: dSundae,
		look: look{skin: rgb(0xb8, 0xd0, 0xe0), jacket: rgb(0x40, 0xc0, 0xc0), trim: colPink, robot: true}},
	{name: "Chef Supreme", title: "Legendary Head Chef", rarity: rLegend, dps: 18, period: 1.3, shot: fiLadle, glow: rgb(0xff, 0xd0, 0x40),
		skill: "Kitchen Nightmare", skillText: "A thunderous ladle smash", skKind: skHit, skDmg: 70, cool: 24, fav: dSoup,
		look: look{skin: skinLight, hair: rgb(0xf4, 0xf0, 0xf0), jacket: rgb(0xf8, 0xf8, 0xff), trim: rgb(0xff, 0xc4, 0x30), hat: hatTallToque, hairStyle: 1, stache: 3, beard: true}},
}

// ---- painting ----

func paintChef(c *canvas, id int) {
	L := chefs[id].look
	ow := 2.4
	if L.robot {
		paintRobot(c, L)
		return
	}
	// shoulders: the chef's jacket
	jk := blob(-62, 56, -54, 18, -26, 4, 0, 6, 26, 4, 54, 18, 62, 56)
	c.part(jk, mat{base: L.jacket, rim: 0.3}, ow)
	if L.jacket.R > 0xf0 && L.jacket.G > 0xf0 { // double-breasted buttons
		for _, b := range [][2]float64{{-12, 26}, {12, 26}, {-14, 44}, {14, 44}} {
			c.part(circ(b[0], b[1], 3.4), mat{base: rgb(0xd8, 0xd8, 0xe8), gloss: 0.6}, 1.2)
		}
	}
	if id == 7 { // epaulettes
		for _, s := range []float64{-1, 1} {
			c.part(ell(s*44, 16, 16, 7).rot(s*44, 16, s*0.4), mat{base: L.trim, gloss: 1}, ow)
		}
	}
	// neck and scarf
	c.part(rrect(-11, -8, 22, 22, 6), mat{base: L.skin}, ow)
	scarf := ell(0, 10, 24, 9)
	if L.hat == hatSombrero || L.hat == hatNinja || id == 0 {
		c.part(scarf, mat{base: L.trim, rim: 0.3}, ow)
		if id == 0 {
			c.part(poly(4, 12, 16, 30, 6, 32), mat{base: L.trim}, ow)
		}
	}
	// hair behind the head
	switch L.hairStyle {
	case 3: // big curly hair
		h := circ(-26, -26, 14).or(circ(26, -26, 14)).or(circ(-22, -6, 12)).or(circ(22, -6, 12)).or(circ(0, -48, 22)).fuzz(1.5, 0.25)
		c.part(h, mat{base: L.hair, rim: 0.3}, ow)
	case 4: // long witch hair
		c.part(blob(-30, -44, -38, -10, -36, 10, -26, 14, -18, -10, 18, -10, 26, 14, 36, 10, 38, -10, 30, -44), mat{base: L.hair, rim: 0.3}, ow)
	}
	// ears and head
	for _, s := range []float64{-1, 1} {
		c.part(circ(s*29, -26, 7), mat{base: L.skin}, ow)
	}
	head := ell(0, -30, 30, 32)
	if L.beard {
		c.part(blob(-28, -24, -24, 4, -10, 18, 10, 18, 24, 4, 28, -24, 0, -14).fuzz(1.4, 0.3), mat{base: L.hair, rim: 0.2}, ow)
	}
	c.part(head, mat{base: L.skin, rim: 0.3}, ow)
	c.volOn(ell(0, -16, 6, 5), head, mat{base: shade(L.skin, 0.9)})
	c.fill(arc(0, -16, 6, 5, 0.2, 2.9, 1.1), alpha(outlineOf(L.skin), 0.7))
	blush := alpha(rgb(0xff, 0x70, 0x80), 0.45)
	if L.skin == skinBrown {
		blush = alpha(rgb(0xd8, 0x40, 0x40), 0.35)
	}
	c.soft(ell(-18, -14, 6, 3.5), blush, 3)
	c.soft(ell(18, -14, 6, 3.5), blush, 3)
	if L.freckles {
		for _, f := range [][2]float64{{-20, -18}, {-16, -15}, {-22, -14}, {20, -18}, {16, -15}, {22, -14}} {
			c.fill(circ(f[0], f[1], 1.1), alpha(rgb(0xb0, 0x60, 0x30), 0.8))
		}
	}
	if L.beauty {
		c.fill(circ(14, -6, 1.6), rgb(0x3a, 0x1a, 0x30))
	}
	// hair on top
	switch L.hairStyle {
	case 0: // a messy tuft
		c.part(blob(-26, -40, -18, -54, -2, -56, 14, -58, 26, -44, 12, -48, 0, -44, -14, -46), mat{base: L.hair}, ow)
	case 1: // grey sideburns
		for _, s := range []float64{-1, 1} {
			c.part(ell(s*26, -38, 7, 12).fuzz(1, 0.4), mat{base: L.hair}, ow)
		}
	case 2: // topknot
		c.part(blob(-28, -36, -24, -56, 0, -62, 24, -56, 28, -36, 0, -46), mat{base: L.hair}, ow)
		c.part(circ(0, -70, 9), mat{base: L.hair, gloss: 0.5}, ow)
	case 4:
		c.part(blob(-30, -40, -20, -58, 10, -60, 30, -44, 18, -40, 4, -48, -10, -42), mat{base: L.hair}, ow)
	}
	// hats
	white := mat{base: rgb(0xfc, 0xfc, 0xff), dark: rgb(0xc8, 0xc8, 0xe0), rim: 0.3}
	switch L.hat {
	case hatToque, hatTallToque:
		tall := 0.0
		if L.hat == hatTallToque {
			tall = 26
		}
		puff := circ(-20, -78-tall, 17).or(circ(20, -78-tall, 17)).or(circ(0, -90-tall, 20)).or(rrect(-26, -82-tall, 52, 30+tall, 6))
		c.part(puff, white, ow)
		band := rrect(-29, -58, 58, 16, 5)
		c.part(band, white, ow)
		if L.hat == hatTallToque {
			c.fill(rrect(-29, -52, 58, 4, 1), L.trim)
			for _, x := range []float64{-14, 0, 14} {
				c.fill(line(1, 1, x, -64, x, -100), alpha(rgb(0xc8, 0xc8, 0xe0), 0.9))
			}
		}
	case hatHeadband:
		c.part(rrect(-31, -54, 62, 12, 5), white, ow)
		c.part(circ(0, -48, 4.5), mat{base: rgb(0xe8, 0x30, 0x30)}, 1.2)
		c.part(poly(28, -50, 44, -58, 42, -46).or(poly(28, -46, 46, -42, 40, -36)), white, ow)
	case hatBandana:
		bd := ell(0, -46, 32, 22).and(rrect(-40, -80, 80, 36, 0))
		c.part(bd, mat{base: rgb(0xe8, 0x30, 0x30), rim: 0.3}, ow)
		for _, d := range [][2]float64{{-16, -54}, {0, -60}, {16, -54}, {-6, -48}, {10, -48}} {
			c.fill(circ(d[0], d[1], 2.6), colWhite)
		}
		c.part(poly(26, -48, 44, -56, 40, -40), mat{base: rgb(0xe8, 0x30, 0x30)}, ow)
	case hatWitch:
		brim := ell(0, -52, 46, 10)
		c.part(brim, mat{base: rgb(0x5a, 0x2a, 0x8a), rim: 0.3}, ow)
		cone := poly(-26, -54, 26, -54, 14, -84, 28, -112, 2, -92)
		c.part(cone, mat{base: rgb(0x6a, 0x34, 0x9e), rim: 0.3}, ow)
		c.part(rrect(-26, -62, 50, 8, 3), mat{base: colPink}, ow*0.8)
		c.part(circ(28, -114, 6), mat{base: rgb(0xe8, 0x20, 0x40), gloss: 1}, ow*0.8)
	case hatSombrero:
		brim := ell(0, -50, 64, 14)
		c.part(brim, mat{base: rgb(0xf0, 0xc8, 0x60), rim: 0.3, tex: 0.1, texF: 0.3}, ow)
		crown := ell(0, -62, 26, 26).and(rrect(-30, -100, 60, 46, 0))
		c.part(crown, mat{base: rgb(0xf4, 0xd0, 0x70), tex: 0.1, texF: 0.3}, ow)
		c.part(rrect(-27, -62, 54, 7, 3), mat{base: rgb(0xe8, 0x30, 0x30)}, ow*0.7)
		c.fill(ell(0, -50, 54, 8).ring(2.2), rgb(0x3a, 0xa0, 0x58))
	case hatNinja:
		hood := ell(0, -36, 33, 34).minus(rrect(-26, -40, 52, 18, 7))
		c.part(hood, mat{base: L.jacket, rim: 0.3}, ow)
		c.part(rrect(-32, -50, 64, 9, 4), mat{base: L.trim}, ow)
		c.part(poly(28, -48, 50, -56, 46, -40).or(poly(28, -44, 50, -36, 42, -30)), mat{base: L.trim}, ow)
	}
}

func paintRobot(c *canvas, L look) {
	ow := 2.4
	metal := mat{base: L.skin, gloss: 1.1, shine: 30, rim: 0.4}
	c.part(blob(-58, 56, -50, 20, -24, 8, 24, 8, 50, 20, 58, 56), mat{base: L.jacket, gloss: 0.8}, ow)
	c.part(rrect(-16, 30, 32, 18, 5), mat{base: rgb(0x2a, 0x3a, 0x50)}, ow)
	for i, col := range []color.RGBA{colRed, colGold, colGreen} {
		c.part(circ(-8+float64(i)*8, 39, 2.6), mat{base: col, gloss: 1}, 1)
	}
	c.part(rrect(-8, -6, 16, 16, 3), metal, ow)
	for _, s := range []float64{-1, 1} {
		c.part(circ(s*34, -30, 8), mat{base: L.jacket, gloss: 1}, ow)
	}
	c.part(rrect(-34, -66, 68, 62, 18), metal, ow)
	c.part(rrect(-26, -56, 52, 40, 10), mat{base: rgb(0x1e, 0x2a, 0x40), gloss: 0.6, flat: 0.5}, ow*0.6)
	// an ice cream antenna
	c.part(line(2.2, 2.2, 0, -66, 0, -80), metal, ow*0.6)
	c.part(poly(-8, -82, 8, -82, 0, -64).or(poly(-8, -82, 8, -82, 0, -64)), mat{base: rgb(0xf0, 0xb8, 0x60), tex: 0.2, texF: 0.5}, ow)
	c.part(circ(0, -88, 9), mat{base: colPink, gloss: 0.6}, ow)
	c.part(circ(2, -98, 3), mat{base: colRed, gloss: 1}, ow*0.6)
}

// paintChefFace paints the face layer: state 0 normal, 1 blink, 2 shouting.
func paintChefFace(c *canvas, id, state int) {
	L := chefs[id].look
	out := rgb(0x1a, 0x0e, 0x20)
	if L.robot {
		g := rgb(0x70, 0xf0, 0xff)
		for _, s := range []float64{-1, 1} {
			x := s * 12
			switch state {
			case 1:
				c.fill(arc(x, -34, 7, 5, math.Pi*1.1, math.Pi*1.9, 2.2), g)
			default:
				c.glowIn(ell(x, -38, 5, 7), g, 6, 0.5)
				c.fill(ell(x, -38, 5, 7), g)
			}
		}
		if state == 2 {
			c.fill(rrect(-10, -26, 20, 6, 3), g)
		} else {
			c.fill(arc(0, -28, 8, 4, 0.3, 2.8, 1.8), g)
		}
		return
	}
	ninja := L.hat == hatNinja
	for _, s := range []float64{-1, 1} {
		x, y, r := s*12, -32.0, 8.0
		paintEye(c, x, y, r, 0.3, state, L.skin)
		bw := 2.2
		if L.brow == 1 || state == 2 {
			c.fill(line(bw, bw*0.8, x-s*9, y-12, x+s*7, y-8), shade(L.hair, 0.8))
		} else {
			c.fill(arc(x, y-6, 8, 6, math.Pi*1.2, math.Pi*1.8, bw*0.7), shade(L.hair, 0.8))
		}
	}
	if ninja { // the mask covers the mouth
		c.part(ell(0, -8, 27, 14).and(rrect(-40, -16, 80, 40, 0)), mat{base: shade(L.jacket, 1.1)}, 2.4)
		return
	}
	switch state {
	case 2:
		m := ell(0, -2, 9, 8).and(rrect(-12, -6, 24, 20, 0))
		c.fill(m.grow(2), out)
		c.fill(m, rgb(0x6a, 0x1c, 0x2a))
		c.fill(ell(0, 4, 5, 3).and(m), rgb(0xff, 0x7a, 0x8a))
		c.fill(rrect(-6, -6, 12, 3.5, 1).and(m), colWhite)
	default:
		c.fill(arc(0, -8, 10, 7, math.Pi*0.18, math.Pi*0.82, 1.9), out)
		if !L.beard {
			c.fill(ell(0, -1.5, 5, 2).and(rrect(-6, -2, 12, 4, 0)), alpha(rgb(0xff, 0x7a, 0x8a), 0.8))
		}
	}
	hc := mat{base: L.hair, rim: 0.2}
	switch L.stache {
	case 1:
		for _, s := range []float64{-1, 1} {
			c.part(line(1.6, 1, s*2, -10, s*10, -9, s*16, -2), hc, 1.6)
		}
	case 2:
		st := blob(-22, -4, -14, -12, 0, -10, 14, -12, 22, -4, 10, -6, 0, -3, -10, -6)
		c.part(st, hc, 2.4)
	case 3:
		for _, s := range []float64{-1, 1} {
			c.part(line(4, 1.5, s*2, -9, s*14, -12, s*24, -10, s*28, -16, s*24, -20), hc, 2.2)
		}
	}
}

// paintEye paints one cartoon eye looking dir (-1..1). state: 0 open, 1 blink,
// 2 determined, 3 hurt. skin covers the eye when closed.
func paintEye(c *canvas, x, y, r, dir float64, state int, skin color.RGBA) {
	out := rgb(0x1a, 0x0e, 0x20)
	switch state {
	case 1:
		c.fill(circ(x, y, r+1), skin)
		c.fill(arc(x, y-r*0.15, r*0.8, r*0.55, math.Pi*0.1, math.Pi*0.9, r*0.15), out)
		return
	case 3:
		c.fill(circ(x, y, r+1), skin)
		k := r * 0.7
		c.fill(line(r*0.15, r*0.15, x-k*dir, y-k*0.7, x+k*0.6*dir, y, x-k*dir, y+k*0.7), out)
		return
	}
	c.fill(ell(x, y, r+2.4, r*1.12+2.4), out)
	c.vol(ell(x, y, r, r*1.12), mat{base: colWhite, dark: rgb(0xd0, 0xcc, 0xe8)})
	px, py := x+dir*r*0.3, y+r*0.1
	pr := r * 0.56
	c.fill(circ(px, py, pr), rgb(0x1a, 0x10, 0x24))
	c.fill(circ(px-pr*0.3, py-pr*0.35, pr*0.38), colWhite)
}

// ---- sprites ----

type chefSprite struct {
	body *pimg
	face [3]*pimg
}

// chefArt returns the painted layers of chef id at scale z.
func chefArt(id int, z float64) *chefSprite {
	key := "chef" + itoa(id) + "@" + num(z*100)
	s := &chefSprite{}
	s.body = paintArt(key, -72, -125, 72, 60, z, func(c *canvas) { paintChef(c, id) })
	for st := 0; st < 3; st++ {
		st := st
		s.face[st] = paintArt(key+"f"+itoa(st), -34, -50, 34, 12, z, func(c *canvas) { paintChefFace(c, id, st) })
	}
	return s
}
