package graphics

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestParseASCIISTL(t *testing.T) {
	data := `solid triangle
facet normal 0 0 1
 outer loop
  vertex 0 0 0
  vertex 1 0 0
  vertex 0 1 0
 endloop
endfacet
endsolid triangle`
	model, err := ParseSTL([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 3 || len(model.Faces) != 1 {
		t.Fatalf("model = %+v", model)
	}
}

func TestParseBinarySTL(t *testing.T) {
	data := make([]byte, 84+50)
	binary.LittleEndian.PutUint32(data[80:84], 1)
	values := []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}
	offset := 96
	for _, value := range values {
		binary.LittleEndian.PutUint32(data[offset:], math.Float32bits(value))
		offset += 4
	}
	model, err := ParseSTL(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 3 || len(model.Faces) != 1 {
		t.Fatalf("binary model = %+v", model)
	}

}
