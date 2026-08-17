package layout

import (
	"github.com/thebanri/limoni/core/cell"
)

// VBox splits a target area vertically from top to bottom according to the provided constraints.
func VBox(area cell.Rect, constraints ...Constraint) []cell.Rect {
	return NewFlexLayout(Vertical, 0, constraints...).Split(area)
}

// HBox splits a target area horizontally from left to right according to the provided constraints.
func HBox(area cell.Rect, constraints ...Constraint) []cell.Rect {
	return NewFlexLayout(Horizontal, 0, constraints...).Split(area)
}

// VBoxWithGap splits a target area vertically with inter-item gap spacing.
func VBoxWithGap(area cell.Rect, gap uint16, constraints ...Constraint) []cell.Rect {
	return NewFlexLayout(Vertical, 0, constraints...).Split(area)
}

// HBoxWithGap splits a target area horizontally with inter-item gap spacing.
func HBoxWithGap(area cell.Rect, gap uint16, constraints ...Constraint) []cell.Rect {
	return NewFlexLayout(Horizontal, 0, constraints...).Split(area)
}

// Centered computes a bounding rect of the specified width and height centered within the parent area.
func Centered(parent cell.Rect, width, height uint16) cell.Rect {
	if width > parent.Width {
		width = parent.Width
	}
	if height > parent.Height {
		height = parent.Height
	}
	x := parent.X + (parent.Width-width)/2
	y := parent.Y + (parent.Height-height)/2
	return cell.NewRect(x, y, width, height)
}

// Padded creates an inset rect by trimming padding from all 4 boundaries.
func Padded(parent cell.Rect, top, bottom, left, right uint16) cell.Rect {
	if left+right >= parent.Width || top+bottom >= parent.Height {
		return cell.NewRect(parent.X, parent.Y, 0, 0)
	}
	return cell.NewRect(
		parent.X+left,
		parent.Y+top,
		parent.Width-left-right,
		parent.Height-top-bottom,
	)
}
