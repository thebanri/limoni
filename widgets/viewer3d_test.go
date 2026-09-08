package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

func TestViewer3D(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 40, 20))
	ctx := cell.NewContext(cell.NewRect(0, 0, 40, 20), cell.Style{})

	viewer := &Viewer3D{
		Model:     graphics.NewCube(2.0),
		RotX:      30,
		RotY:      45,
		Distance:  3.5,
		Scale:     1.0,
		Shading:   "Wireframe",
		Wireframe: true,
	}

	viewer.Draw(ctx, buf)

	// Solid Shading
	viewer.Shading = "Dolu Renkli"
	viewer.Draw(ctx, buf)

	// Lambertian Shading
	viewer.Shading = "Gölgeli"
	viewer.Draw(ctx, buf)

	// Gouraud Shading
	viewer.Shading = "Gouraud"
	viewer.Draw(ctx, buf)

	// Sphere
	viewer.Model = graphics.NewSphere(1.0, 8, 8)
	viewer.Draw(ctx, buf)

	// Pyramid
	viewer.Model = graphics.NewPyramid(2.0, 2.0)
	viewer.Draw(ctx, buf)

	// Malformed model with out-of-bounds face vertex index
	viewer.Model = graphics.Model3D{
		Name: "Corrupt",
		Vertices: []graphics.Vertex3D{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
		},
		Faces: [][]int{
			{0, 1, 999}, // Out of bounds!
		},
	}
	// Must not panic
	viewer.Draw(ctx, buf)

	// DrawTexturedTriangle nil image safety
	canvas := NewCanvas(10, 10)
	canvas.DrawTexturedTriangle(
		graphics.Vertex2D{X: 0, Y: 0},
		graphics.Vertex2D{X: 5, Y: 0},
		graphics.Vertex2D{X: 0, Y: 5},
		graphics.UV{}, graphics.UV{}, graphics.UV{},
		nil,
	)
}
