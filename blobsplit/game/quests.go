package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Quests reward active play with skill points, DNA and tokens. Three daily quests
// (one for each reward) are picked from the date and reset at local midnight;
// claiming all three opens a bonus. A quick quest is always running too: claim it
// and a new one comes after a short wait.

// Things quests count.
const (
	qSplit   = iota // split N times
	qSell           // sell N times
	qMutate         // gain N mutations
	qRoom           // clear N rooms
	qVisitor        // catch N visitors
	qPlay           // play N minutes
	qUpgrade        // buy N upgrade levels
	qRare           // create a biote rarer than 1 in N
	nQuestKinds
)

// Rewards.
const (
	rwPoint = iota
	rwDNA
	rwTokens
)

type quest struct {
	Kind    int     `json:"kind"`
	Target  float64 `json:"target"`
	Have    float64 `json:"have"`
	Reward  int     `json:"reward"`
	Amount  int     `json:"amount"`
	Claimed bool    `json:"claimed"`
}

// daily targets, then quick-quest targets, for each kind
var questTargets = [nQuestKinds][2]float64{
	qSplit: {250, 40}, qSell: {60, 10}, qMutate: {25, 4}, qRoom: {8, 1},
	qVisitor: {3, 1}, qPlay: {30, 5}, qUpgrade: {40, 8}, qRare: {1000, 100},
}

const (
	quickWait   = 5 * 60 // seconds between quick quests
	bonusTokens = 50
	bonusDNA    = 5
)

func (q quest) done() bool { return q.Have >= q.Target }

func (q quest) describe() string {
	n := fmtNum(q.Target)
	switch q.Kind {
	case qSplit:
		return "SPLIT " + n + " TIMES"
	case qSell:
		return "SELL " + n + " TIMES"
	case qMutate:
		return "GAIN " + n + " MUTATIONS"
	case qRoom:
		if q.Target == 1 {
			return "CLEAR A ROOM"
		}
		return "CLEAR " + n + " ROOMS"
	case qVisitor:
		if q.Target == 1 {
			return "CATCH A VISITOR"
		}
		return "CATCH " + n + " VISITORS"
	case qPlay:
		return "PLAY FOR " + n + " MINUTES"
	case qUpgrade:
		return "BUY " + n + " UPGRADE LEVELS"
	case qRare:
		return "MAKE A BIOTE RARER THAN 1/" + n
	}
	return ""
}

func (q quest) rewardText() string {
	switch q.Reward {
	case rwPoint:
		if q.Amount == 1 {
			return "+1 SKILL POINT"
		}
		return fmt.Sprintf("+%d SKILL POINTS", q.Amount)
	case rwDNA:
		return fmt.Sprintf("+%d DNA", q.Amount)
	}
	return fmt.Sprintf("+%d TOKENS", q.Amount)
}

func rewardIcon(r int) string { return []string{"star", "dna", "token"}[r] }

// today is the local date as a day number, so quests reset at the player's midnight.
func today() int {
	y, m, d := time.Now().Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// untilMidnight is the seconds left until the daily quests reset.
func untilMidnight() float64 {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location()).Sub(now).Seconds()
}

// dnaReward grows with your best floor, so DNA quests stay worth doing.
func (g *Game) dnaReward(quick bool) int {
	if quick {
		return 1 + g.st.BestFloor/20
	}
	return 3 + g.st.BestFloor/5
}

// checkQuests makes today's quests if the day has changed, and a quick quest if one
// is due.
func (g *Game) checkQuests() {
	s := &g.st
	if d := today(); s.QuestDay != d || len(s.Quests) != 3 {
		s.QuestDay, s.QuestBonus = d, false
		rng := rand.New(rand.NewSource(int64(d)))
		kinds := rng.Perm(nQuestKinds)[:3]
		s.Quests = nil
		for i, k := range kinds {
			q := quest{Kind: k, Target: questTargets[k][0], Reward: i}
			q.Amount = []int{1, g.dnaReward(false), 40}[i]
			s.Quests = append(s.Quests, q)
		}
	}
	if s.Quick == nil && s.QuickWait <= 0 {
		k := rand.Intn(nQuestKinds)
		if k == qVisitor {
			k = qSplit // visitors are too rare for a five-minute quest
		}
		q := quest{Kind: k, Target: questTargets[k][1], Reward: rwDNA, Amount: g.dnaReward(true)}
		if rand.Intn(2) == 0 {
			q.Reward, q.Amount = rwTokens, 10
		}
		s.Quick = &q
	}
}

