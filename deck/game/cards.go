package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

// ---- rarity ----

const (
	rCommon = iota
	rUncommon
	rRare
	rEpic
	rLegendary
	rarityCount
)

var rarityNames = []string{"Common", "Uncommon", "Rare", "Epic", "Legendary"}

var rarityCol = []color.RGBA{
	{0xb4, 0xbe, 0xcf, 0xff},
	{0x5c, 0xd6, 0x5a, 0xff},
	{0x45, 0xa6, 0xff, 0xff},
	{0xb5, 0x5c, 0xff, 0xff},
	{0xff, 0xa8, 0x1f, 0xff},
}

// ---- cards ----

const (
	kAttack = iota
	kSkill
)

// card illustrations (see cardart.go)
const (
	aSword = iota
	aShield
	aDagger
	aTwin
	aDart
	aBrace
	aHammer
	aAxe
	aTower
	aHorn
	aBook
	aFist
	aFire
	aCloud
	aBolt
	aFang
	aMeteor
	aPhoenix
	aExcalibur
	aHourglass
	aAegis
	aDragon
)

type cardDef struct {
	ID, Name                string
	Cost, Kind, Art, Rarity int
	Dmg, Hits, Block, Draw  int
	Gain, Heal, Vuln, Weak  int
	Poison, Str             int
	Exhaust                 bool
}

var cardDefs = []cardDef{
	// common
	{ID: "strike", Name: "Strike", Cost: 1, Kind: kAttack, Art: aSword, Dmg: 6},
	{ID: "defend", Name: "Defend", Cost: 1, Kind: kSkill, Art: aShield, Block: 5},
	{ID: "jab", Name: "Quick Jab", Cost: 0, Kind: kAttack, Art: aDagger, Dmg: 3},
	{ID: "twin", Name: "Twin Slash", Cost: 1, Kind: kAttack, Art: aTwin, Dmg: 3, Hits: 2},
	{ID: "dart", Name: "Venom Dart", Cost: 1, Kind: kAttack, Art: aDart, Dmg: 2, Poison: 3},
	{ID: "brace", Name: "Brace", Cost: 1, Kind: kSkill, Art: aBrace, Block: 3, Draw: 1},
	// uncommon
	{ID: "bash", Name: "Bash", Cost: 2, Kind: kAttack, Art: aHammer, Rarity: rUncommon, Dmg: 8, Vuln: 2},
	{ID: "heavy", Name: "Heavy Blow", Cost: 2, Kind: kAttack, Art: aAxe, Rarity: rUncommon, Dmg: 14},
	{ID: "wall", Name: "Shield Wall", Cost: 2, Kind: kSkill, Art: aTower, Rarity: rUncommon, Block: 12},
	{ID: "warcry", Name: "War Cry", Cost: 1, Kind: kSkill, Art: aHorn, Rarity: rUncommon, Weak: 2, Block: 3},
	{ID: "insight", Name: "Insight", Cost: 1, Kind: kSkill, Art: aBook, Rarity: rUncommon, Draw: 2},
	{ID: "rage", Name: "Rage", Cost: 1, Kind: kSkill, Art: aFist, Rarity: rUncommon, Str: 2, Exhaust: true},
	{ID: "flurry", Name: "Flurry", Cost: 1, Kind: kAttack, Art: aTwin, Rarity: rUncommon, Dmg: 2, Hits: 4},
	// rare
	{ID: "fireball", Name: "Fireball", Cost: 2, Kind: kAttack, Art: aFire, Rarity: rRare, Dmg: 12, Vuln: 1},
	{ID: "plague", Name: "Plague", Cost: 2, Kind: kSkill, Art: aCloud, Rarity: rRare, Poison: 10, Exhaust: true},
	{ID: "focus", Name: "Focus", Cost: 0, Kind: kSkill, Art: aBolt, Rarity: rRare, Gain: 2, Exhaust: true},
	{ID: "fortress", Name: "Fortress", Cost: 2, Kind: kSkill, Art: aTower, Rarity: rRare, Block: 18},
	{ID: "leech", Name: "Leech", Cost: 1, Kind: kAttack, Art: aFang, Rarity: rRare, Dmg: 6, Heal: 3},
	// epic
	{ID: "meteor", Name: "Meteor", Cost: 3, Kind: kAttack, Art: aMeteor, Rarity: rEpic, Dmg: 30},
	{ID: "toxic", Name: "Toxic Cloud", Cost: 1, Kind: kSkill, Art: aCloud, Rarity: rEpic, Poison: 6, Vuln: 2, Weak: 2},
	{ID: "overdrive", Name: "Overdrive", Cost: 0, Kind: kSkill, Art: aBolt, Rarity: rEpic, Gain: 3, Draw: 2, Exhaust: true},
	{ID: "phoenix", Name: "Phoenix", Cost: 2, Kind: kAttack, Art: aPhoenix, Rarity: rEpic, Dmg: 16, Heal: 6},
	// legendary
	{ID: "excalibur", Name: "Excalibur", Cost: 1, Kind: kAttack, Art: aExcalibur, Rarity: rLegendary, Dmg: 20, Draw: 1},
	{ID: "timewarp", Name: "Time Warp", Cost: 1, Kind: kSkill, Art: aHourglass, Rarity: rLegendary, Draw: 4, Gain: 2, Exhaust: true},
	{ID: "aegis", Name: "Aegis", Cost: 2, Kind: kSkill, Art: aAegis, Rarity: rLegendary, Block: 30, Heal: 8, Exhaust: true},
	{ID: "dragon", Name: "Dragon Breath", Cost: 3, Kind: kAttack, Art: aDragon, Rarity: rLegendary, Dmg: 18, Poison: 10, Vuln: 3},
}

