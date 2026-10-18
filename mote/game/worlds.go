package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Worlds and game modes. Each world is its own save file (worlds/<id>.json) holding the
// terrain, the mote and its inventory. worlds.json lists them for the menu.
//
// Modes decide how hard dying is:
//
//	Easy      less damage, keep everything when you die
//	Normal    drop your blocks and materials, keep tools and utility items
//	Hard      more damage, drop everything, respawn with a temporary kit
//	Hardcore  like Hard, but dying deletes the world

const (
	modeEasy = iota
	modeNormal
	modeHard
	modeHardcore
)

var modeNames = []string{"Easy", "Normal", "Hard", "Hardcore"}

var modeDesc = []string{
	"Take less damage. Keep everything when you die.",
	"Drop your blocks and materials when you die. Keep tools and gear.",
	"Take more damage. Drop everything; respawn with a temporary kit.",
	"Like Hard, but dying deletes the world for good.",
}

// Damage taken is scaled by the mode
var modeDamage = []float64{0.5, 1, 1.5, 1.5}

type WorldInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Seed   string `json:"seed"`
	Mode   int    `json:"mode"`
	Played int64  `json:"played"` // unix seconds
	Deaths int    `json:"deaths"`
}

var (
	world     WorldInfo // the world being played; ID is empty for the demo behind the menu
	worldList []WorldInfo
	bossDown  bool // the Warden has been beaten in this world
)

func worldsDir() string {
	d := dataPath("worlds")
	os.MkdirAll(d, 0o755)
	return d
}

func worldFile(id string) string { return filepath.Join(worldsDir(), id+".json") }

func loadWorldList() {
	worldList = nil
	if b, err := os.ReadFile(dataPath("worlds.json")); err == nil {
		json.Unmarshal(b, &worldList)
	}
	// drop entries whose file is gone
	out := worldList[:0]
	for _, w := range worldList {
		if _, err := os.Stat(worldFile(w.ID)); err == nil {
			out = append(out, w)
		}
	}
	worldList = out
	sort.SliceStable(worldList, func(a, b int) bool { return worldList[a].Played > worldList[b].Played })
	migrateOldSave()
}

func saveWorldList() {
	b, _ := json.MarshalIndent(worldList, "", "  ")
	writeFileAtomic(dataPath("worlds.json"), b)
}

// Keep the world list entry for the current world up to date
func touchWorld() {
	if world.ID == "" {
		return
	}
	world.Played = time.Now().Unix()
	for i := range worldList {
		if worldList[i].ID == world.ID {
			worldList[i] = world
			saveWorldList()
			return
		}
	}
	worldList = append([]WorldInfo{world}, worldList...)
	saveWorldList()
}

// The single save.json from older versions becomes a Normal world
func migrateOldSave() {
	old := dataPath("save.json")
	b, err := os.ReadFile(old)
	if err != nil {
		return
	}
	var st SaveState
	if json.Unmarshal(b, &st) != nil || st.Seed == "" {
		os.Rename(old, old+".old")
		return
	}
	info := WorldInfo{ID: newWorldID(), Name: "My world", Seed: st.Seed, Mode: modeNormal, Played: time.Now().Unix()}
	st.World = info
	if nb, err := json.Marshal(st); err == nil {
		writeFileAtomic(worldFile(info.ID), nb)
		worldList = append([]WorldInfo{info}, worldList...)
		saveWorldList()
	}
	os.Rename(old, old+".old")
	os.Remove(dataPath("meta.json"))
}

