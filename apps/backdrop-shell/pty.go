package main

import "io"

// shellTTY is the pseudo-terminal the shell runs on. The wrapper reads what
// the shell writes into it and writes the user's keys back; the shell believes
// it is on a terminal of its own.
//
// Unix has pseudo-terminals as such. Windows has no equivalent — a console is
// not a device that can be opened — so there the terminal is a ConPTY, whose
// handles and pipes take the same shape (pty_windows.go).
type shellTTY interface {
	io.ReadWriteCloser

	// Name is the terminal's own name, which the shell is told, so a second
	// wrapper started on another terminal can tell it is not inside this one.
	Name() string

	// Resize tells the terminal, and through it the shell, its new size.
	Resize(cols, rows int) error

	// Start runs argv on the terminal with env as its whole environment, and
	// returns once it has started. The terminal owns the child from then on.
	Start(argv, env []string) error

	// ExitCode is the child's exit code, waiting for it if it has not ended.
	// It is asked once the terminal has reached EOF.
	ExitCode() int
}

// sceneTTY is the terminal a scene program draws on, and the program itself:
// it can be stopped while nothing can see it and ended when the scene gives
// way to another.
type sceneTTY interface {
	io.ReadWriteCloser

	// Pause stops the program, and what it started, while its rows cannot be
	// seen, and lets it go on once they can.
	Pause(paused bool)

	// Kill ends the program and everything it started.
	Kill()
}

// openShellTTY makes a terminal of cols×rows for the shell to run on.
func openShellTTY(cols, rows int) (shellTTY, error) {
	return openShellTTYPlatform(cols, rows)
}

// startSceneTTY runs argv on a terminal of w×h, so that a scene program draws
// as it would on a real screen. It returns once the program has started.
func startSceneTTY(argv, env []string, w, h int) (sceneTTY, error) {
	return startSceneTTYPlatform(argv, env, w, h)
}
