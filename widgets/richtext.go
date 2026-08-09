package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
	Lines     []Line
	Style     cell.Style
	Wrap      bool
	Alignment TextAlignment
}

func (t Text) Draw(ctx cell.Context, buf *buffer.Buffer) {
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
