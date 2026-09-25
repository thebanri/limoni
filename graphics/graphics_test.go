package graphics

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// createTestImage builds a 2x2 image with one pixel of each colour.
func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})     // red
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})     // green
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})     // Mavi
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // Beyaz
	return img
}

func TestGetImageID(t *testing.T) {
	img1 := createTestImage()
	img2 := createTestImage()

	id1 := GetImageID(img1)
	id2 := GetImageID(img2)

	if id1 != id2 {
		t.Errorf("Expected identical images to have same ID, got %d and %d", id1, id2)
	}

	// Change the image.
	img3 := image.NewRGBA(image.Rect(0, 0, 2, 2))
	id3 := GetImageID(img3)

	if id1 == id3 {
		t.Errorf("Expected different images to have different IDs, got both as %d", id1)
	}
}

func TestEncodeKitty(t *testing.T) {
	img := createTestImage()

	// Direct transfer (without ID cache)
	esc := EncodeKitty(img, 2, 1, 10, 20, 0, 0, false)
	if !strings.HasPrefix(esc, "\x1b_G") {
		t.Errorf("Expected escape prefix, got %q", esc[:3])
	}
	if !strings.HasSuffix(esc, "\x1b\\") {
		t.Errorf("Expected escape suffix, got %q", esc[len(esc)-2:])
	}
}

func TestEncodeIterm2(t *testing.T) {
	img := createTestImage()
	esc := EncodeIterm2(img, 2, 1, 10, 20, false)

	if !strings.HasPrefix(esc, "\x1b]1337;File=") {
		t.Errorf("Expected iTerm2 escape prefix, got %q", esc[:12])
	}
	if !strings.HasSuffix(esc, "\a") {
		t.Errorf("Expected iTerm2 escape suffix, got %q", esc[len(esc)-1:])
	}
}

func TestEncodeSixel(t *testing.T) {
	img := createTestImage()
	esc := EncodeSixel(img, 2, 1, 10, 20, false)

	if !strings.HasPrefix(esc, "\x1bPq\"1;1;") {
		t.Errorf("Expected Sixel escape prefix, got %q", esc[:7])
	}
	if !strings.HasSuffix(esc, "\x1b\\") {
		t.Errorf("Expected Sixel escape suffix, got %q", esc[len(esc)-2:])
	}
}

func TestDetectProtocol(t *testing.T) {
	// Test Alacritty detection via TERM_PROGRAM
	t.Setenv("TERM_PROGRAM", "Alacritty")
	proto := DetectProtocol()
	if proto != ProtocolHalfBlock {
		t.Errorf("Expected ProtocolHalfBlock for TERM_PROGRAM=Alacritty, got %v", proto)
	}

	// Test Alacritty detection via ALACRITTY_WINDOW_ID
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("ALACRITTY_WINDOW_ID", "12345")
	proto = DetectProtocol()
	if proto != ProtocolHalfBlock {
		t.Errorf("Expected ProtocolHalfBlock for ALACRITTY_WINDOW_ID, got %v", proto)
	}
}

func TestApplyOpacity(t *testing.T) {
	img := createTestImage() // 2x2 image, (0,0) is Red (255,0,0,255)

	// Apply 50% opacity
	opaqueImg := ApplyOpacity(img, 0.5)

	r, g, b, a := opaqueImg.At(0, 0).RGBA()

	// Since RGBA returns 0-65535, 50% of 65535 is around 32767.
	// 32767 >> 8 is 127.
	valA := uint8(a >> 8)
	valR := uint8(r >> 8)

	if valA < 120 || valA > 135 {
		t.Errorf("Expected alpha around 127, got %d", valA)
	}
	if valR < 120 || valR > 135 {
		t.Errorf("Expected red channel around 127, got %d", valR)
	}
	if g != 0 || b != 0 {
		t.Errorf("Expected green/blue to remain 0, got %d, %d", g, b)
	}

	// The transformed image must be reused between frames so native image
	// protocols do not re-upload an unchanged image.
	cachedImg := ApplyOpacity(img, 0.5)
	if opaqueImg != cachedImg {
		t.Fatal("expected ApplyOpacity to reuse the cached transformed image")
	}
}

func TestResizeImage_EdgeCases(t *testing.T) {
	// 1x1 image upscaled
	img1 := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img1.Set(0, 0, color.RGBA{R: 255, G: 128, B: 64, A: 255})
	up := ResizeImage(img1, 10, 10)
	if up.Bounds().Dx() != 10 || up.Bounds().Dy() != 10 {
		t.Fatalf("expected 10x10, got %v", up.Bounds())
	}
	r, g, b, _ := up.At(5, 5).RGBA()
	if uint8(r>>8) != 255 || uint8(g>>8) != 128 || uint8(b>>8) != 64 {
		t.Fatalf("unexpected color after 1x1 upscaling: %d, %d, %d", r>>8, g>>8, b>>8)
	}

	// 10x1 downscaled to 2x1
	imgStrip := image.NewRGBA(image.Rect(0, 0, 10, 1))
	down := ResizeImage(imgStrip, 2, 1)
	if down.Bounds().Dx() != 2 || down.Bounds().Dy() != 1 {
		t.Fatalf("expected 2x1, got %v", down.Bounds())
	}
}

