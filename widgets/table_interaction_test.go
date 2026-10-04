package widgets_test

import (
	"testing"

	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// Header clicks sort, row clicks select, and a frame with both registered
// costs nothing: the handlers live in TableState and are built once.
func TestTableClicksThroughAFrameWithoutAllocating(t *testing.T) {
	state := widgets.NewTableState()
	table := widgets.NewTable().WithID("people").WithHeaders("Name", "Age").
		WithRow("Cem", "41").WithRow("Ada", "36").WithRow("Bora", "29").WithState(state)
	table.SortEnabled = true
	term := testkit.NewTerminal(30, 6)
	draw := func(f *terminal.Frame) { f.RenderWidget(table, f.Area()) }
	term.Draw(draw)

	term.Click(1, 0) // the "Name" header
	term.Draw(draw)
	if state.SortColumn != 0 || state.SortDescending {
		t.Fatalf("after a header click: sort column %d, descending %v", state.SortColumn, state.SortDescending)
	}
	if got := table.Rows[0].Cells[0].Text; got != "Ada" {
		t.Errorf("first row after sorting by name = %q, want Ada", got)
	}
	term.Click(1, 0)
	term.Draw(draw)
	if !state.SortDescending || table.Rows[0].Cells[0].Text != "Cem" {
		t.Errorf("a second click did not reverse the order: first row %q", table.Rows[0].Cells[0].Text)
	}

	term.Click(1, 3) // the third data row on screen (header, rule, then rows)
	if state.Selected != 1 {
		t.Errorf("clicking the second data row selected %d", state.Selected)
	}
	if term.Focused() != "people" {
		t.Errorf("a row click focused %q", term.Focused())
	}

	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}
