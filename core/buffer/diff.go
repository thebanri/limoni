package buffer

import (
	"strconv"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// Diff front (mevcut çizilen) ve back (ekranda olan) tamponları karşılaştırır.
// Sadece değişen hücreleri ve stildeki değişimleri tespit ederek, minimum ANSI escape kodunu
// `out` byte dilimine (slice) ekler ve güncellenmiş dilimi döner.
// Bellek Optimizasyonu: Eğer `out` yeterli kapasiteye sahipse sıfır heap bellek tahsisatı (zero-allocation) ile çalışır.
func Diff(front, back *Buffer, out []byte) ([]byte, error) {
	// Boyutlar uyuşmuyorsa, ekranı temizle ve back tamponu yeniden boyutlandır
	if front.Area.Width != back.Area.Width || front.Area.Height != back.Area.Height {
		back.Resize(front.Area)
		out = append(out, "\x1b[2J"...) // Ekranı temizle
	}

	width := front.Area.Width
	height := front.Area.Height

	var currentStyle cell.Style
	currentStyle.Reset()

	cursorX := uint16(9999) // Geçersiz başlangıç konumu (imleci ilk çizimde zorla konumlandırmak için)
	cursorY := uint16(9999)

	for y := uint16(0); y < height; y++ {
		for x := uint16(0); x < width; x++ {
			idx := int(y)*int(width) + int(x)
			frontCell := &front.Content[idx]
			backCell := &back.Content[idx]

			// Hücre içeriği ve stili tamamen aynıysa atla
			if frontCell.Content == backCell.Content && frontCell.Style == backCell.Style {
				continue
			}

			// Eğer bu hücre bir geniş karakterin devamı (continuation) ise terminale yazma,
			// ancak durum eşitlemesi için backCell'i güncelle ve cursor'ı ilerlet.
			if frontCell.Content == cell.RuneContinuation {
				*backCell = *frontCell
				cursorX++
				if cursorX >= width {
					cursorX = 9999
					cursorY = 9999
				}
				continue
			}

			// Eğer bu hücre bir native resim hücresi ise terminale yazma,
			// ancak durum eşitlemesi için backCell'i güncelle ve cursor'ı ilerlet.
			if frontCell.Content == cell.RuneImage {
				*backCell = *frontCell
				cursorX++
				if cursorX >= width {
					cursorX = 9999
					cursorY = 9999
				}
				continue
			}

			// İmleç doğru konumda değilse konumlandır
			if cursorX != x || cursorY != y {
				out = appendCursor(out, x, y)
				cursorX = x
				cursorY = y
			}

			// Stil güncellenmeli mi?
			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style)
			}

			// Karakteri yaz
			if frontCell.Content == ' ' || frontCell.Content == 0 {
				out = append(out, ' ')
			} else {
				out = utf8.AppendRune(out, frontCell.Content)
			}

			// İmleç pozisyonunu güncelle (terminal karakter yazdıktan sonra sağa kayar)
			cursorX++
			if cursorX >= width {
				// Satır sonuna ulaşıldığında otomatik wrap riskini önlemek için imleç takibini geçersiz kıl
				cursorX = 9999
				cursorY = 9999
			}

			// Back hücresini güncelle ki bir sonraki karede fark olmasın
			*backCell = *frontCell
		}
	}

	// Kare sonunda terminal stilini varsayılana sıfırla (terminal kirlenmesini önlemek için)
	var defaultStyle cell.Style
	defaultStyle.Reset()
	if currentStyle != defaultStyle {
		out, _ = appendStyle(out, currentStyle, defaultStyle)
	}

	return out, nil
}

// appendCursor imleç konumlandırma ANSI escape kodunu ekler. (\x1b[row;colH)
func appendCursor(out []byte, x, y uint16) []byte {
	out = append(out, "\x1b["...)
	out = strconv.AppendInt(out, int64(y+1), 10)
	out = append(out, ';')
	out = strconv.AppendInt(out, int64(x+1), 10)
	return append(out, 'H')
}

// appendStyle stildeki değişiklikleri analiz eder ve sadece değişen kısımlar için ANSI kodlarını ekler.
// Performans: Sıfır tahsisat sağlamak için closure kullanılmadan tamamen düz inline olarak yazılmıştır.
func appendStyle(out []byte, cur, target cell.Style) ([]byte, cell.Style) {
	if cur == target {
		return out, cur
	}

	// 1. Modifikatörlerden biri kaldırılmış mı?
	// Eğer hedef stilde, mevcut stilde olan bir özellik eksikse (örn. Bold'dan normal yazıya geçiş),
	// tek tek modifikatör silme kodu olmadığından tam reset (\x1b[0m) gerekir.
	if (cur.Modifier & ^target.Modifier) != 0 {
		out = append(out, "\x1b[0m"...)
		cur.Reset() // Mevcut stil sıfırlandı
	}

	// 2. Ön Plan (Foreground) Rengi Değişimi
	if cur.Fg != target.Fg {
		switch target.Fg.Type() {
		case cell.ColorDefault:
			out = append(out, "\x1b[39m"...)
		case cell.ColorANSI:
			out = append(out, "\x1b[38;5;"...)
			out = strconv.AppendInt(out, int64(target.Fg.ANSI()), 10)
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Fg.RGB()
			out = append(out, "\x1b[38;2;"...)
			out = strconv.AppendInt(out, int64(r), 10)
			out = append(out, ';')
			out = strconv.AppendInt(out, int64(g), 10)
			out = append(out, ';')
			out = strconv.AppendInt(out, int64(b), 10)
			out = append(out, 'm')
		}
		cur.Fg = target.Fg
	}

	// 3. Arka Plan (Background) Rengi Değişimi
	if cur.Bg != target.Bg {
		switch target.Bg.Type() {
		case cell.ColorDefault:
			out = append(out, "\x1b[49m"...)
		case cell.ColorANSI:
			out = append(out, "\x1b[48;5;"...)
			out = strconv.AppendInt(out, int64(target.Bg.ANSI()), 10)
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Bg.RGB()
			out = append(out, "\x1b[48;2;"...)
			out = strconv.AppendInt(out, int64(r), 10)
			out = append(out, ';')
			out = strconv.AppendInt(out, int64(g), 10)
			out = append(out, ';')
			out = strconv.AppendInt(out, int64(b), 10)
			out = append(out, 'm')
		}
		cur.Bg = target.Bg
	}

	// 4. Yeni Eklenen Modifikatörler
	added := target.Modifier & ^cur.Modifier
	if added != 0 {
		out = append(out, "\x1b["...)
		first := true

		if (added & cell.ModifierBold) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 1, 10)
			first = false
		}
		if (added & cell.ModifierDim) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 2, 10)
			first = false
		}
		if (added & cell.ModifierItalic) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 3, 10)
			first = false
		}
		if (added & cell.ModifierUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 4, 10)
			first = false
		}
		if (added & cell.ModifierBlink) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 5, 10)
			first = false
		}
		if (added & cell.ModifierReverse) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 7, 10)
			first = false
		}
		if (added & cell.ModifierHidden) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 8, 10)
			first = false
		}
		if (added & cell.ModifierStrikethrough) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 9, 10)
			first = false
		}
		if (added & cell.ModifierDoubleUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = strconv.AppendInt(out, 21, 10)
			first = false
		}
		if (added & cell.ModifierUndercurl) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, "4:3"...)
			first = false
		}

		out = append(out, 'm')
		cur.Modifier |= added
	}

	return out, cur
}
