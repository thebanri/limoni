package cell

import (
	"image"

	"github.com/thebanri/limoni/core/driver"
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

	// Hyperlinks reports whether the terminal can show OSC 8 links. A widget
	// that renders link markup uses it to decide whether the URL still has to
	// be written out as text: with hyperlinks the label alone is clickable,
	// without them the reader would otherwise have no way to see the address.
	//
	// Its position in this struct is not cosmetic. Context must stay at or
	// under 128 bytes: above that, a closure capturing it captures by
	// reference instead of by value, which moves the whole Context to the
	// heap — one allocation per widget per frame, in every widget with a
	// fallback click closure. Declared here it lands in the padding after
	// Style and costs nothing; declared after FocusedID it cost 144 B/op
	// across the entire widget catalogue.
	Hyperlinks bool

	// RegisterClick is a callback bridge populated by the terminal layer
	// allowing widgets to register clickable regions during rendering.
	//
	// A closure built during Draw is a heap allocation every frame. For the
	// common cases — focus the widget, toggle a flag, select an item — use
	// RegisterClickAction instead, which allocates nothing.
	RegisterClick func(area Rect, handler func())

	// RegisterClickAction registers what a left click in area does, as data
	// rather than as a closure, so interactive widgets draw without
	// allocating. The frame copies the action; nothing in it needs to outlive
	// the call except the pointers it holds.
	RegisterClickAction func(area Rect, action ClickAction)

	// RegisterScroll makes the mouse wheel over area move *offset by one per
	// notch, kept within [0, max]. It is the allocation-free form of the
	// wheel handler scrolling widgets register.
	RegisterScroll func(area Rect, offset *int, max int)

	// RegisterMouse allows widgets to capture drag and advanced mouse events.
	RegisterMouse func(area Rect, handler func(ev driver.MouseEvent))

	// RegisterEvent registers a capture/target/bubble propagation handler.
	RegisterEvent func(area Rect, phase driver.EventPhase, handler func(*driver.EventContext))

	// CaptureMouse allows widgets to temporarily lock mouse input exclusively.
	CaptureMouse func(handler func(ev driver.MouseEvent))

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

	// Describe registers a nested widget in the semantic tree. The frame
	// registers the widgets it is asked to render; a container that draws a
	// child itself — Block's Child, a component tree — calls Describe after
	// drawing it, or the child is invisible to screen readers, tests and
	// agents. w is the child; if it provides no semantic node, nothing
	// happens. Passing an interface value does not allocate.
	Describe func(w any, area Rect)
}

// ClickAction describes what a left click does, without a closure. Every
// field is optional and they combine: a list row focuses its list and selects
// itself, a checkbox focuses and toggles.
type ClickAction struct {
	// Focus moves focus to the widget with this ID.
	Focus string
	// Toggle flips *Toggle.
	Toggle *bool
	// Select sets *Select to Index.
	Select *int
	Index  int
	// Assign sets *Assign to Value, as a radio button does.
	Assign *string
	Value  string
	// Pointer is the mouse pointer shown over the area, by CSS cursor name:
	// "pointer" for a link, "ew-resize" for a divider. Terminals that cannot
	// change the pointer ignore it. An action with only Pointer set does
	// nothing when clicked.
	Pointer string
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

	if other.Link != 0 {
		merged.Link = other.Link
	}

	if other.Fg.Type() != ColorDefault {
		merged.Fg = other.Fg
	}

	if other.Bg.Type() != ColorDefault {
		merged.Bg = other.Bg
	}

	merged.Modifier |= other.Modifier

	return merged
}
