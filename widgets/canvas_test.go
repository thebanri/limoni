package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestCanvasSetUnset(t *testing.T) {
	c := NewCanvas(1, 1)

	// Set top-left (0,0) and bottom-right (1,3) within the single cell
	style := cell.Style{Fg: cell.NewColorANSI(1)}
	c.Set(0, 0, style)
	c.Set(1, 3, style)

	// Check mask
	expectedMask := brailleOffset[0][0] | brailleOffset[3][1] // 0x01 | 0x80 = 0x81
	if c.grid[0] != expectedMask {
		t.Errorf("Expected mask %02x, got %02x", expectedMask, c.grid[0])
	}

	// Draw and verify cell contents
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 1, 1))
	ctx := cell.NewContext(cell.NewRect(0, 0, 1, 1), cell.Style{})
	c.Draw(ctx, buf)

	cellPtr := buf.Get(0, 0)
	if cellPtr == nil {
		t.Fatal("Expected cell at (0,0) to not be nil")
	}

	expectedRune := rune(0x2800 + int(expectedMask))
	if cellPtr.Content != expectedRune {
		t.Errorf("Expected rune %q, got %q", expectedRune, cellPtr.Content)
	}

	// Unset top-left (0,0)
	c.Unset(0, 0)
	expectedMask = brailleOffset[3][1] // 0x80
	if c.grid[0] != expectedMask {
		t.Errorf("Expected mask after unset to be %02x, got %02x", expectedMask, c.grid[0])
	}

	// Clear canvas
	c.Clear()
	if c.grid[0] != 0 {
		t.Errorf("Expected grid to be clear, got %02x", c.grid[0])
	}
}

func TestCanvasVectorDrawing(t *testing.T) {
	// A 10x10 cell canvas (20x40 virtual pixels)
	c := NewCanvas(10, 10)
	style := cell.Style{}

	// Draw line from (0,0) to (19,39)
	c.DrawLine(0, 0, 19, 39, style)

	// Since it's a diagonal line, the start (0,0) and end (19, 39) must be set
	if (c.grid[0] & brailleOffset[0][0]) == 0 {
		t.Error("Expected start of line to be set")
	}
	lastCellIdx := 9*10 + 9
	if (c.grid[lastCellIdx] & brailleOffset[3][1]) == 0 {
		t.Error("Expected end of line to be set")
	}

	// Draw a circle of radius 5 at (10, 20)
	c.Clear()
	c.DrawCircle(10, 20, 5, style)
	// Point (15, 20) should be on the circle (cx + r, cy)
	// (15, 20) -> cx=7, cy=5. sub-grid: dx=1, dy=0
	cellIdx := 5*10 + 7
	if (c.grid[cellIdx] & brailleOffset[0][1]) == 0 {
		t.Error("Expected (15, 20) on circle to be set")
	}

	// Draw a rectangle
	c.Clear()
	c.DrawRect(0, 0, 10, 10, style)
	// (9, 9) should be set (bottom right of rect)
	cellIdx = 2*10 + 4 // (9,9) -> cx=4, cy=2. dx=1, dy=1
	if (c.grid[cellIdx] & brailleOffset[1][1]) == 0 {
		t.Error("Expected (9,9) on rectangle to be set")
	}

	// Draw quadratic Bezier
	c.Clear()
	c.DrawBezierQuadratic(0, 0, 10, 20, 20, 0, 10, style)

	// Draw cubic Bezier
	c.Clear()
	c.DrawBezierCubic(0, 0, 5, 10, 15, 10, 20, 0, 10, style)
}

func TestCanvasColorBlending(t *testing.T) {
	c := NewCanvas(1, 1)

	// Set pixel at (0,0) with red (255, 0, 0)
	styleRed := cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
	c.Set(0, 0, styleRed)

	// Set pixel at (1,1) with blue (0, 0, 255) in the same character cell
	styleBlue := cell.Style{Fg: cell.NewColorRGB(0, 0, 255)}
	c.Set(1, 1, styleBlue)

	// The foreground color of cell 0 should be blended:
	// R: (255 + 0) / 2 = 127
	// G: (0 + 0) / 2 = 0
	// B: (0 + 255) / 2 = 127
	mixedFg := c.styles[0].Fg
	if mixedFg.Type() != cell.ColorRGB {
		t.Fatalf("Expected mixed color to be RGB, got %v", mixedFg.Type())
	}
	r, g, b := mixedFg.RGB()
	if r != 127 || g != 0 || b != 127 {
		t.Errorf("Expected blended color (127, 0, 127), got (%d, %d, %d)", r, g, b)
	}
}
