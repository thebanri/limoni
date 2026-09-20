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
	term.Draw(func(f *Frame) { f.RenderComponent(c, f.Area()) })
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
	rows := render(t, 21, 5, Place(21, 5, HAlignCenter, AlignMiddle, Label("x")))
	x, y, ok := find(rows, "x")
	if !ok {
		t.Fatal("nothing drawn")
	}
	if x != 10 || y != 2 {
		t.Errorf("centred label at (%d,%d), want (10,2)", x, y)
	}

	rows = render(t, 21, 5, Place(21, 5, HAlignRight, AlignBottom, Label("x")))
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
