//go:build !linux

package backend

import (
	"io"
	"os"
)

// Backend is the portable IO/event shell. Platform-specific raw mode and
// signal implementations can wrap TerminalIO without changing the parser.
type Backend struct {
	in            io.Reader
	out           io.Writer
	width, height uint16
	events        chan Event
	done          chan struct{}
}

func NewBackend(in, out *os.File) *Backend {
	return &Backend{in: in, out: out, events: make(chan Event, 128), done: make(chan struct{})}
}

func NewPortableBackend(io TerminalIO) *Backend {
	w, h, _ := io.Size()
	return &Backend{in: io, out: io, width: w, height: h, events: make(chan Event, 128), done: make(chan struct{})}
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
