package widgets

import (
	"strconv"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// LogSource supplies the lines a LogView shows. The view asks only for the
// lines on screen, so a source can hold millions. For the view to draw
// without allocating, Line must return a string it already has.
type LogSource interface {
	Len() int
	Line(i int) string
}

// LevelSource is an optional extension of LogSource: a source that knows each
// line's severity gets it coloured.
type LevelSource interface {
	Level(i int) LogLevel
}

// LineNumberSource is an optional extension of LogSource for a source that
// shows a subset of a larger log, such as a filtered view: the gutter then
// shows each line's number in the full log, not its position in the subset.
type LineNumberSource interface {
	LineNumber(i int) int
}

// LogLevel is a line's severity.
type LogLevel uint8

const (
	LevelUnknown LogLevel = iota
	LevelTrace
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the level's lower-case name, "" for LevelUnknown.
func (l LogLevel) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	}
	return ""
}

// LogViewState is a LogView's scroll position, selection and follow mode.
type LogViewState struct {
	// Offset is the first line on screen.
	Offset int
	// Selected is the selected line, or -1.
	Selected int
	// Follow keeps the last line in view as lines arrive, like tail -f.
	// Scrolling up turns it off; End turns it back on.
	Follow bool
	// HOffset scrolls every line left by this many columns.
	HOffset int

	height    int // rows drawn last frame, for paging
	followOff int // the Offset Follow set last frame; a different one means the wheel moved it
	lastSel   int // Selected as of last frame; a different one means a click selected a line
	nodes     []accessibility.AccessibilityNode
}

// NewLogViewState returns a state that follows new lines and selects nothing.
func NewLogViewState() *LogViewState {
	return &LogViewState{Selected: -1, Follow: true, lastSel: -1}
}

// HandleKey moves the selection or the view: arrows and j/k by a line, Page
// Up/Down by a screen, Home/g to the top, End/G to the bottom (which also
// resumes following), Left/Right scroll sideways. total is the number of
// lines in the source. It reports whether anything changed.
func (s *LogViewState) HandleKey(key driver.KeyEvent, total int) bool {
	page := s.height - 1
	if page < 1 {
		page = 1
	}
	move := func(delta int) bool {
		if total == 0 {
			return false
		}
		from := s.Selected
		if from < 0 {
			// Nothing selected yet: start from the edge of what is on screen.
			from = s.Offset
			if delta < 0 {
				from = s.Offset + s.height
			}
		}
		to := clampInt(from+delta, 0, total-1)
		s.Selected = to
		s.Follow = false
		s.reveal(to)
		return true
	}
	switch {
	case key.Type == driver.KeyArrowUp || (key.Type == driver.KeyRune && key.Ch == 'k'):
		return move(-1)
	case key.Type == driver.KeyArrowDown || (key.Type == driver.KeyRune && key.Ch == 'j'):
		return move(1)
	case key.Type == driver.KeyPageUp:
		return move(-page)
	case key.Type == driver.KeyPageDown:
		return move(page)
	case key.Type == driver.KeyHome || (key.Type == driver.KeyRune && key.Ch == 'g'):
		if total == 0 {
			return false
		}
		s.Selected, s.Offset, s.Follow = 0, 0, false
		return true
	case key.Type == driver.KeyEnd || (key.Type == driver.KeyRune && key.Ch == 'G'):
		s.Selected, s.Follow = -1, true
		return true
	case key.Type == driver.KeyArrowLeft:
		if s.HOffset == 0 {
			return false
		}
		s.HOffset = max(0, s.HOffset-8)
		return true
	case key.Type == driver.KeyArrowRight:
		s.HOffset += 8
		return true
	}
	return false
}

// Select selects line i and scrolls it into view, as a search hit would.
func (s *LogViewState) Select(i int) {
	s.Selected = i
	s.Follow = false
	s.reveal(i)
}

func (s *LogViewState) reveal(i int) {
	if i < s.Offset {
		s.Offset = i
	} else if s.height > 0 && i >= s.Offset+s.height {
		s.Offset = i - s.height + 1
	}
}

