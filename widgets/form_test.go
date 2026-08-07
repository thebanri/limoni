package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
)

func TestTextInputState(t *testing.T) {
	state := NewTextInputState()
	if state.Value() != "" {
		t.Errorf("NewTextInputState.Value() = %q; boş metin bekleniyordu", state.Value())
	}

	// Karakter ekleme
	state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: 'a'})
	state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: 'b'})
	if state.Value() != "ab" {
		t.Errorf("Value() = %q; 'ab' bekleniyordu", state.Value())
	}
	if state.Cursor != 2 {
		t.Errorf("Cursor = %d; 2 bekleniyordu", state.Cursor)
	}

	// Geri silme (Backspace)
	state.HandleKey(backend.KeyEvent{Type: backend.KeyBackspace})
	if state.Value() != "a" {
		t.Errorf("Value() = %q; 'a' bekleniyordu", state.Value())
	}
	if state.Cursor != 1 {
		t.Errorf("Cursor = %d; 1 bekleniyordu", state.Cursor)
	}

	// Yön tuşuyla sola gitme
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowLeft})
	if state.Cursor != 0 {
		t.Errorf("Cursor = %d; 0 bekleniyordu", state.Cursor)
	}

	// Araya karakter ekleme
	state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: 'z'})
	if state.Value() != "za" {
		t.Errorf("Value() = %q; 'za' bekleniyordu", state.Value())
	}
	if state.Cursor != 1 {
		t.Errorf("Cursor = %d; 1 bekleniyordu", state.Cursor)
	}

	// Delete tuşuyla sağdakini silme
	state.Cursor = 0 // Başa al
	state.HandleKey(backend.KeyEvent{Type: backend.KeyDelete})
	if state.Value() != "a" {
		t.Errorf("Value() = %q; 'a' bekleniyordu", state.Value())
	}
}

func TestCheckboxAndRadio(t *testing.T) {
	// Checkbox toggle testi
	checked := false
	cb := Checkbox{
		ID:      "test_cb",
		Checked: &checked,
		Label:   "Onayla",
	}
	
	// Test size hint
	w, h := cb.SizeHint(cell.NewRect(0, 0, 100, 100))
	if w != 10 || h != 1 {
		t.Errorf("Checkbox.SizeHint() = (%d, %d); (10, 1) bekleniyordu", w, h)
	}

	// RadioButton seçimi
	selected := "OptionA"
	rb := RadioButton{
		ID:       "test_rb",
		Selected: &selected,
		Value:    "OptionB",
		Label:    "Seçenek B",
	}

	rw, rh := rb.SizeHint(cell.NewRect(0, 0, 100, 100))
	if rw != 13 || rh != 1 {
		t.Errorf("RadioButton.SizeHint() = (%d, %d); (13, 1) bekleniyordu", rw, rh)
	}
}
