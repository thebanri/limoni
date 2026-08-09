package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ProgressBar renders a bounded horizontal progress indicator.
type ProgressBar struct {
	Value       float64
	Min         float64
	Max         float64
	Style       cell.Style
	FilledStyle cell.Style
	EmptyStyle  cell.Style
	ShowPercent bool
}

func (p ProgressBar) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.Area.Width == 0 || ctx.Area.Height == 0 || p.Max <= p.Min {
		return
	}
	value := p.Value
	if value < p.Min {
		value = p.Min
	}
	if value > p.Max {
		value = p.Max
	}
	ratio := (value - p.Min) / (p.Max - p.Min)
	filled := int(ratio * float64(ctx.Area.Width))
	for x := uint16(0); x < ctx.Area.Width; x++ {
		style := ctx.Style.Merge(p.EmptyStyle)
		content := '░'
		if int(x) < filled {
			style = ctx.Style.Merge(p.FilledStyle)
			content = '█'
		}
		if style == (cell.Style{}) {
			style = ctx.Style.Merge(p.Style)
		}
		buf.SetCell(ctx.Area.X+x, ctx.Area.Y, cell.Cell{Content: content, Style: style})
	}
	if p.ShowPercent && ctx.Area.Width >= 5 {
		text := itoa(int(ratio*100)) + "%"
		start := int(ctx.Area.X) + (int(ctx.Area.Width)-len([]rune(text)))/2
		buf.SetString(uint16(start), ctx.Area.Y, text, ctx.Style.Merge(p.Style))
	}
}

func (p ProgressBar) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }
