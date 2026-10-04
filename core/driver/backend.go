//go:build unix

package driver

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// Backend coordinates terminal I/O, raw mode management, and the event bus loop.
type Backend struct {
	in         *os.File
	out        *os.File
	portableIO TerminalIO // If non-nil, we are in portable/remote mode
	state      *TermiosState
	events     chan Event
	done       chan struct{}
	sigWinch   chan os.Signal
	width      uint16 // cached for portable mode
	height     uint16 // cached for portable mode
	startOnce  sync.Once
	closeOnce  sync.Once
	closeErr   error
	inline     uint16 // non-zero: render in place, reserving this many rows
	mu         sync.RWMutex
	replies    replyCollector
	looping    atomic.Bool // the event loop that reads replies is running
	reader     *ttyReader  // reads the terminal; paused while it is released
	released   atomic.Bool // the terminal belongs to another program (Release)
	setUp      bool        // Setup has run
}

// SetInline switches the backend to inline rendering: no alternate screen, the
// frame occupying height rows where the cursor already is, and the drawn output
// left in the scrollback on exit. Zero restores full-screen behaviour.
//
// Must be called before Setup.
func (b *Backend) SetInline(height uint16) {
	b.mu.Lock()
	b.inline = height
	b.mu.Unlock()
}

// Inline reports the reserved row count, or zero for full-screen mode.
func (b *Backend) Inline() uint16 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.inline
}

// NewBackend creates a new TTY Driver/Backend instance.
func NewBackend(in, out *os.File) *Backend {
	return &Backend{
		in:     in,
		out:    out,
		events: make(chan Event, 128),
		done:   make(chan struct{}),
	}
}

// NewDriver creates a new TTY Driver instance (alias for NewBackend).
var NewDriver = NewBackend

// NewPortableBackend creates a portable or remote Driver/Backend instance.
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
		b.mu.Lock()
		b.width, b.height = w, h
		b.mu.Unlock()
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

// Setup switches the terminal into raw mode and sends screen setup escape codes
// (alternate screen buffer, hide cursor, SGR mouse tracking, focus in/out reporting, bracketed paste, disable auto-wrap).
//
// A second call does nothing: limoni.New sets the backend up and
// Program.RunTerminal sets up the backend it is given, and a second raw mode
// recorded the raw terminal as the one to restore, so Close left the shell
// with no echo and no line editing.
func (b *Backend) Setup() error {
	if b.setUp {
		return nil
	}
	b.setUp = true
	// Inline mode keeps the normal screen buffer and leaves auto-wrap on: the
	// frame lives among the user's scrollback rather than replacing it, and a
	// row that overflows should wrap the way ordinary terminal output does.
	setupCmds := fullScreenSetupCmds()
	if height := b.Inline(); height > 0 {
		setupCmds = inlineSetupCmds(height)
	}
	setupCmds = b.replies.withProbe(setupCmds)
	if b.portableIO != nil {
		_, err := b.portableIO.Write([]byte(setupCmds))
		return err
	}

	// Enter raw mode
	state, err := MakeRaw(int(b.in.Fd()))
	if err != nil {
		b.setUp = false
		return fmt.Errorf("failed to put terminal in raw mode: %w", err)
	}
	b.state = state

	// Emit terminal control escape codes:
	// \x1b[?1049h - Switch to alternate screen buffer
	// \x1b[?25l   - Hide cursor
	// \x1b[?1003h - Track all mouse movements and clicks (SGR)
	// \x1b[?1006h - Enable SGR mouse extension mode
	// \x1b[?1004h - Enable focus in/out reporting
	// \x1b[?2004h - Enable bracketed paste mode
	// \x1b[?7l    - Disable auto-wrap (prevents lower-right corner shifts)
	if _, err := b.out.WriteString(setupCmds); err != nil {
		b.Close()
		return fmt.Errorf("failed to send terminal setup codes: %w", err)
	}

	return nil
}

// Close restores the terminal to its canonical state and exits the alternate screen buffer.
func (b *Backend) Close() error {
	b.closeOnce.Do(func() {
		// Let answers to the startup queries arrive before the terminal is
		// restored, or they land in the shell as text.
		b.replies.drain(b.looping.Load())

		// Stop event loop
		select {
		case <-b.done:
		default:
			close(b.done)
		}

		if b.sigWinch != nil {
			signal.Stop(b.sigWinch)
		}

		restoreCmds := fullScreenRestoreCmds()
		if height := b.Inline(); height > 0 {
			// Park the cursor below the frame so the shell prompt lands after
			// it, and leave what was drawn on screen.
			restoreCmds = inlineRestoreCmds(height)
		}

		if b.portableIO != nil {
			_, b.closeErr = b.portableIO.Write([]byte(restoreCmds))
			return
		}

		b.out.WriteString(restoreCmds)

		// Restore canonical termios state
		if b.state != nil {
			b.closeErr = Restore(int(b.in.Fd()), b.state)
			b.state = nil
		}
	})
	return b.closeErr
}

