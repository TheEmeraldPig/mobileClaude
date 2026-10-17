package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

// ---- cards ----

const (
	kAttack = iota
	kSkill
)

const (
	iSword = iota
	iShield
	iFlame
	iDrop
	iBolt
	iHeart
	iCards
	iSkull
)

type cardDef struct {
	ID, Name                     string
	Cost, Kind, Icon, Rarity     int // rarity: 0 starter, 1 common, 2 rare
	Dmg, Hits, Block, Draw, Gain int
	Heal, Vuln, Weak, Poison     int
	Exhaust                      bool
}

var cardDefs = []cardDef{
	{ID: "strike", Name: "Strike", Cost: 1, Kind: kAttack, Icon: iSword, Dmg: 6},
	{ID: "defend", Name: "Defend", Cost: 1, Kind: kSkill, Icon: iShield, Block: 5},
	{ID: "bash", Name: "Bash", Cost: 2, Kind: kAttack, Icon: iSword, Dmg: 8, Vuln: 2},
	{ID: "jab", Name: "Quick Jab", Cost: 0, Kind: kAttack, Icon: iSword, Rarity: 1, Dmg: 4},
	{ID: "flurry", Name: "Flurry", Cost: 1, Kind: kAttack, Icon: iSword, Rarity: 1, Dmg: 2, Hits: 3},
	{ID: "heavy", Name: "Heavy Blow", Cost: 2, Kind: kAttack, Icon: iSword, Rarity: 1, Dmg: 15},
	{ID: "wall", Name: "Shield Wall", Cost: 2, Kind: kSkill, Icon: iShield, Rarity: 1, Block: 13},
	{ID: "wind", Name: "Second Wind", Cost: 1, Kind: kSkill, Icon: iShield, Rarity: 1, Block: 4, Draw: 1},
	{ID: "dart", Name: "Venom Dart", Cost: 1, Kind: kSkill, Icon: iDrop, Rarity: 1, Poison: 5},
	{ID: "shout", Name: "War Cry", Cost: 1, Kind: kSkill, Icon: iSkull, Rarity: 1, Weak: 2, Block: 3},
	{ID: "insight", Name: "Insight", Cost: 1, Kind: kSkill, Icon: iCards, Rarity: 1, Draw: 2},
	{ID: "fireball", Name: "Fireball", Cost: 2, Kind: kAttack, Icon: iFlame, Rarity: 2, Dmg: 11, Vuln: 1},
	{ID: "focus", Name: "Focus", Cost: 0, Kind: kSkill, Icon: iBolt, Rarity: 2, Gain: 2, Exhaust: true},
	{ID: "bandage", Name: "Bandage", Cost: 1, Kind: kSkill, Icon: iHeart, Rarity: 2, Heal: 7, Exhaust: true},
	{ID: "plague", Name: "Plague", Cost: 2, Kind: kSkill, Icon: iDrop, Rarity: 2, Poison: 12, Exhaust: true},
}

func cardDefByID(id string) cardDef {
	for _, c := range cardDefs {
		if c.ID == id {
			return c
		}
	}
	return cardDefs[0]
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
	{"Light", "Costs 1 less", 300, 0, color.RGBA{0xff, 0xf6, 0x9a, 0xff}},
	{"Gold", "+6 gold when played", 120, 0, color.RGBA{0xff, 0xc4, 0x2e, 0xff}},
	{"Echo", "Plays twice", 0, 30, color.RGBA{0xa8, 0x6b, 0xff, 0xff}},
}

// A card you own: which card, and its enhancement.
type Card struct {
	ID  string `json:"id"`
	Enh int    `json:"enh,omitempty"`
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
		l = append(l, "Twice")
	}
	if c.Enh == eGold {
		l = append(l, "+6 gold")
	}
	if d.Exhaust {
		l = append(l, "Once per fight")
	}
	return strings.Join(l, "\n")
}

func starterDeck() []Card {
	var d []Card
	for i := 0; i < 5; i++ {
		d = append(d, Card{ID: "strike"})
	}
	for i := 0; i < 4; i++ {
		d = append(d, Card{ID: "defend"})
	}
	return append(d, Card{ID: "bash"})
}

// ---- charms: rule-bending items found during a run (max 4) ----

type charmDef struct{ ID, Name, Desc string }

var charmDefs = []charmDef{
	{"whetstone", "Whetstone", "Attacks deal +2"},
	{"ironskin", "Iron Skin", "Start fights with 8 block"},
	{"battery", "Battery", "+1 energy on your first turn"},
	{"hands", "Quick Hands", "Draw 1 more card each turn"},
	{"fang", "Vampire Fang", "Heal 2 when you play an attack"},
	{"combo", "Combo Belt", "Every 3rd card each turn is doubled"},
	{"coin", "Lucky Coin", "+50% gold from fights"},
	{"thorns", "Thorn Mail", "Enemies take 3 when they hit you"},
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
	shOrc
	shKing
)

type enemyDef struct {
	Name  string
	HP    int
	Shape int
	Col   color.RGBA
	Moves []move
}

var normals = []enemyDef{
	{"Slime", 22, shSlime, color.RGBA{0x6c, 0xd8, 0x4a, 0xff}, []move{{mAttack, 6, 1}, {mAttack, 7, 1}, {mBlock, 6, 0}}},
	{"Goblin", 26, shGoblin, color.RGBA{0x8a, 0xc0, 0x3a, 0xff}, []move{{mAttack, 8, 1}, {mBuff, 2, 0}, {mAttack, 5, 1}}},
	{"Bat", 17, shBat, color.RGBA{0x9b, 0x6b, 0xe0, 0xff}, []move{{mAttack, 3, 2}, {mDebuff, 1, 0}, {mAttack, 3, 3}}},
}

var elite = enemyDef{"Orc Brute", 52, shOrc, color.RGBA{0xd0, 0x6a, 0x48, 0xff}, []move{{mAttack, 12, 1}, {mBlock, 10, 0}, {mAttack, 6, 2}, {mBuff, 3, 0}}}
var boss = enemyDef{"Troll King", 115, shKing, color.RGBA{0x5f, 0xa8, 0x8a, 0xff}, []move{{mAttack, 14, 1}, {mBlock, 14, 0}, {mAttack, 5, 3}, {mBuff, 3, 0}, {mDebuff, 2, 0}}}

var chapterNames = []string{"Goblin Woods", "Bat Caves", "Orc Pass", "Troll Swamp", "Frost Peak", "Shadow Keep"}

func chapterName(ch int) string {
	if ch-1 < len(chapterNames) {
		return chapterNames[ch-1]
	}
	return fmt.Sprintf("The Deep %d", ch)
}

const floorsPerChapter = 10
