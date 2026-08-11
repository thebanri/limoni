package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func TestTerminalDrawAndSnapshot(t *testing.T) {
	testTerm := NewTerminal(6, 2)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetString(0, 0, "hello", cell.Style{})
	})
	if got, want := testTerm.Snapshot(), "hello \n      "; got != want {
		t.Fatalf("snapshot = %q, want %q", got, want)
	}

	testTerm.Resize(3, 1)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetString(0, 0, "world", cell.Style{})
	})
	if got, want := testTerm.Snapshot(), "wor"; got != want {
		t.Fatalf("resized snapshot = %q, want %q", got, want)
	}
}

func TestTerminalStyleSnapshot(t *testing.T) {
	testTerm := NewTerminal(1, 1)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetCell(0, 0, cell.Cell{Content: 'x', Style: cell.Style{Fg: cell.NewColorANSI(2), Modifier: cell.ModifierBold}})
	})
	if got := testTerm.StyleSnapshot(); got != "01000002/00000000/0001" {
		t.Fatalf("style snapshot = %q", got)
	}
	if err := testTerm.AssertSnapshot("x"); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalMouseClickAndCapture(t *testing.T) {
	testTerm := NewTerminal(10, 2)
	clicked := false
	dragged := false
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterClickHandler(cell.NewRect(2, 0, 4, 1), func(ev backend.MouseEvent) {
			clicked = true
			frame.CaptureMouse(func(ev backend.MouseEvent) {
				if ev.Drag {
					dragged = true
				}
			})
		})
	})

	if testTerm.Click(3, 0) != true || !clicked {
		t.Fatal("expected click to be routed")
	}
	if testTerm.Drag(7, 0) != true || !dragged {
		t.Fatal("expected captured drag to be routed")
	}
	if testTerm.Mouse(backend.MouseEvent{X: 0, Y: 0, Button: backend.MouseLeft}) {
		t.Fatal("expected a new click outside the registered area to be ignored")
	}
}

func TestTerminalRenderAndClickWidget(t *testing.T) {
	state := widgets.NewSliderState(0)
	slider := widgets.Slider{ID: "volume", State: state, Min: 0, Max: 100}
	testTerm := NewTerminal(12, 1)
	testTerm.Render(slider, cell.NewRect(0, 0, 11, 1))

	if !testTerm.Click(10, 0) {
		t.Fatal("expected slider click to be routed")
	}
	if state.Value != 100 {
		t.Fatalf("slider value = %d, want 100", state.Value)
	}
}

func TestTerminalFocusAndPropagationSnapshot(t *testing.T) {
	testTerm := NewTerminal(20, 4)
	order := make([]terminal.EventPhase, 0, 3)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventHandler(cell.NewRect(0, 0, 20, 4), terminal.CapturePhase, func(*terminal.EventContext) {
			order = append(order, terminal.CapturePhase)
		})
		frame.RegisterEventHandler(cell.NewRect(2, 1, 4, 1), terminal.TargetPhase, func(*terminal.EventContext) {
			order = append(order, terminal.TargetPhase)
		})
		frame.RegisterEventHandler(cell.NewRect(0, 0, 20, 4), terminal.BubblePhase, func(*terminal.EventContext) {
			order = append(order, terminal.BubblePhase)
		})
		frame.FocusManager.Register("first")
		frame.FocusManager.Register("second")
	})

	if got := testTerm.FocusableIDs(); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("focusable IDs = %v, want [first second]", got)
	}
	if !testTerm.PropagateMouse(backend.MouseEvent{X: 3, Y: 1, Button: backend.MouseLeft}) {
		t.Fatal("expected propagation to be handled")
	}
	if len(order) != 3 || order[0] != terminal.CapturePhase || order[1] != terminal.TargetPhase || order[2] != terminal.BubblePhase {
		t.Fatalf("propagation order = %v, want capture/target/bubble", order)
	}
}

func TestTerminalKeyResizeAndLayerAssertions(t *testing.T) {
	testTerm := NewTerminal(8, 4)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterLayer("popup", terminal.LayerPopup, cell.NewRect(1, 1, 4, 2), 5, nil)
		frame.RegisterLayer("modal", terminal.LayerModal, cell.NewRect(2, 1, 3, 2), 10, nil)
	})
	seen := false
	if !testTerm.SendKey(backend.KeyEvent{Type: backend.KeyEnter}, func(key backend.KeyEvent) bool { seen = key.Type == backend.KeyEnter; return seen }) || !seen {
		t.Fatal("key was not injected")
	}
	if got := testTerm.ResizeEvent().Resize.Width; got != 8 {
		t.Fatalf("resize width = %d", got)
	}
	if ids := testTerm.LayerIDs(); len(ids) != 2 || ids[0] != "popup" || ids[1] != "modal" {
		t.Fatalf("layers = %v", ids)
	}
	if !testTerm.HasModal() {
		t.Fatal("expected modal assertion to be true")
	}
}
