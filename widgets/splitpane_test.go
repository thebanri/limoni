package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// fill paints its whole area with one rune, so a test can see where a pane went.
type fill rune

func (f fill) Draw(ctx cell.Context, buf *buffer.Buffer) {
	for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
		for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: rune(f)})
		}
	}
}

func (f fill) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }

func rowOf(buf *buffer.Buffer, y, width uint16) string {
	r := make([]rune, width)
	for x := uint16(0); x < width; x++ {
		r[x] = buf.CellAt(x, y).Rune()
	}
	return string(r)
}

func TestSplitPaneLayout(t *testing.T) {
	area := cell.NewRect(0, 0, 11, 3)
	for _, tc := range []struct {
		name string
		p    SplitPane
		want string
	}{
		{"even by default", SplitPane{First: fill('a'), Second: fill('b')}, "aaaaa│bbbbb"},
		{"ratio", SplitPane{First: fill('a'), Second: fill('b'), State: &SplitState{Ratio: 0.2}}, "aa│bbbbbbbb"},
		{"min second wins", SplitPane{First: fill('a'), Second: fill('b'), State: &SplitState{Ratio: 0.9}, MinSecond: 4}, "aaaaaa│bbbb"},
		{"min first wins", SplitPane{First: fill('a'), Second: fill('b'), State: &SplitState{Ratio: 0.01}, MinFirst: 3}, "aaa│bbbbbbb"},
	} {
		buf := buffer.NewBuffer(area)
		tc.p.Draw(cell.NewContext(area, cell.Style{}), buf)
		if got := rowOf(buf, 1, 11); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}

	// Stacked: the divider is a row.
	area = cell.NewRect(0, 0, 3, 5)
	buf := buffer.NewBuffer(area)
	SplitPane{Direction: SplitVertical, First: fill('a'), Second: fill('b')}.Draw(cell.NewContext(area, cell.Style{}), buf)
	for y, want := range []string{"aaa", "aaa", "───", "bbb", "bbb"} {
		if got := rowOf(buf, uint16(y), 3); got != want {
			t.Errorf("stacked row %d: %q, want %q", y, got, want)
		}
	}
}

// Pressing the divider captures the mouse; dragging moves it, even when the
// pointer leaves the divider's own cells.
func TestSplitPaneDrag(t *testing.T) {
	area := cell.NewRect(2, 0, 21, 3)
	state := &SplitState{}
	pane := SplitPane{First: fill('a'), Second: fill('b'), State: state}

	var onDivider cell.Rect
	var handler, captured func(driver.MouseEvent)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(r cell.Rect, h func(driver.MouseEvent)) { onDivider, handler = r, h }
	ctx.CaptureMouse = func(h func(driver.MouseEvent)) { captured = h }

	buf := buffer.NewBuffer(cell.NewRect(0, 0, 30, 3))
	pane.Draw(ctx, buf)
	if onDivider != cell.NewRect(12, 0, 1, 3) {
		t.Fatalf("mouse region %v, want the divider column at x=12", onDivider)
	}
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 12, Y: 1})
	if captured == nil {
		t.Fatal("pressing the divider did not capture the mouse")
	}
	captured(driver.MouseEvent{Button: driver.MouseLeft, Drag: true, X: 7, Y: 1})

	pane.Draw(ctx, buf)
	if got := string([]rune(rowOf(buf, 1, 30))[2:23]); got != "aaaaa│bbbbbbbbbbbbbbb" {
		t.Errorf("after drag to x=7: %q", got)
	}
	// Dragged far past the edge: the divider stops at the last cell.
	captured(driver.MouseEvent{Button: driver.MouseLeft, Drag: true, X: 200, Y: 1})
	if state.Ratio != 1 {
		t.Errorf("ratio %v after dragging past the edge, want 1", state.Ratio)
	}
}

func TestSplitPaneKeys(t *testing.T) {
	area := cell.NewRect(0, 0, 11, 1)
	state := &SplitState{}
	pane := SplitPane{First: fill('a'), Second: fill('b'), State: state}
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	pane.Draw(ctx, buf)

	if state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowUp}) {
		t.Error("↑ moved a side-by-side divider")
	}
	for i := 0; i < 2; i++ {
		if !state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowLeft}) {
			t.Fatal("← not handled")
		}
	}
	pane.Draw(ctx, buf)
	if got := rowOf(buf, 0, 11); got != "aaa│bbbbbbb" {
		t.Errorf("two cells left: %q", got)
	}
}

func TestSplitPaneDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	ctx.RegisterMouse = func(cell.Rect, func(driver.MouseEvent)) {}
	for name, p := range map[string]SplitPane{
		"with state":    {First: fill('a'), Second: fill('b'), State: &SplitState{Ratio: 0.3}},
		"without state": {First: fill('a'), Second: fill('b')},
	} {
		p.Draw(ctx, buf)
		if a := testing.AllocsPerRun(50, func() { p.Draw(ctx, buf) }); a != 0 {
			t.Errorf("%s: %.0f allocs per Draw", name, a)
		}
	}
}

func BenchmarkSplitPaneDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	p := SplitPane{First: fill('a'), Second: fill('b'), State: &SplitState{Ratio: 0.3}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Draw(ctx, buf)
	}
}
