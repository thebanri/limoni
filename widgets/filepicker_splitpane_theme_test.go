package widgets_test

import (
	"errors"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// themedPickerState lists a directory without reading the disk: a parent,
// a directory and two files, with the cursor on the first file.
func themedPickerState() *widgets.FilePickerState {
	return &widgets.FilePickerState{
		Dir: "/home/me",
		Entries: []widgets.FileEntry{
			{Name: "..", IsDir: true}, {Name: "src", IsDir: true},
			{Name: "a.go", Size: 12}, {Name: "b.go", Size: 34},
		},
		Selected: 2,
	}
}

func TestFilePickerFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary := surface.Merge(preset.theme.RoleStyle("primary"))
			muted := surface.Merge(preset.theme.RoleStyle("muted"))
			focusBar := surface.Merge(preset.theme.Focus).Reverse()
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name                                       string
				picker                                     widgets.FilePicker
				header, dir, file, size, selected, selSize cell.Style
			}{
				{"default", widgets.FilePicker{}, muted, primary, surface, muted, focusBar, focusBar},
				{"explicit header", widgets.FilePicker{HeaderStyle: custom}, mine, primary, surface, muted, focusBar, focusBar},
				{"explicit dir", widgets.FilePicker{DirStyle: custom}, muted, mine, surface, muted, focusBar, focusBar},
				// An explicit SizeStyle reaches the selected entry too, as
				// without a theme.
				{"explicit size", widgets.FilePicker{SizeStyle: custom}, muted, primary, surface, mine, focusBar, focusBar.Merge(custom)},
				// An explicit SelectedStyle is drawn as it is, not reversed.
				{"explicit selected", widgets.FilePicker{SelectedStyle: custom}, muted, primary, surface, muted, mine, mine},
				// Style is the whole picker's: set, the parts keep to it.
				{"explicit style", widgets.FilePicker{Style: custom}, mine, mine, mine, mine, mine.Reverse(), mine.Reverse()},
			} {
				t.Run(tc.name, func(t *testing.T) {
					picker := tc.picker
					picker.State = themedPickerState()
					term := testkit.NewTerminal(20, 5)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(picker, term.Area()) })
					buf := term.Frame().Buffer
					for _, want := range []struct {
						what  string
						x, y  uint16
						ch    rune
						style cell.Style
					}{
						{"header", 0, 0, '/', tc.header},
						{"directory", 0, 2, 's', tc.dir},
						{"selected file", 0, 3, 'a', tc.selected},
						{"selected size", 19, 3, 'B', tc.selSize},
						{"file", 0, 4, 'b', tc.file},
						{"size", 19, 4, 'B', tc.size},
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

func TestFilePickerFrameThemeError(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			for _, tc := range []struct {
				name   string
				picker widgets.FilePicker
				want   cell.Style
			}{
				{"default", widgets.FilePicker{}, surface.Merge(preset.theme.RoleStyle("error"))},
				{"explicit error", widgets.FilePicker{ErrorStyle: custom}, surface.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					picker := tc.picker
					picker.State = themedPickerState()
					picker.State.Err = errors.New("permission denied")
					term := testkit.NewTerminal(20, 5)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(picker, term.Area()) })
					if got := term.Frame().Buffer.CellAt(0, 0); got.Content != '/' || got.Style != tc.want {
						t.Errorf("header cell(0,0) = %+v; want '/' with %+v", got, tc.want)
					}
				})
			}
		})
	}
}

func TestSplitPaneFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	other := cell.Style{Fg: cell.NewColorRGB(10, 200, 90)}
	reverseBold := cell.Style{Modifier: cell.ModifierReverse | cell.ModifierBold}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			border := surface.Merge(preset.theme.RoleStyle("border"))
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name    string
				focused bool
				pane    widgets.SplitPane
				want    cell.Style
			}{
				{"default", false, widgets.SplitPane{}, border},
				{"focused", true, widgets.SplitPane{}, border.Merge(preset.theme.Focus)},
				{"explicit divider", false, widgets.SplitPane{DividerStyle: custom}, mine},
				// A set DividerStyle keeps the reverse-bold focus mark, as
				// without a theme.
				{"explicit divider focused", true, widgets.SplitPane{DividerStyle: custom}, mine.Merge(reverseBold)},
				{"explicit focused", true, widgets.SplitPane{FocusedDividerStyle: other}, border.Merge(other)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					pane := tc.pane
					pane.ID = "split"
					term := testkit.NewTerminal(9, 3)
					term.Draw(func(f *terminal.Frame) {
						f.SetTheme(preset.theme)
						if tc.focused {
							f.FocusManager.SetFocused("split")
						}
						f.RenderWidget(pane, term.Area())
					})
					// An even split of nine columns puts the divider in the middle.
					if got := term.Frame().Buffer.CellAt(4, 1); got.Content != '│' || got.Style != tc.want {
						t.Errorf("divider cell(4,1) = %+v; want '│' with %+v", got, tc.want)
					}
				})
			}
		})
	}
}

func BenchmarkFilePickerDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 40, Height: 10}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	picker := widgets.FilePicker{ID: "files", State: themedPickerState()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		picker.Draw(ctx, buf)
	}
}

func BenchmarkSplitPaneDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 24}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	ctx.FocusedID = "split"
	pane := widgets.SplitPane{ID: "split", State: &widgets.SplitState{}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pane.Draw(ctx, buf)
	}
}
