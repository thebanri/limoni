package graphics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// LoadPLY loads an ASCII PLY mesh.
func LoadPLY(path string) (Model3D, error) {
	file, err := os.Open(path)
	if err != nil {
		return Model3D{}, err
	}
	defer file.Close()
	model, err := ParsePLY(file)
	if err != nil {
		return Model3D{}, fmt.Errorf("parse PLY %q: %w", path, err)
	}
	model.Name = path
	return model, nil
}

// ParsePLY supports ASCII PLY files with x/y/z vertices and polygon faces.
func ParsePLY(r io.Reader) (Model3D, error) {
	scanner := bufio.NewScanner(r)
	vertexCount, faceCount := 0, 0
	inHeader := true
	for inHeader && scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "format":
			if len(fields) < 2 || fields[1] != "ascii" {
				return Model3D{}, fmt.Errorf("only ASCII PLY is supported")
			}
		case "element":
			if len(fields) >= 3 {
				if fields[1] == "vertex" {
					vertexCount, _ = strconv.Atoi(fields[2])
				}
				if fields[1] == "face" {
					faceCount, _ = strconv.Atoi(fields[2])
				}
			}
		case "end_header":
			inHeader = false
		}
	}
	if err := scanner.Err(); err != nil {
		return Model3D{}, err
	}
	if vertexCount <= 0 {
		return Model3D{}, fmt.Errorf("PLY has no vertices")
	}
	model := Model3D{Vertices: make([]Vertex3D, 0, vertexCount), Faces: make([][]int, 0, faceCount)}
	for len(model.Vertices) < vertexCount && scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		x, e1 := strconv.ParseFloat(fields[0], 64)
		y, e2 := strconv.ParseFloat(fields[1], 64)
		z, e3 := strconv.ParseFloat(fields[2], 64)
		if e1 != nil || e2 != nil || e3 != nil {
			return Model3D{}, fmt.Errorf("invalid PLY vertex")
		}
		model.Vertices = append(model.Vertices, Vertex3D{X: x, Y: y, Z: z})
	}
	for len(model.Faces) < faceCount && scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil || n < 3 || len(fields) < n+1 {
			return Model3D{}, fmt.Errorf("invalid PLY face")
		}
		face := make([]int, n)
		for i := 0; i < n; i++ {
			face[i], err = strconv.Atoi(fields[i+1])
			if err != nil || face[i] < 0 || face[i] >= len(model.Vertices) {
				return Model3D{}, fmt.Errorf("PLY face index out of range")
			}
		}
		model.Faces = append(model.Faces, face)
	}
	if len(model.Faces) == 0 {
		return Model3D{}, fmt.Errorf("PLY has no faces")
	}
	return model, nil
}
