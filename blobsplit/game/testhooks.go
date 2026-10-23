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
//	BLOB_PANEL=shop     open a panel: shop, index or rebirth
//	BLOB_DO=split,sell  actions run one every 20 frames: split, sell, buy0-3, rebirth, mode
//	BLOB_SIZE=1280x720  window size;  BLOB_FRESH=1 ignores the save;  BLOB_NOAUDIO=1
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
	if os.Getenv("BLOB_DEMO") == "" {
		return
	}
	s := &g.st
	s.Money, s.Gems, s.Rebirths = 1.6e9, 14, 3
	s.ValueLv, s.SpeedLv, s.CapLv, s.LuckLv = 60, maxSpeedLv, maxCapLv, 6
	s.GemLv = map[string]int{"auto": 1, "golden": 2}
	s.Auto = false
	s.Found = []int{900, 12, 4, 2, 1, 0, 0, 0}
	g.blobs = nil
	for i := 0; i < 64; i++ {
		r := 0
		switch i {
		case 4, 7, 22:
			r = 2
		case 10, 30, 31, 33:
			r = 1
		case 17:
			r = 3
		case 40:
			r = 4
		}
		g.blobs = append(g.blobs, blob{r: r, born: -10, mt: -10, squash: -10, phase: float64(i) * 1.7})
	}
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
			g.rebirth()
		case a == "mode":
			g.st.BuyMode = (g.st.BuyMode + 1) % len(buyModes)
		case strings.HasPrefix(a, "buy"):
			i, _ := strconv.Atoi(a[3:])
			g.buyCard(i, false)
		}
	}
}

func (g *Game) hookDraw(screen *ebiten.Image) {
	if shotPath == "" || g.frame != shotAt {
		return
	}
	f, err := os.Create(shotPath)
	if err == nil {
		_ = png.Encode(f, screen)
		f.Close()
	}
	g.quit = true
}
