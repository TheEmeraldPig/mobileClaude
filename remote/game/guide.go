package game

// A short guide for a new world, shown under the hearts until it's done (turn it off
// in Settings > Display > Tips). Progress is saved with the world.

type guideStep struct {
	text, touch string // text for PC, and for phones when different
	done        func() bool
}

func hasObj(kind string) bool {
	for _, o := range objs {
		if o.kind == kind {
			return true
		}
	}
	return false
}

var guide = []guideStep{
	{"Hold the pickaxe on a tree trunk to get wood", "", func() bool { return countItem("wood") > 0 }},
	{"Press C to craft planks from wood", "Tap the hammer button to craft planks from wood", func() bool { return countItem("plank") >= 4 || hasObj("workbench") }},
	{"Craft a workbench (12 planks) and place it", "", func() bool { return hasObj("workbench") }},
	{"Right-click the workbench and make a furnace", "Tap the workbench and make a furnace", func() bool { return hasObj("furnace") }},
	{"Dig for copper ore and smelt it at the furnace", "", func() bool { return countItem("copperbar") > 0 || hasObj("anvil") }},
	{"Make a bed so you wake up at home", "", func() bool { return hasObj("bed") }},
}

var (
	guideStepAt int    // saved per world
	guideY      = 60.0 // below the HUD hints
)

func drawGuide() {
	if !prefs.Tips || guideStepAt >= len(guide) {
		return
	}
	for guideStepAt < len(guide) && guide[guideStepAt].done() {
		guideStepAt++
		sfx("gem")
		if guideStepAt == len(guide) {
			toast("Guide complete. Have fun!")
			return
		}
	}
	g := guide[guideStepAt]
	t := g.text
	if isTouch && g.touch != "" {
		t = g.touch
	}
	ShadowText("Next: "+t, 14, guideY, 12, colL)
}
