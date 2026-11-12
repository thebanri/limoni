package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

var chartThemes = []struct {
	name  string
	theme widgets.Theme
}{
	{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
}

// drawThemed renders w in a width×height frame under theme and returns the
// frame's buffer.
func drawThemed(theme widgets.Theme, w widgets.Widget, width, height uint16) *buffer.Buffer {
	term := testkit.NewTerminal(width, height)
	term.Draw(func(f *terminal.Frame) { f.SetTheme(theme); f.RenderWidget(w, term.Area()) })
	return term.Frame().Buffer
}

func TestBarChartFrameTheme(t *testing.T) {
	red := cell.NewColorRGB(240, 80, 120)
	custom := cell.Style{Fg: cell.NewColorRGB(80, 220, 140), Modifier: cell.ModifierBold}
	for _, preset := range chartThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary := preset.theme.RoleStyle("primary").Fg
			muted := surface.Merge(preset.theme.RoleStyle("muted"))
			for _, tc := range []struct {
				name  string
				chart widgets.BarChart
				bar   cell.Color
				label cell.Style
			}{
				{"default", widgets.BarChart{}, primary, muted},
				{"explicit default color", widgets.BarChart{DefaultColor: red}, red, muted},
				{"explicit label style", widgets.BarChart{LabelStyle: custom}, primary, surface.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					for _, dir := range []struct {
						name           string
						direction      widgets.BarChartDirection
						barX, barY     uint16
						labelX, labelY uint16
					}{
						// Vertical: the bar's bottom row, and its label under it.
						{"vertical", widgets.BarVertical, 1, 4, 2, 5},
						// Horizontal: the label at the left, the bar after it.
						{"horizontal", widgets.BarHorizontal, 3, 0, 0, 0},
					} {
						t.Run(dir.name, func(t *testing.T) {
							chart := tc.chart
							chart.Direction = dir.direction
							chart.Data = []widgets.BarData{{Label: "a", Value: 8}, {Label: "b", Value: 4, Color: red}}
							buf := drawThemed(preset.theme, chart, 20, 6)
							want := cell.Style{Fg: tc.bar, Bg: surface.Bg}
							if got := buf.CellAt(dir.barX, dir.barY); got.Content != '█' || got.Style != want {
								t.Errorf("bar cell(%d,%d) = %+v; want '█' with %+v", dir.barX, dir.barY, got, want)
							}
							if got := buf.CellAt(dir.labelX, dir.labelY); got.Content != 'a' || got.Style != tc.label {
								t.Errorf("label cell(%d,%d) = %+v; want 'a' with %+v", dir.labelX, dir.labelY, got, tc.label)
							}
						})
					}
				})
			}
		})
	}

	// The two themes give the bars different colours, so they follow the theme.
	dark, light := widgets.DarkTheme().RoleStyle("primary").Fg, widgets.LightTheme().RoleStyle("primary").Fg
	if dark == light {
		t.Fatalf("dark and light themes share the primary colour %v", dark)
	}
}

func TestLineChartFrameTheme(t *testing.T) {
	red := cell.NewColorRGB(240, 80, 120)
	custom := cell.Style{Fg: cell.NewColorRGB(80, 220, 140), Modifier: cell.ModifierBold}
	for _, preset := range chartThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary := preset.theme.RoleStyle("primary").Fg
			muted := surface.Merge(preset.theme.RoleStyle("muted"))
			for _, tc := range []struct {
				name  string
				color cell.Color
				axis  cell.Style
				line  cell.Color
			}{
				{"default", 0, cell.Style{}, primary},
				{"explicit color", red, cell.Style{}, red},
				{"explicit axis style", 0, custom, primary},
			} {
				t.Run(tc.name, func(t *testing.T) {
					wantAxis := muted
					if tc.axis != (cell.Style{}) {
						wantAxis = surface.Merge(tc.axis)
					}
					chart := widgets.LineChart{
						Datasets:   []widgets.LineDataset{{Name: "in", Data: []float64{0, 10}, Color: tc.color}},
						ShowAxes:   true,
						ShowLegend: true,
						AxisStyle:  tc.axis,
					}
					buf := drawThemed(preset.theme, chart, 20, 8)

					// Legend dot, at the left of the plot, on the theme's surface.
					wantDot := surface.Merge(cell.Style{Fg: tc.line, Modifier: cell.ModifierBold})
					if got := buf.CellAt(6, 0); got.Content != '●' || got.Style != wantDot {
						t.Errorf("legend dot = %+v; want '●' with %+v", got, wantDot)
					}
					// The vertical axis, one column left of the plot.
					if got := buf.CellAt(5, 1); got.Content != '│' || got.Style != wantAxis {
						t.Errorf("axis cell = %+v; want '│' with %+v", got, wantAxis)
					}
					// The line rises from the bottom left corner of the plot.
					if got := buf.CellAt(6, 6); got.Content == ' ' || got.Style.Fg != tc.line {
						t.Errorf("line cell = %+v; want a dot in %v", got, tc.line)
					}
				})
			}
		})
	}
}

