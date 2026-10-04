package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type Sparkline struct {
	// ID is the widget's focus ID.
	ID string
	// Data is the series of numbers to draw.
	Data []float64
	// Style is the default cell style.
	Style cell.Style
	// FocusedStyle is the style applied when focused.
	FocusedStyle cell.Style
	// Color sets the colour of the bars. Default uses the style's foreground colour.
	Color cell.Color
}

// sparklineBlocks are the eight partial heights, one eighth apart. The table
// once began with a space instead of ▁, so a value an eighth of a cell high
// drew nothing and the line had seven steps.
var sparklineBlocks = [...]rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func (s Sparkline) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if len(s.Data) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	if s.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(s.ID)
	}
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && s.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: s.ID})
	} else if s.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(s.ID)
			}
		})
	}

	// Fit the latest N values to the column width
	limit := int(ctx.Area.Width)
	data := s.Data
	if len(data) > limit {
		data = data[len(data)-limit:]
	}

	// Find the maximum (guarding against division by zero)
	maxVal := 0.001
	for _, val := range data {
		if val > maxVal {
			maxVal = val
		}
	}

	barColor := s.Color
	if ctx.IsFocused(s.ID) && s.FocusedStyle.Fg.Type() != cell.ColorDefault {
		barColor = s.FocusedStyle.Fg
	} else if barColor.Type() == cell.ColorDefault {
		barColor = ctx.Style.Fg
	}

	for i, val := range data {
		col := ctx.Area.X + uint16(i)
		ratio := val / maxVal
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}

		// Full bar height per cell and the fractional remainder
		totalHeight := ratio * float64(ctx.Area.Height)
		fullCells := int(totalHeight)
		remainder := totalHeight - float64(fullCells)

		for dy := 0; dy < int(ctx.Area.Height); dy++ {
			y := ctx.Area.Y + ctx.Area.Height - 1 - uint16(dy)
			c := buf.Get(col, y)
			if c == nil {
				continue
			}

			c.Style = c.Style.Merge(s.Style)
			c.Style.Fg = barColor

			if dy < fullCells {
				c.Content = '█'
			} else if dy == fullCells && remainder > 0.05 {
				// The nearest eighth: 0.125 is ▁, 0.5 is ▄.
				blockIdx := int(remainder*8+0.5) - 1
				if blockIdx < 0 {
					blockIdx = 0
				}
				if blockIdx > 7 {
					blockIdx = 7
				}
				c.Content = sparklineBlocks[blockIdx]
			} else {
				// Clear the empty cells (so nothing is left over from the previous render)
				c.Content = ' '
			}
		}
	}
}

func (s Sparkline) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}
