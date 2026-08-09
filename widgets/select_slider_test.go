package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestSelectStateHandleKey(t *testing.T) {
	state := NewSelectState()
	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter}, 3) || !state.Open {
		t.Fatal("enter should open select")
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown}, 3)
	if state.Selected != 1 {
		t.Fatalf("selected = %d; want 1", state.Selected)
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter}, 3)
	if state.Open {
		t.Fatal("enter should close select")
	}
}

func TestSelectDrawRegistersOptionClick(t *testing.T) {
	state := NewSelectState()
	state.Open = true
	selectWidget := Select{ID: "env", Options: []string{"Development", "Production"}, State: state}
	area := cell.NewRect(0, 0, 20, 3)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	var optionHandler func(backend.MouseEvent)
	ctx.RegisterMouse = func(region cell.Rect, handler func(backend.MouseEvent)) {
		if region.Y == 2 {
			optionHandler = handler
		}
	}
	selectWidget.Draw(ctx, buf)
	if optionHandler == nil {
		t.Fatal("option click handler was not registered")
	}
	optionHandler(backend.MouseEvent{Button: backend.MouseLeft})
	if state.Selected != 1 || state.Open {
		t.Fatalf("select state = %+v; want selected 1 and closed", *state)
	}
}

func TestSliderStateClampAndKeyboard(t *testing.T) {
	state := NewSliderState(50)
	state.Set(200, 0, 100)
	if state.Value != 100 {
		t.Fatalf("value = %d; want 100", state.Value)
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyHome}, 0, 100)
	if state.Value != 0 {
		t.Fatalf("home value = %d; want 0", state.Value)
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowRight}, 0, 100)
	if state.Value != 1 {
		t.Fatalf("right value = %d; want 1", state.Value)
	}
}

func TestSliderMouseMapsValue(t *testing.T) {
	state := NewSliderState(0)
	slider := Slider{ID: "volume", State: state, Min: 0, Max: 100}
	area := cell.NewRect(10, 0, 11, 1)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	var mouseHandler func(backend.MouseEvent)
	ctx.RegisterMouse = func(_ cell.Rect, handler func(backend.MouseEvent)) { mouseHandler = handler }
	slider.Draw(ctx, buf)
	mouseHandler(backend.MouseEvent{Button: backend.MouseLeft, X: 20})
	if state.Value != 100 {
		t.Fatalf("value = %d; want 100", state.Value)
	}
}