var cardIndex = func() map[string]int {
	m := map[string]int{}
	for i, c := range cardDefs {
		m[c.ID] = i
	}
	return m
}()

func cardDefByID(id string) cardDef {
	if i, ok := cardIndex[id]; ok {
		return cardDefs[i]
	}
	return cardDefs[0]
}

func cardsOfRarity(r int) []string {
	var ids []string
	for _, c := range cardDefs {
		if c.Rarity == r {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// ---- enhancements: the Balatro part. Any card can carry one. ----

const (
	eNone = iota
	eFoil
	eHolo
	eLight
	eGold
	eEcho
	eCount
)

type enhDef struct {
	Name, Desc string
	Gold, Gems int // price in the Deck screen
	Col        color.RGBA
}

var enhDefs = []enhDef{
	{},
	{"Foil", "+3 damage and block", 150, 0, color.RGBA{0xd8, 0xe6, 0xf2, 0xff}},
	{"Holo", "x1.5 damage and block", 450, 0, color.RGBA{0xff, 0x6b, 0xd6, 0xff}},
	{"Light", "Costs 1 less", 300, 0, color.RGBA{0xff, 0xf2, 0x80, 0xff}},
	{"Gold", "+6 coins when played", 120, 0, color.RGBA{0xff, 0xc4, 0x2e, 0xff}},
	{"Echo", "Plays twice", 0, 30, color.RGBA{0xa8, 0x6b, 0xff, 0xff}},
}

// Card is a card you own: which card, its enhancement, and (in your collection)
// whether it is in your starting deck.
type Card struct {
	ID  string `json:"id"`
	Enh int    `json:"enh,omitempty"`
	In  bool   `json:"in,omitempty"`
}

func (c Card) def() cardDef { return cardDefByID(c.ID) }

// stats is a card's numbers after its enhancement.
func (c Card) stats() cardDef {
	d := c.def()
	switch c.Enh {
	case eFoil:
		if d.Dmg > 0 {
			d.Dmg += 3
		}
		if d.Block > 0 {
			d.Block += 3
		}
	case eHolo:
		d.Dmg = int(math.Ceil(float64(d.Dmg) * 1.5))
		d.Block = int(math.Ceil(float64(d.Block) * 1.5))
	case eLight:
		d.Cost = max(0, d.Cost-1)
	}
	return d
}

func (c Card) text() string {
	d := c.stats()
	var l []string
	if d.Dmg > 0 {
		if d.Hits > 1 {
			l = append(l, fmt.Sprintf("Deal %d x%d", d.Dmg, d.Hits))
		} else {
			l = append(l, fmt.Sprintf("Deal %d", d.Dmg))
		}
	}
	if d.Block > 0 {
		l = append(l, fmt.Sprintf("Block %d", d.Block))
	}
	if d.Poison > 0 {
		l = append(l, fmt.Sprintf("Poison %d", d.Poison))
	}
	if d.Vuln > 0 {
		l = append(l, fmt.Sprintf("Vulnerable %d", d.Vuln))
	}
	if d.Weak > 0 {
		l = append(l, fmt.Sprintf("Weak %d", d.Weak))
	}
	if d.Str > 0 {
		l = append(l, fmt.Sprintf("+%d Strength", d.Str))
	}
	if d.Draw > 0 {
		l = append(l, fmt.Sprintf("Draw %d", d.Draw))
	}
	if d.Gain > 0 {
		l = append(l, fmt.Sprintf("+%d Energy", d.Gain))
	}
	if d.Heal > 0 {
		l = append(l, fmt.Sprintf("Heal %d", d.Heal))
	}
	if c.Enh == eEcho {
		l = append(l, "Plays twice")
	}
	if c.Enh == eGold {
		l = append(l, "+6 coins")
	}
	if d.Exhaust {
		l = append(l, "Once per fight")
	}
	return strings.Join(l, "\n")
}

// starterCollection is what a new player owns.
func starterCollection() []Card {
	return []Card{
		{ID: "strike", In: true}, {ID: "strike", In: true}, {ID: "defend", In: true}, {ID: "defend", In: true},
		{ID: "bash", In: true}, {ID: "strike"}, {ID: "defend"}, {ID: "jab"}, {ID: "twin"}, {ID: "brace"},
	}
}

// ---- starting deck rules ----

const maxLoadout = 5

// checkLoadout enforces the starting-deck budget: up to 5 cards, and at most
// 1 Legendary, or 2 Epics (with no Uncommons), or 3 Rares plus 1 Uncommon.
// Without any of those, any mix of Commons and Uncommons is fine.
func checkLoadout(cards []Card) error {
	if len(cards) == 0 {
		return fmt.Errorf("Your deck needs at least 1 card")
	}
	if len(cards) > maxLoadout {
		return fmt.Errorf("Your starting deck holds %d cards", maxLoadout)
	}
	var n [rarityCount]int
	for _, c := range cards {
		n[c.def().Rarity]++
	}
	switch {
	case n[rLegendary] > 0:
		if n[rLegendary] > 1 {
			return fmt.Errorf("Only 1 Legendary fits")
		}
		if n[rEpic]+n[rRare]+n[rUncommon] > 0 {
			return fmt.Errorf("A Legendary only goes with Commons")
		}
	case n[rEpic] > 0:
		if n[rEpic] > 2 {
			return fmt.Errorf("Only 2 Epics fit")
		}
		if n[rRare]+n[rUncommon] > 0 {
			return fmt.Errorf("Epics only go with Commons")
		}
	case n[rRare] > 0:
		if n[rRare] > 3 {
			return fmt.Errorf("Only 3 Rares fit")
		}
		if n[rUncommon] > 1 {
			return fmt.Errorf("With Rares, only 1 Uncommon fits")
		}
	}
	return nil
}

// ---- charms: rule-bending items found during a run ----

type charmDef struct {
	ID, Name, Desc string
	Col            color.RGBA
}

var charmDefs = []charmDef{
	{"whetstone", "Whetstone", "Attacks deal +2 per hit", color.RGBA{0x9a, 0xa8, 0xc0, 0xff}},
	{"ironskin", "Iron Skin", "Start fights with 8 block", color.RGBA{0x8a, 0x9a, 0xb8, 0xff}},
	{"battery", "Battery", "+1 energy on your first turn", color.RGBA{0xff, 0xd8, 0x30, 0xff}},
	{"hands", "Quick Hands", "Draw 1 more card each turn", color.RGBA{0x5c, 0xc8, 0xff, 0xff}},
	{"fang", "Vampire Fang", "Heal 2 when you play an attack", color.RGBA{0xe0, 0x30, 0x50, 0xff}},
	{"combo", "Combo Belt", "Every 3rd card each turn is doubled", color.RGBA{0xff, 0x8a, 0x2a, 0xff}},
	{"coin", "Lucky Coin", "+50% coins from fights", color.RGBA{0xff, 0xc8, 0x2e, 0xff}},
	{"thorns", "Thorn Mail", "Enemies take 3 when they hit you", color.RGBA{0x5a, 0xa0, 0x40, 0xff}},
	{"tower", "Tower Shield", "Block stays between turns", color.RGBA{0x40, 0x70, 0xd0, 0xff}},
	{"venom", "Venom Gland", "Poison you apply is +2", color.RGBA{0x70, 0xe0, 0x40, 0xff}},
	{"heart", "Heartstone", "+12 max HP", color.RGBA{0xff, 0x50, 0x80, 0xff}},
}

func charmByID(id string) charmDef {
	for _, c := range charmDefs {
		if c.ID == id {
			return c
		}
	}
	return charmDefs[0]
}

// ---- enemies ----

const (
	mAttack = iota
	mBlock
	mBuff   // +strength
	mDebuff // weakens the hero
)

type move struct{ Kind, Val, Hits int }

const (
	shSlime = iota
	shGoblin
	shBat
	shShroom
	shSkeleton
	shOrc
	shGolem
	shKing
)

type enemyDef struct {
	Name  string
	HP    int
	Shape int
	Moves []move
}

var normals = []enemyDef{
	{"Slime", 18, shSlime, []move{{mAttack, 5, 1}, {mAttack, 6, 1}, {mBlock, 5, 0}}},
	{"Goblin", 22, shGoblin, []move{{mAttack, 7, 1}, {mBuff, 2, 0}, {mAttack, 4, 1}}},
	{"Bat", 15, shBat, []move{{mAttack, 3, 2}, {mDebuff, 1, 0}, {mAttack, 2, 3}}},
	{"Shroom", 24, shShroom, []move{{mDebuff, 1, 0}, {mAttack, 8, 1}, {mBlock, 6, 0}}},
	{"Skeleton", 26, shSkeleton, []move{{mAttack, 9, 1}, {mBlock, 8, 0}, {mAttack, 4, 2}}},
}

var elites = []enemyDef{
	{"Orc Brute", 48, shOrc, []move{{mAttack, 12, 1}, {mBlock, 10, 0}, {mAttack, 6, 2}, {mBuff, 3, 0}}},
	{"Stone Golem", 56, shGolem, []move{{mBlock, 14, 0}, {mAttack, 15, 1}, {mDebuff, 1, 0}, {mAttack, 8, 2}}},
}

var boss = enemyDef{"Troll King", 110, shKing, []move{{mAttack, 14, 1}, {mBlock, 14, 0}, {mAttack, 5, 3}, {mBuff, 3, 0}, {mDebuff, 2, 0}}}

// ---- chapters ----

type theme struct {
	Name             string
	Sky1, Sky2       color.RGBA // sky top and horizon
	Far, Near        color.RGBA // mountains and hills
	Ground1, Ground2 color.RGBA
	Leaf             color.RGBA
	Hue              float64 // tint for enemies
}

var themes = []theme{
	{"Goblin Woods", rgb(0x7e, 0xd4, 0xff), rgb(0xd8, 0xf4, 0xff), rgb(0x86, 0xb8, 0xc8), rgb(0x4c, 0x9a, 0x50), rgb(0x8c, 0xc2, 0x58), rgb(0x5a, 0x8e, 0x3c), rgb(0x3c, 0x9a, 0x40), 0},
	{"Bat Caves", rgb(0x2a, 0x22, 0x50), rgb(0x6a, 0x4c, 0x9a), rgb(0x3e, 0x30, 0x66), rgb(0x52, 0x3c, 0x78), rgb(0x7a, 0x6a, 0x8a), rgb(0x4e, 0x40, 0x5e), rgb(0x6c, 0x4c, 0xa0), 0.62},
	{"Orc Pass", rgb(0xff, 0xb0, 0x6a), rgb(0xff, 0xe2, 0xa8), rgb(0xc8, 0x8a, 0x60), rgb(0xa8, 0x68, 0x40), rgb(0xd8, 0xa8, 0x6a), rgb(0xa8, 0x78, 0x48), rgb(0x9a, 0x8a, 0x3a), 0.08},
	{"Troll Swamp", rgb(0x5a, 0x9a, 0x8a), rgb(0xb8, 0xe0, 0xc8), rgb(0x5a, 0x80, 0x7a), rgb(0x3a, 0x6a, 0x58), rgb(0x6a, 0x8a, 0x5a), rgb(0x44, 0x5e, 0x40), rgb(0x2e, 0x7a, 0x5a), 0.38},
	{"Frost Peak", rgb(0x7c, 0xb8, 0xff), rgb(0xe8, 0xf6, 0xff), rgb(0xb8, 0xd0, 0xe8), rgb(0x8c, 0xb0, 0xd8), rgb(0xe4, 0xf0, 0xfa), rgb(0xa8, 0xc4, 0xe0), rgb(0x5a, 0x8a, 0x9a), 0.55},
	{"Shadow Keep", rgb(0x3a, 0x10, 0x2a), rgb(0xa8, 0x3a, 0x4a), rgb(0x4a, 0x1e, 0x36), rgb(0x2e, 0x16, 0x2a), rgb(0x5a, 0x4a, 0x56), rgb(0x36, 0x2a, 0x34), rgb(0x6a, 0x2a, 0x3a), 0.9},
}

func chapterTheme(ch int) theme { return themes[(ch-1)%len(themes)] }

func chapterName(ch int) string {
	t := chapterTheme(ch)
	if ch > len(themes) {
		return fmt.Sprintf("%s %d", t.Name, (ch-1)/len(themes)+1)
	}
	return t.Name
}
