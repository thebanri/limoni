package backend

// EventType represents the category of a terminal event.
type EventType uint8

const (
	EventKey EventType = iota
	EventMouse
	EventResize
	EventFocus
	EventPaste
)

// KeyType represents special keyboard keys.
type KeyType uint16

const (
	KeyRune KeyType = iota
	KeyEsc
	KeyEnter
	KeyBackspace
	KeyTab
	KeySpace

	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight

	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyInsert
	KeyDelete

	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// KeyEvent represents a key press event along with active modifier keys.
type KeyEvent struct {
	Type  KeyType // Key type (Rune or Special Key)
	Ch    rune    // Character rune when Type == KeyRune
	Alt   bool    // Alt key active
	Ctrl  bool    // Ctrl key active
	Shift bool    // Shift key active
}

// MouseButton represents a mouse button action.
type MouseButton uint8

const (
	MouseNone MouseButton = iota
	MouseLeft
	MouseRight
	MouseMiddle
	MouseRelease
	MouseScrollUp
	MouseScrollDown
)

// MouseEvent represents mouse movements, clicks, and coordinate positions.
type MouseEvent struct {
	Button MouseButton // Mouse button clicked or released
	X      uint16      // 0-indexed terminal column
	Y      uint16      // 0-indexed terminal row
	Drag   bool        // Whether this is a mouse drag movement
	Alt    bool        // Alt key held
	Ctrl   bool        // Ctrl key held
	Shift  bool        // Shift key held
}

// ResizeEvent represents a terminal window resize event.
type ResizeEvent struct {
	Width  uint16
	Height uint16
}

// FocusEvent represents terminal focus gain or loss.
type FocusEvent struct {
	Gained bool
}

type PasteEvent struct{ Text string }

// Event is a flat, zero-allocation container unifying all TUI event types.
// Using a value type instead of interfaces prevents heap allocations in high-frequency event loops.
type Event struct {
	Type   EventType
	Key    KeyEvent
	Mouse  MouseEvent
	Resize ResizeEvent
	Focus  FocusEvent
	Paste  PasteEvent
}

// PlatformCapabilityMatrix reports terminal and OS capability support.
type PlatformCapabilityMatrix struct {
	OS             string
	HasRawMode     bool
	HasMouseSGR    bool
	HasFocusReport bool
	HasAltBuffer   bool
	HasIoctlResize bool
}

// GetPlatformCapabilities returns capabilities based on operating system.
func GetPlatformCapabilities(goos string) PlatformCapabilityMatrix {
	switch goos {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     true,
			HasMouseSGR:    true,
			HasFocusReport: true,
			HasAltBuffer:   true,
			HasIoctlResize: true,
		}
	case "windows":
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     true,
			HasMouseSGR:    true,
			HasFocusReport: true,
			HasAltBuffer:   true,
			HasIoctlResize: false,
		}
	default:
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     false,
			HasMouseSGR:    false,
			HasFocusReport: false,
			HasAltBuffer:   false,
			HasIoctlResize: false,
		}
	}
}
