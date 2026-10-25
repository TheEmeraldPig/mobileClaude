package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// The shop: permanent x1.5 boosts, time skips, bundles, No Ads and paying to skip
// ahead (rebirth now, skip 10 floors). Every item has a dollar price and can also be
// bought with tokens, at 1000 tokens per $1.
//
// Real-money payment and ads go through the Purchase and ShowAd hooks, which the
// Android app sets once Google Play Billing and AdMob are added. Until then the
// dollar buttons say payments aren't connected, and an ad is a short placeholder
// before its reward.

const tokensPerDollar = 1000

// Purchase asks the store to charge for the item sku and calls done with whether it
// went through. ShowAd plays a rewarded ad and calls done with whether it was
// watched. Either may call done from any goroutine, now or later.
var (
	Purchase func(sku string, done func(ok bool))
	ShowAd   func(done func(ok bool))
)

const (
	tabBoosts = iota
	tabSkips
	tabBundles
	tabMore
)

var shopTabs = []string{"BOOSTS", "SKIPS", "BUNDLES", "MORE"}

type shopItem struct {
	id, name, desc string
	cents          int
	tab            int
	perks          []string    // permanent perks it gives; owned once you have them all
	give           func(*Game) // what else it does
}

// the five x1.5 boosts
var boostPerks = []string{"xmoney", "xluck", "xdna", "xtokens", "xpoints"}

var shopItems = []shopItem{
	{"xmoney", "Money x1.5", "All biote value x1.5 forever", 499, tabBoosts, []string{"xmoney"}, nil},
	{"xluck", "Mutations x1.5", "Every mutation chance x1.5 forever", 499, tabBoosts, []string{"xluck"}, nil},
	{"xdna", "DNA x1.5", "Rebirths pay 1.5x DNA forever", 499, tabBoosts, []string{"xdna"}, nil},
	{"xtokens", "Tokens x1.5", "Rebirths pay 1.5x tokens forever", 499, tabBoosts, []string{"xtokens"}, nil},
	{"xpoints", "Skill Points x1.5", "Rebirths pay 1.5x skill points forever", 499, tabBoosts, []string{"xpoints"}, nil},
	{"golden", "Golden Biotes", "All biote value x2 forever", 499, tabBoosts, []string{"golden"}, nil},
	{"offline2", "Night Watch", "Idle gains x2 forever", 399, tabBoosts, []string{"offline2"}, nil},

	{"skip1", "Time Skip 1H", "1 hour of income, right now", 99, tabSkips, nil, func(g *Game) { g.timeSkip(3600) }},
	{"skip4", "Time Skip 4H", "4 hours of income, right now", 299, tabSkips, nil, func(g *Game) { g.timeSkip(4 * 3600) }},
	{"skip24", "Time Skip 24H", "24 hours of income, right now", 799, tabSkips, nil, func(g *Game) { g.timeSkip(24 * 3600) }},
	{"rebirth", "Instant Rebirth", "Rebirth now, without the money", 99, tabSkips, nil, func(g *Game) { g.rebirth(true) }},
	{"floors", "Skip 10 Floors", "Jump 10 floors ahead: DNA and cards", 249, tabSkips, nil, func(g *Game) { g.skipFloors(10) }},

	{"starter", "Starter Pack", "2 skill points, a 4h skip and a card", 199, tabBundles, []string{"starter"}, func(g *Game) {
		g.st.Points += 2
		g.timeSkip(4 * 3600)
		g.awardCard()
	}},
	{"evolve", "Evolution Pack", "Money x1.5 and Mutations x1.5", 799, tabBundles, []string{"xmoney", "xluck"}, nil},
	{"galaxy", "Galaxy Pack", "All five x1.5 boosts and No Ads", 1999, tabBundles, append(append([]string{}, boostPerks...), "noads"), nil},

	{"noads", "No Ads", "Every ad reward is instant, no ad", 999, tabMore, []string{"noads"}, nil},
	{"point", "Skill Point", "+1 skill point", 99, tabMore, nil, func(g *Game) { g.st.Points++ }},
	{"card", "Card Pack", "Unlock a random new card", 299, tabMore, nil, func(g *Game) { g.awardCard() }},
}

func (it *shopItem) tokens() int { return it.cents * tokensPerDollar / 100 }

func (it *shopItem) price() string { return fmt.Sprintf("$%d.%02d", it.cents/100, it.cents%100) }

func shopByID(id string) *shopItem {
	for i := range shopItems {
		if shopItems[i].id == id {
			return &shopItems[i]
		}
	}
	return nil
}

// shopTab is the items on one tab of the shop, as indexes into shopItems.
func shopTab(tab int) []int {
	var out []int
	for i, it := range shopItems {
		if it.tab == tab {
			out = append(out, i)
		}
	}
	return out
}

func (g *Game) owned(it *shopItem) bool {
	if len(it.perks) == 0 {
		return false
	}
	for _, p := range it.perks {
		if !g.st.Perks[p] {
			return false
		}
	}
	return true
}

