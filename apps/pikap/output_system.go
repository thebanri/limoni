//go:build darwin || windows

package main

import (
	"io"
	"runtime"
	"time"

	"github.com/ebitengine/oto/v3"
)

// macOS and Windows do not ship the command-line players used on Linux.
// Oto streams the same PCM to the system output without Cgo.
func openSystemOutput() *output {
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
	p.SetBufferSize((sampleRate / 50) * 4)
	name := "macOS AudioToolbox"
	if runtime.GOOS == "windows" {
		name = "Windows audio"
	}
	p.Play()
	return &output{w: w, name: name, finish: func() {
		_ = r.Close()
		p.PauseAndStopReading()
	}}
}
