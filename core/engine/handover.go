package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

// ErrHandoverBusy is passed to an ExecCmd callback when another program
// already has the terminal, or a suspend is already waiting.
var ErrHandoverBusy = errors.New("limoni: the terminal is already handed over")

// handoverMsg asks the terminal side to give the terminal away: to a program
// (run) or to the shell (suspend). The runtime takes it before Update.
type handoverMsg struct {
	run     func() error
	suspend bool
	done    func(error) Msg
}

// Handover is one request from ExecCmd or SuspendCmd, for whoever owns the
// terminal. RunTerminal reads them; a caller driving Draw itself should too:
// call Run (or terminal.Terminal.Suspend when Suspend is set), then Done.
type Handover struct {
	// Run gives the terminal to the program; it is nil for a suspend.
	Run func() error
	// Suspend asks for the process to be stopped, as Ctrl+Z does.
	Suspend bool
	// Done reports the outcome to the model. Call it once, with the error
	// from Run or from the suspend.
	Done func(error)
}

// ExecCmd hands the terminal to cmd — an editor, a pager, a shell — and
// takes it back when cmd exits. The screen is the shell's while it runs, and
// the application repaints afterwards. done turns cmd's error into a message
// for Update; it may be nil, or return nil, for no message.
//
// A nil Stdin, Stdout or Stderr is the process's own, which is the terminal a
// Program started by RunProgram draws on.
//
// The terminal can only be handed over where the application owns one: a
// Unix terminal or a Windows console. Elsewhere — a remote session, the
// browser, input piped in — done receives driver.ErrReleaseUnsupported and
// cmd is not started.
func ExecCmd(cmd *exec.Cmd, done func(error) Msg) Cmd {
	return func(context.Context) Msg {
		return handoverMsg{
			run: func() error {
				if cmd.Stdin == nil {
					cmd.Stdin = os.Stdin
				}
				if cmd.Stdout == nil {
					cmd.Stdout = os.Stdout
				}
				if cmd.Stderr == nil {
					cmd.Stderr = os.Stderr
				}
				return cmd.Run()
			},
			done: done,
		}
	}
}

// SuspendCmd hands the terminal back to the shell and stops the application,
// as Ctrl+Z does in any other program; `fg` resumes it and the screen is
// repainted. Where there is no shell to return to it does nothing.
func SuspendCmd() Cmd {
	return func(context.Context) Msg { return handoverMsg{suspend: true} }
}

// Handovers are the requests ExecCmd and SuspendCmd made, for whoever owns
// the terminal. See Handover.
func (p *Program) Handovers() <-chan Handover { return p.handovers }

// takeHandover passes a handoverMsg to the terminal side. One waits at a
// time: a second request while one is pending fails at once rather than
// blocking Update.
func (p *Program) takeHandover(ctx context.Context, message Msg) bool {
	h, ok := message.(handoverMsg)
	if !ok {
		return false
	}
	done := func(err error) {
		if h.done == nil {
			return
		}
		if msg := h.done(err); msg != nil {
			// Not from the goroutine that took the request: the queue may be
			// full, and Run drains it.
			go func() { _ = p.Send(ctx, msg) }()
		}
	}
	select {
	case p.handovers <- Handover{Run: h.run, Suspend: h.suspend, Done: done}:
	default:
		done(ErrHandoverBusy)
	}
	return true
}
