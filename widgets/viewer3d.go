package widgets

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"reflect"

	"github.com/thebanri/limoni/core/accessibility"
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

	// Shading is one of ShadingTexture, ShadingWireframe, ShadingFlat,
	// ShadingLambert or ShadingGouraud. Empty picks Texture when there is a
	// texture and Wireframe otherwise. The original Turkish names "Dokulu",
	// "Dolu Renkli" and "Gölgeli" are still accepted.
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

	// Marker picks the characters the model is drawn with (Braille by
	// default). MarkerSextant and MarkerQuadrant draw filled faces solid, at
	// a lower resolution, in fonts without Braille.
	Marker Marker

	canvas *Canvas

	// Pixels renders the model as a picture, sent with the terminal's image
	// protocol (kitty, iTerm2, Sixel), instead of as Braille dots: full
	// resolution and colour per pixel. Where the terminal has no image
	// protocol it draws dots as usual. Encoding a picture every frame the
	// model moves costs what the protocol costs — a PNG or Sixel encode and
	// hundreds of kilobytes of allocation — so it is opt-in; a still model
	// is encoded once.
	Pixels bool

	pixels     [2]pixelTarget
	frame      int
	lastKey    viewerKey
	proto      graphics.Protocol
	protoKnown bool

	// Per-vertex scratch, kept between frames so Draw does not allocate once
	// the slices have grown to the model's size.
	rotated   []graphics.Vertex3D
	projected []graphics.Vertex2D
	normals   []graphics.Vector3D
	shades    []float64
}

var defaultViewerFaceColors = []cell.Color{
	cell.NewColorRGB(0, 255, 128),
	cell.NewColorRGB(0, 128, 255),
	cell.NewColorRGB(255, 0, 128),
	cell.NewColorRGB(255, 255, 0),
	cell.NewColorRGB(255, 128, 0),
	cell.NewColorRGB(128, 0, 255),
}

// viewerNear is the near clipping plane, as a distance from the camera.
// graphics.Project refuses anything at 0.1 or closer; triangles crossing the
// plane are cut at it rather than dropped, so a model the camera moves into
// is sliced open instead of losing whole faces.
const viewerNear = 0.11

// clipVertex is a vertex in camera space with the attributes interpolated
// when a triangle is cut at the near plane.
type clipVertex struct {
	pos   graphics.Vertex3D
	uv    graphics.UV
	shade float64
}

func lerpClipVertex(a, b clipVertex, t float64) clipVertex {
	return clipVertex{
		pos: graphics.Vertex3D{
			X: a.pos.X + (b.pos.X-a.pos.X)*t,
			Y: a.pos.Y + (b.pos.Y-a.pos.Y)*t,
			Z: a.pos.Z + (b.pos.Z-a.pos.Z)*t,
		},
		uv:    graphics.UV{U: a.uv.U + (b.uv.U-a.uv.U)*t, V: a.uv.V + (b.uv.V-a.uv.V)*t},
		shade: a.shade + (b.shade-a.shade)*t,
	}
}

// clipNear cuts a triangle at the near plane (Sutherland–Hodgman against a
// single plane) and returns how many of out's vertices form the result: 0
// when the triangle is entirely behind it, 3 or 4 otherwise.
func clipNear(in *[3]clipVertex, dist float64, out *[4]clipVertex) int {
	n := 0
	for i := 0; i < 3; i++ {
		a, b := in[i], in[(i+1)%3]
		da, db := a.pos.Z+dist-viewerNear, b.pos.Z+dist-viewerNear
		if da >= 0 {
			out[n] = a
			n++
		}
		if (da >= 0) != (db >= 0) {
			out[n] = lerpClipVertex(a, b, da/(da-db))
			n++
		}
	}
	return n
}

// placeholderUV is the texture coordinate for corner i of a face with no UVs
// of its own: the image stretched across the face.
func placeholderUV(i, corners int) graphics.UV {
	if corners == 3 {
		return [3]graphics.UV{{U: 0, V: 1}, {U: 1, V: 1}, {U: 0.5, V: 0}}[i]
	}
	return [4]graphics.UV{{U: 0, V: 1}, {U: 1, V: 1}, {U: 1, V: 0}, {U: 0, V: 0}}[i%4]
}

