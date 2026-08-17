package backend

import (
	"io"
	"sync"
	"time"
)

// SSHSessionIO represents an active remote SSH terminal session with PTY support.
type SSHSessionIO interface {
	io.Reader
	io.Writer
	Size() (uint16, uint16, error)
}

// SSHBackend handles remote terminal I/O over SSH connections.
type SSHBackend struct {
	session SSHSessionIO
	events  chan Event
	done    chan struct{}
	mu      sync.RWMutex
	width   uint16
	height  uint16
}

// NewSSHBackend creates a new Backend tailored for remote SSH PTY sessions.
func NewSSHBackend(session SSHSessionIO) *SSHBackend {
	w, h, _ := session.Size()
	if w == 0 || h == 0 {
		w, h = 80, 24
	}
	return &SSHBackend{
		session: session,
		events:  make(chan Event, 128),
		done:    make(chan struct{}),
		width:   w,
		height:  h,
	}
}

// Setup initializes alternate screen, mouse tracking, and focus reporting over SSH.
func (b *SSHBackend) Setup() error {
	setupCmds := "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h"
	_, err := b.session.Write([]byte(setupCmds))
	return err
}

// Close restores the terminal state and exits alternate screen.
func (b *SSHBackend) Close() error {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	restoreCmds := "\x1b[?1004l\x1b[?1006l\x1b[?1003l\x1b[?25h\x1b[?1049l"
	_, err := b.session.Write([]byte(restoreCmds))
	return err
}

// Events returns the event channel.
func (b *SSHBackend) Events() <-chan Event {
	return b.events
}

// StartEventLoop starts the asynchronous event reader for the SSH stream.
func (b *SSHBackend) StartEventLoop() {
	inputChan := make(chan []byte, 32)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := b.session.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				temp := make([]byte, n)
				copy(temp, buf[:n])
				select {
				case inputChan <- temp:
				case <-b.done:
					return
				}
			}
		}
	}()

	go func() {
		var readBuf []byte
		const escTimeoutDuration = 25 * time.Millisecond
		var escTimer *time.Timer
		var escTimerChan <-chan time.Time

		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-b.done:
				if escTimer != nil {
					escTimer.Stop()
				}
				return

			case <-ticker.C:
				if w, h, err := b.session.Size(); err == nil && (w != b.width || h != b.height) {
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
					case <-b.done:
						return
					}
				}

			case chunk := <-inputChan:
				readBuf = append(readBuf, chunk...)
				if escTimer != nil {
					escTimer.Stop()
					escTimer = nil
					escTimerChan = nil
				}

				for len(readBuf) > 0 {
					ev, consumed := ParseBracketedPaste(readBuf)
					if consumed == 0 {
						ev, consumed = ParseEvent(readBuf)
					}
					if consumed > 0 {
						b.events <- ev
						readBuf = readBuf[consumed:]
					} else {
						break
					}
				}

				if len(readBuf) == 1 && readBuf[0] == '\x1b' {
					escTimer = time.NewTimer(escTimeoutDuration)
					escTimerChan = escTimer.C
				}

			case <-escTimerChan:
				if len(readBuf) == 1 && readBuf[0] == '\x1b' {
					b.events <- Event{
						Type: EventKey,
						Key: KeyEvent{
							Type: KeyEsc,
						},
					}
					readBuf = readBuf[:0]
				}
				escTimer = nil
				escTimerChan = nil
			}
		}
	}()
}

// SetSize updates the dimensions (e.g. from an external SSH window change hook).
func (b *SSHBackend) SetSize(w, h uint16) {
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

// Size returns the session dimensions.
func (b *SSHBackend) Size() (uint16, uint16, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.width, b.height, nil
}

// CellPixelSize returns default cell pixel dimensions.
func (b *SSHBackend) CellPixelSize() (uint16, uint16, error) {
	return 10, 20, nil
}

// Write writes bytes to the remote SSH terminal.
func (b *SSHBackend) Write(p []byte) (int, error) {
	return b.session.Write(p)
}

// StartSyncUpdate sends DCS sync update escape sequence.
func (b *SSHBackend) StartSyncUpdate() {
	_, _ = b.session.Write([]byte("\x1b[?2026h"))
}

// EndSyncUpdate sends DCS sync update end escape sequence.
func (b *SSHBackend) EndSyncUpdate() {
	_, _ = b.session.Write([]byte("\x1b[?2026l"))
}
