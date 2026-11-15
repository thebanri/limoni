package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestToastFrameTheme(t *testing.T) {
	for _, preset := range []struct {
		name  string
		theme widgets.Theme
	}{
		{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
	} {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			muted := preset.theme.RoleStyle("muted").Fg
			for _, tc := range []struct {
				name  string
				level widgets.ToastLevel
				role  string
				badge rune
			}{
				{"info", widgets.ToastInfo, "primary", 'ℹ'},
				{"success", widgets.ToastSuccess, "success", '✓'},
				{"warning", widgets.ToastWarning, "warning", '⚠'},
				{"error", widgets.ToastError, "error", '✕'},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tm := widgets.NewToastManager(widgets.ToastTopRight)
					tm.Show("Title", "Message", tc.level, 0)
					term := testkit.NewTerminal(80, 12)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(tm, term.Area()) })
					buf := term.Frame().Buffer

					// A 38-column card, four columns in from the right, two rows down.
					x, y := uint16(80-38-4), uint16(2)
					accent := preset.theme.RoleStyle(tc.role).Fg
					border := cell.Style{Fg: accent, Bg: surface.Bg}
					badge := cell.Style{Fg: accent, Bg: tint(surface.Bg, accent), Modifier: cell.ModifierBold}
					for _, want := range []struct {
						what  string
						x, y  uint16
						ch    rune
						style cell.Style
					}{
						{"corner", x, y, '╭', border},
						{"badge", x + 3, y, tc.badge, badge},
						{"close", x + 38 - 3, y, '✕', cell.Style{Fg: muted, Bg: surface.Bg}},
						{"title", x + 2, y + 1, 'T', surface.Bold()},
						{"message", x + 2, y + 2, 'M', cell.Style{Fg: muted, Bg: surface.Bg}},
						{"time left", x + 2, y + 3, '━', border},
					} {
						if got := buf.CellAt(want.x, want.y); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(%d,%d) = %+v; want %q with %+v", want.what, want.x, want.y, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}

	// The two themes give the card different colours, so it follows the theme.
	dark, light := widgets.DarkTheme().RoleStyle("surface").Bg, widgets.LightTheme().RoleStyle("surface").Bg
	if dark == light {
		t.Fatalf("dark and light themes share the surface colour %v", dark)
	}
}

// tint is bg with a fifth of accent mixed in.
func tint(bg, accent cell.Color) cell.Color {
	r1, g1, b1 := bg.RGB()
	r2, g2, b2 := accent.RGB()
	mix := func(a, b uint8) uint8 { return uint8(float64(a)*0.8 + float64(b)*0.2) }
	return cell.NewColorRGB(mix(r1, r2), mix(g1, g2), mix(b1, b2))
}

// A level coloured from the 256-colour palette cannot be blended, so its
// badge sits on the card's own background.
func TestToastPaletteThemeBadge(t *testing.T) {
	red := cell.NewColorANSI(1)
	theme := widgets.DarkTheme().AddClass("error", cell.Style{Fg: red})
	tm := widgets.NewToastManager(widgets.ToastTopRight)
	tm.Error("Title", "Message")
	term := testkit.NewTerminal(80, 12)
	term.Draw(func(f *terminal.Frame) { f.SetTheme(theme); f.RenderWidget(tm, term.Area()) })
	want := cell.Style{Fg: red, Bg: theme.RoleStyle("surface").Bg, Modifier: cell.ModifierBold}
	if got := term.Frame().Buffer.CellAt(80-38-4+3, 2); got.Content != '✕' || got.Style != want {
		t.Errorf("badge cell = %+v; want '✕' with %+v", got, want)
	}
}

// Without a theme, a toast keeps its own colours.
func TestToastWithoutTheme(t *testing.T) {
	tm := widgets.NewToastManager(widgets.ToastTopRight)
	tm.Error("Title", "Message")
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 12))
	tm.Draw(cell.NewContext(buf.Area, cell.Style{}), buf)
	x, y := uint16(80-38-4), uint16(2)
	bg := cell.NewColorRGB(18, 22, 32)
	for _, want := range []struct {
		what  string
		x, y  uint16
		style cell.Style
	}{
		{"corner", x, y, cell.Style{Fg: cell.NewColorRGB(231, 76, 60), Bg: bg}},
		{"badge", x + 3, y, cell.Style{Fg: cell.NewColorRGB(231, 76, 60), Bg: cell.NewColorRGB(65, 20, 20), Modifier: cell.ModifierBold}},
		{"close", x + 38 - 3, y, cell.Style{Fg: cell.NewColorRGB(160, 170, 185), Bg: bg}},
		{"title", x + 2, y + 1, cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: bg, Modifier: cell.ModifierBold}},
		{"message", x + 2, y + 2, cell.Style{Fg: cell.NewColorRGB(160, 175, 195), Bg: bg}},
	} {
		if got := buf.CellAt(want.x, want.y).Style; got != want.style {
			t.Errorf("%s style = %+v; want %+v", want.what, got, want.style)
		}
	}
}

func BenchmarkToastDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 25}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	tm := widgets.NewToastManager(widgets.ToastTopRight)
	tm.Success("Saved", "3 files written")
	tm.Error("Failed", "network unreachable")
	tm.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.Draw(ctx, buf)
	}
}
