package graphics

import (
	"testing"
)

func TestPrimitives(t *testing.T) {
	// 1. Cube
	cube := NewCube(2.0)
	if len(cube.Vertices) != 8 {
		t.Fatalf("expected 8 vertices for cube, got %d", len(cube.Vertices))
	}
	if len(cube.Faces) != 6 {
		t.Fatalf("expected 6 faces for cube, got %d", len(cube.Faces))
	}
	if len(cube.UVs) != 4 {
		t.Fatalf("expected 4 UV coords for cube, got %d", len(cube.UVs))
	}

	// 2. Pyramid
	pyramid := NewPyramid(2.0, 3.0)
	if len(pyramid.Vertices) != 5 {
		t.Fatalf("expected 5 vertices for pyramid, got %d", len(pyramid.Vertices))
	}
	if len(pyramid.Faces) != 5 {
		t.Fatalf("expected 5 faces for pyramid, got %d", len(pyramid.Faces))
	}

	// 3. Sphere
	sphere := NewSphere(1.0, 10, 10)
	if len(sphere.Vertices) == 0 || len(sphere.Faces) == 0 {
		t.Fatalf("expected non-empty sphere geometry")
	}

	// 4. Torus
	torus := NewTorus(0.8, 0.35, 10, 10)
	if len(torus.Vertices) == 0 || len(torus.Faces) == 0 {
		t.Fatalf("expected non-empty torus geometry")
	}
}

func TestModel3DTextureMethods(t *testing.T) {
	cube := NewCube(2.0)
	if cube.Texture != nil {
		t.Fatalf("expected nil texture initially")
	}

	// Test SetTexture
	cube.SetTexture(nil)
	if cube.Texture != nil {
		t.Fatalf("expected nil texture after setting nil")
	}

	// Test LoadTexture invalid path
	err := cube.LoadTexture("non_existent_file_xyz.png")
	if err == nil {
		t.Fatalf("expected error loading non-existent image")
	}
}
