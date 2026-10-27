package game

import (
	"encoding/json"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The parts of multiplayer that touch the game: other players, world edits,
// stations, and creatures and items owned by another game. See net.go.

// ---- Other players ----

type peer struct {
	st     pStateMsg // the latest state they sent
	pid    string
	seen   int     // frame their last state arrived
	x, y   float64 // drawn position (world px), eased toward where they are
	walk   float64
	blink  int
	hurtCD int // creature contact damage is sent at most this often
}

var peers = map[int]*peer{}

// Mote colours for players 1-3 (the host keeps the violet)
var peerTints = [][2][3]int{
	{{124, 98, 255}, {196, 180, 255}},
	{{240, 150, 70}, {255, 214, 160}},
	{{90, 196, 110}, {176, 238, 170}},
	{{80, 170, 240}, {170, 220, 255}},
}

func tintFor(id int) [2][3]int { return peerTints[((id%4)+4)%4] }

func (p *peer) snap() { p.x, p.y = p.st.X, p.st.Y }

func myState() pStateMsg {
	s := pStateMsg{From: netMyID, Name: playerName(), X: player.x + float64(baseChunk*CW*CELL), Y: player.y,
		VX: player.vx, VY: player.vy, Face: player.face, Lift: player.lifting, Ground: player.onGround,
		Use: useActive, Swing: frame - swingAt, HP: player.hp, MaxHP: player.maxHP, Dead: cur == scDead, Boss: bossDown}
	if handMining && useActive {
		s.Held = "hands"
	} else if selected >= 0 && inv[selected] != nil {
		s.Held = inv[selected].ID
	}
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	s.Aim = math.Atan2(float64(aimY)+0.5-pcy, float64(aimX)+0.5-pcx)
	return s
}

func peerState(s pStateMsg) {
	if s.From == netMyID {
		return
	}
	p := peers[s.From]
	if p == nil {
		p = &peer{}
		peers[s.From] = p
		p.st = s
		p.snap()
		if netRole == netGuest {
			toast(s.Name + " is here")
		}
	}
	if netRole == netHost && p.st.Name != s.Name {
		p.st.Name = s.Name
	}
	p.st, p.seen = s, frame
	if s.Boss && !bossDown {
		bossDown = true
	}
}

func updatePeers() {
	for _, p := range peers {
		// ease toward where they are, carried along by their speed since the last update
		age := float64(min(frame-p.seen, 10))
		tx, ty := p.st.X+p.st.VX*age, p.st.Y+p.st.VY*age
		if math.Hypot(tx-p.x, ty-p.y) > 200 {
			p.x, p.y = tx, ty // teleported
		} else {
			p.x += (tx - p.x) * 0.35
			p.y += (ty - p.y) * 0.35
		}
		if p.st.Ground && math.Abs(p.st.VX) > 0.3 {
			p.walk += math.Abs(p.st.VX) * 0.08
		}
		if p.blink > 0 {
			p.blink--
		} else if (frame+p.st.From*97)%240 == 0 {
			p.blink = 8
		}
	}
	// their items: walking over someone else's dropped item asks its owner for it
	for owner, list := range remoteEnts {
		for _, e := range list {
			e.x = e.wx - float64(baseChunk*CW*CELL)
			if e.hurt > 0 {
				e.hurt--
			}
			if e.kind == entItem && cur == scPlaying && overlap(e) && frame-e.attackTimer > 30 {
				e.attackTimer = frame
				msg := takeMsg{From: netMyID, To: owner, ID: e.nid}
				if netRole == netHost {
					netSend(owner, mTake, msg)
				} else {
					netSend(0, mTake, msg)
				}
			}
		}
	}
}

// A message for one player: the host sends it straight to them, a guest via the host
func netSendTo(id int, typ byte, v any) {
	if netRole == netHost {
		netSend(id, typ, v)
	} else {
		netSend(0, typ, v)
	}
}

// Their motes, drawn like ours but in their own colour, with what they hold
func drawPeers(put func(x, y int, c [3]int, s int), ox, oy int) {
	if len(peers) == 0 {
		return
	}
	saved, sb, sl := player, pcol['B'], pcol['L']
	defer func() { player, pcol['B'], pcol['L'] = saved, sb, sl }()
	for id, p := range peers {
		if p.st.Dead {
			continue
		}
		t := tintFor(id)
		pcol['B'], pcol['L'] = t[0], t[1]
		player.x, player.y = p.x-float64(baseChunk*CW*CELL), p.y
		player.vx, player.vy = p.st.VX, p.st.VY
		player.face, player.lifting, player.onGround = p.st.Face, p.st.Lift, p.st.Ground
		player.walk, player.blink, player.invuln, player.squash, player.idle = p.walk, p.blink, 0, 0, 0
		drawPlayer(put, ox, oy)
		if p.st.Use && p.st.Held != "" && items[p.st.Held] != nil {
			drawPeerHeld(put, p, ox, oy)
		}
	}
}

func drawPeerHeld(put func(x, y int, c [3]int, s int), p *peer, ox, oy int) {
	it := items[p.st.Held]
	aim := p.st.Aim
	left := math.Cos(aim) < 0
	side := 4.5
	if left {
		side = -4.5
	}
	hx := (p.x+player.w/2)/CELL - float64(baseChunk*CW) - float64(ox) + side
	hy := (p.y+player.h/2)/CELL - float64(oy) + 2
	t := float64(frame)
	switch {
	case it.dmg > 0 && !isPick(it.id):
		k := math.Min(1, float64(p.st.Swing)/swingFrames)
		k = 1 - (1-k)*(1-k)
		dir := 1.0
		if left {
			dir = -1
		}
		drawRotIcon(put, it.icon, hx, hy, aim+dir*(-1.5+2.6*k), -math.Pi/4, 1.5, 6.5, left, 1.0)
	case isPick(it.id) && it.id != "hands", it.id == "hoe", it.id == "hammer":
		th := aim + 0.65*math.Sin(t*0.4) - 0.2
		if left {
			th = aim - 0.65*math.Sin(t*0.4) + 0.2
		}
		drawRotIcon(put, it.icon, hx, hy, th, -math.Pi/2, 4, 7.5, left, 0.85)
	default:
		push := 1 + 1.5*math.Abs(math.Sin(t*0.3))
		x0 := int(math.Round(hx+math.Cos(aim)*push)) - 2
		y0 := int(math.Round(hy+math.Sin(aim)*push)) - 2
		for y, r := range it.icon {
			for x := 0; x < 8 && x < len(r); x += 2 {
				if y%2 == 0 {
					if c, ok := iconCol[r[x]]; ok {
						put(x0+x/2, y0+y/2, c, 0)
					}
				}
			}
		}
	}
}

// Name tags over the other motes
func peerLabels() {
	for id, p := range peers {
		if p.st.Dead {
			continue
		}
		t := tintFor(id)
		sx := (p.x + player.w/2 - float64(baseChunk*CW*CELL) - cam.x) * zoom
		sy := (p.y-6-cam.y)*zoom - 4
		name := p.st.Name
		hp, mhp := p.st.HP, max(1, p.st.MaxHP)
		ops = append(ops, func(dst *ebiten.Image) {
			drawText(dst, name, sx, sy, textOpt{size: 12, align: text.AlignCenter, shadow: true, col: color.NRGBA{uint8(t[1][0]), uint8(t[1][1]), uint8(t[1][2]), 255}})
			// a small health bar under the name when hurt
			if hp < mhp {
				w := 36.0
				fill(dst, sx-w/2, sy+9, w, 4, color.NRGBA{20, 14, 30, 200})
				fill(dst, sx-w/2, sy+9, w*float64(hp)/float64(mhp), 4, color.NRGBA{240, 86, 112, 255})
			}
		})
	}
}

// The other motes' boxes, in local px (for creatures and PvP)
func peerBox(p *peer) (x, y, w, h float64) {
	return p.x - float64(baseChunk*CW*CELL), p.y, player.w, player.h
}

// ---- World edits ----

// Each frame, the cells in a box around the mote are remembered before the player acts
// (input, using items, building) and compared after: whatever changed was the player's
// doing and is sent to everyone else.
const watchR = 72

var (
	watchOn          bool
	watchX0, watchY0 int // world cells
	watchBuf         [4][]uint8
	watchOK          []bool // the cell was loaded when the frame began
	pendingEdits     []cellEdit
)

func layerArr(l int) []uint8 {
	switch l {
	case 1:
		return wall
	case 2:
		return paint
	case 3:
		return wpaint
	}
	return grid
}

func netWatchBegin() {
	watchOn = netRole != netOff && (netRole == netHost || netMyID >= 0)
	if !watchOn {
		return
	}
	n := (2*watchR + 1)
	cx, cy := int((player.x+player.w/2)/CELL), int((player.y+player.h/2)/CELL)
	watchX0, watchY0 = cx-watchR+baseChunk*CW, cy-watchR
	if len(watchOK) != n*n {
		watchOK = make([]bool, n*n)
	}
	for j := 0; j < n; j++ {
		y := watchY0 + j
		for i := 0; i < n; i++ {
			x := watchX0 + i - baseChunk*CW
			watchOK[j*n+i] = x >= 0 && x < W && y >= 0 && y < H
		}
	}
	for l := 0; l < 4; l++ {
		if len(watchBuf[l]) != n*n {
			watchBuf[l] = make([]uint8, n*n)
		}
		arr := layerArr(l)
		b := watchBuf[l]
		for j := 0; j < n; j++ {
			y := watchY0 + j
			for i := 0; i < n; i++ {
				if watchOK[j*n+i] {
					b[j*n+i] = arr[y*W+watchX0+i-baseChunk*CW]
				}
			}
		}
	}
}

func netWatchEnd() {
	if !watchOn {
		return
	}
	watchOn = false
	n := (2*watchR + 1)
	for l := 0; l < 4; l++ {
		arr := layerArr(l)
		b := watchBuf[l]
		for j := 0; j < n; j++ {
			y := watchY0 + j
			if y < 0 || y >= H {
				continue
			}
			for i := 0; i < n; i++ {
				x := watchX0 + i - baseChunk*CW
				// only cells that were loaded at the start too (the loaded area can
				// slide along in between)
				if x < 0 || x >= W || !watchOK[j*n+i] {
					continue
				}
				if v := arr[y*W+x]; v != b[j*n+i] {
					pendingEdits = append(pendingEdits, cellEdit{X: watchX0 + i, Y: y, L: byte(l), V: v})
				}
			}
		}
	}
}

func flushEdits() {
	if len(pendingEdits) == 0 {
		return
	}
	// big changes (blueprints, bombs) go in pieces so no single message is huge
	for len(pendingEdits) > 0 {
		k := min(len(pendingEdits), 4000)
		netShare(mEdits, editsMsg{From: netMyID, E: pendingEdits[:k]})
		pendingEdits = pendingEdits[k:]
	}
	pendingEdits = nil
}

// Apply someone else's changes: straight into the loaded world, or into the saved
// copy of chunks that aren't loaded here
func applyEdits(es []cellEdit) {
	byChunk := map[int][]cellEdit{}
	for _, e := range es {
		if e.Y < 0 || e.Y >= H || e.L > 3 {
			continue
		}
		lx := e.X - baseChunk*CW
		if lx >= 0 && lx < W {
			layerArr(int(e.L))[e.Y*W+lx] = e.V
			dirty[lx/CW] = true
			continue
		}
		cx := floorDiv(e.X, CW)
		byChunk[cx] = append(byChunk[cx], e)
	}
	for cx, list := range byChunk {
		for l := byte(0); l < 4; l++ {
			var buf []uint8
			for _, e := range list {
				if e.L != l {
					continue
				}
				if buf == nil {
					buf = chunkLayer(cx, l)
				}
				buf[e.Y*CW+e.X-cx*CW] = e.V
			}
			if buf != nil {
				storeChunkLayer(cx, l, buf)
			}
		}
	}
}

func layerStore(l byte) map[int][]uint16 {
	switch l {
	case 1:
		return savedWalls
	case 2:
		return savedPaint
	case 3:
		return savedWPaint
	}
	return saved
}

// An unloaded chunk's layer as it currently is (saved changes, or fresh from the seed)
func chunkLayer(cx int, l byte) []uint8 {
	if r, ok := layerStore(l)[cx]; ok {
		return unpackChunk(r)
	}
	switch l {
	case 0:
		return genChunk(cx)
	case 1:
		return genWalls(cx)
	}
	return make([]uint8, CW*H)
}

func storeChunkLayer(cx int, l byte, buf []uint8) {
	layerStore(l)[cx] = packBuf(buf)
}

// ---- Stations and storage chests ----

var lastObjs string

func objsKey() string {
	b, _ := json.Marshal(saveObjs())
	return string(b)
}

func shareObjsIfChanged() {
	k := objsKey()
	if k == lastObjs {
		return
	}
	lastObjs = k
	netShare(mObjs, objsMsg{From: netMyID, Objs: saveObjs()})
}

// ---- Creatures and items ----

var (
	nextNid    int
	remoteEnts = map[int][]*Entity{} // by owner
)

func shareEnts() {
	list := make([]entState, 0, len(ents))
	for _, e := range ents {
		if e.nid == 0 {
			nextNid++
			e.nid = nextNid
		}
		list = append(list, entState{ID: e.nid, K: int(e.kind), X: e.x + float64(baseChunk*CW*CELL), Y: e.y,
			VX: e.vx, VY: e.vy, HP: e.hp, Face: e.face, Hurt: e.hurt, T: e.t, Item: e.item, N: e.n})
	}
	netShare(mEnts, entsMsg{From: netMyID, E: list})
}

func remoteEntities(m entsMsg) {
	if m.From == netMyID {
		return
	}
	old := map[int]*Entity{}
	for _, e := range remoteEnts[m.From] {
		old[e.nid] = e
	}
	list := make([]*Entity, 0, len(m.E))
	for _, s := range m.E {
		k := entKind(s.K)
		e := old[s.ID]
		if e == nil {
			e = &Entity{kind: k, nid: s.ID, owner: m.From, remote: true, wx: s.X, y: s.Y}
			if k == entItem {
				e.w, e.h = 4*CELL, 4*CELL
			} else if int(k) < len(kinds) {
				e.w, e.h = float64(kinds[k].w*CELL), float64(kinds[k].h*CELL)
			}
		}
		// ease toward the owner's position so they glide between updates
		if math.Hypot(s.X-e.wx, s.Y-e.y) > 80 {
			e.wx, e.y = s.X, s.Y
		} else {
			e.wx += (s.X - e.wx) * 0.5
			e.y += (s.Y - e.y) * 0.5
		}
		e.vx, e.vy, e.hp, e.face, e.t, e.item, e.n = s.VX, s.VY, s.HP, s.Face, s.T, s.Item, s.N
		if s.Hurt > e.hurt {
			e.hurt = s.Hurt
		}
		e.x = e.wx - float64(baseChunk*CW*CELL)
		list = append(list, e)
	}
	remoteEnts[m.From] = list
}

// Everything to draw: ours and everyone else's
func allEnts() []*Entity {
	if len(remoteEnts) == 0 {
		return ents
	}
	out := append([]*Entity(nil), ents...)
	for _, l := range remoteEnts {
		out = append(out, l...)
	}
	return out
}

// Someone hit one of our creatures
func takeHit(h hitMsg) {
	for _, e := range ents {
		if e.nid == h.ID && e.kind != entItem && e.hurt == 0 {
			e.hp -= h.Dmg
			e.hurt = 12
			e.vx, e.vy = float64(h.Dir)*4, -3
			popDamage(h.Dmg, e.x+e.w/2, e.y, false)
			return
		}
	}
}

// Hit someone else's creature: they work out what happens; we show the hit now
func hitRemote(e *Entity, dmg, dir int) {
	e.hurt = 12
	popDamage(dmg, e.x+e.w/2, e.y, false)
	burst(5, (e.x+e.w/2)/CELL, (e.y+e.h/2)/CELL, [3]int{240, 122, 106}, 1, 1, 14)
	netSendTo(e.owner, mHit, hitMsg{From: netMyID, To: e.owner, ID: e.nid, Dmg: dmg, Dir: dir})
}

// Someone is picking up one of our items
func giveItemTo(t takeMsg) {
	for i, e := range ents {
		if e.nid == t.ID && e.kind == entItem && e.n > 0 && e.pickupDelay == 0 {
			netSendTo(t.From, mGive, takeMsg{From: netMyID, To: t.From, Item: e.item, N: e.n})
			ents = append(ents[:i], ents[i+1:]...)
			return
		}
	}
}

// Our creatures touching another mote hurt it
func hurtPeersOnContact(e *Entity) {
	for id, p := range peers {
		if p.st.Dead || frame < p.hurtCD {
			continue
		}
		x, y, w, h := peerBox(p)
		if e.x < x+w && e.x+e.w > x && e.y < y+h && e.y+e.h > y {
			p.hurtCD = frame + 30
			dir := sign(x + w/2 - (e.x + e.w/2))
			netSendTo(id, mHurt, hitMsg{From: netMyID, To: id, Dmg: kinds[e.kind].dmg, Dir: dir})
		}
	}
}

// PvP: a sword slash also hits other motes along it
func hitPeersAlong(x0, y0, x1, y1, r float64, dmg int) {
	if !world.PvP {
		return
	}
	sx, sy := x1-x0, y1-y0
	l2 := sx*sx + sy*sy
	for id, p := range peers {
		if p.st.Dead || frame < p.hurtCD {
			continue
		}
		bx, by, bw, bh := peerBox(p)
		ex, ey := (bx+bw/2)/CELL, (by+bh/2)/CELL
		t := 0.0
		if l2 > 0 {
			t = math.Max(0, math.Min(1, ((ex-x0)*sx+(ey-y0)*sy)/l2))
		}
		if math.Hypot(ex-(x0+sx*t), ey-(y0+sy*t)) <= r+6 {
			p.hurtCD = frame + 20
			netSendTo(id, mHurt, hitMsg{From: netMyID, To: id, Dmg: dmg, Dir: sign(ex - x0)})
			burst(6, ex, ey, [3]int{240, 122, 106}, 1, 1, 14)
		}
	}
}
