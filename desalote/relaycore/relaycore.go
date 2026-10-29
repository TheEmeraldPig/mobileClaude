// Package relaycore passes messages between a host and its guests (up to 3).
// It is a copy of Mote's (mote/relaycore). Desalote runs it inside a game that hosts a
// table on the local network; online, Desalote uses Mote's relay (mote/relay), which
// runs the same code and passes any game's messages.
//
// Frames (binary WebSocket messages):
//
//	host -> relay:  [to][payload]    to = guest id 1-3, or 0 for every guest
//	relay -> host:  [from][payload]  from = guest id; from 0 is the relay itself, with
//	                                 a JSON payload: {"t":"code","code":"ABCDE"},
//	                                 {"t":"join","id":2} or {"t":"leave","id":2}
//	guest -> relay: [payload]        passed to the host as [guest id][payload]
//	relay -> guest: [payload]        from the host
package relaycore

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	maxGuests = 3
	maxFrame  = 8 << 20 // a world's changes, sent to a guest when they join, can be big
	codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type room struct {
	code   string
	host   *websocket.Conn
	mu     sync.Mutex
	guests [maxGuests + 1]*websocket.Conn // index = guest id; 0 unused
	send   chan []byte                    // frames for the host, in order
}

var (
	roomsMu sync.Mutex
	rooms   = map[string]*room{}
)

func newCode() string {
	b := make([]byte, 5)
	for {
		rand.Read(b)
		for i := range b {
			b[i] = codeChars[int(b[i])%len(codeChars)]
		}
		if _, taken := rooms[string(b)]; !taken {
			return string(b)
		}
	}
}

func control(v any) []byte {
	j, _ := json.Marshal(v)
	return append([]byte{0}, j...)
}

func accept(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(maxFrame)
	return c, nil
}

func HostHandler(w http.ResponseWriter, r *http.Request) {
	c, err := accept(w, r)
	if err != nil {
		return
	}
	roomsMu.Lock()
	rm := &room{code: newCode(), host: c, send: make(chan []byte, 512)}
	rooms[rm.code] = rm
	roomsMu.Unlock()
	log.Printf("room %s opened", rm.code)
	ctx, cancel := context.WithCancel(r.Context())
	defer func() {
		cancel()
		roomsMu.Lock()
		delete(rooms, rm.code)
		roomsMu.Unlock()
		rm.mu.Lock()
		for _, g := range rm.guests {
			if g != nil {
				g.Close(4001, "the host left")
			}
		}
		rm.mu.Unlock()
		c.Close(websocket.StatusNormalClosure, "")
		log.Printf("room %s closed", rm.code)
	}()
	// one writer for the host keeps frames in order
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case f := <-rm.send:
				wctx, wcancel := context.WithTimeout(ctx, 20*time.Second)
				err := c.Write(wctx, websocket.MessageBinary, f)
				wcancel()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()
	rm.toHost(control(map[string]any{"t": "code", "code": rm.code}))
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		if len(data) < 1 {
			continue
		}
		to, payload := int(data[0]), data[1:]
		rm.mu.Lock()
		targets := []*websocket.Conn{}
		if to == 0 {
			for _, g := range rm.guests {
				if g != nil {
					targets = append(targets, g)
				}
			}
		} else if to <= maxGuests && rm.guests[to] != nil {
			targets = append(targets, rm.guests[to])
		}
		rm.mu.Unlock()
		for _, g := range targets {
			wctx, wcancel := context.WithTimeout(ctx, 20*time.Second)
			if err := g.Write(wctx, websocket.MessageBinary, payload); err != nil {
				g.Close(websocket.StatusGoingAway, "too slow")
			}
			wcancel()
		}
	}
}

// Queue a frame for the host; a host that can't keep up is dropped
func (rm *room) toHost(f []byte) {
	select {
	case rm.send <- f:
	default:
		rm.host.Close(websocket.StatusGoingAway, "too slow")
	}
}

func JoinHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("code")))
	c, err := accept(w, r)
	if err != nil {
		return
	}
	roomsMu.Lock()
	rm := rooms[code]
	if rm == nil && code == "" && SingleRoom && len(rooms) == 1 {
		for _, only := range rooms {
			rm = only
		}
	}
	roomsMu.Unlock()
	if rm == nil {
		c.Close(4004, "no world with that code")
		return
	}
	rm.mu.Lock()
	id := 0
	for i := 1; i <= maxGuests; i++ {
		if rm.guests[i] == nil {
			id = i
			break
		}
	}
	if id == 0 {
		rm.mu.Unlock()
		c.Close(4003, "that world is full")
		return
	}
	rm.guests[id] = c
	rm.mu.Unlock()
	log.Printf("room %s: guest %d joined", rm.code, id)
	rm.toHost(control(map[string]any{"t": "join", "id": id}))
	defer func() {
		rm.mu.Lock()
		if rm.guests[id] == c {
			rm.guests[id] = nil
		}
		rm.mu.Unlock()
		rm.toHost(control(map[string]any{"t": "leave", "id": id}))
		c.Close(websocket.StatusNormalClosure, "")
		log.Printf("room %s: guest %d left", rm.code, id)
	}()
	for {
		_, data, err := c.Read(r.Context())
		if err != nil {
			return
		}
		rm.toHost(append([]byte{byte(id)}, data...))
	}
}

// SingleRoom lets guests join without a code when there's only one world open (a
// game hosting on the local network)
var SingleRoom bool

// Rooms is how many worlds are open
func Rooms() int {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	return len(rooms)
}

// Mux serves /host and /join
func Mux() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("/host", HostHandler)
	m.HandleFunc("/join", JoinHandler)
	return m
}
