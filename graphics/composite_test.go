package graphics

import (
	"image"
	"testing"
)

// A decoded photo has nothing to composite: it is returned as it is, with
// no copy made or kept.
func TestFlattenLeavesOpaquePhotosAlone(t *testing.T) {
	img := image.NewYCbCr(image.Rect(0, 0, 8, 8), image.YCbCrSubsampleRatio420)
	if got := FlattenImageRGB(img, 1, 2, 3); got != img {
		t.Fatalf("an opaque photo was copied: %T", got)
	}
}

// The cache holds the image it flattened, so a pointer key cannot be reused
// by another picture at the same address; ForgetImage drops the entry.
func TestForgetImageDropsTheFlattenedCopy(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	first := FlattenImageRGB(img, 0, 0, 0)
	if again := FlattenImageRGB(img, 0, 0, 0); again != first {
		t.Fatal("the flattened copy was not cached")
	}
	ForgetImage(img)
	flattenedCacheMu.RLock()
	defer flattenedCacheMu.RUnlock()
	for _, entry := range flattenedImageCache {
		if entry.src == image.Image(img) {
			t.Fatal("ForgetImage left the flattened copy cached")
		}
	}
}
