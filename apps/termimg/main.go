//go:build unix

// termimg runs a shell, or a command, so that the sixel pictures programs
// print show in a terminal without sixel: in a window of their own inside
// the terminal's X11 window, every pixel kept (internal/overlay), or in half
// blocks where there is no X11 window. Programs are told the terminal has
// sixel, and its size in pixels.
//
//	termimg              the shell, $SHELL
//	termimg yazi         one command
package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"

	"github.com/thebanri/limoni/core/driver"
)

func main() {
	argv := os.Args[1:]
	if len(argv) == 0 {
		argv = []string{shell()}
	}
	code, err := run(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "termimg:", err)
	}
	os.Exit(code)
}

func run(argv []string) (int, error) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return 1, err
	}
	defer ptmx.Close()
	s := newSession(os.Stdout, ptmx)
	defer s.close()

	cmd, err := startOn(tty, argv)
	if err != nil {
		return 1, err
	}

	in := int(os.Stdin.Fd())
	var typed []byte // keys that came while asking the terminal
	if state, err := driver.MakeRaw(in); err == nil {
		defer driver.Restore(in, state)
		var bg color.RGBA
		var ok bool
		var at image.Point
		bg, ok, at, typed = queryTerminal(os.Stdin, os.Stdout)
		if ok && s.ov != nil {
			s.ov.SetBackground(bg)
		}
		// The shadow screen starts where the real cursor is, not at the
		// top: the program's first pictures go under what it prints there.
		s.emu.Write([]byte("\x1b[" + strconv.Itoa(at.Y+1) + ";" + strconv.Itoa(at.X+1) + "H"))
	}
	if s.ov != nil {
		go s.watchScroll()
	}
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	go func() {
		for range winch {
			s.resize()
		}
	}()

	keys := make(chan struct{}, 1)
	go func() {
		var d da1
		if _, err := ptmx.Write(d.filter(typed)); err != nil {
			return
		}
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				select {
				case keys <- struct{}{}:
				default:
				}
				s.typed()
				if _, werr := ptmx.Write(d.filter(buf[:n])); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 64<<10)
	pump := func() {
		for {
			n, err := ptmx.Read(buf)
			// A program writing fast comes in reads of 4 KB; taken together, a
			// screenful and more of them can skip the shadow screen (skipScrolled).
			for err == nil && n < len(buf) && readable(ptmx) {
				var m int
				m, err = ptmx.Read(buf[n:])
				n += m
			}
			if n > 0 {
				if werr := s.output(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return // EIO once the program has gone
			}
		}
	}
	pump()
	code, err := exitCode(cmd.Wait())

	// The pictures live as long as termimg: a command that printed one and
	// ended (gowall --preview) would take it along at once. They stay, on
	// the screen as the command left it, until a key is pressed; a shell's
	// exit is the user leaving already.
	if isShell(argv[0]) || !s.keeps() {
		return code, err
	}
	for len(keys) > 0 { // keys typed while the command ran were its own
		<-keys
	}
	_ = s.output([]byte("\x1b[2mtermimg: a key closes the pictures\x1b[m"))
	<-keys
	_ = s.output([]byte("\r\x1b[K"))
	return code, err
}

// startOn starts argv on the pty's tty, and closes it: the program keeps it.
func startOn(tty *os.File, argv []string) (*exec.Cmd, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	// backdrop-shell's start-up hook would start it again on this new pty,
	// and its emulated screen would swallow the sixel: tell it this pty is
	// already one of its own. TERMIMG is how a program learns it runs under
	// termimg; backdrop-shell sets LIMONI_BACKDROP_SHELL too, without pictures.
	cmd.Env = append(os.Environ(), "LIMONI_BACKDROP_SHELL="+tty.Name(), "TERMIMG=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	err := cmd.Start()
	tty.Close()
	return cmd, err
}

func exitCode(err error) (int, error) {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return 0, err
}

// shell is the user's shell.
func shell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

// isShell says whether name is a shell: its exit is the user leaving, and
// termimg does not wait for a key after it (the Xwayland relaunch runs one).
func isShell(name string) bool {
	switch base := filepath.Base(name); base {
	case "sh", "bash", "zsh", "fish", "dash", "ksh", "mksh", "tcsh", "csh", "nu", "elvish", "xonsh":
		return true
	default:
		return base == filepath.Base(shell())
	}
}

// readable says whether f has something to read at once.
func readable(f *os.File) bool {
	rc, err := f.SyscallConn()
	if err != nil {
		return false
	}
	ok := false
	_ = rc.Control(func(fd uintptr) {
		p := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, _ := unix.Poll(p, 0)
		ok = n > 0 && p[0].Revents&unix.POLLIN != 0
	})
	return ok
}
