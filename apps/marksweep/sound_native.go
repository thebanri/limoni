//go:build !js

package main

// newBrowserMixer has nothing to find outside a browser; see sound_js.go.
func newBrowserMixer() *mixer { return nil }
