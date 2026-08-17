package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

func TestCanvasDepthKeepsNearestPixel(t *testing.T) {
	canvas := NewCanvas(2, 1)
	far := cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
	near := cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}
	if !canvas.SetDepth(0, 0, 2, far) {
		t.Fatal("first depth write should succeed")
	}
	if canvas.SetDepth(0, 0, 3, far) {
		t.Fatal("farther pixel should be rejected")
	}
	if !canvas.SetDepth(0, 0, 1, near) {
		t.Fatal("nearer pixel should replace farther pixel")
	}
}

func TestCanvasLambertAndGouraudShading(t *testing.T) {
	canvas := NewCanvas(10, 10)
	p0 := graphics.Vertex2D{X: 0, Y: 0}
	p1 := graphics.Vertex2D{X: 8, Y: 0}
	p2 := graphics.Vertex2D{X: 4, Y: 8}

	normal := graphics.Vector3D{X: 0, Y: 0, Z: -1}
	light := graphics.DefaultLight()
	baseStyle := cell.Style{Fg: cell.NewColorRGB(200, 100, 50)}

	// Test Lambertian rasterization
	canvas.DrawLambertTriangleDepth(p0, p1, p2, 1.0, 1.0, 1.0, normal, light, baseStyle)
	if canvas.grid[0] == 0 {
		t.Fatal("expected pixels drawn by Lambert rasterizer")
	}

	// Test Gouraud rasterization
	canvas.Reset(10, 10)
	c0 := cell.NewColorRGB(255, 0, 0)
	c1 := cell.NewColorRGB(0, 255, 0)
	c2 := cell.NewColorRGB(0, 0, 255)
	canvas.DrawGouraudTriangleDepth(p0, p1, p2, 1.0, 1.0, 1.0, c0, c1, c2, cell.Style{})
	if canvas.grid[0] == 0 {
		t.Fatal("expected pixels drawn by Gouraud rasterizer")
	}
}
