package widgets

import (
	"runtime"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
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
		{"tabs", Tabs{ID: "t", Titles: []string{"a", "b", "c"}, Selected: 2}, accessibility.RoleTabList, "c", 3, 3},
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

// A secret field must not put its characters into the cell buffer, where the
// screen snapshot, a recording or a terminal scrollback could pick them up.
func TestSecretTextInputNeverDrawsOrExposesItsValue(t *testing.T) {
	state := NewTextInputState()
	state.Text = []rune("hunter2")
	state.Cursor = len(state.Text)
	input := TextInput{ID: "pw", State: state, Secret: true}

	buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 1))
	input.Draw(cell.NewContext(buf.Area, cell.Style{}), buf)
	snap := buf.Snapshot()
	for _, r := range "hunter2" {
		if strings.ContainsRune(snap, r) {
			t.Fatalf("secret character %q drawn into the buffer: %q", r, snap)
		}
	}
	if strings.Count(snap, "•") != 7 {
		t.Errorf("expected 7 mask glyphs, got %q", snap)
	}

	node := input.AccessibilityNode(buf.Area, true)
	if node.Value != "" {
		t.Errorf("secret value exposed on the node: %q", node.Value)
	}
	if node.State&accessibility.StateSensitive == 0 {
		t.Error("secret field not marked StateSensitive")
	}
	line := accessibility.Mode{ScreenReader: true}.LineMode([]accessibility.AccessibilityNode{node})
	if strings.Contains(line, "hunter2") || !strings.Contains(line, "sensitive") {
		t.Errorf("screen reader line = %q", line)
	}
}

// Drawing and describing a table, a tab bar and a tree, with their row, tab
// and item children, allocates nothing once the buffers have grown.
func TestStructuredNodesDoNotAllocate(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 6)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	table := &Table{ID: "tb", Rows: []TableRow{NewRow("r1", "a"), NewRow("r2", "b")}, State: NewTableState(),
		Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 10}, {Type: ConstraintFixed, Value: 10}}}
	tabs := &Tabs{ID: "t", Titles: []string{"a", "b"}, Selected: 1, State: &TabsState{}}
	tree := &TreeView{ID: "tr", Roots: []TreeNode{{ID: "x", Label: "x"}, {ID: "y", Label: "y"}}, State: NewTreeViewState()}
	run := func() {
		table.Draw(ctx, buf)
		_ = table.AccessibilityNode(area, false)
		tabs.Draw(ctx, buf)
		_ = tabs.AccessibilityNode(area, false)
		tree.Draw(ctx, buf)
		_ = tree.AccessibilityNode(area, false)
	}
	run()
	runtime.GC() // a collection must not make the next frame allocate
	if n := len(table.AccessibilityNode(area, false).Children); n != 2 {
		t.Fatalf("table exposes %d rows, want 2", n)
	}
	if got := testing.AllocsPerRun(50, func() { run(); runtime.GC() }); got != 0 {
		t.Errorf("drawing and describing allocated %v times per run, want 0", got)
	}
}
