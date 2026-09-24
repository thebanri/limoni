package widgets

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

var (
	texRed  = cell.NewColorRGB(255, 0, 0)
	texBlue = cell.NewColorRGB(0, 0, 255)
)

// splitTexture is red on its left half and blue on its right.
func splitTexture() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{R: 255, A: 255}
			if x >= 4 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// quadAt is a square facing the camera at depth z, wound so it is front facing.
func quadAt(z, half float64) []graphics.Vertex3D {
	return []graphics.Vertex3D{{X: -half, Y: -half, Z: z}, {X: half, Y: -half, Z: z}, {X: half, Y: half, Z: z}, {X: -half, Y: half, Z: z}}
}

func drawViewer(t *testing.T, v *Viewer3D) *Canvas {
	t.Helper()
	area := cell.NewRect(0, 0, 40, 20)
	v.Draw(cell.NewContext(area, cell.Style{}), buffer.NewBuffer(area))
	return v.canvas
}

// countFg counts the cells with any dot set whose colour is exactly want.
func countFg(c *Canvas, want cell.Color) (matching, lit int) {
	for i, dots := range c.grid {
		if dots == 0 {
			continue
		}
		lit++
		if c.styles[i].Fg == want {
			matching++
		}
	}
	return matching, lit
}

// A textured face behind another must be hidden even when it is drawn last.
// Texture mode was the one mode that skipped the depth buffer.
func TestViewer3DTextureModeIsDepthTested(t *testing.T) {
	vertices := append(quadAt(-0.5, 1), quadAt(0.5, 1)...)
	v := &Viewer3D{
		Image:   splitTexture(),
		Shading: ShadingTexture,
		Model: graphics.Model3D{
			Vertices: vertices,
			Faces:    [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}}, // near first, far last
			UVs:      []graphics.UV{{U: 0.1, V: 0.5}, {U: 0.9, V: 0.5}},
			FaceUVs:  [][]int{{0, 0, 0, 0}, {1, 1, 1, 1}}, // near samples red, far blue
		},
	}
	c := drawViewer(t, v)
	red, lit := countFg(c, texRed)
	blue, _ := countFg(c, texBlue)
	if lit == 0 {
		t.Fatal("nothing drawn")
	}
	if blue != 0 {
		t.Errorf("%d cells show the far (blue) face through the near (red) one", blue)
	}
	if red != lit {
		t.Errorf("%d of %d lit cells are red", red, lit)
	}
}

// The texture follows the model's own UVs, not a stretch of the whole image.
func TestViewer3DTextureUsesModelUVs(t *testing.T) {
	v := &Viewer3D{
		Image:   splitTexture(),
		Shading: ShadingTexture,
		Model: graphics.Model3D{
			Vertices: quadAt(0, 1),
			Faces:    [][]int{{0, 1, 2, 3}},
			// Only the right, blue half of the image.
			UVs:     []graphics.UV{{U: 0.6, V: 1}, {U: 1, V: 1}, {U: 1, V: 0}, {U: 0.6, V: 0}},
			FaceUVs: [][]int{{0, 1, 2, 3}},
		},
	}
	c := drawViewer(t, v)
	if red, lit := countFg(c, texRed); lit == 0 || red != 0 {
		t.Errorf("lit=%d red=%d; the face maps only the blue half, so no cell may be red", lit, red)
	}

	// Without UVs the image is stretched across the face: both halves show.
	v.Model.FaceUVs, v.Model.UVs = nil, nil
	c = drawViewer(t, v)
	red, _ := countFg(c, texRed)
	blue, _ := countFg(c, texBlue)
	if red == 0 || blue == 0 {
		t.Errorf("placeholder mapping: red=%d blue=%d, want both", red, blue)
	}
}

// A face reaching behind the camera is cut at the near plane, not dropped.
func TestViewer3DClipsAtNearPlane(t *testing.T) {
	v := &Viewer3D{
		Shading:  ShadingFlat,
		Distance: 3.5,
		Model: graphics.Model3D{
			// Bottom edge 5 units towards the viewer, 1.5 behind the camera.
			Vertices: []graphics.Vertex3D{{X: -1, Y: -1, Z: -5}, {X: 1, Y: -1, Z: -5}, {X: 1, Y: 1, Z: 0}, {X: -1, Y: 1, Z: 0}},
			Faces:    [][]int{{0, 1, 2, 3}},
		},
	}
	c := drawViewer(t, v)
	if _, lit := countFg(c, 0); lit == 0 {
		t.Fatal("a face crossing the near plane drew nothing")
	}

	// Entirely behind the camera: nothing at all.
	v.Model.Vertices = quadAt(-10, 1)
	c = drawViewer(t, v)
	if _, lit := countFg(c, 0); lit != 0 {
		t.Errorf("a face behind the camera lit %d cells", lit)
	}
}

