package buffer

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// Buffer terminal ekranının hücresel ızgarasını temsil eder.
// 2D slice yerine 1D flat slice (`[]cell.Cell`) kullanarak bellek lekelerini ve cache miss oranlarını minimize eder.
type Buffer struct {
	Area       cell.Rect             // Tamponun kapladığı alan koordinatları
	Content    []cell.Cell           // Bellekte ardışık duran hücre dilimi
	IsDirty    bool                  // Tamponda değişiklik yapılıp yapılmadığını gösterir
	StyleCache map[cell.Style][]byte // Stil geçiş kodları için önbellek

	// clean, tamponun tüm hücrelerinin varsayılan (boşluk + sıfır stil) durumda
	// olduğunun bilindiğini belirtir. Bu bayrak sayesinde hiçbir widget çizilmeyen
	// karelerde Clear() tüm hücre dizisini taramak zorunda kalmaz.
	clean bool
}

// NewBuffer belirtilen alan boyutunda yeni bir Buffer oluşturur.
func NewBuffer(area cell.Rect) *Buffer {
	needed := int(area.Width) * int(area.Height)
	content := make([]cell.Cell, needed)
	b := &Buffer{
		Area:       area,
		Content:    content,
		IsDirty:    true,
		StyleCache: make(map[cell.Style][]byte),
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
	if b.clean {
		return
	}
	for i := range b.Content {
		if b.Content[i].Content != ' ' || b.Content[i].Style != (cell.Style{}) {
			b.Content[i].Reset()
			b.IsDirty = true
		}
	}
	b.clean = true
}

// Invalidate, tampon içeriğinin Content dilimi üzerinden doğrudan değiştirildiğini bildirir.
// Content'i SetCell/SetString dışında değiştiren çağrıcılar bu metodu kullanmalıdır;
// aksi halde Clear() hızlı yolu güncel olmayan hücreleri temizlemeyi atlar.
func (b *Buffer) Invalidate() {
	b.clean = false
	b.IsDirty = true
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
	b.IsDirty = true
	b.clean = false
	b.Clear()
}

// Get koordinattaki hücreye doğrudan işaretçi (pointer) döner.
// Dönen işaretçi üzerinden doğrudan hücre değerleri değiştirilebilir. Geçersiz koordinatta `nil` döner.
func (b *Buffer) Get(x, y uint16) *cell.Cell {
	if x >= b.Area.Width || y >= b.Area.Height {
		return nil
	}
	// Dönen işaretçi üzerinden hücre doğrudan değiştirilebileceği için tampon
	// artık "temiz" kabul edilemez.
	b.clean = false
	return &b.Content[y*b.Area.Width+x]
}

// SetCell belirtilen koordinattaki hücreyi doğrudan değiştirir.
func (b *Buffer) SetCell(x, y uint16, c cell.Cell) {
	if x >= b.Area.Width || y >= b.Area.Height {
		return
	}
	idx := int(y)*int(b.Area.Width) + int(x)
	if b.Content[idx] != c {
		b.Content[idx] = c
		b.IsDirty = true
		b.clean = false
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

		w := cell.RuneWidth(r)
		if w == 0 {
			input = input[size:]
			continue // Sıfır genişlikli birleştirici karakterleri atla, önceki hücreyi ezme
		}
		if currX+uint16(w) > b.Area.Width {
			break // Sınır dışına taşmayı engelle
		}

		idx := y*b.Area.Width + currX
		if b.Content[idx].Content != r || b.Content[idx].Style != style {
			b.Content[idx].Content = r
			b.Content[idx].Style = style
			b.IsDirty = true
			b.clean = false
		}

		if w == 2 {
			if b.Content[idx+1].Content != cell.RuneContinuation {
				b.Content[idx+1].Content = cell.RuneContinuation
				b.IsDirty = true
				b.clean = false
			}
		}

		currX += uint16(w)
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
