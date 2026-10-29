//go:build windows && !steam

package game

// Built-in updater for the Windows .exe (not used in Steam builds).
// Every minute it asks GitHub for the newest Desalote release. If it is newer than this
// build, it downloads the new Desalote.exe, swaps the files and restarts into the new
// version (only while no match is being played).
//
// This repository also releases Mote, whose releases are the repository's "latest". So
// this looks through the list of releases for tags starting with desalote-build-.

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

const (
	releaseAPI = "https://api.github.com/repos/TheEmeraldPig/mobileClaude/releases?per_page=30"
	tagPrefix  = "desalote-build-"
)

var (
	updateReady atomic.Value // path of the downloaded exe
	updateStart bool
	restartAt   int
)

func buildNumber(s string) int {
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
	if p, ok := updateReady.Load().(string); ok && p != "" && restartAt == 0 && !inMatch() {
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
	var rels []struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.NewDecoder(resp.Body).Decode(&rels) != nil {
		return
	}
	best := -1
	for i, r := range rels {
		if strings.HasPrefix(r.Tag, tagPrefix) && (best < 0 || buildNumber(r.Tag) > buildNumber(rels[best].Tag)) {
			best = i
		}
	}
	if best < 0 || buildNumber(rels[best].Tag) <= buildNumber(Version) {
		return
	}
	rel := rels[best]
	for _, a := range rel.Assets {
		if a.Name != "Desalote.exe" {
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
	savePrefs()
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
