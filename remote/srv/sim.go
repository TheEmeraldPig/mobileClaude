package srv

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// One island: its creatures and the players on it. Everything here runs with the
// server's lock held.

type IsleSim struct {
	def     *mmo.IsleDef
	isle    *mmo.Isle
	mobs    []*Mob
	players map[int]*Client
	nextMob int
	spawnT  float64
	bossT   float64
	idle    float64
}

type Mob struct {
	id        int
	kind      *mmo.MobKind
	ki        int
	rar       int
	x, y      float64 // top-left, world px
	vx, vy    float64
	w, h      float64
	hp, maxHP float64
	face      int
	onGround  bool
	target    int
	aware     float64
	wander    float64 // flyers: angle; walkers: direction
	wanderT   float64
	hopCD     float64
	burnDps   float64
	burnT     float64
	burnBy    int
	slowT     float64
	hurtT     float64
	dmgBy     map[int]float64
	boss      bool
	homeX     float64
	homeY     float64
	bumpT     float64
}

type petal struct {
	kind   *mmo.PetalKind
	rar    int
	hp     float64
	reload float64 // > 0: broken, growing back
	cd     float64 // until it can hit again
}

func newIsleSim(d *mmo.IsleDef) *IsleSim {
	return &IsleSim{def: d, isle: mmo.BuildIsle(d), players: map[int]*Client{}, bossT: 90}
}

func (is *IsleSim) join(s *Server, c *Client) {
	is.players[c.id] = c
	info := c.info()
	for _, o := range is.players {
		if o != c {
			o.send(&mmo.SMsg{T: "pinfo", Info: info})
			c.send(&mmo.SMsg{T: "pinfo", Info: o.info()})
		}
	}
}

func (is *IsleSim) leave(s *Server, c *Client) {
	delete(is.players, c.id)
	for _, o := range is.players {
		o.send(&mmo.SMsg{T: "gone", ID: c.id})
	}
	for _, m := range is.mobs {
		if m.target == c.id {
			m.target = 0
		}
	}
}

func (is *IsleSim) broadcastInfo(c *Client) {
	info := c.info()
	for _, o := range is.players {
		if o != c {
			o.send(&mmo.SMsg{T: "pinfo", Info: info})
		}
	}
}

func (c *Client) info() *mmo.PInfo {
	return &mmo.PInfo{ID: c.id, Name: c.ch.Name, Level: c.ch.Level, Load: c.ch.Load}
}

// The equipped petals, rebuilt after the loadout changes. Fresh ones start growing.
func (c *Client) makePetals(ready bool) {
	old := map[mmo.PetalID][]*petal{}
	for _, p := range c.petals {
		id := mmo.MakePetal(p.kind.ID, p.rar)
		old[id] = append(old[id], p)
	}
	c.petals = c.petals[:0]
	for _, id := range c.ch.Load {
		k, r, ok := id.Split()
		if !ok {
			continue
		}
		if l := old[id]; len(l) > 0 { // the same petal moved: keep its state
			c.petals = append(c.petals, l[0])
			old[id] = l[1:]
			continue
		}
		p := &petal{kind: k, rar: r, hp: k.HP * mmo.Power(r)}
		if !ready {
			p.reload = k.Reload
		}
		c.petals = append(c.petals, p)
	}
}

func (c *Client) center() (float64, float64) { return c.x + mmo.MoteW/2, c.y + mmo.MoteH/2 }

// ---- Stepping ----

func (is *IsleSim) step(s *Server) {
	for _, c := range is.players {
		is.stepPlayer(s, c)
	}
	for _, m := range is.mobs {
		is.stepMob(s, m)
	}
	for _, c := range is.players {
		if !c.dead {
			is.petalHits(s, c)
		}
	}
	out := is.mobs[:0]
	for _, m := range is.mobs {
		if m.hp <= 0 {
			is.reward(s, m)
			continue
		}
		out = append(out, m)
	}
	is.mobs = out
	is.spawn(s)
}

