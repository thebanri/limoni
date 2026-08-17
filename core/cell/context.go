package cell

import (
	"image"

	"github.com/thebanri/limoni/core/backend"
)

// Context represents the stack-allocated drawing context passed down to widgets.
// Because it is passed by value, it generates zero heap allocations and keeps memory footprint minimal.
// Child widgets automatically inherit clipping boundaries and cascading style properties from parents.
type Context struct {
	// Area specifies the bounding box within which the widget should draw.
	// Sub-components must not draw outside of this rect.
	Area Rect

	// Style carries cascading styles (colors, modifiers) inherited from parent containers.
	Style Style

	// RegisterClick is a callback bridge populated by the terminal layer
	// allowing widgets to register clickable regions during rendering.
	RegisterClick func(area Rect, handler func())

	// RegisterMouse allows widgets to capture drag and advanced mouse events.
	RegisterMouse func(area Rect, handler func(ev backend.MouseEvent))

	// RegisterEvent registers a capture/target/bubble propagation handler.
	RegisterEvent func(area Rect, phase backend.EventPhase, handler func(*backend.EventContext))

	// CaptureMouse allows widgets to temporarily lock mouse input exclusively.
	CaptureMouse func(handler func(ev backend.MouseEvent))

	// RegisterImage allows widgets to register image rendering requests during the draw pass.
	RegisterImage func(area Rect, img image.Image, zIndex int, transparent bool) bool

	// RegisterFocus registers the widget ID with the focus manager during rendering.
	RegisterFocus func(id string)

	// SetFocus programmatically shifts focus to the target widget ID.
	SetFocus func(id string)

	// FocusedID holds the ID of the currently focused widget.
	FocusedID string

	// ThemeStyle resolves a semantic theme role into a style inherited from the frame.
	ThemeStyle func(role string) Style
}

// IsFocused reports whether the requested widget ID owns the current focus.
func (c Context) IsFocused(id string) bool { return id != "" && c.FocusedID == id }

// NewContext creates and returns a new Context instance.
func NewContext(area Rect, style Style) Context {
	return Context{
		Area:  area,
		Style: style,
	}
}

// Merge combines two styles according to cascading rules and returns a new Style.
// Non-default properties in 'other' override the base style.
// Modifiers (Bold, Italic, etc.) are combined using a bitwise OR operation.
func (s Style) Merge(other Style) Style {
	merged := s

	if other.Fg.Type() != ColorDefault {
		merged.Fg = other.Fg
	}

	if other.Bg.Type() != ColorDefault {
		merged.Bg = other.Bg
	}

	merged.Modifier |= other.Modifier

	return merged
}
