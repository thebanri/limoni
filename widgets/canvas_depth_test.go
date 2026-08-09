package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
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
