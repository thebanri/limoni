//go:build !js

package main

import (
	"slices"
	"strings"
	"testing"
)

// -window opens kitty with the pointer hidden, full screen, running this
// game with the other flags, and marked so it does not offer -window again.
func TestWindowOpensKittyWithoutAPointer(t *testing.T) {
	cmd := windowCmd("/usr/bin/kitty", "/bin/marksweep", []string{"-fps=60"})
	want := []string{"/usr/bin/kitty", "-o", "mouse_hide_wait=0.05 1000 1000000 no", "--start-as=fullscreen", "--title", "Mark & Sweep", "/bin/marksweep", "-fps=60"}
	if !slices.Equal(cmd.Args, want) {
		t.Fatalf("args %q\nwant %q", cmd.Args, want)
	}
	if !slices.Contains(cmd.Env, "MARKSWEEP_WINDOW=1") {
		t.Fatal("the window is not marked as the game's own")
	}
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("MARKSWEEP_WINDOW", "1")
	if kittyHint() != "" {
		t.Fatal("the game's own window offers -window")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("MARKSWEEP_WINDOW", "")
	if strings.Contains(kittyHint(), "window") {
		t.Fatal("a terminal that is not kitty is offered -window")
	}
}
