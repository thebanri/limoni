package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestLineChart_Draw(t *testing.T) {
	datasets := []LineDataset{
		{
			Name:  "Network In",
			Data:  []float64{10, 25, 18, 45, 60, 52, 78, 90},
			Color: cell.NewColorRGB(46, 204, 113),
		},
		{
			Name:  "Network Out",
			Data:  []float64{5, 12, 10, 30, 40, 35, 50, 65},
			Color: cell.NewColorRGB(52, 152, 219),
		},
	}

	chart := LineChart{
		ID:         "net_chart",
		Datasets:   datasets,
		ShowAxes:   true,
		ShowLegend: true,
		XLabels:    []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00"},
	}

	area := cell.NewRect(0, 0, 60, 16)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	chart.Draw(ctx, buf)

	// Check that legend and canvas cells are populated
	c := buf.Get(10, 5)
	if c == nil {
		t.Fatal("expected non-nil cell in line chart plot area")
	}
}
