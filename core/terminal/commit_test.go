package terminal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// lineTerm is the part of a terminal an inline band relies on: a screen
// whose top row scrolls into a scrollback when a newline reaches the bottom,
// auto-wrap, and the cursor movement and erasure DiffInline and Commit use.
type lineTerm struct {
	w, h      int
	x, y      int
	wrap      bool // a character went into the last column
	screen    [][]rune
	scrolled  []string
	last      rune
	unhandled []string
}

func newLineTerm(w, h int) *lineTerm {
	v := &lineTerm{w: w, h: h}
	for range h {
		v.screen = append(v.screen, []rune(strings.Repeat(" ", w)))
	}
	return v
}

func (v *lineTerm) newline() {
	if v.y < v.h-1 {
		v.y++
		return
	}
	v.scrolled = append(v.scrolled, strings.TrimRight(string(v.screen[0]), " "))
	v.screen = append(v.screen[1:], []rune(strings.Repeat(" ", v.w)))
}

func (v *lineTerm) put(r rune) {
	if v.wrap {
		v.x, v.wrap = 0, false
		v.newline()
	}
	v.screen[v.y][v.x] = r
	v.last = r
	if v.x == v.w-1 {
		v.wrap = true
	} else {
		v.x++
	}
}

func (v *lineTerm) feed(out string) {
	rs := []rune(out)
	for i := 0; i < len(rs); i++ {
		switch r := rs[i]; {
		case r == '\r':
			v.x, v.wrap = 0, false
		case r == '\n':
			v.wrap = false
			v.newline()
		case r == 0x1b && i+1 < len(rs) && rs[i+1] == '[':
			j := i + 2
			for j < len(rs) && (rs[j] < 0x40 || rs[j] > 0x7e) {
				j++
			}
			params, final := string(rs[i+2:j]), rs[j]
			i = j
			if strings.HasPrefix(params, "?") || final == 'm' {
				continue
			}
			n, _ := strconv.Atoi(params)
			v.wrap = false
			switch final {
			case 'A':
				v.y = max(0, v.y-max(n, 1))
			case 'B':
				v.y = min(v.h-1, v.y+max(n, 1))
			case 'K':
				for x := v.x; x < v.w; x++ {
					v.screen[v.y][x] = ' '
				}
			case 'J':
				for x := v.x; x < v.w; x++ {
					v.screen[v.y][x] = ' '
				}
				for y := v.y + 1; y < v.h; y++ {
					v.screen[y] = []rune(strings.Repeat(" ", v.w))
				}
			case 'X':
				for x := v.x; x < min(v.w, v.x+max(n, 1)); x++ {
					v.screen[v.y][x] = ' '
				}
			case 'n', 'c', 'q':
				// The capability probe's queries; nothing on screen changes.
			case 'b':
				for range max(n, 1) {
					v.put(v.last)
				}
			default:
				v.unhandled = append(v.unhandled, params+string(final))
			}
		case r == 0x1b && i+1 < len(rs) && rs[i+1] == ']':
			for i < len(rs) && rs[i] != 0x07 && !(rs[i] == 0x1b && i+1 < len(rs) && rs[i+1] == '\\') {
				i++
			}
		case r == 0x1b:
			i++
		case r >= ' ':
			v.put(r)
		}
	}
}

// lines is the scrollback followed by the screen, trailing blanks trimmed.
func (v *lineTerm) lines() []string {
	all := append([]string(nil), v.scrolled...)
	for _, row := range v.screen {
		all = append(all, strings.TrimRight(string(row), " "))
	}
	for len(all) > 0 && all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	return all
}

func newInlineTerminal(t *testing.T, w, h, band uint16) (*Terminal, *driver.MemoryTerminalIO) {
	t.Helper()
	io := driver.NewMemoryTerminalIO(nil, w, h)
	b := driver.NewPortableBackend(io)
	b.SetInline(band)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	term.SetInline(band)
	return term, io
}

