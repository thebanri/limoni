package component

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

// ---------------------------------------------------------------------
// 1. Core Interfaces and Layout Properties
// ---------------------------------------------------------------------

// LayoutProps defines sizing and layout constraints for composable negotiation.
type LayoutProps struct {
	MinWidth  uint16
	MinHeight uint16
	MaxWidth  uint16
	MaxHeight uint16
	Flex      uint16 // 0: Content/fixed size, >0: Flexible expansion weight
}

// Component is the minimal, unified interface for all composable UI elements.
// It satisfies widgets.Widget natively via Draw and SizeHint.
type Component interface {
	// Draw renders the component into the allocated ctx.Area of the 1D contiguous buffer.
	Draw(ctx cell.Context, buf *buffer.Buffer)

	// LayoutInfo returns the component's layout properties and constraints.
	LayoutInfo(maxArea cell.Rect) LayoutProps

	// SizeHint satisfies widgets.Widget, allowing direct interoperability with existing APIs.
	SizeHint(maxArea cell.Rect) (width, height uint16)
}

// ---------------------------------------------------------------------
// 2. Alignment Types
// ---------------------------------------------------------------------

// HAlign defines horizontal alignment.
type HAlign uint8

const (
	AlignLeft HAlign = iota
	AlignCenter
	AlignRight
)

// VAlign defines vertical alignment.
type VAlign uint8

const (
	AlignTop VAlign = iota
	AlignMiddle
	AlignBottom
)

// ---------------------------------------------------------------------
// 3. Decorator / Wrapper Components (Lego Architecture)
// ---------------------------------------------------------------------

// --- PADDING ---

type padComponent struct {
	child                    Component
	top, right, bottom, left uint16
}

