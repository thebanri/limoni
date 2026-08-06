package cell

// Rect terminal ekranındaki dikdörtgen bir alanı temsil eder.
// Bellek Hizalaması: 2 (X) + 2 (Y) + 2 (Width) + 2 (Height) = 8 byte (tam kelime hizalı).
type Rect struct {
	X      uint16
	Y      uint16
	Width  uint16
	Height uint16
}

// NewRect yeni bir Rect örneği oluşturur.
func NewRect(x, y, w, h uint16) Rect {
	return Rect{X: x, Y: y, Width: w, Height: h}
}

// Intersection iki dikdörtgenin kesişim alanını hesaplar.
func (r Rect) Intersection(other Rect) Rect {
	x1 := r.X
	if other.X > x1 {
		x1 = other.X
	}
	y1 := r.Y
	if other.Y > y1 {
		y1 = other.Y
	}

	x2 := r.X + r.Width
	if other.X+other.Width < x2 {
		x2 = other.X + other.Width
	}
	y2 := r.Y + r.Height
	if other.Y+other.Height < y2 {
		y2 = other.Y + other.Height
	}

	if x1 >= x2 || y1 >= y2 {
		return Rect{}
	}
	return Rect{
		X:      x1,
		Y:      y1,
		Width:  x2 - x1,
		Height: y2 - y1,
	}
}

// Contains belirtilen koordinatın dikdörtgen içinde olup olmadığını kontrol eder.
func (r Rect) Contains(x, y uint16) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}
