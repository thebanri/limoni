package main

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// A scene drawn by a program of the user's: -scene-cmd.
//
// Any executable file — a Python script, a shell loop, a Go binary — can be
// a background that fills the screen and follows its size, as the built-in
// scenes do. It runs on a pseudo-terminal the size of the screen, which it
// is also told in LIMONI_COLS and LIMONI_ROWS, and draws there as on any
// terminal: colours, cursor movement, clearing. A terminal rather than a
// pipe, because on a pipe Python holds output back until a buffer fills and
// a newline does not return to the first column; and a program written for
// a terminal finds one.
//
// Its output goes through an emulator of its own, the size of the screen,
// and a row taller: a program that ends its last row with a newline, as
// print does, would otherwise scroll its first row away.
//
// A frame begins where the program moves the cursor home or clears the
// screen (ESC [ H, ESC [ 1 ; 1 H, ESC [ 2 J), and the frame before it is
// shown then, whole, so a half-drawn frame never is; a frame is also shown
// once the program falls quiet, so one that draws a single picture and
// sleeps, or exits, leaves that picture.
//
// A cell the program leaves without a background colour shows the
// terminal's own background, as a space does in art. When the window
// changes size the program is started again with the new size. It is
// stopped (SIGSTOP) while the scene cannot be seen — the window out of
// focus, or covered by a full-screen program — and killed, with anything it
// started, when another background is chosen or the wrapper exits. Its
// standard error is thrown away: the screen belongs to the shell.
type cmdScene struct {
	path string

	// Owned by the session's goroutine, which calls Render and Pause.
	w, h   int
	tty    sceneTTY    // the program this scene started, which it owns
	proc   *os.Process // a live region's program, which the shell started
	paused bool

	// Shared with the goroutine reading the program's output.
	mu      sync.Mutex
	gen     int         // which start of the program the frame belongs to
	frame   []cell.Cell // the last whole frame, w×h
	ready   bool        // frame holds one
	changed chan struct{}

	// cellOf converts the program's cells: sceneCell, or for a live region
	// (live.go) convertCell, since those are the shell's own cells.
	cellOf func(*uv.Cell) cell.Cell
	conn   io.Closer // a live region's connection, which brings the frames
}

// quietGap is how long a program's output must pause before what it drew
// is shown without a new frame beginning.
const quietGap = 25 * time.Millisecond

// frameStarts are the sequences that begin a frame.
var frameStarts = [][]byte{[]byte("\x1b[H"), []byte("\x1b[1;1H"), []byte("\x1b[2J")}

// newCmdScene checks that path is a program that can be run; it is started
// only when the scene is first drawn, so checking a setting starts nothing.
func newCmdScene(path string) (*cmdScene, error) {
	path = expandHome(path)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	if !isRunnable(path) {
		return nil, notRunnable(path)
	}
	return &cmdScene{path: path, changed: make(chan struct{}, 1), cellOf: sceneCell}, nil
}

// Interval is zero: the scene is not drawn on a clock but when the program
// has drawn a frame, which Changed announces.
func (c *cmdScene) Interval() time.Duration { return 0 }

// Changed receives when there is a new frame to show.
func (c *cmdScene) Changed() <-chan struct{} { return c.changed }

func (c *cmdScene) Render(dst *buffer.Buffer, _ time.Duration) {
	w, h := int(dst.Area.Width), int(dst.Area.Height)
	if w != c.w || h != c.h {
		c.start(w, h)
	}
	c.mu.Lock()
	if c.ready {
		copy(dst.Content, c.frame)
	} else {
		for i := range dst.Content {
			dst.Content[i] = cell.Cell{Content: ' '}
		}
	}
	c.mu.Unlock()
}

// start runs the program for a screen of w×h, stopping the one before.
func (c *cmdScene) start(w, h int) {
	c.stop()
	c.w, c.h = w, h
	c.mu.Lock()
	c.gen++
	gen := c.gen
	c.frame = make([]cell.Cell, w*h)
	c.ready = false
	c.mu.Unlock()
	if w == 0 || h == 0 {
		return
	}
	argv := []string{c.path}
	cols, rows := strconv.Itoa(w), strconv.Itoa(h)
	env := append(withoutEnv(os.Environ(), envNested), "LIMONI_COLS="+cols, "LIMONI_ROWS="+rows)
	// The terminal gives it a session of its own, so stopping and killing the
	// group reaches whatever it started too — the sleep in a shell loop, say.
	tty, err := startSceneTTY(argv, env, w, h)
	if err != nil {
		return // the scene stays empty; the shell is what matters
	}
	c.tty = tty
	if c.paused {
		tty.Pause(true)
	}
	go func() {
		c.read(gen, tty, w, h)
		tty.Close()
	}()
}

// stop ends the running program and everything it started: asked first,
// then killed a second later if it is still there.
func (c *cmdScene) stop() {
	if c.tty != nil {
		c.tty.Kill()
		c.tty = nil
	}
	if c.proc != nil {
		terminateProcess(c.proc)
		c.proc = nil
	}
}

