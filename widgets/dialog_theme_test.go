package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// themedDialog is a 40×10 dialog: its title on row 1, a separator on row 3,
// the message on row 4, the sub-message on row 6 and the buttons on row 8.
func themedDialog() widgets.Dialog {
	return widgets.Dialog{
		ID: "quit", Title: "Quit?", Message: "Save changes?", SubMessage: "Unsaved work is lost.",
		Buttons: []widgets.DialogButton{{Text: "Save"}, {Text: "Discard"}},
	}
}

// dialogCell is one cell of themedDialog that a test checks.
type dialogCell struct {
	what string
	x, y uint16
	ch   rune
}

var (
	dialogBody      = dialogCell{"body", 1, 2, ' '}
	dialogCorner    = dialogCell{"top left corner", 0, 0, '╭'}
	dialogCornerEnd = dialogCell{"top right corner", 39, 0, '╮'}
	dialogTitle     = dialogCell{"title", 17, 1, 'Q'}
	dialogMessage   = dialogCell{"message", 13, 4, 'S'}
	dialogSub       = dialogCell{"sub-message", 9, 6, 'U'}
	// Nothing holds the frame's focus and FocusedButton is 0, so the first
	// button is drawn focused.
	dialogFocused     = dialogCell{"focused button", 7, 8, '['}
	dialogOtherButton = dialogCell{"other button", 21, 8, '['}
)

func drawThemedDialog(theme widgets.Theme, d widgets.Dialog) *buffer.Buffer {
	term := testkit.NewTerminal(40, 10)
	term.Draw(func(f *terminal.Frame) { f.SetTheme(theme); f.RenderWidget(d, term.Area()) })
	return term.Frame().Buffer
}

func checkDialogCells(t *testing.T, buf *buffer.Buffer, want map[dialogCell]cell.Style) {
	t.Helper()
	for c, style := range want {
		if got := buf.CellAt(c.x, c.y); got.Content != c.ch || got.Style != style {
			t.Errorf("%s cell(%d,%d) = %+v; want %q with %+v", c.what, c.x, c.y, got, c.ch, style)
		}
	}
}

// quarterTowards is the RGB colour a quarter of the way from a to b.
func quarterTowards(a, b cell.Color) cell.Color {
	r1, g1, b1 := a.RGB()
	r2, g2, b2 := b.RGB()
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*0.75 + float64(y)*0.25) }
	return cell.NewColorRGB(mix(r1, r2), mix(g1, g2), mix(b1, b2))
}

func TestDialogFrameTheme(t *testing.T) {
	red := cell.NewColorRGB(240, 80, 120)
	custom := cell.Style{Fg: cell.NewColorRGB(80, 220, 140), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range []struct {
		name  string
		theme widgets.Theme
	}{
		{"dark", widgets.DarkTheme()}, {"light", widgets.LightTheme()},
	} {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			bg, text := surface.Bg, surface.Fg
			muted := preset.theme.RoleStyle("muted").Fg
			border := preset.theme.RoleStyle("border").Fg
			focus := preset.theme.Focus.Fg
			themed := map[dialogCell]cell.Style{
				dialogBody:        surface,
				dialogCorner:      {Fg: border, Bg: bg},
				dialogCornerEnd:   {Fg: focus, Bg: bg},
				dialogTitle:       {Fg: text, Bg: bg, Modifier: cell.ModifierBold},
				dialogMessage:     {Fg: text, Bg: bg, Modifier: cell.ModifierBold},
				dialogSub:         {Fg: muted, Bg: bg, Modifier: cell.ModifierItalic},
				dialogFocused:     {Fg: bg, Bg: focus, Modifier: cell.ModifierBold},
				dialogOtherButton: {Fg: text, Bg: quarterTowards(bg, border)},
			}
			for _, tc := range []struct {
				name string
				set  func(*widgets.Dialog)
				want map[dialogCell]cell.Style
			}{
				{"default", func(*widgets.Dialog) {}, nil},
				// Style is the body's: its colours replace the surface, and the
				// message is drawn in its foreground.
				{"explicit style", func(d *widgets.Dialog) { d.Style = custom }, map[dialogCell]cell.Style{
					dialogBody:      custom,
					dialogCorner:    {Fg: border, Bg: custom.Bg},
					dialogCornerEnd: {Fg: focus, Bg: custom.Bg},
					dialogTitle:     {Fg: text, Bg: custom.Bg, Modifier: cell.ModifierBold},
					dialogMessage:   {Fg: custom.Fg, Bg: custom.Bg, Modifier: cell.ModifierBold},
					dialogSub:       {Fg: muted, Bg: custom.Bg, Modifier: cell.ModifierItalic},
				}},
				{"explicit border", func(d *widgets.Dialog) { d.BorderStyle = cell.Style{Fg: red} }, map[dialogCell]cell.Style{
					dialogCorner: {Fg: red, Bg: bg},
				}},
				{"explicit header", func(d *widgets.Dialog) { d.HeaderStyle = custom }, map[dialogCell]cell.Style{
					dialogTitle: {Fg: custom.Fg, Bg: custom.Bg, Modifier: cell.ModifierBold},
				}},
				{"explicit button", func(d *widgets.Dialog) { d.ButtonStyle = custom }, map[dialogCell]cell.Style{
					dialogOtherButton: custom,
				}},
				// ButtonFocusedStyle's background also ends the border.
				{"explicit focused button", func(d *widgets.Dialog) { d.ButtonFocusedStyle = custom }, map[dialogCell]cell.Style{
					dialogCornerEnd: {Fg: custom.Bg, Bg: bg},
					dialogFocused:   {Fg: custom.Fg, Bg: custom.Bg, Modifier: cell.ModifierBold},
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					d := themedDialog()
					tc.set(&d)
					want := make(map[dialogCell]cell.Style, len(themed))
					for c, style := range themed {
						want[c] = style
					}
					for c, style := range tc.want {
						want[c] = style
					}
					checkDialogCells(t, drawThemedDialog(preset.theme, d), want)
				})
			}
		})
	}

	// The two themes give the dialog different colours, so it follows the theme.
	dark, light := widgets.DarkTheme().RoleStyle("surface").Bg, widgets.LightTheme().RoleStyle("surface").Bg
	if dark == light {
		t.Fatalf("dark and light themes share the surface colour %v", dark)
	}
}

