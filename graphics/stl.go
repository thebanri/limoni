package graphics

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// LoadSTL loads either ASCII or binary STL geometry.
func LoadSTL(path string) (Model3D, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Model3D{}, err
	}
	model, err := ParseSTL(data)
	if err != nil {
		return Model3D{}, fmt.Errorf("parse STL %q: %w", path, err)
	}
	model.Name = path
	return model, nil
}

// ParseSTL detects and parses binary or ASCII STL data.
func ParseSTL(data []byte) (Model3D, error) {
	if len(data) >= 84 {
		triangleCount := int(binary.LittleEndian.Uint32(data[80:84]))
		if triangleCount >= 0 && 84+triangleCount*50 == len(data) {
			return parseBinarySTL(data, triangleCount)
		}
	}
	return parseASCIISTL(strings.NewReader(string(data)))
}

func parseBinarySTL(data []byte, triangleCount int) (Model3D, error) {
	model := Model3D{Faces: make([][]int, 0, triangleCount)}
	offset := 84
	for i := 0; i < triangleCount; i++ {
		offset += 12 // normal
		face := make([]int, 3)
		for j := 0; j < 3; j++ {
			if offset+12 > len(data) {
				return Model3D{}, fmt.Errorf("binary triangle %d is truncated", i)
			}
			x := math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
			y := math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4:]))
			z := math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8:]))
			face[j] = len(model.Vertices)
			model.Vertices = append(model.Vertices, Vertex3D{X: float64(x), Y: float64(y), Z: float64(z)})
			offset += 12
		}
		model.Faces = append(model.Faces, face)
		offset += 2 // attribute byte count
	}
	return model, nil
}

func parseASCIISTL(r io.Reader) (Model3D, error) {
	model := Model3D{}
	scanner := bufio.NewScanner(r)
	var face []int
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 4 || fields[0] != "vertex" {
			continue
		}
		x, err1 := strconv.ParseFloat(fields[1], 64)
		y, err2 := strconv.ParseFloat(fields[2], 64)
		z, err3 := strconv.ParseFloat(fields[3], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			return Model3D{}, fmt.Errorf("invalid ASCII vertex")
		}
		face = append(face, len(model.Vertices))
		model.Vertices = append(model.Vertices, Vertex3D{X: x, Y: y, Z: z})
		if len(face) == 3 {
			model.Faces = append(model.Faces, face)
			face = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return Model3D{}, err
	}
	if len(model.Vertices) == 0 || len(model.Faces) == 0 {
		return Model3D{}, fmt.Errorf("STL contains no triangles")
	}
	return model, nil
}
