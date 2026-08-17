package graphics

import (
	"bufio"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Model3D is the geometry consumed by the wireframe/solid/textured renderer.
type Model3D struct {
	Name          string
	Vertices      []Vertex3D
	Faces         [][]int
	FaceUVs       [][]int
	UVs           []UV
	FaceMaterials []string
	Materials     map[string]Material3D
	Texture       image.Image
	TexturePath   string
}

// LoadTexture loads an image from disk and sets it as the model's texture.
func (m *Model3D) LoadTexture(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	m.Texture = img
	m.TexturePath = path
	return nil
}

// SetTexture sets an in-memory image as the model's texture.
func (m *Model3D) SetTexture(img image.Image) {
	m.Texture = img
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
	model.Materials = make(map[string]Material3D)
	if err := loadOBJMaterialLibraries(path, &model); err != nil {
		return Model3D{}, err
	}
	return model, nil
}

// ParseOBJ parses OBJ geometry from a reader.
func ParseOBJ(r io.Reader) (Model3D, error) {
	var model Model3D
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	lineNo := 0
	currentMaterial := ""
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
		case "usemtl":
			if len(fields) >= 2 {
				currentMaterial = fields[1]
			}
		case "vt":
			if len(fields) < 2 {
				return Model3D{}, fmt.Errorf("line %d: texture coordinate requires u", lineNo)
			}
			u, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return Model3D{}, fmt.Errorf("line %d: invalid texture u: %w", lineNo, err)
			}
			v := 0.0
			if len(fields) >= 3 {
				v, err = strconv.ParseFloat(fields[2], 64)
				if err != nil {
					return Model3D{}, fmt.Errorf("line %d: invalid texture v: %w", lineNo, err)
				}
			}
			model.UVs = append(model.UVs, UV{U: u, V: v})
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
			faceUVs := make([]int, 0, len(fields)-1)
			for _, token := range fields[1:] {
				vertexIndex, uvIndex, err := parseOBJIndices(token, len(model.Vertices), len(model.UVs))
				if err != nil {
					return Model3D{}, fmt.Errorf("line %d: %w", lineNo, err)
				}
				face = append(face, vertexIndex)
				faceUVs = append(faceUVs, uvIndex)
			}
			// Keep polygons; the renderer can draw their edges and fill them.
			model.Faces = append(model.Faces, face)
			model.FaceUVs = append(model.FaceUVs, faceUVs)
			model.FaceMaterials = append(model.FaceMaterials, currentMaterial)
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

func loadOBJMaterialLibraries(objPath string, model *Model3D) error {
	file, err := os.Open(objPath)
	if err != nil {
		return err
	}
	defer file.Close()
	baseDir := filepath.Dir(objPath)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) >= 2 && fields[0] == "mtllib" {
			materials, loadErr := LoadMTL(filepath.Join(baseDir, fields[1]))
			if loadErr != nil {
				return fmt.Errorf("load MTL %q: %w", fields[1], loadErr)
			}
			for name, material := range materials {
				model.Materials[name] = material
			}
		}
	}
	return scanner.Err()
}

func parseOBJIndices(token string, vertexCount, uvCount int) (int, int, error) {
	parts := strings.Split(token, "/")
	vertexIndex, err := strconv.Atoi(parts[0])
	if err != nil || vertexIndex == 0 {
		return 0, -1, fmt.Errorf("invalid face vertex %q", token)
	}
	if vertexIndex < 0 {
		vertexIndex = vertexCount + vertexIndex
	} else {
		vertexIndex--
	}
	if vertexIndex < 0 || vertexIndex >= vertexCount {
		return 0, -1, fmt.Errorf("face vertex %q is out of range", token)
	}
	uvIndex := -1
	if len(parts) >= 2 && parts[1] != "" && uvCount > 0 {
		uvIndex, err = strconv.Atoi(parts[1])
		if err != nil || uvIndex == 0 {
			return 0, -1, fmt.Errorf("invalid face UV %q", token)
		}
		if uvIndex < 0 {
			uvIndex = uvCount + uvIndex
		} else {
			uvIndex--
		}
		if uvIndex < 0 || uvIndex >= uvCount {
			return 0, -1, fmt.Errorf("face UV %q is out of range", token)
		}
	}
	return vertexIndex, uvIndex, nil
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
