package limoni

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/testkit"
)

// The root package is a facade over component, widgets and layout. A facade
// fails quietly — wired to the wrong constructor, or the right one with the
// arguments in the wrong order — so these render each helper and look at what
// actually lands on the screen.
func render(t *testing.T, w, h uint16, c Component) []string {
	t.Helper()
	term := testkit.NewTerminal(w, h)
	term.Draw(func(f *Frame) { f.RenderWidget(c, f.Area()) })
	return strings.Split(strings.TrimRight(term.Snapshot(), "\n"), "\n")
}

// cellAt returns the rune at a position, or ' ' outside the grid.
func cellAt(rows []string, x, y int) rune {
	if y < 0 || y >= len(rows) {
		return ' '
	}
	r := []rune(rows[y])
	if x < 0 || x >= len(r) {
		return ' '
	}
	return r[x]
}

func find(rows []string, s string) (x, y int, ok bool) {
	for i, row := range rows {
		if j := strings.Index(row, s); j >= 0 {
			return len([]rune(row[:j])), i, true
		}
	}
	return 0, 0, false
}

func TestPaddingMovesContentByTheRightAmountOnEachSide(t *testing.T) {
	rows := render(t, 20, 6, Pad(Label("x"), 2, 0, 0, 3))
	x, y, ok := find(rows, "x")
	if !ok {
		t.Fatalf("nothing drawn:\n%s", strings.Join(rows, "\n"))
	}
	if x != 3 || y != 2 {
		t.Errorf("Pad(top 2, left 3) put the label at (%d,%d), want (3,2)", x, y)
	}

	rows = render(t, 20, 6, PadAll(Label("x"), 1))
	if x, y, _ = find(rows, "x"); x != 1 || y != 1 {
		t.Errorf("PadAll(1) → (%d,%d), want (1,1)", x, y)
	}

	// PadAxis takes horizontal first, then vertical — the pair most easily
	// swapped in a facade.
	rows = render(t, 20, 6, PadAxis(Label("x"), 4, 1))
	if x, y, _ = find(rows, "x"); x != 4 || y != 1 {
		t.Errorf("PadAxis(h 4, v 1) → (%d,%d), want (4,1)", x, y)
	}
}

func TestMarginMatchesPaddingForAPlainChild(t *testing.T) {
	m := render(t, 20, 6, Margin(Label("x"), 1, 0, 0, 2))
	x, y, ok := find(m, "x")
	if !ok || x != 2 || y != 1 {
		t.Errorf("Margin(top 1, left 2) → (%d,%d)", x, y)
	}
	if a := render(t, 20, 6, MarginAll(Label("x"), 2)); strings.Join(a, "\n") != strings.Join(render(t, 20, 6, PadAll(Label("x"), 2)), "\n") {
		t.Error("MarginAll(2) and PadAll(2) should place a plain label identically")
	}
	if x, y, _ = find(render(t, 20, 6, MarginAxis(Label("x"), 3, 2)), "x"); x != 3 || y != 2 {
		t.Errorf("MarginAxis(h 3, v 2) → (%d,%d)", x, y)
	}
}

func TestConstrainAndTheSingleAxisLimits(t *testing.T) {
	long := strings.Repeat("ab", 20)

	rows := render(t, 30, 3, MaxWidth(6, Label(long)))
	if got := strings.TrimRight(rows[0], " "); len([]rune(got)) > 6 {
		t.Errorf("MaxWidth(6) drew %d columns: %q", len([]rune(got)), got)
	}

	rows = render(t, 30, 6, MaxHeight(2, VStack(Label("a"), Label("b"), Label("c"), Label("d"))))
	if strings.TrimSpace(rows[2]) != "" {
		t.Errorf("MaxHeight(2) drew a third row: %q", rows[2])
	}

	rows = render(t, 30, 4, Constrain(0, 5, 0, 2, Label(long)))
	if got := strings.TrimRight(rows[0], " "); len([]rune(got)) > 5 {
		t.Errorf("Constrain maxW 5 drew %q", got)
	}

	// The minimums reserve room even for a child that asks for less.
	rows = render(t, 30, 4, Border(MinWidth(10, Label("x")), SymbolsRounded, NewStyle()))
	if width := len([]rune(strings.TrimRight(rows[0], " "))); width < 10 {
		t.Errorf("MinWidth(10) left a %d-column box", width)
	}
	rows = render(t, 30, 8, Border(MinHeight(4, Label("x")), SymbolsRounded, NewStyle()))
	drawn := 0
	for _, r := range rows {
		if strings.TrimSpace(r) != "" {
			drawn++
		}
	}
	if drawn < 4 {
		t.Errorf("MinHeight(4) left %d rows", drawn)
	}
}

