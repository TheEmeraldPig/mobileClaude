package game

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Multiplayer: up to 4 motes in one world, joined with a 5-letter code from anywhere.
//
// Everyone connects to a small relay server (mote/relay); the host's game runs the
// world and the relay just passes messages between it and its guests. Guests talk only
// to the host, which handles each message and passes it on to the other guests.
//
// What travels:
//   - World changes. Everyone builds the same terrain from the seed, so only changes
//     are sent: a guest joining gets all of the host's changed chunks, and after that
//     each game sends the block, wall and paint cells its own player changed (found by
//     comparing a box around the player before and after their actions each frame,
//     see netWatch). Falling sand and flowing water run on each game by itself.
//   - Stations and storage chests: the whole list, whenever it changes.
//   - Players: position, movement, held item and health, a few times a second.
//   - Creatures and dropped items belong to the game that spawned them, which runs
//     them and sends where they are; hits on them, and picking up items, are sent to
//     their owner, and a creature touching someone else's mote sends that mote damage.
//   - The host also sends the time of day, the PvP setting and whether the Warden is
//     beaten, and keeps each guest's character in its world save.

// RelayURL is the relay server; set at build time with
// -ldflags "-X github.com/TheEmeraldPig/mobileClaude/mote/game.RelayURL=wss://..."
// and changeable in Settings.
var RelayURL = ""

func relayURL() string {
	u := strings.TrimSpace(prefs.Relay)
	if u == "" {
		u = RelayURL
	}
	u = strings.TrimRight(u, "/")
	if strings.HasPrefix(u, "https://") {
		u = "wss://" + u[len("https://"):]
	} else if strings.HasPrefix(u, "http://") {
		u = "ws://" + u[len("http://"):]
	}
	return u
}

const (
	netOff = iota
	netHost
	netGuest
)

const maxPlayers = 4

var (
	netRole   int
	netMyID   int // 0 for the host, 1-3 for guests
	netCode   string
	netStatus string // shown in the menus while connecting, or why it ended
	netCancel context.CancelFunc
	netOutQ   chan []byte
	inboxMu   sync.Mutex
	inbox     [][]byte
	netClosed bool // the connection ended (the reason is in netStatus)
)

// Message types (the first byte of each message; the rest is JSON)
const (
	mHello   = iota + 1 // guest -> host: who I am
	mWelcome            // host -> guest: the world
	mPState             // a player's state
	mEdits              // changed cells
	mObjs               // the stations and storage chests
	mEnts               // an owner's creatures and dropped items
	mHit                // -> owner: a creature was hit
	mHurt               // -> player: you were hurt
	mTake               // -> owner: I'm picking up this item
	mGive               // owner -> taker: here it is
	mTime               // host -> all: time of day and settings
	mChar               // guest -> host: my character, to keep in the world save
	mGone               // host -> all: a player left
	mDenied             // host -> guest: you can't join
)

// ---- Connection ----

func netConnect(url string, host bool) {
	netClose()
	ctx, cancel := context.WithCancel(context.Background())
	netCancel = cancel
	netOutQ = make(chan []byte, 1024)
	netClosed = false
	inboxMu.Lock()
	inbox = nil
	inboxMu.Unlock()
	netCtx = ctx
	if url != "" {
		netDial(url)
	}
}

var netCtx context.Context

