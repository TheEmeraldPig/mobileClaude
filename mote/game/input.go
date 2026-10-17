package game

import (
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input: keyboard, mouse and touch, turned into game actions.
// Each pointer (the mouse, or one finger) gets a role when it goes down:
// the UI, the joystick, or the world (use an item / look around).

var isTouch = runtime.GOOS == "android" || runtime.GOOS == "ios"

type gameInput struct{ left, right, lev bool }

var input gameInput

type stickState struct {
	x      float64
	up     bool
	id     int // touch id, -2 when free
	moving bool
	hold   int // frames held still
	travel float64
	kx, ky float64 // knob offset
}

var stick = stickState{id: -2}

// The UI pointer for this frame
type uiPointer struct {
	x, y           float64
	startX, startY float64
	down, pressed  bool
	released       bool
	consumed       bool
	id             int
	active         bool
}

var ptr uiPointer

type pointerRole int

const (
	roleNone pointerRole = iota
	roleUI
	roleStick
	roleWorld
)

type pointerInfo struct {
	role         pointerRole
	x, y, sx, sy float64
}

var pointers = map[int]*pointerInfo{}

// HUD areas from the last frame, used to decide what a new touch is for
var hudRects []Rect

const mouseID = -1

func keyByName(name string) (ebiten.Key, bool) {
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if k.String() == name {
			return k, true
		}
	}
	return 0, false
}

func bindDown(act string) bool {
	if k, ok := keyByName(prefs.Binds[act]); ok && ebiten.IsKeyPressed(k) {
		return true
	}
	switch act {
	case "left":
		return ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	case "right":
		return ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	case "lev":
		return ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeySpace)
	}
	return false
}

func bindJustPressed(act string) bool {
	k, ok := keyByName(prefs.Binds[act])
	return ok && inpututil.IsKeyJustPressed(k)
}

func keyLabel(name string) string {
	switch name {
	case "ArrowLeft":
		return "Left"
	case "ArrowRight":
		return "Right"
	case "ArrowUp":
		return "Up"
	case "ArrowDown":
		return "Down"
	}
	if len(name) > 5 && name[:5] == "Digit" {
		return name[5:]
	}
	return name
}

func stickCenter() (float64, float64) { return prefs.StickX * logicalW, prefs.StickY * logicalH }

const stickR = 56.0

// Gather pointers, assign roles to new ones, and fill ptr for the UI.
func readPointers(menuOpen bool) {
	type raw struct {
		id      int
		x, y    float64
		pressed bool
	}
	var cur []raw
	if !isTouch {
		mx, my := ebiten.CursorPosition()
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			cur = append(cur, raw{mouseID, float64(mx) / uiScale, float64(my) / uiScale, inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)})
		}
		hoverX, hoverY = float64(mx)/uiScale, float64(my)/uiScale
	}
	for _, id := range ebiten.AppendTouchIDs(nil) {
		tx, ty := ebiten.TouchPosition(id)
		cur = append(cur, raw{int(id), float64(tx) / uiScale, float64(ty) / uiScale, inpututil.TouchPressDuration(id) == 1})
		isTouch = true
	}

	ptr.pressed, ptr.released, ptr.consumed = false, false, false
	seen := map[int]bool{}
	for _, r := range cur {
		seen[r.id] = true
		p := pointers[r.id]
		if p == nil {
			p = &pointerInfo{sx: r.x, sy: r.y}
			pointers[r.id] = p
			p.role = roleWorld
			cx, cy := stickCenter()
			switch {
			case menuOpen:
				p.role = roleUI
			case isTouch && stick.id == -2 && math.Hypot(r.x-cx, r.y-cy) <= stickR+8:
				p.role = roleStick
			default:
				for _, h := range hudRects {
					if h.Has(r.x, r.y) {
						p.role = roleUI
						break
					}
				}
			}
			switch p.role {
			case roleUI:
				if !ptr.active {
					ptr = uiPointer{x: r.x, y: r.y, startX: r.x, startY: r.y, down: true, pressed: true, id: r.id, active: true}
				}
			case roleStick:
				stick.id, stick.travel, stick.hold, stick.moving = r.id, 0, 0, false
			case roleWorld:
				startWorldPointer(r.id, r.x, r.y)
			}
		}
		p.x, p.y = r.x, r.y
		switch p.role {
		case roleUI:
			if ptr.active && ptr.id == r.id {
				ptr.x, ptr.y, ptr.down = r.x, r.y, true
			}
		case roleStick:
			moveStick(r.x, r.y)
		case roleWorld:
			moveWorldPointer(r.id, r.x, r.y)
		}
	}
	for id, p := range pointers {
		if seen[id] {
			continue
		}
		switch p.role {
		case roleUI:
			if ptr.active && ptr.id == id {
				ptr.released, ptr.down, ptr.active = true, false, false
			}
		case roleStick:
			releaseStick()
		case roleWorld:
			endWorldPointer(id)
		}
		delete(pointers, id)
	}
}

