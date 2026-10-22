//go:build !js && !linux

package main

// Outside Linux a terminal has no pointer lock: the mouse's movement turns
// the view (game.mouse), and at the window's edge the view keeps turning
// (edgeTurn).
type pointer struct{}

func newPointer() *pointer   { return &pointer{} }
func (*pointer) drain(*game) {}
func (*pointer) want(*game)  {}
func (*pointer) close()      {}

// problem is a line for the title screen: in kitty, -window hides the
// pointer at least.
func (*pointer) problem() string { return kittyHint() }
