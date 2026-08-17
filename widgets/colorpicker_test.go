package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestColorPicker_StateAndHex(t *testing.T) {
	state := NewColorPickerState(255, 128, 0)
	if state.HexInput != "FF8000" {
		t.Errorf("expected hex FF8000, got %s", state.HexInput)
	}

	ok := state.SetHex("#00FF55")
	if !ok || state.Red != 0 || state.Green != 255 || state.Blue != 85 {
		t.Errorf("expected RGB (0, 255, 85), got (%d, %d, %d)", state.Red, state.Green, state.Blue)
	}

	// Tab through modes
	state.HandleKey(backend.KeyEvent{Type: backend.KeyTab}, nil)
	if state.ActiveMode != 1 {
		t.Errorf("expected active mode 1 (RGB), got %d", state.ActiveMode)
	}

	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowRight}, nil)
	if state.Red != 5 {
		t.Errorf("expected Red 5 after +5, got %d", state.Red)
	}
}

func TestColorPicker_Draw(t *testing.T) {
	state := NewColorPickerState(255, 0, 128)
	cp := ColorPicker{
		ID:          "color_picker",
		State:       state,
		ShowPreview: true,
	}

	area := cell.NewRect(0, 0, 40, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	cp.Draw(ctx, buf)

	c := buf.Get(0, 0)
	if c == nil {
		t.Fatal("expected buffer cell at (0, 0)")
	}
}
