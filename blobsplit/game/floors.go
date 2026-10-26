package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Floors: the challenge is always on. Every floor has five rooms, each with a
// requirement; the fifth is a boss with a twist. Clearing a room pays DNA and moves
// on; running out of sells sends you back to room 1 of the floor, with nothing else
// lost. Every tenth floor cleared (past your best) unlocks an ability card. The
// requirement is only shown in the CHALLENGE screen.

const (
	roomsPerFloor = 5
	floorsPerCard = 10
)

// Kinds of room requirement.
const (
	reqEarn    = iota // earn a total within the sells
	reqSingle         // one sale worth at least the target
	reqMutants        // sell a board with at least N mutated biotes
	reqGain           // create a biote with a given mutation
)

type req struct {
	kind   int
	target float64
	count  int
	mut    int
	sells  int
	boss   bool
}

// floorBase is the money scale of a floor: three times the last.
func floorBase(f int) float64 { return 25 * math.Pow(4, float64(f-1)) }

// roomReq works out the current room's requirement. It is the same every time for a
// given floor and room, so it doesn't change when you come back to it.
func (g *Game) roomReq() req {
	return g.reqFor(g.st.Floor, g.st.Room)
}

func (g *Game) reqFor(f, room int) req {
	rng := rand.New(rand.NewSource(int64(f*31 + room)))
	base := floorBase(f) * g.m.target
	extra := g.m.sells
	if room == roomsPerFloor-1 {
		r := req{kind: reqEarn, target: base * 3, sells: baseSells + extra, boss: true}
		if bosses[g.st.Boss%len(bosses)].id == "leech" {
			r.sells -= 2
		}
		r.sells = max(1, r.sells)
		return r
	}
	kind := reqEarn
	if room > 0 {
		kind = []int{reqEarn, reqSingle, reqMutants, reqGain}[rng.Intn(4)]
	}
	k := []float64{1, 1.3, 1.6, 2}[room]
	switch kind {
	case reqSingle:
		return req{kind: kind, target: base * 0.6 * k, sells: baseSells + extra}
	case reqMutants:
		return req{kind: kind, count: min(2+f, 48), sells: 6 + extra}
	case reqGain:
		return req{kind: kind, mut: 1 + rng.Intn(min(nMut-1, 1+f/3)), sells: 10 + extra}
	}
	return req{kind: reqEarn, target: base * k, sells: baseSells + extra}
}

// describe is what a requirement asks, in a few words.
func (r req) describe() string {
	switch r.kind {
	case reqSingle:
		return "MAKE ONE SALE WORTH $" + fmtNum(r.target)
	case reqMutants:
		return fmt.Sprintf("SELL %d MUTATED BIOTES AT ONCE", r.count)
	case reqGain:
		return "CREATE A " + upper(rarities[r.mut].Name) + " BIOTE"
	}
	return "EARN $" + fmtNum(r.target)
}

// progress is how far the room is, 0-1, and a label for it.
func (g *Game) progress(r req) (float64, string) {
	s := &g.st
	switch r.kind {
	case reqSingle:
		return math.Min(1, s.RoomBest/r.target), "BEST SALE $" + fmtNum(s.RoomBest)
	case reqMutants:
		return math.Min(1, s.RoomBest/float64(r.count)), fmt.Sprintf("BEST %d / %d", int(s.RoomBest), r.count)
	case reqGain:
		return 0, "NOT YET"
	}
	return math.Min(1, s.RoomEarned/r.target), "$" + fmtNum(s.RoomEarned) + " / $" + fmtNum(r.target)
}

// roomSale counts a sale (worth v, with mutated mutated biotes) towards the room.
func (g *Game) roomSale(v float64, mutated int) {
	s := &g.st
	r := g.roomReq()
	s.RoomSells++
	done := false
	switch r.kind {
	case reqEarn:
		s.RoomEarned += v
		done = s.RoomEarned >= r.target
	case reqSingle:
		s.RoomBest = math.Max(s.RoomBest, v)
		done = v >= r.target
	case reqMutants:
		s.RoomBest = math.Max(s.RoomBest, float64(mutated))
		done = mutated >= r.count
	}
	if done {
		g.clearRoom()
	} else if s.RoomSells >= r.sells {
		g.failRoom()
	}
}

// gained is called when a split creates mutation m: a "create" room may be done.
func (g *Game) gained(m int) {
	if r := g.roomReq(); r.kind == reqGain && r.mut == m {
		g.clearRoom()
	}
}

func (g *Game) resetRoom() {
	s := &g.st
	s.RoomEarned, s.RoomBest, s.RoomSells = 0, 0, 0
}

