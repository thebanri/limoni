package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
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

func TestTerminalSemanticRegionMouseDispatch(t *testing.T) {
	testTerm := NewTerminal(20, 4)
	var seen []backend.MouseButton
	entered := false
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventRegion(terminal.EventRegion{
			Area:  cell.NewRect(1, 1, 8, 2),
			ID:    "viewport",
			Phase: terminal.TargetPhase,
			Handler: func(ctx *terminal.EventContext) {
				seen = append(seen, ctx.Mouse.Button)
			},
			OnEnter: func(ctx *terminal.EventContext) {
				entered = ctx.RegionID == "viewport"
			},
		})
	})

	if !testTerm.Mouse(backend.MouseEvent{X: 2, Y: 1, Button: backend.MouseNone}) {
		t.Fatal("semantic hover event was not handled")
	}
	if !entered {
		t.Fatal("semantic region enter callback was not called")
	}
	if !testTerm.Click(2, 1) {
		t.Fatal("semantic click was not handled")
	}
	if !testTerm.Mouse(backend.MouseEvent{X: 2, Y: 1, Button: backend.MouseScrollDown}) {
		t.Fatal("semantic wheel event was not handled")
	}

	want := []backend.MouseButton{backend.MouseNone, backend.MouseLeft, backend.MouseScrollDown}
	if len(seen) != len(want) {
		t.Fatalf("semantic event count = %d, want %d", len(seen), len(want))
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("semantic event %d = %d, want %d", i, seen[i], want[i])
		}
	}
}

func TestTerminalAccessibilityAssertionAndKeySequence(t *testing.T) {
	testTerm := NewTerminal(10, 2)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterAccessibility(accessibility.AccessibilityNode{ID: "save", Role: accessibility.RoleButton, Label: "Save", Bounds: cell.NewRect(0, 0, 4, 1)})
	})
	if err := testTerm.AssertAccessibilityContains(accessibility.Mode{ScreenReader: true}, "button#save"); err != nil {
		t.Fatal(err)
	}
	keys := []backend.KeyEvent{{Type: backend.KeyRune, Ch: 'a'}, {Type: backend.KeyEnter}}
	if got := testTerm.SendKeys(keys, func(backend.KeyEvent) bool { return true }); got != len(keys) {
		t.Fatalf("handled keys = %d, want %d", got, len(keys))
	}
	if err := testTerm.ValidateAccessibility(); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalImageRegistrationAssertion(t *testing.T) {
	testTerm := NewTerminal(10, 2)
	area := cell.NewRect(1, 0, 4, 2)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RenderWidget(imageRegistrationWidget{area: area, zIndex: 7}, area)
	})
	if err := testTerm.AssertImageRegistration(area, 7); err != nil {
		t.Fatal(err)
	}
}

type imageRegistrationWidget struct {
	area   cell.Rect
	zIndex int
}

func (w imageRegistrationWidget) Draw(ctx cell.Context, _ *buffer.Buffer) {
	if ctx.RegisterImage != nil {
		ctx.RegisterImage(w.area, nil, w.zIndex, false)
	}
}

func (imageRegistrationWidget) SizeHint(area cell.Rect) (uint16, uint16) {
	return area.Width, area.Height
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
	if err := testTerm.AssertLayerZIndex("modal", 10); err != nil {
		t.Fatal(err)
	}
	if err := testTerm.AssertModalIsolation("modal"); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalEventTraceAssertion(t *testing.T) {
	testTerm := NewTerminal(10, 2)
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventHandler(cell.NewRect(0, 0, 10, 2), terminal.CapturePhase, func(*terminal.EventContext) {})
		frame.RegisterEventHandler(cell.NewRect(1, 0, 3, 1), terminal.TargetPhase, func(*terminal.EventContext) {})
		frame.RegisterEventHandler(cell.NewRect(0, 0, 10, 2), terminal.BubblePhase, func(*terminal.EventContext) {})
	})
	testTerm.PropagateMouse(backend.MouseEvent{X: 2, Y: 0, Button: backend.MouseLeft})
	if err := testTerm.AssertEventTrace(":capture", ":target", ":bubble"); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalAdvancedErgonomics(t *testing.T) {
	testTerm := NewTerminal(10, 2)

	// 1. Test style snapshot assertion
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetString(0, 0, "A", cell.Style{Fg: cell.NewColorANSI(1), Bg: cell.NewColorANSI(4), Modifier: cell.ModifierBold})
	})
	styleSnapshot := testTerm.StyleSnapshot()
	if err := testTerm.AssertStyleSnapshot(styleSnapshot); err != nil {
		t.Fatal(err)
	}

	// 2. Test hover enter/leave assertions and EventTraceEntries
	testTerm.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventRegion(terminal.EventRegion{
			ID: "btn", Area: cell.NewRect(0, 0, 4, 1), Phase: terminal.TargetPhase,
			Handler: func(*terminal.EventContext) {},
			OnEnter: func(*terminal.EventContext) {},
			OnLeave: func(*terminal.EventContext) {},
		})
	})
	testTerm.MovePointer(1, 0) // triggers hover enter on btn
	if err := testTerm.AssertHoverEnter("btn"); err != nil {
		t.Fatal(err)
	}
	testTerm.MovePointer(8, 0) // triggers hover leave on btn
	if err := testTerm.AssertHoverLeave("btn"); err != nil {
		t.Fatal(err)
	}

	// Verify EventTraceEntries
	entries := testTerm.EventTraceEntries()
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 trace entries, got %d", len(entries))
	}
	if entries[0].RegionID != "btn" || entries[0].Action != "enter" {
		t.Errorf("first entry = %+v", entries[0])
	}
	if entries[1].RegionID != "btn" || entries[1].Action != "leave" {
		t.Errorf("second entry = %+v", entries[1])
	}

	// 3. Test ResizeAndRender and AssertRedrawChanged
	p := &widgets.Paragraph{Text: "Limoni"}
	snapBefore := testTerm.ResizeAndRender(p, 10, 2)
	p2 := &widgets.Paragraph{Text: "Changed"}
	snapAfter := testTerm.ResizeAndRender(p2, 10, 2)
	if err := testTerm.AssertRedrawChanged(snapBefore, snapAfter); err != nil {
		t.Fatal(err)
	}
}
