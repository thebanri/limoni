package terminal

import (
	"runtime"
	"testing"
)

func TestSetCapabilitiesOverridesDetection(t *testing.T) {
	term := &Terminal{caps: DetectCapabilities()}

	want := CapabilityProfile{
		TrueColor:      true,
		Colors256:      true,
		MouseSupport:   false,
		BracketedPaste: false,
		SyncOutput:     false,
	}
	term.SetCapabilities(want)

	if got := term.Capabilities(); got != want {
		t.Fatalf("Capabilities() = %+v, want %+v", got, want)
	}
}

// SetCapabilities must tolerate a nil receiver, so callers can configure a
// terminal that failed to construct without a nil check at every call site.
func TestSetCapabilitiesNilReceiver(t *testing.T) {
	var term *Terminal
	term.SetCapabilities(CapabilityProfile{TrueColor: true})
}

func TestDetectCapabilitiesTrueColorFromColorterm(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("js/wasm always reports truecolor")
	}
	t.Setenv("COLORTERM", "truecolor")

	profile := DetectCapabilities()
	if !profile.TrueColor || !profile.Colors256 {
		t.Fatalf("COLORTERM=truecolor gave %+v, want 24-bit color", profile)
	}
}

func TestDetectCapabilitiesDumbTerminalDisablesSync(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("js/wasm does not read the environment")
	}
	t.Setenv("TERM", "dumb")

	if DetectCapabilities().SyncOutput {
		t.Fatal("TERM=dumb should disable synchronized output")
	}
}

// Synchronized output is safe to send blindly — unsupported terminals ignore
// the sequence — so it must stay on for an ordinary terminal.
func TestDetectCapabilitiesSyncDefaultsOn(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("js/wasm takes the browser path")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("LIMONI_NO_SYNC", "")

	if !DetectCapabilities().SyncOutput {
		t.Fatal("synchronized output should default to enabled")
	}
}

func TestDetectCapabilitiesRespectsNoSyncEscapeHatch(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("js/wasm does not read the environment")
	}
	t.Setenv("LIMONI_NO_SYNC", "1")

	if DetectCapabilities().SyncOutput {
		t.Fatal("LIMONI_NO_SYNC=1 should disable synchronized output")
	}
}