func drawLines(lines ...string) func(f *Frame) {
	return func(f *Frame) {
		for i, l := range lines {
			f.RenderWidget(widgets.NewLabel(l), cell.NewRect(0, uint16(i), f.Buffer.Area.Width, 1))
		}
	}
}

// What a chat does: messages committed above the band, from inside a frame
// and outside one, while the band shrinks and grows. The emulated terminal
// must end with every message in order, the band's last frame under them,
// and nothing of an earlier band left anywhere.
func TestCommitPrintsAboveTheBand(t *testing.T) {
	term, io := newInlineTerminal(t, 30, 8, 5)
	term.SetInlineFit(true)

	steps := []func() error{
		func() error { return term.Draw(drawLines("> ")) },
		func() error { return term.Commit(1, widgets.NewLabel("first message")) },
		func() error { return term.Draw(drawLines("streaming 1", "streaming 2", "streaming 3", "> ")) },
		func() error {
			return term.Draw(func(f *Frame) {
				if err := f.Commit(2, widgets.NewParagraph("reply one\nreply two")); err != nil {
					t.Error(err)
				}
				drawLines("> typing")(f)
			})
		},
		func() error { return term.Commit(1, widgets.NewLabel(strings.Repeat("x", 30))) },
		func() error { return term.Draw(drawLines("thinking", "still thinking", "> ")) },
		// The band shrinks with nothing committed: its old rows must go.
		func() error { return term.Draw(drawLines("> done")) },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}

	v := newLineTerm(30, 8)
	v.feed(string(io.Output()))
	got := v.lines()
	want := []string{"first message", "reply one", "reply two", strings.Repeat("x", 30), "> done"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("terminal shows\n%q\nwant\n%q", got, want)
	}
	if len(v.unhandled) > 0 {
		t.Errorf("sequences the model does not know: %q", v.unhandled)
	}
}

// Enough commits to push the first ones off the screen: they must be in the
// scrollback, in order, and the band must stay at the bottom.
func TestCommitScrollsIntoScrollback(t *testing.T) {
	term, io := newInlineTerminal(t, 20, 6, 2)
	var want []string
	for i := range 12 {
		msg := "message " + strconv.Itoa(i)
		want = append(want, msg)
		if err := term.Commit(1, widgets.NewLabel(msg)); err != nil {
			t.Fatal(err)
		}
		if err := term.Draw(drawLines("band", "> ")); err != nil {
			t.Fatal(err)
		}
	}
	v := newLineTerm(20, 6)
	v.feed(string(io.Output()))
	want = append(want, "band", ">")
	if got := v.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("terminal shows\n%q\nwant\n%q", got, want)
	}
}

// A height of zero measures: a Markdown by its wrapped rows, anything else
// by SizeHint at the terminal's width.
func TestCommitMeasuresAtZero(t *testing.T) {
	term, io := newInlineTerminal(t, 16, 8, 1)
	if err := term.Commit(0, widgets.NewMarkdown("# Title\n\nwords that wrap at sixteen")); err != nil {
		t.Fatal(err)
	}
	if err := term.Commit(0, widgets.NewParagraph("one\ntwo")); err != nil {
		t.Fatal(err)
	}
	if err := term.Draw(drawLines(">")); err != nil {
		t.Fatal(err)
	}
	v := newLineTerm(16, 8)
	v.feed(string(io.Output()))
	want := []string{"Title", "", "words that wrap", "at sixteen", "one", "two", ">"}
	if got := v.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("terminal shows\n%q\nwant\n%q", got, want)
	}
}

func TestCommitNeedsInline(t *testing.T) {
	io := driver.NewMemoryTerminalIO(nil, 20, 5)
	term, err := New(driver.NewPortableBackend(io))
	if err != nil {
		t.Fatal(err)
	}
	if err := term.Commit(1, widgets.NewLabel("x")); err != ErrNotInline {
		t.Errorf("full screen Commit = %v, want ErrNotInline", err)
	}
}
