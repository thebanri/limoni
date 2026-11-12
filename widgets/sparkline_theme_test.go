package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestSparklineFrameTheme(t *testing.T) {
	red, green := cell.NewColorRGB(240, 80, 120), cell.NewColorRGB(80, 220, 140)
	for _, preset := range []struct {
		name  string
		theme widgets.Theme
	}{
		{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
	} {
		t.Run(preset.name, func(t *testing.T) {
			primary := preset.theme.RoleStyle("primary").Fg
			for _, tc := range []struct {
				name  string
				spark widgets.Sparkline
				fg    cell.Color
			}{
				{"default", widgets.Sparkline{}, primary},
				{"explicit color", widgets.Sparkline{Color: red}, red},
				{"explicit style", widgets.Sparkline{Style: cell.Style{Fg: green}}, green},
				{"color over style", widgets.Sparkline{Color: red, Style: cell.Style{Fg: green}}, red},
			} {
				t.Run(tc.name, func(t *testing.T) {
					term := testkit.NewTerminal(3, 1)
					tc.spark.Data = []float64{1, 4, 8}
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(tc.spark, term.Area()) })
					want := tc.spark.Style
					want.Fg = tc.fg
					got := term.Frame().Buffer.CellAt(2, 0)
					if got.Content != '█' || got.Style != want {
						t.Errorf("tallest bar = %+v; want '█' with %+v", got, want)
					}
				})
			}
		})
	}
}

// Without a theme, an unset colour is still the frame's foreground, and a
// set Style's foreground is used for the bars as the Color comment says.
func TestSparklineWithoutTheme(t *testing.T) {
	green := cell.NewColorRGB(80, 220, 140)
	fg := cell.Style{Fg: cell.NewColorRGB(10, 20, 30)}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 3, 1))
	ctx := cell.NewContext(buf.Area, fg)
	widgets.Sparkline{Data: []float64{1, 4, 8}}.Draw(ctx, buf)
	if got := buf.CellAt(2, 0).Style.Fg; got != fg.Fg {
		t.Errorf("unset colour drew in %v, want the frame's %v", got, fg.Fg)
	}
	widgets.Sparkline{Data: []float64{1, 4, 8}, Style: cell.Style{Fg: green}}.Draw(ctx, buf)
	if got := buf.CellAt(2, 0).Style.Fg; got != green {
		t.Errorf("Style's foreground drew in %v, want %v", got, green)
	}
}

func BenchmarkSparklineDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 4}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	spark := widgets.Sparkline{Data: []float64{10, 20, 15, 30, 45, 12, 18, 25, 35}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spark.Draw(ctx, buf)
	}
}
