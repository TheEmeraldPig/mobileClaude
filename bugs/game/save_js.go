//go:build js

package game

import (
	"errors"
	"syscall/js"
)

// The web build keeps its save in the browser's localStorage.
const saveKey = "bugbrigade_save"

func readSave() (b []byte, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("no localStorage")
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", saveKey)
	if v.IsNull() || v.IsUndefined() {
		return nil, errors.New("no save")
	}
	return []byte(v.String()), nil
}

func writeSave(b []byte) {
	defer func() { recover() }()
	js.Global().Get("localStorage").Call("setItem", saveKey, string(b))
}
