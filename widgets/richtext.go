package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Span is a styled fragment of a line.
type Span struct {
	Text  string
	Style cell.Style
}

// Line is an ordered collection of styled spans.
type Line struct {
	Spans []Span
}

func NewLine(spans ...Span) Line { return Line{Spans: spans} }

// Text renders multiple rich-text lines without allocating during Draw.
type Text struct {
	Lines []Line
	Style cell.Style
}

func (t Text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	for lineIndex, line := range t.Lines {
		y := ctx.Area.Y + uint16(lineIndex)
		if y >= ctx.Area.Y+ctx.Area.Height {
			break
		}
		x := ctx.Area.X
		for _, span := range line.Spans {
			if x >= ctx.Area.X+ctx.Area.Width {
				break
			}
			style := ctx.Style.Merge(t.Style).Merge(span.Style)
			buf.SetString(x, y, span.Text, style)
			x += uint16(visualWidth(span.Text))
		}
	}
}

func (t Text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	var width uint16
	for _, line := range t.Lines {
		var lineWidth uint16
		for _, span := range line.Spans {
			lineWidth += uint16(visualWidth(span.Text))
		}
		if lineWidth > width {
			width = lineWidth
		}
	}
	if width > maxArea.Width {
		width = maxArea.Width
	}
	height := uint16(len(t.Lines))
	if height > maxArea.Height {
		height = maxArea.Height
	}
	return width, height
}
