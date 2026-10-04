package graphics

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const cubeOBJ = `mtllib cube.mtl
mtllib missing.mtl
v -1 -1 -1
v 1 -1 -1
v 1 1 -1
v -1 1 -1
v -1 -1 1
v 1 -1 1
v 1 1 1
v -1 1 1
usemtl red
f 1 2 3 4
usemtl blue
f 5 6 7 8
`

const cubeMTL = `newmtl red
Kd 1 0 0
newmtl blue
Kd 0 0 1
`

// An OBJ on disk picks up its material library: each face gets its
// material's colour. A missing library is skipped, not an error.
func TestLoadOBJReadsItsMaterials(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cube.mtl", cubeMTL)
	path := writeFile(t, dir, "cube.obj", cubeOBJ)

	model, err := LoadOBJ(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 8 || len(model.Faces) != 2 || model.Name != path {
		t.Fatalf("got %d vertices, %d faces, name %q", len(model.Vertices), len(model.Faces), model.Name)
	}
	if len(model.FaceColors) != 2 || model.FaceColors[0] != cell.NewColorRGB(255, 0, 0) || model.FaceColors[1] != cell.NewColorRGB(0, 0, 255) {
		t.Errorf("face colours %v, want red then blue", model.FaceColors)
	}
	if _, err := LoadOBJ(filepath.Join(dir, "nope.obj")); err == nil {
		t.Error("a missing file loaded")
	}
}

// LoadModel picks the parser by extension, and by content when the
// extension says nothing.
func TestLoadModelByExtensionAndByContent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cube.mtl", cubeMTL)
	ply := "ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nelement face 1\nproperty list uchar int vertex_indices\nend_header\n0 0 0\n1 0 0\n0 1 0\n3 0 1 2\n"
	stl := "solid t\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid t\n"
	glb := createSampleGLB(t)

	cases := []struct {
		name, body string
		faces      int
	}{
		{"cube.obj", cubeOBJ, 2},
		{"tri.ply", ply, 1},
		{"tri.stl", stl, 1},
		{"tri.glb", string(glb), 1},
		// No telling extension: the content decides.
		{"cube.mesh", cubeOBJ, 2},
		{"tri.model", ply, 1},
		{"tri.data", stl, 1},
		{"tri.bin", string(glb), 1},
	}
	for _, c := range cases {
		model, err := LoadModel(writeFile(t, dir, c.name, c.body))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if len(model.Faces) != c.faces {
			t.Errorf("%s: %d faces, want %d", c.name, len(model.Faces), c.faces)
		}
	}
	if _, err := LoadModel(writeFile(t, dir, "notes.txt", "\x00\x01 nothing here")); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("an unknown format: %v", err)
	}
	for _, missing := range []string{"a.ply", "a.stl", "a.gltf", "a.glb", "a.unknown"} {
		if _, err := LoadModel(filepath.Join(dir, missing)); err == nil {
			t.Errorf("missing %s loaded", missing)
		}
	}
}

