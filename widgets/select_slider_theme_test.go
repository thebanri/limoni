package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

var formThemes = []struct {
	name  string
	theme widgets.Theme
}{
	{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
}

// drawFocused renders w under theme, with the focus on id when focused.
func drawFocused(theme widgets.Theme, w widgets.Widget, id string, focused bool, width, height uint16) *buffer.Buffer {
	term := testkit.NewTerminal(width, height)
	term.Draw(func(f *terminal.Frame) {
		f.SetTheme(theme)
		if focused {
			f.FocusManager.SetFocused(id)
		}
		f.RenderWidget(w, term.Area())
	})
	return term.Frame().Buffer
}

func TestSelectFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	fallback := cell.Style{Modifier: cell.ModifierReverse | cell.ModifierBold}
	for _, preset := range formThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			focus := surface.Merge(preset.theme.Focus)
			for _, tc := range []struct {
				name                   string
				focused                bool
				style, focusedStyle    cell.Style
				selectedStyle          cell.Style
				field, option, current cell.Style
			}{
				{name: "normal", field: surface, option: surface, current: focus},
				{name: "focused", focused: true, field: focus, option: surface, current: focus},
				{name: "explicit style", focused: true, style: custom, field: surface.Merge(custom).Merge(fallback), option: surface, current: focus},
				{name: "explicit focus", focused: true, focusedStyle: custom, field: surface.Merge(custom), option: surface, current: focus},
				{name: "explicit selected", selectedStyle: custom, field: surface, option: surface, current: surface.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					state := widgets.NewSelectState()
					state.Selected, state.Open = 1, true
					sel := widgets.Select{ID: "pick", Options: []string{"one", "two"}, State: state,
						Style: tc.style, FocusedStyle: tc.focusedStyle, SelectedStyle: tc.selectedStyle}
					buf := drawFocused(preset.theme, sel, "pick", tc.focused, 12, 3)
					for _, want := range []struct {
						what  string
						y     uint16
						ch    rune
						style cell.Style
					}{{"field", 0, 't', tc.field}, {"option", 1, 'o', tc.option}, {"selected option", 2, 't', tc.current}} {
						if got := buf.CellAt(1, want.y); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(1,%d) = %+v; want %q with %+v", want.what, want.y, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}
}

func TestSliderFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	fallback := cell.Style{Modifier: cell.ModifierReverse | cell.ModifierBold}
	for _, preset := range formThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary, muted := surface.Merge(preset.theme.RoleStyle("primary")), surface.Merge(preset.theme.RoleStyle("muted"))
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name                 string
				focused              bool
				slider               widgets.Slider
				filled, thumb, track cell.Style
			}{
				{"normal", false, widgets.Slider{}, primary, surface, muted},
				{"focused", true, widgets.Slider{}, primary, surface.Merge(preset.theme.Focus), muted},
				// A set Style is the whole slider's, as it always was
				// without a frame style.
				{"explicit style", false, widgets.Slider{Style: custom}, mine, mine, mine},
				{"explicit style focused", true, widgets.Slider{Style: custom}, mine, mine.Merge(fallback), mine},
				{"explicit focus", true, widgets.Slider{FocusedStyle: custom}, mine, mine, mine},
				{"explicit filled", false, widgets.Slider{FilledStyle: custom}, mine, surface, muted},
				{"explicit track", false, widgets.Slider{TrackStyle: custom}, primary, surface, mine},
				{"explicit thumb focused", true, widgets.Slider{ThumbStyle: custom}, primary, mine.Merge(fallback), muted},
			} {
				t.Run(tc.name, func(t *testing.T) {
					slider := tc.slider
					slider.ID, slider.State, slider.Min, slider.Max = "volume", widgets.NewSliderState(5), 0, 10
					buf := drawFocused(preset.theme, slider, "volume", tc.focused, 11, 1)
					for _, want := range []struct {
						what  string
						x     uint16
						ch    rune
						style cell.Style
					}{{"filled", 2, '─', tc.filled}, {"thumb", 5, '●', tc.thumb}, {"track", 8, '─', tc.track}} {
						if got := buf.CellAt(want.x, 0); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(%d,0) = %+v; want %q with %+v", want.what, want.x, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}

	// The two themes give the slider different colours, so it follows the theme.
	dark, light := widgets.DarkTheme().Focus.Fg, widgets.LightTheme().Focus.Fg
	if dark == light {
		t.Fatalf("dark and light themes share the focus colour %v", dark)
	}
}

// A hovered option looks selected, under a frame style too; it used to lose
// its highlight whenever the frame had a style.
func TestSelectHoverFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range formThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			for _, tc := range []struct {
				name  string
				hover cell.Style
				want  cell.Style
			}{
				{"default", cell.Style{}, surface.Merge(preset.theme.Focus)},
				{"explicit hover", custom, surface.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					state := widgets.NewSelectState()
					state.Hovered, state.Open = 0, true
					sel := widgets.Select{ID: "pick", Options: []string{"one", "two"}, State: state, HoverStyle: tc.hover}
					buf := drawFocused(preset.theme, sel, "pick", false, 12, 3)
					if got := buf.CellAt(1, 1); got.Content != 'o' || got.Style != tc.want {
						t.Errorf("hovered option = %+v; want 'o' with %+v", got, tc.want)
					}
				})
			}
		})
	}
}

// Under a frame style but no theme, a Slider's Style and FocusedStyle reach
// its parts. They used to be dropped whenever the frame had a style.
func TestSliderStyleUnderFrameStyle(t *testing.T) {
	frame := cell.Style{Fg: cell.NewColorRGB(10, 20, 30), Bg: cell.NewColorRGB(200, 210, 220)}
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120)}
	for _, tc := range []struct {
		name    string
		focused bool
		slider  widgets.Slider
		want    cell.Style
	}{
		{"style", false, widgets.Slider{Style: custom}, frame.Merge(custom)},
		{"focused style", true, widgets.Slider{FocusedStyle: custom}, frame.Merge(custom)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			area := cell.NewRect(0, 0, 11, 1)
			buf := buffer.NewBuffer(area)
			ctx := cell.NewContext(area, frame)
			if tc.focused {
				ctx.FocusedID = "volume"
			}
			slider := tc.slider
			slider.ID, slider.State, slider.Min, slider.Max = "volume", widgets.NewSliderState(5), 0, 10
			slider.Draw(ctx, buf)
			for _, x := range []uint16{2, 5, 8} {
				if got := buf.CellAt(x, 0).Style; got != tc.want {
					t.Errorf("cell(%d,0) style = %+v; want %+v", x, got, tc.want)
				}
			}
		})
	}
}

func BenchmarkSelectDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 40, Height: 6}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	ctx.FocusedID = "pick"
	state := widgets.NewSelectState()
	state.Open = true
	sel := widgets.Select{ID: "pick", Options: []string{"one", "two", "three"}, State: state}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sel.Draw(ctx, buf)
	}
}

func BenchmarkSliderDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 40, Height: 1}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	ctx.FocusedID = "volume"
	slider := widgets.Slider{ID: "volume", State: widgets.NewSliderState(5), Min: 0, Max: 10}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slider.Draw(ctx, buf)
	}
}
