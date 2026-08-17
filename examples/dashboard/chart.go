package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

type ChartMode int

const (
	ChartModeLine ChartMode = iota
	ChartModeBar
	ChartModeSparkline
	ChartModeArea
)

var chartModeNames = []string{"Line (Braille)", "Spectrum (Bar)", "Quad Grid (Spark)", "Shaded Area"}

var blockRunes = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func interpolateColor(c1, c2 cell.Color, t float64) cell.Color {
	if t <= 0 {
		return c1
	}
	if t >= 1 {
		return c2
	}
	r1, g1, b1 := c1.RGB()
	r2, g2, b2 := c2.RGB()

	r := uint8(float64(r1)*(1.0-t) + float64(r2)*t)
	g := uint8(float64(g1)*(1.0-t) + float64(g2)*t)
	b := uint8(float64(b1)*(1.0-t) + float64(b2)*t)
	return cell.NewColorRGB(r, g, b)
}

func getCPUGradient(level float64) cell.Color {
	mint := cell.NewColorRGB(0, 255, 140)
	cyan := cell.NewColorRGB(0, 220, 255)
	amber := cell.NewColorRGB(255, 190, 0)
	flame := cell.NewColorRGB(255, 45, 85)

	if level < 0.4 {
		return interpolateColor(mint, cyan, level/0.4)
	} else if level < 0.7 {
		return interpolateColor(cyan, amber, (level-0.4)/0.3)
	} else {
		return interpolateColor(amber, flame, (level-0.7)/0.3)
	}
}

func getRAMGradient(level float64) cell.Color {
	sky := cell.NewColorRGB(0, 210, 255)
	purple := cell.NewColorRGB(145, 40, 255)
	pink := cell.NewColorRGB(255, 0, 130)

	if level < 0.5 {
		return interpolateColor(sky, purple, level/0.5)
	} else {
		return interpolateColor(purple, pink, (level-0.5)/0.5)
	}
}

type SeriesStats struct {
	Min  float64
	Max  float64
	Avg  float64
	Last float64
	P95  float64
}

func calcStats(history []float64) SeriesStats {
	if len(history) == 0 {
		return SeriesStats{}
	}
	minVal := history[0]
	maxVal := history[0]
	sum := 0.0

	sorted := make([]float64, len(history))
	copy(sorted, history)
	sort.Float64s(sorted)

	for _, v := range history {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
		sum += v
	}

	p95Idx := int(float64(len(sorted)-1) * 0.95)
	return SeriesStats{
		Min:  minVal,
		Max:  maxVal,
		Avg:  sum / float64(len(history)),
		Last: history[len(history)-1],
		P95:  sorted[p95Idx],
	}
}

func calcMovingAverage(data []float64, k int) []float64 {
	if len(data) == 0 || k <= 1 {
		return data
	}
	res := make([]float64, len(data))
	for i := range data {
		start := i - k + 1
		if start < 0 {
			start = 0
		}
		sum := 0.0
		count := 0
		for j := start; j <= i; j++ {
			sum += data[j]
			count++
		}
		res[i] = sum / float64(count)
	}
	return res
}

type SystemChart struct {
	Mode        ChartMode
	CPUHistory  []float64
	RAMHistory  []float64
	DiskHistory []float64
	NetHistory  []float64
	CPUPeaks    []float64
	RAMPeaks    []float64
	CurrentCPU  float64
	CurrentRAM  float64
	UsedRAMGB   float64
	TotalRAMGB  float64
	TextColor   cell.Color

	canvas *widgets.Canvas
}

func (sc *SystemChart) NextMode() {
	sc.Mode = (sc.Mode + 1) % 4
}

