package terminal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/thebanri/limoni/widgets"
)

// reflowTerm is a terminal that rewraps on a width change as kitty does: rows
// the terminal wrapped itself are joined again and cut to the new width, the
// cursor keeps its place in the text, and a row that ends in a newline stays a
// line of its own. Only what an inline band needs is modelled.
type reflowTerm struct {
	w, h      int
	x, y      int // y counts from the first row ever kept
	wrap      bool
	last      rune
	rows      [][]rune
	soft      []bool // row i continues row i-1
	unhandled []string
}

func newReflowTerm(w, h int) *reflowTerm {
	v := &reflowTerm{w: w, h: h}
	for range h {
		v.rows, v.soft = append(v.rows, nil), append(v.soft, false)
	}
	return v
}

func (v *reflowTerm) top() int { return len(v.rows) - v.h }

func (v *reflowTerm) newline(soft bool) {
	if v.y == len(v.rows)-1 {
		v.rows, v.soft = append(v.rows, nil), append(v.soft, soft)
	} else if soft {
		v.soft[v.y+1] = true
	}
	v.y++
}

func (v *reflowTerm) put(r rune) {
	if v.wrap {
		v.x, v.wrap = 0, false
		v.newline(true)
	}
	for len(v.rows[v.y]) <= v.x {
		v.rows[v.y] = append(v.rows[v.y], ' ')
	}
	v.rows[v.y][v.x] = r
	v.last = r
	if v.x == v.w-1 {
		v.wrap = true
	} else {
		v.x++
	}
}

func (v *reflowTerm) cutRow(y, from int) {
	if from < len(v.rows[y]) {
		v.rows[y] = v.rows[y][:from]
	}
}

func (v *reflowTerm) feed(out string) {
	rs := []rune(out)
	for i := 0; i < len(rs); i++ {
		switch r := rs[i]; {
		case r == '\r':
			v.x, v.wrap = 0, false
		case r == '\n':
			v.wrap = false
			v.newline(false)
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
				v.y = max(v.top(), v.y-max(n, 1))
			case 'B':
				v.y = min(len(v.rows)-1, v.y+max(n, 1))
			case 'K':
				v.cutRow(v.y, v.x)
			case 'J':
				v.cutRow(v.y, v.x)
				for y := v.y + 1; y < len(v.rows); y++ {
					v.rows[y], v.soft[y] = nil, false
				}
			case 'n', 'c', 'q', 'X':
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

// resize rewraps every line to width w. The cursor stays at its offset in its
// line; at the end of a line that fills its last row it waits to wrap.
func (v *reflowTerm) resize(w int) {
	var lines [][]rune
	curLine, curOff := 0, 0
	for y, row := range v.rows {
		if !v.soft[y] || len(lines) == 0 {
			lines = append(lines, nil)
		}
		if y == v.y {
			curLine, curOff = len(lines)-1, len(lines[len(lines)-1])+v.x
			if v.wrap {
				curOff++
			}
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], padTo(row, v.w, y < len(v.rows)-1 && v.soft[y+1])...)
	}
	v.w = w
	v.rows, v.soft = nil, nil
	for li, l := range lines {
		l = []rune(strings.TrimRight(string(l), " "))
		if li == curLine && curOff > len(l) {
			l = append(l, []rune(strings.Repeat(" ", curOff-len(l)))...)
		}
		for i := 0; i == 0 || i*w < len(l); i++ {
			v.rows = append(v.rows, l[min(i*w, len(l)):min((i+1)*w, len(l))])
			v.soft = append(v.soft, i > 0)
		}
		if li == curLine {
			row := len(v.rows) - 1 - (len(l)-1)/w
			if len(l) == 0 {
				row = len(v.rows) - 1
			}
			v.y, v.x, v.wrap = row+curOff/w, curOff%w, false
			if curOff > 0 && curOff == len(l) && curOff%w == 0 {
				v.y, v.x, v.wrap = row+curOff/w-1, w-1, true
			}
		}
	}
	// Blank rows below the cursor are screen, not content: they never push
	// the text above them into the scrollback.
	for len(v.rows) > v.y+1 && len(v.rows[len(v.rows)-1]) == 0 {
		v.rows, v.soft = v.rows[:len(v.rows)-1], v.soft[:len(v.soft)-1]
	}
	for len(v.rows) < v.h {
		v.rows, v.soft = append(v.rows, nil), append(v.soft, false)
	}
}

// padTo gives a row that a wrap continues its full width, since the cells it
// holds are all used; a row that ends a line keeps its length.
func padTo(row []rune, w int, continued bool) []rune {
	if !continued {
		return row
	}
	for len(row) < w {
		row = append(row, ' ')
	}
	return row
}

// lines are the logical lines on screen and above it, trailing blanks cut.
func (v *reflowTerm) lines() []string {
	var out []string
	for y, row := range v.rows {
		if v.soft[y] && len(out) > 0 {
			out[len(out)-1] += string(row)
		} else {
			out = append(out, string(row))
		}
	}
	for i := range out {
		out[i] = strings.TrimRight(out[i], " ")
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// A width change reflows the band's rows and the committed ones: after every
// resize the screen must show the committed lines whole and the band once, at
// its new width, with no row of an older band left behind.
func TestInlineBandSurvivesWidthChanges(t *testing.T) {
	term, io := newInlineTerminal(t, 40, 12, 3)
	term.SetInlineFit(true)
	v := newReflowTerm(40, 12)
	fed := 0
	sync := func() {
		out := io.Output()
		v.feed(string(out[fed:]))
		fed = len(out)
	}
	band := func(f *Frame) {
		w := int(f.Buffer.Area.Width)
		drawLines(strings.Repeat("=", w), "status", strings.Repeat("-", w))(f)
	}
	bandLines := func(w int) []string {
		return []string{strings.Repeat("=", w), "status", strings.Repeat("-", w)}
	}

	if err := term.Draw(band); err != nil {
		t.Fatal(err)
	}
	if err := term.Commit(1, widgets.NewLabel("committed one")); err != nil {
		t.Fatal(err)
	}
	if err := term.Draw(band); err != nil {
		t.Fatal(err)
	}
	sync()
	want := append([]string{"committed one"}, bandLines(40)...)
	if got := v.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("before any resize:\n%q\nwant\n%q", got, want)
	}

	for _, w := range []int{25, 12, 30, 60, 16, 40} {
		v.resize(w)
		term.driver.SetSize(uint16(w), 12)
		if err := term.Draw(band); err != nil {
			t.Fatal(err)
		}
		sync()
		// The committed row was cut by the terminal's own reflow, not by us;
		// it is still one logical line.
		want := append([]string{"committed one"}, bandLines(w)...)
		if got := v.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("after resizing to %d columns:\n%q\nwant\n%q", w, got, want)
		}
	}

	// Leaving puts the shell's prompt right under the band.
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}
	sync()
	// The driver leaves a blank row under the band before the prompt.
	if want := len(v.lines()) + 1; v.y != want {
		t.Errorf("cursor on row %d after Close, want %d", v.y, want)
	}
	if len(v.unhandled) > 0 {
		t.Errorf("sequences the model does not know: %q", v.unhandled)
	}
}
