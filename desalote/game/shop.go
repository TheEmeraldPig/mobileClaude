package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The shop between rounds, in tabs so it fits a phone:
//   - Auction: charms and a hex go to the highest bidder (Race: a personal shop with set prices)
//   - Tomes: level up one of your picked hands
//   - Trade: sell charms to the bank, list them for other players, buy theirs

func shopScreen(p *PlayerState) {
	top := playerPills(8) + 10
	charmRow(p, top)
	race := view.Mode == ModeRace
	// what's next, and the clock
	var next string
	if race {
		next = fmt.Sprintf("Next: ante %d, %s (%s)", p.Ante+1, blindNames[p.Blind], commas(view.targetFor(p.Ante, p.Blind)))
	} else {
		next = fmt.Sprintf("Next: ante %d, %s", view.Ante+1, blindNames[view.Blind])
		if view.isShowdown(view.Blind) {
			next += " - a showdown"
		} else {
			next += " (" + commas(view.targetFor(view.Ante, view.Blind)) + ")"
		}
	}
	title := "SHOP"
	if !race {
		title = fmt.Sprintf("AUCTION  %ds", view.Secs)
	}
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, title, 14, top+12, textOpt{size: 16, display: true, col: colT, shadow: true})
		drawText(dst, next, 14, top+32, textOpt{size: 12, col: colMuted})
	})
	// the items, spread over the table
	bh := 40.0
	by := logicalH - bh - 12
	area := Rect{14, top + 50, logicalW - 28, by - 14 - (top + 50)}
	if race {
		raceShop(p, area)
	} else {
		auction(p, area)
	}
	// along the bottom: money, the other menus, done
	free := p.Money
	if !race {
		free -= view.committed(p.ID, -1)
	}
	moneyS := fmt.Sprintf("$%d", p.Money)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, moneyS, 16, by+bh/2-2, textOpt{size: 22, display: true, col: colT, shadow: true})
	})
	if free != p.Money {
		Text(fmt.Sprintf("$%d free", free), 18, by-8, 11, colMuted, text.AlignStart)
	}
	bw := math.Min(130, (logicalW-100-40)/4)
	x := 100.0
	if Button(Rect{x, by, bw, bh}, "Tomes", styleSlate, !p.Out, nil) {
		ov = ovTomes
	}
	x += bw + 10
	label := "Trade"
	if n := otherListings(); n > 0 {
		label = fmt.Sprintf("Trade (%d)", n)
	}
	if Button(Rect{x, by, bw, bh}, label, styleSlate, !p.Out, nil) {
		ov = ovTrade
	}
	x += bw + 10
	if len(p.Hexes) > 0 && view.Mode != ModeCoop && Button(Rect{x, by, bw, bh}, fmt.Sprintf("Hexes (%d)", len(p.Hexes)), styleViolet, !p.Out, nil) {
		ov = ovHexes
	}
	dw := math.Min(170, bw+40)
	br := Rect{logicalW - dw - 12, by, dw, bh}
	switch {
	case p.Out:
		Text("You're out", br.X+br.W/2, br.Y+18, 13, colMuted, text.AlignCenter)
	case race:
		if Button(br, "Next round", styleGold, true, nil) {
			act(action{T: "rcont"})
		}
	case p.Ready:
		if Button(br, "Waiting (undo)", styleSlate, true, nil) {
			act(action{T: "unready"})
		}
	default:
		if Button(br, "Done shopping", styleGold, true, nil) {
			act(action{T: "ready"})
		}
	}
}

// Charms other players have put up for sale
func otherListings() int {
	n := 0
	for _, l := range view.Market {
		if l.Seller != myID {
			n++
		}
	}
	return n
}

// One shop item box: art, name, description
func itemBox(r Rect, kind, id string) {
	Panel(r)
	art := math.Min(72, r.H*0.3)
	ax, ay := r.X+(r.W-art)/2, r.Y+8
	name, desc := "", ""
	switch kind {
	case "charm":
		d := charms[id]
		name, desc = d.name, d.desc
		ops = append(ops, func(dst *ebiten.Image) { drawCharm(dst, id, ax, ay, art, false) })
		if d.rarity > 0 {
			Text([]string{"", "uncommon", "rare"}[d.rarity], r.X+r.W-6, r.Y+10, 9, colT, text.AlignEnd)
		}
	case "hex":
		d := hexes[id]
		name, desc = "Hex: "+d.name, d.desc
		ops = append(ops, func(dst *ebiten.Image) { drawHex(dst, ax+art*0.15, ay, art*0.7, art) })
	}
	Text(name, r.X+r.W/2, ay+art+14, 13, colInk, text.AlignCenter)
	size := 11.0
	if wrapHeight(desc, r.W-12, size) > r.H-art-80 {
		size = 10
	}
	wrapText(desc, r.X+6, ay+art+32, r.W-12, size, colMuted, text.AlignStart)
}

