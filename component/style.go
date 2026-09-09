package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

type styleComponent struct {
	child Component
	style cell.Style
	hasFg bool
	fg    cell.Color
	hasBg bool
	bg    cell.Color
}

// WithStyle cascades a cell.Style into the component subtree by merging it into ctx.Style.
func WithStyle(style cell.Style, child Component) Component {
	if child == nil {
		return Empty()
	}
	return &styleComponent{
		child: child,
		style: style,
	}
}

// WithForeground cascades a foreground color override into the component subtree.
func WithForeground(color cell.Color, child Component) Component {
	if child == nil {
		return Empty()
	}
	return &styleComponent{
		child: child,
		hasFg: true,
		fg:    color,
	}
}

// WithBackground cascades a background color override into the component subtree.
func WithBackground(color cell.Color, child Component) Component {
	if child == nil {
		return Empty()
	}
	return &styleComponent{
		child: child,
		hasBg: true,
		bg:    color,
	}
}

func (s *styleComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if s.child == nil {
		return
	}

	// Mutate context copy on stack frame (zero heap allocations)
	newCtx := ctx
	if s.hasFg {
		newCtx.Style.Fg = s.fg
	} else if s.hasBg {
		newCtx.Style.Bg = s.bg
	} else {
		newCtx.Style = newCtx.Style.Merge(s.style)
	}

	s.child.Draw(newCtx, buf)
}

func (s *styleComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	if s.child == nil {
		return LayoutProps{}
	}
	return s.child.LayoutInfo(maxArea)
}

func (s *styleComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	if s.child == nil {
		return 0, 0
	}
	return s.child.SizeHint(maxArea)
}

func (s *styleComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	if s.child == nil {
		return false
	}
	return DispatchEvent(s.child, ctx, ev)
}