func TestPlaceAligns(t *testing.T) {
	rows := render(t, 21, 5, Place(21, 5, AlignCenter, AlignMiddle, Label("x")))
	x, y, ok := find(rows, "x")
	if !ok {
		t.Fatal("nothing drawn")
	}
	if x != 10 || y != 2 {
		t.Errorf("centred label at (%d,%d), want (10,2)", x, y)
	}

	rows = render(t, 21, 5, Place(21, 5, AlignRight, AlignBottom, Label("x")))
	if x, y, _ = find(rows, "x"); x != 20 || y != 4 {
		t.Errorf("bottom-right label at (%d,%d), want (20,4)", x, y)
	}
}

func TestDividersDrawALineAcrossTheirAxis(t *testing.T) {
	rows := render(t, 10, 3, Divider())
	if got := strings.TrimSpace(rows[0]); got == "" {
		t.Fatal("Divider drew nothing")
	} else if len([]rune(got)) != 10 {
		t.Errorf("Divider drew %d columns of %q, want 10", len([]rune(got)), got)
	}

	rows = render(t, 12, 3, HStack(VDivider(), Label("x")))
	col := []rune{cellAt(rows, 0, 0), cellAt(rows, 0, 1), cellAt(rows, 0, 2)}
	for _, r := range col {
		if r == ' ' {
			t.Errorf("VDivider left a gap in its column: %q", string(col))
			break
		}
	}

	rows = render(t, 20, 3, DividerWithTitle("Logs", SymbolsSingle, NewStyle()))
	if !strings.Contains(rows[0], "Logs") {
		t.Errorf("DividerWithTitle lost its title: %q", rows[0])
	}
}

func TestSpacerPushesContentApart(t *testing.T) {
	rows := render(t, 12, 3, VStack(Label("top"), Spacer(), Label("bottom")))
	if _, y, _ := find(rows, "top"); y != 0 {
		t.Errorf("top at row %d", y)
	}
	if _, y, ok := find(rows, "bottom"); !ok || y != 2 {
		t.Errorf("Spacer did not push 'bottom' to the last row (row %d)", y)
	}
}

func TestForEachAndDynamicAndTextRef(t *testing.T) {
	items := []string{"one", "two", "three"}
	rows := render(t, 20, 4, VStack(ForEach(items, func(s string, i int) Component {
		return Label(s)
	})...))
	for i, want := range items {
		if got := strings.TrimSpace(rows[i]); got != want {
			t.Errorf("row %d = %q, want %q", i, got, want)
		}
	}

	calls := 0
	dyn := Dynamic(func() Component {
		calls++
		return Label("built")
	})
	if calls != 0 {
		t.Error("Dynamic should not call its supplier before it is drawn")
	}
	if rows = render(t, 20, 2, dyn); strings.TrimSpace(rows[0]) != "built" {
		t.Errorf("Dynamic drew %q", rows[0])
	}
	if calls == 0 {
		t.Error("Dynamic never called its supplier")
	}

	text := "before"
	ref := TextRef(&text)
	if rows = render(t, 20, 2, ref); strings.TrimSpace(rows[0]) != "before" {
		t.Errorf("TextRef drew %q", rows[0])
	}
	text = "after"
	if rows = render(t, 20, 2, ref); strings.TrimSpace(rows[0]) != "after" {
		t.Errorf("TextRef did not follow the pointer: %q", rows[0])
	}
}

func TestBorderHelpersDrawOnlyTheEdgeTheyName(t *testing.T) {
	rows := render(t, 10, 3, TopBorder(Label("x"), '-', NewStyle()))
	if strings.TrimSpace(rows[0]) == "" {
		t.Error("TopBorder drew no top edge")
	}
	if strings.TrimSpace(rows[2]) != "" {
		t.Errorf("TopBorder also drew a bottom edge: %q", rows[2])
	}

	rows = render(t, 10, 3, BottomBorder(Label("x"), '-', NewStyle()))
	if strings.TrimSpace(rows[2]) == "" {
		t.Error("BottomBorder drew no bottom edge")
	}
}

