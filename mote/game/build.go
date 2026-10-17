package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Advanced building: shapes (brush, line, box, filled box, circle), brush size,
// rotating and flipping blueprints and stations, background walls and paint.
// A see-through hologram shows what will be built before you build it.
//
// Keyboard: B changes the shape, - and = change the size, R rotates, F flips,
// C changes the paint colour. On phones the build bar above the hotbar does the same.

const (
	shapeBrush = iota
	shapeLine
	shapeBox
	shapeFill
	shapeCircle
)

var shapeNames = []string{"Brush", "Line", "Box", "Filled box", "Circle"}

var (
	buildShape = shapeBrush
	brushSize  = 3 // radius in cells for the brush (0 = a single cell)
	buildRot   int // quarter turns for blueprints
	buildFlip  bool
	paintIdx   = 1 // selected paint colour; 0 removes paint

	shapeOn    bool // a shape is being dragged out
	shapeX0    int
	shapeY0    int
	shapeCells [][2]int // cells of the shape being dragged (for the hologram)
)

// Paint colours (index 0 = no paint)
var paintRGB = [][3]int{
	{0, 0, 0},
	{226, 72, 72}, {240, 146, 60}, {240, 210, 80}, {120, 200, 90}, {60, 160, 120},
	{70, 170, 220}, {80, 100, 220}, {150, 90, 220}, {230, 110, 190}, {245, 245, 250},
	{130, 130, 140}, {40, 38, 50}, {140, 90, 56},
}

var paintNames = []string{"Remove paint", "Red", "Orange", "Yellow", "Green", "Teal", "Sky", "Blue", "Purple", "Pink",
	"White", "Grey", "Black", "Brown"}

// What kind of building the held item does
const (
	buildNone = iota
	buildBlock
	buildWall
	buildPaint
	buildHammer
	buildOneShot // blueprints and stations: hologram, rotate/flip
)

func heldBuildKind() int {
	if selected < 0 || inv[selected] == nil {
		return buildNone
	}
	it := items[inv[selected].ID]
	switch {
	case it.wall != 0:
		return buildWall
	case it.id == "paintbrush":
		return buildPaint
	case it.id == "hammer":
		return buildHammer
	case it.blueprint != "" || it.station != "" || it.id == "door":
		return buildOneShot
	case it.place != 0 && !it.once:
		return buildBlock
	}
	return buildNone
}

func usesShapes(k int) bool {
	return k == buildBlock || k == buildWall || k == buildPaint || k == buildHammer
}

// ---- Hologram position for one-shot builds ----

var (
	holoX, holoY int  // cell the hologram sits on (touch)
	holoSet      bool // a touch has positioned the hologram
)

// Where the hologram of a blueprint or station goes this frame
func holoAim() (int, int, bool) {
	if isTouch {
		if !holoSet {
			holoX = int((player.x+player.w/2)/CELL) + player.face*24
			holoY = int((player.y + player.h) / CELL)
		}
		return holoX, holoY, true
	}
	return int(math.Floor((cam.x + hoverX/zoom) / CELL)), int(math.Floor((cam.y + hoverY/zoom) / CELL)), true
}

// One-shot items on touch: the first tap moves the hologram, a tap on it builds.
// Returns the cell to build at and whether to build now.
func oneShotTarget() (int, int, bool) {
	tx, ty := int(math.Floor((cam.x+aimSX/zoom)/CELL)), int(math.Floor((cam.y+aimSY/zoom)/CELL))
	if !isTouch {
		return tx, ty, true
	}
	if !holoSet || math.Hypot(float64(tx-holoX), float64(ty-holoY)) > 12 {
		holoX, holoY, holoSet = tx, ty, true
		return tx, ty, false
	}
	return holoX, holoY, true
}

// ---- Shapes ----

