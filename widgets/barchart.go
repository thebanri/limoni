package widgets

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// BarChartDirection specifies whether bars grow vertically or horizontally.
type BarChartDirection uint8

const (
	BarVertical BarChartDirection = iota
	BarHorizontal
)

// BarData represents a single bar or category in the chart.
type BarData struct {
	Label string
	Value float64
	Color cell.Color
}

// BarChart renders vertical and horizontal bar charts with customizable symbols and labels.
type BarChart struct {
	ID             string
	Data           []BarData
	Direction      BarChartDirection
	BarWidth       int
	BarGap         int
	Max            float64
	Min            float64
	ShowValues     bool
	ValueFormatter func(val float64) string
	Style          cell.Style
	LabelStyle     cell.Style
	DefaultColor   cell.Color
}

var verticalBlockSymbols = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Draw renders the bar chart to the buffer.
func (bc BarChart) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 4 || area.Height < 3 || len(bc.Data) == 0 {
		return
	}

	baseStyle := ctx.Style.Merge(bc.Style)
	labelStyle := baseStyle.Merge(bc.LabelStyle)
	if bc.LabelStyle == (cell.Style{}) {
		labelStyle = baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(180, 190, 205)})
	}
	defaultColor := bc.DefaultColor
	if defaultColor == 0 {
		defaultColor = cell.NewColorRGB(0, 200, 255)
	}

	// Calculate Min & Max
	maxVal := bc.Max
	minVal := bc.Min
	if maxVal <= minVal {
		maxVal = bc.Data[0].Value
		minVal = bc.Data[0].Value
		for _, d := range bc.Data {
			if d.Value > maxVal {
				maxVal = d.Value
			}
			if d.Value < minVal {
				minVal = d.Value
			}
		}
		if minVal > 0 {
			minVal = 0
		}
		if maxVal == minVal {
			maxVal = minVal + 10
		}
	}
	rangeVal := maxVal - minVal

	formatter := bc.ValueFormatter
	if formatter == nil {
		formatter = func(v float64) string {
			if math.Abs(v-math.Round(v)) < 1e-4 {
				return fmt.Sprintf("%d", int(v))
			}
			return fmt.Sprintf("%.1f", v)
		}
	}

	barWidth := bc.BarWidth
	if barWidth <= 0 {
		barWidth = 3
	}
	barGap := bc.BarGap
	if barGap < 0 {
		barGap = 1
	}

	if bc.Direction == BarVertical {
		// Vertical Bars: Labels at bottom, bars growing upwards
		labelHeight := uint16(1)
		chartHeight := int(area.Height) - int(labelHeight)
		if bc.ShowValues {
			chartHeight--
		}
		if chartHeight < 1 {
			return
		}

		cursorX := area.X + 1
		for _, bar := range bc.Data {
			if cursorX+uint16(barWidth) > area.X+area.Width {
				break
			}

			norm := (bar.Value - minVal) / rangeVal
			if norm < 0 {
				norm = 0
			}
			if norm > 1 {
				norm = 1
			}

			fullRows := int(norm * float64(chartHeight))
			fraction := (norm*float64(chartHeight) - float64(fullRows)) * 8.0
			subIndex := int(fraction)
			if subIndex < 0 {
				subIndex = 0
			}
			if subIndex > 8 {
				subIndex = 8
			}

			barColor := bar.Color
			if barColor == 0 {
				barColor = defaultColor
			}
			barStyle := cell.Style{Fg: barColor, Bg: baseStyle.Bg}

			// Value label above bar
			if bc.ShowValues && fullRows <= chartHeight {
				valText := formatter(bar.Value)
				valX := cursorX + uint16((barWidth-utf8.RuneCountInString(valText))/2)
				valY := area.Y + uint16(chartHeight-fullRows)
				if valY >= area.Y {
					buf.SetString(valX, valY, valText, labelStyle)
				}
			}

			// Draw full filled rows from bottom upwards
			for r := 0; r < fullRows; r++ {
				py := area.Y + uint16(chartHeight-1-r)
				if bc.ShowValues {
					py++
				}
				for w := 0; w < barWidth; w++ {
					buf.SetCell(cursorX+uint16(w), py, cell.Cell{Content: '█', Style: barStyle})
				}
			}

			// Partial top character
			if subIndex > 0 && fullRows < chartHeight {
				py := area.Y + uint16(chartHeight-1-fullRows)
				if bc.ShowValues {
					py++
				}
				for w := 0; w < barWidth; w++ {
					buf.SetCell(cursorX+uint16(w), py, cell.Cell{Content: verticalBlockSymbols[subIndex], Style: barStyle})
				}
			}

			// Category Label at bottom
			lblY := area.Y + area.Height - 1
			lblText := bar.Label
			if utf8.RuneCountInString(lblText) > barWidth {
				lblText = string([]rune(lblText)[:barWidth])
			}
			lblX := cursorX + uint16((barWidth-utf8.RuneCountInString(lblText))/2)
			buf.SetString(lblX, lblY, lblText, labelStyle)

			cursorX += uint16(barWidth + barGap)
		}

	} else {
		// Horizontal Bars: Categories on left, bars growing rightwards
		maxLabelWidth := 0
		for _, bar := range bc.Data {
			l := utf8.RuneCountInString(bar.Label)
			if l > maxLabelWidth {
				maxLabelWidth = l
			}
		}
		if maxLabelWidth > 12 {
			maxLabelWidth = 12
		}

		labelColWidth := uint16(maxLabelWidth + 2)
		maxBarWidth := int(area.Width) - int(labelColWidth) - 8
		if maxBarWidth < 4 {
			maxBarWidth = 4
		}

		cursorY := area.Y
		for _, bar := range bc.Data {
			if cursorY >= area.Y+area.Height {
				break
			}

			// Category label on left
			buf.SetString(area.X, cursorY, bar.Label, labelStyle)

			norm := (bar.Value - minVal) / rangeVal
			if norm < 0 {
				norm = 0
			}
			if norm > 1 {
				norm = 1
			}

			fullCols := int(norm * float64(maxBarWidth))
			barColor := bar.Color
			if barColor == 0 {
				barColor = defaultColor
			}
			barStyle := cell.Style{Fg: barColor, Bg: baseStyle.Bg}

			barStartX := area.X + labelColWidth
			for c := 0; c < fullCols; c++ {
				buf.SetCell(barStartX+uint16(c), cursorY, cell.Cell{Content: '█', Style: barStyle})
			}

			// Value text at end of bar
			if bc.ShowValues {
				valStr := " " + formatter(bar.Value)
				buf.SetString(barStartX+uint16(fullCols), cursorY, valStr, labelStyle)
			}

			cursorY += uint16(1 + barGap)
		}
	}
}

// SizeHint returns preferred dimensions for BarChart.
func (bc BarChart) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	if bc.Direction == BarVertical {
		w := uint16(len(bc.Data) * (bc.BarWidth + bc.BarGap))
		if w > maxArea.Width {
			w = maxArea.Width
		}
		return w, maxArea.Height
	}
	h := uint16(len(bc.Data) * (1 + bc.BarGap))
	if h > maxArea.Height {
		h = maxArea.Height
	}
	return maxArea.Width, h
}
