//go:build unix

package backend

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"
)

// Backend terminal I/O işlemlerini, Raw Mode yönetimini ve Event Bus olay döngüsünü koordine eder.
type Backend struct {
	in       *os.File
	out      *os.File
	state    *TermiosState
	events   chan Event
	done     chan struct{}
	sigWinch chan os.Signal
}

// NewBackend yeni bir TTY Backend örneği oluşturur.
func NewBackend(in, out *os.File) *Backend {
	return &Backend{
		in:     in,
		out:    out,
		events: make(chan Event, 128),
		done:   make(chan struct{}),
	}
}

// Setup terminali Raw Mode'a geçirir ve ekran hazırlık kodlarını (Alt Screen, Mouse, Focus) gönderir.
func (b *Backend) Setup() error {
	// Raw Mode'a geçiş yap
	state, err := MakeRaw(int(b.in.Fd()))
	if err != nil {
		return fmt.Errorf("terminal raw moda gecirilemedi: %w", err)
	}
	b.state = state

	// Terminal kontrol kaçış kodlarını gönder:
	// \x1b[?1049h - Alternatif Ekran Tamponuna geç
	// \x1b[?25l   - İmleci gizle
	// \x1b[?1003h - Tüm fare hareketlerini ve tıklamalarını izle (SGR)
	// \x1b[?1006h - SGR fare uzantı modunu aç
	// \x1b[?1004h - Odaklanma (Focus In/Out) raporlamasını aç
	setupCmds := "\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h\x1b[?1004h"
	if _, err := b.out.WriteString(setupCmds); err != nil {
		b.Close()
		return fmt.Errorf("ekran hazirlik kodlari gonderilemedi: %w", err)
	}

	return nil
}

// Close terminali eski özgün ayarlarına döndürür ve alternatif ekrandan çıkar.
func (b *Backend) Close() error {
	// Olay döngüsünü durdur
	close(b.done)

	if b.sigWinch != nil {
		signal.Stop(b.sigWinch)
	}

	// Kontrol kaçış kodlarını geri al:
	// \x1b[?1004l - Odaklanma raporlamasını kapat
	// \x1b[?1006l - SGR fare modunu kapat
	// \x1b[?1003l - Fare takibini kapat
	// \x1b[?25h   - İmleci göster
	// \x1b[?1049l - Ana Ekran Tamponuna geri dön
	restoreCmds := "\x1b[?1004l\x1b[?1006l\x1b[?1003l\x1b[?25h\x1b[?1049l"
	b.out.WriteString(restoreCmds)

	// Raw Mode'dan çık, eski termios ayarlarına dön
	if b.state != nil {
		return Restore(int(b.in.Fd()), b.state)
	}
	return nil
}

// Events olay akışını dinleyen kanal alıcısını döner.
func (b *Backend) Events() <-chan Event {
	return b.events
}

// StartEventLoop klavye, fare, odak ve resize olaylarını yakalayan asenkron olay döngüsünü başlatır.
func (b *Backend) StartEventLoop() {
	// 1. SIGWINCH (Pencere boyut değişimi) yakalayıcıyı başlat
	b.sigWinch = make(chan os.Signal, 1)
	signal.Notify(b.sigWinch, unix.SIGWINCH)

	go func() {
		for {
			select {
			case <-b.sigWinch:
				w, h, err := b.Size()
				if err == nil {
					b.events <- Event{
						Type: EventResize,
						Resize: ResizeEvent{
							Width:  w,
							Height: h,
						},
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

			case chunk := <-inputChan:
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
						b.events <- ev
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

// Size terminal pencerisinin mevcut satır ve sütun boyutunu döner.
func (b *Backend) Size() (uint16, uint16, error) {
	ws, err := unix.IoctlGetWinsize(int(b.out.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return ws.Col, ws.Row, nil
}

// CellPixelSize terminal hücresinin piksel cinsinden genişlik ve yüksekliğini döner.
// Eğer terminal piksel bilgilerini raporlamıyorsa veya hata oluşursa varsayılan olarak (10, 20) döner.
func (b *Backend) CellPixelSize() (uint16, uint16, error) {
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
	return b.out.Write(p)
}

// StartSyncUpdate modern terminallerde senkron güncellemeyi başlatır (\x1b[?2026h).
// Bu ekran yırtılmalarını (tearing/flicker) engeller.
func (b *Backend) StartSyncUpdate() {
	b.out.WriteString("\x1b[?2026h")
}

// EndSyncUpdate senkron güncellemeyi kapatır (\x1b[?2026l).
func (b *Backend) EndSyncUpdate() {
	b.out.WriteString("\x1b[?2026l")
}
