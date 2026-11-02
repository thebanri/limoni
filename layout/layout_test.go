package layout

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestFlexLayoutHorizontalRatios(t *testing.T) {
	// An area 100 wide, split horizontally 1:2:2
	area := cell.NewRect(0, 0, 100, 10)
	lay := NewFlexLayout(Horizontal, 0, Ratio(1), Ratio(2), Ratio(2))

	rects := lay.Split(area)
	if len(rects) != 3 {
		t.Fatalf("wrong number of parts. Want: 3, got: %d", len(rects))
	}

	// Ratios: 1/5 (20%), 2/5 (40%), 2/5 (40%)
	// Expected widths: 20, 40, 40
	expectedWidths := []uint16{20, 40, 40}
	expectedX := []uint16{0, 20, 60}

	for i, r := range rects {
		if r.Width != expectedWidths[i] {
			t.Errorf("index %d: wrong width. Want: %d, got: %d", i, expectedWidths[i], r.Width)
		}
		if r.X != expectedX[i] {
			t.Errorf("index %d: wrong X. Want: %d, got: %d", i, expectedX[i], r.X)
		}
		if r.Height != 10 {
			t.Errorf("height should not have changed")
		}
	}
}

func TestSelectBreakpointUsesWidestEligibleVariant(t *testing.T) {
	breakpoints := []Breakpoint{
		{Name: "compact", MinWidth: 0},
		{Name: "medium", MinWidth: 80},
		{Name: "wide", MinWidth: 120},
	}
	if got := SelectBreakpoint(100, breakpoints); got.Name != "medium" {
		t.Fatalf("breakpoint at width 100 = %+v, want medium", got)
	}
	if got := SelectBreakpoint(140, breakpoints); got.Name != "wide" {
		t.Fatalf("breakpoint at width 140 = %+v, want wide", got)
	}
	if got := SelectBreakpoint(0, nil); got != (Breakpoint{}) {
		t.Fatalf("empty breakpoint selection = %+v, want zero value", got)
	}
}

func TestResolveBreakpointFallbackAndUnsortedInput(t *testing.T) {
	values := []BreakpointValue[Direction]{
		{MinWidth: 120, Value: Horizontal},
		{MinWidth: 40, Value: Vertical},
	}
	if got := ResolveBreakpoint(20, values, Horizontal); got != Horizontal {
		t.Fatalf("fallback direction = %v, want horizontal", got)
	}
	if got := ResolveBreakpoint(80, values, Horizontal); got != Vertical {
		t.Fatalf("medium direction = %v, want vertical", got)
	}
	if got := ResolveBreakpoint(160, values, Vertical); got != Horizontal {
		t.Fatalf("wide direction = %v, want horizontal", got)
	}
}

func TestFlexLayoutVerticalMixed(t *testing.T) {
	// An area 50 high, split vertically into Fixed(10), Percentage(20) (20% of 50 = 10) and Fill() (the remaining 30)
	area := cell.NewRect(0, 0, 80, 50)
	lay := NewFlexLayout(Vertical, 0, Fixed(10), Percentage(20), Fill())

	rects := lay.Split(area)
	if len(rects) != 3 {
		t.Fatalf("wrong number of parts")
	}

	expectedHeights := []uint16{10, 10, 30}
	expectedY := []uint16{0, 10, 20}

	for i, r := range rects {
		if r.Height != expectedHeights[i] {
			t.Errorf("index %d: wrong height. Want: %d, got: %d", i, expectedHeights[i], r.Height)
		}
		if r.Y != expectedY[i] {
			t.Errorf("index %d: wrong Y. Want: %d, got: %d", i, expectedY[i], r.Y)
		}
		if r.Width != 80 {
			t.Errorf("width should not have changed")
		}
	}
}

