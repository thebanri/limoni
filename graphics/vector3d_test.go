package graphics

import (
	"math"
	"testing"
)

func TestVertex3DRotations(t *testing.T) {
	// 90 derece rotasyon testleri
	v := Vertex3D{X: 1, Y: 0, Z: 0}

	// Z etrafında 90 derece rotasyon: (1, 0, 0) -> (0, 1, 0)
	vRotZ := v.RotateZ(90.0)
	if math.Abs(vRotZ.X) > 0.0001 || math.Abs(vRotZ.Y-1.0) > 0.0001 || math.Abs(vRotZ.Z) > 0.0001 {
		t.Errorf("Z rotasyonu başarısız: got %+v, expected (0, 1, 0)", vRotZ)
	}

	// Y etrafında 90 derece rotasyon: (1, 0, 0) -> (0, 0, -1)
	vRotY := v.RotateY(90.0)
	if math.Abs(vRotY.X) > 0.0001 || math.Abs(vRotY.Y) > 0.0001 || math.Abs(vRotY.Z+1.0) > 0.0001 {
		t.Errorf("Y rotasyonu başarısız: got %+v, expected (0, 0, -1)", vRotY)
	}
}

func TestProjection(t *testing.T) {
	v := Vertex3D{X: 0, Y: 0, Z: 0}

	// Ortalanmış olmalı
	x, y, visible := Project(v, 100, 100, 5.0, 50.0)
	if !visible {
		t.Error("Merkez nokta görünür olmalı")
	}
	if x != 50.0 || y != 50.0 {
		t.Errorf("Projeksiyon ortalaması başarısız: (%f, %f), expected (50, 50)", x, y)
	}
}
