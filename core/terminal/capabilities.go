package terminal

import (
	"os"
	"runtime"
	"strings"

	"github.com/thebanri/limoni/graphics"
)

// CapabilityProfile defines the capability flags supported by the active terminal.
type CapabilityProfile struct {
	TrueColor      bool
	Colors256      bool
	MouseSupport   bool
	BracketedPaste bool
	SyncOutput     bool
	GraphicsProto  graphics.Protocol
}

// DetectCapabilities automatically detects the active terminal's capability profile using environment variables.
func DetectCapabilities() CapabilityProfile {
	profile := CapabilityProfile{
		TrueColor:      false,
		Colors256:      false,
		MouseSupport:   true, // Most modern terminals support mouse reporting
		BracketedPaste: true, // Most modern terminals support bracketed paste
		SyncOutput:     true, // Synchronized Output (?2026) enables atomic tear-free frames (safely ignored if unsupported)
		GraphicsProto:  graphics.DetectProtocol(),
	}

	// Under js/wasm there is no process environment to inspect, but the host is
	// a browser terminal emulator (xterm.js and friends), all of which speak
	// 24-bit color. Without this the browser playground would be downsampled to
	// 16 colors purely because COLORTERM is absent.
	if runtime.GOOS == "js" {
		profile.TrueColor = true
		profile.Colors256 = true
		return profile
	}

	term := os.Getenv("TERM")
	if term == "dumb" || os.Getenv("LIMONI_NO_SYNC") == "1" {
		profile.SyncOutput = false
	}

	// 1. Detect TrueColor support
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		profile.TrueColor = true
		profile.Colors256 = true
	}

	if strings.Contains(term, "direct") {
		profile.TrueColor = true
		profile.Colors256 = true
	} else if strings.Contains(term, "256color") {
		profile.Colors256 = true
	}

	// Some known modern terminals support TrueColor
	termProg := os.Getenv("TERM_PROGRAM")
	if termProg == "kitty" || termProg == "WezTerm" || termProg == "Ghostty" || termProg == "iTerm.app" || termProg == "Apple_Terminal" {
		profile.TrueColor = true
		profile.Colors256 = true
	}

	return profile
}
