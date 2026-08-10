package widgets

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// SelectState stores the selected option and whether the option list is open.
type SelectState struct {
	Selected int
	Hovered  int
	Open     bool
}

func NewSelectState() *SelectState { return &SelectState{Selected: 0, Hovered: -1} }

// HandleKey handles keyboard navigation for a Select.
func (s *SelectState) HandleKey(ev backend.KeyEvent, optionCount int) bool {
	if s == nil || optionCount == 0 {
		return false
	}
	switch ev.Type {
	case backend.KeyArrowUp:
		if s.Open && s.Selected > 0 {
			s.Selected--
		}
		return true
	case backend.KeyArrowDown:
		if s.Open && s.Selected < optionCount-1 {
			s.Selected++
		}
		return true
	case backend.KeyEnter, backend.KeySpace:
		s.Open = !s.Open
		return true
	case backend.KeyEsc:
		if s.Open {
			s.Open = false
			return true
		}
	}
	return false
}

// Select is a keyboard- and mouse-interactive dropdown field.
type Select struct {
	ID            string
	Options       []string
	State         *SelectState
	Style         cell.Style
	FocusedStyle  cell.Style
	OptionStyle   cell.Style
	SelectedStyle cell.Style
	HoverStyle    cell.Style
	BorderStyle   cell.Style
	OnChange      func(index int, option string)
}

func (s Select) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if s.ID == "" || s.State == nil || len(s.Options) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	if s.State.Selected < 0 || s.State.Selected >= len(s.Options) {
		s.State.Selected = 0
	}
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(s.ID)
	}

	fieldStyle := ctx.Style.Merge(s.Style)
	if ctx.FocusedID == s.ID {
		fieldStyle = fieldStyle.Merge(s.FocusedStyle)
	}
	for x := uint16(0); x < ctx.Area.Width; x++ {
		buf.SetCell(ctx.Area.X+x, ctx.Area.Y, cell.Cell{Content: ' ', Style: fieldStyle})
	}
	label := s.Options[s.State.Selected]
	if ctx.Area.Width > 2 {
		buf.SetString(ctx.Area.X+1, ctx.Area.Y, clipString(label+" ▾", int(ctx.Area.Width)-2), fieldStyle)
	}
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(s.ID)
			}
			s.State.Open = !s.State.Open
		})
	}
	if !s.State.Open || ctx.Area.Height < 2 {
		return
	}

	optionStyle := ctx.Style.Merge(s.OptionStyle)
	selectedStyle := ctx.Style.Merge(s.SelectedStyle)

	maxVisible := int(ctx.Area.Height) - 1
	if maxVisible <= 0 {
		return
	}

	startIdx := 0
	if s.State.Selected >= maxVisible {
		startIdx = s.State.Selected - maxVisible + 1
	}
	if startIdx+maxVisible > len(s.Options) {
		startIdx = len(s.Options) - maxVisible
	}
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(s.Options) {
		endIdx = len(s.Options)
	}

	for i := startIdx; i < endIdx; i++ {
		option := s.Options[i]
		y := ctx.Area.Y + 1 + uint16(i-startIdx)
		style := optionStyle
		if i == s.State.Selected {
			style = selectedStyle
		}
		if i == s.State.Hovered {
			style = ctx.Style.Merge(s.HoverStyle)
			if style == (cell.Style{}) {
				style = selectedStyle
			}
		}

		for x := uint16(0); x < ctx.Area.Width; x++ {
			buf.SetCell(ctx.Area.X+x, y, cell.Cell{Content: ' ', Style: style})
		}
		buf.SetString(ctx.Area.X+1, y, clipString(option, int(ctx.Area.Width)-2), style)

		// Draw scroll indicators on the right edge if there is overflow
		if i == startIdx && startIdx > 0 && ctx.Area.Width > 2 {
			buf.SetCell(ctx.Area.X+ctx.Area.Width-2, y, cell.Cell{Content: '▲', Style: style})
		} else if i == endIdx-1 && endIdx < len(s.Options) && ctx.Area.Width > 2 {
			buf.SetCell(ctx.Area.X+ctx.Area.Width-2, y, cell.Cell{Content: '▼', Style: style})
		}

		if ctx.RegisterMouse != nil {
			index := i
			ctx.RegisterMouse(cell.NewRect(ctx.Area.X, y, ctx.Area.Width, 1), func(ev backend.MouseEvent) {
				if ev.Button == backend.MouseNone {
					s.State.Hovered = index
					return
				}
				if ev.Button == backend.MouseLeft && !ev.Drag {
					s.State.Selected = index
					s.State.Hovered = -1
					s.State.Open = false
					if s.OnChange != nil {
						s.OnChange(index, s.Options[index])
					}
					if ctx.SetFocus != nil {
						ctx.SetFocus(s.ID)
					}
				}
			})
		}
	}
}

func (s Select) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, 1
}
