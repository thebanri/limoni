//go:build linux

package driver

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// suspendChild is the child side of TestSuspendStopsBeforeSettingUpAgain:
// a backend on the terminal it was given, suspended for real.
func suspendChild() {
	b := NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		os.Exit(2)
	}
	// SIGTSTP blocked on this thread, as a thread created by C may have it:
	// the kernel hands the signal to another thread, and the stop lands
	// asynchronously while this one runs on.
	runtime.LockOSThread()
	var set unix.Sigset_t
	set.Val[0] = 1 << (uint(unix.SIGTSTP) - 1)
	_ = unix.PthreadSigmask(unix.SIG_BLOCK, &set, nil)
	if err := b.Suspend(); err != nil {
		os.Exit(3)
	}
	_, _ = os.Stdout.WriteString("RESUMED")
	_ = b.Close()
	os.Exit(0)
}

func TestMain(m *testing.M) {
	if os.Getenv("LIMONI_SUSPEND_CHILD") == "1" {
		suspendChild()
	}
	os.Exit(m.Run())
}

// Suspending stops the process before it sets the terminal up again. The stop
// lands asynchronously when another thread takes the signal; the backend used
// to run on into raw mode and the setup sequence, capability probe and all,
// and only then stop — the shell read the terminal's answers as a command
// line, and the fg typed next with them.
func TestSuspendStopsBeforeSettingUpAgain(t *testing.T) {
	master, slave := openPTY(t)
	var mu sync.Mutex
	var out bytes.Buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := master.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	output := func() string { mu.Lock(); defer mu.Unlock(); return out.String() }

	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "LIMONI_SUSPEND_CHILD=1", "LIMONI_PROBE=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	// A process group of its own, so the SIGTSTP sent to the group stops
	// the child and not the test.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = unix.Kill(pid, unix.SIGKILL); _, _ = cmd.Process.Wait() })

	var ws unix.WaitStatus
	if _, err := unix.Wait4(pid, &ws, unix.WUNTRACED, nil); err != nil {
		t.Fatal(err)
	}
	if !ws.Stopped() {
		t.Fatalf("the child did not stop: %v; output %q", ws, output())
	}
	time.Sleep(200 * time.Millisecond) // whatever it wrote, read it
	stopped := output()
	if n := strings.Count(stopped, "\x1b[?1049h"); n != 1 {
		t.Fatalf("the screen was set up %d times before the stop: it ran on past SIGTSTP\n%q", n, stopped)
	}
	if !strings.Contains(stopped, "\x1b[?1049l") {
		t.Fatalf("the screen was not handed back before the stop: %q", stopped)
	}

	if err := unix.Kill(pid, unix.SIGCONT); err != nil {
		t.Fatal(err)
	}
	if _, err := unix.Wait4(pid, &ws, 0, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(output(), "RESUMED") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := output(); strings.Count(after, "\x1b[?1049h") != 2 || !strings.Contains(after, "RESUMED") {
		t.Fatalf("after SIGCONT the screen was not set up again: %q", after)
	}
}
