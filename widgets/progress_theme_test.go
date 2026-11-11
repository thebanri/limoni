package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestProgressBarFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range []struct {
		name  string
		theme widgets.Theme
	}{
		{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
	} {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary, muted := surface.Merge(preset.theme.RoleStyle("primary")), surface.Merge(preset.theme.RoleStyle("muted"))
			for _, tc := range []struct {
				name                 string
				bar                  widgets.ProgressBar
				filled, empty, label cell.Style
			}{
				{"default", widgets.ProgressBar{}, primary, muted, surface},
				{"explicit filled", widgets.ProgressBar{FilledStyle: custom}, surface.Merge(custom), muted, surface},
				{"explicit empty", widgets.ProgressBar{EmptyStyle: custom}, primary, surface.Merge(custom), surface},
				{"modifier filled", widgets.ProgressBar{FilledStyle: cell.Style{Modifier: cell.ModifierBold}}, surface.Bold(), muted, surface},
				// Style is the label's: set, the bar is left as it was.
				{"explicit style", widgets.ProgressBar{Style: custom}, surface, surface, surface.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					term := testkit.NewTerminal(10, 1)
					tc.bar.Value, tc.bar.Max, tc.bar.ShowPercent = 50, 100, true
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(tc.bar, term.Area()) })
					for _, want := range []struct {
						x     uint16
						ch    rune
						style cell.Style
					}{{0, '█', tc.filled}, {9, '░', tc.empty}, {5, '%', tc.label}} {
						got := term.Frame().Buffer.CellAt(want.x, 0)
						if got.Content != want.ch || got.Style != want.style {
							t.Errorf("cell(%d,0) = %+v; want %q with %+v", want.x, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}

	// The two themes give the bar different colours, so it follows the theme.
	dark, light := widgets.DarkTheme().RoleStyle("primary").Fg, widgets.LightTheme().RoleStyle("primary").Fg
	if dark == light {
		t.Fatalf("dark and light themes share the primary colour %v", dark)
	}
}

func BenchmarkProgressBarDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 1}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	bar := widgets.ProgressBar{Value: 42, Max: 100, ShowPercent: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Draw(ctx, buf)
	}
}
