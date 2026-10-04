//go:build linux || darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

const ptySupported = true

// controlFD runs f on file's descriptor without taking it out of the
// runtime's poller, so a Close still wakes a blocked Read.
func controlFD(file *os.File, f func(fd int) error) error {
	raw, err := file.SyscallConn()
	if err != nil {
		return err
	}
	var ferr error
	if err := raw.Control(func(fd uintptr) { ferr = f(int(fd)) }); err != nil {
		return err
	}
	return ferr
}

// setPTYSize tells the terminal side its size; the kernel sends the program
// SIGWINCH.
func setPTYSize(master *os.File, cols, rows uint16) error {
	return controlFD(master, func(fd int) error {
		return unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{Col: cols, Row: rows})
	})
}

// startInPTY starts cmd as the session leader of a new session whose
// controlling terminal is a fresh pseudo-terminal of size cols×rows.
func startInPTY(cmd *exec.Cmd, cols, rows uint16) (*os.File, error) {
	master, path, err := openPTY()
	if err != nil {
		return nil, fmt.Errorf("open a pseudo-terminal: %w", err)
	}
	tty, err := os.OpenFile(path, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer tty.Close() // the child has its own copies
	// The size is set once the terminal side is open: macOS refuses
	// TIOCSWINSZ on a master whose terminal nobody has opened yet.
	if err := setPTYSize(tty, cols, rows); err != nil {
		master.Close()
		return nil, fmt.Errorf("set the terminal size: %w", err)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		master.Close()
		return nil, fmt.Errorf("start %s: %w", cmd.Path, err)
	}
	return master, nil
}

// hangUp ends the program's session the way closing a terminal window does.
func hangUp(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGHUP)
}

// killSession kills every process in the program's session group.
func killSession(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
