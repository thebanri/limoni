package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// ViewportState holds the scroll position of a Viewport and owns the offscreen
// buffer the child is rendered into.
//
// The buffer is retained across frames and only reallocated when the content
// size changes, so a steady-state scroll costs zero heap allocations.
type ViewportState struct {
	// OffsetY is the first visible content row.
	OffsetY int
	// OffsetX is the first visible content column.
	OffsetX int

	// contentHeight and contentWidth record the last measured child size, and
	// visibleHeight the viewport height, so callers can clamp, page and query
	// position without re-measuring.
	contentHeight int
	contentWidth  int
	visibleHeight int

	scratch *buffer.Buffer
}

// NewViewportState returns a viewport scrolled to the top-left.
func NewViewportState() *ViewportState { return &ViewportState{} }

// ContentSize reports the child size measured during the last Draw.
// It returns zeroes before the first frame.
func (s *ViewportState) ContentSize() (width, height int) {
	if s == nil {
		return 0, 0
	}
	return s.contentWidth, s.contentHeight
}

// ScrollBy moves the viewport by delta rows, clamping to the content bounds
// measured during the last Draw. Negative values scroll up.
func (s *ViewportState) ScrollBy(delta int) {
	if s == nil {
		return
	}
	s.OffsetY += delta
	s.clampVertical()
}

// ScrollTo jumps to an absolute content row, clamped to the content bounds.
func (s *ViewportState) ScrollTo(row int) {
	if s == nil {
		return
	}
	s.OffsetY = row
	s.clampVertical()
}

// ScrollHorizontallyBy moves the viewport by delta columns.
func (s *ViewportState) ScrollHorizontallyBy(delta int) {
	if s == nil {
		return
	}
	s.OffsetX += delta
	if s.OffsetX < 0 {
		s.OffsetX = 0
	}
}

// GotoTop scrolls to the first content row.
func (s *ViewportState) GotoTop() { s.ScrollTo(0) }

// GotoBottom scrolls so the last content row is visible.
func (s *ViewportState) GotoBottom() { s.ScrollTo(s.contentHeight) }

// AtTop reports whether the viewport shows the first content row.
func (s *ViewportState) AtTop() bool { return s == nil || s.OffsetY <= 0 }

// AtBottom reports whether the viewport shows the last content row.
// It returns true before the first frame, when no content has been measured.
func (s *ViewportState) AtBottom() bool {
	if s == nil || s.visibleHeight <= 0 {
		return true
	}
	return s.OffsetY >= s.contentHeight-s.visibleHeight
}

// ScrollPercent reports the scroll position as a fraction between 0 and 1.
// Content that fits entirely in the viewport reports 1.
func (s *ViewportState) ScrollPercent() float64 {
	if s == nil || s.visibleHeight <= 0 {
		return 0
	}
	maxOffset := s.contentHeight - s.visibleHeight
	if maxOffset <= 0 {
		return 1
	}
	return float64(s.OffsetY) / float64(maxOffset)
}

func (s *ViewportState) clampVertical() {
	if s.OffsetY < 0 {
		s.OffsetY = 0
	}
	if s.visibleHeight <= 0 {
		return
	}
	if maxOffset := s.contentHeight - s.visibleHeight; s.OffsetY > maxOffset {
		s.OffsetY = maxOffset
	}
	if s.OffsetY < 0 {
		s.OffsetY = 0
	}
}

