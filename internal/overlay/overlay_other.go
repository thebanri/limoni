//go:build !linux

package overlay

import (
	"errors"
	"image/color"
)

// Overlay is only implemented on Linux.
type Overlay struct{}

// Open fails: there is no overlay on this platform yet.
func Open() (*Overlay, error) { return nil, errors.New("overlay: not supported on this platform") }

// Show does nothing.
func (o *Overlay) Show([]Picture, int, int) error { return nil }

// CellSize knows nothing.
func (o *Overlay) CellSize(int, int) (int, int) { return 0, 0 }

// SetCache does nothing.
func (o *Overlay) SetCache(int) {}

// SetBackground does nothing.
func (o *Overlay) SetBackground(color.Color) {}

// Trim does nothing.
func (o *Overlay) Trim() error { return nil }

// Close does nothing.
func (o *Overlay) Close() error { return nil }

// Anchor has nothing to read.
func (o *Overlay) Anchor(int) bool { return false }

// Follow does nothing.
func (o *Overlay) Follow() error { return nil }

// Moved is always false.
func (o *Overlay) Moved() bool { return false }

// Hide does nothing.
func (o *Overlay) Hide() error { return nil }

// Reset does nothing.
func (o *Overlay) Reset() error { return nil }
