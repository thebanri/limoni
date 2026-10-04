package terminal_test

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// actionWidget registers one ClickAction over its whole area.
type actionWidget struct{ action cell.ClickAction }

func (w actionWidget) Draw(ctx cell.Context, _ *buffer.Buffer) {
	if ctx.RegisterFocus != nil && w.action.Focus != "" {
		ctx.RegisterFocus(w.action.Focus)
	}
	ctx.RegisterClickAction(ctx.Area, w.action)
}
func (w actionWidget) SizeHint(r cell.Rect) (uint16, uint16) { return r.Width, r.Height }

// Each kind of ClickAction does what it says when its area is clicked: the
// allocation-free stand-ins for the closures widgets used to register.
func TestClickActionsDoWhatTheySay(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 20, 4))
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	toggled := false
	selected := -1
	assigned := ""
	draw := func(f *terminal.Frame) {
		f.RenderWidget(actionWidget{cell.ClickAction{Focus: "box", Toggle: &toggled}}, cell.NewRect(0, 0, 20, 1))
		f.RenderWidget(actionWidget{cell.ClickAction{Select: &selected, Index: 3}}, cell.NewRect(0, 1, 20, 1))
		f.RenderWidget(actionWidget{cell.ClickAction{Assign: &assigned, Value: "dark"}}, cell.NewRect(0, 2, 20, 1))
		f.RenderWidget(actionWidget{cell.ClickAction{Pointer: "pointer"}}, cell.NewRect(0, 3, 20, 1))
	}
	click := func(y uint16) {
		t.Helper()
		if err := term.Draw(draw); err != nil {
			t.Fatal(err)
		}
		term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseLeft, X: 5, Y: y})
	}

	click(0)
	if !toggled || term.FocusManager().Focused() != "box" {
		t.Errorf("toggle: %v, focused %q", toggled, term.FocusManager().Focused())
	}
	click(0)
	if toggled {
		t.Error("a second click did not toggle back")
	}
	click(1)
	if selected != 3 {
		t.Errorf("select set %d, want 3", selected)
	}
	click(2)
	if assigned != "dark" {
		t.Errorf("assign set %q", assigned)
	}
	// A pointer-only action is not a click target: nothing changes.
	toggled, selected, assigned = false, -1, ""
	click(3)
	if toggled || selected != -1 || assigned != "" {
		t.Errorf("a pointer-only region acted on a click")
	}
}

// Behind an open modal, a widget's click actions are not registered and a
// click outside the modal goes to the modal's ClickOutside instead.
func TestClickActionsBehindAModalAreInert(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 20, 6))
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	behind, inside, outside := false, false, false
	modalArea := cell.NewRect(5, 2, 10, 3)
	draw := func(f *terminal.Frame) {
		f.RegisterModal("confirm", modalArea, func() { outside = true })
		f.RenderWidget(actionWidget{cell.ClickAction{Toggle: &behind}}, cell.NewRect(0, 0, 20, 1))
		f.RenderWidget(actionWidget{cell.ClickAction{Toggle: &inside}}, cell.NewRect(6, 3, 5, 1))
	}
	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseLeft, X: 1, Y: 0})
	if behind || !outside {
		t.Errorf("a click behind the modal: acted %v, ClickOutside %v", behind, outside)
	}
	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	term.RouteMouseEvent(driver.MouseEvent{Button: driver.MouseLeft, X: 7, Y: 3})
	if !inside {
		t.Error("a click inside the modal did nothing")
	}
}
