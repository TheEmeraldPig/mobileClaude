package game

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Creatures, as the server reports them (see others.go), drawn with Mote's sprites.
// Positions are world px on the island.

type entKind int

const (
	entGloom   entKind = iota // crawls along cave floors and hops at you
	entBat                    // flutters through dark caves and swoops
	entItem                   // (unused: kept so the sprite numbers stay put)
	entShade                  // prowls and pounces
	entWarden                 // the boss of Cinder Isle
	entSkitter                // small, quick and skittish
	entWisp                   // a cold glowing spirit; hovers, then dives
	entBrute                  // big, slow and tough
)

type Entity struct {
	kind       entKind
	mk         *mmo.MobKind
	nid        int // the server's id for it
	rar        int
	x, y, w, h float64
	tx, ty     float64 // where the server last saw it
	tvx, tvy   float64 // and its speed then (px per second)
	hpk        int     // health left, 0-1000
	flags      int
	seenAt     int
	face       int
	t          int // age in frames
	hurt       int // flash timer
}

var ents []*Entity

var gloomSprite = []string{
	"..OOOO..",
	".OGGGGO.",
	"OGGGGGGO",
	"OGeGGeGO",
	"OGGGGGGO",
	".O.O.O..",
}

var batSprite = [][]string{{
	"O.....O",
	"OO.O.OO",
	"OOOOOOO",
	".OeOeO.",
	"..O.O..",
}, {
	".......",
	"..O.O..",
	"OOOOOOO",
	"OOeOeOO",
	"O.O.O.O",
}}

var skitterSprite = [][]string{{
	".O..O.",
	"ORRRRO",
	"OeRReO",
	"O.OO.O",
}, {
	"O....O",
	"ORRRRO",
	"OeRReO",
	".O..O.",
}}

var wispSprite = [][]string{{
	"..WW..",
	".WVVW.",
	"WVeeVW",
	"WVVVVW",
	".WVVW.",
	"..W.W.",
}, {
	"..WW..",
	".WVVW.",
	"WVeeVW",
	"WVVVVW",
	".WVVW.",
	".W.W..",
}}

var bruteSprite = []string{
	"...OOOOOO...",
	"..OBBBBBBO..",
	".OBBBBBBBBO.",
	"OBBeBBBBeBBO",
	"OBBBBBBBBBBO",
	"OBMBBBBBBMBO",
	".OBBBBBBBBO.",
	".OBO....OBO.",
	".OO......OO.",
}

var entCol = map[byte][3]int{'O': {30, 22, 46}, 'G': {82, 64, 120}, 'S': {40, 34, 70}, 'P': {150, 60, 140}, 'K': {255, 120, 220},
	'R': {128, 74, 54}, 'W': {120, 200, 255}, 'V': {200, 240, 255}, 'B': {84, 78, 96}, 'M': {70, 110, 66}}

var wardenSprite = []string{
	"..OO........OO..",
	".OPPO......OPPO.",
	"OPPPPOOOOOOPPPPO",
	"OPPPPPPPPPPPPPPO",
	".OPPeePPPPeePPO.",
	".OPPeePKKPeePPO.",
	"..OPPPPKKPPPPO..",
	"..OPPPPPPPPPPO..",
	"...OPOPPPPOPO...",
	"...OO.OPPO.OO...",
	"......OPPO......",
	".......OO.......",
}

func drawEntities(lit, put func(x, y int, c [3]int, s int), glow func(x, y int, c [3]int, a float64), ox, oy int) {
	for _, e := range ents {
		bx, by := int(math.Round(e.x/CELL))-ox, int(math.Round(e.y/CELL))-oy
		// rare creatures have a soft halo in their rarity's colour
		if e.rar >= 2 {
			rc := mmo.Rarities[e.rar].Col
			a := (0.12 + 0.06*float64(e.rar)) * (0.85 + 0.15*math.Sin(float64(frame)*0.1+float64(e.nid)))
			cw, ch := e.w/CELL, e.h/CELL
			r := math.Max(cw, ch)*0.5 + 4
			mx, my := float64(bx)+cw/2, float64(by)+ch/2
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					if d := math.Hypot(dx, dy) / r; d < 1 {
						glow(int(mx+dx), int(my+dy), rc, a*(1-d)*(1-d))
					}
				}
			}
		}
		{
			rows := gloomSprite
			switch e.kind {
			case entBat:
				rows = batSprite[(e.t/6)%2]
			case entWarden:
				rows = wardenSprite
			case entSkitter:
				rows = skitterSprite[(e.t/4)%2]
			case entWisp:
				rows = wispSprite[(e.t/10)%2]
			case entBrute:
				rows = bruteSprite
			}
			for y, r := range rows {
				for x := 0; x < len(r); x++ {
					sx := x
					if e.face < 0 {
						sx = len(r) - 1 - x
					}
					ch := r[sx]
					switch {
					case ch == '.':
					case ch == 'e':
						put(bx+x, by+y, [3]int{255, 90, 90}, 0) // glowing eyes, visible in the dark
					case e.hurt > 0:
						put(bx+x, by+y, [3]int{255, 255, 255}, 0)
					case e.flags&mmo.MobBurn != 0 && (x+y+e.t/4)%5 == 0:
						put(bx+x, by+y, [3]int{255, 150, 60}, 0)
					case e.flags&mmo.MobSlow != 0 && ch != 'O':
						lit(bx+x, by+y, mix(entCol[ch], [3]int{176, 214, 255}, 0.5), 0)
					case e.kind == entShade && ch == 'G':
						lit(bx+x, by+y, entCol['S'], 0)
					case e.kind == entWarden || e.kind == entWisp:
						put(bx+x, by+y, entCol[ch], 0) // these glow, visible in the dark
					default:
						lit(bx+x, by+y, entCol[ch], 0)
					}
				}
			}
		}
	}
}
