package cell

import (
	"testing"
	"unsafe"
)

func TestColorPackingRGB(t *testing.T) {
	r, g, b := uint8(255), uint8(128), uint8(64)
	color := NewColorRGB(r, g, b)

	if color.Type() != ColorRGB {
		t.Errorf("Beklenen renk tipi ColorRGB, alınan: %v", color.Type())
	}

	gotR, gotG, gotB := color.RGB()
	if gotR != r || gotG != g || gotB != b {
		t.Errorf("RGB çözme başarısız. Beklenen: (%d, %d, %d), alınan: (%d, %d, %d)", r, g, b, gotR, gotG, gotB)
	}
}

func TestColorPackingANSI(t *testing.T) {
	code := uint8(105)
	color := NewColorANSI(code)

	if color.Type() != ColorANSI {
		t.Errorf("Beklenen renk tipi ColorANSI, alınan: %v", color.Type())
	}

	if color.ANSI() != code {
		t.Errorf("ANSI kodu eşleşmedi. Beklenen: %d, alınan: %d", code, color.ANSI())
	}
}

func TestStyleModifiers(t *testing.T) {
	var style Style
	style.Reset()

	if style.HasModifier(ModifierBold) {
		t.Errorf("Yeni sıfırlanmış stilde Bold olmamalıdır")
	}

	style = style.AddModifier(ModifierBold).AddModifier(ModifierItalic)
	if !style.HasModifier(ModifierBold) || !style.HasModifier(ModifierItalic) {
		t.Errorf("Modifiers eklenemedi")
	}

	style = style.RemoveModifier(ModifierBold)
	if style.HasModifier(ModifierBold) {
		t.Errorf("ModifierBold kaldırılamadı")
	}
	if !style.HasModifier(ModifierItalic) {
		t.Errorf("İlişkisiz özellik (Italic) kayboldu")
	}
}

func TestMemoryAlignment(t *testing.T) {
	sizeStyle := unsafe.Sizeof(Style{})
	sizeCell := unsafe.Sizeof(Cell{})

	// Target sizes:
	// Style: Fg (4) + Bg (4) + Modifier (2) + padding (2) = 12 bytes
	// Cell: Content (4) + Style (12) = 16 bytes
	const expectedStyleSize = 12
	const expectedCellSize = 16

	t.Logf("Boyutlar - Style: %d byte, Cell: %d byte", sizeStyle, sizeCell)

	if sizeStyle != expectedStyleSize {
		t.Errorf("Style struct boyutu %d olmalıydı, alınan: %d", expectedStyleSize, sizeStyle)
	}

	if sizeCell != expectedCellSize {
		t.Errorf("Cell struct boyutu %d olmalıydı, alınan: %d", expectedCellSize, sizeCell)
	}
}
