package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Storage chests: a placeable chest with 24 slots. Tap (or click) a stack to move it
// to the other side; the buttons move everything at once.

var chestObj *Obj

func openStorage() {
	for _, o := range objs {
		if o.kind == "storage" && objInReach(o) {
			if o.items == nil {
				o.items = make([]*Stack, storageSize)
			}
			chestObj = o
			openScreen(scChest)
			sfx("chest")
			return
		}
	}
}

// Move a stack into a list of slots, merging first; returns what didn't fit
func moveInto(dst []*Stack, s *Stack) int {
	left := s.N
	if !items[s.ID].tool {
		for _, d := range dst {
			if d != nil && d.ID == s.ID && !d.Temp && d.N < stackMax {
				k := min(left, stackMax-d.N)
				d.N += k
				left -= k
			}
		}
	}
	for i := range dst {
		if left == 0 {
			break
		}
		if dst[i] == nil {
			dst[i] = &Stack{ID: s.ID, N: left}
			left = 0
		}
	}
	return left
}

func storageMenu() {
	o := chestObj
	if o == nil || !objStillThere(o) || !objInReach(o) {
		play()
		return
	}
	r := sheet(940, 460)
	if _, cl := head(r, "STORAGE CHEST", false); cl {
		play()
		return
	}
	gap := 7.0
	top := r.Y + 76
	colW := (r.W - 36 - 24) / 2
	// chest: 6 x 4, inventory: 8 x 5
	slotC := math.Min(56, math.Min((colW-5*gap)/6, (r.H-150-3*gap)/4))
	slotI := math.Min(52, math.Min((colW-7*gap)/8, (r.H-150-4*gap)/5))
	cx, ix := r.X+18, r.X+18+colW+24
	Label("CHEST", cx, top-10)
	Label("YOUR ITEMS", ix, top-10)
	moved := ""
	for i := 0; i < storageSize; i++ {
		sr := Rect{cx + float64(i%6)*(slotC+gap), top + float64(i/6)*(slotC+gap), slotC, slotC}
		if stackSlot(sr, o.items[i]) && o.items[i] != nil {
			s := o.items[i]
			left := moveInto(inv, s)
			if left == s.N {
				moved = "Your bag is full."
			} else {
				moved = fmt.Sprintf("Took %d %s.", s.N-left, items[s.ID].name)
				if left == 0 {
					o.items[i] = nil
				} else {
					s.N = left
				}
				sfx("pickup")
			}
		}
	}
	for i := 0; i < invSize; i++ {
		sr := Rect{ix + float64(i%8)*(slotI+gap), top + float64(i/8)*(slotI+gap), slotI, slotI}
		if stackSlot(sr, inv[i]) && inv[i] != nil {
			s := inv[i]
			if s.Temp {
				moved = "The temporary kit can't be stored."
				continue
			}
			left := moveInto(o.items, s)
			if left == s.N {
				moved = "The chest is full."
			} else {
				moved = fmt.Sprintf("Stored %d %s.", s.N-left, items[s.ID].name)
				if left == 0 {
					inv[i] = nil
					if selected == i {
						selected = -1
					}
				} else {
					s.N = left
				}
				sfx("place")
			}
		}
	}
	by := r.Y + r.H - 52
	if Button(Rect{cx, by, 150, 36}, "Take all", styleViolet, true, nil) {
		for i, s := range o.items {
			if s == nil {
				continue
			}
			if left := moveInto(inv, s); left == 0 {
				o.items[i] = nil
			} else {
				s.N = left
			}
		}
		sfx("pickup")
	}
	if Button(Rect{ix, by, 190, 36}, "Store backpack", styleSlate, true, nil) {
		for i := hotbarN; i < invSize; i++ {
			s := inv[i]
			if s == nil || s.Temp {
				continue
			}
			if left := moveInto(o.items, s); left == 0 {
				inv[i] = nil
			} else {
				s.N = left
			}
		}
		sfx("place")
	}
	if moved != "" {
		chestMsg, chestMsgAt = moved, frame
	}
	if chestMsg != "" && frame-chestMsgAt < 150 {
		Text(chestMsg, r.X+r.W-18, by+18, 12, colT, text.AlignEnd)
	} else {
		Text("Tap a stack to move it across.", r.X+r.W-18, by+18, 12, colMuted, text.AlignEnd)
	}
}

var (
	chestMsg   string
	chestMsgAt int
)

// A slot showing a stack; true when tapped
func stackSlot(r Rect, s *Stack) bool {
	clicked := ptr.released && r.Has(ptr.x, ptr.y) && r.Has(ptr.startX, ptr.startY)
	hover := !isTouch && r.Has(hoverX, hoverY)
	ops = append(ops, func(dst *ebiten.Image) {
		body := colPanelLo
		if hover {
			body = rgb(0x1c1838)
		}
		bevel(dst, r, body, rgb(0x0a0817), colPanelHi, false)
		if s == nil {
			return
		}
		isz := r.W * 0.62
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(isz/8*uiScale, isz/8*uiScale)
		op.GeoM.Translate((r.X+(r.W-isz)/2)*uiScale, (r.Y+(r.H-isz)/2)*uiScale)
		if s.Temp {
			op.ColorScale.Scale(0.75, 0.85, 1, 0.65)
		}
		dst.DrawImage(items[s.ID].img, op)
		if !items[s.ID].tool {
			drawText(dst, fmt.Sprint(s.N), r.X+r.W-4, r.Y+r.H-8, textOpt{size: 12, align: text.AlignEnd, shadow: true})
		}
	})
	return clicked
}
