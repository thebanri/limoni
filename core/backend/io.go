package backend

import (
	"bytes"
	"io"
	"sync"
)

// TerminalIO is the injectable terminal input/output boundary.
type TerminalIO interface {
	io.Reader
	io.Writer
	Size() (uint16, uint16, error)
}

// MemoryTerminalIO is a deterministic IO implementation for tests and PTY adapters.
type MemoryTerminalIO struct {
	mu            sync.Mutex
	in            *bytes.Reader
	out           bytes.Buffer
	Width, Height uint16
}

func NewMemoryTerminalIO(input []byte, width, height uint16) *MemoryTerminalIO {
	return &MemoryTerminalIO{in: bytes.NewReader(input), Width: width, Height: height}
}
func (m *MemoryTerminalIO) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.in.Read(p)
}
func (m *MemoryTerminalIO) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.out.Write(p)
}
func (m *MemoryTerminalIO) Size() (uint16, uint16, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Width, m.Height, nil
}
func (m *MemoryTerminalIO) Output() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]byte, m.out.Len())
	copy(result, m.out.Bytes())
	return result
}
