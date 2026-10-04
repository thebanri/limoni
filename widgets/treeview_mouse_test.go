package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// Through a real frame: a click selects the row under it and opens or closes
// a parent, the wheel scrolls, and a frame allocates nothing.
func TestTreeViewMouseThroughAFrame(t *testing.T) {
	state := widgets.NewTreeViewState()
	tree := &widgets.TreeView{ID: "files", State: state, Roots: []widgets.TreeNode{
		{ID: "src", Label: "src", Expanded: true, Children: []widgets.TreeNode{
			{ID: "main.go", Label: "main.go"}, {ID: "util.go", Label: "util.go"},
		}},
		{ID: "docs", Label: "docs", Children: []widgets.TreeNode{{ID: "a.md", Label: "a.md"}}},
		{ID: "go.mod", Label: "go.mod"},
	}}
	term := testkit.NewTerminal(20, 3)
	draw := func(f *terminal.Frame) { f.RenderWidget(tree, f.Area()) }
	term.Draw(draw)

	term.Click(4, 1) // main.go
	if state.SelectedID != "main.go" || term.Focused() != "files" {
		t.Fatalf("click on row 1: selected %q, focused %q", state.SelectedID, term.Focused())
	}
	term.Draw(draw)
	term.Click(1, 0) // src: closes it
	term.Draw(draw)
	if state.SelectedID != "src" || state.IsExpanded("src", true) {
		t.Errorf("click on src: selected %q, still open %v", state.SelectedID, state.IsExpanded("src", true))
	}
	term.Click(1, 1) // docs, now on row 1: opens it
	term.Draw(draw)
	if state.SelectedID != "docs" || !state.IsExpanded("docs", false) {
		t.Errorf("click on docs: selected %q, open %v", state.SelectedID, state.IsExpanded("docs", false))
	}

	term.Mouse(driver.MouseEvent{Button: driver.MouseScrollDown, X: 1, Y: 1})
	if state.Offset != 1 {
		t.Errorf("wheel down: offset %d, want 1", state.Offset)
	}

	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}
