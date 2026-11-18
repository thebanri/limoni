//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"golang.org/x/sys/windows"
)

// isTerminal reports whether f is a console.
func isTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

// windowSize is the size of the console on fd, in cells: the visible window of
// the screen buffer, not the buffer, which is taller.
func windowSize(fd int) (w, h int) {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info); err != nil {
		return 80, 24
	}
	w = int(info.Window.Right-info.Window.Left) + 1
	h = int(info.Window.Bottom-info.Window.Top) + 1
	if w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// makeRaw puts the console in raw VT mode and returns the undo. limoni's
// Windows raw mode takes both handles, since input and output are separate
// consoles modes there.
func makeRaw(in, out *os.File) (func(), error) {
	state, err := driver.MakeRaw(in.Fd(), out.Fd())
	if err != nil {
		return nil, err
	}
	return func() { _ = driver.RestoreConsole(state) }, nil
}

// waitReadable reports whether f has input within d: a console handle is
// signalled, rather than polled, when input records are waiting.
func waitReadable(f *os.File, d time.Duration) bool {
	ev, err := windows.WaitForSingleObject(windows.Handle(f.Fd()), uint32(d/time.Millisecond))
	return err == nil && ev == windows.WAIT_OBJECT_0
}

// resizePoll is how often the console's size is read. A Windows console has
// no signal for a resize; a change is cheap to notice this way, and resize()
// reads the size again so a missed or doubled poll only costs a frame.
const resizePoll = 100 * time.Millisecond

// watchResize reports a change of the console's size on changed until done is
// closed. The console of a ConPTY follows the size Windows Terminal gives it.
func watchResize(outFd int, changed chan<- struct{}, done <-chan struct{}) {
	watchResizeWith(outFd, windowSize, resizePoll, changed, done)
}

func watchResizeWith(outFd int, size func(int) (int, int), interval time.Duration, changed chan<- struct{}, done <-chan struct{}) {
	w, h := size(outFd)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			nw, nh := size(outFd)
			if nw != w || nh != h {
				w, h = nw, nh
				select {
				case changed <- struct{}{}:
				default: // one pending check is enough; resize reads the size
				}
			}
		case <-done:
			return
		}
	}
}

// defaultShell is the command a run without one starts: PowerShell where it is
// installed, else the command processor Windows names.
func defaultShell() []string {
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return []string{path}
		}
	}
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		return []string{comspec}
	}
	return []string{"cmd.exe"}
}

// procGetConsoleWindow answers with the HWND of the console this process is
// attached to, which is the identity of the terminal window.
var procGetConsoleWindow = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow")

// ttyName names the terminal behind f. There is no /proc on Windows; the
// console window is the terminal's identity instead, and a window opened from
// inside a wrapper has a console of its own, with a window of its own on
// screen - unlike the hidden one a pseudo console has. Windows Terminal and
// VS Code host every shell in a pseudo console, so all of them get the same
// name here; the environment variable is what the nested check rests on.
func ttyName(f *os.File) string {
	if !isTerminal(f) {
		return ""
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 || !windows.IsWindowVisible(windows.HWND(hwnd)) {
		// No window on screen: the console behind f is a pseudo console, and
		// a wrapper's own is what it calls itself (pty_windows.go).
		return pseudoConsole
	}
	return "console:" + strconv.FormatUint(uint64(hwnd), 16)
}

// execProcess runs the command in this console and waits for it, since Windows
// has no way to replace one process with another. It returns the exit code.
func execProcess(argv []string) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	if err != nil {
		return 0, err
	}
	return 0, nil
}

// platformHooks are the PowerShell profiles. The current-user, all-hosts
// profile is the one used, so Windows Terminal, an editor's terminal and a
// plain console all pick the hook up. PowerShell 7 and Windows PowerShell 5.1
// keep their profiles in different folders.
func platformHooks() []hook {
	home, _ := os.UserHomeDir()
	docs := filepath.Join(home, "Documents")
	return []hook{
		{shell: "pwsh", file: filepath.Join(docs, "PowerShell", "Profile.ps1")},
		{shell: "powershell", file: filepath.Join(docs, "WindowsPowerShell", "Profile.ps1")},
	}
}

// userID is always 0 on Windows: there is no numeric user id, and the
// temporary directory the sockets go in is already the user's own.
func userID() int { return 0 }

// checkPrivateDir has nothing to check on Windows: the temporary directory is
// per user and its access is the NTFS ACL's, which Go does not report in the
// permission bits it prints.
func checkPrivateDir(path string) error { return nil }

// pathext is the list of extensions Windows starts as programs, the user's
// own list or the one Windows ships with.
func pathext() []string {
	list := os.Getenv("PATHEXT")
	if list == "" {
		list = ".COM;.EXE;.BAT;.CMD"
	}
	return strings.Split(list, ";")
}

// notRunnable says why a file cannot be started, in the platform's terms.
func notRunnable(path string) error {
	return fmt.Errorf("%s is not a program Windows can start; give it an extension PATHEXT names (.exe, .bat, .cmd), or start it through its interpreter", path)
}

// isRunnable reports whether a regular file at path can be started as a
// program. Windows has no execute bit; a program is a file whose extension
// PATHEXT names.
func isRunnable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	for _, e := range pathext() {
		if e != "" && strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// sceneProgramIn is the program inside a scene folder: the file of the
// folder's own name, which on Windows carries an extension (cosmos\cosmos.exe).
// The file's own name is returned, in the case it was written in.
func sceneProgramIn(dir, name string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.EqualFold(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), name) {
			continue
		}
		if p := filepath.Join(dir, e.Name()); isRunnable(p) {
			return p
		}
	}
	return ""
}

// update does nothing on Windows: the foreground program is read from
// /proc, which Windows does not have, and the tab-title trick it serves is a
// Konsole one.
func (p *procName) update(t shellTTY) {}

// setLocked does nothing on Windows: a process cannot rename itself, and the
// terminal titles that would show the name are not set from here.
func (p *procName) setLocked(name string) {}

// pauseProcess does nothing on Windows: there is no signal that stops a
// process, and suspending its threads would need ntdll, not a console API. A
// scene program keeps drawing that nobody sees; it is ended as usual.
func pauseProcess(p *os.Process, paused bool) {}

// terminateProcess ends p and the programs it started, which a job object
// would gather; without one, the process itself is killed.
func terminateProcess(p *os.Process) {
	if p == nil {
		return
	}
	_ = p.Kill()
}
