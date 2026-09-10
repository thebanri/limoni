package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// overlayComponent places an overlay component on top of a base component
// at an absolute (x, y) offset within the base's bounding rect.
// The overlay only overrides the cells it explicitly writes to; the base
// layer shows through in all other areas.
//
// Equivalent to Lipgloss's PlaceOverlay(x, y, fg, bg).
type overlayComponent struct {
	base    Component
	overlay Component
	x, y    uint16
}

// Overlay places an overlay component on top of a base component at
// absolute position (x, y) within the base's bounding area.
//
// The base is drawn first (full area), then the overlay is drawn at the
// specified offset. The overlay's bounding rect is clamped so it cannot
// exceed the base's area.
//
// Example:
//
//	limoni.Overlay(
//	    background,
//	    limoni.Border(modal, limoni.SymbolsSingle, style),
//	    10, 5,  // position from top-left of base
//	)
func Overlay(base, overlay Component, x, y uint16) Component {
	return &overlayComponent{
		base:    base,
		overlay: overlay,
		x:       x,
		y:       y,
	}
}

func (o *overlayComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	// Draw the base layer at full bounding rect.
	o.base.Draw(ctx, buf)

	// Compute clamped sub-area for the overlay.
	area := ctx.Area
	if o.x >= area.Width || o.y >= area.Height {
		return // overlay is completely outside the base area
	}
	subArea := cell.Rect{
		X:      area.X + o.x,
		Y:      area.Y + o.y,
		Width:  area.Width - o.x,
		Height: area.Height - o.y,
	}
	subCtx := cell.NewContext(subArea, ctx.Style)
	o.overlay.Draw(subCtx, buf)
}

func (o *overlayComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	// Overlay inherits the base's layout props.
	return o.base.LayoutInfo(maxArea)
}

func (o *overlayComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return o.base.SizeHint(maxArea)
}

func (o *overlayComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	// Overlay gets first crack at events (topmost layer).
	area := ctx.Area
	if o.x < area.Width && o.y < area.Height {
		subArea := cell.Rect{
			X:      area.X + o.x,
			Y:      area.Y + o.y,
			Width:  area.Width - o.x,
			Height: area.Height - o.y,
		}
		subCtx := cell.NewContext(subArea, ctx.Style)
		if DispatchEvent(o.overlay, subCtx, ev) {
			return true
		}
	}
	return DispatchEvent(o.base, ctx, ev)
}
