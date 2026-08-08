package graphics

import (
	"math"
)

// Vertex3D 3 boyutlu uzayda bir noktayı temsil eder.
type Vertex3D struct {
	X, Y, Z float64
}

// Vertex2D 2 boyutlu bir noktayı temsil eder.
type Vertex2D struct {
	X, Y float64
}

// UV doku (texture) koordinatlarını temsil eder.
type UV struct {
	U, V float64
}

// RotateX, noktayı X ekseni etrafında belirtilen derece cinsinden döndürür.
func (v Vertex3D) RotateX(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X,
		Y: v.Y*cos - v.Z*sin,
		Z: v.Y*sin + v.Z*cos,
	}
}

// RotateY, noktayı Y ekseni etrafında belirtilen derece cinsinden döndürür.
func (v Vertex3D) RotateY(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// RotateZ, noktayı Z ekseni etrafında belirtilen derece cinsinden döndürür.
func (v Vertex3D) RotateZ(angle float64) Vertex3D {
	rad := angle * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)
	return Vertex3D{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
		Z: v.Z,
	}
}

// Project, 3D koordinatı 2D ekran koordinatlarına (perspektif projeksiyon) dönüştürür.
// distance kameranın objeye olan mesafesi, scale ise ekrandaki büyüklük çarpanıdır.
func Project(v Vertex3D, screenW, screenH, distance, scale float64) (x, y float64, visible bool) {
	// Obje kameranın arkasında kalıyorsa çizme
	if v.Z+distance <= 0.1 {
		return 0, 0, false
	}

	factor := scale / (v.Z + distance)
	x = screenW/2.0 + v.X*factor
	// TUI koordinatlarında Y ekseni aşağı doğrudur, bu yüzden Y'yi ters çeviriyoruz.
	y = screenH/2.0 - v.Y*factor

	return x, y, true
}
