package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Multiplayer menus: the Multiplayer screen (your name, joining with a code, hosting
// one of your worlds), and the multiplayer part of the pause menu.

var (
	codeEntry     string
	mpField       int  // which field the keyboard types into: 0 code, 1 name, 2 server
	hostAfterOpen bool // the next world opened from the Worlds list is opened to friends
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Give this device a name and an ID the first time
func ensureIdentity() {
	changed := false
	if strings.TrimSpace(prefs.Name) == "" {
		prefs.Name = randomPlayerName()
		changed = true
	}
	if prefs.PlayerID == "" {
		b := make([]byte, 8)
		rand.Read(b)
		prefs.PlayerID = hex.EncodeToString(b)
		changed = true
	}
	if changed {
		savePrefs()
	}
}

func openMultiplayer() {
	ensureIdentity()
	if netRole != netGuest {
		netStatus = ""
	}
	mpField = 0
	openScreen(scMultiplayer)
}

func multiplayerMenu() {
	r := sheet(660, 400)
	if _, cl := head(r, "MULTIPLAYER", false); cl {
		if netRole == netGuest && netMyID < 0 {
			leaveMultiplayer()
		}
		openScreen(scMain)
		return
	}
	x := r.X + 18
	w := r.W - 36
	y := r.Y + 62
	// your name
	Label("YOUR NAME", x, y+6)
	nf := Rect{x + 110, y - 8, w - 110 - 104, 32}
	Inset(nf, colPanelLo)
	caret := ""
	if mpField == 1 && !isTouch && (frame/30)%2 == 0 {
		caret = "_"
	}
	Text(playerName()+caret, nf.X+10, nf.Y+16, 15, colInk, text.AlignStart)
	if ptr.released && nf.Has(ptr.x, ptr.y) {
		mpField = 1
	}
	if Button(Rect{nf.X + nf.W + 10, nf.Y, 94, 32}, "Shuffle", styleSlate, true, nil) {
		prefs.Name = randomPlayerName()
		savePrefs()
	}
	y += 50
	// join with a code
	Label("JOIN A FRIEND", x, y)
	y += 16
	cf := Rect{x, y, 190, 40}
	Inset(cf, colPanelLo)
	shown := codeEntry
	if mpField == 0 && !isTouch && (frame/30)%2 == 0 && len(codeEntry) < 5 {
		shown += "_"
	}
	if codeEntry == "" && mpField != 0 {
		Text("Code", cf.X+12, cf.Y+20, 18, colMuted, text.AlignStart)
	}
	Text(shown, cf.X+12, cf.Y+20, 22, colT, text.AlignStart)
	if ptr.released && cf.Has(ptr.x, ptr.y) {
		mpField = 0
	}
	busy := netRole == netGuest
	if Button(Rect{cf.X + cf.W + 12, y, 120, 40}, "Join", styleGold, len(codeEntry) == 5 && !busy, icoPlay) {
		joinWorld(codeEntry)
	}
	if busy && Button(Rect{cf.X + cf.W + 144, y, 100, 40}, "Cancel", styleSlate, true, nil) {
		leaveMultiplayer()
		openScreen(scMultiplayer)
	}
	y += 50
	// phones: letter keys for the code
	if isTouch {
		cols := 16
		kw := (w - float64(cols-1)*4) / float64(cols)
		for i, ch := range codeAlphabet + "<" {
			kr := Rect{x + float64(i%cols)*(kw+4), y + float64(i/cols)*34, kw, 30}
			if i == len(codeAlphabet) {
				kr = Rect{x + w - kw*2 - 4, y + 2*34, kw*2 + 4, 30}
			}
			if Button(kr, string(ch), styleSlate, true, nil) {
				if ch == '<' {
					if len(codeEntry) > 0 {
						codeEntry = codeEntry[:len(codeEntry)-1]
					}
				} else if len(codeEntry) < 5 {
					codeEntry += string(ch)
				}
			}
		}
		y += 3*34 + 4
	}
	// host
	Label("HOST", x, y+4)
	y += 18
	if Button(Rect{x, y, 250, 36}, "Open one of my worlds", styleViolet, !busy, icoWorld) {
		hostAfterOpen = true
		openScreen(scWorlds)
	}
	Text("It opens to friends and shows a code to share.", x+262, y+18, 12, colMuted, text.AlignStart)
	y += 46
	// what's happening
	if netStatus != "" {
		Text(netStatus, x, y+8, 14, colT, text.AlignStart)
	}
	// the server (PCs can change it)
	srv := relayURL()
	if srv == "" {
		srv = "none set"
	}
	sy := r.Y + r.H - 22
	label := "Server: " + srv
	if !isTouch {
		if mpField == 2 {
			label = "Server: " + prefs.Relay
			if (frame/30)%2 == 0 {
				label += "_"
			}
		}
		sr := Rect{x, sy - 12, w, 24}
		if ptr.released && sr.Has(ptr.x, ptr.y) {
			mpField = 2
			if prefs.Relay == "" {
				prefs.Relay = RelayURL
			}
		}
	}
	Text(label, x, sy, 11, colMuted, text.AlignStart)
	// typing
	if !isTouch {
		for _, ch := range ebiten.AppendInputChars(nil) {
			if ch < 32 {
				continue
			}
			switch mpField {
			case 0:
				c := strings.ToUpper(string(ch))
				if strings.Contains(codeAlphabet, c) && len(codeEntry) < 5 {
					codeEntry += c
				}
			case 1:
				if len(prefs.Name) < 16 {
					prefs.Name += string(ch)
					savePrefs()
				}
			case 2:
				if len(prefs.Relay) < 120 {
					prefs.Relay += string(ch)
					savePrefs()
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			switch {
			case mpField == 0 && len(codeEntry) > 0:
				codeEntry = codeEntry[:len(codeEntry)-1]
			case mpField == 1 && len(prefs.Name) > 0:
				prefs.Name = prefs.Name[:len(prefs.Name)-1]
				savePrefs()
			case mpField == 2 && len(prefs.Relay) > 0:
				prefs.Relay = prefs.Relay[:len(prefs.Relay)-1]
				savePrefs()
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			mpField = (mpField + 1) % 3
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(codeEntry) == 5 && !busy {
			joinWorld(codeEntry)
		}
	}
}

// The multiplayer box in the pause menu (where the controls list usually is)
func pauseMultiplayer(kr Rect) {
	Inset(kr, colPanelLo)
	x, y := kr.X+12, kr.Y+16
	Label("MULTIPLAYER", x, y)
	y += 22
	if netRole == netHost {
		if netCode == "" {
			Text("Opening your world...", x, y+8, 14, colT, text.AlignStart)
		} else {
			Text("Share this code:", x, y+8, 13, colMuted, text.AlignStart)
			Text(netCode, x+130, y+8, 24, colT, text.AlignStart)
			if canCopy() {
				cl := "Copy"
				if frame < copiedUntil {
					cl = "Copied"
				}
				if Button(Rect{kr.X + kr.W - 88, y - 6, 76, 28}, cl, styleSlate, true, nil) {
					if copyText(netCode) {
						copiedUntil = frame + 90
					}
				}
			}
		}
		y += 34
	} else {
		Text("In "+world.Name, x, y+8, 14, colInk, text.AlignStart)
		y += 30
	}
	// who's here
	Text(fmt.Sprintf("Players (%d of %d)", len(peers)+1, maxPlayers), x, y+6, 12, colMuted, text.AlignStart)
	y += 20
	t := tintFor(netMyID)
	nameRow(x, y, playerName()+" (you)", t)
	y += 20
	for id, p := range peers {
		nameRow(x, y, p.st.Name, tintFor(id))
		y += 20
	}
	y += 8
	if netRole == netHost {
		Label("PLAYERS CAN HURT EACH OTHER", x, y+6)
		if c := seg(x, y+16, []string{"No", "Yes"}, b2i(world.PvP)); c >= 0 {
			world.PvP = c == 1
			netShare(mTime, timeMsg{T: worldTime, PvP: world.PvP, Boss: bossDown, Mode: world.Mode})
		}
	} else if world.PvP {
		Text("Players can hurt each other here.", x, y+8, 12, colT, text.AlignStart)
	}
}

func nameRow(x, y float64, name string, t [2][3]int) {
	c := t[0]
	ops = append(ops, func(dst *ebiten.Image) {
		fill(dst, x, y-5, 10, 10, colO)
		fill(dst, x+2, y-3, 6, 6, rgbColor(c))
	})
	Text(name, x+18, y, 13, colInk, text.AlignStart)
}

func rgbColor(c [3]int) color.Color {
	return color.NRGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255}
}
