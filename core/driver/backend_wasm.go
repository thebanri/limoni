//go:build js && wasm

package driver

import (
	"os"
	"sync"
	"syscall/js"
)

// Backend manages WebAssembly browser execution with xterm.js / DOM events.
type Backend struct {
	events       chan Event
	done         chan struct{}
	width        uint16
	height       uint16
	inlineHeight uint16
	inlineMu     sync.RWMutex
	replies      replyCollector
}

// NewBackend creates a new WASM Backend instance.
func NewBackend(in, out *os.File) *Backend {
	return &Backend{
		events: make(chan Event, 128),
		done:   make(chan struct{}),
		width:  80,
		height: 24,
	}
}

// NewPortableBackend creates a portable WASM Backend instance.
func NewPortableBackend(io TerminalIO) *Backend {
	w, h, _ := io.Size()
	if w == 0 || h == 0 {
		w, h = 80, 24
	}
	return &Backend{
		events: make(chan Event, 128),
		done:   make(chan struct{}),
		width:  w,
		height: h,
	}
}

// SetSize updates the dimensions in WASM.
func (b *Backend) SetSize(w, h uint16) {
	b.width = w
	b.height = h
}

// The terminal control sequences are the ones every backend sends; see
// fullScreenSetupCmds. xterm.js implements them, and omitting them is what
// once left the browser playground with a blinking cursor over the render, no
// mouse reporting, and auto-wrap corrupting full-width frames.

// Setup initializes WASM JS callbacks and screen setup.
func (b *Backend) Setup() error {
	global := js.Global()
	if global.Truthy() {
		// Register a global JS callback for input injection: window.__limoni_input(data)
		inputCb := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				parseChunk([]byte(args[0].String()), func(ev Event) {
					if b.replies.record(ev) {
						return
					}
					select {
					case b.events <- ev:
					default:
					}
				})
			}
			return nil
		})
		global.Set("__limoni_input", inputCb)

		// Register a global JS callback for resize: window.__limoni_resize(w, h)
		resizeCb := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) >= 2 {
				w := uint16(args[0].Int())
				h := uint16(args[1].Int())
				b.width, b.height = w, h
				select {
				case b.events <- Event{
					Type:   EventResize,
					Resize: ResizeEvent{Width: w, Height: h},
				}:
				default:
				}
			}
			return nil
		})
		global.Set("__limoni_resize", resizeCb)
	}

	// Written after the callbacks are registered, so the output bridge is in
	// place by the time the first bytes are emitted.
	setup := fullScreenSetupCmds()
	if height := b.Inline(); height > 0 {
		setup = inlineSetupCmds(height)
	}
	// xterm.js answers the probe. There is no shell to leak late replies
	// into, so Close does not wait for them here.
	setup = b.replies.withProbe(setup)
	_, err := b.Write([]byte(setup))
	return err
}

// Close cleans up JS bindings and stops event delivery.
func (b *Backend) Close() error {
	select {
	case <-b.done:
		return nil
	default:
		close(b.done)
	}
	restore := fullScreenRestoreCmds()
	if height := b.Inline(); height > 0 {
		restore = inlineRestoreCmds(height)
	}
	_, err := b.Write([]byte(restore))
	return err
}

// Events returns the event channel.
func (b *Backend) Events() <-chan Event {
	return b.events
}

// StartEventLoop is a no-op on WASM since input is delivered via JS callbacks.
func (b *Backend) StartEventLoop() {}

// Size returns the terminal dimensions.
func (b *Backend) Size() (uint16, uint16, error) {
	if b.width == 0 || b.height == 0 {
		return 80, 24, nil
	}
	return b.width, b.height, nil
}

// CellPixelSize returns default cell pixel dimensions.
func (b *Backend) CellPixelSize() (uint16, uint16, error) {
	return 10, 20, nil
}

// Write outputs ANSI bytes to stdout / JS terminal.
func (b *Backend) Write(p []byte) (int, error) {
	global := js.Global()
	if global.Truthy() && global.Get("__limoni_output").Truthy() {
		global.Call("__limoni_output", string(p))
		return len(p), nil
	}
	return os.Stdout.Write(p)
}

// StartSyncUpdate is a no-op on WASM.
func (b *Backend) StartSyncUpdate() {}

// EndSyncUpdate is a no-op on WASM.
func (b *Backend) EndSyncUpdate() {}

// SetInline switches the backend to inline rendering: no alternate screen, the
// frame occupying height rows where the cursor already is, and the drawn output
// left in the scrollback on exit. Zero restores full-screen behaviour.
//
// Must be called before Setup.
func (b *Backend) SetInline(height uint16) {
	b.inlineMu.Lock()
	b.inlineHeight = height
	b.inlineMu.Unlock()
}

// Inline reports the reserved row count, or zero for full-screen mode.
func (b *Backend) Inline() uint16 {
	b.inlineMu.RLock()
	defer b.inlineMu.RUnlock()
	return b.inlineHeight
}
