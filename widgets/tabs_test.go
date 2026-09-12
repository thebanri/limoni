package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestTabsDrawLayout(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 1)
	buf := buffer.NewBuffer(area)

	Tabs{Titles: []string{"Home", "Charts"}, Selected: 0}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	// " Home " + "│" + " Charts "
	if got, want := row(buf, 0, 0, 16), " Home │ Charts  "; got != want {
		t.Fatalf("tab bar = %q, want %q", got, want)
	}
}

func TestTabsWidthMatchesRendering(t *testing.T) {
	tabs := Tabs{Titles: []string{"Home", "Charts", "Settings"}}

	area := cell.NewRect(0, 0, 60, 1)
	buf := buffer.NewBuffer(area)
	tabs.Draw(cell.NewContext(area, cell.Style{}), buf)

	// Width must equal the number of non-blank trailing cells actually used.
	used := uint16(0)
	for x := uint16(0); x < area.Width; x++ {
		if c := buf.Get(x, 0); c != nil && c.Content != ' ' {
			used = x + 1
		}
	}
	// The rendered run ends with the trailing pad space, so Width is one more.
	if got := tabs.Width(); got != used+1 {
		t.Fatalf("Width() = %d, rendered content ends at %d (+1 pad)", got, used)
	}
}

func TestTabsSelectedStyleAppliesToPadding(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 1)
	buf := buffer.NewBuffer(area)

	accent := cell.Style{Fg: cell.NewColorRGB(0, 255, 200)}
	Tabs{Titles: []string{"One", "Two"}, Selected: 1, SelectedStyle: accent}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	// " One │ Two " -> the second tab starts at index 6 with its leading pad.
	padCell := buf.Get(6, 0)
	if padCell == nil || padCell.Style.Fg != accent.Fg {
		t.Fatalf("selected tab padding did not get the selected style: %+v", padCell)
	}
}

func TestTabsCustomDividerAndPadding(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 1)
	buf := buffer.NewBuffer(area)

	empty := ""
	Tabs{Titles: []string{"ab", "cd"}, Divider: &empty, PaddingZero: true}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 4), "abcd"; got != want {
		t.Fatalf("dividerless tabs = %q, want %q", got, want)
	}
}

func TestTabsClickRegions(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 1)
	buf := buffer.NewBuffer(area)

	type region struct {
		rect    cell.Rect
		handler func()
	}
	var regions []region

	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterClick = func(r cell.Rect, h func()) { regions = append(regions, region{r, h}) }

	selected := -1
	Tabs{
		Titles:   []string{"Home", "Charts"},
		OnSelect: func(i int) { selected = i },
	}.Draw(ctx, buf)

	if len(regions) != 2 {
		t.Fatalf("registered %d click regions, want 2", len(regions))
	}
	// " Home " is 6 wide starting at 0; "│" takes one; " Charts " starts at 7.
	if regions[0].rect.X != 0 || regions[0].rect.Width != 6 {
		t.Fatalf("first region = %+v, want x=0 width=6", regions[0].rect)
	}
	if regions[1].rect.X != 7 || regions[1].rect.Width != 8 {
		t.Fatalf("second region = %+v, want x=7 width=8", regions[1].rect)
	}

	regions[1].handler()
	if selected != 1 {
		t.Fatalf("clicking the second tab selected %d, want 1", selected)
	}
}

// A bar wider than its area must clip instead of writing out of bounds.
func TestTabsClipsToNarrowArea(t *testing.T) {
	area := cell.NewRect(0, 0, 8, 1)
	buf := buffer.NewBuffer(area)

	Tabs{Titles: []string{"Alpha", "Beta", "Gamma"}}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 8), " Alpha │"; got != want {
		t.Fatalf("clipped tab bar = %q, want %q", got, want)
	}
}

func TestTabsWideRunesMeasuredCorrectly(t *testing.T) {
	tabs := Tabs{Titles: []string{"日本"}}
	// Two double-width runes plus two pad columns.
	if got, want := tabs.Width(), uint16(6); got != want {
		t.Fatalf("Width() with wide runes = %d, want %d", got, want)
	}
}

func TestTabsEmptyIsNoop(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	buf := buffer.NewBuffer(area)
	Tabs{}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got := row(buf, 0, 0, 10); got != "          " {
		t.Fatalf("empty tabs drew %q, want blank", got)
	}
}

func TestTabsSizeHint(t *testing.T) {
	tabs := Tabs{Titles: []string{"Home", "Charts"}}
	if w, h := tabs.SizeHint(cell.NewRect(0, 0, 100, 5)); w != tabs.Width() || h != 1 {
		t.Fatalf("SizeHint = (%d,%d), want (%d,1)", w, h, tabs.Width())
	}
	if w, _ := tabs.SizeHint(cell.NewRect(0, 0, 4, 1)); w != 4 {
		t.Fatalf("SizeHint in a narrow area = %d, want 4", w)
	}
}

func BenchmarkTabsDraw(b *testing.B) {
	area := cell.NewRect(0, 0, 80, 1)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	tabs := Tabs{Titles: []string{"Home", "Charts", "Tables", "3D", "Settings"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tabs.Selected = i % 5
		tabs.Draw(ctx, buf)
	}
}
