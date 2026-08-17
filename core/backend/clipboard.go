package backend

import (
	"encoding/base64"
	"fmt"
	"io"
)

// OSC 52 terminal clipboard control sequences.
// Works seamlessly across local terminal emulators (Alacritty, Kitty, WezTerm, iTerm2, Windows Terminal)
// as well as remote SSH sessions without requiring X11 forwarding or native OS clip utilities.

// SetClipboardString returns the OSC 52 ANSI escape sequence to copy text to the host clipboard.
func SetClipboardString(text string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// Primary and Clipboard selection targets: 'c'
	return fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
}

// WriteClipboard writes the OSC 52 copy sequence directly to an output stream (e.g. stdout or SSH session).
func WriteClipboard(w io.Writer, text string) error {
	if w == nil {
		return fmt.Errorf("backend: nil writer passed to WriteClipboard")
	}
	_, err := io.WriteString(w, SetClipboardString(text))
	return err
}

// ReadClipboardRequest returns the OSC 52 ANSI escape query to request clipboard content from the terminal.
func ReadClipboardRequest() string {
	return "\x1b]52;c;?\x07"
}