// Connect (in the background) to url, for the connection netConnect set up
func netDial(url string) {
	ctx, out := netCtx, netOutQ
	go func() {
		dctx, dcancel := context.WithTimeout(ctx, 75*time.Second) // free servers can take a while to wake
		c, _, err := websocket.Dial(dctx, url, nil)
		dcancel()
		if err != nil {
			netFail("Couldn't reach the multiplayer server")
			return
		}
		c.SetReadLimit(16 << 20)
		go func() {
			for {
				select {
				case <-ctx.Done():
					c.Close(websocket.StatusNormalClosure, "")
					return
				case f := <-out:
					wctx, wcancel := context.WithTimeout(ctx, 30*time.Second)
					err := c.Write(wctx, websocket.MessageBinary, f)
					wcancel()
					if err != nil {
						netFail("Lost the connection")
						return
					}
				}
			}
		}()
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				reason := "Lost the connection"
				switch websocket.CloseStatus(err) {
				case 4004:
					reason = "There's no world with that code"
				case 4003:
					reason = "That world is full"
				case 4001:
					reason = "The host closed their world"
				case websocket.StatusNormalClosure:
					reason = ""
				}
				if ctx.Err() == nil {
					netFail(reason)
				}
				return
			}
			inboxMu.Lock()
			inbox = append(inbox, data)
			inboxMu.Unlock()
		}
	}()
}

func netFail(reason string) {
	inboxMu.Lock()
	inbox = append(inbox, []byte{0xff})
	inboxMu.Unlock()
	failReason.Store(reason)
}

var failReason syncString

type syncString struct {
	mu sync.Mutex
	s  string
}

func (s *syncString) Store(v string) { s.mu.Lock(); s.s = v; s.mu.Unlock() }
func (s *syncString) Load() string   { s.mu.Lock(); defer s.mu.Unlock(); return s.s }

func netClose() {
	if netCancel != nil {
		// a moment's grace so the last messages (like a guest's character) get out
		c := netCancel
		time.AfterFunc(600*time.Millisecond, c)
		netCancel = nil
	}
	netRole, netCode = netOff, ""
	peers = map[int]*peer{}
	remoteEnts = map[int][]*Entity{}
	guestJoined = map[int]bool{}
}

// Send a message. A guest's always goes to the host; the host's goes to guest `to`
// (0 = all guests).
func netSend(to int, typ byte, v any) {
	if netRole == netOff || netOutQ == nil {
		return
	}
	j, err := json.Marshal(v)
	if err != nil {
		return
	}
	var f []byte
	if netRole == netHost {
		f = append([]byte{byte(to), typ}, j...)
	} else {
		f = append([]byte{typ}, j...)
	}
	select {
	case netOutQ <- f:
	default: // the connection is hopelessly behind; it will drop soon anyway
	}
}

// From the host: send to every guest except one
func netSendOthers(except int, typ byte, v any) {
	for id := range guestJoined {
		if id != except {
			netSend(id, typ, v)
		}
	}
}

// Broadcast from anyone: the host sends to all guests, a guest to the host (which
// passes it on)
func netShare(typ byte, v any) {
	netSend(0, typ, v)
}

// ---- Hosting and joining ----

var guestJoined = map[int]bool{} // host: guests that are connected and have said hello

// Open the world to friends on the same network: this game runs the hub, connects
// to it as the host, and announces the world (see netlan.go)
func startHosting() {
	netConnect("", true) // closes any old connection first
	port, err := startLANHub()
	if err != nil {
		netStatus = "Couldn't open your world: " + err.Error()
		toast(netStatus)
		return
	}
	netDial(fmt.Sprintf("ws://127.0.0.1:%d/host", port))
	netRole, netMyID = netHost, 0
	netStatus = "Opening your world..."
	updateAd()
	startAnnouncing()
}

// Host through the online relay instead (only when one is set up)
func startHostingOnline() {
	u := relayURL()
	if u == "" {
		return
	}
	netConnect(u+"/host", true)
	netRole, netMyID = netHost, 0
	netStatus = "Opening your world online..."
}

// Join a world on the local network: addr is "host:port" (or a typed address)
func joinLAN(addr string) {
	u := lanJoinURL(addr)
	if u == "" {
		return
	}
	beginJoin(u + "/join")
}

// Join through the online relay with a code
func joinWorld(code string) {
	u := relayURL()
	if u == "" {
		netStatus = "Online codes aren't set up in this build"
		return
	}
	beginJoin(u + "/join?code=" + strings.ToUpper(code))
}

