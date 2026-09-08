package terminal

import (
	"os"
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
		MouseSupport:   true,  // Most modern terminals support mouse reporting
		BracketedPaste: true,  // Most modern terminals support bracketed paste
		SyncOutput:     false, // Synchronized Output (?2026) is gated on known supporting terminals
		GraphicsProto:  graphics.DetectProtocol(),
	}

	// 1. Detect TrueColor support
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		profile.TrueColor = true
		profile.Colors256 = true
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "direct") {
		profile.TrueColor = true
		profile.Colors256 = true
	} else if strings.Contains(term, "256color") {
		profile.Colors256 = true
	}

	// Some known modern terminals support TrueColor and Synchronized Output (?2026)
	termProg := os.Getenv("TERM_PROGRAM")
	if termProg == "kitty" || termProg == "WezTerm" || termProg == "Ghostty" || termProg == "iTerm.app" || termProg == "Apple_Terminal" {
		profile.TrueColor = true
		profile.Colors256 = true
		if termProg != "Apple_Terminal" {
			profile.SyncOutput = true
		}
	}

	if os.Getenv("WT_SESSION") != "" || strings.Contains(term, "alacritty") || strings.Contains(term, "foot") || strings.Contains(term, "ghostty") || strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") {
		profile.SyncOutput = true
	}

	return profile
}
