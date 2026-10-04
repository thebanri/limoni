//go:build !unix

package driver

import "errors"

// ErrReleaseUnsupported is returned by Release where the terminal cannot be
// handed to another program. On Windows the console reader cannot yet be
// stopped without losing input, and the browser has no other program to run.
var ErrReleaseUnsupported = errors.New("limoni: the terminal cannot be handed to another program on this platform")

// Release is not supported here.
func (b *Backend) Release(fn func() error) error { return ErrReleaseUnsupported }
