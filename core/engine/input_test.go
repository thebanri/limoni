package engine

import (
	"testing"

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
