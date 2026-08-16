package bubbletea

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestLipglossStyleBuilder(t *testing.T) {
	st := NewStyle().
		Foreground(cell.NewColorRGB(255, 0, 0)).
		Background(cell.NewColorRGB(0, 0, 255)).
		Bold(true).
		Italic(true).
		Padding(1, 2)

	rendered := st.Render("Hello Limoni")
	if !strings.HasPrefix(rendered, "\n  Hello Limoni  ") {
		t.Fatalf("unexpected rendered string: %q", rendered)
	}

	cellStyle := st.ToCellStyle()
	if cellStyle.Fg != cell.NewColorRGB(255, 0, 0) {
		t.Fatalf("unexpected Fg: %v", cellStyle.Fg)
	}
	if cellStyle.Bg != cell.NewColorRGB(0, 0, 255) {
		t.Fatalf("unexpected Bg: %v", cellStyle.Bg)
	}
	if cellStyle.Modifier&(cell.ModifierBold|cell.ModifierItalic) == 0 {
		t.Fatalf("expected modifiers, got %v", cellStyle.Modifier)
	}
}
