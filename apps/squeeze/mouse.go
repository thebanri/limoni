package main

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/thebanri/limoni/core/driver"
)

// toEmulatorMouse turns a mouse event at (x, y) in the agent's screen into
// the one the emulator encodes for it.
func toEmulatorMouse(m driver.MouseEvent, x, y int) uv.MouseEvent {
	var mod uv.KeyMod
	if m.Shift {
		mod |= uv.ModShift
	}
	if m.Alt {
		mod |= uv.ModAlt
	}
	if m.Ctrl {
		mod |= uv.ModCtrl
	}
	ev := uv.Mouse{X: x, Y: y, Mod: mod}
	switch m.Button {
	case driver.MouseLeft:
		ev.Button = uv.MouseLeft
	case driver.MouseMiddle:
		ev.Button = uv.MouseMiddle
	case driver.MouseRight:
		ev.Button = uv.MouseRight
	case driver.MouseScrollUp:
		ev.Button = uv.MouseWheelUp
	case driver.MouseScrollDown:
		ev.Button = uv.MouseWheelDown
	case driver.MouseScrollLeft:
		ev.Button = uv.MouseWheelLeft
	case driver.MouseScrollRight:
		ev.Button = uv.MouseWheelRight
	case driver.MouseRelease:
		return uv.MouseReleaseEvent(ev)
	}
	switch {
	case m.Drag, m.Button == driver.MouseNone:
		return uv.MouseMotionEvent(ev)
	case ev.Button >= uv.MouseWheelUp && ev.Button <= uv.MouseWheelRight:
		return uv.MouseWheelEvent(ev)
	}
	return uv.MouseClickEvent(ev)
}

func isWheel(b driver.MouseButton) bool {
	return b == driver.MouseScrollUp || b == driver.MouseScrollDown
}
