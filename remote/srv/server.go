// Package srv is Remote's game server: it keeps every island and its creatures, runs
// the fights, and owns every character. The server program (remote/server) runs it
// online, and the game can run it inside itself for playing offline.
package srv

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/coder/websocket"
)

const (
	tickRate  = 20 // simulation steps per second
	dt        = 1.0 / tickRate
	snapEvery = 2 // ticks between snapshots (10 a second)
	saveEvery = 30.0
	idleKick  = 25.0 // seconds without a message before a connection is dropped
	maxOnline = 200
)

type Config struct {
	DataDir string
	Secret  []byte // signs character backups; keep it the same across restarts
	Logf    func(format string, args ...any)
}

type Server struct {
	mu      sync.Mutex
	cfg     Config
	store   *Store
	isles   map[string]*IsleSim
	clients map[int]*Client
	byAcct  map[string]*Client
	nextID  int
	t       float64
	ticks   int
	rng     *mrand.Rand
	saveAt  float64
}

// SecretFrom reads the backup-signing secret from REMOTE_SECRET, or keeps one in the
// data folder (which is lost when Render wipes the disk: set REMOTE_SECRET there).
func SecretFrom(dataDir string) []byte {
	if s := os.Getenv("REMOTE_SECRET"); s != "" {
		return []byte(s)
	}
	p := filepath.Join(dataDir, "secret")
	if b, err := os.ReadFile(p); err == nil && len(b) >= 16 {
		return b
	}
	b := make([]byte, 32)
	rand.Read(b)
	s := []byte(hex.EncodeToString(b))
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(p, s, 0o600)
	return s
}

func New(cfg Config) (*Server, error) {
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	st, err := NewStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, store: st, isles: map[string]*IsleSim{}, clients: map[int]*Client{},
		byAcct: map[string]*Client{}, rng: mrand.New(mrand.NewSource(time.Now().UnixNano()))}
	return s, nil
}

// Run steps the world until ctx ends, then saves everyone
func (s *Server) Run(ctx context.Context) {
	tk := time.NewTicker(time.Second / tickRate)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			for _, c := range s.clients {
				s.saveChar(c)
			}
			s.mu.Unlock()
			return
		case <-tk.C:
			s.mu.Lock()
			s.step()
			s.mu.Unlock()
		}
	}
}

func (s *Server) step() {
	s.t += dt
	s.ticks++
	for _, is := range s.isles {
		is.regrowStep(s)
		if len(is.players) > 0 {
			is.idle = 0
			is.step(s)
		} else {
			is.idle += dt
		}
	}
	if s.ticks%snapEvery == 0 {
		for _, is := range s.isles {
			if len(is.players) > 0 {
				is.sendSnaps(s)
			}
		}
	}
	for _, c := range s.clients {
		if s.t-c.lastMsg > idleKick {
			c.kick("No messages for a while")
		}
		if c.backupDirty && s.t-c.backupAt > 60 {
			s.sendBackup(c)
		}
	}
	if s.t >= s.saveAt {
		s.saveAt = s.t + saveEvery
		for _, c := range s.clients {
			if c.saveDirty {
				s.saveChar(c)
			}
		}
	}
}

// The island sim for an id, built the first time someone goes there
func (s *Server) isle(id string) *IsleSim {
	if is := s.isles[id]; is != nil {
		return is
	}
	d := mmo.IsleByID(id)
	if d == nil {
		return s.isle("haven")
	}
	is := newIsleSim(d)
	s.isles[id] = is
	return is
}

// ---- HTTP ----

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.serveWS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := len(s.clients)
		s.mu.Unlock()
		fmt.Fprintf(w, "Remote server: %d online\n", n)
	})
	return mux
}

type Client struct {
	s           *Server
	id          int
	acct        string
	ch          *mmo.Char
	conn        *websocket.Conn
	out         chan []byte
	cancel      context.CancelFunc
	gone        bool
	isle        *IsleSim
	epoch       int
	x, y        float64
	vx, vy      float64
	face, mode  int
	flags       int
	orbit       float64
	hp          float64
	dead        bool
	invuln      float64
	regenWait   float64
	lavaCD      float64
	petals      []*petal
	lastMsg     float64
	tool        int
	aim         float64 // radians, of the last swing
	swingT      float64 // > 0 just after a swing (shown to others)
	swungAt     float64
	mineAt      float64
	lastIn      float64
	saveDirty   bool
	backupDirty bool
	backupAt    float64
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionContextTakeover})
	if err != nil {
		return
	}
	conn.SetReadLimit(64 << 10)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c := &Client{s: s, conn: conn, out: make(chan []byte, 256), cancel: cancel}
	s.mu.Lock()
	if len(s.clients) >= maxOnline {
		s.mu.Unlock()
		conn.Close(4003, "The server is full")
		return
	}
	s.nextID++
	c.id = s.nextID
	c.lastMsg = s.t
	s.mu.Unlock()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-c.out:
				wctx, wc := context.WithTimeout(ctx, 10*time.Second)
				err := conn.Write(wctx, websocket.MessageText, b)
				wc()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var m mmo.CMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		s.mu.Lock()
		c.lastMsg = s.t
		s.handle(c, &m)
		gone := c.gone
		s.mu.Unlock()
		if gone {
			break
		}
	}
	s.mu.Lock()
	s.drop(c)
	s.mu.Unlock()
	conn.Close(websocket.StatusNormalClosure, "")
}

