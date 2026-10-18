package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Food: the ingredients monsters drop, the recipes the kitchen cooks from them,
// the buffs dishes give in the fight and the combos between dishes.

const (
	inFlour = iota
	inTomato
	inCheese
	inVeggie
	inMeat
	inSugar
	inEgg
	inSpice
	numIngredients
)

var ingName = []string{"Flour", "Tomato", "Cheese", "Veggies", "Meat", "Sugar", "Egg", "Spice"}

// Dish effects in the fight.
const (
	efChef     = iota // chef damage %
	efTap             // tap damage %
	efCrit            // crit chance (percentage points)
	efCoins           // coins %
	efBossTime        // extra seconds on boss timers
	efSkill           // skill charge speed %
	efDrops           // ingredient drops %
	efBossDmg         // damage to bosses %
	efAll             // all damage %
	numEffects
)

var effectFmt = []string{"+%d%% chef damage", "+%d%% tap damage", "+%d%% crit chance", "+%d%% coins", "+%ds boss timer", "+%d%% skill charge", "+%d%% ingredients", "+%d%% boss damage", "+%d%% all damage"}

const (
	stCounter = iota
	stStove
	stOven
	stGrill
	stFreezer
	numStations
)

var stationName = []string{"Counter", "Stove", "Oven", "Grill", "Freezer"}
var stationCost = []float64{0, 150, 1500, 12000, 90000}

const (
	coStarter = iota
	coMain
	coDessert
)

var courseName = []string{"Starter", "Main", "Dessert"}

type recipe struct {
	name    string
	course  int
	station int
	need    [numIngredients]int
	secs    float64 // cooking time at station level 1
	eff     int
	val     float64
}

func needs(pairs ...int) (n [numIngredients]int) {
	for i := 0; i+1 < len(pairs); i += 2 {
		n[pairs[i]] = pairs[i+1]
	}
	return
}

const (
	dToast = iota
	dSalad
	dSoup
	dOmelette
	dPizza
	dBurger
	dTaco
	dChili
	dPancakes
	dCake
	dSundae
)

var recipes = []recipe{
	{"Buttered Toast", coStarter, stCounter, needs(inFlour, 2), 12, efTap, 25},
	{"Garden Salad", coStarter, stCounter, needs(inVeggie, 2, inTomato, 1), 16, efBossTime, 4},
	{"Tomato Soup", coStarter, stStove, needs(inTomato, 3, inVeggie, 1), 25, efChef, 30},
	{"Cheesy Omelette", coMain, stStove, needs(inEgg, 2, inCheese, 1), 25, efCrit, 6},
	{"Pizza", coMain, stOven, needs(inFlour, 2, inTomato, 2, inCheese, 2), 50, efChef, 70},
	{"Big Burger", coMain, stGrill, needs(inFlour, 1, inMeat, 2, inVeggie, 1, inCheese, 1), 50, efTap, 90},
	{"Spicy Taco", coMain, stGrill, needs(inFlour, 1, inMeat, 1, inSpice, 2), 40, efCoins, 60},
	{"Fire Chili", coMain, stStove, needs(inMeat, 2, inTomato, 2, inSpice, 2), 55, efBossDmg, 80},
	{"Pancake Stack", coDessert, stStove, needs(inFlour, 2, inEgg, 2, inSugar, 1), 35, efSkill, 40},
	{"Choco Cake", coDessert, stOven, needs(inFlour, 3, inEgg, 2, inSugar, 3), 80, efAll, 40},
	{"Sundae", coDessert, stFreezer, needs(inSugar, 3, inEgg, 1), 45, efDrops, 100},
}

// A combo is a set of dishes on the table at once that adds a bonus effect.
type combo struct {
	name   string
	dishes []int
	eff    int
	val    float64
}

