package component_test

import (
	"testing"

	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

func TestPaddingLayoutAndDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 10})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 10}, cell.NewStyle())

	txt := component.Text("Hi")
	padded := component.Pad(txt, 2, 3, 2, 3)

	props := padded.LayoutInfo(ctx.Area)
	if props.MinWidth != 2+6 { // "Hi" (2) + left(3) + right(3) = 8
		t.Fatalf("expected MinWidth 8, got %d", props.MinWidth)
	}
	if props.MinHeight != 1+4 { // 1 line + top(2) + bottom(2) = 5
		t.Fatalf("expected MinHeight 5, got %d", props.MinHeight)
	}

	padded.Draw(ctx, buf)

	// Character 'H' must be at X=3, Y=2
	c := buf.Get(3, 2)
	if c == nil || c.Content != 'H' {
		t.Fatalf("expected 'H' at (3, 2), got %v", c)
	}
	c2 := buf.Get(4, 2)
	if c2 == nil || c2.Content != 'i' {
		t.Fatalf("expected 'i' at (4, 2), got %v", c2)
	}

	// (0, 0) and (2, 2) must be empty space
	if cEmpty := buf.Get(0, 0); cEmpty != nil && cEmpty.Content != ' ' {
		t.Fatalf("expected space at (0, 0), got %q", cEmpty.Content)
	}
}

func TestBorderLayoutAndDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	txt := component.Text("X")
	bordered := component.Border(txt, widgets.SymbolsRounded, cell.NewStyle())

	props := bordered.LayoutInfo(ctx.Area)
	if props.MinWidth != 1+2 {
		t.Fatalf("expected MinWidth 3, got %d", props.MinWidth)
	}
	if props.MinHeight != 1+2 {
		t.Fatalf("expected MinHeight 3, got %d", props.MinHeight)
	}

	bordered.Draw(ctx, buf)

	// Corners for SymbolsRounded: TopLeft '╭', TopRight '╮', BottomLeft '╰', BottomRight '╯'
	if c := buf.Get(0, 0); c == nil || c.Content != '╭' {
		t.Fatalf("expected '╭' at (0,0), got %v", c)
	}
	if c := buf.Get(9, 0); c == nil || c.Content != '╮' {
		t.Fatalf("expected '╮' at (9,0), got %v", c)
	}
	if c := buf.Get(0, 4); c == nil || c.Content != '╰' {
		t.Fatalf("expected '╰' at (0,4), got %v", c)
	}
	if c := buf.Get(9, 4); c == nil || c.Content != '╯' {
		t.Fatalf("expected '╯' at (9,4), got %v", c)
	}

	// Content 'X' should be inside at (1, 1)
	if c := buf.Get(1, 1); c == nil || c.Content != 'X' {
		t.Fatalf("expected 'X' at (1,1), got %v", c)
	}
}

func TestAlignAndCenter(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	txt := component.Text("AB") // 2x1
	centered := component.Center(txt)

	centered.Draw(ctx, buf)

	// In 10x5 area:
	// X = (10 - 2) / 2 = 4
	// Y = (5 - 1) / 2 = 2
	if c := buf.Get(4, 2); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A' at (4, 2), got %v", c)
	}
	if c := buf.Get(5, 2); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B' at (5, 2), got %v", c)
	}
}

func TestVStackDistribution(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 10})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 10}, cell.NewStyle())

	item1 := component.Text("Item 1") // 1 line fixed
	item2 := component.Text("Item 2") // 1 line fixed
	stack := component.VStack(item1, item2).WithGap(1)

	stack.Draw(ctx, buf)

	// Item 1 at Y=0
	if c := buf.Get(0, 0); c == nil || c.Content != 'I' {
		t.Fatalf("expected 'I' at (0,0), got %v", c)
	}
	// Gap at Y=1 is empty
	if c := buf.Get(0, 1); c != nil && c.Content != ' ' {
		t.Fatalf("expected gap at (0,1), got %v", c)
	}
	// Item 2 at Y=2 (Y=0 + 1 + gap=1)
	if c := buf.Get(0, 2); c == nil || c.Content != 'I' {
		t.Fatalf("expected 'I' at (0,2), got %v", c)
	}
}