func shapeCellsFor(shape, x0, y0, x1, y1, size int) [][2]int {
	var out [][2]int
	seen := map[[2]int]bool{}
	add := func(x, y int) {
		k := [2]int{x, y}
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	// cap the size of one shape
	x1 = x0 + max(-80, min(80, x1-x0))
	y1 = y0 + max(-80, min(80, y1-y0))
	switch shape {
	case shapeLine:
		n := max(abs(x1-x0), abs(y1-y0))
		for k := 0; k <= n; k++ {
			t := 0.0
			if n > 0 {
				t = float64(k) / float64(n)
			}
			x := x0 + int(math.Round(float64(x1-x0)*t))
			y := y0 + int(math.Round(float64(y1-y0)*t))
			for dy := 0; dy <= size/2; dy++ { // thickness grows a little with size
				for dx := 0; dx <= size/2; dx++ {
					add(x+dx, y+dy)
				}
			}
		}
	case shapeBox, shapeFill:
		ax, bx := min(x0, x1), max(x0, x1)
		ay, by := min(y0, y1), max(y0, y1)
		t := max(1, size/2+1)
		for y := ay; y <= by; y++ {
			for x := ax; x <= bx; x++ {
				if shape == shapeFill || x-ax < t || bx-x < t || y-ay < t || by-y < t {
					add(x, y)
				}
			}
		}
	case shapeCircle:
		r := math.Hypot(float64(x1-x0), float64(y1-y0))
		t := float64(max(1, size/2+1))
		ri := int(math.Ceil(r))
		for dy := -ri; dy <= ri; dy++ {
			for dx := -ri; dx <= ri; dx++ {
				d := math.Hypot(float64(dx), float64(dy))
				if d <= r+0.5 && d > r-t {
					add(x0+dx, y0+dy)
				}
			}
		}
	}
	return out
}

func shapeAimCell() (int, int) {
	return int(math.Floor((cam.x + aimSX/zoom) / CELL)), int(math.Floor((cam.y + aimSY/zoom) / CELL))
}

// Called when a world pointer goes down with a shape-building item
func startShape() {
	shapeX0, shapeY0 = shapeAimCell()
	shapeOn = true
	shapeCells = shapeCellsFor(buildShape, shapeX0, shapeY0, shapeX0, shapeY0, brushSize)
}

func updateShape() {
	if !shapeOn {
		return
	}
	x1, y1 := shapeAimCell()
	shapeCells = shapeCellsFor(buildShape, shapeX0, shapeY0, x1, y1, brushSize)
}

// Called when the pointer is released: build the whole shape at once
func finishShape() {
	if !shapeOn {
		return
	}
	updateShape()
	shapeOn = false
	n := applyCells(shapeCells)
	shapeCells = nil
	if n > 0 {
		sfx("build")
	}
}

// Within reach of the mote (shapes can be big, so a bit further than normal)
func cellInBuildReach(x, y int) bool {
	pcx, pcy := (player.x+player.w/2)/CELL, (player.y+player.h/2)/CELL
	return math.Hypot(float64(x)-pcx, float64(y)-pcy) <= reach*2
}

// ---- Undo ----
// Each press of the pointer (a brush stroke or a shape) is one action that can be undone:
// cells that are still as you left them go back, and their items come back (a hammered
// wall goes back up and its item is taken back).

type cellChange struct {
	wx, y    int
	layer    byte // 'g' block, 'w' wall, 'p' block paint, 'q' wall paint
	old, now uint8
	give     string // item to give back when this change is undone ("" none)
	take     string // item to take back when it's undone
}

type buildAction struct{ changes []cellChange }

var (
	undoStack []*buildAction
	curAction *buildAction
)

func beginAction() { curAction = &buildAction{} }

func endAction() {
	if curAction != nil && len(curAction.changes) > 0 {
		undoStack = append(undoStack, curAction)
		if len(undoStack) > 20 {
			undoStack = undoStack[1:]
		}
	}
	curAction = nil
}

func record(c cellChange, x int) {
	if curAction != nil {
		c.wx = x + baseChunk*CW
		curAction.changes = append(curAction.changes, c)
	}
}

func layerOf(l byte) []uint8 {
	switch l {
	case 'g':
		return grid
	case 'w':
		return wall
	case 'p':
		return paint
	}
	return wpaint
}

func undoBuild() {
	if len(undoStack) == 0 {
		toast("Nothing to undo")
		return
	}
	a := undoStack[len(undoStack)-1]
	undoStack = undoStack[:len(undoStack)-1]
	n := 0
	for k := len(a.changes) - 1; k >= 0; k-- {
		c := a.changes[k]
		x := c.wx - baseChunk*CW
		if x < 0 || x >= W {
			continue
		}
		l := layerOf(c.layer)
		i := c.y*W + x
		if l[i] != c.now { // changed since (mined, painted over...): leave it
			continue
		}
		if c.take != "" {
			if countItem(c.take) == 0 {
				continue
			}
			removeItem(c.take, 1)
		}
		l[i] = c.old
		if c.give != "" {
			giveItem(c.give, 1)
		}
		markDirty(x)
		n++
	}
	if n > 0 {
		sfx("dig")
		toast("Undone")
	} else {
		toast("Nothing left to undo there")
	}
}

// Apply the held item to these cells; returns how many changed
func applyCells(cells [][2]int) int {
	kind := heldBuildKind()
	n := 0
	for _, c := range cells {
		x, y := c[0], c[1]
		if x < 0 || x >= W || y < 0 || y >= H || !cellInBuildReach(x, y) || selected < 0 || inv[selected] == nil {
			continue
		}
		i := y*W + x
		it := items[inv[selected].ID]
		switch kind {
		case buildBlock:
			if !isReplaceable(grid[i]) || (grid[i] == WATER && it.place == WATER) {
				continue
			}
			if paint[i] != 0 {
				record(cellChange{y: y, layer: 'p', old: paint[i], now: 0}, x)
			}
			record(cellChange{y: y, layer: 'g', old: grid[i], now: it.place, give: it.id}, x)
			grid[i] = it.place
			shade[i] = int8(hash32(x+baseChunk*CW, y, 9)%21) - 10
			paint[i] = 0
			takeSelected(1)
			n++
		case buildWall:
			if wall[i] == it.wall {
				continue
			}
			if wpaint[i] != 0 {
				record(cellChange{y: y, layer: 'q', old: wpaint[i], now: 0}, x)
			}
			ch := cellChange{y: y, layer: 'w', old: wall[i], now: it.wall, give: it.id}
			if wall[i] != 0 { // swap walls: the old one comes back
				giveItem(wallItem[wall[i]], 1)
				ch.take = wallItem[wall[i]]
			}
			record(ch, x)
			wall[i] = it.wall
			wpaint[i] = 0
			takeSelected(1)
			n++
		case buildPaint:
			if grid[i] != EMPTY && grid[i] != WATER && grid[i] != TORCH {
				if paint[i] != uint8(paintIdx) {
					record(cellChange{y: y, layer: 'p', old: paint[i], now: uint8(paintIdx)}, x)
					paint[i] = uint8(paintIdx)
					n++
				}
			} else if wall[i] != 0 && wpaint[i] != uint8(paintIdx) {
				record(cellChange{y: y, layer: 'q', old: wpaint[i], now: uint8(paintIdx)}, x)
				wpaint[i] = uint8(paintIdx)
				n++
			}
		case buildHammer:
			if wall[i] != 0 {
				if wpaint[i] != 0 {
					record(cellChange{y: y, layer: 'q', old: wpaint[i], now: 0}, x)
				}
				record(cellChange{y: y, layer: 'w', old: wall[i], now: 0, take: wallItem[wall[i]]}, x)
				giveItem(wallItem[wall[i]], 1)
				wall[i], wpaint[i] = 0, 0
				n++
			}
		}
		markDirty(x)
	}
	return n
}

func markDirty(x int) {
	if s := x / CW; s >= 0 && s < NCH {
		dirty[s] = true
	}
}

// The brush: every frame while held, apply in a circle around the aim
func brushCells() [][2]int {
	var out [][2]int
	r := brushSize
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r+r/2 {
				out = append(out, [2]int{aimX + dx, aimY + dy})
			}
		}
	}
	return out
}

