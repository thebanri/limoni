package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// constraintComponent wraps a child component and clamps its bounding box and layout properties.
type constraintComponent struct {
	child                  Component
	minW, maxW, minH, maxH uint16
}

// Constrain enforces minimum and maximum width and height bounds on a child component.
// Set any parameter to 0 to leave it unconstrained.
func Constrain(minW, maxW, minH, maxH uint16, child Component) Component {
	return &constraintComponent{
		child: child,
		minW:  minW,
		maxW:  maxW,
		minH:  minH,
		maxH:  maxH,
	}
}

// MaxWidth clamps the maximum width of a child component.
func MaxWidth(maxW uint16, child Component) Component {
	return Constrain(0, maxW, 0, 0, child)
}

// MaxHeight clamps the maximum height of a child component.
func MaxHeight(maxH uint16, child Component) Component {
	return Constrain(0, 0, 0, maxH, child)
}

// MinWidth ensures a child component takes at least minW width.
func MinWidth(minW uint16, child Component) Component {
	return Constrain(minW, 0, 0, 0, child)
}

// MinHeight ensures a child component takes at least minH height.
func MinHeight(minH uint16, child Component) Component {
	return Constrain(0, 0, minH, 0, child)
}

func (c *constraintComponent) clampArea(area cell.Rect) cell.Rect {
	r := area
	if c.maxW > 0 && r.Width > c.maxW {
		r.Width = c.maxW
	}
	if c.maxH > 0 && r.Height > c.maxH {
		r.Height = c.maxH
	}
	return r
}

func (c *constraintComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	ctx.Area = c.clampArea(ctx.Area)
	c.child.Draw(ctx, buf)
}

func (c *constraintComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	clampedMax := maxArea
	if c.maxW > 0 && clampedMax.Width > c.maxW {
		clampedMax.Width = c.maxW
	}
	if c.maxH > 0 && clampedMax.Height > c.maxH {
		clampedMax.Height = c.maxH
	}

	props := c.child.LayoutInfo(clampedMax)
	if c.minW > 0 && props.MinWidth < c.minW {
		props.MinWidth = c.minW
	}
	if c.maxW > 0 {
		if props.MaxWidth == 0 || props.MaxWidth > c.maxW {
			props.MaxWidth = c.maxW
		}
		if props.MinWidth > c.maxW {
			props.MinWidth = c.maxW
		}
	}
	if c.minH > 0 && props.MinHeight < c.minH {
		props.MinHeight = c.minH
	}
	if c.maxH > 0 {
		if props.MaxHeight == 0 || props.MaxHeight > c.maxH {
			props.MaxHeight = c.maxH
		}
		if props.MinHeight > c.maxH {
			props.MinHeight = c.maxH
		}
	}
	return props
}

func (c *constraintComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := c.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

func (c *constraintComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	ctx.Area = c.clampArea(ctx.Area)
	if ev != nil && ev.Type == driver.EventMouse && !ctx.Area.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}
	return DispatchEvent(c.child, ctx, ev)
}

// Place positions a child component inside a fixed bounding box of (width x height)
// aligned horizontally and vertically according to hAlign and vAlign.
// Equivalent to Lipgloss's lipgloss.Place().
//
// Example:
//
//	limoni.Place(80, 24, limoni.AlignCenter, limoni.AlignMiddle, modalDialog)
func Place(width, height uint16, hAlign HAlign, vAlign VAlign, child Component) Component {
	return Fixed(width, height, Align(child, hAlign, vAlign))
}