// questAdd counts n of something towards the running quests.
func (g *Game) questAdd(kind int, n float64) {
	s := &g.st
	add := func(q *quest) {
		if q.Kind != kind || q.Claimed {
			return
		}
		was := q.done()
		if kind == qRare {
			q.Have = math.Max(q.Have, n) // n is how rare the biote was
		} else {
			q.Have += n
		}
		if !was && q.done() {
			y := g.H * 0.22
			for _, f := range g.floats {
				if f.y >= y && f.t < 1 {
					y = f.y + 20*g.u // below any message still showing
				}
			}
			g.floats = append(g.floats, floater{s: "QUEST DONE! " + q.describe(), x: g.W / 2, y: y, c: colT, size: 16})
			g.play("buy")
		}
	}
	for i := range s.Quests {
		add(&s.Quests[i])
	}
	if s.Quick != nil {
		add(s.Quick)
	}
}

// claimable is how many finished quests are waiting to be claimed (for the badge).
func (g *Game) claimable() int {
	n := 0
	all := true
	for _, q := range g.st.Quests {
		if q.done() && !q.Claimed {
			n++
		}
		all = all && q.Claimed
	}
	if q := g.st.Quick; q != nil && q.done() {
		n++
	}
	if all && !g.st.QuestBonus && len(g.st.Quests) > 0 {
		n++
	}
	return n
}

func (g *Game) payReward(r, n int) {
	switch r {
	case rwPoint:
		g.st.Points += n
	case rwDNA:
		g.st.Gems += n
	default:
		g.st.Tokens += n
	}
}

// claimQuest claims daily quest i, or the quick quest (i == 3), or the bonus (4).
func (g *Game) claimQuest(i int) {
	s := &g.st
	switch {
	case i < len(s.Quests):
		q := &s.Quests[i]
		if q.Claimed || !q.done() {
			g.play("deny")
			return
		}
		q.Claimed = true
		g.payReward(q.Reward, q.Amount)
	case i == 3:
		if s.Quick == nil || !s.Quick.done() {
			g.play("deny")
			return
		}
		g.payReward(s.Quick.Reward, s.Quick.Amount)
		s.Quick, s.QuickWait = nil, quickWait
	default:
		for _, q := range s.Quests {
			if !q.Claimed {
				g.play("deny")
				return
			}
		}
		if s.QuestBonus {
			g.play("deny")
			return
		}
		s.QuestBonus = true
		g.payReward(rwTokens, bonusTokens)
		g.payReward(rwDNA, bonusDNA)
		g.confetti(80)
	}
	g.confetti(30)
	g.play("rebirth")
	g.save()
}

// tickQuests counts play time and the wait for the next quick quest.
func (g *Game) tickQuests() {
	s := &g.st
	if g.frame%60 == 0 {
		g.checkQuests()
	}
	if s.QuickWait > 0 {
		s.QuickWait -= dt
	}
	g.playT += dt
	if g.playT >= 60 {
		g.playT -= 60
		g.questAdd(qPlay, 1)
	}
}

// ---- the QUESTS panel ----

func (g *Game) layQuests(L *layout, R func(x, y, w, h float64) rect, px, py, pw float64) {
	for i := 0; i < 4; i++ {
		y := py + 50 + float64(i)*50
		L.rows = append(L.rows, R(px+12, y, pw-24, 44))
		L.btns = append(L.btns, R(px+pw-12-8-96, y+8, 96, 28))
	}
	L.action = R(px+12, py+252, pw-24, 40) // the bonus
}