// Draw renders the 3D model onto the terminal buffer.
//
// Faces of any size are fanned into triangles, cut at the near plane,
// back-face culled and depth tested, in every shading mode. Gouraud shading
// uses vertex normals averaged over the faces that share each vertex.
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

	shading := canonicalShading(v.Shading)
	if shading == "" {
		if texture != nil {
			shading = ShadingTexture
		} else {
			shading = ShadingWireframe
		}
	}

	pixels := v.Pixels && ctx.RegisterImage != nil && v.imageProtocol() != graphics.ProtocolHalfBlock
	var target raster3D
	var virtualW, virtualH float64
	redraw := true
	if pixels {
		key := v.viewKey(area, dist, scaleFactor, shading, texture)
		if key == v.lastKey && v.pixels[v.frame%2].img != nil {
			// Nothing that shows has changed: send the same picture, which
			// the terminal already has, instead of encoding a new one.
			redraw = false
		} else {
			v.lastKey = key
			pw, ph := int(area.Width)*viewerCellPixelsW, int(area.Height)*viewerCellPixelsH
			// Two buffers in turn, so each new picture is a different image
			// to the terminal; the one reused was sent two frames ago and
			// its cached encoding is stale.
			v.frame++
			pt := &v.pixels[v.frame%2]
			if pt.img != nil {
				graphics.ForgetImage(pt.img)
			}
			pt.reset(pw, ph, color.RGBA{})
			target, virtualW, virtualH = pt, float64(pw), float64(ph)
		}
	} else {
		if v.canvas == nil {
			v.canvas = NewCanvas(area.Width, area.Height)
		} else {
			v.canvas.Reset(area.Width, area.Height)
		}
		v.canvas.Marker = v.Marker
		target, virtualW, virtualH = v.canvas, float64(area.Width)*2, float64(area.Height)*4
	}
	baseScale := virtualH * 0.40 * scaleFactor
	if redraw && len(v.Model.Vertices) > 0 && len(v.Model.Faces) > 0 {
		v.rasterize(target, virtualW, virtualH, baseScale, dist, shading, texture)
	}

	if !pixels {
		v.canvas.Draw(ctx, buf)
		return
	}
	img := v.pixels[v.frame%2].img
	if !ctx.RegisterImage(area, img, 0, true) {
		return
	}
	// The picture covers the area: mark the cells so no text is drawn
	// through it, keeping the background the picture's transparency shows.
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = cell.RuneImage
				c.Style.Reset()
				c.Style.Bg = ctx.Style.Bg
			}
		}
	}
}

// raster3D is what Viewer3D draws on: a Canvas of Braille dots, or a
// pixelTarget sent to the terminal as a picture.
type raster3D interface {
	DrawFilledTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, style cell.Style)
	DrawLambertTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, normal graphics.Vector3D, light graphics.Light, baseStyle cell.Style)
	DrawGouraudTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, c0, c1, c2 cell.Color, baseStyle cell.Style)
	DrawTexturedTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, uv0, uv1, uv2 graphics.UV, img image.Image)
	DrawLine(x1, y1, x2, y2 int, style cell.Style)
}

// Pixels per terminal cell when the model is rendered as a picture. The
// terminal scales the picture to the area, so this sets the detail, not the
// size; 8×16 is the common cell and keeps the aspect of the Braille grid.
const (
	viewerCellPixelsW = 8
	viewerCellPixelsH = 16
)

// viewerKey is everything that decides what a picture shows. The model is
// compared by its slices, not their contents: changing vertices in place
// needs a new slice, or a change to any other field, to be redrawn.
type viewerKey struct {
	area              cell.Rect
	rotX, rotY, rotZ  float64
	dist, scale       float64
	shading           string
	vertices          *graphics.Vertex3D
	faces             *[]int
	nVertices, nFaces int
	uvs               *graphics.UV
	texture           uintptr
	faceColors        *cell.Color
	light             graphics.Light
	wireframe         bool
	wireStyle         cell.Style
}

func (v *Viewer3D) viewKey(area cell.Rect, dist, scale float64, shading string, texture image.Image) viewerKey {
	k := viewerKey{
		area: area, rotX: v.RotX, rotY: v.RotY, rotZ: v.RotZ, dist: dist, scale: scale, shading: shading,
		nVertices: len(v.Model.Vertices), nFaces: len(v.Model.Faces), texture: imageIdentity(texture),
		light: v.Light, wireframe: v.Wireframe, wireStyle: v.WireframeStyle,
	}
	if len(v.Model.Vertices) > 0 {
		k.vertices = &v.Model.Vertices[0]
	}
	if len(v.Model.Faces) > 0 {
		k.faces = &v.Model.Faces[0]
	}
	if len(v.Model.UVs) > 0 {
		k.uvs = &v.Model.UVs[0]
	}
	if len(v.FaceColors) > 0 {
		k.faceColors = &v.FaceColors[0]
	}
	return k
}

