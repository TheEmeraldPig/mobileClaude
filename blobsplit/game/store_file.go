//go:build !js

package game

import "os"

// The save is a file (Android app storage, or the user config folder on a PC).

func readSave() ([]byte, error) { return os.ReadFile(savePath()) }

func writeSave(b []byte) {
	_ = os.MkdirAll(dataDir(), 0o755)
	tmp := savePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, savePath())
	}
}