// LogView shows a long, growing list of log lines: scrolled virtually, so only
// the lines on screen are touched; coloured by level when the source knows
// levels; with an optional line-number gutter and highlighted search text.
//
// Drawing does not allocate as long as the source's Line does not.
type LogView struct {
	ID     string
	Label  string // name in the semantic tree; defaults to "Log"
	Source LogSource
	State  *LogViewState

	// LineNumbers draws a gutter with 1-based line numbers.
	LineNumbers bool
	// Highlight marks every case-insensitive occurrence of this text.
	Highlight string

	Style          cell.Style
	SelectedStyle  cell.Style // defaults to reverse video
	HighlightStyle cell.Style // defaults to black on yellow
	GutterStyle    cell.Style // defaults to dim
	// LevelStyles overrides the colour of each level; zero entries keep the
	// default.
	LevelStyles [LevelFatal + 1]cell.Style
}

var defaultLevelStyles = [LevelFatal + 1]cell.Style{
	LevelTrace: {Fg: cell.NewColorRGB(110, 110, 120)},
	LevelDebug: {Fg: cell.NewColorRGB(140, 150, 165)},
	LevelInfo:  {},
	LevelWarn:  {Fg: cell.NewColorRGB(240, 190, 60)},
	LevelError: {Fg: cell.NewColorRGB(240, 90, 80)},
	LevelFatal: {Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(170, 30, 30), Modifier: cell.ModifierBold},
}

// Draw draws the visible lines.
func (lv *LogView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	s := lv.State
	if s == nil {
		s = &LogViewState{Selected: -1, Follow: true}
	}
	total := 0
	if lv.Source != nil {
		total = lv.Source.Len()
	}
	height := int(area.Height)
	s.height = height
	maxOffset := max(0, total-height)

	// The mouse wheel moves Offset directly. If it moved away from where
	// Follow put it last frame, the reader scrolled: stop following. A click
	// sets Selected directly; a line the reader picked must not scroll away
	// under them either.
	if s.Follow && (s.Offset != s.followOff || (s.Selected >= 0 && s.Selected != s.lastSel)) {
		s.Follow = false
	}
	if s.Follow {
		s.Offset = maxOffset
		s.followOff = maxOffset
	}
	s.Offset = clampInt(s.Offset, 0, maxOffset)
	if s.Selected >= total {
		s.Selected = total - 1
	}
	s.lastSel = s.Selected

	if lv.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(lv.ID)
	}
	if ctx.RegisterScroll != nil && lv.State != nil {
		ctx.RegisterScroll(area, &lv.State.Offset, maxOffset)
	}

	base := ctx.Style.Merge(lv.Style)
	selStyle := lv.SelectedStyle
	if selStyle == (cell.Style{}) {
		selStyle = cell.Style{Modifier: cell.ModifierReverse}
	}
	hlStyle := lv.HighlightStyle
	if hlStyle == (cell.Style{}) {
		hlStyle = cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: cell.NewColorRGB(250, 210, 60)}
	}
	gutterStyle := lv.GutterStyle
	if gutterStyle == (cell.Style{}) {
		gutterStyle = cell.Style{Fg: cell.NewColorRGB(95, 100, 115)}
	}
	levels, _ := lv.Source.(LevelSource)

	numbers, _ := lv.Source.(LineNumberSource)
	lineNumber := func(i int) int {
		if numbers != nil {
			return numbers.LineNumber(i)
		}
		return i + 1
	}
	gutter := uint16(0)
	if lv.LineNumbers {
		widest := total
		if total > 0 {
			widest = lineNumber(total - 1) // numbers only grow down the log
		}
		gutter = uint16(digits(widest)) + 1
		if gutter >= area.Width {
			gutter = 0
		}
	}
	textX := area.X + gutter
	textW := area.Width - gutter

	if lv.State != nil {
		lv.State.nodes = lv.State.nodes[:0]
	}
	for row := 0; row < height; row++ {
		y := area.Y + uint16(row)
		i := s.Offset + row
		// Clear the row: lines differ in length, and the previous frame's
		// longer line must not show through.
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: base})
		}
		if i >= total {
			continue
		}
		line := lv.Source.Line(i)

		style := base
		if levels != nil {
			lvl := levels.Level(i)
			ls := lv.LevelStyles[lvl]
			if ls == (cell.Style{}) {
				ls = defaultLevelStyles[lvl]
			}
			style = style.Merge(ls)
		}
		selected := i == s.Selected
		if selected {
			style = style.Merge(selStyle)
			for x := textX; x < area.X+area.Width; x++ {
				buf.SetCell(x, y, cell.Cell{Content: ' ', Style: style})
			}
		}

		if gutter > 0 {
			var num [20]byte
			n := strconv.AppendInt(num[:0], int64(lineNumber(i)), 10)
			x := textX - 1 - uint16(len(n))
			for _, ch := range n {
				buf.SetCell(x, y, cell.Cell{Content: rune(ch), Style: gutterStyle})
				x++
			}
		}

		visible, skipped := skipColumns(line, s.HOffset)
		buf.SetStringWithin(textX, y, visible, style, textW)
		if lv.Highlight != "" {
			lv.drawHighlights(buf, textX, y, textW, line, skipped, style.Merge(hlStyle))
		}

		if ctx.RegisterClickAction != nil && lv.State != nil {
			ctx.RegisterClickAction(cell.Rect{X: area.X, Y: y, Width: area.Width, Height: 1},
				cell.ClickAction{Focus: lv.ID, Select: &lv.State.Selected, Index: i})
		}
		if lv.State != nil {
			st := accessibility.NodeState(0)
			if selected {
				st = accessibility.StateSelected
			}
			lv.State.nodes = append(lv.State.nodes, accessibility.AccessibilityNode{
				Role: accessibility.RoleListItem, Label: line, State: st,
				Bounds:   cell.Rect{X: area.X, Y: y, Width: area.Width, Height: 1},
				Position: i + 1, SetSize: total,
			})
		}
	}
}

