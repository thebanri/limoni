package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/thebanri/limoni/backdrop"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
)

// minFrame is the shortest time between two frames drawn for the shell's
// output. A command that prints a flood is drawn at most this often; a key
// typed after a pause is drawn at once.
const minFrame = 8 * time.Millisecond

// scrollbackLines is how much history Shift+PageUp can reach.
const scrollbackLines = 10000

// forwardedModes are the modes the shell sets that change what the real
// terminal has to send: cursor key encoding, mouse reports and bracketed
// paste. They are passed on as they come. The rest — the alternate screen,
// the cursor, origin mode — belong to the emulated screen and stay there.
var forwardedModes = map[ansi.DECMode]bool{
	ansi.ModeCursorKeys:     true,
	ansi.ModeBracketedPaste: true,
}

type options struct {
	scene     string
	opacity   float64
	fps       float64
	still     bool
	selection bool // select and copy with the mouse, without the background
	image     string
	art       string
	sceneCmd  string // a program that draws the scene (scenecmd.go)
	argv      []string

	// backdrop is the scene the options above chose, loaded before the
	// terminal is taken over so a bad file is reported as usual.
	backdrop terminal.Backdrop
}

type session struct {
	opts   options
	in     *os.File
	out    *os.File
	caps   terminal.CapabilityProfile
	scene  terminal.Backdrop
	start  time.Time
	tty    shellTTY
	emu    *vt.Emulator
	w, h   int
	cursor struct {
		hidden bool
		shape  int // DECSCUSR parameter, 0 for the terminal's default
	}
	childModes map[ansi.DECMode]bool
	outerMouse map[ansi.DECMode]bool // mouse modes the real terminal has on
	ownMouse   atomic.Bool           // the wrapper, not a program in the shell, has the mouse
	sel        selection
	splits     map[int]splitLine // lines a reflow split, by line in the history
	// hist is the history above the screen, oldest first. The emulator's own
	// scrollback is emptied into it after every write, so a resize can
	// reflow it without copying every line of it (see reflow).
	hist []uv.Line
	// earned is how many lines at the start of hist are history proper:
	// scrolled off by output or pushed there by clear. The lines after them
	// a reflow pushed off the screen, and may bring back when the window
	// has room again.
	earned int
	// termBg is the terminal's background colour, which the scene fades into.
	termBg cell.Color
	// rep writes out REP for the emulator, which repeats only ASCII (rep.go).
	rep        repFilter
	name       procName // the foreground program's name, taken on for the terminal's tab
	scroll     int      // lines scrolled back into history, 0 at the live screen
	focused    bool     // the terminal window has focus
	sceneFrame int      // the scene frame last rendered, -1 for none

	live    *liveRegion // rows a program keeps drawing (live.go)
	waiting *liveConn   // a program's connection that came before its region
	liveOK  bool        // the region's rows still hold its print

	app, bg, front, back *buffer.Buffer
	appDirty             bool
	pending              []byte // escape sequences for the real terminal, sent with the next frame
	outBuf               []byte
	lastFrame            time.Time
}

type inputMsg struct {
	events []inputEvent
}

