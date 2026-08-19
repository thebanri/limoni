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
func Diff(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
	// Sıfır-Döngü Hızlı Yol (Zero-Loop Fast-Path): Eğer tamponda hiç değişiklik yapılmadıysa doğrudan dön
	if !front.IsDirty && front.Area.Width == back.Area.Width && front.Area.Height == back.Area.Height {
		return out, nil
	}

	// Hızlı Yol (Fast-Path): Tamponlar tamamen aynıysa hiçbir işlem yapma
	if front.Area.Width == back.Area.Width && front.Area.Height == back.Area.Height {
		identical := true
		for i := range front.Content {
			if front.Content[i] != back.Content[i] {
				identical = false
				break
			}
		}
		if identical {
			front.IsDirty = false
			return out, nil
		}
	}

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
		first := int(-1)
		last := int(-1)
		rowOffset := int(y)*int(width)
		for x := 0; x < int(width); x++ {
			idx := rowOffset + x
			if front.Content[idx] != back.Content[idx] {
				if first == -1 {
					first = x
				}
				last = x
			}
		}
		if first == -1 {
			continue // Bu satırda hiçbir değişiklik yok!
		}

		for x := uint16(first); x <= uint16(last); x++ {
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

			// Eğer bu hücre bir native resim hücresi ise:
			// Önceki karede bu hücrede bir diyalog/metin karakteri varsa,
			// \x1b[0m ile stili sıfırlayıp ECMA-48 ECH (\x1b[<count>X) ile eski karakterleri
			// tek hamlede sil. Böylece resmin üzerinde hiçbir hayalet çizgi kalmaz.
			if frontCell.Content == cell.RuneImage {
				spanEnd := x
				needsErase := false
				for checkX := x; checkX <= uint16(last); checkX++ {
					cIdx := int(y)*int(width) + int(checkX)
					if front.Content[cIdx].Content != cell.RuneImage {
						break
					}
					spanEnd = checkX
					if back.Content[cIdx].Content != cell.RuneImage {
						needsErase = true
					}
				}

				count := int(spanEnd - x + 1)
				if needsErase {
					out = appendCursor(out, x, y)
					cursorX = 9999
					cursorY = 9999
					if currentStyle != (cell.Style{}) {
						out = append(out, "\x1b[0m"...)
						currentStyle.Reset()
					}
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(count), 10)
					out = append(out, 'X')
				} else {
					cursorX = 9999
					cursorY = 9999
				}

				for setX := x; setX <= spanEnd; setX++ {
					cIdx := int(y)*int(width) + int(setX)
					back.Content[cIdx] = front.Content[cIdx]
				}

				x = spanEnd
				continue
			}

			// İmleç doğru konumda değilse konumlandır
			if cursorX != x || cursorY != y {
				if cursorY == y && x > cursorX && (x-cursorX) < 4 {
					for skipX := cursorX; skipX < x; skipX++ {
						skipIdx := int(y)*int(width) + int(skipX)
						skipCell := &front.Content[skipIdx]
						if skipCell.Style != currentStyle {
							out, currentStyle = appendStyle(out, currentStyle, skipCell.Style, trueColor, colors256, front.StyleCache)
						}
						if skipCell.Content == ' ' || skipCell.Content == 0 {
							out = append(out, ' ')
						} else {
							out = utf8.AppendRune(out, skipCell.Content)
						}
						back.Content[skipIdx] = *skipCell
					}
					cursorX = x
				} else {
					out = appendCursor(out, x, y)
					cursorX = x
					cursorY = y
				}
			}

			// Stil güncellenmeli mi?
			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, front.StyleCache)
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
		out, _ = appendStyle(out, currentStyle, defaultStyle, trueColor, colors256, front.StyleCache)
	}

	front.IsDirty = false
	return out, nil
}

// appendCursor imleç konumlandırma ANSI escape kodunu ekler. (\x1b[row;colH)
func appendCursor(out []byte, x, y uint16) []byte {
	out = append(out, "\x1b["...)
	out = appendUint16(out, y+1)
	out = append(out, ';')
	out = appendUint16(out, x+1)
	return append(out, 'H')
}

func getStyleBytes(target cell.Style, trueColor, colors256 bool, cache map[cell.Style][]byte) []byte {
	if cache == nil {
		var out []byte
		var cur cell.Style
		cur.Reset()
		out, _ = appendStyleRaw(out, cur, target, trueColor, colors256)
		return out
	}
	if bytes, ok := cache[target]; ok {
		return bytes
	}

	// Format style starting from default
	var out []byte
	var cur cell.Style
	cur.Reset()
	out, _ = appendStyleRaw(out, cur, target, trueColor, colors256)

	cache[target] = out
	return out
}

func appendStyle(out []byte, cur, target cell.Style, trueColor, colors256 bool, cache map[cell.Style][]byte) ([]byte, cell.Style) {
	if !trueColor {
		target = target.Downsample(trueColor, colors256)
	}
	if cur == target {
		return out, cur
	}

	// If we are resetting anyway, we can use the cached target bytes directly!
	if (cur.Modifier & ^target.Modifier) != 0 {
		out = append(out, "\x1b[0m"...)
		cached := getStyleBytes(target, trueColor, colors256, cache)
		out = append(out, cached...)
		return out, target
	}

	// Otherwise, do standard incremental diff
	return appendStyleRaw(out, cur, target, trueColor, colors256)
}

// appendStyleRaw stildeki değişiklikleri analiz eder ve sadece değişen kısımlar için ANSI kodlarını ekler.
func appendStyleRaw(out []byte, cur, target cell.Style, trueColor, colors256 bool) ([]byte, cell.Style) {
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
			out = appendUint8(out, target.Fg.ANSI())
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Fg.RGB()
			out = append(out, "\x1b[38;2;"...)
			out = appendUint8(out, r)
			out = append(out, ';')
			out = appendUint8(out, g)
			out = append(out, ';')
			out = appendUint8(out, b)
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
			out = appendUint8(out, target.Bg.ANSI())
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Bg.RGB()
			out = append(out, "\x1b[48;2;"...)
			out = appendUint8(out, r)
			out = append(out, ';')
			out = appendUint8(out, g)
			out = append(out, ';')
			out = appendUint8(out, b)
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
			out = append(out, '1')
			first = false
		}
		if (added & cell.ModifierDim) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '2')
			first = false
		}
		if (added & cell.ModifierItalic) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '3')
			first = false
		}
		if (added & cell.ModifierUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '4')
			first = false
		}
		if (added & cell.ModifierBlink) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '5')
			first = false
		}
		if (added & cell.ModifierReverse) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '7')
			first = false
		}
		if (added & cell.ModifierHidden) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '8')
			first = false
		}
		if (added & cell.ModifierStrikethrough) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '9')
			first = false
		}
		if (added & cell.ModifierDoubleUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, "21"...)
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

func appendUint8(out []byte, v uint8) []byte {
	if v >= 100 {
		return append(out, '0'+v/100, '0'+(v/10)%10, '0'+v%10)
	}
	if v >= 10 {
		return append(out, '0'+v/10, '0'+v%10)
	}
	return append(out, '0'+v)
}

func appendUint16(out []byte, v uint16) []byte {
	if v >= 10000 {
		return append(out, '0'+byte(v/10000), '0'+byte((v/1000)%10), '0'+byte((v/100)%10), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 1000 {
		return append(out, '0'+byte(v/1000), '0'+byte((v/100)%10), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 100 {
		return append(out, '0'+byte(v/100), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 10 {
		return append(out, '0'+byte(v/10), '0'+byte(v%10))
	}
	return append(out, '0'+byte(v))
}
