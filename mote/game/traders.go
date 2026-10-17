package game

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Traders: motes of other colours who live in village houses (structures.go). Tap one
// (right-click on PC) to open its shop: buy things with coins, or sell your stuff for
// coins. Coins also drop from creatures and turn up in chests.

type traderKind struct {
	name, title string
	body, light [3]int
	hat         []string // drawn on top of the head ('H' hat colour, 'h' band)
	hatCol      [2][3]int
	stock       []offer
}

type offer struct {
	id    string
	n     int
	price int
}

var traderKinds = []traderKind{
	{name: "Pip", title: "Merchant", body: [3]int{236, 170, 72}, light: [3]int{255, 220, 150},
		hat: []string{"..HHHHHH..", ".hhhhhhhh."}, hatCol: [2][3]int{{120, 60, 150}, {255, 222, 120}},
		stock: []offer{{"torch", 10, 4}, {"bread", 3, 8}, {"potion", 1, 15}, {"bomb", 2, 12}, {"lantern", 1, 60},
			{"seeds", 5, 3}, {"storage", 1, 18}, {"bed", 1, 20}, {"gem", 1, 40}, {"bp_tower", 1, 90}}},
	{name: "Flint", title: "Smith", body: [3]int{150, 156, 172}, light: [3]int{206, 212, 226},
		hat: []string{"...HHHH...", "..hhhhhh.."}, hatCol: [2][3]int{{70, 64, 80}, {220, 120, 60}},
		stock: []offer{{"copperbar", 2, 10}, {"ironbar", 2, 18}, {"copperpick", 1, 45}, {"ironpick", 1, 120},
			{"coppersword", 1, 40}, {"ironsword", 1, 110}, {"copperarmor", 1, 70}, {"ironarmor", 1, 160}, {"anvil", 1, 50}}},
	{name: "Fern", title: "Farmer", body: [3]int{110, 190, 96}, light: [3]int{180, 236, 160},
		hat: []string{"HHHHHHHHHH", "..hhhhhh.."}, hatCol: [2][3]int{{226, 196, 110}, {150, 104, 62}},
		stock: []offer{{"seeds", 10, 5}, {"wheat", 5, 6}, {"berries", 5, 5}, {"bread", 5, 12}, {"hoe", 1, 15},
			{"bucket", 1, 25}, {"bp_farm", 1, 40}, {"wall_plank", 20, 8}, {"heart", 1, 150}}},
}

// What the traders pay for one of an item (0 = they won't buy it)
var sellValue = map[string]int{
	"coal": 1, "copper": 2, "iron": 4, "gold": 7, "crystal": 6, "copperbar": 5, "ironbar": 9, "goldbar": 15,
	"coreshard": 40, "wood": 1, "plank": 0, "wheat": 2, "berries": 1, "bread": 4, "seeds": 0, "gem": 20,
	"heart": 60, "obsidian": 2, "glass": 1, "brick": 1, "leaves": 0, "potion": 7,
	"pickaxe": 4, "copperpick": 20, "ironpick": 50, "goldpick": 90, "corepick": 200, "hoe": 5, "hammer": 5,
	"bucket": 10, "coppersword": 18, "ironsword": 45, "copperarmor": 30, "ironarmor": 70, "lantern": 25,
	"bomb": 4, "workbench": 4, "furnace": 8, "anvil": 20, "storage": 6, "bed": 6, "door": 4, "paintbrush": 3,
}

// ---- In the world ----

type traderSpot struct {
	house vHouse
	lx    float64 // feet centre, local cells
	feet  int     // row the feet stand on top of
}

var traderSpots []traderSpot

// Where each trader in the loaded window stands this frame (they pace about their house)
func updateTraders() {
	traderSpots = traderSpots[:0]
	for _, hs := range loadedHouses() {
		lx := float64(hs.x-baseChunk*CW) + float64(hs.w)/2
		pace := math.Sin(float64(frame)*0.006+float64(hs.id)) * float64(hs.w/2-8)
		traderSpots = append(traderSpots, traderSpot{hs, lx + pace, hs.floor})
	}
}

// The trader whose body covers local cell (x, y), if it's close enough to the mote
func traderAt(x, y int, needReach bool) *traderSpot {
	for i := range traderSpots {
		t := &traderSpots[i]
		if math.Abs(float64(x)-t.lx) <= 6 && y <= t.feet && y >= t.feet-16 {
			if needReach && !traderInReach(t) {
				continue
			}
			return t
		}
	}
	return nil
}

func traderInReach(t *traderSpot) bool {
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	return math.Hypot(t.lx-pcx, float64(t.feet-7)-pcy) <= reach+10
}

func drawTraders(lit func(x, y int, c [3]int, s int), put func(x, y int, c [3]int, s int), ox, oy int) {
	for _, t := range traderSpots {
		k := traderKinds[t.house.kind]
		face := 1
		if math.Cos(float64(frame)*0.006+float64(t.house.id)) < 0 {
			face = -1
		}
		blink := (frame+t.house.id*37)%200 < 7
		rows := playerFrame(blink, false, []string{"stand", "a", "stand", "b"}[(frame/12+t.house.id)%4], 0)
		x0 := int(math.Round(t.lx)) - sprW/2 - ox
		y0 := t.feet - len(rows) + 1 - oy
		bob := 0
		if (frame/30+t.house.id)%2 == 0 {
			bob = 1
		}
		for y, r := range rows {
			for x := 0; x < len(r); x++ {
				sx := x
				if face < 0 {
					sx = len(r) - 1 - x
				}
				ch := r[sx]
				var c [3]int
				switch ch {
				case '.':
					continue
				case 'B':
					c = k.body
				case 'L':
					c = k.light
				case 'W', 'E':
					put(x0+x, y0+y+bob, pcol[ch], 0) // eyes always show
					continue
				default:
					c = pcol[ch]
				}
				lit(x0+x, y0+y+bob, c, 0)
			}
		}
		for y, r := range k.hat {
			for x := 0; x < len(r); x++ {
				switch r[x] {
				case 'H':
					lit(x0+x, y0+1+y+bob, k.hatCol[0], 0)
				case 'h':
					lit(x0+x, y0+1+y+bob, k.hatCol[1], 0)
				}
			}
		}
	}
}

