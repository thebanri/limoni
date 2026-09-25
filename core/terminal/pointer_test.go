package terminal_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func osc22Writes(out []byte) []string {
	var shapes []string
	for _, part := range strings.Split(string(out), "\x1b]22;")[1:] {
		shape, _, _ := strings.Cut(part, "\x1b\\")
		shapes = append(shapes, shape)
	}
	return shapes
}

// Moving over a SplitPane's divider shows a resize arrow, moving off puts
// the default back, and only changes are written. Pressing the divider still
// drags it: the pointer region must not take the click.
func TestPointerShapeFollowsTheMouse(t *testing.T) {
	term, io := memTerm(t, 21, 3)
	caps := term.Capabilities()
	caps.PointerShape = true
	term.SetCapabilities(caps)

	state := &widgets.SplitState{}
	pane := widgets.SplitPane{First: &widgets.Paragraph{Text: "a"}, Second: &widgets.Paragraph{Text: "b"}, State: state}
	draw := func() {
		if err := term.Draw(func(f *terminal.Frame) { f.RenderWidget(pane, cell.NewRect(0, 0, 21, 3)) }); err != nil {
			t.Fatal(err)
		}
	}
	draw()

	before := len(io.Output())
	move := func(x uint16) { term.RouteMouseEvent(driver.MouseEvent{X: x, Y: 1, Button: driver.MouseNone}) }
	move(3)
	move(10) // the divider
	move(10)
	move(15)
	if got := osc22Writes(io.Output()[before:]); strings.Join(got, ",") != "ew-resize," {
		t.Errorf("OSC 22 writes %q, want the arrow once and then the default", got)
	}

	// Drag the divider: the press reaches the pane's handler, and the arrow
	// stays while the drag holds the mouse, even off the divider.
	move(10)
	before = len(io.Output())
	term.RouteMouseEvent(driver.MouseEvent{X: 10, Y: 1, Button: driver.MouseLeft})
	term.RouteMouseEvent(driver.MouseEvent{X: 5, Y: 1, Button: driver.MouseLeft, Drag: true})
	term.RouteMouseEvent(driver.MouseEvent{X: 5, Y: 1, Button: driver.MouseRelease})
	if state.Ratio != 0.25 {
		t.Errorf("drag moved the divider to ratio %v, want 0.25", state.Ratio)
	}
	if got := osc22Writes(io.Output()[before:]); len(got) != 0 {
		t.Errorf("the pointer changed during the drag: %q", got)
	}

	// Closing puts the default pointer back.
	before = len(io.Output())
	_ = term.Close()
	if got := osc22Writes(io.Output()[before:]); len(got) != 1 || got[0] != "" {
		t.Errorf("Close wrote %q, want one reset", got)
	}
}

// Without the capability nothing is written, whatever the widgets ask for.
func TestPointerShapeNeedsCapability(t *testing.T) {
	term, io := memTerm(t, 21, 3)
	caps := term.Capabilities()
	caps.PointerShape = false
	term.SetCapabilities(caps)
	pane := widgets.SplitPane{First: &widgets.Paragraph{Text: "a"}, Second: &widgets.Paragraph{Text: "b"}, State: &widgets.SplitState{}}
	_ = term.Draw(func(f *terminal.Frame) { f.RenderWidget(pane, cell.NewRect(0, 0, 21, 3)) })
	term.RouteMouseEvent(driver.MouseEvent{X: 10, Y: 1, Button: driver.MouseNone})
	_ = term.Close()
	if got := osc22Writes(io.Output()); len(got) != 0 {
		t.Errorf("wrote OSC 22 without the capability: %q", got)
	}
}

func TestNotifyWritesOnlyWithCapability(t *testing.T) {
	term, io := memTerm(t, 10, 2)
	caps := term.Capabilities()
	caps.Notify = terminal.NotifyNone
	term.SetCapabilities(caps)
	if term.Notify("a", "b") || strings.Contains(string(io.Output()), "\x1b]9;") {
		t.Error("notified without the capability")
	}
	caps.Notify = terminal.NotifyOSC9
	term.SetCapabilities(caps)
	before := len(io.Output())
	if !term.Notify("Build", "done") || string(io.Output()[before:]) != "\x1b]9;Build: done\a" {
		t.Errorf("wrote %q", io.Output()[before:])
	}
}

// The kitty keyboard flags are pushed once, before the first frame that
// knows about them, and popped on Close, so the shell gets its own back.
func TestKittyKeyboardPushedAndPopped(t *testing.T) {
	term, io := memTerm(t, 10, 2)
	caps := term.Capabilities()
	caps.KittyKeyboard = true
	term.SetCapabilities(caps)
	for i := 0; i < 3; i++ {
		_ = term.Draw(func(*terminal.Frame) {})
	}
	if got := strings.Count(string(io.Output()), "\x1b[>1u"); got != 1 {
		t.Errorf("pushed %d times, want once", got)
	}
	_ = term.Close()
	out := string(io.Output())
	if strings.Count(out, "\x1b[<u") != 1 || strings.LastIndex(out, "\x1b[<u") < strings.Index(out, "\x1b[>1u") {
		t.Errorf("not popped once after the push: %q", out)
	}

	// A terminal without the protocol is never sent either.
	term, io = memTerm(t, 10, 2)
	caps = term.Capabilities()
	caps.KittyKeyboard = false
	term.SetCapabilities(caps)
	_ = term.Draw(func(*terminal.Frame) {})
	_ = term.Close()
	if strings.Contains(string(io.Output()), "\x1b[>1u") || strings.Contains(string(io.Output()), "\x1b[<u") {
		t.Errorf("kitty keyboard sequences without the capability: %q", io.Output())
	}
}

// SetKeyReleases pushes the event-type flag with the rest, or switches the
// entry already pushed in place; Close pops it all the same.
func TestKeyReleasesChangeTheKittyFlags(t *testing.T) {
	term, io := memTerm(t, 10, 2)
	term.SetKeyReleases(true)
	caps := term.Capabilities()
	caps.KittyKeyboard = true
	term.SetCapabilities(caps)
	_ = term.Draw(func(*terminal.Frame) {})
	if out := string(io.Output()); !strings.Contains(out, "\x1b[>3u") || strings.Contains(out, "\x1b[>1u") {
		t.Fatalf("pushed %q, want flags 3", out)
	}
	term.SetKeyReleases(false)
	term.SetKeyReleases(false) // no second write
	if got := strings.Count(string(io.Output()), "\x1b[=1;1u"); got != 1 {
		t.Errorf("switched back %d times, want once", got)
	}
	_ = term.Close()
	if out := string(io.Output()); strings.Count(out, "\x1b[<u") != 1 || strings.Index(out, "\x1b[<u") < strings.Index(out, "\x1b[=1;1u") {
		t.Errorf("not popped once after the change: %q", out)
	}

	// Asked for after the push, the entry is changed in place.
	term, io = memTerm(t, 10, 2)
	term.SetCapabilities(caps)
	_ = term.Draw(func(*terminal.Frame) {})
	term.SetKeyReleases(true)
	if out := string(io.Output()); !strings.Contains(out, "\x1b[>1u") || !strings.Contains(out, "\x1b[=3;1u") {
		t.Errorf("wrote %q", out)
	}
	_ = term.Close()

	// Without the protocol nothing is written.
	term, io = memTerm(t, 10, 2)
	term.SetKeyReleases(true)
	_ = term.Draw(func(*terminal.Frame) {})
	if strings.Contains(string(io.Output()), "u") && strings.Contains(string(io.Output()), "\x1b[>") {
		t.Errorf("kitty flags without the capability: %q", io.Output())
	}
}
