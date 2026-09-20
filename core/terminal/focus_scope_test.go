package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// Focus scopes are what keeps Tab inside a dialog: while a scope is open,
// only the widgets registered inside it take part in navigation.
func TestFocusScopesKeepNavigationInside(t *testing.T) {
	fm := NewFocusManager()
	fm.Register("page.name")
	fm.BeginScope("dialog")
	if got := fm.ActiveScope(); got != "dialog" {
		t.Errorf("ActiveScope = %q", got)
	}
	fm.Register("dialog.ok")
	fm.Register("dialog.cancel")

	fm.BeginScope("nested")
	if scopes := fm.ActiveScopes(); len(scopes) != 2 || scopes[0] != "dialog" || scopes[1] != "nested" {
		t.Errorf("ActiveScopes = %v, want the whole stack in order", scopes)
	}
	// The returned stack is a copy: writing to it must not move the focus.
	if scopes := fm.ActiveScopes(); len(scopes) > 0 {
		scopes[0] = "tampered"
		if fm.ActiveScopes()[0] != "dialog" {
			t.Error("ActiveScopes handed out its own slice")
		}
	}
	fm.EndScope()
	if got := fm.ActiveScope(); got != "dialog" {
		t.Errorf("after EndScope the active scope is %q, want dialog", got)
	}

	ids := fm.FocusableIDs()
	if len(ids) != 3 {
		t.Fatalf("FocusableIDs = %v, want all three registrations", ids)
	}
	ids[0] = "tampered"
	if fm.FocusableIDs()[0] == "tampered" {
		t.Error("FocusableIDs handed out its own slice")
	}

	fm.EndScope()
	if got := fm.ActiveScope(); got != "" {
		t.Errorf("with every scope closed the active scope is %q", got)
	}
	// One EndScope too many is not a panic and not a negative stack.
	fm.EndScope()
	if got := fm.ActiveScope(); got != "" {
		t.Errorf("unbalanced EndScope left %q", got)
	}

	var nilManager *FocusManager
	if ids := nilManager.FocusableIDs(); ids != nil {
		t.Errorf("a nil manager returned %v", ids)
	}
}

func TestFocusMoves(t *testing.T) {
	fm := NewFocusManager()
	for _, id := range []string{"a", "b", "c"} {
		fm.Register(id)
	}

	fm.Next()
	first := fm.Focused()
	if first == "" {
		t.Fatal("Next from nothing focused nothing")
	}
	fm.Next()
	if second := fm.Focused(); second == first {
		t.Errorf("Next did not move: still %q", second)
	}
	fm.Prev()
	if back := fm.Focused(); back != first {
		t.Errorf("Prev went to %q, want back to %q", back, first)
	}

	fm.SetFocused("c")
	if !fm.IsFocused("c") || fm.IsFocused("a") || fm.IsFocused("") {
		t.Error("SetFocused/IsFocused disagree")
	}

	// Navigation wraps rather than stopping at the end.
	fm.Next()
	if fm.Focused() != "a" {
		t.Errorf("Next past the last widget went to %q, want wrap to a", fm.Focused())
	}

	// NextExcluding skips a whole family of ids, which is how a modal keeps
	// focus off the page behind it.
	fm2 := NewFocusManager()
	for _, id := range []string{"page.one", "page.two", "modal.ok"} {
		fm2.Register(id)
	}
	fm2.SetFocused("modal.ok")
	fm2.NextExcluding("page.")
	if got := fm2.Focused(); got != "modal.ok" {
		t.Errorf("NextExcluding landed on %q, want to stay inside the modal", got)
	}
}

func TestFocusBoundsAndDirectionalMovement(t *testing.T) {
	fm := NewFocusManager()
	fm.Register("left")
	fm.Register("right")
	fm.RegisterBounds("left", cell.NewRect(0, 0, 10, 3))
	fm.RegisterBounds("right", cell.NewRect(20, 0, 10, 3))

	fm.SetFocused("left")
	if !fm.MoveFocus2D(DirRight) {
		t.Fatal("MoveFocus2D(right) reported nothing to move to")
	}
	if got := fm.Focused(); got != "right" {
		t.Errorf("moved to %q, want right", got)
	}
	if fm.MoveFocus2D(DirRight) {
		t.Error("moving right from the rightmost widget should report nothing to do")
	}
	if !fm.MoveFocus2D(DirLeft) || fm.Focused() != "left" {
		t.Errorf("moving back left landed on %q", fm.Focused())
	}
}