func beginJoin(url string) {
	saveGame() // keep the world we're leaving
	netConnect(url, false)
	netRole = netGuest
	netMyID = -1 // until the host says hello back
	netStatus = "Joining..."
	sendHello = true
}

// Stop everything multiplayer: the connection, and the hub if we were hosting
func netShutdown() {
	netClose()
	stopLANHub()
}

var sendHello bool

func leaveMultiplayer() {
	if netRole == netGuest {
		sendChar()
	}
	wasGuest := netRole == netGuest
	netShutdown()
	netStatus = ""
	if wasGuest {
		world = WorldInfo{}
		openScreen(scMain)
	}
}

// ---- Each frame ----

func netUpdate() {
	if netRole == netOff {
		return
	}
	if sendHello && netRole == netGuest {
		sendHello = false
		netSend(0, mHello, helloMsg{Name: playerName(), PID: prefs.PlayerID})
	}
	inboxMu.Lock()
	msgs := inbox
	inbox = nil
	inboxMu.Unlock()
	for _, f := range msgs {
		if len(f) == 1 && f[0] == 0xff { // the connection ended
			reason := failReason.Load()
			wasGuest := netRole == netGuest
			netShutdown()
			netStatus = reason
			if reason != "" {
				toast(reason)
			}
			if wasGuest {
				world = WorldInfo{}
				openScreen(scMultiplayer)
			}
			return
		}
		if netRole == netHost {
			if len(f) < 2 {
				continue
			}
			from := int(f[0])
			if from == 0 {
				hostControl(f[1:])
				continue
			}
			handleMsg(from, f[1], f[2:])
		} else {
			if len(f) < 1 {
				continue
			}
			handleMsg(0, f[0], f[1:])
		}
	}
	if netRole == netGuest && netMyID < 0 {
		return // waiting for the world
	}
	// regular updates
	if frame%3 == 0 {
		netShare(mPState, myState())
	}
	if frame%6 == 0 {
		shareEnts()
	}
	if netRole == netHost && lanServer != nil && frame%30 == 0 {
		updateAd()
	}
	if netRole == netHost && frame%60 == 0 {
		netShare(mTime, timeMsg{T: worldTime, PvP: world.PvP, Boss: bossDown, Mode: world.Mode})
	}
	if netRole == netGuest && frame%600 == 0 {
		sendChar()
	}
	if frame%20 == 0 {
		shareObjsIfChanged()
	}
	flushEdits()
	updatePeers()
}

// Relay notices to the host
func hostControl(b []byte) {
	var c struct {
		T    string `json:"t"`
		Code string `json:"code"`
		ID   int    `json:"id"`
	}
	if json.Unmarshal(b, &c) != nil {
		return
	}
	switch c.T {
	case "code":
		netCode = c.Code
		netStatus = ""
		if codeFile != "" {
			os.WriteFile(codeFile, []byte(c.Code), 0o644)
		}
		toast("Your world is open. Code: " + c.Code)
	case "join":
		// nothing until they say hello
	case "leave":
		if p := peers[c.ID]; p != nil {
			toast(p.st.Name + " left")
		}
		delete(peers, c.ID)
		delete(remoteEnts, c.ID)
		delete(guestJoined, c.ID)
		netSendOthers(c.ID, mGone, goneMsg{ID: c.ID})
	}
}

// ---- Messages ----

type helloMsg struct {
	Name string `json:"name"`
	PID  string `json:"pid"`
}

type welcomeMsg struct {
	You    int               `json:"you"`
	Seed   string            `json:"seed"`
	Mode   int               `json:"mode"`
	Name   string            `json:"name"`
	PvP    bool              `json:"pvp"`
	Time   float64           `json:"time"`
	Boss   bool              `json:"boss"`
	Char   *GuestChar        `json:"char,omitempty"`
	Chunks map[int][]byte    `json:"chunks"`
	Walls  map[int][]byte    `json:"walls"`
	Paint  map[int][]byte    `json:"paint"`
	WPaint map[int][]byte    `json:"wpaint"`
	Objs   []SavedObj        `json:"objs"`
	Peers  map[int]pStateMsg `json:"peers"`
}

