package terminal

import "testing"

func TestFocusManagerIsFocused(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("input")
	if !manager.IsFocused("input") {
		t.Fatal("registered input should be focused")
	}
	if manager.IsFocused("other") {
		t.Fatal("other widget should not be focused")
	}
	manager.SetFocused("other")
	if manager.IsFocused("input") || !manager.IsFocused("other") {
		t.Fatal("focus state did not update")
	}
}