// Under the high-contrast theme, the buttons' labels keep its contrast.
func TestDialogHighContrastButtons(t *testing.T) {
	buf := drawThemedDialog(widgets.HighContrastTheme(), themedDialog())
	for _, c := range []dialogCell{dialogFocused, dialogOtherButton} {
		style := buf.CellAt(c.x, c.y).Style
		if ratio := widgets.ContrastRatio(style.Fg, style.Bg); ratio < 7 {
			t.Errorf("%s contrast = %.1f; want at least 7 (AAA)", c.what, ratio)
		}
	}
}

// A theme in ANSI colours has no RGB channels to blend: the border is drawn
// in one colour and the buttons sit on the surface.
func TestDialogANSITheme(t *testing.T) {
	surface, text := cell.NewColorANSI(0), cell.NewColorANSI(7)
	border, focus := cell.NewColorANSI(8), cell.NewColorANSI(4)
	theme := widgets.Theme{
		Colors: widgets.ThemeColors{Surface: surface, Text: text, Border: border, Muted: cell.NewColorANSI(8)},
		Focus:  cell.Style{Fg: focus, Modifier: cell.ModifierBold},
	}
	checkDialogCells(t, drawThemedDialog(theme, themedDialog()), map[dialogCell]cell.Style{
		dialogBody:        {Fg: text, Bg: surface},
		dialogCorner:      {Fg: border, Bg: surface},
		dialogCornerEnd:   {Fg: border, Bg: surface},
		dialogFocused:     {Fg: surface, Bg: focus, Modifier: cell.ModifierBold},
		dialogOtherButton: {Fg: text, Bg: surface},
	})
}

// Without a theme, the dialog keeps the colours it always drew in.
func TestDialogWithoutTheme(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 10)
	buf := buffer.NewBuffer(area)
	themedDialog().Draw(cell.NewContext(area, cell.Style{}), buf)
	bg := cell.NewColorRGB(18, 20, 24)
	checkDialogCells(t, buf, map[dialogCell]cell.Style{
		dialogBody:        {Fg: cell.NewColorRGB(220, 225, 235), Bg: bg},
		dialogCorner:      {Fg: cell.NewColorRGB(255, 80, 80), Bg: bg},
		dialogCornerEnd:   {Fg: cell.NewColorRGB(255, 0, 255), Bg: bg},
		dialogTitle:       {Fg: cell.NewColorRGB(255, 255, 255), Bg: bg, Modifier: cell.ModifierBold},
		dialogMessage:     {Fg: cell.NewColorRGB(240, 245, 255), Bg: bg, Modifier: cell.ModifierBold},
		dialogSub:         {Fg: cell.NewColorRGB(140, 145, 160), Bg: bg, Modifier: cell.ModifierItalic},
		dialogFocused:     {Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(255, 40, 167), Modifier: cell.ModifierBold},
		dialogOtherButton: {Fg: cell.NewColorRGB(180, 185, 200), Bg: cell.NewColorRGB(35, 40, 50)},
	})

	// An ANSI border colour is drawn as it is, not blended through its code.
	buf = buffer.NewBuffer(area)
	green := cell.NewColorANSI(2)
	d := themedDialog()
	d.BorderStyle = cell.Style{Fg: green}
	d.Draw(cell.NewContext(area, cell.Style{}), buf)
	for _, c := range []dialogCell{dialogCorner, dialogCornerEnd} {
		if got := buf.CellAt(c.x, c.y); got.Style.Fg != green {
			t.Errorf("%s colour = %v; want %v", c.what, got.Style.Fg, green)
		}
	}
}

func BenchmarkDialogDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 80, Height: 25}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(cell.NewRect(10, 5, 50, 12), theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	d := themedDialog()
	d.Shadow = true
	d.State = &widgets.DialogState{}
	d.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Draw(ctx, buf)
	}
}
