//go:build windows

package driver

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"
)

// Backend manages console I/O, raw mode and the event loop on Windows.
type Backend struct {
	in           *os.File
	out          *os.File
	portableIO   TerminalIO
	state        *WindowsConsoleState
	events       chan Event
	done         chan struct{}
	width        uint16
	height       uint16
	startOnce    sync.Once
	closeOnce    sync.Once
	closeErr     error
	inlineHeight uint16
	inlineMu     sync.RWMutex
	replies      replyCollector
	looping      atomic.Bool
}

// NewBackend returns a new Windows Backend.
func NewBackend(in, out *os.File) *Backend {
	return &Backend{
		in:     in,
		out:    out,
		events: make(chan Event, 128),
		done:   make(chan struct{}),
	}
}

// NewPortableBackend returns a new portable (remote connection) Backend.
func NewPortableBackend(io TerminalIO) *Backend {
	w, h, _ := io.Size()
	if w == 0 || h == 0 {
		w, h = 80, 24
	}
	return &Backend{
		portableIO: io,
		events:     make(chan Event, 128),
		done:       make(chan struct{}),
		width:      w,
		height:     h,
	}
}

// SetSize updates the dimensions (e.g. from remote SSH resize).
func (b *Backend) SetSize(w, h uint16) {
	if b.portableIO != nil {
		b.width, b.height = w, h
		select {
		case b.events <- Event{
			Type: EventResize,
			Resize: ResizeEvent{
				Width:  w,
				Height: h,
			},
		}:
		default:
		}
	}
}

// Setup switches the terminal to raw / VT100 mode and sends the screen setup codes.
func (b *Backend) Setup() error {
	if b.portableIO != nil {
		setupCmds := fullScreenSetupCmds()
		if height := b.Inline(); height > 0 {
			setupCmds = inlineSetupCmds(height)
		}
		setupCmds = b.replies.withProbe(setupCmds)
		_, err := b.portableIO.Write([]byte(setupCmds))
		return err
	}

	state, err := MakeRaw(b.in.Fd(), b.out.Fd())
	if err != nil {
		return fmt.Errorf("windows konsolu raw moda gecirilemedi: %w", err)
	}
	b.state = state

	setupCmds := fullScreenSetupCmds()
	if height := b.Inline(); height > 0 {
		setupCmds = inlineSetupCmds(height)
	}
	setupCmds = b.replies.withProbe(setupCmds)
	if _, err := b.out.WriteString(setupCmds); err != nil {
		b.Close()
		return fmt.Errorf("ekran hazirlik kodlari gonderilemedi: %w", err)
	}

	return nil
}

// Close restores the terminal's previous settings and leaves the alternate screen.
func (b *Backend) Close() error {
	b.closeOnce.Do(func() {
		// Let answers to the startup queries arrive before the console is
		// restored, or they land in the shell as text.
		b.replies.drain(b.looping.Load())

		select {
		case <-b.done:
		default:
			close(b.done)
		}

		restoreCmds := fullScreenRestoreCmds()
		if height := b.Inline(); height > 0 {
			restoreCmds = inlineRestoreCmds(height)
		}
		if b.portableIO != nil {
			_, b.closeErr = b.portableIO.Write([]byte(restoreCmds))
			return
		}
		if b.out != nil {
			_, _ = b.out.WriteString(restoreCmds)
		}

		if b.state != nil {
			b.closeErr = RestoreConsole(b.state)
			b.state = nil
		}
	})
	return b.closeErr
}

// Events returns the channel that delivers events.
func (b *Backend) Events() <-chan Event {
	return b.events
}

// StartEventLoop starts the input and event loop on the Windows console.
func (b *Backend) StartEventLoop() {
	b.startOnce.Do(func() {
		b.looping.Store(true)
		b.startEventLoop()
	})
}

func (b *Backend) startEventLoop() {
	var r io.Reader
	switch {
	case b.portableIO != nil:
		r = b.portableIO
	case b.in != nil:
		r = b.in
	default:
		return
	}
	// The console sends no resize signal: poll its size.
	var lastW, lastH uint16
	if w, h, err := b.Size(); err == nil {
		lastW, lastH = w, h
	}
	poll := func() (uint16, uint16, bool) {
		w, h, err := b.Size()
		if err != nil || (w == lastW && h == lastH) {
			return w, h, false
		}
		lastW, lastH = w, h
		return w, h, true
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		b.parseInput(readChunks(r, 512, b.done), ticker.C, poll)
	}()
}

// Size returns the size of the console buffer.
func (b *Backend) Size() (uint16, uint16, error) {
	if b.portableIO != nil {
		w, h, err := b.portableIO.Size()
		if err == nil && w > 0 && h > 0 {
			b.width, b.height = w, h
		}
		if b.width == 0 || b.height == 0 {
			return 80, 24, nil
		}
		return b.width, b.height, nil
	}
	if b.out == nil {
		return 80, 24, nil
	}
	var csbi windows.ConsoleScreenBufferInfo
	err := windows.GetConsoleScreenBufferInfo(windows.Handle(b.out.Fd()), &csbi)
	if err != nil {
		return 80, 24, err
	}
	w := uint16(csbi.Window.Right - csbi.Window.Left + 1)
	h := uint16(csbi.Window.Bottom - csbi.Window.Top + 1)
	if w == 0 || h == 0 {
		return 80, 24, nil
	}
	return w, h, nil
}

// CellPixelSize returns the pixel size of a cell (the Windows default).
func (b *Backend) CellPixelSize() (uint16, uint16, error) {
	return 10, 20, nil
}

// Write writes straight to the console.
func (b *Backend) Write(p []byte) (int, error) {
	if b.portableIO != nil {
		return b.portableIO.Write(p)
	}
	if b.out != nil {
		return b.out.Write(p)
	}
	return 0, nil
}

// StartSyncUpdate begins a synchronised update (\x1b[?2026h).
func (b *Backend) StartSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026h"))
		return
	}
	if b.out != nil {
		_, _ = b.out.WriteString("\x1b[?2026h")
	}
}

// EndSyncUpdate ends the synchronised update (\x1b[?2026l).
func (b *Backend) EndSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026l"))
		return
	}
	if b.out != nil {
		_, _ = b.out.WriteString("\x1b[?2026l")
	}
}

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
