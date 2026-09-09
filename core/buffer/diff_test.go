package buffer

import (
	"bytes"
	"strings"
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
	area := cell.NewRect(0, 0, 120, 40) // 4800 cells
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Baseline frame: fill with baseline content
	baseStyle := cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}
	for y := uint16(0); y < 40; y++ {
		for x := uint16(0); x < 120; x++ {
			front.SetCell(x, y, cell.Cell{Content: '.', Style: baseStyle})
		}
	}
	out := make([]byte, 0, 16384)
	// Initial diff so back buffer matches baseline
	out, _ = Diff(front, back, out, true, true)

	activeStyle := cell.Style{
		Fg: cell.NewColorANSI(9),
		Bg: cell.NewColorANSI(0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Realistic scenario: mutate ~10% of cells (480 cells across 4 rows) each frame
		row := uint16(i % 37)
		for col := uint16(0); col < 120; col++ {
			ch := rune('0' + ((i + int(col)) % 10))
			front.SetCell(col, row, cell.Cell{Content: ch, Style: activeStyle})
			front.SetCell(col, row+1, cell.Cell{Content: ch, Style: activeStyle})
			front.SetCell(col, row+2, cell.Cell{Content: ch, Style: activeStyle})
			front.SetCell(col, (row+10)%40, cell.Cell{Content: ch, Style: activeStyle})
		}

		out = out[:0]
		out, _ = Diff(front, back, out, true, true)
	}
}

func BenchmarkDiff_FullChanges(b *testing.B) {
	area := cell.NewRect(0, 0, 120, 40) // 4800 cells
	front := NewBuffer(area)
	back := NewBuffer(area)

	out := make([]byte, 0, 65536)
	// Initial diff so back buffer is primed
	out, _ = Diff(front, back, out, true, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Realistic 100% full-screen redraw: every single cell mutates each frame
		ch := rune('A' + (i % 26))
		style := cell.Style{Fg: cell.NewColorRGB(uint8(i%256), 200, 50)}
		for y := uint16(0); y < 40; y++ {
			for x := uint16(0); x < 120; x++ {
				front.SetCell(x, y, cell.Cell{Content: ch, Style: style})
			}
		}

		out = out[:0]
		out, _ = Diff(front, back, out, true, true)
	}
}

func TestDiffBufferGetMutationNotSkipped(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// İlk diff - her iki tampon da temiz
	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if front.IsDirty {
		t.Fatal("front should not be dirty after clean diff")
	}

	// Buffer.Get ile doğrudan hücre mutasyonu yap
	cellPtr := front.Get(2, 2)
	if cellPtr == nil {
		t.Fatal("Get(2, 2) returned nil")
	}
	cellPtr.Content = 'Z'
	cellPtr.Style = cell.Style{Fg: cell.NewColorANSI(2)}

	if !front.IsDirty {
		t.Fatal("front.IsDirty must be true after calling Buffer.Get")
	}

	// Diff doğrudan hücre mutasyonunu yakalamalı ve kaçış kodu üretmeli
	out, err = Diff(front, back, out[:0], true, true)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Diff output should not be empty after direct cell mutation via Get")
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

func TestDiffStyleCacheCap(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Simulate rendering thousands of unique RGB styles
	var out []byte
	for i := 0; i < 5000; i++ {
		r := uint8(i % 256)
		g := uint8((i / 256) % 256)
		b := uint8((i * 7) % 256)
		front.SetCell(0, 0, cell.Cell{
			Content: 'A',
			Style:   cell.Style{Fg: cell.NewColorRGB(r, g, b), Modifier: cell.ModifierBold},
		})
		var err error
		out, err = Diff(front, back, out[:0], true, true)
		if err != nil {
			t.Fatalf("Diff error: %v", err)
		}
	}

	// Verify StyleCache does not exceed the 2048 entry threshold
	if len(front.StyleCache) > 2048 {
		t.Errorf("front.StyleCache grew unbounded: len = %d, want <= 2048", len(front.StyleCache))
	}
}

func TestDiffSanitizesControlCharacters(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Inject control characters into cells directly
	cellPtr := front.Get(0, 0)
	cellPtr.Content = '\n'
	cellPtr2 := front.Get(1, 0)
	cellPtr2.Content = '\x1b'
	cellPtr3 := front.Get(2, 0)
	cellPtr3.Content = '\r'

	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// The output should NOT contain raw \n or \r, and \x1b should only be part of ANSI escape sequences (CSI), not raw \x1b followed by space
	for i := 0; i < len(out); i++ {
		if out[i] == '\n' || out[i] == '\r' {
			t.Errorf("Diff output contains raw newline or carriage return: %q", string(out))
		}
	}
}

func TestDiffRuneImageNeedsErase(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// 1. Back has dialog text, front has RuneImage -> must emit ECH to erase old text
	for x := uint16(0); x < 10; x++ {
		back.SetCell(x, 0, cell.Cell{Content: 'X'})
		front.SetCell(x, 0, cell.Cell{Content: cell.RuneImage})
	}
	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "10X") {
		t.Errorf("Expected ECH erase command '10X' in output, got: %q", outStr)
	}

	// 2. Back is already RuneImage, front is RuneImage -> must emit NOTHING (zero ECH)
	out, err = Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("Expected zero diff bytes when both front and back are RuneImage, got %d bytes: %q", len(out), string(out))
	}

	// 3. Back is RuneInvalid (after ForceFullRedraw / buffer invalidation), front is RuneImage -> must emit ECH to clear any physical hardware artifacts
	front.Invalidate()
	for x := uint16(0); x < 10; x++ {
		back.SetCell(x, 0, cell.Cell{Content: cell.RuneInvalid})
	}
	out, err = Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if !strings.Contains(string(out), "10X") {
		t.Errorf("Expected ECH '10X' when back is RuneInvalid, got: %q", string(out))
	}
}

