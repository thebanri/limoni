// Package testkit provides deterministic, terminal-independent helpers for
// testing Limoni widgets and frames.
package testkit

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
)

// Terminal is a fixed-size in-memory terminal used by deterministic tests.
// It deliberately does not start a backend, raw mode, event loop, or output
// writer.
type Terminal struct {
	buffer *buffer.Buffer
	frame  *terminal.Frame
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
	if t.frame.FocusManager != nil {
		t.frame.FocusManager.Clear()
	}
	if fn != nil {
		fn(t.frame)
	}
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
