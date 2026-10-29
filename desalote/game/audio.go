package game

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// A tiny synthesiser: sound effects and a generated chiptune loop.
// Everything is made from oscillators and noise, so there are no sound files.

const sampleRate = 44100

type wave int

const (
	sine wave = iota
	square
	triangle
	noiseBand
	noiseHigh
	epiano // a soft electric piano: a sine with a quickly fading bell overtone
	pluck  // a mellow plucked string: a triangle with a fast fade
)

type voice struct {
	w            wave
	f0, f1       float64 // start/end frequency (filter frequency for noise)
	dur, t       float64 // seconds
	vol          float64
	phase        float64
	music        bool
	lp, bp, prev float64 // filter state
}

type synth struct {
	mu       sync.Mutex
	voices   []*voice
	sfxVol   float64
	musicVol float64
	musicOn  bool
	// music clock
	clock     int64
	nextStep  int64
	stepIdx   int
	phrase    []*note
	phraseBar int
	rnd       *rand.Rand
}

type note struct{ m, len int }

var (
	syn      = &synth{sfxVol: 0.6, musicVol: 0.55, rnd: rand.New(rand.NewSource(1))}
	audioCtx *audio.Context
)

func initAudio() {
	// --no-audio (or DESALOTE_NOAUDIO=1) skips sound, for machines without an audio device
	if os.Getenv("DESALOTE_NOAUDIO") == "1" || hasArg("--no-audio") {
		return
	}
	audioCtx = audio.NewContext(sampleRate)
	p, err := audioCtx.NewPlayerF32(syn)
	if err != nil {
		return
	}
	p.SetBufferSize(60 * 1e6) // 60ms
	p.Play()
}

