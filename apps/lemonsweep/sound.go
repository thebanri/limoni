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

// Copied from Castle Lemonstein, with its own sounds.
//
// Sound is synthesised at start-up — no sample files — and streamed as raw
// 16-bit stereo PCM to whichever player the system has: pw-play (PipeWire),
// pacat (PulseAudio), aplay (ALSA) or sox's play. macOS and Windows use native
// audio (sound_system.go), and browsers use Web Audio (sound_js.go).
// Where there is no audio output the game is silent and says
// so on exit.
//
// The game side of it allocates nothing: play hands a small value to the
// mixer's channel and never waits. The mixer runs on its own goroutine.

type sfx uint8

const (
	sfxShot   sfx = iota // the sweeper fires
	sfxHit               // a shot lands
	sfxPop               // a widget comes apart
	sfxBoom              // panic(): an explosion
	sfxZap               // a goroutine chain
	sfxPing              // a shot off a wall
	sfxReload            // the collector starts
	sfxReady             // the heap is clean again
	sfxHurt              // the player is hit
	sfxLevel             // a level up: cards
	sfxPick              // a card taken
	sfxRoar              // a boss arrives
	sfxDeath             // the player's last frame
	sfxDash              // a dash
	sfxByte              // freed bytes picked up
	sfxSpit              // an enemy fires
	sfxMenu              // a menu moves
	sfxCharge            // an enemy winds up an attack
	sfxFreeze            // a foe freezes
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
	req     chan sfxReq
	clips   [nSfx][]float32
	cmd     *exec.Cmd
	w       io.WriteCloser
	player  string
	master  atomic.Uint32 // float32 bits
	once    sync.Once
	done    chan struct{}
	cleanup func() // release a native output after closing the PCM stream
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

// newMixer prefers browser or native audio, then tries external players.
// It returns nil if no audio output can be started.
func newMixer() *mixer {
	if m := newBrowserMixer(); m != nil {
		return m
	}
	if m := newSystemMixer(); m != nil {
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
			_ = w.Close()
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
		if m.w != nil {
			_ = m.w.Close()
		}
		if m.cleanup != nil {
			m.cleanup()
		}
		if m.cmd == nil {
			return // native and browser audio have no process to wait for
		}
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
	m.clips[sfxShot] = mk(0.18, func(s *synth) {
		s.tone(0, 0.16, 190, 48, 0.55, 5, sine)     // the thump
		s.hiss(0, 0.06, 9500, 2600, 0.5, 11)        // the crack
		s.tone(0, 0.05, 1300, 520, 0.12, 8, square) // the action
		s.hiss(0.02, 0.18, 1800, 300, 0.18, 6)      // the tail in the room
	})
	m.clips[sfxHit] = mk(0.08, func(s *synth) {
		s.tone(0, 0.08, 520, 260, 0.25, 7, square)
	})
	m.clips[sfxPop] = mk(0.32, func(s *synth) {
		s.tone(0, 0.12, 900, 1800, 0.22, 3, sine)
		s.hiss(0, 0.3, 3000, 400, 0.35, 6)
		s.tone(0.05, 0.3, 300, 120, 0.2, 5, square)
	})
	m.clips[sfxBoom] = mk(0.6, func(s *synth) {
		s.hiss(0, 0.6, 1200, 80, 0.9, 4)
		s.tone(0, 0.5, 110, 40, 0.6, 4, sine)
	})
	m.clips[sfxZap] = mk(0.22, func(s *synth) {
		for t := 0.0; t < 0.2; t += 0.04 {
			s.tone(t, t+0.04, 2400, 900, 0.14, 2, saw)
		}
		s.hiss(0, 0.2, 9000, 4000, 0.15, 5)
	})
	m.clips[sfxPing] = mk(0.12, func(s *synth) {
		s.tone(0, 0.12, 2900, 2600, 0.12, 6, sine)
	})
	m.clips[sfxReload] = mk(0.4, func(s *synth) {
		s.tone(0, 0.08, 300, 300, 0.25, 4, square)
		s.hiss(0.08, 0.4, 600, 3000, 0.25, 2)
	})
	m.clips[sfxReady] = mk(0.25, func(s *synth) {
		s.tone(0, 0.06, 900, 900, 0.25, 3, square)
		s.tone(0.08, 0.2, 1400, 1400, 0.22, 4, square)
	})
	m.clips[sfxHurt] = mk(0.35, func(s *synth) {
		s.tone(0, 0.35, 180, 70, 0.55, 3, square)
		s.hiss(0, 0.2, 900, 200, 0.4, 5)
	})
	m.clips[sfxLevel] = mk(0.8, func(s *synth) {
		for i, f := range [4]float64{523, 659, 784, 1047} {
			t := float64(i) * 0.09
			s.tone(t, t+0.45, f, f, 0.2, 3, square)
			s.tone(t, t+0.45, f*2, f*2, 0.05, 4, sine)
		}
	})
	m.clips[sfxPick] = mk(0.3, func(s *synth) {
		s.tone(0, 0.3, 700, 1400, 0.25, 3, sine)
		s.tone(0, 0.15, 1400, 1400, 0.1, 5, square)
	})
	m.clips[sfxRoar] = mk(1.4, func(s *synth) {
		s.tone(0, 1.4, 80, 45, 0.5, 1.2, func(p float64) float64 { return saw(p) + 0.4*square(p*1.5) })
		s.hiss(0, 1.4, 600, 150, 0.45, 1.5)
	})
	m.clips[sfxDeath] = mk(1.6, func(s *synth) {
		for i, f := range [4]float64{660, 520, 390, 260} {
			t := float64(i) * 0.25
			s.tone(t, t+0.5, f, f*0.9, 0.25, 3, square)
		}
		s.hiss(0.9, 1.6, 500, 60, 0.4, 2)
	})
	m.clips[sfxDash] = mk(0.25, func(s *synth) {
		s.hiss(0, 0.25, 600, 6000, 0.4, 3)
	})
	m.clips[sfxByte] = mk(0.1, func(s *synth) {
		s.tone(0, 0.1, 1800, 2600, 0.1, 5, sine)
	})
	m.clips[sfxSpit] = mk(0.2, func(s *synth) {
		s.tone(0, 0.2, 400, 900, 0.18, 4, saw)
		s.hiss(0, 0.1, 2000, 800, 0.2, 6)
	})
	m.clips[sfxCharge] = mk(0.55, func(s *synth) {
		s.tone(0, 0.5, 160, 640, 0.28, 0.8, saw)
		s.tone(0, 0.5, 320, 1280, 0.1, 1, square)
	})
	m.clips[sfxFreeze] = mk(0.3, func(s *synth) {
		s.tone(0, 0.3, 3200, 2400, 0.12, 5, sine)
		s.hiss(0, 0.2, 9000, 6000, 0.2, 7)
	})
	m.clips[sfxMenu] = mk(0.06, func(s *synth) {
		s.tone(0, 0.06, 1200, 1200, 0.15, 6, square)
	})
}
