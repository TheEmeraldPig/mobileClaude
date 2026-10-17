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

// Bug Brigade: your team of bugs fights off the slugs. Bugs attack on their own,
// tapping the field hits too, skills charge up, and every 10 slugs a boss
// arrives with a timer. Coins level bugs up; gems buy eggs that hatch new bugs.

var timing = os.Getenv("BUGS_TIMING") != ""

// scaleOverride (BUGS_SCALE) sets the device scale factor, for screenshots.
var scaleOverride, _ = strconv.ParseFloat(os.Getenv("BUGS_SCALE"), 64)

const dt = 1.0 / 60

type slug struct {
	kind   int
	boss   bool
	hp     float64
	max    float64
	x, tx  float64
	hit    float64 // white flash, fades
	hurt   int     // frames of the hurt face
	dead   float64 // dying animation 0..1, 0 = alive
	gone   bool
	phase  float64
	coins  float64
	spawn  float64 // pop-in 0..1
	squish float64 // wobble from being hit
	trail  float64 // health bar damage trail
}

type shot struct {
	x0, y0, x1, y1 float64
	t, dur         float64
	dmg            float64
	col            color.RGBA
	crit           bool
	target         *slug
	big            float64 // size
	skill          bool
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
)

type particle struct {
	x, y, vx, vy, g float64
	t, life, size   float64
	rot, vr         float64
	kind            int
	col             color.RGBA
	tx, ty          float64 // coins fly to the counter
}

