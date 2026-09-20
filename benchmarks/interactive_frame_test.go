package benchmarks

import (
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// interactiveScreen is a frame with widgets that register click and wheel
// handlers, drawn through a real Terminal with a theme set. The per-widget
// benchmarks call Draw without a frame, so they never saw the closures those
// handlers used to be: this frame allocated 15 times before click actions.
type interactiveScreen struct {
	checked  bool
	checkbox *widgets.Checkbox
	input    *widgets.TextInput
	list     *widgets.List
	block    *widgets.Block
}

func newInteractiveScreen() *interactiveScreen {
	s := &interactiveScreen{}
	state := widgets.NewTextInputState()
	state.SetValue("hello")
	s.checkbox = &widgets.Checkbox{ID: "check", Checked: &s.checked, Label: "Check"}
	s.input = &widgets.TextInput{ID: "input", State: state}
	s.list = &widgets.List{ID: "list", Items: []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta"}, State: widgets.NewListState()}
	s.block = widgets.NewBlock().WithTitle("panel")
	return s
}

func (s *interactiveScreen) draw(f *terminal.Frame) {
	f.SetTheme(widgets.DarkTheme())
	f.RenderWidget(s.checkbox, cell.NewRect(0, 0, 20, 1))
	f.RenderWidget(s.input, cell.NewRect(0, 1, 20, 1))
	f.RenderWidget(s.list, cell.NewRect(0, 2, 20, 4))
	f.RenderWidget(s.block, cell.NewRect(20, 0, 20, 6))
}

func newBenchTerminal(tb testing.TB) *terminal.Terminal {
	tb.Setenv("LIMONI_PROBE", "0")
	term, err := terminal.New(driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 80, 24)))
	if err != nil {
		tb.Fatal(err)
	}
	return term
}

func BenchmarkInteractiveFrame(b *testing.B) {
	term := newBenchTerminal(b)
	s := newInteractiveScreen()
	_ = term.Draw(s.draw)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = term.Draw(s.draw)
	}
}

// The allocation-free registrations still do what the closures did.
func TestInteractiveFrameHandlesClicks(t *testing.T) {
	term := newBenchTerminal(t)
	s := newInteractiveScreen()
	click := func(x, y uint16) {
		t.Helper()
		if err := term.Draw(s.draw); err != nil {
			t.Fatal(err)
		}
		term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseLeft, X: x, Y: y})
	}

	click(1, 0)
	if !s.checked {
		t.Error("clicking the checkbox did not toggle it")
	}
	click(1, 1)
	if got := term.FocusManager().Focused(); got != "input" {
		t.Errorf("clicking the input focused %q", got)
	}
	click(1, 4)
	if s.list.State.Selected != 2 {
		t.Errorf("clicking the third row selected %d", s.list.State.Selected)
	}
	if got := term.FocusManager().Focused(); got != "list" {
		t.Errorf("clicking a row focused %q", got)
	}

	_ = term.Draw(s.draw)
	for i := 0; i < 10; i++ {
		term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseScrollDown, X: 1, Y: 3})
	}
	if s.list.State.Offset != 3 { // 7 items, 4 rows: the offset stops at 3
		t.Errorf("wheel scrolled to offset %d, want 3", s.list.State.Offset)
	}
	term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseScrollUp, X: 1, Y: 3})
	if s.list.State.Offset != 2 {
		t.Errorf("wheel up left offset %d, want 2", s.list.State.Offset)
	}
}

// A widget inside a Block used to be invisible to the semantic tree — the
// frame only described widgets it rendered itself — and it lost every
// Context field Block did not copy by hand, the click actions among them.
func TestNestedWidgetsAreDescribedAndStayAllocationFree(t *testing.T) {
	term := newBenchTerminal(t)
	checked := false
	box := &widgets.Checkbox{ID: "nested", Checked: &checked, Label: "Nested"}
	block := &widgets.Block{Title: "outer", Borders: widgets.BorderAll, Child: box}
	var tree []accessibility.AccessibilityNode
	draw := func(f *terminal.Frame) {
		f.RenderWidget(block, cell.NewRect(0, 0, 30, 5))
		if tree == nil {
			tree = f.AccessibilityTree()
		}
	}

	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range tree {
		if n.ID == "nested" {
			found = true
		}
	}
	if !found {
		t.Fatal("the checkbox inside the block is not in the semantic tree")
	}
	term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseLeft, X: 2, Y: 1})
	if !checked {
		t.Fatal("clicking the nested checkbox did not toggle it")
	}
	if a := testing.AllocsPerRun(100, func() { _ = term.Draw(draw) }); a != 0 {
		t.Fatalf("a frame with a nested checkbox allocates %.0f times", a)
	}
}