// buyItem buys an item with real money (through Purchase) or with tokens.
func (g *Game) buyItem(id string, real bool) {
	it := shopByID(id)
	if it == nil || g.owned(it) {
		g.play("deny")
		return
	}
	if real {
		if Purchase == nil {
			g.notice("PAYMENTS ARRIVE WITH THE GOOGLE PLAY RELEASE. USE TOKENS FOR NOW.")
			g.play("deny")
			return
		}
		Purchase(id, func(ok bool) {
			if ok {
				g.paid <- id
			}
		})
		return
	}
	if g.st.Tokens < it.tokens() {
		g.notice(fmt.Sprintf("NEEDS %s TOKENS", itoa(it.tokens())))
		g.play("deny")
		return
	}
	g.st.Tokens -= it.tokens()
	g.grant(it)
}

// grant gives what an item was bought for.
func (g *Game) grant(it *shopItem) {
	for _, p := range it.perks {
		g.st.Perks[p] = true
	}
	if it.give != nil {
		it.give(g)
	}
	g.confetti(60)
	g.floats = append(g.floats, floater{s: upper(it.name) + "!", x: g.W / 2, y: g.H * 0.4, c: colT, size: 26})
	g.play("rebirth")
	g.save()
}

// timeSkip pays sec seconds of full income at once.
func (g *Game) timeSkip(sec float64) {
	gain := g.incomeRate() * sec
	g.st.Money += gain
	g.st.Earned += gain
	g.floats = append(g.floats, floater{s: "+$" + fmtNum(gain), x: g.W / 2, y: g.H * 0.55, c: colG, size: 34})
	g.soldT = g.t
}

// skipFloors jumps n floors ahead, as if they were cleared: DNA at the next rebirth
// and a card for every tenth floor past your best.
func (g *Game) skipFloors(n int) {
	s := &g.st
	for k := 0; k < n; k++ {
		s.Floor++
		s.RunFloors++
		if c := s.Floor - 1; c > s.BestFloor {
			s.BestFloor = c
			if c%floorsPerCard == 0 {
				g.awardCard()
			}
		}
	}
	s.Room = 0
	s.Boss = rand.Intn(len(bosses))
	g.resetRoom()
}

// ---- ads ----

const adLen = 3.0 // seconds the placeholder ad runs

type adState struct {
	on     bool
	t      float64
	reward func()
}

// watchAd shows a rewarded ad, then gives reward. With No Ads the reward is instant.
func (g *Game) watchAd(reward func()) {
	if g.st.Perks["noads"] {
		reward()
		g.play("rebirth")
		g.save()
		return
	}
	g.ad = adState{on: true, t: g.t, reward: reward}
	if ShowAd != nil {
		ShowAd(func(ok bool) { g.adDone <- ok })
	}
}

func (g *Game) finishAd(ok bool) {
	r := g.ad.reward
	g.ad = adState{}
	if ok && r != nil {
		r()
		g.play("rebirth")
		g.save()
	}
}

// tickShop takes in purchases and ads that finished, and ends a placeholder ad.
func (g *Game) tickShop() {
	for {
		select {
		case id := <-g.paid:
			if it := shopByID(id); it != nil {
				g.grant(it)
			}
			continue
		case ok := <-g.adDone:
			g.finishAd(ok)
			continue
		default:
		}
		break
	}
	if g.ad.on && ShowAd == nil && g.t-g.ad.t >= adLen {
		g.finishAd(true)
	}
}

// drawAd covers the screen while an ad runs.
func (g *Game) drawAd(dst *ebiten.Image) {
	if !g.ad.on {
		return
	}
	u := g.u
	fill(dst, 0, 0, g.W, g.H, color.NRGBA{5, 2, 15, 235})
	r := rect{g.W/2 - 150*u, g.H/2 - 80*u, 300 * u, 160 * u}
	bevel(dst, r, skDark, math.Max(1, math.Round(u*1.3)))
	say(dst, "ADVERTISEMENT", face(fBig, 20*u), r.cx(), r.y+24*u, alCenter, colT, math.Round(2*u))
	if ShowAd != nil {
		say(dst, "LOADING...", face(fBig, 18*u), r.cx(), r.cy(), alCenter, colW, math.Round(1.5*u))
		return
	}
	left := math.Ceil(adLen - (g.t - g.ad.t))
	say(dst, fmt.Sprintf("REWARD IN %.0f", left), face(fBig, 26*u), r.cx(), r.cy(), alCenter, colW, math.Round(2*u))
	k := math.Min(1, (g.t-g.ad.t)/adLen)
	bar := rect{r.x + 20*u, r.y + r.h - 46*u, r.w - 40*u, 10 * u}
	fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
	fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k, bar.h-2*u, colG)
	say(dst, "AD PLACEHOLDER: ADS COME WITH THE GOOGLE PLAY RELEASE", face(fSmall, 6*u), r.cx(), r.y+r.h-18*u, alCenter, rgb(0xd8ccff), math.Round(u))
}

// adTag is the little "AD" (or "FREE" with No Ads) mark on buttons that play an ad.
func (g *Game) adTag(dst *ebiten.Image, x, y float64) {
	u := g.u
	label := "AD"
	if g.st.Perks["noads"] {
		label = "FREE"
	}
	w := textW(label, face(fSmall, 6*u)) + 8*u
	r := rect{x - w/2, y - 6*u, w, 12 * u}
	bevel(dst, r, skin{rgb(0x2a9fd6), rgb(0x5fe8ff), rgb(0x1a5a8a)}, math.Max(1, math.Round(u)))
	say(dst, label, face(fSmall, 6*u), r.cx(), r.cy(), alCenter, colW, math.Round(u))
}