// imageIdentity is the address behind an image, which is what the key
// compares: == on an image.Image panics when its dynamic type holds a slice.
func imageIdentity(img image.Image) uintptr {
	if img == nil {
		return 0
	}
	if rv := reflect.ValueOf(img); rv.Kind() == reflect.Pointer {
		return rv.Pointer()
	}
	return 1 // a value type: treat every frame's as the same picture
}

func (v *Viewer3D) imageProtocol() graphics.Protocol {
	if !v.protoKnown {
		v.proto, v.protoKnown = graphics.DetectProtocol(), true
	}
	return v.proto
}

// rasterize projects, clips, culls and shades the model onto t, whose
// drawing space is w×h dots.
func (v *Viewer3D) rasterize(t raster3D, virtualW, virtualH, baseScale, dist float64, shading string, texture image.Image) {
	model := &v.Model
	if cap(v.rotated) < len(model.Vertices) {
		v.rotated = make([]graphics.Vertex3D, len(model.Vertices))
	}
	if cap(v.projected) < len(model.Vertices) {
		v.projected = make([]graphics.Vertex2D, len(model.Vertices))
	}
	rotated := v.rotated[:len(model.Vertices)]
	projected := v.projected[:len(model.Vertices)]
	rot := newEulerRotation(v.RotX, v.RotY, v.RotZ)
	for i, vert := range model.Vertices {
		r := rot.apply(vert)
		rotated[i] = r
		// Vertices behind the near plane are only reached through clipNear.
		if r.Z+dist >= viewerNear {
			x, y, _ := graphics.Project(r, virtualW, virtualH, dist, baseScale)
			projected[i] = graphics.Vertex2D{X: x, Y: y}
		}
	}
	inFront := func(i int) bool { return rotated[i].Z+dist >= viewerNear }

	faceColors := v.FaceColors
	if len(faceColors) == 0 {
		faceColors = defaultViewerFaceColors
	}

	light := v.Light
	if light.Direction.X == 0 && light.Direction.Y == 0 && light.Direction.Z == 0 {
		light = graphics.DefaultLight()
	}

	wireStyle := v.WireframeStyle
	if wireStyle == (cell.Style{}) {
		wireStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 128)}
	}

	var shades []float64
	if shading == ShadingGouraud {
		shades = v.vertexShades(rotated, light)
	}

	var tri [3]clipVertex
	var clipped [4]clipVertex
	var screen [4]graphics.Vertex2D

	for faceIdx, face := range model.Faces {
		if len(face) < 3 || !facesInRange(face, len(rotated)) {
			continue
		}

		col := faceColors[faceIdx%len(faceColors)]
		faceStyle := cell.Style{Fg: col}

		var faceUVs []int
		if faceIdx < len(model.FaceUVs) && len(model.FaceUVs[faceIdx]) == len(face) {
			faceUVs = model.FaceUVs[faceIdx]
		}
		corner := func(i int) clipVertex {
			cv := clipVertex{pos: rotated[face[i]], uv: placeholderUV(i, len(face))}
			if faceUVs != nil && faceUVs[i] >= 0 && faceUVs[i] < len(model.UVs) {
				cv.uv = model.UVs[faceUVs[i]]
			}
			if shades != nil {
				cv.shade = shades[face[i]]
			}
			return cv
		}

		anyFront := false
		for k := 1; k+1 < len(face); k++ {
			i0, i1, i2 := face[0], face[k], face[k+1]
			var n int
			if inFront(i0) && inFront(i1) && inFront(i2) {
				screen[0], screen[1], screen[2] = projected[i0], projected[i1], projected[i2]
				if !frontFacing(&screen) {
					continue
				}
				anyFront = true
				if shading == ShadingWireframe {
					continue
				}
				clipped[0], clipped[1], clipped[2] = corner(0), corner(k), corner(k+1)
				n = 3
			} else {
				tri[0], tri[1], tri[2] = corner(0), corner(k), corner(k+1)
				if n = clipNear(&tri, dist, &clipped); n < 3 {
					continue
				}
				for i := 0; i < n; i++ {
					x, y, _ := graphics.Project(clipped[i].pos, virtualW, virtualH, dist, baseScale)
					screen[i] = graphics.Vertex2D{X: x, Y: y}
				}
				// Clipping keeps the winding, so one polygon faces one way.
				if !frontFacing(&screen) {
					continue
				}
				anyFront = true
				if shading == ShadingWireframe {
					continue
				}
			}

			var normal graphics.Vector3D
			if shading == ShadingLambert {
				normal = outwardNormal(rotated[i0], rotated[i1], rotated[i2])
			}
			for j := 1; j+1 < n; j++ {
				a, b, c := clipped[0], clipped[j], clipped[j+1]
				sa, sb, sc := screen[0], screen[j], screen[j+1]
				switch shading {
				case ShadingTexture:
					if texture != nil {
						t.DrawTexturedTriangleDepth(sa, sb, sc, a.pos.Z, b.pos.Z, c.pos.Z, a.uv, b.uv, c.uv, texture)
					} else {
						t.DrawFilledTriangleDepth(sa, sb, sc, a.pos.Z, b.pos.Z, c.pos.Z, faceStyle)
					}
				case ShadingFlat:
					t.DrawFilledTriangleDepth(sa, sb, sc, a.pos.Z, b.pos.Z, c.pos.Z, faceStyle)
				case ShadingLambert:
					t.DrawLambertTriangleDepth(sa, sb, sc, a.pos.Z, b.pos.Z, c.pos.Z, normal, light, faceStyle)
				case ShadingGouraud:
					t.DrawGouraudTriangleDepth(sa, sb, sc, a.pos.Z, b.pos.Z, c.pos.Z,
						graphics.ApplyShade(col, a.shade), graphics.ApplyShade(col, b.shade), graphics.ApplyShade(col, c.shade), cell.Style{})
				}
			}
		}

		if anyFront && (shading == ShadingWireframe || v.Wireframe) {
			for i := range face {
				a, b := face[i], face[(i+1)%len(face)]
				if inFront(a) && inFront(b) {
					drawClippedLine(t, projected[a], projected[b], virtualW, virtualH, wireStyle)
				} else {
					v.drawEdge(t, rotated[a], rotated[b], dist, virtualW, virtualH, baseScale, wireStyle)
				}
			}
		}
	}

}

