package driver

// Driver represents the low-level terminal I/O, raw mode controller, and event loop.
// Backend is retained as an alias for backward compatibility.
type Driver = Backend

// EventType represents the category of a terminal event.
type EventType uint8

const (
	EventNone EventType = iota
	EventKey
	EventMouse
	EventResize
	EventFocus
	EventPaste
	// EventReply is the terminal answering a query Limoni sent — device
	// attributes, a mode report, its name and version. Backends consume these
	// themselves (see TerminalReport); applications do not normally see them.
	EventReply
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

// ReplyKind says which query a ReplyEvent answers.
type ReplyKind uint8

const (
	ReplyNone ReplyKind = iota
	// ReplyPrimaryDA is the answer to DA1 (CSI c). Every terminal answers it,
	// which makes it the sentinel: once it arrives, every query sent before it
	// has been answered or never will be.
	ReplyPrimaryDA
	// ReplyMode is a DECRPM mode report (CSI ? mode ; setting $ y).
	ReplyMode
	// ReplyVersion is an XTVERSION report (DCS > | name-and-version ST).
	ReplyVersion
	// ReplyKittyKeyboard reports the progressive keyboard flags (CSI ? flags u).
	ReplyKittyKeyboard
	// ReplyCursor is a cursor position report (CSI row ; col R), the answer to
	// CSI 6 n. The same bytes are what some terminals send for a modified F3,
	// which Limoni has never delivered as a key; they are read as a report.
	ReplyCursor
)

// ReplyEvent carries a terminal's answer to a query.
type ReplyEvent struct {
	Kind ReplyKind
	// Mode and Setting hold a DECRPM report. Setting follows DECRQM: 0 not
	// recognised, 1 set, 2 reset, 3 permanently set, 4 permanently reset.
	Mode, Setting int
	// Attributes holds the DA1 attribute list as a bit set (attribute n is
	// bit n, for n < 64). Attribute 4 is sixel graphics.
	Attributes uint64
	// Flags holds the Kitty keyboard flags.
	Flags int
	// Row and Col hold a cursor position report, 1-based.
	Row, Col int
	// Version holds the XTVERSION text, e.g. "kitty(0.39.1)" or "tmux 3.5a".
	Version string
}

// Event is a flat, zero-allocation container unifying all TUI event types.
// Using a value type instead of interfaces prevents heap allocations in high-frequency event loops.
type Event struct {
	Type   EventType
	Key    KeyEvent
	Mouse  MouseEvent
	Resize ResizeEvent
	Focus  FocusEvent
	Paste  PasteEvent
	Reply  ReplyEvent
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
