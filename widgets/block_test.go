package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// dummyWidget is a helper widget to test child nested drawing
type dummyWidget struct {
	width  uint16
	height uint16
}

func (d dummyWidget) Draw(ctx cell.Context, buf *buffer.Buffer) {
	// Child widget draws 'D' at its top-left coordinate
	buf.SetCell(ctx.Area.X, ctx.Area.Y, cell.Cell{Content: 'D', Style: ctx.Style})
}

func (d dummyWidget) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return d.width, d.height
}

func TestBlockBordersAndBackground(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := buffer.NewBuffer(area)

	block := Block{
		Borders:       BorderAll,
		BorderSymbols: SymbolsRounded,
		Style:         cell.Style{Bg: cell.NewColorANSI(1)}, // Red background
	}

	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

	// Corners check
	topLeft := buf.Get(0, 0)
	if topLeft.Content != '╭' || topLeft.Style.Bg.ANSI() != 1 {
		t.Errorf("Top-left corner drawing or background color failed: %c", topLeft.Content)
	}

	bottomRight := buf.Get(9, 4)
	if bottomRight.Content != '╯' {
		t.Errorf("Bottom-right corner drawing failed: %c", bottomRight.Content)
	}

	// Inner fill check
	inner := buf.Get(2, 2)
	if inner.Content != ' ' || inner.Style.Bg.ANSI() != 1 {
		t.Errorf("Inner block area should be filled with red background")
	}
}

func TestBlockTitle(t *testing.T) {
	area := cell.NewRect(0, 0, 15, 3)
	buf := buffer.NewBuffer(area)

	block := Block{
		Title:          "TUI",
		TitleAlignment: AlignCenter,
		Borders:        BorderTop,
	}

	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

	// " TUI " is 5 chars. (15 - 5) / 2 = 5. So it starts at X = 5.
	// Expected top row: `───── TUI ─────`
	expected := " TUI "
	for i, r := range expected {
		c := buf.Get(uint16(5+i), 0)
		if c == nil || c.Content != r {
			t.Errorf("Title character at index %d did not match: %c", i, c.Content)
		}
	}
}

func TestBlockChildDrawingAndSizeHint(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	buf := buffer.NewBuffer(area)

	child := dummyWidget{width: 5, height: 3}
	block := Block{
		Borders:     BorderAll,
		PaddingLeft: 2, // Left padding = 2, Left border = 1 -> Child X offset = 3
		PaddingTop:  1, // Top padding = 1, Top border = 1 -> Child Y offset = 2
		Child:       child,
	}

	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

	// Check if child drew its 'D' at X=3, Y=2
	childCell := buf.Get(3, 2)
	if childCell == nil || childCell.Content != 'D' {
		t.Errorf("Child drawing failed. Expected 'D' at (3, 2), got: %c", childCell.Content)
	}

	// SizeHint negotiation test
	// Overhead width: borderL(1) + borderR(1) + paddingL(2) + paddingR(0) = 4
	// Overhead height: borderT(1) + borderB(1) + paddingT(1) + paddingB(0) = 3
	// Child wishes: 5x3.
	// Total hint: (5+4) x (3+3) = 9x6.
	w, h := block.SizeHint(area)
	if w != 9 || h != 6 {
		t.Errorf("SizeHint calculation failed. Expected 9x6, got %dx%d", w, h)
	}
}
