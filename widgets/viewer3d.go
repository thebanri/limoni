package widgets

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// Viewer3D is a high-level widget that renders 3D models with rotation, lighting,
// shading (Wireframe, Solid, Lambertian, Gouraud), and texture mapping.
type Viewer3D struct {
	// ID is the widget focus identifier.
	ID string

	// Model is the 3D geometry to render. Use graphics.NewCube, graphics.NewPyramid,
	// graphics.NewSphere, or graphics.LoadOBJ/LoadSTL/LoadPLY.
	Model graphics.Model3D

	// ImagePath is the optional file path to a PNG/JPEG texture.
	// If specified and Model.Texture is nil, it is automatically loaded and applied.
	ImagePath string

	// Image is an optional in-memory texture image to map onto the 3D model.
	Image image.Image

	// RotX, RotY, RotZ are the Euler rotation angles in degrees.
	RotX float64
	RotY float64
	RotZ float64

	// Distance is the camera distance from the object (default: 3.5).
	Distance float64

	// Scale is the zoom/scale multiplier (default: 1.0).
	Scale float64

	// Shading mode: "Dokulu" (Texture mapped), "Wireframe", "Dolu Renkli" (Flat),
	// "Gölgeli" (Lambertian), "Gouraud" (Smooth interpolated).
	Shading string

	// Wireframe overlays edges on top of shaded faces.
	Wireframe bool

	// WireframeStyle is the cell style for wireframe lines.
	WireframeStyle cell.Style

	// FocusedStyle is applied to wireframe/highlight when the viewer is focused.
	FocusedStyle cell.Style

	// Light is the directional light source for Lambertian and Gouraud shading.
	Light graphics.Light

	// FaceColors is an optional palette for coloring distinct faces in solid mode.
	FaceColors []cell.Color

	canvas *Canvas
}

