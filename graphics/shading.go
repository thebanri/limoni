package graphics

import (
	"math"

	"github.com/thebanri/limoni/core/cell"
)

// Vector3D represents a 3D direction or point for lighting calculations.
type Vector3D struct {
	X, Y, Z float64
}

// Normalize scales the vector to unit length (length = 1.0).
func (v Vector3D) Normalize() Vector3D {
	len := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if len < 1e-9 {
		return Vector3D{0, 0, 0}
	}
	return Vector3D{v.X / len, v.Y / len, v.Z / len}
}

// Dot computes the dot product of two 3D vectors.
func (v Vector3D) Dot(other Vector3D) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross computes the cross product of two 3D vectors.
func (v Vector3D) Cross(other Vector3D) Vector3D {
	return Vector3D{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// CalculateNormal computes the normalized surface normal for a triangle (v0, v1, v2).
func CalculateNormal(v0, v1, v2 Vertex3D) Vector3D {
	edge1 := Vector3D{v1.X - v0.X, v1.Y - v0.Y, v1.Z - v0.Z}
	edge2 := Vector3D{v2.X - v0.X, v2.Y - v0.Y, v2.Z - v0.Z}
	normal := edge1.Cross(edge2)
	return normal.Normalize()
}

// Light represents a directional light source in 3D space.
type Light struct {
	Direction Vector3D // Normalized direction towards the light source
	Ambient   float64  // Base ambient light level [0..1]
	Diffuse   float64  // Diffuse light intensity multiplier [0..1]
}

// DefaultLight returns a balanced directional light coming from upper-right-front.
func DefaultLight() Light {
	dir := Vector3D{X: 0.5, Y: 0.8, Z: -0.7}.Normalize()
	return Light{
		Direction: dir,
		Ambient:   0.25,
		Diffuse:   0.75,
	}
}

// CalculateIntensity computes the Lambertian diffuse lighting intensity for a surface normal.
// Result is clamped to [0.0, 1.0].
func (l Light) CalculateIntensity(normal Vector3D) float64 {
	dot := normal.Dot(l.Direction)
	if dot < 0 {
		dot = 0
	}
	intensity := l.Ambient + l.Diffuse*dot
	if intensity > 1.0 {
		intensity = 1.0
	}
	if intensity < 0.0 {
		intensity = 0.0
	}
	return intensity
}

// ApplyShade modulates an RGB Color by a lighting intensity [0.0, 1.0].
func ApplyShade(c cell.Color, intensity float64) cell.Color {
	r, g, b := c.RGB()
	sr := uint8(math.Round(float64(r) * intensity))
	sg := uint8(math.Round(float64(g) * intensity))
	sb := uint8(math.Round(float64(b) * intensity))
	return cell.NewColorRGB(sr, sg, sb)
}

// InterpolateColor blends two RGB colors by factor t in [0.0, 1.0].
func InterpolateColor(c1, c2 cell.Color, t float64) cell.Color {
	if t <= 0 {
		return c1
	}
	if t >= 1 {
		return c2
	}
	r1, g1, b1 := c1.RGB()
	r2, g2, b2 := c2.RGB()

	r := uint8(float64(r1)*(1-t) + float64(r2)*t)
	g := uint8(float64(g1)*(1-t) + float64(g2)*t)
	b := uint8(float64(b1)*(1-t) + float64(b2)*t)
	return cell.NewColorRGB(r, g, b)
}

// BarycentricColor interpolates 3 colors using barycentric coordinates (l1, l2, l3).
func BarycentricColor(c0, c1, c2 cell.Color, l1, l2, l3 float64) cell.Color {
	r0, g0, b0 := c0.RGB()
	r1, g1, b1 := c1.RGB()
	r2, g2, b2 := c2.RGB()

	r := uint8(math.Min(255, math.Max(0, float64(r0)*l1+float64(r1)*l2+float64(r2)*l3)))
	g := uint8(math.Min(255, math.Max(0, float64(g0)*l1+float64(g1)*l2+float64(g2)*l3)))
	b := uint8(math.Min(255, math.Max(0, float64(b0)*l1+float64(b1)*l2+float64(b2)*l3)))
	return cell.NewColorRGB(r, g, b)
}

// IsBackface checks if a projected 2D triangle is facing away from the camera.
// In screen space (Y downwards), a clockwise winding has cross product > 0 (front-facing),
// and counter-clockwise has cross product <= 0 (back-facing).
func IsBackface(p0, p1, p2 Vertex2D) bool {
	return ((p1.X-p0.X)*(p2.Y-p0.Y) - (p1.Y-p0.Y)*(p2.X-p0.X)) <= 0
}
