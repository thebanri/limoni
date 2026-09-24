package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// toggle is a custom widget as an application would write one: it embeds
// widgets.Accessible for its semantics and never calls RegisterFocus.
type toggle struct {
	widgets.Accessible
	label string
}

func (t toggle) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, t.label, cell.Style{})
}

func (t toggle) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// Tab reaches custom widgets built on widgets.Accessible, in the order they
// are drawn, and skips the ones whose role is not interactive or that are
// disabled.
func TestTabReachesAccessibleWidgets(t *testing.T) {
	term := NewTerminal(30, 5)
	draw := func() {
		term.Draw(func(f *terminal.Frame) {
			f.RenderWidget(toggle{widgets.Accessible{ID: "wifi", Role: accessibility.RoleCheckbox, Label: "Wi-Fi"}, "Wi-Fi"}, cell.NewRect(0, 0, 30, 1))
			f.RenderWidget(toggle{widgets.Accessible{ID: "note", Role: accessibility.RoleGeneric, Label: "note"}, "a note"}, cell.NewRect(0, 1, 30, 1))
			f.RenderWidget(toggle{widgets.Accessible{ID: "off", Role: accessibility.RoleButton, State: accessibility.StateDisabled}, "Off"}, cell.NewRect(0, 2, 30, 1))
			f.RenderWidget(toggle{widgets.Accessible{ID: "save", Role: accessibility.RoleButton, Label: "Save"}, "Save"}, cell.NewRect(0, 3, 30, 1))
		})
	}
	draw()
	ids := term.FocusableIDs()
	if len(ids) != 2 || ids[0] != "wifi" || ids[1] != "save" {
		t.Fatalf("focusable %v, want [wifi save]", ids)
	}
	if err := term.AssertFocused("wifi"); err != nil {
		t.Error(err)
	}
	term.Frame().FocusManager.Next()
	draw()
	if err := term.AssertFocused("save"); err != nil {
		t.Error(err)
	}
	// The focused node says so in the semantic tree, as a screen reader and
	// an agent read it.
	for _, n := range term.AccessibilityTree() {
		if n.ID == "save" && n.State&accessibility.StateFocused == 0 {
			t.Error("the focused widget's node is not marked focused")
		}
	}
}