func mtof(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

func (s *synth) add(v *voice) {
	s.mu.Lock()
	if len(s.voices) < 96 {
		s.voices = append(s.voices, v)
	}
	s.mu.Unlock()
}

func tone(f0, f1, dur, vol float64, w wave) {
	syn.add(&voice{w: w, f0: f0, f1: f1, dur: dur, vol: vol})
}
func noiseFx(dur, f, vol, fTo float64) {
	if fTo == 0 {
		fTo = f
	}
	syn.add(&voice{w: noiseBand, f0: f, f1: fTo, dur: dur, vol: vol})
}

func sfx(name string) {
	switch name {
	case "click":
		tone(900, 700, 0.05, 0.06, square)
	case "open":
		tone(520, 780, 0.08, 0.08, triangle)
	case "close":
		tone(700, 450, 0.08, 0.08, triangle)
	case "deal": // a card flicked onto the table
		noiseFx(0.05, 2400+rand.Float64()*800, 0.14, 900)
	case "select":
		tone(640, 860, 0.04, 0.05, triangle)
	case "unselect":
		tone(700, 520, 0.04, 0.04, triangle)
	case "play":
		noiseFx(0.09, 1600, 0.18, 500)
		tone(330, 495, 0.1, 0.06, triangle)
	case "tick": // a scoring card counting up
		tone(1100+rand.Float64()*200, 1500, 0.04, 0.05, square)
	case "score":
		for k, f := range []float64{523, 659, 784} {
			syn.add(&voice{w: triangle, f0: f, f1: f, dur: 0.14, vol: 0.07, t: -0.06 * float64(k)})
		}
	case "special":
		for k, f := range []float64{392, 523, 659, 784, 1046} {
			syn.add(&voice{w: square, f0: f, f1: f * 1.01, dur: 0.18, vol: 0.05, t: -0.07 * float64(k)})
		}
		noiseFx(0.5, 3000, 0.08, 6000)
	case "discard":
		noiseFx(0.12, 1200, 0.16, 300)
	case "coin":
		tone(1320, 1320, 0.06, 0.05, square)
		syn.add(&voice{w: square, f0: 1760, f1: 1760, dur: 0.12, vol: 0.05, t: -0.06})
	case "bid":
		tone(880, 1175, 0.07, 0.07, square)
	case "outbid":
		tone(600, 400, 0.12, 0.07, square)
	case "hex":
		tone(300, 90, 0.4, 0.12, square)
		noiseFx(0.4, 500, 0.15, 150)
	case "win":
		for k, f := range []float64{523, 659, 784, 1046} {
			syn.add(&voice{w: square, f0: f, f1: f, dur: 0.12, vol: 0.06, t: -0.08 * float64(k)})
		}
	case "lose":
		tone(400, 60, 0.45, 0.15, triangle)
		noiseFx(0.25, 700, 0.12, 200)
	case "empty":
		tone(520, 160, 0.25, 0.1, square)
	}
}

func setVolumes() {
	syn.mu.Lock()
	syn.sfxVol = 0.75 * float64(prefs.SfxVol) / 5
	syn.musicVol = 0.8 * float64(prefs.MusicVol) / 5
	syn.musicOn = prefs.MusicVol > 0
	syn.mu.Unlock()
}

// ---- Music: "Ember Lounge", a slow swung card-room groove ----
// An 8-bar loop in A minor (Am9 Dm9 G9 Cmaj7 Fmaj7 Dm7 E7 Am): electric-piano chords
// stabbing on the offbeats, a walking bass, brushed drums and a sparse plucked melody
// that changes every time round.

const bpm = 86

var stepSamples = int64(sampleRate * 60 / bpm / 4) // 16th notes

var chords = []struct {
	root  int
	tones [4]int
}{
	{45, [4]int{60, 64, 67, 71}}, // Am9
	{38, [4]int{60, 64, 65, 69}}, // Dm9
	{43, [4]int{59, 62, 65, 69}}, // G9
	{48, [4]int{59, 64, 67, 71}}, // Cmaj7
	{41, [4]int{57, 60, 64, 69}}, // Fmaj7
	{38, [4]int{57, 60, 62, 65}}, // Dm7
	{40, [4]int{56, 59, 62, 64}}, // E7
	{45, [4]int{57, 60, 64, 69}}, // Am
}

var penta = []int{69, 72, 74, 76, 79, 81, 84}

// A new melody for the next two bars: a few notes, often resting
func (s *synth) newPhrase() {
	s.phrase = s.phrase[:0]
	for st := 0; st < 32; st += 2 {
		if s.rnd.Float64() < 0.62 || st%16 == 14 {
			s.phrase = append(s.phrase, nil)
			continue
		}
		l := 2
		if s.rnd.Float64() < 0.35 {
			l = 4
		}
		s.phrase = append(s.phrase, &note{penta[s.rnd.Intn(len(penta))], l})
	}
}

// mv adds a music voice, starting delay seconds from now
func (s *synth) mv(w wave, m int, dur, vol, delay float64) {
	s.voices = append(s.voices, &voice{w: w, f0: mtof(m), f1: mtof(m), dur: dur, vol: vol, music: true, t: -delay})
}

func (s *synth) scheduleStep(i int) {
	bar, st := (i/16)%len(chords), i%16
	ch := chords[bar]
	next := chords[(bar+1)%len(chords)]
	stepSec := float64(stepSamples) / sampleRate
	swing := 0.0
	if st%4 == 2 { // the offbeat 8ths come a little late
		swing = stepSec * 0.35
	}
	if st == 0 && bar%2 == 0 {
		s.newPhrase()
		s.phraseBar = bar
	}
	// electric piano: a soft held chord on the one, stabs on the "and" of 2 and on 4
	if st == 0 {
		for _, t := range ch.tones {
			s.mv(epiano, t, stepSec*12, 0.065, 0)
		}
	}
	if st == 6 || (st == 12 && bar%2 == 1) {
		for k, t := range ch.tones {
			s.mv(epiano, t, stepSec*2.5, 0.09, swing+float64(k)*0.006)
		}
	}
	// walking bass: root, fifth, octave, then a step into the next chord
	switch st {
	case 0:
		s.mv(sine, ch.root, stepSec*3.6, 0.5, 0)
	case 4:
		s.mv(sine, ch.root+7, stepSec*3.6, 0.42, 0)
	case 8:
		s.mv(sine, ch.root+12, stepSec*3.6, 0.4, 0)
	case 12:
		approach := next.root - 1
		if (bar+i/128)%2 == 1 {
			approach = next.root + 2
		}
		s.mv(sine, approach, stepSec*3.4, 0.4, 0)
	}
	// melody, from the phrase for these two bars
	if st%2 == 0 {
		k := ((bar-s.phraseBar+len(chords))%2)*8 + st/2
		if k < len(s.phrase) {
			if n := s.phrase[k]; n != nil {
				s.mv(pluck, n.m, stepSec*float64(n.len)*1.6, 0.11, swing)
			}
		}
	}
	// brushed drums: a soft kick, a rim click on 2 and 4, a swung shaker
	if st == 0 || st == 10 {
		s.voices = append(s.voices, &voice{w: sine, f0: 90, f1: 40, dur: 0.22, vol: 0.4, music: true})
	}
	if st == 4 || st == 12 {
		s.voices = append(s.voices, &voice{w: noiseBand, f0: 2200, f1: 1600, dur: 0.07, vol: 0.12, music: true})
	}
	if st%2 == 0 {
		v := 0.025
		if st%4 == 2 {
			v = 0.04
		}
		s.voices = append(s.voices, &voice{w: noiseHigh, f0: 8000, f1: 8000, dur: 0.05, vol: v, music: true, t: -swing})
	}
}

// Read fills the audio buffer with float32 stereo samples.
func (s *synth) Read(p []byte) (int, error) {
	frames := len(p) / 8
	s.mu.Lock()
	defer s.mu.Unlock()
	dt := 1.0 / sampleRate
	for f := 0; f < frames; f++ {
		if s.musicOn && s.clock >= s.nextStep {
			s.scheduleStep(s.stepIdx)
			s.stepIdx++
			s.nextStep += stepSamples
		} else if !s.musicOn {
			s.nextStep = s.clock
		}
		s.clock++
		var sfxSum, musSum float64
		live := s.voices[:0]
		for _, v := range s.voices {
			v.t += dt
			if v.t < 0 { // delayed start
				live = append(live, v)
				continue
			}
			if v.t >= v.dur {
				continue
			}
			k := v.t / v.dur
			freq := v.f0 * math.Pow(v.f1/v.f0, k)
			// envelope: 10ms attack, exponential decay
			env := math.Pow(0.0001/1, k)
			if v.t < 0.01 {
				env *= v.t / 0.01
			}
			var smp float64
			switch v.w {
			case sine, square, triangle, epiano, pluck:
				v.phase += freq * dt
				v.phase -= math.Floor(v.phase)
				switch v.w {
				case epiano:
					smp = math.Sin(v.phase*2*math.Pi) + 0.45*math.Sin(v.phase*4*math.Pi)*math.Exp(-v.t*9) + 0.12*math.Sin(v.phase*6*math.Pi)*math.Exp(-v.t*14)
				case pluck:
					smp = (4*math.Abs(v.phase-0.5) - 1) * math.Exp(-v.t*3)
				case sine:
					smp = math.Sin(v.phase * 2 * math.Pi)
				case square:
					if v.phase < 0.5 {
						smp = 1
					} else {
						smp = -1
					}
				case triangle:
					smp = 4*math.Abs(v.phase-0.5) - 1
				}
			case noiseBand: // state-variable band-pass
				n := s.rnd.Float64()*2 - 1
				fc := 2 * math.Sin(math.Pi*math.Min(freq, 8000)/sampleRate)
				v.lp += fc * v.bp
				hp := n - v.lp - 0.8*v.bp
				v.bp += fc * hp
				smp = v.bp * 1.5
			case noiseHigh: // one-pole high-pass
				n := s.rnd.Float64()*2 - 1
				a := 1 / (1 + 2*math.Pi*freq*dt)
				out := a * (v.lp + n - v.prev)
				v.prev, v.lp = n, out
				smp = out
			}
			if v.music {
				musSum += smp * v.vol * env
			} else {
				sfxSum += smp * v.vol * env
			}
			live = append(live, v)
		}
		s.voices = live
		mv := 0.0
		if s.musicOn {
			mv = s.musicVol
		}
		out := float32(math.Tanh(sfxSum*s.sfxVol + musSum*mv))
		bits := math.Float32bits(out)
		binary.LittleEndian.PutUint32(p[f*8:], bits)
		binary.LittleEndian.PutUint32(p[f*8+4:], bits)
	}
	return frames * 8, nil
}
