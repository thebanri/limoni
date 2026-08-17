//go:build js && wasm

package backend

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

	return nil
}

// Close cleans up JS bindings and stops event delivery.
func (b *Backend) Close() error {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	return nil
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
