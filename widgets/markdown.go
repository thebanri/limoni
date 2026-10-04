package widgets

import (
	"image"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/driver"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

type Markdown struct {
	ID string
	// Content is the raw markdown text to parse and draw.
	Content string
	// Style is the default text style.
	Style        cell.Style
	FocusedStyle cell.Style
	ScrollOffset *int
	// Theme colours headings, links, code, quotes and the rest; nil is
	// DefaultMarkdownTheme.
	Theme *MarkdownTheme
	// Images returns the picture an image refers to — the src of
	// ![alt](src) — or nil while there is none to show. An image that stands
	// on a line of its own is drawn with the terminal's image protocol, or in
	// half blocks where there is none; until Images returns it, and for an
	// image inside a sentence, its alt text is shown instead. Images is
	// asked on every draw, so it must be cheap and must not allocate: a map
	// lookup. The layout follows when its answer changes. Fetching is the
	// application's: MarkdownImageSources lists what a document refers to.
	Images func(src string) image.Image
	// MaxImageRows caps the height of a picture in rows; zero is 20.
	MaxImageRows int

	// Caching fields to avoid heap allocation on draw loops
	lastContent   string
	lastStyle     cell.Style
	lastWidth     uint16
	lastBaseStyle cell.Style
	lastLinks     bool
	lastTheme     MarkdownTheme
	cachedLines   []markdownLine
	cachedRows    [][]cell.Cell
	plain         string // the text without markup, for the semantic tree

	// Pictures laid out in cachedRows, and what is kept of each source
	// image while the content stays: see markdown_images.go.
	placements []markdownPlacement
	pictures   map[string]*markdownPicture

	// The scrolling handlers, built once, and the last frame they read.
	onMouse, onDrag   func(driver.MouseEvent)
	lastMaxOffset     int
	lastSetFocus      func(string)
	lastCapture       func(func(driver.MouseEvent))
	dragY, dragOffset int
}

// mouseHandler scrolls with the wheel and, on a left press, focuses the
// widget and starts a drag that scrolls the text. Built once, so a scrolling
// Markdown draws without allocating.
func (m *Markdown) mouseHandler() func(driver.MouseEvent) {
	if m.onMouse == nil {
		m.onDrag = func(ev driver.MouseEvent) {
			if ev.Button != driver.MouseRelease && ev.Drag && m.ScrollOffset != nil {
				*m.ScrollOffset = clampMarkdownOffset(m.dragOffset-(int(ev.Y)-m.dragY), m.lastMaxOffset)
			}
		}
		m.onMouse = func(ev driver.MouseEvent) {
			if m.ScrollOffset == nil {
				return
			}
			switch ev.Button {
			case driver.MouseScrollUp:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset-1, m.lastMaxOffset)
			case driver.MouseScrollDown:
				*m.ScrollOffset = clampMarkdownOffset(*m.ScrollOffset+1, m.lastMaxOffset)
			case driver.MouseLeft:
				// Scroll the text by dragging vertically inside the clicked area.
				if m.lastSetFocus != nil {
					m.lastSetFocus(m.ID)
				}
				m.dragY, m.dragOffset = int(ev.Y), *m.ScrollOffset
				if m.lastCapture != nil {
					m.lastCapture(m.onDrag)
				}
			}
		}
	}
	return m.onMouse
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

// WithTheme sets the colours; nil is DefaultMarkdownTheme.
func (m *Markdown) WithTheme(t *MarkdownTheme) *Markdown {
	m.Theme = t
	return m
}

// WithImages sets where pictures come from; see Markdown.Images.
func (m *Markdown) WithImages(images func(src string) image.Image) *Markdown {
	m.Images = images
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

// markdownLine is one parsed line of the source, or one block that is laid
// out as a unit (a table).
type markdownLine struct {
	isDivider bool
	isHeader  bool // a level 1 or 2 heading with text right below it: a blank row follows
	prefix    string
	// prefixStyle draws the prefix; zero is the list marker colour.
	prefixStyle cell.Style
	// indent is how many columns the line starts in: nested list items and
	// quotes.
	indent int
	// cont is drawn at the start of a wrapped continuation row, after the
	// indent; empty means as many spaces as the prefix is wide. A quote
	// repeats its bar.
	cont     string
	segments []StyledSegment

	code  *markdownCode  // a line of a fenced code block
	table *markdownTable // a whole table
	// image is a picture standing on a line of its own. segments hold its
	// alt text, drawn instead while there is no picture to show.
	image *markdownImage
}

// markdownCode is a line inside a fenced code block, highlighted as the
// language named after the opening fence when Limoni knows it.
type markdownCode struct {
	text  string
	spans []codeSpan
}

// markdownTable is a GitHub-style pipe table: a header row and body rows,
// each a list of cells.
type markdownTable struct {
	header []markdownCell
	rows   [][]markdownCell
	align  []byte // per column: 'l', 'c' or 'r'
}

type markdownCell struct {
	segments []rawSegment
	width    int
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

// MarkdownTheme is how a Markdown document is coloured. Each style is merged
// over the text's own, so a zero field leaves that part in the text's style.
type MarkdownTheme struct {
	// Headings are levels 1 to 6. A heading whose style has a background is
	// drawn with a space either side, as a label.
	Headings [6]cell.Style
	Link     cell.Style
	// Code is inline `code`; CodeBlock is the background of a fenced block,
	// whose text is coloured by DefaultCodeTheme.
	Code      cell.Style
	CodeBlock cell.Style
	// Quote is the bar before quoted text, which is also set in italics.
	Quote  cell.Style
	Bullet cell.Style // list markers
	Rule   cell.Style // horizontal rules and table rules
	// Image is the alt text drawn for a picture that is not shown.
	Image cell.Style
}

// DefaultMarkdownTheme is the theme a Markdown without one draws in.
var DefaultMarkdownTheme = MarkdownTheme{
	Headings: [6]cell.Style{
		{Fg: cell.NewColorRGB(0, 255, 255), Modifier: cell.ModifierBold},
		{Fg: cell.NewColorRGB(0, 255, 0), Modifier: cell.ModifierBold},
		{Fg: cell.NewColorRGB(255, 200, 80), Modifier: cell.ModifierBold},
		{Modifier: cell.ModifierBold},
		{Modifier: cell.ModifierBold},
		{Modifier: cell.ModifierBold},
	},
	Link:      cell.Style{Fg: cell.NewColorRGB(100, 160, 255), Modifier: cell.ModifierUnderline},
	Code:      cell.Style{Fg: cell.NewColorRGB(255, 100, 100), Bg: cell.NewColorRGB(45, 45, 45)},
	CodeBlock: cell.Style{Bg: cell.NewColorRGB(30, 32, 38)},
	Quote:     cell.Style{Fg: cell.NewColorRGB(110, 118, 129)},
	Bullet:    cell.Style{Fg: cell.NewColorRGB(0, 255, 0)},
	Rule:      cell.Style{Fg: cell.NewColorRGB(100, 100, 100)},
	Image:     cell.Style{Fg: cell.NewColorRGB(110, 118, 129), Modifier: cell.ModifierItalic},
}

func (m *Markdown) theme() *MarkdownTheme {
	if m.Theme != nil {
		return m.Theme
	}
	return &DefaultMarkdownTheme
}

// parse turns the source into styled lines. links says whether the terminal
// can show OSC 8 hyperlinks, which changes the output: with them, `[text](url)`
// renders as a clickable "text"; without them the address is written out too,
// because a reader who cannot click it still needs to see where it points.
//
// It reads the parts of GitHub-flavoured Markdown a terminal can show:
// headings, emphasis, inline code, links, bullet, numbered and task lists
// (nested by indentation), quotes, rules, fenced code blocks highlighted by
// language, and pipe tables. A code block that has not been closed yet — a
// reply still streaming in — runs to the end of the text.
func (m *Markdown) parse(baseStyle cell.Style, links bool) {
	theme := m.theme()
	if m.Content == m.lastContent && m.Style == m.lastStyle && baseStyle == m.lastBaseStyle &&
		links == m.lastLinks && *theme == m.lastTheme && m.cachedLines != nil {
		return
	}
	if m.Content != m.lastContent {
		m.forgetPictures()
	}

	m.lastContent = m.Content
	m.lastStyle = m.Style
	m.lastBaseStyle = baseStyle
	m.lastLinks = links
	m.lastTheme = *theme
	m.lastWidth = 0
	m.cachedLines = m.cachedLines[:0]
	m.cachedRows = nil

	lines := strings.Split(strings.ReplaceAll(m.Content, "\r\n", "\n"), "\n")
	var plain strings.Builder

	// Fenced code: the fence that opened the block, the language, and the
	// lexer's state carried from line to line.
	fence := ""
	var lang *Language
	open, openKind, openDepth := "", TokenPlain, 0

	for i := 0; i < len(lines); i++ {
		rawLine := lines[i]
		trimmed := strings.TrimSpace(rawLine)

		if fence != "" {
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]) == "" {
				fence = ""
				continue
			}
			text := expandTabs(rawLine, 4)
			var spans []codeSpan
			if lang != nil {
				spans, open, openKind, openDepth = lexLine(text, lang, open, openKind, openDepth)
			}
			m.cachedLines = append(m.cachedLines, markdownLine{code: &markdownCode{text: text, spans: spans}})
			plain.WriteString(text)
			plain.WriteByte('\n')
			continue
		}
		if f := codeFence(trimmed); f != "" {
			fence = f
			lang = LanguageByName(strings.TrimSpace(strings.TrimLeft(trimmed, f[:1])))
			open, openKind, openDepth = "", TokenPlain, 0
			continue
		}

		// A pipe table: a header row, a delimiter row, then body rows.
		if strings.HasPrefix(trimmed, "|") && i+1 < len(lines) && isTableDelimiter(lines[i+1]) {
			table := &markdownTable{}
			table.header = m.tableCells(trimmed, baseStyle.Merge(cell.Style{Modifier: cell.ModifierBold}), links, theme)
			table.align = tableAlignment(lines[i+1], len(table.header))
			i += 2
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				table.rows = append(table.rows, m.tableCells(strings.TrimSpace(lines[i]), baseStyle, links, theme))
			}
			i--
			m.cachedLines = append(m.cachedLines, markdownLine{table: table})
			for _, row := range append([][]markdownCell{table.header}, table.rows...) {
				for c, cl := range row {
					if c > 0 {
						plain.WriteString(" | ")
					}
					for _, seg := range cl.segments {
						plain.WriteString(seg.Text)
					}
				}
				plain.WriteByte('\n')
			}
			continue
		}

		// Blank lines between blocks are one break, however many there are:
		// HTML converted to Markdown often leaves two or three.
		if trimmed == "" && i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			continue
		}

		if isThematicBreak(trimmed) {
			m.cachedLines = append(m.cachedLines, markdownLine{isDivider: true})
			continue
		}

		line := markdownLine{}
		text := trimmed
		lineStyle := baseStyle

		// Quotes: "> text", nested as "> > text".
		for strings.HasPrefix(text, ">") {
			text = strings.TrimSpace(strings.TrimPrefix(text, ">"))
			line.prefix += "│ "
			line.prefixStyle = baseStyle.Merge(theme.Quote)
			lineStyle = lineStyle.Merge(cell.Style{Modifier: cell.ModifierItalic})
		}
		if line.prefix != "" {
			line.cont = line.prefix
		}

		if level, rest := headingLevel(text); level > 0 {
			text = rest
			heading := theme.Headings[level-1]
			lineStyle = lineStyle.Merge(heading)
			// A blank row sets a main heading off from what follows, unless
			// the source already has one there.
			line.isHeader = level <= 2 && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != ""
			if heading.Bg.Type() != cell.ColorDefault {
				text = " " + text + " "
			}
		} else if alt, src, href, ok := blockImage(text); ok && line.prefix == "" {
			line.image = &markdownImage{alt: alt, src: src}
			placeholder := imagePlaceholder(alt)
			style := lineStyle.Merge(theme.Image)
			if href != "" {
				style = linkedStyle(style, theme, href, links)
			}
			line.segments = wordSegments(line.segments, []rawSegment{{Text: placeholder, Style: style}})
			plain.WriteString("[image: " + alt + "]\n")
			m.cachedLines = append(m.cachedLines, line)
			continue
		} else if marker, rest, ok := listMarker(text); ok {
			// Nesting follows the source's indentation, two columns a level.
			line.indent = min(leadingColumns(rawLine)/2*2, 8)
			text = rest
			line.prefix += marker
		}

		segments := parseInline(text, lineStyle, links, theme)
		line.segments = wordSegments(line.segments, segments)
		for _, seg := range segments {
			plain.WriteString(seg.Text)
		}
		plain.WriteByte('\n')
		m.cachedLines = append(m.cachedLines, line)
	}
	m.plain = strings.TrimRight(plain.String(), "\n")
}