func (sc *SystemChart) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 15 || area.Height < 4 {
		return
	}

	textColor := sc.TextColor
	if textColor.Type() == cell.ColorDefault {
		textColor = cell.NewColorRGB(230, 235, 245)
	}

	cpuStats := calcStats(sc.CPUHistory)
	ramStats := calcStats(sc.RAMHistory)

	headerY := area.Y
	buf.SetString(area.X, headerY, "PERFORMANCE", cell.Style{Fg: textColor, Modifier: cell.ModifierBold})

	modeBadge := fmt.Sprintf(" [Mode: %s (m)] ", chartModeNames[sc.Mode])
	buf.SetString(area.X+13, headerY, modeBadge, cell.Style{Fg: cell.NewColorRGB(180, 185, 205), Bg: cell.NewColorRGB(25, 30, 42)})

	cpuBadge := fmt.Sprintf(" CPU %4.1f%% (P95: %4.1f%%) ", sc.CurrentCPU, cpuStats.P95)
	cpuBadgeStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 140), Bg: cell.NewColorRGB(10, 35, 20), Modifier: cell.ModifierBold}

	ramBadge := fmt.Sprintf(" RAM %4.1f%% (%.1fG/%.1fG) ", sc.CurrentRAM, sc.UsedRAMGB, sc.TotalRAMGB)
	ramBadgeStyle := cell.Style{Fg: cell.NewColorRGB(0, 210, 255), Bg: cell.NewColorRGB(10, 25, 45), Modifier: cell.ModifierBold}

	currX := area.X + area.Width
	if int(area.Width) > len([]rune(cpuBadge))+len([]rune(ramBadge))+34 {
		currX -= uint16(len([]rune(ramBadge)))
		buf.SetString(currX, headerY, ramBadge, ramBadgeStyle)
		currX -= 1

		currX -= uint16(len([]rune(cpuBadge)))
		buf.SetString(currX, headerY, cpuBadge, cpuBadgeStyle)
	}

	plotArea := cell.NewRect(area.X, area.Y+1, area.Width, area.Height-1)
	if plotArea.Width < 4 || plotArea.Height < 2 {
		return
	}

	switch sc.Mode {
	case ChartModeLine:
		sc.drawLineChart(ctx, buf, plotArea)
	case ChartModeBar:
		sc.drawSideBySideSpectrum(buf, plotArea)
	case ChartModeSparkline:
		sc.drawSparklineGrid(buf, plotArea, cpuStats, ramStats)
	case ChartModeArea:
		sc.drawAreaChart(ctx, buf, plotArea)
	}
}

// MODE 1: Line Chart (Braille Vector Curves + Moving Average + Grid)
func (sc *SystemChart) drawLineChart(ctx cell.Context, buf *buffer.Buffer, area cell.Rect) {
	yAxisW := uint16(5)
	if area.Width <= yAxisW+6 {
		return
	}
	plotW := area.Width - yAxisW
	plotH := area.Height
	plotX := area.X + yAxisW
	plotY := area.Y

	axisStyle := cell.Style{Fg: cell.NewColorRGB(110, 115, 135)}
	gridStyle := cell.Style{Fg: cell.NewColorRGB(28, 32, 44)}

	buf.SetString(area.X, plotY, "100", axisStyle)
	buf.SetString(area.X+3, plotY, "┼", axisStyle)

	if plotH >= 5 {
		q3Y := plotY + (plotH / 4)
		buf.SetString(area.X, q3Y, " 75", axisStyle)
		buf.SetString(area.X+3, q3Y, "┼", axisStyle)

		midY := plotY + (plotH / 2)
		buf.SetString(area.X, midY, " 50", axisStyle)
		buf.SetString(area.X+3, midY, "┼", axisStyle)

		q1Y := plotY + (plotH * 3 / 4)
		buf.SetString(area.X, q1Y, " 25", axisStyle)
		buf.SetString(area.X+3, q1Y, "┼", axisStyle)
	}

	botY := plotY + plotH - 1
	buf.SetString(area.X, botY, "  0", axisStyle)
	buf.SetString(area.X+3, botY, "┴", axisStyle)

	for r := uint16(0); r < plotH; r++ {
		curY := plotY + r
		isGridRow := (r == 0) || (plotH >= 5 && (r == plotH/4 || r == plotH/2 || r == plotH*3/4))
		for c := uint16(0); c < plotW; c++ {
			cObj := buf.Get(plotX+c, curY)
			if cObj != nil {
				if isGridRow {
					cObj.Content = '┄'
					cObj.Style = gridStyle
				} else {
					cObj.Content = ' '
					cObj.Style = cell.Style{}
				}
			}
		}
	}

	if sc.canvas == nil {
		sc.canvas = widgets.NewCanvas(plotW, plotH)
	} else {
		sc.canvas.Reset(plotW, plotH)
	}
	canvas := sc.canvas

	virtW := int(plotW) * 2
	virtH := int(plotH) * 4

	cpuCol := cell.Style{Fg: cell.NewColorRGB(0, 255, 140)}
	cpuAvgCol := cell.Style{Fg: cell.NewColorRGB(0, 150, 80)}
	ramCol := cell.Style{Fg: cell.NewColorRGB(0, 210, 255)}
	ramAvgCol := cell.Style{Fg: cell.NewColorRGB(0, 110, 170)}

	drawCurve := func(data []float64, lineStyle, avgStyle cell.Style) {
		numPoints := len(data)
		if numPoints < 2 {
			return
		}

		if numPoints > 6 {
			avg := calcMovingAverage(data, 8)
			for i := 0; i < numPoints-1; i++ {
				x0 := int(float64(i) / float64(numPoints-1) * float64(virtW-1))
				y0 := int((1.0 - (avg[i] / 100.0)) * float64(virtH-1))
				x1 := int(float64(i+1) / float64(numPoints-1) * float64(virtW-1))
				y1 := int((1.0 - (avg[i+1] / 100.0)) * float64(virtH-1))
				if y0 < 0 {
					y0 = 0
				}
				if y0 >= virtH {
					y0 = virtH - 1
				}
				if y1 < 0 {
					y1 = 0
				}
				if y1 >= virtH {
					y1 = virtH - 1
				}
				canvas.DrawLine(x0, y0, x1, y1, avgStyle)
			}
		}

		for i := 0; i < numPoints-1; i++ {
			x0 := int(float64(i) / float64(numPoints-1) * float64(virtW-1))
			y0 := int((1.0 - (data[i] / 100.0)) * float64(virtH-1))
			x1 := int(float64(i+1) / float64(numPoints-1) * float64(virtW-1))
			y1 := int((1.0 - (data[i+1] / 100.0)) * float64(virtH-1))
			if y0 < 0 {
				y0 = 0
			}
			if y0 >= virtH {
				y0 = virtH - 1
			}
			if y1 < 0 {
				y1 = 0
			}
			if y1 >= virtH {
				y1 = virtH - 1
			}
			canvas.DrawLine(x0, y0, x1, y1, lineStyle)
		}
	}

	drawCurve(sc.RAMHistory, ramCol, ramAvgCol)
	drawCurve(sc.CPUHistory, cpuCol, cpuAvgCol)

	plotContext := cell.NewContext(cell.NewRect(plotX, plotY, plotW, plotH), ctx.Style)
	canvas.Draw(plotContext, buf)
}

