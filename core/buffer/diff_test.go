package buffer

import (
	"bytes"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestDiffNoChanges(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	front := NewBuffer(area)
	back := NewBuffer(area)

	out := make([]byte, 0, 1024)
	out, err := Diff(front, back, out, true, true)
	if err != nil {
		t.Fatalf("Diff hatası: %v", err)
	}

	if len(out) != 0 {
		t.Errorf("Değişiklik yokken çıktı üretilmemeliydi. Çıktı: %q", string(out))
	}
}

func TestDiffCharacterChange(t *testing.T) {
	area := cell.NewRect(0, 0, 5, 2)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// (1, 0)'da bir karakter değiştir
	front.SetCell(1, 0, cell.Cell{Content: 'A'})

	out := make([]byte, 0, 1024)
	out, _ = Diff(front, back, out, true, true)

	// Beklenen: İmleç konumlandırma "\x1b[1;2HA" (y+1=1, x+1=2) ve ardından "A"
	expected := "\x1b[1;2HA"
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Beklenen çıktı: %q, Alınan: %q", expected, string(out))
	}

	// Back tamponunun güncellendiğini doğrula
	if back.Get(1, 0).Content != 'A' {
		t.Errorf("Back tamponu güncellenmedi")
	}
}

func TestDiffStyleTransitions(t *testing.T) {
	area := cell.NewRect(0, 0, 5, 2)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// (0, 0)'da Bold ve TrueColor Fg stilinde 'B' yaz
	style := cell.Style{
		Fg:       cell.NewColorRGB(255, 0, 0),
		Bg:       cell.NewColorDefault(),
		Modifier: cell.ModifierBold,
	}
	front.SetCell(0, 0, cell.Cell{Content: 'B', Style: style})

	out := make([]byte, 0, 1024)
	out, _ = Diff(front, back, out, true, true)

	// Beklenen: İmleç (\x1b[1;1H) + Fg RGB (\x1b[38;2;255;0;0m) + Bold (\x1b[1m) + 'B' + Reset style at frame end (\x1b[0m)
	if !bytes.Contains(out, []byte("B")) {
		t.Errorf("Çıktıda karakter bulunamadı: %q", string(out))
	}
	if !bytes.Contains(out, []byte("\x1b[38;2;255;0;0m")) {
		t.Errorf("Çıktıda renk kodu bulunamadı: %q", string(out))
	}
	if !bytes.Contains(out, []byte("\x1b[1m")) {
		t.Errorf("Çıktıda Bold kodu bulunamadı: %q", string(out))
	}
	if !bytes.HasSuffix(out, []byte("\x1b[0m")) {
		t.Errorf("Kare sonunda stil sıfırlama kodu bulunamadı: %q", string(out))
	}
}

func TestDiffModifierRemoval(t *testing.T) {
	area := cell.NewRect(0, 0, 5, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// front tamponunda (0,0) Bold 'A', (1,0) normal 'B' yapalım
	front.SetCell(0, 0, cell.Cell{
		Content: 'A',
		Style:   cell.Style{Modifier: cell.ModifierBold},
	})
	front.SetCell(1, 0, cell.Cell{
		Content: 'B',
		Style:   cell.Style{Modifier: cell.ModifierReset},
	})

	out := make([]byte, 0, 1024)
	out, _ = Diff(front, back, out, true, true)

	// Çıktıda Bold 'A' dan Normal 'B' ye geçerken \x1b[0m (reset) bulunmalıdır.
	// Tam çıktı: \x1b[1;1H\x1b[1mA\x1b[0mB
	expected := "\x1b[1;1H\x1b[1mA\x1b[0mB"
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Beklenen çıktı: %q, Alınan: %q", expected, string(out))
	}
}

// BENCHMARKS
// 120x40 çözünürlüğünde terminal ekranı için performans testleri.

func BenchmarkDiff_NoChanges(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40) // 4800 hücre
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Ön bellek ayırma
	out := make([]byte, 0, 8192)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = out[:0]
		out, _ = Diff(front, back, out, true, true)
	}
}

