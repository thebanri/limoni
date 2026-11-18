//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// pseudoConsole names a pseudo console. It has no window, and that is how a
// wrapper started inside another one is recognised (platform_windows.go).
const pseudoConsole = "pseudoconsole"

// conPTY is a Windows pseudo console: a console of a given size whose screen
// is a pair of pipes the wrapper holds. The shell runs on that console as it
// would on a real one — it has a console handle, a screen with a cursor, and
// an input buffer — while the wrapper reads its output from a pipe and writes
// the user's keys into another.
//
// CreatePseudoConsole, ResizePseudoConsole and ClosePseudoConsole need Windows
// 10 1809 or newer; a program starts on the console through the
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE process attribute, which is why the
// child is created with CreateProcess here rather than with os/exec: a
// *exec.Cmd has no way to carry that attribute.
type conPTY struct {
	hpc  windows.Handle
	in   *os.File // the wrapper writes the user's keys here
	out  *os.File // the console writes the shell's output here
	proc windows.Handle

	// The console's ends of the pipes are ours until a child has been born
	// attached to it: closing them sooner makes the console hand the child
	// invalid standard handles.
	inR, outW *os.File

	closeOnce sync.Once
	codeOnce  sync.Once
	code      int
}

func newConPTY(cols, rows int) (*conPTY, error) {
	// The console's own ends: it reads keys from inR and writes output to
	// outW. The wrapper keeps the other ends.
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		inR.Close()
		inW.Close()
		return nil, err
	}

	var hpc windows.Handle
	err = windows.CreatePseudoConsole(
		windows.Coord{X: int16(cols), Y: int16(rows)},
		windows.Handle(inR.Fd()), windows.Handle(outW.Fd()), 0, &hpc,
	)
	if err != nil {
		inR.Close()
		inW.Close()
		outR.Close()
		outW.Close()
		return nil, fmt.Errorf("create pseudo console: %w", err)
	}
	return &conPTY{hpc: hpc, in: inW, out: outR, inR: inR, outW: outW}, nil
}

// Start runs argv on the console, with env as its whole environment: the
// console is given to the child as a process attribute, so that it is born
// attached to it.
func (t *conPTY) Start(argv, env []string) error {
	line, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(argv))
	if err != nil {
		return err
	}
	block, err := envBlock(env)
	if err != nil {
		return err
	}

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return err
	}
	defer attrs.Delete()
	hpc := t.hpc
	// Windows reads the pointer-sized value itself from lpValue here, not a
	// pointer to it. Passing the address instead makes the child attach to a
	// console of its own: it runs, and its writes land on a screen nobody
	// reads, while the wrapper waits on pipes that stay silent.
	pc := *(*unsafe.Pointer)(unsafe.Pointer(&hpc))
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, pc, unsafe.Sizeof(hpc)); err != nil {
		return err
	}

	si := &windows.StartupInfoEx{
		// Cb is the size of the structure that is actually passed, the
		// extended one, since the attribute list follows it.
		StartupInfo: windows.StartupInfo{
			Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})),
			// No standard handles of our own: the child's are the console's,
			// which it is attached to as it is born.
			Flags: windows.STARTF_USESTDHANDLES,
		},
	}
	si.ProcThreadAttributeList = attrs.List()

	var pi windows.ProcessInformation
	err = windows.CreateProcess(nil, line, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		&block[0], nil, &si.StartupInfo, &pi)
	// The child is attached now; our copies of the console's pipe ends are
	// no longer needed. Keep them until here: the console uses them while
	// the child is being born.
	t.releaseConsoleHandles()
	if err != nil {
		return fmt.Errorf("start %s: %w", argv[0], err)
	}
	windows.CloseHandle(pi.Thread)
	t.proc = pi.Process

	// A pseudo console does not go away by itself: when the program ends,
	// close it, or the wrapper waits on a pipe nobody will ever write to
	// again. The short pause lets the console hand over what the program
	// wrote last before it does.
	go func() {
		_, _ = windows.WaitForSingleObject(t.proc, windows.INFINITE)
		time.Sleep(150 * time.Millisecond)
		t.close()
	}()
	return nil
}

// ExitCode waits for the shell and returns what it exited with.
func (t *conPTY) ExitCode() int {
	t.codeOnce.Do(func() {
		if t.proc == 0 {
			t.code = 1
			return
		}
		_, _ = windows.WaitForSingleObject(t.proc, windows.INFINITE)
		var code uint32
		if err := windows.GetExitCodeProcess(t.proc, &code); err != nil {
			t.code = 1
		} else {
			t.code = int(code)
		}
		_ = windows.CloseHandle(t.proc)
	})
	return t.code
}

// Name is the console's name: it has no window, and a wrapper inside another
// one finds that out the same way (platform_windows.go).
func (t *conPTY) Name() string { return pseudoConsole }

func (t *conPTY) Resize(cols, rows int) error {
	return windows.ResizePseudoConsole(t.hpc, windows.Coord{X: int16(cols), Y: int16(rows)})
}

func (t *conPTY) Read(b []byte) (int, error)  { return t.out.Read(b) }
func (t *conPTY) Write(b []byte) (int, error) { return t.in.Write(b) }

func (t *conPTY) Close() error {
	t.close()
	return t.out.Close()
}

// close ends the console, which also ends the programs attached to it, and
// closes the wrapper's end of its input pipe. More than once is a no-op. The
// output pipe is left open: the console's own end of it goes with the console,
// and that is how the reader learns the console is done — by reading EOF.
func (t *conPTY) close() {
	t.closeOnce.Do(func() {
		t.releaseConsoleHandles()
		windows.ClosePseudoConsole(t.hpc)
		t.in.Close()
	})
}

func (t *conPTY) releaseConsoleHandles() {
	if t.inR != nil {
		t.inR.Close()
		t.inR = nil
	}
	if t.outW != nil {
		t.outW.Close()
		t.outW = nil
	}
}

// envBlock encodes env the way CreateProcess reads it: NUL-terminated
// "NAME=value" strings, sorted by name, then a final NUL.
func envBlock(env []string) ([]uint16, error) {
	sorted := append([]string(nil), env...)
	// Whole-string order is name order here: '=' sorts before every character
	// a name may continue with, so "PATH" stays in front of "PATHEXT".
	sort.Strings(sorted)
	block := make([]uint16, 0, 64)
	for _, kv := range sorted {
		if strings.IndexByte(kv, 0) >= 0 {
			return nil, errors.New("an environment variable contains a NUL byte")
		}
		block = append(block, utf16.Encode([]rune(kv))...)
		block = append(block, 0)
	}
	return append(block, 0), nil
}

func openShellTTYPlatform(cols, rows int) (shellTTY, error) {
	return newConPTY(cols, rows)
}

func startSceneTTYPlatform(argv, env []string, w, h int) (sceneTTY, error) {
	t, err := newConPTY(w, h)
	if err != nil {
		return nil, err
	}
	if err := t.Start(argv, env); err != nil {
		t.Close()
		return nil, err
	}
	return &windowsSceneTTY{t}, nil
}

// windowsSceneTTY is a scene program on a console of its own. Windows has no
// signal that stops a process without reaching into ntdll, so a scene cannot
// be paused; it is ended instead, and its console goes with it.
type windowsSceneTTY struct{ *conPTY }

func (t *windowsSceneTTY) Pause(paused bool) {}

func (t *windowsSceneTTY) Kill() {
	if t.proc != 0 {
		_ = windows.TerminateProcess(t.proc, 1)
	}
}
