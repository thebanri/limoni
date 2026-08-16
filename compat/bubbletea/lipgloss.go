package bubbletea

import (
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// Lipgloss-style lightweight styling helper for Limoni

// Style represents a fluent styling builder compatible with Lipgloss patterns.
type Style struct {
	fg        *cell.Color
	bg        *cell.Color
	bold      bool
	italic    bool
	underline bool
	width     int
	height    int
	padding   [4]int // top, right, bottom, left
}

// NewStyle creates a fresh Style.
func NewStyle() Style {
	return Style{}
}

// Foreground sets the text color.
func (s Style) Foreground(c cell.Color) Style {
	s.fg = &c
	return s
}

// Background sets the background color.
func (s Style) Background(c cell.Color) Style {
	s.bg = &c
	return s
}

// Bold enables bold modifier.
func (s Style) Bold(v bool) Style {
	s.bold = v
	return s
}

// Italic enables italic modifier.
func (s Style) Italic(v bool) Style {
	s.italic = v
	return s
}

// Underline enables underline modifier.
func (s Style) Underline(v bool) Style {
	s.underline = v
	return s
}

// Width sets fixed rendering width.
func (s Style) Width(w int) Style {
	s.width = w
	return s
}

// Height sets fixed rendering height.
func (s Style) Height(h int) Style {
	s.height = h
	return s
}

// Padding sets uniform padding.
func (s Style) Padding(p ...int) Style {
	switch len(p) {
	case 1:
		s.padding = [4]int{p[0], p[0], p[0], p[0]}
	case 2:
		s.padding = [4]int{p[0], p[1], p[0], p[1]}
	case 4:
		s.padding = [4]int{p[0], p[1], p[2], p[3]}
	}
	return s
}

// Render formats a string applying the configured styling.
func (s Style) Render(text string) string {
	var sb strings.Builder
	padTop := s.padding[0]
	padRight := s.padding[1]
	padBottom := s.padding[2]
	padLeft := s.padding[3]

	for i := 0; i < padTop; i++ {
		sb.WriteString("\n")
	}

	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		if padLeft > 0 {
			sb.WriteString(strings.Repeat(" ", padLeft))
		}
		sb.WriteString(line)
		if padRight > 0 {
			sb.WriteString(strings.Repeat(" ", padRight))
		}
		if idx < len(lines)-1 {
			sb.WriteString("\n")
		}
	}

	for i := 0; i < padBottom; i++ {
		sb.WriteString("\n")
	}

	return sb.String()
}

// ToCellStyle converts Lipgloss Style to Limoni cell.Style.
func (s Style) ToCellStyle() cell.Style {
	var res cell.Style
	if s.fg != nil {
		res.Fg = *s.fg
	}
	if s.bg != nil {
		res.Bg = *s.bg
	}
	if s.bold {
		res.Modifier |= cell.ModifierBold
	}
	if s.italic {
		res.Modifier |= cell.ModifierItalic
	}
	if s.underline {
		res.Modifier |= cell.ModifierUnderline
	}
	return res
}
