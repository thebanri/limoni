//go:build unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"golang.org/x/sys/unix"
)

// isTerminal reports whether f is a terminal.
func isTerminal(f *os.File) bool {
	_, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	return err == nil
}

// windowSize is the size of the terminal on fd, in cells.
func windowSize(fd int) (w, h int) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}

// makeRaw puts the terminal in raw mode and returns the undo.
func makeRaw(in, out *os.File) (func(), error) {
	state, err := driver.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	return func() { driver.Restore(int(in.Fd()), state) }, nil
}

// waitReadable reports whether f has input within d. It is what keeps the
// colour query at start-up from blocking on a read for a terminal that answers
// nothing: a poll says whether a read would wait, where a read would wait.
func waitReadable(f *os.File, d time.Duration) bool {
	fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
	for {
		n, err := unix.Poll(fds, int(d/time.Millisecond)+1)
		if err == unix.EINTR {
			continue // a signal arrived before anything did
		}
		return err == nil && n > 0
	}
}

// watchResize reports a change of the terminal's size on changed until done
// is closed. A terminal sends SIGWINCH when its window changes size.
func watchResize(_ int, changed chan<- struct{}, done <-chan struct{}) {
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	for {
		select {
		case <-winch:
			select {
			case changed <- struct{}{}:
			default: // one pending check is enough; resize reads the size
			}
		case <-done:
			return
		}
	}
}

// defaultShell is the command a run without one starts: the user's shell.
func defaultShell() []string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return []string{shell}
	}
	return []string{"/bin/sh"}
}

// ttyName names the terminal behind f, which is how a wrapper knows whether it
// is already inside one: the shell inside a wrapper inherits the variable
// naming it, and its terminal has the same name; a window opened from inside
// inherits the variable too, but has a terminal of its own.
func ttyName(f *os.File) string {
	if name, err := os.Readlink("/proc/self/fd/0"); err == nil {
		return name
	}
	cmd := exec.Command("tty")
	cmd.Stdin = f
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// platformHooks are shells with a start-up file only on this platform.
// Unix has none beyond the fish, bash and zsh ones in install.go.
func platformHooks() []hook { return nil }

// execProcess replaces this process with the command, as if the wrapper had
// never been there. It returns only on failure.
func execProcess(argv []string) (int, error) {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return 0, err
	}
	return 0, syscall.Exec(path, argv, os.Environ())
}

// userID keeps the sockets of different users apart.
func userID() int { return os.Getuid() }

// checkPrivateDir reports an error unless path is closed to other users.
func checkPrivateDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s is open to other users; not listening there", path)
	}
	return nil
}

// notRunnable says why a file cannot be started, in the platform's terms.
func notRunnable(path string) error {
	return fmt.Errorf("%s is not an executable file; make it one with: chmod +x %s", path, path)
}

// isRunnable reports whether a regular file at path can be started as a
// program: the execute bit on Unix.
func isRunnable(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0
}

// sceneProgramIn is the program inside a scene folder: the file of the
// folder's own name (cosmos/cosmos).
func sceneProgramIn(dir, name string) string {
	p := filepath.Join(dir, name)
	if isRunnable(p) {
		return p
	}
	return ""
}

// behind t, if that program changed since the last call: one ioctl when it did
// not.
func (p *procName) update(t shellTTY) {
	u, ok := t.(*unixTTY)
	if !ok {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	pgrp, err := unix.IoctlGetInt(int(u.ptmx.Fd()), unix.TIOCGPGRP)
	if err != nil || pgrp <= 0 || pgrp == p.pgrp {
		return
	}
	p.pgrp = pgrp
	comm, err := os.ReadFile("/proc/" + strconv.Itoa(pgrp) + "/comm")
	if err != nil {
		return
	}
	p.setLocked(strings.TrimSpace(string(comm)))
}

// setLocked renames the wrapper, if the name is new. Writing /proc/self/comm
// renames the main thread, which is the name the process has in stat and ps;
// prctl(PR_SET_NAME) would rename only whichever thread the goroutine happens
// to be on.
func (p *procName) setLocked(name string) {
	if name == "" || name == p.name {
		return
	}
	if os.WriteFile("/proc/self/comm", []byte(name), 0) == nil {
		p.name = name
	}
}

// pauseProcess stops or resumes p and everything it started, so that a scene
// nobody can see costs nothing.
func pauseProcess(p *os.Process, paused bool) {
	if p == nil {
		return
	}
	sig := syscall.SIGSTOP
	if !paused {
		sig = syscall.SIGCONT
	}
	_ = syscall.Kill(-p.Pid, sig)
}

// terminateProcess asks p and its group to end, and kills them a second later
// if they are still there. A stopped process acts on SIGTERM only once it runs,
// hence the SIGCONT.
func terminateProcess(p *os.Process) {
	if p == nil {
		return
	}
	_ = syscall.Kill(-p.Pid, syscall.SIGTERM)
	_ = syscall.Kill(-p.Pid, syscall.SIGCONT)
	pgid := p.Pid
	time.AfterFunc(time.Second, func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })
}
