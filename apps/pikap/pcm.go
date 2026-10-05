package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os/exec"
	"sync/atomic"
)

// A record's sound is decoded once, in full, into memory: 16-bit stereo at
// 44.1 kHz, about 10 MB a minute. That is what lets the platter run
// backwards under a hand — a stream can only go forwards.
//
// ffmpeg does the decoding, so every format it reads plays (Opus, FLAC,
// MP3, AAC, Vorbis, WAV…). It writes into fixed-size chunks that are
// published by the frame count: the audio goroutine reads any frame below
// frames.Load() without a lock, because a chunk is filled before the count
// that covers it is stored.

const (
	sampleRate  = 44100
	chunkFrames = 1 << 15
	maxChunks   = 4 * 3600 * sampleRate / chunkFrames // four hours
)

type pcm struct {
	chunks [maxChunks]*[2 * chunkFrames]int16
	frames atomic.Int64 // decoded so far
	done   atomic.Bool  // no more will come
	err    atomic.Pointer[error]
	cancel context.CancelFunc
}

// at returns the frame at i, which must be below frames.Load().
func (p *pcm) at(i int64) (l, r float32) {
	c := p.chunks[i/chunkFrames]
	j := 2 * (i % chunkFrames)
	return float32(c[j]) / 32768, float32(c[j+1]) / 32768
}

// write appends interleaved samples; only the decoder calls it.
func (p *pcm) write(s []int16) {
	n := p.frames.Load()
	for len(s) >= 2 {
		ci := n / chunkFrames
		if ci >= maxChunks {
			return
		}
		if p.chunks[ci] == nil {
			p.chunks[ci] = new([2 * chunkFrames]int16)
		}
		off := 2 * (n % chunkFrames)
		k := copy(p.chunks[ci][off:], s)
		k &^= 1
		s = s[k:]
		n += int64(k / 2)
		p.frames.Store(n)
	}
}

func (p *pcm) finish(err error) {
	if err != nil {
		p.err.Store(&err)
	}
	p.done.Store(true)
}

func (p *pcm) Err() error {
	if e := p.err.Load(); e != nil {
		return *e
	}
	return nil
}

func (p *pcm) stop() {
	if p != nil && p.cancel != nil {
		p.cancel()
	}
}

var errNoFFmpeg = errors.New("pikap needs ffmpeg to read audio files")

// decodeFile starts ffmpeg on path and returns at once; the sound arrives
// in the background.
func decodeFile(path string) *pcm {
	p := new(pcm)
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		p.finish(errNoFFmpeg)
		return p
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	cmd := exec.CommandContext(ctx, bin, "-v", "error", "-nostdin", "-i", path,
		"-vn", "-f", "s16le", "-ac", "2", "-ar", "44100", "-")
	out, err := cmd.StdoutPipe()
	if err != nil {
		p.finish(err)
		return p
	}
	if err := cmd.Start(); err != nil {
		p.finish(err)
		return p
	}
	go func() {
		err := readPCM(bufio.NewReaderSize(out, 1<<16), p)
		werr := cmd.Wait()
		if err == nil && werr != nil && ctx.Err() == nil && p.frames.Load() == 0 {
			err = werr
		}
		p.finish(err)
	}()
	return p
}

func readPCM(r io.Reader, p *pcm) error {
	raw := make([]byte, 4*4096)
	s := make([]int16, 2*4096)
	for {
		n, err := io.ReadFull(r, raw)
		n &^= 3
		for i := 0; i < n/2; i++ {
			s[i] = int16(binary.LittleEndian.Uint16(raw[2*i:]))
		}
		p.write(s[:n/2])
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