// Name tags over traders the mote is near
func traderLabels() {
	for i := range traderSpots {
		t := &traderSpots[i]
		if !traderInReach(t) {
			continue
		}
		k := traderKinds[t.house.kind]
		sx := (t.lx*CELL - cam.x) * zoom
		sy := (float64(t.feet-17)*CELL - cam.y) * zoom
		label := k.name + " the " + k.title
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, label, sx, sy, textOpt{size: 12, align: text.AlignCenter, shadow: true, col: colT})
		})
	}
}

// ---- The shop ----

var (
	shopKind  int
	shopMsg   string
	shopMsgAt int
	shopTab   int // phones: 0 buy, 1 sell
)

func openShop(kind int) {
	shopKind, shopMsg, shopTab = kind, "", 0
	openScreen(scShop)
	sfx("chest")
}

func coins() int { return countItem("coin") }

func shopMenu() {
	k := traderKinds[shopKind]
	r := sheet(940, 470)
	if _, cl := head(r, strings.ToUpper(k.name+" the "+k.title), false); cl {
		play()
		return
	}
	// coins
	ci := items["coin"]
	IconAt(ci.img, r.X+r.W-190, r.Y+18, 20)
	Text(fmt.Sprintf("%d coins", coins()), r.X+r.W-164, r.Y+28, 15, colT, text.AlignStart)
	compact := isTouch && r.W < 860
	top := r.Y + 60
	colW := (r.W - 36 - 24) / 2
	bx, sx := r.X+18, r.X+18+colW+24
	if compact {
		colW = r.W - 36
		sx = bx
		if Button(Rect{bx, top, 110, 30}, "Buy", map[bool]btnStyle{true: styleViolet, false: styleSlate}[shopTab == 0], true, nil) {
			shopTab = 0
		}
		if Button(Rect{bx + 120, top, 110, 30}, "Sell", map[bool]btnStyle{true: styleViolet, false: styleSlate}[shopTab == 1], true, nil) {
			shopTab = 1
		}
		top += 42
	}
	// buy list
	if !compact || shopTab == 0 {
		if !compact {
			Label("BUY", bx, top+4)
			top += 16
		}
		rh := math.Min(40, (r.Y+r.H-16-top)/float64(len(k.stock))-6)
		y := top
		for _, of := range k.stock {
			it := items[of.id]
			if it == nil {
				continue
			}
			row := Rect{bx, y, colW, rh}
			Inset(row, colSlateLo)
			IconAt(it.img, row.X+8, row.Y+(rh-22)/2, 22)
			name := it.name
			if of.n > 1 {
				name = fmt.Sprintf("%s x%d", it.name, of.n)
			}
			Text(name, row.X+38, row.Y+rh/2, 13, colInk, text.AlignStart)
			can := coins() >= of.price
			if Button(Rect{row.X + row.W - 96, row.Y + (rh-26)/2, 88, 26}, fmt.Sprintf("%d c", of.price), styleGold, can, nil) {
				removeItem("coin", of.price)
				giveItem(of.id, of.n)
				sfx("craft")
				shopMsg, shopMsgAt = "Bought "+name+".", frame
			}
			y += rh + 6
		}
	}
	// sell: tap a stack in your bag
	if !compact || shopTab == 1 {
		y := r.Y + 60
		if compact {
			y = top
		} else {
			Label("SELL (tap a stack)", sx, y+4)
			y += 16
		}
		gap := 6.0
		slot := math.Min(48, (colW-7*gap)/8)
		for i := 0; i < invSize; i++ {
			sr := Rect{sx + float64(i%8)*(slot+gap), y + float64(i/8)*(slot+gap), slot, slot}
			s := inv[i]
			if !stackSlot(sr, s) || s == nil {
				continue
			}
			v := sellValue[s.ID]
			switch {
			case s.ID == "coin":
			case s.Temp:
				shopMsg, shopMsgAt = "The temporary kit can't be sold.", frame
			case v == 0:
				shopMsg, shopMsgAt = k.name+" doesn't want "+items[s.ID].name+".", frame
			default:
				got := v * s.N
				shopMsg, shopMsgAt = fmt.Sprintf("Sold %d %s for %d coins.", s.N, items[s.ID].name, got), frame
				inv[i] = nil
				if selected == i {
					selected = -1
				}
				giveItem("coin", got)
				sfx("pickup")
			}
		}
		hy := y + 5*(slot+gap) + 8
		hover := ""
		for i := 0; i < invSize && !isTouch; i++ {
			sr := Rect{sx + float64(i%8)*(slot+gap), y + float64(i/8)*(slot+gap), slot, slot}
			if sr.Has(hoverX, hoverY) && inv[i] != nil {
				if v := sellValue[inv[i].ID]; v > 0 {
					hover = fmt.Sprintf("%s: %d coins each", items[inv[i].ID].name, v)
				} else if inv[i].ID != "coin" {
					hover = items[inv[i].ID].name + ": not wanted"
				}
			}
		}
		if hover != "" {
			Text(hover, sx, hy+8, 12, colMuted, text.AlignStart)
		}
	}
	if shopMsg != "" && frame-shopMsgAt < 180 {
		Text(shopMsg, r.X+r.W-18, r.Y+r.H-14, 12, colT, text.AlignEnd)
	}
}
