package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
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