type pStateMsg struct {
	From   int     `json:"f"`
	Name   string  `json:"n"`
	X      float64 `json:"x"` // world px
	Y      float64 `json:"y"`
	VX     float64 `json:"vx"`
	VY     float64 `json:"vy"`
	Face   int     `json:"fc"`
	Lift   bool    `json:"l,omitempty"`
	Ground bool    `json:"g,omitempty"`
	Held   string  `json:"h,omitempty"`
	Use    bool    `json:"u,omitempty"`
	Aim    float64 `json:"a"`
	Swing  int     `json:"s"` // frames since the last sword swing
	HP     int     `json:"hp"`
	MaxHP  int     `json:"mhp"`
	Dead   bool    `json:"d,omitempty"`
	Boss   bool    `json:"b,omitempty"` // they've beaten the Warden
}

type cellEdit struct {
	X int   `json:"x"` // world cells
	Y int   `json:"y"`
	L byte  `json:"l"` // layer: 0 blocks, 1 walls, 2 block paint, 3 wall paint
	V uint8 `json:"v"`
}

type editsMsg struct {
	From int        `json:"f"`
	E    []cellEdit `json:"e"`
}

type objsMsg struct {
	From int        `json:"f"`
	Objs []SavedObj `json:"o"`
}

type entState struct {
	ID   int     `json:"i"`
	K    int     `json:"k"`
	X    float64 `json:"x"` // world px
	Y    float64 `json:"y"`
	VX   float64 `json:"vx"`
	VY   float64 `json:"vy"`
	HP   int     `json:"hp"`
	Face int     `json:"fc"`
	Hurt int     `json:"h,omitempty"`
	T    int     `json:"t"`
	Item string  `json:"it,omitempty"`
	N    int     `json:"n,omitempty"`
}

type entsMsg struct {
	From int        `json:"f"`
	E    []entState `json:"e"`
}

type hitMsg struct {
	From int `json:"f"`
	To   int `json:"to"`
	ID   int `json:"i"`
	Dmg  int `json:"d"`
	Dir  int `json:"dir"`
}

type takeMsg struct {
	From int    `json:"f"`
	To   int    `json:"to"`
	ID   int    `json:"i"`
	Item string `json:"it,omitempty"`
	N    int    `json:"n,omitempty"`
}

type timeMsg struct {
	T    float64 `json:"t"`
	PvP  bool    `json:"pvp"`
	Boss bool    `json:"boss"`
	Mode int     `json:"mode"`
}

type charMsg struct {
	From int       `json:"f"`
	PID  string    `json:"pid"`
	Char GuestChar `json:"c"`
}

type goneMsg struct {
	ID int `json:"id"`
}

// A guest's character in someone else's world (kept in that world's save)
type GuestChar struct {
	Name  string   `json:"name"`
	Inv   []*Stack `json:"inv"`
	HP    int      `json:"hp"`
	MaxHP int      `json:"maxHP"`
	WX    float64  `json:"wx"` // world px
	Y     float64  `json:"y"`
	Dead  bool     `json:"dead,omitempty"` // lost for good (Hardcore)
}

// guests[pid] in the host's world; saved with it
var guests = map[string]GuestChar{}

// Messages the host passes on to the other guests as they are
var relayed = map[byte]bool{mPState: true, mEdits: true, mObjs: true, mEnts: true}

