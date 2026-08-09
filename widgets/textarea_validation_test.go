package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
)

func TestTextAreaStateEditing(t *testing.T) {
	state := NewTextAreaState()
	state.SetValue("one")
	state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter})
	state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: '2'})
	if state.Value() != "one\n2" {
		t.Fatalf("value = %q; want multiline text", state.Value())
	}
}

func TestValidator(t *testing.T) {
	rule := Validator{Required: true, MinLength: 3}
	if rule.Validate("") == "" || rule.Validate("ab") == "" || rule.Validate("abc") != "" {
		t.Fatal("validator rules were not applied")
	}
	errors := ValidateFields(map[string]string{"name": ""}, map[string]Validator{"name": rule})
	if errors["name"] == "" {
		t.Fatal("field error should be returned")
	}
}
