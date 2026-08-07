package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// circleMask, dairesel maskeyi temsil eden özel bir image.Image yapısıdır.
type circleMask struct {
	cx, cy int
	r      int
}

func (c *circleMask) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circleMask) Bounds() image.Rectangle {
	d := c.r * 2
	return image.Rect(0, 0, d, d)
}

func (c *circleMask) At(x, y int) color.Color {
	dx := x - c.cx
	dy := y - c.cy
	dist := math.Sqrt(float64(dx*dx + dy*dy))
	if dist <= float64(c.r) {
		return color.Alpha{A: 255}
	}
	// Kenarları yumuşatmak (anti-aliasing) için piksel sınırını yumuşat
	if dist <= float64(c.r)+1.0 {
		delta := float64(c.r) + 1.0 - dist
		return color.Alpha{A: uint8(delta * 255.0)}
	}
	return color.Alpha{A: 0}
}

// ApplyCircleMask, verilen resmi daire şeklinde kırparak (avatar formatında) transparan bir RGBA resim döndürür.
func ApplyCircleMask(src image.Image) image.Image {
	if src == nil {
		return nil
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// En küçük kenara göre kare alanı belirle
	size := w
	if h < size {
		size = h
	}
	if size <= 0 {
		return src
	}

	// Yeni boş bir RGBA resmi oluştur
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	// Ortalamak için başlangıç offsetlerini hesapla
	offsetX := bounds.Min.X + (w-size)/2
	offsetY := bounds.Min.Y + (h-size)/2

	// Maske nesnesini tanımla
	r := size / 2
	mask := &circleMask{cx: r, cy: r, r: r}

	// draw.DrawMask ile maskelenmiş resmi çiz
	draw.DrawMask(
		dst,
		dst.Bounds(),
		src,
		image.Pt(offsetX, offsetY),
		mask,
		image.Point{},
		draw.Over,
	)

	return dst
}
