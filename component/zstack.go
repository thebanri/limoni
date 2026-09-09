package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// ZStackLayout arranges components along the depth axis (Painter's algorithm).
// Background is children[0] and topmost foreground is children[len-1].
type ZStackLayout struct {
	children []Component
}

// ZStack creates a depth-axis container rendering from background to foreground.
// All children share the bounding area, allowing constrained foreground layers
// (such as centered modals or floating toolbars) to overlay background content
// with zero offscreen buffer allocations.
func ZStack(children ...Component) *ZStackLayout {
	return &ZStackLayout{children: children}
}

// Draw renders each layer directly onto the destination buffer in order (background to foreground).
func (z *ZStackLayout) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if len(z.children) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	for _, child := range z.children {
		if child != nil {
			child.Draw(ctx, buf)
		}
	}
}

// LayoutInfo calculates the bounding requirements across all layers.
func (z *ZStackLayout) LayoutInfo(maxArea cell.Rect) LayoutProps {
	var maxW, maxH uint16
	var maxFlex uint16

	for _, c := range z.children {
		if c == nil {
			continue
		}
		p := c.LayoutInfo(maxArea)
		if p.MinWidth > maxW {
			maxW = p.MinWidth
		}
		if p.MinHeight > maxH {
			maxH = p.MinHeight
		}
		if p.Flex > maxFlex {
			maxFlex = p.Flex
		}
	}

	return LayoutProps{
		MinWidth:  maxW,
		MinHeight: maxH,
		Flex:      maxFlex,
	}
}

// SizeHint returns the maximum width and height required across all stacked layers.
func (z *ZStackLayout) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := z.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// HandleEvent dispatches events from foreground to background (top to bottom).
// The topmost layer that handles the event consumes it and terminates propagation.
func (z *ZStackLayout) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	for i := len(z.children) - 1; i >= 0; i-- {
		child := z.children[i]
		if child == nil {
			continue
		}
		if DispatchEvent(child, ctx, ev) {
			return true
		}
	}
	return false
}
