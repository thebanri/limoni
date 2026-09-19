package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func TestTreeView_FlattenAndNavigation(t *testing.T) {
	roots := []TreeNode{
		{
			ID:       "src",
			Label:    "src",
			Expanded: true,
			Children: []TreeNode{
				{
					ID:    "main.go",
					Label: "main.go",
				},
				{
					ID:       "pkg",
					Label:    "pkg",
					Expanded: false,
					Children: []TreeNode{
						{ID: "util.go", Label: "util.go"},
					},
				},
			},
		},
		{
			ID:    "README.md",
			Label: "README.md",
		},
	}

	state := NewTreeViewState()
	flat := state.Flatten(roots)
	if len(flat) != 4 {
		t.Fatalf("expected 4 flat items with pkg collapsed, got %d", len(flat))
	}

	state.Expand("pkg")
	flat = state.Flatten(roots)
	if len(flat) != 5 {
		t.Fatalf("expected 5 flat items with pkg expanded, got %d", len(flat))
	}

	state.Collapse("src")
	flat = state.Flatten(roots)
	if len(flat) != 2 {
		t.Fatalf("expected 2 flat items with src collapsed, got %d", len(flat))
	}

	// Keyboard Navigation
	state.Expand("src")
	state.Select("src")
	state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowDown}, roots)
	if state.SelectedID != "main.go" {
		t.Errorf("expected selectedID main.go, got %s", state.SelectedID)
	}

	state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowUp}, roots)
	if state.SelectedID != "src" {
		t.Errorf("expected selectedID src, got %s", state.SelectedID)
	}
}

func TestTreeView_Draw(t *testing.T) {
	roots := []TreeNode{
		{
			ID:       "root",
			Label:    "Project Root",
			Icon:     "📁",
			Expanded: true,
			Children: []TreeNode{
				{ID: "file1", Label: "config.json", Icon: "⚙️"},
			},
		},
	}

	state := NewTreeViewState()
	state.Select("file1")

	tree := TreeView{
		ID:         "test-tree",
		Roots:      roots,
		State:      state,
		ShowGuides: true,
	}

	area := cell.NewRect(0, 0, 40, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	tree.Draw(ctx, buf)

	// Check that first row has Project Root
	cell0 := buf.Get(0, 0)
	if cell0 == nil {
		t.Fatal("expected buffer cell at (0, 0)")
	}

	w, h := tree.SizeHint(area)
	if w != 40 || h != 2 {
		t.Errorf("expected SizeHint (40, 2), got (%d, %d)", w, h)
	}
}

// Guide lines continue through the columns of every ancestor that has a
// sibling below it. Flatten used to build each node's ancestry with
// append(parentEnd, isLast), whose siblings shared one backing array, so a
// later sibling overwrote earlier ones: here the lines for a1 and a1x
// vanished from their descendants' rows.
func TestTreeGuidesFollowEveryAncestor(t *testing.T) {
	leaf := func(id string) TreeNode { return TreeNode{ID: id, Label: id} }
	roots := []TreeNode{
		{ID: "a", Label: "a", Expanded: true, Children: []TreeNode{
			{ID: "a1", Label: "a1", Expanded: true, Children: []TreeNode{
				{ID: "a1x", Label: "a1x", Expanded: true, Children: []TreeNode{leaf("p"), leaf("q")}},
				{ID: "a1y", Label: "a1y", Expanded: true, Children: []TreeNode{leaf("r")}},
			}},
			leaf("a2"),
		}},
		{ID: "b", Label: "b", Expanded: true, Children: []TreeNode{leaf("b1"), leaf("b2")}},
	}
	area := cell.NewRect(0, 0, 14, 11)
	buf := buffer.NewBuffer(area)
	TreeView{ID: "t", Roots: roots, State: NewTreeViewState(), ShowGuides: true}.Draw(cell.NewContext(area, cell.Style{}), buf)

	want := []string{
		"▼ a",
		"│ ▼ a1",
		"│ │ ▼ a1x",
		"│ │ │ ├─ p",
		"│ │ │ └─ q",
		"│ │ ▼ a1y",
		"│ │   └─ r",
		"│ └─ a2",
		"▼ b",
		"  ├─ b1",
		"  └─ b2",
	}
	got := strings.Split(buf.Snapshot(), "\n")
	for i, line := range want {
		if i >= len(got) || strings.TrimRight(got[i], " ") != line {
			t.Fatalf("row %d:\n got %q\nwant %q\nscreen:\n%s", i, strings.TrimRight(got[i], " "), line, buf.Snapshot())
		}
	}
}
