package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestLabelParagraphFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Modifier: cell.ModifierBold}
	for _, preset := range []struct {
		name  string
		theme widgets.Theme
	}{
		{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
	} {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			text := surface.Merge(preset.theme.RoleStyle("text"))
			for _, tc := range []struct {
				name    string
				focused bool
				widget  func() widgets.Widget
				want    cell.Style
			}{
				{"label default", false, func() widgets.Widget { return widgets.NewLabel("Hi") }, text},
				{"label explicit", false, func() widgets.Widget { return widgets.NewLabel("Hi").WithStyle(custom) }, surface.Merge(custom)},
				{"paragraph default", false, func() widgets.Widget { return widgets.NewParagraph("Hi").WithID("p") }, text},
				{"paragraph explicit", false, func() widgets.Widget { return widgets.NewParagraph("Hi").WithStyle(custom).WithID("p") }, surface.Merge(custom)},
				{"paragraph focused", true, func() widgets.Widget { return widgets.NewParagraph("Hi").WithID("p") }, text.Merge(preset.theme.Focus)},
				{"paragraph explicit focus", true, func() widgets.Widget {
					return widgets.NewParagraph("Hi").WithFocusedStyle(custom).WithID("p")
				}, text.Merge(custom)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					term := testkit.NewTerminal(10, 1)
					w := tc.widget()
					term.Draw(func(f *terminal.Frame) {
						f.SetTheme(preset.theme)
						if tc.focused {
							f.FocusManager.SetFocused("p")
						}
						f.RenderWidget(w, term.Area())
					})
					if got := term.Frame().Buffer.CellAt(0, 0); got.Content != 'H' || got.Style != tc.want {
						t.Fatalf("cell = %+v; want 'H' with %+v", got, tc.want)
					}
				})
			}
		})
	}

	// The two themes give the text different colours, so it follows the theme.
	if widgets.DarkTheme().RoleStyle("text").Fg == widgets.LightTheme().RoleStyle("text").Fg {
		t.Fatal("dark and light themes share the text colour")
	}
}

func TestLabelParagraphWithoutTheme(t *testing.T) {
	for _, w := range []widgets.Widget{widgets.NewLabel("Hi"), widgets.NewParagraph("Hi")} {
		term := testkit.NewTerminal(10, 1)
		term.Draw(func(f *terminal.Frame) { f.RenderWidget(w, term.Area()) })
		if got := term.Frame().Buffer.CellAt(0, 0).Style; got.Fg.Type() != cell.ColorDefault {
			t.Errorf("%T without a theme drew with %+v; want the default foreground", w, got)
		}
	}
}