// codeFence returns the fence a line opens a code block with ("```" or
// "~~~", or longer), or "".
func codeFence(line string) string {
	for _, c := range []string{"`", "~"} {
		if strings.HasPrefix(line, c+c+c) {
			n := len(line) - len(strings.TrimLeft(line, c))
			// An info string may not contain a backquote: ```go`x is text.
			if c == "`" && strings.Contains(line[n:], "`") {
				return ""
			}
			return line[:n]
		}
	}
	return ""
}

// LanguageByName finds the Language a Markdown fence names: "go", "py",
// "typescript", "sh". It returns nil for a language Limoni cannot highlight.
func LanguageByName(name string) *Language {
	switch strings.ToLower(name) {
	case "go", "golang":
		return LanguageGo
	case "py", "python", "python3":
		return LanguagePython
	case "js", "javascript", "jsx", "ts", "typescript", "tsx", "mjs":
		return LanguageJavaScript
	case "rs", "rust":
		return LanguageRust
	case "c", "h", "cpp", "c++", "cc", "java", "cs", "csharp":
		return LanguageC
	case "sh", "bash", "zsh", "shell", "console", "fish":
		return LanguageShell
	case "json", "jsonc":
		return LanguageJSON
	case "yaml", "yml":
		return LanguageYAML
	}
	return nil
}

