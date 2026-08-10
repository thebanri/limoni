package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/backend"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type Markdown struct {
	ID string
	// Content, parse edilip çizilecek olan ham markdown metnidir.
	Content string
	// Style, varsayılan metin stilini tanımlar.
	Style        cell.Style
	FocusedStyle cell.Style
	ScrollOffset *int

	// Caching fields to avoid heap allocation on draw loops
	lastContent string
	lastStyle   cell.Style
	cachedLines []markdownLine
}

type markdownLine struct {
	isDivider bool
	isHeader  bool
	prefix    string
	segments  []StyledSegment
}

type StyledSegment struct {
	Style     cell.Style
	Words     []string
	WordRunes [][]rune // Pre-calculated runes for word wrap length calculations and printing!
}

type rawSegment struct {
	Text  string
	Style cell.Style
}

func (m *Markdown) parse(baseStyle cell.Style) {
	if m.Content == m.lastContent && m.Style == m.lastStyle {
		return
	}

	m.lastContent = m.Content
	m.lastStyle = m.Style
	m.cachedLines = nil

	lines := strings.Split(m.Content, "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// 1. Horizontal Divider: ---
		if line == "---" {
			m.cachedLines = append(m.cachedLines, markdownLine{
				isDivider: true,
			})
			continue
		}

		// 2. Headers & Lists
		lineStyle := baseStyle
		prefix := ""
		isHeader := false

		if strings.HasPrefix(line, "# ") {
			line = strings.TrimPrefix(line, "# ")
			lineStyle = baseStyle.Merge(cell.Style{
				Fg:       cell.NewColorRGB(0, 255, 255), // Cyan
				Modifier: cell.ModifierBold,
			})
			isHeader = true
		} else if strings.HasPrefix(line, "## ") {
			line = strings.TrimPrefix(line, "## ")
			lineStyle = baseStyle.Merge(cell.Style{
				Fg:       cell.NewColorRGB(0, 255, 0), // Green
				Modifier: cell.ModifierBold,
			})
			isHeader = true
		} else if strings.HasPrefix(line, "- ") {
			line = strings.TrimPrefix(line, "- ")
			prefix = "• "
		} else if strings.HasPrefix(line, "* ") {
			line = strings.TrimPrefix(line, "* ")
			prefix = "• "
		}

		rawSegments := parseInlineStyles(line, lineStyle)
		var segments []StyledSegment
		for _, rawSeg := range rawSegments {
			words := strings.Split(rawSeg.Text, " ")
			var wordRunes [][]rune
			for _, word := range words {
				wordRunes = append(wordRunes, []rune(word))
			}
			segments = append(segments, StyledSegment{
				Style:     rawSeg.Style,
				Words:     words,
				WordRunes: wordRunes,
			})
		}

		m.cachedLines = append(m.cachedLines, markdownLine{
			isHeader: isHeader,
			prefix:   prefix,
			segments: segments,
		})
	}
}

func (m *Markdown) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if m.Content == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	if m.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(m.ID)
	}
	if m.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(m.ID)
			}
		})
	}
	baseStyle := ctx.Style.Merge(m.Style)
	if m.ID != "" && ctx.FocusedID == m.ID {
		baseStyle = baseStyle.Merge(m.FocusedStyle)
	}
	m.parse(baseStyle)

	y := ctx.Area.Y
	maxY := ctx.Area.Y + ctx.Area.Height
	visualLines := m.visualLineCount(ctx.Area.Width)
	maxOffset := maxMarkdownOffset(visualLines, int(ctx.Area.Height))
	offset := 0
	if m.ScrollOffset != nil {
		offset = *m.ScrollOffset
		offset = clampMarkdownOffset(offset, maxOffset)
		*m.ScrollOffset = offset
	}
	if ctx.RegisterMouse != nil && m.ScrollOffset != nil {
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			switch ev.Button {
			case backend.MouseScrollUp:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset-1, maxOffset)
			case backend.MouseScrollDown:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset+1, maxOffset)
			case backend.MouseLeft:
				// Tıklanan alan içinde dikey sürükleme ile metni kaydır.
				// Resize tutamacı child area'nın dışında olduğu için bu handler
				// yükseklik değiştirme sürüklemesiyle çakışmaz.
				if ctx.SetFocus != nil {
					ctx.SetFocus(m.ID)
				}
				startY := int(ev.Y)
				startOffset := *m.ScrollOffset
				if ctx.CaptureMouse != nil {
					ctx.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							return
						}
						if dragEv.Drag {
							deltaY := int(dragEv.Y) - startY
							*m.ScrollOffset = clampMarkdownOffset(startOffset-deltaY, maxOffset)
						}
					})
				}
			}
		})
	}

	for lineIndex, line := range m.cachedLines {
		if lineIndex < offset {
			continue
		}
		if y >= maxY {
			break
		}

		if line.isDivider {
			divStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(100, 100, 100)})
			for col := ctx.Area.X; col < ctx.Area.X+ctx.Area.Width; col++ {
				buf.SetCell(col, y, cell.Cell{Content: '┄', Style: divStyle})
			}
			y++
			continue
		}

		currX := ctx.Area.X
		indent := uint16(0)
		if line.prefix != "" {
			buf.SetString(ctx.Area.X, y, line.prefix, baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}))
			currX += uint16(len(line.prefix))
			indent = currX - ctx.Area.X
		}

		for _, seg := range line.segments {
			for idx, wordRunes := range seg.WordRunes {
				if idx > 0 && currX < ctx.Area.X+ctx.Area.Width {
					buf.SetCell(currX, y, cell.Cell{Content: ' ', Style: seg.Style})
					currX++
				}

				wordLen := uint16(len(wordRunes))
				// Eğer kelime satıra sığmıyorsa alt satıra geç (ve indent uygula)
				if currX+wordLen >= ctx.Area.X+ctx.Area.Width {
					y++
					if y >= maxY {
						return
					}
					currX = ctx.Area.X + indent
				}

				// Kelimeyi çiz
				for _, r := range wordRunes {
					if currX >= ctx.Area.X+ctx.Area.Width {
						break
					}
					buf.SetCell(currX, y, cell.Cell{Content: r, Style: seg.Style})
					currX++
				}
			}
		}

		// Satır sonu: Eğer başlık ise altını boş bırakıp 2 satır atla
		if line.isHeader {
			y += 2
		} else {
			y++
		}
	}
}