func TestDiffWideCharactersPartialRedraw(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Frame 1: Write wide emoji and text
	front.SetString(0, 0, "🚀ABC", cell.Style{})
	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff Frame 1 error: %v", err)
	}
	if !bytes.Contains(out, []byte("🚀")) || !bytes.Contains(out, []byte("ABC")) {
		t.Fatalf("Frame 1 missing expected content: %q", string(out))
	}

	// Frame 2: Only change the last character ('C' -> 'Z')
	front.SetCell(4, 0, cell.Cell{Content: 'Z'})
	out, err = Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff Frame 2 error: %v", err)
	}

	// It should ONLY position to (5, 1) and emit 'Z', no continuation chars or garbage
	if bytes.Contains(out, []byte{0xFE, 0xFF}) || bytes.Contains(out, []byte("🚀")) {
		t.Errorf("Frame 2 output shouldn't re-emit emoji or continuation: %q", string(out))
	}
	if !bytes.Contains(out, []byte("Z")) {
		t.Errorf("Frame 2 output should contain 'Z': %q", string(out))
	}

	// Frame 3: Replace the wide character with another wide character ("🔥")
	front.SetString(0, 0, "🔥", cell.Style{})
	out, err = Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff Frame 3 error: %v", err)
	}
	if !bytes.Contains(out, []byte("🔥")) {
		t.Errorf("Frame 3 output should contain '🔥': %q", string(out))
	}
	// Must not emit 0xFFFE
	if bytes.Contains(out, []byte("\xef\xbf\xbe")) {
		t.Errorf("Frame 3 must not emit 0xFFFE continuation bytes: %q", string(out))
	}
}

func TestDiffSkipAvoidsWideRune(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Fill buffer with "A🔥B"
	front.SetString(0, 0, "A🔥B", cell.Style{})
	_, _ = Diff(front, back, nil, true, true)

	// Change cell 0 ('A' -> 'X') and cell 3 ('B' -> 'Y')
	front.SetCell(0, 0, cell.Cell{Content: 'X'})
	front.SetCell(3, 0, cell.Cell{Content: 'Y'})

	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}

	// It must NOT print the continuation or re-print 🔥 as single char in skip
	if bytes.Contains(out, []byte("\xef\xbf\xbe")) {
		t.Errorf("Diff must never emit RuneContinuation (0xFFFE): %q", string(out))
	}
}

func TestDiffContinuationRestorationOnModalDrag(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 2)
	front := NewBuffer(area)
	back := NewBuffer(area)

	// Step 1: Draw a wide character at x=5,6 ("🔴")
	front.SetString(5, 0, "🔴", cell.Style{})
	_, _ = Diff(front, back, nil, true, true)

	// Step 2: Overlay dialog border directly at continuation cell x=6
	c := front.Get(6, 0)
	c.Content = '│'
	c.Style = cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
	_, _ = Diff(front, back, nil, true, true)

	if back.Get(6, 0).Content != '│' {
		t.Fatalf("back[6] = %q; want '│'", back.Get(6, 0).Content)
	}

	// Step 3: Remove dialog: redraw background. Wide rune is back at x=5,6
	front = NewBuffer(area)
	front.SetString(5, 0, "🔴", cell.Style{})

	out, err := Diff(front, back, nil, true, true)
	if err != nil {
		t.Fatalf("Diff restore error: %v", err)
	}

	// The diff MUST emit "🔴" at column 6 (1-based), completely replacing the dialog border '│'
	if !bytes.Contains(out, []byte("🔴")) {
		t.Fatalf("Diff output must contain wide rune 🔴 to overwrite dialog border: %q", string(out))
	}
	if back.Get(6, 0).Content != cell.RuneContinuation {
		t.Fatalf("back[6] after restore = %q; want RuneContinuation", back.Get(6, 0).Content)
	}
}



