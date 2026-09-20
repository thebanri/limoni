//go:build unix

package main

import "os"

// openTTY opens the controlling terminal, for keys while stdin is a pipe.
func openTTY() (*os.File, error) { return os.OpenFile("/dev/tty", os.O_RDWR, 0) }
