//go:build freebsd || openbsd || netbsd || dragonfly

package backend

import (
	"golang.org/x/sys/unix"
)

// TermiosState terminalin önceki özgün termios ayarlarını saklar.
type TermiosState struct {
	termios unix.Termios
}

// MakeRaw terminali Raw Mode'a (ham mod) geçirir ve eski ayarları geri yüklemek üzere döner.
// BSD sistemlerinde TIOCGETA / TIOCSETA ioctl çağrılarını CGO'suz kullanır.
func MakeRaw(fd int) (*TermiosState, error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}

	oldState := &TermiosState{termios: *termios}
	raw := *termios

	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN

	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

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
