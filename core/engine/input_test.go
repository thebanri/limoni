package engine

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func TestMessageFromBackend(t *testing.T) {
	key := MessageFromBackend(driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyEnter}})
	if _, ok := key.(KeyPressMsg); !ok {
		t.Fatalf("key message = %T, want KeyPressMsg", key)
	}

	mouse := MessageFromBackend(driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{X: 4, Y: 2, Button: driver.MouseLeft}})
	press, ok := mouse.(MousePressMsg)
	if !ok || press.Position.X != 4 || press.Position.Y != 2 {
		t.Fatalf("mouse message = %#v, want mouse press at (4,2)", mouse)
	}

	wheel := MessageFromBackend(driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseScrollDown}})
	if got, ok := wheel.(MouseWheelMsg); !ok || got.DeltaY != -1 {
		t.Fatalf("wheel message = %#v, want delta -1", wheel)
	}

	blur := MessageFromBackend(driver.Event{Type: driver.EventFocus, Focus: driver.FocusEvent{Gained: false}})
	if _, ok := blur.(BlurMsg); !ok {
		t.Fatalf("focus loss message = %T, want BlurMsg", blur)
	}
}

// The bytes a terminal sends, through the parser, to the message a model
// sees. Hover used to arrive as a MousePressMsg with Button == MouseNone, so a
// model that handled presses without checking the button clicked on hover.
func TestMouseReportsBecomeTheRightMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Msg
	}{
		{"press", "\x1b[<0;5;3M", MousePressMsg{Position: cell.Point{X: 4, Y: 2}, Button: driver.MouseLeft}},
		{"right press", "\x1b[<2;5;3M", MousePressMsg{Position: cell.Point{X: 4, Y: 2}, Button: driver.MouseRight}},
		{"hover", "\x1b[<35;57;16M", MouseMotionMsg{Position: cell.Point{X: 56, Y: 15}, Button: driver.MouseNone}},
		{"drag", "\x1b[<32;6;3M", MouseMotionMsg{Position: cell.Point{X: 5, Y: 2}, Button: driver.MouseLeft}},
		{"release", "\x1b[<0;6;3m", MouseReleaseMsg{Position: cell.Point{X: 5, Y: 2}, Button: driver.MouseRelease}},
		{"wheel up", "\x1b[<64;10;4M", MouseWheelMsg{DeltaY: 1, Position: cell.Point{X: 9, Y: 3}}},
		{"wheel down", "\x1b[<65;10;4M", MouseWheelMsg{DeltaY: -1, Position: cell.Point{X: 9, Y: 3}}},
		{"wheel left", "\x1b[<66;10;4M", MouseWheelMsg{DeltaX: 1, Position: cell.Point{X: 9, Y: 3}}},
		{"wheel right", "\x1b[<67;10;4M", MouseWheelMsg{DeltaX: -1, Position: cell.Point{X: 9, Y: 3}}},
	}
	for _, c := range cases {
		ev, n := driver.ParseEvent([]byte(c.in))
		if n != len(c.in) {
			t.Fatalf("%s: parser consumed %d of %d bytes", c.name, n, len(c.in))
		}
		if got := MessageFromDriver(ev); got != c.want {
			t.Errorf("%s: got %#v, want %#v", c.name, got, c.want)
		}
	}
}
