package widgets_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// With a State, a dialog's buttons focus on hover, run their handler on a
// click, report the index to OnButtonHover, and a frame allocates nothing.
// Without one, the same clicks still work.
func TestDialogButtonsWithAndWithoutState(t *testing.T) {
	for _, withState := range []bool{true, false} {
		saved, hovered := 0, -1
		dialog := &widgets.Dialog{
			ID:      "quit",
			Title:   "Unsaved changes",
			Message: "You have unsaved changes in three files. Save them before quitting?",
			Buttons: []widgets.DialogButton{
				{Text: "Save", Handler: func() { saved++ }},
				{Text: "Discard"},
			},
			OnButtonHover: func(i int) { hovered = i },
		}
		if withState {
			dialog.State = &widgets.DialogState{}
		}
		term := testkit.NewTerminal(40, 9)
		draw := func(f *terminal.Frame) { f.RenderWidget(dialog, f.Area()) }
		term.Draw(draw)

		rows := strings.Split(term.Snapshot(), "\n")
		save := strings.Index(rows[7], "[ Save ]")
		if save < 0 {
			t.Fatalf("state %v: no Save button on the button row:\n%s", withState, term.Snapshot())
		}
		term.Click(uint16(save+2), 7)
		if saved != 1 || hovered != 0 || term.Focused() != "quit_btn_0" {
			t.Errorf("state %v: click on Save: saved %d, hovered %d, focused %q", withState, saved, hovered, term.Focused())
		}
		term.Draw(draw)
		discard := strings.Index(rows[7], "[ Discard ]")
		term.Click(uint16(discard+2), 7)
		if saved != 1 || hovered != 1 || term.Focused() != "quit_btn_1" {
			t.Errorf("state %v: click on Discard: saved %d, hovered %d, focused %q", withState, saved, hovered, term.Focused())
		}

		if withState {
			if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
				t.Errorf("a frame with a DialogState allocated %v times", n)
			}
		}
	}
}

// The message wraps at word boundaries, each line centred.
func TestDialogWrapsItsMessage(t *testing.T) {
	dialog := &widgets.Dialog{
		ID:      "d",
		Message: "one two three four five six seven",
		State:   &widgets.DialogState{},
	}
	term := testkit.NewTerminal(20, 6)
	term.Render(dialog, term.Area())
	rows := strings.Split(term.Snapshot(), "\n")
	want := []string{"one two three", "four five six", "seven"}
	for i, w := range want {
		if got := strings.Trim(rows[1+i], "│ "); got != w {
			t.Errorf("line %d = %q, want %q\n%s", i, got, w, term.Snapshot())
		}
	}
}