func BenchmarkDiff_PartialChanges(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40) // 4800 hücre
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Ekranın %10'unu değiştir (TUI uygulamaları için gerçekçi senaryo)
	for y := uint16(0); y < 40; y += 10 {
		for x := uint16(0); x < 120; x += 10 {
			front.SetCell(x, y, cell.Cell{
				Content: 'X',
				Style: cell.Style{
					Fg: cell.NewColorANSI(9),
					Bg: cell.NewColorANSI(0),
				},
			})
		}
	}

	out := make([]byte, 0, 16384)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = out[:0]
		// Her turda back tamponunu sıfırlayarak değişikliklerin tekrar diff'e düşmesini sağlıyoruz
		back.Clear()
		out, _ = Diff(front, back, out, true, true)
	}
}

func BenchmarkDiff_FullChanges(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Tüm hücreleri rastgele doldur
	style := cell.Style{Fg: cell.NewColorRGB(100, 200, 50)}
	for y := uint16(0); y < 40; y++ {
		for x := uint16(0); x < 120; x++ {
			front.SetCell(x, y, cell.Cell{Content: 'A', Style: style})
		}
	}

	out := make([]byte, 0, 65536)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = out[:0]
		back.Clear()
		out, _ = Diff(front, back, out, true, true)
	}
}

func TestDiffWideCharacters(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// SetString ile emoji yaz
	style := cell.Style{}
	front.SetString(0, 0, "🔴A", style)

	// front buffer hücrelerini doğrula
	if front.Get(0, 0).Content != '🔴' {
		t.Errorf("Beklenen emoji U+1F534, alınan: %c", front.Get(0, 0).Content)
	}
	if front.Get(1, 0).Content != cell.RuneContinuation {
		t.Errorf("Beklenen continuation karakteri U+FFFE, alınan: %x", front.Get(1, 0).Content)
	}
	if front.Get(2, 0).Content != 'A' {
		t.Errorf("Beklenen karakter A, alınan: %c", front.Get(2, 0).Content)
	}

	out := make([]byte, 0, 1024)
	out, err := Diff(front, back, out, true, true)
	if err != nil {
		t.Fatalf("Diff hatası: %v", err)
	}

	// 🔴 (U+1F534) utf-8 olarak 4 byte kaplar. A ise 1 byte.
	// Diff çıktısında continuation hücresi (index 1) yazılmamalıdır.
	// Yani sadece 🔴 ve A yazılmalıdır.
	if !bytes.Contains(out, []byte("🔴")) {
		t.Errorf("Çıktı emojiyi içermeliydi: %q", string(out))
	}
	if !bytes.Contains(out, []byte("A")) {
		t.Errorf("Çıktı 'A' karakterini içermeliydi: %q", string(out))
	}
}

func TestDiffColorDownsampling(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 1)

	// 1. Test downsampling RGB to 256 colors
	front256 := NewBuffer(area)
	back256 := NewBuffer(area)
	// Neon Purple: RGB(255, 0, 255)
	front256.SetCell(0, 0, cell.Cell{
		Content: 'X',
		Style: cell.Style{
			Fg: cell.NewColorRGB(255, 0, 255),
		},
	})
	out256, _ := Diff(front256, back256, nil, false, true)
	// Expected Fg code should be \x1b[38;5;201m (index 201 is pure Magenta in 256-color cube)
	if !bytes.Contains(out256, []byte("\x1b[38;5;201m")) {
		t.Errorf("Expected 256 color code for magenta in out, got: %q", string(out256))
	}

	// 2. Test downsampling RGB/256 to 16 colors
	front16 := NewBuffer(area)
	back16 := NewBuffer(area)
	front16.SetCell(0, 0, cell.Cell{
		Content: 'Y',
		Style: cell.Style{
			Fg: cell.NewColorRGB(255, 0, 255),
		},
	})
	out16, _ := Diff(front16, back16, nil, false, false)
	// Expected Fg code should be \x1b[38;5;13m or \x1b[38;5;5m (ansi 13 is bright magenta, ansi 5 is magenta)
	// Since 16 colors uses ColorANSI type, it writes \x1b[38;5;<ansi>m
	if !bytes.Contains(out16, []byte("\x1b[38;5;13m")) && !bytes.Contains(out16, []byte("\x1b[38;5;5m")) {
		t.Errorf("Expected 16-color ANSI code (13 or 5) for RGB magenta, got: %q", string(out16))
	}
}


