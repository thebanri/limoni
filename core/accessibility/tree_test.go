package accessibility

import (
	"github.com/thebanri/limoni/core/cell"
	"testing"
)

func TestAccessibilityTreeAndModes(t *testing.T) {
	root := AccessibilityNode{ID: "root", Role: RoleDialog, Bounds: cell.NewRect(0, 0, 20, 5)}
	root.AddChild(AccessibilityNode{ID: "ok", Role: RoleButton, Label: "OK", State: StateFocused})
	if node := root.Find("ok"); node == nil || node.Label != "OK" || node.State != StateFocused {
		t.Fatalf("node lookup failed: %+v", node)
	}
	mode := (Mode{ScreenReader: true}).Normalize()
	if !mode.NoMouse {
		if (Mode{ReducedMotion: true}).ShouldAnimate() {
			t.Fatal("reduced motion should disable animation")
		}
		if got := (Mode{ASCIIOnly: true}).TextFallback("A✓"); got != "A?" {
			t.Fatalf("fallback = %q", got)
		}
		t.Fatal("screen reader mode should disable mouse")
	}
}
