package game

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings, stored as JSON in the user's config folder (on Android, the app's files
// folder passed in by the Android activity). Matches aren't saved: they're played live.

type Prefs struct {
	SfxVol   int    `json:"sfxVol"`   // 0-5
	MusicVol int    `json:"musicVol"` // 0-5
	Full     bool   `json:"fullscreen"`
	UISize   int    `json:"uiSize"` // 0 small .. 3 largest, 1 normal
	Name     string `json:"name"`   // your name at the table
	PlayerID string `json:"playerID"`
	Relay    string `json:"relay"` // online server, if not the built-in one
	Fast     bool   `json:"fastAnim"`
	// the last match settings you hosted
	Mode   int `json:"mode"`
	Length int `json:"length"`
	Lives  int `json:"lives"`
}

var prefs = Prefs{SfxVol: 4, MusicVol: 3, Full: true, UISize: 1, Mode: ModeLastStanding, Length: 1, Lives: 3}

// DataDir can be set before the game starts (the Android wrapper does this).
var DataDir string

func dataPath(name string) string {
	dir := DataDir
	if dir == "" {
		if d, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(d, "Desalote")
		} else {
			dir = "."
		}
	}
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, name)
}

func loadPrefs() {
	if b, err := os.ReadFile(dataPath("settings.json")); err == nil {
		json.Unmarshal(b, &prefs)
	}
	prefs.SfxVol = clampInt(prefs.SfxVol, 0, 5)
	prefs.MusicVol = clampInt(prefs.MusicVol, 0, 5)
	prefs.UISize = clampInt(prefs.UISize, 0, 3)
	prefs.Mode = clampInt(prefs.Mode, 0, numModes-1)
	prefs.Length = clampInt(prefs.Length, 0, len(lengthAntes)-1)
	prefs.Lives = clampInt(prefs.Lives, 1, 5)
	if prefs.PlayerID == "" {
		b := make([]byte, 8)
		rand.Read(b)
		prefs.PlayerID = hex.EncodeToString(b)
	}
	if prefs.Name == "" {
		prefs.Name = randomPlayerName()
	}
	savePrefs()
}

func savePrefs() {
	b, _ := json.MarshalIndent(prefs, "", "  ")
	tmp := dataPath("settings.json.tmp")
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, dataPath("settings.json"))
	}
}

var nameA = []string{"Lucky", "Sly", "Velvet", "Brass", "Midnight", "Copper", "Silent", "Jolly", "Crooked", "Gilded"}
var nameB = []string{"Ace", "Knave", "Dealer", "Joker", "Shark", "Queen", "Gambit", "Fox", "Raven", "Bluff"}

func randomPlayerName() string {
	b := make([]byte, 2)
	rand.Read(b)
	return nameA[int(b[0])%len(nameA)] + " " + nameB[int(b[1])%len(nameB)]
}

func playerName() string {
	if prefs.Name == "" {
		return "Player"
	}
	return prefs.Name
}
