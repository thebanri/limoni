package benchmarks

import (
	"image"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

// What a moving model costs per frame in Viewer3D's pixel mode, kitty
// encoding included: the number quoted in its documentation. It allocates by
// nature — a PNG a frame — so it lives here, outside the widgets package
// whose every benchmark CI holds to zero allocations.
func BenchmarkViewer3DPixelsKitty(b *testing.B) {
	b.Setenv("LIMONI_GRAPHICS", "kitty")
	area := cell.NewRect(0, 0, 60, 24)
	buf := buffer.NewBuffer(area)
	var last image.Image
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterImage = func(_ cell.Rect, img image.Image, _ int, _ bool) bool { last = img; return true }
	v := &widgets.Viewer3D{Model: graphics.NewSphere(1, 24, 24), Shading: widgets.ShadingGouraud, Pixels: true}
	v.Draw(ctx, buf)
	if last == nil {
		b.Skip("no image protocol detected")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.RotY = float64(i)
		v.Draw(ctx, buf)
		_ = graphics.GetCachedEscapeSequence(last, area.Width, area.Height, 8, 16, graphics.ProtocolKitty, -1, true)
	}
}
