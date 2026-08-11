package terminal

import (
	"bytes"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"testing"
)

func TestFrameAccessibilityRegistration(t *testing.T) {
	f := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), NewFocusManager())
	f.RegisterAccessibility(accessibility.AccessibilityNode{ID: "submit", Role: accessibility.RoleButton, Label: "Submit"})
	tree := f.AccessibilityTree()
	if len(tree) != 1 || tree[0].ID != "submit" {
		t.Fatalf("tree = %+v", tree)
	}
}

func TestFrameValidatesAccessibilityTree(t *testing.T) {
	f := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), NewFocusManager())
	f.RegisterAccessibility(accessibility.AccessibilityNode{ID: "save", Role: accessibility.RoleButton})
	if err := f.ValidateAccessibility(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestFrameAccessibilityLineMode(t *testing.T) {
	f := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), NewFocusManager())
	f.RegisterAccessibility(accessibility.AccessibilityNode{ID: "ok", Role: accessibility.RoleButton, Label: "OK", Bounds: cell.NewRect(1, 0, 4, 1)})
	want := "button#ok \"OK\" bounds=1,0 4x1"
	if got := f.AccessibilityLineMode(accessibility.Mode{ScreenReader: true}); got != want {
		t.Fatalf("line mode = %q, want %q", got, want)
	}
}

func TestFrameWritesAccessibilityLineMode(t *testing.T) {
	f := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), NewFocusManager())
	f.RegisterAccessibility(accessibility.AccessibilityNode{ID: "ok", Role: accessibility.RoleButton, Label: "OK", Bounds: cell.NewRect(0, 0, 2, 1)})
	var output bytes.Buffer
	if err := f.WriteAccessibilityLineMode(&output, accessibility.Mode{ScreenReader: true}); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("expected accessibility writer output")
	}
}
