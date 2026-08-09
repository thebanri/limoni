package widgets

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// SliderState stores the current value of a slider.
type SliderState struct {
	Value int
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
func (s *SliderState) HandleKey(ev backend.KeyEvent, min, max int) bool {
	if s == nil {
		return false
	}
	switch ev.Type {
	case backend.KeyArrowLeft, backend.KeyArrowDown:
		s.Set(s.Value-1, min, max)
		return true
	case backend.KeyArrowRight, backend.KeyArrowUp:
		s.Set(s.Value+1, min, max)
		return true
	case backend.KeyHome:
		s.Set(min, min, max)
		return true
	case backend.KeyEnd:
		s.Set(max, min, max)
		return true
	}
	return false
}

// Slider is a horizontal mouse- and keyboard-controlled numeric slider.
type Slider struct {
	ID           string
	State        *SliderState
	Min          int
	Max          int
	Style        cell.Style
	TrackStyle   cell.Style
	FilledStyle  cell.Style
	ThumbStyle   cell.Style
	FocusedStyle cell.Style
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
	if ctx.FocusedID == s.ID {
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
		buf.SetCell(ctx.Area.X+uint16(x), ctx.Area.Y, cell.Cell{Content: content, Style: cellStyle})
	}
	setValue := func(x uint16) {
		relative := int(x) - int(ctx.Area.X)
		if relative < 0 {
			relative = 0
		}
		if relative >= width {
			relative = width - 1
		}
		value := s.Min + relative*(s.Max-s.Min)/(width-1)
		s.State.Set(value, s.Min, s.Max)
	}
	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			if ev.Button != backend.MouseLeft {
				return
			}
			if !ev.Drag {
				if ctx.SetFocus != nil {
					ctx.SetFocus(s.ID)
				}
				setValue(ev.X)
				if ctx.CaptureMouse != nil {
					ctx.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							return
						}
						if dragEv.Drag {
							setValue(dragEv.X)
						}
					})
				}
			}
		})
	}
}

func (s Slider) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }
