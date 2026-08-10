package cell

// Point identifies a terminal cell coordinate.
type Point struct {
	X uint16
	Y uint16
}

// NewPoint creates a terminal cell coordinate.
func NewPoint(x, y uint16) Point { return Point{X: x, Y: y} }
