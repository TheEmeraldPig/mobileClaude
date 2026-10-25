package game

import (
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Test hooks for checking layouts without a phone (run under Xvfb):
//
//	BLOB_SHOT=out.png   save a screenshot after BLOB_SHOT_AT frames (default 90) and quit
//	BLOB_DEMO=1         start with a late-game board: money, gems, upgrades and rare blobs
//	BLOB_PANEL=tree     open a screen: tree, challenge, index, rebirth, blob, world
//	BLOB_CLOCK=0.75     time of day (0.25 midday, 0.75 midnight); BLOB_EVENT=meteor starts an event
//	BLOB_DO=split,sell  actions run one every 20 frames: split, sell, buy0-2, rebirth, mode, win
//	BLOB_SIZE=1280x720  window size;  BLOB_FRESH=1 ignores the save;  BLOB_NOAUDIO=1
//	BLOB_TUT=2          tutorial step;  BLOB_DRAWER=0 closes the upgrades drawer;  BLOB_SCROLL=60
//	BLOB_DO actions also: visitor, ad, buff (all buffs on), away, gift, offer, tab1-3
var (
	shotPath = os.Getenv("BLOB_SHOT")
	shotAt   = 90
	doList   []string
)

// WindowSize returns the size from BLOB_SIZE, or 0, 0.
func WindowSize() (int, int) {
	w, h, ok := strings.Cut(os.Getenv("BLOB_SIZE"), "x")
	if !ok {
		return 0, 0
	}
	a, _ := strconv.Atoi(w)
	b, _ := strconv.Atoi(h)
	return a, b
}

func (g *Game) hookInit() {
	if n, err := strconv.Atoi(os.Getenv("BLOB_SHOT_AT")); err == nil {
		shotAt = n
	}
	if s := os.Getenv("BLOB_DO"); s != "" {
		doList = strings.Split(s, ",")
	}
	g.panel = os.Getenv("BLOB_PANEL")
	if c, err := strconv.ParseFloat(os.Getenv("BLOB_CLOCK"), 64); err == nil {
		g.st.Clock = c * dayLen // 0-1 through the day: 0.25 midday, 0.75 midnight
	}
	if n, err := strconv.Atoi(os.Getenv("BLOB_SEL")); err == nil {
		defer func() { g.sel = n }() // after the demo board is set up
	}
	if n, err := strconv.Atoi(os.Getenv("BLOB_TUT")); err == nil {
		defer func() { g.st.Tut = n }()
	} else if os.Getenv("BLOB_DEMO") != "" || g.panel != "" {
		g.st.Tut = len(tutSteps)
	}
	if os.Getenv("BLOB_DRAWER") == "0" {
		g.st.DrawerShut, g.drawerK = true, 0
	}
	if n, err := strconv.ParseFloat(os.Getenv("BLOB_SCROLL"), 64); err == nil {
		g.scroll = n
	}
	switch g.panel {
	case "away":
		g.away, g.awayGain = 3*3600+720, 4.2e6
	case "visitor":
		g.offer = visitor{buff: 1, gift: true}
	}
	if e := os.Getenv("BLOB_EVENT"); e != "" {
		g.st.Event, g.st.EventLeft, g.eventT = e, 40, -10
	}
	if os.Getenv("BLOB_DEMO") == "" {
		return
	}
	s := &g.st
	s.Money, s.Gems, s.Rebirths, s.Points = 1.6e9, 14, 3, 4
	s.ValueLv, s.SpeedLv, s.CapLv = 60, maxSpeedLv, maxCapLv
	s.Tree = map[string]bool{"origin": true, "quick1": true, "auto": true, "luck1": true, "slot1": true}
	s.Cards = []string{"clover", "shade", "jackpot", "prism"}
	s.Floor, s.Room, s.BestFloor, s.RunFloors, s.RoomEarned, s.RoomSells = 13, 1, 12, 2, 2.1e4, 1
	s.Found = []int{900, 12, 4, 2, 1, 0, 0, 0}
	g.blobs = nil
	for i := 0; i < 64; i++ {
		m := 0
		switch i {
		case 4, 7, 22:
			m = 1 << mShadow
		case 10, 30, 31, 33:
			m = 1 << mGold
		case 17:
			m = 1<<mFrost | 1<<mGold
		case 40:
			m = 1<<mHeart | 1<<mShadow | 1<<mGold
		}
		g.blobs = append(g.blobs, blob{m: m, born: -10, mt: -10, squash: -10, phase: float64(i) * 1.7})
	}
	g.sel = 40
	g.shown = s.Money
}

func (g *Game) hookUpdate() {
	if len(doList) > 0 && g.frame >= 10 && g.frame%20 == 0 {
		a := doList[0]
		doList = doList[1:]
		switch {
		case a == "split":
			g.cool = 99
			g.trySplit(true)
		case a == "sell":
			g.sell()
		case a == "rebirth":
			g.rebirth(false)
		case a == "win":
			g.clearRoom()
		case a == "visitor":
			g.sendVisitor()
			g.vis.t0 = g.t - visitLen*0.4
		case a == "ad":
			g.watchAd(func() { g.st.Money++ })
		case a == "buff":
			for _, b := range buffs {
				if b.secs > 0 {
					g.st.Buffs[b.id] = b.secs
				}
			}
		case a == "away":
			g.giveIdle(5000)
		case a == "gift", a == "offer":
			g.offer = visitor{buff: 0, gift: a == "gift"}
			g.panel = "visitor"
		case strings.HasPrefix(a, "tab"):
			g.shopTab, _ = strconv.Atoi(a[3:])
		case a == "mode":
			g.st.BuyMode = (g.st.BuyMode + 1) % len(buyModes)
		case strings.HasPrefix(a, "buy"):
			i, _ := strconv.Atoi(a[3:])
			g.buyCard(i, false)
		}
	}
}

func (g *Game) hookDraw(screen *ebiten.Image) {
	if shotPath == "" || g.frame < shotAt || g.quit {
		return
	}
	f, err := os.Create(shotPath)
	if err == nil {
		_ = png.Encode(f, screen)
		f.Close()
	}
	g.quit = true
}
