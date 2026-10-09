// dashboard_gauges shows the gauge widgets on a small, simulated build-farm
// dashboard: BigText for the headline number, Gauge for the machine's load,
// LineGauge for jobs in progress, and a LineChart drawn with sextants.
//
//	go run ./examples/dashboard_gauges
//
// Keys: Space pauses, m cycles the chart's marker, q or Esc quits.
//
// The numbers are made up from sine waves, so the example behaves the same
// on every machine; main_test.go draws it with uitest.
package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

const historyLen = 120

var (
	accent = cell.NewColorRGB(0, 220, 170)
	blue   = cell.NewColorRGB(0, 170, 255)
	orange = cell.NewColorRGB(255, 150, 60)
	muted  = cell.Style{Fg: cell.NewColorRGB(120, 135, 160)}
)

var markers = []struct {
	marker widgets.Marker
	name   string
}{
	{widgets.MarkerSextant, "sextant"},
	{widgets.MarkerBraille, "braille"},
	{widgets.MarkerQuadrant, "quadrant"},
	{widgets.MarkerHalfBlock, "half block"},
	{widgets.MarkerBlock, "block"},
}

// dashboard is the application state. Its draw method is what limoni.Run
// calls, which is also what lets main_test.go drive it with uitest.
type dashboard struct {
	tick   int
	paused bool
	marker int

	builds   int
	cpu, mem []float64
}

func newDashboard() *dashboard {
	d := &dashboard{}
	// Fill the chart before the first frame, so it never starts empty.
	for range historyLen {
		d.step()
	}
	return d
}

// step advances the simulation by one sample.
func (d *dashboard) step() {
	d.tick++
	x := float64(d.tick) / 10
	cpu := 55 + 30*math.Sin(x) + 8*math.Sin(x*3.7)
	mem := 62 + 12*math.Sin(x*0.4+1) + 3*math.Sin(x*2.3)
	d.cpu = appendCapped(d.cpu, cpu)
	d.mem = appendCapped(d.mem, mem)
	if d.tick%7 == 0 {
		d.builds++
	}
}

func appendCapped(s []float64, v float64) []float64 {
	s = append(s, v)
	if len(s) > historyLen {
		s = s[len(s)-historyLen:]
	}
	return s
}

func last(s []float64) float64 { return s[len(s)-1] }

// job is a progress ratio that loops, offset per job.
func (d *dashboard) job(period, offset int) float64 {
	return float64((d.tick+offset)%period) / float64(period-1)
}

func (d *dashboard) draw(f *limoni.Frame, ev *limoni.Event) bool {
	if ev == nil {
		if !d.paused {
			d.step()
		}
	} else if ev.Type == limoni.EventKey {
		switch ev.Key.Type {
		case limoni.KeyEsc:
			return false
		case limoni.KeySpace:
			d.paused = !d.paused
		case limoni.KeyRune:
			switch ev.Key.Ch {
			case 'q', 'Q':
				return false
			case ' ':
				d.paused = !d.paused
			case 'm', 'M':
				d.marker = (d.marker + 1) % len(markers)
			}
		}
	}

	rows := layout.VBox(f.Area(), layout.Fixed(6), layout.Fill(), layout.Fixed(1))
	top := layout.HBox(rows[0], layout.Fixed(38), layout.Fill())
	body := layout.HBox(rows[1], layout.Fixed(38), layout.Fill())

	panel := func(title string, child widgets.Widget) widgets.Block {
		return widgets.Block{
			Title:         title,
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   muted,
			PaddingLeft:   1,
			PaddingRight:  1,
			Child:         child,
		}
	}

	f.RenderWidget(panel(" Builds today — BigText ", widgets.BigText{
		ID:        "builds",
		Text:      strconv.Itoa(1200 + d.builds),
		Size:      widgets.BigTextHalfHeight,
		Alignment: widgets.AlignCenter,
		Style:     cell.Style{Fg: accent},
	}), top[0])

	f.RenderWidget(panel(" Load — Gauge ", &gauges{
		{ID: "cpu", Ratio: last(d.cpu) / 100, Label: fmt.Sprintf("CPU %.0f%%", last(d.cpu)), GaugeStyle: cell.Style{Fg: blue}},
		{ID: "mem", Ratio: last(d.mem) / 100, Label: fmt.Sprintf("Memory %.0f%%", last(d.mem)), GaugeStyle: cell.Style{Fg: orange}},
	}), top[1])

	f.RenderWidget(panel(" Jobs — LineGauge ", &lineGauges{
		{ID: "job-api", Ratio: d.job(40, 0), Label: "api      ", FilledStyle: cell.Style{Fg: accent}, UnfilledStyle: muted},
		{ID: "job-web", Ratio: d.job(55, 20), Label: "web      ", FilledStyle: cell.Style{Fg: blue}, UnfilledStyle: muted},
		{ID: "job-docs", Ratio: d.job(25, 5), Label: "docs     ", FilledStyle: cell.Style{Fg: orange}, UnfilledStyle: muted},
		{ID: "job-wasm", Ratio: d.job(70, 50), Label: "wasm     ", FilledStyle: cell.Style{Fg: accent}, UnfilledStyle: muted},
	}), body[0])

	m := markers[d.marker]
	f.RenderWidget(panel(" History — LineChart, "+m.name+" marker ", widgets.LineChart{
		ID: "history",
		Datasets: []widgets.LineDataset{
			{Name: "CPU", Data: d.cpu, Color: blue},
			{Name: "Memory", Data: d.mem, Color: orange},
		},
		MinY:       0,
		MaxY:       100,
		ShowAxes:   true,
		ShowLegend: true,
		Marker:     m.marker,
	}), body[1])

	state := "running"
	if d.paused {
		state = "paused"
	}
	f.RenderWidget(widgets.StatusBar{
		Left: []widgets.StatusItem{
			{Key: "Space", Text: "pause"},
			{Key: "m", Text: "marker"},
			{Key: "q", Text: "quit"},
		},
		Right:    []widgets.StatusItem{{Text: state}},
		Style:    muted,
		KeyStyle: cell.Style{Fg: accent, Modifier: cell.ModifierBold},
	}, rows[2])
	return true
}

// gauges stacks Gauges one row apart, so a Block can hold them as one child.
type gauges []widgets.Gauge

func (g *gauges) Draw(ctx cell.Context, buf *limoni.Buffer) {
	for i, gauge := range *g {
		y := ctx.Area.Y + uint16(2*i)
		if y >= ctx.Area.Y+ctx.Area.Height {
			return
		}
		row := ctx
		row.Area = cell.NewRect(ctx.Area.X, y, ctx.Area.Width, 1)
		gauge.Draw(row, buf)
	}
}

func (g *gauges) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, min(uint16(2*len(*g)), maxArea.Height)
}

// lineGauges stacks LineGauges one row apart.
type lineGauges []widgets.LineGauge

func (g *lineGauges) Draw(ctx cell.Context, buf *limoni.Buffer) {
	for i, gauge := range *g {
		y := ctx.Area.Y + 1 + uint16(2*i)
		if y >= ctx.Area.Y+ctx.Area.Height {
			return
		}
		row := ctx
		row.Area = cell.NewRect(ctx.Area.X, y, ctx.Area.Width, 1)
		gauge.Draw(row, buf)
	}
}

func (g *lineGauges) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, min(uint16(2*len(*g)+1), maxArea.Height)
}

func main() {
	err := limoni.Run(newDashboard().draw, limoni.WithFPS(5), limoni.WithTitle("Limoni gauges"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "limoni: %v\n", err)
		os.Exit(1)
	}
}
