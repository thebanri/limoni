package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// Span is a styled fragment of a line.
type Span struct {
	Text    string
	Style   cell.Style
	Role    string
	OnClick func()
}

// Line is an ordered collection of styled spans.
type Line struct{ Spans []Span }

func NewLine(spans ...Span) Line { return Line{Spans: spans} }

type TextAlignment uint8

const (
	AlignTextLeft TextAlignment = iota
	AlignTextCenter
	AlignTextRight
)

// Text renders multiple rich-text lines with optional cell-aware wrapping.
type Text struct {
	ID           string
	Lines        []Line
	Style        cell.Style
	FocusedStyle cell.Style
	Wrap         bool
	Alignment    TextAlignment
}

func (t Text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}
	if t.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(t.ID)
			}
		})
	}

	lines := t.Lines
	if t.Wrap {
		lines = wrapTextLines(lines, int(ctx.Area.Width))
	}
	for lineIndex, line := range lines {
		y := ctx.Area.Y + uint16(lineIndex)
		if y >= ctx.Area.Y+ctx.Area.Height {
			break
		}
		lineWidth := richLineWidth(line)
		x := int(ctx.Area.X)
		if lineWidth < int(ctx.Area.Width) {
			switch t.Alignment {
			case AlignTextCenter:
				x += (int(ctx.Area.Width) - lineWidth) / 2
			case AlignTextRight:
				x += int(ctx.Area.Width) - lineWidth
			}
		}
		for _, span := range line.Spans {
			if x >= int(ctx.Area.X+ctx.Area.Width) {
				break
			}
			style := ctx.Style.Merge(t.Style)
			if ctx.IsFocused(t.ID) {
				style = style.Merge(t.FocusedStyle)
			}
			if span.Role != "" && ctx.ThemeStyle != nil {
				style = style.Merge(ctx.ThemeStyle(span.Role))
			}
			style = style.Merge(span.Style)
			spanWidth := visualWidth(span.Text)
			if span.OnClick != nil && ctx.RegisterClick != nil && spanWidth > 0 {
				clickX := uint16(x)
				clickWidth := uint16(spanWidth)
				handler := span.OnClick
				ctx.RegisterClick(cell.NewRect(clickX, y, clickWidth, 1), handler)
			}
			buf.SetString(uint16(x), y, span.Text, style)
			x += spanWidth
		}
	}
}

func (t Text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	lines := t.Lines
	if t.Wrap {
		lines = wrapTextLines(lines, int(maxArea.Width))
	}
	var width uint16
	for _, line := range lines {
		if w := uint16(richLineWidth(line)); w > width {
			width = w
		}
	}
	if width > maxArea.Width {
		width = maxArea.Width
	}
	height := uint16(len(lines))
	if height > maxArea.Height {
		height = maxArea.Height
	}
	return width, height
}

// Measure provides explicit size negotiation for Text (RichText).
func (t Text) Measure(maxArea cell.Rect) layout.Measure {
	w, h := t.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

func richLineWidth(line Line) int {
	width := 0
	for _, span := range line.Spans {
		width += visualWidth(span.Text)
	}
	return width
}

func wrapTextLines(lines []Line, maxWidth int) []Line {
	if maxWidth <= 0 {
		return nil
	}
	result := make([]Line, 0, len(lines))
	current := Line{}
	currentWidth := 0
	flush := func() { result = append(result, current); current = Line{}; currentWidth = 0 }
	for _, line := range lines {
		for _, span := range line.Spans {
			text := span.Text
			for len(text) > 0 {
				r, size := utf8.DecodeRuneInString(text)
				if r == '\n' {
					flush()
					text = text[size:]
					continue
				}
				w := cell.RuneWidth(r)
				if w == 0 {
					text = text[size:]
					continue
				}
				if currentWidth > 0 && currentWidth+w > maxWidth {
					flush()
				}
				current.Spans = append(current.Spans, Span{Text: string(r), Style: span.Style, Role: span.Role, OnClick: span.OnClick})
				currentWidth += w
				text = text[size:]
			}
		}
		flush()
	}
	if len(result) == 0 {
		result = append(result, Line{})
	}
	return result
}

// TextFromRichText parses a tag-formatted text string (e.g. "Hello <fg=red>World</>\nNew line!")
// and builds a Text widget ready for rendering.
func TextFromRichText(text string, baseStyle cell.Style, theme Theme) Text {
	resolver := func(tag string) cell.Style {
		return theme.RoleStyle(tag)
	}
	cells := cell.ParseRichText(text, baseStyle, resolver)

	var lines []Line
	var currentLine Line

	for _, c := range cells {
		if c.Content == '\n' {
			lines = append(lines, currentLine)
			currentLine = Line{}
			continue
		}
		// Group consecutive characters with the same style into a single Span to keep rendering fast
		n := len(currentLine.Spans)
		if n > 0 && currentLine.Spans[n-1].Style == c.Style {
			currentLine.Spans[n-1].Text += string(c.Content)
		} else {
			currentLine.Spans = append(currentLine.Spans, Span{
				Text:  string(c.Content),
				Style: c.Style,
			})
		}
	}
	lines = append(lines, currentLine)

	return Text{
		Lines: lines,
		Style: baseStyle,
	}
}

