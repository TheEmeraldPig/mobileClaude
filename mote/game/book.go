package game

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The guide book: chapters about everything in the game, opened from the pause menu.

type chapter struct {
	title string
	icon  string // item id for the chapter's icon
	pages []string
}

var book = []chapter{
	{"Getting started", "pickaxe", []string{
		"You are a mote: a little levitating creature in a world of pixels. Walk, levitate (it uses the bar under you, which refills on the ground) and use the item in your hand on the world.",
		"First steps: hold your pickaxe on a tree trunk for wood. Make planks by hand (crafting), then a workbench, and place it. The workbench makes most other things.",
		"Night is darker and shades prowl the surface. Light torches, build a shelter with a door, and make a bed: sleeping in it (tap it) sets where you wake up after fainting.",
	}},
	{"Mining and tools", "copperpick", []string{
		"Every block has a hardness. Hold the pickaxe over blocks and they crack and break; soft dirt and sand go fast, stone takes longer, ores and obsidian much longer.",
		"Pickaxes come in tiers: stone, copper, iron, gold and core. Better ones dig faster and wider. Iron, gold and crystal need copper or better; obsidian and the Core need iron or better.",
		"Right-click with a pickaxe (or switch Mining to Back walls on phones) to dig out background walls. The hammer takes walls down too.",
	}},
	{"Crafting and stations", "workbench", []string{
		"Hand crafting (C, or the hammer button) makes planks, torches and the workbench. Everything else is made at a station: tap or right-click it to open its own menu.",
		"Workbench: tools, building blocks, walls, doors, beds, storage, the furnace and the anvil, and blueprints. Furnace: bars from ore, glass, bread. Anvil: metal tools, weapons and armor.",
		"Stations stand behind you, so you can walk past them. Hold a pickaxe on one to pick it back up. Storage chests hold 24 stacks.",
	}},
	{"Building", "bp_hut", []string{
		"Pick a block, wall or the paintbrush and the build bar appears: brush, line, box, filled box and circle, plus a size. A see-through hologram shows what you'll build before you let go.",
		"Walls go behind you and keep creatures from spawning in front of them. Paint any block or wall in 13 colours. Ctrl+Z (or Undo) takes back the last stroke and returns the items.",
		"Blueprints build whole structures at once: a hut, a tower, a bridge, a farm and a bunker. Rotate (R) and flip (F) them before placing.",
	}},
	{"Light and darkness", "torch", []string{
		"Anything not lit is hidden. Light has colour: torches are warm, crystals cyan, lava red, the Core pink, and you glow violet. A lantern in your bag makes your glow brighter.",
		"Sunlight reaches down until it hits the ground, so caves are dark. Bats avoid bright torchlight; other creatures only appear in the dark.",
	}},
	{"Creatures", "coppersword", []string{
		"Gloomlings crawl cave floors, bats flit about, skitters dart in and back off, wisps glow and circle before diving, and brutes are slow but tough. Shades walk the surface at night.",
		"They find their way around walls and through tunnels, but you are always a little faster. They drop materials and coins. Swords hit harder than a pickaxe; armor in your bag blocks damage.",
		"Deep down, the Warden guards the Core. Beat it and the Core can be mined for shards that make the Core pickaxe.",
	}},
	{"Places to explore", "crystal", []string{
		"Graves on the surface hide a small chest. Brick dungeons lie underground, and much bigger ruined temples, three floors high, deeper still.",
		"Deserts have pyramids with a treasure room. High above the land float sky islands with gold, crystal and a shrine: levitate up to reach them.",
		"Villages sit on the surface. Each house has a trader.",
	}},
	{"Traders and coins", "coin", []string{
		"Pip the Merchant, Flint the Smith and Fern the Farmer live in village houses. Tap them (right-click on PC) to buy and sell.",
		"Coins come from selling ore, bars, food and gear, from creatures, and from chests. Traders sell things that are hard to find, like healing potions, heart crystals and better tools.",
	}},
	{"Game modes", "heart", []string{
		"Easy: you take less damage and keep everything when you faint. Normal: you drop your blocks and materials but keep tools and gear.",
		"Hard: more damage, you drop everything and wake up with a temporary kit that fades once you pick up another tool. Hardcore: like Hard, but fainting deletes the world.",
	}},
}

var (
	bookChapter int
	bookPage    int
	bookFrom    = scPause
)

func openBook(from screen) {
	bookFrom = from
	openScreen(scBook)
}

func bookMenu() {
	r := sheet(900, 460)
	back, cl := head(r, "GUIDE BOOK", true)
	if back {
		openScreen(bookFrom)
		return
	}
	if cl {
		if bookFrom == scMain {
			openScreen(scMain)
		} else {
			play()
		}
		return
	}
	// chapters down the left
	listW := math.Min(250, r.W*0.32)
	y := r.Y + 60
	bh := math.Min(36, (r.H-76)/float64(len(book))-5)
	for i, c := range book {
		st := styleSlate
		if i == bookChapter {
			st = styleViolet
		}
		var ico *ebiten.Image
		if it := items[c.icon]; it != nil {
			ico = it.img
		}
		if Button(Rect{r.X + 18, y, listW, bh}, c.title, st, true, ico) {
			bookChapter, bookPage = i, 0
		}
		y += bh + 5
	}
	// the page
	c := book[bookChapter]
	pr := Rect{r.X + 36 + listW, r.Y + 60, r.W - 54 - listW, r.H - 76}
	Inset(pr, colSlateLo)
	ops = append(ops, func(dst *ebiten.Image) {
		drawText(dst, strings.ToUpper(c.title), pr.X+18, pr.Y+22, textOpt{size: 13, display: true, col: colT, shadow: true})
	})
	size := 15.0
	if isTouch {
		size = 14
	}
	wrapText(c.pages[bookPage], pr.X+18, pr.Y+56, pr.W-36, size, colInk, text.AlignStart)
	// page turning
	n := len(c.pages)
	if n > 1 {
		py := pr.Y + pr.H - 44
		if Button(Rect{pr.X + 14, py, 40, 32}, "<", styleSlate, bookPage > 0, nil) {
			bookPage--
		}
		if Button(Rect{pr.X + pr.W - 54, py, 40, 32}, ">", styleSlate, bookPage < n-1, nil) {
			bookPage++
		}
		Text(itoa(bookPage+1)+" / "+itoa(n), pr.X+pr.W/2, py+16, 13, colMuted, text.AlignCenter)
	}
}