// "played 5 minutes ago"
func ago(unix int64) string {
	d := time.Since(time.Unix(unix, 0))
	switch {
	case unix == 0:
		return "new"
	case d < time.Minute:
		return "played just now"
	case d < time.Hour:
		return fmt.Sprintf("played %d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("played %d h ago", int(d.Hours()))
	}
	return fmt.Sprintf("played %d days ago", int(d.Hours()/24))
}

func newWorldID() string {
	return fmt.Sprintf("w%d%03d", time.Now().Unix(), rand.Intn(1000))
}

var nameA = []string{"Amber", "Hollow", "Quiet", "Violet", "Mossy", "Silver", "Drifting", "Sunken", "Bright", "Misty", "Copper", "Starlit"}
var nameB = []string{"Vale", "Hills", "Reach", "Deep", "Isles", "Fields", "Hollow", "Peaks", "Shore", "Wilds", "Glade", "Caverns"}

func randomWorldName() string {
	return nameA[rand.Intn(len(nameA))] + " " + nameB[rand.Intn(len(nameB))]
}

// Make a new world and start playing it
func createNewWorld(name, seed string, mode int) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = randomWorldName()
	}
	saveGame() // keep the world we're leaving
	createWorld(seed)
	world = WorldInfo{ID: newWorldID(), Name: name, Seed: seedText, Mode: mode}
	bossDown = false
	guideStepAt = 0
	player.maxHP = 10
	player.hp = player.maxHP
	for i := range inv {
		inv[i] = nil
	}
	starterKit(mode >= modeHard)
	objs = objs[:0]
	started = false
	play()
	saveGame()
	toast("Welcome to " + world.Name)
}

// Load a world from the list and start playing it
func openWorld(info WorldInfo) bool {
	if world.ID != "" && world.ID != info.ID {
		saveGame() // keep the world we're leaving
	}
	b, err := os.ReadFile(worldFile(info.ID))
	if err != nil {
		toast("That world could not be opened")
		return false
	}
	var st SaveState
	if json.Unmarshal(b, &st) != nil || !restoreState(&st) {
		toast("That world could not be opened")
		return false
	}
	world = info
	world.Mode = st.World.Mode
	if st.World.Name != "" {
		world.Name = st.World.Name
	}
	play()
	touchWorld()
	return true
}

func deleteWorld(id string) {
	os.Remove(worldFile(id))
	out := worldList[:0]
	for _, w := range worldList {
		if w.ID != id {
			out = append(out, w)
		}
	}
	worldList = out
	saveWorldList()
	if world.ID == id {
		world = WorldInfo{}
		started = false
	}
}

// ---- Dying ----

var (
	deathLines []string
	deathLost  bool // hardcore: the world is gone
	deathMode  int
)

// Things you keep on Normal: tools, gear, torches, food and other usable items
func isUtility(id string) bool {
	it := items[id]
	return it != nil && (it.tool || it.once || it.station != "")
}

func die() {
	world.Deaths++
	player.hp = 0
	deathMode = world.Mode
	ex, ey := player.x, player.y
	dropAt := func(s *Stack) { // scatter a little so the pile is easy to spot
		dropItem(s.ID, s.N, ex+player.w/2, ey, 0)
		e := ents[len(ents)-1]
		e.vx, e.vy = (rand.Float64()-0.5)*4, -2-rand.Float64()*2
	}
	lost := 0
	for i, s := range inv {
		if s == nil {
			continue
		}
		if s.Temp { // the temporary kit is gone either way
			inv[i] = nil
			continue
		}
		switch world.Mode {
		case modeEasy:
		case modeNormal:
			if !isUtility(s.ID) {
				dropAt(s)
				inv[i] = nil
				lost += s.N
			}
		default:
			dropAt(s)
			inv[i] = nil
			lost += s.N
		}
	}
	selected = -1
	burst(40, (ex+player.w/2)/CELL, (ey+player.h/2)/CELL, [3]int{240, 86, 112}, 2, 2, 40)
	sfx("die")
	deathLost = world.Mode == modeHardcore
	switch {
	case deathLost:
		deathLines = []string{"Hardcore: " + world.Name + " is gone for good."}
		deleteWorld(world.ID)
	case world.Mode == modeEasy:
		deathLines = []string{"You keep all your items on Easy."}
	case lost > 0:
		deathLines = []string{fmt.Sprintf("You dropped %d items where you fell.", lost), "Go back and pick them up."}
	default:
		deathLines = []string{"You had nothing to drop."}
	}
	if !deathLost {
		saveGame()
	}
	cur = scDead
	clearPointers()
}

func respawn() {
	player.hp = player.maxHP
	player.vx, player.vy, player.energy, player.invuln = 0, 0, 1, 90
	respawnPoint()
	if world.Mode >= modeHard {
		starterKit(true)
	}
	ents = keepItems(ents)
	play()
	saveGame()
}

// Drop creatures (not items) when the mote respawns, so it doesn't land among them
func keepItems(es []*Entity) []*Entity {
	out := es[:0]
	for _, e := range es {
		if e.kind == entItem {
			out = append(out, e)
		}
	}
	return out
}

