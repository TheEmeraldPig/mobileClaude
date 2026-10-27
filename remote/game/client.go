package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/TheEmeraldPig/mobileClaude/remote/srv"
	"github.com/coder/websocket"
)

// Talking to the game server (remote/srv). The server owns characters, creatures and
// fights; the game moves its own mote, sends where it is ten times a second, and draws
// what the server says is around it.
//
// "Play solo" runs the same server inside the game on this device, so the game works
// with no internet (with its own characters, kept in the "offline" folder).

// ServerURL is the online server; set at build time with
// -ldflags "-X github.com/TheEmeraldPig/mobileClaude/remote/game.ServerURL=wss://..."
// and changeable in Settings.
var ServerURL = ""

const (
	netOff = iota
	netConnecting
	netOnline
)

var (
	netState     int
	netSolo      bool   // connected to the server inside this game
	netStatus    string // shown on the main menu: what's happening, or why it ended
	netURL       string
	netSince     int // frame the connection attempt started
	netCancel    context.CancelFunc
	netOut       chan []byte
	inboxMu      sync.Mutex
	inbox        [][]byte
	me           *mmo.Char
	myID         int
	epoch        int
	isleID       string
	srvT         float64 // server time at srvTFrame
	srvTFrame    int
	myHP         float64
	myMaxHP      float64 = 100
	myAlive              = -1 // bit i: petal i is out
	myOrbit              = mmo.OrbitNormal
	myMode       int
	awayNews     *mmo.Away
	online       int
	isleCounts   = map[string]int{}
	localSrv     *srv.Server
	localURL     string
	reconnecting bool
)

// The server's clock, running on between snapshots
func serverTime() float64 { return srvT + float64(frame-srvTFrame)/60 }

func serverURL() string {
	u := strings.TrimSpace(prefs.Server)
	if u == "" {
		u = ServerURL
	}
	if u == "" {
		return ""
	}
	u = strings.TrimRight(u, "/")
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + u[len("https://"):]
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + u[len("http://"):]
	case !strings.HasPrefix(u, "ws://") && !strings.HasPrefix(u, "wss://"):
		u = "wss://" + u
	}
	if !strings.HasSuffix(u, "/ws") {
		u += "/ws"
	}
	return u
}

func playOnline() {
	u := serverURL()
	if u == "" {
		netStatus = "No online server is set up yet: try Play solo"
		return
	}
	netSolo = false
	connect(u)
}

func playSolo() {
	if localSrv == nil {
		dir := dataPath("offline")
		s, err := srv.New(srv.Config{DataDir: dir, Secret: srv.SecretFrom(dir), Logf: func(string, ...any) {}})
		if err != nil {
			netStatus = "Couldn't start solo play: " + err.Error()
			return
		}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			netStatus = "Couldn't start solo play: " + err.Error()
			return
		}
		go http.Serve(ln, s.Handler())
		go s.Run(context.Background())
		localSrv = s
		localURL = fmt.Sprintf("ws://%s/ws", ln.Addr())
	}
	netSolo = true
	connect(localURL)
}

func connect(url string) {
	netClose()
	ctx, cancel := context.WithCancel(context.Background())
	netCancel, netURL, netState, netSince = cancel, url, netConnecting, frame
	netStatus = "Connecting..."
	out := make(chan []byte, 256)
	netOut = out
	inboxMu.Lock()
	inbox = nil
	inboxMu.Unlock()
	push := func(b []byte) {
		inboxMu.Lock()
		inbox = append(inbox, b)
		inboxMu.Unlock()
	}
	go func() {
		dctx, dc := context.WithTimeout(ctx, 90*time.Second) // a sleeping free server takes a while to wake
		c, _, err := websocket.Dial(dctx, url, nil)
		dc()
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("connecting to %s: %v", url, err)
				push([]byte(`{"t":"fail","text":"Couldn't reach the server"}`))
			}
			return
		}
		c.SetReadLimit(8 << 20)
		push([]byte(`{"t":"open"}`))
		go func() {
			for {
				select {
				case <-ctx.Done():
					c.Close(websocket.StatusNormalClosure, "")
					return
				case b := <-out:
					wctx, wc := context.WithTimeout(ctx, 20*time.Second)
					err := c.Write(wctx, websocket.MessageText, b)
					wc()
					if err != nil {
						return
					}
				}
			}
		}()
		for {
			_, b, err := c.Read(ctx)
			if err != nil {
				if ctx.Err() == nil {
					push([]byte(`{"t":"fail","text":"Lost the connection to the server"}`))
				}
				return
			}
			push(b)
		}
	}()
}

func netClose() {
	if netCancel != nil {
		c := netCancel
		time.AfterFunc(300*time.Millisecond, c) // let the last messages out
		netCancel = nil
	}
	netState = netOff
	others = map[int]*other{}
	ents = ents[:0]
}

func send(m mmo.CMsg) {
	if netState == netOff || netOut == nil {
		return
	}
	b, _ := json.Marshal(m)
	select {
	case netOut <- b:
	default:
	}
}

// Back to the main menu, saying why
func leaveGame(why string) {
	netClose()
	netStatus = why
	if why != "" && cur != scMain {
		toast(why)
	}
	cur = scMain
	clearPointers()
}

// ---- Each frame ----

