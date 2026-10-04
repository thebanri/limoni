package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func TestSelectStateHandleKey(t *testing.T) {
	state := NewSelectState()
	if !state.HandleKey(driver.KeyEvent{Type: driver.KeyEnter}, 3) || !state.Open {
		t.Fatal("enter should open select")
	}
	state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowDown}, 3)
	if state.Selected != 1 {
		t.Fatalf("selected = %d; want 1", state.Selected)
	}
	state.HandleKey(driver.KeyEvent{Type: driver.KeyEnter}, 3)
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
	var optionHandler func(driver.MouseEvent)
	ctx.RegisterMouse = func(region cell.Rect, handler func(driver.MouseEvent)) {
		if region.Y == 2 {
			optionHandler = handler
		}
	}
	selectWidget.Draw(ctx, buf)
	if optionHandler == nil {
		t.Fatal("option click handler was not registered")
	}
	optionHandler(driver.MouseEvent{Button: driver.MouseLeft})
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
	state.HandleKey(driver.KeyEvent{Type: driver.KeyHome}, 0, 100)
	if state.Value != 0 {
		t.Fatalf("home value = %d; want 0", state.Value)
	}
	state.HandleKey(driver.KeyEvent{Type: driver.KeyArrowRight}, 0, 100)
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
	var mouseHandler func(driver.MouseEvent)
	ctx.RegisterMouse = func(_ cell.Rect, handler func(driver.MouseEvent)) { mouseHandler = handler }
	slider.Draw(ctx, buf)
	mouseHandler(driver.MouseEvent{Button: driver.MouseLeft, X: 20})
	if state.Value != 100 {
		t.Fatalf("value = %d; want 100", state.Value)
	}
}

// The handler is built once per state and reads the last frame's settings:
// drag through the captured handler, the wheel, focus and OnChange all still
// work, a range changed between frames is honoured, and a frame costs no
// allocation.
func TestSliderHandlersFollowTheLastFrame(t *testing.T) {
	state := NewSliderState(0)
	var changes []int
	focused := ""
	var captured func(driver.MouseEvent)
	var handler func(driver.MouseEvent)
	area := cell.NewRect(10, 0, 11, 1)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }
	ctx.SetFocus = func(id string) { focused = id }
	ctx.CaptureMouse = func(h func(driver.MouseEvent)) { captured = h }

	slider := Slider{ID: "volume", State: state, Min: 0, Max: 10, OnChange: func(v int) { changes = append(changes, v) }}
	slider.Draw(ctx, buf)
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 15})
	if state.Value != 5 || focused != "volume" || captured == nil {
		t.Fatalf("click: value %d, focused %q, captured %v", state.Value, focused, captured != nil)
	}
	captured(driver.MouseEvent{Button: driver.MouseLeft, Drag: true, X: 18})
	if state.Value != 8 {
		t.Errorf("drag to column 18 gave %d, want 8", state.Value)
	}
	captured(driver.MouseEvent{Button: driver.MouseRelease, X: 10})
	if state.Value != 8 {
		t.Errorf("the release moved the thumb to %d", state.Value)
	}
	handler(driver.MouseEvent{Button: driver.MouseScrollUp, X: 12})
	if state.Value != 9 {
		t.Errorf("wheel up gave %d, want 9", state.Value)
	}

	// The next frame halves the range: the same handler uses it.
	slider.Max = 5
	slider.Draw(ctx, buf)
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 20})
	if state.Value != 5 {
		t.Errorf("after the range changed, the right end gave %d, want 5", state.Value)
	}
	if want := []int{5, 8, 9, 5}; len(changes) != len(want) {
		t.Errorf("OnChange saw %v, want %v", changes, want)
	}

	if n := testing.AllocsPerRun(20, func() { slider.Draw(ctx, buf) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}