func TestHStackFlexDistribution(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 5}, cell.NewStyle())

	// 1:3 ratio in a 20-cell width area:
	// left = 20 * 1 / 4 = 5
	// right = 20 - 5 = 15
	left := component.Flex(1, component.Text("L"))
	right := component.Flex(3, component.Text("R"))

	stack := component.HStack(left, right)
	stack.Draw(ctx, buf)

	// Left starts at X=0
	if c := buf.Get(0, 0); c == nil || c.Content != 'L' {
		t.Fatalf("expected 'L' at (0,0), got %v", c)
	}
	// Right starts at X=5
	if c := buf.Get(5, 0); c == nil || c.Content != 'R' {
		t.Fatalf("expected 'R' at (5,0), got %v", c)
	}
}

func TestNestedComposition(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 30, Height: 10})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 30, Height: 10}, cell.NewStyle())

	// VStack:
	// 1. Header with Border(Pad(Text))
	// 2. HStack with two flex panels
	composite := component.VStack(
		component.Fixed(30, 3, component.Border(
			component.Pad(component.Text("HEADER"), 0, 1, 0, 1),
			widgets.SymbolsSingle,
			cell.NewStyle(),
		)),
		component.Flex(1, component.HStack(
			component.Flex(1, component.Text("PANEL A")),
			component.Flex(2, component.Text("PANEL B")),
		)),
	)

	composite.Draw(ctx, buf)

	// Border Top-Left corner at (0,0)
	if c := buf.Get(0, 0); c == nil || c.Content != '┌' {
		t.Fatalf("expected '┌' at (0,0), got %v", c)
	}
	// "HEADER" padded: X=2, Y=1
	if c := buf.Get(2, 1); c == nil || c.Content != 'H' {
		t.Fatalf("expected 'H' at (2,1), got %v", c)
	}
	// Panel A at Y=3, X=0
	if c := buf.Get(0, 3); c == nil || c.Content != 'P' {
		t.Fatalf("expected 'P' at (0,3), got %v", c)
	}
	// Panel B starts at X=10, Y=3 (30 * 1/3 = 10)
	if c := buf.Get(10, 3); c == nil || c.Content != 'P' {
		t.Fatalf("expected 'P' at (10,3), got %v", c)
	}
}

func TestAsComponentAdapter(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 15, Height: 6})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 15, Height: 6}, cell.NewStyle())

	block := widgets.NewBlock().Rounded()
	adapted := component.AsComponent(block)

	props := adapted.LayoutInfo(ctx.Area)
	if props.Flex != 1 {
		t.Fatalf("expected Flex 1 from adapter, got %d", props.Flex)
	}

	adapted.Draw(ctx, buf)

	if c := buf.Get(0, 0); c == nil || c.Content != '╭' {
		t.Fatalf("expected '╭' at (0,0), got %v", c)
	}
}

// ---------------------------------------------------------------------
// Benchmarks: Verifying 0 Heap Allocations on the Hot Path
// ---------------------------------------------------------------------

func BenchmarkVStackDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	stack := component.VStack(
		component.Flex(1, component.Text("Row 1")),
		component.Flex(1, component.Text("Row 2")),
		component.Flex(1, component.Text("Row 3")),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack.Draw(ctx, buf)
	}
}

func BenchmarkBorderDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	bordered := component.Border(
		component.Text("Bordered Content"),
		widgets.SymbolsRounded,
		cell.NewStyle(),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bordered.Draw(ctx, buf)
	}
}

func BenchmarkNestedCompositeDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 120, Height: 40}, cell.NewStyle())

	tree := component.VStack(
		component.Fixed(120, 3, component.Border(
			component.Pad(component.Text("SYSTEM MONITOR"), 0, 1, 0, 1),
			widgets.SymbolsSingle,
			cell.NewStyle(),
		)),
		component.Flex(1, component.HStack(
			component.Flex(1, component.Border(
				component.Text("Navigation"),
				widgets.SymbolsRounded,
				cell.NewStyle(),
			)),
			component.Flex(3, component.Border(
				component.Text("Process Matrix"),
				widgets.SymbolsDouble,
				cell.NewStyle(),
			)),
		)),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Draw(ctx, buf)
	}
}

func TestZStackLayoutAndDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	bg := component.Text("..........")
	fg := component.Center(component.Fixed(2, 1, component.Text("AB")))

	z := component.ZStack(bg, fg)

	props := z.LayoutInfo(ctx.Area)
	if props.MinWidth != 10 || props.MinHeight != 1 {
		t.Fatalf("expected 10x1, got %dx%d", props.MinWidth, props.MinHeight)
	}

	z.Draw(ctx, buf)

	// Foreground 'AB' should be centered at (4,2) and (5,2)
	if c := buf.Get(4, 2); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A' at (4,2), got %v", c)
	}
	if c := buf.Get(5, 2); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B' at (5,2), got %v", c)
	}

	// Background '.' should remain visible at (0,0) and (9,0)
	if c := buf.Get(0, 0); c == nil || c.Content != '.' {
		t.Fatalf("expected '.' at (0,0) from background layer, got %v", c)
	}
	if c := buf.Get(9, 0); c == nil || c.Content != '.' {
		t.Fatalf("expected '.' at (9,0) from background layer, got %v", c)
	}
}

