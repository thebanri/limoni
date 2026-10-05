//go:build !(linux || freebsd || openbsd || netbsd || dragonfly)

package main

import "errors"

// Following other players needs MPRIS, which only desktops on D-Bus have.
// macOS and Windows play files: pikap ~/Music.
type mprisSource struct{ source }

func newMPRIS() (*mprisSource, error) {
	return nil, errors.New("following other players needs MPRIS (Linux, the BSDs); give pikap files or a folder to play instead")
}
