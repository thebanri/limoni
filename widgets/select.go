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

	// Mouse handlers, built once, and the last frame's settings they read.
	onField                     func(driver.MouseEvent)
	onOption                    []func(driver.MouseEvent)
	options                     []string
	id                          string
	disableScroll, disableFocus bool
	onChange                    func(int, string)
	setFocus                    func(string)
}

// remember records what the handlers need from this frame.
func (st *SelectState) remember(s *Select, setFocus func(string)) {
	st.options, st.id = s.Options, s.ID
	st.disableScroll, st.disableFocus = s.DisableScroll, s.DisableFocus
	st.onChange, st.setFocus = s.OnChange, setFocus
}

func (st *SelectState) changed() {
	if st.onChange != nil && st.Selected < len(st.options) {
		st.onChange(st.Selected, st.options[st.Selected])
	}
}

func (st *SelectState) focus() {
	if !st.disableFocus && st.setFocus != nil {
		st.setFocus(st.id)
	}
}

// fieldHandler opens and closes the list on a click and steps through the
// options with the wheel. Built once per state.
func (st *SelectState) fieldHandler() func(driver.MouseEvent) {
	if st.onField == nil {
		st.onField = func(ev driver.MouseEvent) {
			n := len(st.options)
			switch {
			case ev.Button == driver.MouseLeft && !ev.Drag:
				st.focus()
				st.Open = !st.Open
			case st.disableScroll || n == 0:
			case ev.Button == driver.MouseScrollUp:
				st.Selected = (st.Selected - 1 + n) % n
				st.changed()
				st.focus()
			case ev.Button == driver.MouseScrollDown:
				st.Selected = (st.Selected + 1) % n
				st.changed()
				st.focus()
			}
		}
	}
	return st.onField
}

// optionHandler hovers and picks option i of the open list. Built once per
// state and option index.
func (st *SelectState) optionHandler(i int) func(driver.MouseEvent) {
	for len(st.onOption) <= i {
		index := len(st.onOption)
		st.onOption = append(st.onOption, func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseNone {
				st.Hovered = index
				return
			}
			if ev.Button == driver.MouseLeft && !ev.Drag && index < len(st.options) {
				st.Selected, st.Hovered, st.Open = index, -1, false
				st.changed()
				if st.setFocus != nil {
					st.setFocus(st.id)
				}
			}
		})
	}
	return st.onOption[i]
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
	DisableScroll bool // Turns off changing the option with the mouse wheel
	DisableFocus  bool // Turns off taking the focus on click
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
		setClipped(buf, ctx.Area.X+1, ctx.Area.Y, label+indicator, fieldStyle, int(ctx.Area.Width)-2)
	}

	// Mouse click and wheel handler, built once in the state.
	if ctx.RegisterMouse != nil {
		s.State.remember(&s, ctx.SetFocus)
		ctx.RegisterMouse(ctx.Area, s.State.fieldHandler())
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
		setClipped(buf, ctx.Area.X+1, y, option, style, int(ctx.Area.Width)-2)

		// Draw scroll indicators on the right edge if there is overflow
		if i == startIdx && startIdx > 0 && ctx.Area.Width > 2 {
			buf.SetCell(ctx.Area.X+ctx.Area.Width-2, y, cell.Cell{Content: '▲', Style: style})
		} else if i == endIdx-1 && endIdx < len(s.Options) && ctx.Area.Width > 2 {
			buf.SetCell(ctx.Area.X+ctx.Area.Width-2, y, cell.Cell{Content: '▼', Style: style})
		}

		if ctx.RegisterMouse != nil {
			ctx.RegisterMouse(cell.NewRect(ctx.Area.X, y, ctx.Area.Width, 1), s.State.optionHandler(i))
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