func (g *Game) failRoom() {
	g.st.Room = 0
	g.resetRoom()
	g.floats = append(g.floats, floater{s: "ROOM FAILED - BACK TO ROOM 1", x: g.W / 2, y: g.H * 0.3, c: rgb(0xff8a8a), size: 24})
	g.play("deny")
}

func (g *Game) clearRoom() {
	s := &g.st
	msg := fmt.Sprintf("ROOM %d CLEARED!", s.Room+1)
	if s.Room == roomsPerFloor-1 {
		msg = fmt.Sprintf("FLOOR %d CLEARED!  +1 DNA NEXT REBIRTH", s.Floor)
		s.Floor++
		s.Room = 0
		s.RunFloors++
		s.Boss = rand.Intn(len(bosses))
		cleared := s.Floor - 1
		if cleared > s.BestFloor {
			s.BestFloor = cleared
			if cleared%floorsPerCard == 0 {
				g.awardCard()
			}
		}
	} else {
		s.Room++
	}
	g.questAdd(qRoom, 1)
	g.resetRoom()
	g.floats = append(g.floats, floater{s: msg, x: g.W / 2, y: g.H * 0.3, c: colG, size: 24})
	if s.Room == 0 { // a whole floor
		g.confetti(90)
		g.shakeT = g.t
	} else {
		g.confetti(30)
	}
	g.play("rebirth")
	g.save()
}

// awardCard unlocks a card you don't have yet: first a rarity by weight (Lucky Draw
// shifts it towards rarer), then a card of that rarity.
func (g *Game) awardCard() {
	var left []cardDef
	for _, c := range cards {
		if !g.hasCard(c.id) {
			left = append(left, c)
		}
	}
	if len(left) == 0 {
		g.st.Gems += 25 // every card is unlocked already
		return
	}
	w, total := g.cardWeights(left)
	x := rand.Float64() * total
	pick := left[len(left)-1]
	for i := range left {
		if x -= w[i]; x < 0 {
			pick = left[i]
			break
		}
	}
	g.st.Cards = append(g.st.Cards, pick.id)
	g.banner.r, g.banner.t, g.banner.text = pick.mut, g.t, cardRarities[pick.rar].name+" CARD: "+pick.name
}

// cardWeights is how likely each card in left is to be won next.
func (g *Game) cardWeights(left []cardDef) (w []float64, total float64) {
	w = make([]float64, len(left))
	for i, c := range left {
		w[i] = cardRarities[c.rar].odds
		if g.has("market") && c.rar > 0 {
			w[i] *= 1.5 + 0.5*float64(c.rar)
		}
		total += w[i]
	}
	return w, total
}

// rarityOdds is the chance, right now, that the next card won is of each rarity.
// The shop shows these for the Card Pack, which is bought with real money.
func (g *Game) rarityOdds() []float64 {
	var left []cardDef
	for _, c := range cards {
		if !g.hasCard(c.id) {
			left = append(left, c)
		}
	}
	out := make([]float64, len(cardRarities))
	w, total := g.cardWeights(left)
	for i, c := range left {
		out[c.rar] += w[i] / total
	}
	return out
}

