//go:build !windows || steam

package game

// No built-in updater here: Steam builds are updated by Steam, and other platforms
// (Android, Linux, macOS) get updates from their store or a new download.
func checkForUpdate() {}