// ---- Keyboard ----

func buildKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		undoBuild()
	}
	k := heldBuildKind()
	if k == buildNone {
		return
	}
	if usesShapes(k) {
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			buildShape = (buildShape + 1) % len(shapeNames)
			toast("Shape: " + shapeNames[buildShape])
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
			brushSize = max(0, brushSize-1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
			brushSize = min(8, brushSize+1)
		}
	}
	if k == buildPaint && inpututil.IsKeyJustPressed(ebiten.KeyC) {
		paintIdx = (paintIdx + 1) % len(paintRGB)
		toast("Paint: " + paintNames[paintIdx])
	}
	if k == buildOneShot {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) && heldBlueprint() != "" {
			buildRot = (buildRot + 1) % 4
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			buildFlip = !buildFlip
		}
	}
}

// ---- Build bar (above the hotbar) ----

var buildPaletteOpen bool

func buildBar(hotbarTop float64) {
	k := heldBuildKind()
	if k == buildNone {
		buildPaletteOpen = false
		return
	}
	bh := 30.0
	if isTouch {
		bh = 34
	}
	type btn struct {
		label string
		on    bool
		act   func()
	}
	var btns []btn
	if usesShapes(k) {
		for s := range shapeNames {
			s := s
			btns = append(btns, btn{shapeNames[s], buildShape == s, func() { buildShape = s }})
		}
		btns = append(btns, btn{"-", false, func() { brushSize = max(0, brushSize-1) }})
		btns = append(btns, btn{sizeLabel(), false, nil})
		btns = append(btns, btn{"+", false, func() { brushSize = min(8, brushSize+1) }})
		if len(undoStack) > 0 {
			btns = append(btns, btn{"Undo", false, undoBuild})
		}
	}
	if k == buildPaint {
		btns = append(btns, btn{"Colour", buildPaletteOpen, func() { buildPaletteOpen = !buildPaletteOpen }})
	}
	if k == buildOneShot {
		if heldBlueprint() != "" {
			btns = append(btns, btn{"Rotate", false, func() { buildRot = (buildRot + 1) % 4 }})
		}
		btns = append(btns, btn{"Flip", buildFlip, func() { buildFlip = !buildFlip }})
		if isTouch && holoSet {
			btns = append(btns, btn{"Build", true, func() { buildHoloNow() }})
		}
	}
	// lay out centred
	widths := make([]float64, len(btns))
	total := 0.0
	for i, b := range btns {
		widths[i] = math.Max(bh, textWidth(b.label, 13, false)+20)
		total += widths[i] + 6
	}
	x := (logicalW - total) / 2
	y := hotbarTop - bh - 16
	for i, b := range btns {
		r := Rect{x, y, widths[i], bh}
		hudRects = append(hudRects, r)
		st := styleSlate
		if b.on {
			st = styleViolet
		}
		if b.act == nil {
			Inset(r, colPanelLo)
			Text(b.label, r.X+r.W/2, r.Y+r.H/2, 13, colInk, text.AlignCenter)
		} else if Button(r, b.label, st, true, nil) {
			b.act()
		}
		x += widths[i] + 6
	}
	// paint swatch next to the bar
	if k == buildPaint {
		sw := Rect{x + 4, y, bh, bh}
		pc := paintRGB[paintIdx]
		ops = append(ops, func(dst *ebiten.Image) {
			bevel(dst, sw, color.NRGBA{uint8(pc[0]), uint8(pc[1]), uint8(pc[2]), 255}, colInk, colO, false)
			if paintIdx == 0 {
				fill(dst, sw.X+4, sw.Y+sw.H/2-2, sw.W-8, 4, colDanger)
			}
		})
		if buildPaletteOpen {
			palette(y - 12)
		}
	}
}

