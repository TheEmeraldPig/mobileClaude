//go:build !windows

package game

// No clipboard support here yet; the seed is shown on screen so it can be written down.
func canCopy() bool          { return false }
func copyText(s string) bool { return false }
