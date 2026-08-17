//go:build darwin

package backend

import (
	"golang.org/x/sys/unix"
)

// TermiosState terminalin önceki özgün termios ayarlarını saklar.
type TermiosState struct {
	termios unix.Termios
}

// MakeRaw terminali Raw Mode'a (ham mod) geçirir ve eski ayarları geri yüklemek üzere döner.
// macOS (Darwin) üzerinde TIOCGETA / TIOCSETA ioctl çağrılarını CGO'suz kullanır.
func MakeRaw(fd int) (*TermiosState, error) {
	// Mevcut terminal ayarlarını al
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}

	// Eski ayarları yedekle
	oldState := &TermiosState{termios: *termios}

	// Ham mod ayarlarını uygula
	raw := *termios

	// Giriş bayraklarını temizle (Input flags)
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON

	// Çıkış bayraklarını temizle (Output flags)
	raw.Oflag &^= unix.OPOST

	// Kontrol bayraklarını ayarla (Control flags)
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8

	// Yerel bayrakları temizle (Local flags)
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN

	// Okuma parametrelerini ayarla (Control Characters)
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	// Yeni ayarları terminale uygula (TIOCSETA - hemen uygula)
	err = unix.IoctlSetTermios(fd, unix.TIOCSETA, &raw)
	if err != nil {
		return nil, err
	}

	return oldState, nil
}

// Restore terminali eski özgün ayarlarına döndürür.
func Restore(fd int, state *TermiosState) error {
	if state == nil {
		return nil
	}
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, &state.termios)
}
