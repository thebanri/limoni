package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
)

func TestScopedKeybindings(t *testing.T) {
	km := NewKeybindingManager()

	globalTriggered := false
	modalTriggered := false

	// Register global shortcut
	km.Register(Keybinding{
		Key:   backend.KeyRune,
		Ch:    'q',
		Scope: "", // global
		Handler: func() {
			globalTriggered = true
		},
	})

	// Register same shortcut under "modal" scope (should override)
	km.Register(Keybinding{
		Key:   backend.KeyRune,
		Ch:    'q',
		Scope: "modal",
		Handler: func() {
			modalTriggered = true
		},
	})

	// Test 1: Global scope active (no scopes passed)
	ev := backend.KeyEvent{Type: backend.KeyRune, Ch: 'q'}
	if !km.Handle(ev) {
		t.Fatal("Expected keybinding to be handled")
	}
	if !globalTriggered {
		t.Error("Expected global keybinding to trigger")
	}
	if modalTriggered {
		t.Error("Did not expect modal keybinding to trigger")
	}

	// Reset state
	globalTriggered = false
	modalTriggered = false

	// Test 2: Active scope is "modal"
	if !km.Handle(ev, "modal") {
		t.Fatal("Expected keybinding to be handled under modal scope")
	}
	if globalTriggered {
		t.Error("Did not expect global keybinding to trigger when overridden")
	}
	if !modalTriggered {
		t.Error("Expected modal keybinding to trigger")
	}

	// Reset state
	globalTriggered = false
	modalTriggered = false

	// Test 3: Unrelated scope "other" (should fallback to global)
	if !km.Handle(ev, "other") {
		t.Fatal("Expected keybinding to fallback to global")
	}
	if !globalTriggered {
		t.Error("Expected global keybinding to trigger as fallback")
	}
	if modalTriggered {
		t.Error("Did not expect modal keybinding to trigger under unrelated scope")
	}
}
