package engine

import (
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// KeyPressMsg represents a key press delivered to a model.
type KeyPressMsg struct{ Key driver.KeyEvent }

// KeyReleaseMsg represents a key release from an input adapter that supports
// release reporting. The Linux backend currently emits presses only.
type KeyReleaseMsg struct{ Key driver.KeyEvent }

// MousePressMsg represents a mouse button press: Button is MouseLeft,
// MouseMiddle or MouseRight. Pointer movement arrives as MouseMotionMsg, never
// as a press.
type MousePressMsg struct {
	Position cell.Point
	Button   driver.MouseButton
}

// MouseReleaseMsg represents a mouse button release.
type MouseReleaseMsg struct {
	Position cell.Point
	Button   driver.MouseButton
}

// MouseMotionMsg represents the pointer moving. Button is MouseNone when no
// button is held (hover) and the held button while dragging. Terminals with
// any-motion tracking report every move, so a model that only cares about
// clicks can ignore this message.
type MouseMotionMsg struct {
	Position cell.Point
	Button   driver.MouseButton
}

// MouseWheelMsg represents a normalized wheel delta at Position, the cell
// under the pointer. DeltaY is positive for wheel up (away from the user) and
// negative for wheel down, so scrolling content follows the wheel with
// ScrollBy(-DeltaY). DeltaX follows the same rule for a horizontal wheel or
// tilt: positive is left, so a horizontal offset follows with -DeltaX.
type MouseWheelMsg struct {
	DeltaX, DeltaY int
	Position       cell.Point
}

// PasteMsg represents bracketed-paste text supplied by an input adapter.
type PasteMsg struct{ Text string }

// ResizeMsg represents a terminal size change.
type ResizeMsg struct{ Width, Height uint16 }

// FocusMsg represents a terminal focus transition.
type FocusMsg struct{ Gained bool }

// BlurMsg represents loss of terminal focus.
type BlurMsg struct{}

// MessageFromDriver converts driver events that have a typed engine
// equivalent. Resize and focus messages are returned directly; key and mouse
// events are converted according to their button/event kind.
func MessageFromDriver(event driver.Event) Msg {
	switch event.Type {
	case driver.EventKey:
		return KeyPressMsg{Key: event.Key}
	case driver.EventResize:
		return ResizeMsg{Width: event.Resize.Width, Height: event.Resize.Height}
	case driver.EventFocus:
		if event.Focus.Gained {
			return FocusMsg{Gained: true}
		}
		return BlurMsg{}
	case driver.EventPaste:
		return PasteMsg{Text: event.Paste.Text}
	case driver.EventMouse:
		position := cell.Point{X: event.Mouse.X, Y: event.Mouse.Y}
		switch button := event.Mouse.Button; {
		case button == driver.MouseScrollUp:
			return MouseWheelMsg{DeltaY: 1, Position: position}
		case button == driver.MouseScrollDown:
			return MouseWheelMsg{DeltaY: -1, Position: position}
		case button == driver.MouseScrollLeft:
			return MouseWheelMsg{DeltaX: 1, Position: position}
		case button == driver.MouseScrollRight:
			return MouseWheelMsg{DeltaX: -1, Position: position}
		case button == driver.MouseRelease:
			return MouseReleaseMsg{Position: position, Button: button}
		case event.Mouse.Drag || button == driver.MouseNone:
			// A move: hover reports MouseNone, a drag the held button.
			return MouseMotionMsg{Position: position, Button: button}
		default:
			return MousePressMsg{Position: position, Button: button}
		}
	default:
		return nil
	}
}

// MessageFromBackend is a backward-compatible alias for MessageFromDriver.
var MessageFromBackend = MessageFromDriver
