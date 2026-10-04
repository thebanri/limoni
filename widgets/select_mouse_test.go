package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// Through a real frame: a click opens the list, a click on an option picks it
// and reports it, the wheel steps and wraps, and an open list draws without
// allocating.
func TestSelectMouseThroughAFrame(t *testing.T) {
	state := widgets.NewSelectState()
	var picked []string
	sel := &widgets.Select{ID: "size", Options: []string{"S", "M", "L"}, State: state,
		OnChange: func(_ int, option string) { picked = append(picked, option) }}
	term := testkit.NewTerminal(12, 4)
	draw := func(f *terminal.Frame) { f.RenderWidget(sel, f.Area()) }
	term.Draw(draw)

	term.Click(2, 0)
	if !state.Open || term.Focused() != "size" {
		t.Fatalf("a click did not open the list: open %v, focused %q", state.Open, term.Focused())
	}
	term.Draw(draw)
	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("an open list allocated %v times a frame", n)
	}
	term.Click(2, 3) // the third option
	if state.Selected != 2 || state.Open {
		t.Errorf("clicking L: selected %d, open %v", state.Selected, state.Open)
	}
	term.Draw(draw)
	term.Mouse(driver.MouseEvent{Button: driver.MouseScrollDown, X: 2, Y: 0})
	if state.Selected != 0 {
		t.Errorf("wheel down from the last option gave %d, want 0 (wraps)", state.Selected)
	}
	if want := []string{"L", "S"}; len(picked) != 2 || picked[0] != want[0] || picked[1] != want[1] {
		t.Errorf("OnChange saw %v, want %v", picked, want)
	}
}
