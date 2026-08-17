package backend

import (
	"io"
)

// SSHSessionIO represents an active remote SSH terminal session with PTY support.
type SSHSessionIO interface {
	io.Reader
	io.Writer
	Size() (uint16, uint16, error)
}

// SSHBackend handles remote terminal I/O over SSH connections.
type SSHBackend = Backend

// NewSSHBackend creates a new Backend tailored for remote SSH PTY sessions.
func NewSSHBackend(session SSHSessionIO) *SSHBackend {
	return NewPortableBackend(session)
}
