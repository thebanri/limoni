package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestPieChart_Draw(t *testing.T) {
	slices := []PieSlice{
		{Label: "Go", Value: 45, Color: cell.NewColorRGB(0, 200, 255)},
		{Label: "Rust", Value: 30, Color: cell.NewColorRGB(255, 100, 50)},
		{Label: "TypeScript", Value: 25, Color: cell.NewColorRGB(50, 150, 255)},
	}

	chart := PieChart{
		ID:              "lang_chart",
		Data:            slices,
		DonutHoleRatio:  0.4,
		ShowLegend:      true,
		ShowPercentages: true,
	}

	area := cell.NewRect(0, 0, 40, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	chart.Draw(ctx, buf)

	// Verify buffer cells
	c := buf.Get(5, 5)
	if c == nil {
		t.Fatal("expected non-nil cell in pie chart area")
	}
}
