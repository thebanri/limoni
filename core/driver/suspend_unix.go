//go:build unix

package driver

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"
)

// ErrSuspendUnsupported is returned by Suspend where stopping the process and
// restoring a terminal makes no sense: a portable or remote backend (an SSH
// session has no controlling terminal of its own), the browser, and Windows.
var ErrSuspendUnsupported = errors.New("limoni: suspend is not supported on this backend")

// stopSelf stops this process the way Ctrl+Z does, and returns once it is
// continued. It is a variable so a test can stand in for it; stopping the
// test binary would stop the test run.
var stopSelf = stopProcessGroup

// notStopped is how long stopProcessGroup waits for a stop that never comes:
// the kernel discards SIGTSTP sent to an orphaned process group, one whose
// shell has gone, and then there is nothing to wait for.
const notStopped = 250 * time.Millisecond

// stopProcessGroup sends SIGTSTP to the process group, as the terminal does
// for Ctrl+Z, so that a wrapper the application runs under (script, go run)
// stops with it and the shell gets its prompt back.
//
// The stop is not synchronous with kill: the signal may be taken by another
// thread — one created by C, with its own signal mask — while this one goes
// on. It used to go straight on to raw mode and the setup sequence with its
// capability probe, and be stopped only after: the shell then read the
// terminal's answers as a command line, and the fg typed next with them.
// So this waits for SIGCONT, which comes only after the stop, before
// returning.
func stopProcessGroup() error {
	cont := make(chan os.Signal, 1)
	signal.Notify(cont, unix.SIGCONT)
	defer signal.Stop(cont)
	if err := unix.Kill(0, unix.SIGTSTP); err != nil {
		return err
	}
	timer := time.NewTimer(notStopped)
	defer timer.Stop()
	select {
	case <-cont:
	case <-timer.C:
		// Resumed already, or never stopped: either way, go on.
	}
	return nil
}

// Suspend hands the terminal back to the shell, stops this process, and
// returns when it is resumed with the terminal set up again.
//
// The sequence matters: leave the alternate screen and raw mode *before*
// stopping, or the shell inherits a terminal with no echo, no line editing and
// the application's screen still on it. On resume, raw mode and the setup
// sequence are applied again; the caller should repaint, because the shell
// wrote over the screen in the meantime (Terminal.Suspend does).
func (b *Backend) Suspend() error {
	if b.portableIO != nil || b.in == nil || b.out == nil {
		return ErrSuspendUnsupported
	}
	return b.handOver(func() error {
		// Stops here until the shell continues us.
		if err := stopSelf(); err != nil {
			return fmt.Errorf("limoni: suspend: %w", err)
		}
		return nil
	})
}