// The same crossing face in wireframe must stay cheap: without clipping, an
// endpoint just past the near plane projects far off the canvas.
func TestViewer3DWireframeNearPlaneEdge(t *testing.T) {
	v := &Viewer3D{
		Shading:  ShadingWireframe,
		Distance: 3.5,
		Model: graphics.Model3D{
			Vertices: []graphics.Vertex3D{{X: -1, Y: -1, Z: -3.389}, {X: 1, Y: -1, Z: -3.389}, {X: 1, Y: 1, Z: 0}, {X: -1, Y: 1, Z: 0}},
			Faces:    [][]int{{0, 1, 2, 3}},
		},
	}
	c := drawViewer(t, v)
	if _, lit := countFg(c, 0); lit == 0 {
		t.Fatal("wireframe of a face at the near plane drew nothing")
	}
}

// Gouraud lights each vertex with the average normal of the faces around it;
// on a sphere that is the direction from the centre, whatever the facet.
func TestViewer3DGouraudUsesSmoothVertexNormals(t *testing.T) {
	sphere := graphics.NewSphere(1, 24, 24)
	v := &Viewer3D{Model: sphere}
	light := graphics.DefaultLight()
	shades := v.vertexShades(sphere.Vertices, light)

	worst := 0.0
	for i, p := range sphere.Vertices {
		if math.Abs(p.Y) > 0.95 { // the poles' fans are lopsided
			continue
		}
		want := light.CalculateIntensity(graphics.Vector3D{X: p.X, Y: p.Y, Z: p.Z}.Normalize())
		worst = math.Max(worst, math.Abs(shades[i]-want))
	}
	if worst > 0.08 { // inverted normals are off by ~0.75
		t.Errorf("vertex shade differs from the smooth normal's by up to %.3f", worst)
	}
}

// DefaultLight comes from the upper right, in front: that side of a sphere
// must be the bright one.
func TestViewer3DLambertLightsTheSideFacingTheLight(t *testing.T) {
	v := &Viewer3D{Model: graphics.NewSphere(1, 24, 24), Shading: ShadingLambert, Scale: 3, FaceColors: []cell.Color{cell.NewColorRGB(200, 200, 200)}}
	c := drawViewer(t, v)
	brightness := func(cx, cy int) int {
		r, g, b := c.styles[cy*int(c.width)+cx].Fg.RGB()
		return int(r) + int(g) + int(b)
	}
	w, h := int(c.width), int(c.height)
	upperRight, lowerLeft := brightness(w/2+6, h/2-3), brightness(w/2-6, h/2+3)
	if upperRight <= lowerLeft {
		t.Errorf("upper right %d is not brighter than lower left %d", upperRight, lowerLeft)
	}
}

func TestViewer3DDrawDoesNotAllocate(t *testing.T) {
	area := cell.NewRect(0, 0, 60, 24)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	for _, shading := range []string{ShadingWireframe, ShadingFlat, ShadingLambert, ShadingGouraud, ShadingTexture} {
		v := &Viewer3D{Model: graphics.NewSphere(1, 16, 16), Shading: shading, Image: splitTexture(), RotX: 20, RotY: 30, Wireframe: true}
		v.Draw(ctx, buf)
		if got := testing.AllocsPerRun(20, func() { v.RotY += 7; v.Draw(ctx, buf) }); got != 0 {
			t.Errorf("%s: %.0f allocs per Draw, want 0", shading, got)
		}
	}
}

func TestClipSegment(t *testing.T) {
	for _, tc := range []struct {
		name               string
		x0, y0, x1, y1     float64
		ok                 bool
		wx0, wy0, wx1, wy1 float64
	}{
		{"inside", 1, 1, 5, 5, true, 1, 1, 5, 5},
		{"outside", -5, -5, -1, -1, false, 0, 0, 0, 0},
		{"through", -10, 5, 20, 5, true, 0, 5, 10, 5},
		{"far endpoint", 5, 5, 5, 1e9, true, 5, 5, 5, 10},
	} {
		x0, y0, x1, y1, ok := clipSegment(tc.x0, tc.y0, tc.x1, tc.y1, 10, 10)
		if ok != tc.ok || (ok && (x0 != tc.wx0 || y0 != tc.wy0 || x1 != tc.wx1 || y1 != tc.wy1)) {
			t.Errorf("%s: got (%v,%v)-(%v,%v) %v", tc.name, x0, y0, x1, y1, ok)
		}
	}
}

