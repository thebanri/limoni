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
	rows := m.visualRows(ctx.Area.Width, baseStyle)
	// ScrollOffset is a viewport row offset. It must use the same wrapped
	// visual-row coordinate system as the renderer; using parsed source rows
	// makes long lines/header spacing clamp too early and appear stuck.
	contentRows := len(rows)
	maxOffset := maxMarkdownOffset(contentRows, int(ctx.Area.Height))
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

	for row := 0; row < int(ctx.Area.Height); row++ {
		contentRow := offset + row
		if contentRow >= len(rows) {
			break
		}
		for col, item := range rows[contentRow] {
			if col >= int(ctx.Area.Width) {
				break
			}
			buf.SetCell(ctx.Area.X+uint16(col), y+uint16(row), item)
		}
	}
}

// visualRows expands parsed markdown into the exact cell rows used by Draw.
// Keeping scrolling and rendering on this single representation prevents
// wrapped lines and header spacing from drifting apart.
func (m *Markdown) visualRows(width uint16, baseStyle cell.Style) [][]cell.Cell {
	if width == 0 {
		return nil
	}
	rows := make([][]cell.Cell, 0, len(m.cachedLines))
	blank := func() []cell.Cell { return make([]cell.Cell, 0, int(width)) }
	for _, line := range m.cachedLines {
		if line.isDivider {
			row := blank()
			style := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(100, 100, 100)})
			for len(row) < int(width) {
				row = append(row, cell.Cell{Content: '┄', Style: style})
			}
			rows = append(rows, row)
			continue
		}
		indent := 0
		row := blank()
		if line.prefix != "" {
			prefixStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(0, 255, 0)})
			for _, r := range []rune(line.prefix) {
				row = append(row, cell.Cell{Content: r, Style: prefixStyle})
			}
			indent = len(row)
		}
		for _, seg := range line.segments {
			for index, word := range seg.WordRunes {
				space := 0
				if index > 0 {
					space = 1
				}
				if len(row)+space+len(word) >= int(width) && len(row) > indent {
					for len(row) < indent {
						row = append(row, cell.Cell{Content: ' ', Style: baseStyle})
					}
					rows = append(rows, row)
					row = blank()
					for i := 0; i < indent; i++ {
						row = append(row, cell.Cell{Content: ' ', Style: baseStyle})
					}
				}
				if space == 1 && len(row) < int(width) {
					row = append(row, cell.Cell{Content: ' ', Style: seg.Style})
				}
				for _, r := range word {
					if len(row) >= int(width) {
						break
					}
					row = append(row, cell.Cell{Content: r, Style: seg.Style})
				}
			}
		}
		rows = append(rows, row)
		if line.isHeader {
			rows = append(rows, blank(), blank())
		}
	}
	return rows
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

// markdownLineRows returns the number of visual rows occupied by one parsed
// line. It mirrors visualLineCount and is used by the viewport walker.
func markdownLineRows(line markdownLine, width uint16) int {
	if width == 0 {
		return 0
	}
	if line.isDivider {
		return 1
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
	return rows
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
	width = uint16(0)
	for _, line := range m.cachedLines {
		lineWidth := cell.StringWidth(line.prefix)
		for _, segment := range line.segments {
			for _, word := range segment.Words {
				lineWidth += cell.StringWidth(word) + 1
			}
		}
		if uint16(lineWidth) > width {
			width = uint16(lineWidth)
		}
	}
	if width > maxArea.Width {
		width = maxArea.Width
	}
	return width, h
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
