package terminal

import (
	"os"
	"runtime"
	"strings"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/grapheme"
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

	// EraseChar enables ECH (CSI n X) for runs of blanks. It is ECMA-48 and
	// implemented essentially everywhere, so it is on by default.
	EraseChar bool
	// RepeatChar enables REP (CSI n b) for runs of one glyph. Also ECMA-48, but
	// unevenly implemented — a terminal without it would print the escape and
	// corrupt the frame — so it stays off unless the terminal is recognised.
	// The capability handshake turns it on for terminals that name themselves
	// (XTVERSION) and are known to implement it.
	RepeatChar bool
	// Hyperlinks enables OSC 8, which turns a styled span into a clickable
	// link. A terminal that does not implement it is supposed to swallow the
	// sequence, and most do — but the ones that do not would print the URL
	// into the frame, so it stays off unless the terminal is recognised.
	Hyperlinks bool
	// ClusterWidths is on when the terminal confirmed mode 2027: it measures a
	// grapheme cluster as one unit, as Limoni does, so the diff need not
	// re-anchor the cursor after each one. Only the handshake sets it.
	ClusterWidths bool
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
		EraseChar:      true,
	}

	// Under js/wasm there is no process environment to inspect, but the host is
	// a browser terminal emulator (xterm.js and friends), all of which speak
	// 24-bit color. Without this the browser playground would be downsampled to
	// 16 colors purely because COLORTERM is absent.
	if runtime.GOOS == "js" {
		profile.TrueColor = true
		profile.Colors256 = true
		// xterm.js implements REP. Hyperlinks need an addon the page may not
		// have loaded, and there is no way to ask, so they stay off.
		profile.RepeatChar = true
		return profile
	}

	term := os.Getenv("TERM")
	if term == "dumb" || os.Getenv("LIMONI_NO_SYNC") == "1" {
		profile.SyncOutput = false
	}
	if term == "dumb" {
		profile.EraseChar = false
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

	// REP is only enabled where it is known to work. xterm defined it; VTE,
	// kitty, foot, WezTerm and Ghostty implement it. Anything unrecognised
	// keeps it off rather than risking a literal escape on screen.
	switch {
	case termProg == "kitty", termProg == "WezTerm", termProg == "Ghostty",
		termProg == "foot", termProg == "iTerm.app":
		profile.RepeatChar = true
	case strings.HasPrefix(term, "xterm"), strings.HasPrefix(term, "vte"),
		strings.HasPrefix(term, "kitty"), strings.HasPrefix(term, "foot"),
		strings.HasPrefix(term, "alacritty"), strings.HasPrefix(term, "wezterm"):
		profile.RepeatChar = true
	}
	// OSC 8 is implemented by the VTE terminals (GNOME, Tilix), kitty, foot,
	// WezTerm, Ghostty, iTerm2, Konsole, Windows Terminal and Alacritty since
	// 0.11. As with REP, an unrecognised terminal keeps it off.
	switch {
	case termProg == "kitty", termProg == "WezTerm", termProg == "Ghostty",
		termProg == "foot", termProg == "iTerm.app", termProg == "vscode":
		profile.Hyperlinks = true
	}
	// Matched anywhere in TERM, not at the front: kitty sets TERM to
	// "xterm-kitty" and Ghostty to "xterm-ghostty", so a prefix test finds
	// neither and quietly leaves hyperlinks off — which is exactly what
	// happened the first time this was tried in a real terminal.
	for _, name := range hyperlinkTerms {
		if strings.Contains(term, name) {
			profile.Hyperlinks = true
			break
		}
	}
	if os.Getenv("VTE_VERSION") != "" || os.Getenv("WT_SESSION") != "" ||
		os.Getenv("KONSOLE_VERSION") != "" {
		profile.Hyperlinks = true
	}

	// Escape hatches in both directions, until the handshake can ask.
	switch os.Getenv("LIMONI_HYPERLINKS") {
	case "1":
		profile.Hyperlinks = true
	case "0":
		profile.Hyperlinks = false
	}
	switch os.Getenv("LIMONI_REP") {
	case "1":
		profile.RepeatChar = true
	case "0":
		profile.RepeatChar = false
	}

	return profile
}

// hyperlinkTerms are the TERM fragments of terminals that implement OSC 8.
var hyperlinkTerms = []string{"kitty", "ghostty", "foot", "wezterm", "alacritty", "konsole", "contour", "vte", "gnome"}

// knownTerminals lists what Limoni knows about terminals that answer XTVERSION,
// keyed by the lower-cased name before the version. Only positive knowledge is
// recorded: a terminal missing here keeps what the environment suggested.
var knownTerminals = map[string]struct{ trueColor, rep, links bool }{
	"xterm":    {trueColor: false, rep: true}, // 24-bit SGR is accepted but may be approximated
	"kitty":    {trueColor: true, rep: true, links: true},
	"wezterm":  {trueColor: true, rep: true, links: true},
	"foot":     {trueColor: true, rep: true, links: true},
	"ghostty":  {trueColor: true, rep: true, links: true},
	"contour":  {trueColor: true, rep: true, links: true},
	"iterm2":   {trueColor: true, rep: true, links: true},
	"konsole":  {trueColor: true, rep: true, links: true},
	"xterm.js": {trueColor: true, rep: true},
	// tmux interprets REP itself before redrawing on the outer terminal, so
	// REP is safe whatever runs outside it. Colour depth depends on tmux's
	// own configuration and the outer terminal, so it is left alone.
	"tmux": {trueColor: false, rep: true},
}

// WithReport refines a profile guessed from the environment with what the
// terminal said about itself during the capability handshake. It only acts on
// answers: a question the terminal ignored leaves the guess in place. The
// LIMONI_NO_SYNC and LIMONI_REP overrides still win.
func (p CapabilityProfile) WithReport(r driver.TerminalReport) CapabilityProfile {
	if r.SyncOutput != driver.ModeUnknown && os.Getenv("LIMONI_NO_SYNC") != "1" {
		p.SyncOutput = r.SyncOutput.Recognized()
	}
	// A terminal draws clusters the way the buffer lays them out if it says
	// so (mode 2027 on) or if it was measured doing it.
	p.ClusterWidths = grapheme.Clusters() && (r.GraphemeClusters.Enabled() || r.ClusterWidth == 2)

	if known, ok := knownTerminals[strings.ToLower(r.Name)]; ok {
		if known.trueColor {
			p.TrueColor, p.Colors256 = true, true
		}
		if known.rep {
			p.RepeatChar = true
		}
		if known.links {
			p.Hyperlinks = true
		}
	}
	// A measurement beats both the name and the environment.
	switch r.Repeat {
	case driver.Yes:
		p.RepeatChar = true
	case driver.No:
		p.RepeatChar = false
	}
	switch os.Getenv("LIMONI_REP") {
	case "1":
		p.RepeatChar = true
	case "0":
		p.RepeatChar = false
	}
	switch os.Getenv("LIMONI_HYPERLINKS") {
	case "1":
		p.Hyperlinks = true
	case "0":
		p.Hyperlinks = false
	}
	return p
}
