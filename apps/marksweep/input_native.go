//go:build !js

package main

// A terminal has no pointer lock: the mouse's movement turns the view
// (game.mouse), and at the window's edge the view keeps turning (edgeTurn).
type pointer struct{}

func newPointer() *pointer   { return &pointer{} }
func (*pointer) drain(*game) {}
func (*pointer) want(bool)   {}
