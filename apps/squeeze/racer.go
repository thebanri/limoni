package main

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// quiet is the fallback for agents that never signal (see signal.go): this
// long without a byte of output counts as waiting. The agents animate a
// spinner while they work, so a screen that is still for this long is almost
// always a prompt, but a model can think silently for tens of seconds, so it
// is long; the signals are what make waiting prompt. A var so tests can
// shorten it.
var quiet = 30 * time.Second

// racer is one agent at work in its own worktree, on a pseudo-terminal whose
// output an emulator keeps as a screen.
type racer struct {
	name    string
	dir     string // the worktree
	branch  string
	emu     *vt.SafeEmulator
	ptmx    *os.File
	cmd     *exec.Cmd
	started time.Time

	lastOut  atomic.Int64 // unix nanoseconds of the last output
	ended    atomic.Int64 // unix nanoseconds the process exited, 0 while it runs
	notified bool         // the user was told it is waiting; the UI's

	asked   atomic.Bool            // it signalled that it needs the user; cleared by userInput
	askText atomic.Pointer[string] // its own notification text
	sc      scanState              // signal scanner, read loop only
	osc     []byte

	mu    sync.Mutex
	files []fileChange // what it changed, refreshed by refresh
}

// startRacer runs def on the task in dir, on a terminal w×h cells. wake is
// called whenever the screen changes.
func startRacer(def agentDef, name, prompt, dir, branch string, w, h int, wake func()) (*racer, error) {
	r := &racer{name: name, dir: dir, branch: branch, emu: vt.NewSafeEmulator(w, h), started: time.Now()}
	r.cmd = exec.Command(def.Bin, def.Args(prompt, dir)...)
	r.cmd.Dir = dir
	r.cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(r.cmd, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	if err != nil {
		return nil, err
	}
	r.ptmx = ptmx
	r.lastOut.Store(time.Now().UnixNano())

	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				r.scan(buf[:n])
				_, _ = r.emu.Write(buf[:n])
				r.lastOut.Store(time.Now().UnixNano())
				wake()
			}
			if err != nil {
				return
			}
		}
	}()
	// What the emulator sends the program: encoded keys and paste, and its
	// answers to the program's queries (cursor position, device attributes).
	go func() { _, _ = io.Copy(ptmx, r.emu) }()
	go func() {
		_ = r.cmd.Wait()
		r.ended.Store(time.Now().UnixNano())
		wake()
	}()
	return r, nil
}

type state int

const (
	working state = iota
	waiting       // alive, and signalled (or fell silent for quiet): it wants you
	done          // the process exited
)

func (r *racer) state(now time.Time) state {
	switch {
	case r.ended.Load() != 0:
		return done
	case r.asked.Load(), now.Sub(time.Unix(0, r.lastOut.Load())) >= quiet:
		return waiting
	}
	return working
}

// elapsed is how long it has run, or ran.
func (r *racer) elapsed(now time.Time) time.Duration {
	if e := r.ended.Load(); e != 0 {
		now = time.Unix(0, e)
	}
	return now.Sub(r.started)
}

func (r *racer) resize(w, h int) {
	if w == r.emu.Width() && h == r.emu.Height() {
		return
	}
	r.emu.Resize(w, h)
	_ = pty.Setsize(r.ptmx, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
}

// refresh reads what the racer has changed so far.
func (r *racer) refresh(base string) {
	files, err := changes(r.dir, base)
	if err != nil {
		return
	}
	r.mu.Lock()
	r.files = files
	r.mu.Unlock()
}

func (r *racer) changes() (files []fileChange, added, removed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.files {
		added += max(f.Added, 0)
		removed += max(f.Removed, 0)
	}
	return append([]fileChange(nil), r.files...), added, removed
}

// stop hangs up on the agent, as closing its terminal window would.
func (r *racer) stop() {
	if r.ended.Load() == 0 && r.cmd.Process != nil {
		_ = r.cmd.Process.Signal(syscall.SIGHUP)
	}
	// The emulator is left open: x/vt's Close races its own Read, which the
	// goroutine copying input to the agent is blocked in. That goroutine
	// stays until the process exits, and stop is only called on the way out.
	_ = r.ptmx.Close()
}
