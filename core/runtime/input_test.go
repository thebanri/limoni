package runtime

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
)

func TestMessageFromBackend(t *testing.T) {
	key := MessageFromBackend(backend.Event{Type: backend.EventKey, Key: backend.KeyEvent{Type: backend.KeyEnter}})
	if _, ok := key.(KeyPressMsg); !ok {
		t.Fatalf("key message = %T, want KeyPressMsg", key)
	}

	mouse := MessageFromBackend(backend.Event{Type: backend.EventMouse, Mouse: backend.MouseEvent{X: 4, Y: 2, Button: backend.MouseLeft}})
	press, ok := mouse.(MousePressMsg)
	if !ok || press.Position.X != 4 || press.Position.Y != 2 {
		t.Fatalf("mouse message = %#v, want mouse press at (4,2)", mouse)
	}

	wheel := MessageFromBackend(backend.Event{Type: backend.EventMouse, Mouse: backend.MouseEvent{Button: backend.MouseScrollDown}})
	if got, ok := wheel.(MouseWheelMsg); !ok || got.DeltaY != -1 {
		t.Fatalf("wheel message = %#v, want delta -1", wheel)
	}

	blur := MessageFromBackend(backend.Event{Type: backend.EventFocus, Focus: backend.FocusEvent{Gained: false}})
	if _, ok := blur.(BlurMsg); !ok {
		t.Fatalf("focus loss message = %T, want BlurMsg", blur)
	}
}
