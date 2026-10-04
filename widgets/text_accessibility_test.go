package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// A status bar with an ID reads as its key hints; one without costs nothing.
func TestStatusBarInTheSemanticTree(t *testing.T) {
	bar := StatusBar{
		ID:    "keys",
		Left:  []StatusItem{{Key: "^Q", Text: "quit"}, {Key: "/", Text: "filter"}},
		Right: []StatusItem{{Text: "12/114"}},
	}
	bounds := cell.NewRect(0, 0, 40, 1)
	if v := bar.AccessibilityNode(bounds, false).Value; v != "^Q quit  / filter  12/114" {
		t.Fatalf("value %q", v)
	}
	if n := testing.AllocsPerRun(100, func() { _ = bar.AccessibilityNode(bounds, false) }); n != 1 {
		t.Errorf("with an ID: %.0f allocations, want the 1 the doc comment says", n)
	}
	bar.ID = ""
	if n := testing.AllocsPerRun(100, func() { _ = bar.AccessibilityNode(bounds, false) }); n != 0 {
		t.Errorf("without an ID: %.0f allocations, want 0", n)
	}
}

// Rich text with an ID reads as its spans, line by line.
func TestRichTextInTheSemanticTree(t *testing.T) {
	text := Text{ID: "msg", Lines: []Line{
		NewLine(NewSpan("Saved ", cell.Style{}), NewSpan("3 files", cell.Style{Modifier: cell.ModifierBold})),
		NewLine(NewSpan("in 2s", cell.Style{})),
	}}
	bounds := cell.NewRect(0, 0, 40, 2)
	if v := text.AccessibilityNode(bounds, false).Value; v != "Saved 3 files\nin 2s" {
		t.Fatalf("value %q", v)
	}
	text.ID = ""
	if n := testing.AllocsPerRun(100, func() { _ = text.AccessibilityNode(bounds, false) }); n != 0 {
		t.Errorf("without an ID: %.0f allocations, want 0", n)
	}
}

func TestRichTextValueCostsOneAllocation(t *testing.T) {
	text := Text{ID: "msg", Lines: []Line{NewLine(NewSpan("a", cell.Style{}), NewSpan("b", cell.Style{})), NewLine(NewSpan("c", cell.Style{}))}}
	bounds := cell.NewRect(0, 0, 10, 2)
	if n := testing.AllocsPerRun(100, func() { _ = text.AccessibilityNode(bounds, false) }); n != 1 {
		t.Errorf("with an ID: %.0f allocations, want the 1 the doc comment says", n)
	}
}
