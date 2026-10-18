package game

import (
	"encoding/json"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Kitchen Clash: your chefs fight rotten food. The crew throws food on its own,
// tapping the monster hits too, skills charge up, and every 10 monsters a boss
// arrives with a timer. Monsters drop ingredients; the kitchen turns them into
// dishes that power up the fight, more so in combos.

var timing = os.Getenv("KITCHEN_TIMING") != ""

// scaleOverride (KITCHEN_SCALE) sets the device scale factor, for screenshots.
var scaleOverride, _ = strconv.ParseFloat(os.Getenv("KITCHEN_SCALE"), 64)

const dt = 1.0 / 60

type mob struct {
	kind   int
	boss   bool
	hp     float64
	max    float64
	hit    float64 // white flash, fades
	hurt   int     // frames of the hurt face
	dead   float64 // dying animation 0..1, 0 = alive
	gone   bool    // a boss that got away
	phase  float64
	coins  float64
	spawn  float64 // drop-in 0..1
	squish float64
	trail  float64 // health bar damage trail
}

type shot struct {
	x0, y0, x1, y1 float64
	t, dur         float64
	dmg            float64
	icon           int
	crit           bool
	big            float64
	spin           float64
}

type floater struct {
	x, y, t, life, size float64
	s                   string
	col                 color.RGBA
}

const (
	ptPuff = iota
	ptSpark
	ptCoin
	ptStar
	ptGoo
	ptConfetti
	ptIng   // an ingredient flying to the kitchen tab
	ptSteam // steam over a pot
)

type particle struct {
	x, y, vx, vy, g float64
	t, life, size   float64
	rot, vr         float64
	kind            int
	col             color.RGBA
	tx, ty          float64
	icon            int
}

type chefState struct {
	atk    float64 // throw animation 0..1, <0 idle
	cool   float64
	charge float64
	blink  int
	fired  bool
	cast   float64
	pop    float64
}

type hit struct {
	r  rect
	id string
}

type pointer struct {
	down, touch, moved bool
	id                 ebiten.TouchID
	sx, sy, x, y       float64
	lastY              float64
	field              bool
}

const (
	tabChefs = iota
	tabCrew
	tabKitchen
	tabShop
	numTabs
)

type Game struct {
	p    *Profile
	w, h float64
	rng  *rand.Rand
	now  func() int64

	frame int
	t     float64

	// the fight
	mon       *mob
	shots     []*shot
	fl        []floater
	parts     []particle
	team      []chefState
	bossT     float64
	bossOn    bool
	frenzyT   float64
	critT     float64
	goldT     float64
	shake     float64
	flash     float64
	spawnCD   float64
	banner    string
	bannerSub string
	bannerT   float64
	coinPulse float64
	gemPulse  float64
	ingPulse  float64
	shownCoin float64

	// interface
	tab       int
	tabT      float64
	scroll    [numTabs]float64
	scrollMax [numTabs]float64
	vel       float64
	mult      int
	hits      []hit
	ptr       pointer
	toast     string
	toastT    float64
	pressedID string
	pressT    map[string]float64
	station   int // selected kitchen station

	// overlays
	hireID    int
	hireT     float64
	hireNew   bool
	hireQueue []int
	welcome   away
	welcomeT  float64
	cookR     int // recipe in the cooking minigame, -1 = none
	cookT     float64
	cookDone  float64 // time the needle was stopped, 0 = still moving
	cookQ     float64
	cookPos   float64

	lastSave float64
	script   *script
}

func NewGame() *Game {
	g := &Game{w: W, h: 960, now: func() int64 { return time.Now().Unix() }, hireID: -1, cookR: -1, pressT: map[string]float64{}}
	seed := time.Now().UnixNano()
	g.script = loadScript()
	if g.script != nil {
		seed = 1
	}
	g.rng = rand.New(rand.NewSource(seed))
	now := g.now()
	g.p = newProfile(now)
	if b, err := readSave(); err == nil {
		p := &Profile{}
		if json.Unmarshal(b, p) == nil {
			p.fix()
			g.p = p
		}
	}
	g.arrive(now)
	g.shownCoin = g.p.Coins
	g.syncTeam()
	return g
}

// arrive pays out the time away.
func (g *Game) arrive(now int64) {
	if a := g.p.offline(now); a.coins > 0 || a.total() > 0 {
		g.welcome, g.welcomeT = a, 0
	}
	g.p.LastSeen = now
}

// Save writes the profile.
func (g *Game) Save() {
	g.p.LastSeen = g.now()
	b, _ := json.Marshal(g.p)
	writeSave(b)
}

func (g *Game) Layout(w, h int) (int, int) {
	fw, fh := g.LayoutF(float64(w), float64(h))
	return int(fw), int(fh)
}

// LayoutF renders at the screen's real pixel size; S maps layout units to pixels.
func (g *Game) LayoutF(w, h float64) (float64, float64) {
	g.w = W
	g.h = math.Max(860, math.Round(h*W/math.Max(w, 1)))
	dsf := ebiten.Monitor().DeviceScaleFactor()
	if scaleOverride > 0 {
		dsf = scaleOverride * W / math.Max(w, 1)
	}
	S = math.Max(0.5, math.Min(3, w*dsf/W))
	return W * S, g.h * S
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 2.6 }

// ---- layout ----

func (g *Game) fieldH() float64  { return math.Round(math.Max(430, g.h*0.5)) }
func (g *Game) groundY() float64 { return g.fieldH() - 46 }
func (g *Game) skillY() float64  { return g.fieldH() }
func (g *Game) panelY() float64  { return g.fieldH() + 96 }
func (g *Game) tabsY() float64   { return g.h - 86 }
func (g *Game) monX() float64    { return W / 2 }

// slotX is the middle of crew slot i in the bar under the field.
func slotX(i int) float64 { return 62 + float64(i)*104 }

func (g *Game) monScale(m *mob) float64 {
	if m.boss {
		return 1.45
	}
	return 1.4
}

func (g *Game) syncTeam() {
	for len(g.team) < len(g.p.Team) {
		g.team = append(g.team, chefState{atk: -1, cool: g.rng.Float64()})
	}
	g.team = g.team[:len(g.p.Team)]
}

// ---- update ----

func (g *Game) Update() error {
	g.frame++
	g.t += dt
	if g.script != nil {
		if err := g.script.step(g); err != nil {
			return err
		}
	}
	g.updatePointer()
	g.syncTeam()
	if g.hireID < 0 && len(g.hireQueue) > 0 && g.cookR < 0 {
		g.hireID, g.hireQueue = g.hireQueue[0], g.hireQueue[1:]
		g.hireT = 0
		g.hireNew = g.p.hire(g.hireID)
		g.syncTeam()
		g.Save()
	}
	if g.hireID >= 0 {
		g.hireT += dt
	}
	if g.welcome.secs > 0 {
		g.welcomeT += dt
	}
	g.updateCook()
	g.p.tickKitchen(dt, !g.modal())
	g.updateField()
	g.stepEffects()
	if g.toastT > 0 {
		g.toastT -= dt
	}
	g.tabT += dt
	for k, v := range g.pressT {
		if v -= dt * 4; v <= 0 {
			delete(g.pressT, k)
		} else {
			g.pressT[k] = v
		}
	}
	if !g.ptr.down && g.vel != 0 {
		g.scrollBy(g.vel)
		g.vel *= 0.92
		if math.Abs(g.vel) < 0.2 {
			g.vel = 0
		}
	}
	g.shownCoin += (g.p.Coins - g.shownCoin) * 0.18
	if math.Abs(g.p.Coins-g.shownCoin) < 1 {
		g.shownCoin = g.p.Coins
	}
	if g.t-g.lastSave > 10 {
		g.lastSave = g.t
		g.Save()
	}
	return nil
}

func (g *Game) modal() bool { return g.hireID >= 0 || g.welcome.secs > 0 || g.cookR >= 0 }

// onField is true when a point is on the fight (not the kitchen view).
func (g *Game) onField(x, y float64) bool {
	return g.tab != tabKitchen && !g.modal() && y < g.fieldH() && y > 70
}

func (g *Game) updatePointer() {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		fx, fy := float64(x)/S, float64(y)/S
		if !g.ptr.down {
			g.ptr = pointer{down: true, touch: true, id: id, sx: fx, sy: fy, x: fx, y: fy, lastY: fy}
			g.ptr.field = g.pointerDown(fx, fy)
		} else if g.onField(fx, fy) && g.hitAt(fx, fy) == "" {
			g.tapField(fx, fy)
		}
	}
	p := &g.ptr
	if !p.down {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			fx, fy := float64(x)/S, float64(y)/S
			*p = pointer{down: true, sx: fx, sy: fy, x: fx, y: fy, lastY: fy}
			p.field = g.pointerDown(fx, fy)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.cookR >= 0 {
			g.stopNeedle()
		}
		if _, wy := ebiten.Wheel(); wy != 0 {
			g.scrollBy(wy * 40)
		}
		return
	}
	released := false
	if p.touch {
		if inpututil.IsTouchJustReleased(p.id) {
			released = true
		} else {
			x, y := ebiten.TouchPosition(p.id)
			p.x, p.y = float64(x)/S, float64(y)/S
		}
	} else {
		x, y := ebiten.CursorPosition()
		p.x, p.y = float64(x)/S, float64(y)/S
		released = !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	}
	if math.Hypot(p.x-p.sx, p.y-p.sy) > 12 && !p.field {
		p.moved = true
	}
	if p.moved {
		dy := p.y - p.lastY
		g.scrollBy(dy)
		g.vel = dy
	}
	p.lastY = p.y
	if released {
		p.down = false
		if !p.moved && !p.field {
			g.click(p.sx, p.sy)
		}
	}
}

