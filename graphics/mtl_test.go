package graphics

import (
	"strings"
	"testing"
)

func TestParseMTL(t *testing.T) {
	materials, err := ParseMTL(strings.NewReader("newmtl Red\nKd 1.0 0.25 0\nnewmtl Blue\nKd 0 0.5 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if materials["Red"].R != 255 || materials["Red"].G != 64 || materials["Red"].B != 0 {
		t.Fatalf("red material = %+v", materials["Red"])
	}
	if materials["Blue"].G != 128 {
		t.Fatalf("blue material = %+v", materials["Blue"])
	}
}

func TestParseOBJMaterials(t *testing.T) {
	model, err := ParseOBJ(strings.NewReader("usemtl Red\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.FaceMaterials) != 1 || model.FaceMaterials[0] != "Red" {
		t.Fatalf("face materials = %v", model.FaceMaterials)
	}
}
