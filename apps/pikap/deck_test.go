package main

import (
	"math"
	"os"
	"testing"
	"time"
)

// full returns a record of n seconds of a steady ramp, decoded already, so
// a test can tell from a sample where the needle is.
func full(seconds float64) *pcm {
	p := new(pcm)
	n := int(seconds * sampleRate)
	s := make([]int16, 2*n)
	for i := 0; i < n; i++ {
		v := int16(i % 20000)
		s[2*i], s[2*i+1] = v, v
	}
	p.write(s)
	p.finish(nil)
	return p
}

// run steps the deck for the given seconds, 10 ms at a time.
func run(d *deck, seconds float64) {
	buf := make([]float32, 2*sampleRate/100)
	for t := 0.0; t < seconds; t += 0.01 {
		d.step(buf)
	}
}

func TestMotorSpinsUpAndPlaysInTime(t *testing.T) {
	d := newDeck()
	d.crackle.Store(false)
	d.load(full(10))
	d.motor.Store(true)
	run(d, 3)
	if s := d.platterSpeed(); math.Abs(s-1) > 0.01 {
		t.Fatalf("after 3 s the platter turns at %.3f of 33⅓, want 1", s)
	}
	// Spin-up costs a little under the time constant.
	if p := d.position(); p < 2.6 || p > 3 {
		t.Fatalf("after 3 s from rest the needle is at %.2f s, want 2.6…3", p)
	}
	// And the platter turned as far as the needle went.
	if a, p := d.platterAngle(), d.position(); math.Abs(a/rev33-p) > 0.01 {
		t.Fatalf("platter at %.3f turns, needle at %.3f s: they disagree", a, p)
	}
}

func TestPauseCoastsToAStop(t *testing.T) {
	d := newDeck()
	d.load(full(20))
	d.motor.Store(true)
	run(d, 2)
	d.motor.Store(false)
	run(d, 0.2)
	if s := d.platterSpeed(); s <= 0.2 || s >= 0.9 {
		t.Fatalf("0.2 s after pause the speed is %.2f: it should be slowing, not stopped or unchanged", s)
	}
	run(d, 2)
	if s := d.platterSpeed(); s != 0 {
		t.Fatalf("2 s after pause the platter still turns at %.3f", s)
	}
	p := d.position()
	run(d, 1)
	if d.position() != p {
		t.Fatal("a stopped record's needle moved")
	}
}

func TestPauseSlowsTheSound(t *testing.T) {
	// While the platter coasts down, the needle reads the ramp ever more
	// slowly: the samples of successive chunks climb by less each time.
	d := newDeck()
	d.crackle.Store(false)
	d.setVolume(1)
	d.load(full(20))
	d.motor.Store(true)
	run(d, 1)
	d.motor.Store(false)
	buf := make([]float32, 2*441)
	var prev float64 = -1
	var steps []float64
	for i := 0; i < 5; i++ {
		d.step(buf)
		p := d.position()
		if prev >= 0 {
			steps = append(steps, p-prev)
		}
		prev = p
	}
	for i := 1; i < len(steps); i++ {
		if steps[i] >= steps[i-1] {
			t.Fatalf("the needle does not slow down: steps %v", steps)
		}
	}
}

func TestHandPullsTheRecordBack(t *testing.T) {
	d := newDeck()
	d.load(full(20))
	d.motor.Store(true)
	run(d, 5)
	before := d.position()
	// A hand takes the record and pulls it back half a turn.
	a := d.platterAngle()
	d.holdAt(a)
	d.hand.Store(true)
	run(d, 0.05)
	d.holdAt(a - 0.5)
	run(d, 0.5)
	back := d.position()
	want := before - 0.5/rev33
	if math.Abs(back-want) > 0.15 {
		t.Fatalf("half a turn back from %.2f s: at %.2f s, want about %.2f", before, back, want)
	}
	// Let go, and the motor takes it forward again.
	d.hand.Store(false)
	run(d, 1)
	if d.position() <= back+0.5 {
		t.Fatalf("after letting go the record did not play on: %.2f → %.2f", back, d.position())
	}
}

func TestEndOfRecordIsNoticed(t *testing.T) {
	d := newDeck()
	d.load(full(1))
	d.motor.Store(true)
	run(d, 1.5)
	if !d.ended.Load() {
		t.Fatal("the needle ran off the end of the record and nobody noticed")
	}
}

func TestNeedleWaitsForDecoding(t *testing.T) {
	// A record still being decoded: the needle may not run past what has
	// arrived, and must not read chunks that are not there.
	p := new(pcm)
	p.write(make([]int16, 2*sampleRate/2)) // half a second
	d := newDeck()
	d.load(p)
	d.motor.Store(true)
	run(d, 2)
	if pos := d.position(); pos > 0.5 {
		t.Fatalf("the needle is at %.2f s with only 0.5 s decoded", pos)
	}
	if d.ended.Load() {
		t.Fatal("a record still arriving was taken as ended")
	}
}

func TestSeek(t *testing.T) {
	d := newDeck()
	d.load(full(30))
	d.seek(12.5)
	run(d, 0.01)
	if p := d.position(); math.Abs(p-12.5) > 0.01 {
		t.Fatalf("seek to 12.5 s: at %.3f", p)
	}
}

func TestDemoRecordIsMusicNotSilence(t *testing.T) {
	s := synthDemo(8)
	var peak, sum float64
	for _, v := range s {
		a := math.Abs(float64(v)) / 32768
		peak = max(peak, a)
		sum += a * a
	}
	rms := math.Sqrt(sum / float64(len(s)))
	if peak < 0.2 || peak > 0.999 || rms < 0.03 {
		t.Fatalf("demo record: peak %.3f, rms %.3f — silent or clipped", peak, rms)
	}
}

// TestOutputTakesTheStream sends half a second of silence to the system's
// player, which fails if it does not accept the format:
//
//	PIKAP_SOUND=1 go test -run TestOutputTakesTheStream
func TestOutputTakesTheStream(t *testing.T) {
	if os.Getenv("PIKAP_SOUND") == "" {
		t.Skip("set PIKAP_SOUND=1 to try the system's audio output")
	}
	out := openOutput()
	if out == nil {
		t.Skip("no audio output here")
	}
	d := newDeck()
	d.crackle.Store(false)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { d.run(out.w, stop); close(done) }()
	time.Sleep(500 * time.Millisecond)
	close(stop)
	<-done
	if out.cmd != nil && out.cmd.ProcessState != nil {
		t.Fatalf("%s exited: %v", out.name, out.cmd.ProcessState)
	}
	out.close()
	t.Logf("played through %s", out.name)
}
