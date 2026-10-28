package mmo

import (
	"errors"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Char is a player's character. The server owns it; the game gets a copy after
// every change.
type Char struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Level int             `json:"lvl"`
	XP    int64           `json:"xp"` // towards the next level
	Coins int64           `json:"coins"`
	Look  Look            `json:"look"`
	Items map[string]int  `json:"items"` // resources from mining
	Inv   map[PetalID]int `json:"inv"`   // petals in the bag (kept for when petals come back)
	Load  []PetalID       `json:"load"`  // equipped petal in each slot ("" = empty)
	Isle  string          `json:"isle"`
	Kills int64           `json:"kills"`
	Seen  int64           `json:"seen"` // unix time last online
	Seq   int64           `json:"seq"`  // goes up on every save, so the newest backup wins
}

func NewChar(id, name string) *Char {
	c := &Char{ID: id, Name: name, Level: 1, Inv: map[PetalID]int{}, Items: map[string]int{}, Isle: "haven", Look: DefaultLook}
	for i := 0; i < SlotsAt(1); i++ {
		c.Load = append(c.Load, MakePetal("spark", 0))
	}
	c.Inv[MakePetal("leaf", 0)] = 1
	return c
}

// Fix up a character loaded from disk or a backup
func (c *Char) Normalize() {
	if c.Level < 1 {
		c.Level = 1
	}
	if c.Inv == nil {
		c.Inv = map[PetalID]int{}
	}
	if c.Items == nil {
		c.Items = map[string]int{}
	}
	for id, n := range c.Items {
		if n <= 0 || ResourceByID(id) == nil {
			delete(c.Items, id)
		}
	}
	c.Look.Valid()
	for id, n := range c.Inv {
		if n <= 0 || !id.Valid() {
			delete(c.Inv, id)
		}
	}
	n := SlotsAt(c.Level)
	for len(c.Load) < n {
		c.Load = append(c.Load, "")
	}
	for len(c.Load) > n { // shouldn't happen, but never lose a petal
		if p := c.Load[len(c.Load)-1]; p != "" {
			c.Inv[p]++
		}
		c.Load = c.Load[:len(c.Load)-1]
	}
	for i, p := range c.Load {
		if p != "" && !p.Valid() {
			c.Load[i] = ""
		}
	}
	if IsleByID(c.Isle) == nil {
		c.Isle = "haven"
	}
	c.Name = CleanName(c.Name)
}

func (c *Char) MaxHP() float64 { return MaxHPAt(c.Level) }

// AddXP adds experience and returns how many levels were gained
func (c *Char) AddXP(n int64) int {
	c.XP += n
	ups := 0
	for c.Level < MaxLevel && c.XP >= XPNeed(c.Level) {
		c.XP -= XPNeed(c.Level)
		c.Level++
		ups++
	}
	if ups > 0 {
		c.Normalize() // new slots
	}
	return ups
}

func (c *Char) Give(p PetalID, n int) {
	if p.Valid() && n > 0 {
		c.Inv[p] += n
	}
}

// Equip puts a petal from the bag into a slot (the one there goes back in the bag);
// an empty id takes the slot's petal off.
func (c *Char) Equip(slot int, p PetalID) error {
	if slot < 0 || slot >= len(c.Load) {
		return errors.New("no such slot")
	}
	if p != "" {
		if c.Inv[p] <= 0 {
			return errors.New("you don't have that petal")
		}
		c.Inv[p]--
		if c.Inv[p] == 0 {
			delete(c.Inv, p)
		}
	}
	if old := c.Load[slot]; old != "" {
		c.Inv[old]++
	}
	c.Load[slot] = p
	return nil
}

// Swap two slots
func (c *Char) Swap(a, b int) {
	if a >= 0 && b >= 0 && a < len(c.Load) && b < len(c.Load) {
		c.Load[a], c.Load[b] = c.Load[b], c.Load[a]
	}
}

// Combine five petals from the bag into one of the next rarity
func (c *Char) Combine(p PetalID) (PetalID, error) {
	k, r, ok := p.Split()
	if !ok {
		return "", errors.New("unknown petal")
	}
	if r >= NRarity-1 {
		return "", errors.New("that's already the rarest")
	}
	if c.Inv[p] < CombineN {
		return "", errors.New("you need 5 of them in your bag")
	}
	cost := CombineCost(r)
	if c.Coins < cost {
		return "", errors.New("not enough coins")
	}
	c.Coins -= cost
	c.Inv[p] -= CombineN
	if c.Inv[p] == 0 {
		delete(c.Inv, p)
	}
	out := MakePetal(k.ID, r+1)
	c.Inv[out]++
	return out, nil
}

