//go:build unix

package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// unixTTY is a pseudo-terminal from creack/pty: the master the wrapper reads
// and writes, and the slave the shell runs on until it is closed.
type unixTTY struct {
	ptmx *os.File
	tty  *os.File
	cmd  *exec.Cmd
}

func openShellTTYPlatform(cols, rows int) (shellTTY, error) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}
	if err := pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
		ptmx.Close()
		tty.Close()
		return nil, err
	}
	return &unixTTY{ptmx: ptmx, tty: tty}, nil
}

// Start gives the child the terminal, in a session of its own so that it is
// the terminal's foreground group: that is what makes the shell the program
// whose name a terminal's tab shows, and what makes the ioctls in procname.go
// answer about the shell rather than the wrapper.
func (t *unixTTY) Start(argv, env []string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = t.tty, t.tty, t.tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	t.cmd = cmd
	// The child holds the slave now; a terminal whose last slave is closed
	// reads EOF, and that is what ends the session.
	t.tty.Close()
	return nil
}

// ExitCode waits for the shell and returns what it exited with. The write end
// of the terminal is the child's, so the wrapper has already read EOF when it
// is asked.
func (t *unixTTY) ExitCode() int {
	if t.cmd == nil {
		return 1
	}
	if err := t.cmd.Wait(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		return 1
	}
	return 0
}

// Name is the slave's name (the same one, after Start closed it).
func (t *unixTTY) Name() string { return t.tty.Name() }

func (t *unixTTY) Resize(cols, rows int) error {
	return pty.Setsize(t.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (t *unixTTY) Read(b []byte) (int, error)  { return t.ptmx.Read(b) }
func (t *unixTTY) Write(b []byte) (int, error) { return t.ptmx.Write(b) }
func (t *unixTTY) Close() error                { return t.ptmx.Close() }

// unixSceneTTY is a scene program on a pseudo-terminal of its own, in a
// session of its own so that stopping and killing the group reaches whatever
// it started too — the sleep in a shell loop, say.
type unixSceneTTY struct {
	f   *os.File
	cmd *exec.Cmd
}

func startSceneTTYPlatform(argv, env []string, w, h int) (sceneTTY, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	// Its standard error goes nowhere: the screen belongs to the shell.
	cmd.Stderr = io.Discard
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	if err != nil {
		return nil, err
	}
	return &unixSceneTTY{f: f, cmd: cmd}, nil
}

func (t *unixSceneTTY) Read(b []byte) (int, error)  { return t.f.Read(b) }
func (t *unixSceneTTY) Write(b []byte) (int, error) { return t.f.Write(b) }

// Close ends the terminal and reaps the program, so that nothing of it is
// left behind when the scene gives way to another.
func (t *unixSceneTTY) Close() error {
	err := t.f.Close()
	_ = t.cmd.Wait()
	return err
}

func (t *unixSceneTTY) Pause(paused bool) {
	if t.cmd.Process == nil {
		return
	}
	sig := syscall.SIGSTOP
	if !paused {
		sig = syscall.SIGCONT
	}
	_ = syscall.Kill(-t.cmd.Process.Pid, sig)
}

// Kill asks the program and its group to end, and kills them a second later if
// they are still there. A stopped process acts on SIGTERM only once it runs,
// hence the SIGCONT.
func (t *unixSceneTTY) Kill() {
	if t.cmd.Process == nil {
		return
	}
	pid := t.cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = syscall.Kill(-pid, syscall.SIGCONT)
	time.AfterFunc(time.Second, func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
}
