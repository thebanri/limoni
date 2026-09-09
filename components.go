package limoni

import (
	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// ---------------------------------------------------------------------
// 1. Composable Component Types & Aliases
// ---------------------------------------------------------------------

// Component is the minimal, unified interface for all composable UI elements.
type Component = component.Component

// LayoutProps defines sizing and layout constraints for composable negotiation.
type LayoutProps = component.LayoutProps

// StackLayout arranges components linearly along a primary axis.
type StackLayout = component.StackLayout

// ZStackLayout arranges components in depth layers (Painter's algorithm).
type ZStackLayout = component.ZStackLayout

// InteractiveComponent represents a composable element capable of handling input and mouse events.
type InteractiveComponent = component.InteractiveComponent

// Composable alignment aliases
type HAlign = component.HAlign
type VAlign = component.VAlign

const (
	HAlignLeft   = component.AlignLeft
	HAlignCenter = component.AlignCenter
	HAlignRight  = component.AlignRight

	AlignTop    = component.AlignTop
	AlignMiddle = component.AlignMiddle
	AlignBottom = component.AlignBottom
)

// Composable Flexbox distribution & alignment
type JustifyContent = component.JustifyContent

const (
	JustifyStart        = component.JustifyStart
	JustifyCenter       = component.JustifyCenter
	JustifyEnd          = component.JustifyEnd
	JustifySpaceBetween = component.JustifySpaceBetween
	JustifySpaceAround  = component.JustifySpaceAround
	JustifySpaceEvenly  = component.JustifySpaceEvenly
)

type AlignItems = component.AlignItems

const (
	AlignItemsStretch = component.AlignItemsStretch
	AlignItemsStart   = component.AlignItemsStart
	AlignItemsCenter  = component.AlignItemsCenter
	AlignItemsEnd     = component.AlignItemsEnd
)

// ---------------------------------------------------------------------
// 2. Composable Layout Primitives
// ---------------------------------------------------------------------

// VStack creates a vertical stack arranging children top-to-bottom.
func VStack(children ...Component) *StackLayout {
	return component.VStack(children...)
}

// HStack creates a horizontal stack arranging children left-to-right.
func HStack(children ...Component) *StackLayout {
	return component.HStack(children...)
}

// ZStack creates a depth-axis container rendering from background to foreground.
// All children share the same bounding area without offscreen buffer allocations.
func ZStack(children ...Component) *ZStackLayout {
	return component.ZStack(children...)
}

// ---------------------------------------------------------------------
// 3. Composable Decorators & Wrappers (Lego Architecture)
// ---------------------------------------------------------------------

// Pad adds inner spacing around any component.
func Pad(child Component, top, right, bottom, left uint16) Component {
	return component.Pad(child, top, right, bottom, left)
}

// PadAll adds uniform padding on all 4 sides of a component.
func PadAll(child Component, padding uint16) Component {
	return component.PadAll(child, padding)
}

// PadAxis adds symmetric horizontal and vertical padding.
func PadAxis(child Component, horizontal, vertical uint16) Component {
	return component.PadAxis(child, horizontal, vertical)
}

// Border wraps any component with a decorative border.
func Border(child Component, symbols widgets.BorderSymbols, style Style) Component {
	return component.Border(child, symbols, style)
}

// AlignComponent aligns a component within its allocated area according to horizontal and vertical rules.
func AlignComponent(child Component, h HAlign, v VAlign) Component {
	return component.Align(child, h, v)
}

// Center centers a component both horizontally and vertically within its allocated area.
func Center(child Component) Component {
	return component.Center(child)
}

// Flex sets the expansion weight of a child component inside a StackLayout (VStack / HStack).
func Flex(weight uint16, child Component) Component {
	return component.Flex(weight, child)
}

// FixedSize forces a fixed width and height onto a child component.
// (Named FixedSize to avoid collision with layout.Fixed constraints).
func FixedSize(width, height uint16, child Component) Component {
	return component.Fixed(width, height, child)
}

// ---------------------------------------------------------------------
// 4. Declarative Conditional & Dynamic Views
// ---------------------------------------------------------------------

// Empty returns a zero-sized no-op component.
func Empty() Component {
	return component.Empty()
}

// When renders then component if condition is true, otherwise the optional otherwise component (or Empty()).
func When(condition bool, then Component, otherwise ...Component) Component {
	return component.When(condition, then, otherwise...)
}

// Match inspects value against cases map and returns the matching component, or defaultCase (or Empty()).
func Match[T comparable](value T, cases map[T]Component, defaultCase ...Component) Component {
	return component.Match(value, cases, defaultCase...)
}

// ---------------------------------------------------------------------
// 5. Style Context Cascading
// ---------------------------------------------------------------------

// WithStyle cascades a Style into the component subtree by merging it into Context.Style.
func WithStyle(style Style, child Component) Component {
	return component.WithStyle(style, child)
}

// WithForeground cascades a foreground color override into the component subtree.
func WithForeground(color Color, child Component) Component {
	return component.WithForeground(color, child)
}

// WithBackground cascades a background color override into the component subtree.
func WithBackground(color Color, child Component) Component {
	return component.WithBackground(color, child)
}

// ---------------------------------------------------------------------
// 6. Interactive Event Propagation & Local Hit-Testing
// ---------------------------------------------------------------------

// OnClick wraps any component with a click handler.
func OnClick(child Component, handler func(ev MouseEvent)) InteractiveComponent {
	return component.OnClick(child, handler)
}

// OnEvent attaches an arbitrary event listener (keyboard, mouse, resize, paste) to a component.
func OnEvent(child Component, handler func(ctx cell.Context, ev *driver.Event) bool) InteractiveComponent {
	return component.OnEvent(child, handler)
}

// DispatchEvent routes an event down a component tree starting from root within ctx.Area.
func DispatchEvent(root Component, ctx cell.Context, ev *driver.Event) bool {
	return component.DispatchEvent(root, ctx, ev)
}

// ---------------------------------------------------------------------
// 7. Adapters & Lightweight Primitives
// ---------------------------------------------------------------------

// AsComponent wraps an existing widgets.Widget to satisfy the Component interface.
func AsComponent(w widgets.Widget) Component {
	return component.AsComponent(w)
}

// Label creates an ultra-lightweight inline text component.
func Label(content string, style ...Style) Component {
	return component.Text(content, style...)
}
