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
