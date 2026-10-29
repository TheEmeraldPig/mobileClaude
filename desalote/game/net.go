package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Multiplayer: up to 4 players at one table.
//
// Everyone connects to a message hub: the host's own game runs one on the local
// network (netlan.go), or an online relay passes messages for a 5-letter code. The
// host's game runs the match (match.go) and sends all of it to everyone whenever it
// changes; guests send their actions (picks, scores, bids, hexes, trades) to the host.
// Each player's cards are dealt and played on their own game.

// RelayURL is the online relay; set at build time with
// -ldflags "-X github.com/TheEmeraldPig/mobileClaude/desalote/game.RelayURL=wss://..."
// and changeable in Settings. It is the same relay Mote uses (it only passes messages).
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

var (
	netRole   int
	netCode   string
	netStatus string // shown in the menus while connecting, or why it ended
	netCancel context.CancelFunc
	netOutQ   chan []byte
	netCtx    context.Context
	inboxMu   sync.Mutex
	inbox     [][]byte
)

// Message types (the first byte of each message; the rest is JSON)
const (
	mHello   = iota + 1 // guest -> host: who I am
	mWelcome            // host -> guest: your player id
	mState              // host -> all: the whole match
	mAct                // guest -> host: something I did
	mEvent              // host -> guest: a message to show
	mDenied             // host -> guest: you can't join
)

const protocol = 1

type helloMsg struct {
	Game  string `json:"game"`
	Proto int    `json:"proto"`
	Name  string `json:"name"`
	PID   string `json:"pid"`
}

// ---- Connection ----

func netConnect(url string) {
	netClose()
	ctx, cancel := context.WithCancel(context.Background())
	netCancel = cancel
	netCtx = ctx
	netOutQ = make(chan []byte, 1024)
	inboxMu.Lock()
	inbox = nil
	inboxMu.Unlock()
	if url != "" {
		netDial(url)
	}
}

// Connect (in the background) to url, for the connection netConnect set up
func netDial(url string) {
	ctx, out := netCtx, netOutQ
	go func() {
		dctx, dcancel := context.WithTimeout(ctx, 75*time.Second) // free servers can take a while to wake
		c, _, err := websocket.Dial(dctx, url, nil)
		dcancel()
		if err != nil {
			netFail("Couldn't reach the table")
			return
		}
		c.SetReadLimit(4 << 20)
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
					reason = "There's no table with that code"
				case 4003:
					reason = "That table is full"
				case 4001:
					reason = "The host closed the table"
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

var failReason syncString

type syncString struct {
	mu sync.Mutex
	s  string
}

func (s *syncString) Store(v string) { s.mu.Lock(); s.s = v; s.mu.Unlock() }
func (s *syncString) Load() string   { s.mu.Lock(); defer s.mu.Unlock(); return s.s }

func netFail(reason string) {
	failReason.Store(reason)
	inboxMu.Lock()
	inbox = append(inbox, []byte{0xff})
	inboxMu.Unlock()
}

func netClose() {
	if netCancel != nil {
		c := netCancel
		time.AfterFunc(400*time.Millisecond, c) // a moment for the last messages to get out
		netCancel = nil
	}
	netRole, netCode = netOff, ""
	guests = map[int]bool{}
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
	default: // hopelessly behind; it will drop soon anyway
	}
}

// ---- Hosting and joining ----

var guests = map[int]bool{} // host: guests that are connected and have said hello

// Open the table to friends on the same network: this game runs the hub, connects to
// it as the host, and announces the table (netlan.go)
func startHostingLAN() {
	netConnect("")
	port, err := startLANHub()
	if err != nil {
		netStatus = "Couldn't open the table: " + err.Error()
		return
	}
	netDial(fmt.Sprintf("ws://127.0.0.1:%d/host", port))
	netRole = netHost
	netStatus = "Opening the table..."
	updateAd()
	startAnnouncing()
}

// Also (or instead) open the table through the online relay, for a join code
func startHostingOnline() {
	u := relayURL()
	if u == "" {
		return
	}
	stopLANHub()
	netConnect(u + "/host")
	netRole = netHost
	netStatus = "Getting a code from the online server..."
}

func joinLAN(addr string) {
	if u := lanJoinURL(addr); u != "" {
		beginJoin(u + "/join")
	}
}

func joinCode(code string) {
	u := relayURL()
	if u == "" {
		netStatus = "Online codes aren't set up in this build"
		return
	}
	beginJoin(u + "/join?code=" + strings.ToUpper(code))
}

var sendHello bool

func beginJoin(url string) {
	netConnect(url)
	netRole = netGuest
	myID = -1 // until the host welcomes us
	netStatus = "Joining..."
	sendHello = true
}

// Stop everything multiplayer: the connection, and the hub if we were hosting
func netShutdown() {
	netClose()
	stopLANHub()
}

// ---- Each frame ----

var lastSentRev = -1

func netUpdate() {
	if netRole == netOff {
		return
	}
	if sendHello && netRole == netGuest {
		sendHello = false
		netSend(0, mHello, helloMsg{Game: "desalote", Proto: protocol, Name: playerName(), PID: prefs.PlayerID})
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
			if wasGuest {
				leaveTable(reason)
			} else if reason != "" {
				toast(reason)
			}
			return
		}
		if netRole == netHost {
			if len(f) < 2 {
				continue
			}
			if f[0] == 0 {
				hostControl(f[1:])
				continue
			}
			hostMsg(int(f[0]), f[1], f[2:])
		} else if len(f) >= 1 {
			guestMsg(f[0], f[1:])
		}
	}
	// the host shares the match whenever it changes (and now and then anyway)
	if netRole == netHost && host != nil && len(guests) > 0 && ((host.Rev != lastSentRev && frame%4 == 0) || frame%90 == 0) {
		lastSentRev = host.Rev
		netSend(0, mState, host)
	}
	if netRole == netHost && lanServer != nil && frame%30 == 0 {
		updateAd()
	}
}