func (is *IsleSim) stepPlayer(s *Server, c *Client) {
	if c.dead {
		return
	}
	c.invuln -= dt
	c.regenWait -= dt
	mhp := c.ch.MaxHP()
	heal := 0.0
	if c.regenWait <= 0 {
		heal += mhp * 0.015
	}
	c.orbit += (mmo.OrbitFor(c.mode) - c.orbit) * math.Min(1, dt*10)
	for _, p := range c.petals {
		if p.reload > 0 {
			if p.reload -= dt; p.reload <= 0 {
				p.reload, p.hp = 0, p.kind.HP*mmo.Power(p.rar)
			}
			continue
		}
		p.cd -= dt
		heal += p.kind.Heal * mmo.HealMult(p.rar)
		if p.kind.Mend > 0 && mhp-c.hp >= p.kind.Mend*mmo.HealMult(p.rar)*0.8 {
			c.hp = math.Min(mhp, c.hp+p.kind.Mend*mmo.HealMult(p.rar))
			p.reload = p.kind.Reload
		}
	}
	c.hp = math.Min(mhp, c.hp+heal*dt)
	// lava burns
	c.lavaCD -= dt
	if c.lavaCD <= 0 && is.touches(c.x, c.y, mmo.MoteW, mmo.MoteH, mmo.LAVA) {
		c.lavaCD = 0.5
		is.hurt(s, c, mhp*0.08, 0)
	}
}

func (is *IsleSim) touches(x, y, w, h float64, t uint8) bool {
	x0, x1 := int(math.Floor(x/mmo.Cell)), int(math.Floor((x+w-0.01)/mmo.Cell))
	y0, y1 := int(math.Floor(y/mmo.Cell)), int(math.Floor((y+h-0.01)/mmo.Cell))
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			if is.isle.At(cx, cy) == t {
				return true
			}
		}
	}
	return false
}

func (is *IsleSim) hurt(s *Server, c *Client, dmg float64, dir int) {
	if c.dead || c.invuln > 0 {
		return
	}
	c.hp -= dmg
	c.invuln = 0.6
	c.regenWait = 5
	c.send(&mmo.SMsg{T: "hurt", Dmg: int(math.Ceil(dmg)), Dir: dir, HP: math.Max(0, c.hp)})
	if c.hp <= 0 {
		c.hp = 0
		c.dead = true
		c.send(&mmo.SMsg{T: "dead"})
	}
}

// Creatures: notice the nearest mote and go after it, each in its own way
func (is *IsleSim) stepMob(s *Server, m *Mob) {
	k := m.kind
	m.hurtT -= dt
	m.hopCD -= dt
	m.bumpT -= dt
	if m.burnT > 0 {
		m.burnT -= dt
		d := m.burnDps * dt
		m.hp -= d
		if m.dmgBy != nil {
			m.dmgBy[m.burnBy] += d
		}
	}
	m.slowT -= dt
	mx, my := m.x+m.w/2, m.y+m.h/2
	// pick a target
	var tgt *Client
	if c := is.players[m.target]; c != nil && !c.dead {
		cx, cy := c.center()
		if math.Hypot(cx-mx, cy-my) < k.Sense*1.6 {
			tgt = c
		}
	}
	if tgt == nil {
		best := k.Sense
		for _, c := range is.players {
			if c.dead {
				continue
			}
			cx, cy := c.center()
			if d := math.Hypot(cx-mx, cy-my); d < best {
				best, tgt = d, c
			}
		}
	}
	m.target = 0
	if tgt != nil {
		m.target = tgt.id
	}
	sp := k.Speed * (1 + 0.04*float64(m.rar))
	if m.slowT > 0 {
		sp *= 0.5
	}
	var dx, dy float64
	if tgt != nil {
		cx, cy := tgt.center()
		dx, dy = cx-mx, cy-my
	}
	m.wanderT -= dt
	if k.Fly {
		var vx, vy float64
		if tgt != nil {
			d := math.Max(1, math.Hypot(dx, dy))
			wob := math.Sin(s.t*3+float64(m.id)) * 0.35
			vx, vy = (dx/d-dy/d*wob)*sp, (dy/d+dx/d*wob)*sp
		} else {
			if m.wanderT <= 0 {
				m.wanderT = 1.5 + s.rng.Float64()*2.5
				m.wander = s.rng.Float64() * 2 * math.Pi
			}
			hx, hy := m.homeX-mx, m.homeY-my // drift home when far
			vx = math.Cos(m.wander)*sp*0.35 + hx*0.3
			vy = math.Sin(m.wander)*sp*0.35 + hy*0.3
		}
		if m.bumpT > 0 { // backing off after a hit
			vx, vy = -vx*0.6, -vy*0.6
		}
		m.vx += (vx - m.vx) * math.Min(1, dt*3)
		m.vy += (vy - m.vy) * math.Min(1, dt*3)
		if is.touches(m.x, m.y, m.w, m.h, mmo.WATER) {
			m.vy -= 400 * dt
		}
	} else {
		want := 0.0
		if tgt != nil && math.Abs(dy) < 220 {
			want = sign(dx) * sp
			if k.Hop && m.onGround && m.hopCD <= 0 && math.Abs(dx) < 110 {
				m.vy = -430
				m.vx = sign(dx) * sp * 1.8
				m.hopCD = 1.6
			}
		} else {
			if m.wanderT <= 0 {
				m.wanderT = 1 + s.rng.Float64()*3
				m.wander = float64(s.rng.Intn(3) - 1)
			}
			want = m.wander * sp * 0.4
		}
		if m.bumpT > 0 {
			want = -want
		}
		if m.onGround {
			m.vx += (want - m.vx) * math.Min(1, dt*8)
		}
		m.vy = math.Min(m.vy+1500*dt, 600)
		if is.touches(m.x, m.y, m.w, m.h, mmo.WATER) {
			m.vy = math.Max(m.vy-2400*dt, -120)
		}
	}
	if m.vx > 1 {
		m.face = 1
	} else if m.vx < -1 {
		m.face = -1
	}
	if is.moveMob(m) && !k.Fly && m.onGround && m.vx != 0 {
		m.vy = -380 // blocked: jump
	}
	// touching a mote hurts it
	for _, c := range is.players {
		if c.dead || c.invuln > 0 {
			continue
		}
		if c.x < m.x+m.w && c.x+mmo.MoteW > m.x && c.y < m.y+m.h && c.y+mmo.MoteH > m.y {
			cx, _ := c.center()
			is.hurt(s, c, k.Damage(m.rar), int(sign(cx-mx)))
			m.bumpT = 0.5
		}
	}
	// fell out of the world
	if m.y > float64(mmo.GH*mmo.Cell) {
		m.hp = -1
		m.dmgBy = nil
	}
}

