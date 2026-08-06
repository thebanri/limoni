//go:build linux

package backend

import (
	"golang.org/x/sys/unix"
)

// TermiosState terminalin önceki özgün termios ayarlarını saklar.
type TermiosState struct {
	termios unix.Termios
}

// MakeRaw terminali Raw Mode'a (ham mod) geçirir ve eski ayarları geri yüklemek üzere döner.
// Pure Go ve CGO içermeyen bu metod, doğrudan Linux TCGETS/TCSETS ioctl çağrılarını kullanır.
func MakeRaw(fd int) (*TermiosState, error) {
	// Mevcut terminal ayarlarını al
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	// Eski ayarları yedekle
	oldState := &TermiosState{termios: *termios}

	// Ham mod ayarlarını uygula
	raw := *termios

	// Giriş bayraklarını temizle (Input flags)
	// BRKINT: Kesme durumunda SIGINT gönderme
	// ICRNL: Carriage Return (\r) karakterini Line Feed (\n) yapma
	// INPCK: Parite kontrolünü kapat
	// ISTRIP: 8. biti silme
	// IXON: Yazılımsal akış kontrolünü (Ctrl-S / Ctrl-Q) devre dışı bırak
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON

	// Çıkış bayraklarını temizle (Output flags)
	// OPOST: Çıkış karakterleri üzerinde işlem yapmayı kapat (\n -> \r\n dönüşümü engellenir)
	raw.Oflag &^= unix.OPOST

	// Kontrol bayraklarını ayarla (Control flags)
	// CSIZE: Karakter boyut maskesini temizle
	// PARENB: Pariteyi devre dışı bırak
	// CS8: Karakter boyutunu 8-bit yap
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8

	// Yerel bayrakları temizle (Local flags)
	// ECHO: Girilen karakterlerin ekrana yazılmasını engelle
	// ECHONL: Satır sonu karakterini ekrana yazma
	// ICANON: Kanonik modu kapat (Girdi satır satır değil karakter karakter okunur)
	// ISIG: Sinyal karakterlerini (Ctrl-C: SIGINT, Ctrl-Z: SIGTSTP) devre dışı bırak
	// IEXTEN: Genişletilmiş girdi işlemlerini (Ctrl-V) kapat
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN

	// Okuma parametrelerini ayarla (Control Characters)
	// VMIN = 1: En az 1 byte girdi gelene kadar okumayı engelle (blocking read)
	// VTIME = 0: Bekleme zaman aşımını kapat
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	// Yeni ayarları terminale uygula (TCSETS - hemen uygula)
	err = unix.IoctlSetTermios(fd, unix.TCSETS, &raw)
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
	return unix.IoctlSetTermios(fd, unix.TCSETS, &state.termios)
}
