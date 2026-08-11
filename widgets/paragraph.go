package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Paragraph, çok satırlı metinleri gösteren görsel bileşendir.
// Metinlerin alan sınırlarına göre otomatik kelime kelime aşağı kaydırılmasını (Word Wrap) destekler.
type Paragraph struct {
	// Text, gösterilecek olan metin içeriğidir. Yeni satır (\n) karakterlerini destekler.
	Text string
	// Style, metnin yazı rengi, arka planı ve modifikatör stillerini belirler.
	Style cell.Style
	// Wrap, metnin sınır genişliğine göre otomatik olarak alt satıra kaydırılıp kaydırılmayacağını belirler.
	Wrap bool
}

// Draw, metni çözümler, gerekliyse satır genişliğine göre böler ve terminal tamponuna çizer.
func (p Paragraph) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	mergedStyle := ctx.Style.Merge(p.Style)

	// Metni satırlara ayır
	var lines []string
	if p.Wrap {
		lines = wrapText(p.Text, area.Width)
	} else {
		lines = splitLines(p.Text)
	}

	// Sınır yüksekliğini aşmayacak şekilde satır satır çiz
	for i, line := range lines {
		if uint16(i) >= area.Height {
			break
		}
		buf.SetString(area.X, area.Y+uint16(i), line, mergedStyle)
	}
}

// SizeHint, metnin kaplamak istediği en uygun genişlik ve yüksekliği raporlar.
// Düzen Pazarlığı: Eğer Wrap aktifse, verilen genişliğe (maxArea.Width) göre metnin kaç satır tutacağını hesaplar.
func (p Paragraph) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if len(p.Text) == 0 {
		return 0, 0
	}

	var lines []string
	if p.Wrap && maxArea.Width > 0 {
		lines = wrapText(p.Text, maxArea.Width)
	} else {
		lines = splitLines(p.Text)
	}

	// En uzun satırın genişliğini bul
	maxW := 0
	for _, line := range lines {
		if width := cell.StringWidth(line); width > maxW {
			maxW = width
		}
	}

	w := uint16(maxW)
	h := uint16(len(lines))

	// Üst sınırları aşma
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}

// wrapText, uzun bir metni kelime sınırlarından bölerek satır genişliğini (width) aşmayacak şekilde satırlara ayırır.
func wrapText(text string, width uint16) []string {
	if width == 0 {
		return nil
	}

	var wrappedLines []string
	rawLines := splitLines(text)

	for _, rawLine := range rawLines {
		if len(rawLine) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := splitWords(rawLine)
		var currentLine string

		for _, word := range words {
			if len(currentLine) == 0 {
				currentLine = word
			} else if cell.StringWidth(currentLine)+1+cell.StringWidth(word) <= int(width) {
				currentLine += " " + word
			} else {
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = word
			}
		}
		if len(currentLine) > 0 {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}

	return wrappedLines
}

// splitLines, metni yeni satır (\n) karakterine göre ham satırlara ayırır.
func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start <= len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// splitWords, bir satırı boşluk karakterlerine göre kelimelere böler.
func splitWords(s string) []string {
	var words []string
	start := -1
	for i, r := range s {
		if r == ' ' {
			if start != -1 {
				words = append(words, s[start:i])
				start = -1
			}
		} else {
			if start == -1 {
				start = i
			}
		}
	}
	if start != -1 {
		words = append(words, s[start:])
	}
	return words
}