func (g *Game) drawQuests(dst *ebiten.Image, L layout) {
	u, p := g.u, L.p
	s := &g.st
	soft := rgb(0xd8ccff)
	say(dst, "NEW IN "+fmtDur(untilMidnight()), face(fSmall, 7*u), L.panel.x+16*u, L.panel.y+22*u, alLeft, soft, math.Round(u))
	row := func(k int, q *quest, title string) {
		r := L.rows[k]
		bevel(dst, r, skViolet, p)
		say(dst, title, face(fSmall, 6*u), r.x+10*u, r.y+9*u, alLeft, colT, math.Round(u))
		if q == nil {
			say(dst, "NEXT QUICK QUEST IN "+fmtTime(s.QuickWait), face(fSmall, 8*u), r.x+10*u, r.y+25*u, alLeft, soft, math.Round(u))
			return
		}
		say(dst, q.describe(), face(fSmall, 8*u), r.x+10*u, r.y+22*u, alLeft, colW, math.Round(u))
		// progress
		bw := r.w - 10*u - 230*u
		k2 := math.Min(1, q.Have/q.Target)
		label := fmtNum(math.Min(q.Have, q.Target)) + "/" + fmtNum(q.Target)
		if q.Kind == qRare {
			k2, label = 0, "NOT YET"
			if q.done() {
				k2, label = 1, "DONE"
			}
		}
		bar := rect{r.x + 10*u, r.y + 32*u, bw, 6 * u}
		fill(dst, bar.x, bar.y, bar.w, bar.h, colO)
		fill(dst, bar.x+u, bar.y+u, (bar.w-2*u)*k2, bar.h-2*u, colG)
		say(dst, label, face(fSmall, 6*u), bar.x+bar.w+6*u, bar.cy(), alLeft, colW, math.Round(u))
		// reward and claim button
		icon(dst, rewardIcon(q.Reward), r.x+r.w-120*u, r.cy(), 14*u)
		say(dst, q.rewardText(), face(fSmall, 7*u), r.x+r.w-130*u, r.cy(), alRight, colT, math.Round(u))
		b := g.pop(L.btns[k])
		switch {
		case q.Claimed:
			bevel(dst, b, skDark, math.Max(1, p-1))
			say(dst, "CLAIMED", face(fSmall, 8*u), b.cx(), b.cy(), alCenter, soft, math.Round(u))
		case q.done():
			bevel(dst, grow(b, 1+0.05*math.Abs(math.Sin(g.t*5))), skGreen, math.Max(1, p-1))
			say(dst, "CLAIM", face(fBig, 15*u), b.cx(), b.cy()-u, alCenter, colW, math.Round(1.5*u))
		default:
			bevel(dst, b, skGrey, math.Max(1, p-1))
			say(dst, "IN PROGRESS", face(fSmall, 6*u), b.cx(), b.cy(), alCenter, colW, math.Round(u))
		}
	}
	for i := range s.Quests {
		row(i, &s.Quests[i], "DAILY")
	}
	row(3, s.Quick, "QUICK QUEST")

	// the bonus for finishing all three
	all := len(s.Quests) > 0
	for _, q := range s.Quests {
		all = all && q.Claimed
	}
	b := L.action
	sk := skGrey
	text := fmt.Sprintf("CLAIM ALL 3 DAILY QUESTS FOR A BONUS: +%d TOKENS, +%d DNA", bonusTokens, bonusDNA)
	switch {
	case s.QuestBonus:
		sk, text = skDark, "BONUS CLAIMED. NEW QUESTS AT MIDNIGHT."
	case all:
		sk, text = skGold, fmt.Sprintf("TAP FOR YOUR BONUS: +%d TOKENS, +%d DNA!", bonusTokens, bonusDNA)
		b = grow(b, 1+0.03*math.Abs(math.Sin(g.t*4)))
	}
	bevel(dst, g.pop(b), sk, p)
	say(dst, text, face(fSmall, 8*u), b.cx(), b.cy(), alCenter, colW, math.Round(u))
	if all && !s.QuestBonus {
		g.shimmer(dst, b, 0)
	}
}
