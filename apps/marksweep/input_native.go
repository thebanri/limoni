//go:build !js

package main

// A terminal has no pointer lock: the crosshair follows the mouse, and the
// view turns near the edges (edgeTurn).
type pointer struct{}

func newPointer() *pointer   { return &pointer{} }
func (*pointer) drain(*game) {}
func (*pointer) want(bool)   {}
