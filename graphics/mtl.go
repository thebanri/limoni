package graphics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Material3D contains the diffuse color of an OBJ material.
type Material3D struct {
	Name string
	R    uint8
	G    uint8
	B    uint8
}

func LoadMTL(path string) (map[string]Material3D, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ParseMTL(file)
}

func ParseMTL(r io.Reader) (map[string]Material3D, error) {
	materials := make(map[string]Material3D)
	var current *Material3D
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch fields[0] {
		case "newmtl":
			if len(fields) < 2 {
				return nil, fmt.Errorf("line %d: newmtl requires a name", lineNo)
			}
			if current != nil {
				materials[current.Name] = *current
			}
			current = &Material3D{Name: fields[1], R: 255, G: 255, B: 255}
		case "Kd":
			if current == nil || len(fields) < 4 {
				continue
			}
			r, err1 := strconv.ParseFloat(fields[1], 64)
			g, err2 := strconv.ParseFloat(fields[2], 64)
			b, err3 := strconv.ParseFloat(fields[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("line %d: invalid Kd", lineNo)
			}
			current.R = toMaterialByte(r)
			current.G = toMaterialByte(g)
			current.B = toMaterialByte(b)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		materials[current.Name] = *current
	}
	return materials, nil
}

func toMaterialByte(value float64) uint8 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return uint8(value*255 + 0.5)
}
