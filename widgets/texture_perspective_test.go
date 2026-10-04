package widgets

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/thebanri/limoni/graphics"
)

// A floor receding from the camera, textured red on the left half and blue
// on the right. The boundary is a straight line in 3D, so it must be a
// straight line on screen. With UVs interpolated in screen space (affine) it
// bends where the quad's two triangles meet; interpolated as u/w, v/w and
// 1/w it does not.
func TestTextureMappingIsPerspectiveCorrect(t *testing.T) {
	tex := image.NewRGBA(image.Rect(0, 0, 2, 1))
	tex.Set(0, 0, color.RGBA{R: 255, A: 255})
	tex.Set(1, 0, color.RGBA{B: 255, A: 255})

	const size, dist, scale = 128.0, 2.0, 100.0
	// Near edge low and wide, far edge high and narrow; a diagonal split.
	quad := [4]graphics.Vertex3D{{X: -1, Y: -1, Z: 0}, {X: 1, Y: -1, Z: 0}, {X: 1, Y: 1, Z: 4}, {X: -1, Y: 1, Z: 4}}
	uv := [4]graphics.UV{{U: 0, V: 1}, {U: 1, V: 1}, {U: 1, V: 0}, {U: 0, V: 0}}

	// boundary renders the floor and returns, per row, the column where red
	// turns blue.
	boundary := func(perspective bool) map[int]int {
		var target pixelTarget
		target.reset(int(size), int(size), color.RGBA{A: 255})
		var screen [4]graphics.Vertex2D
		for i, v := range quad {
			x, y, _ := graphics.Project(v, size, size, dist, scale)
			screen[i] = graphics.Vertex2D{X: x, Y: y}
		}
		w := func(i int) float64 {
			if perspective {
				return quad[i].Z + dist
			}
			return 1
		}
		for _, tri := range [2][3]int{{0, 1, 2}, {0, 2, 3}} {
			a, b, c := tri[0], tri[1], tri[2]
			target.DrawTexturedTrianglePerspective(screen[a], screen[b], screen[c],
				quad[a].Z, quad[b].Z, quad[c].Z, w(a), w(b), w(c), uv[a], uv[b], uv[c], tex)
		}
		edges := map[int]int{}
		for y := 0; y < int(size); y++ {
			for x := 1; x < int(size); x++ {
				left, right := target.img.RGBAAt(x-1, y), target.img.RGBAAt(x, y)
				if left.R == 255 && right.B == 255 {
					edges[y] = x
					break
				}
			}
		}
		return edges
	}

	// worst is how far the boundary strays from the line through its first
	// and last rows.
	worst := func(edges map[int]int) float64 {
		first, last := math.MaxInt, -1
		for y := range edges {
			first, last = min(first, y), max(last, y)
		}
		if last-first < 10 {
			t.Fatalf("the boundary spans only rows %d..%d", first, last)
		}
		x0, x1 := float64(edges[first]), float64(edges[last])
		dev := 0.0
		for y, x := range edges {
			expected := x0 + (x1-x0)*float64(y-first)/float64(last-first)
			dev = math.Max(dev, math.Abs(float64(x)-expected))
		}
		return dev
	}

	correct, affine := worst(boundary(true)), worst(boundary(false))
	t.Logf("boundary strays %.1f px perspective-correct, %.1f px affine", correct, affine)
	if dev := correct; dev > 1.5 {
		t.Errorf("perspective-correct mapping bends the boundary by %.1f pixels", dev)
	}
	// The test can tell: affine mapping does bend it.
	if dev := affine; dev < 3 {
		t.Errorf("affine mapping bends the boundary by only %.1f pixels; the scene does not show the warp", dev)
	}
}
