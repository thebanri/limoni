package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// Paragraph, çok satırlı metinleri gösteren görsel bileşendir.
// Metinlerin alan sınırlarına göre otomatik kelime kelime aşağı kaydırılmasını (Word Wrap) destekler.
type Paragraph struct {
	// ID, widget odak kimliğidir.
	ID string
	// Text, gösterilecek olan metin içeriğidir. Yeni satır (\n) karakterlerini destekler.
	Text string
	// Style, metnin yazı rengi, arka planı ve modifikatör stillerini belirler.
	Style cell.Style
	// FocusedStyle, paragraf odaklandığında uygulanacak stildir.
	FocusedStyle cell.Style
	// Wrap, metnin sınır genişliğine göre otomatik olarak alt satıra kaydırılıp kaydırılmayacağını belirler.
	Wrap bool

	// Caching fields to avoid heap allocation on draw loops
	lastText    string
	lastWidth   uint16
	lastWrap    bool
	cachedLines []string
}

// NewParagraph creates a new Paragraph widget with wrapping enabled by default.
func NewParagraph(text string) *Paragraph {
	return &Paragraph{
		Text: text,
		Wrap: true,
	}
}

// WithWrap enables or disables word-wrapping.
func (p *Paragraph) WithWrap(wrap bool) *Paragraph {
	p.Wrap = wrap
	return p
}

// WithStyle sets the paragraph text style.
func (p *Paragraph) WithStyle(style cell.Style) *Paragraph {
	p.Style = style
	return p
}

// WithFocusedStyle sets the style applied when focused.
func (p *Paragraph) WithFocusedStyle(style cell.Style) *Paragraph {
	p.FocusedStyle = style
	return p
}

// WithID sets the widget focus and event ID.
func (p *Paragraph) WithID(id string) *Paragraph {
	p.ID = id
	return p
}

// Draw, metni çözümler, gerekliyse satır genişliğine göre böler ve terminal tamponuna çizer.
func (p *Paragraph) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	if p.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}
	if p.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(p.ID)
			}
		})
	}

	mergedStyle := ctx.Style.Merge(p.Style)
	if ctx.IsFocused(p.ID) {
		mergedStyle = mergedStyle.Merge(p.FocusedStyle)
	}

	// Metni satırlara ayır ve önbelleğe al
	if p.Text != p.lastText || area.Width != p.lastWidth || p.Wrap != p.lastWrap || p.cachedLines == nil {
		p.lastText = p.Text
		p.lastWidth = area.Width
		p.lastWrap = p.Wrap
		if p.Wrap {
			p.cachedLines = wrapText(p.Text, area.Width)
		} else {
			p.cachedLines = splitLines(p.Text)
		}
	}

	bg := mergedStyle.Bg
	if bg.Type() == cell.ColorDefault && ctx.Style.Bg.Type() != cell.ColorDefault {
		bg = ctx.Style.Bg
	}

	// Sınır yüksekliğini aşmayacak şekilde satır satır çiz
	for i, line := range p.cachedLines {
		if uint16(i) >= area.Height {
			break
		}
		currY := area.Y + uint16(i)
		written := buf.SetStringWithin(area.X, currY, line, mergedStyle, area.Width)
		if bg.Type() != cell.ColorDefault {
			for x := area.X + written; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
	if bg.Type() != cell.ColorDefault {
		for i := len(p.cachedLines); uint16(i) < area.Height; i++ {
			currY := area.Y + uint16(i)
			for x := area.X; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
}

// SizeHint, metnin kaplamak istediği en uygun genişlik ve yüksekliği raporlar.
// Düzen Pazarlığı: Eğer Wrap aktifse, verilen genişliğe (maxArea.Width) göre metnin kaç satır tutacağını hesaplar.
func (p *Paragraph) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if len(p.Text) == 0 {
		return 0, 0
	}

	// Metni satırlara ayır ve önbelleğe al
	if p.Text != p.lastText || maxArea.Width != p.lastWidth || p.Wrap != p.lastWrap || p.cachedLines == nil {
		p.lastText = p.Text
		p.lastWidth = maxArea.Width
		p.lastWrap = p.Wrap
		if p.Wrap && maxArea.Width > 0 {
			p.cachedLines = wrapText(p.Text, maxArea.Width)
		} else {
			p.cachedLines = splitLines(p.Text)
		}
	}

	// En uzun satırın genişliğini bul
	maxW := 0
	for _, line := range p.cachedLines {
		if width := cell.StringWidth(line); width > maxW {
			maxW = width
		}
	}

	w := uint16(maxW)
	h := uint16(len(p.cachedLines))

	// Üst sınırları aşma
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}

// Measure provides explicit size negotiation for Paragraph.
func (p *Paragraph) Measure(maxArea cell.Rect) layout.Measure {
	w, h := p.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
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
			wordW := cell.StringWidth(word)
			// Kelimenin kendisi tek başına satır genişliğini aşıyorsa, kelimeyi harf harf böl
			if wordW > int(width) {
				chunks := breakWord(word, int(width))
				for _, chunk := range chunks {
					chunkW := cell.StringWidth(chunk)
					if len(currentLine) == 0 {
						currentLine = chunk
					} else if cell.StringWidth(currentLine)+1+chunkW <= int(width) {
						currentLine += " " + chunk
					} else {
						wrappedLines = append(wrappedLines, currentLine)
						currentLine = chunk
					}
				}
				continue
			}

			if len(currentLine) == 0 {
				currentLine = word
			} else if cell.StringWidth(currentLine)+1+wordW <= int(width) {
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

func breakWord(word string, width int) []string {
	if width <= 0 {
		return nil
	}
	var chunks []string
	var current strings.Builder
	currentW := 0

	for _, r := range word {
		rw := cell.RuneWidth(r)
		if currentW+rw > width && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
			currentW = 0
		}
		current.WriteRune(r)
		currentW += rw
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

// splitLines, metni yeni satır (\n) karakterine göre ham satırlara ayırır (Windows \r\n dahil temizlenir).
func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			line := text[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start <= len(text) {
		line := text[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
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

// AccessibilityNode returns the semantic node description for Paragraph.
//
// Static text is invisible to a screen reader unless something announces it,
// which is why a plain paragraph carries a node at all.
func (p *Paragraph) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{
		ID:     p.ID,
		Role:   accessibility.RoleGeneric,
		Label:  "Text",
		Value:  p.Text,
		State:  state,
		Bounds: bounds,
	}
}