func TestStackJustifyContent(t *testing.T) {
	item1 := component.Fixed(4, 1, component.Text("AAAA"))
	item2 := component.Fixed(4, 1, component.Text("BBBB"))

	// 1. JustifyCenter
	bufCenter := buffer.NewBuffer(cell.Rect{Width: 20, Height: 1})
	ctxCenter := cell.NewContext(cell.Rect{Width: 20, Height: 1}, cell.NewStyle())
	stackCenter := component.HStack(item1, item2).WithJustify(component.JustifyCenter)
	stackCenter.Draw(ctxCenter, bufCenter)

	if c := bufCenter.Get(6, 0); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A' at X=6 for JustifyCenter, got %v", c)
	}
	if c := bufCenter.Get(10, 0); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B' at X=10 for JustifyCenter, got %v", c)
	}

	// 2. JustifySpaceBetween
	bufSB := buffer.NewBuffer(cell.Rect{Width: 20, Height: 1})
	stackSB := component.HStack(item1, item2).WithJustify(component.JustifySpaceBetween)
	stackSB.Draw(ctxCenter, bufSB)

	if c := bufSB.Get(0, 0); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A' at X=0 for JustifySpaceBetween, got %v", c)
	}
	if c := bufSB.Get(16, 0); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B' at X=16 for JustifySpaceBetween, got %v", c)
	}
}

func TestStackAlignItems(t *testing.T) {
	item := component.Fixed(4, 1, component.Text("TEST"))

	// AlignItemsCenter: Y = (5 - 1) / 2 = 2
	bufCenter := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	ctx := cell.NewContext(cell.Rect{Width: 10, Height: 5}, cell.NewStyle())
	stackCenter := component.HStack(item).WithAlignItems(component.AlignItemsCenter)
	stackCenter.Draw(ctx, bufCenter)

	if c := bufCenter.Get(0, 2); c == nil || c.Content != 'T' {
		t.Fatalf("expected 'T' at Y=2 for AlignItemsCenter, got %v", c)
	}

	// AlignItemsEnd: Y = 5 - 1 = 4
	bufEnd := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	stackEnd := component.HStack(item).WithAlignItems(component.AlignItemsEnd)
	stackEnd.Draw(ctx, bufEnd)

	if c := bufEnd.Get(0, 4); c == nil || c.Content != 'T' {
		t.Fatalf("expected 'T' at Y=4 for AlignItemsEnd, got %v", c)
	}
}

func TestConditionalWhenAndMatch(t *testing.T) {
	a := component.Text("A")
	b := component.Text("B")

	if component.When(true, a, b) != a {
		t.Fatalf("expected a for condition true")
	}
	if component.When(false, a, b) != b {
		t.Fatalf("expected b for condition false")
	}
	if component.When(false, a) == nil {
		t.Fatalf("expected empty non-nil component")
	}

	cases := map[string]component.Component{
		"active": a,
		"paused": b,
	}
	if component.Match("active", cases, nil) != a {
		t.Fatalf("expected match active -> a")
	}
	if component.Match("unknown", cases, b) != b {
		t.Fatalf("expected match fallback -> b")
	}
}

func TestStyleCascading(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{Width: 10, Height: 1}, cell.NewStyle())

	fgColor := cell.NewColorRGB(255, 128, 0)
	styled := component.WithForeground(fgColor, component.Text("OK"))
	styled.Draw(ctx, buf)

	c := buf.Get(0, 0)
	if c == nil || c.Content != 'O' || c.Style.Fg != fgColor {
		t.Fatalf("expected 'O' with cascaded foreground color, got %v", c)
	}
}

