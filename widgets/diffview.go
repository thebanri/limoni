package widgets

import (
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// diffLine is one row of a diff: a line kept, added or removed, or the gap
// between two hunks.
type diffLine struct {
	op   byte // ' ', '+', '-', or '~' for a gap
	num  int  // line number in the new source, or the old one for '-'
	code codeLine
}

// DiffViewState holds a line diff between two versions of a source, with
// both sides highlighted. SetSources does the work; drawing only reads it.
type DiffViewState struct {
	// Offset is the first row shown; HOffset the first column.
	Offset, HOffset int

	lines          []diffLine
	added, removed int
	maxNum         int
}

// NewDiffViewState diffs oldSrc against newSrc; see SetSources.
func NewDiffViewState(oldSrc, newSrc string, lang *Language, context int) *DiffViewState {
	s := &DiffViewState{}
	s.SetSources(oldSrc, newSrc, lang, context)
	return s
}

// Stats is the number of lines added and removed.
func (s *DiffViewState) Stats() (added, removed int) { return s.added, s.removed }

// Rows is the number of rows the diff takes, gaps between hunks included.
func (s *DiffViewState) Rows() int { return len(s.lines) }

// SetSources diffs oldSrc against newSrc line by line and highlights both as
// lang (nil for plain text). Only changed lines and context lines around
// each change are kept, with a gap row where lines were left out; a
// negative context keeps every line.
func (s *DiffViewState) SetSources(oldSrc, newSrc string, lang *Language, context int) {
	oldLines, newLines := highlightLines(oldSrc, lang), highlightLines(newSrc, lang)
	ops := diffOps(oldLines, newLines)

	s.lines, s.added, s.removed, s.maxNum = s.lines[:0], 0, 0, max(len(oldLines), len(newLines))
	// keep marks the rows within context of a change.
	keep := make([]bool, len(ops))
	for i, op := range ops {
		if op == ' ' {
			continue
		}
		for j := max(0, i-context); j < len(ops) && (context < 0 || j <= i+context); j++ {
			keep[j] = true
		}
	}
	o, n, skipped := 0, 0, false
	for i, op := range ops {
		line := diffLine{op: op}
		switch op {
		case ' ':
			line.num, line.code = n+1, newLines[n]
			o++
			n++
		case '+':
			line.num, line.code = n+1, newLines[n]
			n++
			s.added++
		case '-':
			line.num, line.code = o+1, oldLines[o]
			o++
			s.removed++
		}
		if context >= 0 && !keep[i] {
			skipped = true
			continue
		}
		if skipped && len(s.lines) > 0 {
			s.lines = append(s.lines, diffLine{op: '~'})
		}
		skipped = false
		s.lines = append(s.lines, line)
	}
}

// highlightLines splits src into lines and lexes them in order, so a
// comment or string that spans lines is coloured on each.
func highlightLines(src string, lang *Language) []codeLine {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if src == "" {
		return nil
	}
	raw := strings.Split(strings.TrimSuffix(src, "\n"), "\n")
	lines := make([]codeLine, len(raw))
	var open string
	openKind, openDepth := TokenPlain, 0
	for i, r := range raw {
		lines[i].text = expandTabs(r, 4)
		if lang != nil {
			lines[i].spans, open, openKind, openDepth = lexLine(lines[i].text, lang, open, openKind, openDepth)
		}
	}
	return lines
}

// maxDiffWork bounds the work diffOps spends searching for a minimal script,
// in diagonal steps and line comparisons. A small edit costs a few thousand;
// two unrelated files of 100,000 lines would need ~10^10, and past the budget
// the part still unsolved is shown as removed and then added.
const maxDiffWork = 1 << 26

// diffOps is the edit script from a to b, one op per row: ' ' for a line
// kept, '-' removed from a, '+' added from b. It is a minimal script,
// found with Myers' O((N+M)·D) algorithm in its linear-space form (middle
// snake, then recursion on both halves), after trimming the common prefix
// and suffix. Within a change, removals come before additions.
func diffOps(a, b []codeLine) []byte {
	ops := make([]byte, 0, len(a)+len(b))
	k := len(a) + len(b) + 2
	m := myers{vf: make([]int, 2*k+2), vb: make([]int, 2*k+2), off: k, work: maxDiffWork}
	ops = m.diff(a, b, ops)
	// Splitting at snakes can interleave '-' and '+' inside one change; sort
	// each run (the counts, and so the minimality, are unchanged).
	for i := 0; i < len(ops); {
		if ops[i] == ' ' {
			i++
			continue
		}
		j, minus := i, 0
		for j < len(ops) && ops[j] != ' ' {
			if ops[j] == '-' {
				minus++
			}
			j++
		}
		for p := i; p < j; p++ {
			if p-i < minus {
				ops[p] = '-'
			} else {
				ops[p] = '+'
			}
		}
		i = j
	}
	return ops
}

// myers holds the furthest-reaching tables shared by every recursion level
// and the remaining work budget.
type myers struct {
	vf, vb []int
	off    int
	work   int
}

func (m *myers) diff(a, b []codeLine, ops []byte) []byte {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre].text == b[pre].text {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf].text == b[len(b)-1-suf].text {
		suf++
	}
	for range pre {
		ops = append(ops, ' ')
	}
	a, b = a[pre:len(a)-suf], b[pre:len(b)-suf]
	if len(a) == 0 || len(b) == 0 {
		ops = appendRun(ops, '-', len(a))
		ops = appendRun(ops, '+', len(b))
	} else if x, y, u, v, ok := m.middleSnake(a, b); ok {
		ops = m.diff(a[:x], b[:y], ops)
		ops = appendRun(ops, ' ', u-x)
		ops = m.diff(a[u:], b[v:], ops)
	} else {
		ops = appendRun(ops, '-', len(a))
		ops = appendRun(ops, '+', len(b))
	}
	return appendRun(ops, ' ', suf)
}

