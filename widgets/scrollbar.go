package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// ScrollbarOrientation selects the axis a Scrollbar is drawn along.
type ScrollbarOrientation uint8

const (
	// ScrollbarVertical draws the bar top-to-bottom along the right edge of its area.
	ScrollbarVertical ScrollbarOrientation = iota
	// ScrollbarHorizontal draws the bar left-to-right along the bottom edge of its area.
	ScrollbarHorizontal
)

// Default scrollbar glyphs. The track uses a light shade and the thumb a full
// block so the bar reads correctly even in 16-color terminals where the track
// and thumb styles may collapse to the same color.
const (
	ScrollbarTrackSymbol = '░'
	ScrollbarThumbSymbol = '█'
)

// ScrollbarMetrics computes the thumb position and length for a scrollbar.
//
// contentLength is the total number of scrollable units, viewportLength the
// number currently visible, offset the index of the first visible unit, and
// trackLength the number of cells available to draw into.
//
// The returned thumb is always at least one cell long, is clamped inside the
// track, and reaches the far end exactly when offset is at its maximum — so
// "the thumb is at the bottom" reliably means "the last item is visible".
// A zero-length thumb is returned when the content fits and no bar is needed.
func ScrollbarMetrics(contentLength, viewportLength, offset, trackLength int) (thumbStart, thumbLength int) {
	if trackLength <= 0 || viewportLength <= 0 || contentLength <= viewportLength {
		return 0, 0
	}

	thumbLength = (viewportLength * trackLength) / contentLength
	if thumbLength < 1 {
		thumbLength = 1
	}
	if thumbLength > trackLength {
		thumbLength = trackLength
	}

	maxOffset := contentLength - viewportLength
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	travel := trackLength - thumbLength
	if travel > 0 && maxOffset > 0 {
		thumbStart = (offset * travel) / maxOffset
	}
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart > travel {
		thumbStart = travel
	}
	return thumbStart, thumbLength
}

// ScrollbarOffsetAt maps a click position along the track back to a content
// offset, centring the thumb on the clicked cell. It is the inverse of
// ScrollbarMetrics and is used for click-to-jump and thumb dragging.
func ScrollbarOffsetAt(contentLength, viewportLength, trackLength, position int) int {
	if trackLength <= 0 || viewportLength <= 0 || contentLength <= viewportLength {
		return 0
	}
	_, thumbLength := ScrollbarMetrics(contentLength, viewportLength, 0, trackLength)
	travel := trackLength - thumbLength
	maxOffset := contentLength - viewportLength

	position -= thumbLength / 2
	if travel <= 0 {
		return 0
	}
	if position < 0 {
		position = 0
	}
	if position > travel {
		position = travel
	}
	return (position * maxOffset) / travel
}

// Scrollbar renders a track and a proportional thumb for a scrollable region.
//
// It is a presentation widget: it owns no scroll state. Feed it the content and
// viewport lengths plus the current offset, and handle OnScroll to move the
// region it describes.
//
//	widgets.Scrollbar{
//		ContentLength:  len(rows),
//		ViewportLength: int(area.Height),
//		Offset:         state.Offset,
//		OnScroll:       func(off int) { state.Offset = off },
//	}
type Scrollbar struct {
	// Orientation selects the axis. The zero value is vertical.
	Orientation ScrollbarOrientation

	// ContentLength is the total number of scrollable units.
	ContentLength int
	// ViewportLength is the number of units visible at once.
	ViewportLength int
	// Offset is the index of the first visible unit.
	Offset int

	// TrackStyle and ThumbStyle override the theme defaults when set.
	TrackStyle cell.Style
	ThumbStyle cell.Style

	// TrackSymbol and ThumbSymbol override the default glyphs when non-zero.
	TrackSymbol rune
	ThumbSymbol rune

	// AlwaysVisible draws the track even when the content fits the viewport.
	// By default the widget renders nothing in that case, so layouts do not
	// show a dead bar next to short content.
	AlwaysVisible bool

	// OnScroll, when set, makes the bar interactive: clicking the track jumps
	// to that position and dragging the thumb scrolls continuously. The handler
	// receives the new first-visible-unit index.
	OnScroll func(offset int)
}

// Draw renders the scrollbar into the context area.
func (s Scrollbar) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	trackLength := int(area.Height)
	if s.Orientation == ScrollbarHorizontal {
		trackLength = int(area.Width)
	}

	thumbStart, thumbLength := ScrollbarMetrics(s.ContentLength, s.ViewportLength, s.Offset, trackLength)
	if thumbLength == 0 && !s.AlwaysVisible {
		return
	}

	base := ctx.Style
	trackStyle := base.Merge(s.TrackStyle)
	if s.TrackStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
		trackStyle = base.Merge(ctx.ThemeStyle("muted"))
	}
	thumbStyle := base.Merge(s.ThumbStyle)
	if s.ThumbStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
		thumbStyle = base.Merge(ctx.ThemeStyle("focus"))
	}

	trackSymbol := s.TrackSymbol
	if trackSymbol == 0 {
		trackSymbol = ScrollbarTrackSymbol
	}
	thumbSymbol := s.ThumbSymbol
	if thumbSymbol == 0 {
		thumbSymbol = ScrollbarThumbSymbol
	}

	for i := 0; i < trackLength; i++ {
		x, y := area.X, area.Y+uint16(i)
		if s.Orientation == ScrollbarHorizontal {
			x, y = area.X+uint16(i), area.Y
		}
		c := buf.Get(x, y)
		if c == nil {
			continue
		}
		if thumbLength > 0 && i >= thumbStart && i < thumbStart+thumbLength {
			c.Content = thumbSymbol
			c.Style = c.Style.Merge(thumbStyle)
			continue
		}
		c.Content = trackSymbol
		c.Style = c.Style.Merge(trackStyle)
	}

	s.registerMouse(ctx, area, trackLength)
}

func (s Scrollbar) registerMouse(ctx cell.Context, area cell.Rect, trackLength int) {
	if s.OnScroll == nil || ctx.RegisterMouse == nil || s.ContentLength <= s.ViewportLength {
		return
	}
	onScroll := s.OnScroll
	contentLength, viewportLength := s.ContentLength, s.ViewportLength
	horizontal := s.Orientation == ScrollbarHorizontal

	ctx.RegisterMouse(area, func(ev driver.MouseEvent) {
		switch ev.Button {
		case driver.MouseScrollUp:
			offset := s.Offset - 1
			if offset < 0 {
				offset = 0
			}
			onScroll(offset)
			return
		case driver.MouseScrollDown:
			offset := s.Offset + 1
			if max := contentLength - viewportLength; offset > max {
				offset = max
			}
			onScroll(offset)
			return
		case driver.MouseLeft:
		default:
			return
		}

		position := int(ev.Y) - int(area.Y)
		if horizontal {
			position = int(ev.X) - int(area.X)
		}
		onScroll(ScrollbarOffsetAt(contentLength, viewportLength, trackLength, position))
	})
}

// SizeHint reports the scrollbar's preferred size: one cell thick along its
// cross axis and as long as the available area along its main axis.
func (s Scrollbar) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if s.Orientation == ScrollbarHorizontal {
		return maxArea.Width, 1
	}
	return 1, maxArea.Height
}