// vertexShades lights every vertex with the average of the normals of the
// faces around it, which is what makes Gouraud shading smooth; with one
// normal per face it would draw exactly what Lambert draws.
func (v *Viewer3D) vertexShades(rotated []graphics.Vertex3D, light graphics.Light) []float64 {
	n := len(rotated)
	if cap(v.normals) < n {
		v.normals = make([]graphics.Vector3D, n)
		v.shades = make([]float64, n)
	}
	normals, shades := v.normals[:n], v.shades[:n]
	for i := range normals {
		normals[i] = graphics.Vector3D{}
	}
	for _, face := range v.Model.Faces {
		if len(face) < 3 || !facesInRange(face, n) {
			continue
		}
		for k := 1; k+1 < len(face); k++ {
			fn := outwardNormal(rotated[face[0]], rotated[face[k]], rotated[face[k+1]])
			for _, idx := range [3]int{face[0], face[k], face[k+1]} {
				normals[idx].X += fn.X
				normals[idx].Y += fn.Y
				normals[idx].Z += fn.Z
			}
		}
	}
	for i := range shades {
		shades[i] = light.CalculateIntensity(normals[i].Normalize())
	}
	return shades
}

// drawEdge draws one wireframe edge, cut at the near plane and at the canvas
// edges; an endpoint just past the near plane projects thousands of pixels
// away, and Bresenham would walk every one of them.
func (v *Viewer3D) drawEdge(canvas raster3D, a, b graphics.Vertex3D, dist, w, h, scale float64, style cell.Style) {
	da, db := a.Z+dist-viewerNear, b.Z+dist-viewerNear
	if da < 0 && db < 0 {
		return
	}
	if da < 0 || db < 0 {
		t := da / (da - db)
		cut := graphics.Vertex3D{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t, Z: a.Z + (b.Z-a.Z)*t}
		if da < 0 {
			a = cut
		} else {
			b = cut
		}
	}
	x0, y0, _ := graphics.Project(a, w, h, dist, scale)
	x1, y1, _ := graphics.Project(b, w, h, dist, scale)
	drawClippedLine(canvas, graphics.Vertex2D{X: x0, Y: y0}, graphics.Vertex2D{X: x1, Y: y1}, w, h, style)
}

