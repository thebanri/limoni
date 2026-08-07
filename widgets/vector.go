package widgets

import "github.com/thebanri/limoni/core/cell"

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