func TestInteractiveEventRouting(t *testing.T) {
	ctx := cell.NewContext(cell.Rect{X: 10, Y: 10, Width: 20, Height: 5}, cell.NewStyle())

	clicked := false
	button := component.OnClick(component.Text("Click Me"), func(ev driver.MouseEvent) {
		clicked = true
	})

	// Click inside (15, 12)
	evInside := &driver.Event{
		Type:  driver.EventMouse,
		Mouse: driver.MouseEvent{X: 15, Y: 12, Button: driver.MouseLeft},
	}
	if !component.DispatchEvent(button, ctx, evInside) {
		t.Fatalf("expected click to be handled")
	}
	if !clicked {
		t.Fatalf("expected clicked handler to be called")
	}

	// Click outside (5, 5)
	clicked = false
	evOutside := &driver.Event{
		Type:  driver.EventMouse,
		Mouse: driver.MouseEvent{X: 5, Y: 5, Button: driver.MouseLeft},
	}
	if component.DispatchEvent(button, ctx, evOutside) {
		t.Fatalf("expected click outside to NOT be handled")
	}
	if clicked {
		t.Fatalf("expected clicked handler NOT to be called for click outside")
	}

	// Event routing through ZStack
	topClicked := false
	topBtn := component.OnClick(component.Text("Top"), func(ev driver.MouseEvent) {
		topClicked = true
	})
	z := component.ZStack(button, topBtn)
	if !component.DispatchEvent(z, ctx, evInside) {
		t.Fatalf("expected ZStack to handle event")
	}
	if !topClicked {
		t.Fatalf("expected topmost component to consume event first")
	}
}

// ---------------------------------------------------------------------
// New Primitive Tests (Lipgloss/Glyph Gap Closure)
// ---------------------------------------------------------------------

func TestSpacerExpansion(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 40, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 40, Height: 1}, cell.NewStyle())

	// HStack: Text "A", Spacer, Text "B"
	// Text "A" at X=0, Spacer fills middle, Text "B" pushed to right side.
	stack := component.HStack(
		component.Text("A"),
		component.Spacer(),
		component.Text("B"),
	)

	stack.Draw(ctx, buf)

	if c := buf.Get(0, 0); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A' at (0,0), got %v", c)
	}
	// "B" should be at X=39 (rightmost)
	if c := buf.Get(39, 0); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B' at (39,0), got %v", c)
	}

	// Spacer LayoutInfo: Flex > 0, MinWidth/MinHeight = 0
	spacer := component.Spacer()
	props := spacer.LayoutInfo(cell.Rect{Width: 100, Height: 50})
	if props.Flex == 0 {
		t.Fatalf("expected Spacer to have Flex > 0")
	}
	if props.MinWidth != 0 || props.MinHeight != 0 {
		t.Fatalf("expected Spacer to have zero intrinsic size")
	}
}

func TestDividerDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 1}, cell.NewStyle())

	div := component.Divider()
	div.Draw(ctx, buf)

	// All 20 cells should be '─'
	for x := uint16(0); x < 20; x++ {
		c := buf.Get(x, 0)
		if c == nil || c.Content != '─' {
			t.Fatalf("expected '─' at (%d,0), got %v", x, c)
		}
	}
}

func TestDividerWithTitleDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 30, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 30, Height: 1}, cell.NewStyle())

	div := component.DividerWithTitle("Section", widgets.SymbolsSingle, cell.NewStyle())
	div.Draw(ctx, buf)

	// First two chars should be '─'
	if c := buf.Get(0, 0); c == nil || c.Content != '─' {
		t.Fatalf("expected '─' at (0,0), got %v", c)
	}
	// Title 'S' starts at X=3
	if c := buf.Get(3, 0); c == nil || c.Content != 'S' {
		t.Fatalf("expected 'S' at (3,0), got %v", c)
	}
}

func TestVDividerDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 1, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 1, Height: 5}, cell.NewStyle())

	vd := component.VDivider()
	vd.Draw(ctx, buf)

	for y := uint16(0); y < 5; y++ {
		c := buf.Get(0, y)
		if c == nil || c.Content != '│' {
			t.Fatalf("expected '│' at (0,%d), got %v", y, c)
		}
	}
}

func TestMarginLayoutAndDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 10})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 10}, cell.NewStyle())

	// Margin(top=2, right=3, bottom=1, left=4)
	m := component.Margin(component.Text("Hi"), 2, 3, 1, 4)
	m.Draw(ctx, buf)

	// 'H' should appear at (4, 2)
	if c := buf.Get(4, 2); c == nil || c.Content != 'H' {
		t.Fatalf("expected 'H' at (4,2), got %v", c)
	}
	if c := buf.Get(5, 2); c == nil || c.Content != 'i' {
		t.Fatalf("expected 'i' at (5,2), got %v", c)
	}

	// LayoutInfo should include margin in MinWidth/MinHeight
	props := m.LayoutInfo(cell.Rect{Width: 20, Height: 10})
	if props.MinWidth != 2+4+3 { // "Hi" width=2 + left=4 + right=3
		t.Fatalf("expected MinWidth=9, got %d", props.MinWidth)
	}
	if props.MinHeight != 1+2+1 { // "Hi" height=1 + top=2 + bottom=1
		t.Fatalf("expected MinHeight=4, got %d", props.MinHeight)
	}
}

