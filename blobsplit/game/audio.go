package game

import (
	"bytes"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

var (
	actx   *audio.Context
	sounds = map[string][]byte{}
	music  *audio.Player
)

// A note: frequency sweep from f0 to f1 over dur seconds, starting at start.
type note struct{ start, dur, f0, f1, vol float64 }

func initAudio() {
	if os.Getenv("BLOB_NOAUDIO") != "" {
		return
	}
	actx = audio.NewContext(sampleRate)
	sounds["split"] = synth([]note{{0, 0.09, 420, 900, 0.30}})
	sounds["sell"] = synth([]note{{0, 0.07, 988, 988, 0.25}, {0.07, 0.16, 1319, 1319, 0.25}})
	sounds["buy"] = synth([]note{{0, 0.06, 660, 990, 0.25}})
	sounds["deny"] = synth([]note{{0, 0.12, 220, 150, 0.25}})
	sounds["rare"] = synth([]note{{0, 0.08, 784, 784, 0.22}, {0.08, 0.08, 988, 988, 0.22}, {0.16, 0.08, 1175, 1175, 0.22}, {0.24, 0.25, 1568, 1568, 0.22}})
	sounds["rebirth"] = synth([]note{{0, 0.6, 200, 1200, 0.25}, {0.3, 0.5, 400, 1600, 0.15}})
	sounds["tap"] = synth([]note{{0, 0.035, 900, 700, 0.15}})
	b := makeMusic()
	p, err := actx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b))))
	if err == nil {
		p.SetVolume(0.55)
		music = p
	}
}

func (g *Game) updateMusic() {
	if music == nil {
		return
	}
	if g.st.NoMusic {
		music.Pause()
	} else {
		music.Play()
	}
}

func midi(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

// makeMusic writes the background tune: an 8-bar chiptune loop in A minor
// (Am F C G) with bass, arpeggios, a melody and drums.
func makeMusic() []byte {
	const bpm = 110
	beat := 60.0 / bpm
	eighth, sixteenth := beat/2, beat/4
	total := int(32 * beat * sampleRate)
	buf := make([]float64, total)
	// add a voice: wave(phase) shaped by a fast attack and exponential decay
	add := func(start, dur, f, vol, decay float64, wave func(ph float64) float64) {
		s0, n := int(start*sampleRate), int(dur*sampleRate)
		ph := 0.0
		for i := 0; i < n; i++ {
			t := float64(i) / sampleRate
			ph += f / sampleRate
			env := math.Min(1, float64(i)/120) * math.Exp(-t*decay) * math.Min(1, float64(n-i)/300)
			buf[(s0+i)%total] += wave(ph) * env * vol
		}
	}
	tri := func(ph float64) float64 { return 4*math.Abs(ph-math.Floor(ph+0.5)) - 1 }
	sq := func(duty float64) func(float64) float64 {
		return func(ph float64) float64 {
			if ph-math.Floor(ph) < duty {
				return 1
			}
			return -1
		}
	}
	chords := [][]int{{57, 60, 64}, {53, 57, 60}, {48, 52, 55}, {55, 59, 62}} // Am F C G
	roots := []int{45, 41, 48, 43}
	melody := [][]int{
		{76, -1, 74, 76, -1, 72, 69, -1}, {72, -1, 74, -1, 76, -1, -1, -1},
		{72, -1, 69, 72, -1, 74, 72, -1}, {69, -1, 67, -1, 69, -1, -1, -1},
		{76, -1, 79, 76, -1, 74, 72, -1}, {74, -1, 76, -1, 79, -1, -1, -1},
		{81, -1, 79, 76, -1, 74, 76, -1}, {74, -1, 72, -1, 71, -1, -1, -1},
	}
	seed := uint32(7)
	noise := func(float64) float64 {
		seed = seed*1664525 + 1013904223
		return float64(seed>>8)/(1<<23) - 1
	}
	for bar := 0; bar < 8; bar++ {
		c := bar / 2
		t0 := float64(bar) * 4 * beat
		for e := 0; e < 8; e++ {
			// bass: root and octave, eighth notes
			m := roots[c]
			if e%2 == 1 {
				m += 12
			}
			add(t0+float64(e)*eighth, eighth*0.95, midi(m), 0.30, 3, tri)
			// melody
			if n := melody[bar][e]; n > 0 {
				d := eighth * 0.9
				if e+1 < 8 && melody[bar][e+1] < 0 {
					d = eighth * 1.8
				}
				add(t0+float64(e)*eighth, d, midi(n), 0.09, 2.5, sq(0.25))
			}
		}
		// arpeggio: chord tones an octave up, sixteenth notes
		for s := 0; s < 16; s++ {
			ch := chords[c]
			m := ch[s%3] + 12*(s/3%2)
			add(t0+float64(s)*sixteenth, sixteenth*0.8, midi(m), 0.035, 9, sq(0.5))
		}
		// drums: kick on 1 and 3, snare on 2 and 4, hats on the off-beats
		for b := 0; b < 4; b++ {
			bt := t0 + float64(b)*beat
			if b%2 == 0 {
				s0 := int(bt * sampleRate)
				ph := 0.0
				for i := 0; i < int(0.18*sampleRate); i++ {
					t := float64(i) / sampleRate
					ph += (45 + 110*math.Exp(-t*30)) / sampleRate
					buf[(s0+i)%total] += math.Sin(2*math.Pi*ph) * math.Exp(-t*14) * 0.55
				}
			} else {
				add(bt, 0.14, 1, 0.16, 22, noise)
				add(bt, 0.08, 185, 0.12, 30, tri)
			}
			add(bt+eighth, 0.035, 1, 0.05, 60, noise)
		}
	}
	out := make([]byte, total*4)
	for i, v := range buf {
		s := int16(math.Tanh(v*0.9) * 30000)
		out[i*4], out[i*4+1] = byte(s), byte(s>>8)
		out[i*4+2], out[i*4+3] = byte(s), byte(s>>8)
	}
	return out
}

// synth makes a soft square-wave chiptune sound as 16-bit stereo samples.
func synth(notes []note) []byte {
	end := 0.0
	for _, n := range notes {
		end = math.Max(end, n.start+n.dur)
	}
	count := int(end * sampleRate)
	buf := make([]float64, count)
	for _, n := range notes {
		ph := 0.0
		s0 := int(n.start * sampleRate)
		ln := int(n.dur * sampleRate)
		for i := 0; i < ln && s0+i < count; i++ {
			t := float64(i) / float64(ln)
			f := n.f0 + (n.f1-n.f0)*t
			ph += f / sampleRate
			// square softened with a sine so it isn't harsh
			v := 0.6*math.Copysign(1, math.Sin(2*math.Pi*ph)) + 0.4*math.Sin(2*math.Pi*ph)
			env := math.Min(1, float64(i)/80) * (1 - t) * (1 - t)
			buf[s0+i] += v * env * n.vol
		}
	}
	out := make([]byte, count*4)
	for i, v := range buf {
		s := int16(math.Max(-1, math.Min(1, v)) * 32000)
		out[i*4], out[i*4+1] = byte(s), byte(s>>8)
		out[i*4+2], out[i*4+3] = byte(s), byte(s>>8)
	}
	return out
}

func (g *Game) play(name string) {
	if actx == nil || !g.st.Sound {
		return
	}
	if b, ok := sounds[name]; ok {
		actx.NewPlayerFromBytes(b).Play()
	}
}
