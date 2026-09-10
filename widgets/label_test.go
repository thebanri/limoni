package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestLabelDraw(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 5))
	label := NewLabel("Hello\nWorld").WithStyle(cell.Style{
		Fg: cell.NewColorRGB(255, 255, 255),
		Bg: cell.NewColorRGB(18, 20, 26),
	})

	ctx := cell.NewContext(cell.NewRect(0, 0, 10, 3), cell.NewStyle())
	label.Draw(ctx, buf)

	c0 := buf.Get(0, 0)
	if c0 == nil || c0.Content != 'H' || c0.Style.Bg != cell.NewColorRGB(18, 20, 26) {
		t.Fatalf("expected 'H' with bg RGB(18, 20, 26), got %v", c0)
	}

	// Verify trailing cell in row 0 is filled with background
	cTrailing := buf.Get(7, 0)
	if cTrailing == nil || cTrailing.Content != ' ' || cTrailing.Style.Bg != cell.NewColorRGB(18, 20, 26) {
		t.Fatalf("expected trailing space with bg RGB(18, 20, 26), got %v", cTrailing)
	}

	// Verify row 2 (beyond text lines) is filled with background
	cRow2 := buf.Get(3, 2)
	if cRow2 == nil || cRow2.Content != ' ' || cRow2.Style.Bg != cell.NewColorRGB(18, 20, 26) {
		t.Fatalf("expected empty row filled with bg RGB(18, 20, 26), got %v", cRow2)
	}
}

func TestLabelSizeHint(t *testing.T) {
	label := NewLabel("Short\nMuch longer line")
	w, h := label.SizeHint(cell.NewRect(0, 0, 50, 10))
	if w != 16 || h != 2 {
		t.Fatalf("expected (16, 2), got (%d, %d)", w, h)
	}
}
