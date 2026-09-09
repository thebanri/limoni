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

// BorderEdges specifies which sides of a border to render.
type BorderEdges = component.BorderEdges

const (
	BorderEdgeTop        = component.BorderEdgeTop
	BorderEdgeRight      = component.BorderEdgeRight
	BorderEdgeBottom     = component.BorderEdgeBottom
	BorderEdgeLeft       = component.BorderEdgeLeft
	BorderEdgeAll        = component.BorderEdgeAll
	BorderEdgeHorizontal = component.BorderEdgeHorizontal
	BorderEdgeVertical   = component.BorderEdgeVertical
)

// Border wraps any component with a decorative 4-sided border.
func Border(child Component, symbols widgets.BorderSymbols, style Style) Component {
	return component.Border(child, symbols, style)
}

// BorderCustom wraps any component with selective border edges.
func BorderCustom(child Component, symbols widgets.BorderSymbols, style Style, edges BorderEdges) Component {
	return component.BorderCustom(child, symbols, style, edges)
}

// TopBorder wraps a component with a single top border rule.
func TopBorder(child Component, symbol rune, style Style) Component {
	return component.TopBorder(child, symbol, style)
}

// BottomBorder wraps a component with a single bottom border rule (e.g. tab underline).
func BottomBorder(child Component, symbol rune, style Style) Component {
	return component.BottomBorder(child, symbol, style)
}

// Margin adds outer spacing around any component (outside any border).
func Margin(child Component, top, right, bottom, left uint16) Component {
	return component.Margin(child, top, right, bottom, left)
}

// MarginAll adds uniform outer spacing on all 4 sides.
func MarginAll(child Component, m uint16) Component {
	return component.MarginAll(child, m)
}

// MarginAxis adds symmetric horizontal and vertical outer spacing.
func MarginAxis(child Component, horizontal, vertical uint16) Component {
	return component.MarginAxis(child, horizontal, vertical)
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

// Constrain enforces minimum and maximum width and height bounds on a child component.
func Constrain(minW, maxW, minH, maxH uint16, child Component) Component {
	return component.Constrain(minW, maxW, minH, maxH, child)
}

// MaxWidth clamps the maximum width of a child component.
func MaxWidth(maxW uint16, child Component) Component {
	return component.MaxWidth(maxW, child)
}

// MaxHeight clamps the maximum height of a child component.
func MaxHeight(maxH uint16, child Component) Component {
	return component.MaxHeight(maxH, child)
}

// MinWidth ensures a child component takes at least minW width.
func MinWidth(minW uint16, child Component) Component {
	return component.MinWidth(minW, child)
}

// MinHeight ensures a child component takes at least minH height.
func MinHeight(minH uint16, child Component) Component {
	return component.MinHeight(minH, child)
}

// Place positions a child component inside a fixed bounding box of (width x height)
// aligned horizontally and vertically according to hAlign and vAlign.
// Equivalent to Lipgloss's lipgloss.Place().
func Place(width, height uint16, hAlign HAlign, vAlign VAlign, child Component) Component {
	return component.Place(width, height, hAlign, vAlign, child)
}

// Spacer returns an expanding, invisible flexible component.
// In an HStack, it expands horizontally pushing adjacent items apart.
// In a VStack, it expands vertically.
func Spacer(weight ...uint16) Component {
	return component.Spacer(weight...)
}

// Divider creates a horizontal rule filling 100% of the available width with 1 row height.
func Divider(style ...Style) Component {
	return component.Divider(style...)
}

// DividerWithTitle creates a horizontal rule with a title centered in the divider.
func DividerWithTitle(title string, symbols widgets.BorderSymbols, style Style) Component {
	return component.DividerWithTitle(title, symbols, style)
}

// VDivider creates a vertical rule filling 100% of the available height with 1 column width.
func VDivider(style ...Style) Component {
	return component.VDivider(style...)
}

// ForEach maps a slice of items of type T to a slice of Components using the provided mapping function.
// The resulting slice can be directly spread into container layouts such as VStack(...) or HStack(...).
func ForEach[T any](items []T, fn func(item T, index int) Component) []Component {
	return component.ForEach(items, fn)
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

// Dynamic constructs a reactive component whose child subtree is resolved at render time.
func Dynamic(supplier func() Component) Component {
	return component.Dynamic(supplier)
}

// TextRef creates a dynamic, data-driven text component bound directly to a string variable pointer.
func TextRef(ptr *string, style ...Style) Component {
	return component.TextRef(ptr, style...)
}

// TextFn creates a dynamic text component evaluated via a getter function on each frame.
func TextFn(getter func() string, style ...Style) Component {
	return component.TextFn(getter, style...)
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

// OnKey binds a specific keyboard key to a component.
func OnKey(child Component, key driver.KeyType, handler func(ev driver.KeyEvent) bool) InteractiveComponent {
	return component.OnKey(child, key, handler)
}

// OnRune binds a specific character rune to a component.
func OnRune(child Component, r rune, handler func(ev driver.KeyEvent) bool) InteractiveComponent {
	return component.OnRune(child, r, handler)
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
