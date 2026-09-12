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

func TestKoreanHangulAndCJKRuneWidth(t *testing.T) {
	hangul := []rune{'한', '글', '안', '녕', '하', '세', '요', '가', '힣'}
	for _, r := range hangul {
		if w := RuneWidth(r); w != 2 {
			t.Errorf("RuneWidth(Korean %c / U+%04X) = %d; want 2", r, r, w)
		}
	}
	text := "안녕하세요" // 5 Korean chars = 10 columns
	if w := StringWidth(text); w != 10 {
		t.Errorf("StringWidth(%q) = %d; want 10", text, w)
	}

	cjk := []rune{'漢', '字', '日', '本', '語', 'あ', 'い', 'う'}
	for _, r := range cjk {
		if w := RuneWidth(r); w != 2 {
			t.Errorf("RuneWidth(CJK %c / U+%04X) = %d; want 2", r, r, w)
		}
	}
}

func TestControlCharactersRuneWidth(t *testing.T) {
	controls := []rune{0, '\n', '\r', '\t', '\x1b', 0x07, 0x1F, 0x7F, 0x80, 0x9F}
	for _, r := range controls {
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(control U+%04X) = %d; want 0", r, w)
		}
	}
}

// The BMP lookup table is a performance shortcut around runeWidthSlow. If the
// two ever disagree, layout breaks in a way that is invisible until something
// renders wrong, so check every codepoint the table covers.
func TestRuneWidthTableMatchesRangeLogic(t *testing.T) {
	for r := rune(0); r < 0x10000; r++ {
		if got, want := RuneWidth(r), runeWidthSlow(r); got != want {
			t.Fatalf("RuneWidth(%#x) = %d, runeWidthSlow = %d", r, got, want)
		}
	}
}

func TestRuneWidthAboveBMPUsesRangeLogic(t *testing.T) {
	for r := rune(0x10000); r <= 0x10FFFF; r++ {
		if got, want := RuneWidth(r), runeWidthSlow(r); got != want {
			t.Fatalf("RuneWidth(%#x) = %d, runeWidthSlow = %d", r, got, want)
		}
	}
}

func TestRuneWidthKnownValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    rune
		want int
	}{
		{"ascii letter", 'A', 1},
		{"space", ' ', 1},
		{"nul", 0, 0},
		{"del", 0x7F, 0},
		{"soft hyphen", 0x00AD, 0},
		{"combining acute", 0x0301, 0},
		{"zero width joiner", 0x200D, 0},
		{"variation selector 16", 0xFE0F, 0},
		{"box drawing horizontal", '─', 1},
		{"box drawing corner", '┌', 1},
		{"hangul jamo", 0x1100, 2},
		{"cjk ideograph", '日', 2},
		{"hangul syllable", 0xAC00, 2},
		{"fullwidth exclamation", 0xFF01, 2},
		{"wide dingbat", 0x2705, 2},
		{"emoji plane 1", 0x1F600, 2},
		{"negative rune", -1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuneWidth(tc.r); got != tc.want {
				t.Errorf("RuneWidth(%#x) = %d, want %d", tc.r, got, tc.want)
			}
		})
	}
}

func BenchmarkRuneWidth(b *testing.B) {
	for _, tc := range []struct {
		name string
		r    rune
	}{
		{"ascii", 'A'},
		{"box-drawing", '─'},
		{"cjk", '日'},
		{"emoji", 0x1F600},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			var total int
			for i := 0; i < b.N; i++ {
				total += RuneWidth(tc.r)
			}
			_ = total
		})
	}
}
