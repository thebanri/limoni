//go:build linux

package driver

import (
	"errors"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// readLine reads one line from the terminal the way a program run in it
// would, through its own descriptor, giving up after d.
func readLine(t *testing.T, path string, d time.Duration) string {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, int(d/time.Millisecond))
	if err != nil || n == 0 {
		return ""
	}
	buf := make([]byte, 256)
	m, _ := unix.Read(int(f.Fd()), buf)
	return string(buf[:max(m, 0)])
}

// While the terminal belongs to another program, what is typed is that
// program's: the event loop must not read it. A reader blocked in read(2)
// takes the first keys typed into the editor, which is what happened before
// the reader could be paused. Afterwards keys reach the application again.
func TestReleaseLeavesTheInputToTheProgram(t *testing.T) {
	b, master := startTTY(t)
	path, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(int(b.in.Fd())))
	if err != nil {
		t.Skip(err)
	}

	if _, err := master.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	if ev := nextEvent(t, b); ev.Key.Ch != 'a' {
		t.Fatalf("before Release got %+v, want 'a'", ev)
	}

	var got string
	err = b.Release(func() error {
		tio, err := unix.IoctlGetTermios(int(b.in.Fd()), unix.TCGETS)
		if err != nil {
			return err
		}
		if tio.Lflag&unix.ECHO == 0 || tio.Lflag&unix.ICANON == 0 {
			t.Error("the program was handed a terminal still in raw mode")
		}
		if _, err := master.Write([]byte("typed into the editor\n")); err != nil {
			return err
		}
		got = readLine(t, path, 2*time.Second)
		return errors.New("the editor's own error")
	})
	if err == nil || err.Error() != "the editor's own error" {
		t.Fatalf("Release = %v, want the program's error", err)
	}
	if got != "typed into the editor\n" {
		t.Fatalf("the program read %q: the event loop took its input", got)
	}
	noEvent(t, b, 100*time.Millisecond, "the event loop read input meant for the program")

	tio, err := unix.IoctlGetTermios(int(b.in.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if tio.Lflag&unix.ECHO != 0 {
		t.Fatal("Release did not put the terminal back into raw mode")
	}
	if _, err := master.Write([]byte("b")); err != nil {
		t.Fatal(err)
	}
	if ev := nextEvent(t, b); ev.Key.Ch != 'b' {
		t.Fatalf("after Release got %+v, want 'b'", ev)
	}
}

// Ctrl+C in the program the terminal was handed to interrupts the whole
// foreground process group. The application must outlive it; before, the
// signal handler closed the terminal and exited with 130.
func TestReleaseIgnoresTheProgramsInterrupt(t *testing.T) {
	b, _ := startTTY(t)
	err := b.Release(func() error {
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond) // the handler runs on its own goroutine
		return nil
	})
	if err != nil {
		t.Fatalf("Release = %v", err)
	}
	select {
	case <-b.done:
		t.Fatal("an interrupt during Release closed the backend")
	default:
	}
}

// A backend with no terminal of its own refuses, and does not run fn.
func TestReleaseRefusedWithoutATerminal(t *testing.T) {
	b := NewPortableBackend(NewMemoryTerminalIO(nil, 40, 10))
	ran := false
	if err := b.Release(func() error { ran = true; return nil }); err != ErrReleaseUnsupported {
		t.Fatalf("Release = %v, want ErrReleaseUnsupported", err)
	}
	if ran {
		t.Fatal("Release ran the program without a terminal to give it")
	}
}
