package graphics

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func createSampleGLB(t *testing.T) []byte {
	// Simple triangle GLB
	vertices := []float32{
		0.0, 1.0, 0.0,
		-1.0, -1.0, 0.0,
		1.0, -1.0, 0.0,
	}
	indices := []uint16{0, 1, 2}

	var binBuf bytes.Buffer
	for _, f := range vertices {
		_ = binary.Write(&binBuf, binary.LittleEndian, f)
	}
	for _, idx := range indices {
		_ = binary.Write(&binBuf, binary.LittleEndian, idx)
	}

	binData := binBuf.Bytes()
	for len(binData)%4 != 0 {
		binData = append(binData, 0x00)
	}

	bv0 := 0
	bv1 := 1
	posAcc := 0
	idxAcc := 1

	doc := gltfJSON{
		Meshes: []gltfMesh{
			{
				Name: "SampleMesh",
				Primitives: []gltfPrimitive{
					{
						Attributes: map[string]int{"POSITION": posAcc},
						Indices:    &idxAcc,
					},
				},
			},
		},
		Accessors: []gltfAccessor{
			{
				BufferView:    &bv0,
				ByteOffset:    0,
				ComponentType: compTypeFloat,
				Count:         3,
				Type:          "VEC3",
			},
			{
				BufferView:    &bv1,
				ByteOffset:    0,
				ComponentType: compTypeUnsignedShort,
				Count:         3,
				Type:          "SCALAR",
			},
		},
		BufferViews: []gltfBufferView{
			{
				Buffer:     0,
				ByteOffset: 0,
				ByteLength: 36, // 3 * 3 * 4
			},
			{
				Buffer:     0,
				ByteOffset: 36,
				ByteLength: 6, // 3 * 2
			},
		},
		Buffers: []gltfBuffer{
			{
				ByteLength: len(binData),
			},
		},
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}

	totalLen := 12 + 8 + len(jsonBytes) + 8 + len(binData)

	var glb bytes.Buffer
	// Header
	_ = binary.Write(&glb, binary.LittleEndian, uint32(glbMagic))
	_ = binary.Write(&glb, binary.LittleEndian, uint32(2))
	_ = binary.Write(&glb, binary.LittleEndian, uint32(totalLen))

	// Chunk 0 (JSON)
	_ = binary.Write(&glb, binary.LittleEndian, uint32(len(jsonBytes)))
	_ = binary.Write(&glb, binary.LittleEndian, uint32(chunkJSON))
	glb.Write(jsonBytes)

	// Chunk 1 (BIN)
	_ = binary.Write(&glb, binary.LittleEndian, uint32(len(binData)))
	_ = binary.Write(&glb, binary.LittleEndian, uint32(chunkBIN))
	glb.Write(binData)

	return glb.Bytes()
}

func TestParseGLB(t *testing.T) {
	glbBytes := createSampleGLB(t)

	model, err := ParseGLB(glbBytes)
	if err != nil {
		t.Fatalf("ParseGLB failed: %v", err)
	}

	if len(model.Vertices) != 3 {
		t.Fatalf("expected 3 vertices, got %d", len(model.Vertices))
	}
	if len(model.Faces) != 1 {
		t.Fatalf("expected 1 face, got %d", len(model.Faces))
	}

	if model.Vertices[0].Y != 1.0 || model.Vertices[1].X != -1.0 || model.Vertices[2].X != 1.0 {
		t.Errorf("unexpected vertices: %+v", model.Vertices)
	}
}

func TestLoadModel(t *testing.T) {
	tempDir := t.TempDir()
	glbFile := filepath.Join(tempDir, "test.glb")
	if err := os.WriteFile(glbFile, createSampleGLB(t), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	model, err := LoadModel(glbFile)
	if err != nil {
		t.Fatalf("LoadModel(glb) failed: %v", err)
	}
	if len(model.Vertices) != 3 {
		t.Errorf("expected 3 vertices, got %d", len(model.Vertices))
	}
}

func TestDuckTextureDecoding(t *testing.T) {
	duckPath := filepath.Join("..", "examples", "ascii3d", "duck.glb")
	model, err := LoadGLB(duckPath)
	if err != nil {
		t.Fatalf("LoadGLB duck.glb failed: %v", err)
	}

	yellowCount := 0
	orangeCount := 0
	blackCount := 0

	for _, c := range model.FaceColors {
		r, g, b := c.RGB()
		if r > 180 && g > 150 && b < 100 {
			yellowCount++
		}
		if r > 180 && g < 140 {
			orangeCount++
		}
		if r < 50 && g < 50 && b < 50 {
			blackCount++
		}
	}

	t.Logf("Duck.glb texture colors decoded: Yellow faces: %d, Orange beak faces: %d, Dark eye faces: %d (Total: %d)",
		yellowCount, orangeCount, blackCount, len(model.FaceColors))

	if yellowCount == 0 {
		t.Errorf("expected yellow body faces from texture")
	}
	if orangeCount == 0 {
		t.Errorf("expected orange beak faces from texture")
	}
}
