package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)


func TestTextInputState(t *testing.T) {
	state := NewTextInputState()
	if state.Value() != "" {
		t.Errorf("NewTextInputState.Value() = %q; boş metin bekleniyordu", state.Value())
	}

	// Karakter ekleme
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'a'})
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'b'})
	if state.Value() != "ab" {
		t.Errorf("Value() = %q; 'ab' bekleniyordu", state.Value())
	}
	if state.Cursor != 2 {
		t.Errorf("Cursor = %d; 2 bekleniyordu", state.Cursor)
	}

	// Geri silme (Backspace)
	state.HandleKey(driver.KeyEvent{Type: driver.KeyBackspace})
	if state.Value() != "a" {
		t.Errorf("Value() = %q; 'a' bekleniyordu", state.Value())
	}
	if state.Cursor != 1 {
		t.Errorf("Cursor = %d; 1 bekleniyordu", state.Cursor)
	}

	// Yön tuşuyla sola gitme
	state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowLeft})
	if state.Cursor != 0 {
		t.Errorf("Cursor = %d; 0 bekleniyordu", state.Cursor)
	}

	// Araya karakter ekleme
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'z'})
	if state.Value() != "za" {
		t.Errorf("Value() = %q; 'za' bekleniyordu", state.Value())
	}
	if state.Cursor != 1 {
		t.Errorf("Cursor = %d; 1 bekleniyordu", state.Cursor)
	}

	// Delete tuşuyla sağdakini silme
	state.Cursor = 0 // Başa al
	state.HandleKey(driver.KeyEvent{Type: driver.KeyDelete})
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

func TestTextInputMultilineAndSelection(t *testing.T) {
	state := NewTextInputState()
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'h'})
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'i'})

	// Shift+Enter should insert newline
	handled := state.HandleKey(driver.KeyEvent{Type: driver.KeyEnter, Shift: true})
	if !handled {
		t.Fatal("Shift+Enter was not handled")
	}
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '!'})

	if state.Value() != "hi\n!" {
		t.Fatalf("state.Value() = %q; want 'hi\\n!'", state.Value())
	}

	// Test Draw with selection and software cursor
	ti := NewTextInput("inp1").
		WithState(state).
		WithSelection(0, 2).
		WithFocused(true)

	buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 1))
	ctx := cell.Context{
		Area: cell.NewRect(0, 0, 20, 1),
	}

	ti.Draw(ctx, buf)

	// Verify buffer rendered characters and selection style
	// First character 'h' at col 0 should have selection style (or reverse)
	c0 := buf.Get(0, 0)
	if c0 == nil || c0.Content != 'h' {
		t.Fatalf("buf.Get(0,0) content = %c; want 'h'", c0.Content)
	}
	if c0.Style.Modifier&cell.ModifierBold == 0 {
		t.Errorf("expected selected character to have bold/selection modifier")
	}

	// Verify newline replacement character is rendered ('↵')
	hasReturnSymbol := false
	for x := uint16(0); x < 20; x++ {
		c := buf.Get(x, 0)
		if c != nil && c.Content == '↵' {
			hasReturnSymbol = true
			break
		}
	}
	if !hasReturnSymbol {
		t.Errorf("expected newline to be rendered with return symbol '↵'")
	}
}

