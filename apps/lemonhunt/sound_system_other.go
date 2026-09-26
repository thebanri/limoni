//go:build !darwin && !windows

package main

// Other platforms use the external players or Web Audio in sound_js.go.
func newSystemMixer() *mixer { return nil }