// MODE 2: Side-by-Side Full-Height Spectrum Columns (Left: CPU, Right: RAM)
func (sc *SystemChart) drawSideBySideSpectrum(buf *buffer.Buffer, area cell.Rect) {
	halfW := area.Width / 2
	if halfW < 8 || area.Height < 2 {
		return
	}

	leftArea := cell.NewRect(area.X, area.Y, halfW-1, area.Height)
	rightArea := cell.NewRect(area.X+halfW, area.Y, area.Width-halfW, area.Height)

	sepX := area.X + halfW - 1
	sepStyle := cell.Style{Fg: cell.NewColorRGB(50, 55, 70)}
	for r := uint16(0); r < area.Height; r++ {
		buf.SetCell(sepX, area.Y+r, cell.Cell{Content: '│', Style: sepStyle})
	}

	if len(sc.CPUPeaks) != int(leftArea.Width) {
		sc.CPUPeaks = make([]float64, leftArea.Width)
	}
	if len(sc.RAMPeaks) != int(rightArea.Width) {
		sc.RAMPeaks = make([]float64, rightArea.Width)
	}

	sc.renderSpectrumCard(buf, leftArea, sc.CPUHistory, sc.CPUPeaks, "CPU LOAD SPECTRUM", sc.CurrentCPU, "%", getCPUGradient, cell.NewColorRGB(0, 255, 140))
	sc.renderSpectrumCard(buf, rightArea, sc.RAMHistory, sc.RAMPeaks, "RAM SPECTRUM", sc.CurrentRAM, "%", getRAMGradient, cell.NewColorRGB(0, 210, 255))
}

