package buffer

import (
	"testing"
	"unsafe"

	"github.com/thebanri/limoni/core/cell"
)

func TestNewBuffer(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	buf := NewBuffer(area)

	if buf.Area.Width != 20 || buf.Area.Height != 10 {
		t.Errorf("Buffer alanı hatalı. Beklenen: 20x10, alınan: %dx%d", buf.Area.Width, buf.Area.Height)
	}

	if len(buf.Content) != 200 {
		t.Errorf("İçerik boyutu hatalı. Beklenen: 200, alınan: %d", len(buf.Content))
	}

	// Tüm hücrelerin boşluk karakteri ve default style ile başladığını doğrula
	for i, c := range buf.Content {
		if c.Content != ' ' {
			t.Errorf("İndeks %d varsayılan karakter boşluk olmalıydı, alınan: %q", i, c.Content)
		}
		if c.Style.Fg.Type() != cell.ColorDefault || c.Style.Bg.Type() != cell.ColorDefault {
			t.Errorf("İndeks %d varsayılan stil default olmalıydı", i)
		}
	}
}

func TestBufferGetSet(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := NewBuffer(area)

	c := cell.Cell{
		Content: 'X',
		Style: cell.Style{
			Fg: cell.NewColorANSI(9),
			Bg: cell.NewColorANSI(0),
		},
	}

	buf.SetCell(2, 2, c)

	got := buf.Get(2, 2)
	if got == nil {
		t.Fatalf("Hücre nil dönmemeliydi")
	}

	if got.Content != 'X' || got.Style.Fg.ANSI() != 9 {
		t.Errorf("Hücre değeri doğru set edilmedi")
	}

	// Geçersiz koordinat testi
	if out := buf.Get(10, 5); out != nil {
		t.Errorf("Sınır dışı koordinat nil dönmeliydi")
	}
}

func TestBufferSetString(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 3)
	buf := NewBuffer(area)

	style := cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
	buf.SetString(2, 1, "Merhaba", style)

	// "Merhaba" 7 karakter. (2, 1)'den başlayarak (8, 1)'e kadar yazar.
	expected := "Merhaba"
	for i, r := range expected {
		got := buf.Get(uint16(2+i), 1)
		if got == nil || got.Content != r {
			t.Errorf("Karakter %d eşleşmedi. Beklenen: %c", i, r)
		}
		if got.Style.Fg.Type() != cell.ColorRGB {
			t.Errorf("Karakter %d rengi TrueColor olmalıydı", i)
		}
	}

	// Sınır aşımı kontrolü (Wrap olmamalı, kesilmeli)
	buf.Clear()
	buf.SetString(7, 1, "UzunMetin", style) // (7, 1)'de başlar. Sadece "Uzu" yazabilmeli (genişlik 10)
	
	if buf.Get(9, 1).Content != 'u' {
		t.Errorf("Sınırda kesilme hatalı. (9,1) 'u' olmalı, alınan: %c", buf.Get(9, 1).Content)
	}
	
	// Geçersiz koordinatta SetString paniklememeli
	buf.SetString(20, 20, "Test", style)
}

func TestBufferResizeAllocation(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 10) // 100 hücre
	buf := NewBuffer(area)

	ptrBefore := unsafe.Pointer(&buf.Content[0])

	// Daha küçük veya eşit boyuta resize
	buf.Resize(cell.NewRect(0, 0, 5, 5)) // 25 hücre
	ptrAfter := unsafe.Pointer(&buf.Content[0])

	if ptrBefore != ptrAfter {
		t.Errorf("Kapasite yeterliyken bellek yeniden tahsis edildi (re-allocated)")
	}

	// Kapasiteyi aşan boyuta resize
	buf.Resize(cell.NewRect(0, 0, 15, 10)) // 150 hücre
	ptrNew := unsafe.Pointer(&buf.Content[0])

	if ptrBefore == ptrNew {
		t.Errorf("Kapasite aşılmasına rağmen bellek adresi değişmedi")
	}
}
