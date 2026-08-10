package graphics

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// createTestImage, test amaçlı 2x2 basit renkli bir resim oluşturur.
func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})   // Kırmızı
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})   // Yeşil
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})   // Mavi
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

	// Resmi değiştir
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
}


