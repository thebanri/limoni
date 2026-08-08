package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
)

func TestKeybindingManager_New(t *testing.T) {
	km := NewKeybindingManager()
	if km == nil {
		t.Fatal("NewKeybindingManager nil dönmemeli")
	}
	if len(km.AllBindings()) != 0 {
		t.Fatalf("yeni yöneticide %d kısayol var; 0 bekleniyordu", len(km.AllBindings()))
	}
}

func TestKeybindingManager_Register(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: backend.KeyRune, Ch: 'p', Ctrl: true, Label: "Palet", Category: "Genel"})
	km.Register(Keybinding{Key: backend.KeyEsc, Label: "Kapat", Category: "Genel"})

	if len(km.AllBindings()) != 2 {
		t.Fatalf("AllBindings uzunluğu = %d; 2 bekleniyordu", len(km.AllBindings()))
	}
}

func TestKeybindingManager_Handle_Rune(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key: backend.KeyRune, Ch: 'p', Ctrl: true,
		Handler: func() { ran = true },
	})

	// Eşleşen tuş
	if !km.Handle(backend.KeyEvent{Type: backend.KeyRune, Ch: 'p', Ctrl: true}) {
		t.Fatal("eşleşen kısayol true dönmeli")
	}
	if !ran {
		t.Fatal("handler çalışmalı")
	}

	// Yanlış karakter
	ran = false
	if km.Handle(backend.KeyEvent{Type: backend.KeyRune, Ch: 'x', Ctrl: true}) {
		t.Fatal("eşleşmeyen karakter false dönmeli")
	}
	if ran {
		t.Fatal("eşleşmeyen karakter handler'ı çalıştırmamalı")
	}

	// Ctrl eksik
	if km.Handle(backend.KeyEvent{Type: backend.KeyRune, Ch: 'p'}) {
		t.Fatal("Ctrl'suz tuş eşleşmemeli")
	}
}

func TestKeybindingManager_Handle_SpecialKey(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key:     backend.KeyEsc,
		Handler: func() { ran = true },
	})

	if !km.Handle(backend.KeyEvent{Type: backend.KeyEsc}) {
		t.Fatal("Esc eşleşmeli")
	}
	if !ran {
		t.Fatal("handler çalışmalı")
	}

	// Farklı tuş eşleşmemeli
	if km.Handle(backend.KeyEvent{Type: backend.KeyEnter}) {
		t.Fatal("Enter Esc kısayoluyla eşleşmemeli")
	}
}

func TestKeybindingManager_Handle_Shift(t *testing.T) {
	km := NewKeybindingManager()
	ran := false
	km.Register(Keybinding{
		Key: backend.KeyTab, Shift: true,
		Handler: func() { ran = true },
	})

	if !km.Handle(backend.KeyEvent{Type: backend.KeyTab, Shift: true}) {
		t.Fatal("Shift+Tab eşleşmeli")
	}
	if !ran {
		t.Fatal("handler çalışmalı")
	}

	// Shift'siz Tab eşleşmemeli
	ran = false
	if km.Handle(backend.KeyEvent{Type: backend.KeyTab}) {
		t.Fatal("Shift'siz Tab eşleşmemeli")
	}
	if ran {
		t.Fatal("Shift'siz Tab handler'ı çalıştırmamalı")
	}
}

func TestKeybindingManager_Handle_FirstMatchWins(t *testing.T) {
	km := NewKeybindingManager()
	firstRan, secondRan := false, false
	km.Register(Keybinding{
		Key: backend.KeyRune, Ch: 'a',
		Handler: func() { firstRan = true },
	})
	km.Register(Keybinding{
		Key: backend.KeyRune, Ch: 'a',
		Handler: func() { secondRan = true },
	})

	km.Handle(backend.KeyEvent{Type: backend.KeyRune, Ch: 'a'})
	if !firstRan {
		t.Fatal("ilk kayıtlı handler çalışmalı")
	}
	if secondRan {
		t.Fatal("ilk eşleşme kazandığı için ikinci handler çalışmamalı")
	}
}

func TestKeybindingManager_Handle_NoMatch(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: backend.KeyRune, Ch: 'q', Ctrl: true})

	if km.Handle(backend.KeyEvent{Type: backend.KeyArrowUp}) {
		t.Fatal("kayıtlı olmayan tuş false dönmeli")
	}
}

func TestKeybindingManager_ToCommandItems(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{
		Key: backend.KeyRune, Ch: 'p', Ctrl: true,
		Label: "Komut Paletini Aç/Kapa", Category: "Genel",
		Handler: func() {},
	})
	// Label'sız kısayollar CommandItem'a dönüşmemeli
	km.Register(Keybinding{Key: backend.KeyEsc, Handler: func() {}})

	items := km.ToCommandItems()
	if len(items) != 1 {
		t.Fatalf("ToCommandItems uzunluğu = %d; 1 bekleniyordu", len(items))
	}
	if items[0].Label != "Komut Paletini Aç/Kapa" {
		t.Fatalf("Label = %q; 'Komut Paletini Aç/Kapa' bekleniyordu", items[0].Label)
	}
	if items[0].Category != "Genel" {
		t.Fatalf("Category = %q; 'Genel' bekleniyordu", items[0].Category)
	}
	if items[0].Detail != "Ctrl+P" {
		t.Fatalf("Detail = %q; 'Ctrl+P' bekleniyordu", items[0].Detail)
	}
	if items[0].Handler == nil {
		t.Fatal("Handler nil olmamalı")
	}
}

func TestFormatKeybinding(t *testing.T) {
	cases := []struct {
		name string
		kb   Keybinding
		want string
	}{
		{"rune", Keybinding{Key: backend.KeyRune, Ch: 'p', Ctrl: true}, "Ctrl+P"},
		{"rune shift", Keybinding{Key: backend.KeyRune, Ch: 'n', Ctrl: true, Shift: true}, "Ctrl+Shift+N"},
		{"tab", Keybinding{Key: backend.KeyTab}, "Tab"},
		{"esc", Keybinding{Key: backend.KeyEsc}, "Esc"},
		{"enter", Keybinding{Key: backend.KeyEnter}, "Enter"},
		{"space", Keybinding{Key: backend.KeySpace}, "Space"},
		{"backspace", Keybinding{Key: backend.KeyBackspace}, "Backspace"},
		{"arrow up", Keybinding{Key: backend.KeyArrowUp}, "↑"},
		{"arrow down", Keybinding{Key: backend.KeyArrowDown}, "↓"},
		{"arrow left", Keybinding{Key: backend.KeyArrowLeft}, "←"},
		{"arrow right", Keybinding{Key: backend.KeyArrowRight}, "→"},
		{"unknown", Keybinding{Key: backend.KeyF1}, "?"},
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
