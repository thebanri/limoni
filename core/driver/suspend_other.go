//go:build !unix

package driver

import "errors"

// ErrSuspendUnsupported is returned by Suspend on platforms with no job
// control to hand the terminal back to.
var ErrSuspendUnsupported = errors.New("limoni: suspend is not supported on this platform")

// Suspend is not supported here.
func (b *Backend) Suspend() error { return ErrSuspendUnsupported }
