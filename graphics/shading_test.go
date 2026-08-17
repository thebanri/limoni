package graphics

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestVector3DMath(t *testing.T) {
	v1 := Vector3D{X: 1, Y: 0, Z: 0}
	v2 := Vector3D{X: 0, Y: 1, Z: 0}

	cross := v1.Cross(v2)
	if cross.X != 0 || cross.Y != 0 || cross.Z != 1 {
		t.Fatalf("unexpected cross product: %+v", cross)
	}

	norm := cross.Normalize()
	if norm.Z != 1.0 {
		t.Fatalf("unexpected normalized vector: %+v", norm)
	}

	dot := v1.Dot(v2)
	if dot != 0 {
		t.Fatalf("expected dot product 0, got %f", dot)
	}
}

func TestCalculateNormal(t *testing.T) {
	v0 := Vertex3D{X: 0, Y: 0, Z: 0}
	v1 := Vertex3D{X: 1, Y: 0, Z: 0}
	v2 := Vertex3D{X: 0, Y: 1, Z: 0}

	normal := CalculateNormal(v0, v1, v2)
	if normal.Z != 1.0 {
		t.Fatalf("expected normal (0, 0, 1), got %+v", normal)
	}
}

func TestLambertianLighting(t *testing.T) {
	light := DefaultLight()
	normal := Vector3D{X: 0, Y: 0, Z: -1}.Normalize()

	intensity := light.CalculateIntensity(normal)
	if intensity < 0.0 || intensity > 1.0 {
		t.Fatalf("intensity out of bounds: %f", intensity)
	}

	baseColor := cell.NewColorRGB(200, 100, 50)
	shaded := ApplyShade(baseColor, 0.5)
	r, g, b := shaded.RGB()
	if r != 100 || g != 50 || b != 25 {
		t.Fatalf("expected shaded RGB (100, 50, 25), got (%d, %d, %d)", r, g, b)
	}
}

func TestBarycentricColor(t *testing.T) {
	c0 := cell.NewColorRGB(255, 0, 0)
	c1 := cell.NewColorRGB(0, 255, 0)
	c2 := cell.NewColorRGB(0, 0, 255)

	// Center of triangle (1/3, 1/3, 1/3)
	centerColor := BarycentricColor(c0, c1, c2, 1.0/3.0, 1.0/3.0, 1.0/3.0)
	r, g, b := centerColor.RGB()
	if r != 85 || g != 85 || b != 85 {
		t.Fatalf("expected blended gray (85, 85, 85), got (%d, %d, %d)", r, g, b)
	}
}

func TestBackfaceCulling(t *testing.T) {
	p0 := Vertex2D{X: 0, Y: 0}
	p1 := Vertex2D{X: 10, Y: 0}
	p2 := Vertex2D{X: 5, Y: 10}

	// Clockwise in screen coordinates (Y downwards)
	if IsBackface(p0, p1, p2) {
		t.Fatal("expected front-facing triangle")
	}

	if !IsBackface(p0, p2, p1) {
		t.Fatal("expected back-facing triangle")
	}
}
