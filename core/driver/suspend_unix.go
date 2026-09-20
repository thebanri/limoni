//go:build unix

package driver

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// ErrSuspendUnsupported is returned by Suspend where stopping the process and
// restoring a terminal makes no sense: a portable or remote backend (an SSH
// session has no controlling terminal of its own), the browser, and Windows.
var ErrSuspendUnsupported = errors.New("limoni: suspend is not supported on this backend")

// stopSelf stops this process the way Ctrl+Z does. It is a variable so a test
// can stand in for it; stopping the test binary would stop the test run.
var stopSelf = func() error { return unix.Kill(unix.Getpid(), unix.SIGTSTP) }

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

	restore := fullScreenRestoreCmds()
	if height := b.Inline(); height > 0 {
		restore = inlineRestoreCmds(height)
	}
	if _, err := b.out.WriteString(restore); err != nil {
		return err
	}
	if b.state != nil {
		if err := Restore(int(b.in.Fd()), b.state); err != nil {
			return err
		}
		b.state = nil
	}

	// Stops here until the shell continues us.
	if err := stopSelf(); err != nil {
		return fmt.Errorf("limoni: suspend: %w", err)
	}

	state, err := MakeRaw(int(b.in.Fd()))
	if err != nil {
		return fmt.Errorf("limoni: resume: %w", err)
	}
	b.state = state
	setup := fullScreenSetupCmds()
	if height := b.Inline(); height > 0 {
		setup = inlineSetupCmds(height)
	}
	// The terminal may be a different one, or the same one reconfigured, so
	// ask it again what it supports.
	setup = b.replies.withProbe(setup)
	_, err = b.out.WriteString(setup)
	return err
}
