package widgets

import (
	"image"
	"image/color"
	"math"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// dotTarget is what the depth-tested triangle rasterisers draw on: a Canvas,
// whose dots are Braille sub-cells, or a pixelTarget, whose dots are pixels.
type dotTarget interface {
	dotSize() (w, h int)
	SetDepth(x, y int, depth float64, style cell.Style) bool
}

func (c *Canvas) dotSize() (int, int) { return int(c.width) * 2, int(c.height) * 4 }

// triangleBounds is the triangle's bounding box clipped to the target, and
// the barycentric denominator; ok is false for a degenerate triangle.
func triangleBounds(p0, p1, p2 graphics.Vertex2D, w, h int) (minX, maxX, minY, maxY int, denom float64, ok bool) {
	minX = max(0, int(math.Min(p0.X, math.Min(p1.X, p2.X))))
	maxX = min(w-1, int(math.Max(p0.X, math.Max(p1.X, p2.X))))
	minY = max(0, int(math.Min(p0.Y, math.Min(p1.Y, p2.Y))))
	maxY = min(h-1, int(math.Max(p0.Y, math.Max(p1.Y, p2.Y))))
	denom = (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
	return minX, maxX, minY, maxY, denom, math.Abs(denom) >= 1e-6
}

func fillTriangleDepth[T dotTarget](t T, p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, style cell.Style) {
	w, h := t.dotSize()
	minX, maxX, minY, maxY, denom, ok := triangleBounds(p0, p1, p2, w, h)
	if !ok {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx, fy := float64(x), float64(y)
			l1 := ((p1.Y-p2.Y)*(fx-p2.X) + (p2.X-p1.X)*(fy-p2.Y)) / denom
			l2 := ((p2.Y-p0.Y)*(fx-p2.X) + (p0.X-p2.X)*(fy-p2.Y)) / denom
			l3 := 1 - l1 - l2
			if l1 >= -0.005 && l2 >= -0.005 && l3 >= -0.005 {
				t.SetDepth(x, y, l1*z0+l2*z1+l3*z2, style)
			}
		}
	}
}

func gouraudTriangleDepth[T dotTarget](t T, p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, c0, c1, c2 cell.Color, baseStyle cell.Style) {
	w, h := t.dotSize()
	minX, maxX, minY, maxY, denom, ok := triangleBounds(p0, p1, p2, w, h)
	if !ok {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx, fy := float64(x), float64(y)
			l1 := ((p1.Y-p2.Y)*(fx-p2.X) + (p2.X-p1.X)*(fy-p2.Y)) / denom
			l2 := ((p2.Y-p0.Y)*(fx-p2.X) + (p0.X-p2.X)*(fy-p2.Y)) / denom
			l3 := 1 - l1 - l2
			if l1 >= -0.005 && l2 >= -0.005 && l3 >= -0.005 {
				pixelStyle := baseStyle
				pixelStyle.Fg = graphics.BarycentricColor(c0, c1, c2, l1, l2, l3)
				t.SetDepth(x, y, l1*z0+l2*z1+l3*z2, pixelStyle)
			}
		}
	}
}

func texturedTriangleDepth[T dotTarget](t T, p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, uv0, uv1, uv2 graphics.UV, img image.Image) {
	if img == nil || img.Bounds().Empty() {
		return
	}
	w, h := t.dotSize()
	minX, maxX, minY, maxY, denom, ok := triangleBounds(p0, p1, p2, w, h)
	if !ok {
		return
	}
	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx, fy := float64(x), float64(y)
			l1 := ((p1.Y-p2.Y)*(fx-p2.X) + (p2.X-p1.X)*(fy-p2.Y)) / denom
			l2 := ((p2.Y-p0.Y)*(fx-p2.X) + (p0.X-p2.X)*(fy-p2.Y)) / denom
			l3 := 1.0 - l1 - l2
			if l1 >= -0.005 && l2 >= -0.005 && l3 >= -0.005 {
				u := l1*uv0.U + l2*uv1.U + l3*uv2.U
				v := l1*uv0.V + l2*uv1.V + l3*uv2.V
				tx := min(max(int(u*float64(imgW)), 0), imgW-1)
				ty := min(max(int(v*float64(imgH)), 0), imgH-1)
				r, g, b, a := texel(img, bounds.Min.X+tx, bounds.Min.Y+ty)
				if a >= 8000 {
					t.SetDepth(x, y, l1*z0+l2*z1+l3*z2, cell.Style{Fg: cell.NewColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))})
				}
			}
		}
	}
}

// DrawFilledTriangleDepth rasterizes a triangle with interpolated z-buffering.
func (c *Canvas) DrawFilledTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, style cell.Style) {
	fillTriangleDepth(c, p0, p1, p2, z0, z1, z2, style)
}

// DrawLambertTriangleDepth rasterizes a triangle with Lambertian diffuse shading and z-buffering.
func (c *Canvas) DrawLambertTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, normal graphics.Vector3D, light graphics.Light, baseStyle cell.Style) {
	fillTriangleDepth(c, p0, p1, p2, z0, z1, z2, lambertStyle(normal, light, baseStyle))
}