func TestFlexLayoutGap(t *testing.T) {
	// An area 20 wide, split horizontally into 3 Fixed(5) with a 2-cell gap between them (gap = 2).
	// Total gap: 2 * 2 = 4. Net usable space: 20 - 4 = 16.
	// Fixed areas: 5 + 5 + 5 = 15. The remaining cell is not shared out, since all of them are fixed.
	area := cell.NewRect(0, 0, 20, 5)
	lay := NewFlexLayout(Horizontal, 2, Fixed(5), Fixed(5), Fixed(5))

	rects := lay.Split(area)

	expectedX := []uint16{0, 7, 14} // (0 + 5 + 2 = 7), (7 + 5 + 2 = 14)
	for i, r := range rects {
		if r.Width != 5 {
			t.Errorf("width should be 5, got: %d", r.Width)
		}
		if r.X != expectedX[i] {
			t.Errorf("index %d: wrong X. Want: %d, got: %d", i, expectedX[i], r.X)
		}
	}
}

func TestFlexLayoutExceededScaling(t *testing.T) {
	// Constraints adding up to 20 applied to a small area 10 wide.
	// They must shrink proportionally and not exceed the area.
	area := cell.NewRect(0, 0, 10, 5)
	lay := NewFlexLayout(Horizontal, 0, Fixed(10), Fixed(10)) // Total 20, area 10

	rects := lay.Split(area)
	// Expected: each shrinks to a width of 5
	if rects[0].Width != 5 || rects[1].Width != 5 {
		t.Errorf("overflow scaling failed. Got widths: %d and %d", rects[0].Width, rects[1].Width)
	}
}

func TestFlexLayoutMinMaxConstraints(t *testing.T) {
	// An area 50 wide, split horizontally into Min(15), Max(10) and Fill()
	// Total usable space: 50
	// 1. Min(15) starts with 15.
	// 2. Max(10) starts with 0.
	// 3. Fill() starts with 0.
	// Remaining space: 50 - 15 = 35.
	// The remaining 35 is shared among those that can grow: Min (weight 1), Max (weight 1, limit 10), Fill (weight 1).
	// Iteration 1: total weight = 3. remaining = 35.
	// Share: 35 / 3 = 11.
	// - Min gets +11. Total: 15 + 11 = 26.
	// - Max gets +11. Total: 11. But as it is Max(10), it locks at 10.
	// - Fill gets +11. Total: 11.
	// Iteration 2: Max is now inactive. remaining = 3.
	// The remaining 3 is shared between Min and Fill.
	// - Min gets +1. Total: 26 + 1 = 27.
	// - The rounding remainder of 1 goes to the first active element from the left (Min): +1. Min total = 28.
	// - Fill gets +1. Fill total = 12.
	// Total sizes: Min=28, Max=10, Fill=12. Total = 50.

	area := cell.NewRect(0, 0, 50, 10)
	lay := NewFlexLayout(Horizontal, 0, Min(15), Max(10), Fill())

	rects := lay.Split(area)
	if len(rects) != 3 {
		t.Fatalf("wrong number of parts. Want: 3, got: %d", len(rects))
	}

	if rects[0].Width != 28 {
		t.Errorf("Min(15): wrong width. Want: 28, got: %d", rects[0].Width)
	}
	if rects[1].Width != 10 {
		t.Errorf("Max(10): wrong width. Want: 10, got: %d", rects[1].Width)
	}
	if rects[2].Width != 12 {
		t.Errorf("Fill(): wrong width. Want: 12, got: %d", rects[2].Width)
	}
}

func TestFlexLayoutFitContent(t *testing.T) {
	area := cell.NewRect(0, 0, 100, 10)
	lay := NewFlexLayout(Horizontal, 0, FitContent(), FitContent())

	rects := lay.Split(area, 25, 45)
	if len(rects) != 2 {
		t.Fatalf("wrong number of parts. Want: 2, got: %d", len(rects))
	}

	if rects[0].Width != 25 {
		t.Errorf("FitContent 1: wrong width. Want: 25, got: %d", rects[0].Width)
	}
	if rects[1].Width != 45 {
		t.Errorf("FitContent 2: wrong width. Want: 45, got: %d", rects[1].Width)
	}
}

