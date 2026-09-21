package driver

import (
	"errors"
	"io"
	"os"
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
	stopped       bool
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

// Stop kills the process, closes its stdin and reaps it. Writes after Stop
// fail instead of landing in a pipe nobody reads, and a second Stop is a no-op.
func (p *ExecPTYAdapter) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped || p.cmd.Process == nil {
		return nil
	}
	p.stopped = true
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	// The exit status of a killed process is the kill; it is not an error here.
	_ = p.cmd.Wait()
	return nil
}
func (p *ExecPTYAdapter) Read(b []byte) (int, error) {
	if p.stdout == nil {
		return 0, errors.New("pty not started")
	}
	return p.stdout.Read(b)
}
func (p *ExecPTYAdapter) Write(b []byte) (int, error) {
	p.mu.Lock()
	in, stopped := p.stdin, p.stopped
	p.mu.Unlock()
	if in == nil {
		return 0, errors.New("pty not started")
	}
	if stopped {
		return 0, errors.New("pty stopped")
	}
	return in.Write(b)
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
