package main

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/thebanri/limoni/core/cell"
)

// Selecting and copying.
//
// A terminal's own selection copies whatever is in the cells, and the
// scene's stars and the characters of ASCII art are in the cells like any
// text: selecting a command's output copied the picture behind it too. No
// terminal lets a cell be marked as not for copying. So, as tmux does, the
// wrapper takes the mouse, draws the selection itself, and puts on the
// clipboard only what the shell wrote — through the system's clipboard
// tool, or with OSC 52 where there is none (see copySelection).
//
// A program in the shell that asks for the mouse (vim with mouse=a, btop,
// htop) gets it, as before; the wrapper takes it back when the program lets
// go. Shift+drag still makes the terminal's own selection, background and
// all, in terminals that offer it.

// selectionColor is the background of selected cells.
var selectionColor = cell.NewColorRGB(64, 88, 150)

// doubleClick is the longest pause between the clicks of a double or triple
// click.
const doubleClick = 400 * time.Millisecond

// spot is a cell in the whole history: line 0 is the oldest line kept, and
// the live screen follows the scrollback. It stays on the same text while new
// output scrolls it up.
type spot struct{ line, x int }

func (a spot) before(b spot) bool { return a.line < b.line || a.line == b.line && a.x < b.x }

type selection struct {
	anchor, head spot
	unit         int  // 1 cells, 2 words, 3 lines
	active       bool // something is selected (and highlighted)
	dragging     bool // the left button is down
	moved        bool // it was dragged, not only clicked

	lastClick time.Time
	lastSpot  spot
	clicks    int

	copied string // the last selection, for a middle-click paste
}

// history is the number of scrollback lines before the live screen; the
// alternate screen has none.
func (s *session) history() int {
	if s.emu.IsAltScreen() {
		return 0
	}
	return len(s.hist)
}

// spotAt is the history cell under screen cell (x, y).
func (s *session) spotAt(x, y int) spot {
	x = min(max(x, 0), s.w-1)
	y = min(max(y, 0), s.h-1)
	return spot{line: s.history() - s.scroll + y, x: x}
}

// cellAt is the emulated cell at a spot, or nil past the end of a line.
func (s *session) cellAt(p spot) *uv.Cell {
	if h := s.history(); p.line < h {
		if l := s.hist[p.line]; p.x >= 0 && p.x < len(l) {
			return &l[p.x]
		}
		return nil
	} else {
		return s.emu.CellAt(p.x, p.line-h)
	}
}

// bounds is the selection in order, grown to whole words or lines.
func (s *session) bounds() (from, to spot) {
	from, to = s.sel.anchor, s.sel.head
	if to.before(from) {
		from, to = to, from
	}
	switch s.sel.unit {
	case 2:
		for from.x > 0 && isWordCell(s.cellAt(spot{from.line, from.x - 1})) {
			from.x--
		}
		for to.x < s.w-1 && isWordCell(s.cellAt(spot{to.line, to.x + 1})) {
			to.x++
		}
	case 3:
		from.x, to.x = 0, s.w-1
	}
	return from, to
}

func (s *session) selected(p spot) bool {
	if !s.sel.active {
		return false
	}
	from, to := s.bounds()
	return !p.before(from) && !to.before(p)
}

// isWordCell reports whether a cell belongs to a word for a double click:
// letters, digits, and the punctuation of paths and addresses, so a double
// click takes all of ~/src/app/main.go or user@host:8080.
func isWordCell(c *uv.Cell) bool {
	if c == nil || c.Content == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(c.Content)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_./~:@%+=#?&", r)
}

