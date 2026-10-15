//go:build windows

package game

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
	moveMemory       = kernel32.NewProc("RtlMoveMemory")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

func canCopy() bool { return true }

// copyText puts text on the Windows clipboard.
func copyText(s string) bool {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return false
	}
	if r, _, _ := openClipboard.Call(0); r == 0 {
		return false
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	size := uintptr(len(u) * 2)
	h, _, _ := globalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return false
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return false
	}
	moveMemory.Call(p, uintptr(unsafe.Pointer(&u[0])), size)
	globalUnlock.Call(h)
	r, _, _ := setClipboardData.Call(cfUnicodeText, h)
	return r != 0
}