// pointerDown handles a fresh press; taps on the monster act at once.
func (g *Game) pointerDown(x, y float64) bool {
	g.vel = 0
	if g.cookR >= 0 {
		g.stopNeedle()
		return true
	}
	if g.onField(x, y) && g.hitAt(x, y) == "" {
		g.tapField(x, y)
		return true
	}
	if id := g.hitAt(x, y); id != "" {
		g.pressedID = id
	}
	return false
}

func (g *Game) hitAt(x, y float64) string {
	for i := len(g.hits) - 1; i >= 0; i-- {
		if g.hits[i].r.has(x, y) {
			return g.hits[i].id
		}
	}
	return ""
}

func (g *Game) click(x, y float64) {
	id := g.hitAt(x, y)
	g.pressedID = ""
	if id == "" {
		if g.modal() {
			g.press("modal")
		}
		return
	}
	g.pressT[id] = 1
	g.press(id)
}

func (g *Game) scrollBy(dy float64) {
	if g.ptr.down && g.ptr.sy < g.panelY() {
		return
	}
	t := g.tab
	g.scroll[t] = math.Max(0, math.Min(g.scrollMax[t], g.scroll[t]-dy))
}

// press runs a button by its id.
func (g *Game) press(id string) {
	name, arg, _ := strings.Cut(id, ":")
	n, _ := strconv.Atoi(arg)
	p := g.p
	fail := func(err error) bool {
		if err != nil {
			g.say(err.Error())
			return true
		}
		return false
	}
	switch name {
	case "modal":
		if g.hireID >= 0 && g.hireT > 2.2 {
			g.hireID = -1
		}
	case "collect", "collect2":
		if name == "collect2" {
			if p.Gems < 30 {
				g.say("Not enough gems")
				return
			}
			p.Gems -= 30
			g.welcome.coins *= 2
			for i := range g.welcome.ing {
				g.welcome.ing[i] *= 2
			}
		}
		g.coinBurst(g.welcome.coins, W/2, g.h/2, 12)
		for i, c := range g.welcome.ing {
			p.Ing[i] += c
		}
		g.welcome = away{}
		g.Save()
	case "tab":
		if g.tab != n {
			g.tab, g.tabT = n, 0
		}
	case "mult":
		g.mult = (g.mult + 1) % 3
	case "up":
		if fail(p.levelUp(n, g.buyCount(n))) {
			return
		}
		g.levelFx(n)
		g.Save()
	case "skill":
		g.useSkill(n)
	case "boss":
		if p.Farming {
			p.Farming = false
			p.Wave = wavesPerStage
		}
	case "team":
		if fail(p.toggleTeam(n)) {
			return
		}
		g.syncTeam()
		for i, t := range p.Team {
			if t == n {
				g.team[i] = chefState{atk: -1, cool: 0.5}
			}
		}
		g.Save()
	case "buy":
		it := shopItems[n]
		c, err := p.buy(it, g.rng)
		if fail(err) {
			return
		}
		if it.price != "" {
			g.say("Test purchase: +" + itoa(it.give) + " gems")
			g.gemPulse = 1
		}
		if it.id == "coins" {
			g.coinPulse = 1
		}
		if it.id == "groceries" {
			g.ingPulse = 1
			g.say("+5 of every ingredient")
		}
		if c >= 0 {
			g.hireQueue = append(g.hireQueue, c)
		}
		g.Save()
	case "free":
		c, err := p.freeBox(g.now(), g.rng)
		if fail(err) {
			return
		}
		g.hireQueue = append(g.hireQueue, c)
		g.Save()
	case "station":
		g.station = n
		g.scroll[tabKitchen] = 0
		if c := p.Cook[n]; c.R >= 0 && c.Left <= 0 {
			if d, err := p.collect(n); !fail(err) {
				g.say(recipes[d.R].name + " is ready to serve!")
				g.Save()
			}
		}
	case "build":
		if fail(p.upgradeStation(n)) {
			return
		}
		g.say(stationName[n] + " is now level " + itoa(p.Stations[n]))
		g.Save()
	case "cook":
		if fail(p.canCook(n)) {
			return
		}
		g.cookR, g.cookT, g.cookDone = n, 0, 0
	case "serve":
		if fail(p.serve(n)) {
			return
		}
		g.ingPulse = 1
		r := p.Table[len(p.Table)-1].R
		msg := recipes[r].name + " served!"
		if p.fullCourse() {
			msg = "Full course! Every dish x1.5"
		}
		for _, ci := range p.activeCombos() {
			for _, d := range combos[ci].dishes {
				if d == r {
					msg = "Combo: " + combos[ci].name
				}
			}
		}
		g.showBanner(msg, effectText(recipes[r].eff, recipes[r].val))
		g.Save()
	}
}

