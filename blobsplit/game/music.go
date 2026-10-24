package game

import (
	"bytes"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Music: spacey synth loops written in code, one for day, one for night and one for
// each event. Each is 8 bars of slow pads, a sub bass, an echoing arpeggio, bell
// melodies and sparkles; they differ in scale, tempo, chords and mix. A track is made
// the first time it's needed (in the background) and the game crossfades to it.

type track struct {
	bpm     float64
	root    int   // MIDI note of the scale's first degree
	scale   []int // semitones of the mode
	prog    []int // a chord (scale degree) every two bars
	pad     float64
	bright  float64 // pad filter: 0 dark .. 1 bright
	arp     float64
	arpRate int // arpeggio notes per beat
	bell    float64
	density float64 // chance of a bell note each beat
	bass    float64
	drums   float64
	sparkle float64
	echo    float64
	seed    int64
}

var (
	ionian   = []int{0, 2, 4, 5, 7, 9, 11}
	lydian   = []int{0, 2, 4, 6, 7, 9, 11}
	dorian   = []int{0, 2, 3, 5, 7, 9, 10}
	aeolian  = []int{0, 2, 3, 5, 7, 8, 10}
	phrygian = []int{0, 1, 3, 5, 7, 8, 10}
	mixolyd  = []int{0, 2, 4, 5, 7, 9, 10}
)

var tracks = map[string]track{
	// day: bright and floating
	"day": {bpm: 84, root: 60, scale: lydian, prog: []int{0, 1, 5, 4}, pad: 0.16, bright: 0.7, arp: 0.07, arpRate: 2, bell: 0.08, density: 0.45, bass: 0.22, drums: 0.35, sparkle: 0.03, echo: 0.45, seed: 1},
	// night: slow, deep and wide
	"night":    {bpm: 64, root: 57, scale: aeolian, prog: []int{0, 5, 2, 6}, pad: 0.2, bright: 0.25, arp: 0.05, arpRate: 1, bell: 0.07, density: 0.3, bass: 0.28, drums: 0, sparkle: 0.05, echo: 0.6, seed: 2},
	"meteor":   {bpm: 118, root: 62, scale: aeolian, prog: []int{0, 6, 5, 6}, pad: 0.13, bright: 0.5, arp: 0.08, arpRate: 4, bell: 0.07, density: 0.5, bass: 0.26, drums: 0.6, sparkle: 0.06, echo: 0.5, seed: 3},
	"blizzard": {bpm: 70, root: 59, scale: dorian, prog: []int{0, 3, 6, 4}, pad: 0.12, bright: 0.85, arp: 0.06, arpRate: 3, bell: 0.1, density: 0.6, bass: 0.15, drums: 0, sparkle: 0.09, echo: 0.65, seed: 4},
	"goldrush": {bpm: 104, root: 55, scale: mixolyd, prog: []int{0, 3, 4, 6}, pad: 0.13, bright: 0.75, arp: 0.08, arpRate: 3, bell: 0.08, density: 0.55, bass: 0.26, drums: 0.7, sparkle: 0.05, echo: 0.4, seed: 5},
	"aurora":   {bpm: 58, root: 62, scale: lydian, prog: []int{0, 4, 1, 5}, pad: 0.22, bright: 0.55, arp: 0.04, arpRate: 2, bell: 0.07, density: 0.35, bass: 0.2, drums: 0, sparkle: 0.1, echo: 0.7, seed: 6},
	"bloom":    {bpm: 88, root: 53, scale: ionian, prog: []int{0, 5, 3, 4}, pad: 0.17, bright: 0.6, arp: 0.06, arpRate: 2, bell: 0.09, density: 0.5, bass: 0.22, drums: 0.3, sparkle: 0.04, echo: 0.5, seed: 7},
	"heatwave": {bpm: 76, root: 52, scale: phrygian, prog: []int{0, 1, 0, 6}, pad: 0.15, bright: 0.4, arp: 0.07, arpRate: 2, bell: 0.06, density: 0.35, bass: 0.32, drums: 0.45, sparkle: 0.02, echo: 0.45, seed: 8},
	"eclipse":  {bpm: 54, root: 49, scale: phrygian, prog: []int{0, 5, 1, 0}, pad: 0.22, bright: 0.15, arp: 0.03, arpRate: 1, bell: 0.06, density: 0.2, bass: 0.34, drums: 0, sparkle: 0.04, echo: 0.7, seed: 9},
}

var musicState struct {
	sync.Mutex
	pcm      map[string][]byte // finished tracks
	making   map[string]bool
	cur, old *audio.Player
	curID    string
	fade     float64 // 0..1 through the crossfade
}

const musicVolume = 0.5

// musicWant is the track for right now: the event's, or day's or night's.
func (g *Game) musicWant() string {
	if _, ok := tracks[g.st.Event]; ok {
		return g.st.Event
	}
	if g.night() {
		return "night"
	}
	return "day"
}

// tickMusic starts making the track that should play, crossfades to it when it's
// ready, and pauses or resumes for the music button.
func (g *Game) tickMusic() {
	if actx == nil {
		return
	}
	ms := &musicState
	ms.Lock()
	defer ms.Unlock()
	if ms.pcm == nil {
		ms.pcm, ms.making = map[string][]byte{}, map[string]bool{}
	}
	if g.st.NoMusic {
		for _, p := range []*audio.Player{ms.cur, ms.old} {
			if p != nil && p.IsPlaying() {
				p.Pause()
			}
		}
		return
	}
	want := g.musicWant()
	if want != ms.curID {
		if b, ok := ms.pcm[want]; ok {
			if ms.old != nil {
				ms.old.Close()
			}
			p, err := actx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b))))
			if err == nil {
				ms.old, ms.cur, ms.curID, ms.fade = ms.cur, p, want, 0
				p.SetVolume(0)
				p.Play()
			}
		} else if !ms.making[want] {
			ms.making[want] = true
			go func(id string) {
				b := makeTrack(tracks[id])
				ms.Lock()
				ms.pcm[id] = b
				ms.Unlock()
			}(want)
		}
	}
	// crossfade over about two seconds
	if ms.fade < 1 {
		ms.fade = math.Min(1, ms.fade+dt/2)
	}
	if ms.cur != nil {
		ms.cur.SetVolume(musicVolume * ms.fade)
		if !ms.cur.IsPlaying() {
			ms.cur.Play()
		}
	}
	if ms.old != nil {
		ms.old.SetVolume(musicVolume * (1 - ms.fade))
		if ms.fade >= 1 {
			ms.old.Close()
			ms.old = nil
		} else if !ms.old.IsPlaying() {
			ms.old.Play()
		}
	}
}

