package layout

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestGridLayoutBasic(t *testing.T) {
	parent := cell.NewRect(0, 0, 100, 40)

	// Columns: 20 fixed, 50%, 1fr (kalan: 100 - 20 - 50 = 30)
	// Rows: 10 fixed, 2fr, 1fr (kalan: 40 - 10 = 30)
	// Gap: 0
	gridLayout := NewGridLayout(
		[]GridConstraint{GridFixed(20), GridPercentage(50), GridFraction(1)},
		[]GridConstraint{GridFixed(10), GridFraction(2), GridFraction(1)},
		0,
	)

	areas := gridLayout.Split(parent)

	// Col widths: 20, 50, 30
	// Row heights: 10, 20, 10
	if areas.colW[0] != 20 || areas.colW[1] != 50 || areas.colW[2] != 30 {
		t.Errorf("Unexpected column widths resolved: %v", areas.colW)
	}
	if areas.rowH[0] != 10 || areas.rowH[1] != 20 || areas.rowH[2] != 10 {
		t.Errorf("Unexpected row heights resolved: %v", areas.rowH)
	}

	// Test Cell dimensions
	c00 := areas.Cell(0, 0)
	if c00.Area.Width != 20 || c00.Area.Height != 10 {
		t.Errorf("Cell(0,0) dimensions = %v; expected 20x10", c00.Area)
	}

	// Test Span(rowSpan=2, colSpan=2) starting at (0, 0)
	// Should merge Row 0 & 1, Col 0 & 1
	// Total width: col 0 (20) + col 1 (50) = 70
	// Total height: row 0 (10) + row 1 (20) = 30
	spanRect := c00.Span(2, 2)
	if spanRect.Width != 70 || spanRect.Height != 30 {
		t.Errorf("Span(2,2) dimensions = %v; expected 70x30", spanRect)
	}
	if spanRect.X != parent.X || spanRect.Y != parent.Y {
		t.Errorf("Span(2,2) coordinates = (%d,%d); expected (%d,%d)", spanRect.X, spanRect.Y, parent.X, parent.Y)
	}
}

func TestGridLayoutWithGaps(t *testing.T) {
	parent := cell.NewRect(0, 0, 52, 22)

	// Gap = 2
	// Columns: 10 fixed, 2fr, 1fr (total gap: 4. available: 52 - 4 = 48. remaining for fr: 48 - 10 = 38)
	// Rows: 10 fixed, 10 fixed (total gap: 2. available: 22 - 2 = 20)
	gridLayout := NewGridLayout(
		[]GridConstraint{GridFixed(10), GridFraction(2), GridFraction(1)},
		[]GridConstraint{GridFixed(10), GridFixed(10)},
		2,
	)

	areas := gridLayout.Split(parent)

	// Gap: 2
	// Col widths resolved: 10, 25 (approx of 38 * 2/3), 12 (approx of 38 * 1/3)
	// Total col width: 10 + 2 + 25 + 2 + 12 = 51 (approximated fr division)
	if len(areas.colW) != 3 || areas.colW[0] != 10 {
		t.Errorf("Unexpected column resolving: %v", areas.colW)
	}
	if len(areas.rowH) != 2 || areas.rowH[0] != 10 || areas.rowH[1] != 10 {
		t.Errorf("Unexpected row resolving: %v", areas.rowH)
	}

	// Test Span with gap
	c00 := areas.Cell(0, 0)
	spanRect := c00.Span(2, 2) // Merges row 0 & 1, col 0 & 1
	// Height should be row 0 (10) + gap (2) + row 1 (10) = 22
	if spanRect.Height != 22 {
		t.Errorf("Span(2,2) height with gap = %d; expected 22", spanRect.Height)
	}
}