// Move with collisions, a cell at a time; walkers step up small ledges. Reports
// whether it bumped into a wall.
func (is *IsleSim) moveMob(m *Mob) bool {
	bumped := false
	stepX := m.vx * dt
	for math.Abs(stepX) > 0 {
		st := sign(stepX) * math.Min(mmo.Cell, math.Abs(stepX))
		switch {
		case is.isle.BoxFree(m.x+st, m.y, m.w, m.h):
			m.x += st
		case !m.kind.Fly && m.onGround && is.isle.BoxFree(m.x+st, m.y-2*mmo.Cell, m.w, m.h):
			m.x += st
			m.y -= 2 * mmo.Cell
		default:
			m.vx = 0
			bumped = true
			stepX = 0
			continue
		}
		stepX -= st
	}
	stepY := m.vy * dt
	m.onGround = false
	for math.Abs(stepY) > 0 {
		st := sign(stepY) * math.Min(mmo.Cell, math.Abs(stepY))
		if is.isle.BoxFree(m.x, m.y+st, m.w, m.h) {
			m.y += st
			stepY -= st
			continue
		}
		if st > 0 {
			m.onGround = true
		}
		m.vy = 0
		break
	}
	if !m.kind.Fly && !m.onGround && m.vy == 0 && !is.isle.BoxFree(m.x, m.y+1, m.w, m.h) {
		m.onGround = true
	}
	return bumped
}