// A text glTF with its buffer inline as a data: URI.
func TestLoadGLTFWithAnInlineBuffer(t *testing.T) {
	var bin bytes.Buffer
	for _, f := range []float32{0, 1, 0, -1, -1, 0, 1, -1, 0} {
		_ = binary.Write(&bin, binary.LittleEndian, f)
	}
	for _, i := range []uint16{0, 1, 2, 0} { // the last is padding
		_ = binary.Write(&bin, binary.LittleEndian, i)
	}
	bv0, bv1 := 0, 1
	doc := gltfJSON{
		Meshes: []gltfMesh{{Primitives: []gltfPrimitive{{Attributes: map[string]int{"POSITION": 0}, Indices: &bv1}}}},
		Accessors: []gltfAccessor{
			{BufferView: &bv0, ComponentType: compTypeFloat, Count: 3, Type: "VEC3"},
			{BufferView: &bv1, ComponentType: compTypeUnsignedShort, Count: 3, Type: "SCALAR"},
		},
		BufferViews: []gltfBufferView{{Buffer: 0, ByteLength: 36}, {Buffer: 0, ByteOffset: 36, ByteLength: 6}},
		Buffers:     []gltfBuffer{{ByteLength: bin.Len(), URI: "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(bin.Bytes())}},
	}
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	model, err := LoadModel(writeFile(t, dir, "tri.gltf", string(body)))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Vertices) != 3 || len(model.Faces) != 1 || model.Vertices[0].Y != 1 {
		t.Errorf("got %d vertices %v, %d faces", len(model.Vertices), model.Vertices, len(model.Faces))
	}
	if _, err := LoadGLTF(writeFile(t, dir, "bad.gltf", "{not json")); err == nil {
		t.Error("broken JSON loaded")
	}
}

// The duck is built in: it needs no file and has the model's colours.
func TestNewDuck(t *testing.T) {
	duck := NewDuck()
	if len(duck.Vertices) < 100 || len(duck.Faces) < 100 {
		t.Fatalf("the duck has %d vertices and %d faces", len(duck.Vertices), len(duck.Faces))
	}
}

// The escape-sequence cache returns the same sequence for the same image
// and request, a different one for another protocol, and a fresh one once
// the image is forgotten and its pixels change.
func TestEscapeSequenceCache(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	kitty := GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolKitty, 0, false)
	if kitty == "" || kitty != GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolKitty, 0, false) {
		t.Fatalf("kitty sequence not cached: %q", kitty)
	}
	if sixel := GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolSixel, 0, false); sixel == kitty || !strings.HasPrefix(sixel, "\x1bP") {
		t.Errorf("sixel sequence %q", sixel)
	}
	if iterm := GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolIterm2, 0, false); !strings.Contains(iterm, "1337;File=") {
		t.Errorf("iTerm2 sequence %q", iterm)
	}

	img.Set(1, 1, color.RGBA{G: 255, A: 255})
	if stale := GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolKitty, 0, false); stale != kitty {
		t.Errorf("the cache did not answer for an image changed in place (that is what ForgetImage is for)")
	}
	ForgetImage(img)
	if fresh := GetCachedEscapeSequence(img, 2, 1, 2, 4, ProtocolKitty, 0, false); fresh == kitty {
		t.Errorf("after ForgetImage the old picture was sent again")
	}
}

// Images of any type hash by their pixels.
func TestGetImageIDOfOtherImageTypes(t *testing.T) {
	a := image.NewGray(image.Rect(0, 0, 3, 3))
	b := image.NewGray(image.Rect(0, 0, 3, 3))
	if GetImageID(a) != GetImageID(b) {
		t.Error("equal gray images hash differently")
	}
	b.SetGray(1, 1, color.Gray{Y: 200})
	if GetImageID(a) == GetImageID(b) {
		t.Error("different gray images hash the same")
	}
	if GetImageID(nil) != 0 {
		t.Error("nil image has an ID")
	}
	n := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	n.Set(0, 0, color.NRGBA{B: 9, A: 255})
	if GetImageID(n) == GetImageID(image.NewNRGBA(image.Rect(0, 0, 2, 2))) {
		t.Error("different NRGBA images hash the same")
	}
}

func TestCropImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(2, 3, color.RGBA{R: 7, A: 255})
	got := CropImage(img, image.Rect(2, 2, 10, 10))
	if got.Bounds() != image.Rect(0, 0, 2, 2) {
		t.Fatalf("bounds %v: the crop must stop at the image", got.Bounds())
	}
	if r, _, _, _ := got.At(0, 1).RGBA(); r>>8 != 7 {
		t.Errorf("pixel moved: got r=%d", r>>8)
	}
	if CropImage(img, image.Rect(10, 10, 12, 12)) != nil || CropImage(nil, image.Rect(0, 0, 1, 1)) != nil {
		t.Error("an empty crop is not nil")
	}
}
