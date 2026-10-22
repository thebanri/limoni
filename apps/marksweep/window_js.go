//go:build js

package main

import "errors"

// The browser locks the pointer itself (input_js.go).

func kittyHint() string { return "" }

func openWindow([]string) error { return errors.New("-window is for kitty") }
