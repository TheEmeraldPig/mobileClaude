package game

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

// Settings and save game, stored as JSON in the user's config folder
// (on Android, the app's files folder passed in by the Android activity).

type Prefs struct {
	LastWorld string            `json:"lastWorld"`
	FOV       int               `json:"fov"`
	SfxVol    int               `json:"sfxVol"`   // 0-5
	MusicVol  int               `json:"musicVol"` // 0-5
	Binds     map[string]string `json:"binds"`
	StickX    float64           `json:"stickX"`
	StickY    float64           `json:"stickY"`
	BtnSide   string            `json:"btnSide"`
	Seed      string            `json:"seed"`
	Full      bool              `json:"fullscreen"`
}

var prefs = Prefs{FOV: 3, SfxVol: 4, MusicVol: 3, StickX: 0.12, StickY: 0.62, BtnSide: "right", Full: true,
	Binds: map[string]string{"left": "A", "right": "D", "lev": "W", "inv": "E", "drop": "Q", "craft": "C"}}

// DataDir can be set before the game starts (the Android wrapper does this).
var DataDir string

func dataPath(name string) string {
	dir := DataDir
	if dir == "" {
		if d, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(d, "Mote")
		} else {
			dir = "."
		}
	}
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, name)
}

func loadPrefs() {
	b, err := os.ReadFile(dataPath("settings.json"))
	if err != nil {
		return
	}
	def := prefs.Binds
	json.Unmarshal(b, &prefs)
	for k, v := range def {
		if prefs.Binds[k] == "" {
			prefs.Binds[k] = v
		}
	}
	if prefs.FOV < 1 || prefs.FOV > 5 {
		prefs.FOV = 3
	}
	prefs.SfxVol = max(0, min(5, prefs.SfxVol))
	prefs.MusicVol = max(0, min(5, prefs.MusicVol))
}

func savePrefs() {
	b, _ := json.MarshalIndent(prefs, "", "  ")
	writeFileAtomic(dataPath("settings.json"), b)
}

// ---- Save game ----

type SaveState struct {
	V         int              `json:"v"`
	Seed      string           `json:"seed"`
	BaseChunk int              `json:"baseChunk"`
	Chunks    map[int][]uint16 `json:"chunks"`
	Started   bool             `json:"started"`
	PlayerWX  float64          `json:"playerWX"`
	PlayerY   float64          `json:"playerY"`
	Energy    float64          `json:"energy"`
	Face      int              `json:"face"`
	Inv       []*Stack         `json:"inv"`
	Selected  int              `json:"selected"`
	Time      float64          `json:"time"`
	HP        int              `json:"hp"`
	SpawnWX   int              `json:"spawnWX"`
	SpawnY    int              `json:"spawnY"`
	Drops     []SavedDrop      `json:"drops"`
	MaxHP     int              `json:"maxHP"`
	World     WorldInfo        `json:"world"`
	BossDown  bool             `json:"bossDown"`
	Objs      []SavedObj       `json:"objs"`
	Walls     map[int][]uint16 `json:"walls,omitempty"`
	Paint     map[int][]uint16 `json:"paint,omitempty"`
	WPaint    map[int][]uint16 `json:"wpaint,omitempty"`
	Run       struct {         // older saves kept the boss here
		BossDown bool `json:"bossDown"`
	} `json:"run"`
}

type SavedDrop struct {
	ID string  `json:"id"`
	N  int     `json:"n"`
	WX float64 `json:"wx"`
	Y  float64 `json:"y"`
}

// The whole game state. When you add new state (items, player stats...), add it here
// and in restoreState so it survives saves and updates.
func serializeState() SaveState {
	chunks := map[int][]uint16{}
	for k, v := range saved {
		chunks[k] = v
	}
	for i := 0; i < NCH; i++ {
		if dirty[i] {
			chunks[baseChunk+i] = packChunk(i)
			saveLayers(i)
		}
	}
	var drops []SavedDrop
	for _, e := range ents {
		if e.kind == entItem {
			drops = append(drops, SavedDrop{e.item, e.n, e.x + float64(baseChunk*CW*CELL), e.y})
		}
	}
	return SaveState{V: 1, Seed: seedText, BaseChunk: baseChunk, Chunks: chunks, Started: started,
		PlayerWX: player.x + float64(baseChunk*CW*CELL), PlayerY: player.y, Energy: player.energy, Face: player.face,
		Inv: append([]*Stack(nil), inv...), Selected: selected, Time: worldTime, HP: player.hp, SpawnWX: spawnWX, SpawnY: spawnY, Drops: drops,
		MaxHP: player.maxHP, World: world, BossDown: bossDown, Objs: saveObjs(),
		Walls: savedWalls, Paint: savedPaint, WPaint: savedWPaint}
}

func restoreState(st *SaveState) bool {
	if st == nil || st.V != 1 || st.Seed == "" {
		return false
	}
	seedText = st.Seed
	seedN = seedFrom(seedText)
	prefs.Seed = seedText
	saved = map[int][]uint16{}
	for k, v := range st.Chunks {
		saved[k] = v
	}
	savedWalls, savedPaint, savedWPaint = map[int][]uint16{}, map[int][]uint16{}, map[int][]uint16{}
	for _, l := range []struct{ to, from map[int][]uint16 }{{savedWalls, st.Walls}, {savedPaint, st.Paint}, {savedWPaint, st.WPaint}} {
		for k, v := range l.from {
			l.to[k] = v
		}
	}
	baseChunk = st.BaseChunk
	ents = ents[:0]
	for i := 0; i < NCH; i++ {
		loadSlot(i)
	}
	player.vx, player.vy, player.squash = 0, 0, 0
	player.energy, player.face = st.Energy, st.Face
	if player.face == 0 {
		player.face = 1
	}
	player.x = st.PlayerWX - float64(baseChunk*CW*CELL)
	player.y = st.PlayerY
	followPlayer()
	unstick()
	updatePlayerCells()
	for i := range inv {
		inv[i] = nil
		if i < len(st.Inv) && st.Inv[i] != nil && items[st.Inv[i].ID] != nil {
			inv[i] = st.Inv[i]
		}
	}
	selected = st.Selected
	if selected < -1 || selected >= hotbarN {
		selected = -1
	}
	worldTime, player.hp, spawnWX, spawnY = st.Time, st.HP, st.SpawnWX, st.SpawnY
	player.maxHP, bossDown = st.MaxHP, st.BossDown || st.Run.BossDown
	if player.maxHP <= 0 {
		player.maxHP = 10
	}
	world = st.World
	loadObjs(st.Objs)
	if player.hp <= 0 {
		player.hp = player.maxHP
	}
	for _, d := range st.Drops {
		if items[d.ID] != nil {
			dropItem(d.ID, d.N, d.WX-float64(baseChunk*CW*CELL), d.Y, 0)
		}
	}
	started = st.Started
	return true
}

// Save the world being played (the demo world behind the menu is never saved)
func saveGame() {
	if world.ID == "" || cur == scDead && deathLost {
		return
	}
	st := serializeState()
	b, err := json.Marshal(st)
	if err == nil {
		writeFileAtomic(worldFile(world.ID), b)
		touchWorld()
	}
}

func writeFileAtomic(path string, b []byte) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, path)
	}
}

func randomSeed() string { return strconv.Itoa(100000000 + rand.Intn(900000000)) }
