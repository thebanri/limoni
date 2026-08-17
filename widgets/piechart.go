package widgets

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// PieSlice defines a single slice in a pie or donut chart.
type PieSlice struct {
	Label string
	Value float64
	Color cell.Color
}

// PieChart renders pie and donut charts using Braille subpixels and color legends.
type PieChart struct {
	ID              string
	Data            []PieSlice
	DonutHoleRatio  float64 // 0.0 for solid pie, 0.3 - 0.6 for donut
	ShowLegend      bool
	ShowPercentages bool
	Style           cell.Style
}

var defaultPieColors = []cell.Color{
	cell.NewColorRGB(52, 152, 219),  // Sky Blue
	cell.NewColorRGB(46, 204, 113),  // Emerald
	cell.NewColorRGB(241, 196, 15),  // Sunflower Yellow
	cell.NewColorRGB(231, 76, 60),   // Coral Red
	cell.NewColorRGB(155, 89, 182),  // Amethyst
	cell.NewColorRGB(26, 188, 156),  // Turquoise
	cell.NewColorRGB(230, 126, 34),  // Orange
	cell.NewColorRGB(149, 165, 166), // Silver
}

// Draw renders the pie/donut chart and legends.
func (pc PieChart) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 10 || area.Height < 6 || len(pc.Data) == 0 {
		return
	}

	baseStyle := ctx.Style.Merge(pc.Style)

	// Calculate total sum
	total := 0.0
	for _, slice := range pc.Data {
		if slice.Value > 0 {
			total += slice.Value
		}
	}
	if total <= 0 {
		return
	}

	// Layout: Chart on left, Legend on right (if space permits)
	chartW := area.Height * 2
	if chartW > area.Width {
		chartW = area.Width
	}
	chartH := area.Height
	if pc.ShowLegend && area.Width > chartW+14 {
		chartW = area.Height * 2
		if chartW > area.Width-16 {
			chartW = area.Width - 16
		}
	}

	// Subpixel dimensions
	virtW := int(chartW) * 2
	virtH := int(chartH) * 4
	canvas := NewCanvas(chartW, chartH)

	centerX := float64(virtW) / 2.0
	centerY := float64(virtH) / 2.0
	radius := math.Min(centerX, centerY) - 2.0
	innerRadius := radius * pc.DonutHoleRatio
	if innerRadius < 0 {
		innerRadius = 0
	}

	// Compute start and end angles for each slice
	type sliceAngle struct {
		startAngle float64
		endAngle   float64
		color      cell.Color
		percent    float64
		label      string
	}
	var angles []sliceAngle
	curAngle := -math.Pi / 2.0 // Start at top 12 o'clock

	for i, slice := range pc.Data {
		if slice.Value <= 0 {
			continue
		}
		pct := slice.Value / total
		sweep := pct * 2.0 * math.Pi

		col := slice.Color
		if col == 0 {
			col = defaultPieColors[i%len(defaultPieColors)]
		}

		angles = append(angles, sliceAngle{
			startAngle: curAngle,
			endAngle:   curAngle + sweep,
			color:      col,
			percent:    pct * 100.0,
			label:      slice.Label,
		})
		curAngle += sweep
	}

	// Rasterize Braille subpixels
	for y := 0; y < virtH; y++ {
		for x := 0; x < virtW; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist >= innerRadius && dist <= radius {
				angle := math.Atan2(dy, dx)
				if angle < -math.Pi/2.0 {
					angle += 2.0 * math.Pi
				}

				for _, sa := range angles {
					if angle >= sa.startAngle && angle <= sa.endAngle {
						canvas.Set(x, y, cell.Style{Fg: sa.color})
						break
					}
				}
			}
		}
	}

	// Render Canvas
	canvas.Draw(cell.NewContext(cell.NewRect(area.X, area.Y, chartW, chartH), baseStyle), buf)

	// Draw Legend on right
	if pc.ShowLegend && area.Width > chartW+4 {
		legendX := area.X + chartW + 2
		legendMaxW := area.Width - (chartW + 2)

		for i, sa := range angles {
			ly := area.Y + uint16(i)
			if ly >= area.Y+area.Height {
				break
			}

			dotStyle := cell.Style{Fg: sa.color, Modifier: cell.ModifierBold}
			buf.SetString(legendX, ly, "● ", dotStyle)

			text := sa.label
			if pc.ShowPercentages {
				text += fmt.Sprintf(" (%.1f%%)", sa.percent)
			}
			if utf8.RuneCountInString(text) > int(legendMaxW)-2 {
				text = string([]rune(text)[:legendMaxW-2])
			}
			buf.SetString(legendX+2, ly, text, baseStyle)
		}
	}
}

// SizeHint returns the preferred dimensions for PieChart.
func (pc PieChart) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}
