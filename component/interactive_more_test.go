package component_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
)

// OnClick draws its child and, clicked through a frame, calls the handler
// with a left press at the component's origin; a frame allocates nothing.
func TestOnClickThroughAFrame(t *testing.T) {
	var got []driver.MouseEvent
	button := component.OnClick(component.Text("press"), func(ev driver.MouseEvent) { got = append(got, ev) })
	term := testkit.NewTerminal(20, 3)
	draw := func(f *terminal.Frame) { f.RenderComponent(button, cell.NewRect(2, 1, 10, 1)) }
	term.Draw(draw)
	if !strings.Contains(term.Snapshot(), "press") {
		t.Fatalf("the child was not drawn:\n%s", term.Snapshot())
	}
	term.Click(4, 1)
	if len(got) != 1 || got[0].Button != driver.MouseLeft || got[0].X != 2 || got[0].Y != 1 {
		t.Fatalf("handler saw %+v", got)
	}
	term.Draw(draw)
	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}

// Events dispatched down a tree reach the handlers that match them.
func TestKeyAndEventHandlersDispatch(t *testing.T) {
	quit, saved, clicked := false, false, false
	tree := component.VStack(
		component.OnKey(component.Text("a"), driver.KeyEsc, func(driver.KeyEvent) bool { quit = true; return true }),
		component.OnRune(component.Text("b"), 's', func(driver.KeyEvent) bool { saved = true; return true }),
		component.OnClick(component.Text("c"), func(driver.MouseEvent) { clicked = true }),
	)
	ctx := cell.NewContext(cell.NewRect(0, 0, 10, 3), cell.Style{})
	component.DispatchEvent(tree, ctx, &driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyRune, Ch: 's'}})
	component.DispatchEvent(tree, ctx, &driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyEsc}})
	if !quit || !saved {
		t.Errorf("Esc reached OnKey: %v, s reached OnRune: %v", quit, saved)
	}
	if component.DispatchEvent(tree, ctx, &driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyRune, Ch: 'x'}}) {
		t.Error("a key no one handles was reported handled")
	}
	component.DispatchEvent(tree, ctx, &driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseLeft, X: 0, Y: 2}})
	if !clicked {
		t.Error("a click on the third row did not reach OnClick")
	}
}

// Dividers and single-edge borders draw their rule.
func TestDividersAndEdgeBorders(t *testing.T) {
	term := testkit.NewTerminal(6, 4)
	term.Draw(func(f *terminal.Frame) {
		f.RenderComponent(component.TopBorder(component.Text("x"), '=', cell.Style{}), cell.NewRect(0, 0, 6, 2))
		f.RenderComponent(component.HStack(component.Text("ab"), component.VDividerCustom('!'), component.Text("cd")), cell.NewRect(0, 2, 6, 1))
		f.RenderComponent(component.DividerCustom('~'), cell.NewRect(0, 3, 6, 1))
	})
	rows := strings.Split(term.Snapshot(), "\n")
	if rows[0] != "======" || !strings.HasPrefix(rows[1], "x") {
		t.Errorf("top border rows %q", rows[:2])
	}
	if !strings.Contains(rows[2], "!") || rows[3] != "~~~~~~" {
		t.Errorf("divider rows %q", rows[2:4])
	}
}
