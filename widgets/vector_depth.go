package widgets

import (
	"math"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// DrawFilledTriangleDepth rasterizes a triangle with interpolated z-buffering.
func (c *Canvas) DrawFilledTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, style cell.Style) {
	minX := int(math.Min(p0.X, math.Min(p1.X, p2.X)))
	maxX := int(math.Max(p0.X, math.Max(p1.X, p2.X)))
	minY := int(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))
	maxY := int(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))
	canvasW, canvasH := int(c.width)*2, int(c.height)*4
	if minX < 0 {
		minX = 0
	}
	if maxX >= canvasW {
		maxX = canvasW - 1
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= canvasH {
		maxY = canvasH - 1
	}
	denom := (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
	if math.Abs(denom) < 1e-6 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx, fy := float64(x), float64(y)
			l1 := ((p1.Y-p2.Y)*(fx-p2.X) + (p2.X-p1.X)*(fy-p2.Y)) / denom
			l2 := ((p2.Y-p0.Y)*(fx-p2.X) + (p0.X-p2.X)*(fy-p2.Y)) / denom
			l3 := 1 - l1 - l2
			if l1 >= -0.005 && l2 >= -0.005 && l3 >= -0.005 {
				c.SetDepth(x, y, l1*z0+l2*z1+l3*z2, style)
			}
		}
	}
}

// DrawLambertTriangleDepth rasterizes a triangle with Lambertian diffuse shading and z-buffering.
func (c *Canvas) DrawLambertTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, normal graphics.Vector3D, light graphics.Light, baseStyle cell.Style) {
	intensity := light.CalculateIntensity(normal)
	shadedFg := graphics.ApplyShade(baseStyle.Fg, intensity)
	shadedStyle := baseStyle
	shadedStyle.Fg = shadedFg
	c.DrawFilledTriangleDepth(p0, p1, p2, z0, z1, z2, shadedStyle)
}

// DrawGouraudTriangleDepth rasterizes a triangle with per-vertex color interpolation (Gouraud shading) and z-buffering.
func (c *Canvas) DrawGouraudTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, c0, c1, c2 cell.Color, baseStyle cell.Style) {
	minX := int(math.Min(p0.X, math.Min(p1.X, p2.X)))
	maxX := int(math.Max(p0.X, math.Max(p1.X, p2.X)))
	minY := int(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))
	maxY := int(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))
	canvasW, canvasH := int(c.width)*2, int(c.height)*4
	if minX < 0 {
		minX = 0
	}
	if maxX >= canvasW {
		maxX = canvasW - 1
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= canvasH {
		maxY = canvasH - 1
	}
	denom := (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
	if math.Abs(denom) < 1e-6 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx, fy := float64(x), float64(y)
			l1 := ((p1.Y-p2.Y)*(fx-p2.X) + (p2.X-p1.X)*(fy-p2.Y)) / denom
			l2 := ((p2.Y-p0.Y)*(fx-p2.X) + (p0.X-p2.X)*(fy-p2.Y)) / denom
			l3 := 1 - l1 - l2
			if l1 >= -0.005 && l2 >= -0.005 && l3 >= -0.005 {
				pixelColor := graphics.BarycentricColor(c0, c1, c2, l1, l2, l3)
				pixelStyle := baseStyle
				pixelStyle.Fg = pixelColor
				c.SetDepth(x, y, l1*z0+l2*z1+l3*z2, pixelStyle)
			}
		}
	}
}
