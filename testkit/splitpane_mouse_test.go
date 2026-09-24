package testkit

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

// The divider is dragged through the real frame routing: the press lands on
// the divider's mouse region, the handler captures the mouse, and the drag
// reaches the capture wherever the pointer is.
func TestSplitPaneDividerDragsThroughTheFrame(t *testing.T) {
	state := &widgets.SplitState{}
	pane := widgets.SplitPane{
		First:  &widgets.Paragraph{Text: "left"},
		Second: &widgets.Paragraph{Text: "right"},
		State:  state,
	}
	term := NewTerminal(41, 5)
	area := cell.NewRect(0, 0, 41, 5)

	term.Render(pane, area)
	if !term.Click(20, 2) {
		t.Fatal("pressing the divider at x=20 was not routed")
	}
	if !term.Drag(10, 4) {
		t.Fatal("the drag was not routed to the capture")
	}
	if want := 0.25; state.Ratio != want {
		t.Fatalf("ratio after dragging to x=10 of 41: %v, want %v", state.Ratio, want)
	}
	term.Render(pane, area)
	firstLine := strings.SplitN(term.Snapshot(), "\n", 2)[0]
	if got := []rune(firstLine); len(got) < 11 || got[10] != '│' {
		t.Errorf("divider not redrawn at x=10:\n%s", firstLine)
	}
}
