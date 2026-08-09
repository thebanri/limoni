package graphics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Model3D is the geometry consumed by the wireframe/solid renderer.
type Model3D struct {
	Name     string
	Vertices []Vertex3D
	Faces    [][]int
}

// LoadOBJ loads a Wavefront OBJ file without external dependencies.
// It supports vertices and polygon faces in v, v/vt, v//vn and v/vt/vn forms.
func LoadOBJ(path string) (Model3D, error) {
	file, err := os.Open(path)
	if err != nil {
		return Model3D{}, err
	}
	defer file.Close()
	model, err := ParseOBJ(file)
	if err != nil {
		return Model3D{}, fmt.Errorf("parse OBJ %q: %w", path, err)
	}
	model.Name = path
	return model, nil
}

// ParseOBJ parses OBJ geometry from a reader.
func ParseOBJ(r io.Reader) (Model3D, error) {
	var model Model3D
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return Model3D{}, fmt.Errorf("line %d: vertex requires x y z", lineNo)
			}
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return Model3D{}, fmt.Errorf("line %d: invalid vertex x: %w", lineNo, err)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return Model3D{}, fmt.Errorf("line %d: invalid vertex y: %w", lineNo, err)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return Model3D{}, fmt.Errorf("line %d: invalid vertex z: %w", lineNo, err)
			}
			model.Vertices = append(model.Vertices, Vertex3D{X: x, Y: y, Z: z})
		case "f":
			if len(fields) < 4 {
				return Model3D{}, fmt.Errorf("line %d: face requires at least three vertices", lineNo)
			}
			face := make([]int, 0, len(fields)-1)
			for _, token := range fields[1:] {
				index, err := parseOBJVertexIndex(token, len(model.Vertices))
				if err != nil {
					return Model3D{}, fmt.Errorf("line %d: %w", lineNo, err)
				}
				face = append(face, index)
			}
			// Keep polygons; the renderer can draw their edges and fill them.
			model.Faces = append(model.Faces, face)
		}
	}
	if err := scanner.Err(); err != nil {
		return Model3D{}, err
	}
	if len(model.Vertices) == 0 || len(model.Faces) == 0 {
		return Model3D{}, fmt.Errorf("OBJ contains no renderable vertices or faces")
	}
	return model, nil
}

func parseOBJVertexIndex(token string, vertexCount int) (int, error) {
	parts := strings.SplitN(token, "/", 2)
	index, err := strconv.Atoi(parts[0])
	if err != nil || index == 0 {
		return 0, fmt.Errorf("invalid face vertex %q", token)
	}
	if index < 0 {
		index = vertexCount + index
	} else {
		index--
	}
	if index < 0 || index >= vertexCount {
		return 0, fmt.Errorf("face vertex %q is out of range", token)
	}
	return index, nil
}

// Normalize centers the model at the origin and scales its largest dimension
// to the requested size, making files with arbitrary units fit the viewport.
func (m *Model3D) Normalize(size float64) {
	if m == nil || len(m.Vertices) == 0 || size <= 0 {
		return
	}
	min, max := m.Vertices[0], m.Vertices[0]
	for _, v := range m.Vertices[1:] {
		if v.X < min.X {
			min.X = v.X
		}
		if v.Y < min.Y {
			min.Y = v.Y
		}
		if v.Z < min.Z {
			min.Z = v.Z
		}
		if v.X > max.X {
			max.X = v.X
		}
		if v.Y > max.Y {
			max.Y = v.Y
		}
		if v.Z > max.Z {
			max.Z = v.Z
		}
	}
	center := Vertex3D{X: (min.X + max.X) / 2, Y: (min.Y + max.Y) / 2, Z: (min.Z + max.Z) / 2}
	span := max.X - min.X
	if y := max.Y - min.Y; y > span {
		span = y
	}
	if z := max.Z - min.Z; z > span {
		span = z
	}
	if span == 0 {
		return
	}
	scale := size / span
	for i, v := range m.Vertices {
		m.Vertices[i] = Vertex3D{X: (v.X - center.X) * scale, Y: (v.Y - center.Y) * scale, Z: (v.Z - center.Z) * scale}
	}
}
