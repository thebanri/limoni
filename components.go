package limoni

import (
	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/widgets"
)

// ---------------------------------------------------------------------
// Composable Component Types & Aliases
// ---------------------------------------------------------------------

// Component is the minimal, unified interface for all composable UI elements.
type Component = component.Component

// LayoutProps defines sizing and layout constraints for composable negotiation.
type LayoutProps = component.LayoutProps

// StackLayout arranges components linearly along a primary axis.
type StackLayout = component.StackLayout

// Composable alignment aliases
type HAlign = component.HAlign
type VAlign = component.VAlign

const (
	AlignTop    = component.AlignTop
	AlignMiddle = component.AlignMiddle
	AlignBottom = component.AlignBottom
)

// ---------------------------------------------------------------------
// Composable Decorators & Wrappers (Lego Architecture)
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
// Composable Layout Primitives
// ---------------------------------------------------------------------

// VStack creates a vertical stack arranging children top-to-bottom.
func VStack(children ...Component) *StackLayout {
	return component.VStack(children...)
}

// HStack creates a horizontal stack arranging children left-to-right.
func HStack(children ...Component) *StackLayout {
	return component.HStack(children...)
}

// ---------------------------------------------------------------------
// Adapters & Lightweight Primitives
// ---------------------------------------------------------------------

// AsComponent wraps an existing widgets.Widget to satisfy the Component interface.
func AsComponent(w widgets.Widget) Component {
	return component.AsComponent(w)
}

// Label creates an ultra-lightweight inline text component.
func Label(content string, style ...Style) Component {
	return component.Text(content, style...)
}
