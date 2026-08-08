package widgets

import (
	"image"
	"math"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// DrawLine, Bresenham Çizgi Algoritmasını kullanarak canvas üzerine iki nokta arasına çizgi çizer.
func (c *Canvas) DrawLine(x1, y1, x2, y2 int, style cell.Style) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		c.Set(x1, y1, style)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// DrawCircle, Midpoint (Bresenham) Daire Algoritmasını kullanarak belirtilen merkez ve yarıçapta bir çember çizer.
func (c *Canvas) DrawCircle(cx, cy, r int, style cell.Style) {
	if r < 0 {
		return
	}
	x := r
	y := 0
	d := 3 - 2*r

	for x >= y {
		c.Set(cx+x, cy+y, style)
		c.Set(cx+y, cy+x, style)
		c.Set(cx-y, cy+x, style)
		c.Set(cx-x, cy+y, style)
		c.Set(cx-x, cy-y, style)
		c.Set(cx-y, cy-x, style)
		c.Set(cx+y, cy-x, style)
		c.Set(cx+x, cy-y, style)

		if d < 0 {
			d = d + 4*y + 6
		} else {
			d = d + 4*(y-x) + 10
			x--
		}
		y++
	}
}

// DrawRect, sol üst köşe koordinatları, genişlik ve yüksekliği belirtilen bir dikdörtgen çizer.
func (c *Canvas) DrawRect(x, y, w, h int, style cell.Style) {
	if w <= 0 || h <= 0 {
		return
	}
	// Yatay çizgiler
	c.DrawLine(x, y, x+w-1, y, style)
	c.DrawLine(x, y+h-1, x+w-1, y+h-1, style)
	// Dikey çizgiler
	c.DrawLine(x, y, x, y+h-1, style)
	c.DrawLine(x+w-1, y, x+w-1, y+h-1, style)
}

// DrawBezierQuadratic, başlangıç (x0, y0), kontrol (x1, y1) ve bitiş (x2, y2) noktalarıyla belirlenen
// ikinci dereceden (quadratic) Bezier eğrisini çizer. 'steps' çizimin kaç adımdan oluşacağını belirler (varsayılan: 50).
func (c *Canvas) DrawBezierQuadratic(x0, y0, x1, y1, x2, y2 int, steps int, style cell.Style) {
	if steps <= 0 {
		steps = 50
	}

	prevX, prevY := x0, y0
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		oneMinusT := 1.0 - t

		a := oneMinusT * oneMinusT
		b := 2.0 * oneMinusT * t
		d := t * t

		currX := int(a*float64(x0) + b*float64(x1) + d*float64(x2) + 0.5)
		currY := int(a*float64(y0) + b*float64(y1) + d*float64(y2) + 0.5)

		c.DrawLine(prevX, prevY, currX, currY, style)
		prevX, prevY = currX, currY
	}
}

// DrawBezierCubic, başlangıç (x0, y0), iki kontrol (x1, y1), (x2, y2) ve bitiş (x3, y3) noktalarıyla
// belirlenen üçüncü dereceden (cubic) Bezier eğrisini çizer. 'steps' çizimin kaç adımdan oluşacağını belirler (varsayılan: 50).
func (c *Canvas) DrawBezierCubic(x0, y0, x1, y1, x2, y2, x3, y3 int, steps int, style cell.Style) {
	if steps <= 0 {
		steps = 50
	}

	prevX, prevY := x0, y0
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		oneMinusT := 1.0 - t

		a := oneMinusT * oneMinusT * oneMinusT
		b := 3.0 * oneMinusT * oneMinusT * t
		d := 3.0 * oneMinusT * t * t
		e := t * t * t

		currX := int(a*float64(x0) + b*float64(x1) + d*float64(x2) + e*float64(x3) + 0.5)
		currY := int(a*float64(y0) + b*float64(y1) + d*float64(y2) + e*float64(y3) + 0.5)

		c.DrawLine(prevX, prevY, currX, currY, style)
		prevX, prevY = currX, currY
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// DrawTexturedTriangle, doku (texture) haritalaması kullanarak Canvas üzerine dokulu bir üçgen çizer.
func (c *Canvas) DrawTexturedTriangle(p0, p1, p2 graphics.Vertex2D, uv0, uv1, uv2 graphics.UV, img image.Image) {
	// Üçgenin sınır kutusunu (bounding box) hesapla
	minX := int(math.Min(p0.X, math.Min(p1.X, p2.X)))
	maxX := int(math.Max(p0.X, math.Max(p1.X, p2.X)))
	minY := int(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))
	maxY := int(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))

	// Canvas sınırlarına kırp (clip)
	canvasW := int(c.width) * 2
	canvasH := int(c.height) * 4

	if minX < 0 { minX = 0 }
	if maxX >= canvasW { maxX = canvasW - 1 }
	if minY < 0 { minY = 0 }
	if maxY >= canvasH { maxY = canvasH - 1 }

	denom := (p1.Y - p2.Y)*(p0.X - p2.X) + (p2.X - p1.X)*(p0.Y - p2.Y)
	if math.Abs(denom) < 1e-6 {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	imgMinX := img.Bounds().Min.X
	imgMinY := img.Bounds().Min.Y

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			fx := float64(x)
			fy := float64(y)

			// Barycentric koordinatları hesapla
			lambda1 := ((p1.Y - p2.Y)*(fx - p2.X) + (p2.X - p1.X)*(fy - p2.Y)) / denom
			lambda2 := ((p2.Y - p0.Y)*(fx - p2.X) + (p0.X - p2.X)*(fy - p2.Y)) / denom
			lambda3 := 1.0 - lambda1 - lambda2

			// Eğer piksel üçgenin içindeyse (küçük tolerans payı ile)
			if lambda1 >= -0.005 && lambda2 >= -0.005 && lambda3 >= -0.005 {
				u := lambda1*uv0.U + lambda2*uv1.U + lambda3*uv2.U
				v := lambda1*uv0.V + lambda2*uv1.V + lambda3*uv2.V

				// Doku koordinatlarını piksel koordinatlarına eşle
				tx := int(u * imgW)
				ty := int(v * imgH)
				if tx < 0 { tx = 0 }
				if tx >= int(imgW) { tx = int(imgW) - 1 }
				if ty < 0 { ty = 0 }
				if ty >= int(imgH) { ty = int(imgH) - 1 }

				col := img.At(imgMinX+tx, imgMinY+ty)
				r, g, b, a := col.RGBA()

				uR := uint8(r>>8)
				uG := uint8(g>>8)
				uB := uint8(b>>8)

				// Gri/beyaz dama tahtası arka planı tespit edip şeffaf yap (color-keying)
				isGray := false
				diff1 := int(uR) - int(uG)
				diff2 := int(uG) - int(uB)
				if diff1 < 0 { diff1 = -diff1 }
				if diff2 < 0 { diff2 = -diff2 }
				if diff1 <= 15 && diff2 <= 15 {
					isGray = true
				}

				// Sadece görünür ve arka plan olmayan pikselleri çiz
				if a > 32768 && !isGray {
					style := cell.Style{Fg: cell.NewColorRGB(uR, uG, uB)}
					c.Set(x, y, style)
				}
			}
		}
	}
}
