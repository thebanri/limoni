//go:build linux

package driver

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// openPTY returns the two ends of a new pseudo-terminal.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no /dev/ptmx: %v", err)
	}
	if err := unix.IoctlSetPointerInt(int(m.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatalf("unlockpt: %v", err)
	}
	n, err := unix.IoctlGetInt(int(m.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatalf("ptsname: %v", err)
	}
	s, err := os.OpenFile("/dev/pts/"+itoa(n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("open slave: %v", err)
	}
	t.Cleanup(func() { s.Close(); m.Close() })
	return m, s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Suspending leaves the alternate screen and raw mode before stopping, and
// resuming puts both back. Getting the order wrong hands the shell a terminal
// with no echo and the application's screen still on it.
func TestSuspendRestoresTheTerminalThenSetsItUpAgain(t *testing.T) {
	master, slave := openPTY(t)
	b := NewBackend(slave, slave)
	if err := b.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	raw, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Lflag&unix.ECHO != 0 {
		t.Fatal("Setup did not turn echo off")
	}

	// Stand in for the SIGTSTP: stopping the test binary would stop the run.
	// While "stopped", the terminal must be the shell's again.
	stopped := false
	restore := stopSelf
	stopSelf = func() error {
		stopped = true
		tio, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
		if err != nil {
			t.Error(err)
		} else if tio.Lflag&unix.ECHO == 0 {
			t.Error("the shell got the terminal back still in raw mode")
		}
		return nil
	}
	t.Cleanup(func() { stopSelf = restore })

	go func() { // the master end must be read, or writes block
		buf := make([]byte, 4096)
		for {
			if _, err := master.Read(buf); err != nil {
				return
			}
		}
	}()

	if err := b.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if !stopped {
		t.Fatal("suspend did not stop the process")
	}
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if after.Lflag&unix.ECHO != 0 {
		t.Fatal("resume did not put the terminal back into raw mode")
	}
}

// A backend with no controlling terminal — a remote session, an in-memory one
// — says so instead of stopping the process it is embedded in.
func TestSuspendRefusedWithoutATerminal(t *testing.T) {
	b := NewPortableBackend(NewMemoryTerminalIO(nil, 40, 10))
	if err := b.Suspend(); err != ErrSuspendUnsupported {
		t.Fatalf("Suspend = %v, want ErrSuspendUnsupported", err)
	}
}