// Pause stops the program while the scene cannot be seen, and lets it go on
// once it can: a background nobody sees costs nothing.
func (c *cmdScene) Pause(p bool) {
	if p == c.paused {
		return
	}
	c.paused = p
	if c.tty != nil {
		c.tty.Pause(p)
		return
	}
	pauseProcess(c.proc, p)
}

// Close ends the program for good.
func (c *cmdScene) Close() error {
	c.stop()
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// read feeds the program's output to an emulator of the screen's size and
// hands on each whole frame.
func (c *cmdScene) read(gen int, out io.ReadWriter, w, h int) {
	emu := vt.NewEmulator(w, h+1)
	// The emulator's answers to the program's queries go back to it; an
	// emulator whose answers are not read stops at the first one.
	go func() { _, _ = io.Copy(out, emu) }()
	defer func() {
		if pw, ok := emu.InputPipe().(io.Closer); ok {
			_ = pw.Close()
		}
	}()

	chunks := make(chan []byte, 8)
	go func() {
		defer close(chunks)
		buf := make([]byte, 64<<10)
		for {
			n, err := out.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	var rep repFilter // REP and titles, as for the shell (rep.go)
	var carry []byte  // the start of a frame sequence cut off by a read
	drawn := false    // something was drawn since the last frame was shown
	quiet := time.NewTimer(time.Hour)
	quiet.Stop()
	for {
		select {
		case b, ok := <-chunks:
			if !ok {
				if len(carry) > 0 {
					_, _ = emu.Write(rep.filter(carry))
					drawn = true
				}
				if drawn {
					c.publish(gen, emu, w, h)
				}
				return
			}
			b = append(carry, b...)
			b, carry = splitCarry(b)
			for len(b) > 0 {
				i, n := nextFrameStart(b)
				if i < 0 {
					_, _ = emu.Write(rep.filter(b))
					drawn = true
					break
				}
				if i > 0 {
					_, _ = emu.Write(rep.filter(b[:i]))
					drawn = true
				}
				if drawn {
					c.publish(gen, emu, w, h)
					drawn = false
				}
				_, _ = emu.Write(rep.filter(b[i : i+n]))
				b = b[i+n:]
			}
			if drawn {
				quiet.Reset(quietGap)
			}
		case <-quiet.C:
			if drawn {
				c.publish(gen, emu, w, h)
				drawn = false
			}
		}
	}
}

// nextFrameStart finds the first sequence in b that begins a frame: where
// it is and how long, or -1.
func nextFrameStart(b []byte) (at, n int) {
	for i := 0; i < len(b); i++ {
		j := bytes.IndexByte(b[i:], 0x1b)
		if j < 0 {
			return -1, 0
		}
		i += j
		for _, s := range frameStarts {
			if bytes.HasPrefix(b[i:], s) {
				return i, len(s)
			}
		}
	}
	return -1, 0
}

// splitCarry keeps back the end of b when it could be the start of a
// frame sequence the next read completes.
func splitCarry(b []byte) (rest, carry []byte) {
	for k := min(len(b), 5); k > 0; k-- {
		tail := b[len(b)-k:]
		for _, s := range frameStarts {
			if len(tail) < len(s) && bytes.HasPrefix(s, tail) {
				return b[:len(b)-k], append([]byte(nil), tail...)
			}
		}
	}
	return b, nil
}

// publish copies the emulated screen into the frame Render shows, unless
// the program has been started again since.
func (c *cmdScene) publish(gen int, emu *vt.Emulator, w, h int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gen != c.gen {
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c.frame[y*w+x] = c.cellOf(emu.CellAt(x, y))
		}
	}
	c.ready = true
	select {
	case c.changed <- struct{}{}:
	default: // one is waiting already
	}
}

// sceneCell turns a cell the program drew into a cell of the scene. Colours
// become RGB, so -opacity fades them like any scene; a cell with no
// background stays without one, and shows the terminal's. A scene's
// characters are one column wide, so a wide one becomes a space.
func sceneCell(c *uv.Cell) cell.Cell {
	if c == nil {
		return cell.Cell{Content: ' '}
	}
	out := cell.Cell{Content: ' ', Style: cell.Style{Fg: rgbOf(c.Style.Fg), Bg: rgbOf(c.Style.Bg)}}
	if c.Width == 1 && c.Content != "" {
		out.Content = cell.ClusterContent(c.Content, 1)
	}
	if c.Style.Attrs&uv.AttrBold != 0 {
		out.Style.Modifier = cell.ModifierBold
	}
	if c.Style.Attrs&uv.AttrReverse != 0 {
		out.Style.Fg, out.Style.Bg = out.Style.Bg, out.Style.Fg
	}
	return out
}

func rgbOf(c color.Color) cell.Color {
	if c == nil {
		return cell.NewColorDefault()
	}
	r, g, b, _ := c.RGBA()
	return cell.NewColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// withoutEnv is env without the variable key.
func withoutEnv(env []string, key string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return out
}

// closeScene ends a background that runs something of its own.
func closeScene(bd any) {
	if c, ok := bd.(io.Closer); ok {
		_ = c.Close()
	}
}
