package main

import "math"

// The demo record is what plays when there is nothing else: a minute of
// slow electric piano over a bass and a brushed beat, synthesised at
// start-up. It needs no files and no ffmpeg, which is also what makes it
// the tests' record.

const demoSeconds = 64

func demoTrack() *track {
	return &track{title: "Side A, Groove 1", artist: "pikap", album: "The Demo Record", demo: true,
		dur: demoSeconds, artKey: "\x00demo"}
}

func demoRecord() *pcm {
	p := new(pcm)
	go func() {
		p.write(synthDemo(demoSeconds))
		p.finish(nil)
	}()
	return p
}

// synthDemo renders the piece: Am7 – Fmaj7 – Cmaj7 – G6 at 76 bpm.
func synthDemo(seconds int) []int16 {
	n := seconds * sampleRate
	l := make([]float32, n)
	r := make([]float32, n)
	const bpm = 76.0
	beat := 60 / bpm
	chords := [4][4]float64{
		{57, 60, 64, 67}, // A C E G
		{53, 57, 60, 64}, // F A C E
		{48, 55, 59, 64}, // C G B E
		{55, 59, 62, 64}, // G B D E
	}
	hz := func(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }
	seed := uint32(7)
	noise := func() float32 {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return float32(seed)/float32(math.MaxUint32)*2 - 1
	}
	// note adds a decaying tone with a few harmonics: an electric piano,
	// more or less.
	note := func(t0, dur, f, amp, pan float64, harm [3]float64) {
		i0 := int(t0 * sampleRate)
		i1 := min(n, i0+int(dur*sampleRate))
		for i := i0; i < i1; i++ {
			t := float64(i-i0) / sampleRate
			env := math.Min(1, t*200) * math.Exp(-t*2.2)
			v := math.Sin(2*math.Pi*f*t) + harm[0]*math.Sin(4*math.Pi*f*t)*math.Exp(-t*6) +
				harm[1]*math.Sin(6*math.Pi*f*t)*math.Exp(-t*9) + harm[2]*math.Sin(2*math.Pi*f*1.003*t)
			s := float32(v * env * amp)
			l[i] += s * float32(1-pan)
			r[i] += s * float32(1+pan)
		}
	}
	bars := int(float64(seconds) / (4 * beat))
	for bar := 0; bar < bars; bar++ {
		c := chords[bar%4]
		t := float64(bar) * 4 * beat
		// The chord, rolled, and again on the "and" of three.
		for k, m := range c {
			note(t+float64(k)*0.03, 3.5*beat, hz(m), 0.09, float64(k)/3-0.5, [3]float64{0.4, 0.15, 0.3})
			note(t+2.5*beat+float64(k)*0.02, 1.5*beat, hz(m), 0.05, float64(k)/3-0.5, [3]float64{0.4, 0.15, 0.3})
		}
		// A melody over it, from the chord's notes an octave up.
		pattern := [4]int{3, 2, 1, 2}
		if bar%2 == 1 {
			pattern = [4]int{1, 3, 2, 0}
		}
		for k, pi := range pattern {
			if (bar+k)%3 == 2 {
				continue
			}
			note(t+float64(k)*beat+0.5*beat, beat*1.2, hz(c[pi]+12), 0.07, 0.25, [3]float64{0.2, 0.05, 0.4})
		}
		// Bass: root on one, fifth on three.
		note(t, 1.8*beat, hz(c[0]-24), 0.3, 0, [3]float64{0.3, 0, 0})
		note(t+2*beat, 1.8*beat, hz(c[0]-17), 0.25, 0, [3]float64{0.3, 0, 0})
		for b := 0; b < 4; b++ {
			bt := t + float64(b)*beat
			if b == 0 || b == 2 {
				// Kick: a sine dropping in pitch.
				i0 := int(bt * sampleRate)
				ph := 0.0
				for i := i0; i < min(n, i0+sampleRate/4); i++ {
					tt := float64(i-i0) / sampleRate
					ph += (45 + 90*math.Exp(-tt*30)) / sampleRate
					s := float32(math.Sin(2*math.Pi*ph) * math.Exp(-tt*9) * 0.45)
					l[i] += s
					r[i] += s
				}
			} else {
				// Snare, brushed: filtered noise.
				i0 := int(bt * sampleRate)
				var y float32
				for i := i0; i < min(n, i0+sampleRate/5); i++ {
					tt := float64(i-i0) / sampleRate
					y += (noise() - y) * 0.4
					s := y * float32(math.Exp(-tt*16)*0.16)
					l[i] += s * 0.8
					r[i] += s
				}
			}
			// Hats on the eighths.
			for h := 0; h < 2; h++ {
				i0 := int((bt + float64(h)*beat/2) * sampleRate)
				var y, prev float32
				for i := i0; i < min(n, i0+sampleRate/20); i++ {
					tt := float64(i-i0) / sampleRate
					x := noise()
					y = x - prev // high-passed
					prev = x
					s := y * float32(math.Exp(-tt*60)*0.035)
					l[i] += s
					r[i] += s * 0.7
				}
			}
		}
	}
	out := make([]int16, 2*n)
	for i := 0; i < n; i++ {
		fade := float32(1)
		if tail := float64(n-i) / sampleRate; tail < 3 {
			fade = float32(tail / 3)
		}
		out[2*i] = clip16(float32(math.Tanh(float64(l[i] * fade))))
		out[2*i+1] = clip16(float32(math.Tanh(float64(r[i] * fade))))
	}
	return out
}