func TestEncodeSixelMapsColoursPastThePaletteToTheNearest(t *testing.T) {
	// The first row fills all 256 palette entries, the last being white; the
	// second band is a near-white the palette has no room for. It must be
	// drawn with white, not with entry 0.
	img := image.NewRGBA(image.Rect(0, 0, 256, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 256; x++ {
			c := color.RGBA{A: 255}
			switch {
			case y >= 6:
				c = color.RGBA{R: 250, G: 250, B: 250, A: 255}
			case y == 0 && x == 255:
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			case y == 0:
				c.R = uint8(x)
			}
			img.SetRGBA(x, y, c)
		}
	}
	out := EncodeSixel(img, 32, 1, 8, 12, false)
	bands := strings.Split(out, "-")
	if len(bands) < 2 {
		t.Fatalf("expected two sixel bands, got %q", out)
	}
	second := bands[1]
	if !strings.Contains(second, "#255") {
		t.Errorf("near-white band does not use the white entry #255: %q", second)
	}
	if strings.Contains(second, "#0!") || strings.Contains(second, "#0~") {
		t.Errorf("near-white band fell back to entry 0: %q", second)
	}
}

func TestEncodeSixel_Transparent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	// Set half transparent, half opaque
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0}) // Transparent

	sixel := EncodeSixel(img, 4, 2, 1, 2, true)
	if !strings.HasPrefix(sixel, "\x1bPq") {
		t.Fatalf("expected sixel header, got %q", sixel)
	}
	if !strings.HasSuffix(sixel, "\x1b\\") {
		t.Fatalf("expected sixel terminator, got %q", sixel)
	}
}

func TestBuildPaletteMatchesRGBAModel(t *testing.T) {
	bounds := image.Rect(3, 5, 5, 7)
	rgba := image.NewRGBA(bounds)
	rgba.SetRGBA(3, 5, color.RGBA{R: 200, G: 120, B: 80, A: 128})
	rgba.SetRGBA(4, 5, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	nrgba := image.NewNRGBA(bounds)
	nrgba.SetNRGBA(3, 5, color.NRGBA{R: 200, G: 120, B: 80, A: 128})
	nrgba.SetNRGBA(4, 5, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	gray := image.NewGray(bounds)
	gray.SetGray(3, 5, color.Gray{Y: 80})
	gray.SetGray(4, 5, color.Gray{Y: 160})
	ycbcr := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio444)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ycbcr.Y[ycbcr.YOffset(x, y)] = uint8(80 + 20*x + 10*y)
			c := ycbcr.COffset(x, y)
			ycbcr.Cb[c], ycbcr.Cr[c] = 128, 128
		}
	}

	images := []struct {
		name string
		img  image.Image
	}{
		{name: "RGBA", img: rgba},
		{name: "NRGBA", img: nrgba},
		{name: "Gray", img: gray},
		{name: "YCbCr", img: ycbcr},
	}
	for _, tc := range images {
		t.Run(tc.name, func(t *testing.T) {
			var want color.Palette
			seen := make(map[color.Color]struct{})
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					c := color.RGBAModel.Convert(tc.img.At(x, y))
					if _, ok := seen[c]; !ok {
						want = append(want, c)
						seen[c] = struct{}{}
					}
				}
			}

			got := buildPalette(tc.img, 256)
			if len(got) != len(want) {
				t.Fatalf("palette length = %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("palette[%d] = %#v, want %#v", i, got[i], want[i])
				}
			}
		})
	}
}

func TestCacheBounds(t *testing.T) {
	// Verify that inserting > 256 images doesn't leak memory or panic
	for i := 0; i < 300; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 2, 2))
		_ = ApplyOpacity(img, 0.8)
		_ = FlattenImage(img, color.RGBA{R: uint8(i % 255), G: 0, B: 0, A: 255})
		_ = ApplyCircleMask(img)
	}
}

func TestApplyShade_ANSI(t *testing.T) {
	ansiRed := cell.NewColorANSI(9) // Bright red -> (255, 0, 0)
	shaded := ApplyShade(ansiRed, 0.5)
	r, g, b := shaded.RGB()
	if r < 120 || r > 135 || g != 0 || b != 0 {
		t.Fatalf("expected shaded red (~128, 0, 0), got (%d, %d, %d)", r, g, b)
	}
}

func BenchmarkGetImageID_1080p(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetImageID(img)
	}
}

func BenchmarkResizeImageContain(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ResizeImageContain(img, 120, 40, true)
	}
}

func BenchmarkEncodeKitty(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeKitty(img, 20, 10, 8, 16, 42, 0, true)
	}
}

func BenchmarkEncodeSixel(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeSixel(img, 20, 10, 8, 16, true)
	}
}

// A picture that reaches the last row must not move the cursor below it:
// the terminal would scroll the screen and the diff would draw every later
// frame a row off. Found by running Viewer3D's pixel mode in kitty 0.48.2.
func TestImagePlacementsKeepTheCursor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if seq := EncodeKitty(img, 2, 2, 8, 16, 1, -1, true); !strings.Contains(seq, ",C=1,") {
		t.Errorf("kitty placement without C=1: %.80q", seq)
	}
	if seq := EncodeIterm2(img, 2, 2, 8, 16, true); !strings.Contains(seq, "doNotMoveCursor=1") {
		t.Errorf("iTerm2 placement without doNotMoveCursor=1: %.80q", seq)
	}
}
