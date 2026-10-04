//go:build linux

package driver

import (
	"os"
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// startTTY sets up a backend on the terminal side of a real pseudo-terminal
// and drains what it writes, so a full output buffer never blocks it.
func startTTY(t *testing.T) (*Backend, *os.File) {
	t.Helper()
	t.Setenv("LIMONI_PROBE", "0")
	master, slave := openPTY(t)
	b := NewBackend(slave, slave)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	b.StartEventLoop()
	t.Cleanup(func() { _ = b.Close() })
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := master.Read(buf); err != nil {
				return
			}
		}
	}()
	return b, master
}

// The TTY path reads a real terminal: the same ESC handling as the portable
// path, through the kernel's line discipline in raw mode.
func TestTTYLoopEscAndSplitSequences(t *testing.T) {
	b, master := startTTY(t)
	checkEscHandling(t, b, func(s string) {
		if _, err := master.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	})
}

// A window resize is the terminal's new size, reported on SIGWINCH.
func TestTTYLoopReportsSIGWINCH(t *testing.T) {
	b, master := startTTY(t)
	if err := unix.IoctlSetWinsize(int(master.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: 101, Row: 33}); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	for {
		ev := nextEvent(t, b)
		if ev.Type == EventResize {
			if ev.Resize.Width != 101 || ev.Resize.Height != 33 {
				t.Fatalf("resize to %dx%d, want 101x33", ev.Resize.Width, ev.Resize.Height)
			}
			return
		}
	}
}

// SIGTERM restores the terminal before the process exits with 130: the
// shell gets back a terminal with echo on, not one left in raw mode. Run in
// a child process, because the handler calls os.Exit.
func TestTTYLoopRestoresTheTerminalOnSIGTERM(t *testing.T) {
	if path := os.Getenv("LIMONI_SIGTERM_TTY"); path != "" {
		slave, err := os.OpenFile(path, os.O_RDWR|unix.O_NOCTTY, 0)
		if err != nil {
			os.Exit(2)
		}
		b := NewBackend(slave, slave)
		if err := b.Setup(); err != nil {
			os.Exit(3)
		}
		b.StartEventLoop()
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		select {} // the handler exits
	}

	master, slave := openPTY(t)
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := master.Read(buf); err != nil {
				return
			}
		}
	}()
	cmd := exec.Command(os.Args[0], "-test.run=^TestTTYLoopRestoresTheTerminalOnSIGTERM$")
	cmd.Env = append(os.Environ(), "LIMONI_SIGTERM_TTY="+slave.Name(), "LIMONI_PROBE=0")
	err := cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 130 {
		t.Fatalf("the child ended with %v, want exit status 130", err)
	}
	tio, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if tio.Lflag&unix.ECHO == 0 || tio.Lflag&unix.ICANON == 0 {
		t.Error("after SIGTERM the terminal was left in raw mode")
	}
}