func sizeLabel() string {
	if buildShape == shapeBrush {
		return "Size " + itoa(brushSize+1)
	}
	return "Thick " + itoa(brushSize/2+1)
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// Pick a paint colour
func palette(bottom float64) {
	s := 30.0
	if isTouch {
		s = 34
	}
	n := len(paintRGB)
	total := float64(n)*(s+6) - 6
	x := (logicalW - total) / 2
	y := bottom - s - 6
	bg := Rect{x - 10, y - 10, total + 20, s + 20}
	hudRects = append(hudRects, bg)
	Panel(bg)
	for i := 0; i < n; i++ {
		r := Rect{x + float64(i)*(s+6), y, s, s}
		pc := paintRGB[i]
		sel := i == paintIdx
		clicked := ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY)
		ops = append(ops, func(dst *ebiten.Image) {
			body := color.Color(color.NRGBA{uint8(pc[0]), uint8(pc[1]), uint8(pc[2]), 255})
			if i == 0 {
				body = colPanelLo
			}
			hi := color.Color(colO)
			if sel {
				hi = colT
			}
			bevel(dst, r, body, hi, hi, false)
			if i == 0 {
				fill(dst, r.X+5, r.Y+r.H/2-2, r.W-10, 4, colDanger)
			}
		})
		if clicked {
			paintIdx = i
			buildPaletteOpen = false
			sfx("click")
			toast("Paint: " + paintNames[i])
		}
	}
}

// ---- Holograms for shapes and the brush ----

func drawBuildHologram(blend func(x, y int, c [3]int, a float64), ox, oy int) {
	if cur != scPlaying {
		return
	}
	k := heldBuildKind()
	if !usesShapes(k) {
		return
	}
	var cells [][2]int
	switch {
	case shapeOn:
		cells = shapeCells
	case buildShape == shapeBrush && (useActive || !isTouch):
		if !isTouch && !useActive {
			aimSX, aimSY = hoverX, hoverY
		}
		aimCell()
		cells = brushCells()
	case !isTouch:
		x, y := int(math.Floor((cam.x+hoverX/zoom)/CELL)), int(math.Floor((cam.y+hoverY/zoom)/CELL))
		cells = [][2]int{{x, y}}
	}
	if len(cells) == 0 {
		return
	}
	it := items[inv[selected].ID]
	c := [3]int{120, 255, 160}
	switch k {
	case buildBlock:
		c = mix(blockRGB[it.place], c, 0.35)
	case buildWall:
		c = mix(shadeRGB(blockRGB[it.wall], 0.6), c, 0.35)
	case buildPaint:
		c = paintRGB[paintIdx]
		if paintIdx == 0 {
			c = [3]int{255, 110, 110}
		}
	case buildHammer:
		c = [3]int{255, 170, 120}
	}
	a := 0.4 + 0.12*math.Sin(float64(frame)*0.15)
	left := inv[selected].N
	for _, p := range cells {
		col := c
		if !cellInBuildReach(p[0], p[1]) {
			col = [3]int{255, 110, 110}
		} else if (k == buildBlock || k == buildWall) && left <= 0 {
			col = [3]int{255, 110, 110}
		} else if k == buildBlock || k == buildWall {
			left--
		}
		blend(p[0]-ox, p[1]-oy, col, a)
	}
}