func appendRun(ops []byte, op byte, n int) []byte {
	for range n {
		ops = append(ops, op)
	}
	return ops
}

// middleSnake searches forwards from the start and backwards from the end
// until the two meet, and returns the snake (x,y)→(u,v) where they did. ok is
// false when the work budget ran out or the edit is a single line (which the
// trimmed, non-empty inputs of diff rule out).
func (m *myers) middleSnake(a, b []codeLine) (x, y, u, v int, ok bool) {
	n, mm := len(a), len(b)
	delta := n - mm
	odd := delta&1 != 0
	off, vf, vb := m.off, m.vf, m.vb
	vf[off+1], vb[off+1] = 0, 0
	for d := 0; d <= (n+mm+1)/2; d++ {
		m.work -= 2 * (d + 1)
		if m.work < 0 {
			return 0, 0, 0, 0, false
		}
		for k := -d; k <= d; k += 2 {
			var px int
			if k == -d || (k != d && vf[off+k-1] < vf[off+k+1]) {
				px = vf[off+k+1]
			} else {
				px = vf[off+k-1] + 1
			}
			py := px - k
			sx, sy := px, py
			for px < n && py < mm && a[px].text == b[py].text {
				px++
				py++
			}
			m.work -= px - sx
			vf[off+k] = px
			if c := delta - k; odd && c >= -(d-1) && c <= d-1 && px+vb[off+c] >= n {
				return sx, sy, px, py, 2*d-1 > 1
			}
		}
		for k := -d; k <= d; k += 2 {
			var px int
			if k == -d || (k != d && vb[off+k-1] < vb[off+k+1]) {
				px = vb[off+k+1]
			} else {
				px = vb[off+k-1] + 1
			}
			py := px - k
			sx, sy := px, py
			for px < n && py < mm && a[n-1-px].text == b[mm-1-py].text {
				px++
				py++
			}
			m.work -= px - sx
			vb[off+k] = px
			if c := delta - k; !odd && c >= -d && c <= d && px+vf[off+c] >= n {
				return n - px, mm - py, n - sx, mm - sy, 2*d > 1
			}
		}
	}
	return 0, 0, 0, 0, false
}

