package layout

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestBuilder_VBoxAndHBox(t *testing.T) {
	area := cell.NewRect(0, 0, 100, 50)

	// Vertical 3-way split: Header(5), Content(Fill), Footer(3)
	vRows := VBox(area, Fixed(5), Fill(), Fixed(3))
	if len(vRows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(vRows))
	}
	if vRows[0].Height != 5 || vRows[2].Height != 3 || vRows[1].Height != 42 {
		t.Errorf("unexpected row heights: %d, %d, %d", vRows[0].Height, vRows[1].Height, vRows[2].Height)
	}

	// Horizontal 2-way split: Sidebar(30%), Main(70%)
	hCols := HBox(vRows[1], Percentage(30), Percentage(70))
	if len(hCols) != 2 {
		t.Fatalf("expected 2 cols, got %d", len(hCols))
	}
	if hCols[0].Width != 30 || hCols[1].Width != 70 {
		t.Errorf("unexpected col widths: %d, %d", hCols[0].Width, hCols[1].Width)
	}
}

func TestBuilder_CenteredAndPadded(t *testing.T) {
	area := cell.NewRect(10, 10, 80, 40)

	center := Centered(area, 40, 20)
	if center.X != 30 || center.Y != 20 || center.Width != 40 || center.Height != 20 {
		t.Errorf("unexpected centered rect: %+v", center)
	}

	padded := Padded(area, 2, 2, 4, 4)
	if padded.X != 14 || padded.Y != 12 || padded.Width != 72 || padded.Height != 36 {
		t.Errorf("unexpected padded rect: %+v", padded)
	}
}

func TestBuilder_VBoxAndHBoxWithGap(t *testing.T) {
	area := cell.NewRect(0, 0, 100, 50)

	// Vertical split with gap: 3 items of Fixed(10) with gap 2
	// Heights: 10, 10, 10. Gap between 0-1 and 1-2 is 2.
	// Y coords: 0, 12, 24.
	vRows := VBoxWithGap(area, 2, Fixed(10), Fixed(10), Fixed(10))
	if len(vRows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(vRows))
	}
	if vRows[0].Y != 0 || vRows[1].Y != 12 || vRows[2].Y != 24 {
		t.Errorf("unexpected Y coords with gap: %d, %d, %d", vRows[0].Y, vRows[1].Y, vRows[2].Y)
	}

	// Horizontal split with gap: 2 items of Fixed(40) with gap 5
	// X coords: 0, 45.
	hCols := HBoxWithGap(area, 5, Fixed(40), Fixed(40))
	if len(hCols) != 2 {
		t.Fatalf("expected 2 cols, got %d", len(hCols))
	}
	if hCols[0].X != 0 || hCols[1].X != 45 {
		t.Errorf("unexpected X coords with gap: %d, %d", hCols[0].X, hCols[1].X)
	}
}
