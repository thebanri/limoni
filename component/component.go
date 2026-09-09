package component

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
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

// HAlign defines horizontal alignment (compatible with widgets.Alignment).
type HAlign = widgets.Alignment

const (
	AlignLeft   = widgets.AlignLeft
	AlignCenter = widgets.AlignCenter
	AlignRight  = widgets.AlignRight
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

// --- PADDING & MARGIN (BOX MODEL) ---

type padComponent struct {
	child                    Component
	top, right, bottom, left uint16
}

// Pad adds inner spacing around any component (inside any border).
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

func (p *padComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	area := ctx.Area
	hPad := p.left + p.right
	vPad := p.top + p.bottom

	if area.Width <= hPad || area.Height <= vPad {
		return false
	}

	innerArea := cell.Rect{
		X:      area.X + p.left,
		Y:      area.Y + p.top,
		Width:  area.Width - hPad,
		Height: area.Height - vPad,
	}

	if ev != nil && ev.Type == driver.EventMouse && !innerArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}

	ctx.Area = innerArea
	return DispatchEvent(p.child, ctx, ev)
}

type marginComponent struct {
	child                    Component
	top, right, bottom, left uint16
}

// Margin adds outer spacing around any component (outside any border).
func Margin(child Component, top, right, bottom, left uint16) Component {
	return &marginComponent{
		child:  child,
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

// MarginAll adds uniform outer spacing on all 4 sides.
func MarginAll(child Component, m uint16) Component {
	return Margin(child, m, m, m, m)
}

// MarginAxis adds symmetric horizontal and vertical outer spacing.
func MarginAxis(child Component, horizontal, vertical uint16) Component {
	return Margin(child, vertical, horizontal, vertical, horizontal)
}

func (m *marginComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	hMargin := m.left + m.right
	vMargin := m.top + m.bottom

	if area.Width <= hMargin || area.Height <= vMargin {
		return
	}

	innerArea := cell.Rect{
		X:      area.X + m.left,
		Y:      area.Y + m.top,
		Width:  area.Width - hMargin,
		Height: area.Height - vMargin,
	}

	ctx.Area = innerArea
	m.child.Draw(ctx, buf)
}

func (m *marginComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	hMargin := m.left + m.right
	vMargin := m.top + m.bottom

	innerMax := maxArea
	if innerMax.Width > hMargin {
		innerMax.Width -= hMargin
	} else {
		innerMax.Width = 0
	}
	if innerMax.Height > vMargin {
		innerMax.Height -= vMargin
	} else {
		innerMax.Height = 0
	}

	props := m.child.LayoutInfo(innerMax)
	props.MinWidth += hMargin
	props.MinHeight += vMargin
	if props.MaxWidth > 0 {
		props.MaxWidth += hMargin
	}
	if props.MaxHeight > 0 {
		props.MaxHeight += vMargin
	}
	return props
}

func (m *marginComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := m.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

func (m *marginComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	area := ctx.Area
	hMargin := m.left + m.right
	vMargin := m.top + m.bottom

	if area.Width <= hMargin || area.Height <= vMargin {
		return false
	}

	innerArea := cell.Rect{
		X:      area.X + m.left,
		Y:      area.Y + m.top,
		Width:  area.Width - hMargin,
		Height: area.Height - vMargin,
	}

	if ev != nil && ev.Type == driver.EventMouse && !innerArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}

	ctx.Area = innerArea
	return DispatchEvent(m.child, ctx, ev)
}

// --- BORDER WITH SELECTIVE EDGES ---

// BorderEdges specifies which sides of a border to render.
type BorderEdges uint8

const (
	BorderEdgeTop BorderEdges = 1 << iota
	BorderEdgeRight
	BorderEdgeBottom
	BorderEdgeLeft

	BorderEdgeAll        = BorderEdgeTop | BorderEdgeRight | BorderEdgeBottom | BorderEdgeLeft
	BorderEdgeHorizontal = BorderEdgeTop | BorderEdgeBottom
	BorderEdgeVertical   = BorderEdgeLeft | BorderEdgeRight
)

type borderComponent struct {
	child   Component
	symbols widgets.BorderSymbols
	style   cell.Style
	edges   BorderEdges
}

// Border wraps any component with a decorative 4-sided border.
func Border(child Component, symbols widgets.BorderSymbols, style cell.Style) Component {
	return BorderCustom(child, symbols, style, BorderEdgeAll)
}

// BorderCustom wraps any component with selective border edges (e.g. only top, bottom, or sides).
func BorderCustom(child Component, symbols widgets.BorderSymbols, style cell.Style, edges BorderEdges) Component {
	if edges == 0 {
		edges = BorderEdgeAll
	}
	return &borderComponent{
		child:   child,
		symbols: symbols,
		style:   style,
		edges:   edges,
	}
}

// TopBorder wraps a component with a single top border rule.
func TopBorder(child Component, symbol rune, style cell.Style) Component {
	syms := widgets.BorderSymbols{Horizontal: symbol}
	return BorderCustom(child, syms, style, BorderEdgeTop)
}

// BottomBorder wraps a component with a single bottom border rule (e.g. tab underline).
func BottomBorder(child Component, symbol rune, style cell.Style) Component {
	syms := widgets.BorderSymbols{Horizontal: symbol}
	return BorderCustom(child, syms, style, BorderEdgeBottom)
}

func (b *borderComponent) paddings() (top, right, bottom, left uint16) {
	if b.edges&BorderEdgeTop != 0 {
		top = 1
	}
	if b.edges&BorderEdgeRight != 0 {
		right = 1
	}
	if b.edges&BorderEdgeBottom != 0 {
		bottom = 1
	}
	if b.edges&BorderEdgeLeft != 0 {
		left = 1
	}
	return
}

func (b *borderComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	padTop, padRight, padBottom, padLeft := b.paddings()
	hPad := padLeft + padRight
	vPad := padTop + padBottom

	if area.Width < hPad || area.Height < vPad {
		return
	}

	finalStyle := ctx.Style.Merge(b.style)
	maxX := area.X + area.Width - 1
	maxY := area.Y + area.Height - 1

	hasTop := b.edges&BorderEdgeTop != 0
	hasBottom := b.edges&BorderEdgeBottom != 0
	hasLeft := b.edges&BorderEdgeLeft != 0
	hasRight := b.edges&BorderEdgeRight != 0

	// Draw top edge
	if hasTop && area.Height > 0 {
		for x := area.X; x <= maxX; x++ {
			var ch rune
			if x == area.X && hasLeft {
				ch = b.symbols.TopLeft
			} else if x == maxX && hasRight {
				ch = b.symbols.TopRight
			} else {
				ch = b.symbols.Horizontal
			}
			buf.SetCell(x, area.Y, cell.Cell{Content: ch, Style: finalStyle})
		}
	}

	// Draw bottom edge
	if hasBottom && area.Height > 0 {
		for x := area.X; x <= maxX; x++ {
			var ch rune
			if x == area.X && hasLeft {
				ch = b.symbols.BottomLeft
			} else if x == maxX && hasRight {
				ch = b.symbols.BottomRight
			} else {
				ch = b.symbols.Horizontal
			}
			buf.SetCell(x, maxY, cell.Cell{Content: ch, Style: finalStyle})
		}
	}

	// Draw left edge
	if hasLeft && area.Width > 0 {
		startY := area.Y + padTop
		endY := maxY - padBottom
		for y := startY; y <= endY; y++ {
			buf.SetCell(area.X, y, cell.Cell{Content: b.symbols.Vertical, Style: finalStyle})
		}
	}

	// Draw right edge
	if hasRight && area.Width > 0 {
		startY := area.Y + padTop
		endY := maxY - padBottom
		for y := startY; y <= endY; y++ {
			buf.SetCell(maxX, y, cell.Cell{Content: b.symbols.Vertical, Style: finalStyle})
		}
	}

	// Render child within inner bounds
	if area.Width > hPad && area.Height > vPad {
		innerCtx := ctx
		innerCtx.Area = cell.Rect{
			X:      area.X + padLeft,
			Y:      area.Y + padTop,
			Width:  area.Width - hPad,
			Height: area.Height - vPad,
		}
		b.child.Draw(innerCtx, buf)
	}
}

func (b *borderComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	padTop, padRight, padBottom, padLeft := b.paddings()
	hPad := padLeft + padRight
	vPad := padTop + padBottom

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

	props := b.child.LayoutInfo(innerMax)
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

func (b *borderComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := b.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

func (b *borderComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	padTop, padRight, padBottom, padLeft := b.paddings()
	hPad := padLeft + padRight
	vPad := padTop + padBottom

	if ctx.Area.Width <= hPad || ctx.Area.Height <= vPad {
		return false
	}

	innerArea := cell.Rect{
		X:      ctx.Area.X + padLeft,
		Y:      ctx.Area.Y + padTop,
		Width:  ctx.Area.Width - hPad,
		Height: ctx.Area.Height - vPad,
	}

	if ev != nil && ev.Type == driver.EventMouse && !innerArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}

	ctx.Area = innerArea
	return DispatchEvent(b.child, ctx, ev)
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

func (a *alignComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return false
	}

	props := a.child.LayoutInfo(area)
	targetW := props.MinWidth
	targetH := props.MinHeight
	if targetW == 0 || targetW > area.Width {
		targetW = area.Width
	}
	if targetH == 0 || targetH > area.Height {
		targetH = area.Height
	}

	var posX, posY uint16
	switch a.hAlign {
	case AlignCenter:
		posX = area.X + (area.Width-targetW)/2
	case AlignRight:
		posX = area.X + area.Width - targetW
	default: // AlignLeft
		posX = area.X
	}

	switch a.vAlign {
	case AlignMiddle:
		posY = area.Y + (area.Height-targetH)/2
	case AlignBottom:
		posY = area.Y + area.Height - targetH
	default: // AlignTop
		posY = area.Y
	}

	innerArea := cell.Rect{
		X:      posX,
		Y:      posY,
		Width:  targetW,
		Height: targetH,
	}

	if ev != nil && ev.Type == driver.EventMouse && !innerArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}

	ctx.Area = innerArea
	return DispatchEvent(a.child, ctx, ev)
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

func (f *flexComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return DispatchEvent(f.child, ctx, ev)
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

func (fc *fixedComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return DispatchEvent(fc.child, ctx, ev)
}

// ---------------------------------------------------------------------
// 4. Adapters & Lightweight Primitives
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

func (a *WidgetAdapter) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	if ev != nil && ev.Type == driver.EventMouse && !ctx.Area.Contains(ev.Mouse.X, ev.Mouse.Y) {
		return false
	}
	if a.w != nil {
		if inter, ok := a.w.(interface {
			HandleEvent(cell.Context, *driver.Event) bool
		}); ok {
			return inter.HandleEvent(ctx, ev)
		}
	}
	return false
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
	finalStyle := ctx.Style.Merge(t.style)
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
				Style:   finalStyle,
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

func (t *textComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}