// makeTrack writes one 8-bar loop as 16-bit stereo. Every note wraps around the end,
// and the echo runs over the loop twice, so it repeats without a seam.
func makeTrack(t track) []byte {
	beat := 60 / t.bpm
	bars := 8
	total := int(float64(bars*4) * beat * sampleRate)
	L := make([]float64, total)
	R := make([]float64, total)
	wet := make([]float64, total) // the part that echoes: arpeggio, bells, sparkles
	rng := rand.New(rand.NewSource(t.seed))
	note := func(deg, oct int) float64 {
		n := len(t.scale)
		o := deg / n
		if deg < 0 && deg%n != 0 {
			o--
		}
		d := ((deg % n) + n) % n
		return midi(t.root + 12*(oct+o) + t.scale[d])
	}
	at := func(sec float64) int { return int(sec * sampleRate) }

	chordLen := 2 * 4 * beat
	for ci, deg := range t.prog {
		start := float64(ci) * chordLen
		// pad: a triad, each note two detuned saws, softened by a low-pass filter,
		// fading in and out so the chords overlap
		for k, step := range []int{0, 2, 4} {
			f := note(deg+step, 0)
			for _, det := range []float64{0.996, 1.004} {
				ph, lp := rng.Float64(), 0.0
				a := 0.01 + 0.09*t.bright
				n := at(chordLen + 1.2)
				pan := 0.3 + 0.4*float64(k)/2
				for i := 0; i < n; i++ {
					tm := float64(i) / sampleRate
					ph += f * det / sampleRate
					saw := 2*(ph-math.Floor(ph)) - 1
					lp += a * (saw - lp)
					env := math.Min(1, tm/1.0) * math.Min(1, (chordLen+1.2-tm)/1.2)
					v := lp * env * t.pad * 0.5
					j := (at(start) + i) % total
					L[j] += v * (1 - pan)
					R[j] += v * pan
				}
			}
		}
		// sub bass: the chord's root, long and soft
		if t.bass > 0 {
			f := note(deg, -2)
			for b := 0; b < 2; b++ {
				s0 := at(start + float64(b)*4*beat)
				n := at(4 * beat)
				for i := 0; i < n; i++ {
					tm := float64(i) / sampleRate
					env := math.Min(1, tm/0.08) * math.Exp(-tm*0.6) * math.Min(1, float64(n-i)/2000)
					v := (math.Sin(2*math.Pi*f*tm) + 0.25*math.Sin(4*math.Pi*f*tm)) * env * t.bass
					j := (s0 + i) % total
					L[j] += v
					R[j] += v
				}
			}
		}
		// arpeggio: plucked chord notes climbing two octaves
		if t.arp > 0 {
			steps := 8 * t.arpRate
			pattern := []int{0, 2, 4, 7, 9, 11, 9, 7}
			for s := 0; s < steps; s++ {
				if rng.Float64() < 0.12 {
					continue
				}
				f := note(deg+pattern[s%len(pattern)], 1)
				s0 := at(start + float64(s)*beat/float64(t.arpRate))
				n := at(0.5)
				for i := 0; i < n; i++ {
					tm := float64(i) / sampleRate
					v := (math.Sin(2*math.Pi*f*tm) + 0.3*math.Sin(6*math.Pi*f*tm)*math.Exp(-tm*20)) * math.Exp(-tm*7) * math.Min(1, tm*400) * t.arp
					wet[(s0+i)%total] += v
				}
			}
		}
	}
	// bells: an FM bell melody wandering over the scale
	deg := 4
	for b := 0; b < bars*4; b++ {
		if rng.Float64() > t.density {
			continue
		}
		deg += []int{-2, -1, -1, 1, 1, 2, 3, -3}[rng.Intn(8)]
		deg = max(0, min(10, deg))
		f := note(deg, 1)
		s0 := at(float64(b)*beat + float64(rng.Intn(2))*beat/2)
		n := at(2.5)
		for i := 0; i < n; i++ {
			tm := float64(i) / sampleRate
			idx := 2.5 * math.Exp(-tm*3)
			v := math.Sin(2*math.Pi*f*tm+idx*math.Sin(2*math.Pi*f*3.5*tm)) * math.Exp(-tm*1.6) * math.Min(1, tm*300) * t.bell
			wet[(s0+i)%total] += v
		}
	}
	// sparkles: tiny high twinkles
	for k := 0; k < int(t.sparkle*400); k++ {
		f := note(rng.Intn(7), 3)
		s0 := rng.Intn(total)
		n := at(0.6)
		for i := 0; i < n; i++ {
			tm := float64(i) / sampleRate
			wet[(s0+i)%total] += math.Sin(2*math.Pi*f*tm) * math.Exp(-tm*9) * math.Min(1, tm*500) * 0.05
		}
	}
	// drums: a soft kick and brushed hats
	if t.drums > 0 {
		for b := 0; b < bars*4; b++ {
			s0 := at(float64(b) * beat)
			if b%2 == 0 {
				ph := 0.0
				for i := 0; i < at(0.25); i++ {
					tm := float64(i) / sampleRate
					ph += (42 + 70*math.Exp(-tm*25)) / sampleRate
					v := math.Sin(2*math.Pi*ph) * math.Exp(-tm*10) * 0.5 * t.drums
					L[(s0+i)%total] += v
					R[(s0+i)%total] += v
				}
			}
			hp := 0.0
			h0 := at(float64(b)*beat + beat/2)
			for i := 0; i < at(0.06); i++ {
				tm := float64(i) / sampleRate
				x := rng.Float64()*2 - 1
				hp = x - hp*0.6
				v := hp * math.Exp(-tm*50) * 0.05 * t.drums
				L[(h0+i)%total] += v * 0.8
				R[(h0+i)%total] += v
			}
		}
	}
	// echo: different delay times left and right make it wide; two passes fill the seam
	dl, dr := at(beat*0.75), at(beat*1.0)
	el, er := make([]float64, total), make([]float64, total)
	fb := 0.45 * t.echo / 0.6
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < total; i++ {
			el[i] = wet[i] + el[(i-dl+total)%total]*fb
			er[i] = wet[i] + er[(i-dr+total)%total]*fb
		}
	}
	out := make([]byte, total*4)
	for i := 0; i < total; i++ {
		l := L[i] + wet[i]*0.6 + el[i]*0.5*t.echo
		r := R[i] + wet[i]*0.6 + er[i]*0.5*t.echo
		sl := int16(math.Tanh(l*1.1) * 30000)
		sr := int16(math.Tanh(r*1.1) * 30000)
		out[i*4], out[i*4+1] = byte(sl), byte(sl>>8)
		out[i*4+2], out[i*4+3] = byte(sr), byte(sr>>8)
	}
	return out
}
