package testkit

import (
	"testing"

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