var combos = []combo{
	{"Breakfast Club", []int{dOmelette, dPancakes}, efTap, 60},
	{"Fiesta!", []int{dTaco, dChili}, efCoins, 100},
	{"Italian Night", []int{dPizza, dSalad}, efChef, 60},
	{"Diner Deluxe", []int{dBurger, dSundae}, efBossTime, 5},
	{"Bakery Rush", []int{dToast, dCake}, efSkill, 50},
	{"Soup & Sandwich", []int{dSoup, dToast}, efCrit, 5},
}

// fullCourse: a starter, a main and a dessert together make every dish 50% stronger.
const fullCourse = 1.5

// ---- icons ----

// Food icon kinds: ingredients 0..7, dishes 100+, extra props 200+.
const (
	fiDish  = 100
	fiKnife = 200 + iota
	fiLadle
	fiBox
	fiGoldBox
	fiHeart
	fiPlate
	fiFlame
)

// foodIcon returns a painted food icon size units across, centred.
func foodIcon(kind int, size float64) *pimg {
	r := size / 2
	return paintArt("fi"+itoa(kind)+"@"+num(size*10), -r-3, -r-3, r+3, r+3, 1, func(c *canvas) {
		paintFood(c, kind, size/40)
	})
}

func drawFood(dst *ebiten.Image, kind int, x, y, size float64) {
	foodIcon(kind, size).draw(dst, x, y, 1, cs1)
}

var (
	colBread  = rgb(0xf0, 0xb8, 0x60)
	colCrust  = rgb(0xb8, 0x6a, 0x28)
	colTomato = rgb(0xf0, 0x3c, 0x2c)
	colCheese = rgb(0xff, 0xd0, 0x3a)
	colLeaf   = rgb(0x5c, 0xc0, 0x38)
	colMeat   = rgb(0xa8, 0x5a, 0x34)
)