// Pad adds inner spacing around any component.
func Pad(child Component, top, right, bottom, left uint16) Component {
	return &padComponent{
		child:  child,
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

// PadAll adds uniform padding on all 4 sides.
func PadAll(child Component, padding uint16) Component {
	return Pad(child, padding, padding, padding, padding)
}

// PadAxis adds symmetric horizontal and vertical padding.
func PadAxis(child Component, horizontal, vertical uint16) Component {
	return Pad(child, vertical, horizontal, vertical, horizontal)
}

func (p *padComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	hPad := p.left + p.right
	vPad := p.top + p.bottom

	if area.Width <= hPad || area.Height <= vPad {
		return
	}

	innerArea := cell.Rect{
		X:      area.X + p.left,
		Y:      area.Y + p.top,
		Width:  area.Width - hPad,
		Height: area.Height - vPad,
	}

	// Pass context by value on the stack (zero heap allocation)
	ctx.Area = innerArea
	p.child.Draw(ctx, buf)
}

func (p *padComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	hPad := p.left + p.right
	vPad := p.top + p.bottom

	innerMax := maxArea
	if innerMax.Width > hPad {
		innerMax.Width -= hPad
	} else {
		innerMax.Width = 0
	}
	if innerMax.Height > vPad {
		innerMax.Height -= vPad
	} else {
		innerMax.Height = 0
	}

	props := p.child.LayoutInfo(innerMax)
	props.MinWidth += hPad
	props.MinHeight += vPad
	if props.MaxWidth > 0 {
		props.MaxWidth += hPad
	}
	if props.MaxHeight > 0 {
		props.MaxHeight += vPad
	}
	return props
}

func (p *padComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := p.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// --- BORDER ---

type borderComponent struct {
	child   Component
	symbols widgets.BorderSymbols
	style   cell.Style
}

// Border wraps any component with a decorative border.
func Border(child Component, symbols widgets.BorderSymbols, style cell.Style) Component {
	return &borderComponent{
		child:   child,
		symbols: symbols,
		style:   style,
	}
}

func (b *borderComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 2 || area.Height < 2 {
		return
	}

	maxX := area.X + area.Width - 1
	maxY := area.Y + area.Height - 1

	// Draw corners
	buf.SetCell(area.X, area.Y, cell.Cell{Content: b.symbols.TopLeft, Style: b.style})
	buf.SetCell(maxX, area.Y, cell.Cell{Content: b.symbols.TopRight, Style: b.style})
	buf.SetCell(area.X, maxY, cell.Cell{Content: b.symbols.BottomLeft, Style: b.style})
	buf.SetCell(maxX, maxY, cell.Cell{Content: b.symbols.BottomRight, Style: b.style})

	// Horizontal border segments
	for x := area.X + 1; x < maxX; x++ {
		buf.SetCell(x, area.Y, cell.Cell{Content: b.symbols.Horizontal, Style: b.style})
		buf.SetCell(x, maxY, cell.Cell{Content: b.symbols.Horizontal, Style: b.style})
	}

	// Vertical border segments
	for y := area.Y + 1; y < maxY; y++ {
		buf.SetCell(area.X, y, cell.Cell{Content: b.symbols.Vertical, Style: b.style})
		buf.SetCell(maxX, y, cell.Cell{Content: b.symbols.Vertical, Style: b.style})
	}

	// Render child within inner bounds
	innerCtx := ctx
	innerCtx.Area = cell.Rect{
		X:      area.X + 1,
		Y:      area.Y + 1,
		Width:  area.Width - 2,
		Height: area.Height - 2,
	}
	b.child.Draw(innerCtx, buf)
}

func (b *borderComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	innerMax := maxArea
	if innerMax.Width > 2 {
		innerMax.Width -= 2
	} else {
		innerMax.Width = 0
	}
	if innerMax.Height > 2 {
		innerMax.Height -= 2
	} else {
		innerMax.Height = 0
	}

	props := b.child.LayoutInfo(innerMax)
	props.MinWidth += 2
	props.MinHeight += 2
	if props.MaxWidth > 0 {
		props.MaxWidth += 2
	}
	if props.MaxHeight > 0 {
		props.MaxHeight += 2
	}
	return props
}

func (b *borderComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := b.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// --- ALIGNMENT ---

type alignComponent struct {
	child  Component
	hAlign HAlign
	vAlign VAlign
}

// Align aligns a component within its allocated area according to horizontal and vertical rules.
func Align(child Component, h HAlign, v VAlign) Component {
	return &alignComponent{child: child, hAlign: h, vAlign: v}
}

// Center centers a component both horizontally and vertically.
func Center(child Component) Component {
	return Align(child, AlignCenter, AlignMiddle)
}

func (a *alignComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	props := a.child.LayoutInfo(area)

	targetW := props.MinWidth
	if targetW == 0 || targetW > area.Width {
		targetW = area.Width
	}
	targetH := props.MinHeight
	if targetH == 0 || targetH > area.Height {
		targetH = area.Height
	}

	var posX uint16
	switch a.hAlign {
	case AlignCenter:
		if area.Width > targetW {
			posX = area.X + (area.Width-targetW)/2
		} else {
			posX = area.X
		}
	case AlignRight:
		if area.Width > targetW {
			posX = area.X + (area.Width - targetW)
		} else {
			posX = area.X
		}
	default: // AlignLeft
		posX = area.X
	}

	var posY uint16
	switch a.vAlign {
	case AlignMiddle:
		if area.Height > targetH {
			posY = area.Y + (area.Height-targetH)/2
		} else {
			posY = area.Y
		}
	case AlignBottom:
		if area.Height > targetH {
			posY = area.Y + (area.Height - targetH)
		} else {
			posY = area.Y
		}
	default: // AlignTop
		posY = area.Y
	}

	innerCtx := ctx
	innerCtx.Area = cell.Rect{
		X:      posX,
		Y:      posY,
		Width:  targetW,
		Height: targetH,
	}
	a.child.Draw(innerCtx, buf)
}

func (a *alignComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	props := a.child.LayoutInfo(maxArea)
	// Align components naturally expand to fill available space unless explicitly constrained
	if props.Flex == 0 {
		props.Flex = 1
	}
	return props
}

func (a *alignComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := a.child.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// --- FLEX & FIXED MODIFIERS ---

type flexComponent struct {
	child  Component
	weight uint16
}

// Flex sets the expansion weight of a child component inside a StackLayout (VStack / HStack).
func Flex(weight uint16, child Component) Component {
	if weight == 0 {
		weight = 1
	}
	return &flexComponent{child: child, weight: weight}
}

func (f *flexComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	f.child.Draw(ctx, buf)
}

func (f *flexComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	props := f.child.LayoutInfo(maxArea)
	props.Flex = f.weight
	return props
}

func (f *flexComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return f.child.SizeHint(maxArea)
}

type fixedComponent struct {
	child  Component
	width  uint16
	height uint16
}

// Fixed forces a fixed width and height onto a child component.
func Fixed(width, height uint16, child Component) Component {
	return &fixedComponent{child: child, width: width, height: height}
}

func (fc *fixedComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	fc.child.Draw(ctx, buf)
}

func (fc *fixedComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	props := fc.child.LayoutInfo(maxArea)
	if fc.width > 0 {
		props.MinWidth = fc.width
		props.MaxWidth = fc.width
	}
	if fc.height > 0 {
		props.MinHeight = fc.height
		props.MaxHeight = fc.height
	}
	props.Flex = 0 // Fixed size components do not flex
	return props
}

func (fc *fixedComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return fc.width, fc.height
}

// ---------------------------------------------------------------------
// 4. Composable Layout Primitives (VStack & HStack)
// ---------------------------------------------------------------------

// StackLayout arranges components linearly along a primary axis.
type StackLayout struct {
	isVertical bool
	gap        uint16
	children   []Component
}

// VStack creates a vertical stack arranging children top-to-bottom.
func VStack(children ...Component) *StackLayout {
	return &StackLayout{isVertical: true, gap: 0, children: children}
}

// HStack creates a horizontal stack arranging children left-to-right.
func HStack(children ...Component) *StackLayout {
	return &StackLayout{isVertical: false, gap: 0, children: children}
}

// WithGap sets the cell gap between adjacent children.
func (s *StackLayout) WithGap(gap uint16) *StackLayout {
	s.gap = gap
	return s
}

func (s *StackLayout) Draw(ctx cell.Context, buf *buffer.Buffer) {
	n := len(s.children)
	if n == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Zero-allocation scratch arrays for up to 32 items
	var sizes [32]uint16
	var props [32]LayoutProps

	count := n
	if count > 32 {
		count = 32
	}

	var totalFixed uint16
	var totalFlex uint16

	availPrimary := ctx.Area.Height
	if !s.isVertical {
		availPrimary = ctx.Area.Width
	}

	totalGaps := uint16(count-1) * s.gap
	if availPrimary > totalGaps {
		availPrimary -= totalGaps
	} else {
		availPrimary = 0
	}

	// Measure pass
	for i := 0; i < count; i++ {
		props[i] = s.children[i].LayoutInfo(ctx.Area)
		if props[i].Flex > 0 {
			totalFlex += props[i].Flex
		} else {
			fixedSize := props[i].MinHeight
			if !s.isVertical {
				fixedSize = props[i].MinWidth
			}
			sizes[i] = fixedSize
			totalFixed += fixedSize
		}
	}

	// Flex distribution pass
	var remaining uint16
	if availPrimary > totalFixed {
		remaining = availPrimary - totalFixed
	}

	if totalFlex > 0 && remaining > 0 {
		var allocatedFlex uint16
		var flexItemsCount uint16

		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexItemsCount++
			}
		}

		var flexIndex uint16
		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexIndex++
				var size uint16
				if flexIndex == flexItemsCount {
					// Assign exact remainder to last flex item to prevent rounding gaps
					size = remaining - allocatedFlex
				} else {
					size = (remaining * props[i].Flex) / totalFlex
					allocatedFlex += size
				}
				sizes[i] = size
			}
		}
	}

	// Arrange and Draw pass
	currentOffset := uint16(0)
	for i := 0; i < count; i++ {
		var childArea cell.Rect
		if s.isVertical {
			childH := sizes[i]
			if currentOffset+childH > ctx.Area.Height {
				if currentOffset < ctx.Area.Height {
					childH = ctx.Area.Height - currentOffset
				} else {
					childH = 0
				}
			}
			childArea = cell.Rect{
				X:      ctx.Area.X,
				Y:      ctx.Area.Y + currentOffset,
				Width:  ctx.Area.Width,
				Height: childH,
			}
		} else {
			childW := sizes[i]
			if currentOffset+childW > ctx.Area.Width {
				if currentOffset < ctx.Area.Width {
					childW = ctx.Area.Width - currentOffset
				} else {
					childW = 0
				}
			}
			childArea = cell.Rect{
				X:      ctx.Area.X + currentOffset,
				Y:      ctx.Area.Y,
				Width:  childW,
				Height: ctx.Area.Height,
			}
		}

		if childArea.Width > 0 && childArea.Height > 0 {
			childCtx := ctx
			childCtx.Area = childArea
			s.children[i].Draw(childCtx, buf)
		}

		currentOffset += sizes[i] + s.gap
	}
}

func (s *StackLayout) LayoutInfo(maxArea cell.Rect) LayoutProps {
	var totalW, totalH uint16
	var maxFlex uint16

	for _, c := range s.children {
		p := c.LayoutInfo(maxArea)
		if s.isVertical {
			totalH += p.MinHeight
			if p.MinWidth > totalW {
				totalW = p.MinWidth
			}
		} else {
			totalW += p.MinWidth
			if p.MinHeight > totalH {
				totalH = p.MinHeight
			}
		}
		if p.Flex > maxFlex {
			maxFlex = p.Flex
		}
	}

	if len(s.children) > 1 {
		gapTotal := uint16(len(s.children)-1) * s.gap
		if s.isVertical {
			totalH += gapTotal
		} else {
			totalW += gapTotal
		}
	}

	return LayoutProps{
		MinWidth:  totalW,
		MinHeight: totalH,
		Flex:      maxFlex,
	}
}

func (s *StackLayout) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := s.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// ---------------------------------------------------------------------
// 5. Adapters & Lightweight Primitives
// ---------------------------------------------------------------------

// WidgetAdapter turns any legacy or monolithic widgets.Widget into a Component.
type WidgetAdapter struct {
	w widgets.Widget
}

// AsComponent wraps an existing widgets.Widget to satisfy the Component interface.
func AsComponent(w widgets.Widget) Component {
	if comp, ok := w.(Component); ok {
		return comp
	}
	return &WidgetAdapter{w: w}
}

func (a *WidgetAdapter) Draw(ctx cell.Context, buf *buffer.Buffer) {
	a.w.Draw(ctx, buf)
}

func (a *WidgetAdapter) LayoutInfo(maxArea cell.Rect) LayoutProps {
	w, h := a.w.SizeHint(maxArea)
	return LayoutProps{
		MinWidth:  w,
		MinHeight: h,
		Flex:      1, // By default, monolithic widgets expand to fill given area
	}
}

func (a *WidgetAdapter) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return a.w.SizeHint(maxArea)
}

// --- LIGHTWEIGHT INLINE TEXT ---

type textComponent struct {
	lines []string
	style cell.Style
}

// Text creates an ultra-lightweight inline text component.
func Text(content string, style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	lines := strings.Split(content, "\n")
	return &textComponent{lines: lines, style: st}
}

func (t *textComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	for row, line := range t.lines {
		if uint16(row) >= area.Height {
			break
		}
		col := uint16(0)
		for _, r := range line {
			rw := cell.RuneWidth(r)
			if col+uint16(rw) > area.Width {
				break
			}
			buf.SetCell(area.X+col, area.Y+uint16(row), cell.Cell{
				Content: r,
				Style:   t.style,
			})
			col += uint16(rw)
		}
	}
}

func (t *textComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	var maxW uint16
	for _, l := range t.lines {
		w := uint16(cell.StringWidth(l))
		if w > maxW {
			maxW = w
		}
	}
	return LayoutProps{
		MinWidth:  maxW,
		MinHeight: uint16(len(t.lines)),
		Flex:      0, // Inline text has fixed intrinsic size
	}
}

func (t *textComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := t.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}