// buyCount is how many levels the x1/x10/max switch buys for chef id.
func (g *Game) buyCount(id int) int {
	switch g.mult {
	case 1:
		return 10
	case 2:
		return g.p.maxLevels(id)
	}
	return 1
}

// ---- cooking minigame ----

// needle is where the swinging needle is, 0..1, for a given time.
func needle(t float64) float64 {
	v := math.Mod(t*1.1, 2)
	if v > 1 {
		v = 2 - v
	}
	return easeInOut(v)
}

func easeInOut(t float64) float64 { return t * t * (3 - 2*t) }

func (g *Game) stopNeedle() {
	if g.cookDone > 0 || g.cookT < 0.3 {
		return
	}
	g.cookPos = needle(g.cookT)
	g.cookDone = g.cookT
	d := math.Abs(g.cookPos - 0.5)
	switch {
	case d < 0.07:
		g.cookQ = 1.5
	case d < 0.22:
		g.cookQ = 1.0
	default:
		g.cookQ = 0.7
	}
}

func (g *Game) updateCook() {
	if g.cookR < 0 {
		return
	}
	g.cookT += dt
	if g.cookDone > 0 && g.cookT-g.cookDone > 1.1 {
		if err := g.p.startCook(g.cookR, g.cookQ); err != nil {
			g.say(err.Error())
		}
		g.station = recipes[g.cookR].station
		g.cookR = -1
		g.Save()
	}
}