// DiffView shows a DiffViewState: a gutter of line numbers, a + or - for
// each line added or removed, those lines on a green or red background, and
// the code highlighted as in CodeView.
type DiffView struct {
	ID    string
	State *DiffViewState
	// Theme styles each token kind; nil is DefaultCodeTheme.
	Theme *CodeTheme
	// HideLineNumbers drops the gutter.
	HideLineNumbers bool

	Style       cell.Style
	GutterStyle cell.Style
	// AddedStyle and RemovedStyle are merged over added and removed lines;
	// zero is a dark green and a dark red background.
	AddedStyle, RemovedStyle cell.Style
}

// Draw renders the visible rows. It does not allocate.
func (dv DiffView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	s := dv.State
	if area.Width == 0 || area.Height == 0 || s == nil {
		return
	}
	if dv.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(dv.ID)
	}
	theme := dv.Theme
	if theme == nil {
		theme = &DefaultCodeTheme
	}
	base := ctx.Style.Merge(dv.Style)
	gutterStyle := base.Merge(dv.GutterStyle)
	if dv.GutterStyle == (cell.Style{}) {
		gutterStyle = base.Merge(cell.Style{Fg: cell.NewColorRGB(110, 110, 110)})
	}
	added, removed := dv.AddedStyle, dv.RemovedStyle
	if added == (cell.Style{}) {
		added = cell.Style{Bg: cell.NewColorRGB(28, 56, 34)}
	}
	if removed == (cell.Style{}) {
		removed = cell.Style{Bg: cell.NewColorRGB(66, 30, 32)}
	}
	s.Offset = max(0, min(s.Offset, len(s.lines)-int(area.Height)))

	// The gutter: the number, a space, the sign, a space.
	gutter := 2
	if !dv.HideLineNumbers {
		gutter += digits(s.maxNum) + 1
	}
	if gutter >= int(area.Width) {
		gutter = 0
	}
	textX := area.X + uint16(gutter)
	textW := int(area.Width) - gutter
	code := CodeView{}

	for row := 0; row < int(area.Height); row++ {
		y := area.Y + uint16(row)
		i := s.Offset + row
		lineStyle := base
		if i < len(s.lines) {
			switch s.lines[i].op {
			case '+':
				lineStyle = base.Merge(added)
			case '-':
				lineStyle = base.Merge(removed)
			}
		}
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: lineStyle})
		}
		if i >= len(s.lines) {
			continue
		}
		line := s.lines[i]
		if line.op == '~' {
			buf.SetStringWithin(area.X, y, "⋯", gutterStyle, area.Width)
			continue
		}
		if gutter > 0 {
			gs := gutterStyle.Merge(cell.Style{Bg: lineStyle.Bg})
			if !dv.HideLineNumbers {
				var num [20]byte
				n := strconv.AppendInt(num[:0], int64(line.num), 10)
				x := area.X + uint16(gutter-3-len(n))
				for _, ch := range n {
					buf.SetCell(x, y, cell.Cell{Content: rune(ch), Style: gs})
					x++
				}
			}
			if line.op != ' ' {
				buf.SetCell(textX-2, y, cell.Cell{Content: rune(line.op), Style: lineStyle})
			}
		}
		code.drawLine(buf, textX, y, textW, line.code, s.HOffset, theme, lineStyle)
	}
}

// SizeHint is as tall as the diff and as wide as offered.
func (dv DiffView) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	if dv.State == nil {
		return 0, 0
	}
	return maxArea.Width, uint16(min(len(dv.State.lines), int(maxArea.Height)))
}

// AccessibilityNode is a read-only text area labelled with the counts; its
// value is the first row shown.
func (dv DiffView) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.Node {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	node := accessibility.Node{ID: dv.ID, Role: accessibility.RoleGeneric, Label: "Diff", State: state, Bounds: bounds}
	if s := dv.State; s != nil && len(s.lines) > 0 {
		i := max(0, min(s.Offset, len(s.lines)-1))
		node.Value = s.lines[i].code.text
		node.Position, node.SetSize = i+1, len(s.lines)
	}
	return node
}
