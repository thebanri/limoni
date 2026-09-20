package graphics

import (
	"math"
)

// Vertex3D is a point in three-dimensional space.
type Vertex3D struct {
	X, Y, Z float64
}

// Vertex2D is a point on the two-dimensional screen plane.
type Vertex2D struct {
	X, Y float64
}

// UV is a texture coordinate.
type UV struct {
	U, V float64
}

// RotateX rotates the point around the X axis. The angle is in degrees.
func (v Vertex3D) RotateX(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X,
		Y: v.Y*cos - v.Z*sin,
		Z: v.Y*sin + v.Z*cos,
	}
}

// RotateY rotates the point around the Y axis. The angle is in degrees.
func (v Vertex3D) RotateY(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// RotateZ rotates the point around the Z axis. The angle is in degrees.
func (v Vertex3D) RotateZ(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
		Z: v.Z,
	}
}

// Project maps a 3D point onto the screen with a perspective projection.
// distance is how far the camera sits from the object, scale the resulting
// size multiplier on screen.
func Project(v Vertex3D, screenW, screenH, distance, scale float64) (x, y float64, visible bool) {
	// Behind the camera: nothing to draw.
	if v.Z+distance <= 0.1 {
		return 0, 0, false
	}

	factor := scale / (v.Z + distance)
	x = screenW/2.0 + v.X*factor
	// Screen coordinates grow downwards, so Y is flipped.
	y = screenH/2.0 - v.Y*factor

	return x, y, true
}
