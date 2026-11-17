package main

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

// spot is a cell of a racer's output: line counts from the top of its
// scrollback, so a selection stays on its text as the agent prints more.
type spot struct{ line, col int }

func (a spot) before(b spot) bool { return a.line < b.line || a.line == b.line && a.col < b.col }

// selection is text the user is dragging over, or dragged over, in an
// agent's screen. Squeeze holds the mouse, so the terminal cannot select;
// for an agent that does not take the mouse itself, squeeze does.
type selection struct {
	r        *racer
	from, to spot
	dragging bool
}

// ordered is the selection from its first cell to its last.
func (s *selection) ordered() (spot, spot) {
	if s.to.before(s.from) {
		return s.to, s.from
	}
	return s.from, s.to
}

func (s *selection) empty() bool { return s.r == nil || s.from == s.to }

// has reports whether the cell at line, col is selected.
func (s *selection) has(line, col int) bool {
	if s.empty() {
		return false
	}
	a, b := s.ordered()
	return !(spot{line, col}).before(a) && !b.before(spot{line, col})
}

// cellAt is a cell of the racer's output by line from the top of its
// scrollback.
func (r *racer) cellAt(line, col int) *uv.Cell {
	if sb := r.emu.ScrollbackLen(); line < sb {
		return r.emu.ScrollbackCellAt(col, line)
	} else {
		return r.emu.CellAt(col, line-sb)
	}
}

// spotAt is the cell under (x, y) of the racer's pane as it is drawn.
func (r *racer) spotAt(x, y int) spot {
	sb := r.emu.ScrollbackLen()
	return spot{line: sb - min(r.scroll, sb) + y, col: x}
}

// text is what the selection covers, a line per row, without the blanks
// that pad the rows out.
func (s *selection) text() string {
	if s.empty() {
		return ""
	}
	a, b := s.ordered()
	w := s.r.emu.Width()
	var lines []string
	for line := a.line; line <= b.line; line++ {
		x0, x1 := 0, w-1
		if line == a.line {
			x0 = a.col
		}
		if line == b.line {
			x1 = b.col
		}
		var sb strings.Builder
		for x := x0; x <= x1; x++ {
			c := s.r.cellAt(line, x)
			switch {
			case c == nil:
				sb.WriteByte(' ')
			case c.Content != "":
				sb.WriteString(c.Content)
			case c.Width != 0:
				sb.WriteByte(' ')
			} // a wide character's second cell adds nothing
		}
		lines = append(lines, strings.TrimRight(sb.String(), " "))
	}
	return strings.Join(lines, "\n")
}