// Petals around a mote hitting creatures
func (is *IsleSim) petalHits(s *Server, c *Client) {
	n := len(c.petals)
	if n == 0 {
		return
	}
	cx, cy := c.center()
	reach := c.orbit + mmo.PetalR + 40
	for i, p := range c.petals {
		if p.reload > 0 || p.cd > 0 || p.kind.Dmg == 0 && p.kind.Mend > 0 {
			continue
		}
		px, py := mmo.PetalPos(cx, cy, c.orbit, s.t, i, n)
		for _, m := range is.mobs {
			if m.hp <= 0 {
				continue
			}
			mx, my := m.x+m.w/2, m.y+m.h/2
			if math.Abs(mx-cx) > reach+m.w || math.Abs(my-cy) > reach+m.h {
				continue
			}
			// circle against box
			nx, ny := math.Max(m.x, math.Min(px, m.x+m.w)), math.Max(m.y, math.Min(py, m.y+m.h))
			if (nx-px)*(nx-px)+(ny-py)*(ny-py) > mmo.PetalR*mmo.PetalR {
				continue
			}
			pow := mmo.Power(p.rar)
			dmg := p.kind.Dmg * pow
			m.hp -= dmg
			if m.dmgBy == nil {
				m.dmgBy = map[int]float64{}
			}
			m.dmgBy[c.id] += dmg
			m.hurtT = 0.15
			m.aware = 3
			if m.target == 0 {
				m.target = c.id
			}
			if p.kind.Burn > 0 {
				m.burnDps, m.burnT, m.burnBy = math.Max(m.burnDps, p.kind.Burn*pow), 3, c.id
			}
			if p.kind.Slow > 0 {
				m.slowT = math.Max(m.slowT, p.kind.Slow+0.25*float64(p.rar))
			}
			// a little knockback, less for big creatures
			d := math.Max(1, math.Hypot(mx-cx, my-cy))
			kb := 140 / (1 + float64(m.rar)*0.5) / math.Max(1, m.w/16)
			m.vx += (mx - cx) / d * kb
			if m.kind.Fly {
				m.vy += (my - cy) / d * kb
			}
			p.hp -= m.kind.Damage(m.rar)
			p.cd = 0.25
			if p.hp <= 0 {
				p.reload = p.kind.Reload
			}
			break
		}
	}
}

// A creature died: everyone who did a fair share of the damage gets the rewards
func (is *IsleSim) reward(s *Server, m *Mob) {
	if m.dmgBy == nil {
		return
	}
	top, topID := 0.0, 0
	for id, d := range m.dmgBy {
		if d > top {
			top, topID = d, id
		}
	}
	for id, d := range m.dmgBy {
		c := is.players[id]
		if c == nil || (d < m.maxHP*0.1 && id != topID) {
			continue
		}
		l := &mmo.Loot{Mob: m.kind.ID, Rar: m.rar, XP: m.kind.XPFor(m.rar), Coins: m.kind.CoinsFor(m.rar),
			X: int(m.x + m.w/2), Y: int(m.y + m.h/2)}
		if s.rng.Float64() < m.kind.DropChance {
			r := m.rar
			if !m.boss && r > 0 && s.rng.Float64() < 0.7 {
				r--
			}
			kind := m.kind.Drops[s.rng.Intn(len(m.kind.Drops))]
			l.Petals = append(l.Petals, mmo.MakePetal(kind, r))
		}
		if m.boss { // bosses also drop a second petal of their own rarity
			l.Petals = append(l.Petals, mmo.MakePetal(m.kind.Drops[s.rng.Intn(len(m.kind.Drops))], m.rar))
		}
		c.ch.Coins += l.Coins
		c.ch.Kills++
		for _, p := range l.Petals {
			c.ch.Give(p, 1)
		}
		l.Levels = c.ch.AddXP(l.XP)
		if l.Levels > 0 {
			c.makePetals(false)
			c.hp = c.ch.MaxHP()
			is.broadcastInfo(c)
			s.cfg.Logf("%s reached level %d", c.ch.Name, c.ch.Level)
		}
		c.send(&mmo.SMsg{T: "loot", Loot: l})
		s.charChanged(c)
	}
	if m.boss {
		is.bossT = 300
		for _, c := range is.players {
			c.send(&mmo.SMsg{T: "msg", Text: "The Warden is beaten! It wakes again in 5 minutes."})
		}
	}
}

// Keep the island stocked with creatures, away from the spawn point and from players
func (is *IsleSim) spawn(s *Server) {
	d := is.def
	if d.Safe || len(d.Mobs) == 0 {
		return
	}
	is.spawnT -= dt
	if d.Boss != "" {
		alive := false
		for _, m := range is.mobs {
			alive = alive || m.boss
		}
		if !alive {
			if is.bossT -= dt; is.bossT <= 0 {
				r := 4
				switch u := s.rng.Float64(); {
				case u < 0.05:
					r = 5
				case u < 0.4:
					r = 3
				}
				if m := is.place(s, mmo.MobByID(d.Boss), r); m != nil {
					m.boss = true
					is.bossT = 300
					for _, c := range is.players {
						c.send(&mmo.SMsg{T: "msg", Text: "A " + mmo.Rarities[r].Name + " Warden has woken somewhere on the island!"})
					}
				}
			}
		}
	}
	if is.spawnT > 0 || len(is.mobs) >= d.MaxMobs {
		return
	}
	is.spawnT = 0.25
	if len(is.mobs) < d.MaxMobs/2 {
		is.spawnT = 0.05 // fill up quickly at first
	}
	is.place(s, mmo.MobByID(mmo.PickMob(d, s.rng.Float64())), mmo.PickRarity(d, s.rng.Float64()))
}

