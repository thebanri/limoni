package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// TextAreaState stores multiline text and a rune cursor.
type TextAreaState struct {
	Text   []rune
	Cursor int
}

func NewTextAreaState() *TextAreaState { return &TextAreaState{Text: make([]rune, 0, 128)} }
func (s *TextAreaState) Value() string {
	if s == nil {
		return ""
	}
	return string(s.Text)
}
func (s *TextAreaState) SetValue(value string) { s.Text = []rune(value); s.Cursor = len(s.Text) }

func (s *TextAreaState) HandleKey(ev backend.KeyEvent) bool {
	if s == nil {
		return false
	}
	switch ev.Type {
	case backend.KeyRune:
		s.Text = append(s.Text, 0)
		copy(s.Text[s.Cursor+1:], s.Text[s.Cursor:])
		s.Text[s.Cursor] = ev.Ch
		s.Cursor++
		return true
	case backend.KeyEnter:
		s.Text = append(s.Text, 0)
		copy(s.Text[s.Cursor+1:], s.Text[s.Cursor:])
		s.Text[s.Cursor] = '\n'
		s.Cursor++
		return true
	case backend.KeyBackspace:
		if s.Cursor == 0 {
			return false
		}
		s.Text = append(s.Text[:s.Cursor-1], s.Text[s.Cursor:]...)
		s.Cursor--
		return true
	case backend.KeyDelete:
		if s.Cursor >= len(s.Text) {
			return false
		}
		s.Text = append(s.Text[:s.Cursor], s.Text[s.Cursor+1:]...)
		return true
	case backend.KeyArrowLeft:
		if s.Cursor > 0 {
			s.Cursor--
			return true
		}
	case backend.KeyArrowRight:
		if s.Cursor < len(s.Text) {
			s.Cursor++
			return true
		}
	case backend.KeyHome:
		s.Cursor = lineStart(s.Text, s.Cursor)
		return true
	case backend.KeyEnd:
		s.Cursor = lineEnd(s.Text, s.Cursor)
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
type TextArea struct {
	ID           string
	State        *TextAreaState
	Style        cell.Style
	FocusedStyle cell.Style
}

func (a TextArea) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if a.State == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(a.ID)
	}
	if ctx.RegisterClick != nil {
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
			buf.SetCell(ctx.Area.X+x, ctx.Area.Y+y, cell.Cell{Content: ' ', Style: style})
		}
	}
	lines := strings.Split(string(a.State.Text), "\n")
	for row, line := range lines {
		if row >= int(ctx.Area.Height) {
			break
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(row), clipString(line, int(ctx.Area.Width)), style)
	}
}
func (a TextArea) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }
