package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/driver"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
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
	lastContent   string
	lastStyle     cell.Style
	lastWidth     uint16
	lastBaseStyle cell.Style
	lastLinks     bool
	cachedLines   []markdownLine
	cachedRows    [][]cell.Cell
}

// NewMarkdown creates a new Markdown widget.
func NewMarkdown(content string) *Markdown {
	return &Markdown{
		Content: content,
	}
}

// WithStyle sets the default markdown text style.
func (m *Markdown) WithStyle(s cell.Style) *Markdown {
	m.Style = s
	return m
}

// WithFocusedStyle sets the focused markdown style.
func (m *Markdown) WithFocusedStyle(s cell.Style) *Markdown {
	m.FocusedStyle = s
	return m
}

// WithID sets the markdown widget ID.
func (m *Markdown) WithID(id string) *Markdown {
	m.ID = id
	return m
}

// WithScrollOffset binds an external scroll offset pointer.
func (m *Markdown) WithScrollOffset(offset *int) *Markdown {
	m.ScrollOffset = offset
	return m
}

// appendClusters appends text to row one grapheme cluster per cell, a wide
// cluster followed by its continuation cell, and stops at width.
func appendClusters(row []cell.Cell, text string, style cell.Style, width int) []cell.Cell {
	for rest := text; rest != ""; {
		cluster, w, next := cell.NextCluster(rest)
		rest = next
		if w == 0 {
			continue
		}
		if len(row)+w > width {
			break
		}
		row = append(row, cell.Cell{Content: cell.ClusterContent(cluster, w), Style: style})
		if w == 2 {
			row = append(row, cell.Cell{Content: cell.RuneContinuation, Style: style})
		}
	}
	return row
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
	WordRunes [][]rune // the words as runes; kept for callers, the layout reads Words
	// WordWidths are the words' widths in columns, measured by grapheme
	// cluster once at parse time: a combining accent adds nothing and a ZWJ
	// family emoji is two columns, not six.
	WordWidths []int
}

type rawSegment struct {
	Text  string
	Style cell.Style
}

// parse turns the source into styled lines. links says whether the terminal
// can show OSC 8 hyperlinks, which changes the output: with them, `[text](url)`
// renders as a clickable "text"; without them the address is written out too,
// because a reader who cannot click it still needs to see where it points.
func (m *Markdown) parse(baseStyle cell.Style, links bool) {
	if m.Content == m.lastContent && m.Style == m.lastStyle && baseStyle == m.lastBaseStyle &&
		links == m.lastLinks && m.cachedLines != nil {
		return
	}

	m.lastContent = m.Content
	m.lastStyle = m.Style
	m.lastBaseStyle = baseStyle
	m.lastLinks = links
	m.lastWidth = 0
	m.cachedLines = nil
	m.cachedRows = nil

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

		rawSegments := parseInlineStyles(line, lineStyle, links)
		var segments []StyledSegment
		for _, rawSeg := range rawSegments {
			words := strings.Split(rawSeg.Text, " ")
			var wordRunes [][]rune
			widths := make([]int, len(words))
			for i, word := range words {
				wordRunes = append(wordRunes, []rune(word))
				widths[i] = cell.StringWidth(word)
			}
			segments = append(segments, StyledSegment{
				Style:      rawSeg.Style,
				Words:      words,
				WordRunes:  wordRunes,
				WordWidths: widths,
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
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && m.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: m.ID})
	} else if m.ID != "" && ctx.RegisterClick != nil {
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
	if baseStyle.Bg.Type() == cell.ColorDefault && ctx.ThemeStyle != nil {
		if surf := ctx.ThemeStyle("surface"); surf.Bg.Type() != cell.ColorDefault {
			baseStyle.Bg = surf.Bg
		} else if base := ctx.ThemeStyle("base"); base.Bg.Type() != cell.ColorDefault {
			baseStyle.Bg = base.Bg
		}
	}
	m.parse(baseStyle, ctx.Hyperlinks)

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
		ctx.RegisterMouse(ctx.Area, func(ev driver.MouseEvent) {
			switch ev.Button {
			case driver.MouseScrollUp:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset-1, maxOffset)
			case driver.MouseScrollDown:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset+1, maxOffset)
			case driver.MouseLeft:
				// Tıklanan alan içinde dikey sürükleme ile metni kaydır.
				// Resize tutamacı child area'nın dışında olduğu için bu handler
				// yükseklik değiştirme sürüklemesiyle çakışmaz.
				if ctx.SetFocus != nil {
					ctx.SetFocus(m.ID)
				}
				startY := int(ev.Y)
				startOffset := *m.ScrollOffset
				if ctx.CaptureMouse != nil {
					ctx.CaptureMouse(func(dragEv driver.MouseEvent) {
						if dragEv.Button == driver.MouseRelease {
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
		var rowCells []cell.Cell
		if contentRow < len(rows) {
			rowCells = rows[contentRow]
		}
		for col := 0; col < int(ctx.Area.Width); col++ {
			if col < len(rowCells) {
				buf.SetCellDirect(ctx.Area.X+uint16(col), y+uint16(row), rowCells[col])
			} else if baseStyle.Bg.Type() != cell.ColorDefault {
				buf.SetCellDirect(ctx.Area.X+uint16(col), y+uint16(row), cell.Cell{
					Content: ' ',
					Style:   baseStyle,
				})
			}
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
	if width == m.lastWidth && baseStyle == m.lastBaseStyle && m.cachedRows != nil {
		return m.cachedRows
	}
	m.lastWidth = width
	m.lastBaseStyle = baseStyle

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
			row = appendClusters(row, line.prefix, prefixStyle, int(width))
			indent = len(row)
		}
		for _, seg := range line.segments {
			for index, word := range seg.Words {
				wordWidth := seg.WordWidths[index]
				space := 0
				if index > 0 {
					space = 1
				}
				if len(row)+space+wordWidth >= int(width) && len(row) > indent {
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
				row = appendClusters(row, word, seg.Style, int(width))
			}
		}
		rows = append(rows, row)
		if line.isHeader {
			rows = append(rows, blank(), blank())
		}
	}
	m.cachedRows = rows
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
			lineWidth = cell.StringWidth(line.prefix)
			indent = lineWidth
		}
		rows := 1
		for _, segment := range line.segments {
			for index := range segment.Words {
				wordWidth := segment.WordWidths[index]
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
		lineWidth = cell.StringWidth(line.prefix)
		indent = lineWidth
	}
	rows := 1
	for _, segment := range line.segments {
		for index := range segment.Words {
			wordWidth := segment.WordWidths[index]
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
	// SizeHint has no draw context, so it measures with whatever the last
	// draw used. Before the first draw that is the wider form, with the URLs
	// written out, which errs towards asking for too much room rather than
	// too little.
	m.parse(baseStyle, m.lastLinks)

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

// Measure provides explicit size negotiation for Markdown.
func (m *Markdown) Measure(maxArea cell.Rect) layout.Measure {
	w, h := m.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

// markdownLinkStyle is how a link is drawn whether or not the terminal can
// make it clickable, so that link text is recognisable either way.
func markdownLinkStyle(baseStyle cell.Style) cell.Style {
	return baseStyle.Merge(cell.Style{
		Fg:       cell.NewColorRGB(100, 160, 255),
		Modifier: cell.ModifierUnderline,
	})
}

// parseMarkdownLink reads `[label](url)` starting at the opening bracket. It
// returns ok=false for anything that is not a complete link — a lone bracket,
// a reference-style link, an unclosed URL — which is then drawn as the
// literal text it is.
func parseMarkdownLink(runes []rune, start int) (label, url string, next int, ok bool) {
	n := len(runes)
	i := start + 1
	labelStart := i
	for i < n && runes[i] != ']' {
		if runes[i] == '[' || runes[i] == '\n' {
			return "", "", 0, false
		}
		i++
	}
	if i >= n || i+1 >= n || runes[i+1] != '(' {
		return "", "", 0, false
	}
	label = string(runes[labelStart:i])
	i += 2
	urlStart := i
	for i < n && runes[i] != ')' {
		if runes[i] == ' ' {
			// `[text](url "title")` — the title is not rendered, and a bare
			// space in a URL means this is not one.
			return "", "", 0, false
		}
		i++
	}
	if i >= n || label == "" || i == urlStart {
		return "", "", 0, false
	}
	return label, string(runes[urlStart:i]), i + 1, true
}

func parseInlineStyles(text string, baseStyle cell.Style, links bool) []rawSegment {
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
		} else if runes[i] == '[' {
			label, url, next, isLink := parseMarkdownLink(runes, i)
			if !isLink {
				curr = append(curr, runes[i])
				i++
				continue
			}
			if len(curr) > 0 {
				segments = append(segments, rawSegment{Text: string(curr), Style: style})
				curr = nil
			}
			linkStyle := markdownLinkStyle(style)
			if links {
				linkStyle = linkStyle.WithLink(url)
			}
			segments = append(segments, rawSegment{Text: label, Style: linkStyle})
			if !links {
				segments = append(segments, rawSegment{
					Text:  " (" + url + ")",
					Style: style.Merge(cell.Style{Modifier: cell.ModifierDim}),
				})
			}
			i = next
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
