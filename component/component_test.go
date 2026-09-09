package component_test

import (
	"testing"

	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