// run starts the shell behind the scene and returns its exit code.
func run(opts options) (int, error) {
	s := &session{
		opts:       opts,
		in:         os.Stdin,
		out:        os.Stdout,
		caps:       terminal.DetectCapabilities(),
		childModes: map[ansi.DECMode]bool{},
		outerMouse: map[ansi.DECMode]bool{},
		focused:    true,
		sceneFrame: -1,
	}
	s.w, s.h = windowSize(int(s.out.Fd()))

	restore, err := makeRaw(s.in, s.out)
	if err != nil {
		return 1, err
	}
	defer restore()

	fg, bg, typeahead := terminalColors(s.in, s.out)
	s.termBg = bg
	s.scene = fadeInto(opts.backdrop, bg, opts.opacity)

	// The shell is told which terminal is the wrapper's: its own. A terminal
	// window opened from inside it inherits the variable but has a terminal
	// of its own, and so gets a background of its own.
	tty, err := openShellTTY(s.w, s.h)
	if err != nil {
		return 1, err
	}
	exe, _ := os.Executable()
	env := append(withOwnDir(os.Environ(), exe), envNested+"="+tty.Name(), envSocket+"="+controlSocket())
	if err := tty.Start(opts.argv, env); err != nil {
		tty.Close()
		return 1, err
	}
	s.tty = tty
	// Named after the shell from the start; the shell's first output, by
	// when it has surely replaced the forked copy of this process, is when
	// its own name can be read (procname.go).
	s.name.set(filepath.Base(opts.argv[0]))
	defer s.tty.Close()

	s.emu = vt.NewEmulator(s.w, s.h)
	s.emu.SetScrollbackSize(scrollbackLines)
	if fg.Type() == cell.ColorRGB {
		r, g, b := fg.RGB()
		s.emu.SetDefaultForegroundColor(rgba(r, g, b))
	}
	if bg.Type() == cell.ColorRGB {
		r, g, b := bg.RGB()
		s.emu.SetDefaultBackgroundColor(rgba(r, g, b))
	}
	s.rep.title = s.setTitle // titles the emulator would print (rep.go)
	s.emu.SetCallbacks(vt.Callbacks{
		Bell:             func() { s.pending = append(s.pending, '\a') },
		Title:            s.setTitle,
		CursorVisibility: func(v bool) { s.cursor.hidden = !v },
		CursorStyle:      s.setCursorShape,
		EnableMode:       func(m ansi.Mode) { s.setChildMode(m, true) },
		DisableMode:      func(m ansi.Mode) { s.setChildMode(m, false) },
	})
	s.emu.RegisterOscHandler(oscLive, s.markLive)

	s.app = buffer.NewBuffer(cell.NewRect(0, 0, uint16(s.w), uint16(s.h)))
	s.bg = buffer.NewBuffer(s.app.Area)
	s.front = buffer.NewBuffer(s.app.Area)
	s.back = buffer.NewEmptyBuffer() // a size mismatch: the first frame is drawn whole

	// The alternate screen, so the user's scrollback is left as it was; focus
	// reports, so the scene can rest while the window is in the background;
	// and no alternate scroll, or the wheel would send arrow keys to the
	// shell and walk its history.
	s.write("\x1b[?1049h\x1b[?1004h\x1b[?1007l\x1b[H\x1b[2J")
	s.syncMouse()
	defer s.write("\x1b[?1004l\x1b[?1049l\x1b[0 q\x1b[?25h" + resetForwardedModes())

	output := make(chan []byte, 64)
	go readShell(s.tty, output)
	go func() { _, _ = io.Copy(s.tty, s.emu) }() // the emulator's answers to the shell's queries
	inputs := make(chan inputMsg, 16)
	go s.readInput(inputs)
	if len(typeahead) > 0 {
		_, _ = s.tty.Write(typeahead)
	}

	// A terminal window changing size reaches Unix as SIGWINCH; Windows has
	// to poll the console. Both feed the same check in the loop below.
	resizes := make(chan struct{}, 1)
	resizeDone := make(chan struct{})
	defer close(resizeDone)
	go watchResize(int(s.out.Fd()), resizes, resizeDone)

	// Settings changed while running (backdrop-shell opacity, enable,
	// reset) arrive here. Without the socket the terminal still works; it
	// only will not follow such changes until it is opened again.
	reloads := make(chan struct{}, 1)
	lives := make(chan liveConn)
	if stop, err := listenControl(reloads, lives); err == nil {
		defer stop()
	}
	defer func() {
		s.endLive()
		if s.waiting != nil {
			s.waiting.c.Close()
		}
	}()

	s.start = time.Now()
	var ticker *time.Ticker
	var tickerIv time.Duration
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()
	settle := time.NewTimer(time.Hour)
	settle.Stop()
	var settled <-chan time.Time

	defer func() { closeScene(s.opts.backdrop) }()
	s.draw()
	for {
		// The scene ticks only while it can be seen: the window has focus
		// and something on screen is left clear for it. A program drawing
		// the scene is stopped meanwhile, and says itself when it has drawn.
		seen := s.focused && s.sceneVisible()
		var drawn <-chan struct{}
		if p, ok := s.opts.backdrop.(*cmdScene); ok {
			p.Pause(!seen)
			if seen {
				drawn = p.Changed()
			}
		}
		var liveDrawn <-chan struct{}
		if s.live != nil && s.live.scene != nil {
			shown := s.focused && s.liveSeen()
			s.live.scene.Pause(!shown)
			if shown {
				liveDrawn = s.live.scene.Changed()
			}
		}
		var tick <-chan time.Time
		if iv := s.interval(); iv > 0 && seen {
			if ticker == nil {
				ticker, tickerIv = time.NewTicker(iv), iv
			} else if iv != tickerIv {
				ticker.Reset(iv) // a reload changed the scene or its pace
				tickerIv = iv
			}
			tick = ticker.C
		} else if ticker != nil {
			ticker.Stop()
			ticker = nil
		}

		select {
		case data, ok := <-output:
			if !ok {
				return s.tty.ExitCode(), nil
			}
			s.feed(data, output)
			s.name.update(s.tty)
			if since := time.Since(s.lastFrame); since < minFrame {
				if settled == nil {
					settle.Reset(minFrame - since)
					settled = settle.C
				}
				continue
			}
			s.draw()
		case <-settled:
			settled = nil
			s.draw()
		case msg := <-inputs:
			s.handleInput(msg)
		case <-tick:
			s.draw()
		case <-drawn:
			s.sceneFrame = -1 // a still scene is drawn again only then
			s.draw()
		case <-liveDrawn:
			s.draw()
		case lc := <-lives:
			s.connectLive(lc)
		case <-resizes:
			s.resize()
		case <-reloads:
			s.reload()
		}
	}
}

