package widgets

import (
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// TokenKind classifies a run of source text for highlighting.
type TokenKind uint8

const (
	TokenPlain TokenKind = iota
	TokenKeyword
	TokenType
	TokenString
	TokenNumber
	TokenComment
	TokenFunction
	TokenPunctuation
	tokenKinds
)

// Language is what CodeView needs to know to highlight a language. It is a
// lexer, not a parser: good enough to colour code, not to understand it.
type Language struct {
	Name     string
	Keywords []string
	Types    []string // built-in types and constants: int, bool, nil, true
	// LineComments start a comment that runs to the end of the line.
	LineComments []string
	// BlockComment opens and closes a comment that may span lines.
	BlockComment [2]string
	// Quotes are the single-line string delimiters, such as `"` and `'`.
	Quotes string
	// RawStrings open and close strings that may span lines: Go's
	// backquotes, Python's triple quotes.
	RawStrings [][2]string
	// Backslash marks whether \ escapes the next character inside Quotes.
	Backslash bool
}

var (
	LanguageGo = &Language{
		Name: "go",
		Keywords: []string{"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough",
			"for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select",
			"struct", "switch", "type", "var"},
		Types: []string{"any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64",
			"int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64",
			"uintptr", "true", "false", "nil", "iota", "append", "cap", "clear", "close", "copy", "delete", "len",
			"make", "max", "min", "new", "panic", "print", "println", "recover"},
		LineComments: []string{"//"},
		BlockComment: [2]string{"/*", "*/"},
		Quotes:       `"'`,
		RawStrings:   [][2]string{{"`", "`"}},
		Backslash:    true,
	}
	LanguagePython = &Language{
		Name: "python",
		Keywords: []string{"and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif",
			"else", "except", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "match",
			"case", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield"},
		Types: []string{"True", "False", "None", "self", "int", "float", "str", "bytes", "bool", "list", "dict",
			"set", "tuple", "object", "print", "len", "range", "super", "isinstance"},
		LineComments: []string{"#"},
		Quotes:       `"'`,
		RawStrings:   [][2]string{{`"""`, `"""`}, {`'''`, `'''`}},
		Backslash:    true,
	}
	LanguageJavaScript = &Language{
		Name: "javascript",
		Keywords: []string{"async", "await", "break", "case", "catch", "class", "const", "continue", "debugger",
			"default", "delete", "do", "else", "export", "extends", "finally", "for", "from", "function", "if",
			"import", "in", "instanceof", "interface", "let", "new", "of", "return", "static", "super", "switch",
			"this", "throw", "try", "type", "typeof", "var", "void", "while", "with", "yield", "enum", "implements"},
		Types: []string{"true", "false", "null", "undefined", "NaN", "Infinity", "number", "string", "boolean",
			"any", "unknown", "never", "object", "Array", "Object", "Promise", "Map", "Set", "console"},
		LineComments: []string{"//"},
		BlockComment: [2]string{"/*", "*/"},
		Quotes:       `"'`,
		RawStrings:   [][2]string{{"`", "`"}},
		Backslash:    true,
	}
	LanguageRust = &Language{
		Name: "rust",
		Keywords: []string{"as", "async", "await", "break", "const", "continue", "crate", "dyn", "else", "enum",
			"extern", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub", "ref",
			"return", "static", "struct", "trait", "type", "unsafe", "use", "where", "while"},
		Types: []string{"bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16",
			"u32", "u64", "u128", "usize", "str", "String", "Vec", "Option", "Result", "Some", "None", "Ok", "Err",
			"Self", "self", "true", "false", "Box"},
		LineComments: []string{"//"},
		BlockComment: [2]string{"/*", "*/"},
		Quotes:       `"`,
		Backslash:    true,
	}
	LanguageC = &Language{
		Name: "c",
		Keywords: []string{"auto", "break", "case", "class", "const", "constexpr", "continue", "default", "delete",
			"do", "else", "enum", "extern", "for", "goto", "if", "inline", "namespace", "new", "private", "protected",
			"public", "register", "return", "sizeof", "static", "struct", "switch", "template", "this", "typedef",
			"typename", "union", "using", "virtual", "volatile", "while", "#include", "#define", "#if", "#ifdef",
			"#ifndef", "#endif", "#else", "#pragma"},
		Types: []string{"bool", "char", "double", "float", "int", "long", "short", "signed", "unsigned", "void",
			"size_t", "int8_t", "int16_t", "int32_t", "int64_t", "uint8_t", "uint16_t", "uint32_t", "uint64_t",
			"true", "false", "NULL", "nullptr", "std"},
		LineComments: []string{"//"},
		BlockComment: [2]string{"/*", "*/"},
		Quotes:       `"'`,
		Backslash:    true,
	}
	LanguageShell = &Language{
		Name: "shell",
		Keywords: []string{"if", "then", "else", "elif", "fi", "for", "while", "until", "do", "done", "case",
			"esac", "in", "function", "return", "local", "export", "set", "unset", "source", "exit"},
		Types:        []string{"echo", "cd", "printf", "read", "test", "true", "false"},
		LineComments: []string{"#"},
		Quotes:       `"'`,
		Backslash:    true,
	}
	LanguageJSON = &Language{
		Name:      "json",
		Types:     []string{"true", "false", "null"},
		Quotes:    `"`,
		Backslash: true,
	}
	LanguageYAML = &Language{
		Name:         "yaml",
		Types:        []string{"true", "false", "null", "yes", "no", "on", "off"},
		LineComments: []string{"#"},
		Quotes:       `"'`,
		Backslash:    true,
	}
)

// LanguageForFile picks a Language from a file name's extension, or nil.
func LanguageForFile(name string) *Language {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go":
		return LanguageGo
	case ".py", ".pyi":
		return LanguagePython
	case ".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx":
		return LanguageJavaScript
	case ".rs":
		return LanguageRust
	case ".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".java", ".cs":
		return LanguageC
	case ".sh", ".bash", ".zsh":
		return LanguageShell
	case ".json":
		return LanguageJSON
	case ".yaml", ".yml":
		return LanguageYAML
	}
	return nil
}

// codeSpan is a run of one line, from byte start to end, of one kind.
type codeSpan struct {
	start, end int
	kind       TokenKind
}

type codeLine struct {
	text  string
	spans []codeSpan
}

// CodeViewState holds the source, split into highlighted lines, and the
// scroll position. SetSource does the work; drawing only reads it.
type CodeViewState struct {
	// Offset is the first line shown; HOffset the first column.
	Offset, HOffset int
	// Cursor is the highlighted line, or -1 for none.
	Cursor int

	lines    []codeLine
	language *Language
	rows     int
}

// NewCodeViewState highlights src as lang (nil for plain text).
func NewCodeViewState(src string, lang *Language) *CodeViewState {
	s := &CodeViewState{Cursor: -1}
	s.SetSource(src, lang)
	return s
}

// Lines is the number of lines of source.
func (s *CodeViewState) Lines() int { return len(s.lines) }

// SetSource replaces the source and highlights it. Tabs are expanded to the
// next multiple of four columns.
func (s *CodeViewState) SetSource(src string, lang *Language) {
	s.language = lang
	s.lines = s.lines[:0]
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var open string // closer of the block comment or raw string still open
	openKind := TokenPlain
	for _, raw := range strings.Split(src, "\n") {
		text := expandTabs(raw, 4)
		var spans []codeSpan
		if lang != nil {
			spans, open, openKind = lexLine(text, lang, open, openKind)
		}
		s.lines = append(s.lines, codeLine{text: text, spans: spans})
	}
	if n := len(s.lines); n > 1 && s.lines[n-1].text == "" {
		s.lines = s.lines[:n-1] // a trailing newline is not a line
	}
}

func expandTabs(s string, width int) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := width - col%width
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func isIdentRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// lexLine splits one line into spans. open is the closer of a block comment
// or raw string carried in from the previous line; the returned open is the
// one carried out of this line.
func lexLine(text string, lang *Language, open string, openKind TokenKind) ([]codeSpan, string, TokenKind) {
	var spans []codeSpan
	add := func(start, end int, kind TokenKind) {
		if start >= end {
			return
		}
		if n := len(spans); n > 0 && spans[n-1].kind == kind && spans[n-1].end == start {
			spans[n-1].end = end
			return
		}
		spans = append(spans, codeSpan{start, end, kind})
	}
	i := 0
	if open != "" {
		end := strings.Index(text, open)
		if end < 0 {
			add(0, len(text), openKind)
			return spans, open, openKind
		}
		i = end + len(open)
		add(0, i, openKind)
		open = ""
	}
	for i < len(text) {
		rest := text[i:]
		if lc := hasAnyPrefix(rest, lang.LineComments); lc {
			add(i, len(text), TokenComment)
			return spans, "", TokenPlain
		}
		if bc := lang.BlockComment; bc[0] != "" && strings.HasPrefix(rest, bc[0]) {
			end := strings.Index(rest[len(bc[0]):], bc[1])
			if end < 0 {
				add(i, len(text), TokenComment)
				return spans, bc[1], TokenComment
			}
			stop := i + len(bc[0]) + end + len(bc[1])
			add(i, stop, TokenComment)
			i = stop
			continue
		}
		if rs, ok := rawStringAt(rest, lang.RawStrings); ok {
			end := strings.Index(rest[len(rs[0]):], rs[1])
			if end < 0 {
				add(i, len(text), TokenString)
				return spans, rs[1], TokenString
			}
			stop := i + len(rs[0]) + end + len(rs[1])
			add(i, stop, TokenString)
			i = stop
			continue
		}
		r, size := utf8.DecodeRuneInString(rest)
		switch {
		case strings.ContainsRune(lang.Quotes, r):
			j := i + size
			for j < len(text) {
				c := text[j]
				if c == '\\' && lang.Backslash {
					j += 2
					continue
				}
				j++
				if rune(c) == r {
					break
				}
			}
			j = min(j, len(text))
			add(i, j, TokenString)
			i = j
		case unicode.IsDigit(r) || (r == '.' && len(rest) > 1 && rest[1] >= '0' && rest[1] <= '9'):
			j := i + size
			for j < len(text) {
				c, n := utf8.DecodeRuneInString(text[j:])
				if !isIdentRune(c) && c != '.' {
					break
				}
				j += n
			}
			add(i, j, TokenNumber)
			i = j
		case isIdentRune(r) || (r == '#' && lang.Name == "c") || (r == '$' && lang.Name == "shell"):
			j := i + size
			for j < len(text) {
				c, n := utf8.DecodeRuneInString(text[j:])
				if !isIdentRune(c) {
					break
				}
				j += n
			}
			word := text[i:j]
			kind := TokenPlain
			switch {
			case contains(lang.Keywords, word):
				kind = TokenKeyword
			case contains(lang.Types, word):
				kind = TokenType
			case strings.HasPrefix(strings.TrimLeft(text[j:], " "), "("):
				kind = TokenFunction
			case lang.Name == "yaml" && strings.HasPrefix(text[j:], ":"):
				kind = TokenKeyword
			}
			add(i, j, kind)
			i = j
		case unicode.IsSpace(r):
			add(i, i+size, TokenPlain)
			i += size
		default:
			add(i, i+size, TokenPunctuation)
			i += size
		}
	}
	return spans, "", TokenPlain
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func rawStringAt(s string, pairs [][2]string) ([2]string, bool) {
	for _, p := range pairs {
		if p[0] != "" && strings.HasPrefix(s, p[0]) {
			return p, true
		}
	}
	return [2]string{}, false
}

func contains(list []string, word string) bool {
	for _, w := range list {
		if w == word {
			return true
		}
	}
	return false
}

// HandleKey scrolls: ↑ ↓ ← → by one, Page Up/Down by a screen, Home and
// End to the ends. With a Cursor it moves the cursor and keeps it in view.
func (s *CodeViewState) HandleKey(ev driver.KeyEvent) bool {
	if s == nil {
		return false
	}
	page := max(s.rows-1, 1)
	last := max(len(s.lines)-1, 0)
	move := func(d int) {
		if s.Cursor >= 0 {
			s.Cursor = max(0, min(s.Cursor+d, last))
			return
		}
		s.Offset = max(0, min(s.Offset+d, max(len(s.lines)-s.rows, 0)))
	}
	switch ev.Type {
	case driver.KeyArrowUp:
		move(-1)
	case driver.KeyArrowDown:
		move(1)
	case driver.KeyPageUp:
		move(-page)
	case driver.KeyPageDown:
		move(page)
	case driver.KeyHome:
		move(-len(s.lines))
	case driver.KeyEnd:
		move(len(s.lines))
	case driver.KeyArrowLeft:
		if s.HOffset == 0 {
			return false
		}
		s.HOffset--
	case driver.KeyArrowRight:
		s.HOffset++
	default:
		return false
	}
	return true
}

// CodeTheme is the style of each kind of token.
type CodeTheme [tokenKinds]cell.Style

// DefaultCodeTheme is a muted dark theme.
var DefaultCodeTheme = CodeTheme{
	TokenPlain:       {Fg: cell.NewColorRGB(212, 212, 212)},
	TokenKeyword:     {Fg: cell.NewColorRGB(197, 134, 192)},
	TokenType:        {Fg: cell.NewColorRGB(78, 201, 176)},
	TokenString:      {Fg: cell.NewColorRGB(206, 145, 120)},
	TokenNumber:      {Fg: cell.NewColorRGB(181, 206, 168)},
	TokenComment:     {Fg: cell.NewColorRGB(106, 153, 85), Modifier: cell.ModifierItalic},
	TokenFunction:    {Fg: cell.NewColorRGB(220, 220, 170)},
	TokenPunctuation: {Fg: cell.NewColorRGB(160, 160, 160)},
}

// CodeView shows source code with line numbers and syntax highlighting.
// Highlighting is done once, in CodeViewState.SetSource; Draw only paints.
type CodeView struct {
	ID    string
	State *CodeViewState
	// Theme styles each token kind; nil is DefaultCodeTheme.
	Theme *CodeTheme
	// HideLineNumbers drops the gutter.
	HideLineNumbers bool

	Style       cell.Style
	GutterStyle cell.Style
	CursorStyle cell.Style
}

// Draw renders the visible lines. It does not allocate.
func (cv CodeView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	s := cv.State
	if area.Width == 0 || area.Height == 0 || s == nil {
		return
	}
	if cv.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cv.ID)
	}
	theme := cv.Theme
	if theme == nil {
		theme = &DefaultCodeTheme
	}
	base := ctx.Style.Merge(cv.Style)
	rows := int(area.Height)
	s.rows = rows
	if s.Cursor >= 0 {
		if s.Cursor < s.Offset {
			s.Offset = s.Cursor
		}
		if s.Cursor >= s.Offset+rows {
			s.Offset = s.Cursor - rows + 1
		}
	}
	s.Offset = max(0, min(s.Offset, len(s.lines)-1))

	gutter := 0
	if !cv.HideLineNumbers {
		gutter = digits(len(s.lines)) + 1
		if gutter >= int(area.Width) {
			gutter = 0
		}
	}
	gutterStyle := base.Merge(cv.GutterStyle)
	if cv.GutterStyle == (cell.Style{}) {
		gutterStyle = base.Merge(cell.Style{Fg: cell.NewColorRGB(110, 110, 110)})
	}
	textX := area.X + uint16(gutter)
	textW := int(area.Width) - gutter

	for row := 0; row < rows; row++ {
		y := area.Y + uint16(row)
		i := s.Offset + row
		lineStyle := base
		if i == s.Cursor {
			cs := cv.CursorStyle
			if cs == (cell.Style{}) {
				cs = cell.Style{Bg: cell.NewColorRGB(45, 45, 55)}
			}
			lineStyle = base.Merge(cs)
		}
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: lineStyle})
		}
		if i >= len(s.lines) {
			continue
		}
		if gutter > 0 {
			var num [20]byte
			n := strconv.AppendInt(num[:0], int64(i+1), 10)
			x := area.X + uint16(gutter-1-len(n))
			for _, ch := range n {
				buf.SetCell(x, y, cell.Cell{Content: rune(ch), Style: gutterStyle})
				x++
			}
		}
		cv.drawLine(buf, textX, y, textW, s.lines[i], s.HOffset, theme, lineStyle)
	}
}