// Draw renders the 3D model onto the terminal buffer.
func (v *Viewer3D) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 2 || area.Height < 2 {
		return
	}

	if v.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(v.ID)
	}
	if v.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(v.ID)
			}
		})
	}

	// Auto-load texture from ImagePath if provided
	texture := v.Image
	if texture == nil && v.Model.Texture != nil {
		texture = v.Model.Texture
	}
	if texture == nil && v.ImagePath != "" {
		if file, err := os.Open(v.ImagePath); err == nil {
			if img, _, err := image.Decode(file); err == nil {
				texture = img
				v.Image = img
			}
			file.Close()
		}
	}

	dist := v.Distance
	if dist <= 0.1 {
		dist = 3.5
	}
	scaleFactor := v.Scale
	if scaleFactor <= 0 {
		scaleFactor = 1.0
	}

	shading := v.Shading
	if shading == "" {
		if texture != nil {
			shading = "Dokulu"
		} else {
			shading = "Wireframe"
		}
	}

	canvasW := area.Width
	canvasH := area.Height
	if v.canvas == nil {
		v.canvas = NewCanvas(canvasW, canvasH)
	} else {
		v.canvas.Reset(canvasW, canvasH)
	}
	canvas := v.canvas

	virtualW := float64(canvasW) * 2.0
	virtualH := float64(canvasH) * 4.0

	vertices := v.Model.Vertices
	faces := v.Model.Faces
	if len(vertices) == 0 || len(faces) == 0 {
		return
	}

	// Rotate & Project vertices
	rotated := make([]graphics.Vertex3D, len(vertices))
	projected := make([]struct {
		x, y, z float64
		visible bool
	}, len(vertices))

	baseScale := virtualH * 0.40 * scaleFactor

	for i, vert := range vertices {
		rot := vert.RotateY(v.RotY).RotateX(v.RotX).RotateZ(v.RotZ)
		rotated[i] = rot

		px, py, visible := graphics.Project(rot, virtualW, virtualH, dist, baseScale)
		projected[i] = struct {
			x, y, z float64
			visible bool
		}{x: px, y: py, z: rot.Z, visible: visible}
	}

	faceColors := v.FaceColors
	if len(faceColors) == 0 {
		faceColors = []cell.Color{
			cell.NewColorRGB(0, 255, 128),
			cell.NewColorRGB(0, 128, 255),
			cell.NewColorRGB(255, 0, 128),
			cell.NewColorRGB(255, 255, 0),
			cell.NewColorRGB(255, 128, 0),
			cell.NewColorRGB(128, 0, 255),
		}
	}

	light := v.Light
	if light.Direction.X == 0 && light.Direction.Y == 0 && light.Direction.Z == 0 {
		light = graphics.DefaultLight()
	}

	wireStyle := v.WireframeStyle
	if wireStyle == (cell.Style{}) {
		wireStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 128)}
	}

	for faceIdx, face := range faces {
		if len(face) < 3 {
			continue
		}

		p0 := projected[face[0]]
		p1 := projected[face[1]]
		p2 := projected[face[2]]

		if !p0.visible || !p1.visible || !p2.visible {
			continue
		}

		isQuad := len(face) == 4
		var p3 struct {
			x, y, z float64
			visible bool
		}
		if isQuad {
			p3 = projected[face[3]]
			if !p3.visible {
				continue
			}
		}

		// Backface culling: only render front-facing polygons to keep models completely solid & opaque
		cross1 := (p1.x-p0.x)*(p2.y-p0.y) - (p1.y-p0.y)*(p2.x-p0.x)
		isFrontFacing := cross1 < 0
		if isQuad {
			cross2 := (p2.x-p0.x)*(p3.y-p0.y) - (p2.y-p0.y)*(p3.x-p0.x)
			isFrontFacing = isFrontFacing || cross2 < 0
		}
		if !isFrontFacing {
			continue
		}

		col := faceColors[faceIdx%len(faceColors)]
		faceStyle := cell.Style{Fg: col}

		v0, v1, v2 := rotated[face[0]], rotated[face[1]], rotated[face[2]]
		norm0 := graphics.CalculateNormal(v0, v1, v2)

		switch shading {
		case "Dokulu":
			if texture != nil {
				if isQuad {
					uv0 := graphics.UV{U: 0.0, V: 1.0}
					uv1 := graphics.UV{U: 1.0, V: 1.0}
					uv2 := graphics.UV{U: 1.0, V: 0.0}
					uv3 := graphics.UV{U: 0.0, V: 0.0}
					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p1.x, Y: p1.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						uv0, uv1, uv2, texture,
					)
					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						graphics.Vertex2D{X: p3.x, Y: p3.y},
						uv0, uv2, uv3, texture,
					)
				} else {
					uv0 := graphics.UV{U: 0.0, V: 1.0}
					uv1 := graphics.UV{U: 1.0, V: 1.0}
					uv2 := graphics.UV{U: 0.5, V: 0.0}
					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p1.x, Y: p1.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						uv0, uv1, uv2, texture,
					)
				}
			} else {
				if isQuad {
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, faceStyle)
				} else {
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
				}
			}

		case "Dolu Renkli":
			if isQuad {
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, faceStyle)
			} else {
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
			}

		case "Gölgeli":
			if isQuad {
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, norm0, light, faceStyle)
				v3 := rotated[face[3]]
				norm1 := graphics.CalculateNormal(v0, v2, v3)
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, norm1, light, faceStyle)
			} else {
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, norm0, light, faceStyle)
			}

		case "Gouraud":
			n0, n1, n2 := norm0, norm0, norm0
			c0 := graphics.ApplyShade(col, light.CalculateIntensity(n0))
			c1 := graphics.ApplyShade(col, light.CalculateIntensity(n1))
			c2 := graphics.ApplyShade(col, light.CalculateIntensity(n2))
			if isQuad {
				n3 := norm0
				c3 := graphics.ApplyShade(col, light.CalculateIntensity(n3))
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, c0, c1, c2, cell.Style{})
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, c0, c2, c3, cell.Style{})
			} else {
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, c0, c1, c2, cell.Style{})
			}
		}

		if shading == "Wireframe" || v.Wireframe {
			canvas.DrawLine(int(p0.x), int(p0.y), int(p1.x), int(p1.y), wireStyle)
			canvas.DrawLine(int(p1.x), int(p1.y), int(p2.x), int(p2.y), wireStyle)
			if isQuad {
				canvas.DrawLine(int(p2.x), int(p2.y), int(p3.x), int(p3.y), wireStyle)
				canvas.DrawLine(int(p3.x), int(p3.y), int(p0.x), int(p0.y), wireStyle)
			} else {
				canvas.DrawLine(int(p2.x), int(p2.y), int(p0.x), int(p0.y), wireStyle)
			}
		}
	}

	canvas.Draw(ctx, buf)
}