func (c *Client) send(m *mmo.SMsg) {
	if c.gone {
		return
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	select {
	case c.out <- b:
	default: // hopelessly behind: let it go
		c.kick("")
	}
}

// End a connection after sending a last message
func (c *Client) kick(why string) {
	if c.gone {
		return
	}
	if why != "" {
		b, _ := json.Marshal(&mmo.SMsg{T: "bye", Text: why})
		select {
		case c.out <- b:
		default:
		}
	}
	c.gone = true
	cancel := c.cancel
	time.AfterFunc(300*time.Millisecond, cancel)
	c.s.drop(c)
}

func (s *Server) drop(c *Client) {
	if _, ok := s.clients[c.id]; !ok {
		return
	}
	c.gone = true
	if c.isle != nil {
		c.isle.leave(s, c)
	}
	delete(s.clients, c.id)
	if s.byAcct[c.acct] == c {
		delete(s.byAcct, c.acct)
	}
	if c.ch != nil {
		s.saveChar(c)
		s.cfg.Logf("%s left (%d online)", c.ch.Name, len(s.clients))
	}
}

func (s *Server) saveChar(c *Client) {
	if c.ch == nil {
		return
	}
	c.ch.Seen = time.Now().Unix()
	if c.isle != nil {
		c.ch.Isle = c.isle.def.ID
	}
	c.ch.Seq++
	if err := s.store.Save(c.ch); err != nil {
		s.cfg.Logf("saving %s: %v", c.ch.ID, err)
	}
	c.saveDirty = false
}

func (s *Server) sendBackup(c *Client) {
	c.backupDirty = false
	c.backupAt = s.t
	c.ch.Seq++
	c.send(&mmo.SMsg{T: "backup", Backup: signBackup(s.cfg.Secret, c.ch)})
}

// The character changed: tell its game, and save it soon
func (s *Server) charChanged(c *Client) {
	c.saveDirty, c.backupDirty = true, true
	c.send(&mmo.SMsg{T: "char", Char: c.ch})
}

// ---- Messages ----

func (s *Server) handle(c *Client, m *mmo.CMsg) {
	if c.ch == nil {
		if m.T == "hello" {
			s.hello(c, m)
		}
		return
	}
	switch m.T {
	case "in":
		s.input(c, m)
	case "travel":
		if d := mmo.IsleByID(m.Isle); d != nil && !c.dead && c.isle.def != d {
			s.moveTo(c, s.isle(d.ID))
		}
	case "respawn": // get up where you fell, or back in Haven
		if c.dead {
			c.dead = false
			c.hp = c.ch.MaxHP()
			c.invuln = 2
			to := c.isle
			if m.Isle == "haven" {
				to = s.isle("haven")
			}
			s.moveTo(c, to)
		}
	case "home":
		if !c.dead {
			s.moveTo(c, s.isle("haven"))
		}
	case "equip":
		if c.ch.Equip(m.Slot, m.Petal) == nil {
			s.loadoutChanged(c)
		}
	case "swap":
		c.ch.Swap(m.Slot, m.Slot2)
		s.loadoutChanged(c)
	case "combine":
		if out, err := c.ch.Combine(m.Petal); err != nil {
			c.send(&mmo.SMsg{T: "msg", Text: err.Error()})
		} else {
			c.send(&mmo.SMsg{T: "combined", Text: string(out)})
			s.charChanged(c)
		}
	case "emote":
		if m.Emote >= 0 && m.Emote < len(mmo.Emotes) {
			for _, o := range c.isle.players {
				o.send(&mmo.SMsg{T: "emote", ID: c.id, Emote: m.Emote})
			}
		}
	case "swing":
		c.isle.swing(s, c, m.Aim)
	case "mine":
		c.isle.mine(s, c, m.CX, m.CY)
	case "look":
		if m.Look != nil {
			c.ch.Look = *m.Look
			c.ch.Look.Valid()
			s.charChanged(c)
			c.isle.broadcastInfo(c)
		}
	case "name":
		c.ch.Name = mmo.CleanName(m.Name)
		s.charChanged(c)
		c.isle.broadcastInfo(c)
	}
}

func (s *Server) hello(c *Client, m *mmo.CMsg) {
	if m.Ver != mmo.Protocol {
		c.kick("This game is out of date: update it to play online")
		return
	}
	if !validKey(m.Key) {
		c.kick("Bad sign-in")
		return
	}
	acct := AccountID(m.Key)
	if old := s.byAcct[acct]; old != nil {
		old.kick("You signed in somewhere else")
	}
	ch := s.store.Load(acct)
	if m.Backup != "" {
		if bk, err := readBackup(s.cfg.Secret, m.Backup); err == nil && bk.ID == acct && (ch == nil || bk.Seq > ch.Seq) {
			if ch == nil {
				s.cfg.Logf("restored %s from its backup", bk.Name)
			}
			ch = bk
		}
	}
	if ch == nil {
		ch = mmo.NewChar(acct, m.Name)
	}
	if m.Name != "" {
		ch.Name = mmo.CleanName(m.Name)
	}
	if m.Look != nil {
		ch.Look = *m.Look
	}
	ch.Normalize()
	var away *mmo.Away
	if ch.Seen > 0 {
		away = mmo.AwayGains(ch, mmo.IsleByID(ch.Isle), time.Now().Unix()-ch.Seen, s.rng.Float64)
		if away != nil {
			ch.TakeAway(away)
		}
	}
	c.ch, c.acct = ch, acct
	s.clients[c.id] = c
	s.byAcct[acct] = c
	c.hp = ch.MaxHP()
	c.invuln = 2
	c.makePetals(true)
	c.ch.Seq++
	c.send(&mmo.SMsg{T: "welcome", ID: c.id, Char: ch, Away: away, Backup: signBackup(s.cfg.Secret, ch), Time: s.t})
	s.saveChar(c)
	s.cfg.Logf("%s joined (%d online)", ch.Name, len(s.clients))
	s.moveTo(c, s.isle(ch.Isle))
}

// Put a player on an island at its spawn point
func (s *Server) moveTo(c *Client, is *IsleSim) {
	if c.isle != nil && c.isle != is {
		c.isle.leave(s, c)
	}
	c.isle = is
	c.ch.Isle = is.def.ID
	c.x, c.y = is.isle.SpawnPos()
	c.vx, c.vy = 0, 0
	c.lastIn = s.t
	c.epoch++
	c.orbit = mmo.OrbitNormal
	// "isle" first: the game clears its list of players when it arrives, then hears who's here
	c.send(&mmo.SMsg{T: "isle", Isle: is.def.ID, X: c.x, Y: c.y, Epoch: c.epoch, HP: c.hp, MaxHP: c.ch.MaxHP(), Time: s.t})
	is.join(s, c)
	c.saveDirty = true
}

// The game's own position for its mote. The server trusts it within reason: a mote
// can't move faster than it could fall or be knocked back.
func (s *Server) input(c *Client, m *mmo.CMsg) {
	if m.Epoch != c.epoch || c.dead {
		return
	}
	el := max(0.02, s.t-c.lastIn)
	c.lastIn = s.t
	x := max(0, min(float64(mmo.GW*mmo.Cell)-mmo.PlayerW, m.X))
	y := max(-200, min(float64(mmo.GH*mmo.Cell)-mmo.PlayerH, m.Y))
	dx, dy := x-c.x, y-c.y
	lim := 900*el + 40
	if d := dx*dx + dy*dy; d > lim*lim {
		k := lim / sqrt(d)
		x, y = c.x+dx*k, c.y+dy*k
	}
	c.x, c.y, c.vx, c.vy = x, y, max(-900, min(900, m.VX)), max(-900, min(900, m.VY))
	if m.Face < 0 {
		c.face = -1
	} else {
		c.face = 1
	}
	c.mode = max(0, min(2, m.Mode))
	c.flags = m.Flags & (mmo.FlagLift | mmo.FlagGround)
	c.tool = max(0, min(1, m.Tool))
	if m.Aim != 0 {
		c.aim = m.Aim
	}
}

func (s *Server) loadoutChanged(c *Client) {
	c.makePetals(false)
	s.charChanged(c)
	c.isle.broadcastInfo(c)
}
