//go:build windows && !steam

package game

// Built-in updater for the Windows .exe (not used in Steam builds, where Steam updates the game).
// Every minute it asks GitHub for the newest Remote release. If it is newer than this build,
// it downloads the new Remote.exe, saves the game, swaps the files and restarts into the new
// version with --resume, so you carry on where you were.

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Mote shares this repository and owns its "latest" release, so Remote lists the releases
// and only looks at its own tags (remote-build-N).
const (
	releaseAPI = "https://api.github.com/repos/TheEmeraldPig/mobileClaude/releases?per_page=30"
	tagPrefix  = "remote-build-"
)

var (
	updateReady atomic.Value // path of the downloaded exe
	updateStart bool
	restartAt   int
)

func buildNumber(s string) int {
	if !strings.HasPrefix(s, tagPrefix) {
		return -1
	}
	s = strings.TrimPrefix(s, tagPrefix)
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}

func checkForUpdate() {
	if Version == "dev" {
		return
	}
	if !updateStart {
		updateStart = true
		go func() {
			for {
				pollRelease()
				time.Sleep(60 * time.Second)
			}
		}()
	}
	if p, ok := updateReady.Load().(string); ok && p != "" && restartAt == 0 {
		toast("New version, updating...")
		restartAt = frame + 90
	}
	if restartAt > 0 && frame >= restartAt {
		applyUpdate(updateReady.Load().(string))
	}
}

func pollRelease() {
	if p, _ := updateReady.Load().(string); p != "" {
		return
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(releaseAPI)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	type release struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	var rels []release
	if json.NewDecoder(resp.Body).Decode(&rels) != nil {
		return
	}
	var rel release
	for _, r := range rels {
		if buildNumber(r.Tag) > buildNumber(rel.Tag) {
			rel = r
		}
	}
	if buildNumber(rel.Tag) <= buildNumber(tagPrefix+Version) {
		return
	}
	for _, a := range rel.Assets {
		if a.Name != "Remote.exe" {
			continue
		}
		exe, err := os.Executable()
		if err != nil {
			return
		}
		r, err := client.Get(a.URL)
		if err != nil {
			return
		}
		defer r.Body.Close()
		tmp := exe + ".new"
		f, err := os.Create(tmp)
		if err != nil {
			return
		}
		_, err = io.Copy(f, r.Body)
		f.Close()
		if err == nil {
			updateReady.Store(tmp)
		}
		return
	}
}

func applyUpdate(newExe string) {
	saveGame()
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	os.Remove(old)
	if os.Rename(exe, old) != nil { // a running exe can be renamed on Windows
		return
	}
	if os.Rename(newExe, exe) != nil {
		os.Rename(old, exe)
		return
	}
	cmd := exec.Command(exe, "--resume")
	if cmd.Start() == nil {
		os.Exit(0)
	}
}

func init() {
	// clean up after the previous update
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
	}
}