// headingLevel reads "### text": the level, 1 to 6, and the text.
func headingLevel(line string) (int, string) {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level == len(line) || line[level] != ' ' {
		return 0, line
	}
	return level, strings.TrimSpace(line[level:])
}

// listMarker reads a list item's marker: "- ", "* ", "+ " (a bullet), "1. "
// or "1) " (numbered), each optionally followed by a task box "[ ] " or
// "[x] ". It returns what to draw before the text and the text.
func listMarker(line string) (marker, rest string, ok bool) {
	switch {
	case strings.HasPrefix(line, "- "), strings.HasPrefix(line, "* "), strings.HasPrefix(line, "+ "):
		marker, rest = "• ", line[2:]
	default:
		n := 0
		for n < len(line) && n < 9 && line[n] >= '0' && line[n] <= '9' {
			n++
		}
		if n == 0 || n+1 >= len(line) || (line[n] != '.' && line[n] != ')') || line[n+1] != ' ' {
			return "", line, false
		}
		marker, rest = line[:n]+". ", line[n+2:]
	}
	switch {
	case strings.HasPrefix(rest, "[ ] "):
		marker, rest = marker+"[ ] ", rest[4:]
	case strings.HasPrefix(rest, "[x] "), strings.HasPrefix(rest, "[X] "):
		marker, rest = marker+"[x] ", rest[4:]
	}
	return marker, rest, true
}

