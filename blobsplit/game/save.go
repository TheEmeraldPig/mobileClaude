package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is everything that is saved.
type State struct {
	Money    float64        `json:"money"`
	Gems     int            `json:"gems"`
	ValueLv  int            `json:"valueLv"`
	SpeedLv  int            `json:"speedLv"`
	CapLv    int            `json:"capLv"`
	LuckLv   int            `json:"luckLv"`
	Rebirths int            `json:"rebirths"`
	GemLv    map[string]int `json:"gemLv"`
	Auto     bool           `json:"auto"`
	AutoSell bool           `json:"autoSell"`
	Sound    bool           `json:"sound"`
	Found    []int          `json:"found"` // blobs of each rarity ever made
	Blobs    []int          `json:"blobs"` // rarities of the blobs on the board
	Earned   float64        `json:"earned"`
}

func newState() State {
	return State{GemLv: map[string]int{}, Sound: true, Found: make([]int, len(rarities)), Blobs: []int{0}}
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
	if json.Unmarshal(b, &st) != nil {
		return newState()
	}
	if st.GemLv == nil {
		st.GemLv = map[string]int{}
	}
	for len(st.Found) < len(rarities) {
		st.Found = append(st.Found, 0)
	}
	if len(st.Blobs) == 0 {
		st.Blobs = []int{0}
	}
	return st
}

func (g *Game) save() {
	if os.Getenv("BLOB_SHOT") != "" {
		return // screenshot runs don't touch the real save
	}
	g.st.Blobs = g.st.Blobs[:0]
	for _, b := range g.blobs {
		g.st.Blobs = append(g.st.Blobs, b.r)
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
