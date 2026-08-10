package terminal

import (
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
