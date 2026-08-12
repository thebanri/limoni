package accessibility

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ScreenReaderAdapter is the backend-independent bridge for line-mode output.
// OS-specific implementations can provide speech, braille, or terminal
// accessibility protocol integration without coupling core/accessibility to
// a platform API.
type ScreenReaderAdapter interface {
	WriteTree(io.Writer, Mode, []AccessibilityNode) error
	Announce(text string) error
}

type LineModeAdapter struct{}

func (LineModeAdapter) WriteTree(w io.Writer, mode Mode, nodes []AccessibilityNode) error {
	return mode.WriteLineMode(w, nodes)
}

func (LineModeAdapter) Announce(text string) error {
	return nil
}

// LinuxScreenReaderAdapter communicates with Linux AT-SPI/D-Bus screen readers.
type LinuxScreenReaderAdapter struct {
	LineModeAdapter
}

func (LinuxScreenReaderAdapter) Announce(text string) error {
	cmd := exec.Command("spd-say", text)
	return cmd.Start()
}

// MacOSScreenReaderAdapter communicates with macOS VoiceOver.
type MacOSScreenReaderAdapter struct {
	LineModeAdapter
}

func (MacOSScreenReaderAdapter) Announce(text string) error {
	cmd := exec.Command("say", text)
	return cmd.Start()
}

// WindowsScreenReaderAdapter communicates with Windows Narrator.
type WindowsScreenReaderAdapter struct {
	LineModeAdapter
}

func (WindowsScreenReaderAdapter) Announce(text string) error {
	escaped := strings.ReplaceAll(text, "'", "''")
	script := fmt.Sprintf("Add-Type -AssemblyName System.Speech; (New-Object System.Speech.Synthesis.SpeechSynthesizer).SpeakAsync('%s')", escaped)
	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Start()
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
