package graphics

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

//go:embed duck.glb
var embeddedDuckGLB []byte

// LoadModel loads a 3D model from disk, automatically detecting the format
// (.glb, .gltf, .obj, .stl, .ply) by file extension or content magic headers.
func LoadModel(path string) (Model3D, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".glb":
		return LoadGLB(path)
	case ".gltf":
		return LoadGLTF(path)
	case ".obj":
		return LoadOBJ(path)
	case ".stl":
		return LoadSTL(path)
	case ".ply":
		return LoadPLY(path)
	}

	// Fallback to inspecting initial file bytes
	data, err := os.ReadFile(path)
	if err != nil {
		return Model3D{}, err
	}

	if len(data) >= 4 && string(data[:4]) == "glTF" {
		return ParseGLB(data)
	}
	if bytes.HasPrefix(data, []byte("ply")) {
		return ParsePLY(bytes.NewReader(data))
	}
	if bytes.HasPrefix(data, []byte("solid")) || (len(data) >= 84 && !bytes.Contains(data[:80], []byte{0})) {
		if model, err := ParseSTL(data); err == nil && len(model.Vertices) > 0 {
			return model, nil
		}
	}
	if strings.Contains(string(data[:min(len(data), 512)]), "v ") {
		return ParseOBJ(bytes.NewReader(data))
	}

	return Model3D{}, fmt.Errorf("unknown or unsupported 3D model format for %q", path)
}

// NewDuck creates the canonical high-resolution 3D rubber duck model (Khronos Duck.glb)
// with vibrant yellow body, bright orange beak, and black eyes.
func NewDuck() Model3D {
	var model Model3D
	var err error

	if len(embeddedDuckGLB) > 0 {
		model, err = ParseGLB(embeddedDuckGLB)
	}
	if err != nil || len(model.Vertices) == 0 {
		// Fallback to file if present
		model, err = LoadGLB("examples/ascii3d/duck.glb")
	}

	if err == nil && len(model.Vertices) > 0 {
		model.Name = "Duck"
		model.Normalize(2.0)
		return model
	}

	// Fallback procedural duck
	return generateProceduralDuck()
}

func generateProceduralDuck() Model3D {
	model := Model3D{
		Name: "Duck",
	}

	yellowBody := cell.NewColorRGB(255, 225, 20)
	yellowHead := cell.NewColorRGB(255, 230, 30)
	yellowAccent := cell.NewColorRGB(245, 210, 15)
	orangeBeak := cell.NewColorRGB(255, 95, 0)
	blackEye := cell.NewColorRGB(20, 20, 20)

	addSpherePart := func(center Vertex3D, radiusX, radiusY, radiusZ float64, lats, longs int, faceCol cell.Color) {
		startIdx := len(model.Vertices)
		for lat := 0; lat <= lats; lat++ {
			theta := float64(lat) * math.Pi / float64(lats)
			sinTheta, cosTheta := math.Sin(theta), math.Cos(theta)
			for lon := 0; lon <= longs; lon++ {
				phi := float64(lon) * 2 * math.Pi / float64(longs)
				sinPhi, cosPhi := math.Sin(phi), math.Cos(phi)
				x := center.X + radiusX*sinTheta*cosPhi
				y := center.Y + radiusY*cosTheta
				z := center.Z + radiusZ*sinTheta*sinPhi
				model.Vertices = append(model.Vertices, Vertex3D{X: x, Y: y, Z: z})
			}
		}

		for lat := 0; lat < lats; lat++ {
			for lon := 0; lon < longs; lon++ {
				first := startIdx + lat*(longs+1) + lon
				second := first + longs + 1
				model.Faces = append(model.Faces, []int{first, second, first + 1})
				model.FaceColors = append(model.FaceColors, faceCol)
				model.Faces = append(model.Faces, []int{second, second + 1, first + 1})
				model.FaceColors = append(model.FaceColors, faceCol)
			}
		}
	}

	addSpherePart(Vertex3D{X: 0, Y: -0.25, Z: -0.05}, 0.90, 0.72, 1.10, 14, 18, yellowBody)
	addSpherePart(Vertex3D{X: 0, Y: 0.28, Z: 0.40}, 0.58, 0.52, 0.58, 10, 14, yellowBody)
	addSpherePart(Vertex3D{X: 0, Y: 0.75, Z: 0.52}, 0.56, 0.56, 0.56, 12, 16, yellowHead)
	addSpherePart(Vertex3D{X: 0, Y: 0.62, Z: 0.92}, 0.30, 0.16, 0.38, 10, 14, orangeBeak)
	addSpherePart(Vertex3D{X: -0.38, Y: 0.88, Z: 0.74}, 0.10, 0.10, 0.10, 6, 8, blackEye)
	addSpherePart(Vertex3D{X: 0.38, Y: 0.88, Z: 0.74}, 0.10, 0.10, 0.10, 6, 8, blackEye)
	addSpherePart(Vertex3D{X: -0.80, Y: -0.15, Z: 0.05}, 0.22, 0.42, 0.65, 8, 10, yellowAccent)
	addSpherePart(Vertex3D{X: 0.80, Y: -0.15, Z: 0.05}, 0.22, 0.42, 0.65, 8, 10, yellowAccent)

	model.Normalize(2.0)
	return model
}
