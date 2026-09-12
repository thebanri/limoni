//go:build js && wasm

package driver

import (
	"os"
	"syscall/js"
)

// Backend manages WebAssembly browser execution with xterm.js / DOM events.
type Backend struct {
	events chan Event
	done   chan struct{}
	width  uint16
	height uint16
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

// Terminal control sequences, identical to the Unix and Windows backends.
// xterm.js implements all of them, and omitting them here is what left the
// browser playground with a blinking cursor over the render, no mouse
// reporting, and auto-wrap corrupting full-width frames.
//
//	\x1b[?1049h - alternate screen buffer
//	\x1b[?25l   - hide cursor
//	\x1b[?1003h - track all mouse movement and clicks
//	\x1b[?1006h - SGR mouse extension
//	\x1b[?1004h - focus in/out reporting
//	\x1b[?2004h - bracketed paste
//	\x1b[?7l    - disable auto-wrap
const (
	wasmSetupCmds   = "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h\x1b[?2004h\x1b[?7l"
	wasmRestoreCmds = "\x1b[0m\x1b[?7h\x1b[?2004l\x1b[?1004l\x1b[?1006l\x1b[?1003l\x1b[?25h\x1b[?1049l"
)

// Setup initializes WASM JS callbacks and screen setup.
func (b *Backend) Setup() error {
	global := js.Global()
	if global.Truthy() {
		// Register a global JS callback for input injection: window.__limoni_input(data)
		inputCb := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				str := args[0].String()
				bytes := []byte(str)
				for len(bytes) > 0 {
					ev, consumed := ParseBracketedPaste(bytes)
					if consumed == 0 {
						ev, consumed = ParseEvent(bytes)
					}
					if consumed > 0 {
						select {
						case b.events <- ev:
						default:
						}
						bytes = bytes[consumed:]
					} else {
						break
					}
				}
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
	_, err := b.Write([]byte(wasmSetupCmds))
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
	_, err := b.Write([]byte(wasmRestoreCmds))
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
