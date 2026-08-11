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
