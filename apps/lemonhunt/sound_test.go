package main

import (
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"
)

func TestEverySoundIsSynthesised(t *testing.T) {
	m := &mixer{}
	m.synth()
	for s := sfx(0); s < nSfx; s++ {
		clip := m.clips[s]
		if len(clip) == 0 {
			t.Errorf("sound %d is empty", s)
			continue
		}
		peak := 0.0
		for _, v := range clip {
			peak = math.Max(peak, math.Abs(float64(v)))
		}
		if peak < 0.05 || peak > 2 {
			t.Errorf("sound %d peaks at %.3f", s, peak)
		}
	}
}

// The mixer, given a pipe for a player, turns a request into stereo sound
// on the side it came from.
func TestTheMixerPansAndPlays(t *testing.T) {
	r, w := io.Pipe()
	m := &mixer{req: make(chan sfxReq, 8), w: w, done: make(chan struct{})}
	m.setVolume(1)
	m.synth()
	go m.run()
	defer close(m.done)

	m.play(sfxSquirt, 1, -1) // hard left
	var left, right float64
	buf := make([]byte, 4096)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && left < 50 {
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i+3 < n; i += 4 {
			left += math.Abs(float64(int16(binary.LittleEndian.Uint16(buf[i:])))) / 32768
			right += math.Abs(float64(int16(binary.LittleEndian.Uint16(buf[i+2:])))) / 32768
		}
	}
	if left < 50 {
		t.Fatalf("heard %.1f on the left, want a squirt", left)
	}
	if right > left*0.01 {
		t.Errorf("a sound panned hard left reached the right: %.1f against %.1f", right, left)
	}
}

// Asking for a sound from the game never waits, even with the mixer stuck.
func TestPlayNeverBlocks(t *testing.T) {
	m := &mixer{req: make(chan sfxReq, 2)}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			m.play(sfxStep, 1, 0)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("play blocked on a full queue")
	}
}
