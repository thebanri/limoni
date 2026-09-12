package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// SelectState stores the selected option and whether the option list is open.
type SelectState struct {
	Selected int
	Hovered  int
	Open     bool
}

func NewSelectState() *SelectState { return &SelectState{Selected: 0, Hovered: -1} }

// HandleKey handles keyboard navigation for a Select.
func (s *SelectState) HandleKey(ev driver.KeyEvent, optionCount int) bool {
	if s == nil || optionCount == 0 {
		return false
	}
	switch ev.Type {
	case driver.KeyArrowUp, driver.KeyArrowLeft:
		if s.Selected > 0 {
			s.Selected--
		} else {
			s.Selected = optionCount - 1
		}
		return true
	case driver.KeyArrowDown, driver.KeyArrowRight:
		if s.Selected < optionCount-1 {
			s.Selected++
		} else {
			s.Selected = 0
		}
		return true
	case driver.KeyEnter, driver.KeySpace:
		s.Open = !s.Open
		return true
	case driver.KeyEsc:
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
	DisableScroll bool // Fare tekerleğiyle seçenek değiştirmeyi kapatır
	DisableFocus  bool // Tıklamayla odak almayı kapatır
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
	if ctx.IsFocused(s.ID) {
		fieldStyle = fieldStyle.Merge(s.FocusedStyle)
	}
	for x := uint16(0); x < ctx.Area.Width; x++ {
		px := ctx.Area.X + x
		py := ctx.Area.Y
		if c := buf.Get(px, py); c != nil {
			c.Content = ' '
			c.Style = c.Style.Merge(fieldStyle)
		} else {
			buf.SetCell(px, py, cell.Cell{Content: ' ', Style: fieldStyle})
		}
	}
	label := s.Options[s.State.Selected]
	indicator := " ▾"
	if s.State.Open {
		indicator = " ▴"
	}
	if ctx.Area.Width > 2 {
		buf.SetString(ctx.Area.X+1, ctx.Area.Y, clipString(label+indicator, int(ctx.Area.Width)-2), fieldStyle)
	}

	// Fare tıklama ve tekerlek işleyicisi
	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(ctx.Area, func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseLeft && !ev.Drag {
				if !s.DisableFocus && ctx.SetFocus != nil {
					ctx.SetFocus(s.ID)
				}
				s.State.Open = !s.State.Open
				return
			}
			if !s.DisableScroll {
				if ev.Button == driver.MouseScrollUp {
					if s.State.Selected > 0 {
						s.State.Selected--
					} else {
						s.State.Selected = len(s.Options) - 1
					}
					if s.OnChange != nil {
						s.OnChange(s.State.Selected, s.Options[s.State.Selected])
					}
					if !s.DisableFocus && ctx.SetFocus != nil {
						ctx.SetFocus(s.ID)
					}
				} else if ev.Button == driver.MouseScrollDown {
					if s.State.Selected < len(s.Options)-1 {
						s.State.Selected++
					} else {
						s.State.Selected = 0
					}
					if s.OnChange != nil {
						s.OnChange(s.State.Selected, s.Options[s.State.Selected])
					}
					if !s.DisableFocus && ctx.SetFocus != nil {
						ctx.SetFocus(s.ID)
					}
				}
			}
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
			ctx.RegisterMouse(cell.NewRect(ctx.Area.X, y, ctx.Area.Width, 1), func(ev driver.MouseEvent) {
				if ev.Button == driver.MouseNone {
					s.State.Hovered = index
					return
				}
				if ev.Button == driver.MouseLeft && !ev.Drag {
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

// AccessibilityNode returns the semantic node description for Select.
func (s Select) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}

	selected, position, value := -1, 0, ""
	if s.State != nil {
		if s.State.Open {
			state |= accessibility.StateExpanded
		}
		selected = s.State.Selected
	}
	if selected >= 0 && selected < len(s.Options) {
		state |= accessibility.StateSelected
		value = s.Options[selected]
		position = selected + 1
	}

	return accessibility.AccessibilityNode{
		ID:       s.ID,
		Role:     accessibility.RoleList,
		Label:    "Select",
		Value:    value,
		State:    state,
		Bounds:   bounds,
		Position: position,
		SetSize:  len(s.Options),
	}
}
