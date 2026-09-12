package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
)

// The semantic tree is only as good as its coverage: a widget that does not
// implement Provider is silent to a screen reader, however well it renders.
// These are the widgets a keyboard user navigates, so a regression here is a
// regression in the accessibility story, not a missing nicety.
func TestInteractiveWidgetsProvideAccessibilityNodes(t *testing.T) {
	bounds := cell.NewRect(0, 0, 20, 5)
	checked := true

	for _, tc := range []struct {
		name      string
		widget    any
		wantRole  accessibility.Role
		wantValue string
		wantPos   int
		wantSize  int
	}{
		{"list", List{ID: "l", Items: []string{"alpha", "beta", "gamma"}, State: &ListState{Selected: 1}}, accessibility.RoleList, "beta", 2, 3},
		{"select", Select{ID: "s", Options: []string{"one", "two"}, State: &SelectState{Selected: 1}}, accessibility.RoleList, "two", 2, 2},
		{"tabs", Tabs{ID: "t", Titles: []string{"a", "b", "c"}, Selected: 2}, accessibility.RoleList, "c", 3, 3},
		{"table", Table{ID: "tb", Rows: []TableRow{NewRow("r1"), NewRow("r2")}, State: &TableState{Selected: 0}}, accessibility.RoleTable, "r1", 1, 2},
		{"dialog", Dialog{ID: "d", Title: "Confirm", Message: "Are you sure?"}, accessibility.RoleDialog, "Are you sure?", 0, 0},
		{"textarea", TextArea{ID: "ta", State: &TextAreaState{Text: []rune("hello")}}, accessibility.RoleInput, "hello", 0, 0},
		{"paragraph", &Paragraph{ID: "p", Text: "body"}, accessibility.RoleGeneric, "body", 0, 0},
		{"checkbox", Checkbox{ID: "c", Label: "Agree", Checked: &checked}, accessibility.RoleCheckbox, "true", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, ok := tc.widget.(accessibility.Provider)
			if !ok {
				t.Fatalf("%T does not implement accessibility.Provider", tc.widget)
			}
			node := provider.AccessibilityNode(bounds, false)
			if node.Role != tc.wantRole {
				t.Errorf("Role = %v, want %v", node.Role, tc.wantRole)
			}
			if node.Value != tc.wantValue {
				t.Errorf("Value = %q, want %q", node.Value, tc.wantValue)
			}
			if node.Position != tc.wantPos || node.SetSize != tc.wantSize {
				t.Errorf("Position/SetSize = %d/%d, want %d/%d", node.Position, node.SetSize, tc.wantPos, tc.wantSize)
			}
			if node.Bounds != bounds {
				t.Errorf("Bounds = %v, want %v", node.Bounds, bounds)
			}
		})
	}
}

func TestAccessibilityNodeReportsFocus(t *testing.T) {
	node := List{ID: "l", Items: []string{"a"}, State: &ListState{Selected: 0}}.
		AccessibilityNode(cell.NewRect(0, 0, 5, 1), true)
	if node.State&accessibility.StateFocused == 0 {
		t.Error("focused list did not report StateFocused")
	}
	if node.State&accessibility.StateSelected == 0 {
		t.Error("list with a selection did not report StateSelected")
	}
}

func TestLineModeRendersPosition(t *testing.T) {
	node := Tabs{ID: "nav", Titles: []string{"one", "two", "three"}, Selected: 1}.
		AccessibilityNode(cell.NewRect(0, 0, 10, 1), false)
	line := accessibility.Mode{ScreenReader: true}.LineMode([]accessibility.AccessibilityNode{node})
	if !strings.Contains(line, "position=2/3") {
		t.Errorf("line mode did not render position: %s", line)
	}
}

// Nodes are rebuilt on every frame, so constructing one must not allocate.
func TestAccessibilityNodeConstructionDoesNotAllocate(t *testing.T) {
	bounds := cell.NewRect(0, 0, 20, 5)
	list := List{ID: "l", Items: []string{"alpha", "beta", "gamma"}, State: &ListState{Selected: 1}}
	table := Table{ID: "tb", Rows: []TableRow{NewRow("r1"), NewRow("r2")}, State: &TableState{Selected: 0}}
	tabs := Tabs{ID: "t", Titles: []string{"a", "b"}, Selected: 1}

	if got := testing.AllocsPerRun(100, func() {
		_ = list.AccessibilityNode(bounds, false)
		_ = table.AccessibilityNode(bounds, false)
		_ = tabs.AccessibilityNode(bounds, false)
	}); got != 0 {
		t.Errorf("building nodes allocated %v times per run, want 0", got)
	}
}
