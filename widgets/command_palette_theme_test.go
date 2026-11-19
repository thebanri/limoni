package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// themedPalette is a palette open on a 60×20 area with query typed in. Its
// panel is 36 columns wide at (12, 2): the title is in the top border, the
// search box on row 3, a separator on row 4 and the results from row 5.
func themedPalette(t testing.TB, query string) widgets.CommandPalette {
	t.Helper()
	state := widgets.NewCommandPaletteState()
	state.AllItems = []widgets.CommandItem{
		{Label: "Open file", Detail: "Ctrl+O"},
		{Label: "Close", Detail: "Ctrl+W"},
		{Label: "Quit", Detail: "Ctrl+Q"},
	}
	state.Open()
	for _, r := range query {
		state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: r})
	}
	if query == "o" && (len(state.Filtered) != 2 || state.Filtered[0].Label != "Open file") {
		t.Fatalf("query %q filtered to %+v; want Open file, then Close", query, state.Filtered)
	}
	return widgets.CommandPalette{ID: "palette", State: state}
}

// paletteCell is one cell of themedPalette that a test checks.
type paletteCell struct {
	what string
	x, y uint16
	ch   rune
}

// The cells of themedPalette with the query "o": "Open file" is selected on
// row 5 and "Close" is on row 6, with their "o" matched.
var (
	paletteCorner     = paletteCell{"top left corner", 12, 2, '╭'}
	paletteTitle      = paletteCell{"title", 17, 2, 'C'}
	paletteQuery      = paletteCell{"query", 15, 3, 'o'}
	paletteSeparator  = paletteCell{"separator", 12, 4, '├'}
	paletteMarker     = paletteCell{"selection marker", 13, 5, '▸'}
	paletteSelMatch   = paletteCell{"selected item's match", 15, 5, 'O'}
	paletteSelected   = paletteCell{"selected item", 16, 5, 'p'}
	paletteSelDetail  = paletteCell{"selected item's detail", 40, 5, 'C'}
	paletteItem       = paletteCell{"item", 15, 6, 'C'}
	paletteItemMatch  = paletteCell{"item's match", 17, 6, 'o'}
	paletteItemDetail = paletteCell{"item's detail", 40, 6, 'C'}
	paletteCount      = paletteCell{"result count", 41, 7, '2'}
	// With the query empty, the placeholder is in the search box, its first
	// letter under the cursor.
	palettePlaceholder = paletteCell{"placeholder", 16, 3, 'e'}
)

func drawThemedPalette(theme *widgets.Theme, cp widgets.CommandPalette) *buffer.Buffer {
	term := testkit.NewTerminal(60, 20)
	term.Draw(func(f *terminal.Frame) {
		if theme != nil {
			f.SetTheme(*theme)
		}
		f.RenderWidget(cp, term.Area())
	})
	return term.Frame().Buffer
}

func checkPaletteCells(t *testing.T, buf *buffer.Buffer, want map[paletteCell]cell.Style) {
	t.Helper()
	for c, style := range want {
		if got := buf.CellAt(c.x, c.y); got.Content != c.ch || got.Style != style {
			t.Errorf("%s cell(%d,%d) = %+v; want %q with %+v", c.what, c.x, c.y, got, c.ch, style)
		}
	}
}

func TestCommandPaletteFrameTheme(t *testing.T) {
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
			inputBg := preset.theme.RoleStyle("background").Bg
			border := preset.theme.RoleStyle("border").Fg
			primary := preset.theme.RoleStyle("primary").Fg
			muted := preset.theme.RoleStyle("muted").Fg
			selBg := quarterTowards(bg, preset.theme.Focus.Fg)
			bold := cell.ModifierBold
			themed := map[paletteCell]cell.Style{
				paletteCorner:     {Fg: border, Bg: bg},
				paletteTitle:      {Fg: primary, Bg: bg, Modifier: bold},
				paletteQuery:      {Fg: text, Bg: inputBg},
				paletteSeparator:  {Fg: border, Bg: bg},
				paletteMarker:     {Fg: primary, Bg: selBg},
				paletteSelMatch:   {Fg: primary, Bg: selBg, Modifier: bold},
				paletteSelected:   {Fg: text, Bg: selBg},
				paletteSelDetail:  {Fg: muted, Bg: selBg},
				paletteItem:       {Fg: text, Bg: bg},
				paletteItemMatch:  {Fg: primary, Bg: bg, Modifier: bold},
				paletteItemDetail: {Fg: muted, Bg: bg},
				paletteCount:      {Fg: muted, Bg: bg},
			}
			for _, tc := range []struct {
				name string
				set  func(*widgets.CommandPalette)
				want map[paletteCell]cell.Style
			}{
				{"default", func(*widgets.CommandPalette) {}, nil},
				// Style is the panel's background: the parts drawn on the
				// panel sit on it, and the items take its colours.
				{"explicit style", func(cp *widgets.CommandPalette) { cp.Style = custom }, map[paletteCell]cell.Style{
					paletteCorner:     {Fg: border, Bg: custom.Bg},
					paletteTitle:      {Fg: primary, Bg: custom.Bg, Modifier: bold},
					paletteSeparator:  {Fg: border, Bg: custom.Bg},
					paletteItem:       {Fg: text, Bg: custom.Bg},
					paletteItemMatch:  {Fg: primary, Bg: custom.Bg, Modifier: bold},
					paletteItemDetail: {Fg: muted, Bg: custom.Bg},
					paletteCount:      {Fg: muted, Bg: custom.Bg},
				}},
				{"explicit input", func(cp *widgets.CommandPalette) { cp.InputStyle = custom }, map[paletteCell]cell.Style{
					paletteQuery: custom,
				}},
				{"explicit item", func(cp *widgets.CommandPalette) { cp.ItemStyle = custom }, map[paletteCell]cell.Style{
					paletteItem:      custom,
					paletteItemMatch: {Fg: primary, Bg: custom.Bg, Modifier: bold},
				}},
				{"explicit selected", func(cp *widgets.CommandPalette) { cp.SelectedStyle = custom }, map[paletteCell]cell.Style{
					paletteMarker:    {Fg: primary, Bg: custom.Bg},
					paletteSelMatch:  {Fg: primary, Bg: custom.Bg, Modifier: bold},
					paletteSelected:  custom,
					paletteSelDetail: {Fg: muted, Bg: custom.Bg},
				}},
				// The selected row's detail keeps the selection's background.
				{"explicit detail", func(cp *widgets.CommandPalette) { cp.DetailStyle = custom }, map[paletteCell]cell.Style{
					paletteSelDetail:  {Fg: custom.Fg, Bg: selBg},
					paletteItemDetail: custom,
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					cp := themedPalette(t, "o")
					tc.set(&cp)
					want := make(map[paletteCell]cell.Style, len(themed))
					for c, style := range themed {
						want[c] = style
					}
					for c, style := range tc.want {
						want[c] = style
					}
					checkPaletteCells(t, drawThemedPalette(&preset.theme, cp), want)
				})
			}

			t.Run("placeholder", func(t *testing.T) {
				checkPaletteCells(t, drawThemedPalette(&preset.theme, themedPalette(t, "")), map[paletteCell]cell.Style{
					palettePlaceholder: {Fg: muted, Bg: inputBg},
				})
			})
		})
	}
}

