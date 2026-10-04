package widgets

import (
	"slices"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// TextAreaState stores multiline text and a rune cursor.
type TextAreaState struct {
	Text   []rune
	Cursor int

	// Top is the first visible row of the wrapped text. Draw moves it to
	// keep the cursor in view.
	Top int

	// text is Text as UTF-8, rebuilt only when Text changes.
	text  string
	built []rune
}

func NewTextAreaState() *TextAreaState { return &TextAreaState{Text: make([]rune, 0, 128)} }

// Value returns the text as a string.
func (s *TextAreaState) Value() string {
	if s == nil {
		return ""
	}
	return s.str()
}

// str is Text as a string, converted only when Text has changed.
func (s *TextAreaState) str() string {
	if !slices.Equal(s.built, s.Text) {
		s.text = string(s.Text)
		s.built = append(s.built[:0], s.Text...)
	}
	return s.text
}
func (s *TextAreaState) SetValue(value string) { s.Text = []rune(value); s.Cursor = len(s.Text) }

func (s *TextAreaState) HandleKey(ev driver.KeyEvent) bool {
	if s == nil {
		return false
	}
	switch ev.Type {
	case driver.KeyRune:
		if ev.Ctrl || ev.Alt {
			return false // a command, not text
		}
		s.Text = append(s.Text, 0)
		copy(s.Text[s.Cursor+1:], s.Text[s.Cursor:])
		s.Text[s.Cursor] = ev.Ch
		s.Cursor++
		return true
	case driver.KeyEnter:
		s.Text = append(s.Text, 0)
		copy(s.Text[s.Cursor+1:], s.Text[s.Cursor:])
		s.Text[s.Cursor] = '\n'
		s.Cursor++
		return true
	// Deleting and moving go by grapheme cluster, as in TextInput: one
	// Backspace removes a whole emoji sequence or accented letter.
	case driver.KeyBackspace:
		if s.Cursor == 0 {
			return false
		}
		start, _ := clusterBounds(string(s.Text), s.Cursor)
		s.Text = append(s.Text[:start], s.Text[s.Cursor:]...)
		s.Cursor = start
		return true
	case driver.KeyDelete:
		if s.Cursor >= len(s.Text) {
			return false
		}
		_, end := clusterBounds(string(s.Text), s.Cursor)
		s.Text = append(s.Text[:s.Cursor], s.Text[end:]...)
		return true
	case driver.KeyArrowLeft:
		if s.Cursor > 0 {
			s.Cursor, _ = clusterBounds(string(s.Text), s.Cursor)
			return true
		}
	case driver.KeyArrowRight:
		if s.Cursor < len(s.Text) {
			_, s.Cursor = clusterBounds(string(s.Text), s.Cursor)
			return true
		}
	case driver.KeyHome:
		s.Cursor = lineStart(s.Text, s.Cursor)
		return true
	case driver.KeyEnd:
		s.Cursor = lineEnd(s.Text, s.Cursor)
		return true
	case driver.KeyArrowUp:
		start := lineStart(s.Text, s.Cursor)
		if start == 0 {
			return false
		}
		prev := lineStart(s.Text, start-1)
		s.Cursor = min(prev+(s.Cursor-start), start-1)
		return true
	case driver.KeyArrowDown:
		end := lineEnd(s.Text, s.Cursor)
		if end >= len(s.Text) {
			return false
		}
		column := s.Cursor - lineStart(s.Text, s.Cursor)
		s.Cursor = min(end+1+column, lineEnd(s.Text, end+1))
		return true
	}
	return false
}

func lineStart(text []rune, cursor int) int {
	for cursor > 0 && text[cursor-1] != '\n' {
		cursor--
	}
	return cursor
}
func lineEnd(text []rune, cursor int) int {
	for cursor < len(text) && text[cursor] != '\n' {
		cursor++
	}
	return cursor
}

// TextArea is a multiline focusable text editor.
//
// Long lines wrap at the width, by grapheme cluster; the view follows the
// cursor, which is drawn while the area has the focus. Drawing does not
// allocate.
type TextArea struct {
	ID           string
	State        *TextAreaState
	Style        cell.Style
	FocusedStyle cell.Style
	// Placeholder is shown, dimmed, while the text is empty.
	Placeholder string
	// Focused draws the area as focused whatever the focus manager says,
	// for applications that track focus themselves.
	Focused bool
}

// walkWrapped calls visit for each grapheme cluster of text laid out in
// width columns: its row and column, and the rune index it starts at. A
// newline ends a row and is not visited. It returns the row and column the
// cursor (a rune index) falls on, and the number of rows.
func walkWrapped(text string, width, cursor int, visit func(row, col int, cluster string, w int)) (cursorRow, cursorCol, rows int) {
	row, col, runeIdx := 0, 0, 0
	cursorRow, cursorCol = -1, 0
	for rest := text; rest != ""; {
		cluster, w, next := cell.NextCluster(rest)
		n := utf8.RuneCountInString(cluster)
		if cluster == "\n" {
			if cursorRow < 0 && runeIdx >= cursor {
				cursorRow, cursorCol = row, col
			}
			row, col = row+1, 0
		} else {
			if col+w > width && col > 0 {
				row, col = row+1, 0
			}
			if cursorRow < 0 && runeIdx+n > cursor {
				cursorRow, cursorCol = row, col
			}
			if visit != nil {
				visit(row, col, cluster, w)
			}
			col += w
		}
		runeIdx += n
		rest = next
	}
	if cursorRow < 0 {
		cursorRow, cursorCol = row, col
		if col >= width {
			cursorRow, cursorCol = row+1, 0
		}
	}
	return cursorRow, cursorCol, max(row, cursorRow) + 1
}

func (a TextArea) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if a.State == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(a.ID)
	}
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && a.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: a.ID})
	} else if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(a.ID)
			}
		})
	}
	style := ctx.Style.Merge(a.Style)
	if ctx.IsFocused(a.ID) {
		style = style.Merge(a.FocusedStyle)
	}
	for y := uint16(0); y < ctx.Area.Height; y++ {
		for x := uint16(0); x < ctx.Area.Width; x++ {
			px := ctx.Area.X + x
			py := ctx.Area.Y + y
			if c := buf.Get(px, py); c != nil {
				c.Content = ' '
				c.Style = c.Style.Merge(style)
			} else {
				buf.SetCell(px, py, cell.Cell{Content: ' ', Style: style})
			}
		}
	}
	focused := a.Focused || ctx.IsFocused(a.ID)
	st := a.State
	width, height := int(ctx.Area.Width), int(ctx.Area.Height)
	if len(st.Text) == 0 && a.Placeholder != "" {
		setClipped(buf, ctx.Area.X, ctx.Area.Y, a.Placeholder, style.Merge(cell.Style{Modifier: cell.ModifierDim}), width)
	}
	text := st.str()
	st.Cursor = clampInt(st.Cursor, 0, len(st.Text))
	cursorRow, cursorCol, _ := walkWrapped(text, width, st.Cursor, nil)
	if cursorRow < st.Top {
		st.Top = cursorRow
	} else if cursorRow >= st.Top+height {
		st.Top = cursorRow - height + 1
	}
	top := st.Top
	walkWrapped(text, width, st.Cursor, func(row, col int, cluster string, w int) {
		if row < top || row >= top+height || col+w > width {
			return
		}
		x, y := ctx.Area.X+uint16(col), ctx.Area.Y+uint16(row-top)
		buf.SetCell(x, y, cell.Cell{Content: cell.ClusterContent(cluster, w), Style: style})
		if w == 2 {
			buf.SetCell(x+1, y, cell.Cell{Content: cell.RuneContinuation, Style: style})
		}
	})
	if focused && cursorCol < width {
		paintCursor(buf, ctx.Area.X+uint16(cursorCol), ctx.Area.Y+uint16(cursorRow-top))
	}
}
func (a TextArea) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }

// Measure provides explicit size negotiation for TextArea.
func (a TextArea) Measure(maxArea cell.Rect) layout.Measure {
	w, h := a.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowScroll,
	}
}

// AccessibilityNode returns the semantic node description for TextArea.
func (a TextArea) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{
		ID:     a.ID,
		Role:   accessibility.RoleInput,
		Label:  "Text Area",
		Value:  a.State.Value(),
		State:  state,
		Bounds: bounds,
	}
}
