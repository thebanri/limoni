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
		Style:         cell.Style{Bg: cell.NewColorANSI(1)},
	}

	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

	topLeft := buf.Get(0, 0)
	if topLeft.Content != '╭' || topLeft.Style.Bg.ANSI() != 1 {
		t.Errorf("Top-left corner drawing or background color failed: %c", topLeft.Content)
	}

	bottomRight := buf.Get(9, 4)
	if bottomRight.Content != '╯' {
		t.Errorf("Bottom-right corner drawing failed: %c", bottomRight.Content)
	}

	inner := buf.Get(2, 2)
	if (inner.Content != ' ' && inner.Content != '█') || inner.Style.Bg.ANSI() != 1 {
		t.Errorf("Inner block area should be filled with red background")
	}
}

func TestBlockTitle(t *testing.T) {
	area := cell.NewRect(0, 0, 15, 3)
	buf := buffer.NewBuffer(area)

	block := Block{Title: "TUI", TitleAlignment: AlignCenter, Borders: BorderTop}
	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

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
		PaddingLeft: 2,
		PaddingTop:  1,
		Child:       child,
	}

	ctx := cell.NewContext(area, cell.Style{})
	block.Draw(ctx, buf)

	childCell := buf.Get(3, 2)
	if childCell == nil || childCell.Content != 'D' {
		t.Errorf("Child drawing failed. Expected 'D' at (3, 2), got: %c", childCell.Content)
	}

	w, h := block.SizeHint(area)
	if w != 9 || h != 6 {
		t.Errorf("SizeHint calculation failed. Expected 9x6, got %dx%d", w, h)
	}
}

type areaProbe struct{ area cell.Rect }

func (p *areaProbe) Draw(ctx cell.Context, _ *buffer.Buffer) { p.area = ctx.Area }
func (p *areaProbe) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

func TestBlockMarginAndPaddingBoxModel(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	buf := buffer.NewBuffer(area)
	probe := &areaProbe{}
	block := Block{
		Borders: BorderAll,
		Margin:  Insets{Top: 1, Right: 2, Bottom: 1, Left: 2},
		Padding: Insets{Top: 1, Right: 2, Bottom: 1, Left: 2},
		Child:   probe,
	}
	block.Draw(cell.NewContext(area, cell.Style{}), buf)

	want := cell.NewRect(5, 3, 10, 4)
	if probe.area != want {
		t.Fatalf("child area = %+v; want %+v", probe.area, want)
	}
}
