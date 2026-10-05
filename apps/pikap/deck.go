package main

import (
	"encoding/binary"
	"io"
	"math"
	"sync/atomic"
	"time"
)

// The deck is the turntable itself: a platter with inertia, a motor that
// drives it, a needle that plays whatever passes under it,
// and a hand that can stop or push the record. Sound follows the platter,
// so a record slowing down sounds like one, and a record pushed backwards
// plays backwards.
//
// The audio goroutine owns the physics. The interface sets what a person
// sets — motor, speed switch, hand, volume — through atomics and a command
// channel, and reads back where the platter and the needle are.

const (
	rev33 = 100.0 / 3 / 60 // revolutions a second at 33⅓

	spinUp   = 0.22 // time constant of the motor, seconds
	spinDown = 0.45 // of the platter coasting to a stop
	handTau  = 0.03 // how closely the platter follows a hand
)

type deckCmd struct {
	load *pcm
	seek float64 // seconds; ignored when load is set
}

type deck struct {
	cmds chan deckCmd

	// Set by the interface.
	motor   atomic.Bool
	hand    atomic.Bool   // a hand is on the record
	handAt  atomic.Uint64 // where it holds it, in revolutions (float64 bits)
	volume  atomic.Uint32 // float32 bits
	crackle atomic.Bool

	// Read by the interface.
	posOut   atomic.Uint64 // needle position, seconds
	angleOut atomic.Uint64 // platter angle, revolutions
	speedOut atomic.Uint64 // platter speed, 1 = 33⅓
	ended    atomic.Bool   // the needle reached the end of the record
	loaded   atomic.Pointer[pcm]

	// The audio goroutine's own.
	rec   *pcm
	pos   float64 // frames
	angle float64
	speed float64
	seed  uint32
	pop   float32 // the current crackle click, decaying
	hiss  float32
}

func newDeck() *deck {
	d := &deck{cmds: make(chan deckCmd, 16), seed: 0x9e3779b9}
	d.setVolume(0.8)
	d.crackle.Store(true)
	return d
}

func (d *deck) setVolume(v float32) { d.volume.Store(math.Float32bits(max(0, min(1, v)))) }
func (d *deck) getVolume() float32  { return math.Float32frombits(d.volume.Load()) }

func (d *deck) load(p *pcm) {
	d.loaded.Store(p)
	d.ended.Store(false)
	d.cmds <- deckCmd{load: p}
}

func (d *deck) seek(sec float64) {
	select {
	case d.cmds <- deckCmd{seek: max(0, sec)}:
	default:
	}
}

func (d *deck) holdAt(rev float64) { d.handAt.Store(math.Float64bits(rev)) }

func (d *deck) position() float64     { return math.Float64frombits(d.posOut.Load()) }
func (d *deck) platterAngle() float64 { return math.Float64frombits(d.angleOut.Load()) }
func (d *deck) platterSpeed() float64 { return math.Float64frombits(d.speedOut.Load()) }

func (d *deck) rnd() float32 {
	d.seed ^= d.seed << 13
	d.seed ^= d.seed >> 17
	d.seed ^= d.seed << 5
	return float32(d.seed)/float32(math.MaxUint32)*2 - 1
}

// step plays len(out)/2 frames into out, interleaved stereo.
func (d *deck) step(out []float32) {
	for {
		select {
		case c := <-d.cmds:
			if c.load != nil {
				d.rec = c.load
				d.pos = 0
			} else {
				d.pos = c.seek * sampleRate
			}
			continue
		default:
		}
		break
	}

	n := len(out) / 2
	dt := float64(n) / sampleRate
	const target = 1.0
	end := d.speed
	if d.hand.Load() {
		// The platter goes where the hand takes it, a little behind.
		want := math.Float64frombits(d.handAt.Load())
		v := (want - d.angle) / handTau / rev33
		end = max(-12, min(12, v))
	} else if d.motor.Load() {
		end = target + (d.speed-target)*math.Exp(-dt/spinUp)
	} else {
		end = d.speed * math.Exp(-dt/spinDown)
		// Friction takes the last of it, or it would coast forever.
		if f := 0.6 * dt; math.Abs(end) < f {
			end = 0
		} else {
			end -= math.Copysign(f, end)
		}
	}

	var total int64
	done := false
	if d.rec != nil {
		total = d.rec.frames.Load()
		done = d.rec.done.Load()
	}
	vol := math.Float32frombits(d.volume.Load())
	crackle := d.crackle.Load()
	start := d.speed
	for i := 0; i < n; i++ {
		s := start + (end-start)*float64(i+1)/float64(n)
		d.angle += s * rev33 / sampleRate
		var l, r float32
		if d.rec != nil {
			d.pos += s
			if d.pos < 0 {
				d.pos = 0
			}
			if hi := float64(total - 2); d.pos > hi {
				d.pos = max(0, hi)
				if done {
					d.ended.Store(true)
				}
			}
			if total >= 2 {
				j := int64(d.pos)
				f := float32(d.pos - float64(j))
				l0, r0 := d.rec.at(j)
				l1, r1 := d.rec.at(j + 1)
				l = l0 + (l1-l0)*f
				r = r0 + (r1-r0)*f
				// A record hardly turning is quieter too: below a crawl
				// the needle hardly moves. The slowest speed the motor
				// runs at is still heard in full.
				if a := float32(math.Abs(s)); a < 0.2 {
					g := a / 0.2
					l *= g
					r *= g
				}
			}
		}
		if crackle && d.rec != nil {
			a := float32(min(1.5, math.Abs(s)))
			if d.rnd() > 1-0.0006*a && d.rnd() > 0.8 {
				d.pop = (0.08 + 0.12*(d.rnd()+1)) * a
				if d.rnd() < 0 {
					d.pop = -d.pop
				}
			}
			d.hiss += (d.rnd()*0.006*a - d.hiss) * 0.35
			c := d.pop + d.hiss
			d.pop *= 0.72
			l += c
			r += c * 0.9
		}
		out[2*i] = l * vol
		out[2*i+1] = r * vol
	}
	d.speed = end
	d.posOut.Store(math.Float64bits(d.pos / sampleRate))
	d.angleOut.Store(math.Float64bits(d.angle))
	d.speedOut.Store(math.Float64bits(d.speed))
}

// run plays into w 10 ms at a time until stop closes, keeping about 60 ms
// ahead of the clock so a hand on the record is heard at once. With no
// output it keeps the same time silently, and the record still turns.
func (d *deck) run(w io.Writer, stop <-chan struct{}) {
	const chunk = sampleRate / 100
	buf := make([]float32, 2*chunk)
	raw := make([]byte, 4*chunk)
	start := time.Now()
	written := 0
	for {
		select {
		case <-stop:
			return
		default:
		}
		d.step(buf)
		if w != nil {
			for i, v := range buf {
				binary.LittleEndian.PutUint16(raw[2*i:], uint16(clip16(v)))
			}
			if _, err := w.Write(raw); err != nil {
				w = nil
			}
		}
		written += chunk
		ahead := time.Duration(written)*time.Second/sampleRate - time.Since(start)
		if ahead > 60*time.Millisecond {
			time.Sleep(ahead - 45*time.Millisecond)
		} else if ahead < -200*time.Millisecond {
			// The output stalled (a suspended laptop, a busy player): start
			// the clock again rather than rushing to catch up.
			start, written = time.Now(), 0
		}
	}
}

func clip16(v float32) int16 {
	switch {
	case v > 1:
		v = 1
	case v < -1:
		v = -1
	}
	return int16(v * 32767)
}