type bugState struct {
	atk    float64 // attack animation 0..1, <0 idle
	cool   float64
	charge float64
	hop    float64 // tap hop 0..1
	blink  int
	fired  bool
	cast   float64 // skill animation 0..1
	pop    float64 // joined the team animation
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

type Game struct {
	p    *Profile
	w, h float64
	rng  *rand.Rand
	now  func() int64

	frame int
	t     float64 // seconds since start

	// battle
	slugs     []*slug
	shots     []*shot
	fl        []floater
	parts     []particle
	team      []bugState
	bossT     float64 // seconds left on the boss, 0 = no boss
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
	shownCoin float64

	// interface
	tab       int
	tabT      float64 // tab change animation
	scroll    [3]float64
	scrollMax [3]float64
	vel       float64
	mult      int // buy x1, x10, max
	hits      []hit
	ptr       pointer
	toast     string
	toastT    float64
	pressedID string
	pressT    map[string]float64
	enterT    float64

	// overlays
	hatchID    int
	hatchT     float64
	hatchNew   bool
	hatchQueue []int
	welcome    float64
	welcomeT   float64
	welcomeS   int64

	lastSave float64
	script   *script
}

func NewGame() *Game {
	g := &Game{w: W, h: 960, now: func() int64 { return time.Now().Unix() }, hatchID: -1, pressT: map[string]float64{}}
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
	if c, secs := g.p.offline(now); c > 0 {
		g.welcome, g.welcomeS = c, secs
	}
	g.p.LastSeen = now
	g.shownCoin = g.p.Coins
	g.syncTeam()
	return g
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
func (g *Game) groundY() float64 { return g.fieldH() - 60 }
func (g *Game) skillY() float64  { return g.fieldH() }
func (g *Game) panelY() float64  { return g.fieldH() + 96 }
func (g *Game) tabsY() float64   { return g.h - 86 }

// bugPos is where team member i stands and how big it is.
func (g *Game) bugPos(i int) (x, y, sc float64) {
	gy := g.groundY()
	p := [][3]float64{{262, gy + 12, 1.0}, {150, gy + 18, 1.0}, {206, gy - 34, 0.86}, {46, gy + 14, 1.0}, {98, gy - 36, 0.86}}[i%5]
	x, y, sc = p[0], p[1], p[2]
	if bugs[g.p.Team[i]].fly {
		y -= 34 * sc
	}
	return
}

func slugSlot(i int) float64 { return 402 + float64(i)*112 }

// ---- team ----

func (g *Game) syncTeam() {
	for len(g.team) < len(g.p.Team) {
		g.team = append(g.team, bugState{atk: -1, cool: g.rng.Float64(), pop: 0})
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
	if g.hatchID < 0 && len(g.hatchQueue) > 0 {
		g.hatchID, g.hatchQueue = g.hatchQueue[0], g.hatchQueue[1:]
		g.hatchT = 0
		g.hatchNew = g.p.hatch(g.hatchID)
		g.syncTeam()
		g.Save()
	}
	if g.hatchID >= 0 {
		g.hatchT += dt
	}
	if g.welcome > 0 {
		g.welcomeT += dt
	}
	g.updateField()
	g.stepEffects()
	if g.toastT > 0 {
		g.toastT -= dt
	}
	g.tabT += dt
	g.enterT += dt
	for k, v := range g.pressT {
		if v -= dt * 4; v <= 0 {
			delete(g.pressT, k)
		} else {
			g.pressT[k] = v
		}
	}
	// list momentum
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

func (g *Game) modal() bool { return g.hatchID >= 0 || g.welcome > 0 }

func (g *Game) updatePointer() {
	// Every new finger on the battlefield is a tap, so two-finger drumming works.
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		fx, fy := float64(x)/S, float64(y)/S
		if !g.ptr.down {
			g.ptr = pointer{down: true, touch: true, id: id, sx: fx, sy: fy, x: fx, y: fy, lastY: fy}
			g.ptr.field = g.pointerDown(fx, fy)
		} else if !g.modal() && fy < g.fieldH() && fy > 70 {
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

// pointerDown handles a fresh press; taps on the battlefield act at once.
func (g *Game) pointerDown(x, y float64) bool {
	g.vel = 0
	if !g.modal() && y < g.fieldH() && y > 70 && g.hitAt(x, y) == "" {
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
	switch name {
	case "modal":
		if g.hatchID >= 0 && g.hatchT > 2.2 {
			g.hatchID = -1
		}
	case "collect", "collect2":
		if name == "collect2" {
			if p.Gems < 30 {
				g.say("Not enough gems")
				return
			}
			p.Gems -= 30
			g.welcome *= 2
		}
		g.addCoins(g.welcome, W/2, g.h/2)
		g.welcome = 0
		g.Save()
	case "tab":
		if g.tab != n {
			g.tab, g.tabT = n, 0
		}
	case "mult":
		g.mult = (g.mult + 1) % 3
	case "up":
		lv := g.buyCount(n)
		if err := p.levelUp(n, lv); err != nil {
			g.say(err.Error())
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
		if err := p.toggleTeam(n); err != nil {
			g.say(err.Error())
			return
		}
		g.syncTeam()
		for i, t := range p.Team {
			if t == n {
				g.team[i] = bugState{atk: -1, cool: 0.5}
			}
		}
		g.Save()
	case "buy":
		it := shopItems[n]
		egg, err := p.buy(it, g.rng)
		if err != nil {
			g.say(err.Error())
			return
		}
		if it.price != "" {
			g.say("Test purchase: +" + itoa(it.give) + " gems")
			g.gemPulse = 1
		}
		if it.id == "coins" {
			g.coinPulse = 1
		}
		if egg >= 0 {
			g.hatchQueue = append(g.hatchQueue, egg)
		}
		g.Save()
	case "free":
		egg, err := p.freeEgg(g.now(), g.rng)
		if err != nil {
			g.say(err.Error())
			return
		}
		g.hatchQueue = append(g.hatchQueue, egg)
		g.Save()
	}
}

// buyCount is how many levels the x1/x10/max switch buys for bug id.
func (g *Game) buyCount(id int) int {
	switch g.mult {
	case 1:
		return 10
	case 2:
		return g.p.maxLevels(id)
	}
	return 1
}

func (g *Game) addCoins(v, x, y float64) {
	g.p.Coins += v
	n := int(math.Min(14, 3+math.Log10(v+1)*2))
	for i := 0; i < n; i++ {
		a := g.rng.Float64() * 2 * math.Pi
		sp := 2 + g.rng.Float64()*3
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 3, g: 0.25,
			life: 0.9 + float64(i)*0.03, size: 1, kind: ptCoin, tx: 70, ty: 30})
	}
}

// ---- the battle ----

func (g *Game) front() *slug {
	for _, s := range g.slugs {
		if s.dead == 0 && !s.gone {
			return s
		}
	}
	return nil
}

func (g *Game) updateField() {
	p := g.p
	if g.frenzyT > 0 {
		g.frenzyT -= dt
	}
	if g.critT > 0 {
		g.critT -= dt
	}
	if g.goldT > 0 {
		g.goldT -= dt
	}
	if g.shake > 0 {
		g.shake = math.Max(0, g.shake-dt*3)
	}
	if g.flash > 0 {
		g.flash = math.Max(0, g.flash-dt*3)
	}
	g.coinPulse = math.Max(0, g.coinPulse-dt*3)
	g.gemPulse = math.Max(0, g.gemPulse-dt*3)
	if g.bannerT > 0 {
		g.bannerT -= dt
	}
	// boss time
	if g.bossOn {
		if !g.modal() {
			g.bossT -= dt
		}
		if b := g.front(); b != nil && b.boss && g.bossT <= 0 {
			// The boss gets away.
			b.gone = true
			b.dead = 0.001
			g.bossOn = false
			p.loseBoss()
			g.showBanner("Out of time!", "Level up your bugs and try again")
			g.Save()
		}
	}
	// spawn
	alive := 0
	for _, s := range g.slugs {
		if s.dead == 0 {
			alive++
		}
	}
	wantBoss := !p.Farming && p.Wave >= wavesPerStage
	if wantBoss && !g.bossOn && alive == 0 {
		g.bossOn = true
		g.bossT = bossTime
		k := (p.Stage - 1) % len(bossKinds)
		hp := bossHP(p.Stage)
		g.slugs = append(g.slugs, &slug{kind: k, boss: true, hp: hp, max: hp, trail: hp, x: 680, tx: slugSlot(0) + 40, coins: bossCoins(p.Stage)})
		g.showBanner("Boss!", slugName(k, true))
	}
	g.spawnCD -= dt
	if !wantBoss && !g.bossOn && alive < 3 && g.spawnCD <= 0 {
		g.spawnCD = 0.7
		k := g.rng.Intn(min(len(slugKinds), 2+p.Stage/2))
		hp := slugHP(p.Stage)
		g.slugs = append(g.slugs, &slug{kind: k, hp: hp, max: hp, trail: hp, x: 680 + float64(alive)*40, coins: slugCoins(p.Stage), phase: g.rng.Float64() * 6})
	}
	// slugs crawl to their spots
	i := 0
	for _, s := range g.slugs {
		if s.dead > 0 {
			s.dead += dt * 1.6
			continue
		}
		if !s.boss {
			s.tx = slugSlot(i)
		}
		i++
		s.spawn = math.Min(1, s.spawn+dt*2.5)
		if s.x > s.tx {
			s.x = math.Max(s.tx, s.x-dt*150)
			s.phase += dt * 9
		} else {
			s.phase += dt * 2.5
		}
		s.hit = math.Max(0, s.hit-dt*6)
		s.squish *= 0.9
		if s.hurt > 0 {
			s.hurt--
		}
		s.trail += (s.hp - s.trail) * 0.06
	}
	keep := g.slugs[:0]
	for _, s := range g.slugs {
		if s.dead < 1 {
			keep = append(keep, s)
		}
	}
	g.slugs = keep
	// the brigade attacks
	f := g.front()
	ready := f != nil && f.x < 560
	speed := 1.0
	if g.frenzyT > 0 {
		speed = 2
	}
	for i := range g.team {
		b := &g.team[i]
		id := p.Team[i]
		sp := bugs[id]
		b.pop = math.Min(1, b.pop+dt*2)
		if g.bossOn || ready {
			b.charge = math.Min(1, b.charge+dt/sp.cool)
		}
		if b.hop > 0 {
			b.hop = math.Max(0, b.hop-dt*5)
		}
		if b.cast > 0 {
			b.cast += dt * 1.6
			if b.cast >= 1 {
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
				if t := g.front(); t != nil {
					x, y, sc := g.bugPos(i)
					crit := g.rng.Float64() < 0.05
					dmg := p.dps(id) * sp.period
					if crit {
						dmg *= 3
					}
					g.fire(x+40*sc, y-58*sc, t, dmg, sp.shot, crit, 1)
				}
			}
			if b.atk >= 1 {
				b.atk = -1
			}
			continue
		}
		b.cool -= dt * speed
		if b.cool <= 0 && ready && !g.modal() {
			b.cool = sp.period * (0.92 + g.rng.Float64()*0.16)
			b.atk, b.fired = 0, false
		}
	}
	// shots fly
	keepS := g.shots[:0]
	for _, s := range g.shots {
		s.t += dt / s.dur
		if s.t >= 1 {
			g.land(s)
			continue
		}
		if g.frame%2 == 0 {
			x, y := s.pos()
			g.parts = append(g.parts, particle{x: x, y: y, life: 0.3, size: 0.5 * s.big, kind: ptSpark, col: s.col})
		}
		keepS = append(keepS, s)
	}
	g.shots = keepS
}

func (s *shot) pos() (float64, float64) {
	t := s.t
	x := s.x0 + (s.x1-s.x0)*t
	arcH := math.Min(80, math.Abs(s.x1-s.x0)*0.25)
	y := s.y0 + (s.y1-s.y0)*t - math.Sin(t*math.Pi)*arcH
	return x, y
}

func (g *Game) fire(x, y float64, t *slug, dmg float64, col color.RGBA, crit bool, big float64) {
	sc := 1.0
	if t.boss {
		sc = 1.45
	}
	tx := t.x - 20*sc + g.rng.Float64()*30*sc
	ty := g.groundY() - (40+g.rng.Float64()*30)*sc
	g.shots = append(g.shots, &shot{x0: x, y0: y, x1: tx, y1: ty, dur: 0.32 + math.Abs(tx-x)/1400, dmg: dmg, col: col, crit: crit, target: t, big: big})
}

// land applies a shot's damage where it hits.
func (g *Game) land(s *shot) {
	t := s.target
	if t == nil || t.dead > 0 {
		t = g.front()
	}
	if t == nil {
		return
	}
	g.damage(t, s.dmg, s.crit, s.x1, s.y1)
	for k := 0; k < 5; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		g.parts = append(g.parts, particle{x: s.x1, y: s.y1, vx: math.Cos(a) * 2.5, vy: math.Sin(a) * 2.5, life: 0.35, size: 0.8 * s.big, kind: ptStar, col: s.col, rot: a})
	}
}

// damage hurts a slug and handles it dying.
func (g *Game) damage(t *slug, dmg float64, crit bool, x, y float64) {
	if t.dead > 0 {
		return
	}
	t.hp -= dmg
	t.hit = 1
	t.hurt = 14
	t.squish = 1
	col, size := colWhite, 22.0
	if crit {
		col, size = rgb(0xff, 0xd8, 0x30), 30
	}
	g.fl = append(g.fl, floater{x: x + (g.rng.Float64()-0.5)*20, y: y - 10, life: 0.9, size: size, s: num(dmg), col: col})
	if t.hp > 0 {
		return
	}
	t.hp = 0
	t.dead = 0.001
	p := g.p
	coins := t.coins
	if g.goldT > 0 {
		coins *= 3
	}
	sc := 1.0
	if t.boss {
		sc = 1.45
	}
	cy := g.groundY() - 50*sc
	// poof
	for k := 0; k < 14; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		sp := 1 + g.rng.Float64()*3
		g.parts = append(g.parts, particle{x: t.x + math.Cos(a)*30*sc, y: cy + math.Sin(a)*20*sc, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 1, life: 0.6 + g.rng.Float64()*0.3, size: (0.6 + g.rng.Float64()*0.6) * sc, kind: ptPuff, col: colWhite})
		g.parts = append(g.parts, particle{x: t.x, y: cy, vx: math.Cos(a) * sp * 1.5, vy: math.Sin(a)*sp*1.5 - 3, g: 0.3, life: 0.8, size: 0.5 * sc, kind: ptGoo, col: slugColor(t)})
	}
	if t.boss {
		g.bossOn = false
		egg := p.beatBoss()
		g.coinBurst(coins-t.coins, t.x, cy, 14) // beatBoss paid the coins; this adds any gold rush bonus
		g.gemPulse = 1
		g.shake = 1
		g.flash = 0.8
		for k := 0; k < 60; k++ {
			a := g.rng.Float64() * 2 * math.Pi
			sp := 3 + g.rng.Float64()*6
			g.parts = append(g.parts, particle{x: t.x, y: cy, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 5, g: 0.22, life: 1.6, size: 1, kind: ptConfetti,
				col: []color.RGBA{colGold, colPink, colBlue, colGreen, colPurple}[k%5], rot: a, vr: (g.rng.Float64() - 0.5) * 0.4})
		}
		if area(p.Stage) != area(p.Stage-1) {
			g.showBanner("New area!", areaOf(p.Stage).name)
		} else {
			g.showBanner("Stage clear!", "+"+itoa(bossGems(p.Stage-1))+" gems")
		}
		if egg {
			g.hatchQueue = append(g.hatchQueue, rollBug(g.rng, eggs[0].weights))
		}
		g.Save()
		return
	}
	p.beatSlug(0)
	g.coinBurst(coins, t.x, cy, 4)
}

// coinBurst pays coins out as flying coins that count up when they reach the counter.
func (g *Game) coinBurst(v, x, y float64, n int) {
	g.p.Coins += v
	for i := 0; i < n; i++ {
		a := -math.Pi/2 + (g.rng.Float64()-0.5)*2
		sp := 3 + g.rng.Float64()*3
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, g: 0.3,
			life: 1.0 + float64(i)*0.04, size: 1, kind: ptCoin, tx: 64, ty: 30})
	}
}

func slugColor(t *slug) color.RGBA {
	if t.boss {
		return bossKinds[t.kind%len(bossKinds)].col
	}
	return slugKinds[t.kind%len(slugKinds)].col
}

// tapField is a finger on the battlefield: a direct hit on the nearest slug.
func (g *Game) tapField(x, y float64) {
	t := g.front()
	g.p.Taps++
	for k := 0; k < 6; k++ {
		a := g.rng.Float64() * 2 * math.Pi
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * 3, vy: math.Sin(a) * 3, life: 0.3, size: 0.7, kind: ptStar, col: colGold, rot: a})
	}
	g.parts = append(g.parts, particle{x: x, y: y, life: 0.25, size: 1.4, kind: ptSpark, col: colWhite})
	if t == nil || t.x > 580 {
		return
	}
	crit := g.rng.Float64() < 0.08 || g.critT > 0
	dmg := g.p.tapDamage()
	if crit {
		dmg *= 4
	}
	sc := 1.0
	if t.boss {
		sc = 1.45
	}
	g.damage(t, dmg, crit, t.x-10*sc+(g.rng.Float64()-0.5)*40*sc, g.groundY()-60*sc)
	if len(g.team) > 0 {
		g.team[g.rng.Intn(len(g.team))].hop = 1
		for i := range g.team {
			g.team[i].charge = math.Min(1, g.team[i].charge+0.004)
		}
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
	t := g.front()
	if t == nil {
		g.say("No slugs to hit")
		return
	}
	id := g.p.Team[i]
	sp := bugs[id]
	b.charge = 0
	b.cast = 0.001
	x, y, sc := g.bugPos(i)
	dmg := g.p.dps(id) * sp.skDmg
	switch sp.skKind {
	case skAll:
		for _, s := range g.slugs {
			if s.dead == 0 {
				g.fire(x+40*sc, y-60*sc, s, dmg, sp.shot, true, 2)
			}
		}
		g.showBanner(sp.skill, "")
		return
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
	if id == 3 && t.boss {
		dmg *= 3
	}
	for k := 0; k < 3; k++ {
		g.fire(x+40*sc, y-60*sc, t, dmg/3, sp.shot, true, 2.2)
	}
	g.shake = 0.6
	g.showBanner(sp.skill, "")
}

func (g *Game) levelFx(id int) {
	for i, t := range g.p.Team {
		if t == id {
			x, y, sc := g.bugPos(i)
			g.team[i].hop = 1
			for k := 0; k < 12; k++ {
				a := -math.Pi/2 + (g.rng.Float64()-0.5)*2.4
				g.parts = append(g.parts, particle{x: x + (g.rng.Float64()-0.5)*60*sc, y: y - 20*sc, vx: math.Cos(a), vy: math.Sin(a)*2 - 1.5, life: 0.8, size: 0.7, kind: ptStar, col: colGreen})
			}
			g.fl = append(g.fl, floater{x: x, y: y - 120*sc, life: 1, size: 22, s: "Level up!", col: rgb(0xb0, 0xff, 0x60)})
		}
	}
}

func (g *Game) showBanner(s, sub string) { g.banner, g.bannerSub, g.bannerT = s, sub, 2.2 }

func (g *Game) stepEffects() {
	keep := g.parts[:0]
	for _, q := range g.parts {
		q.t += dt
		if q.t >= q.life {
			if q.kind == ptCoin {
				g.coinPulse = 1
			}
			continue
		}
		if q.kind == ptCoin && q.t > 0.45 {
			// fly to the coin counter
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
