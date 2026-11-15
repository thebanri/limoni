package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

var chromeThemes = []struct {
	name  string
	theme widgets.Theme
}{
	{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
}

func TestPopupFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			border := surface.Merge(preset.theme.RoleStyle("border"))
			focus := surface.Merge(preset.theme.Focus)
			muted := surface.Merge(preset.theme.RoleStyle("muted"))
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name                             string
				popup                            widgets.Popup
				border, selected, disabled, item cell.Style
			}{
				{"default", widgets.Popup{}, border, focus, muted, surface},
				{"explicit border", widgets.Popup{BorderStyle: custom}, mine, focus, muted, surface},
				{"explicit selected", widgets.Popup{SelectedStyle: custom}, border, mine, muted, surface},
				{"explicit disabled", widgets.Popup{DisabledStyle: custom}, border, focus, mine, surface},
				// Style is the menu's background: set, the items keep to it.
				{"explicit style", widgets.Popup{Style: custom}, border, mine, mine, mine},
			} {
				t.Run(tc.name, func(t *testing.T) {
					state := widgets.NewPopupState()
					state.Open()
					state.Selected = 0
					popup := tc.popup
					popup.ID, popup.Label, popup.State = "menu", "Menu", state
					popup.Items = []widgets.PopupItem{{Text: "One"}, {Text: "Two", Disabled: true}, {Text: "Three"}}
					term := testkit.NewTerminal(20, 6)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(popup, term.Area()) })
					buf := term.Frame().Buffer
					// The menu opens under the button: its border on row 1,
					// then one row per item.
					for _, want := range []struct {
						what  string
						x, y  uint16
						ch    rune
						style cell.Style
					}{
						{"border", 0, 1, '╭', tc.border},
						{"selected marker", 1, 2, '▸', tc.selected},
						{"selected item", 2, 2, 'O', tc.selected},
						{"disabled item", 2, 3, 'T', tc.disabled},
						{"item", 2, 4, 'T', tc.item},
					} {
						if got := buf.CellAt(want.x, want.y); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(%d,%d) = %+v; want %q with %+v", want.what, want.x, want.y, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}
}

func TestStatusBarFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary := surface.Merge(preset.theme.RoleStyle("primary"))
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name      string
				bar       widgets.StatusBar
				key, text cell.Style
			}{
				{"default", widgets.StatusBar{}, primary, surface},
				{"explicit key", widgets.StatusBar{KeyStyle: custom}, mine, surface},
				// Style is the whole row's: set, keys are drawn in it.
				{"explicit style", widgets.StatusBar{Style: custom}, mine, mine},
			} {
				t.Run(tc.name, func(t *testing.T) {
					bar := tc.bar
					bar.Left = []widgets.StatusItem{{Key: "^Q", Text: "quit"}}
					term := testkit.NewTerminal(20, 1)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(bar, term.Area()) })
					buf := term.Frame().Buffer
					for _, want := range []struct {
						what  string
						x     uint16
						ch    rune
						style cell.Style
					}{{"key", 0, '^', tc.key}, {"text", 3, 'q', tc.text}} {
						if got := buf.CellAt(want.x, 0); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(%d,0) = %+v; want %q with %+v", want.what, want.x, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}

	// The two themes give the keys different colours, so they follow the theme.
	dark, light := widgets.DarkTheme().RoleStyle("primary").Fg, widgets.LightTheme().RoleStyle("primary").Fg
	if dark == light {
		t.Fatalf("dark and light themes share the primary colour %v", dark)
	}
}

func BenchmarkPopupDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 40, Height: 8}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	state := widgets.NewPopupState()
	state.Open()
	state.Selected = 0
	popup := widgets.Popup{ID: "menu", Label: "Menu", State: state,
		Items: []widgets.PopupItem{{Text: "One"}, {Text: "Two", Disabled: true}, {Text: "Three"}}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		popup.Draw(ctx, buf)
	}
}

func BenchmarkStatusBarDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 1}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	bar := widgets.StatusBar{
		Left:  []widgets.StatusItem{{Key: "^Q", Text: "quit"}, {Key: "^S", Text: "save"}},
		Right: []widgets.StatusItem{{Text: "ln 12, col 4"}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Draw(ctx, buf)
	}
}