func netUpdate() {
	if netState == netOff {
		return
	}
	if netState == netConnecting && !netSolo && frame-netSince == 300 {
		netStatus = "Waking the server up. This can take up to a minute..."
	}
	inboxMu.Lock()
	msgs := inbox
	inbox = nil
	inboxMu.Unlock()
	for _, b := range msgs {
		var m mmo.SMsg
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		handle(&m)
		if netState == netOff {
			return
		}
	}
	if netState == netOnline && cur != scMain && frame%6 == 0 {
		fl := 0
		if player.lifting {
			fl |= mmo.FlagLift
		}
		if player.onGround {
			fl |= mmo.FlagGround
		}
		send(mmo.CMsg{T: "in", Epoch: epoch, X: player.x, Y: player.y, VX: player.vx * 60, VY: player.vy * 60,
			Face: player.face, Mode: myMode, Flags: fl})
	}
}

func handle(m *mmo.SMsg) {
	switch m.T {
	case "open":
		netStatus = "Signing in..."
		send(mmo.CMsg{T: "hello", Ver: mmo.Protocol, Key: prefs.Key, Name: prefs.Name, Backup: prefs.Backups[netURL]})
	case "fail":
		// a connection that was working dropped (a phone went to sleep, the Wi-Fi blinked):
		// try once more straight away
		if netState == netOnline && !reconnecting {
			reconnecting = true
			url := netURL
			connect(url)
			netStatus = "Reconnecting..."
			toast("Reconnecting...")
			return
		}
		leaveGame(m.Text)
	case "bye":
		leaveGame(m.Text)
	case "welcome":
		netState = netOnline
		reconnecting = false
		netStatus = ""
		myID, me, awayNews = m.ID, m.Char, m.Away
		prefs.Name = me.Name
		keepBackup(m.Backup)
		srvT, srvTFrame = m.Time, frame
	case "backup":
		keepBackup(m.Backup)
	case "isle":
		if m.Isle != isleID || curIsle == nil {
			isleID = m.Isle
			loadIsle(m.Isle)
		}
		ents = ents[:0]
		others = map[int]*other{}
		epoch = m.Epoch
		myHP, myMaxHP = m.HP, m.MaxHP
		srvT, srvTFrame = m.Time, frame
		spawnPlayerAt(m.X, m.Y)
		player.invuln = 0
		if awayNews != nil {
			openScreen(scAway)
		} else {
			play()
		}
		if d := mmo.IsleByID(m.Isle); d != nil {
			toast(strings.ToUpper(d.Name))
		}
	case "snap":
		srvT, srvTFrame = m.Time, frame
		readSnap(m)
	case "char":
		if m.Char != nil {
			if me != nil && m.Char.Level > me.Level {
				myMaxHP = m.Char.MaxHP()
			}
			me = m.Char
		}
	case "pinfo":
		if m.Info != nil {
			o := otherFor(m.Info.ID)
			o.info = m.Info
		}
	case "gone":
		delete(others, m.ID)
	case "hurt":
		myHP = m.HP
		p := &player
		if m.Dir == 0 {
			m.Dir = -p.face
		}
		p.vx, p.vy = float64(m.Dir)*4.5, -4
		p.invuln = 36
		popDamage(m.Dmg, p.x+p.w/2, p.y, true)
		shake = max(shake, 5)
		burst(10, (p.x+p.w/2)/CELL, (p.y+p.h/2)/CELL, [3]int{240, 122, 106}, 1.4, 1.4, 22)
		sfx("hurt")
	case "dead":
		myHP = 0
		sfx("die")
		openScreen(scDead)
	case "emote":
		if m.ID == myID {
			myEmote, myEmoteUntil = m.Emote, frame+180
		} else if o := others[m.ID]; o != nil {
			o.emote, o.emoteUntil = m.Emote, frame+180
		}
	case "loot":
		gotLoot(m.Loot)
	case "combined":
		toast("Made a " + mmo.PetalID(m.Text).Name() + "!")
		sfx("levelup")
	case "msg":
		toast(m.Text)
	}
}

// Keep the server's signed copy of the character (not for solo play: that server's
// disk is this device, and its address changes every time)
func keepBackup(b string) {
	if b != "" && netURL != "" && !netSolo {
		prefs.Backups[netURL] = b
		savePrefs()
	}
}

func gotLoot(l *mmo.Loot) {
	if l == nil {
		return
	}
	x, y := float64(l.X), float64(l.Y)
	if k := mmo.MobByID(l.Mob); k != nil {
		c := mmo.Rarities[l.Rar].Col
		burst(16, x/CELL, y/CELL, c, 1.6, 1.2, 30)
	}
	sfx("die")
	popText(fmt.Sprintf("+%d XP", l.XP), x, y-10, colL)
	if l.Coins > 0 {
		popText(fmt.Sprintf("+%d coins", l.Coins), x, y+4, colT)
	}
	for i, p := range l.Petals {
		_, r, _ := p.Split()
		c := mmo.Rarities[r].Col
		popText(p.Name(), x, y-26-float64(i)*12, rgb(uint32(c[0])<<16|uint32(c[1])<<8|uint32(c[2])))
		sfx("pickup")
	}
	if l.Levels > 0 && me != nil {
		toast(fmt.Sprintf("LEVEL %d!", me.Level+l.Levels))
		sfx("levelup")
		burst(30, (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL, tipRGB, 2, 2, 40)
	}
}

// ---- Snapshots ----

func readSnap(m *mmo.SMsg) {
	seen := map[int]bool{}
	for _, s := range m.P {
		if s.ID == myID {
			myHP, myMaxHP, myAlive = float64(s.HP), float64(s.MaxHP), s.Alive
			continue
		}
		o := otherFor(s.ID)
		if o.seen == 0 {
			o.x, o.y = float64(s.X), float64(s.Y)
		}
		o.st, o.seen = s, frame
		seen[s.ID] = true
	}
	for id, o := range others {
		if !seen[id] && o.seen > 0 {
			delete(others, id)
		}
	}
	readMobs(m.M)
}
