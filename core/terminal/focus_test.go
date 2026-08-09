package terminal

import "testing"

func TestFocusManagerScopeNavigation(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("outside")
	manager.BeginScope("modal")
	manager.Register("first")
	manager.Register("second")
	manager.SetFocused("first")
	manager.Next()
	if manager.Focused() != "second" {
		t.Fatalf("scope next = %q; want second", manager.Focused())
	}
	manager.Next()
	if manager.Focused() != "first" {
		t.Fatalf("scope wrap = %q; want first", manager.Focused())
	}
	manager.EndScope()
	manager.SetFocused("outside")
	manager.Next()
	if manager.Focused() != "first" {
		t.Fatalf("global navigation = %q; want first", manager.Focused())
	}
}

func TestFocusManagerNextExcluding(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("tab_Giriş")
	manager.Register("input")
	manager.Register("tab_Ayarlar")
	manager.Register("slider")
	manager.SetFocused("input")
	manager.NextExcluding("tab_")
	if manager.Focused() != "slider" {
		t.Fatalf("next non-tab focus = %q; want slider", manager.Focused())
	}
}

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
