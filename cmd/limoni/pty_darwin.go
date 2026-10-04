//go:build darwin

package main

import (
	"bytes"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// openPTY opens a pseudo-terminal pair and returns its controlling side and
// the path of the terminal side.
func openPTY() (*os.File, string, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, "", err
	}
	var name [128]byte
	err = controlFD(master, func(fd int) error {
		if err := unix.IoctlSetInt(fd, unix.TIOCPTYGRANT, 0); err != nil { // grantpt
			return err
		}
		if err := unix.IoctlSetInt(fd, unix.TIOCPTYUNLK, 0); err != nil { // unlockpt
			return err
		}
		// ptsname: TIOCPTYGNAME writes the path into a 128-byte buffer.
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0])))
		if errno != 0 {
			return errno
		}
		return nil
	})
	if err != nil {
		master.Close()
		return nil, "", err
	}
	return master, string(name[:bytes.IndexByte(name[:], 0)]), nil
}
