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

func TestGreekExtendedRuneWidth(t *testing.T) {
	// U+1F00..U+1F1F are Greek Extended letters and must have width 1 (not 0)
	greekChars := []rune{'ἀ', 'ἁ', 'ἂ', 'ἃ', 'ἄ', 'ἅ', 'ἆ', 'ἇ', 'Ἀ', 'Ἑ', 'Ἕ'}
	for _, r := range greekChars {
		w := RuneWidth(r)
		if w != 1 {
			t.Errorf("RuneWidth(%c / U+%04X) = %d; want 1", r, r, w)
		}
	}
	text := "ἀρχή"
	if w := StringWidth(text); w != 4 {
		t.Errorf("StringWidth(%q) = %d; want 4", text, w)
	}
}

func TestCombiningMarksRuneWidth(t *testing.T) {
	combiningMarks := []rune{
		0x0300, // Combining Grave Accent
		0x0301, // Combining Acute Accent
		0x1AB0, // Combining Doubled Circumflex Accent
		0x1DC0, // Combining Dotted Grave Accent
		0x20D0, // Combining Left Harpoon Above
		0xFE20, // Combining Ligature Left Half
	}
	for _, r := range combiningMarks {
		w := RuneWidth(r)
		if w != 0 {
			t.Errorf("RuneWidth(U+%04X) = %d; want 0", r, w)
		}
	}
}
