package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestBarChart_DrawVertical(t *testing.T) {
	data := []BarData{
		{Label: "Q1", Value: 25, Color: cell.NewColorRGB(0, 255, 128)},
		{Label: "Q2", Value: 50, Color: cell.NewColorRGB(0, 200, 255)},
		{Label: "Q3", Value: 75, Color: cell.NewColorRGB(255, 128, 0)},
		{Label: "Q4", Value: 100, Color: cell.NewColorRGB(255, 50, 50)},
	}

	chart := BarChart{
		ID:         "sales_chart",
		Data:       data,
		Direction:  BarVertical,
		BarWidth:   4,
		BarGap:     2,
		ShowValues: true,
	}

	area := cell.NewRect(0, 0, 40, 12)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	chart.Draw(ctx, buf)

	// Verify cells are drawn
	c := buf.Get(1, area.Height-2)
	if c == nil {
		t.Fatal("expected non-nil cell in bar chart area")
	}
}

func TestBarChart_DrawHorizontal(t *testing.T) {
	data := []BarData{
		{Label: "CPU", Value: 42.5},
		{Label: "Memory", Value: 85.0},
		{Label: "Disk", Value: 15.2},
	}

	chart := BarChart{
		ID:         "system_chart",
		Data:       data,
		Direction:  BarHorizontal,
		ShowValues: true,
	}

	area := cell.NewRect(0, 0, 50, 8)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	chart.Draw(ctx, buf)

	c := buf.Get(0, 0)
	if c == nil {
		t.Fatal("expected non-nil cell at (0, 0)")
	}
}