func cookWord(q float64) (string, color.RGBA) {
	switch {
	case q >= 1.5:
		return "PERFECT!", rgb(0xff, 0xe0, 0x40)
	case q >= 1:
		return "Good", rgb(0x9c, 0xf0, 0x50)
	}
	return "Okay...", rgb(0xff, 0xa0, 0x60)
}

// ---- the fight ----

func (g *Game) alive() *mob {
	if g.mon != nil && g.mon.dead == 0 {
		return g.mon
	}
	return nil
}

func (g *Game) updateField() {
	p := g.p
	e := p.effects()
	for _, t := range []*float64{&g.frenzyT, &g.critT, &g.goldT, &g.bannerT} {
		if *t > 0 {
			*t -= dt
		}
	}
	g.shake = math.Max(0, g.shake-dt*3)
	g.flash = math.Max(0, g.flash-dt*3)
	g.coinPulse = math.Max(0, g.coinPulse-dt*3)
	g.gemPulse = math.Max(0, g.gemPulse-dt*3)
	g.ingPulse = math.Max(0, g.ingPulse-dt*3)
	// the boss timer
	if g.bossOn {
		if !g.modal() {
			g.bossT -= dt
		}
		if m := g.alive(); m != nil && m.boss && g.bossT <= 0 {
			m.gone = true
			m.dead = 0.001
			g.bossOn = false
			p.loseBoss()
			g.showBanner("Out of time!", "Level up your chefs, cook up buffs, try again")
			g.Save()
		}
	}
	// the next monster
	m := g.mon
	if m != nil && m.dead > 0 {
		m.dead += dt * 1.8
		if m.dead >= 1 {
			g.mon, m = nil, nil
			g.spawnCD = 0.25
		}
	}
	if m == nil {
		g.spawnCD -= dt
		if g.spawnCD <= 0 {
			if !p.Farming && p.Wave >= wavesPerStage {
				g.bossOn = true
				g.bossT = bossTime + e[efBossTime]
				k := (p.Stage - 1) % len(bosses)
				hp := bossHP(p.Stage)
				g.mon = &mob{kind: k, boss: true, hp: hp, max: hp, trail: hp, coins: bossCoins(p.Stage)}
				g.showBanner("Boss!", bosses[k].name)
			} else {
				fam := areaMonsters[area(p.Stage)%len(areaMonsters)]
				k := fam[g.rng.Intn(len(fam))]
				hp := monHP(p.Stage)
				g.mon = &mob{kind: k, hp: hp, max: hp, trail: hp, coins: monCoins(p.Stage), phase: g.rng.Float64() * 6}
			}
			m = g.mon
		}
	}
	if m != nil && m.dead == 0 {
		m.spawn = math.Min(1, m.spawn+dt*2.4)
		m.phase += dt * 2.6
		m.hit = math.Max(0, m.hit-dt*6)
		m.squish *= 0.9
		if m.hurt > 0 {
			m.hurt--
		}
		m.trail += (m.hp - m.trail) * 0.06
	}
	// the crew throws food
	ready := g.alive() != nil && g.alive().spawn > 0.6
	speed := 1.0
	if g.frenzyT > 0 {
		speed = 2
	}
	for i := range g.team {
		b := &g.team[i]
		id := p.Team[i]
		c := chefs[id]
		b.pop = math.Min(1, b.pop+dt*2)
		if ready {
			b.charge = math.Min(1, b.charge+dt/c.cool*(1+e[efSkill]/100))
		}
		if b.cast > 0 {
			if b.cast += dt * 1.6; b.cast >= 1 {
				b.cast = 0
			}
		}
		if b.blink > 0 {
			b.blink--
		} else if g.rng.Float64() < 0.006 {
			b.blink = 8
		}
		if b.atk >= 0 {
			b.atk += dt * 3.2 * math.Max(1, speed*0.8)
			if b.atk > 0.4 && !b.fired {
				b.fired = true
				if t := g.alive(); t != nil {
					crit := g.rng.Float64() < 0.05+e[efCrit]/100
					dmg := p.dps(id) * c.period
					if crit {
						dmg *= 3
					}
					g.fire(slotX(i), g.skillY()+20, dmg, c.shot, crit, 1)
				}
			}
			if b.atk >= 1 {
				b.atk = -1
			}
			continue
		}
		b.cool -= dt * speed
		if b.cool <= 0 && ready && !g.modal() {
			b.cool = c.period * (0.92 + g.rng.Float64()*0.16)
			b.atk, b.fired = 0, false
		}
	}
	// shots fly
	keep := g.shots[:0]
	for _, s := range g.shots {
		s.t += dt / s.dur
		if s.t >= 1 {
			g.land(s)
			continue
		}
		keep = append(keep, s)
	}
	g.shots = keep
}