// cardOddsText writes rarityOdds, e.g. "ODDS NOW:  COMMON 50%  UNCOMMON 28% ...",
// with names cut to 4 letters when short is set.
func (g *Game) cardOddsText(short bool) string {
	s := "ODDS NOW:"
	for i, p := range g.rarityOdds() {
		pc := fmt.Sprintf("%.1f", p*100)
		if p >= 0.1 || p == 0 {
			pc = fmt.Sprintf("%.0f", p*100)
		}
		name := cardRarities[i].name
		if short {
			name = name[:min(4, len(name))]
		}
		s += fmt.Sprintf("  %s %s%%", name, pc)
	}
	return s
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

// ---- the CHALLENGE screen ----

type challengeLayout struct {
	back  rect
	left  rect
	rooms []rect
	right rect
	tabs  []rect // one per card rarity
	cards []rect // the shown rarity's cards, in cardsOf order
	shown []int  // their indexes in cards
	info  rect
	skip  [2]rect // skip 10 floors: dollars, tokens
}

func (g *Game) layChallenge() challengeLayout {
	u := math.Max(g.u, 1)
	wu := g.W / u
	R := func(x, y, w, h float64) rect { return rect{x * u, y * u, w * u, h * u} }
	var C challengeLayout
	C.back = R(wu-40, 8, 32, 32)
	lw := math.Min(330, wu*0.5)
	C.left = R(12, 50, lw, 298)
	rw := (lw - 24 - 4*6) / 5
	for i := 0; i < roomsPerFloor; i++ {
		C.rooms = append(C.rooms, R(24+float64(i)*(rw+6), 92, rw, 38))
	}
	rx := 12 + lw + 12
	pw := wu - rx - 12
	C.right = R(rx, 50, pw, 298)
	C.skip[0] = R(lw-24-110-6-72, 348-12-30, 72, 30)
	C.skip[1] = R(lw-24-110+12, 348-12-30, 110, 30)
	tw := (pw - 24 - 4*4) / float64(len(cardRarities))
	for i := range cardRarities {
		C.tabs = append(C.tabs, R(rx+12+float64(i)*(tw+4), 74, tw, 20))
	}
	C.shown = cardsOf(g.cardTab)
	cols, rows := 6, 4
	cw := math.Min((pw-24-float64(cols-1)*5)/float64(cols), 40)
	ch := math.Min(cw*1.25, (176-float64(rows-1)*5)/float64(rows))
	gx := rx + (pw-(float64(cols)*cw+float64(cols-1)*5))/2
	for k := range C.shown {
		C.cards = append(C.cards, R(gx+float64(k%cols)*(cw+5), 100+float64(k/cols)*(ch+5), cw, ch))
	}
	C.info = R(rx+12, 100+float64(rows)*(ch+5)+2, pw-24, 348-(100+float64(rows)*(ch+5)+2)-8)
	return C
}

func (g *Game) tapChallenge(x, y float64) {
	C := g.layChallenge()
	if C.back.in(x, y) {
		g.panel = ""
		g.play("tap")
		return
	}
	for i, r := range C.skip {
		if r.in(x, y) {
			g.buyItem("floors", i == 0)
			return
		}
	}
	for i, r := range C.tabs {
		if r.in(x, y) {
			g.cardTab, g.sel = i, -1
			g.play("tap")
		}
	}
	for k, r := range C.cards {
		if r.in(x, y) {
			g.sel = C.shown[k]
			g.play("tap")
		}
	}
}

// drawSpace is the starry backdrop of the full-screen menus.
func (g *Game) drawSpace(dst *ebiten.Image) {
	u := g.u
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
}

func (g *Game) drawChallenge(dst *ebiten.Image) {
	u := g.u
	C := g.layChallenge()
	p := math.Max(1, math.Round(u*1.3))
	soft := rgb(0xd8ccff)
	s := &g.st
	g.drawSpace(dst)
	say(dst, "CHALLENGE", face(fBig, 24*u), g.W/2, 24*u, alCenter, colT, math.Round(2*u))
	icon(dst, "dna", 20*u, 24*u, 20*u)
	say(dst, fmt.Sprintf("%d DNA  -  %d FLOORS THIS RUN", s.Gems, s.RunFloors), face(fBig, 15*u), 32*u, 24*u, alLeft, rgb(0x9ff0ff), math.Round(1.5*u))
	bevel(dst, C.back, skOrange, p)
	say(dst, "X", face(fBig, 18*u), C.back.cx(), C.back.cy()-u, alCenter, colW, math.Round(1.5*u))

	// the floor and its rooms
	bevel(dst, C.left, skDark, p)
	lx := C.left.x + 12*u
	say(dst, "FLOOR "+itoa(s.Floor), face(fBig, 22*u), lx, C.left.y+20*u, alLeft, colW, math.Round(2*u))
	next := (s.BestFloor/floorsPerCard + 1) * floorsPerCard
	say(dst, fmt.Sprintf("BEST %d  -  NEXT CARD AT FLOOR %d", s.BestFloor, next), face(fSmall, 7*u), C.left.x+C.left.w-12*u, C.left.y+20*u, alRight, soft, math.Round(u))
	for i, r := range C.rooms {
		sk := skGrey
		switch {
		case i < s.Room:
			sk = skGreen
		case i == s.Room:
			sk = skGold
		}
		if i == roomsPerFloor-1 && i != s.Room {
			sk = skin{rgb(0x6a1f3a), rgb(0xc84f6a), rgb(0x3a1020)}
		}
		bevel(dst, r, sk, p)
		label := itoa(i + 1)
		if i == roomsPerFloor-1 {
			label = "BOSS"
		}
		say(dst, label, face(fBig, 15*u), r.cx(), r.cy()-u, alCenter, colW, math.Round(1.5*u))
	}
	r := g.roomReq()
	y := C.rooms[0].y + C.rooms[0].h + 22*u
	title := fmt.Sprintf("ROOM %d", s.Room+1)
	if r.boss {
		title = "BOSS: " + upper(bosses[s.Boss%len(bosses)].name)
	}
	say(dst, title, face(fBig, 16*u), lx, y, alLeft, colT, math.Round(1.5*u))
	y += 22 * u
	y = g.wrapSay(dst, r.describe()+fmt.Sprintf(" WITHIN %d SELLS", r.sells), lx, y, C.left.w-24*u, 9, colW)
	if r.boss {
		y = g.wrapSay(dst, "TWIST: "+bosses[s.Boss%len(bosses)].desc, lx, y+2*u, C.left.w-24*u, 8, rgb(0xff9fb0))
	}
	k, label := g.progress(r)
	bar := rect{lx, y + 4*u, C.left.w - 24*u, 10 * u}
	fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
	fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
	y += 26 * u
	say(dst, label, face(fSmall, 7*u), lx, y, alLeft, colW, math.Round(u))
	say(dst, fmt.Sprintf("SELLS LEFT %d", r.sells-s.RoomSells), face(fSmall, 7*u), C.left.x+C.left.w-12*u, y, alRight, colT, math.Round(u))
	y += 18 * u
	if r.boss {
		say(dst, "CLEAR IT FOR +1 DNA AT YOUR NEXT REBIRTH", face(fSmall, 7*u), lx, y, alLeft, rgb(0x9ff0ff), math.Round(u))
	}
	g.wrapSay(dst, "Run out of sells and you go back to room 1, with nothing else lost. Every 10th floor you clear unlocks a card.", lx, C.left.y+C.left.h-76*u, C.left.w-24*u, 6, soft)
	say(dst, "SKIP 10 FLOORS", face(fBig, 14*u), lx, C.skip[0].cy()-u, alLeft, colT, math.Round(1.5*u))
	g.drawPriceButtons(dst, shopByID("floors"), g.pop(C.skip[0]), g.pop(C.skip[1]), p, "")

	// unlocked cards, one rarity at a time
	bevel(dst, C.right, skDark, p)
	say(dst, fmt.Sprintf("CARDS  %d/%d", len(s.Cards), len(cards)), face(fBig, 15*u), C.right.x+12*u, C.right.y+12*u, alLeft, colW, math.Round(1.5*u))
	say(dst, "ALL UNLOCKED CARDS ARE ACTIVE", face(fSmall, 6*u), C.right.x+C.right.w-12*u, C.right.y+12*u, alRight, soft, math.Round(u))
	owned := map[string]bool{}
	for _, id := range s.Cards {
		owned[id] = true
	}
	for i, r := range C.tabs {
		rr := cardRarities[i]
		n, have := 0, 0
		for _, k := range cardsOf(i) {
			n++
			if owned[cards[k].id] {
				have++
			}
		}
		sk := skin{mul(rr.sk.b, 0.45), mul(rr.sk.l, 0.55), mul(rr.sk.s, 0.6)}
		if i == g.cardTab {
			sk = rr.sk
		}
		bevel(dst, r, sk, math.Max(1, p-1))
		say(dst, fmt.Sprintf("%s %d/%d", rr.name[:min(4, len(rr.name))], have, n), face(fSmall, 6*u), r.cx(), r.cy(), alCenter, colW, math.Round(u))
	}
	for k, r := range C.cards {
		i := C.shown[k]
		c := &cards[i]
		if !owned[c.id] {
			fill(dst, r.x, r.y, r.w, r.h, color.NRGBA{10, 5, 25, 160})
			vector.StrokeRect(dst, float32(r.x+u), float32(r.y+u), float32(r.w-2*u), float32(r.h-2*u), float32(math.Max(1, u)), color.NRGBA{160, 140, 220, 90}, false)
			say(dst, "?", face(fBig, 14*u), r.cx(), r.cy(), alCenter, rgb(0x6a6090), 0)
		} else {
			g.drawCardFace(dst, r, c, math.Max(1, p-1), false)
		}
		if g.sel == i {
			vector.StrokeRect(dst, float32(r.x-u), float32(r.y-u), float32(r.w+2*u), float32(r.h+2*u), float32(math.Max(1, 2*u)), colW, false)
		}
	}
	if g.sel >= 0 && g.sel < len(cards) {
		c := &cards[g.sel]
		if owned[c.id] {
			say(dst, c.name, face(fSmall, 8*u), C.info.x, C.info.y+8*u, alLeft, colW, math.Round(u))
			say(dst, cardRarities[c.rar].name+" "+cardTypes[c.typ].name, face(fSmall, 6*u), C.info.x+C.info.w, C.info.y+8*u, alRight, rgb(cardTypes[c.typ].col), math.Round(u))
			g.wrapSay(dst, c.desc, C.info.x, C.info.y+22*u, C.info.w, 8, colT)
		} else {
			say(dst, "LOCKED "+cardRarities[c.rar].name+" "+cardTypes[c.typ].name+" CARD", face(fSmall, 7*u), C.info.x, C.info.y+8*u, alLeft, soft, math.Round(u))
			g.wrapSay(dst, "Won by clearing floors: one card every 10 floors past your best.", C.info.x, C.info.y+22*u, C.info.w, 7, soft)
		}
	}
}