// leadingColumns is how far a line is indented, a tab counting as four.
func leadingColumns(line string) int {
	n := 0
	for _, r := range line {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// isTableDelimiter reports whether line is a table's delimiter row:
// "| --- | :-: |".
func isTableDelimiter(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.Contains(line, "-") {
		return false
	}
	for _, r := range line {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "|"), "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func tableAlignment(delimiter string, columns int) []byte {
	align := make([]byte, columns)
	for i, spec := range splitTableRow(delimiter) {
		if i >= columns {
			break
		}
		left, right := strings.HasPrefix(spec, ":"), strings.HasSuffix(spec, ":")
		switch {
		case left && right:
			align[i] = 'c'
		case right:
			align[i] = 'r'
		default:
			align[i] = 'l'
		}
	}
	return align
}

func (m *Markdown) tableCells(line string, style cell.Style, links bool, theme *MarkdownTheme) []markdownCell {
	var cells []markdownCell
	for _, text := range splitTableRow(line) {
		segs := parseInline(text, style, links, theme)
		w := 0
		for _, s := range segs {
			w += cell.StringWidth(s.Text)
		}
		cells = append(cells, markdownCell{segments: segs, width: w})
	}
	return cells
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
		m.lastMaxOffset, m.lastSetFocus, m.lastCapture = maxOffset, ctx.SetFocus, ctx.CaptureMouse
		ctx.RegisterMouse(ctx.Area, m.mouseHandler())
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
	m.drawPictures(ctx, buf, offset, baseStyle)
}

// visualRows expands parsed markdown into the exact cell rows used by Draw.
// Keeping scrolling and rendering on this single representation prevents
// wrapped lines and header spacing from drifting apart.
func (m *Markdown) visualRows(width uint16, baseStyle cell.Style) [][]cell.Cell {
	if width == 0 {
		return nil
	}
	if width == m.lastWidth && baseStyle == m.lastBaseStyle && m.cachedRows != nil && !m.picturesChanged() {
		return m.cachedRows
	}
	m.lastWidth = width
	m.lastBaseStyle = baseStyle
	m.cachedRows, m.placements = m.buildRows(width, baseStyle, m.placements[:0])
	return m.cachedRows
}

// buildRows lays the parsed lines out at width, and appends where each
// picture went to placements.
func (m *Markdown) buildRows(width uint16, baseStyle cell.Style, placements []markdownPlacement) ([][]cell.Cell, []markdownPlacement) {
	w := int(width)
	theme := m.theme()
	rows := make([][]cell.Cell, 0, len(m.cachedLines))
	blank := func() []cell.Cell { return make([]cell.Cell, 0, w) }
	pad := func(row []cell.Cell, n int, style cell.Style) []cell.Cell {
		for len(row) < n && len(row) < w {
			row = append(row, cell.Cell{Content: ' ', Style: style})
		}
		return row
	}
	for _, line := range m.cachedLines {
		switch {
		case line.isDivider:
			row := blank()
			style := baseStyle.Merge(theme.Rule)
			for len(row) < w {
				row = append(row, cell.Cell{Content: '┄', Style: style})
			}
			rows = append(rows, row)
			continue
		case line.code != nil:
			rows = append(rows, codeRow(line.code, w, baseStyle, theme))
			continue
		case line.table != nil:
			rows = append(rows, tableRows(line.table, w, baseStyle, theme)...)
			continue
		case line.image != nil:
			if img := m.picture(line.image.src); img != nil {
				cols, height, show := m.pictureSize(img, w-line.indent)
				if !show {
					continue // a tracking pixel: nothing to see, and no alt text either
				}
				placements = append(placements, markdownPlacement{
					src: line.image.src, img: img,
					row: len(rows), rows: height, col: line.indent, cols: cols,
				})
				for range height {
					rows = append(rows, blank())
				}
				continue
			}
		}

		row := pad(blank(), line.indent, baseStyle)
		if line.prefix != "" {
			prefixStyle := line.prefixStyle
			if prefixStyle == (cell.Style{}) {
				prefixStyle = baseStyle.Merge(theme.Bullet)
			}
			row = appendClusters(row, line.prefix, prefixStyle, w)
		}
		indent := len(row)
		startRow := func() []cell.Cell {
			row := pad(blank(), line.indent, baseStyle)
			if line.cont != "" {
				return appendClusters(row, line.cont, line.prefixStyle, w)
			}
			return pad(row, indent, baseStyle)
		}
		for _, seg := range line.segments {
			for index, word := range seg.Words {
				wordWidth := seg.WordWidths[index]
				space := 0
				if index > 0 {
					space = 1
				}
				if len(row)+space+wordWidth >= w && len(row) > indent {
					rows = append(rows, pad(row, indent, baseStyle))
					row = startRow()
					space = 0
				}
				if space == 1 && len(row) < w {
					row = append(row, cell.Cell{Content: ' ', Style: seg.Style})
				}
				// A word wider than a whole row — a long address, mostly — is
				// broken across rows rather than cut off at the edge.
				for wordWidth > w-len(row) && w-len(row) > 0 {
					head, _ := cell.Truncate(word, w-len(row))
					if head == "" {
						break // a wide character in a one-column gap
					}
					row = appendClusters(row, head, seg.Style, w)
					rows = append(rows, row)
					row = startRow()
					word = word[len(head):]
					wordWidth = cell.StringWidth(word)
				}
				row = appendClusters(row, word, seg.Style, w)
			}
		}
		rows = append(rows, row)
		if line.isHeader {
			rows = append(rows, blank())
		}
	}
	return rows, placements
}

// codeRow draws one line of a code block on its own background, cut at the
// width rather than wrapped, as an editor shows it.
func codeRow(code *markdownCode, width int, baseStyle cell.Style, theme *MarkdownTheme) []cell.Cell {
	bg := baseStyle.Merge(theme.CodeBlock)
	row := make([]cell.Cell, 0, width)
	row = append(row, cell.Cell{Content: ' ', Style: bg})
	pos := 0
	draw := func(text string, kind TokenKind) {
		style := bg.Merge(DefaultCodeTheme[kind])
		style.Bg = bg.Bg
		row = appendClusters(row, text, style, width)
	}
	for _, span := range code.spans {
		if span.start > pos {
			draw(code.text[pos:span.start], TokenPlain)
		}
		draw(code.text[span.start:span.end], span.kind)
		pos = span.end
	}
	if pos < len(code.text) {
		draw(code.text[pos:], TokenPlain)
	}
	for len(row) < width {
		row = append(row, cell.Cell{Content: ' ', Style: bg})
	}
	return row
}

// tableRows draws a table with box-drawing rules between its columns. When
// it is wider than width, the widest columns give up room first and cells
// are cut with an ellipsis.
func tableRows(t *markdownTable, width int, baseStyle cell.Style, theme *MarkdownTheme) [][]cell.Cell {
	cols := len(t.header)
	for _, r := range t.rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return nil
	}
	widths := make([]int, cols)
	measure := func(row []markdownCell) {
		for i, c := range row {
			widths[i] = max(widths[i], c.width)
		}
	}
	measure(t.header)
	for _, r := range t.rows {
		measure(r)
	}
	// Each column has a space either side and a rule after it but the last.
	total := func() int {
		n := cols - 1
		for _, cw := range widths {
			n += cw + 2
		}
		return n
	}
	for total() > width {
		widest := 0
		for i := range widths {
			if widths[i] > widths[widest] {
				widest = i
			}
		}
		if widths[widest] <= 1 {
			break
		}
		widths[widest]--
	}

	ruleStyle := baseStyle.Merge(theme.Rule)
	line := func(row []markdownCell) []cell.Cell {
		out := make([]cell.Cell, 0, width)
		for i := 0; i < cols; i++ {
			if i > 0 {
				out = append(out, cell.Cell{Content: '│', Style: ruleStyle})
			}
			var c markdownCell
			if i < len(row) {
				c = row[i]
			}
			lead := 0
			if room := widths[i] - c.width; room > 0 {
				switch t.align[min(i, len(t.align)-1)] {
				case 'r':
					lead = room
				case 'c':
					lead = room / 2
				}
			}
			start := len(out)
			out = append(out, cell.Cell{Content: ' ', Style: baseStyle})
			for j := 0; j < lead; j++ {
				out = append(out, cell.Cell{Content: ' ', Style: baseStyle})
			}
			limit := start + 1 + widths[i]
			for _, seg := range c.segments {
				out = appendClusters(out, seg.Text, seg.Style, limit)
			}
			if c.width > widths[i] && widths[i] > 0 {
				out[limit-1] = cell.Cell{Content: '…', Style: baseStyle}
			}
			for len(out) < limit+1 {
				out = append(out, cell.Cell{Content: ' ', Style: baseStyle})
			}
		}
		if len(out) > width {
			out = out[:width]
		}
		return out
	}
	rows := [][]cell.Cell{line(t.header)}
	rule := make([]cell.Cell, 0, width)
	for i := 0; i < cols; i++ {
		if i > 0 {
			rule = append(rule, cell.Cell{Content: '┼', Style: ruleStyle})
		}
		for j := 0; j < widths[i]+2; j++ {
			rule = append(rule, cell.Cell{Content: '─', Style: ruleStyle})
		}
	}
	if len(rule) > width {
		rule = rule[:width]
	}
	rows = append(rows, rule)
	for _, r := range t.rows {
		rows = append(rows, line(r))
	}
	return rows
}

// visualLineCount is how many rows the text takes at width, wrapping
// included.
func (m *Markdown) visualLineCount(width uint16) int {
	if width == 0 {
		return 0
	}
	rows, _ := m.buildRows(width, m.lastBaseStyle, nil)
	return len(rows)
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

// SizeHint reports the unwrapped size: as tall as the source's lines and
// blocks, as wide as the widest of them. It does not lay the text out at
// maxArea's width, so a layout that measures at one width and draws at
// another does not rebuild the rows every frame.
func (m *Markdown) SizeHint(maxArea cell.Rect) (width, height uint16) {
	baseStyle := cell.Style{}.Merge(m.Style)
	// SizeHint has no draw context, so it measures with whatever the last
	// draw used. Before the first draw that is the wider form, with the URLs
	// written out, which errs towards asking for too much room rather than
	// too little.
	m.parse(baseStyle, m.lastLinks)

	h, wMax := 0, 0
	for _, line := range m.cachedLines {
		switch {
		case line.table != nil:
			h += 2 + len(line.table.rows)
			tw := len(line.table.header) - 1
			for _, c := range line.table.header {
				tw += c.width + 2
			}
			wMax = max(wMax, tw)
			continue
		case line.code != nil:
			h++
			wMax = max(wMax, cell.StringWidth(line.code.text)+1)
			continue
		case line.isDivider:
			h++
			continue
		case line.isHeader:
			h += 2
		default:
			h++
		}
		lineWidth := line.indent + cell.StringWidth(line.prefix)
		for _, segment := range line.segments {
			for _, ww := range segment.WordWidths {
				lineWidth += ww + 1
			}
		}
		wMax = max(wMax, lineWidth)
	}
	return uint16(min(wMax, int(maxArea.Width))), uint16(min(h, 0xFFFF))
}

// AccessibilityNode describes the text as a screen reader or an agent reads
// it: the words without the markup.
func (m *Markdown) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	value := m.plain
	if m.lastContent != m.Content || m.cachedLines == nil {
		value = m.Content // not drawn yet
	}
	return accessibility.AccessibilityNode{
		ID:     m.ID,
		Role:   accessibility.RoleGeneric,
		Label:  "Markdown",
		Value:  value,
		State:  state,
		Bounds: bounds,
	}
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

// linkedStyle is how a link to url is drawn: in the theme's link style, and
// clickable where the terminal shows OSC 8 hyperlinks. A link to a fragment
// of the page ("#notes") goes nowhere in a terminal, so it is plain text.
func linkedStyle(style cell.Style, theme *MarkdownTheme, url string, links bool) cell.Style {
	if strings.HasPrefix(url, "#") {
		return style
	}
	style = style.Merge(theme.Link)
	if links {
		style = style.WithLink(url)
	}
	return style
}

// parseMarkdownLink reads `[label](url)` starting at the opening bracket. It
// returns ok=false for anything that is not a complete link — a lone bracket,
// a reference-style link, an unclosed URL — which is then drawn as the
// literal text it is. A title, `[label](url "title")`, is read and dropped.
// The label may itself be an image: `[![alt](src)](url)`.
func parseMarkdownLink(runes []rune, start int) (label, url string, next int, ok bool) {
	n := len(runes)
	i := start + 1
	labelStart := i
	if i+1 < n && runes[i] == '!' && runes[i+1] == '[' {
		// An image as the label: skip over it whole.
		if _, _, after, isImage := parseMarkdownLink(runes, i+1); isImage {
			i = after
		}
	}
	for i < n && runes[i] != ']' {
		if runes[i] == '[' || runes[i] == '\n' {
			return "", "", 0, false
		}
		if runes[i] == '\\' && i+1 < n {
			i++ // an escaped bracket belongs to the label
		}
		i++
	}
	if i >= n || i+1 >= n || runes[i+1] != '(' {
		return "", "", 0, false
	}
	label = string(runes[labelStart:i])
	i += 2
	urlStart := i
	urlEnd := -1
	for i < n && runes[i] != ')' {
		if runes[i] == ' ' {
			// A space ends the address: what follows must be a title in
			// quotes, or this is not a link.
			urlEnd = i
			for i < n && runes[i] == ' ' {
				i++
			}
			if i >= n || (runes[i] != '"' && runes[i] != '\'') {
				return "", "", 0, false
			}
			quote := runes[i]
			i++
			for i < n && runes[i] != quote {
				i++
			}
			if i >= n {
				return "", "", 0, false
			}
			i++
			for i < n && runes[i] == ' ' {
				i++
			}
			if i >= n || runes[i] != ')' {
				return "", "", 0, false
			}
			break
		}
		i++
	}
	if urlEnd < 0 {
		urlEnd = i
	}
	if i >= n || urlEnd == urlStart {
		return "", "", 0, false
	}
	return label, string(runes[urlStart:urlEnd]), i + 1, true
}

// blockImage reports whether a line is nothing but an image, `![alt](src)`,
// or a linked one, `[![alt](src)](href)`.
func blockImage(text string) (alt, src, href string, ok bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "![") && !strings.HasPrefix(text, "[![") {
		return "", "", "", false
	}
	runes := []rune(text)
	if runes[0] == '!' {
		alt, src, next, ok := parseMarkdownLink(runes, 1)
		if !ok || next != len(runes) {
			return "", "", "", false
		}
		return unescapeMarkdown(alt), src, "", true
	}
	label, href, next, ok := parseMarkdownLink(runes, 0)
	if !ok || next != len(runes) {
		return "", "", "", false
	}
	inner := []rune(label)
	if len(inner) < 2 || inner[0] != '!' {
		return "", "", "", false
	}
	alt, src, next, ok = parseMarkdownLink(inner, 1)
	if !ok || next != len(inner) {
		return "", "", "", false
	}
	return unescapeMarkdown(alt), src, href, true
}

// imagePlaceholder is the text drawn for a picture that is not shown.
func imagePlaceholder(alt string) string {
	if alt == "" {
		return "▣ image"
	}
	return "▣ " + alt
}

// isMarkdownPunct reports whether a backslash before r escapes it: the
// ASCII punctuation CommonMark lets a backslash escape.
func isMarkdownPunct(r rune) bool {
	return r < 0x80 && strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r)
}

