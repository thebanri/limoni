package graphics

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

const (
	glbMagic  = 0x46546C67 // "glTF"
	chunkJSON = 0x4E4F534A // "JSON"
	chunkBIN  = 0x004E4942 // "BIN\0"

	compTypeUnsignedByte  = 5121
	compTypeUnsignedShort = 5123
	compTypeUnsignedInt   = 5125
	compTypeFloat         = 5126
)

type gltfJSON struct {
	Scene       *int             `json:"scene,omitempty"`
	Scenes      []gltfScene      `json:"scenes,omitempty"`
	Nodes       []gltfNode       `json:"nodes,omitempty"`
	Meshes      []gltfMesh       `json:"meshes,omitempty"`
	Accessors   []gltfAccessor   `json:"accessors,omitempty"`
	BufferViews []gltfBufferView `json:"bufferViews,omitempty"`
	Buffers     []gltfBuffer     `json:"buffers,omitempty"`
	Materials   []gltfMaterial   `json:"materials,omitempty"`
	Textures    []gltfTexture    `json:"textures,omitempty"`
	Images      []gltfImage      `json:"images,omitempty"`
}

type gltfScene struct {
	Name  string `json:"name,omitempty"`
	Nodes []int  `json:"nodes,omitempty"`
}

type gltfNode struct {
	Name     string `json:"name,omitempty"`
	Mesh     *int   `json:"mesh,omitempty"`
	Children []int  `json:"children,omitempty"`
}

type gltfMesh struct {
	Name       string          `json:"name,omitempty"`
	Primitives []gltfPrimitive `json:"primitives,omitempty"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices,omitempty"`
	Material   *int           `json:"material,omitempty"`
	Mode       *int           `json:"mode,omitempty"` // default 4 = TRIANGLES
}

type gltfMaterial struct {
	Name                 string                    `json:"name,omitempty"`
	PbrMetallicRoughness *gltfPbrMetallicRoughness `json:"pbrMetallicRoughness,omitempty"`
}

type gltfPbrMetallicRoughness struct {
	BaseColorFactor  []float64        `json:"baseColorFactor,omitempty"`
	BaseColorTexture *gltfTextureInfo `json:"baseColorTexture,omitempty"`
	MetallicFactor   *float64         `json:"metallicFactor,omitempty"`
	RoughnessFactor  *float64         `json:"roughnessFactor,omitempty"`
}

type gltfTextureInfo struct {
	Index int `json:"index"`
}

type gltfTexture struct {
	Source *int `json:"source,omitempty"`
}

type gltfImage struct {
	BufferView *int   `json:"bufferView,omitempty"`
	MimeType   string `json:"mimeType,omitempty"`
	URI        string `json:"uri,omitempty"`
}

type gltfAccessor struct {
	BufferView    *int      `json:"bufferView,omitempty"`
	ByteOffset    int       `json:"byteOffset,omitempty"`
	ComponentType int       `json:"componentType"`
	Count         int       `json:"count"`
	Type          string    `json:"type"` // "SCALAR", "VEC2", "VEC3", etc.
	Max           []float64 `json:"max,omitempty"`
	Min           []float64 `json:"min,omitempty"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset,omitempty"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride,omitempty"`
}

type gltfBuffer struct {
	ByteLength int    `json:"byteLength"`
	URI        string `json:"uri,omitempty"`
}

// LoadGLB loads a binary glTF 2.0 (.glb) file and resolves textures and geometry.
func LoadGLB(path string) (Model3D, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Model3D{}, err
	}
	model, err := ParseGLB(data)
	if err != nil {
		return Model3D{}, fmt.Errorf("parse GLB %q: %w", path, err)
	}
	model.Name = path
	return model, nil
}