func (s *shot) pos() (float64, float64) {
	t := s.t
	x := s.x0 + (s.x1-s.x0)*t
	y := s.y0 + (s.y1-s.y0)*t - math.Sin(t*math.Pi)*90
	return x, y
}

// fire throws a food item from x,y at the monster.
func (g *Game) fire(x, y, dmg float64, icon int, crit bool, big float64) {
	m := g.alive()
	if m == nil {
		return
	}
	sc := g.monScale(m)
	tx := g.monX() + (g.rng.Float64()-0.5)*80*sc
	ty := g.groundY() - (60+g.rng.Float64()*80)*sc
	g.shots = append(g.shots, &shot{x0: x, y0: y, x1: tx, y1: ty, dur: 0.42, dmg: dmg, icon: icon, crit: crit, big: big, spin: (g.rng.Float64() - 0.5) * 20})
}

func (g *Game) land(s *shot) {
	m := g.alive()
	if m == nil {
		return
	}
	dmg := s.dmg
	if m.boss {
		dmg *= 1 + g.p.effects()[efBossDmg]/100
	}
	g.damage(m, dmg, s.crit, s.x1, s.y1)
	for k := 0; k < 6; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		g.parts = append(g.parts, particle{x: s.x1, y: s.y1, vx: math.Cos(a) * 3, vy: math.Sin(a)*3 - 1, g: 0.2, life: 0.45, size: 0.5 * s.big, kind: ptGoo, col: monOf(m.kind, m.boss).col})
	}
	g.parts = append(g.parts, particle{x: s.x1, y: s.y1, life: 0.25, size: 1.2 * s.big, kind: ptSpark, col: colWhite})
}