func TestTextTransformsRewriteWhatIsDrawn(t *testing.T) {
	if got := strings.TrimSpace(render(t, 20, 2, Uppercase(Label("quiet")))[0]); got != "QUIET" {
		t.Errorf("Uppercase drew %q", got)
	}
	if got := strings.TrimSpace(render(t, 20, 2, Lowercase(Label("LOUD")))[0]); got != "loud" {
		t.Errorf("Lowercase drew %q", got)
	}
	if got := strings.TrimSpace(render(t, 20, 2, Mask(Label("hunter2"), '*'))[0]); got != "*******" {
		t.Errorf("Mask drew %q, want the password hidden", got)
	}
	// Transform is the general form the three above are built on. It sweeps
	// the whole bounding area, blanks included — which is why Mask guards
	// whitespace and why "abc" shifted by one leaves '!' (space + 1) behind
	// it. That is the documented behaviour, not an accident.
	rot := Transform(Label("abc"), func(r rune) rune { return r + 1 })
	row := render(t, 20, 2, rot)[0]
	if !strings.HasPrefix(row, "bcd") {
		t.Errorf("Transform drew %q", row)
	}
	guarded := Transform(Label("abc"), func(r rune) rune {
		if r <= ' ' {
			return r
		}
		return r + 1
	})
	if got := strings.TrimSpace(render(t, 20, 2, guarded)[0]); got != "bcd" {
		t.Errorf("a whitespace-guarded transform drew %q", got)
	}
}

func TestOverlayDrawsOnTopAtTheGivenOffset(t *testing.T) {
	rows := render(t, 12, 3, Overlay(Label(strings.Repeat(".", 12)), Label("X"), 4, 1))
	if got := cellAt(rows, 4, 1); got != 'X' {
		t.Errorf("overlay landed as %q at (4,1); row: %q", got, rows[1])
	}
	if got := cellAt(rows, 0, 0); got != '.' {
		t.Errorf("the base was lost: %q", rows[0])
	}
}

func TestInlineKeepsAComponentToOneRow(t *testing.T) {
	rows := render(t, 12, 4, Inline(VStack(Label("one"), Label("two"))))
	if strings.TrimSpace(rows[1]) != "" {
		t.Errorf("Inline drew a second row: %q", rows[1])
	}
}

func TestTextFnAndBorderCustom(t *testing.T) {
	// The getter is consulted while measuring as well as while drawing, so
	// the count goes up by more than one per frame. What matters is that a
	// later frame shows the later value.
	n := 0
	comp := TextFn(func() string {
		n++
		return "call " + strings.Repeat("!", n)
	})
	first := strings.TrimSpace(render(t, 20, 2, comp)[0])
	if !strings.HasPrefix(first, "call !") {
		t.Errorf("TextFn drew %q", first)
	}
	second := strings.TrimSpace(render(t, 20, 2, comp)[0])
	if len(second) <= len(first) {
		t.Errorf("TextFn did not pick up the new value: %q then %q", first, second)
	}

	// Only the edges asked for are drawn.
	rows := render(t, 10, 3, BorderCustom(Label("x"), SymbolsSingle, NewStyle(), BorderEdgeTop))
	if strings.TrimSpace(rows[0]) == "" {
		t.Error("BorderCustom drew no top edge")
	}
	if strings.TrimSpace(rows[2]) != "" {
		t.Errorf("BorderCustom drew an edge it was not asked for: %q", rows[2])
	}
}

