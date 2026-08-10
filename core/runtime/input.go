package runtime

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
)

// KeyPressMsg represents a key press delivered to a model.
type KeyPressMsg struct{ Key backend.KeyEvent }

// KeyReleaseMsg represents a key release from an input adapter that supports
// release reporting. The Linux backend currently emits presses only.
type KeyReleaseMsg struct{ Key backend.KeyEvent }

// MousePressMsg represents a mouse button press.
type MousePressMsg struct {
	Position cell.Point
	Button   backend.MouseButton
}

// MouseReleaseMsg represents a mouse button release.
type MouseReleaseMsg struct {
	Position cell.Point
	Button   backend.MouseButton
}

// MouseWheelMsg represents a normalized wheel delta.
type MouseWheelMsg struct{ DeltaX, DeltaY int }

// PasteMsg represents bracketed-paste text supplied by an input adapter.
type PasteMsg struct{ Text string }

// ResizeMsg represents a terminal size change.
type ResizeMsg struct{ Width, Height uint16 }

// FocusMsg represents a terminal focus transition.
type FocusMsg struct{ Gained bool }

// BlurMsg represents loss of terminal focus.
type BlurMsg struct{}

// MessageFromBackend converts backend events that have a typed runtime
// equivalent. Resize and focus messages are returned directly; key and mouse
// events are converted according to their button/event kind.
func MessageFromBackend(event backend.Event) Msg {
	switch event.Type {
	case backend.EventKey:
		return KeyPressMsg{Key: event.Key}
	case backend.EventResize:
		return ResizeMsg{Width: event.Resize.Width, Height: event.Resize.Height}
	case backend.EventFocus:
		if event.Focus.Gained {
			return FocusMsg{Gained: true}
		}
		return BlurMsg{}
	case backend.EventMouse:
		position := cell.Point{X: event.Mouse.X, Y: event.Mouse.Y}
		switch event.Mouse.Button {
		case backend.MouseScrollUp:
			return MouseWheelMsg{DeltaY: 1}
		case backend.MouseScrollDown:
			return MouseWheelMsg{DeltaY: -1}
		case backend.MouseRelease:
			return MouseReleaseMsg{Position: position, Button: event.Mouse.Button}
		default:
			return MousePressMsg{Position: position, Button: event.Mouse.Button}
		}
	default:
		return nil
	}
}