func paintFood(c *canvas, kind int, u float64) {
	ow := 2.0 * u
	P := func(s sh, col color.RGBA) { c.part(s, mat{base: col, gloss: 0.6, rim: 0.4}, ow) }
	switch kind {
	case inFlour:
		sack := blob(-8*u, -12*u, 8*u, -12*u, 15*u, 2*u, 14*u, 16*u, -14*u, 16*u, -15*u, 2*u)
		P(sack, rgb(0xf4, 0xec, 0xdc))
		c.part(ell(0, -14*u, 9*u, 4*u), mat{base: rgb(0xd8, 0xc8, 0xb0)}, ow)
		for i := -1; i <= 1; i++ {
			x := float64(i) * 4 * u
			c.part(ell(x, 4*u-math.Abs(float64(i))*2*u, 2.2*u, 4.5*u).rot(x, 4*u, float64(i)*0.4), mat{base: rgb(0xf0, 0xc0, 0x40)}, ow*0.5)
		}
	case inTomato:
		P(ell(0, 2*u, 16*u, 14*u), colTomato)
		c.part(poly(-9*u, -12*u, 0, -9*u, 9*u, -12*u, 4*u, -7*u, 0, -16*u, -4*u, -7*u), mat{base: colLeaf}, ow*0.7)
	case inCheese:
		w := poly(-16*u, 10*u, -16*u, -2*u, 14*u, -12*u, 16*u, 10*u)
		P(w, colCheese)
		for _, h := range [][3]float64{{-6, 3, 3}, {6, 1, 2.4}, {2, -6, 1.8}} {
			c.fill(circ(h[0]*u, h[1]*u, h[2]*u), rgb(0xd8, 0x98, 0x20))
		}
	case inVeggie:
		c.part(line(4*u, 3*u, 0, 16*u, 0, 2*u), mat{base: rgb(0x9a, 0xd8, 0x5a)}, ow)
		fl := circ(-8*u, -2*u, 8*u).or(circ(8*u, -2*u, 8*u)).or(circ(0, -9*u, 9*u)).or(circ(0, 2*u, 7*u))
		P(fl, rgb(0x3a, 0xa0, 0x38))
	case inMeat:
		P(ell(4*u, -4*u, 13*u, 11*u).rot(4*u, -4*u, -0.6), colMeat)
		c.part(line(3*u, 3*u, -6*u, 6*u, -13*u, 13*u), mat{base: rgb(0xf8, 0xf0, 0xe0)}, ow)
		c.part(circ(-15*u, 11*u, 3.5*u).or(circ(-11*u, 15*u, 3.5*u)), mat{base: rgb(0xf8, 0xf0, 0xe0)}, ow)
	case inSugar:
		P(poly(-18*u, -6*u, -10*u, 0, -18*u, 6*u).or(poly(18*u, -6*u, 10*u, 0, 18*u, 6*u)), rgb(0xff, 0xb0, 0xd8))
		P(circ(0, 0, 11*u), colPink)
		c.fill(arc(0, 0, 6*u, 6*u, -2.5, -0.5, 1.6*u), alpha(colWhite, 0.7))
	case inEgg:
		P(blob(0, -16*u, 10*u, -8*u, 12*u, 5*u, 0, 15*u, -12*u, 5*u, -10*u, -8*u), rgb(0xff, 0xf6, 0xe8))
	case inSpice:
		ch := line(6*u, 1.5*u, -8*u, -8*u, 4*u, -2*u, 12*u, 12*u)
		P(ch, rgb(0xe8, 0x28, 0x20))
		c.part(line(2*u, 2*u, -8*u, -8*u, -12*u, -14*u), mat{base: colLeaf}, ow*0.7)
	case fiDish + dToast:
		sl := rrect(-14*u, -10*u, 28*u, 26*u, 4*u).or(circ(-7*u, -10*u, 8*u)).or(circ(7*u, -10*u, 8*u))
		P(sl, colCrust)
		c.volOn(sl.grow(-3*u), sl, mat{base: colBread})
		c.part(rrect(-6*u, -4*u, 12*u, 8*u, 2*u), mat{base: rgb(0xff, 0xe8, 0x80), gloss: 0.8}, ow*0.6)
	case fiDish + dSalad, fiDish + dSoup, fiDish + dChili:
		bowl := ell(0, 2*u, 18*u, 14*u).and(rrect(-20*u, 2*u, 40*u, 16*u, 0))
		top := ell(0, 2*u, 18*u, 5*u)
		switch kind - fiDish {
		case dSalad:
			for _, l := range [][3]float64{{-8, -2, 7}, {6, -4, 7}, {0, -7, 6}} {
				c.part(circ(l[0]*u, l[1]*u, l[2]*u), mat{base: colLeaf}, ow)
			}
			c.part(circ(-2*u, -1*u, 3*u), mat{base: colTomato, gloss: 0.8}, ow*0.6)
		case dChili:
			for _, b := range [][2]float64{{-6, -1}, {4, -3}, {9, 0}} {
				c.part(ell(b[0]*u, b[1]*u, 3*u, 2*u), mat{base: rgb(0x8a, 0x2a, 0x20)}, ow*0.5)
			}
		}
		P(bowl, rgb(0xf0, 0xf0, 0xff))
		if kind-fiDish != dSalad {
			col := colTomato
			if kind-fiDish == dChili {
				col = rgb(0xb8, 0x3a, 0x20)
			}
			c.part(top, mat{base: col, flat: 0.4, gloss: 0.5}, ow)
			c.fill(arc(-2*u, -10*u, 3*u, 5*u, 2, 4.3, 1.2*u).or(arc(4*u, -12*u, 3*u, 5*u, 2, 4.3, 1.2*u)), alpha(colWhite, 0.6))
		}
	case fiDish + dOmelette:
		P(ell(0, 4*u, 18*u, 10*u).and(rrect(-20*u, -6*u, 40*u, 20*u, 0)), rgb(0xff, 0xd8, 0x40))
		c.fill(arc(0, 4*u, 14*u, 6*u, 3.3, 6.1, 1.4*u), rgb(0xe8, 0xa8, 0x20))
		c.part(circ(-4*u, -2*u, 2.5*u).or(circ(5*u, 0, 2*u)), mat{base: colLeaf}, ow*0.5)
	case fiDish + dPizza:
		P(circ(0, 0, 18*u), colCrust)
		c.volOn(circ(0, 0, 14.5*u), circ(0, 0, 18*u), mat{base: colCheese, gloss: 0.4})
		for _, p := range [][2]float64{{-6, -6}, {6, -4}, {0, 6}, {-8, 5}, {8, 7}} {
			c.part(circ(p[0]*u, p[1]*u, 3.2*u), mat{base: rgb(0xd8, 0x30, 0x28), gloss: 0.6}, ow*0.5)
		}
	case fiDish + dBurger:
		P(ell(0, 10*u, 16*u, 6*u), colBread)
		P(rrect(-17*u, 1*u, 34*u, 7*u, 3.5*u), rgb(0x6a, 0x34, 0x1c))
		c.part(poly(-17*u, 0, 17*u, 0, 10*u, 5*u, -10*u, 5*u), mat{base: colCheese}, ow*0.7)
		c.part(rrect(-18*u, -3*u, 36*u, 4*u, 2*u).fuzz(1*u, 0.6/u), mat{base: colLeaf}, ow*0.7)
		top := ell(0, -4*u, 17*u, 13*u).and(rrect(-20*u, -20*u, 40*u, 16*u, 0))
		P(top, colBread)
		for _, s := range [][2]float64{{-6, -11}, {2, -13}, {8, -9}, {-1, -8}} {
			c.fill(ell(s[0]*u, s[1]*u, 1.6*u, 1*u), rgb(0xff, 0xf6, 0xe0))
		}
	case fiDish + dTaco:
		shell := ell(0, 6*u, 18*u, 14*u).and(rrect(-20*u, -2*u, 40*u, 24*u, 0))
		c.part(circ(-8*u, -3*u, 6*u).or(circ(1*u, -5*u, 6*u)).or(circ(9*u, -2*u, 5*u)), mat{base: colLeaf}, ow)
		c.part(circ(-3*u, -1*u, 3*u).or(circ(6*u, 0, 3*u)), mat{base: colTomato}, ow*0.5)
		P(shell, rgb(0xf8, 0xc8, 0x50))
	case fiDish + dPancakes:
		for i := 0; i < 3; i++ {
			y := 10*u - float64(i)*7*u
			P(ell(0, y, 17*u, 5*u), rgb(0xe8, 0xa8, 0x50))
		}
		c.part(ell(0, -8*u, 12*u, 3.5*u).or(capsule(8*u, -7*u, 9*u, 2*u, 2*u, 2.5*u)), mat{base: rgb(0x9a, 0x50, 0x20), gloss: 1}, ow*0.6)
		c.part(rrect(-3*u, -14*u, 6*u, 5*u, 1*u), mat{base: rgb(0xff, 0xf0, 0x90)}, ow*0.5)
	case fiDish + dCake:
		sl := poly(-16*u, 14*u, -16*u, -4*u, 16*u, -14*u, 16*u, 14*u)
		P(sl, rgb(0x7a, 0x40, 0x24))
		c.fill(rrect(-16*u, 2*u, 32*u, 3*u, 0).and(sl), rgb(0xff, 0xe0, 0xf0))
		c.part(poly(-17*u, -4*u, 17*u, -15*u, 17*u, -10*u, -17*u, 1*u), mat{base: colPink}, ow*0.6)
		c.part(circ(8*u, -17*u, 4*u), mat{base: colTomato, gloss: 1}, ow*0.6)
	case fiDish + dSundae:
		c.part(poly(-12*u, -2*u, 12*u, -2*u, 4*u, 12*u, -4*u, 12*u).or(rrect(-2*u, 10*u, 4*u, 6*u, 1*u)).or(rrect(-9*u, 15*u, 18*u, 4*u, 2*u)), mat{base: rgb(0xd0, 0xf0, 0xff), gloss: 1}, ow)
		P(circ(-6*u, -6*u, 7*u), rgb(0xff, 0xf4, 0xe0))
		P(circ(6*u, -6*u, 7*u), colPink)
		P(circ(0, -12*u, 7*u), rgb(0x8a, 0x50, 0x30))
		c.part(circ(1*u, -20*u, 3*u), mat{base: colTomato, gloss: 1}, ow*0.6)
	case fiKnife:
		c.part(poly(-4*u, -18*u, 5*u, -16*u, 4*u, 6*u, -4*u, 6*u), mat{base: rgb(0xe0, 0xe8, 0xf8), gloss: 1.2, shine: 30}, ow)
		c.part(rrect(-4*u, 6*u, 8*u, 13*u, 3*u), mat{base: rgb(0x5a, 0x30, 0x20)}, ow)
	case fiLadle:
		c.part(line(2.4*u, 2.4*u, 10*u, -18*u, 2*u, 2*u), mat{base: rgb(0xff, 0xc8, 0x30), gloss: 1}, ow)
		c.part(ell(-2*u, 8*u, 10*u, 8*u), mat{base: rgb(0xff, 0xc8, 0x30), gloss: 1.2, shine: 30}, ow)
	case fiBox, fiGoldBox:
		col := rgb(0xd8, 0x96, 0x50)
		rib := rgb(0xf0, 0x40, 0x50)
		if kind == fiGoldBox {
			col, rib = rgb(0xff, 0xc4, 0x30), rgb(0x8a, 0x40, 0xf0)
		}
		b := rrect(-15*u, -6*u, 30*u, 22*u, 3*u)
		P(b, col)
		P(rrect(-17*u, -12*u, 34*u, 8*u, 3*u), light(col, 0.15))
		c.part(rrect(-3*u, -12*u, 6*u, 28*u, 1*u), mat{base: rib, gloss: 0.6}, ow*0.6)
		c.part(ell(-6*u, -15*u, 6*u, 4*u).rot(-6*u, -15*u, 0.5).or(ell(6*u, -15*u, 6*u, 4*u).rot(6*u, -15*u, -0.5)), mat{base: rib, gloss: 0.6}, ow*0.6)
		c.fill(line(2*u, 2*u, -5*u, 2*u, 0, 8*u, 0, 11*u), colWhite) // a question mark-ish squiggle
	case fiHeart:
		h := circ(-6*u, -4*u, 8*u).or(circ(6*u, -4*u, 8*u)).or(poly(-13.5*u, 0, 13.5*u, 0, 0, 15*u))
		P(h, rgb(0xff, 0x40, 0x70))
	case fiPlate:
		c.part(ell(0, 4*u, 19*u, 9*u), mat{base: rgb(0xf8, 0xf8, 0xff), dark: rgb(0xc8, 0xc8, 0xe0), flat: 0.4}, ow)
		c.fill(ell(0, 3*u, 12*u, 5*u).ring(1.2*u), rgb(0xc8, 0xc8, 0xe0))
	case fiFlame:
		f := blob(0, -18*u, 8*u, -4*u, 10*u, 8*u, 0, 15*u, -10*u, 8*u, -8*u, -4*u)
		c.part(f, mat{base: rgb(0xff, 0x8a, 0x20), flat: 0.5}, ow)
		c.volOn(blob(0, -6*u, 5*u, 3*u, 5*u, 10*u, 0, 13*u, -5*u, 10*u, -5*u, 3*u), f, mat{base: rgb(0xff, 0xe0, 0x50), flat: 0.5})
	}
}

// effectText describes an effect: "+40% chef damage".
func effectText(k int, v float64) string { return fmt.Sprintf(effectFmt[k], int(math.Round(v))) }
