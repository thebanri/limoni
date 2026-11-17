package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	color   string // its agent's
	hooked  bool   // its agent's Hooked
	dir     string // the worktree
	branch  string
	emu     *vt.SafeEmulator
	ptmx    *os.File
	cmd     *exec.Cmd
	started time.Time
	busy    time.Duration // time spent working before the current stretch; the UI's
	since   time.Time     // when the current working stretch began

	lastOut atomic.Int64 // unix nanoseconds of the last output that printed something
	ended   atomic.Int64 // unix nanoseconds the process exited, 0 while it runs
	idle    bool         // it was not working at the last frame; the UI's

	asked   atomic.Bool            // it signalled that it needs the user; cleared by userInput or working
	typed   atomic.Int64           // when the user last typed to it (not submitting)
	askText atomic.Pointer[string] // its own notification text
	sc      scanState              // signal scanner, read loop only
	osc     []byte
	clean   []byte // scan's output buffer
	drew    bool   // scan saw a printed character; only that counts as activity
	csi     []byte
	mouse   atomic.Uint32 // the mouse modes it turned on (signal.go's mouseModes)
	scroll  int           // lines of scrollback shown above the screen; the UI's

	mu    sync.Mutex
	files []fileChange // what it changed, refreshed by refresh

	chk check // the -check command's runs in its worktree

	task    string // the job it works on
	job     int    // which of the UI's jobs
	subsDir string // where its hooks leave their input (subCmd)
	subMu   sync.Mutex
	subs    []*subagent
}

// startRacer runs def on the task in dir, on a terminal w×h cells. wake is
// called whenever the screen changes.
func startRacer(def agentDef, name, prompt, dir, branch string, w, h int, wake func()) (*racer, error) {
	r := &racer{name: name, dir: dir, branch: branch, emu: vt.NewSafeEmulator(w, h), started: time.Now()}
	r.since = r.started
	r.color, r.hooked = def.Color, def.Hooked
	r.cmd = exec.Command(def.Bin, def.Args(prompt, dir)...)
	r.cmd.Dir = dir
	// pty.StartWithSize, by hand: the agent is told its terminal's path, so
	// its hooks, which run without one, can signal on it (see claudeHooks).
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}
	defer tty.Close()
	if err := pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)}); err != nil {
		_ = ptmx.Close()
		return nil, err
	}
	r.cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "SQUEEZE_TTY="+tty.Name())
	// Its hooks' input goes beside the worktrees, out of its diff.
	r.subsDir = filepath.Join(filepath.Dir(filepath.Dir(dir)), "subs", name)
	if err := os.MkdirAll(r.subsDir, 0o755); err == nil {
		r.cmd.Env = append(r.cmd.Env, "SQUEEZE_SUBS="+r.subsDir)
	}
	if def.Env != nil {
		r.cmd.Env = append(r.cmd.Env, def.Env(dir)...)
	}
	r.cmd.Stdin, r.cmd.Stdout, r.cmd.Stderr = tty, tty, tty
	r.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := r.cmd.Start(); err != nil {
		_ = ptmx.Close()
		return nil, err
	}
	r.ptmx = ptmx
	r.lastOut.Store(time.Now().UnixNano())

	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				r.drew = false
				_, _ = r.emu.Write(r.scan(buf[:n]))
				// agy re-sends its modes every few seconds while it waits:
				// output that prints nothing is not the agent working.
				if r.drew {
					r.printed()
				}
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
	case (r.asked.Load() || now.Sub(time.Unix(0, r.lastOut.Load())) >= quiet) && !r.background():
		return waiting
	}
	return working
}

// elapsed is how long it has worked: the clock stops while it waits for
// the user or has exited.
func (r *racer) elapsed(now time.Time) time.Duration {
	if r.idle {
		return r.busy
	}
	return r.busy + now.Sub(r.since)
}

// pause and resume track working stretches, as the UI sees them change.
// A stretch ends at the last thing the agent printed: waiting is noticed
// later (a signal after its last frame, or quiet seconds of silence).
func (r *racer) pause() {
	end := time.Unix(0, r.lastOut.Load())
	if e := r.ended.Load(); e != 0 {
		end = time.Unix(0, e)
	}
	if end.After(r.since) {
		r.busy += end.Sub(r.since)
	}
}

func (r *racer) resume(now time.Time) { r.since = now }

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
	r.chk.stop()
	if r.ended.Load() == 0 && r.cmd.Process != nil {
		_ = r.cmd.Process.Signal(syscall.SIGHUP)
	}
	// The emulator is left open: x/vt's Close races its own Read, which the
	// goroutine copying input to the agent is blocked in. That goroutine
	// stays until the process exits, and stop is only called on the way out.
	_ = r.ptmx.Close()
}