// visualLineCount mirrors the renderer's one-cell-per-row layout, including
// wrapped words and the two blank rows reserved after headers.
func (m *Markdown) visualLineCount(width uint16) int {
	if width == 0 {
		return 0
	}
	count := 0
	for _, line := range m.cachedLines {
		if line.isDivider {
			count++
			continue
		}
		lineWidth := 0
		indent := 0
		if line.prefix != "" {
			lineWidth = len([]rune(line.prefix))
			indent = lineWidth
		}
		rows := 1
		for _, segment := range line.segments {
			for index, word := range segment.WordRunes {
				wordWidth := len(word)
				space := 0
				if index > 0 {
					space = 1
				}
				if lineWidth+space+wordWidth >= int(width) {
					rows++
					lineWidth = indent + wordWidth
				} else {
					lineWidth += space + wordWidth
				}
			}
		}
		if line.isHeader {
			rows += 2
		}
		count += rows
	}
	return count
}

func maxMarkdownOffset(lineCount, visibleHeight int) int {
	if lineCount <= 0 || visibleHeight <= 0 || lineCount <= visibleHeight {
		return 0
	}
	return lineCount - visibleHeight
}

func clampMarkdownOffset(offset, maxOffset int) int {
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func (m *Markdown) SizeHint(maxArea cell.Rect) (width, height uint16) {
	baseStyle := cell.Style{}.Merge(m.Style)
	m.parse(baseStyle)

	h := uint16(0)
	for _, line := range m.cachedLines {
		if line.isDivider {
			h++
			continue
		}
		if line.isHeader {
			h += 3
		} else {
			h++
		}
	}
	return maxArea.Width, h
}

func parseInlineStyles(text string, baseStyle cell.Style) []rawSegment {
	var segments []rawSegment
	runes := []rune(text)
	var curr []rune
	i := 0
	n := len(runes)
	style := baseStyle

	for i < n {
		if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			if len(curr) > 0 {
				segments = append(segments, rawSegment{Text: string(curr), Style: style})
				curr = nil
			}
			if (style.Modifier & cell.ModifierBold) != 0 {
				style.Modifier &= ^cell.ModifierBold
			} else {
				style.Modifier |= cell.ModifierBold
			}
			i += 2
		} else if runes[i] == '*' {
			if len(curr) > 0 {
				segments = append(segments, rawSegment{Text: string(curr), Style: style})
				curr = nil
			}
			if (style.Modifier & cell.ModifierItalic) != 0 {
				style.Modifier &= ^cell.ModifierItalic
			} else {
				style.Modifier |= cell.ModifierItalic
			}
			i++
		} else if runes[i] == '`' {
			if len(curr) > 0 {
				segments = append(segments, rawSegment{Text: string(curr), Style: style})
				curr = nil
			}
			codeStyle := baseStyle.Merge(cell.Style{
				Fg: cell.NewColorRGB(255, 100, 100),
				Bg: cell.NewColorRGB(45, 45, 45),
			})
			i++
			var codeRunes []rune
			for i < n && runes[i] != '`' {
				codeRunes = append(codeRunes, runes[i])
				i++
			}
			if i < n {
				i++
			}
			segments = append(segments, rawSegment{Text: string(codeRunes), Style: codeStyle})
		} else {
			curr = append(curr, runes[i])
			i++
		}
	}
	if len(curr) > 0 {
		segments = append(segments, rawSegment{Text: string(curr), Style: style})
	}
	return segments
}
