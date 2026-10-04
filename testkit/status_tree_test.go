package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

// Key hints drawn as a StatusBar with an ID are in the tree an agent reads;
// they used to be plain cells, invisible to it.
func TestStatusBarReachesTheTree(t *testing.T) {
	term := NewTerminal(40, 2)
	term.Render(widgets.StatusBar{ID: "keys", Left: []widgets.StatusItem{{Key: "q", Text: "quit"}}}, cell.NewRect(0, 1, 40, 1))
	for _, n := range term.AccessibilityTree() {
		if n.ID == "keys" {
			if n.Value != "q quit" {
				t.Fatalf("value %q", n.Value)
			}
			return
		}
	}
	t.Fatalf("no node for the bar: %+v", term.AccessibilityTree())
}