// unescapeMarkdown drops the backslash from escaped punctuation.
func unescapeMarkdown(text string) string {
	if !strings.Contains(text, "\\") {
		return text
	}
	runes := []rune(text)
	out := runes[:0]
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && isMarkdownPunct(runes[i+1]) {
			i++
		}
		out = append(out, runes[i])
	}
	return string(out)
}

// isThematicBreak reports whether a line is a horizontal rule: three or more
// of the same -, * or _, which may be spaced ("* * *").
func isThematicBreak(line string) bool {
	if line == "" {
		return false
	}
	mark, count := line[0], 0
	if mark != '-' && mark != '*' && mark != '_' {
		return false
	}
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case mark:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

// wordSegments splits parsed segments into words and measures them, once,
// for the layout.
func wordSegments(dst []StyledSegment, segments []rawSegment) []StyledSegment {
	for _, seg := range segments {
		words := strings.Split(seg.Text, " ")
		wordRunes := make([][]rune, len(words))
		widths := make([]int, len(words))
		for i, word := range words {
			wordRunes[i] = []rune(word)
			widths[i] = cell.StringWidth(word)
		}
		dst = append(dst, StyledSegment{
			Style:      seg.Style,
			Words:      words,
			WordRunes:  wordRunes,
			WordWidths: widths,
		})
	}
	return dst
}

// emphasisMark reports whether the single star at i opens emphasis (closing
// false: text follows it) or closes it (closing true: text precedes it).
func emphasisMark(runes []rune, i int, closing bool) bool {
	if closing {
		return i > 0 && runes[i-1] != ' '
	}
	return i+1 < len(runes) && runes[i+1] != ' '
}

// parseInlineStyles reads inline markup in the default theme.
func parseInlineStyles(text string, baseStyle cell.Style, links bool) []rawSegment {
	return parseInline(text, baseStyle, links, &DefaultMarkdownTheme)
}

// parseInline reads emphasis, strikethrough, code, links, images and
// backslash escapes into styled segments.
func parseInline(text string, baseStyle cell.Style, links bool, theme *MarkdownTheme) []rawSegment {
	var segments []rawSegment
	runes := []rune(text)
	var curr []rune
	i := 0
	n := len(runes)
	style := baseStyle
	flush := func() {
		if len(curr) > 0 {
			segments = append(segments, rawSegment{Text: string(curr), Style: style})
			curr = nil
		}
	}

	for i < n {
		if runes[i] == '\\' && i+1 < n && isMarkdownPunct(runes[i+1]) {
			curr = append(curr, runes[i+1])
			i += 2
		} else if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			flush()
			if (style.Modifier & cell.ModifierBold) != 0 {
				style.Modifier &= ^cell.ModifierBold
			} else {
				style.Modifier |= cell.ModifierBold
			}
			i += 2
		} else if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
			flush()
			style.Modifier ^= cell.ModifierStrikethrough
			i += 2
		} else if runes[i] == '*' && !emphasisMark(runes, i, style.Modifier&cell.ModifierItalic != 0) {
			// "2 * 3": a star with space on the side that would open (or
			// close) emphasis is the character itself.
			curr = append(curr, runes[i])
			i++
		} else if runes[i] == '*' {
			flush()
			if (style.Modifier & cell.ModifierItalic) != 0 {
				style.Modifier &= ^cell.ModifierItalic
			} else {
				style.Modifier |= cell.ModifierItalic
			}
			i++
		} else if runes[i] == '!' && i+1 < n && runes[i+1] == '[' {
			// An image inside a sentence is not drawn as a picture: its alt
			// text stands in for it.
			alt, _, next, isImage := parseMarkdownLink(runes, i+1)
			if !isImage {
				curr = append(curr, runes[i])
				i++
				continue
			}
			flush()
			segments = append(segments, rawSegment{Text: imagePlaceholder(unescapeMarkdown(alt)), Style: style.Merge(theme.Image)})
			i = next
		} else if runes[i] == '[' {
			label, url, next, isLink := parseMarkdownLink(runes, i)
			if !isLink || label == "" {
				curr = append(curr, runes[i])
				i++
				continue
			}
			flush()
			linkStyle := linkedStyle(style, theme, url, links)
			for _, seg := range parseInline(label, linkStyle, false, theme) {
				if links && linkStyle.Link != 0 {
					seg.Style = seg.Style.WithLink(url)
				}
				segments = append(segments, seg)
			}
			// Where the address cannot be clicked it is written out, unless
			// the text already is the address.
			if !links && !strings.HasPrefix(url, "#") && unescapeMarkdown(label) != url {
				segments = append(segments, rawSegment{
					Text:  " (" + url + ")",
					Style: style.Merge(cell.Style{Modifier: cell.ModifierDim}),
				})
			}
			i = next
		} else if runes[i] == '`' {
			flush()
			codeStyle := baseStyle.Merge(theme.Code)
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
	flush()
	return segments
}
