package game

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
)

// State is everything that is saved.
type State struct {
	Ver      int     `json:"ver"`
	Money    float64 `json:"money"`
	Gems     int     `json:"gems"` // DNA, spent in the market
	Points   int     `json:"points"`
	ValueLv  int     `json:"valueLv"`
	SpeedLv  int     `json:"speedLv"`
	CapLv    int     `json:"capLv"`
	Rebirths int     `json:"rebirths"`
	Auto     bool    `json:"auto"`
	AutoSell bool    `json:"autoSell"`
	Sound    bool    `json:"sound"`
	NoMusic  bool    `json:"noMusic"`
	BuyMode  int     `json:"buyMode"` // index into buyModes
	Found    []int   `json:"found"`   // times each mutation has been gained
	Board    []int   `json:"board"`   // mutation masks of the biotes on the board
	Earned   float64 `json:"earned"`

	Tree  map[string]bool `json:"tree"`  // skill tree nodes unlocked
	Cards []string        `json:"cards"` // ability cards held

	// rounds
	Ante        int      `json:"ante"`
	Round       int      `json:"round"` // 0 Petri, 1 Colony, 2 Outbreak
	RoundEarned float64  `json:"roundEarned"`
	RoundSells  int      `json:"roundSells"`
	Boss        int      `json:"boss"`
	RunBosses   int      `json:"runBosses"` // Outbreaks beaten since the last rebirth
	Offers      []string `json:"offers"`    // cards in the market
	Market      bool     `json:"market"`    // the market is open
	MarketMsg   string   `json:"marketMsg"`
	Rerolled    bool     `json:"rerolled"`

	// day, night and events
	Clock     float64 `json:"clock"`
	Event     string  `json:"event"`
	EventLeft float64 `json:"eventLeft"`
	NextEvent float64 `json:"nextEvent"`

	// from older versions
	OldBlobs []int          `json:"blobs,omitempty"`
	OldGemLv map[string]int `json:"gemLv,omitempty"`
}

const stateVer = 2

func newState() State {
	return State{
		Ver: stateVer, Sound: true, Found: make([]int, nMut), Board: []int{0},
		Tree: map[string]bool{}, Ante: 1, Boss: rand.Intn(len(bosses)),
		Clock: dayLen * 0.05, NextEvent: 90,
	}
}

func dataDir() string {
	if DataDir != "" {
		return DataDir
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "BlobSplit")
	}
	return "."
}

func savePath() string { return filepath.Join(dataDir(), "save.json") }

func loadState() State {
	st := newState()
	if os.Getenv("BLOB_FRESH") != "" {
		return st
	}
	b, err := os.ReadFile(savePath())
	if err != nil {
		return st
	}
	st.Ver = 0 // a save without a version is from Blob Split
	if json.Unmarshal(b, &st) != nil {
		return newState()
	}
	if st.Ver < 2 {
		// Blob Split saves: a board of single rarities becomes masks, and the gem
		// shop is replaced by the skill tree, so rebirths so far turn into points.
		st.Board = nil
		for _, r := range st.OldBlobs {
			if r > 0 && r < nMut {
				st.Board = append(st.Board, 1<<r)
			} else {
				st.Board = append(st.Board, 0)
			}
		}
		st.Points += 2 * st.Rebirths
		st.Auto, st.AutoSell = false, false
		st.Ver = stateVer
	}
	st.OldBlobs, st.OldGemLv = nil, nil
	if st.Tree == nil {
		st.Tree = map[string]bool{}
	}
	for len(st.Found) < nMut {
		st.Found = append(st.Found, 0)
	}
	if len(st.Board) == 0 {
		st.Board = []int{0}
	}
	if st.Ante < 1 {
		st.Ante = 1
	}
	return st
}

func (g *Game) save() {
	if os.Getenv("BLOB_SHOT") != "" {
		return // screenshot runs don't touch the real save
	}
	g.st.Board = g.st.Board[:0]
	for _, b := range g.blobs {
		g.st.Board = append(g.st.Board, b.m)
	}
	b, err := json.Marshal(g.st)
	if err != nil {
		return
	}
	_ = os.MkdirAll(dataDir(), 0o755)
	tmp := savePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, savePath())
	}
}

var current *Game

// SaveNow is called by the Android app when it goes to the background.
func SaveNow() {
	if current != nil {
		current.mu.Lock()
		defer current.mu.Unlock()
		current.save()
	}
}