// A theme in ANSI colours has no RGB channels to blend: the selected row is
// filled with the focus colour. Where that is also the accent, the marker and
// the matched letters take the row's text colour so they stay visible.
func TestCommandPaletteANSITheme(t *testing.T) {
	surface, text, blue := cell.NewColorANSI(0), cell.NewColorANSI(7), cell.NewColorANSI(4)
	theme := widgets.Theme{
		Colors: widgets.ThemeColors{
			Surface: surface, Background: surface, Text: text, Border: cell.NewColorANSI(8),
			Muted: cell.NewColorANSI(8), Primary: blue,
		},
		Focus: cell.Style{Fg: blue, Modifier: cell.ModifierBold},
	}
	bold := cell.ModifierBold
	checkPaletteCells(t, drawThemedPalette(&theme, themedPalette(t, "o")), map[paletteCell]cell.Style{
		paletteMarker:    {Fg: surface, Bg: blue},
		paletteSelMatch:  {Fg: surface, Bg: blue, Modifier: bold},
		paletteSelected:  {Fg: surface, Bg: blue},
		paletteItemMatch: {Fg: blue, Bg: surface, Modifier: bold},
	})
}

// Without a theme, the palette keeps the colours it always drew in.
func TestCommandPaletteWithoutTheme(t *testing.T) {
	bg := cell.NewColorRGB(30, 30, 40)
	inputBg := cell.NewColorRGB(50, 50, 65)
	selBg := cell.NewColorRGB(50, 100, 200)
	cyan := cell.NewColorRGB(100, 200, 255)
	bold := cell.ModifierBold
	checkPaletteCells(t, drawThemedPalette(nil, themedPalette(t, "o")), map[paletteCell]cell.Style{
		paletteCorner:     {Fg: cyan, Bg: bg},
		paletteTitle:      {Fg: cyan, Bg: bg, Modifier: bold},
		paletteQuery:      {Fg: cell.NewColorRGB(255, 255, 255), Bg: inputBg},
		paletteSeparator:  {Fg: cell.NewColorRGB(60, 60, 80), Bg: bg},
		paletteMarker:     {Fg: cyan, Bg: selBg},
		paletteSelMatch:   {Fg: cyan, Bg: selBg, Modifier: bold},
		paletteSelected:   {Fg: cell.NewColorRGB(255, 255, 255), Bg: selBg},
		paletteSelDetail:  {Fg: cell.NewColorRGB(120, 120, 150), Bg: selBg},
		paletteItem:       {Fg: cell.NewColorRGB(200, 200, 210), Bg: bg},
		paletteItemMatch:  {Fg: cyan, Bg: bg, Modifier: bold},
		paletteItemDetail: {Fg: cell.NewColorRGB(120, 120, 150), Bg: bg},
		paletteCount:      {Fg: cell.NewColorRGB(80, 80, 100), Bg: bg},
	})
	checkPaletteCells(t, drawThemedPalette(nil, themedPalette(t, "")), map[paletteCell]cell.Style{
		palettePlaceholder: {Fg: cell.NewColorRGB(120, 120, 140), Bg: inputBg},
	})
}

func BenchmarkCommandPaletteDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 60, Height: 20}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	cp := themedPalette(b, "o")
	cp.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.Draw(ctx, buf)
	}
}
