//go:build darwin || windows

package main

import (
	"io"
	"runtime"
	"time"

	"github.com/ebitengine/oto/v3"
)

// macOS and Windows do not ship the command-line PCM players used on Linux.
// Oto streams the same stereo mix to the system audio output without an
// external player or Cgo, including in cross-compiled binaries.
func newSystemMixer() *mixer {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   40 * time.Millisecond,
	})
	if err != nil {
		return nil
	}
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}

	r, w := io.Pipe()
	p := ctx.NewPlayer(r)
	p.SetBufferSize((sampleRate / 50) * 4) // 20 ms of 16-bit stereo PCM
	player := "macOS AudioToolbox"
	if runtime.GOOS == "windows" {
		player = "Windows audio"
	}
	m := &mixer{
		req: make(chan sfxReq, 64), w: w, player: player,
		done: make(chan struct{}),
		cleanup: func() {
			// Unblock a pending read before waiting for playback to stop.
			_ = r.Close()
			p.PauseAndStopReading()
		},
	}
	m.setVolume(1)
	m.synth()
	go m.run()
	p.Play()
	return m
}
