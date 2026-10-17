package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- Items, inventory, hotbar, crafting, using items ----
// Icons are 8x8 pixel maps drawn with the same palette as the world and the mote.

var iconCol = map[byte][3]int{
	'O': {34, 28, 64}, 'S': {200, 206, 218}, 's': {130, 136, 150}, 'H': {150, 100, 60}, 'h': {110, 70, 40},
	'M': {170, 176, 190}, 'm': {110, 116, 130}, 'W': {59, 142, 232}, 'w': {140, 196, 250},
	'Y': {226, 194, 107}, 'y': {190, 158, 80}, 'G': {120, 126, 140}, 'g': {90, 96, 110},
	'B': {124, 98, 255}, 'L': {196, 180, 255}, 'D': {90, 67, 214}, 'T': {255, 222, 120},
	'K': {60, 58, 70}, 'k': {100, 98, 112}, 'R': {196, 108, 84}, 'r': {140, 70, 56},
	'c': {210, 126, 70}, 'C': {150, 84, 46}, 'i': {214, 190, 172}, 'I': {150, 134, 124},
	'u': {246, 204, 70}, 'U': {190, 150, 40}, 'x': {120, 232, 255}, 'X': {210, 250, 255}, 'z': {60, 150, 200},
	'f': {255, 150, 60}, 'e': {88, 168, 74}, 'E': {50, 112, 52}, 'd': {122, 86, 60}, 'p': {178, 132, 84},
	'P': {130, 92, 56}, 'q': {236, 120, 170}, 'Q': {180, 60, 90}, 'v': {130, 50, 140}, 'a': {240, 86, 112},
	'A': {255, 170, 180}, 'n': {200, 160, 100}, 'b': {44, 30, 64}, 'j': {80, 60, 140},
}

type Item struct {
	id, name, desc string
	tool, once     bool
	place          uint8
	icon           []string
	img            *ebiten.Image
	reach          int // tool radius in cells
	dmg            int // damage to creatures
	heal           int // food: half-hearts restored
	armor          int // damage blocked while carried
	blueprint      string
	station        string // a workstation you place in the world
	wall           uint8  // a background wall of this block type
}

// A rough lump in two shades
func lumpIcon(a, d byte) []string {
	rows := []string{"........", "..OOOO..", ".OaAaaO.", "OaaAaAaO", "OAaaaAaO", ".OaAaaO.", "..OOOO..", "........"}
	for i, r := range rows {
		rows[i] = strings.NewReplacer("a", string(a), "A", string(d)).Replace(r)
	}
	return rows
}

// A metal bar in two shades
func barIcon(a, d byte) []string {
	rows := []string{"........", "........", "...OOOO.", "..OaaaAO", ".OaaaAAO", "OAAAAAO.", "OOOOOO..", "........"}
	for i, r := range rows {
		rows[i] = strings.NewReplacer("a", string(a), "A", string(d)).Replace(r)
	}
	return rows
}

// A block in two shades
func blockIcon(a, d byte) []string {
	rows := []string{"OOOOOOOO", "OaaAaaaO", "OaAaaaAO", "OaaaAaaO", "OAaaaaaO", "OaaAaAaO", "OaaaaaAO", "OOOOOOOO"}
	for i, r := range rows {
		rows[i] = strings.NewReplacer("a", string(a), "A", string(d)).Replace(r)
	}
	return rows
}

// A rolled-up blueprint
var bpIcon = []string{"........", ".OOOOOO.", "OwWWWWwO", "OWwWWwWO", "OWWwwWWO", "OwWWWWwO", ".OOOOOO.", "........"}

var itemOrder = []string{"pickaxe", "ironpick", "corepick", "hoe", "hammer", "paintbrush", "bucket", "coppersword", "ironsword",
	"copperarmor", "ironarmor", "lantern", "torch", "workbench", "furnace", "anvil", "bed", "storage",
	"bp_hut", "bp_tower", "bp_bridge", "bp_farm", "bp_bunker",
	"dirt", "sand", "stone", "brick", "plank", "wood", "glass", "leaves", "obsidian", "water",
	"wall_dirt", "wall_stone", "wall_plank", "wall_brick", "wall_glass", "wall_obsidian",
	"coal", "copper", "iron", "gold", "crystal", "copperbar", "ironbar", "goldbar", "coreshard",
	"seeds", "wheat", "berries", "bread", "heart", "gem", "bomb"}