// setTerminalColor takes an answer to the colour queries that came after
// the wait at start-up, as it can when the terminal window is still opening:
// the scene is faded into the real background, and the shell is told it.
func (s *session) setTerminalColor(osc string, c cell.Color) {
	if c.Type() != cell.ColorRGB {
		return
	}
	r, g, b := c.RGB()
	if osc == "10" {
		s.emu.SetDefaultForegroundColor(rgba(r, g, b))
		return
	}
	s.emu.SetDefaultBackgroundColor(rgba(r, g, b))
	s.termBg = c
	s.scene = fadeInto(s.opts.backdrop, c, s.opts.opacity)
	s.sceneFrame = -1
	s.draw()
}

// fadeInto shows a background at the given opacity over the terminal's own
// background colour.
func fadeInto(bd terminal.Backdrop, termBg cell.Color, opacity float64) terminal.Backdrop {
	return backdrop.Fade(bd, termBg, opacity)
}

// interval is how often the scene is drawn: its own pace, or slower if the
// user capped the frame rate.
func (s *session) interval() time.Duration {
	iv := s.scene.Interval()
	if iv > 0 && s.opts.fps > 0 {
		iv = max(iv, time.Duration(float64(time.Second)/s.opts.fps))
	}
	return iv
}

// feed hands the shell's output to the emulator, along with whatever else
// is already waiting, so a burst is one frame.
func (s *session) feed(data []byte, output chan []byte) {
	s.write1(data)
	for {
		select {
		case more, ok := <-output:
			if !ok {
				s.appDirty = true
				return
			}
			s.write1(more)
			continue
		default:
		}
		break
	}
	s.appDirty = true
	s.scroll = min(s.scroll, s.history())
}

// write1 hands one chunk of the shell's output to the emulator and takes
// the lines it scrolled off into hist. "Erase saved lines" (CSI 3 J, which
// clear sends) empties hist too: the emulator only empties its own
// scrollback, which hist has already emptied.
func (s *session) write1(data []byte) {
	data = s.rep.filter(data)
	_, _ = s.emu.Write(data)
	if bytes.Contains(data, []byte("\x1b[3J")) {
		s.emu.ClearScrollback()
		s.hist, s.splits, s.earned = nil, nil, 0
		s.sel.active, s.sel.dragging = false, false
		s.scroll = 0
		s.endLive()
		return
	}
	s.drainHistory()
	if s.live != nil && s.live.print == nil {
		s.live.print = s.liveCells(nil)
	}
}

// drainHistory moves the lines the emulator scrolled off the screen into
// hist, keeping at most scrollbackLines of them.
func (s *session) drainHistory() {
	sb := s.emu.Scrollback()
	if sb.Len() == 0 {
		return
	}
	// Output scrolling a screen no reflow has touched makes history proper;
	// after lines a reflow pushed off, it joins those.
	proper := s.earned == len(s.hist)
	for _, l := range sb.Lines() {
		if n := len(l); n > 0 && l[n-1].Width > 1 {
			l = trimLine(l) // the scrollback drops the empty cell after a wide character
		}
		s.hist = append(s.hist, l)
	}
	sb.Clear()
	if proper {
		s.earned = len(s.hist)
	}
	if drop := len(s.hist) - scrollbackLines; drop > 0 {
		s.dropHistory(drop)
	}
}

// dropHistory forgets the oldest n lines, and moves what refers to lines by
// number along with the rest.
func (s *session) dropHistory(n int) {
	s.hist = append(s.hist[:0:0], s.hist[n:]...)
	if len(s.splits) > 0 {
		moved := map[int]splitLine{}
		for i, sp := range s.splits {
			if i >= n {
				moved[i-n] = sp
			}
		}
		s.splits = moved
	}
	s.sel.anchor.line -= n
	s.sel.head.line -= n
	s.sel.lastSpot.line -= n
	s.scroll = min(s.scroll, len(s.hist))
	s.earned = max(0, s.earned-n)
	if s.live != nil {
		if s.live.line -= n; s.live.line < 0 {
			s.endLive()
		}
	}
}