func TestLayoutConstraintHelpers(t *testing.T) {
	area := NewRect(0, 0, 30, 10)

	// Ratio splits what is left in proportion: 30 columns as 20 and 10.
	parts := SplitHorizontal(area, Ratio(2), Ratio(1))
	if len(parts) != 2 {
		t.Fatalf("got %d parts", len(parts))
	}
	if parts[0].Width != 20 || parts[1].Width != 10 {
		t.Errorf("Ratio(2)/Ratio(1) gave %d and %d columns, want 20 and 10", parts[0].Width, parts[1].Width)
	}
	if parts[1].X != 20 {
		t.Errorf("the second pane starts at %d, want 20", parts[1].X)
	}

	// Fixed and Percentage are measured against the same area.
	parts = SplitHorizontal(area, Fixed(7), Fill())
	if parts[0].Width != 7 || parts[1].Width != 23 {
		t.Errorf("Fixed(7)/Fill() gave %d and %d", parts[0].Width, parts[1].Width)
	}
	parts = SplitHorizontal(area, Percentage(50), Fill())
	if parts[0].Width != 15 {
		t.Errorf("Percentage(50) of 30 = %d, want 15", parts[0].Width)
	}

	// Min and Max bound a greedy neighbour.
	parts = SplitHorizontal(area, Min(8), Fill())
	if parts[0].Width < 8 {
		t.Errorf("Min(8) got %d columns", parts[0].Width)
	}
	parts = SplitHorizontal(area, Max(6), Fill())
	if parts[0].Width > 6 {
		t.Errorf("Max(6) took %d columns", parts[0].Width)
	}

	// The vertical split is the same arithmetic on the other axis.
	rowsOf := SplitVertical(area, Ratio(1), Ratio(1))
	if rowsOf[0].Height != 5 || rowsOf[1].Y != 5 {
		t.Errorf("SplitVertical gave %+v", rowsOf)
	}
}

