package widgets

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// LineDataset holds one line graph series with its color and data points.
type LineDataset struct {
	Name  string
	Data  []float64
	Color cell.Color
}

// LineChart renders smooth Braille-based multi-series line graphs with labeled axes.
type LineChart struct {
	ID         string
	Datasets   []LineDataset
	MinY       float64
	MaxY       float64
	XLabels    []string
	ShowAxes   bool
	ShowGrid   bool
	ShowLegend bool
	Style      cell.Style
	AxisStyle  cell.Style
}

// Draw renders the line chart with Braille subpixels and axes.
func (lc LineChart) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 12 || area.Height < 5 || len(lc.Datasets) == 0 {
		return
	}

	baseStyle := ctx.Style.Merge(lc.Style)
	axisStyle := baseStyle.Merge(lc.AxisStyle)
	if lc.AxisStyle == (cell.Style{}) {
		axisStyle = baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(120, 135, 155)})
	}

	// Calculate Min & Max Y
	minY := lc.MinY
	maxY := lc.MaxY
	if maxY <= minY {
		first := true
		for _, ds := range lc.Datasets {
			for _, v := range ds.Data {
				if first {
					minY = v
					maxY = v
					first = false
				} else {
					if v < minY {
						minY = v
					}
					if v > maxY {
						maxY = v
					}
				}
			}
		}
		if maxY == minY {
			maxY = minY + 10
		}
	}
	rangeY := maxY - minY

	// Plot area calculation
	yAxisWidth := uint16(6)
	xAxisHeight := uint16(1)
	legendHeight := uint16(0)
	if lc.ShowLegend {
		legendHeight = 1
	}

	plotX := area.X + yAxisWidth
	plotY := area.Y + legendHeight
	plotW := area.Width - yAxisWidth - 1
	plotH := area.Height - legendHeight - xAxisHeight
	if plotW < 4 || plotH < 2 {
		return
	}

	// Draw Legend at top
	if lc.ShowLegend {
		legendCursor := area.X + yAxisWidth
		for _, ds := range lc.Datasets {
			dsColor := ds.Color
			if dsColor == 0 {
				dsColor = cell.NewColorRGB(0, 200, 255)
			}
			dotStyle := cell.Style{Fg: dsColor, Modifier: cell.ModifierBold}
			buf.SetString(legendCursor, area.Y, "● ", dotStyle)
			legendCursor += 2
			buf.SetString(legendCursor, area.Y, ds.Name+"  ", baseStyle)
			legendCursor += uint16(utf8.RuneCountInString(ds.Name) + 2)
		}
	}

	// Draw Y-Axis Labels
	yTopLabel := fmt.Sprintf("%5.1f", maxY)
	yMidLabel := fmt.Sprintf("%5.1f", (minY+maxY)/2)
	yBotLabel := fmt.Sprintf("%5.1f", minY)

	buf.SetString(area.X, plotY, yTopLabel, axisStyle)
	buf.SetString(area.X, plotY+plotH/2, yMidLabel, axisStyle)
	buf.SetString(area.X, plotY+plotH-1, yBotLabel, axisStyle)

	// Draw Axes Lines
	if lc.ShowAxes {
		for y := uint16(0); y < plotH; y++ {
			buf.SetCell(plotX-1, plotY+y, cell.Cell{Content: '│', Style: axisStyle})
		}
		buf.SetCell(plotX-1, plotY+plotH, cell.Cell{Content: '└', Style: axisStyle})
		for x := uint16(0); x < plotW; x++ {
			buf.SetCell(plotX+x, plotY+plotH, cell.Cell{Content: '─', Style: axisStyle})
		}
	}

	// Draw X-Axis Labels at bottom
	if len(lc.XLabels) > 0 {
		labelY := plotY + plotH + 1
		if labelY < area.Y+area.Height {
			numLabels := len(lc.XLabels)
			step := float64(plotW) / float64(numLabels)
			for i, lbl := range lc.XLabels {
				lx := plotX + uint16(float64(i)*step)
				if lx+uint16(utf8.RuneCountInString(lbl)) <= area.X+area.Width {
					buf.SetString(lx, labelY, lbl, axisStyle)
				}
			}
		}
	}

	// Create Braille Canvas for subpixel line rendering
	canvas := NewCanvas(plotW, plotH)
	virtW := int(plotW) * 2
	virtH := int(plotH) * 4

	for _, ds := range lc.Datasets {
		if len(ds.Data) < 2 {
			continue
		}

		dsColor := ds.Color
		if dsColor == 0 {
			dsColor = cell.NewColorRGB(0, 220, 255)
		}
		lineStyle := cell.Style{Fg: dsColor}

		numPoints := len(ds.Data)
		stepX := float64(virtW-1) / float64(numPoints-1)

		for i := 0; i < numPoints-1; i++ {
			x0 := float64(i) * stepX
			norm0 := (ds.Data[i] - minY) / rangeY
			if norm0 < 0 {
				norm0 = 0
			}
			if norm0 > 1 {
				norm0 = 1
			}
			y0 := float64(virtH-1) - (norm0 * float64(virtH-1))

			x1 := float64(i+1) * stepX
			norm1 := (ds.Data[i+1] - minY) / rangeY
			if norm1 < 0 {
				norm1 = 0
			}
			if norm1 > 1 {
				norm1 = 1
			}
			y1 := float64(virtH-1) - (norm1 * float64(virtH-1))

			canvas.DrawLine(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x1)), int(math.Round(y1)), lineStyle)
		}
	}

	// Render Braille canvas onto plot area
	canvas.Draw(cell.NewContext(cell.NewRect(plotX, plotY, plotW, plotH), baseStyle), buf)
}

// SizeHint returns preferred dimensions for LineChart.
func (lc LineChart) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}
