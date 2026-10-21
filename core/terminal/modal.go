package terminal

import "github.com/thebanri/limoni/core/cell"

// LayerType is the kind of a layer.
type LayerType uint8

const (
	// LayerModal is a modal window layer that traps the focus.
	LayerModal LayerType = iota
	// LayerPopup is a light layer, such as a dropdown menu. It does not trap the focus.
	LayerPopup
)

// Layer is one overlapping drawing layer in the layered rendering system.
// Each layer filters the events inside its area and keeps them from leaking to the layers below.
// Layers with a larger Z-Index sit on top (the largest z-index is topmost).
type Layer struct {
	// ID uniquely identifies the layer.
	ID string
	// Type is the kind of layer (Modal or Popup).
	Type LayerType
	// Area is the part of the screen the layer covers.
	Area cell.Rect
	// ClickOutside is called when a click lands outside the modal's area.
	ClickOutside func()
	// ZIndex sets the layer's drawing order. Larger is higher.
	ZIndex int
}

// ContainsRect reports whether child lies entirely within parent.
func ContainsRect(parent, child cell.Rect) bool {
	return child.X >= parent.X &&
		child.Y >= parent.Y &&
		int(child.X)+int(child.Width) <= int(parent.X)+int(parent.Width) &&
		int(child.Y)+int(child.Height) <= int(parent.Y)+int(parent.Height)
}

// Intersects reports whether two rectangles intersect.
func Intersects(r1, r2 cell.Rect) bool {
	return r1.X < r2.X+r2.Width &&
		r2.X < r1.X+r1.Width &&
		r1.Y < r2.Y+r2.Height &&
		r2.Y < r1.Y+r1.Height
}

// CenterRect returns a rectangle of the given width and height centred in parent.
func CenterRect(parent cell.Rect, w, h uint16) cell.Rect {
	if w > parent.Width {
		w = parent.Width
	}
	if h > parent.Height {
		h = parent.Height
	}

	x := parent.X + (parent.Width-w)/2
	y := parent.Y + (parent.Height-h)/2

	return cell.NewRect(x, y, w, h)
}

// ScaleRect scales a rectangle by the given progress (0.0 -> 1.0),
// keeping its centre where it is.
func ScaleRect(base cell.Rect, progress float64) cell.Rect {
	if progress <= 0 {
		return cell.NewRect(base.X+base.Width/2, base.Y+base.Height/2, 0, 0)
	}
	if progress >= 1.0 {
		return base
	}

	w := uint16(float64(base.Width) * progress)
	h := uint16(float64(base.Height) * progress)

	// Round w and h to even numbers to avoid jitter and sub-pixel alignment drift
	if w%2 != 0 && w < base.Width {
		w++
	}
	if h%2 != 0 && h < base.Height {
		h++
	}

	x := base.X + (base.Width-w)/2
	y := base.Y + (base.Height-h)/2

	return cell.NewRect(x, y, w, h)
}

// SlideUpRect slides a rectangle smoothly from below the bottom edge (off screen) up to its
// target row (Y), by the given progress (0.0 -> 1.0).
func SlideUpRect(base cell.Rect, parentHeight uint16, progress float64) cell.Rect {
	if progress <= 0 {
		return cell.NewRect(base.X, parentHeight, base.Width, base.Height)
	}
	if progress >= 1.0 {
		return base
	}

	startY := parentHeight
	y := startY - uint16(float64(startY-base.Y)*progress)

	return cell.NewRect(base.X, y, base.Width, base.Height)
}