func handleMsg(from int, typ byte, b []byte) {
	// directed messages: the host forwards those that aren't for it
	if netRole == netHost && from != 0 {
		switch typ {
		case mHit, mHurt, mTake, mGive:
			var to struct {
				To int `json:"to"`
			}
			json.Unmarshal(b, &to)
			if to.To != 0 {
				if guestJoined[to.To] {
					netSend(to.To, typ, json.RawMessage(b))
				}
				return
			}
		}
		if relayed[typ] && guestJoined[from] {
			netSendOthers(from, typ, json.RawMessage(b))
		}
	}
	switch typ {
	case mHello:
		if netRole == netHost {
			var h helloMsg
			if json.Unmarshal(b, &h) == nil {
				welcomeGuest(from, h)
			}
		}
	case mWelcome:
		var w welcomeMsg
		if json.Unmarshal(b, &w) == nil {
			enterGuestWorld(&w)
		}
	case mPState:
		var s pStateMsg
		if json.Unmarshal(b, &s) == nil {
			peerState(s)
		}
	case mEdits:
		var e editsMsg
		if json.Unmarshal(b, &e) == nil {
			applyEdits(e.E)
		}
	case mObjs:
		var o objsMsg
		if json.Unmarshal(b, &o) == nil {
			loadObjs(o.Objs)
			lastObjs = objsKey()
		}
	case mEnts:
		var e entsMsg
		if json.Unmarshal(b, &e) == nil {
			remoteEntities(e)
		}
	case mHit:
		var h hitMsg
		if json.Unmarshal(b, &h) == nil {
			takeHit(h)
		}
	case mHurt:
		var h hitMsg
		if json.Unmarshal(b, &h) == nil && cur != scDead {
			hurtPlayer(h.Dmg, h.Dir)
		}
	case mTake:
		var t takeMsg
		if json.Unmarshal(b, &t) == nil {
			giveItemTo(t)
		}
	case mGive:
		var t takeMsg
		if json.Unmarshal(b, &t) == nil && items[t.Item] != nil {
			got := addItem(t.Item, t.N)
			if got > 0 {
				sfx("pickup")
				popItem(t.Item, got)
			}
			if t.N > got { // no room: it stays on the ground, ours now
				dropItem(t.Item, t.N-got, player.x, player.y, 0)
			}
		}
	case mDenied:
		if netRole == netGuest {
			netFail("That world is full")
		}
	case mTime:
		var t timeMsg
		if json.Unmarshal(b, &t) == nil {
			// ease toward the host's clock rather than jumping
			d := t.T - worldTime
			if math.Abs(d) > 0.02 && math.Abs(d) < 0.98 {
				worldTime = t.T
			} else {
				worldTime += (math.Mod(d+1.5, 1) - 0.5) * 0.2
				worldTime = math.Mod(worldTime+1, 1)
			}
			world.PvP, world.Mode = t.PvP, t.Mode
			if t.Boss && !bossDown {
				bossDown = true
			}
		}
	case mChar:
		var c charMsg
		if json.Unmarshal(b, &c) == nil && netRole == netHost && c.PID != "" {
			guests[c.PID] = c.Char
		}
	case mGone:
		var g goneMsg
		if json.Unmarshal(b, &g) == nil {
			if p := peers[g.ID]; p != nil {
				toast(p.st.Name + " left")
			}
			delete(peers, g.ID)
			delete(remoteEnts, g.ID)
		}
	}
}

// ---- Joining: the host's side ----

func welcomeGuest(id int, h helloMsg) {
	if len(guestJoined) >= maxPlayers-1 {
		netSend(id, mDenied, struct{}{})
		return
	}
	name := strings.TrimSpace(h.Name)
	if name == "" {
		name = "Guest"
	}
	if len(name) > 16 {
		name = name[:16]
	}
	w := welcomeMsg{You: id, Seed: seedText, Mode: world.Mode, Name: world.Name, PvP: world.PvP, Time: worldTime,
		Boss: bossDown, Objs: saveObjs(), Peers: map[int]pStateMsg{0: myState()}}
	if c, ok := guests[h.PID]; ok && !c.Dead {
		c := c
		w.Char = &c
	}
	// every changed chunk: the saved ones plus the loaded ones that changed
	st := serializeState()
	w.Chunks, w.Walls, w.Paint, w.WPaint = packMap(st.Chunks), packMap(st.Walls), packMap(st.Paint), packMap(st.WPaint)
	for pid, p := range peers {
		w.Peers[pid] = p.st
	}
	netSend(id, mWelcome, w)
	guestJoined[id] = true
	peers[id] = &peer{st: pStateMsg{From: id, Name: name}, pid: h.PID, seen: frame}
	toast(name + " joined")
}

