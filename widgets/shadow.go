package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DrawShadow draws a clean, dark drop shadow relative to a given bounding area.
// It alpha-darkens existing background and foreground colors in the shadow area
// so the underlying content is preserved in a smooth, dimmed shadow tone.
// offsetX (typically 2) determines the right shadow width.
// offsetY (typically 1) determines the bottom shadow height.
func DrawShadow(buf *buffer.Buffer, area cell.Rect, offsetX, offsetY uint16) {
	if area.Width == 0 || area.Height == 0 || (offsetX == 0 && offsetY == 0) {
		return
	}

	dimCell := func(c *cell.Cell) {
		if c == nil {
			return
		}
		// Darken background color (35% brightness or deep shadow tone)
		if c.Style.Bg.Type() == cell.ColorRGB {
			r, g, b := c.Style.Bg.RGB()
			c.Style.Bg = cell.NewColorRGB(uint8(float64(r)*0.35), uint8(float64(g)*0.35), uint8(float64(b)*0.35))
		} else {
			c.Style.Bg = cell.NewColorRGB(8, 10, 14)
		}

		// Darken foreground text so background characters stay subtly visible
		if c.Style.Fg.Type() == cell.ColorRGB {
			r, g, b := c.Style.Fg.RGB()
			c.Style.Fg = cell.NewColorRGB(uint8(float64(r)*0.35), uint8(float64(g)*0.35), uint8(float64(b)*0.35))
		} else {
			c.Style.Fg = cell.NewColorRGB(45, 50, 60)
		}
	}

	// 1. Right Shadow Column
	if offsetX > 0 {
		for dy := offsetY; dy < area.Height; dy++ {
			sy := area.Y + dy
			for dx := uint16(0); dx < offsetX; dx++ {
				sx := area.X + area.Width + dx
				if c := buf.Get(sx, sy); c != nil {
					dimCell(c)
				}
			}
		}
	}

	// 2. Bottom Shadow Row (including bottom-right corner overlap)
	if offsetY > 0 {
		for dx := offsetX; dx < area.Width+offsetX; dx++ {
			sx := area.X + dx
			for dy := uint16(0); dy < offsetY; dy++ {
				sy := area.Y + area.Height + dy
				if c := buf.Get(sx, sy); c != nil {
					dimCell(c)
				}
			}
		}
	}
}