func BenchmarkViewer3DDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	for _, shading := range []string{ShadingWireframe, ShadingLambert, ShadingGouraud, ShadingTexture} {
		b.Run(shading, func(b *testing.B) {
			v := &Viewer3D{Model: graphics.NewSphere(1, 24, 24), Shading: shading, Image: splitTexture(), RotX: 20}
			v.Draw(ctx, buf)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v.RotY = float64(i % 360)
				v.Draw(ctx, buf)
			}
		})
	}
}

func TestEulerRotationMatchesRotateChain(t *testing.T) {
	p := graphics.Vertex3D{X: 0.3, Y: -1.2, Z: 2.5}
	for _, a := range [][3]float64{{0, 0, 0}, {30, 45, 60}, {-170, 95, 12}, {359, 1, 180}} {
		want := p.RotateY(a[1]).RotateX(a[0]).RotateZ(a[2])
		got := newEulerRotation(a[0], a[1], a[2]).apply(p)
		if math.Abs(got.X-want.X)+math.Abs(got.Y-want.Y)+math.Abs(got.Z-want.Z) > 1e-9 {
			t.Errorf("angles %v: got %+v, want %+v", a, got, want)
		}
	}
}

type imageRecorder struct {
	area cell.Rect
	img  image.Image
	n    int
}

func (r *imageRecorder) ctx(area cell.Rect) cell.Context {
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterImage = func(a cell.Rect, img image.Image, _ int, _ bool) bool {
		r.area, r.img = a, img
		r.n++
		return true
	}
	return ctx
}

// With Pixels and an image protocol, the model becomes a picture at 8×16
// pixels a cell, lit like the dots would be, and the cells under it are
// reserved for it.
func TestViewer3DPixels(t *testing.T) {
	area := cell.NewRect(2, 1, 20, 10)
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 30, 12))
	rec := &imageRecorder{}
	v := &Viewer3D{Model: graphics.NewSphere(1, 24, 24), Shading: ShadingLambert, Scale: 1.5, Pixels: true,
		FaceColors: []cell.Color{cell.NewColorRGB(200, 200, 200)}}
	v.proto, v.protoKnown = graphics.ProtocolKitty, true
	v.Draw(rec.ctx(area), buf)

	img, ok := rec.img.(*image.RGBA)
	if !ok || rec.area != area || img.Rect.Dx() != 160 || img.Rect.Dy() != 160 {
		t.Fatalf("registered %T %v at %v", rec.img, img.Rect, rec.area)
	}
	if buf.CellAt(5, 5).Content != cell.RuneImage || buf.CellAt(1, 5).Content == cell.RuneImage {
		t.Error("the cells under the picture, and only those, are reserved")
	}
	lum := func(x, y int) int { c := img.RGBAAt(x, y); return int(c.R) + int(c.G) + int(c.B) }
	if img.RGBAAt(0, 0).A != 0 {
		t.Error("the corner outside the sphere is not transparent")
	}
	// The sphere's radius is about 27 pixels: scale 160·0.4·1.5 over distance 3.5.
	if ur, ll := lum(80+12, 80-12), lum(80-12, 80+12); ur <= ll || ll == 0 {
		t.Errorf("upper right %d not brighter than lower left %d", ur, ll)
	}

	// A still model is not rendered or sent again; a moved one is, as a
	// different image.
	first := rec.img
	v.Draw(rec.ctx(area), buf)
	if rec.img != first {
		t.Error("a still model produced a new picture")
	}
	v.RotY = 30
	v.Draw(rec.ctx(area), buf)
	if rec.img == first {
		t.Error("a rotated model reused the old picture")
	}
	ctx := rec.ctx(area)
	if n := testing.AllocsPerRun(10, func() { v.RotY++; v.Draw(ctx, buf) }); n != 0 {
		t.Errorf("%.0f allocs per moving pixel frame in the widget (encoding is the terminal's)", n)
	}

	// No image protocol: dots, as without Pixels.
	v.proto = graphics.ProtocolHalfBlock
	rec.n = 0
	v.Draw(rec.ctx(area), buf)
	if rec.n != 0 || buf.CellAt(5, 5).Content == cell.RuneImage {
		t.Error("drew a picture without an image protocol")
	}
}

// Viewer3D is a Widget, so layouts and Frame.RenderWidget can place it.
var _ Widget = (*Viewer3D)(nil)
