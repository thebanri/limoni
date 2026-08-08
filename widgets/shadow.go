package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DrawShadow draws a clean, dark drop shadow relative to a given bounding area.
// It clears any background characters in the shadow area and blends it to a very dark shade.
// offsetX (typically 2) determines the right shadow width.
// offsetY (typically 1) determines the bottom shadow height.
func DrawShadow(buf *buffer.Buffer, area cell.Rect, offsetX, offsetY uint16) {
	if area.Width == 0 || area.Height == 0 || (offsetX == 0 && offsetY == 0) {
		return
	}

	shadowBg := cell.NewColorRGB(6, 7, 9)

	// 1. Right Shadow Column
	if offsetX > 0 {
		for dy := offsetY; dy < area.Height; dy++ {
			sy := area.Y + dy
			for dx := uint16(0); dx < offsetX; dx++ {
				sx := area.X + area.Width + dx
				if c := buf.Get(sx, sy); c != nil {
					c.Content = ' '
					c.Style.Bg = shadowBg
					c.Style.Fg = shadowBg
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
					c.Content = ' '
					c.Style.Bg = shadowBg
					c.Style.Fg = shadowBg
				}
			}
		}
	}
}
