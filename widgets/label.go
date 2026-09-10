package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// Label is a lightweight stateless widget for rendering single- or multi-line text.
type Label struct {
	Text  string
	Style cell.Style
}

// NewLabel creates a new Label widget.
func NewLabel(text string) Label {
	return Label{Text: text}
}

// WithStyle returns a copy of Label with the given style.
func (l Label) WithStyle(style cell.Style) Label {
	l.Style = style
	return l
}

// Draw renders the label text within the context area.
func (l Label) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 || l.Text == "" {
		return
	}

	mergedStyle := ctx.Style.Merge(l.Style)
	bg := mergedStyle.Bg
	if bg.Type() == cell.ColorDefault && ctx.Style.Bg.Type() != cell.ColorDefault {
		bg = ctx.Style.Bg
	}

	lines := strings.Split(l.Text, "\n")
	for i, line := range lines {
		if uint16(i) >= area.Height {
			break
		}
		currY := area.Y + uint16(i)
		written := buf.SetStringWithin(area.X, currY, line, mergedStyle, area.Width)
		if bg.Type() != cell.ColorDefault {
			for x := area.X + written; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
	if bg.Type() != cell.ColorDefault {
		for i := len(lines); uint16(i) < area.Height; i++ {
			currY := area.Y + uint16(i)
			for x := area.X; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
}

// SizeHint calculates the preferred size for the label.
func (l Label) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	lines := strings.Split(l.Text, "\n")
	maxW := 0
	for _, line := range lines {
		w := cell.StringWidth(line)
		if w > maxW {
			maxW = w
		}
	}
	w := uint16(maxW)
	h := uint16(len(lines))
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}
	return w, h
}

// Measure provides layout negotiation support.
func (l Label) Measure(maxArea cell.Rect) layout.Measure {
	w, h := l.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}