func auction(p *PlayerState, area Rect) {
	n := len(view.Auction)
	if n == 0 {
		return
	}
	area, w, gap := spreadItems(area, n, 280)
	for i, it := range view.Auction {
		r := Rect{area.X + float64(i)*(w+gap), area.Y, w, area.H}
		if quickMod() && ptr.released && !ptr.consumed && r.Has(ptr.x, ptr.y) {
			ptr.consumed = true // Ctrl+click: bid straight away
			act(action{T: "bid", Item: i})
			sfx("bid")
		}
		itemBox(Rect{r.X, r.Y, r.W, r.H - 70}, it.Kind, it.ID)
		ly := r.Y + r.H - 62
		switch {
		case it.Bidder < 0:
			Text(fmt.Sprintf("From $%d", it.Price), r.X+r.W/2, ly, 12, colMuted, text.AlignCenter)
		case it.Bidder == myID:
			Text(fmt.Sprintf("You lead: $%d", it.Bid), r.X+r.W/2, ly, 12, colT, text.AlignCenter)
		default:
			who := "?"
			if q := view.player(it.Bidder); q != nil {
				who = q.Name
			}
			Text(fmt.Sprintf("%s: $%d", who, it.Bid), r.X+r.W/2, ly, 12, colDanger, text.AlignCenter)
		}
		next := max(it.Price, it.Bid+1)
		can := !p.Out && it.Bidder != myID && next <= p.Money-view.committed(myID, i) &&
			(it.Kind != "charm" || len(p.Charms) < maxCharms)
		label := fmt.Sprintf("Bid $%d", next)
		if it.Bidder == myID {
			label = "Leading"
		}
		if Button(Rect{r.X + 4, r.Y + r.H - 44, r.W - 8, 36}, label, styleGold, can, nil) {
			act(action{T: "bid", Item: i})
			sfx("bid")
		}
	}
}

// Lay n shop items out across the table, card-sized and centred
func spreadItems(area Rect, n int, maxH float64) (Rect, float64, float64) {
	gap := 16.0
	w := math.Min(200, (area.W-gap*float64(n-1))/float64(n))
	h := math.Min(area.H, maxH)
	total := w*float64(n) + gap*float64(n-1)
	return Rect{area.X + (area.W-total)/2, area.Y + (area.H-h)/2, total, h}, w, gap
}

func raceShop(p *PlayerState, area Rect) {
	n := len(p.Shop)
	area, w, gap := spreadItems(area, n, 270)
	for i, it := range p.Shop {
		r := Rect{area.X + float64(i)*(w+gap), area.Y, w, area.H}
		if quickMod() && ptr.released && !ptr.consumed && r.Has(ptr.x, ptr.y) {
			ptr.consumed = true // Ctrl+click: buy straight away
			act(action{T: "rbuy", Item: i})
			sfx("coin")
		}
		itemBox(Rect{r.X, r.Y, r.W, r.H - 50}, it.Kind, it.ID)
		label := fmt.Sprintf("Buy $%d", it.Price)
		if it.Sold {
			label = "Bought"
		}
		can := !it.Sold && p.Money >= it.Price && (it.Kind != "charm" || len(p.Charms) < maxCharms)
		if Button(Rect{r.X + 4, r.Y + r.H - 44, r.W - 8, 36}, label, styleGold, can, nil) {
			act(action{T: "rbuy", Item: i})
			sfx("coin")
		}
	}
}