func (sc *SystemChart) renderSpectrumCard(buf *buffer.Buffer, area cell.Rect, history, peaks []float64, title string, currentVal float64, unit string, gradFn func(float64) cell.Color, accentCol cell.Color) {
	w, h := area.Width, area.Height
	if w < 4 || h < 2 {
		return
	}

	for r := uint16(0); r < h; r++ {
		curY := area.Y + r
		for c := uint16(0); c < w; c++ {
			cObj := buf.Get(area.X+c, curY)
			if cObj != nil {
				cObj.Content = ' '
				cObj.Style = cell.Style{}
			}
		}
	}

	cardTitle := fmt.Sprintf("[%s: %4.1f%s]", title, currentVal, unit)
	buf.SetString(area.X+1, area.Y, cardTitle, cell.Style{Fg: accentCol, Modifier: cell.ModifierBold})

	plotY := area.Y + 1
	plotH := h - 1
	if plotH < 1 {
		return
	}

	numPoints := len(history)
	if numPoints == 0 {
		return
	}

	for c := uint16(0); c < w; c++ {
		targetX := area.X + c
		dataIdx := int(float64(c) / float64(w-1) * float64(numPoints-1))
		if dataIdx < 0 {
			dataIdx = 0
		}
		if dataIdx >= numPoints {
			dataIdx = numPoints - 1
		}

		val := history[dataIdx]
		if val < 0 {
			val = 0
		}
		if val > 100 {
			val = 100
		}

		if val >= peaks[c] {
			peaks[c] = val
		} else {
			peaks[c] = peaks[c] - 0.8
			if peaks[c] < 0 {
				peaks[c] = 0
			}
		}

		totalLevels := (val / 100.0) * float64(plotH)
		peakRow := (peaks[c] / 100.0) * float64(plotH)

		for r := uint16(0); r < plotH; r++ {
			rowFromBottom := float64(plotH - 1 - r)
			curY := plotY + r

			gradLevel := float64(plotH-1-r) / float64(plotH-1)
			blockStyle := cell.Style{Fg: gradFn(gradLevel)}

			if totalLevels >= rowFromBottom+1.0 {
				buf.SetCell(targetX, curY, cell.Cell{Content: '█', Style: blockStyle})
			} else if totalLevels > rowFromBottom {
				subIdx := int(math.Round((totalLevels - rowFromBottom) * 8.0))
				if subIdx < 1 {
					subIdx = 1
				}
				if subIdx > 8 {
					subIdx = 8
				}
				buf.SetCell(targetX, curY, cell.Cell{Content: blockRunes[subIdx], Style: blockStyle})
			} else if peakRow >= rowFromBottom && peakRow < rowFromBottom+1.0 && peaks[c] > 3.0 {
				buf.SetCell(targetX, curY, cell.Cell{Content: '▔', Style: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold}})
			}
		}
	}
}

// MODE 3: Quad Sparkline Telemetry Grid (2x2 Cards: CPU, RAM, Disk, Net)
func (sc *SystemChart) drawSparklineGrid(buf *buffer.Buffer, area cell.Rect, cpuStats, ramStats SeriesStats) {
	cardW := area.Width / 2
	cardH := area.Height / 2
	if cardW < 8 || cardH < 2 {
		return
	}

	diskStats := SeriesStats{Min: 30, Max: 35, Avg: 33, Last: 33, P95: 34}
	netStats := SeriesStats{Min: 0.1, Max: 8.5, Avg: 2.4, Last: 1.8, P95: 6.2}

	sc.drawMiniSparkCard(buf, area.X, area.Y, cardW-1, cardH, "CPU LOAD", sc.CPUHistory, cpuStats, "%", cell.NewColorRGB(0, 255, 140))
	sc.drawMiniSparkCard(buf, area.X+cardW, area.Y, area.Width-cardW, cardH, "MEMORY (RAM)", sc.RAMHistory, ramStats, "%", cell.NewColorRGB(0, 210, 255))
	sc.drawMiniSparkCard(buf, area.X, area.Y+cardH, cardW-1, area.Height-cardH, "DISK I/O (/)", sc.DiskHistory, diskStats, "%", cell.NewColorRGB(180, 120, 255))
	sc.drawMiniSparkCard(buf, area.X+cardW, area.Y+cardH, area.Width-cardW, area.Height-cardH, "NETWORK (Rx)", sc.NetHistory, netStats, "M", cell.NewColorRGB(255, 190, 0))
}