func TestFlexLayoutFitContentMixed(t *testing.T) {
	// An area 100 wide
	// FitContent() (size = 15)
	// Fixed(20)
	// Fill() (the remaining 100 - 15 - 20 = 65)
	area := cell.NewRect(0, 0, 100, 10)
	lay := NewFlexLayout(Horizontal, 0, FitContent(), Fixed(20), Fill())

	rects := lay.Split(area, 15)
	if len(rects) != 3 {
		t.Fatalf("wrong number of parts. Want: 3, got: %d", len(rects))
	}

	if rects[0].Width != 15 {
		t.Errorf("FitContent: wrong width. Want: 15, got: %d", rects[0].Width)
	}
	if rects[1].Width != 20 {
		t.Errorf("Fixed: wrong width. Want: 20, got: %d", rects[1].Width)
	}
	if rects[2].Width != 65 {
		t.Errorf("Fill: wrong width. Want: 65, got: %d", rects[2].Width)
	}
}

func TestFlexLayout_MoreThan32Constraints(t *testing.T) {
	// Create 50 constraints (exceeding old 32 bitmask limit)
	constraints := make([]Constraint, 50)
	for i := range constraints {
		constraints[i] = Fill()
	}
	area := cell.NewRect(0, 0, 100, 20)
	lay := NewFlexLayout(Horizontal, 0, constraints...)
	rects := lay.Split(area)

	if len(rects) != 50 {
		t.Fatalf("expected 50 rects, got %d", len(rects))
	}

	totalW := uint16(0)
	for i, r := range rects {
		if r.Width == 0 {
			t.Errorf("element %d got 0 width unexpectedly", i)
		}
		totalW += r.Width
	}
	if totalW != 100 {
		t.Errorf("expected sum of 50 elements to equal 100, got %d", totalW)
	}
}

func TestFlexLayout_GapOverflowProtection(t *testing.T) {
	// 5 items in a 10-wide area with gap = 10 (total gap = 40 > 10)
	area := cell.NewRect(0, 0, 10, 5)
	lay := NewFlexLayout(Horizontal, 10, Fixed(5), Fixed(5), Fixed(5), Fixed(5), Fixed(5))
	rects := lay.Split(area)

	if len(rects) != 5 {
		t.Fatalf("expected 5 rects, got %d", len(rects))
	}
	for i, r := range rects {
		if r.Width != 0 {
			t.Errorf("element %d expected width 0 due to excessive gap, got %d", i, r.Width)
		}
	}
}

func BenchmarkFlexLayout_Split_3Way(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40)
	lay := NewFlexLayout(Vertical, 0, Fixed(3), Fill(), Fixed(3))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lay.Split(area)
	}
}

func BenchmarkFlexLayout_Split_Complex(b *testing.B) {
	area := cell.NewRect(0, 0, 200, 60)
	lay := NewFlexLayout(Horizontal, 1,
		Fixed(15),
		Percentage(25),
		Ratio(2),
		Ratio(1),
		Min(10),
		Max(30),
		Fill(),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lay.Split(area)
	}
}

func BenchmarkFlexLayout_Split_50Constraints(b *testing.B) {
	area := cell.NewRect(0, 0, 500, 100)
	constraints := make([]Constraint, 50)
	for i := range constraints {
		if i%2 == 0 {
			constraints[i] = Fixed(5)
		} else {
			constraints[i] = Fill()
		}
	}
	lay := NewFlexLayout(Horizontal, 0, constraints...)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lay.Split(area)
	}
}

func BenchmarkGridLayout_Split(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40)
	grid := NewGridLayout(
		[]GridConstraint{GridFixed(10), GridPercentage(30), GridFraction(2), GridFraction(1)},
		[]GridConstraint{GridFixed(3), GridFraction(1), GridFixed(2)},
		1,
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = grid.Split(area)
	}
}

func BenchmarkArrange_Aligned(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40)
	measures := []Measure{
		{IdealWidth: 20, IdealHeight: 10, MaxWidth: 120, MaxHeight: 40},
		{IdealWidth: 40, IdealHeight: 25, MaxWidth: 120, MaxHeight: 40},
		{IdealWidth: 30, IdealHeight: 15, MaxWidth: 120, MaxHeight: 40},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ArrangeAligned(area, measures, Horizontal, 1, AlignCenter)
	}
}
