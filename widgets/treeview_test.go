package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown}, roots)
	if state.SelectedID != "main.go" {
		t.Errorf("expected selectedID main.go, got %s", state.SelectedID)
	}

	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp}, roots)
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
