package accessibility

import (
	"bytes"
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

func TestLineModeSerializesTreeDeterministically(t *testing.T) {
	root := AccessibilityNode{
		ID: "dialog", Role: RoleDialog, Label: "Ayarlar", Bounds: cell.NewRect(1, 2, 20, 5),
		State: StateExpanded,
		Children: []AccessibilityNode{{
			ID: "save", Role: RoleButton, Label: "Kaydet", Value: "hazır",
			Description: "Değişiklikleri uygular", State: StateFocused | StateSelected,
			Bounds: cell.NewRect(3, 4, 8, 1),
		}},
	}
	want := "dialog#dialog \"Ayarlar\" state=expanded bounds=1,2 20x5\n  button#save \"Kaydet\" value=\"hazır\" description=\"Değişiklikleri uygular\" state=focused,selected bounds=3,4 8x1"
	if got := (Mode{}).LineMode([]AccessibilityNode{root}); got != want {
		t.Fatalf("line mode = %q, want %q", got, want)
	}
}

func TestLineModeHonorsASCIIFallback(t *testing.T) {
	node := AccessibilityNode{ID: "ok✓", Role: RoleButton, Label: "Kaydet ✓", Bounds: cell.NewRect(0, 0, 2, 1)}
	got := (Mode{ASCIIOnly: true}).LineMode([]AccessibilityNode{node})
	want := "button#ok? \"Kaydet ?\" bounds=0,0 2x1"
	if got != want {
		t.Fatalf("ASCII line mode = %q, want %q", got, want)
	}
}

func TestWriteLineMode(t *testing.T) {
	var output bytes.Buffer
	node := AccessibilityNode{ID: "ok", Role: RoleButton, Label: "OK", Bounds: cell.NewRect(0, 0, 2, 1)}
	if err := (Mode{}).WriteLineMode(&output, []AccessibilityNode{node}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "button#ok \"OK\" bounds=0,0 2x1\n" {
		t.Fatalf("line-mode output = %q", output.String())
	}
}

func TestValidateTree(t *testing.T) {
	if err := ValidateTree([]AccessibilityNode{{ID: "save", Role: RoleButton}}); err == nil {
		t.Fatal("expected unlabeled interactive node error")
	}
	if err := ValidateTree([]AccessibilityNode{{ID: "same"}, {ID: "same"}}); err == nil {
		t.Fatal("expected duplicate ID error")
	}
	if err := ValidateTree([]AccessibilityNode{{ID: "save", Role: RoleButton, Label: "Save"}}); err != nil {
		t.Fatal(err)
	}
}

func TestLineModeAdapter(t *testing.T) {
	var output bytes.Buffer
	nodes := []AccessibilityNode{{ID: "ok", Role: RoleButton, Label: "OK", Bounds: cell.NewRect(0, 0, 2, 1)}}
	if err := (LineModeAdapter{}).WriteTree(&output, Mode{}, nodes); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("expected adapter output")
	}
}
