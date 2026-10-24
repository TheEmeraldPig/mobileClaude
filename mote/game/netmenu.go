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
	addrEntry     string
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
	mpField = 3
	openScreen(scMultiplayer)
}

func multiplayerMenu() {
	startFinding()
	r := sheet(700, 440)
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
	busy := netRole == netGuest
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
	y += 44
	// worlds being hosted on this network
	Label("WORLDS ON THIS WI-FI", x, y)
	y += 14
	list := foundWorlds()
	rh := 38.0
	if len(list) == 0 {
		box := Rect{x, y, w, rh}
		Inset(box, colSlateLo)
		dots := strings.Repeat(".", (frame/20)%4)
		Text("Looking for worlds"+dots, box.X+12, box.Y+rh/2, 14, colMuted, text.AlignStart)
		Text("A friend picks Host below on the same Wi-Fi or hotspot", box.X+box.W-12, box.Y+rh/2, 11, colMuted, text.AlignEnd)
		y += rh + 6
	}
	for i, f := range list {
		if i >= 3 {
			break
		}
		row := Rect{x, y, w, rh}
		Inset(row, colSlateLo)
		Text(f.ad.World, row.X+12, row.Y+13, 14, colInk, text.AlignStart)
		Text(fmt.Sprintf("%s  -  %d of %d players  -  %s", f.ad.Host, f.ad.Players, maxPlayers, modeNames[max(0, min(len(modeNames)-1, f.ad.Mode))]),
			row.X+12, row.Y+28, 11, colMuted, text.AlignStart)
		if Button(Rect{row.X + row.W - 100, row.Y + 4, 92, rh - 8}, "Join", styleGold, !busy && f.ad.Players < maxPlayers, icoPlay) {
			joinLAN(f.addr)
		}
		y += rh + 6
	}
	y += 4
	// or type the host's address
	Label("OR JOIN BY ADDRESS", x, y)
	y += 14
	af := Rect{x, y, 250, 34}
	Inset(af, colPanelLo)
	shown := addrEntry
	if mpField == 3 && !isTouch && (frame/30)%2 == 0 {
		shown += "_"
	}
	if addrEntry == "" && (mpField != 3 || isTouch) {
		Text("e.g. 192.168.1.20", af.X+10, af.Y+17, 14, colMuted, text.AlignStart)
	}
	Text(shown, af.X+10, af.Y+17, 15, colInk, text.AlignStart)
	if ptr.released && af.Has(ptr.x, ptr.y) {
		mpField = 3
	}
	if Button(Rect{af.X + af.W + 10, y, 100, 34}, "Join", styleViolet, addrEntry != "" && !busy, icoPlay) {
		joinLAN(addrEntry)
	}
	if busy && Button(Rect{af.X + af.W + 120, y, 100, 34}, "Cancel", styleSlate, true, nil) {
		leaveMultiplayer()
		openScreen(scMultiplayer)
	}
	y += 40
	// phones: a number pad for the address
	if isTouch {
		keys := "1234567890.<"
		kw := (w - 11*5) / 12
		for i, ch := range keys {
			if Button(Rect{x + float64(i)*(kw+5), y, kw, 30}, string(ch), styleSlate, true, nil) {
				if ch == '<' {
					if len(addrEntry) > 0 {
						addrEntry = addrEntry[:len(addrEntry)-1]
					}
				} else if len(addrEntry) < 21 {
					addrEntry += string(ch)
				}
			}
		}
		y += 36
	}
	// host
	y += 4
	Label("HOST", x, y+4)
	y += 18
	if Button(Rect{x, y, 250, 36}, "Open one of my worlds", styleGold, !busy, icoWorld) {
		hostAfterOpen = true
		openScreen(scWorlds)
	}
	Text("Friends on the same Wi-Fi will see it here.", x+262, y+18, 12, colMuted, text.AlignStart)
	y += 46
	// what's happening
	if netStatus != "" {
		Text(netStatus, x, y+6, 14, colT, text.AlignStart)
	}
	// online codes, only when this build has a server for them
	if relayURL() != "" && !isTouch {
		cy := r.Y + r.H - 26
		Text("Online code:", x, cy, 12, colMuted, text.AlignStart)
		cf := Rect{x + 96, cy - 13, 110, 26}
		Inset(cf, colPanelLo)
		cs := codeEntry
		if mpField == 0 && (frame/30)%2 == 0 {
			cs += "_"
		}
		Text(cs, cf.X+8, cf.Y+13, 14, colT, text.AlignStart)
		if ptr.released && cf.Has(ptr.x, ptr.y) {
			mpField = 0
		}
		if Button(Rect{cf.X + cf.W + 8, cy - 13, 70, 26}, "Join", styleSlate, len(codeEntry) == 5 && !busy, nil) {
			joinWorld(codeEntry)
		}
	}
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
			case 3:
				if strings.ContainsRune("0123456789.:abcdefABCDEF", ch) && len(addrEntry) < 40 {
					addrEntry += string(ch)
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
			case mpField == 3 && len(addrEntry) > 0:
				addrEntry = addrEntry[:len(addrEntry)-1]
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			mpField = map[int]int{3: 1, 1: 3, 0: 3}[mpField]
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !busy {
			if mpField == 3 && addrEntry != "" {
				joinLAN(addrEntry)
			} else if mpField == 0 && len(codeEntry) == 5 {
				joinWorld(codeEntry)
			}
		}
	}
}

// The multiplayer box in the pause menu (where the controls list usually is)
func pauseMultiplayer(kr Rect) {
	Inset(kr, colPanelLo)
	x, y := kr.X+12, kr.Y+16
	Label("MULTIPLAYER", x, y)
	y += 22
	if netRole == netHost && lanServer != nil {
		// on the local network: friends see it in their list, or type the address
		Text("Open to friends on this Wi-Fi.", x, y+8, 13, colInk, text.AlignStart)
		y += 22
		addr := myAddress()
		if lanAt != lanPort && addr != "" {
			addr = fmt.Sprintf("%s:%d", addr, lanAt)
		}
		if addr != "" {
			Text("Your address:", x, y+8, 13, colMuted, text.AlignStart)
			Text(addr, x+108, y+8, 18, colT, text.AlignStart)
		}
		y += 30
	} else if netRole == netHost {
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