// damage hurts the monster and handles it being beaten.
func (g *Game) damage(m *mob, dmg float64, crit bool, x, y float64) {
	if m.dead > 0 {
		return
	}
	m.hp -= dmg
	m.hit = 1
	m.hurt = 14
	m.squish = 1
	col, size := colWhite, 24.0
	if crit {
		col, size = rgb(0xff, 0xd8, 0x30), 34
	}
	g.fl = append(g.fl, floater{x: x + (g.rng.Float64()-0.5)*30, y: y - 10, life: 0.9, size: size, s: num(dmg), col: col})
	if m.hp > 0 {
		return
	}
	m.hp = 0
	m.dead = 0.001
	p := g.p
	sc := g.monScale(m)
	cx, cy := g.monX(), g.groundY()-90*sc
	for k := 0; k < 16; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		sp := 1 + g.rng.Float64()*3
		g.parts = append(g.parts, particle{x: cx + math.Cos(a)*50*sc, y: cy + math.Sin(a)*50*sc, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 1, life: 0.6 + g.rng.Float64()*0.3, size: (0.8 + g.rng.Float64()*0.8) * sc, kind: ptPuff})
	}
	// ingredients pop out and fly to the kitchen tab
	mo := monOf(m.kind, m.boss)
	n := p.dropCount(m.boss)
	for i := 0; i < n; i++ {
		ing := mo.drop[g.rng.Intn(len(mo.drop))]
		p.Ing[ing]++
		if i < 8 {
			a := -math.Pi/2 + (g.rng.Float64()-0.5)*2.2
			sp := 4 + g.rng.Float64()*3
			g.parts = append(g.parts, particle{x: cx, y: cy, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, g: 0.3, life: 1.2 + float64(i)*0.05, size: 1, kind: ptIng, icon: ing, tx: tabX(tabKitchen), ty: g.tabsY() + 30})
		}
	}
	if m.boss {
		g.bossOn = false
		box := p.beatBoss()
		g.coinBurst(0, cx, cy, 14)
		if g.goldT > 0 {
			g.coinBurst(bossCoins(p.Stage-1)*2, cx, cy, 0)
		}
		g.gemPulse = 1
		g.shake = 1
		g.flash = 0.8
		for k := 0; k < 60; k++ {
			a := g.rng.Float64() * 2 * math.Pi
			sp := 3 + g.rng.Float64()*6
			g.parts = append(g.parts, particle{x: cx, y: cy, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 5, g: 0.22, life: 1.6, size: 1, kind: ptConfetti,
				col: []color.RGBA{colGold, colPink, colBlue, colGreen, colPurple}[k%5], rot: a, vr: (g.rng.Float64() - 0.5) * 0.4})
		}
		if area(p.Stage) != area(p.Stage-1) {
			g.showBanner("New area!", areaOf(p.Stage).name)
		} else {
			g.showBanner("Stage clear!", "+"+itoa(bossGems(p.Stage-1))+" gems")
		}
		if box {
			g.hireQueue = append(g.hireQueue, rollChef(g.rng, boxes[0].weights))
		}
		g.Save()
		return
	}
	coins := m.coins * p.coinMult()
	if g.goldT > 0 {
		coins *= 3
	}
	p.beatMonster(0)
	g.coinBurst(coins, cx, cy, 5)
}

// coinBurst pays coins out as flying coins.
func (g *Game) coinBurst(v, x, y float64, n int) {
	g.p.Coins += v
	for i := 0; i < n; i++ {
		a := -math.Pi/2 + (g.rng.Float64()-0.5)*2
		sp := 3 + g.rng.Float64()*3
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, g: 0.3,
			life: 1.0 + float64(i)*0.04, size: 1, kind: ptCoin, tx: 30, ty: 33})
	}
}

