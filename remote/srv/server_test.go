package srv

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
	"github.com/coder/websocket"
)

type testClient struct {
	t    *testing.T
	c    *websocket.Conn
	ctx  context.Context
	msgs chan *mmo.SMsg
}

func dial(t *testing.T, url string) *testClient {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.SetReadLimit(4 << 20)
	tc := &testClient{t: t, c: c, ctx: ctx, msgs: make(chan *mmo.SMsg, 4096)}
	go func() {
		for {
			_, b, err := c.Read(ctx)
			if err != nil {
				close(tc.msgs)
				return
			}
			var m mmo.SMsg
			json.Unmarshal(b, &m)
			tc.msgs <- &m
		}
	}()
	return tc
}

func (tc *testClient) send(m mmo.CMsg) {
	b, _ := json.Marshal(m)
	if err := tc.c.Write(tc.ctx, websocket.MessageText, b); err != nil {
		tc.t.Fatal(err)
	}
}

// Wait for a message of type typ
func (tc *testClient) want(typ string) *mmo.SMsg {
	tc.t.Helper()
	for m := range tc.msgs {
		if m.T == typ {
			return m
		}
	}
	tc.t.Fatalf("connection ended waiting for %q", typ)
	return nil
}

// A data folder that is removed after the server has finished its last save
func tmpDir(t *testing.T) string {
	d, err := os.MkdirTemp("", "remote-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		time.Sleep(300 * time.Millisecond)
		os.RemoveAll(d)
	})
	return d
}

func startServer(t *testing.T, dir string) (*Server, string, context.CancelFunc) {
	s, err := New(Config{DataDir: dir, Secret: []byte("test-secret-test-secret"), Logf: t.Logf})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	hs := httptest.NewServer(s.Handler())
	t.Cleanup(hs.Close)
	return s, "ws" + strings.TrimPrefix(hs.URL, "http") + "/ws", cancel
}

func TestJoinFightAndLoot(t *testing.T) {
	s, url, stop := startServer(t, tmpDir(t))
	defer stop()
	tc := dial(t, url)
	tc.send(mmo.CMsg{T: "hello", Ver: mmo.Protocol, Key: "test-key-0123456789", Name: "Tester"})
	w := tc.want("welcome")
	if w.Char == nil || w.Char.Name != "Tester" || w.Char.Level != 1 || w.Backup == "" {
		t.Fatalf("bad welcome: %+v", w)
	}
	is := tc.want("isle")
	if is.Isle != "haven" {
		t.Fatalf("started on %q", is.Isle)
	}
	tc.send(mmo.CMsg{T: "travel", Isle: "meadow"})
	is = tc.want("isle")
	if is.Isle != "meadow" {
		t.Fatalf("travelled to %q", is.Isle)
	}
	// wait for creatures, then stand right next to one (as the game would after walking there)
	time.Sleep(600 * time.Millisecond)
	s.mu.Lock()
	sim := s.isles["meadow"]
	if len(sim.mobs) == 0 {
		s.mu.Unlock()
		t.Fatal("no creatures spawned")
	}
	m := sim.mobs[0]
	m.rar, m.hp, m.maxHP = 0, 5, 5 // make it quick
	mx, my := m.x+m.w/2, m.y+m.h/2
	for _, c := range sim.players {
		c.invuln = 100 // don't let the test die
	}
	s.mu.Unlock()
	deadline := time.Now().Add(8 * time.Second)
	got := false
	for !got && time.Now().Before(deadline) {
		s.mu.Lock()
		mx, my = m.x+m.w/2, m.y+m.h/2
		s.mu.Unlock()
		// sit on top of the creature: the petals circle 30 px out, so they sweep over it
		tc.send(mmo.CMsg{T: "in", Epoch: is.Epoch, X: mx - mmo.MoteW/2 + 30, Y: my - mmo.MoteH/2})
		select {
		case msg := <-tc.msgs:
			if msg != nil && msg.T == "loot" {
				got = true
				if msg.Loot.XP <= 0 {
					t.Errorf("no XP in %+v", msg.Loot)
				}
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	if !got {
		t.Fatal("petals never killed the creature")
	}
	ch := tc.want("char")
	if ch.Char.Kills != 1 || ch.Char.XP <= 0 && ch.Char.Level == 1 {
		t.Errorf("kill not counted: %+v", ch.Char)
	}
}

func TestBackupRestoresAfterWipe(t *testing.T) {
	_, url, stop := startServer(t, tmpDir(t))
	tc := dial(t, url)
	tc.send(mmo.CMsg{T: "hello", Ver: mmo.Protocol, Key: "another-key-0123456789", Name: "Keeper"})
	w := tc.want("welcome")
	tc.c.Close(websocket.StatusNormalClosure, "")
	stop()
	// a new server with an empty disk but the same secret
	bk := w.Char
	bk.Coins = 0
	_, url2, stop2 := startServer(t, tmpDir(t))
	defer stop2()
	tc2 := dial(t, url2)
	tc2.send(mmo.CMsg{T: "hello", Ver: mmo.Protocol, Key: "another-key-0123456789", Backup: w.Backup})
	w2 := tc2.want("welcome")
	if w2.Char.Name != "Keeper" {
		t.Fatalf("backup not used: got %q", w2.Char.Name)
	}
	// a forged backup is ignored
	tc3 := dial(t, url2)
	forged := strings.Replace(w.Backup, "a", "b", 1)
	tc3.send(mmo.CMsg{T: "hello", Ver: mmo.Protocol, Key: "third-key-0123456789", Backup: forged})
	if w3 := tc3.want("welcome"); w3.Char.Name != "Mote" {
		t.Fatalf("forged backup accepted: %q", w3.Char.Name)
	}
}

func TestAwayGains(t *testing.T) {
	c := mmo.NewChar("x", "Idle")
	n := 0
	roll := func() float64 { n++; return float64(n%10) / 10 }
	a := mmo.AwayGains(c, mmo.IsleByID("meadow"), 3*3600, roll)
	if a == nil || a.Kills == 0 || a.XP == 0 {
		t.Fatalf("no gains after 3 hours: %+v", a)
	}
	t.Logf("3h on Greenreach at level 1: %d kills, %d XP, %d coins, %v", a.Kills, a.XP, a.Coins, a.Petals)
	if mmo.AwayGains(c, mmo.IsleByID("haven"), 3*3600, roll) != nil {
		t.Error("gains in the safe town")
	}
}
