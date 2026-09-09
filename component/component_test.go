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
