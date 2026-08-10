// Package testkit provides deterministic, terminal-independent helpers for
// testing Limoni widgets and frames.
package testkit

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// Terminal is a fixed-size in-memory terminal used by deterministic tests.
// It deliberately does not start a backend, raw mode, event loop, or output
// writer.
type Terminal struct {
	buffer  *buffer.Buffer
	frame   *terminal.Frame
	capture func(ev backend.MouseEvent)
}

// NewTerminal creates an in-memory terminal with its origin at (0, 0).
func NewTerminal(width, height uint16) *Terminal {
	area := cell.NewRect(0, 0, width, height)
	focus := terminal.NewFocusManager()
	return &Terminal{
		buffer: buffer.NewBuffer(area),
		frame:  terminal.NewFrame(nil, focus),
	}
}

// Draw clears the test terminal and renders one deterministic frame.
func (t *Terminal) Draw(fn func(frame *terminal.Frame)) {
	if t == nil {
		return
	}
	t.buffer.Clear()
	t.frame.Buffer = t.buffer
	t.frame.Reset()
	t.capture = nil
	if t.frame.FocusManager != nil {
		t.frame.FocusManager.Clear()
	}
	if fn != nil {
		fn(t.frame)
	}
}

// Render draws a widget into area using the deterministic test frame.
func (t *Terminal) Render(widget widgets.Widget, area cell.Rect) {
	t.Draw(func(frame *terminal.Frame) {
		frame.RenderWidget(widget, area)
	})
}

// Mouse dispatches a mouse event through the regions registered by the last
// Draw call. It returns true when a region or a mouse capture handled it.
func (t *Terminal) Mouse(ev backend.MouseEvent) bool {
	if t == nil || t.frame == nil {
		return false
	}
	if t.capture != nil {
		t.capture(ev)
		if ev.Button == backend.MouseRelease {
			t.capture = nil
		}
		return true
	}

	if ev.Button != backend.MouseLeft && ev.Button != backend.MouseNone && ev.Button != backend.MouseScrollUp && ev.Button != backend.MouseScrollDown {
		return false
	}
	if ev.Button == backend.MouseLeft && ev.Drag {
		return false
	}

	for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
		region := t.frame.ClickRegions[i]
		if !region.Area.Contains(ev.X, ev.Y) {
			continue
		}
		if ev.Button != backend.MouseLeft && (!region.MouseOnly || (ev.Button != backend.MouseNone && ev.Button != backend.MouseScrollUp && ev.Button != backend.MouseScrollDown)) {
			continue
		}
		region.Handler(ev)
		t.capture = t.frame.TakeMouseCapture()
		return true
	}
	return false
}

// Click dispatches a left-button click at the given position.
func (t *Terminal) Click(x, y uint16) bool {
	return t.Mouse(backend.MouseEvent{X: x, Y: y, Button: backend.MouseLeft})
}

// Drag sends a captured drag event followed by a mouse release.
func (t *Terminal) Drag(x, y uint16) bool {
	if !t.Mouse(backend.MouseEvent{X: x, Y: y, Button: backend.MouseLeft, Drag: true}) {
		return false
	}
	t.Mouse(backend.MouseEvent{X: x, Y: y, Button: backend.MouseRelease})
	return true
}

// Snapshot returns the plain-text cell snapshot of the most recently drawn
// frame. Empty cells and wide-rune continuation cells are represented as
// spaces.
func (t *Terminal) Snapshot() string {
	if t == nil || t.buffer == nil {
		return ""
	}
	return t.buffer.Snapshot()
}

// Resize changes the fixed test surface. The next Draw uses the new size.
func (t *Terminal) Resize(width, height uint16) {
	if t == nil {
		return
	}
	t.buffer.Resize(cell.NewRect(0, 0, width, height))
}

// Area returns the current test surface bounds.
func (t *Terminal) Area() cell.Rect {
	if t == nil || t.buffer == nil {
		return cell.Rect{}
	}
	return t.buffer.Area
}

// Frame returns the most recently used frame. It is useful for assertions on
// focus, layers, registered regions, and other frame metadata.
func (t *Terminal) Frame() *terminal.Frame {
	if t == nil {
		return nil
	}
	return t.frame
}

// Focused returns the ID of the currently focused widget.
func (t *Terminal) Focused() string {
	if t == nil || t.frame == nil || t.frame.FocusManager == nil {
		return ""
	}
	return t.frame.FocusManager.Focused()
}

// FocusableIDs returns the focusable widget IDs registered by the last frame.
func (t *Terminal) FocusableIDs() []string {
	if t == nil || t.frame == nil || t.frame.FocusManager == nil {
		return nil
	}
	return t.frame.FocusManager.FocusableIDs()
}

// PropagateMouse dispatches a mouse event through the frame's capture,
// target, and bubble event handlers.
func (t *Terminal) PropagateMouse(ev backend.MouseEvent) bool {
	if t == nil || t.frame == nil {
		return false
	}
	return t.frame.DispatchEventRegions(ev)
}