func TestBorderEdgesSelectiveDraw(t *testing.T) {
	// Bottom-only border
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 3})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 3}, cell.NewStyle())

	bottomOnly := component.BottomBorder(component.Text("Tab"), '─', cell.NewStyle())
	bottomOnly.Draw(ctx, buf)

	// Text should appear at (0,0)
	if c := buf.Get(0, 0); c == nil || c.Content != 'T' {
		t.Fatalf("expected 'T' at (0,0), got %v", c)
	}
	// Bottom border at Y=2 (height=3, so maxY=2)
	if c := buf.Get(0, 2); c == nil || c.Content != '─' {
		t.Fatalf("expected '─' at (0,2), got %v", c)
	}
	// No left border at (0,1)
	if c := buf.Get(0, 1); c != nil && c.Content == '│' {
		t.Fatalf("expected NO vertical border at (0,1) for bottom-only border")
	}
}

func TestBorderCustomEdges(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 12, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 12, Height: 5}, cell.NewStyle())

	// Left + Bottom edges only
	bc := component.BorderCustom(
		component.Text("X"),
		widgets.SymbolsSingle,
		cell.NewStyle(),
		component.BorderEdgeLeft|component.BorderEdgeBottom,
	)
	bc.Draw(ctx, buf)

	// Left border at X=0
	if c := buf.Get(0, 0); c == nil || c.Content != '│' {
		t.Fatalf("expected '│' at (0,0), got %v", c)
	}
	// Bottom border at maxY=4
	if c := buf.Get(5, 4); c == nil || c.Content != '─' {
		t.Fatalf("expected '─' at (5,4), got %v", c)
	}
	// Bottom-left corner
	if c := buf.Get(0, 4); c == nil || c.Content != '└' {
		t.Fatalf("expected '└' at (0,4), got %v", c)
	}
}

func TestConstrainAndMaxWidth(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	// MaxWidth(10, Text("Hello World Long Text"))
	// Should clamp draw area to 10 columns
	constrained := component.MaxWidth(10, component.Text("Hello World Long"))
	constrained.Draw(ctx, buf)

	// 'H' at (0,0)
	if c := buf.Get(0, 0); c == nil || c.Content != 'H' {
		t.Fatalf("expected 'H' at (0,0), got %v", c)
	}
	// Beyond 10 columns should NOT have text content from the source string (clipped)
	// "Hello World Long" => index 10 would be 'L' if unconstrained
	if c := buf.Get(10, 0); c != nil && c.Content == 'L' {
		t.Fatalf("expected text to be clipped at column 10 due to MaxWidth, but 'L' leaked through")
	}

	// LayoutInfo should reflect constraint
	props := constrained.LayoutInfo(cell.Rect{Width: 80, Height: 24})
	if props.MaxWidth != 10 {
		t.Fatalf("expected MaxWidth=10, got %d", props.MaxWidth)
	}
}

func TestPlaceAlignment(t *testing.T) {
	// Place(20, 10, AlignCenter, AlignMiddle, Text("X"))
	// "X" should be centered in a 20x10 box
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 10})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 10}, cell.NewStyle())

	placed := component.Place(20, 10, component.AlignCenter, component.AlignMiddle, component.Text("X"))
	placed.Draw(ctx, buf)

	// X has width=1, so centered in 20 => X=9 or 10 depending on rounding
	// Height=1, centered in 10 => Y=4 or 5
	foundX := false
	for x := uint16(0); x < 20; x++ {
		for y := uint16(0); y < 10; y++ {
			if c := buf.Get(x, y); c != nil && c.Content == 'X' {
				foundX = true
				// Should be roughly centered
				if x < 5 || x > 15 || y < 2 || y > 8 {
					t.Fatalf("'X' at (%d,%d) is not centered in 20x10 box", x, y)
				}
			}
		}
	}
	if !foundX {
		t.Fatalf("expected 'X' to be drawn somewhere in the placed area")
	}
}

