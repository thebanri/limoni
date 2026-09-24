package widgets

import (
	"math"
	"strconv"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// SplitDirection is how a SplitPane arranges its two panes.
type SplitDirection uint8

const (
	// SplitHorizontal puts the panes side by side with a vertical divider.
	SplitHorizontal SplitDirection = iota
	// SplitVertical stacks them with a horizontal divider.
	SplitVertical
)

// SplitState is where a SplitPane's divider sits. Keep one per pane across
// frames; dragging the divider and HandleKey move it.
type SplitState struct {
	// Ratio is the first pane's share of the space, 0 to 1. The zero value
	// is treated as an even split until the divider is first moved.
	Ratio float64

	moved bool

	// Geometry of the last frame, for the mouse handlers.
	origin, total     int
	minFirst, minLast int
	direction         SplitDirection
	capture           func(func(driver.MouseEvent))
	onMouse, onDrag   func(driver.MouseEvent)
}

func (s *SplitState) ratio() float64 {
	if !s.moved && s.Ratio == 0 {
		return 0.5
	}
	return s.Ratio
}

// SetRatio moves the divider; ratio is clamped to [0, 1].
func (s *SplitState) SetRatio(ratio float64) {
	if math.IsNaN(ratio) {
		return
	}
	s.Ratio = math.Max(0, math.Min(1, ratio))
	s.moved = true
}

// HandleKey moves the divider one cell with the arrow keys along the split
// (←/→ side by side, ↑/↓ stacked). It uses the size of the last frame, so
// call it after the pane has been drawn once.
func (s *SplitState) HandleKey(ev driver.KeyEvent) bool {
	if s == nil || s.total < 2 {
		return false
	}
	step := 0
	switch {
	case s.direction == SplitHorizontal && ev.Type == driver.KeyArrowLeft,
		s.direction == SplitVertical && ev.Type == driver.KeyArrowUp:
		step = -1
	case s.direction == SplitHorizontal && ev.Type == driver.KeyArrowRight,
		s.direction == SplitVertical && ev.Type == driver.KeyArrowDown:
		step = 1
	default:
		return false
	}
	s.placeDivider(s.dividerAt(s.total) + step)
	return true
}

// dividerAt is the divider's offset in a span of total cells, kept clear of
// both panes' minimum sizes when there is room for them.
func (s *SplitState) dividerAt(total int) int {
	pos := int(math.Round(s.ratio() * float64(total-1)))
	return clampDivider(pos, total, s.minFirst, s.minLast)
}

func clampDivider(pos, total, minFirst, minLast int) int {
	if hi := total - 1 - minLast; pos > hi {
		pos = hi
	}
	if pos < minFirst {
		pos = minFirst
	}
	if pos > total-1 {
		pos = total - 1
	}
	if pos < 0 {
		pos = 0
	}
	return pos
}

// placeDivider moves the divider to offset pos of the last frame's span.
func (s *SplitState) placeDivider(pos int) {
	pos = clampDivider(pos, s.total, s.minFirst, s.minLast)
	s.SetRatio(float64(pos) / float64(s.total-1))
}

func (s *SplitState) coordinate(ev driver.MouseEvent) int {
	if s.direction == SplitHorizontal {
		return int(ev.X) - s.origin
	}
	return int(ev.Y) - s.origin
}

// handlers are built once per state, so registering them each frame does
// not allocate.
func (s *SplitState) handlers() func(driver.MouseEvent) {
	if s.onMouse == nil {
		s.onDrag = func(ev driver.MouseEvent) {
			if ev.Drag {
				s.placeDivider(s.coordinate(ev))
			}
		}
		s.onMouse = func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseLeft && !ev.Drag && s.capture != nil {
				s.capture(s.onDrag)
			}
		}
	}
	return s.onMouse
}

// SplitPane shows two widgets side by side or stacked, with a divider the
// mouse can drag between them.
type SplitPane struct {
	ID        string
	Direction SplitDirection
	First     Widget
	Second    Widget
	State     *SplitState

	// MinFirst and MinSecond keep each pane at least this many cells wide
	// (or tall) while the divider moves, as far as the space allows.
	MinFirst, MinSecond uint16

	DividerStyle        cell.Style
	FocusedDividerStyle cell.Style
	// DividerRune replaces │ (side by side) or ─ (stacked).
	DividerRune rune
}

// Draw lays out and draws both panes and the divider. With a State it does
// not allocate after the first frame; without one the split stays even.
func (p SplitPane) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	if p.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}

	total := int(area.Width)
	origin := int(area.X)
	if p.Direction == SplitVertical {
		total, origin = int(area.Height), int(area.Y)
	}
	ratio := 0.5
	if s := p.State; s != nil {
		s.origin, s.total, s.direction = origin, total, p.Direction
		s.minFirst, s.minLast = int(p.MinFirst), int(p.MinSecond)
		ratio = s.ratio()
	}
	pos := clampDivider(int(math.Round(ratio*float64(total-1))), total, int(p.MinFirst), int(p.MinSecond))

	first, divider, second := area, area, area
	if p.Direction == SplitHorizontal {
		first.Width = uint16(pos)
		divider.X, divider.Width = area.X+uint16(pos), 1
		second.X = divider.X + 1
		second.Width = area.Width - uint16(pos) - 1
	} else {
		first.Height = uint16(pos)
		divider.Y, divider.Height = area.Y+uint16(pos), 1
		second.Y = divider.Y + 1
		second.Height = area.Height - uint16(pos) - 1
	}

	if p.First != nil && first.Width > 0 && first.Height > 0 {
		child := ctx
		child.Area = first
		p.First.Draw(child, buf)
	}
	if p.Second != nil && second.Width > 0 && second.Height > 0 {
		child := ctx
		child.Area = second
		p.Second.Draw(child, buf)
	}

	style := ctx.Style.Merge(p.DividerStyle)
	if ctx.IsFocused(p.ID) {
		style = style.Merge(p.FocusedDividerStyle)
	}
	glyph := p.DividerRune
	if glyph == 0 {
		glyph = '│'
		if p.Direction == SplitVertical {
			glyph = '─'
		}
	}
	for y := divider.Y; y < divider.Y+divider.Height; y++ {
		for x := divider.X; x < divider.X+divider.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: glyph, Style: style})
		}
	}

	if p.State != nil && ctx.RegisterMouse != nil {
		p.State.capture = ctx.CaptureMouse
		ctx.RegisterMouse(divider, p.State.handlers())
		// A resize arrow over the divider, where the terminal can show one.
		if ctx.RegisterClickAction != nil {
			pointer := "ew-resize"
			if p.Direction == SplitVertical {
				pointer = "ns-resize"
			}
			ctx.RegisterClickAction(divider, cell.ClickAction{Pointer: pointer})
		}
	}
}

// SizeHint takes all the space offered.
func (p SplitPane) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// AccessibilityNode describes the divider, which is what a keyboard user
// moves: a slider whose value is the first pane's share.
func (p SplitPane) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	ratio := 0.5
	if p.State != nil {
		ratio = p.State.ratio()
	}
	return accessibility.AccessibilityNode{
		ID:     p.ID,
		Role:   accessibility.RoleSlider,
		Label:  "Split",
		Value:  strconv.Itoa(int(math.Round(ratio*100))) + "%",
		State:  state,
		Bounds: bounds,
	}
}
