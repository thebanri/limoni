package buffer

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// Buffer terminal ekranının hücresel ızgarasını temsil eder.
// 2D slice yerine 1D flat slice (`[]cell.Cell`) kullanarak bellek lekelerini ve cache miss oranlarını minimize eder.
type Buffer struct {
	Area    cell.Rect   // Tamponun kapladığı alan koordinatları
	Content []cell.Cell // Bellekte ardışık duran hücre dilimi
}

// NewBuffer belirtilen alan boyutunda yeni bir Buffer oluşturur.
func NewBuffer(area cell.Rect) *Buffer {
	needed := int(area.Width) * int(area.Height)
	content := make([]cell.Cell, needed)
	b := &Buffer{
		Area:    area,
		Content: content,
	}
	b.Clear()
	return b
}

// NewEmptyBuffer boş (0x0 boyutunda) bir Buffer oluşturur.
func NewEmptyBuffer() *Buffer {
	return NewBuffer(cell.Rect{})
}

// Clear tüm tamponu temizler ve varsayılan hücre değerlerine (boşluk, default style) sıfırlar.
func (b *Buffer) Clear() {
	for i := range b.Content {
		b.Content[i].Reset()
	}
}

// Resize tampon alanını yeniden boyutlandırır.
// Sıfır-Tahsisat (Zero-Allocation) Optimizasyonu: Eğer mevcut kapasite (capacity) yeni boyut için yeterliyse,
// yeni bellek tahsisatı yapmadan dilimi (slice) yeniden dilimler (re-slice).
func (b *Buffer) Resize(area cell.Rect) {
	b.Area = area
	needed := int(area.Width) * int(area.Height)
	if cap(b.Content) >= needed {
		b.Content = b.Content[:needed]
	} else {
		// Yetersiz kapasite durumunda yeni alan tahsis edilir
		b.Content = make([]cell.Cell, needed)
	}
	b.Clear()
}

// Get koordinattaki hücreye doğrudan işaretçi (pointer) döner.
// Dönen işaretçi üzerinden doğrudan hücre değerleri değiştirilebilir. Geçersiz koordinatta `nil` döner.
func (b *Buffer) Get(x, y uint16) *cell.Cell {
	if x >= b.Area.Width || y >= b.Area.Height {
		return nil
	}
	return &b.Content[y*b.Area.Width+x]
}

// SetCell belirtilen koordinattaki hücreyi doğrudan değiştirir.
func (b *Buffer) SetCell(x, y uint16, c cell.Cell) {
	if idx := b.index(x, y); idx != -1 {
		b.Content[idx] = c
	}
}

// SetString belirtilen koordinattan başlayarak bir metni (string) yazdırır.
// Satır sonuna gelindiğinde yazma işlemi otomatik olarak kesilir (wrap yapılmaz, widget seviyesinde çözülür).
func (b *Buffer) SetString(x, y uint16, s string, style cell.Style) {
	if y >= b.Area.Height || x >= b.Area.Width {
		return
	}

	currX := x
	input := s
	for len(input) > 0 && currX < b.Area.Width {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError {
			break
		}

		idx := y*b.Area.Width + currX
		b.Content[idx].Content = r
		b.Content[idx].Style = style

		currX++
		input = input[size:]
	}
}

// index koordinatı flat slice indeksine dönüştürür. Sınır dışı ise -1 döner.
func (b *Buffer) index(x, y uint16) int {
	if x >= b.Area.Width || y >= b.Area.Height {
		return -1
	}
	return int(y)*int(b.Area.Width) + int(x)
}
