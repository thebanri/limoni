package widgets_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// Through a real frame: the button opens the menu, hovering an item selects
// it, clicking one closes the menu and runs its handler, and an open menu
// draws without allocating.
func TestPopupMouseThroughAFrame(t *testing.T) {
	state := widgets.NewPopupState()
	ran := ""
	popup := &widgets.Popup{ID: "file", Label: "File", State: state, Items: []widgets.PopupItem{
		{Text: "Open", Handler: func() { ran = "open" }},
		{Text: "Save", Handler: func() { ran = "save" }},
	}}
	term := testkit.NewTerminal(24, 8)
	draw := func(f *terminal.Frame) { f.RenderWidget(popup, f.Area()) }
	term.Draw(draw)

	term.Click(1, 0)
	if !state.IsOpen || term.Focused() != "file" {
		t.Fatalf("the button did not open the menu: open %v, focused %q", state.IsOpen, term.Focused())
	}
	term.Draw(draw)
	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("an open menu allocated %v times a frame", n)
	}
	rows := strings.Split(term.Snapshot(), "\n")
	saveY := -1
	for y, row := range rows {
		if strings.Contains(row, "Save") {
			saveY = y
		}
	}
	if saveY < 0 {
		t.Fatalf("no Save item on screen:\n%s", term.Snapshot())
	}
	saveX := uint16(strings.Index(rows[saveY], "Save"))
	term.Mouse(driver.MouseEvent{Button: driver.MouseNone, X: saveX, Y: uint16(saveY)})
	if state.Selected != 1 {
		t.Errorf("hovering Save selected %d", state.Selected)
	}
	term.Click(saveX, uint16(saveY))
	if ran != "save" || state.IsOpen {
		t.Errorf("clicking Save: ran %q, still open %v", ran, state.IsOpen)
	}
}
