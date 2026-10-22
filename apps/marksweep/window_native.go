//go:build !js

package main

import (
	"os"
	"os/exec"
	"strings"
)

// No program running in a terminal can hide the mouse pointer: kitty has no
// escape for it, and on Wayland nothing inside the terminal may lock it.
// kitty's own mouse_hide_wait can, though: hidden once the mouse rests for
// 50 ms, and shown again only after a billion mouse events. -window opens
// the game in a kitty window of its own, full screen, with that set.
const hiddenPointer = "mouse_hide_wait=0.05 1000 1000000 no"

// inOwnWindow reports whether this is the game -window opened.
func inOwnWindow() bool { return os.Getenv("MARKSWEEP_WINDOW") == "1" }

// kittyHint is the line the title screen shows in kitty, where -window
// would hide the pointer.
func kittyHint() string {
	if inOwnWindow() || !strings.Contains(os.Getenv("TERM"), "kitty") {
		return ""
	}
	if _, err := exec.LookPath("kitty"); err != nil {
		return ""
	}
	return "marksweep -window: its own kitty window, full screen, no mouse pointer"
}

// openWindow starts the game again in a kitty window of its own, with the
// rest of its flags, and returns once that window is open.
func openWindow(args []string) error {
	kitty, err := exec.LookPath("kitty")
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return windowCmd(kitty, exe, args).Start()
}

func windowCmd(kitty, exe string, args []string) *exec.Cmd {
	cmd := exec.Command(kitty, append([]string{"-o", hiddenPointer, "--start-as=fullscreen", "--title", "Mark & Sweep", exe}, args...)...)
	cmd.Env = append(os.Environ(), "MARKSWEEP_WINDOW=1")
	return cmd
}