var hoverX, hoverY float64

func moveStick(x, y float64) {
	cx, cy := stickCenter()
	dx, dy := x-cx, y-cy
	stick.travel = math.Max(stick.travel, math.Hypot(dx, dy))
	if !stick.moving {
		stick.hold++
		if stick.hold > 36 && stick.travel < 12 { // held still ~0.6s: start dragging the stick
			stick.moving = true
			sfx("click")
		}
	}
	if stick.moving {
		prefs.StickX = math.Min(0.95, math.Max(0.05, x/logicalW))
		prefs.StickY = math.Min(0.95, math.Max(0.1, y/logicalH))
		stick.x, stick.up, stick.kx, stick.ky = 0, false, 0, 0
		return
	}
	l := math.Hypot(dx, dy)
	if l > stickR {
		dx, dy = dx*stickR/l, dy*stickR/l
	}
	stick.kx, stick.ky = dx, dy
	stick.x = dx / stickR
	if math.Abs(stick.x) < 0.15 {
		stick.x = 0
	}
	stick.up = dy/stickR < -0.5
}

func releaseStick() {
	if stick.moving {
		savePrefs()
	}
	stick = stickState{id: -2}
}

// ---- World pointer: use the selected item, or look around ----

var (
	peekID         = -99
	useID          = -99
	peekX, peekY   float64 // target, screen px
	peekSX, peekSY float64 // smoothed
)

// A quick tap on a station opens it (while a pickaxe held on it mines it)
var tapObj struct {
	id     int
	o      *Obj
	frame  int
	sx, sy float64
}

func startWorldPointer(id int, x, y float64) {
	// tapping a chest within reach opens it, whatever you are holding
	aimSX, aimSY = x, y
	inReach := aimCell()
	if inReach && aimX >= 0 && aimX < W && aimY >= 0 && aimY < H && grid[aimY*W+aimX] == CHEST {
		openChest(aimX, aimY)
		return
	}
	if inReach && aimX >= 0 && aimX < W && aimY >= 0 && aimY < H && isDoor(grid[aimY*W+aimX]) &&
		(selected < 0 || inv[selected] == nil || !isPick(inv[selected].ID)) {
		toggleDoor(aimX, aimY)
		return
	}
	if t := traderAt(aimX, aimY, true); t != nil && (selected < 0 || inv[selected] == nil || !isPick(inv[selected].ID)) {
		openShop(t.house.kind)
		return
	}
	tapObj.o = nil
	if o := objAt(aimX, aimY); o != nil && objInReach(o) {
		if selected < 0 || inv[selected] == nil || !isPick(inv[selected].ID) {
			openStation(o.kind)
			return
		}
		tapObj.id, tapObj.o, tapObj.frame, tapObj.sx, tapObj.sy = id, o, frame, x, y
	}
	if selected >= 0 && inv[selected] != nil && useID == -99 {
		useID, useActive = id, true
		aimSX, aimSY = x, y
		if items[inv[selected].ID].once {
			useOnce()
		} else if usesShapes(heldBuildKind()) {
			beginAction()
			if buildShape != shapeBrush {
				startShape()
			}
		}
		return
	}
	if peekID == -99 {
		peekID = id
		setPeek(x, y)
	}
}

func moveWorldPointer(id int, x, y float64) {
	if id == peekID {
		setPeek(x, y)
	}
	if id == useID {
		aimSX, aimSY = x, y
		updateShape()
	}
}

func endWorldPointer(id int) {
	if id == peekID {
		peekID, peekX, peekY = -99, 0, 0
	}
	if id == useID {
		finishShape()
		endAction()
		useID, useActive = -99, false
	}
	if tapObj.o != nil && tapObj.id == id {
		o := tapObj.o
		tapObj.o = nil
		if frame-tapObj.frame < 14 && math.Hypot(aimSX-tapObj.sx, aimSY-tapObj.sy) < 12 && objStillThere(o) {
			openStation(o.kind)
		}
	}
}

func objStillThere(o *Obj) bool {
	for _, p := range objs {
		if p == o {
			return true
		}
	}
	return false
}

func setPeek(x, y float64) {
	peekX = math.Max(-logicalW/2, math.Min(logicalW/2, x-logicalW/2))
	peekY = math.Max(-logicalH/2, math.Min(logicalH/2, y-logicalH/2))
}

func clearPointers() {
	for id := range pointers {
		delete(pointers, id)
	}
	ptr = uiPointer{}
	releaseStick()
	peekID, useID, useActive, peekX, peekY = -99, -99, false, 0, 0
	shapeOn, shapeCells = false, nil
	tapObj.o = nil
}