func (is *IsleSim) place(s *Server, k *mmo.MobKind, rar int) *Mob {
	d, isle := is.def, is.isle
	w, h := k.SizePx()
	for try := 0; try < 30; try++ {
		cx := d.Ocean + 30 + s.rng.Intn(mmo.GW-2*d.Ocean-60)
		if abs(cx-isle.SpawnX) < 110 {
			continue // the spawn point stays calm
		}
		surf := isle.SurfAt(cx)
		cy := surf - 60 + s.rng.Intn(320)
		x, y := float64(cx*mmo.Cell)-w/2, float64(cy*mmo.Cell)
		if !k.Fly { // drop to the floor below
			ok := false
			for fall := 0; fall < 120; fall++ {
				if !isle.BoxFree(x, y+mmo.Cell, w, h) {
					ok = true
					break
				}
				y += mmo.Cell
			}
			if !ok {
				continue
			}
		}
		if !isle.BoxFree(x, y, w, h) || is.touches(x, y, w, h, mmo.WATER) || is.touches(x-20, y-20, w+40, h+40, mmo.LAVA) {
			continue
		}
		near := false
		for _, c := range is.players {
			pcx, pcy := c.center()
			if math.Abs(pcx-(x+w/2)) < 300 && math.Abs(pcy-(y+h/2)) < 220 {
				near = true
			}
		}
		if near {
			continue
		}
		is.nextMob++
		hp := k.MaxHP(rar)
		m := &Mob{id: is.nextMob, kind: k, ki: mobIndex(k), rar: rar, x: x, y: y, w: w, h: h, hp: hp, maxHP: hp,
			face: 1, homeX: x + w/2, homeY: y + h/2}
		is.mobs = append(is.mobs, m)
		return m
	}
	return nil
}

func mobIndex(k *mmo.MobKind) int {
	for i, m := range mmo.Mobs {
		if m == k {
			return i
		}
	}
	return 0
}

// ---- Snapshots ----

func (is *IsleSim) sendSnaps(s *Server) {
	ps := make([]mmo.PSnap, 0, len(is.players))
	for _, c := range is.players {
		fl := c.flags
		if c.dead {
			fl |= mmo.FlagDead
		}
		alive := 0
		for i, p := range c.petals {
			if p.reload <= 0 && i < 30 {
				alive |= 1 << i
			}
		}
		ps = append(ps, mmo.PSnap{ID: c.id, X: int(c.x), Y: int(c.y), VX: int(c.vx), VY: int(c.vy), Face: c.face,
			Flags: fl, HP: int(math.Ceil(c.hp)), MaxHP: int(c.ch.MaxHP()), Orbit: int(c.orbit), Alive: alive})
	}
	for _, c := range is.players {
		ms := make([]mmo.MSnap, 0, 64)
		cx, cy := c.center()
		for _, m := range is.mobs {
			if math.Abs(m.x-cx) > 1500 || math.Abs(m.y-cy) > 1000 {
				continue
			}
			fl := 0
			if m.hurtT > 0 {
				fl |= mmo.MobHurt
			}
			if m.burnT > 0 {
				fl |= mmo.MobBurn
			}
			if m.slowT > 0 {
				fl |= mmo.MobSlow
			}
			ms = append(ms, mmo.MSnap{ID: m.id, Kind: m.ki, Rar: m.rar, X: int(m.x), Y: int(m.y), VX: int(m.vx), VY: int(m.vy),
				Face: m.face, HP: int(math.Ceil(math.Max(0, m.hp) / m.maxHP * 1000)), Flags: fl})
		}
		c.send(&mmo.SMsg{T: "snap", Time: s.t, P: ps, M: ms})
	}
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func sqrt(v float64) float64 { return math.Sqrt(v) }

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