// Viewport renders a child widget that may be larger than the available area
// and shows a scrollable window onto it.
//
// It is the general-purpose scroll container: List and Table implement their own
// virtualised scrolling because they can skip invisible rows entirely, but any
// other widget — a Paragraph, a Markdown document, a composed dashboard panel —
// becomes scrollable by wrapping it here.
//
//	widgets.Viewport{
//		Child: &widgets.Paragraph{Text: longText},
//		State: state,
//		Scrollbar: true,
//	}
//
// The child is drawn once per frame into an offscreen buffer sized to the
// content, then the visible window is copied out. Content height is taken from
// ContentHeight when set, otherwise from the child's SizeHint.
//
// # Cost
//
// Rendering is O(content), not O(visible): the child is drawn in full every
// frame even though only a window is shown, and the offscreen buffer holds
// contentWidth × contentHeight cells. Both the draw and the buffer are
// allocation-free in steady state — the buffer is reused until the content size
// changes — but the per-frame work grows with the content.
//
// Measured on an 80×24 viewport: content that fits takes ~6.6 µs per frame and
// skips the offscreen buffer entirely; 1,000 rows of content takes ~960 µs.
// That is fine for a long document or a tall panel, and wrong for a large
// dataset. For thousands of rows use List or Table, which virtualise and render
// only the visible rows, or drive this widget with ContentHeight and a child
// that clips to ctx.Area itself.
type Viewport struct {
	// ID registers the viewport with the focus manager when set.
	ID string

	// Child is the widget rendered inside the viewport.
	Child Widget

	// State carries the scroll offsets. A nil State renders the child
	// unscrolled, which makes Viewport safe to construct before wiring state.
	State *ViewportState

	// ContentHeight overrides the measured child height. Set it when the child
	// cannot report its own size, for example a Canvas drawing a tall diagram.
	ContentHeight int
	// ContentWidth overrides the measured child width.
	ContentWidth int

	// Style is merged into the inherited context style for the whole area.
	Style cell.Style

	// Scrollbar draws a vertical scrollbar along the right edge, reserving one
	// column from the child's width.
	Scrollbar bool
	// ScrollbarTrackStyle and ScrollbarThumbStyle override theme defaults.
	ScrollbarTrackStyle cell.Style
	ScrollbarThumbStyle cell.Style

	// MouseWheelStep is the number of rows scrolled per wheel notch.
	// Values below one fall back to three, matching common terminal behaviour.
	MouseWheelStep int

	// DisableMouse turns off wheel scrolling and scrollbar interaction.
	DisableMouse bool
}

// Draw renders the visible window of the child widget.
func (v Viewport) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 || v.Child == nil {
		return
	}

	if v.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(v.ID)
	}

	style := ctx.Style.Merge(v.Style)

	contentArea := area
	if v.Scrollbar && area.Width > 1 {
		contentArea.Width--
	}

	contentWidth, contentHeight := v.measure(contentArea)
	state := v.State

	offsetX, offsetY := 0, 0
	if state != nil {
		state.contentWidth = contentWidth
		state.contentHeight = contentHeight
		state.visibleHeight = int(contentArea.Height)
		state.clampVertical()
		if state.OffsetX < 0 {
			state.OffsetX = 0
		}
		if maxX := contentWidth - int(contentArea.Width); state.OffsetX > maxX {
			if maxX < 0 {
				maxX = 0
			}
			state.OffsetX = maxX
		}
		offsetX, offsetY = state.OffsetX, state.OffsetY
	}

	v.drawWindow(ctx, buf, contentArea, style, contentWidth, contentHeight, offsetX, offsetY)

	if v.Scrollbar && area.Width > 1 {
		bar := Scrollbar{
			ContentLength:  contentHeight,
			ViewportLength: int(contentArea.Height),
			Offset:         offsetY,
			TrackStyle:     v.ScrollbarTrackStyle,
			ThumbStyle:     v.ScrollbarThumbStyle,
		}
		if state != nil && !v.DisableMouse {
			bar.OnScroll = func(offset int) { state.ScrollTo(offset) }
		}
		barCtx := ctx
		barCtx.Area = cell.NewRect(area.X+area.Width-1, area.Y, 1, area.Height)
		barCtx.Style = style
		bar.Draw(barCtx, buf)
	}

	v.registerWheel(ctx, area, contentArea)
}