// drawLine paints one line from column hoff, cluster by cluster, each in
// the style of the span it falls in.
func (cv CodeView) drawLine(buf *buffer.Buffer, x, y uint16, width int, line codeLine, hoff int, theme *CodeTheme, lineStyle cell.Style) {
	col, end := 0, x+uint16(width)
	span := 0
	for pos := 0; pos < len(line.text); {
		cluster, w, _ := cell.NextCluster(line.text[pos:])
		kind := TokenPlain
		for span < len(line.spans) && line.spans[span].end <= pos {
			span++
		}
		if span < len(line.spans) && line.spans[span].start <= pos {
			kind = line.spans[span].kind
		}
		if col >= hoff {
			if x+uint16(w) > end {
				return
			}
			x += buf.SetStringWithin(x, y, cluster, lineStyle.Merge(theme[kind]), end-x)
		} else if col+w > hoff {
			x += uint16(col + w - hoff) // a wide cluster cut by the scroll: leave its half blank
		}
		col += w
		pos += len(cluster)
	}
}

// SizeHint takes all the space offered.
func (cv CodeView) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// AccessibilityNode is a read-only text area; its value is the cursor line
// or, without a cursor, the first line shown.
func (cv CodeView) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	node := accessibility.AccessibilityNode{ID: cv.ID, Role: accessibility.RoleGeneric, Label: "Code", State: state, Bounds: bounds}
	if s := cv.State; s != nil && len(s.lines) > 0 {
		if s.language != nil {
			node.Label = s.language.Name + " code"
		}
		i := s.Offset
		if s.Cursor >= 0 {
			i = s.Cursor
		}
		i = max(0, min(i, len(s.lines)-1))
		node.Value = s.lines[i].text
		node.Position, node.SetSize = i+1, len(s.lines)
	}
	return node
}
