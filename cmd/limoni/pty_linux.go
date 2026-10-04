//go:build linux

package main

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// openPTY opens a pseudo-terminal pair and returns its controlling side and
// the path of the terminal side.
func openPTY() (*os.File, string, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, "", err
	}
	var n int
	err = controlFD(master, func(fd int) error {
		if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil { // unlockpt
			return err
		}
		var err error
		n, err = unix.IoctlGetInt(fd, unix.TIOCGPTN) // ptsname
		return err
	})
	if err != nil {
		master.Close()
		return nil, "", err
	}
	return master, "/dev/pts/" + strconv.Itoa(n), nil
}
