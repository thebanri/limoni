package widgets

import (
	"strconv"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// SliderState stores the current value of a slider. Keep one per slider
// across frames: it also holds the mouse handler, built once, so drawing the
// slider does not allocate.
type SliderState struct {
	Value int

	// The last frame's geometry and settings, read by the mouse handlers.
	x, width                    int
	min, max                    int
	id                          string
	disableScroll, disableFocus bool
	onChange                    func(int)
	setFocus                    func(string)
	capture                     func(func(driver.MouseEvent))
	onMouse, onDrag             func(driver.MouseEvent)

	// The semantic node's text, rebuilt only when what it says changes.
	label              string
	labelMin, labelMax int
	value              string
	valueOf            int
	valueSet           bool
}

func NewSliderState(value int) *SliderState { return &SliderState{Value: value} }

// Set clamps the slider value to its configured range.
func (s *SliderState) Set(value, min, max int) {
	if s == nil {
		return
	}
	if min > max {
		min, max = max, min
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	s.Value = value
}

// HandleKey adjusts the slider with arrow keys.
func (s *SliderState) HandleKey(ev driver.KeyEvent, min, max int) bool {
	if s == nil {
		return false
	}
	switch ev.Type {
	case driver.KeyArrowLeft, driver.KeyArrowDown:
		s.Set(s.Value-1, min, max)
		return true
	case driver.KeyArrowRight, driver.KeyArrowUp:
		s.Set(s.Value+1, min, max)
		return true
	case driver.KeyHome:
		s.Set(min, min, max)
		return true
	case driver.KeyEnd:
		s.Set(max, min, max)
		return true
	}
	return false
}

// Slider is a horizontal mouse- and keyboard-controlled numeric slider.
type Slider struct {
	ID            string
	State         *SliderState
	Min           int
	Max           int
	Style         cell.Style
	TrackStyle    cell.Style
	FilledStyle   cell.Style
	ThumbStyle    cell.Style
	FocusedStyle  cell.Style
	DisableScroll bool // Turns off changing the value with the mouse wheel
	DisableFocus  bool // Turns off taking the focus on click
	OnChange      func(value int)
}

func (s Slider) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if s.ID == "" || s.State == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 || s.Max <= s.Min {
		return
	}
	s.State.Set(s.State.Value, s.Min, s.Max)
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(s.ID)
	}
	style := ctx.Style.Merge(s.Style)
	if ctx.IsFocused(s.ID) {
		style = style.Merge(s.FocusedStyle)
	}
	track := ctx.Style.Merge(s.TrackStyle)
	filled := ctx.Style.Merge(s.FilledStyle)
	thumb := ctx.Style.Merge(s.ThumbStyle)
	if track == (cell.Style{}) {
		track = style
	}
	if filled == (cell.Style{}) {
		filled = style
	}
	if thumb == (cell.Style{}) {
		thumb = style
	}

	width := int(ctx.Area.Width)
	position := (s.State.Value - s.Min) * (width - 1) / (s.Max - s.Min)
	for x := 0; x < width; x++ {
		cellStyle := track
		content := '─'
		if x <= position {
			cellStyle = filled
		}
		if x == position {
			content = '●'
			cellStyle = thumb
		}
		px := ctx.Area.X + uint16(x)
		py := ctx.Area.Y
		if c := buf.Get(px, py); c != nil {
			c.Content = content
			c.Style = c.Style.Merge(cellStyle)
		} else {
			buf.SetCell(px, py, cell.Cell{Content: content, Style: cellStyle})
		}
	}
	if ctx.RegisterMouse != nil {
		st := s.State
		st.x, st.width, st.min, st.max = int(ctx.Area.X), width, s.Min, s.Max
		st.id, st.disableScroll, st.disableFocus = s.ID, s.DisableScroll, s.DisableFocus
		st.onChange, st.setFocus, st.capture = s.OnChange, ctx.SetFocus, ctx.CaptureMouse
		ctx.RegisterMouse(ctx.Area, st.handlers())
	}
}

// handlers are built once per state and read the last frame's settings, so
// registering them each frame does not allocate.
func (s *SliderState) handlers() func(driver.MouseEvent) {
	if s.onMouse == nil {
		s.onDrag = func(ev driver.MouseEvent) {
			if ev.Button != driver.MouseRelease && ev.Drag {
				s.setFromColumn(ev.X)
			}
		}
		s.onMouse = func(ev driver.MouseEvent) {
			if !s.disableScroll && (ev.Button == driver.MouseScrollUp || ev.Button == driver.MouseScrollDown) {
				step := 1
				if ev.Button == driver.MouseScrollDown {
					step = -1
				}
				s.change(s.Value + step)
				s.focus()
				return
			}
			if ev.Button != driver.MouseLeft || ev.Drag {
				return
			}
			s.focus()
			s.setFromColumn(ev.X)
			if s.capture != nil {
				s.capture(s.onDrag)
			}
		}
	}
	return s.onMouse
}

func (s *SliderState) focus() {
	if !s.disableFocus && s.setFocus != nil {
		s.setFocus(s.id)
	}
}

func (s *SliderState) change(value int) {
	s.Set(value, s.min, s.max)
	if s.onChange != nil {
		s.onChange(s.Value)
	}
}

// setFromColumn moves the thumb to screen column x.
func (s *SliderState) setFromColumn(x uint16) {
	relative := int(x) - s.x
	if relative < 0 {
		relative = 0
	}
	if relative >= s.width {
		relative = s.width - 1
	}
	if s.width <= 1 {
		s.change(s.min)
		return
	}
	s.change(s.min + relative*(s.max-s.min)/(s.width-1))
}

func (s Slider) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// AccessibilityNode returns the semantic node description for Slider.
func (s Slider) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	label, val := "", ""
	if st := s.State; st != nil {
		if st.label == "" || st.labelMin != s.Min || st.labelMax != s.Max {
			st.label = "Slider range " + strconv.Itoa(s.Min) + " to " + strconv.Itoa(s.Max)
			st.labelMin, st.labelMax = s.Min, s.Max
		}
		if !st.valueSet || st.valueOf != st.Value {
			st.value, st.valueOf, st.valueSet = strconv.Itoa(st.Value), st.Value, true
		}
		label, val = st.label, st.value
	} else {
		label = "Slider range " + strconv.Itoa(s.Min) + " to " + strconv.Itoa(s.Max)
	}
	return accessibility.AccessibilityNode{
		ID:     s.ID,
		Role:   accessibility.RoleSlider,
		Label:  label,
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}
