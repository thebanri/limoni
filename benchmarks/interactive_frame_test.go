package benchmarks

import (
	"testing"

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