func drawClippedLine(canvas raster3D, p, q graphics.Vertex2D, w, h float64, style cell.Style) {
	if p.X >= 0 && p.Y >= 0 && q.X >= 0 && q.Y >= 0 && p.X < w && q.X < w && p.Y < h && q.Y < h {
		canvas.DrawLine(int(p.X), int(p.Y), int(q.X), int(q.Y), style)
		return
	}
	if x0, y0, x1, y1, ok := clipSegment(p.X, p.Y, q.X, q.Y, w-1, h-1); ok {
		canvas.DrawLine(int(x0), int(y0), int(x1), int(y1), style)
	}
}

// frontFacing reports whether the first three screen points turn the way
// front faces do: counter-clockwise, which is a negative cross product with
// y growing down.
func frontFacing(s *[4]graphics.Vertex2D) bool {
	return (s[1].X-s[0].X)*(s[2].Y-s[0].Y)-(s[1].Y-s[0].Y)*(s[2].X-s[0].X) < 0
}

// eulerRotation is RotateY, then RotateX, then RotateZ with the sines and
// cosines worked out once per frame instead of three times per vertex.
type eulerRotation struct{ sx, cx, sy, cy, sz, cz float64 }

func newEulerRotation(x, y, z float64) eulerRotation {
	const rad = math.Pi / 180
	var r eulerRotation
	r.sx, r.cx = math.Sincos(x * rad)
	r.sy, r.cy = math.Sincos(y * rad)
	r.sz, r.cz = math.Sincos(z * rad)
	return r
}

func (r eulerRotation) apply(v graphics.Vertex3D) graphics.Vertex3D {
	v = graphics.Vertex3D{X: v.X*r.cy + v.Z*r.sy, Y: v.Y, Z: -v.X*r.sy + v.Z*r.cy}
	v = graphics.Vertex3D{X: v.X, Y: v.Y*r.cx - v.Z*r.sx, Z: v.Y*r.sx + v.Z*r.cx}
	return graphics.Vertex3D{X: v.X*r.cz - v.Y*r.sz, Y: v.X*r.sz + v.Y*r.cz, Z: v.Z}
}

// clipSegment clips a segment to [0,maxX]×[0,maxY] (Liang–Barsky).
func clipSegment(x0, y0, x1, y1, maxX, maxY float64) (float64, float64, float64, float64, bool) {
	t0, t1 := 0.0, 1.0
	dx, dy := x1-x0, y1-y0
	for _, e := range [4][2]float64{{-dx, x0}, {dx, maxX - x0}, {-dy, y0}, {dy, maxY - y0}} {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return 0, 0, 0, 0, false
			}
			continue
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return 0, 0, 0, 0, false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return 0, 0, 0, 0, false
			}
			if r < t1 {
				t1 = r
			}
		}
	}
	return x0 + t0*dx, y0 + t0*dy, x0 + t1*dx, y0 + t1*dy, true
}

// outwardNormal is the normal pointing out of a face wound the way Viewer3D
// draws front faces (counter-clockwise on screen, camera looking down +Z).
// graphics.CalculateNormal returns the opposite one for that winding, and
// lighting with it lit the side of the model facing away from the light.
func outwardNormal(v0, v1, v2 graphics.Vertex3D) graphics.Vector3D {
	n := graphics.CalculateNormal(v0, v1, v2)
	return graphics.Vector3D{X: -n.X, Y: -n.Y, Z: -n.Z}
}

func facesInRange(face []int, n int) bool {
	for _, idx := range face {
		if idx < 0 || idx >= n {
			return false
		}
	}
	return true
}

// Shading modes for Viewer3D.Shading.
const (
	ShadingTexture   = "Texture"   // texture-mapped, needs a texture and UVs
	ShadingWireframe = "Wireframe" // edges only
	ShadingFlat      = "Flat"      // one colour per face
	ShadingLambert   = "Lambert"   // diffuse lighting per face
	ShadingGouraud   = "Gouraud"   // lighting interpolated across each face
)

// canonicalShading maps the names Viewer3D first shipped with, which were
// Turkish, to the constants above, so applications written against them keep
// working.
func canonicalShading(s string) string {
	switch s {
	case "Dokulu":
		return ShadingTexture
	case "Dolu Renkli":
		return ShadingFlat
	case "Gölgeli":
		return ShadingLambert
	}
	return s
}

// SizeHint takes all the space offered: a model scales to its area.
func (v *Viewer3D) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// AccessibilityNode describes the viewer as an image named after its model.
func (v *Viewer3D) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	label := v.Model.Name
	if label == "" {
		label = "3D model"
	}
	return accessibility.AccessibilityNode{ID: v.ID, Role: accessibility.RoleImage, Label: label, State: state, Bounds: bounds}
}
