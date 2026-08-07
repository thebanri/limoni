package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// brailleOffset, 2x4 piksel alt ızgarasındaki (x, y) koordinatlarını Braille bitmask değerlerine eşler.
// y: 0..3, x: 0..1
var brailleOffset = [4][2]byte{
	{0x01, 0x08}, // y=0: x=0 -> Dot 1 (0x01), x=1 -> Dot 4 (0x08)
	{0x02, 0x10}, // y=1: x=0 -> Dot 2 (0x02), x=1 -> Dot 5 (0x10)
	{0x04, 0x20}, // y=2: x=0 -> Dot 3 (0x04), x=1 -> Dot 6 (0x20)
	{0x40, 0x80}, // y=3: x=0 -> Dot 7 (0x40), x=1 -> Dot 8 (0x80)
}

// Canvas, hücre başına 2x4 sanal piksel çözünürlüğünde (Braille karakterleri kullanarak)
// terminal üzerinde yüksek çözünürlüklü vektör çizimleri yapmayı sağlayan görsel bileşendir.
type Canvas struct {
	width  uint16
	height uint16
	grid   []byte
	styles []cell.Style
}

// NewCanvas, belirtilen hücre genişlik ve yüksekliğinde yeni bir Canvas oluşturur.
// Sanal çizim alanı çözünürlüğü: (width * 2) x (height * 4) piksel olacaktır.
func NewCanvas(width, height uint16) *Canvas {
	return &Canvas{
		width:  width,
		height: height,
		grid:   make([]byte, int(width)*int(height)),
		styles: make([]cell.Style, int(width)*int(height)),
	}
}

// Reset, canvas boyutunu günceller ve iç tamponları bellek tahsisatı yapmadan sıfırlar (kapasite yeterliyse).
func (c *Canvas) Reset(width, height uint16) {
	needed := int(width) * int(height)
	c.width = width
	c.height = height

	if cap(c.grid) >= needed {
		c.grid = c.grid[:needed]
		for i := range c.grid {
			c.grid[i] = 0
		}
	} else {
		c.grid = make([]byte, needed)
	}

	if cap(c.styles) >= needed {
		c.styles = c.styles[:needed]
		for i := range c.styles {
			c.styles[i].Reset()
		}
	} else {
		c.styles = make([]cell.Style, needed)
	}
}


// Set, canvas üzerindeki sanal (px, py) pikselini aktif hale getirir ve rengini/stilini günceller.
// Koordinatlar sınır dışındaysa işlem yok sayılır (clipping).
func (c *Canvas) Set(px, py int, style cell.Style) {
	if px < 0 || py < 0 || px >= int(c.width)*2 || py >= int(c.height)*4 {
		return
	}

	cx := px / 2
	cy := py / 4
	dx := px % 2
	dy := py % 4

	idx := cy*int(c.width) + cx
	hasExisting := c.grid[idx] != 0
	c.grid[idx] |= brailleOffset[dy][dx]

	if hasExisting {
		prevFg := c.styles[idx].Fg
		newFg := style.Fg
		if prevFg.Type() == cell.ColorRGB && newFg.Type() == cell.ColorRGB {
			r1, g1, b1 := prevFg.RGB()
			r2, g2, b2 := newFg.RGB()
			mixedFg := cell.NewColorRGB(
				uint8((int(r1)+int(r2))/2),
				uint8((int(g1)+int(g2))/2),
				uint8((int(b1)+int(b2))/2),
			)
			c.styles[idx].Fg = mixedFg
		} else {
			c.styles[idx] = c.styles[idx].Merge(style)
		}
	} else {
		c.styles[idx] = c.styles[idx].Merge(style)
	}
}

// Unset, canvas üzerindeki sanal (px, py) pikselini pasif hale getirir.
// Koordinatlar sınır dışındaysa işlem yok sayılır.
func (c *Canvas) Unset(px, py int) {
	if px < 0 || py < 0 || px >= int(c.width)*2 || py >= int(c.height)*4 {
		return
	}

	cx := px / 2
	cy := py / 4
	dx := px % 2
	dy := py % 4

	idx := cy*int(c.width) + cx
	c.grid[idx] &= ^brailleOffset[dy][dx]
}

// Clear, tüm canvas'ı temizler; pikselleri sıfırlar ve stilleri varsayılana döndürür.
func (c *Canvas) Clear() {
	for i := range c.grid {
		c.grid[i] = 0
		c.styles[i].Reset()
	}
}

// Draw, canvas içeriğini terminal tamponuna (buffer.Buffer) Braille karakterleri olarak çizer.
func (c *Canvas) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	drawW := c.width
	if drawW > area.Width {
		drawW = area.Width
	}
	drawH := c.height
	if drawH > area.Height {
		drawH = area.Height
	}

	for y := uint16(0); y < drawH; y++ {
		for x := uint16(0); x < drawW; x++ {
			idx := int(y)*int(c.width) + int(x)
			mask := c.grid[idx]

			var r rune
			if mask == 0 {
				r = ' '
			} else {
				r = rune(0x2800 + int(mask))
			}

			style := ctx.Style.Merge(c.styles[idx])
			buf.SetCell(area.X+x, area.Y+y, cell.Cell{
				Content: r,
				Style:   style,
			})
		}
	}
}

// SizeHint, verilen üst sınırlara göre bu canvas'ın tercih ettiği boyutları döner.
func (c *Canvas) SizeHint(maxArea cell.Rect) (width, height uint16) {
	w := c.width
	h := c.height
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}
	return w, h
}
