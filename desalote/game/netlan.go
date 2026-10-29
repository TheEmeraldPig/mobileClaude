package game

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TheEmeraldPig/mobileClaude/desalote/relaycore"
)

// Playing on the same Wi-Fi (or a phone's hotspot), with no server anywhere: the hosting
// game runs the message hub itself (relaycore, the same code as the online relay) and
// announces its table on the network a few times a second. Games on the Join screen
// hear the announcements and list the tables; joining connects straight to the host.
// Friends can also join by typing the host's address. (The ports differ from Mote's,
// so the two games never see each other's announcements.)

const (
	lanPort  = 47480 // the host's hub (it tries a few ports up from here)
	findPort = 47479 // announcements
)

var (
	lanServer  *http.Server
	lanAt      int // the port the hub listens on
	announceOn bool
)

// Start the hub on this device (once)
func startLANHub() (int, error) {
	if lanServer != nil {
		return lanAt, nil
	}
	relaycore.SingleRoom = true
	var ln net.Listener
	var err error
	for p := lanPort; p < lanPort+10; p++ {
		if ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p)); err == nil {
			lanAt = p
			break
		}
	}
	if err != nil {
		return 0, err
	}
	lanServer = &http.Server{Handler: relaycore.Mux()}
	go lanServer.Serve(ln)
	return lanAt, nil
}

func stopLANHub() {
	announceOn = false
	if lanServer != nil {
		lanServer.Close()
		lanServer = nil
	}
}

// ---- Announcing a hosted table ----

type lanAd struct {
	Desalote int    `json:"desalote"` // 1: this is a Desalote announcement
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Players  int    `json:"players"`
	Max      int    `json:"max"`
	Code     string `json:"code"`
	Mode     int    `json:"mode"`
	Open     bool   `json:"open"` // still in the lobby
}

var (
	adMu  sync.Mutex
	adNow lanAd // what the announcer sends (updated from the game each second)
)

func startAnnouncing() {
	if announceOn {
		return
	}
	announceOn = true
	go func() {
		c, err := net.ListenUDP("udp4", nil)
		if err != nil {
			return
		}
		defer c.Close()
		for announceOn {
			adMu.Lock()
			b, _ := json.Marshal(adNow)
			adMu.Unlock()
			for _, a := range broadcastAddrs() {
				c.WriteToUDP(b, &net.UDPAddr{IP: a, Port: findPort})
			}
			time.Sleep(700 * time.Millisecond)
		}
	}()
}

// Called each second while hosting
func updateAd() {
	adMu.Lock()
	if host != nil {
		adNow = lanAd{Desalote: 1, Host: playerName(), Port: lanAt, Players: len(host.Players), Max: host.maxPlayers(),
			Code: netCode, Mode: host.Mode, Open: host.Phase == "lobby"}
	}
	adMu.Unlock()
}

// Everywhere to send announcements: the general broadcast address plus each local
// network's own (some phones and routers only pass one or the other)
func broadcastAddrs() []net.IP {
	out := []net.IP{net.IPv4bcast, net.IPv4(127, 0, 0, 1)} // (and this device, for a second copy of the game)
	for _, ip := range localIPs() {
		if v4 := ip.To4(); v4 != nil {
			out = append(out, net.IPv4(v4[0], v4[1], v4[2], 255))
		}
	}
	return out
}

// This device's addresses on local networks
func localIPs() []net.IP {
	var out []net.IP
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && !n.IP.IsLoopback() && !n.IP.IsLinkLocalUnicast() {
				out = append(out, n.IP.To4())
			}
		}
	}
	if len(out) == 0 {
		// newer Android hides the interface list: ask which address would be used to
		// reach out (nothing is actually sent)
		if c, err := net.Dial("udp4", "192.168.0.1:9"); err == nil {
			if a, ok := c.LocalAddr().(*net.UDPAddr); ok && !a.IP.IsLoopback() {
				out = append(out, a.IP.To4())
			}
			c.Close()
		}
	}
	return out
}

// The address friends can type to join (the first local one)
func myAddress() string {
	ips := localIPs()
	if len(ips) == 0 {
		return ""
	}
	// prefer the usual home and hotspot ranges
	sort.SliceStable(ips, func(a, b int) bool { return ips[a][0] == 192 && ips[b][0] != 192 })
	return ips[0].String()
}

// ---- Finding tables ----

type foundTable struct {
	ad   lanAd
	addr string // host:port to connect to
	seen time.Time
}

var (
	findMu   sync.Mutex
	found    = map[string]*foundTable{}
	findConn *net.UDPConn
)

func startFinding() {
	if findConn != nil {
		return
	}
	c, err := net.ListenUDP("udp4", &net.UDPAddr{Port: findPort})
	if err != nil {
		return
	}
	findConn = c
	go func() {
		buf := make([]byte, 2048)
		for {
			n, from, err := c.ReadFromUDP(buf)
			if err != nil {
				return
			}
			var ad lanAd
			if json.Unmarshal(buf[:n], &ad) != nil || ad.Desalote != 1 || ad.Port == 0 {
				continue
			}
			addr := net.JoinHostPort(from.IP.String(), fmt.Sprint(ad.Port))
			// one entry per hosted table (it can be heard on more than one address;
			// keep the network one over this device's own)
			key := ad.Code + "|" + ad.Host + "|" + fmt.Sprint(ad.Port)
			findMu.Lock()
			if old := found[key]; old == nil || !from.IP.IsLoopback() || strings.HasPrefix(old.addr, "127.") {
				found[key] = &foundTable{ad: ad, addr: addr, seen: time.Now()}
			} else {
				old.seen = time.Now()
			}
			findMu.Unlock()
		}
	}()
}

func stopFinding() {
	if findConn != nil {
		findConn.Close()
		findConn = nil
	}
}

// Tables heard from in the last few seconds (not our own)
func foundTables() []*foundTable {
	findMu.Lock()
	defer findMu.Unlock()
	var out []*foundTable
	for k, f := range found {
		if time.Since(f.seen) > 4*time.Second {
			delete(found, k)
			continue
		}
		if lanServer != nil && f.ad.Port == lanAt && isMyIP(f.addr) {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].addr < out[b].addr })
	return out
}

func isMyIP(addr string) bool {
	h, _, _ := net.SplitHostPort(addr)
	for _, ip := range localIPs() {
		if ip.String() == h {
			return true
		}
	}
	return h == "127.0.0.1"
}

// Turn what someone typed into a join address: "192.168.1.5" or "192.168.1.5:47471"
func lanJoinURL(typed string) string {
	t := strings.TrimSpace(typed)
	if t == "" {
		return ""
	}
	if !strings.Contains(t, ":") {
		t = net.JoinHostPort(t, fmt.Sprint(lanPort))
	}
	return "ws://" + t
}
