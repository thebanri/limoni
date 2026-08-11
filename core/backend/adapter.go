package backend

import "io"

// PTYAdapter is the portable boundary for SSH/PTY and ConPTY integrations.
// Platform implementations can provide lifecycle and size notifications
// without changing parser or runtime contracts.
type PTYAdapter interface {
	TerminalIO
	Start() error
	Stop() error
	Resize(width, height uint16) error
}

// MemoryPTYAdapter is a deterministic adapter for integration tests.
type MemoryPTYAdapter struct{ *MemoryTerminalIO }

func NewMemoryPTYAdapter(width, height uint16) *MemoryPTYAdapter {
	return &MemoryPTYAdapter{NewMemoryTerminalIO(nil, width, height)}
}
func (m *MemoryPTYAdapter) Start() error { return nil }
func (m *MemoryPTYAdapter) Stop() error  { return nil }
func (m *MemoryPTYAdapter) Resize(width, height uint16) error {
	m.Width, m.Height = width, height
	return nil
}

var _ io.ReadWriter = (*MemoryPTYAdapter)(nil)
