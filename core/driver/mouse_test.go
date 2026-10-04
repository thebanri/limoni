package driver

import (
	"strings"
	"testing"
)

// Without the mouse, the setup sequence asks for no mouse reporting, and
// nothing else in it changes.
func TestSetupSequenceWithoutTheMouse(t *testing.T) {
	for _, inline := range []uint16{0, 3} {
		with, without := setupSequence(inline, true), setupSequence(inline, false)
		if !strings.Contains(with, "\x1b[?1003h") || strings.Contains(without, "\x1b[?1003h") || strings.Contains(without, "\x1b[?1006h") {
			t.Fatalf("inline %d: with %q, without %q", inline, with, without)
		}
		if strings.Replace(with, mouseOnSeq, "", 1) != without {
			t.Fatalf("inline %d: more than the mouse changed", inline)
		}
	}
}

// Before Setup, SetMouse decides what Setup sends; after it, the change is
// written at once, and only when it is a change.
func TestSetMouse(t *testing.T) {
	io := NewMemoryTerminalIO(nil, 20, 5)
	b := NewPortableBackend(io)
	if err := b.SetMouse(false); err != nil {
		t.Fatal(err)
	}
	if len(io.Output()) != 0 {
		t.Fatalf("SetMouse before Setup wrote %q", io.Output())
	}
	t.Setenv("LIMONI_PROBE", "0")
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	if out := string(io.Output()); strings.Contains(out, "\x1b[?1003h") {
		t.Fatalf("Setup asked for the mouse after SetMouse(false): %q", out)
	}
	before := len(io.Output())
	_ = b.SetMouse(false)
	if len(io.Output()) != before {
		t.Fatal("SetMouse wrote although nothing changed")
	}
	_ = b.SetMouse(true)
	if got := string(io.Output()[before:]); got != mouseOnSeq {
		t.Fatalf("SetMouse(true) after Setup wrote %q, want %q", got, mouseOnSeq)
	}
	if !b.MouseEnabled() {
		t.Fatal("MouseEnabled is false after SetMouse(true)")
	}
}
