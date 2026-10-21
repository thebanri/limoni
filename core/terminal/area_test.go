package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// A view written for the whole screen, drawn into a pane: it sees the pane as
// its area, and nothing it draws or registers reaches outside.
func TestAreaHoldsAViewInItsPane(t *testing.T) {
	f := textFrame(12, 4)
	f.initClosures()
	pane := cell.NewRect(6, 1, 6, 3)

	f.BeginArea(pane)
	if got := f.Area(); got != pane {
		t.Fatalf("Area() = %+v, want the pane", got)
	}
	child := f.Area()
	f.RenderWidget(widgets.Block{Borders: widgets.BorderAll}, child)
	f.Text(int(child.X)-3, int(child.Y)+1, "overflowing", cell.Style{})           // starts left of the pane
	f.RegisterClickHandler(cell.NewRect(0, 0, 12, 4), func(driver.MouseEvent) {}) // the whole screen
	f.RegisterClickHandler(cell.NewRect(0, 0, 3, 1), func(driver.MouseEvent) {})  // only outside
	f.EndArea()

	if got := f.Area(); got != cell.NewRect(0, 0, 12, 4) {
		t.Fatalf("after EndArea Area() = %+v", got)
	}
	want := []string{"", "      ┌────┐", "      rflowi", "      └────┘"}
	for y, w := range want {
		if got := row(f, y); got != w {
			t.Errorf("row %d = %q, want %q", y, got, w)
		}
	}
	if len(f.ClickRegions) != 1 || f.ClickRegions[0].Area != pane {
		t.Fatalf("click regions = %+v, want one, cut to the pane", f.ClickRegions)
	}
}

func TestAreasNest(t *testing.T) {
	f := textFrame(20, 10)
	f.BeginArea(cell.NewRect(2, 2, 10, 5))
	f.BeginArea(cell.NewRect(8, 0, 10, 4)) // reaches out of the outer one
	if got := f.Area(); got != cell.NewRect(8, 2, 4, 2) {
		t.Fatalf("inner Area() = %+v, want it cut to the outer area", got)
	}
	f.EndArea()
	if got := f.Area(); got != cell.NewRect(2, 2, 10, 5) {
		t.Fatalf("after the inner EndArea Area() = %+v", got)
	}
	f.Text(0, 3, "abcdefghijklmnop", cell.Style{})
	if got := row(f, 3); got != "  cdefghijkl" {
		t.Fatalf("outer clip after the inner ended: %q", got)
	}
	f.EndArea()
	f.EndArea() // one too many does nothing
	f.Text(0, 0, "free", cell.Style{})
	if got := row(f, 0); got != "free" {
		t.Fatalf("row 0 = %q", got)
	}
}

func TestAreaLeftOpenIsClosedByReset(t *testing.T) {
	f := textFrame(10, 2)
	f.BeginArea(cell.NewRect(5, 0, 5, 1))
	f.Reset()
	f.Text(0, 1, "all", cell.Style{})
	if got := row(f, 1); got != "all" || f.Area() != cell.NewRect(0, 0, 10, 2) {
		t.Fatalf("after Reset: row 1 = %q, Area() = %+v", got, f.Area())
	}
}

func TestAreaDoesNotAllocate(t *testing.T) {
	f := textFrame(40, 10)
	pane := cell.NewRect(10, 2, 20, 6)
	allocs := testing.AllocsPerRun(100, func() {
		f.BeginArea(pane)
		f.Text(10, 2, "hello", cell.Style{})
		f.EndArea()
	})
	if allocs != 0 {
		t.Fatalf("BeginArea/EndArea allocated %.0f times per frame", allocs)
	}
}