func lambertStyle(normal graphics.Vector3D, light graphics.Light, baseStyle cell.Style) cell.Style {
	shaded := baseStyle
	shaded.Fg = graphics.ApplyShade(baseStyle.Fg, light.CalculateIntensity(normal))
	return shaded
}

// DrawGouraudTriangleDepth rasterizes a triangle with per-vertex color interpolation (Gouraud shading) and z-buffering.
func (c *Canvas) DrawGouraudTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, c0, c1, c2 cell.Color, baseStyle cell.Style) {
	gouraudTriangleDepth(c, p0, p1, p2, z0, z1, z2, c0, c1, c2, baseStyle)
}

// DrawTexturedTriangleDepth rasterizes a triangle with texture mapping and z-buffering.
func (c *Canvas) DrawTexturedTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, uv0, uv1, uv2 graphics.UV, img image.Image) {
	if c == nil || c.width == 0 || c.height == 0 {
		return
	}
	texturedTriangleDepth(c, p0, p1, p2, z0, z1, z2, uv0, uv1, uv2, img)
}

// pixelTarget is an RGBA image with a z-buffer, for rendering a model as a
// picture the terminal shows with an image protocol.
type pixelTarget struct {
	img   *image.RGBA
	depth []float64
}

func (p *pixelTarget) reset(w, h int, bg color.RGBA) {
	if p.img == nil || p.img.Rect.Dx() != w || p.img.Rect.Dy() != h {
		p.img = image.NewRGBA(image.Rect(0, 0, w, h))
		p.depth = make([]float64, w*h)
	}
	for i := 0; i < len(p.img.Pix); i += 4 {
		p.img.Pix[i], p.img.Pix[i+1], p.img.Pix[i+2], p.img.Pix[i+3] = bg.R, bg.G, bg.B, bg.A
	}
	for i := range p.depth {
		p.depth[i] = math.Inf(1)
	}
}

func (p *pixelTarget) dotSize() (int, int) { return p.img.Rect.Dx(), p.img.Rect.Dy() }

func (p *pixelTarget) SetDepth(x, y int, depth float64, style cell.Style) bool {
	w, h := p.dotSize()
	if x < 0 || y < 0 || x >= w || y >= h || depth >= p.depth[y*w+x] {
		return false
	}
	p.depth[y*w+x] = depth
	p.set(x, y, style)
	return true
}

func (p *pixelTarget) set(x, y int, style cell.Style) {
	w, h := p.dotSize()
	if x < 0 || y < 0 || x >= w || y >= h {
		return
	}
	r, g, b := style.Fg.RGB()
	i := (y*w + x) * 4
	p.img.Pix[i], p.img.Pix[i+1], p.img.Pix[i+2], p.img.Pix[i+3] = r, g, b, 255
}

func (p *pixelTarget) DrawFilledTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, style cell.Style) {
	fillTriangleDepth(p, p0, p1, p2, z0, z1, z2, style)
}

func (p *pixelTarget) DrawLambertTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, normal graphics.Vector3D, light graphics.Light, baseStyle cell.Style) {
	fillTriangleDepth(p, p0, p1, p2, z0, z1, z2, lambertStyle(normal, light, baseStyle))
}

func (p *pixelTarget) DrawGouraudTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, c0, c1, c2 cell.Color, baseStyle cell.Style) {
	gouraudTriangleDepth(p, p0, p1, p2, z0, z1, z2, c0, c1, c2, baseStyle)
}

func (p *pixelTarget) DrawTexturedTriangleDepth(p0, p1, p2 graphics.Vertex2D, z0, z1, z2 float64, uv0, uv1, uv2 graphics.UV, img image.Image) {
	texturedTriangleDepth(p, p0, p1, p2, z0, z1, z2, uv0, uv1, uv2, img)
}

// DrawLine draws a one-pixel line (Bresenham), not depth tested, as the
// canvas draws wireframe edges.
func (p *pixelTarget) DrawLine(x1, y1, x2, y2 int, style cell.Style) {
	dx, dy := abs(x2-x1), abs(y2-y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		p.set(x1, y1, style)
		if x1 == x2 && y1 == y2 {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// texel returns the 16-bit RGBA of one texture pixel. image.Image.At boxes
// its color.Color, which is a heap allocation per sampled pixel; the formats
// textures actually arrive in (PNG decodes to RGBA/NRGBA, JPEG to YCbCr) are
// read directly instead.
func texel(img image.Image, x, y int) (r, g, b, a uint32) {
	switch m := img.(type) {
	case *image.RGBA:
		return m.RGBAAt(x, y).RGBA()
	case *image.NRGBA:
		return m.NRGBAAt(x, y).RGBA()
	case *image.YCbCr:
		return m.YCbCrAt(x, y).RGBA()
	case *image.Gray:
		return m.GrayAt(x, y).RGBA()
	case *image.Paletted:
		if i := int(m.ColorIndexAt(x, y)); i < len(m.Palette) {
			return m.Palette[i].RGBA()
		}
		return color.Transparent.RGBA()
	}
	return img.At(x, y).RGBA()
}
