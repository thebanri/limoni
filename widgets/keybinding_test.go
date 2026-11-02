package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/driver"
)

func TestKeybindingManager_New(t *testing.T) {
	km := NewKeybindingManager()
	if km == nil {
		t.Fatal("NewKeybindingManager should not return nil")
	}
	if len(km.AllBindings()) != 0 {
		t.Fatalf("new manager has %d bindings; want 0", len(km.AllBindings()))
	}
}

func TestKeybindingManager_Register(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'p', Ctrl: true, Label: "Palet", Category: "Genel"})
	km.Register(Keybinding{Key: driver.KeyEsc, Label: "Kapat", Category: "Genel"})

	if len(km.AllBindings()) != 2 {
		t.Fatalf("len(AllBindings) = %d; want 2", len(km.AllBindings()))
	}
}

func TestKeybindingManager_Handle_Rune(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key: driver.KeyRune, Ch: 'p', Ctrl: true,
		Handler: func() { ran = true },
	})

	// Matching key
	if !km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 'p', Ctrl: true}) {
		t.Fatal("a matching binding should return true")
	}
	if !ran {
		t.Fatal("handler should run")
	}

	// Wrong character
	ran = false
	if km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 'x', Ctrl: true}) {
		t.Fatal("a non-matching rune should return false")
	}
	if ran {
		t.Fatal("a non-matching rune should not run the handler")
	}

	// Ctrl eksik
	if km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 'p'}) {
		t.Fatal("the key without Ctrl should not match")
	}
}

func TestKeybindingManager_Handle_SpecialKey(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key:     driver.KeyEsc,
		Handler: func() { ran = true },
	})

	if !km.Handle(driver.KeyEvent{Type: driver.KeyEsc}) {
		t.Fatal("Esc should match")
	}
	if !ran {
		t.Fatal("handler should run")
	}

	// A different key must not match
	if km.Handle(driver.KeyEvent{Type: driver.KeyEnter}) {
		t.Fatal("Enter should not match the Esc binding")
	}
}

func TestKeybindingManager_Handle_Shift(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key: driver.KeyTab, Shift: true,
		Handler: func() { ran = true },
	})

	if !km.Handle(driver.KeyEvent{Type: driver.KeyTab, Shift: true}) {
		t.Fatal("Shift+Tab should match")
	}
	if !ran {
		t.Fatal("handler should run")
	}

	// Tab without Shift must not match
	ran = false
	if km.Handle(driver.KeyEvent{Type: driver.KeyTab}) {
		t.Fatal("Tab without Shift should not match")
	}
	if ran {
		t.Fatal("Tab without Shift should not run the handler")
	}
}

func TestKeybindingManager_Handle_FirstMatchWins(t *testing.T) {
	km := NewKeybindingManager()
	firstRan, secondRan := false, false
	km.Register(Keybinding{
		Key: driver.KeyRune, Ch: 'a',
		Handler: func() { firstRan = true },
	})
	km.Register(Keybinding{
		Key: driver.KeyRune, Ch: 'a',
		Handler: func() { secondRan = true },
	})

	km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 'a'})
	if !firstRan {
		t.Fatal("the first registered handler should run")
	}
	if secondRan {
		t.Fatal("the first match wins, so the second handler should not run")
	}
}

func TestKeybindingManager_Handle_NoMatch(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'q', Ctrl: true})

	if km.Handle(driver.KeyEvent{Type: driver.KeyArrowUp}) {
		t.Fatal("an unregistered key should return false")
	}
}

func TestKeybindingManager_ToCommandItems(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{
		Key: driver.KeyRune, Ch: 'p', Ctrl: true,
		Label: "Komut Paletini Aç/Kapa", Category: "Genel",
		Handler: func() {},
	})
	// Shortcuts without a Label must not become CommandItems
	km.Register(Keybinding{Key: driver.KeyEsc, Handler: func() {}})

	items := km.ToCommandItems()
	if len(items) != 1 {
		t.Fatalf("len(ToCommandItems) = %d; want 1", len(items))
	}
	if items[0].Label != "Komut Paletini Aç/Kapa" {
		t.Fatalf("Label = %q; want 'Komut Paletini Aç/Kapa'", items[0].Label)
	}
	if items[0].Category != "Genel" {
		t.Fatalf("Category = %q; 'Genel' bekleniyordu", items[0].Category)
	}
	if items[0].Detail != "Ctrl+P" {
		t.Fatalf("Detail = %q; 'Ctrl+P' bekleniyordu", items[0].Detail)
	}
	if items[0].Handler == nil {
		t.Fatal("Handler should not be nil")
	}
}

func TestFormatKeybinding(t *testing.T) {
	cases := []struct {
		name string
		kb   Keybinding
		want string
	}{
		{"rune", Keybinding{Key: driver.KeyRune, Ch: 'p', Ctrl: true}, "Ctrl+P"},
		{"rune shift", Keybinding{Key: driver.KeyRune, Ch: 'n', Ctrl: true, Shift: true}, "Ctrl+Shift+N"},
		{"tab", Keybinding{Key: driver.KeyTab}, "Tab"},
		{"esc", Keybinding{Key: driver.KeyEsc}, "Esc"},
		{"enter", Keybinding{Key: driver.KeyEnter}, "Enter"},
		{"space", Keybinding{Key: driver.KeySpace}, "Space"},
		{"backspace", Keybinding{Key: driver.KeyBackspace}, "Backspace"},
		{"arrow up", Keybinding{Key: driver.KeyArrowUp}, "↑"},
		{"arrow down", Keybinding{Key: driver.KeyArrowDown}, "↓"},
		{"arrow left", Keybinding{Key: driver.KeyArrowLeft}, "←"},
		{"arrow right", Keybinding{Key: driver.KeyArrowRight}, "→"},
		{"unknown", Keybinding{Key: driver.KeyF1}, "?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatKeybinding(tc.kb)
			if got != tc.want {
				t.Fatalf("formatKeybinding = %q; %q bekleniyordu", got, tc.want)
			}
		})
	}
}

// Alt is matched like Ctrl: a binding without it does not fire on Alt+key,
// so an application can keep the Alt layer for itself.
func TestKeybindingManagerMatchesAlt(t *testing.T) {
	km := NewKeybindingManager()
	var fired string
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 't', Alt: true, Handler: func() { fired = "alt+t" }})
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'x', Handler: func() { fired = "x" }})

	if !km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 't', Alt: true}) || fired != "alt+t" {
		t.Fatalf("Alt+t fired %q", fired)
	}
	if km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 't'}) {
		t.Error("t without Alt fired the Alt+t binding")
	}
	if km.Handle(driver.KeyEvent{Type: driver.KeyRune, Ch: 'x', Alt: true}) {
		t.Error("Alt+x fired the binding for x")
	}
	if got := formatKeybinding(Keybinding{Key: driver.KeyRune, Ch: 'k', Ctrl: true, Alt: true}); got != "Ctrl+Alt+K" {
		t.Errorf("formatKeybinding = %q", got)
	}
}