// Petals in the bag, rarest first, then by kind
func (c *Char) Sorted() []PetalID {
	var ids []PetalID
	for id := range c.Inv {
		ids = append(ids, id)
	}
	order := map[string]int{}
	for i, p := range Petals {
		order[p.ID] = i
	}
	sort.Slice(ids, func(i, j int) bool {
		ki, ri, _ := ids[i].Split()
		kj, rj, _ := ids[j].Split()
		if ri != rj {
			return ri > rj
		}
		return order[ki.ID] < order[kj.ID]
	})
	return ids
}

// CleanName keeps a name short and printable
func CleanName(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if n >= 16 {
			break
		}
		if unicode.IsPrint(r) {
			b.WriteRune(r)
			n++
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		out = "Mote"
	}
	return out
}

// ---- Away gains (like IdleOn: your mote keeps fighting while you're offline) ----

type Away struct {
	Secs   int64           `json:"secs"`
	Isle   string          `json:"isle"`
	Kills  int64           `json:"kills"`
	XP     int64           `json:"xp"`
	Coins  int64           `json:"coins"`
	Levels int             `json:"levels"`
	Petals map[PetalID]int `json:"petals"`
}

const (
	AwayMax = 12 * 3600 // gains stop after 12 hours away
	AwayMin = 120
)

// Strength of a loadout: damage per second its petals could deal
func LoadoutPower(load []PetalID) float64 {
	p := 0.0
	for _, id := range load {
		if k, r, ok := id.Split(); ok {
			p += k.Dmg * Power(r) / k.Reload
			p += k.Burn * Power(r)
		}
	}
	return p
}

// AwayGains works out what a character earned on an island while offline for secs.
// roll gives random numbers in 0..1 (the server passes its own).
func AwayGains(c *Char, d *IsleDef, secs int64, roll func() float64) *Away {
	if d == nil || d.Safe || len(d.Mobs) == 0 || secs < AwayMin {
		return nil
	}
	secs = min(secs, AwayMax)
	// an average creature on this island
	var hp, xp, coins, wsum float64
	for _, mc := range d.Mobs {
		m := MobByID(mc.Kind)
		for r, rw := range d.Rarity {
			w := mc.Weight * rw
			hp += w * m.MaxHP(r)
			xp += w * float64(m.XPFor(r))
			coins += w * float64(m.CoinsFor(r))
			wsum += w
		}
	}
	hp, xp, coins = hp/wsum, xp/wsum, coins/wsum
	// fighting idle is slower than playing: at best one creature every 20 seconds
	perSec := math.Min(1.0/20, FightPower(c)/hp/30)
	if c.Level < d.Level {
		perSec *= math.Max(0.1, 1-0.15*float64(d.Level-c.Level))
	}
	kills := int64(perSec * float64(secs))
	if kills <= 0 {
		return nil
	}
	a := &Away{Secs: secs, Isle: d.ID, Kills: kills, XP: int64(xp * float64(kills) * 0.6),
		Coins: int64(coins * float64(kills) * 0.6), Petals: map[PetalID]int{}}
	// drops: about one in eight kills, no rarer than the island's common creatures
	top := 0
	for r, rw := range d.Rarity {
		if rw >= 20 {
			top = r
		}
	}
	for i := int64(0); PetalsOn && i < min(kills/8, 400); i++ {
		mc := pickMob(d, roll())
		m := MobByID(mc)
		kind := m.Drops[int(roll()*float64(len(m.Drops)))%len(m.Drops)]
		r := top
		if roll() < 0.6 && r > 0 {
			r--
		}
		a.Petals[MakePetal(kind, r)]++
	}
	return a
}

// Apply away gains to a character
func (c *Char) TakeAway(a *Away) {
	c.Kills += a.Kills
	c.Coins += a.Coins
	a.Levels = c.AddXP(a.XP)
	for p, n := range a.Petals {
		c.Give(p, n)
	}
}

// pickMob picks a creature kind on an island from a number in 0..1
func pickMob(d *IsleDef, u float64) string {
	sum := 0.0
	for _, m := range d.Mobs {
		sum += m.Weight
	}
	u *= sum
	for _, m := range d.Mobs {
		if u -= m.Weight; u < 0 {
			return m.Kind
		}
	}
	return d.Mobs[len(d.Mobs)-1].Kind
}

func PickMob(d *IsleDef, u float64) string { return pickMob(d, u) }

// PickRarity picks a rarity from the island's weights
func PickRarity(d *IsleDef, u float64) int {
	sum := 0.0
	for _, w := range d.Rarity {
		sum += w
	}
	u *= sum
	for r, w := range d.Rarity {
		if u -= w; u < 0 {
			return r
		}
	}
	return 0
}
