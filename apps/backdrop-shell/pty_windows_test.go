//go:build windows

package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// readAll reads until the console reaches EOF, failing rather than hanging
// if a console that should have ended never does.
func readAll(t *testing.T, r io.Reader) []byte {
	t.Helper()
	type read struct {
		b   []byte
		err error
	}
	done := make(chan read, 1)
	go func() {
		var all []byte
		buf := make([]byte, 64<<10)
		for {
			n, err := r.Read(buf)
			all = append(all, buf[:n]...)
			if err != nil {
				done <- read{all, err}
				return
			}
		}
	}()
	select {
	case got := <-done:
		if got.err != io.EOF && !errors.Is(got.err, os.ErrClosed) {
			t.Fatalf("read: %v", got.err)
		}
		return got.b
	case <-time.After(20 * time.Second):
		t.Fatal("no EOF from the pseudo console within 20s")
		return nil
	}
}

// TestConPTYRunsAProgram checks the promise the wrapper's terminal makes: a
// program runs on a console of its own size, what it writes comes back, and
// the exit code it ended with is the one the session returns.
func TestConPTYRunsAProgram(t *testing.T) {
	cmdExe, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Skip("no cmd.exe")
	}
	tty, err := openShellTTY(100, 30)
	if err != nil {
		t.Skipf("no pseudo console on this Windows: %v", err)
	}
	defer tty.Close()

	if err := tty.Resize(120, 40); err != nil {
		t.Errorf("Resize: %v", err)
	}
	env := append(os.Environ(), "PROMPT=")
	if err := tty.Start([]string{cmdExe, "/c", "echo conpty-ok"}, env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	out := readAll(t, tty)
	if !strings.Contains(string(out), "conpty-ok") {
		t.Fatalf("output does not hold what the program wrote: %q", out)
	}
	if code := tty.ExitCode(); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
}

// TestConPTYReportsTheExitCode checks that a program that fails is reported
// as failing, rather than the wrapper's own code standing in for it.
func TestConPTYReportsTheExitCode(t *testing.T) {
	cmdExe, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Skip("no cmd.exe")
	}
	tty, err := openShellTTY(80, 24)
	if err != nil {
		t.Skipf("no pseudo console on this Windows: %v", err)
	}
	defer tty.Close()

	env := append(os.Environ(), "PROMPT=")
	if err := tty.Start([]string{cmdExe, "/c", "exit 3"}, env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_ = readAll(t, tty)
	if code := tty.ExitCode(); code != 3 {
		t.Fatalf("exit code %d, want 3", code)
	}
}

// TestConPTYSceneProgram checks the other side of the seam: a scene program
// draws on a console of the scene's size, and the scene can end it.
func TestConPTYSceneProgram(t *testing.T) {
	cmdExe, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Skip("no cmd.exe")
	}
	tty, err := startSceneTTY([]string{cmdExe, "/c", "echo scene-ok"}, os.Environ(), 40, 12)
	if err != nil {
		t.Skipf("no pseudo console on this Windows: %v", err)
	}
	defer tty.Close()

	out := readAll(t, tty)
	if !strings.Contains(string(out), "scene-ok") {
		t.Fatalf("output does not hold what the program wrote: %q", out)
	}
	tty.Kill() // nothing running by now; it must not hang or panic
}
