package game

import (
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

var (
	actx   *audio.Context
	sounds = map[string][]byte{}
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
}

// updateMusic applies the music button at once; tickMusic does the rest each frame.
func (g *Game) updateMusic() { g.tickMusic() }

func midi(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

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
