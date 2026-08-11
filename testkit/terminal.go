// Package testkit provides deterministic, terminal-independent helpers for
// testing Limoni widgets and frames.
package testkit

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
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

	// Dispatch semantic regions before legacy click regions, matching the production router.
	if ev.Button == backend.MouseNone {
		t.frame.DispatchPointerMove(ev)
	}
	if t.frame.DispatchEventRegions(ev) {
		t.capture = t.frame.TakeMouseCapture()
		return true
	}
	t.capture = t.frame.TakeMouseCapture()

	for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
		region := t.frame.ClickRegions[i]
		if !region.Area.Contains(ev.X, ev.Y) {
			continue
		}
		if region.MouseOnly {
			if ev.Button != backend.MouseLeft && ev.Button != backend.MouseNone && ev.Button != backend.MouseScrollUp && ev.Button != backend.MouseScrollDown {
				continue
			}
		} else if ev.Button != backend.MouseLeft {
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

// StyleSnapshot returns a deterministic style representation for every cell.
func (t *Terminal) StyleSnapshot() string {
	if t == nil || t.buffer == nil {
		return ""
	}
	var b strings.Builder
	for y := uint16(0); y < t.buffer.Area.Height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := uint16(0); x < t.buffer.Area.Width; x++ {
			c := t.buffer.Get(x, y)
			fmt.Fprintf(&b, "%08x/%08x/%04x", uint32(c.Style.Fg), uint32(c.Style.Bg), uint16(c.Style.Modifier))
			if x+1 < t.buffer.Area.Width {
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

// AssertSnapshot returns a descriptive error when the text snapshot differs.
func (t *Terminal) AssertSnapshot(expected string) error {
	if got := t.Snapshot(); got != expected {
		return fmt.Errorf("snapshot mismatch:\n%s", FormatSnapshotDiff(DiffSnapshot(expected, got)))
	}
	return nil
}

func (t *Terminal) AssertHovered(id string) error {
	if got := t.HoveredRegionID(); got != id {
		return fmt.Errorf("hovered region=%q want %q", got, id)
	}
	return nil
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

// MovePointer updates hover state for the current frame.
func (t *Terminal) MovePointer(x, y uint16) bool {
	if t == nil || t.frame == nil {
		return false
	}
	return t.frame.DispatchPointerMove(backend.MouseEvent{X: x, Y: y, Button: backend.MouseNone})
}

// HoveredRegionID returns the currently hovered event region ID.
func (t *Terminal) HoveredRegionID() string {
	if t == nil || t.frame == nil {
		return ""
	}
	return t.frame.HoveredRegionID()
}

// AccessibilityTree returns semantic nodes from the last frame.
func (t *Terminal) AccessibilityTree() []accessibility.AccessibilityNode {
	if t == nil || t.frame == nil {
		return nil
	}
	return t.frame.AccessibilityTree()
}

// ValidateAccessibility validates the semantic tree from the last draw.
func (t *Terminal) ValidateAccessibility() error {
	if t == nil || t.frame == nil {
		return nil
	}
	return t.frame.ValidateAccessibility()
}

// AssertImageRegistration verifies an image area and z-index registered by
// the current frame.
func (t *Terminal) AssertImageRegistration(area cell.Rect, zIndex int) error {
	if t == nil || t.frame == nil {
		return fmt.Errorf("image registration: terminal is nil")
	}
	for _, region := range t.frame.ImageRegionsSnapshot() {
		if region.Area == area && region.ZIndex == zIndex {
			return nil
		}
	}
	return fmt.Errorf("image registration not found: area=%+v z=%d", area, zIndex)
}

// AccessibilityLineMode returns the semantic snapshot as screen-reader-safe
// line-oriented text.
func (t *Terminal) AccessibilityLineMode(mode accessibility.Mode) string {
	if t == nil || t.frame == nil {
		return ""
	}
	return t.frame.AccessibilityLineMode(mode)
}

// WriteAccessibilityLineMode streams the last semantic snapshot to a writer.
func (t *Terminal) WriteAccessibilityLineMode(w io.Writer, mode accessibility.Mode) error {
	if t == nil || t.frame == nil {
		return mode.WriteLineMode(w, nil)
	}
	return t.frame.WriteAccessibilityLineMode(w, mode)
}

// AssertAccessibilityContains checks that the line-mode semantic snapshot
// contains the requested text.
func (t *Terminal) AssertAccessibilityContains(mode accessibility.Mode, text string) error {
	snapshot := t.AccessibilityLineMode(mode)
	if !strings.Contains(snapshot, text) {
		return fmt.Errorf("accessibility snapshot %q does not contain %q", snapshot, text)
	}
	return nil
}

// SendKeys dispatches a deterministic sequence of key events. It stops at the
// first unhandled key and returns its index, or len(keys) on success.
func (t *Terminal) SendKeys(keys []backend.KeyEvent, handler func(backend.KeyEvent) bool) (handled int) {
	if handler == nil {
		return 0
	}
	for i, key := range keys {
		if !handler(key) {
			return i
		}
	}
	return len(keys)
}

// ClickAt dispatches a deterministic click using the supplied timestamp.
func (t *Terminal) ClickAt(x, y uint16, at time.Time) bool {
	if t == nil || t.frame == nil {
		return false
	}
	return t.frame.DispatchClick(backend.MouseEvent{X: x, Y: y, Button: backend.MouseLeft}, at)
}

// SendKey delivers a deterministic key event to a caller-provided handler.
// It is useful for testing application-level key routing without a TTY.
func (t *Terminal) SendKey(key backend.KeyEvent, handler func(backend.KeyEvent) bool) bool {
	if t == nil || handler == nil {
		return false
	}
	return handler(key)
}

// ResizeEvent returns a backend resize event for the current test surface.
func (t *Terminal) ResizeEvent() backend.Event {
	area := t.Area()
	return backend.Event{Type: backend.EventResize, Resize: backend.ResizeEvent{Width: area.Width, Height: area.Height}}
}

// LayerIDs returns the registered layer IDs in frame order.
func (t *Terminal) LayerIDs() []string {
	if t == nil || t.frame == nil {
		return nil
	}
	ids := make([]string, len(t.frame.Layers))
	for i, layer := range t.frame.Layers {
		ids[i] = layer.ID
	}
	return ids
}

// HasModal reports whether the current frame has an active modal layer.
func (t *Terminal) HasModal() bool {
	return t != nil && t.frame != nil && t.frame.TopmostModal() != nil
}

// AssertFocused verifies the current focus owner.
func (t *Terminal) AssertFocused(id string) error {
	if got := t.Focused(); got != id {
		return fmt.Errorf("focused widget = %q, want %q", got, id)
	}
	return nil
}

// AssertLayerZIndex verifies a registered layer's z-index.
func (t *Terminal) AssertLayerZIndex(id string, zIndex int) error {
	if t == nil || t.frame == nil {
		return fmt.Errorf("layer assertion: terminal is nil")
	}
	for _, layer := range t.frame.Layers {
		if layer.ID == id {
			if layer.ZIndex != zIndex {
				return fmt.Errorf("layer %q z-index = %d, want %d", id, layer.ZIndex, zIndex)
			}
			return nil
		}
	}
	return fmt.Errorf("layer %q not found", id)
}

// AssertModalIsolation verifies that a modal exists and is the topmost layer.
func (t *Terminal) AssertModalIsolation(id string) error {
	if t == nil || t.frame == nil {
		return fmt.Errorf("modal assertion: terminal is nil")
	}
	top := t.frame.TopmostModal()
	if top == nil || top.ID != id {
		return fmt.Errorf("topmost modal = %q, want %q", func() string {
			if top == nil {
				return ""
			}
			return top.ID
		}(), id)
	}
	return nil
}

// AssertEventTrace compares the deterministic propagation/hover trace.
func (t *Terminal) AssertEventTrace(expected ...string) error {
	if t == nil || t.frame == nil {
		return fmt.Errorf("event trace: terminal is nil")
	}
	actual := t.frame.EventTrace()
	if len(actual) != len(expected) {
		return fmt.Errorf("event trace = %v, want %v", actual, expected)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return fmt.Errorf("event trace = %v, want %v", actual, expected)
		}
	}
	return nil
}