// Notices from the hub to the host
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
		if lanServer != nil {
			netStatus = "Friends on this Wi-Fi can join"
		} else {
			netStatus = "Online: friends join with code " + c.Code
		}
	case "leave":
		if guests[c.ID] {
			delete(guests, c.ID)
			if host != nil {
				host.removePlayer(c.ID)
			}
		}
	}
}

func hostMsg(from int, typ byte, b []byte) {
	switch typ {
	case mHello:
		var h helloMsg
		if json.Unmarshal(b, &h) != nil || h.Game != "desalote" {
			netSend(from, mDenied, map[string]string{"why": "That isn't a Desalote table"})
			return
		}
		if h.Proto != protocol {
			netSend(from, mDenied, map[string]string{"why": "You and the host have different versions of Desalote"})
			return
		}
		if host == nil || host.Phase != "lobby" {
			netSend(from, mDenied, map[string]string{"why": "That match has already started"})
			return
		}
		if len(host.Players) >= host.maxPlayers() {
			netSend(from, mDenied, map[string]string{"why": "That table is full"})
			return
		}
		name := strings.TrimSpace(h.Name)
		if name == "" {
			name = "Guest"
		}
		if len(name) > 16 {
			name = name[:16]
		}
		guests[from] = true
		host.addPlayer(from, name, false)
		host.logf("%s sat down", name)
		netSend(from, mWelcome, map[string]int{"id": from})
		netSend(from, mState, host)
		sfx("open")
	case mAct:
		var a action
		if !guests[from] || json.Unmarshal(b, &a) != nil {
			return
		}
		hostAct(from, a)
	}
}

func guestMsg(typ byte, b []byte) {
	switch typ {
	case mWelcome:
		var w struct {
			ID int `json:"id"`
		}
		if json.Unmarshal(b, &w) == nil {
			myID = w.ID
			netStatus = ""
			openScreen(scTable)
		}
	case mState:
		var m Match
		if json.Unmarshal(b, &m) == nil {
			view = &m
		}
	case mEvent:
		var e struct {
			Text string `json:"text"`
			Sfx  string `json:"sfx"`
		}
		if json.Unmarshal(b, &e) == nil {
			toast(e.Text)
			if e.Sfx != "" {
				sfx(e.Sfx)
			}
		}
	case mDenied:
		var d struct {
			Why string `json:"why"`
		}
		json.Unmarshal(b, &d)
		netShutdown()
		netStatus = d.Why
		toast(d.Why)
	}
}

// Tell one player something (the host itself just sees a toast)
func tell(id int, text, sound string) {
	if id == myID {
		toast(text)
		if sound != "" {
			sfx(sound)
		}
		return
	}
	if guests[id] {
		netSend(id, mEvent, map[string]string{"text": text, "sfx": sound})
	}
}
