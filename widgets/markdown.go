package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type Markdown struct {
	// Content, parse edilip çizilecek olan ham markdown metnidir.
	Content string
	// Style, varsayılan metin stilini tanımlar.
	Style cell.Style
}

type StyledSegment struct {
	Text  string
	Style cell.Style
}

func (m Markdown) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if m.Content == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	baseStyle := ctx.Style.Merge(m.Style)
	lines := strings.Split(m.Content, "\n")
	y := ctx.Area.Y
	maxY := ctx.Area.Y + ctx.Area.Height

	for _, rawLine := range lines {
		if y >= maxY {
			break
		}

		line := strings.TrimSpace(rawLine)

		// 1. Horizontal Divider: ---
		if line == "---" {
			divStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(100, 100, 100)})
			for col := ctx.Area.X; col < ctx.Area.X+ctx.Area.Width; col++ {
				buf.SetCell(col, y, cell.Cell{Content: '┄', Style: divStyle})
			}
			y++
			continue
		}

		// 2. Headers: # Title, ## Title
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

		segments := parseInlineStyles(line, lineStyle)

		// Word Wrap ve Çizim
		currX := ctx.Area.X
		indent := uint16(0)
		if prefix != "" {
			buf.SetString(ctx.Area.X, y, prefix, baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}))
			currX += uint16(len(prefix))
			indent = currX - ctx.Area.X
		}

		for _, seg := range segments {
			words := strings.Split(seg.Text, " ")
			for idx, word := range words {
				if idx > 0 && currX < ctx.Area.X+ctx.Area.Width {
					buf.SetCell(currX, y, cell.Cell{Content: ' ', Style: seg.Style})
					currX++
				}

				wordLen := uint16(len([]rune(word)))
				// Eğer kelime satıra sığmıyorsa alt satıra geç (ve indent uygula)
				if currX+wordLen >= ctx.Area.X+ctx.Area.Width {
					y++
					if y >= maxY {
						return
					}
					currX = ctx.Area.X + indent
				}

				// Kelimeyi çiz
				for _, r := range word {
					if currX >= ctx.Area.X+ctx.Area.Width {
						break
					}
					buf.SetCell(currX, y, cell.Cell{Content: r, Style: seg.Style})
					currX++
				}
			}
		}

		// Satır sonu: Eğer başlık ise altını boş bırakıp 2 satır atla
		if isHeader {
			y += 2
		} else {
			y++
		}
	}
}

func (m Markdown) SizeHint(maxArea cell.Rect) (width, height uint16) {
	lines := strings.Split(m.Content, "\n")
	h := uint16(len(lines))
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") || strings.HasPrefix(l, "## ") {
			h += 2
		}
	}
	return maxArea.Width, h
}

func parseInlineStyles(text string, baseStyle cell.Style) []StyledSegment {
	var segments []StyledSegment
	runes := []rune(text)
	var curr []rune
	i := 0
	n := len(runes)
	style := baseStyle

	for i < n {
		if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			if len(curr) > 0 {
				segments = append(segments, StyledSegment{Text: string(curr), Style: style})
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
				segments = append(segments, StyledSegment{Text: string(curr), Style: style})
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
				segments = append(segments, StyledSegment{Text: string(curr), Style: style})
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
			segments = append(segments, StyledSegment{Text: string(codeRunes), Style: codeStyle})
		} else {
			curr = append(curr, runes[i])
			i++
		}
	}
	if len(curr) > 0 {
		segments = append(segments, StyledSegment{Text: string(curr), Style: style})
	}
	return segments
}
