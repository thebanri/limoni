//go:build !unix

package main

import (
	"errors"
	"os"
)

// openTTY is not supported here: on Windows, pass a file instead of a pipe.
func openTTY() (*os.File, error) {
	return nil, errors.New("reading a pipe is not supported on this platform; pass a file")
}