func (sc *SystemChart) drawMiniSparkCard(buf *buffer.Buffer, x, y, w, h uint16, title string, data []float64, stats SeriesStats, unit string, col cell.Color) {
	if w < 6 || h < 2 {
		return
	}

	for r := uint16(0); r < h; r++ {
		for c := uint16(0); c < w; c++ {
			cObj := buf.Get(x+c, y+r)
			if cObj != nil {
				cObj.Content = ' '
				cObj.Style = cell.Style{}
			}
		}
	}

	titleText := fmt.Sprintf("[%s] %4.1f%s (Avg: %4.1f%s)", title, stats.Last, unit, stats.Avg, unit)
	buf.SetString(x+1, y, titleText, cell.Style{Fg: col, Modifier: cell.ModifierBold})

	sparkY := y + 1
	sparkH := h - 1
	numPoints := len(data)
	if numPoints == 0 || sparkH < 1 {
		return
	}

	for c := uint16(0); c < w-2; c++ {
		targetX := x + 1 + c
		dataIdx := int(float64(c) / float64(w-3) * float64(numPoints-1))
		if dataIdx < 0 {
			dataIdx = 0
		}
		if dataIdx >= numPoints {
			dataIdx = numPoints - 1
		}

		val := data[dataIdx]
		if val < 0 {
			val = 0
		}
		if val > 100 {
			val = 100
		}

		levels := (val / 100.0) * float64(sparkH)
		for r := uint16(0); r < sparkH; r++ {
			rowFromBottom := float64(sparkH - 1 - r)
			curY := sparkY + r

			if levels >= rowFromBottom+1.0 {
				buf.SetCell(targetX, curY, cell.Cell{Content: '█', Style: cell.Style{Fg: col}})
			} else if levels > rowFromBottom {
				subIdx := int((levels - rowFromBottom) * 8.0)
				if subIdx < 1 {
					subIdx = 1
				}
				if subIdx > 8 {
					subIdx = 8
				}
				buf.SetCell(targetX, curY, cell.Cell{Content: blockRunes[subIdx], Style: cell.Style{Fg: col}})
			}
		}
	}
}

// MODE 4: Area Chart (Sub-cell Dithered Shading)
func (sc *SystemChart) drawAreaChart(ctx cell.Context, buf *buffer.Buffer, area cell.Rect) {
	if sc.canvas == nil {
		sc.canvas = widgets.NewCanvas(area.Width, area.Height)
	} else {
		sc.canvas.Reset(area.Width, area.Height)
	}
	canvas := sc.canvas

	virtW := int(area.Width) * 2
	virtH := int(area.Height) * 4

	cpuLine := cell.Style{Fg: cell.NewColorRGB(0, 255, 140)}
	cpuArea := cell.Style{Fg: cell.NewColorRGB(0, 75, 40)}
	ramLine := cell.Style{Fg: cell.NewColorRGB(0, 210, 255)}
	ramArea := cell.Style{Fg: cell.NewColorRGB(0, 55, 95)}

	drawFilledArea := func(history []float64, lineStyle, areaStyle cell.Style) {
		if len(history) < 2 {
			return
		}
		numPoints := len(history)
		points := make([][2]int, numPoints)

		for i, val := range history {
			if val < 0 {
				val = 0
			}
			if val > 100 {
				val = 100
			}
			x := int(float64(i) / float64(numPoints-1) * float64(virtW-1))
			y := int((1.0 - (val / 100.0)) * float64(virtH-1))
			if y < 0 {
				y = 0
			}
			if y >= virtH {
				y = virtH - 1
			}
			points[i] = [2]int{x, y}
		}

		for i := 0; i < len(points)-1; i++ {
			x0, y0 := points[i][0], points[i][1]
			x1, y1 := points[i+1][0], points[i+1][1]
			steps := x1 - x0
			if steps <= 0 {
				steps = 1
			}

			for s := 0; s <= steps; s++ {
				t := float64(s) / float64(steps)
				cx := x0 + s
				cy := int(float64(y0)*(1.0-t) + float64(y1)*t)
				for fy := cy + 1; fy < virtH; fy++ {
					if (cx+fy)%2 == 0 {
						canvas.Set(cx, fy, areaStyle)
					}
				}
			}
			canvas.DrawLine(x0, y0, x1, y1, lineStyle)
		}
	}

	if len(sc.RAMHistory) > 0 {
		drawFilledArea(sc.RAMHistory, ramLine, ramArea)
	}
	if len(sc.CPUHistory) > 0 {
		drawFilledArea(sc.CPUHistory, cpuLine, cpuArea)
	}

	plotContext := cell.NewContext(area, ctx.Style)
	canvas.Draw(plotContext, buf)
}

func (sc *SystemChart) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}
