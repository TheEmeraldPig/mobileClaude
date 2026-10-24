//go:build js

package game

import (
	"errors"
	"syscall/js"
)

// In a browser (the web build) the save lives in localStorage. It can be missing or
// refuse to work (private windows, blocked site data), so every call is guarded and
// the game just starts fresh without it.

const webKey = "biote-save"

func readSave() (b []byte, err error) {
	defer func() {
		if recover() != nil {
			b, err = nil, errors.New("no storage")
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", webKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, errors.New("no save")
	}
	return []byte(v.String()), nil
}

func writeSave(b []byte) {
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", webKey, string(b))
}
