package main

import (
	"io"
	"os/exec"
	"time"
)

// output is where the deck's sound goes: raw 16-bit stereo PCM at 44.1 kHz
// piped to whichever player the system has — pw-play (PipeWire), pacat
// (PulseAudio), aplay (ALSA) or sox's play — as in Castle Lemonstein and
// Lemon Drop. macOS and Windows use native audio (output_system.go).
type output struct {
	w      io.WriteCloser
	name   string
	cmd    *exec.Cmd
	finish func()
}

var players = [...][]string{
	{"pw-play", "--raw", "--rate", "44100", "--channels", "2", "--format", "s16", "--latency", "40ms", "-"},
	{"pacat", "--raw", "--rate=44100", "--channels=2", "--format=s16le", "--latency-msec=40"},
	{"aplay", "-q", "-t", "raw", "-f", "S16_LE", "-r", "44100", "-c", "2", "-B", "60000"},
	{"play", "-q", "-t", "raw", "-r", "44100", "-e", "signed", "-b", "16", "-c", "2", "-"},
}

// openOutput returns nil when there is nowhere to play to.
func openOutput() *output {
	if o := openSystemOutput(); o != nil {
		return o
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
		return &output{w: w, name: p[0], cmd: cmd}
	}
	return nil
}

func (o *output) close() {
	_ = o.w.Close()
	if o.finish != nil {
		o.finish()
	}
	if o.cmd == nil {
		return
	}
	done := make(chan struct{})
	go func() { _ = o.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		_ = o.cmd.Process.Kill()
	}
}
