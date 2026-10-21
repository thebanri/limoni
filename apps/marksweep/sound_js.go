//go:build js

package main

import (
	"encoding/binary"
	"math"
	"syscall/js"
)

// In a browser there is no player to pipe PCM into. The page makes a Web
// Audio context when the player clicks Run — browsers start sound only from a
// user gesture — and leaves it at window.__limoni_audio. Each clip is copied
// into it once, as an AudioBuffer; a sound is then a one-shot buffer source
// through a gain and a stereo panner, and the browser does the mixing.
//
// The game side is unchanged: play still hands a value to the channel and
// never waits. Calls into JavaScript allocate, so they stay on the mixer's
// goroutine, away from the frame.

func newBrowserMixer() *mixer {
	ctx := js.Global().Get("__limoni_audio")
	if !ctx.Truthy() {
		return nil
	}
	m := &mixer{req: make(chan sfxReq, 64), player: "Web Audio", done: make(chan struct{})}
	m.setVolume(1)
	m.synth()

	var buffers [nSfx]js.Value
	for i, clip := range m.clips {
		if len(clip) == 0 {
			continue
		}
		raw := make([]byte, len(clip)*4)
		for j, v := range clip {
			binary.LittleEndian.PutUint32(raw[j*4:], math.Float32bits(v))
		}
		bytes := js.Global().Get("Uint8Array").New(len(raw))
		js.CopyBytesToJS(bytes, raw)
		samples := js.Global().Get("Float32Array").New(bytes.Get("buffer"))
		buf := ctx.Call("createBuffer", 1, len(clip), sampleRate)
		buf.Call("copyToChannel", samples, 0)
		buffers[i] = buf
	}

	go func() {
		for {
			select {
			case r := <-m.req:
				if !buffers[r.s].Truthy() {
					continue
				}
				vol := r.vol * math.Float32frombits(m.master.Load())
				if vol <= 0 {
					continue
				}
				src := ctx.Call("createBufferSource")
				src.Set("buffer", buffers[r.s])
				gain := ctx.Call("createGain")
				gain.Get("gain").Set("value", vol)
				pan := ctx.Call("createStereoPanner")
				pan.Get("pan").Set("value", r.pan)
				src.Call("connect", gain)
				gain.Call("connect", pan)
				pan.Call("connect", ctx.Get("destination"))
				src.Call("start")
			case <-m.done:
				return
			}
		}
	}()
	return m
}
