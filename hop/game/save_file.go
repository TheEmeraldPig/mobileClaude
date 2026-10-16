//go:build !js

package game

import (
	"os"
	"path/filepath"
)

func savePath() string { return filepath.Join(DataDir, "hop_save.json") }

func readSave() ([]byte, error) { return os.ReadFile(savePath()) }

func writeSave(b []byte) {
	tmp := savePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, savePath())
	}
}
