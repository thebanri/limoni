//go:build unix

package driver

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
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
	mu         sync.RWMutex
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
func (b *Backend) Setup() error {
	setupCmds := "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h\x1b[?2004h\x1b[?7l"
	if b.portableIO != nil {
		_, err := b.portableIO.Write([]byte(setupCmds))
		return err
	}

	// Enter raw mode
	state, err := MakeRaw(int(b.in.Fd()))
	if err != nil {
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
		// Stop event loop
		select {
		case <-b.done:
		default:
			close(b.done)
		}

		if b.sigWinch != nil {
			signal.Stop(b.sigWinch)
		}

		restoreCmds := "\x1b[0m\x1b[?7h\x1b[?2004l\x1b[?1004l\x1b[?1006l\x1b[?1003l\x1b[?25h\x1b[?1049l"

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
		b.startEventLoop()
	})
}

func (b *Backend) startEventLoop() {
	if b.portableIO != nil {
		inputChan := make(chan []byte, 32)
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := b.portableIO.Read(buf)
				if err != nil {
					close(inputChan)
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
					if w, h, err := b.portableIO.Size(); err == nil {
						b.mu.Lock()
						if w != b.width || h != b.height {
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
						} else {
							b.mu.Unlock()
						}
					}

				case chunk, ok := <-inputChan:
					if !ok {
						return
					}
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
							select {
							case b.events <- ev:
							case <-b.done:
								return
							}
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
						select {
						case b.events <- Event{
							Type: EventKey,
							Key: KeyEvent{
								Type: KeyEsc,
							},
						}:
						case <-b.done:
							return
						}
						readBuf = readBuf[:0]
					}
					escTimer = nil
					escTimerChan = nil
				}
			}
		}()
		return
	}

	// 1. SIGWINCH (Pencere boyut değişimi) yakalayıcıyı başlat
	b.sigWinch = make(chan os.Signal, 1)
	signal.Notify(b.sigWinch, unix.SIGWINCH)

	// Harici sonlandırma sinyalleri (SIGINT, SIGTERM) geldiğinde terminali koru
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, unix.SIGTERM)
	go func() {
		select {
		case <-sigTerm:
			_ = b.Close()
			os.Exit(130)
		case <-b.done:
			signal.Stop(sigTerm)
			return
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

	// 2. TTY Girdi Okuyucu ve ESC Zaman Aşımı Olay Döngüsünü başlat
	inputChan := make(chan []byte, 32)
	go func() {
		buf := make([]byte, 512)
		for {
			n, err := b.in.Read(buf)
			if err != nil {
				// Hata durumunda veya dosya kapandığında okuyucu goroutine sonlanır
				close(inputChan)
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

		for {
			select {
			case <-b.done:
				if escTimer != nil {
					escTimer.Stop()
				}
				return

			case chunk, ok := <-inputChan:
				if !ok {
					return
				}
				readBuf = append(readBuf, chunk...)

				// Eğer ESC zamanlayıcı aktifse durdur (yeni karakter geldi, escape sequence devam ediyor olabilir)
				if escTimer != nil {
					escTimer.Stop()
					escTimer = nil
					escTimerChan = nil
				}

				// Tamponu ayrıştır
				for len(readBuf) > 0 {
					ev, consumed := ParseBracketedPaste(readBuf)
					if consumed == 0 {
						ev, consumed = ParseEvent(readBuf)
					}
					if consumed > 0 {
						if ev.Type != EventNone {
							select {
							case b.events <- ev:
							case <-b.done:
								return
							}
						}
						readBuf = readBuf[consumed:]
					} else {
						// Tamamlanmamış bir dizi var
						break
					}
				}

				// Eğer tamponda sadece tek bir '\x1b' (Escape) kaldıysa, ESC tuşu olup olmadığını
				// anlamak için bir zaman aşımı başlatıyoruz.
				if len(readBuf) == 1 && readBuf[0] == '\x1b' {
					escTimer = time.NewTimer(escTimeoutDuration)
					escTimerChan = escTimer.C
				}

			case <-escTimerChan:
				// Zaman aşımı doldu ve yeni byte gelmedi. Bu durumda tamponda bekleyen '\x1b'
				// doğrudan ESC tuşu basımı olarak kabul edilir.
				if len(readBuf) == 1 && readBuf[0] == '\x1b' {
					select {
					case b.events <- Event{
						Type: EventKey,
						Key: KeyEvent{
							Type: KeyEsc,
						},
					}:
					case <-b.done:
						return
					}
					readBuf = readBuf[:0]
				}
				escTimer = nil
				escTimerChan = nil
			}
		}
	}()
}

// Size terminal pencerisinin mevcut satır ve sütun boyutunu döner.
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

// CellPixelSize terminal hücresinin piksel cinsinden genişlik ve yüksekliğini döner.
// Eğer terminal piksel bilgilerini raporlamıyorsa veya hata oluşursa varsayılan olarak (10, 20) döner.
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

// Write doğrudan terminal çıkışına veri yazar.
func (b *Backend) Write(p []byte) (int, error) {
	if b.portableIO != nil {
		return b.portableIO.Write(p)
	}
	return b.out.Write(p)
}

// StartSyncUpdate modern terminallerde senkron güncellemeyi başlatır (\x1b[?2026h).
// Bu ekran yırtılmalarını (tearing/flicker) engeller.
func (b *Backend) StartSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026h"))
		return
	}
	b.out.WriteString("\x1b[?2026h")
}

// EndSyncUpdate senkron güncellemeyi kapatır (\x1b[?2026l).
func (b *Backend) EndSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026l"))
		return
	}
	b.out.WriteString("\x1b[?2026l")
}