// tapField is a finger on the fight: a direct hit on the monster.
func (g *Game) tapField(x, y float64) {
	g.p.Taps++
	for k := 0; k < 6; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * 3, vy: math.Sin(a) * 3, life: 0.3, size: 0.7, kind: ptStar, col: colGold, rot: a})
	}
	g.parts = append(g.parts, particle{x: x, y: y, life: 0.25, size: 1.4, kind: ptSpark, col: colWhite})
	m := g.alive()
	if m == nil || m.spawn < 0.5 {
		return
	}
	e := g.p.effects()
	crit := g.rng.Float64() < 0.08+e[efCrit]/100 || g.critT > 0
	dmg := g.p.tapDamage()
	if crit {
		dmg *= 4
	}
	if m.boss {
		dmg *= 1 + e[efBossDmg]/100
	}
	g.damage(m, dmg, crit, x, y)
	for i := range g.team {
		g.team[i].charge = math.Min(1, g.team[i].charge+0.004)
	}
}

func (g *Game) useSkill(i int) {
	if i >= len(g.team) {
		return
	}
	b := &g.team[i]
	if b.charge < 1 {
		g.say("Still charging")
		return
	}
	m := g.alive()
	if m == nil {
		g.say("Wait for the next monster")
		return
	}
	id := g.p.Team[i]
	c := chefs[id]
	b.charge = 0
	b.cast = 0.001
	dmg := g.p.dps(id) * c.skDmg
	n := 3
	switch c.skKind {
	case skMulti:
		n = 8
	case skFrenzy:
		g.frenzyT = 8
	case skCrit:
		g.critT = 8
	case skTime:
		if g.bossOn {
			g.bossT += 6
		}
	case skGold:
		g.goldT = 15
	}
	if id == 3 && m.boss {
		dmg *= 3
	}
	for k := 0; k < n; k++ {
		g.fire(slotX(i)+(g.rng.Float64()-0.5)*30, g.skillY()+10, dmg/float64(n), c.shot, true, 1.8)
		g.shots[len(g.shots)-1].dur += float64(k) * 0.06
	}
	g.shake = 0.6
	g.showBanner(c.skill, "")
}

func (g *Game) levelFx(id int) {
	for i, t := range g.p.Team {
		if t == id {
			x, y := slotX(i), g.skillY()+46
			g.team[i].cast = 0.001
			for k := 0; k < 12; k++ {
				a := -math.Pi/2 + (g.rng.Float64()-0.5)*2.4
				g.parts = append(g.parts, particle{x: x + (g.rng.Float64()-0.5)*60, y: y, vx: math.Cos(a), vy: math.Sin(a)*2 - 1.5, life: 0.8, size: 0.7, kind: ptStar, col: colGreen})
			}
			g.fl = append(g.fl, floater{x: x, y: y - 50, life: 1, size: 20, s: "Level up!", col: rgb(0xb0, 0xff, 0x60)})
		}
	}
}

func (g *Game) showBanner(s, sub string) { g.banner, g.bannerSub, g.bannerT = s, sub, 2.2 }

func (g *Game) stepEffects() {
	keep := g.parts[:0]
	for _, q := range g.parts {
		q.t += dt
		if q.t >= q.life {
			switch q.kind {
			case ptCoin:
				g.coinPulse = 1
			case ptIng:
				g.ingPulse = 1
			}
			continue
		}
		if (q.kind == ptCoin || q.kind == ptIng) && q.t > 0.45 {
			k := easeIn((q.t - 0.45) / (q.life - 0.45))
			q.x += (q.tx - q.x) * (0.08 + k*0.5)
			q.y += (q.ty - q.y) * (0.08 + k*0.5)
		} else {
			q.vy += q.g
			q.x += q.vx
			q.y += q.vy
			q.vx *= 0.96
			if q.kind != ptConfetti && q.kind != ptGoo {
				q.vy *= 0.96
			}
		}
		q.rot += q.vr
		keep = append(keep, q)
	}
	g.parts = keep
	keepF := g.fl[:0]
	for _, f := range g.fl {
		f.t += dt
		if f.t < f.life {
			keepF = append(keepF, f)
		}
	}
	g.fl = keepF
}

func tabX(i int) float64 { return W/numTabs*float64(i) + W/numTabs/2 }