// The options are the public way to configure an application, and each one
// only sets a field — which is exactly the kind of wiring that goes wrong
// silently. Reading the config back is the whole test.
func configOf(opts ...AppOption) appConfig {
	var cfg appConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

func TestAppOptionsSetWhatTheyName(t *testing.T) {
	if cfg := configOf(WithFPS(30)); cfg.fps != 30 {
		t.Errorf("WithFPS(30) → fps %d", cfg.fps)
	}
	if cfg := configOf(WithSuspend()); !cfg.suspendOnCtrlZ {
		t.Error("WithSuspend did not enable Ctrl+Z")
	}
	if cfg := configOf(WithInlineFit(9)); cfg.inlineHeight != 9 || !cfg.inlineFit {
		t.Errorf("WithInlineFit(9) → height %d, fit %v", cfg.inlineHeight, cfg.inlineFit)
	}
	if cfg := configOf(WithInline(7)); cfg.inlineHeight != 7 {
		t.Errorf("WithInline(7) → height %d", cfg.inlineHeight)
	}
	if cfg := configOf(WithTitle("zest")); !cfg.hasTitle || cfg.title != "zest" {
		t.Errorf("WithTitle → %q (set: %v)", cfg.title, cfg.hasTitle)
	}
	if cfg := configOf(WithCatchCtrlC(true)); !cfg.catchCtrlC {
		t.Error("WithCatchCtrlC(true) did not take")
	}
	if cfg := configOf(WithoutDefaultQuitKeys()); !cfg.catchCtrlC {
		t.Error("WithoutDefaultQuitKeys should hand Ctrl+C to the application")
	}

	policy := AutomationPolicy{ExposeScreen: true, AllowInput: true}
	cfg := configOf(WithAutomation("/tmp/x.sock", policy))
	if cfg.automationPath != "/tmp/x.sock" || cfg.automationPolicy != policy {
		t.Errorf("WithAutomation → %q %+v", cfg.automationPath, cfg.automationPolicy)
	}

	// Later options win, and a nil option is ignored rather than panicking.
	if cfg := configOf(WithFPS(10), nil, WithFPS(60)); cfg.fps != 60 {
		t.Errorf("the last option should win: fps %d", cfg.fps)
	}
}

// The state constructors are what an application holds between frames. A
// wrong zero value here shows up as a widget that will not scroll or select.
func TestStateConstructorsStartUsable(t *testing.T) {
	// Selected starts at -1, meaning nothing is selected — a fresh list must
	// not act as though its first row had been chosen.
	if s := NewListState(); s == nil || s.Selected != -1 || s.Offset != 0 {
		t.Errorf("NewListState = %+v", s)
	}
	if s := NewTableState(); s == nil || s.Selected != -1 || s.SortColumn != -1 {
		t.Errorf("NewTableState = %+v", s)
	}
	if s := NewTextInputState(); s == nil || len(s.Text) != 0 || s.Cursor != 0 {
		t.Errorf("NewTextInputState = %+v", s)
	}
	if s := NewTextAreaState(); s == nil {
		t.Error("NewTextAreaState = nil")
	}
	if s := NewSelectState(); s == nil {
		t.Error("NewSelectState = nil")
	}
	if s := NewSliderState(42); s == nil || s.Value != 42 {
		t.Errorf("NewSliderState(42) = %+v", s)
	}
	if s := NewViewportState(); s == nil {
		t.Error("NewViewportState = nil")
	}
}

// Text builders: the pieces a rich line is assembled from.
func TestTextBuildersRenderTheirContent(t *testing.T) {
	line := NewLine(NewSpan("red", Fg(ColorRed)), NewSpan(" plain", NewStyle()))
	text := NewText(line)

	term := testkit.NewTerminal(20, 2)
	term.Render(text, term.Area())
	got := strings.TrimSpace(strings.Split(term.Snapshot(), "\n")[0])
	if got != "red plain" {
		t.Errorf("rendered %q, want %q", got, "red plain")
	}
}

func TestScrollbarMetricsGrowAndClamp(t *testing.T) {
	const track = 10

	// Half the content visible: the thumb takes half the track.
	start, length := ScrollbarMetrics(20, 10, 0, track)
	if length != 5 {
		t.Errorf("thumb length %d for half the content, want 5", length)
	}
	if start != 0 {
		t.Errorf("at the top the thumb starts at 0, got %d", start)
	}

	// At the bottom it sits against the end of the track and no further.
	start, length = ScrollbarMetrics(20, 10, 10, track)
	if start+length != track {
		t.Errorf("thumb %d at %d does not reach the end of a %d-row track", length, start, track)
	}

	// An offset past the end is clamped, not extrapolated.
	if s2, _ := ScrollbarMetrics(20, 10, 999, track); s2 != start {
		t.Errorf("an over-scrolled offset moved the thumb to %d, want %d", s2, start)
	}

	// Content that fits needs no thumb at all.
	if start, length = ScrollbarMetrics(5, 10, 0, track); start != 0 || length != 0 {
		t.Errorf("content that fits gave a thumb of %d at %d", length, start)
	}

	// A thumb never disappears while there is something to scroll.
	if _, length = ScrollbarMetrics(10000, 10, 0, track); length < 1 {
		t.Error("a very long document left no thumb to grab")
	}
}

// The style helpers each produce exactly their own modifier and nothing else,
// which is the one way a facade like this goes wrong.
func TestStyleHelpersEachSetOnlyTheirOwnBit(t *testing.T) {
	cases := []struct {
		name string
		got  Style
		want Modifier
	}{
		{"Bold", Bold(), ModifierBold},
		{"Italic", Italic(), ModifierItalic},
		{"Underline", Underline(), ModifierUnderline},
		{"Dim", Dim(), ModifierDim},
		{"Reverse", Reverse(), ModifierReverse},
	}
	for _, c := range cases {
		if c.got.Modifier != c.want {
			t.Errorf("%s() set %v, want exactly %v", c.name, c.got.Modifier, c.want)
		}
		if c.got.Fg != ColorDefault || c.got.Bg != ColorDefault {
			t.Errorf("%s() also set a colour: %+v", c.name, c.got)
		}
	}

	if got := Fg(ColorRed); got.Fg != ColorRed || got.Bg != ColorDefault {
		t.Errorf("Fg set %+v", got)
	}
	if got := Bg(ColorBlue); got.Bg != ColorBlue || got.Fg != ColorDefault {
		t.Errorf("Bg set %+v", got)
	}
	if got := ANSI(200); got != ANSI(200) || got.ANSI() != 200 {
		t.Errorf("ANSI(200) = %v", got)
	}
	if got := Hyperlink("https://example.com/facade"); got.LinkURL() != "https://example.com/facade" {
		t.Errorf("Hyperlink → %q", got.LinkURL())
	}
	// Hex tolerates the forms people actually write, and refuses the rest.
	for _, bad := range []string{"", "#12", "nope", "#12345"} {
		if got := Hex(bad); got != ColorDefault {
			t.Errorf("Hex(%q) = %v, want the default colour", bad, got)
		}
	}
	if got := Hex("00FF00"); got != RGB(0, 255, 0) {
		t.Errorf("Hex without '#' = %v", got)
	}
}
