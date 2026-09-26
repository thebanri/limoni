package main

import (
	"encoding/binary"
	"io"
	"math"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Sound works as in Castle Lemonstein (apps/castle-lemonstein/sound.go), from which the
// mixer is taken: synthesised at start-up, no sample files, and streamed
// as raw 16-bit stereo PCM to whichever player the system has — pw-play
// (PipeWire), pacat (PulseAudio), aplay (ALSA) or sox's play — or, in a
// browser, to Web Audio (sound_js.go). Where there is none the game is
// silent and says so on exit.
//
// The game side of it allocates nothing: play hands a small value to the
// mixer's channel and never waits. The mixer runs on its own goroutine.

type sfx uint8

const (
	sfxMove sfx = iota
	sfxTurn
	sfxLand
	sfxDrop
	sfxTrickle
	sfxClear
	sfxChain
	sfxLevel
	sfxStart
	sfxPause
	sfxOver
	nSfx
)

type speaker interface {
	play(s sfx, vol, pan float32)
	setVolume(v float32) // 0 … 1, for everything
}

const sampleRate = 22050

type sfxReq struct {
	s        sfx
	vol, pan float32
}

type mixer struct {
	req    chan sfxReq
	clips  [nSfx][]float32
	cmd    *exec.Cmd
	w      io.WriteCloser
	player string
	master atomic.Uint32 // float32 bits
	once   sync.Once
	done   chan struct{}
}

type voice struct {
	clip   []float32
	pos    int
	gl, gr float32
}

var players = [...][]string{
	{"pw-play", "--raw", "--rate", "22050", "--channels", "2", "--format", "s16", "--latency", "40ms", "-"},
	{"pacat", "--raw", "--rate=22050", "--channels=2", "--format=s16le", "--latency-msec=40"},
	{"aplay", "-q", "-t", "raw", "-f", "S16_LE", "-r", "22050", "-c", "2", "-B", "60000"},
	{"play", "-q", "-t", "raw", "-r", "22050", "-e", "signed", "-b", "16", "-c", "2", "-"},
}

// newMixer starts the first player it finds, or returns nil. In a browser
// the player is the page's Web Audio context.
func newMixer() *mixer {
	if m := newBrowserMixer(); m != nil {
		return m
	}
	for _, p := range players {
		if _, err := exec.LookPath(p[0]); err != nil {
			continue
		}
		cmd := exec.Command(p[0], p[1:]...)
		w, err := cmd.StdinPipe()
		if err != nil {
			continue
		}
		if err := cmd.Start(); err != nil {
			continue
		}
		m := &mixer{req: make(chan sfxReq, 64), cmd: cmd, w: w, player: p[0], done: make(chan struct{})}
		m.setVolume(1)
		m.synth()
		go m.run()
		return m
	}
	return nil
}

func (m *mixer) play(s sfx, vol, pan float32) {
	select {
	case m.req <- sfxReq{s, vol, pan}:
	default: // the mixer is behind: drop the sound rather than the frame
	}
}

func (m *mixer) setVolume(v float32) { m.master.Store(math.Float32bits(v)) }

func (m *mixer) close() {
	m.once.Do(func() {
		close(m.done)
		if m.cmd == nil {
			return // the browser's: nothing to wait for
		}
		_ = m.w.Close()
		done := make(chan struct{})
		go func() { _ = m.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(300 * time.Millisecond):
			_ = m.cmd.Process.Kill()
		}
	})
}

// run mixes 10 ms at a time. It keeps no more than about 60 ms ahead of the
// clock, or a pipe's worth of sound would queue up and every effect would
// arrive late.
func (m *mixer) run() {
	const chunk = sampleRate / 100
	var voices [24]voice
	out := make([]byte, chunk*4)
	start := time.Now()
	written := 0
	for {
		for {
			select {
			case r := <-m.req:
				for i := range voices {
					if voices[i].clip == nil {
						l := r.vol * float32(math.Min(1, 1-float64(r.pan)))
						rr := r.vol * float32(math.Min(1, 1+float64(r.pan)))
						voices[i] = voice{clip: m.clips[r.s], gl: l, gr: rr}
						break
					}
				}
				continue
			case <-m.done:
				return
			default:
			}
			break
		}
		master := math.Float32frombits(m.master.Load())
		for i := 0; i < chunk; i++ {
			var l, r float32
			for v := range voices {
				vc := &voices[v]
				if vc.clip == nil {
					continue
				}
				s := vc.clip[vc.pos]
				l += s * vc.gl
				r += s * vc.gr
				vc.pos++
				if vc.pos >= len(vc.clip) {
					vc.clip = nil
				}
			}
			binary.LittleEndian.PutUint16(out[i*4:], uint16(clip16(l*master)))
			binary.LittleEndian.PutUint16(out[i*4+2:], uint16(clip16(r*master)))
		}
		if _, err := m.w.Write(out); err != nil {
			return
		}
		written += chunk
		ahead := time.Duration(written)*time.Second/sampleRate - time.Since(start)
		if ahead > 60*time.Millisecond {
			time.Sleep(ahead - 50*time.Millisecond)
		}
	}
}

func clip16(v float32) int16 {
	v = float32(math.Tanh(float64(v))) // soft clip: many sounds at once stay musical
	return int16(v * 32000)
}

// ── synthesis ────────────────────────────────────────────────────────────

type synth struct {
	buf  []float32
	seed uint32
}

func newSynth(seconds float64) *synth {
	return &synth{buf: make([]float32, int(seconds*sampleRate)), seed: 12345}
}

func (s *synth) noise() float32 {
	s.seed ^= s.seed << 13
	s.seed ^= s.seed >> 17
	s.seed ^= s.seed << 5
	return float32(s.seed)/float32(math.MaxUint32)*2 - 1
}

// tone adds a sine sweeping from f0 to f1 Hz between t0 and t1 seconds,
// shaped by a quick attack and an exponential decay.
func (s *synth) tone(t0, t1, f0, f1, amp, decay float64, wave func(float64) float64) {
	i0, i1 := int(t0*sampleRate), int(t1*sampleRate)
	phase := 0.0
	for i := i0; i < i1 && i < len(s.buf); i++ {
		t := float64(i-i0) / float64(i1-i0)
		f := f0 * math.Pow(f1/f0, t)
		phase += f / sampleRate
		env := math.Min(1, t*(t1-t0)*400) * math.Exp(-t*decay)
		s.buf[i] += float32(wave(phase) * amp * env)
	}
}

// hiss adds noise between t0 and t1, low-passed at cutoff Hz (sweeping to
// cutoff1), shaped like tone.
func (s *synth) hiss(t0, t1, cutoff, cutoff1, amp, decay float64) {
	i0, i1 := int(t0*sampleRate), int(t1*sampleRate)
	var y float32
	for i := i0; i < i1 && i < len(s.buf); i++ {
		t := float64(i-i0) / float64(i1-i0)
		c := cutoff * math.Pow(cutoff1/cutoff, t)
		a := float32(1 - math.Exp(-2*math.Pi*c/sampleRate))
		y += (s.noise() - y) * a
		env := math.Min(1, t*(t1-t0)*300) * math.Exp(-t*decay)
		s.buf[i] += y * float32(amp*env)
	}
}

func sine(p float64) float64   { return math.Sin(2 * math.Pi * p) }
func square(p float64) float64 { return math.Copysign(0.6, math.Sin(2*math.Pi*p)) }
func saw(p float64) float64    { return 2*(p-math.Floor(p)) - 1 }

func (m *mixer) synth() {
	mk := func(sec float64, f func(s *synth)) []float32 {
		s := newSynth(sec)
		f(s)
		return s.buf
	}
	m.clips[sfxMove] = mk(0.04, func(s *synth) {
		s.tone(0, 0.04, 900, 800, 0.14, 8, square)
	})
	m.clips[sfxTurn] = mk(0.07, func(s *synth) {
		s.tone(0, 0.07, 700, 1100, 0.16, 5, square)
		s.hiss(0, 0.03, 4000, 2000, 0.08, 8)
	})
	// A block landing and giving way: a soft thump, then sand.
	m.clips[sfxLand] = mk(0.35, func(s *synth) {
		s.tone(0, 0.12, 150, 70, 0.45, 5, sine)
		s.hiss(0, 0.35, 2500, 600, 0.35, 3.5)
	})
	m.clips[sfxDrop] = mk(0.4, func(s *synth) {
		s.hiss(0, 0.1, 800, 3500, 0.25, 1) // the rush down
		s.tone(0.08, 0.22, 120, 55, 0.6, 4, sine)
		s.hiss(0.08, 0.4, 2800, 500, 0.4, 3)
	})
	// Sand running: short and quiet, played again and again while it runs.
	m.clips[sfxTrickle] = mk(0.14, func(s *synth) {
		s.hiss(0, 0.14, 5000, 3000, 0.22, 1.5)
		for t := 0.0; t < 0.13; t += 0.023 {
			s.hiss(t, t+0.012, 7000, 5000, 0.12, 6) // single grains
		}
	})
	// A clear: a bright chord and the sand going.
	m.clips[sfxClear] = mk(0.7, func(s *synth) {
		for i, f := range [3]float64{1047, 1319, 1568} {
			t := float64(i) * 0.05
			s.tone(t, t+0.5, f, f, 0.2, 4, sine)
			s.tone(t, t+0.5, f*2, f*2, 0.05, 6, sine)
		}
		s.hiss(0, 0.7, 6000, 900, 0.3, 3)
	})
	m.clips[sfxChain] = mk(0.8, func(s *synth) {
		for i, f := range [4]float64{1319, 1568, 1976, 2637} {
			t := float64(i) * 0.06
			s.tone(t, t+0.5, f, f, 0.18, 4, sine)
			s.tone(t, t+0.5, f/2, f/2, 0.1, 4, square)
		}
		s.hiss(0, 0.8, 7000, 1200, 0.3, 3)
	})
	m.clips[sfxLevel] = mk(0.9, func(s *synth) {
		for i, f := range [...]float64{523, 659, 784, 1047} {
			t := float64(i) * 0.09
			d := 0.25
			if i == 3 {
				d = 0.6
			}
			s.tone(t, t+d, f, f, 0.2, 3, square)
		}
	})
	m.clips[sfxStart] = mk(0.45, func(s *synth) {
		s.tone(0, 0.12, 784, 784, 0.2, 3, square)
		s.tone(0.12, 0.45, 1175, 1175, 0.2, 4, square)
	})
	m.clips[sfxPause] = mk(0.1, func(s *synth) {
		s.tone(0, 0.1, 1200, 800, 0.18, 5, square)
	})
	m.clips[sfxOver] = mk(1.4, func(s *synth) {
		for i, f := range [...]float64{392, 330, 277, 220} {
			t := float64(i) * 0.25
			s.tone(t, t+0.5, f, f*0.97, 0.22, 3, square)
		}
		s.hiss(0.9, 1.4, 1500, 300, 0.3, 3)
	})
}