// selectedText is what the shell wrote under the selection, one line per
// row, without trailing blanks — never the background.
func (s *session) selectedText() string {
	from, to := s.bounds()
	var b strings.Builder
	for line := from.line; line <= to.line; line++ {
		x0, x1 := 0, s.w-1
		if line == from.line {
			x0 = from.x
		}
		if line == to.line {
			x1 = to.x
		}
		var row strings.Builder
		for x := x0; x <= x1; x++ {
			c := s.cellAt(spot{line, x})
			switch {
			case c == nil:
				row.WriteByte(' ')
			case c.Width == 0 && c.Content == "":
				// the second column of a wide character
			case c.Content == "":
				row.WriteByte(' ')
			default:
				row.WriteString(c.Content)
			}
		}
		b.WriteString(strings.TrimRight(row.String(), " "))
		if line < to.line {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// mouse handles a mouse report while the wrapper has the mouse.
func (s *session) mouse(m mouseEvent) {
	switch {
	case m.wheel():
		s.wheel(m.which() == 0)
	case m.which() == 0 && m.motion():
		if !s.sel.dragging {
			return
		}
		// Dragging past the top or bottom row scrolls the history.
		if m.y <= 0 && s.scroll < s.history() {
			s.scroll++
		} else if m.y >= s.h-1 && s.scroll > 0 {
			s.scroll--
		}
		s.sel.head = s.spotAt(m.x, m.y)
		s.sel.moved = true
		s.sel.active = true
		s.appDirty = true
	case m.which() == 0 && !m.release && m.button&16 != 0: // Ctrl
		s.openLink(m.x, m.y)
	case m.which() == 0 && !m.release:
		s.clickLive(m.x, m.y)
		p := s.spotAt(m.x, m.y)
		now := time.Now()
		if p == s.sel.lastSpot && now.Sub(s.sel.lastClick) < doubleClick {
			s.sel.clicks = s.sel.clicks%3 + 1
		} else {
			s.sel.clicks = 1
		}
		s.sel.lastClick, s.sel.lastSpot = now, p
		s.sel.anchor, s.sel.head = p, p
		s.sel.unit = s.sel.clicks
		s.sel.active = s.sel.clicks > 1
		s.sel.dragging, s.sel.moved = true, false
		s.appDirty = true
	case m.which() == 0 && m.release:
		s.sel.dragging = false
		if s.sel.active {
			s.copySelection()
		}
	case m.which() == 1 && !m.release && !m.motion():
		s.paste(s.sel.copied)
	}
}

// wheel scrolls the history, or, on the alternate screen where there is
// none, sends arrow keys the way terminals do for less and man.
func (s *session) wheel(up bool) {
	if s.emu.IsAltScreen() {
		key := "\x1b[B"
		if up {
			key = "\x1b[A"
		}
		if s.childModes[ansi.ModeCursorKeys] {
			key = "\x1bO" + key[2:]
		}
		_, _ = s.tty.Write([]byte(strings.Repeat(key, 3)))
		return
	}
	if up {
		s.scroll = min(s.scroll+3, s.history())
	} else {
		s.scroll = max(s.scroll-3, 0)
	}
	s.appDirty = true
}

// copySelection puts the selected text on the clipboard.
//
// On this machine, it goes through the system's own clipboard tool —
// wl-copy on Wayland, xclip or xsel on X11, pbcopy on macOS, clip on
// Windows — onto both the clipboard (Ctrl+V) and the primary selection
// (middle-click elsewhere), as a terminal's own selection does. That works
// in every terminal and for any length: OSC 52 is ignored by some terminals,
// and copying fastfetch's output that way failed in Alacritty while it
// worked in kitty. Over SSH, or with no tool, the text goes to the terminal
// with OSC 52, the only way it can reach the machine the user sits at.
func (s *session) copySelection() {
	text := s.selectedText()
	if text == "" {
		return
	}
	s.sel.copied = text
	if copyLocal(text) {
		return
	}
	s.write("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07")
}

// clipboardTools are the commands that set the clipboard and the primary
// selection, per display system, in the order they are tried. Windows has no
// primary selection: clip sets the clipboard and nothing else.
var clipboardTools = []struct {
	env                string // the variable that says this display is in use
	goos               string // the only system the tool exists on, "" for any
	clipboard, primary []string
}{
	{"WAYLAND_DISPLAY", "", []string{"wl-copy"}, []string{"wl-copy", "--primary"}},
	{"DISPLAY", "", []string{"xclip", "-selection", "clipboard"}, []string{"xclip", "-selection", "primary"}},
	{"DISPLAY", "", []string{"xsel", "--clipboard", "--input"}, []string{"xsel", "--primary", "--input"}},
	{"", "darwin", []string{"pbcopy"}, nil},
	{"", "windows", []string{"clip"}, nil},
}

// copyLocal copies with the first clipboard tool that works here, and
// reports whether one did. It is not tried over SSH, where the local
// clipboard is the server's, not the user's.
func copyLocal(text string) bool {
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return false
	}
	for _, t := range clipboardTools {
		if t.goos != "" && runtime.GOOS != t.goos {
			continue
		}
		if t.env != "" && os.Getenv(t.env) == "" {
			continue
		}
		if _, err := exec.LookPath(t.clipboard[0]); err != nil {
			continue
		}
		if runClip(t.clipboard, text) != nil {
			continue
		}
		if t.primary != nil {
			_ = runClip(t.primary, text)
		}
		return true
	}
	return false
}

// runClip runs a clipboard tool with text on its input. wl-copy and xclip
// fork a process that keeps serving the selection and return at once; the
// timeout is for one that does not.
func runClip(argv []string, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// paste types text into the shell, bracketed if the shell asked for it.
func (s *session) paste(text string) {
	if text == "" {
		return
	}
	if s.childModes[ansi.ModeBracketedPaste] {
		text = "\x1b[200~" + text + "\x1b[201~"
	}
	_, _ = s.tty.Write([]byte(text))
}

// clearSelection drops the highlight, as typing does.
func (s *session) clearSelection() {
	if s.sel.active {
		s.sel.active = false
		s.appDirty = true
	}
}

// highlight paints the selection over the composed frame: the shell's text
// on the selection colour, and the background left out, so what is shown
// selected is exactly what is copied.
func (s *session) highlight() {
	if !s.sel.active {
		return
	}
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			if !s.selected(s.spotAt(x, y)) {
				continue
			}
			i := y*s.w + x
			c := s.app.Content[i]
			c.Style.Bg = selectionColor
			c.Style.Modifier &^= cell.ModifierReverse
			s.front.Content[i] = c
		}
	}
}

// Mouse modes: the wrapper's own (button events, SGR coordinates), or, while
// a program in the shell has asked for the mouse, exactly the program's.
var mouseModes = []ansi.DECMode{
	ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseButtonEvent,
	ansi.ModeMouseAnyEvent, ansi.DECMode(1005), ansi.ModeMouseExtSgr, ansi.DECMode(1015),
}

func isMouseMode(m ansi.DECMode) bool {
	for _, mm := range mouseModes {
		if m == mm {
			return true
		}
	}
	return false
}

// childWantsMouse reports whether the program in the shell asked for mouse
// reports.
func (s *session) childWantsMouse() bool {
	return s.childModes[ansi.ModeMouseX10] || s.childModes[ansi.ModeMouseNormal] ||
		s.childModes[ansi.ModeMouseButtonEvent] || s.childModes[ansi.ModeMouseAnyEvent]
}

// syncMouse sets the terminal's mouse modes to whoever should have the
// mouse, sending only what changed.
func (s *session) syncMouse() {
	own := s.opts.selection && !s.childWantsMouse()
	s.ownMouse.Store(own)
	for _, m := range mouseModes {
		want := s.childModes[m]
		if own {
			want = m == ansi.ModeMouseButtonEvent || m == ansi.ModeMouseExtSgr
		}
		if want == s.outerMouse[m] {
			continue
		}
		s.outerMouse[m] = want
		s.pending = appendMode(s.pending, m, want)
	}
	if !own {
		s.sel.dragging = false
	}
}

// openLink opens the link (OSC 8) under a Ctrl+click in the browser, as a
// terminal does: the terminal cannot, since the wrapper has the mouse. Only
// web addresses: anything printed can carry a link, a file being cat too.
func (s *session) openLink(x, y int) {
	c := s.cellAt(s.spotAt(x, y))
	if c == nil {
		return
	}
	if u := c.Link.URL; strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		openURL(u)
	}
}

// openURL is the system's opener; tests replace it.
var openURL = func(u string) {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, u)
	if cmd.Start() == nil {
		go func() { _ = cmd.Wait() }()
	}
}