// Sleeping in a bed makes it your respawn point
func useBed() {
	for _, o := range objs {
		if o.kind == "bed" && objInReach(o) {
			w, _ := o.size()
			spawnWX, spawnY = o.wx+w/2, o.y
			toast("Spawn point set")
			sfx("gem")
			burst(12, float64(o.lx()+w/2), float64(o.y), tipRGB, 1, 1, 30)
			return
		}
	}
}

// Respawn at the bed if it's still there, otherwise on the surface at the spawn column
func respawnPoint() {
	if spawnY > 0 {
		for _, o := range objs {
			w, _ := o.size()
			if o.kind == "bed" && o.wx+w/2 == spawnWX && o.y == spawnY {
				shiftWindow(floorDiv(spawnWX, CW) - NCH/2 - baseChunk)
				player.x = float64((spawnWX-baseChunk*CW)*CELL) - player.w/2
				player.y = float64((spawnY+5)*CELL) - player.h
				unstick()
				updatePlayerCells()
				placeCamera()
				return
			}
		}
		spawnY = 0
		toast("Your bed is gone, so you woke up outside")
	}
	teleportTo(spawnWX)
}

// Move the mote to dry ground at world column wx, sliding the loaded window there
func teleportTo(wx int) {
	shiftWindow(floorDiv(wx, CW) - NCH/2 - baseChunk)
	lx := wx - baseChunk*CW
	for d := 0; d < W/2-40; d++ {
		if safeSpawn(lx + d) {
			lx += d
			break
		}
		if safeSpawn(lx - d) {
			lx -= d
			break
		}
	}
	player.x = float64(lx * CELL)
	player.y = float64((groundTop(lx) - 15) * CELL)
	unstick()
	updatePlayerCells()
	placeCamera()
}

// The starting kit. On Hard and Hardcore it is temporary: it vanishes as soon as you
// pick up any other tool or utility item (torches don't count), and when you die.
func starterKit(temp bool) {
	for _, k := range []struct {
		id string
		n  int
	}{{"pickaxe", 1}, {"hoe", 1}, {"torch", 10}, {"berries", 4}} {
		if !temp && countItem(k.id) > 0 && items[k.id].tool {
			continue
		}
		for i, s := range inv {
			if s == nil {
				inv[i] = &Stack{ID: k.id, N: k.n, Temp: temp}
				break
			}
		}
	}
	if !temp {
		addItem("bp_hut", 1)
	}
	selected = -1
}

func hasTemp() bool {
	for _, s := range inv {
		if s != nil && s.Temp {
			return true
		}
	}
	return false
}

// Picking up a real tool or utility item ends the temporary kit
// (the temporary pickaxe stays until you have a real one, so you can always dig)
func clearTemp() {
	n := 0
	realPick := false
	for _, s := range inv {
		if s != nil && !s.Temp && isPick(s.ID) {
			realPick = true
		}
	}
	for i, s := range inv {
		if s != nil && s.Temp && (realPick || !isPick(s.ID)) {
			inv[i] = nil
			n++
		}
	}
	if n > 0 {
		toast("Your temporary kit fades away")
	}
}

// ---- Screens ----

func deathMenu() {
	r := sheet(560, 220)
	title := "YOU FAINTED"
	if deathLost {
		title = "YOUR WORLD IS LOST"
	}
	Title(title, r.X+18, r.Y+28)
	y := r.Y + 70
	for _, l := range deathLines {
		Text(l, r.X+18, y, 15, colInk, 0)
		y += 26
	}
	Text(fmt.Sprintf("Mode: %s", modeNames[deathMode]), r.X+18, y+6, 13, colMuted, 0)
	bw := (r.W - 36 - 14) / 2
	by := r.Y + r.H - 58
	if deathLost {
		if Button(Rect{r.X + 18, by, r.W - 36, 40}, "Main menu", styleGold, true, icoHome) {
			openScreen(scMain)
		}
		return
	}
	if Button(Rect{r.X + 18, by, bw, 40}, "Respawn", styleGold, true, icoPlay) {
		respawn()
	}
	if Button(Rect{r.X + 32 + bw, by, bw, 40}, "Main menu", styleSlate, true, icoHome) {
		respawn()
		saveGame()
		openScreen(scMain)
	}
}
