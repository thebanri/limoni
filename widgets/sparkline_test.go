package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestSparklineDrawing(t *testing.T) {
	// A buffer of size 5 columns, 2 rows
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 5, 2))
	data := []float64{0.0, 5.0, 10.0} // Max is 10

	s := Sparkline{
		Data: data,
	}

	ctx := cell.NewContext(cell.NewRect(0, 0, 5, 2), cell.Style{})
	s.Draw(ctx, buf)

	// Column 0: 0.0 -> Empty spaces on both rows
	c0_0 := buf.Get(0, 0)
	c0_1 := buf.Get(0, 1)
	if c0_0.Content != ' ' || c0_1.Content != ' ' {
		t.Errorf("Expected column 0 to be empty, got (%q, %q)", c0_0.Content, c0_1.Content)
	}

	// Column 2: 10.0 (Max) -> Both rows should be fully drawn with '█'
	c2_0 := buf.Get(2, 0)
	c2_1 := buf.Get(2, 1)
	if c2_0.Content != '█' || c2_1.Content != '█' {
		t.Errorf("Expected column 2 to be fully filled, got (%q, %q)", c2_0.Content, c2_1.Content)
	}
}