func TestForEachSliceGeneration(t *testing.T) {
	items := []string{"Alpha", "Beta", "Gamma"}
	components := component.ForEach(items, func(item string, i int) component.Component {
		return component.Text(item)
	})

	if len(components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(components))
	}

	// Draw and verify each item
	for i, label := range items {
		buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
		ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())
		components[i].Draw(ctx, buf)
		if c := buf.Get(0, 0); c == nil || c.Content != rune(label[0]) {
			t.Fatalf("ForEach[%d]: expected '%c' at (0,0), got %v", i, label[0], c)
		}
	}

	// ForEach with nil slice should return nil
	result := component.ForEach([]int{}, func(item int, i int) component.Component {
		return component.Text("x")
	})
	if result != nil {
		t.Fatalf("expected nil for empty slice, got %v", result)
	}
}

func TestDynamicComponent(t *testing.T) {
	mode := 0
	dynamic := component.Dynamic(func() component.Component {
		if mode == 0 {
			return component.Text("A")
		}
		return component.Text("B")
	})

	buf := buffer.NewBuffer(cell.Rect{Width: 5, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 5, Height: 1}, cell.NewStyle())

	// Mode 0: should render "A"
	dynamic.Draw(ctx, buf)
	if c := buf.Get(0, 0); c == nil || c.Content != 'A' {
		t.Fatalf("expected 'A', got %v", c)
	}

	// Mode 1: should render "B"
	mode = 1
	buf = buffer.NewBuffer(cell.Rect{Width: 5, Height: 1})
	dynamic.Draw(ctx, buf)
	if c := buf.Get(0, 0); c == nil || c.Content != 'B' {
		t.Fatalf("expected 'B', got %v", c)
	}
}

func TestTextRefComponent(t *testing.T) {
	msg := "Hello"
	ref := component.TextRef(&msg)

	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())
	ref.Draw(ctx, buf)

	if c := buf.Get(0, 0); c == nil || c.Content != 'H' {
		t.Fatalf("expected 'H', got %v", c)
	}

	// Mutate the variable and re-draw
	msg = "World"
	buf = buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ref.Draw(ctx, buf)
	if c := buf.Get(0, 0); c == nil || c.Content != 'W' {
		t.Fatalf("expected 'W' after mutation, got %v", c)
	}
}

func TestTextFnComponent(t *testing.T) {
	counter := 0
	fn := component.TextFn(func() string {
		counter++
		return "Call"
	})

	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())
	fn.Draw(ctx, buf)

	if c := buf.Get(0, 0); c == nil || c.Content != 'C' {
		t.Fatalf("expected 'C', got %v", c)
	}
	if counter != 1 {
		t.Fatalf("expected getter called once during Draw, got %d", counter)
	}
}

func TestOnKeyBinding(t *testing.T) {
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	escaped := false
	comp := component.OnKey(component.Text("Panel"), driver.KeyEsc, func(ev driver.KeyEvent) bool {
		escaped = true
		return true
	})

	// Send Escape key
	evEsc := &driver.Event{
		Type: driver.EventKey,
		Key:  driver.KeyEvent{Type: driver.KeyEsc},
	}
	if !component.DispatchEvent(comp, ctx, evEsc) {
		t.Fatalf("expected Escape key to be handled")
	}
	if !escaped {
		t.Fatalf("expected escaped flag to be set")
	}

	// Send different key - should NOT be handled by OnKey
	escaped = false
	evEnter := &driver.Event{
		Type: driver.EventKey,
		Key:  driver.KeyEvent{Type: driver.KeyEnter},
	}
	// OnEvent passes through to child, which is plain Text (no handler)
	if component.DispatchEvent(comp, ctx, evEnter) {
		t.Fatalf("expected Enter key NOT to be handled by Escape handler")
	}
}

func TestOnRuneBinding(t *testing.T) {
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	quitPressed := false
	comp := component.OnRune(component.Text("App"), 'q', func(ev driver.KeyEvent) bool {
		quitPressed = true
		return true
	})

	evQ := &driver.Event{
		Type: driver.EventKey,
		Key:  driver.KeyEvent{Type: driver.KeyRune, Ch: 'q'},
	}
	if !component.DispatchEvent(comp, ctx, evQ) {
		t.Fatalf("expected 'q' rune to be handled")
	}
	if !quitPressed {
		t.Fatalf("expected quitPressed to be set")
	}

	// Different rune should not trigger
	quitPressed = false
	evX := &driver.Event{
		Type: driver.EventKey,
		Key:  driver.KeyEvent{Type: driver.KeyRune, Ch: 'x'},
	}
	if component.DispatchEvent(comp, ctx, evX) {
		t.Fatalf("expected 'x' rune NOT to be handled by 'q' handler")
	}
}

// ---------------------------------------------------------------------
// Benchmarks: Verifying 0 Heap Allocations on the Hot Path
// ---------------------------------------------------------------------

func BenchmarkZStackDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	z := component.ZStack(
		component.Text("Layer 1"),
		component.Center(component.Fixed(20, 5, component.Text("Modal"))),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		z.Draw(ctx, buf)
	}
}

func BenchmarkFlexboxJustifyAndAlignZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 100, Height: 20})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 100, Height: 20}, cell.NewStyle())

	stack := component.HStack(
		component.Fixed(10, 2, component.Text("Item 1")),
		component.Fixed(10, 2, component.Text("Item 2")),
		component.Fixed(10, 2, component.Text("Item 3")),
	).WithJustify(component.JustifySpaceBetween).WithAlignItems(component.AlignItemsCenter)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack.Draw(ctx, buf)
	}
}

func BenchmarkStyleCascadeZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	styled := component.WithForeground(
		cell.NewColorRGB(0, 255, 128),
		component.WithBackground(
			cell.NewColorRGB(10, 10, 20),
			component.Text("Cascaded Style"),
		),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		styled.Draw(ctx, buf)
	}
}

func BenchmarkSpacerDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 1}, cell.NewStyle())

	stack := component.HStack(
		component.Text("L"),
		component.Spacer(),
		component.Text("R"),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack.Draw(ctx, buf)
	}
}

func BenchmarkDividerDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 1}, cell.NewStyle())

	div := component.Divider()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		div.Draw(ctx, buf)
	}
}

func BenchmarkMarginDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	m := component.MarginAll(component.Text("Content"), 2)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Draw(ctx, buf)
	}
}

func BenchmarkBorderEdgesDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	bc := component.BorderCustom(
		component.Text("Tab"),
		widgets.SymbolsSingle,
		cell.NewStyle(),
		component.BorderEdgeBottom|component.BorderEdgeLeft,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bc.Draw(ctx, buf)
	}
}

func BenchmarkConstrainDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	c := component.MaxWidth(40, component.Text("Constrained Content"))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Draw(ctx, buf)
	}
}

// ---------------------------------------------------------------------
// Overlay Tests
// ---------------------------------------------------------------------

func TestOverlayDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 20, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 5}, cell.NewStyle())

	base := component.Text("BACKGROUND")
	overlay := component.Text("FG")

	comp := component.Overlay(base, overlay, 5, 2)
	comp.Draw(ctx, buf)

	// Base: 'B' at (0,0)
	if c := buf.Get(0, 0); c == nil || c.Content != 'B' {
		t.Fatalf("expected base 'B' at (0,0), got %v", c)
	}
	// Overlay: 'F' at (5,2)
	if c := buf.Get(5, 2); c == nil || c.Content != 'F' {
		t.Fatalf("expected overlay 'F' at (5,2), got %v", c)
	}
	if c := buf.Get(6, 2); c == nil || c.Content != 'G' {
		t.Fatalf("expected overlay 'G' at (6,2), got %v", c)
	}
}

func TestOverlayOutOfBounds(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 3})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 3}, cell.NewStyle())

	base := component.Text("OK")
	overlay := component.Text("X")

	// Overlay at x=100 — completely out of bounds, should not panic
	comp := component.Overlay(base, overlay, 100, 0)
	comp.Draw(ctx, buf)

	if c := buf.Get(0, 0); c == nil || c.Content != 'O' {
		t.Fatalf("expected base 'O' at (0,0), got %v", c)
	}
}

func TestOverlayEventRouting(t *testing.T) {
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 20, Height: 10}, cell.NewStyle())

	overlayClicked := false
	base := component.Text("Base")
	overlay := component.OnClick(component.Text("Btn"), func(ev driver.MouseEvent) {
		overlayClicked = true
	})

	comp := component.Overlay(base, overlay, 5, 3)

	// Click at (6, 3) — inside overlay area
	ev := &driver.Event{
		Type:  driver.EventMouse,
		Mouse: driver.MouseEvent{X: 6, Y: 3, Button: driver.MouseLeft},
	}
	if !component.DispatchEvent(comp, ctx, ev) {
		t.Fatalf("expected overlay click to be handled")
	}
	if !overlayClicked {
		t.Fatalf("expected overlayClicked to be true")
	}
}

// ---------------------------------------------------------------------
// Transform Tests
// ---------------------------------------------------------------------