// PieChart keeps its own palette, but its legend sits on the theme's surface.
func TestPieChartLegendFrameTheme(t *testing.T) {
	red := cell.NewColorRGB(240, 80, 120)
	for _, preset := range chartThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			chart := widgets.PieChart{Data: []widgets.PieSlice{{Label: "a", Value: 1, Color: red}}, ShowLegend: true}
			buf := drawThemed(preset.theme, chart, 40, 6)
			// The chart is twice as wide as it is tall; the legend starts two
			// columns after it.
			want := surface.Merge(cell.Style{Fg: red, Modifier: cell.ModifierBold})
			if got := buf.CellAt(14, 0); got.Content != '●' || got.Style != want {
				t.Errorf("legend dot = %+v; want '●' with %+v", got, want)
			}
			if got := buf.CellAt(16, 0); got.Content != 'a' || got.Style != surface {
				t.Errorf("legend label = %+v; want 'a' with %+v", got, surface)
			}
		})
	}
}

// Without a theme, the charts keep the colours they always drew in.
func TestChartsWithoutTheme(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 8)
	buf := buffer.NewBuffer(area)
	widgets.BarChart{Data: []widgets.BarData{{Label: "a", Value: 8}}}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got, want := buf.CellAt(1, 6).Style.Fg, cell.NewColorRGB(0, 200, 255); got != want {
		t.Errorf("bar colour = %v; want %v", got, want)
	}
	if got, want := buf.CellAt(2, 7).Style.Fg, cell.NewColorRGB(180, 190, 205); got != want {
		t.Errorf("bar label colour = %v; want %v", got, want)
	}

	buf = buffer.NewBuffer(area)
	chart := widgets.LineChart{Datasets: []widgets.LineDataset{{Name: "in", Data: []float64{0, 10}}}, ShowAxes: true, ShowLegend: true}
	chart.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got, want := buf.CellAt(6, 0).Style.Fg, cell.NewColorRGB(0, 200, 255); got != want {
		t.Errorf("legend dot colour = %v; want %v", got, want)
	}
	if got, want := buf.CellAt(6, 6).Style.Fg, cell.NewColorRGB(0, 220, 255); got != want {
		t.Errorf("line colour = %v; want %v", got, want)
	}
	if got, want := buf.CellAt(5, 1).Style.Fg, cell.NewColorRGB(120, 135, 155); got != want {
		t.Errorf("axis colour = %v; want %v", got, want)
	}
}

func BenchmarkBarChartDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 25}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	bc := widgets.BarChart{Data: []widgets.BarData{{Label: "a", Value: 3}, {Label: "b", Value: 7}, {Label: "c", Value: 5}}, ShowValues: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bc.Draw(ctx, buf)
	}
}

func BenchmarkLineChartDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 25}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	lc := widgets.LineChart{
		Datasets: []widgets.LineDataset{{Name: "in", Data: []float64{10, 25, 18, 45, 60, 52, 78, 90}}},
		ShowAxes: true, ShowLegend: true, XLabels: []string{"a", "b", "c"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lc.Draw(ctx, buf)
	}
}
