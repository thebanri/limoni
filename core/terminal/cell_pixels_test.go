package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// contextRecorder keeps the draw context it was given.
type contextRecorder struct{ ctx *cell.Context }

func (r contextRecorder) Draw(ctx cell.Context, _ *buffer.Buffer) { *r.ctx = ctx }

func (r contextRecorder) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// A widget is told the size of a cell in pixels, as the driver reports it
// (10×20 for an in-memory terminal), so it can render a picture at the size
// the terminal encodes it at.
func TestDrawContextCarriesTheCellPixelSize(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 20, 6))
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = term.Close() }()

	wantW, wantH, _ := b.CellPixelSize()
	var got cell.Context
	if err := term.Draw(func(f *Frame) {
		f.RenderWidget(contextRecorder{&got}, cell.NewRect(0, 0, 20, 6))
	}); err != nil {
		t.Fatal(err)
	}
	if got.CellPixelW == 0 || uint16(got.CellPixelW) != wantW || uint16(got.CellPixelH) != wantH {
		t.Errorf("the draw context says %d×%d pixels a cell, the driver %d×%d",
			got.CellPixelW, got.CellPixelH, wantW, wantH)
	}
}

// A cell too large for the context's byte is given as the largest it holds,
// not wrapped around to a small one.
func TestPixelsByteClamps(t *testing.T) {
	for _, c := range []struct {
		px   uint16
		want uint8
	}{{0, 0}, {22, 22}, {255, 255}, {256, 255}, {300, 255}} {
		if got := pixelsByte(c.px); got != c.want {
			t.Errorf("pixelsByte(%d) = %d, want %d", c.px, got, c.want)
		}
	}
}
