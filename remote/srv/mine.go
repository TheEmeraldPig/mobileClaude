package srv

import (
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Mining and the sword. Mined blocks turn into worn ones (they're never removed),
// every player on the island is told, and they grow back after the island's Regrow time.

type regrow struct {
	x, y int
	orig uint8
	at   float64
}

// Mine a cell for a player, if the island allows it and the player is close enough
func (is *IsleSim) mine(s *Server, c *Client, x, y int) {
	if c.dead || x < 0 || y < 0 || x >= mmo.GW || y >= mmo.GH {
		return
	}
	i := y*mmo.GW + x
	t := is.isle.Cells[i]
	yd, ok := is.def.YieldAt(t)
	cx, cy := c.center()
	near := math.Hypot(float64(x*mmo.Cell+1)-cx, float64(y*mmo.Cell+1)-cy) <= mmo.MineReach+8
	// digging takes time: a game can't mine faster than the block allows
	fast := s.t-c.mineAt < yd.Time*0.7
	if !ok || !near || fast {
		c.send(&mmo.SMsg{T: "edits", Edits: []mmo.Edit{{X: x, Y: y, T: t}}}) // put it back on their screen
		return
	}
	c.mineAt = s.t
	var es []mmo.Edit
	mmo.Brush(func(dx, dy int) {
		bx, by := x+dx, y+dy
		if bx < 0 || by < 0 || bx >= mmo.GW || by >= mmo.GH {
			return
		}
		i := by*mmo.GW + bx
		old := is.isle.Cells[i]
		b, ok := is.def.YieldAt(old)
		if !ok {
			return
		}
		is.isle.Cells[i] = b.Becomes
		is.edits[i] = b.Becomes
		es = append(es, mmo.Edit{X: bx, Y: by, T: b.Becomes})
		if is.def.Regrow > 0 {
			is.regrow = append(is.regrow, regrow{bx, by, old, s.t + is.def.Regrow})
		}
	})
	for _, o := range is.players {
		o.send(&mmo.SMsg{T: "edits", Edits: es})
	}
	c.ch.Items[yd.Item]++
	c.swingT = 0.3
	s.charChanged(c)
}

// Mined blocks whose time is up grow back (runs even with nobody on the island)
func (is *IsleSim) regrowStep(s *Server) {
	if len(is.regrow) == 0 || is.regrow[0].at > s.t {
		return
	}
	n := 0
	var es []mmo.Edit
	for n < len(is.regrow) && is.regrow[n].at <= s.t {
		r := is.regrow[n]
		n++
		is.isle.Cells[r.y*mmo.GW+r.x] = r.orig
		delete(is.edits, r.y*mmo.GW+r.x)
		es = append(es, mmo.Edit{X: r.x, Y: r.y, T: r.orig})
	}
	for _, o := range is.players {
		o.send(&mmo.SMsg{T: "edits", Edits: es})
	}
	is.regrow = is.regrow[n:]
}

// Everything that's different from the generated island, for a player arriving
func (is *IsleSim) allEdits() []mmo.Edit {
	out := make([]mmo.Edit, 0, len(is.edits))
	for i, t := range is.edits {
		out = append(out, mmo.Edit{X: i % mmo.GW, Y: i / mmo.GW, T: t})
	}
	return out
}

// A sword swing: creatures in an arc in front of the player get hit
func (is *IsleSim) swing(s *Server, c *Client, aim float64) {
	if c.dead || s.t-c.swungAt < mmo.SwordEvery*0.8 {
		return
	}
	c.swungAt, c.swingT, c.aim = s.t, 0.3, aim
	cx, cy := c.center()
	dmg := mmo.SwordDamage(c.ch.Level)
	for _, m := range is.mobs {
		if m.hp <= 0 {
			continue
		}
		// the closest point of the creature to the player
		nx, ny := math.Max(m.x, math.Min(cx, m.x+m.w)), math.Max(m.y, math.Min(cy, m.y+m.h))
		dx, dy := nx-cx, ny-cy
		d := math.Hypot(dx, dy)
		if d > mmo.SwordReach || (d > 6 && math.Abs(mmo.AngleDiff(math.Atan2(dy, dx), aim)) > mmo.SwordArc) {
			continue
		}
		is.damage(m, c, dmg)
		mx, my := m.x+m.w/2, m.y+m.h/2
		k := math.Max(1, math.Hypot(mx-cx, my-cy))
		kb := 220 / (1 + float64(m.rar)*0.5) / math.Max(1, m.w/16)
		m.vx += (mx - cx) / k * kb
		m.vy -= 120 / (1 + float64(m.rar))
	}
}

// Damage a creature on behalf of a player
func (is *IsleSim) damage(m *Mob, c *Client, dmg float64) {
	m.hp -= dmg
	if m.dmgBy == nil {
		m.dmgBy = map[int]float64{}
	}
	m.dmgBy[c.id] += dmg
	m.hurtT = 0.15
	if m.target == 0 {
		m.target = c.id
	}
}
