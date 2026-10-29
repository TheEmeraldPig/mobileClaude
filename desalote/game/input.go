package game

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
)

// Input: the mouse or one finger drives the UI (ptr). Everything is menus and cards, so
// there are no other pointer roles.

var isTouch = runtime.GOOS == "android" || runtime.GOOS == "ios"

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

const mouseID = -1

var hoverX, hoverY float64

func readPointers() {
	ptr.pressed, ptr.released, ptr.consumed = false, false, false
	if !isTouch {
		mx, my := ebiten.CursorPosition()
		hoverX, hoverY = float64(mx)/uiScale, float64(my)/uiScale
	}
	ids := ebiten.AppendTouchIDs(nil)
	if len(ids) > 0 {
		isTouch = true
	}
	// the pointer that drives the UI: the first finger, or the left mouse button
	down, x, y, id := false, 0.0, 0.0, 0
	if len(ids) > 0 {
		tx, ty := ebiten.TouchPosition(ids[0])
		down, x, y, id = true, float64(tx)/uiScale, float64(ty)/uiScale, int(ids[0])
		if ptr.active && ptr.id != id { // the old finger lifted and a new one is down
			ptr.released, ptr.active, ptr.down = true, false, false
			return
		}
	} else if !isTouch && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		down, x, y, id = true, float64(mx)/uiScale, float64(my)/uiScale, mouseID
	}
	if scriptTap.on {
		down, x, y, id = scriptTap.down, scriptTap.x, scriptTap.y, -7
	}
	switch {
	case down && !ptr.active:
		ptr = uiPointer{x: x, y: y, startX: x, startY: y, down: true, pressed: true, id: id, active: true}
	case down:
		ptr.x, ptr.y = x, y
	case ptr.active:
		ptr.released, ptr.down, ptr.active = true, false, false
	}
}

func clearPointers() { ptr = uiPointer{} }
