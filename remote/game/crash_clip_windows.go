//go:build windows

package game

import (
	"strings"
	"syscall"
	"unsafe"
)

// The crash screen's own clipboard (kept separate from the game's, so it works even
// in games without copy support)
var (
	crashUser32   = syscall.NewLazyDLL("user32.dll")
	crashKernel32 = syscall.NewLazyDLL("kernel32.dll")
)

func crashClipboard(s string) bool {
	u, err := syscall.UTF16FromString(strings.ReplaceAll(s, "\n", "\r\n"))
	if err != nil {
		return false
	}
	if r, _, _ := crashUser32.NewProc("OpenClipboard").Call(0); r == 0 {
		return false
	}
	defer crashUser32.NewProc("CloseClipboard").Call()
	crashUser32.NewProc("EmptyClipboard").Call()
	size := uintptr(len(u) * 2)
	h, _, _ := crashKernel32.NewProc("GlobalAlloc").Call(0x0002, size) // GMEM_MOVEABLE
	if h == 0 {
		return false
	}
	p, _, _ := crashKernel32.NewProc("GlobalLock").Call(h)
	if p == 0 {
		return false
	}
	crashKernel32.NewProc("RtlMoveMemory").Call(p, uintptr(unsafe.Pointer(&u[0])), size)
	crashKernel32.NewProc("GlobalUnlock").Call(h)
	r, _, _ := crashUser32.NewProc("SetClipboardData").Call(13, h) // CF_UNICODETEXT
	return r != 0
}
