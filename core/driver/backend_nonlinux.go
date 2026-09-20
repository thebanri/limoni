//go:build !unix && !windows && !js

package driver

import (
	"io"
	"os"
	"sync"
)

// Backend is the portable IO/event shell. Platform-specific raw mode and
// signal implementations can wrap TerminalIO without changing the parser.
type Backend struct {
	in            io.Reader
	out           io.Writer
	width, height uint16
	events        chan Event
	done          chan struct{}
	inlineHeight  uint16
	inlineMu      sync.RWMutex
	replies       replyCollector // never sent: this backend writes no setup
}

func NewBackend(in, out *os.File) *Backend {
	return &Backend{in: in, out: out, events: make(chan Event, 128), done: make(chan struct{})}
}

func NewPortableBackend(io TerminalIO) *Backend {
	w, h, _ := io.Size()
	return &Backend{in: io, out: io, width: w, height: h, events: make(chan Event, 128), done: make(chan struct{})}
}

func (b *Backend) SetSize(w, h uint16) {
	b.width, b.height = w, h
}

func (b *Backend) Setup() error { return nil }
func (b *Backend) Close() error {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	return nil
}
func (b *Backend) Events() <-chan Event                   { return b.events }
func (b *Backend) StartEventLoop()                        {}
func (b *Backend) Size() (uint16, uint16, error)          { return b.width, b.height, nil }
func (b *Backend) CellPixelSize() (uint16, uint16, error) { return 10, 20, nil }
func (b *Backend) Write(p []byte) (int, error)            { return b.out.Write(p) }
func (b *Backend) StartSyncUpdate()                       {}
func (b *Backend) EndSyncUpdate()                         {}

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
