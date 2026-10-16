package game

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// Roguelite runs: when the mote faints the run is over. You earn Essence for how far you
// got, and spend it between runs on permanent upgrades. Reaching the Core wins the run.

type Meta struct {
	Essence   int            `json:"essence"`
	Upgrades  map[string]int `json:"upgrades"`
	Runs      int            `json:"runs"`
	Wins      int            `json:"wins"`
	BestDepth int            `json:"bestDepth"`
}

var meta = Meta{Upgrades: map[string]int{}}

type Upgrade struct {
	id, name, desc string
	max            int
	baseCost       int
}

// Permanent upgrades bought between runs. Add new ones here.
var upgrades = []Upgrade{
	{"vitality", "Vitality", "+1 heart at the start of every run", 5, 15},
	{"digging", "Digging", "Pickaxes dig a wider area", 3, 20},
	{"levitation", "Levitation", "+1.5 seconds of levitation", 4, 15},
	{"satchel", "Satchel", "Start runs with more gear", 3, 25},
	{"luck", "Luck", "Chests hold an extra item", 3, 30},
}

func upgradeCost(u Upgrade) int {
	lvl := meta.Upgrades[u.id]
	return u.baseCost * (lvl + 1) * (lvl + 2) / 2
}

func loadMeta() {
	b, err := os.ReadFile(dataPath("meta.json"))
	if err == nil {
		json.Unmarshal(b, &meta)
	}
	if meta.Upgrades == nil {
		meta.Upgrades = map[string]int{}
	}
}

func saveMeta() {
	b, _ := json.MarshalIndent(meta, "", "  ")
	writeFileAtomic(dataPath("meta.json"), b)
}

func buyUpgrade(u Upgrade) bool {
	if meta.Upgrades[u.id] >= u.max || meta.Essence < upgradeCost(u) {
		return false
	}
	meta.Essence -= upgradeCost(u)
	meta.Upgrades[u.id]++
	saveMeta()
	sfx("levelup")
	return true
}

// ---- The current run ----

type RunStats struct {
	Chests   int  `json:"chests"`
	Kills    int  `json:"kills"`
	MaxDepth int  `json:"maxDepth"`
	Frames   int  `json:"frames"`
	BossDown bool `json:"bossDown"`
}

var (
	run       RunStats
	lastRun   RunStats // shown on the run-over screen
	lastWon   bool
	lastGain  int
	lastLines []string
)

func baseMaxHP() int { return 10 + 2*meta.Upgrades["vitality"] }

func levSecs() float64 { return levSeconds + 1.5*float64(meta.Upgrades["levitation"]) }

// Start a fresh run in a new world
func startRun(seed string) {
	run = RunStats{}
	createWorld(seed)
	player.maxHP = baseMaxHP()
	player.hp = player.maxHP
	starterKit()
	started = false
	play()
	saveGame()
	toast("Something hums far below...")
}

// Track how deep the mote has been
func trackRun() {
	run.Frames++
	wx := baseChunk*CW + int((player.x+player.w/2)/CELL)
	d := int((player.y+player.h)/CELL) - surfaceAt(wx)
	if d > run.MaxDepth {
		run.MaxDepth = d
	}
}

func essenceFor(r RunStats, won bool) (int, []string) {
	days := float64(r.Frames) / dayFrames
	lines := []string{
		fmt.Sprintf("Deepest point: %d", r.MaxDepth),
		fmt.Sprintf("Chests opened: %d", r.Chests),
		fmt.Sprintf("Creatures beaten: %d", r.Kills),
		fmt.Sprintf("Days survived: %.1f", days),
	}
	e := r.MaxDepth/8 + r.Chests*6 + r.Kills*2 + int(days*8)
	if r.BossDown {
		e += 40
		lines = append(lines, "Warden defeated: +40")
	}
	if won {
		e += 120
		lines = append(lines, "Reached the Core: +120")
	}
	return e, lines
}

func endRun(won bool) {
	lastRun, lastWon = run, won
	lastGain, lastLines = essenceFor(run, won)
	meta.Essence += lastGain
	meta.Runs++
	if won {
		meta.Wins++
	}
	if run.MaxDepth > meta.BestDepth {
		meta.BestDepth = run.MaxDepth
	}
	saveMeta()
	os.Remove(dataPath("save.json")) // the run is over for good
	started = false
	cur = scRunOver
	clearPointers()
	if won {
		sfx("levelup")
	} else {
		sfx("die")
	}
}

func winRun() {
	burst(80, (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL, [3]int{255, 120, 220}, 3, 3, 60)
	shake = 14
	endRun(true)
}

// ---- Screens ----

func runOverMenu() {
	r := sheet(640, 380)
	title := "RUN OVER"
	if lastWon {
		title = "YOU REACHED THE CORE!"
	}
	Title(title, r.X+18, r.Y+28)
	y := r.Y + 66
	for _, l := range lastLines {
		Text(l, r.X+18, y, 14, colInk, 0)
		y += 24
	}
	y += 8
	Label(fmt.Sprintf("ESSENCE EARNED: %d", lastGain), r.X+18, y)
	Text(fmt.Sprintf("You now have %d Essence", meta.Essence), r.X+18, y+22, 13, colMuted, 0)
	bw := (r.W - 36 - 28) / 3
	by := r.Y + r.H - 56
	if Button(Rect{r.X + 18, by, bw, 38}, "Upgrades", styleGold, true, icoGear) {
		upgradesFrom = scRunOver
		openScreen(scUpgrades)
	}
	if Button(Rect{r.X + 32 + bw, by, bw, 38}, "New run", styleViolet, true, icoPlay) {
		startRun("")
	}
	if Button(Rect{r.X + 46 + 2*bw, by, bw, 38}, "Main menu", styleSlate, true, icoHome) {
		openScreen(scMain)
	}
}

var upgradesFrom = scMain

func upgradesMenu() {
	r := sheet(720, 420)
	back, cl := head(r, "UPGRADES", true)
	if back || cl {
		openScreen(upgradesFrom)
		return
	}
	Label(fmt.Sprintf("ESSENCE: %d", meta.Essence), r.X+r.W-230, r.Y+28)
	y := r.Y + 60
	rh := math.Min(58, (r.H-80)/float64(len(upgrades))-6)
	for _, u := range upgrades {
		row := Rect{r.X + 18, y, r.W - 36, rh}
		Inset(row, colSlateLo)
		lvl := meta.Upgrades[u.id]
		Text(u.name, row.X+12, row.Y+rh*0.32, 15, colInk, 0)
		Text(u.desc, row.X+12, row.Y+rh*0.7, 12, colMuted, 0)
		// level pips
		for k := 0; k < u.max; k++ {
			c := colPanelHi
			if k < lvl {
				c = colT
			}
			px, py := row.X+row.W-250+float64(k)*16, row.Y+rh/2-5
			cc := c
			ops = append(ops, func(dst *ebiten.Image) { fill(dst, px, py, 10, 10, cc) })
		}
		label := fmt.Sprintf("%d", upgradeCost(u))
		enabled := lvl < u.max && meta.Essence >= upgradeCost(u)
		if lvl >= u.max {
			label = "Max"
		}
		if Button(Rect{row.X + row.W - 96, row.Y + (rh-30)/2, 84, 30}, label, styleViolet, enabled, nil) {
			buyUpgrade(u)
		}
		y += rh + 6
	}
}
