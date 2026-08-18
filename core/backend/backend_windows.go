//go:build windows

package backend

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// Backend Windows platformunda konsol I/O, Raw mode ve event döngüsünü yönetir.
type Backend struct {
	in         *os.File
	out        *os.File
	portableIO TerminalIO
	state      *WindowsConsoleState
	events     chan Event
	done       chan struct{}
	width      uint16
	height     uint16
}

// NewBackend yeni bir Windows Backend örneği oluşturur.
func NewBackend(in, out *os.File) *Backend {
	return &Backend{
		in:     in,
		out:    out,
		events: make(chan Event, 128),
		done:   make(chan struct{}),
	}
}

// NewPortableBackend yeni bir taşınabilir/uzaktan bağlantı Backend örneği oluşturur.
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

// Setup terminali Raw / VT100 moduna geçirir ve ekran hazırlık kodlarını gönderir.
func (b *Backend) Setup() error {
	if b.portableIO != nil {
		setupCmds := "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h"
		_, err := b.portableIO.Write([]byte(setupCmds))
		return err
	}

	state, err := MakeRaw(b.in.Fd(), b.out.Fd())
	if err != nil {
		return fmt.Errorf("windows konsolu raw moda gecirilemedi: %w", err)
	}
	b.state = state

	setupCmds := "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h"
	if _, err := b.out.WriteString(setupCmds); err != nil {
		b.Close()
		return fmt.Errorf("ekran hazirlik kodlari gonderilemedi: %w", err)
	}

	return nil
}

// Close terminali eski ayarlarına döndürür ve alternatif ekrandan çıkar.
func (b *Backend) Close() error {
	select {
	case <-b.done:
	default:
		close(b.done)
	}

	restoreCmds := "\x1b[?1004l\x1b[?1006l\x1b[?1003l\x1b[?25h\x1b[?1049l"
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte(restoreCmds))
		return nil
	}
	if b.out != nil {
		_, _ = b.out.WriteString(restoreCmds)
	}

	if b.state != nil {
		return RestoreConsole(b.state)
	}
	return nil
}

// Events olay akışını dinleyen kanal alıcısını döner.
func (b *Backend) Events() <-chan Event {
	return b.events
}

// StartEventLoop Windows konsolunda girdi ve olay döngüsünü başlatır.
func (b *Backend) StartEventLoop() {
	inputChan := make(chan []byte, 32)
	go func() {
		buf := make([]byte, 512)
		for {
			var n int
			var err error
			if b.portableIO != nil {
				n, err = b.portableIO.Read(buf)
			} else if b.in != nil {
				n, err = b.in.Read(buf)
			} else {
				return
			}
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

		// Periyodik pencere boyutu kontrolü (Windows için)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		var lastW, lastH uint16
		if w, h, err := b.Size(); err == nil {
			lastW, lastH = w, h
		}

		for {
			select {
			case <-b.done:
				if escTimer != nil {
					escTimer.Stop()
				}
				return

			case <-ticker.C:
				if w, h, err := b.Size(); err == nil && (w != lastW || h != lastH) {
					lastW, lastH = w, h
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

// Size konsol tamponu boyutunu döner.
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

// CellPixelSize hücresel piksel boyutunu döner (Windows varsayılanı).
func (b *Backend) CellPixelSize() (uint16, uint16, error) {
	return 10, 20, nil
}

// Write doğrudan konsola yazar.
func (b *Backend) Write(p []byte) (int, error) {
	if b.portableIO != nil {
		return b.portableIO.Write(p)
	}
	if b.out != nil {
		return b.out.Write(p)
	}
	return 0, nil
}

// StartSyncUpdate senkron güncellemeyi başlatır (\x1b[?2026h).
func (b *Backend) StartSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026h"))
		return
	}
	if b.out != nil {
		_, _ = b.out.WriteString("\x1b[?2026h")
	}
}

// EndSyncUpdate senkron güncellemeyi kapatır (\x1b[?2026l).
func (b *Backend) EndSyncUpdate() {
	if b.portableIO != nil {
		_, _ = b.portableIO.Write([]byte("\x1b[?2026l"))
		return
	}
	if b.out != nil {
		_, _ = b.out.WriteString("\x1b[?2026l")
	}
}