var items = map[string]*Item{
	"pickaxe": {name: "Pickaxe", tool: true, desc: "Digs blocks and opens chests", reach: 4, dmg: 1, icon: []string{
		".SSSSSs.", "S..hH..s", "...hH...", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"ironpick": {name: "Iron pickaxe", tool: true, desc: "Digs a much bigger area", reach: 6, dmg: 2, icon: []string{
		".iiiiiI.", "i..hH..I", "i..hH..I", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"hoe": {name: "Hoe", tool: true, desc: "Turns dirt into soil for seeds, and harvests crops", reach: 2, icon: []string{
		"..SSSs..", ".S..hH..", "....hH..", "....hH..", "....hH..", "....hH..", "....hH..", "....hh.."}},
	"bucket": {name: "Bucket", tool: true, desc: "Scoops up water", reach: 5, icon: []string{
		"..O..O..", ".O....O.", "OMMMMMMO", "OMwWWWmO", ".MWWWWm.", ".MMMMmm.", "..Mmmm..", "........"}},
	"coppersword": {name: "Copper sword", tool: true, desc: "Hits creatures", reach: 7, dmg: 2, icon: []string{
		"......Oc", ".....Occ", "....Occ.", "...Occ..", "h.Occ...", ".hcc....", ".hh.....", "h..h...."}},
	"ironsword": {name: "Iron sword", tool: true, desc: "Hits creatures hard", reach: 8, dmg: 4, icon: []string{
		"......Oi", ".....Oii", "....OiI.", "...OiI..", "T.OiI...", ".TiI....", ".hT.....", "h..T...."}},
	"copperarmor": {name: "Copper armor", tool: true, armor: 1, desc: "Blocks some damage while you carry it", icon: []string{
		"........", ".OO..OO.", "OccOOccO", "OcccccCO", ".OcccCO.", ".OccCCO.", ".OcCCCO.", "..OOOO.."}},
	"ironarmor": {name: "Iron armor", tool: true, armor: 2, desc: "Blocks a lot of damage while you carry it", icon: []string{
		"........", ".OO..OO.", "OiiOOiiO", "OiiiiiIO", ".OiiiIO.", ".OiiIIO.", ".OiIIIO.", "..OOOO.."}},
	"lantern": {name: "Lantern", tool: true, desc: "Lights up more around you while you carry it", icon: []string{
		"...OO...", "..O..O..", ".OOOOOO.", ".OxXxxO.", ".OXxxxO.", ".OxxxzO.", ".OOOOOO.", "........"}},
	"torch": {name: "Torch", once: true, place: TORCH, desc: "Place a light that never goes out", icon: []string{
		"...f....", "..fTf...", "..TTT...", "...h....", "...h....", "...h....", "...h....", "........"}},
	"furnace": {name: "Furnace", once: true, station: "furnace", desc: "A station: smelt ores into bars, bake bread, make glass", icon: []string{
		"OOOOOOOO", "OGGgGGGO", "OGOOOOgO", "OgOfTOGO", "OGOTfOGO", "OGOOOOgO", "OGGGgGGO", "OOOOOOOO"}},
	"workbench": {name: "Workbench", once: true, station: "workbench", desc: "A station: tools, building blocks, walls and blueprints", icon: []string{
		"........", "........", "OOOOOOOO", "OppppppO", "OPPPPPPO", "OhOOOOhO", "OhO..OhO", "OOO..OOO"}},
	"anvil": {name: "Anvil", once: true, station: "anvil", desc: "A station: metal tools, weapons and armor", icon: []string{
		"........", "........", "OOOOOOO.", "OiiiiiIO", ".OIIIIO.", "..OmmO..", ".OmmmmO.", "OOOOOOOO"}},
	"storage": {name: "Storage chest", once: true, station: "storage", desc: "Place it, then tap it to store items", icon: []string{
		"........", ".OOOOOO.", "OPPPPPPO", "OppppppO", "OOOTTOOO", "OPPTTPPO", "OppppppO", "OOOOOOOO"}},
	"bed": {name: "Bed", once: true, station: "bed", desc: "Place it, then tap it to make it your respawn point", icon: []string{
		"........", "........", "O.......", "OhOOOOOO", "OhwwRRRO", "OhOOOOhO", "OhO..OhO", "OOO..OOO"}},
	"hammer": {name: "Hammer", tool: true, desc: "Knocks down background walls", reach: 3, icon: []string{
		".OOOOO..", "OSSSSsO.", "OSSSSsO.", ".OOhOO..", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"paintbrush": {name: "Paintbrush", tool: true, desc: "Paints blocks and walls. Pick a colour on the build bar (C)", reach: 3, icon: []string{
		"......qa", ".....qaa", "....OqO.", "...OhO..", "..OhO...", ".OhO....", "OhO.....", "OO......"}},
	"coreshard": {name: "Core shard", desc: "A piece of the Core. Forge it into the Core pickaxe", icon: []string{
		"...OO...", "..OqAO..", ".OqAAqO.", "OqAAAAqO", ".OqAAqO.", "..OqqO..", "...OO...", "........"}},
	"corepick": {name: "Core pickaxe", tool: true, desc: "Digs a huge area and hits hard", reach: 9, dmg: 5, icon: []string{
		".qqqqqA.", "q..hH..A", "q..hH..A", "...hH...", "...hH...", "...hH...", "...hH...", "...hh..."}},
	"sand":     {name: "Sand", place: SAND, desc: "Falls when nothing holds it up", icon: blockIcon('Y', 'y')},
	"dirt":     {name: "Dirt", place: DIRT, desc: "Ground. Till it with a hoe for crops", icon: blockIcon('d', 'P')},
	"stone":    {name: "Stone", place: STONE, desc: "A solid building block", icon: blockIcon('G', 'g')},
	"plank":    {name: "Plank", place: PLANK, desc: "A wooden building block", icon: blockIcon('p', 'P')},
	"obsidian": {name: "Obsidian", place: OBSIDIAN, desc: "Very dark, very solid", icon: blockIcon('b', 'j')},
	"brick": {name: "Brick", place: BRICK, desc: "A sturdy building block", icon: []string{
		"OOOOOOOO", "ORRrORRO", "OrrrOrrO", "OOOOOOOO", "ORORRrRO", "OrOrrrrO", "OOOOOOOO", "........"}},
	"glass": {name: "Glass", place: GLASS, desc: "A window block that lets light through", icon: []string{
		"OOOOOOOO", "OwXwwwwO", "OXwwwwwO", "OwwwwwwO", "OwwwwwXO", "OwwwwXwO", "OwwwwwwO", "OOOOOOOO"}},
	"wood": {name: "Wood", place: WOOD, desc: "From trees. Make planks and tools", icon: []string{
		"........", ".OOOOOO.", "OhHHHHhO", "OHnnnnHO", "OHnhhnHO", "OHnnnnHO", "OhHHHHhO", ".OOOOOO."}},
	"leaves": {name: "Leaves", place: LEAVES, desc: "Bushy. Sometimes hides seeds", icon: lumpIcon('e', 'E')},
	"water": {name: "Water", place: WATER, desc: "Pour water", icon: []string{
		"...OO...", "..OwWO..", "..OwWO..", ".OwWWWO.", "OwWWWWWO", "OWWWWWWO", ".OWWWWO.", "..OOOO.."}},
	"coal":      {name: "Coal", place: COAL, desc: "Fuel for torches and the furnace", icon: lumpIcon('K', 'k')},
	"copper":    {name: "Copper ore", place: COPPER, desc: "Smelt it into copper bars", icon: lumpIcon('c', 'C')},
	"iron":      {name: "Iron ore", place: IRON, desc: "Smelt it into iron bars", icon: lumpIcon('i', 'I')},
	"gold":      {name: "Gold ore", place: GOLD, desc: "Smelt it into gold bars", icon: lumpIcon('u', 'U')},
	"crystal":   {name: "Crystal", place: CRYSTAL, desc: "Glows softly. Grows in caves", icon: lumpIcon('x', 'z')},
	"copperbar": {name: "Copper bar", desc: "For copper tools and armor", icon: barIcon('c', 'C')},
	"ironbar":   {name: "Iron bar", desc: "For iron tools and armor", icon: barIcon('i', 'I')},
	"goldbar":   {name: "Gold bar", desc: "For lanterns and gems", icon: barIcon('u', 'U')},
	"seeds": {name: "Seeds", desc: "Plant them on soil (till dirt with a hoe)", icon: []string{
		"........", "...O....", "..OnO.O.", "...O.OnO", ".O...OO.", "OnO.....", ".O..O...", "...OnO.."}},
	"wheat": {name: "Wheat", desc: "Bake it into bread at a furnace", icon: []string{
		"..T.T...", ".TnTnT..", "..TnT...", "...e....", "...e.T..", "...eTnT.", "...e.T..", "...e...."}},
	"berries": {name: "Berries", once: true, heal: 2, desc: "Eat to heal a little", icon: []string{
		"....e...", "...eE...", "..OO.OO.", ".OaaOaaO", ".OaAOaAO", "..OOaaO.", "...OaAO.", "....OO.."}},
	"bread": {name: "Bread", once: true, heal: 6, desc: "Eat to heal three hearts", icon: []string{
		"........", "..OOOO..", ".OnnTnO.", "OnTnnTnO", "OnnnnnnO", "OHHHHHHO", ".OOOOOO.", "........"}},
	"heart": {name: "Heart crystal", once: true, desc: "Use it to gain an extra heart", icon: []string{
		"........", ".OO.OO..", "OAaOaaO.", "OaaaaaO.", ".OaaaO..", "..OaO...", "...O....", "........"}},
	"gem": {name: "Energy gem", once: true, desc: "Refills levitation", icon: []string{
		"...OO...", "..OLLO..", ".OLBBBO.", "OBLBBBDO", ".OBBBDO.", "..OBDO..", "...OO...", "........"}},
	"bomb": {name: "Bomb", once: true, desc: "Blasts a hole", icon: []string{
		".....T..", "....T...", "..OOOh..", ".OKKKKO.", "OKkKKKKO", "OKKKKKKO", ".OKKKKO.", "..OOOO.."}},
	"bp_hut":    {name: "Hut blueprint", once: true, blueprint: "hut", desc: "Builds a small house where you aim", icon: bpIcon},
	"bp_tower":  {name: "Tower blueprint", once: true, blueprint: "tower", desc: "Builds a tall watchtower", icon: bpIcon},
	"bp_bridge": {name: "Bridge blueprint", once: true, blueprint: "bridge", desc: "Builds a long plank bridge", icon: bpIcon},
	"bp_farm":   {name: "Farm blueprint", once: true, blueprint: "farm", desc: "Builds a watered field ready for seeds", icon: bpIcon},
	"bp_bunker": {name: "Bunker blueprint", once: true, blueprint: "bunker", desc: "Builds a safe stone shelter", icon: bpIcon},
}

// Background walls: one item per wall type, made from its block at a workbench
var wallTypes = []struct {
	id    string
	block uint8
	name  string
}{{"wall_dirt", DIRT, "Dirt wall"}, {"wall_stone", STONE, "Stone wall"}, {"wall_plank", PLANK, "Plank wall"},
	{"wall_brick", BRICK, "Brick wall"}, {"wall_glass", GLASS, "Glass wall"}, {"wall_obsidian", OBSIDIAN, "Obsidian wall"}}

var wallItem = map[uint8]string{}

func initItems() {
	for _, w := range wallTypes {
		src := items[blockItem[w.block]]
		icon := make([]string, 8)
		for y, r := range src.icon {
			b := []byte(r)
			for x := range b {
				if (x+y)%2 == 0 && b[x] != '.' && b[x] != 'O' {
					b[x] = 'O' // darker, dotted: it sits behind things
				}
			}
			icon[y] = string(b)
		}
		items[w.id] = &Item{name: w.name, wall: w.block, desc: "A background wall. Walk past it; creatures won't spawn in front of walls", icon: icon}
		wallItem[w.block] = w.id
	}
	for id, it := range items {
		it.id = id
		img := ebiten.NewImage(8, 8)
		for y, r := range it.icon {
			for x := 0; x < 8 && x < len(r); x++ {
				if c, ok := iconCol[r[x]]; ok {
					img.Set(x, y, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
				}
			}
		}
		it.img = img
	}
	initBlueprints()
}

const (
	stackMax = 999
	hotbarN  = 8
	invSize  = hotbarN + 32 // hotbar plus a 4x8 backpack
)

type Stack struct {
	ID   string `json:"id"`
	N    int    `json:"n"`
	Temp bool   `json:"temp,omitempty"` // part of the temporary kit (Hard modes)
}

var (
	inv      = make([]*Stack, invSize)
	selected = -1
)

func addItem(id string, n int) int {
	if n <= 0 || items[id] == nil {
		return 0
	}
	if id != "torch" && isUtility(id) && hasTemp() {
		clearTemp()
	}
	left := n
	if !items[id].tool {
		for _, s := range inv {
			if s != nil && s.ID == id && !s.Temp && s.N < stackMax {
				k := min(left, stackMax-s.N)
				s.N += k
				left -= k
				if left == 0 {
					break
				}
			}
		}
	}
	for left > 0 {
		i := -1
		for j, s := range inv {
			if s == nil {
				i = j
				break
			}
		}
		if i < 0 {
			break
		}
		k := min(left, stackMax)
		if items[id].tool {
			k = 1
		}
		inv[i] = &Stack{ID: id, N: k}
		left -= k
	}
	return n - left
}

// Add items, dropping whatever doesn't fit at the mote's feet
func giveItem(id string, n int) {
	if got := addItem(id, n); got < n {
		dropItem(id, n-got, player.x, player.y, 0)
	}
}

func takeSelected(n int) bool {
	if selected < 0 || inv[selected] == nil {
		return false
	}
	inv[selected].N -= n
	if inv[selected].N <= 0 {
		inv[selected] = nil
	}
	return true
}

func countItem(id string) int {
	t := 0
	for _, s := range inv {
		if s != nil && s.ID == id {
			t += s.N
		}
	}
	return t
}

func removeItem(id string, n int) {
	for i := len(inv) - 1; i >= 0 && n > 0; i-- {
		s := inv[i]
		if s == nil || s.ID != id {
			continue
		}
		k := min(n, s.N)
		s.N -= k
		n -= k
		if s.N == 0 {
			inv[i] = nil
		}
	}
}

// Best armor carried
func armorLevel() int {
	best := 0
	for _, s := range inv {
		if s != nil && items[s.ID].armor > best {
			best = items[s.ID].armor
		}
	}
	return best
}

// Drop n items from a slot into the world in front of the mote
func dropSlot(i, n int) {
	s := inv[i]
	if s == nil {
		return
	}
	if s.Temp {
		toast("Temporary items can't be dropped")
		return
	}
	n = min(n, s.N)
	throwItem(s.ID, n)
	s.N -= n
	if s.N <= 0 {
		inv[i] = nil
	}
}

// Move a slot between the hotbar and the backpack (merging stacks where possible)
func quickMove(i int) {
	s := inv[i]
	if s == nil {
		return
	}
	from, to := hotbarN, invSize
	if i >= hotbarN {
		from, to = 0, hotbarN
	}
	if !items[s.ID].tool {
		for j := from; j < to && s.N > 0; j++ {
			if t := inv[j]; t != nil && t.ID == s.ID && t.N < stackMax {
				k := min(s.N, stackMax-t.N)
				t.N += k
				s.N -= k
			}
		}
	}
	if s.N == 0 {
		inv[i] = nil
		return
	}
	for j := from; j < to; j++ {
		if inv[j] == nil {
			inv[j], inv[i] = s, nil
			return
		}
	}
}

// Split half of a stack into an empty slot
func splitStack(i int) {
	s := inv[i]
	if s == nil || s.N < 2 {
		return
	}
	for j := hotbarN; j < invSize; j++ {
		if inv[j] == nil {
			half := s.N / 2
			inv[j] = &Stack{ID: s.ID, N: half, Temp: s.Temp}
			s.N -= half
			return
		}
	}
}

// Merge stacks and sort the backpack in item order
func sortBackpack() {
	totals := map[string]int{}
	var tools []string
	for i := hotbarN; i < invSize; i++ {
		if s := inv[i]; s != nil {
			if items[s.ID].tool {
				tools = append(tools, s.ID)
			} else {
				totals[s.ID] += s.N
			}
			inv[i] = nil
		}
	}
	rank := map[string]int{}
	for k, id := range itemOrder {
		rank[id] = k
	}
	sort.Slice(tools, func(a, b int) bool { return rank[tools[a]] < rank[tools[b]] })
	j := hotbarN
	for _, id := range tools {
		inv[j] = &Stack{ID: id, N: 1}
		j++
	}
	for _, id := range itemOrder {
		for n := totals[id]; n > 0 && j < invSize; j++ {
			k := min(n, stackMax)
			inv[j] = &Stack{ID: id, N: k}
			n -= k
		}
	}
}

// ---- Crafting ----
// Basic recipes are made by hand (in the inventory). Everything else needs a station:
// open the workbench, furnace or anvil to use its recipes.

type Recipe struct {
	out   string
	n     int
	needs [][2]any // {id, count}
	at    string   // station id, "" = by hand
}

var recipes = []Recipe{
	// by hand
	{"plank", 4, [][2]any{{"wood", 1}}, ""},
	{"torch", 4, [][2]any{{"coal", 1}, {"wood", 1}}, ""},
	{"workbench", 1, [][2]any{{"plank", 12}}, ""},
	// workbench
	{"pickaxe", 1, [][2]any{{"wood", 6}, {"stone", 10}}, "workbench"},
	{"hoe", 1, [][2]any{{"wood", 4}, {"stone", 4}}, "workbench"},
	{"hammer", 1, [][2]any{{"wood", 4}, {"stone", 6}}, "workbench"},
	{"paintbrush", 1, [][2]any{{"wood", 2}, {"plank", 2}, {"berries", 1}}, "workbench"},
	{"furnace", 1, [][2]any{{"stone", 20}, {"coal", 4}}, "workbench"},
	{"anvil", 1, [][2]any{{"ironbar", 6}, {"stone", 10}}, "workbench"},
	{"bed", 1, [][2]any{{"plank", 12}, {"leaves", 10}}, "workbench"},
	{"storage", 1, [][2]any{{"plank", 20}}, "workbench"},
	{"torch", 8, [][2]any{{"coal", 1}, {"wood", 2}}, "workbench"},
	{"brick", 10, [][2]any{{"sand", 10}, {"water", 5}}, "workbench"},
	{"bomb", 1, [][2]any{{"sand", 10}, {"coal", 2}}, "workbench"},
	{"wall_dirt", 4, [][2]any{{"dirt", 1}}, "workbench"},
	{"wall_stone", 4, [][2]any{{"stone", 1}}, "workbench"},
	{"wall_plank", 4, [][2]any{{"plank", 1}}, "workbench"},
	{"wall_brick", 4, [][2]any{{"brick", 1}}, "workbench"},
	{"wall_glass", 4, [][2]any{{"glass", 1}}, "workbench"},
	{"wall_obsidian", 4, [][2]any{{"obsidian", 1}}, "workbench"},
	{"bp_hut", 1, [][2]any{{"plank", 40}, {"brick", 20}, {"glass", 4}}, "workbench"},
	{"bp_farm", 1, [][2]any{{"dirt", 30}, {"brick", 10}, {"water", 10}}, "workbench"},
	{"bp_bridge", 1, [][2]any{{"plank", 30}}, "workbench"},
	{"bp_tower", 1, [][2]any{{"brick", 60}, {"plank", 20}, {"glass", 6}}, "workbench"},
	{"bp_bunker", 1, [][2]any{{"stone", 90}, {"glass", 4}}, "workbench"},
	// furnace
	{"copperbar", 1, [][2]any{{"copper", 2}, {"coal", 1}}, "furnace"},
	{"ironbar", 1, [][2]any{{"iron", 2}, {"coal", 1}}, "furnace"},
	{"goldbar", 1, [][2]any{{"gold", 2}, {"coal", 1}}, "furnace"},
	{"glass", 4, [][2]any{{"sand", 4}, {"coal", 1}}, "furnace"},
	{"bread", 1, [][2]any{{"wheat", 3}}, "furnace"},
	// anvil
	{"bucket", 1, [][2]any{{"copperbar", 3}}, "anvil"},
	{"coppersword", 1, [][2]any{{"copperbar", 6}, {"wood", 2}}, "anvil"},
	{"copperarmor", 1, [][2]any{{"copperbar", 10}}, "anvil"},
	{"ironsword", 1, [][2]any{{"ironbar", 6}, {"wood", 2}}, "anvil"},
	{"ironpick", 1, [][2]any{{"ironbar", 8}, {"wood", 4}}, "anvil"},
	{"ironarmor", 1, [][2]any{{"ironbar", 12}}, "anvil"},
	{"lantern", 1, [][2]any{{"crystal", 4}, {"goldbar", 2}}, "anvil"},
	{"gem", 1, [][2]any{{"crystal", 2}, {"goldbar", 1}}, "anvil"},
	{"corepick", 1, [][2]any{{"coreshard", 8}, {"ironbar", 6}}, "anvil"},
}

// Crafting categories, shown as tabs
var craftCats = []string{"Tools", "Building", "Stations", "Materials", "Food & gear"}

func recipeCat(r Recipe) int {
	it := items[r.out]
	switch {
	case it.station != "":
		return 2
	case it.tool && it.armor == 0 && it.id != "lantern":
		return 0
	case it.id == "bomb":
		return 0
	case it.place != 0 || it.wall != 0 || it.blueprint != "" || it.id == "torch":
		return 1
	case it.heal > 0 || it.armor > 0 || it.id == "gem" || it.id == "lantern":
		return 4
	}
	return 3
}

func canCraft(r Recipe) bool {
	for _, nd := range r.needs {
		if countItem(nd[0].(string)) < nd[1].(int) {
			return false
		}
	}
	return true
}

// How many times the recipe can be made with what you carry
func craftableTimes(r Recipe) int {
	n := 9999
	for _, nd := range r.needs {
		n = min(n, countItem(nd[0].(string))/nd[1].(int))
	}
	return n
}

func craft(r Recipe) string {
	if !canCraft(r) {
		return ""
	}
	room := false
	for _, s := range inv {
		if s == nil || (s.ID == r.out && !s.Temp && !items[r.out].tool && s.N+r.n <= stackMax) {
			room = true
			break
		}
	}
	if !room {
		return "No room in your inventory."
	}
	for _, nd := range r.needs {
		removeItem(nd[0].(string), nd[1].(int))
	}
	addItem(r.out, r.n)
	sfx("give")
	if r.n > 1 {
		return fmt.Sprintf("Crafted %d %s.", r.n, items[r.out].name)
	}
	return "Crafted " + items[r.out].name + "."
}

// ---- Chests ----

type lootEntry struct {
	id       string
	min, max int
	chance   float64
}

var graveLoot = []lootEntry{{"seeds", 3, 8, 0.8}, {"torch", 4, 10, 0.7}, {"berries", 2, 5, 0.6}, {"bread", 1, 2, 0.4},
	{"copper", 4, 10, 0.5}, {"coal", 4, 10, 0.5}, {"heart", 1, 1, 0.12}, {"bomb", 1, 3, 0.3}, {"bp_farm", 1, 1, 0.1}}

var dungeonLoot = []lootEntry{{"ironbar", 3, 8, 0.6}, {"goldbar", 2, 5, 0.4}, {"heart", 1, 1, 0.4}, {"gem", 1, 2, 0.5},
	{"bomb", 2, 5, 0.5}, {"bread", 2, 4, 0.5}, {"copperarmor", 1, 1, 0.15}, {"ironsword", 1, 1, 0.12},
	{"crystal", 3, 8, 0.4}, {"bp_tower", 1, 1, 0.12}, {"bp_bunker", 1, 1, 0.15}}

// Open the chest touching cell (x, y): loot is decided by the seed and the chest's position
func openChest(x, y int) {
	// find the whole chest (3x2) and clear it
	minX, minY := x, y
	stack := [][2]int{{x, y}}
	seen := map[[2]int]bool{{x, y}: true}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		grid[c[1]*W+c[0]] = EMPTY
		minX, minY = min(minX, c[0]), min(minY, c[1])
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if n[0] >= 0 && n[0] < W && n[1] >= 0 && n[1] < H && !seen[n] && grid[n[1]*W+n[0]] == CHEST {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	wx := baseChunk*CW + minX
	table := graveLoot
	if minY > H*55/100 {
		table = dungeonLoot
	}
	rolls := 3
	got := 0
	for k := 0; k < len(table)*2 && got < rolls; k++ {
		e := table[k%len(table)]
		h := hash01(wx+k, minY, 121)
		if h > e.chance {
			continue
		}
		n := e.min + int(hash01(wx+k, minY, 122)*float64(e.max-e.min+1))
		dropItem(e.id, n, float64(minX*CELL), float64((minY-2)*CELL), 0)
		got++
	}
	burst(20, float64(minX+1), float64(minY), tipRGB, 1.5, 1.5, 30)
	sfx("chest")
	toast("Chest opened!")
}

// ---- Using items: aim is where the pointer is, reach is limited around the mote ----

const reach = 46

var (
	useActive bool
	aimSX     float64
	aimSY     float64
	aimX      int
	aimY      int
	useTick   int
)

func aimCell() bool {
	aimX = int(math.Floor((cam.x + aimSX/zoom) / CELL))
	aimY = int(math.Floor((cam.y + aimSY/zoom) / CELL))
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	return math.Hypot(float64(aimX)-pcx, float64(aimY)-pcy) <= reach
}

func eachInCircle(r int, fn func(x, y, i int)) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r {
				continue
			}
			x, y := aimX+dx, aimY+dy
			if x >= 0 && x < W && y >= 0 && y < H {
				fn(x, y, y*W+x)
			}
		}
	}
}

func digRadius(it *Item) int { return it.reach }

// Mined Core cells, turned into a shard every 10
var coreDust int

func useHeld() {
	if !useActive || selected < 0 || inv[selected] == nil {
		return
	}
	id := inv[selected].ID
	it := items[id]
	if it.once || !aimCell() {
		return
	}
	useTick++
	switch {
	case id == "pickaxe" || id == "ironpick" || id == "corepick":
		mineObjects(digRadius(it))
		got := map[string]int{}
		chest, core := [2]int{-1, -1}, 0
		eachInCircle(digRadius(it), func(x, y, i int) {
			t := grid[i]
			switch t {
			case CHEST:
				chest = [2]int{x, y}
				return
			case CORE:
				if bossDown {
					grid[i] = EMPTY
					core++
				} else {
					core = -1000
				}
				return
			}
			name := blockItem[t]
			if name == "" {
				return
			}
			got[name]++
			if t == CROP3 {
				got["seeds"] += 1 + rand.Intn(2)
			}
			if t == LEAVES && rand.Float64() < 0.08 {
				got["seeds"]++
			}
			grid[i] = EMPTY
			if rand.Float64() < 0.12 {
				burst(1, float64(x), float64(y), blockRGB[t], 0.8, 0.8, 18)
			}
		})
		if got["leaves"] > 0 { // leaves mostly crumble away
			got["leaves"] = (got["leaves"] + 3) / 4
		}
		for name, n := range got {
			giveItem(name, n)
		}
		if len(got) > 0 && useTick%4 == 0 {
			sfx("dig")
		}
		if chest[0] >= 0 {
			openChest(chest[0], chest[1])
		}
		switch {
		case core > 0:
			coreDust += core
			if coreDust >= 10 {
				giveItem("coreshard", coreDust/10)
				coreDust %= 10
			}
			if useTick%6 == 0 {
				burst(3, float64(aimX), float64(aimY), blockRGB[CORE], 1.4, 1.4, 24)
			}
		case core < 0 && useTick%40 == 1:
			toast("The Warden's power still holds the Core")
		}
		if useTick%20 == 1 {
			hitCreatures(aimX, aimY, digRadius(it), it.dmg)
		}
	case id == "hoe":
		n := 0
		eachInCircle(it.reach, func(x, y, i int) {
			switch grid[i] {
			case DIRT, GRASS:
				grid[i] = SOIL
				n++
			case CROP3:
				grid[i] = EMPTY
				giveItem("wheat", 1+rand.Intn(2))
				giveItem("seeds", 1+rand.Intn(2))
				n++
			}
		})
		if n > 0 && useTick%5 == 1 {
			sfx("dig")
		}
	case id == "bucket":
		n := 0
		eachInCircle(it.reach, func(x, y, i int) {
			if grid[i] == WATER {
				grid[i] = EMPTY
				n++
			}
		})
		if n > 0 {
			giveItem("water", n)
			if useTick%5 == 0 {
				sfx("slurp")
			}
		}
	case id == "seeds":
		n := 0
		eachInCircle(3, func(x, y, i int) {
			if grid[i] == EMPTY && y+1 < H && grid[i+W] == SOIL && inv[selected] != nil {
				grid[i] = CROP1
				takeSelected(1)
				n++
			}
		})
		if n > 0 {
			sfx("place")
		}
	case it.dmg > 0: // swords: swing every 18 frames while held
		if useTick%18 == 1 {
			swing(it)
		}
	case usesShapes(heldBuildKind()):
		// the brush builds (or paints, or knocks down walls) every frame; other shapes
		// are dragged out and built when you let go
		if buildShape != shapeBrush {
			return
		}
		if n := applyCells(brushCells()); n > 0 && useTick%4 == 1 {
			sfx("place")
		}
	}
}

func swing(it *Item) {
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	// swing toward the aim, but never further than the sword reaches
	dx, dy := float64(aimX)-pcx, float64(aimY)-pcy
	d := math.Hypot(dx, dy)
	r := float64(it.reach + 6)
	if d > r {
		dx, dy = dx/d*r, dy/d*r
	}
	if dx != 0 {
		player.face = sign(dx)
	}
	cx, cy := int(pcx+dx), int(pcy+dy)
	for k := 0; k < 10; k++ {
		a := math.Atan2(dy, dx) + (float64(k)-5)*0.15
		burst(1, pcx+math.Cos(a)*r*0.8, pcy+math.Sin(a)*r*0.8, [3]int{230, 230, 250}, 0.2, 0.2, 8)
	}
	sfx("swing")
	hitCreatures(cx, cy, it.reach, it.dmg)
}

func useOnce() {
	if selected < 0 || inv[selected] == nil {
		return
	}
	it := items[inv[selected].ID]
	switch {
	case it.heal > 0:
		if player.hp >= player.maxHP {
			toast("You're already at full health")
			return
		}
		takeSelected(1)
		player.hp = min(player.maxHP, player.hp+it.heal)
		sfx("eat")
		burst(8, (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL, [3]int{240, 86, 112}, 0.8, 1, 24)
	case it.id == "heart":
		takeSelected(1)
		player.maxHP = min(30, player.maxHP+2)
		player.hp = player.maxHP
		sfx("gem")
		toast("Max health up!")
	case it.id == "gem":
		takeSelected(1)
		player.energy = 1
		sfx("gem")
		burst(14, (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL, glowRGB, 1.2, 1.2, 30)
	case it.id == "bomb":
		if !aimCell() {
			return
		}
		takeSelected(1)
		explode(aimX, aimY, 12)
	case it.blueprint != "":
		if tx, ty, now := oneShotTarget(); now {
			placeBlueprint(it, tx, ty)
		}
	case it.station != "":
		if tx, ty, now := oneShotTarget(); now && placeStation(it, tx, ty) {
			holoSet = false
		}
	case it.place != 0: // torches: one at a time
		if !aimCell() || aimX < 1 || aimX >= W-1 || aimY < 1 || aimY >= H {
			return
		}
		i := aimY*W + aimX
		if !isReplaceable(grid[i]) {
			return
		}
		grid[i] = it.place
		shade[i] = 0
		markDirty(aimX)
		takeSelected(1)
		sfx("place")
	}
}

// Touch: the Build button on the build bar builds at the hologram
func buildHoloNow() {
	if selected < 0 || inv[selected] == nil || !holoSet {
		return
	}
	it := items[inv[selected].ID]
	switch {
	case it.blueprint != "":
		placeBlueprint(it, holoX, holoY)
	case it.station != "":
		if placeStation(it, holoX, holoY) {
			holoSet = false
		}
	}
}

func explode(cx, cy, r int) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := cx+dx, cy+dy
			if dx*dx+dy*dy > r*r || x < 0 || x >= W || y < 0 || y >= H {
				continue
			}
			i := y*W + x
			if t := grid[i]; t == BEDROCK || t == CORE || t == CHEST {
				continue
			}
			if grid[i] != EMPTY && rand.Float64() < 0.15 {
				burst(1, float64(x), float64(y), blockRGB[grid[i]], 2.4, 2.4, 40)
			}
			grid[i] = EMPTY
		}
	}
	burst(30, float64(cx), float64(cy), tipRGB, 3, 3, 26)
	burst(20, float64(cx), float64(cy), [3]int{240, 122, 106}, 2.5, 2.5, 22)
	hitCreatures(cx, cy, r, 6)
	shake = math.Max(shake, 10)
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	dist := math.Hypot(pcx-float64(cx), pcy-float64(cy))
	if rr := float64(r) * 2.5; dist < rr {
		f := (1 - dist/rr) * 9
		player.vy -= f
		d := sign(pcx - float64(cx))
		if d == 0 {
			d = 1
		}
		player.vx += float64(d) * f * 0.6
	}
	sfx("boom")
}

func drawAim(put func(x, y int, c [3]int, s int), ox, oy int) {
	if !useActive || selected < 0 || inv[selected] == nil {
		return
	}
	id := inv[selected].ID
	it := items[id]
	if it.heal > 0 || it.armor > 0 || it.blueprint != "" || it.station != "" || id == "gem" || id == "heart" || id == "lantern" || usesShapes(heldBuildKind()) {
		return
	}
	ok := aimCell()
	r := it.reach
	switch {
	case id == "pickaxe" || id == "ironpick" || id == "corepick":
		r = digRadius(it)
	case id == "bomb":
		r = 12
	case id == "torch":
		r = 1
	case id == "seeds":
		r = 3
	case it.place != 0:
		r = 4
	}
	c := tipRGB
	if !ok {
		c = [3]int{240, 122, 106}
	}
	for a := 0; a < 40; a++ {
		t := float64(a) / 40 * math.Pi * 2
		put(int(math.Round(float64(aimX)+math.Cos(t)*float64(r+1)))-ox, int(math.Round(float64(aimY)+math.Sin(t)*float64(r+1)))-oy, c, 0)
	}
}