func TestTransformUppercase(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())

	comp := component.Uppercase(component.Text("hello"))
	comp.Draw(ctx, buf)

	expected := []rune{'H', 'E', 'L', 'L', 'O'}
	for i, ch := range expected {
		c := buf.Get(uint16(i), 0)
		if c == nil || c.Content != ch {
			t.Fatalf("expected '%c' at (%d,0), got %v", ch, i, c)
		}
	}
}

func TestTransformLowercase(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())

	comp := component.Lowercase(component.Text("HELLO"))
	comp.Draw(ctx, buf)

	expected := []rune{'h', 'e', 'l', 'l', 'o'}
	for i, ch := range expected {
		c := buf.Get(uint16(i), 0)
		if c == nil || c.Content != ch {
			t.Fatalf("expected '%c' at (%d,0), got %v", ch, i, c)
		}
	}
}

func TestTransformMask(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())

	comp := component.Mask(component.Text("Secret"), '•')
	comp.Draw(ctx, buf)

	// All visible characters should be '•'
	for i := 0; i < 6; i++ {
		c := buf.Get(uint16(i), 0)
		if c == nil || c.Content != '•' {
			t.Fatalf("expected '•' at (%d,0), got %v", i, c)
		}
	}
}

func TestTransformCustom(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 1}, cell.NewStyle())

	// ROT13-like: shift 'A'->'B', etc.
	comp := component.Transform(component.Text("ABC"), func(r rune) rune {
		return r + 1
	})
	comp.Draw(ctx, buf)

	expected := []rune{'B', 'C', 'D'}
	for i, ch := range expected {
		c := buf.Get(uint16(i), 0)
		if c == nil || c.Content != ch {
			t.Fatalf("expected '%c' at (%d,0), got %v", ch, i, c)
		}
	}
}

// ---------------------------------------------------------------------
// Inline Test
// ---------------------------------------------------------------------

func TestInlineConstraint(t *testing.T) {
	comp := component.Inline(component.Text("SingleLine"))
	props := comp.LayoutInfo(cell.Rect{Width: 80, Height: 24})

	if props.MaxHeight != 1 {
		t.Fatalf("expected Inline MaxHeight=1, got %d", props.MaxHeight)
	}
}

// ---------------------------------------------------------------------
// Border Preset Tests
// ---------------------------------------------------------------------

func TestBorderPresetsExist(t *testing.T) {
	// Verify all 7 presets exist and have non-zero runes
	presets := []struct {
		name    string
		symbols widgets.BorderSymbols
	}{
		{"Single", widgets.SymbolsSingle},
		{"Double", widgets.SymbolsDouble},
		{"Thick", widgets.SymbolsThick},
		{"Rounded", widgets.SymbolsRounded},
		{"Block", widgets.SymbolsBlock},
		{"OuterHalfBlock", widgets.SymbolsOuterHalfBlock},
		{"InnerHalfBlock", widgets.SymbolsInnerHalfBlock},
	}

	for _, p := range presets {
		if p.symbols.Horizontal == 0 || p.symbols.Vertical == 0 {
			t.Fatalf("%s: Horizontal or Vertical rune is zero", p.name)
		}
		if p.symbols.TopLeft == 0 || p.symbols.TopRight == 0 {
			t.Fatalf("%s: corner rune is zero", p.name)
		}
	}
}

func TestBorderOuterHalfBlockDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.Rect{Width: 10, Height: 5})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 10, Height: 5}, cell.NewStyle())

	b := component.Border(component.Text("X"), widgets.SymbolsOuterHalfBlock, cell.NewStyle())
	b.Draw(ctx, buf)

	// Top-left corner should be ▛
	if c := buf.Get(0, 0); c == nil || c.Content != '▛' {
		t.Fatalf("expected '▛' at (0,0), got %v", c)
	}
}

// ---------------------------------------------------------------------
// New Benchmarks
// ---------------------------------------------------------------------

func BenchmarkOverlayDrawZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 24})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 24}, cell.NewStyle())

	comp := component.Overlay(
		component.Text("Background"),
		component.Text("FG"),
		10, 5,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp.Draw(ctx, buf)
	}
}

func BenchmarkTransformUppercaseZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 1}, cell.NewStyle())

	comp := component.Uppercase(component.Text("hello world this is a test"))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp.Draw(ctx, buf)
	}
}

func BenchmarkTransformMaskZeroAlloc(b *testing.B) {
	buf := buffer.NewBuffer(cell.Rect{Width: 80, Height: 1})
	ctx := cell.NewContext(cell.Rect{X: 0, Y: 0, Width: 80, Height: 1}, cell.NewStyle())

	comp := component.Mask(component.Text("password123"), '•')

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp.Draw(ctx, buf)
	}
}