func (s *session) handleInput(msg inputMsg) {
	for _, ev := range msg.events {
		switch ev.kind {
		case inputMouse:
			if s.ownMouse.Load() {
				s.mouse(ev.mouse)
			}
		case inputColor:
			s.setTerminalColor(ev.osc, ev.color)
		case inputScrollUp:
			if !s.emu.IsAltScreen() {
				s.scroll = min(s.scroll+s.h/2, s.history())
				s.appDirty = true
			}
		case inputScrollDown:
			s.scroll = max(0, s.scroll-s.h/2)
			s.appDirty = true
		case inputFocusIn, inputFocusOut:
			s.focused = ev.kind == inputFocusIn
			if s.childModes[ansi.ModeFocusEvent] {
				report := reportFocusIn
				if ev.kind == inputFocusOut {
					report = reportFocusOut
				}
				_, _ = s.tty.Write(report)
			}
		case inputOther:
			if s.scroll > 0 {
				s.scroll = 0
				s.appDirty = true
			}
			s.clearSelection()
		}
	}
	if s.appDirty {
		s.draw()
	}
}

func (s *session) resize() {
	w, h := windowSize(int(s.out.Fd()))
	if w == s.w && h == s.h {
		return
	}
	// Reflow before telling the shell, so what it draws for the new size
	// lands on the reflowed screen.
	s.reflow(w, h)
	s.w, s.h = w, h
	_ = s.tty.Resize(w, h)
	area := cell.NewRect(0, 0, uint16(w), uint16(h))
	s.app.Resize(area)
	s.bg.Resize(area)
	s.front.Resize(area)
	s.sceneFrame = -1
	s.appDirty = true
	s.scroll = 0
	s.sel.active, s.sel.dragging = false, false // its lines have moved
	s.relocateLive()                            // a reflow moves its rows
	s.draw()
}

// draw composes a frame and sends what changed since the last one.
func (s *session) draw() {
	s.lastFrame = time.Now()
	if s.appDirty {
		s.copyScreen()
		s.appDirty = false
	}
	// The scene is drawn again only when it has moved on a frame; a still
	// one only after a resize.
	if iv := s.interval(); iv <= 0 {
		if s.sceneFrame < 0 {
			s.scene.Render(s.bg, 0)
			s.sceneFrame = 0
		}
	} else if f := int(time.Since(s.start) / iv); f != s.sceneFrame {
		s.scene.Render(s.bg, time.Duration(f)*iv)
		s.sceneFrame = f
	}
	copy(s.front.Content, s.app.Content)
	s.drawLive()
	terminal.ComposeBackdrop(s.front.Content, s.bg.Content)
	s.highlight()
	s.front.Invalidate()

	out := append(s.outBuf[:0], "\x1b[?2026h"...)
	out = append(out, s.pending...)
	s.pending = s.pending[:0]
	bodyStart := len(out)
	out, _ = buffer.DiffWithOptions(s.front, s.back, out, buffer.DiffOptions{
		TrueColor:  s.caps.TrueColor,
		Colors256:  s.caps.Colors256,
		EraseChar:  s.caps.EraseChar,
		RepeatChar: s.caps.RepeatChar,
		Hyperlinks: s.caps.Hyperlinks,
	})
	cx, cy := s.cursorAt()
	out = append(out, "\x1b["...)
	out = strconv.AppendInt(out, int64(cy+1), 10)
	out = append(out, ';')
	out = strconv.AppendInt(out, int64(cx+1), 10)
	out = append(out, 'H')
	if s.cursor.hidden || s.scroll > 0 {
		out = append(out, "\x1b[?25l"...)
	} else {
		out = append(out, "\x1b[?25h"...)
	}
	out = append(out, "\x1b[?2026l"...)
	s.outBuf = out
	if len(out) > bodyStart {
		s.write(string(out))
	}
}

// cursorAt is where the real cursor goes: the shell's cursor, or the
// bottom-right corner while history is shown.
func (s *session) cursorAt() (x, y int) {
	if s.scroll > 0 {
		return s.w - 1, s.h - 1
	}
	p := s.emu.CursorPosition()
	return min(max(p.X, 0), s.w-1), min(max(p.Y, 0), s.h-1)
}

