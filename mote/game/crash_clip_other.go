//go:build !windows

package game

import (
	"os/exec"
	"strings"
)

// The crash screen's clipboard on Mac and Linux: hand the text to the system's
// clipboard tool, whichever is installed (phones have none; the report is in crash.txt)
func crashClipboard(s string) bool {
	for _, c := range [][]string{{"pbcopy"}, {"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(s)
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}
