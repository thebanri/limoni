package graphics

import (
	"strings"
	"testing"
)

func TestParseOBJ(t *testing.T) {
	model, err := ParseOBJ(strings.NewReader(`# square
v -1 -1 0
v 1 -1 0
v 1 1 0
v -1 1 0
f 1/1/1 2/2/1 3/3/1 4/4/1
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 4 || len(model.Faces) != 1 || len(model.Faces[0]) != 4 {
		t.Fatalf("model = %+v; want 4 vertices and one quad", model)
	}
}

func TestParseOBJTextureCoordinates(t *testing.T) {
	model, err := ParseOBJ(strings.NewReader("vt 0 0\nvt 1 0\nvt 0 1\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1/1 2/2 3/3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.UVs) != 3 || len(model.FaceUVs) != 1 || model.FaceUVs[0][2] != 2 {
		t.Fatalf("UV data = %+v / %+v", model.UVs, model.FaceUVs)
	}
}

func TestParseOBJNegativeIndices(t *testing.T) {
	model, err := ParseOBJ(strings.NewReader("v 0 0 0\nv 1 0 0\nv 0 1 0\nf -3 -2 -1\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 2}
	for i, got := range model.Faces[0] {
		if got != want[i] {
			t.Fatalf("face[%d] = %d; want %d", i, got, want[i])
		}
	}
}

func TestModelNormalize(t *testing.T) {
	model := Model3D{Vertices: []Vertex3D{{0, 0, 0}, {10, 0, 0}}}
	model.Normalize(2)
	if model.Vertices[0].X != -1 || model.Vertices[1].X != 1 {
		t.Fatalf("normalized vertices = %+v; want x values -1 and 1", model.Vertices)
	}
}