// measure resolves the content size, preferring explicit overrides and falling
// back to the child's SizeHint measured against a generously tall area.
func (v Viewport) measure(contentArea cell.Rect) (width, height int) {
	width, height = v.ContentWidth, v.ContentHeight
	if width > 0 && height > 0 {
		return width, height
	}

	// Measure against an area as wide as the viewport but effectively unbounded
	// vertically, so wrapping widgets report the height they actually need.
	probe := cell.NewRect(0, 0, contentArea.Width, ^uint16(0))
	hintW, hintH := v.Child.SizeHint(probe)

	if width <= 0 {
		width = int(hintW)
	}
	if height <= 0 {
		height = int(hintH)
	}
	if width <= 0 {
		width = int(contentArea.Width)
	}
	if height <= 0 {
		height = int(contentArea.Height)
	}
	return width, height
}

// drawWindow renders the child offscreen and blits the visible rectangle.
func (v Viewport) drawWindow(
	ctx cell.Context,
	buf *buffer.Buffer,
	contentArea cell.Rect,
	style cell.Style,
	contentWidth, contentHeight, offsetX, offsetY int,
) {
	// Nothing to scroll: draw straight into the destination and skip the
	// offscreen buffer entirely.
	if contentHeight <= int(contentArea.Height) && contentWidth <= int(contentArea.Width) {
		childCtx := ctx
		childCtx.Area = contentArea
		childCtx.Style = style
		v.Child.Draw(childCtx, buf)
		return
	}

	scratch := v.scratchBuffer(contentWidth, contentHeight)
	if scratch == nil {
		return
	}
	scratch.Clear()

	childCtx := ctx
	childCtx.Area = cell.NewRect(0, 0, uint16(contentWidth), uint16(contentHeight))
	childCtx.Style = style
	// Interaction callbacks are coordinate-based and would register offscreen
	// rectangles, so they are dropped for the offscreen pass.
	childCtx.RegisterClick = nil
	childCtx.RegisterMouse = nil
	childCtx.RegisterEvent = nil
	childCtx.RegisterImage = nil
	v.Child.Draw(childCtx, scratch)

	for row := uint16(0); row < contentArea.Height; row++ {
		srcY := offsetY + int(row)
		if srcY < 0 || srcY >= contentHeight {
			continue
		}
		for col := uint16(0); col < contentArea.Width; col++ {
			srcX := offsetX + int(col)
			if srcX < 0 || srcX >= contentWidth {
				continue
			}
			src := scratch.Get(uint16(srcX), uint16(srcY))
			if src == nil {
				continue
			}
			buf.SetCell(contentArea.X+col, contentArea.Y+row, *src)
		}
	}
}

// scratchBuffer returns a reusable offscreen buffer of the requested size,
// reallocating only when the content size changes.
func (v Viewport) scratchBuffer(width, height int) *buffer.Buffer {
	if width <= 0 || height <= 0 {
		return nil
	}
	area := cell.NewRect(0, 0, uint16(width), uint16(height))

	if v.State == nil {
		return buffer.NewBuffer(area)
	}
	if v.State.scratch == nil || v.State.scratch.Area != area {
		v.State.scratch = buffer.NewBuffer(area)
	}
	return v.State.scratch
}

func (v Viewport) registerWheel(ctx cell.Context, area, contentArea cell.Rect) {
	if v.DisableMouse || v.State == nil || ctx.RegisterMouse == nil {
		return
	}
	step := v.MouseWheelStep
	if step < 1 {
		step = 3
	}
	state := v.State

	// Registered on the content area only, so the scrollbar keeps its own
	// click-to-jump handler on the last column.
	wheelArea := contentArea
	if wheelArea.Width == 0 {
		wheelArea = area
	}
	ctx.RegisterMouse(wheelArea, func(ev driver.MouseEvent) {
		switch ev.Button {
		case driver.MouseScrollUp:
			state.ScrollBy(-step)
		case driver.MouseScrollDown:
			state.ScrollBy(step)
		}
	})
}

// SizeHint reports the full area: a viewport fills whatever it is given.
func (v Viewport) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}
