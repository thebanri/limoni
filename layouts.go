package limoni

import (
	"github.com/thebanri/limoni/layout"
)

// SplitHorizontal splits an area horizontally according to constraints.
func SplitHorizontal(area Rect, constraints ...Constraint) []Rect {
	return layout.NewFlexLayout(layout.Horizontal, 0, constraints...).Split(area)
}

// SplitVertical splits an area vertically according to constraints.
func SplitVertical(area Rect, constraints ...Constraint) []Rect {
	return layout.NewFlexLayout(layout.Vertical, 0, constraints...).Split(area)
}

// Percentage creates a percentage-based layout constraint (0-100).
func Percentage(p uint16) Constraint {
	return layout.Percentage(p)
}

// Fixed creates a fixed-size layout constraint.
func Fixed(n uint16) Constraint {
	return layout.Fixed(n)
}

// Fill creates a constraint that expands to take all remaining available space.
func Fill() Constraint {
	return layout.Fill()
}

// Ratio creates a ratio-based layout constraint.
func Ratio(val uint16) Constraint {
	return layout.Ratio(val)
}

// Min creates a constraint that specifies a minimum size.
func Min(m uint16) Constraint {
	return layout.Min(m)
}

// Max creates a constraint that specifies a maximum size.
func Max(m uint16) Constraint {
	return layout.Max(m)
}