func tomes(p *PlayerState, area Rect) {
	y := wrapText("A tome levels up one of your picked hands for the rest of the match (once per hand each shop).", area.X, area.Y+6, area.W, 12, colMuted, text.AlignStart) + 4
	free := p.Money
	if view.Mode != ModeRace {
		free -= view.committed(p.ID, -1)
	}
	rh := math.Min(70, (area.H-34)/3-8)
	for _, h := range p.Picks {
		r := Rect{area.X, y, area.W, rh}
		Inset(r, colSlateLo)
		f := hands[h].fam
		hh := h
		ops = append(ops, func(dst *ebiten.Image) { drawTome(dst, f, r.X+8, r.Y+6, (rh-12)*0.75, rh-12) })
		lv := p.Levels[h]
		b0, m0 := handLevel(h, lv+1)
		b1, m1 := handLevel(h, lv+2)
		tx := r.X + 16 + (rh-12)*0.75
		Text(fmt.Sprintf("%s  Lv %d -> %d", hands[hh].name, lv+1, lv+2), tx, r.Y+rh/2-8, 14, colInk, text.AlignStart)
		Text(fmt.Sprintf("%d x %d  ->  %d x %d", b0, m0, b1, m1), tx, r.Y+rh/2+10, 12, colT, text.AlignStart)
		bought := false
		for _, t := range p.Tomes {
			bought = bought || t == h
		}
		price := tomePrice(p, h)
		label := fmt.Sprintf("$%d", price)
		if bought {
			label = "Bought"
		}
		if Button(Rect{r.X + r.W - 100, r.Y + (rh-34)/2, 90, 34}, label, styleGold, !bought && free >= price && !p.Out, nil) {
			act(action{T: "tome", Hand: h})
			sfx("coin")
		}
		y += rh + 8
	}
}

func trade(p *PlayerState, area Rect) {
	half := (area.W - 12) / 2
	// your charms: sell to the bank or list for players
	Label("YOUR CHARMS", area.X, area.Y+4)
	y := area.Y + 18
	rh := math.Min(40, (area.H-20)/float64(maxCharms)-6)
	for _, id := range p.Charms {
		r := Rect{area.X, y, half, rh}
		Inset(r, colSlateLo)
		Text(charms[id].name, r.X+8, r.Y+rh/2, 12, colInk, text.AlignStart)
		listed := false
		for _, l := range view.Market {
			listed = listed || (l.Seller == myID && l.Charm == id)
		}
		bw := math.Min(70, half*0.26)
		if listed {
			if Button(Rect{r.X + r.W - bw - 4, r.Y + 3, bw, rh - 6}, "Unlist", styleSlate, true, nil) {
				act(action{T: "unlist", Charm: id})
			}
		} else {
			if Button(Rect{r.X + r.W - bw*2 - 8, r.Y + 3, bw, rh - 6}, fmt.Sprintf("Sell $%d", sellPrice(id)), styleSlate, true, nil) {
				act(action{T: "sell", Charm: id})
				sfx("coin")
			}
			if view.Mode != ModeRace || len(view.Players) > 1 {
				if Button(Rect{r.X + r.W - bw - 4, r.Y + 3, bw, rh - 6}, "List", styleViolet, true, nil) {
					ov, priceCharm, priceAmt = ovPrice, id, charms[id].price
				}
			}
		}
		y += rh + 6
	}
	if len(p.Charms) == 0 {
		Text("No charms to sell", area.X+4, y+10, 12, colMuted, text.AlignStart)
	}
	// the market
	mx := area.X + half + 12
	Label("MARKET", mx, area.Y+4)
	y = area.Y + 18
	any := false
	for _, l := range view.Market {
		if l.Seller == myID {
			continue
		}
		any = true
		seller := "?"
		if q := view.player(l.Seller); q != nil {
			seller = q.Name
		}
		r := Rect{mx, y, half, rh}
		Inset(r, colSlateLo)
		Text(charms[l.Charm].name, r.X+8, r.Y+rh/2-6, 12, colInk, text.AlignStart)
		Text("from "+seller, r.X+8, r.Y+rh/2+8, 10, colMuted, text.AlignStart)
		can := !p.Out && len(p.Charms) < maxCharms && p.Money-view.committed(myID, -1) >= l.Price
		if Button(Rect{r.X + r.W - 84, r.Y + 3, 80, rh - 6}, fmt.Sprintf("Buy $%d", l.Price), styleGold, can, nil) {
			act(action{T: "buyl", Item: l.ID})
			sfx("coin")
		}
		y += rh + 6
	}
	if !any {
		wrapText("Nothing for sale. Charms other players list show up here.", mx+4, y+10, half-8, 12, colMuted, text.AlignStart)
	}
}
