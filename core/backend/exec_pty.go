package backend

import (
	"errors"
	"io"
	"os/exec"
	"sync"
)

// ExecPTYAdapter provides a process-backed stream. Native PTY/ConPTY resize
// is intentionally delegated to platform-specific implementations.
type ExecPTYAdapter struct {
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	width, height uint16
	mu            sync.Mutex
}

func NewExecPTYAdapter(name string, args ...string) *ExecPTYAdapter {
	return &ExecPTYAdapter{cmd: exec.Command(name, args...)}
}
func (p *ExecPTYAdapter) Start() error {
	in, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := p.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	p.stdin, p.stdout = in, out
	return p.cmd.Start()
}
func (p *ExecPTYAdapter) Stop() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}
func (p *ExecPTYAdapter) Read(b []byte) (int, error) {
	if p.stdout == nil {
		return 0, errors.New("pty not started")
	}
	return p.stdout.Read(b)
}
func (p *ExecPTYAdapter) Write(b []byte) (int, error) {
	if p.stdin == nil {
		return 0, errors.New("pty not started")
	}
	return p.stdin.Write(b)
}
func (p *ExecPTYAdapter) Size() (uint16, uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.width, p.height, nil
}
func (p *ExecPTYAdapter) Resize(w, h uint16) error {
	p.mu.Lock()
	p.width, p.height = w, h
	p.mu.Unlock()
	return errors.New("pty resize requires platform adapter")
}
