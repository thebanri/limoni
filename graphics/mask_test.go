package graphics

import (
	"image"
	"image/color"
	"testing"
)

func TestApplyCircleMask(t *testing.T) {
	// Create a solid 20x20 white image
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	masked := ApplyCircleMask(src)
	if masked == nil {
		t.Fatal("Expected masked image to be non-nil")
	}

	bounds := masked.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("Expected 20x20 bounds, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Center pixel (10, 10) must be fully opaque
	centerCol := masked.At(10, 10)
	_, _, _, a := centerCol.RGBA()
	if a < 20000 { // Check that it is mostly opaque (Go RGBA uses uint32 bounds [0, 65535])
		t.Errorf("Expected center pixel to be opaque, got alpha %d", a)
	}

	// Corner pixel (0, 0) must be fully transparent
	cornerCol := masked.At(0, 0)
	_, _, _, a2 := cornerCol.RGBA()
	if a2 > 0 {
		t.Errorf("Expected corner pixel (0,0) to be fully transparent, got alpha %d", a2)
	}
}
