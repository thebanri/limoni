package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	timeTick := 0.0
	paused := false

	// Chart Data Buffers
	wave1 := make([]float64, 40)
	wave2 := make([]float64, 40)
	wave3 := make([]float64, 40)

	updateData := func() {
		if paused {
			return
		}
		timeTick += 0.15
		for i := 0; i < 40; i++ {
			x := timeTick + float64(i)*0.2
			wave1[i] = 50.0 + 35.0*math.Sin(x)
			wave2[i] = 45.0 + 30.0*math.Cos(x*0.7)
			wave3[i] = 30.0 + 20.0*math.Sin(x*1.3+1.0)
		}
	}
	updateData()

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			accent := cell.NewColorRGB(0, 255, 180)

			chunks := layout.VBox(area, layout.Fixed(3), layout.Fill(), layout.Fixed(1))

			// Header
			f.RenderWidget(widgets.Block{
				Title:          " 📊 LIMONI DATA VISUALIZATION STUDIO ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accent, Modifier: cell.ModifierBold},
				Child:          &widgets.Paragraph{Text: " Ultra-High Performance Braille Subpixel Curves, Bar Charts & Pie Visualizations "},
			}, chunks[0])

			// Body: Split into Top (LineChart) and Bottom (BarChart + PieChart)
			bodyRows := layout.VBox(chunks[1], layout.Percentage(50), layout.Percentage(50))

			// 1. Top Panel: LineChart (Multi-Series Real-Time Waveform)
			lineChart := widgets.LineChart{
				ID: "main_linechart",
				Datasets: []widgets.LineDataset{
					{
						Name:  "Alpha Wave (Sine)",
						Data:  wave1,
						Color: cell.NewColorRGB(0, 255, 180),
					},
					{
						Name:  "Beta Flux (Cosine)",
						Data:  wave2,
						Color: cell.NewColorRGB(0, 200, 255),
					},
					{
						Name:  "Gamma Drift (Harmonic)",
						Data:  wave3,
						Color: cell.NewColorRGB(255, 120, 50),
					},
				},
				ShowAxes:   true,
				ShowLegend: true,
				XLabels:    []string{"-10s", "-7.5s", "-5s", "-2.5s", "now"},
			}

			f.RenderWidget(widgets.Block{
				Title:         " REAL-TIME TELEMETRY — LineChart (Braille 2x4 Subpixels) ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 135, 160)},
				PaddingLeft:   1,
				PaddingRight:  1,
				Child:         lineChart,
			}, bodyRows[0])

			// 2. Bottom Row: BarChart (Left 50%) + PieChart (Right 50%)
			botCols := layout.HBox(bodyRows[1], layout.Percentage(50), layout.Percentage(50))

			barChart := widgets.BarChart{
				ID: "main_barchart",
				Data: []widgets.BarData{
					{Label: "CPU 0", Value: math.Abs(wave1[len(wave1)-1]), Color: cell.NewColorRGB(46, 204, 113)},
					{Label: "CPU 1", Value: math.Abs(wave2[len(wave2)-1]), Color: cell.NewColorRGB(52, 152, 219)},
					{Label: "Mem", Value: 68.5, Color: cell.NewColorRGB(241, 196, 15)},
					{Label: "Disk", Value: 42.0, Color: cell.NewColorRGB(230, 126, 34)},
					{Label: "Net", Value: math.Abs(wave3[len(wave3)-1]), Color: cell.NewColorRGB(233, 30, 99)},
				},
				Direction:  widgets.BarVertical,
				BarWidth:   5,
				BarGap:     2,
				ShowValues: true,
			}

			f.RenderWidget(widgets.Block{
				Title:         " RESOURCE SPECTRUM — BarChart ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 135, 160)},
				Child:         barChart,
			}, botCols[0])

			pieChart := widgets.PieChart{
				ID: "main_piechart",
				Data: []widgets.PieSlice{
					{Label: "Go Runtime", Value: 45.0, Color: cell.NewColorRGB(0, 200, 255)},
					{Label: "Rust Backend", Value: 30.0, Color: cell.NewColorRGB(255, 100, 50)},
					{Label: "TypeScript", Value: 15.0, Color: cell.NewColorRGB(50, 150, 255)},
					{Label: "Python", Value: 10.0, Color: cell.NewColorRGB(255, 215, 0)},
				},
				DonutHoleRatio:  0.45,
				ShowLegend:      true,
				ShowPercentages: true,
			}

			f.RenderWidget(widgets.Block{
				Title:         " ALLOCATION DISTRIBUTION — PieChart (Donut) ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 135, 160)},
				Child:         pieChart,
			}, botCols[1])

			// Footer
			statusText := " [Space] Pause/Resume Stream   [q] Quit"
			if paused {
				statusText = " [PAUSED] Press Space to resume   [q] Quit"
			}
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child: &widgets.Paragraph{
					Text: statusText,
					Style: cell.Style{
						Fg: cell.NewColorRGB(140, 150, 165),
					},
				},
			}, chunks[2])
		})
	}

	draw()

	renderTicker := time.NewTicker(40 * time.Millisecond) // 25 FPS
	defer renderTicker.Stop()

	for {
		select {
		case <-renderTicker.C:
			updateData()
			draw()
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if ev.Key.Type == backend.KeyRune {
					switch ev.Key.Ch {
					case 'q', 'Q':
						return
					case ' ':
						paused = !paused
					}
					draw()
				}
				if ev.Key.Type == backend.KeyEsc {
					return
				}
			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				draw()
			case backend.EventResize:
				draw()
			}
		}
	}
}