func packMap(m map[int][]uint16) map[int][]byte {
	out := make(map[int][]byte, len(m))
	for k, v := range m {
		b := make([]byte, len(v)*2)
		for i, x := range v {
			binary.LittleEndian.PutUint16(b[i*2:], x)
		}
		out[k] = b
	}
	return out
}

func unpackMap(m map[int][]byte) map[int][]uint16 {
	out := make(map[int][]uint16, len(m))
	for k, b := range m {
		v := make([]uint16, len(b)/2)
		for i := range v {
			v[i] = binary.LittleEndian.Uint16(b[i*2:])
		}
		out[k] = v
	}
	return out
}

// ---- Joining: the guest's side ----

func enterGuestWorld(w *welcomeMsg) {
	netMyID = w.You
	netStatus = ""
	createWorld(w.Seed)         // the same terrain as the host's, from the seed...
	saved = unpackMap(w.Chunks) // ...with all of the host's changes
	savedWalls, savedPaint, savedWPaint = unpackMap(w.Walls), unpackMap(w.Paint), unpackMap(w.WPaint)
	for i := 0; i < NCH; i++ {
		loadSlot(i)
	}
	world = WorldInfo{Name: w.Name, Seed: w.Seed, Mode: w.Mode, PvP: w.PvP}
	worldTime, bossDown = w.Time, w.Boss
	loadObjs(w.Objs)
	lastObjs = objsKey()
	ents = ents[:0]
	for i := range inv {
		inv[i] = nil
	}
	if c := w.Char; c != nil {
		for i := range inv {
			if i < len(c.Inv) && c.Inv[i] != nil && items[c.Inv[i].ID] != nil {
				inv[i] = c.Inv[i]
			}
		}
		player.maxHP, player.hp = max(10, c.MaxHP), c.HP
		if player.hp <= 0 {
			player.hp = player.maxHP
		}
		teleportTo(int(c.WX / CELL))
		player.y = c.Y
		player.x = c.WX - float64(baseChunk*CW*CELL)
		updatePlayerCells()
		unstick()
	} else {
		player.maxHP = 10
		player.hp = player.maxHP
		starterKit(w.Mode >= modeHard)
	}
	for id, s := range w.Peers {
		if id != netMyID {
			peers[id] = &peer{st: s, seen: frame}
			peers[id].snap()
		}
	}
	started = true
	play()
	toast("You joined " + w.Name)
}

func sendChar() {
	if netRole != netGuest || netMyID < 0 {
		return
	}
	c := GuestChar{Name: playerName(), Inv: append([]*Stack(nil), inv...), HP: player.hp, MaxHP: player.maxHP,
		WX: player.x + float64(baseChunk*CW*CELL), Y: player.y, Dead: world.Mode == modeHardcore && cur == scDead}
	netSend(0, mChar, charMsg{From: netMyID, PID: prefs.PlayerID, Char: c})
}

// ---- Names ----

func playerName() string {
	n := strings.TrimSpace(prefs.Name)
	if n == "" {
		return "Mote"
	}
	return n
}

var nameParts = [][]string{
	{"Pip", "Bram", "Fizz", "Nib", "Tuft", "Moss", "Dot", "Wisp", "Glim", "Puck", "Lumen", "Zip"},
	{"bean", "fern", "spark", "pebble", "drop", "bloom", "mote", "flick", "nook", "berry"},
}

func randomPlayerName() string {
	r := func(n int) int { return int(time.Now().UnixNano()/7) % n }
	a := nameParts[0][r(len(nameParts[0]))]
	b := nameParts[1][(r(997)+frame)%len(nameParts[1])]
	return a + b
}