// drawHighlights redraws each case-insensitive occurrence of lv.Highlight in
// line with style, taking the horizontal scroll into account.
func (lv *LogView) drawHighlights(buf *buffer.Buffer, x, y, width uint16, line string, skipped int, style cell.Style) {
	needle := lv.Highlight
	for from := 0; from < len(line); {
		at := indexFold(line[from:], needle)
		if at < 0 {
			return
		}
		start := from + at
		end := start + len(needle)
		col := cell.StringWidth(line[:start]) - skipped
		from = end
		if col < 0 {
			continue // scrolled out on the left
		}
		if col >= int(width) {
			return
		}
		buf.SetStringWithin(x+uint16(col), y, line[start:end], style, width-uint16(col))
	}
}

// indexFold returns the byte index of the first case-insensitive occurrence
// of needle in s, or -1. It folds ASCII letters only, which is what search
// text typed into a log viewer needs, and it does not allocate.
func indexFold(s, needle string) int {
	n := len(needle)
	if n == 0 || n > len(s) {
		return -1
	}
outer:
	for i := 0; i+n <= len(s); i++ {
		for j := 0; j < n; j++ {
			a, b := s[i+j], needle[j]
			if a == b {
				continue
			}
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				continue outer
			}
		}
		// Never report a match that starts inside a multi-byte character.
		if i > 0 && !utf8.RuneStart(s[i]) {
			continue
		}
		return i
	}
	return -1
}

// skipColumns drops whole grapheme clusters from the start of s until at least
// cols columns are gone, returning the rest and the columns actually dropped.
func skipColumns(s string, cols int) (string, int) {
	if cols <= 0 {
		return s, 0
	}
	prefix, w := cell.Truncate(s, cols)
	return s[len(prefix):], w
}

func digits(n int) int {
	d := 1
	for n >= 10 {
		n /= 10
		d++
	}
	return d
}

// SizeHint fills whatever area it is given.
func (lv *LogView) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// Measure reports that a LogView takes whatever space it is given.
func (lv *LogView) Measure(maxArea cell.Rect) layout.Measure {
	return layout.Measure{MaxWidth: maxArea.Width, MaxHeight: maxArea.Height, Overflow: layout.OverflowClip}
}

// AccessibilityNode describes the view as a list whose children are the lines
// on screen, so a screen reader or an agent can read the visible log and
// select a line by its text. The value is the selected line.
func (lv *LogView) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	label := lv.Label
	if label == "" {
		label = "Log"
	}
	total := 0
	if lv.Source != nil {
		total = lv.Source.Len()
	}
	value, position := "", 0
	var rows []accessibility.AccessibilityNode
	if lv.State != nil {
		rows = lv.State.nodes
		if sel := lv.State.Selected; sel >= 0 && sel < total {
			state |= accessibility.StateSelected
			value = lv.Source.Line(sel)
			position = sel + 1
		}
		if lv.State.Follow {
			state |= accessibility.StateBusy // live: new lines keep arriving
		}
	}
	return accessibility.AccessibilityNode{
		ID: lv.ID, Role: accessibility.RoleList, Label: label, Value: value, State: state,
		Bounds: bounds, Position: position, SetSize: total, Children: rows,
	}
}