// Events returns the receive-only channel of incoming driver events.
func (b *Backend) Events() <-chan Event {
	return b.events
}

// StartEventLoop starts the asynchronous event loop polling keyboard, mouse, focus, and resize events.
func (b *Backend) StartEventLoop() {
	b.startOnce.Do(func() {
		b.looping.Store(true)
		b.startEventLoop()
	})
}

func (b *Backend) startEventLoop() {
	if b.portableIO != nil {
		// No SIGWINCH from a portable terminal: poll its size instead.
		poll := func() (uint16, uint16, bool) {
			w, h, err := b.portableIO.Size()
			if err != nil {
				return 0, 0, false
			}
			b.mu.Lock()
			defer b.mu.Unlock()
			if w == b.width && h == b.height {
				return w, h, false
			}
			b.width, b.height = w, h
			return w, h, true
		}
		ticker := time.NewTicker(250 * time.Millisecond)
		go func() {
			defer ticker.Stop()
			b.parseInput(readChunks(b.portableIO, 1024, b.done), ticker.C, poll)
		}()
		return
	}

	// 1. Start the SIGWINCH (window resize) handler
	b.sigWinch = make(chan os.Signal, 1)
	signal.Notify(b.sigWinch, unix.SIGWINCH)

	// Protect the terminal when an external termination signal (SIGINT, SIGTERM) arrives
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, unix.SIGTERM)
	go func() {
		for {
			select {
			case sig := <-sigTerm:
				// Ctrl+C in a program the terminal was released to reaches
				// this process too; that program is the one to stop.
				if sig == os.Interrupt && b.released.Load() {
					continue
				}
				_ = b.Close()
				os.Exit(130)
			case <-b.done:
				signal.Stop(sigTerm)
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-b.sigWinch:
				w, h, err := b.Size()
				if err == nil {
					select {
					case b.events <- Event{
						Type: EventResize,
						Resize: ResizeEvent{
							Width:  w,
							Height: h,
						},
					}:
					case <-b.done:
						return
					default:
					}
				}
			case <-b.done:
				return
			}
		}
	}()

	// 2. Read the TTY and turn its bytes into events. The reader can be
	// paused, so Release can hand the terminal to another program.
	r, err := newTTYReader(b.in)
	if err != nil {
		go b.parseInput(readChunks(b.in, 512, b.done), nil, nil)
		return
	}
	b.reader = r
	go b.parseInput(r.chunks(512, b.done), nil, nil)
}

// Size returns the terminal window's current rows and columns.
func (b *Backend) Size() (uint16, uint16, error) {
	if b.portableIO != nil {
		b.mu.RLock()
		defer b.mu.RUnlock()
		return b.width, b.height, nil
	}

	ws, err := unix.IoctlGetWinsize(int(b.out.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return ws.Col, ws.Row, nil
}

// CellPixelSize returns the width and height of a terminal cell in pixels.
// If the terminal does not report pixel sizes, or an error occurs, it returns (10, 20).
func (b *Backend) CellPixelSize() (uint16, uint16, error) {
	if b.portableIO != nil {
		return 10, 20, nil
	}

	ws, err := unix.IoctlGetWinsize(int(b.out.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 10, 20, err
	}
	if ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 10, 20, nil
	}
	return ws.Xpixel / ws.Col, ws.Ypixel / ws.Row, nil
}

// Write writes data straight to the terminal output.
func (b *Backend) Write(p []byte) (int, error) {
	if b.portableIO != nil {
		return b.portableIO.Write(p)
	}
	return b.out.Write(p)
}

// StartSyncUpdate begins a synchronised update on modern terminals (\x1b[?2026h).
// This prevents tearing and flicker.
func (b *Backend) StartSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026h"))
		return
	}
	b.out.WriteString("\x1b[?2026h")
}

// EndSyncUpdate ends the synchronised update (\x1b[?2026l).
func (b *Backend) EndSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026l"))
		return
	}
	b.out.WriteString("\x1b[?2026l")
}
