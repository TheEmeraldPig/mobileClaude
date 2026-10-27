package game

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings, stored as JSON in the user's config folder (on Android, the app's files
// folder passed in by the Android activity). Characters live on the server; the game
// keeps only the device's sign-in key and the last signed backup of each character.

type Prefs struct {
	FOV      int               `json:"fov"`
	SfxVol   int               `json:"sfxVol"`   // 0-5
	MusicVol int               `json:"musicVol"` // 0-5
	Binds    map[string]string `json:"binds"`
	StickX   float64           `json:"stickX"`
	StickY   float64           `json:"stickY"`
	BtnSide  string            `json:"btnSide"`
	Full     bool              `json:"fullscreen"`
	UISize   int               `json:"uiSize"` // 0 small .. 3 largest, 1 normal
	Tips     bool              `json:"tips"`
	Simple   bool              `json:"simpleLight"` // always the lighter graphics mode
	Fancy    bool              `json:"fancyLight"`  // never switch to it automatically
	Name     string            `json:"name"`
	Key      string            `json:"key"`     // this device's secret sign-in key (one account per device for now)
	Server   string            `json:"server"`  // game server, if not the built-in one
	Backups  map[string]string `json:"backups"` // signed character backups, by server
}

var prefs = Prefs{FOV: 3, SfxVol: 4, MusicVol: 3, StickX: 0.12, StickY: 0.62, BtnSide: "right", Full: true, UISize: 1, Tips: true,
	Binds: map[string]string{"left": "A", "right": "D", "lev": "W", "petals": "E", "map": "M"}}

// DataDir can be set before the game starts (the Android wrapper does this).
var DataDir string

func dataPath(name string) string {
	dir := DataDir
	if dir == "" {
		if d, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(d, "Remote")
		} else {
			dir = "."
		}
	}
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, name)
}

func loadPrefs() {
	if b, err := os.ReadFile(dataPath("settings.json")); err == nil {
		def := prefs.Binds
		json.Unmarshal(b, &prefs)
		for k, v := range def {
			if prefs.Binds[k] == "" {
				prefs.Binds[k] = v
			}
		}
	}
	if prefs.FOV < 1 || prefs.FOV > 5 {
		prefs.FOV = 3
	}
	prefs.SfxVol = max(0, min(5, prefs.SfxVol))
	prefs.UISize = max(0, min(3, prefs.UISize))
	prefs.MusicVol = max(0, min(5, prefs.MusicVol))
	if prefs.Backups == nil {
		prefs.Backups = map[string]string{}
	}
	if prefs.Key == "" {
		b := make([]byte, 24)
		rand.Read(b)
		prefs.Key = hex.EncodeToString(b)
		savePrefs()
	}
}

func savePrefs() {
	b, _ := json.MarshalIndent(prefs, "", "  ")
	writeFileAtomic(dataPath("settings.json"), b)
}

func writeFileAtomic(path string, b []byte) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, path)
	}
}
