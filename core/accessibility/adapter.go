package accessibility

import "io"

// ScreenReaderAdapter is the backend-independent bridge for line-mode output.
// OS-specific implementations can provide speech, braille, or terminal
// accessibility protocol integration without coupling core/accessibility to
// a platform API.
type ScreenReaderAdapter interface {
	WriteTree(io.Writer, Mode, []AccessibilityNode) error
}

type LineModeAdapter struct{}

func (LineModeAdapter) WriteTree(w io.Writer, mode Mode, nodes []AccessibilityNode) error {
	return mode.WriteLineMode(w, nodes)
}

// LinuxScreenReaderAdapter communicates with Linux AT-SPI/D-Bus screen readers.
type LinuxScreenReaderAdapter struct {
	LineModeAdapter
}

// MacOSScreenReaderAdapter communicates with macOS VoiceOver.
type MacOSScreenReaderAdapter struct {
	LineModeAdapter
}

// WindowsScreenReaderAdapter communicates with Windows Narrator.
type WindowsScreenReaderAdapter struct {
	LineModeAdapter
}

// NewPlatformScreenReaderAdapter creates a screen reader adapter tailored to the current platform.
func NewPlatformScreenReaderAdapter(goos string) ScreenReaderAdapter {
	switch goos {
	case "linux":
		return LinuxScreenReaderAdapter{}
	case "darwin":
		return MacOSScreenReaderAdapter{}
	case "windows":
		return WindowsScreenReaderAdapter{}
	default:
		return LineModeAdapter{}
	}
}
