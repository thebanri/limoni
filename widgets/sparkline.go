package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type Sparkline struct {
	// ID, widget odak kimliğidir.
	ID string
	// Data, çizilecek veri geçmişini temsil eden sayılar dizisidir.
	Data []float64
	// Style, varsayılan hücre stilini tanımlar.
	Style cell.Style
	// FocusedStyle, odaklandığında uygulanacak stildir.
	FocusedStyle cell.Style
	// Color, barların rengini belirler. Default ise stilin ön plan rengi kullanılır.
	Color cell.Color
}

var sparklineBlocks = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func (s Sparkline) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if len(s.Data) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	if s.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(s.ID)
	}
	if s.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(s.ID)
			}
		})
	}

	// En son N adet veriyi sütun genişliğine sığdır
	limit := int(ctx.Area.Width)
	data := s.Data
	if len(data) > limit {
		data = data[len(data)-limit:]
	}

	// Max değeri bul (sıfır bölme korumalı)
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

		// Hücre bazında tam bar yüksekliği ve kesir remainder hesabı
		totalHeight := ratio * float64(ctx.Area.Height)
		fullCells := int(totalHeight)
		remainder := totalHeight - float64(fullCells)

		for dy := 0; dy < int(ctx.Area.Height); dy++ {
			y := ctx.Area.Y + ctx.Area.Height - 1 - uint16(dy)
			c := buf.Get(col, y)
			if c == nil {
				continue
			}

			c.Style = ctx.Style.Merge(s.Style)
			c.Style.Fg = barColor

			if dy < fullCells {
				c.Content = '█'
			} else if dy == fullCells && remainder > 0.05 {
				blockIdx := int(remainder * 7.9)
				if blockIdx > 7 {
					blockIdx = 7
				}
				c.Content = sparklineBlocks[blockIdx]
			} else {
				// Boş hücreleri temizle (önceki render kalıntılarını engellemek için)
				c.Content = ' '
			}
		}
	}
}

func (s Sparkline) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}
