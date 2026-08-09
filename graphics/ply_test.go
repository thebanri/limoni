package graphics

import (
	"strings"
	"testing"
)

func TestParseASCIIPLY(t *testing.T) {
	data := "ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nelement face 1\nproperty list uchar int vertex_indices\nend_header\n0 0 0\n1 0 0\n0 1 0\n3 0 1 2\n"
	model, err := ParsePLY(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 3 || len(model.Faces) != 1 {
		t.Fatalf("PLY model = %+v", model)
	}
}
