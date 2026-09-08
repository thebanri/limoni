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

	// Test binary STL with extra trailing padding bytes (e.g. CAD metadata)
	dataWithPadding := append(data, []byte{0xDE, 0xAD, 0xBE, 0xEF}...)
	modelPad, err := ParseSTL(dataWithPadding)
	if err != nil {
		t.Fatalf("expected binary STL with trailing padding to parse successfully, got: %v", err)
	}
	if len(modelPad.Vertices) != 3 || len(modelPad.Faces) != 1 {
		t.Fatalf("padded binary model = %+v", modelPad)
	}

	// Test binary STL with zero triangle count
	zeroData := make([]byte, 84)
	if _, err := ParseSTL(zeroData); err == nil {
		t.Fatalf("expected error for empty/zero-triangle STL")
	}
}
