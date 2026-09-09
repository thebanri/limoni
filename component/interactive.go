package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// InteractiveComponent represents a composable element capable of handling input and mouse events.
type InteractiveComponent interface {
	Component
	HandleEvent(ctx cell.Context, ev *driver.Event) bool
}

type onClickComponent struct {
	child   Component
	handler func(ev driver.MouseEvent)
}

// OnClick wraps any component with a click handler.
// When clicked within its allocated area, the handler is invoked.
func OnClick(child Component, handler func(ev driver.MouseEvent)) InteractiveComponent {
	if child == nil {
		child = Empty()
	}
	return &onClickComponent{
		child:   child,
		handler: handler,
	}
}

func (o *onClickComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if o.child != nil {
		o.child.Draw(ctx, buf)
	}
	if ctx.RegisterClick != nil && o.handler != nil {
		h := o.handler
		area := ctx.Area
		ctx.RegisterClick(area, func() {
			h(driver.MouseEvent{
				X:      area.X,
				Y:      area.Y,
				Button: driver.MouseLeft,
			})
		})
	}
}

func (o *onClickComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	if o.child == nil {
		return LayoutProps{}
	}
	return o.child.LayoutInfo(maxArea)
}

func (o *onClickComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	if o.child == nil {
		return 0, 0
	}
	return o.child.SizeHint(maxArea)
}

func (o *onClickComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	if ev != nil && ev.Type == driver.EventMouse && o.handler != nil {
		if ctx.Area.Contains(ev.Mouse.X, ev.Mouse.Y) {
			if ev.Mouse.Button == driver.MouseLeft || ev.Mouse.Button == driver.MouseRelease {
				o.handler(ev.Mouse)
				return true
			}
		}
	}
	return DispatchEvent(o.child, ctx, ev)
}

type onEventComponent struct {
	child   Component
	handler func(ctx cell.Context, ev *driver.Event) bool
}

// OnEvent attaches an arbitrary event listener (keyboard, mouse, resize, paste) to a component.
func OnEvent(child Component, handler func(ctx cell.Context, ev *driver.Event) bool) InteractiveComponent {
	if child == nil {
		child = Empty()
	}
	return &onEventComponent{
		child:   child,
		handler: handler,
	}
}

func (oe *onEventComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if oe.child != nil {
		oe.child.Draw(ctx, buf)
	}
}

func (oe *onEventComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	if oe.child == nil {
		return LayoutProps{}
	}
	return oe.child.LayoutInfo(maxArea)
}

func (oe *onEventComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	if oe.child == nil {
		return 0, 0
	}
	return oe.child.SizeHint(maxArea)
}

func (oe *onEventComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	if oe.handler != nil && oe.handler(ctx, ev) {
		return true
	}
	return DispatchEvent(oe.child, ctx, ev)
}

// OnKey binds a specific keyboard key (e.g. driver.KeyEsc, driver.KeyEnter) to a component.
// If the key matches and handler returns true, event propagation stops.
func OnKey(child Component, key driver.KeyType, handler func(ev driver.KeyEvent) bool) InteractiveComponent {
	return OnEvent(child, func(ctx cell.Context, ev *driver.Event) bool {
		if ev != nil && ev.Type == driver.EventKey && ev.Key.Type == key && handler != nil {
			return handler(ev.Key)
		}
		return false
	})
}

// OnRune binds a specific character rune (e.g. 'q', 'j', 'k', '?') to a component.
// If the key matches and handler returns true, event propagation stops.
func OnRune(child Component, r rune, handler func(ev driver.KeyEvent) bool) InteractiveComponent {
	return OnEvent(child, func(ctx cell.Context, ev *driver.Event) bool {
		if ev != nil && ev.Type == driver.EventKey && ev.Key.Ch == r && handler != nil {
			return handler(ev.Key)
		}
		return false
	})
}

// DispatchEvent routes an event down a component tree starting from root within ctx.Area.
// Returns true if any component consumed the event.
func DispatchEvent(root Component, ctx cell.Context, ev *driver.Event) bool {
	if root == nil || ev == nil {
		return false
	}
	if ic, ok := root.(InteractiveComponent); ok {
		return ic.HandleEvent(ctx, ev)
	}
	return false
}
