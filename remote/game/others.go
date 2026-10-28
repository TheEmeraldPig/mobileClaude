package game

import (
	"image/color"
	"math"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Other players, creatures and petals, as the server describes them.

// ---- Other players ----

type other struct {
	id         int
	info       *mmo.PInfo
	st         mmo.PSnap // their latest state
	seen       int       // frame it arrived (0 = not yet)
	x, y       float64   // drawn position, eased toward theirs
	walk       float64
	blink      int
	orbit      float64
	swing      float64 // < 0: not swinging; 0..1 through a swing
	emote      int
	emoteUntil int
}

var (
	others       = map[int]*other{}
	myEmote      int
	myEmoteUntil int
)

func otherFor(id int) *other {
	o := others[id]
	if o == nil {
		o = &other{id: id, orbit: mmo.OrbitNormal, swing: -1}
		others[id] = o
	}
	return o
}

// Mote colours for other players
var peerTints = [][2][3]int{
	{{240, 150, 70}, {255, 214, 160}},
	{{90, 196, 110}, {176, 238, 170}},
	{{80, 170, 240}, {170, 220, 255}},
	{{230, 96, 150}, {255, 186, 214}},
	{{220, 200, 80}, {255, 244, 170}},
	{{150, 110, 230}, {214, 196, 255}},
}

func tintFor(id int) [2][3]int { return peerTints[((id%len(peerTints))+len(peerTints))%len(peerTints)] }

// ---- Creatures ----

// mmo.Mobs index -> the sprite to draw it with
var mobSprite = []entKind{entSkitter, entGloom, entBat, entWisp, entBrute, entShade, entWarden}

func readMobs(ms []mmo.MSnap) {
	byID := map[int]*Entity{}
	for _, e := range ents {
		byID[e.nid] = e
	}
	out := ents[:0]
	for _, s := range ms {
		if s.Kind < 0 || s.Kind >= len(mmo.Mobs) {
			continue
		}
		e := byID[s.ID]
		if e == nil {
			mk := mmo.Mobs[s.Kind]
			w, h := mk.SizePx()
			e = &Entity{nid: s.ID, kind: mobSprite[s.Kind], mk: mk, x: float64(s.X), y: float64(s.Y), w: w, h: h, hpk: s.HP, rar: s.Rar}
		}
		delete(byID, s.ID)
		if s.HP < e.hpk {
			lost := float64(e.hpk-s.HP) / 1000 * e.mk.MaxHP(e.rar)
			if lost >= 1 {
				popDamage(int(math.Round(lost)), e.x+e.w/2, e.y-2, false)
			}
		}
		if s.Flags&mmo.MobHurt != 0 && e.hurt == 0 {
			e.hurt = 8
			if math.Hypot(e.x-player.x, e.y-player.y) < 300 {
				sfx("hit")
			}
		}
		e.tx, e.ty, e.tvx, e.tvy = float64(s.X), float64(s.Y), float64(s.VX), float64(s.VY)
		e.hpk, e.rar, e.face, e.flags, e.seenAt = s.HP, s.Rar, s.Face, s.Flags, frame
		if e.face == 0 {
			e.face = 1
		}
		out = append(out, e)
	}
	ents = out
	// the ones that are gone: a puff where they were, if they were nearly beaten
	for _, e := range byID {
		if e.hpk < 400 {
			c := mmo.Rarities[e.rar].Col
			burst(10, (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL, c, 1.2, 1, 26)
		}
	}
}

// Ease everyone toward where the server last saw them, carried along by their speed
func updateRemote() {
	for _, e := range ents {
		e.t++
		if e.hurt > 0 {
			e.hurt--
		}
		age := float64(min(frame-e.seenAt, 12)) / 60
		tx, ty := e.tx+e.tvx*age, e.ty+e.tvy*age
		if math.Hypot(tx-e.x, ty-e.y) > 160 {
			e.x, e.y = tx, ty
		} else {
			e.x += (tx - e.x) * 0.3
			e.y += (ty - e.y) * 0.3
		}
	}
	for _, o := range others {
		if o.seen == 0 {
			continue
		}
		age := float64(min(frame-o.seen, 12)) / 60
		tx, ty := float64(o.st.X)+float64(o.st.VX)*age, float64(o.st.Y)+float64(o.st.VY)*age
		if math.Hypot(tx-o.x, ty-o.y) > 200 {
			o.x, o.y = tx, ty
		} else {
			o.x += (tx - o.x) * 0.35
			o.y += (ty - o.y) * 0.35
		}
		if o.st.Flags&mmo.FlagGround != 0 && math.Abs(float64(o.st.VX)) > 18 {
			o.walk += math.Abs(float64(o.st.VX)) / 60 * 0.08
		}
		if o.blink > 0 {
			o.blink--
		} else if (frame+o.id*37)%240 == 0 {
			o.blink = 7
		}
		o.orbit += (float64(o.st.Orbit) - o.orbit) * 0.3
		if o.swing >= 0 {
			if o.swing += 1.0 / 14; o.swing > 1 {
				o.swing = -1
			}
		}
		if o.st.Flags&mmo.FlagSwing != 0 && o.swing < 0 {
			o.swing = 0
		}
	}
	myOrbit += (mmo.OrbitFor(myMode) - myOrbit) * math.Min(1, 10.0/60)
}

// ---- Petals ----

// The petals actually equipped, in slot order (the server spaces them the same way)
func equipped(load []mmo.PetalID) []mmo.PetalID {
	var out []mmo.PetalID
	for _, p := range load {
		if p.Valid() {
			out = append(out, p)
		}
	}
	return out
}

type petalDraw struct {
	x, y float64 // world px
	id   mmo.PetalID
}

// Where a mote's petals are this frame
func petalsOf(cx, cy, orbit float64, load []mmo.PetalID, alive int) []petalDraw {
	eq := equipped(load)
	t := serverTime()
	var out []petalDraw
	for i, p := range eq {
		if alive&(1<<i) == 0 {
			continue
		}
		x, y := mmo.PetalPos(cx, cy, orbit, t, i, len(eq))
		out = append(out, petalDraw{x, y, p})
	}
	return out
}

func myPetals() []petalDraw {
	if !mmo.PetalsOn || me == nil || cur == scDead {
		return nil
	}
	return petalsOf(player.x+player.w/2, player.y+player.h/2, myOrbit, me.Load, myAlive)
}

func allPetals() []petalDraw {
	out := myPetals()
	for _, o := range others {
		if !mmo.PetalsOn {
			break
		}
		if o.info == nil || o.seen == 0 || o.st.Flags&mmo.FlagDead != 0 {
			continue
		}
		out = append(out, petalsOf(o.x+mmo.PlayerW/2, o.y+mmo.PlayerH/2, o.orbit, o.info.Load, o.st.Alive)...)
	}
	return out
}

// A petal: a round 5x5 bead in its kind's colour, with an outline in its rarity's colour
var petalRows = []string{".ooo.", "oLBBo", "oBBBo", "oBBSo", ".ooo."}

func drawPetal(put func(x, y int, c [3]int, s int), cx, cy int, p mmo.PetalID) {
	k, r, ok := p.Split()
	if !ok {
		return
	}
	body := k.Col
	rim := mmo.Rarities[r].Col
	if r == 0 {
		rim = [3]int{34, 28, 64}
	}
	for y, row := range petalRows {
		for x := 0; x < len(row); x++ {
			var c [3]int
			switch row[x] {
			case 'o':
				c = rim
			case 'B':
				c = body
			case 'L':
				c = mix(body, [3]int{255, 255, 255}, 0.55)
			case 'S':
				c = shadeRGB(body, 0.75)
			default:
				continue
			}
			put(cx-2+x, cy-2+y, c, 0)
		}
	}
}

func drawPetals(put func(x, y int, c [3]int, s int), ox, oy int) {
	for _, p := range allPetals() {
		drawPetal(put, int(math.Round(p.x/CELL))-ox, int(math.Round(p.y/CELL))-oy, p.id)
	}
}

// Petals and glowing creatures light their surroundings
func petalLights(seed func(lx, ly, lvl int, src uint8)) {
	for _, p := range allPetals() {
		k, r, _ := p.id.Split()
		if k.Light == 0 {
			continue
		}
		src := uint8(srcLantern)
		switch k.ID {
		case "glow":
			src = srcCrystal
		case "ember":
			src = srcTorch
		}
		seed(int(p.x/CELL), int(p.y/CELL), min(64, k.Light+3*r), src)
	}
}

// ---- Drawing other motes ----

func drawOthers(put func(x, y int, c [3]int, s int), ox, oy int) {
	for _, o := range others {
		if o.seen == 0 || o.st.Flags&mmo.FlagDead != 0 {
			continue
		}
		look := mmo.DefaultLook
		if o.info != nil {
			look = o.info.Look
		}
		face := o.st.Face
		if face == 0 {
			face = 1
		}
		ground, lift := o.st.Flags&mmo.FlagGround != 0, o.st.Flags&mmo.FlagLift != 0
		ps := pose{face: face, legs: legFrame(ground, lift, float64(o.st.VX)/60, o.walk), blink: o.blink > 0, swing: o.swing, tool: o.st.Tool,
			aim: float64(o.st.Aim) * math.Pi / 180}
		drawBody(put, ox, oy, o.x, o.y, look, ps)
	}
}

// Name tags, health bars and emotes over the players, and labels over creatures
func worldLabels() {
	label := func(x, y float64) (float64, float64) { return (x - cam.x) * zoom, (y - cam.y) * zoom }
	for _, o := range others {
		if o.seen == 0 || o.info == nil || o.st.Flags&mmo.FlagDead != 0 {
			continue
		}
		t := tintFor(o.id)
		sx, sy := label(o.x+mmo.PlayerW/2, o.y-6)
		name := o.info.Name + "  Lv " + itoa(o.info.Level)
		hp, mhp := o.st.HP, max(1, o.st.MaxHP)
		col := color.NRGBA{uint8(t[1][0]), uint8(t[1][1]), uint8(t[1][2]), 255}
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, name, sx, sy-4, textOpt{size: 12, align: text.AlignCenter, shadow: true, col: col})
			if hp < mhp {
				w := 36.0
				fill(dst, sx-w/2, sy+5, w, 4, color.NRGBA{20, 14, 30, 200})
				fill(dst, sx-w/2, sy+5, w*float64(hp)/float64(mhp), 4, color.NRGBA{240, 86, 112, 255})
			}
		})
		if frame < o.emoteUntil {
			emoteBubble(mmo.Emotes[o.emote], sx, sy-24)
		}
	}
	if frame < myEmoteUntil && cur == scPlaying {
		sx, sy := label(player.x+player.w/2, player.y-6)
		emoteBubble(mmo.Emotes[myEmote], sx, sy-12)
	}
	for _, e := range ents {
		if e.mk == nil {
			continue
		}
		hurt := e.hpk < 1000
		if e.rar == 0 && !hurt {
			continue
		}
		sx, sy := label(e.x+e.w/2, e.y-3)
		rc := mix(mmo.Rarities[e.rar].Col, [3]int{255, 255, 255}, 0.3)
		col := color.NRGBA{uint8(rc[0]), uint8(rc[1]), uint8(rc[2]), 255}
		name := e.mk.Name
		if e.rar > 0 {
			name = mmo.Rarities[e.rar].Name + " " + name
		}
		frac := float64(e.hpk) / 1000
		w := math.Max(26, e.w*zoom*0.9)
		ops = append(ops, func(dst *ebiten.Image) {
			if e.rar > 0 {
				drawText(dst, name, sx, sy-12, textOpt{size: 10, align: text.AlignCenter, shadow: true, col: col})
			}
			if hurt {
				fill(dst, sx-w/2, sy-4, w, 4, color.NRGBA{20, 14, 30, 210})
				fill(dst, sx-w/2, sy-4, w*frac, 4, color.NRGBA{120, 230, 110, 255})
			}
		})
	}
}

func emoteBubble(s string, x, y float64) {
	w := textWidth(s, 12, false) + 16
	r := Rect{x - w/2, y - 11, w, 22}
	ops = append(ops, func(dst *ebiten.Image) {
		bevel(dst, r, colInk, colInk, colMuted, false)
		fill(dst, x-3, r.Y+r.H+3, 6, 3, colO)
		drawText(dst, s, x, y, textOpt{size: 12, align: text.AlignCenter, col: colO})
	})
}

func itoa(n int) string { return mmo.FmtNum(int64(n)) }