// ParseGLB parses binary glTF 2.0 data from a byte slice.
func ParseGLB(data []byte) (Model3D, error) {
	if len(data) < 12 {
		return Model3D{}, fmt.Errorf("invalid GLB: header too short (%d bytes)", len(data))
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != glbMagic {
		return Model3D{}, fmt.Errorf("invalid GLB magic: 0x%08X (expected 0x%08X)", magic, glbMagic)
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 2 {
		return Model3D{}, fmt.Errorf("unsupported GLB version: %d (expected 2)", version)
	}

	totalLength := binary.LittleEndian.Uint32(data[8:12])
	if int(totalLength) > len(data) {
		return Model3D{}, fmt.Errorf("GLB length header (%d) exceeds file size (%d)", totalLength, len(data))
	}

	offset := 12
	var jsonChunk []byte
	var binChunk []byte

	for offset+8 <= len(data) {
		chunkLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		chunkType := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8

		if offset+chunkLen > len(data) {
			return Model3D{}, fmt.Errorf("chunk extends beyond GLB payload")
		}

		chunkData := data[offset : offset+chunkLen]
		offset += chunkLen

		switch chunkType {
		case chunkJSON:
			jsonChunk = chunkData
		case chunkBIN:
			binChunk = chunkData
		}
	}

	if len(jsonChunk) == 0 {
		return Model3D{}, fmt.Errorf("GLB missing JSON chunk")
	}

	var doc gltfJSON
	if err := json.Unmarshal(jsonChunk, &doc); err != nil {
		return Model3D{}, fmt.Errorf("decode GLB JSON chunk: %w", err)
	}

	buffers := [][]byte{binChunk}
	return buildModelFromGLTF(&doc, buffers, "")
}

// LoadGLTF loads a text/JSON glTF (.gltf) file and resolves referenced binary buffers and images.
func LoadGLTF(path string) (Model3D, error) {
	file, err := os.Open(path)
	if err != nil {
		return Model3D{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return Model3D{}, err
	}

	var doc gltfJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return Model3D{}, fmt.Errorf("decode glTF JSON %q: %w", path, err)
	}

	baseDir := filepath.Dir(path)
	buffers := make([][]byte, len(doc.Buffers))
	for i, bufDef := range doc.Buffers {
		if strings.HasPrefix(bufDef.URI, "data:") {
			idx := strings.Index(bufDef.URI, ",")
			if idx == -1 {
				return Model3D{}, fmt.Errorf("invalid data URI in buffer %d", i)
			}
			raw, err := base64.StdEncoding.DecodeString(bufDef.URI[idx+1:])
			if err != nil {
				return Model3D{}, fmt.Errorf("decode base64 buffer %d: %w", i, err)
			}
			buffers[i] = raw
		} else if bufDef.URI != "" {
			bufPath := filepath.Join(baseDir, bufDef.URI)
			raw, err := os.ReadFile(bufPath)
			if err != nil {
				return Model3D{}, fmt.Errorf("read glTF buffer %q: %w", bufPath, err)
			}
			buffers[i] = raw
		}
	}

	model, err := buildModelFromGLTF(&doc, buffers, path)
	if err != nil {
		return Model3D{}, fmt.Errorf("build glTF model %q: %w", path, err)
	}
	model.Name = path
	return model, nil
}

type uvCoord struct {
	u, v float64
}

func buildModelFromGLTF(doc *gltfJSON, buffers [][]byte, sourcePath string) (Model3D, error) {
	model := Model3D{
		Name:      sourcePath,
		Materials: make(map[string]Material3D),
	}

	baseDir := ""
	if sourcePath != "" {
		baseDir = filepath.Dir(sourcePath)
	}

	getAccessorBytes := func(accIdx int) ([]byte, gltfAccessor, int, error) {
		if accIdx < 0 || accIdx >= len(doc.Accessors) {
			return nil, gltfAccessor{}, 0, fmt.Errorf("accessor index %d out of range", accIdx)
		}
		acc := doc.Accessors[accIdx]
		if acc.BufferView == nil {
			return nil, acc, 0, fmt.Errorf("accessor %d has no bufferView", accIdx)
		}
		bvIdx := *acc.BufferView
		if bvIdx < 0 || bvIdx >= len(doc.BufferViews) {
			return nil, acc, 0, fmt.Errorf("bufferView index %d out of range", bvIdx)
		}
		bv := doc.BufferViews[bvIdx]
		if bv.Buffer < 0 || bv.Buffer >= len(buffers) {
			return nil, acc, 0, fmt.Errorf("buffer index %d out of range", bv.Buffer)
		}
		bufData := buffers[bv.Buffer]
		start := bv.ByteOffset + acc.ByteOffset
		end := start + bv.ByteLength
		if start < 0 || start > len(bufData) {
			return nil, acc, 0, fmt.Errorf("bufferView offset out of range")
		}
		if end > len(bufData) {
			end = len(bufData)
		}
		stride := bv.ByteStride
		return bufData[start:end], acc, stride, nil
	}

	// Decode all embedded or referenced images
	decodedImages := make([]image.Image, len(doc.Images))
	for i, imgDef := range doc.Images {
		if imgDef.BufferView != nil && *imgDef.BufferView >= 0 && *imgDef.BufferView < len(doc.BufferViews) {
			bv := doc.BufferViews[*imgDef.BufferView]
			if bv.Buffer >= 0 && bv.Buffer < len(buffers) {
				bufData := buffers[bv.Buffer]
				start := bv.ByteOffset
				end := start + bv.ByteLength
				if start >= 0 && end <= len(bufData) {
					if img, _, err := image.Decode(bytes.NewReader(bufData[start:end])); err == nil {
						decodedImages[i] = img
					}
				}
			}
		} else if imgDef.URI != "" {
			if strings.HasPrefix(imgDef.URI, "data:") {
				idx := strings.Index(imgDef.URI, ",")
				if idx != -1 {
					if raw, err := base64.StdEncoding.DecodeString(imgDef.URI[idx+1:]); err == nil {
						if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
							decodedImages[i] = img
						}
					}
				}
			} else if baseDir != "" {
				imgPath := filepath.Join(baseDir, imgDef.URI)
				if raw, err := os.ReadFile(imgPath); err == nil {
					if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
						decodedImages[i] = img
					}
				}
			}
		}
	}

	for _, mesh := range doc.Meshes {
		for _, prim := range mesh.Primitives {
			posAccIdx, hasPos := prim.Attributes["POSITION"]
			if !hasPos {
				continue
			}

			posBytes, posAcc, posStride, err := getAccessorBytes(posAccIdx)
			if err != nil {
				return Model3D{}, err
			}

			vertexOffset := len(model.Vertices)
			elemSize := 12 // 3 * float32
			if posStride < elemSize {
				posStride = elemSize
			}

			for i := 0; i < posAcc.Count; i++ {
				offset := i * posStride
				if offset+12 > len(posBytes) {
					break
				}
				x := math.Float32frombits(binary.LittleEndian.Uint32(posBytes[offset:]))
				y := math.Float32frombits(binary.LittleEndian.Uint32(posBytes[offset+4:]))
				z := math.Float32frombits(binary.LittleEndian.Uint32(posBytes[offset+8:]))
				model.Vertices = append(model.Vertices, Vertex3D{
					X: float64(x),
					Y: float64(y),
					Z: float64(z),
				})
			}

			// UV Coordinates (TEXCOORD_0)
			var uvs []uvCoord
			if uvAccIdx, hasUV := prim.Attributes["TEXCOORD_0"]; hasUV {
				if uvBytes, uvAcc, uvStride, err := getAccessorBytes(uvAccIdx); err == nil {
					uvElemSize := 8 // 2 * float32
					if uvStride < uvElemSize {
						uvStride = uvElemSize
					}
					for i := 0; i < uvAcc.Count; i++ {
						offset := i * uvStride
						if offset+8 > len(uvBytes) {
							break
						}
						u := math.Float32frombits(binary.LittleEndian.Uint32(uvBytes[offset:]))
						v := math.Float32frombits(binary.LittleEndian.Uint32(uvBytes[offset+4:]))
						uvs = append(uvs, uvCoord{u: float64(u), v: float64(v)})
					}
				}
			}

			// Material and Texture resolution
			var primImage image.Image
			var baseColor cell.Color

			if prim.Material != nil && *prim.Material >= 0 && *prim.Material < len(doc.Materials) {
				matDef := doc.Materials[*prim.Material]
				if matDef.PbrMetallicRoughness != nil {
					pbr := matDef.PbrMetallicRoughness
					if pbr.BaseColorTexture != nil {
						texIdx := pbr.BaseColorTexture.Index
						if texIdx >= 0 && texIdx < len(doc.Textures) {
							texDef := doc.Textures[texIdx]
							if texDef.Source != nil && *texDef.Source >= 0 && *texDef.Source < len(decodedImages) {
								primImage = decodedImages[*texDef.Source]
							}
						}
					}
					if len(pbr.BaseColorFactor) >= 3 {
						r := uint8(pbr.BaseColorFactor[0] * 255.0)
						g := uint8(pbr.BaseColorFactor[1] * 255.0)
						b := uint8(pbr.BaseColorFactor[2] * 255.0)
						baseColor = cell.NewColorRGB(r, g, b)
					}
				}
			}

			sampleTextureColor := func(u, v float64) cell.Color {
				if primImage == nil {
					if baseColor != 0 {
						return baseColor
					}
					return 0
				}
				bounds := primImage.Bounds()
				bw := bounds.Dx()
				bh := bounds.Dy()
				if bw <= 0 || bh <= 0 {
					return baseColor
				}

				// Wrap / clamp UV coordinates
				u = u - math.Floor(u)
				v = v - math.Floor(v)

				px := bounds.Min.X + int(u*float64(bw-1))
				py := bounds.Min.Y + int(v*float64(bh-1))

				r, g, b, _ := primImage.At(px, py).RGBA()
				return cell.NewColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			}

			// Indices / Faces
			if prim.Indices != nil {
				idxBytes, idxAcc, _, err := getAccessorBytes(*prim.Indices)
				if err != nil {
					return Model3D{}, err
				}

				indices := make([]int, 0, idxAcc.Count)
				r := bytes.NewReader(idxBytes)

				for i := 0; i < idxAcc.Count; i++ {
					switch idxAcc.ComponentType {
					case compTypeUnsignedByte:
						var b uint8
						if err := binary.Read(r, binary.LittleEndian, &b); err == nil {
							indices = append(indices, int(b))
						}
					case compTypeUnsignedShort:
						var s uint16
						if err := binary.Read(r, binary.LittleEndian, &s); err == nil {
							indices = append(indices, int(s))
						}
					case compTypeUnsignedInt:
						var u uint32
						if err := binary.Read(r, binary.LittleEndian, &u); err == nil {
							indices = append(indices, int(u))
						}
					}
				}

				for i := 0; i+2 < len(indices); i += 3 {
					i0, i1, i2 := indices[i], indices[i+1], indices[i+2]
					model.Faces = append(model.Faces, []int{
						vertexOffset + i0,
						vertexOffset + i1,
						vertexOffset + i2,
					})

					// Sample texture color for this triangle
					if len(uvs) > i0 && len(uvs) > i1 && len(uvs) > i2 {
						avgU := (uvs[i0].u + uvs[i1].u + uvs[i2].u) / 3.0
						avgV := (uvs[i0].v + uvs[i1].v + uvs[i2].v) / 3.0
						model.FaceColors = append(model.FaceColors, sampleTextureColor(avgU, avgV))
					} else {
						model.FaceColors = append(model.FaceColors, baseColor)
					}
				}
			} else {
				// Non-indexed triangles
				added := len(model.Vertices) - vertexOffset
				for i := 0; i+2 < added; i += 3 {
					model.Faces = append(model.Faces, []int{
						vertexOffset + i,
						vertexOffset + i + 1,
						vertexOffset + i + 2,
					})

					if len(uvs) > i+2 {
						avgU := (uvs[i].u + uvs[i+1].u + uvs[i+2].u) / 3.0
						avgV := (uvs[i].v + uvs[i+1].v + uvs[i+2].v) / 3.0
						model.FaceColors = append(model.FaceColors, sampleTextureColor(avgU, avgV))
					} else {
						model.FaceColors = append(model.FaceColors, baseColor)
					}
				}
			}
		}
	}

	if len(model.Vertices) == 0 {
		return Model3D{}, fmt.Errorf("glTF contains no vertices")
	}

	return model, nil
}
