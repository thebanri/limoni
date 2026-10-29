//go:build !js

package main

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

// -window opens kitty with the pointer hidden, full screen, running this
// game with the other flags, and marked so it does not offer -window again.
func TestWindowOpensKittyWithoutAPointer(t *testing.T) {
	cmd := windowCmd("/usr/bin/kitty", "/bin/lemonsweep", []string{"-fps=60"})
	want := []string{"/usr/bin/kitty", "-o", "mouse_hide_wait=0.05 1000 1000000 no", "-o", "repaint_delay=4", "-o", "input_delay=1", "--start-as=fullscreen", "--title", "Lemon Sweep", "/bin/lemonsweep", "-fps=60"}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("args %q\nwant %q", cmd.Args, want)
	}
	if !slices.Contains(cmd.Env, "LEMONSWEEP_WINDOW=1") {
		t.Fatal("the window is not marked as the game's own")
	}
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("LEMONSWEEP_WINDOW", "1")
	if windowHint() != "" {
		t.Fatal("the game's own window offers -window")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("LEMONSWEEP_WINDOW", "")
	if strings.Contains(windowHint(), "window") {
		t.Fatal("a terminal that is not kitty is offered -window")
	}
}

// In Apple's Terminal -window writes a profile with the line spacing that
// makes half blocks fill their cells, running the game in half blocks with
// the user's flags after, quoted for the shell and escaped for the plist.
func TestWindowInAppleTerminal(t *testing.T) {
	p := string(appleProfile("/Users/a b/go/bin/lemonsweep", []string{"-fps=60", "-seed=1&<'x'"}, "QUJD"))
	for _, want := range []string{
		"<key>FontHeightSpacing</key><real>0.75</real>",
		`<key>CommandString</key><string>LEMONSWEEP_WINDOW=1 '/Users/a b/go/bin/lemonsweep' '-fps=60' '-seed=1&amp;&lt;'\''x'\'''</string>`,
		"<key>Font</key><data>QUJD</data>",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("profile lacks %s\n%s", want, p)
		}
	}
	if strings.Contains(string(appleProfile("/x", nil, "")), "<key>Font</key>") {
		t.Error("a font that could not be made is written anyway")
	}
	t.Setenv("LEMONSWEEP_WINDOW", "")
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	if !strings.Contains(windowHint(), "-gui") {
		t.Error("Apple's Terminal is not offered -gui")
	}
	t.Setenv("LEMONSWEEP_WINDOW", "1")
	if windowHint() != "" {
		t.Error("the game's own window offers -window")
	}
}

// The game opens its own window unasked on macOS and Windows, whatever the
// terminal, but not inside -window's Terminal window or over SSH.
func TestAutoGUI(t *testing.T) {
	desktop := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	for _, c := range []struct {
		term, window, ssh string
		want              bool
	}{
		{"Apple_Terminal", "", "", desktop},
		{"Apple_Terminal", "1", "", false},
		{"Apple_Terminal", "", "10.0.0.2 22 10.0.0.1 22", false},
		{"iTerm.app", "", "", desktop},
	} {
		t.Setenv("TERM_PROGRAM", c.term)
		t.Setenv("LEMONSWEEP_WINDOW", c.window)
		t.Setenv("SSH_CONNECTION", c.ssh)
		if got := autoGUI(); got != c.want {
			t.Errorf("%+v: autoGUI %v", c, got)
		}
	}
}
