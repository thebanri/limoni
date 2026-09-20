package graphics

import (
	"math"
	"testing"
)

func TestVertex3DRotations(t *testing.T) {
	// 90 derece rotasyon testleri
	v := Vertex3D{X: 1, Y: 0, Z: 0}

	// 90 degrees around Z: (1, 0, 0) -> (0, 1, 0)
	vRotZ := v.RotateZ(90.0)
	if math.Abs(vRotZ.X) > 0.0001 || math.Abs(vRotZ.Y-1.0) > 0.0001 || math.Abs(vRotZ.Z) > 0.0001 {
		t.Errorf("rotation around Z: got %+v, expected (0, 1, 0)", vRotZ)
	}

	// 90 degrees around Y: (1, 0, 0) -> (0, 0, -1)
	vRotY := v.RotateY(90.0)
	if math.Abs(vRotY.X) > 0.0001 || math.Abs(vRotY.Y) > 0.0001 || math.Abs(vRotY.Z+1.0) > 0.0001 {
		t.Errorf("rotation around Y: got %+v, expected (0, 0, -1)", vRotY)
	}
}

func TestProjection(t *testing.T) {
	v := Vertex3D{X: 0, Y: 0, Z: 0}

	// The origin lands in the middle of the screen.
	x, y, visible := Project(v, 100, 100, 5.0, 50.0)
	if !visible {
		t.Error("the centre point should be visible")
	}
	if x != 50.0 || y != 50.0 {
		t.Errorf("projection centre: (%f, %f), expected (50, 50)", x, y)
	}
}