// copyScreen converts the emulated screen — or, scrolled back, the stretch
// of history and screen being looked at — into Limoni cells.
func (s *session) copyScreen() {
	w, h := s.w, s.h
	history := s.history()
	for y := 0; y < h; y++ {
		line := history - s.scroll + y
		row := s.app.Content[y*w : (y+1)*w]
		for x := 0; x < w; x++ {
			row[x] = convertCell(s.cellAt(spot{line: line, x: x}))
		}
	}
	s.liveOK = s.liveIntact()
}

// sceneVisible reports whether any cell leaves the scene showing. A
// full-screen program with its own background covers it all, and then the
// scene need not move.
func (s *session) sceneVisible() bool {
	for _, c := range s.app.Content {
		if c.Style.Bg.Type() == cell.ColorDefault && c.Style.Modifier&cell.ModifierReverse == 0 {
			return true
		}
	}
	return false
}

// setTitle passes the program's window title on to the terminal.
func (s *session) setTitle(t string) { s.pending = append(s.pending, ansi.SetWindowTitle(t)...) }

func (s *session) setCursorShape(style vt.CursorStyle, blink bool) {
	// DECSCUSR: 1-2 block, 3-4 underline, 5-6 bar; odd blinks.
	shape := 2*int(style) + 1
	if !blink {
		shape++
	}
	if shape == s.cursor.shape {
		return
	}
	s.cursor.shape = shape
	s.pending = append(s.pending, "\x1b["...)
	s.pending = strconv.AppendInt(s.pending, int64(shape), 10)
	s.pending = append(s.pending, " q"...)
}

func (s *session) setChildMode(m ansi.Mode, on bool) {
	dm, ok := m.(ansi.DECMode)
	if !ok {
		return
	}
	s.childModes[dm] = on
	switch {
	case isMouseMode(dm):
		// Who has the mouse may have changed; the terminal gets the
		// modes of whoever does.
		if s.outerMouse != nil {
			s.syncMouse()
		}
	case dm == ansi.ModeNumericKeypad:
		// DECKPAM / DECKPNM
		if on {
			s.pending = append(s.pending, "\x1b="...)
		} else {
			s.pending = append(s.pending, "\x1b>"...)
		}
	case forwardedModes[dm]:
		s.pending = appendMode(s.pending, dm, on)
	}
	if len(s.pending) > 0 && s.emu != nil {
		// A mode change must reach the terminal before the next key it
		// affects, not with the next frame.
		s.write(string(s.pending))
		s.pending = s.pending[:0]
	}
}

// readInput passes the keyboard to the shell, keeping back what the wrapper
// handles itself.
func (s *session) readInput(inputs chan<- inputMsg) {
	buf := make([]byte, 4096)
	var pass, carry []byte
	for {
		n, err := s.in.Read(buf)
		if n > 0 {
			var events []inputEvent
			in := buf[:n]
			if len(carry) > 0 {
				in = append(carry, in...)
			}
			pass, carry = splitInput(in, pass[:0], s.ownMouse.Load(), func(ev inputEvent) {
				if ev.kind == inputOther && len(events) > 0 && events[len(events)-1].kind == inputOther {
					return
				}
				events = append(events, ev)
			})
			// buf is read into again: keep the cut-off reply in its own
			// slice. One that never ends is not a reply; let it go.
			if len(carry) > 1024 {
				carry = nil
			}
			carry = append([]byte(nil), carry...)
			if len(pass) > 0 {
				_, _ = s.tty.Write(pass)
				if bytes.IndexByte(pass, '\r') >= 0 {
					// A command may be starting; by then it has.
					time.AfterFunc(150*time.Millisecond, func() { s.name.update(s.tty) })
				}
			}
			if len(events) > 0 {
				inputs <- inputMsg{events: events}
			}
		}
		if err != nil {
			return
		}
	}
}

func readShell(tty shellTTY, output chan<- []byte) {
	defer close(output)
	for {
		buf := make([]byte, 32*1024)
		n, err := tty.Read(buf)
		if n > 0 {
			output <- buf[:n]
		}
		if err != nil {
			return
		}
	}
}

// appendMode appends DECSET or DECRST for a mode.
func appendMode(out []byte, m ansi.DECMode, on bool) []byte {
	out = append(out, "\x1b[?"...)
	out = strconv.AppendInt(out, int64(m), 10)
	if on {
		return append(out, 'h')
	}
	return append(out, 'l')
}

func (s *session) write(str string) {
	_, _ = s.out.WriteString(str)
}

func resetForwardedModes() string {
	out := "\x1b>"
	for m := range forwardedModes {
		out += "\x1b[?" + strconv.Itoa(int(m)) + "l"
	}
	for _, m := range mouseModes {
		out += "\x1b[?" + strconv.Itoa(int(m)) + "l"
	}
	return out
}
