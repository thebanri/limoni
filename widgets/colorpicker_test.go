package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestColorPicker_HSVAndRGB(t *testing.T) {
	state := NewColorPickerState(255, 0, 0)
	if state.HexInput != "FF0000" {
		t.Errorf("expected hex FF0000, got %s", state.HexInput)
	}
	if state.Hue != 0 || state.Sat != 1.0 || state.Val != 1.0 {
		t.Errorf("expected HSV (0, 1, 1), got (%.1f, %.1f, %.1f)", state.Hue, state.Sat, state.Val)
	}

	state.SetHSV(120, 1.0, 1.0) // Pure Green
	if state.Red != 0 || state.Green != 255 || state.Blue != 0 {
		t.Errorf("expected pure green (0, 255, 0), got (%d, %d, %d)", state.Red, state.Green, state.Blue)
	}

	ok := state.SetHex("#0000FF") // Pure Blue
	if !ok || state.Blue != 255 || state.Hue != 240 {
		t.Errorf("expected pure blue (0, 0, 255), Hue 240, got (%d, %d, %d), Hue %.1f", state.Red, state.Green, state.Blue, state.Hue)
	}
}

func TestColorPicker_Draw2D(t *testing.T) {
	state := NewColorPickerState(255, 128, 0)
	cp := ColorPicker{
		ID:          "kde_color_picker",
		State:       state,
		ShowPreview: true,
	}

	area := cell.NewRect(0, 0, 52, 14)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	cp.Draw(ctx, buf)

	// Verify buffer cells in 2D gradient field area
	c := buf.Get(5, 5)
	if c == nil {
		t.Fatal("expected non-nil cell in 2D color picker field")
	}
}

func TestColorPicker_HandleKey(t *testing.T) {
	state := NewColorPickerState(255, 255, 255)
	state.ActiveMode = 0 // 2D Field

	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowRight}, nil)
	state.HandleKey(backend.KeyEvent{Type: backend.KeyTab}, nil)
	if state.ActiveMode != 1 { // Switched to Hue bar
		t.Errorf("expected active mode 1, got %d", state.ActiveMode)
	}
}
