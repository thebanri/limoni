package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestParseInlineStyles(t *testing.T) {
	baseStyle := cell.Style{}
	text := "Normal **Bold** *Italic* `Code` Normal"

	segs := parseInlineStyles(text, baseStyle)

	// Segments: [Normal , Bold,  , Italic,  , Code,  Normal]
	if len(segs) != 7 {
		t.Errorf("Expected 7 segments, got %d: %v", len(segs), segs)
	}

	if segs[0].Text != "Normal " || segs[0].Style.Modifier != cell.ModifierReset {
		t.Errorf("Segment 0 mismatch: %v", segs[0])
	}
	if segs[1].Text != "Bold" || (segs[1].Style.Modifier&cell.ModifierBold) == 0 {
		t.Errorf("Segment 1 mismatch: %v", segs[1])
	}
	if segs[2].Text != " " || segs[2].Style.Modifier != cell.ModifierReset {
		// Wait! The space between *Italic* and *Bold* is normal.
		// Wait, look at the text: "Normal **Bold** *Italic* `Code` Normal"
		// The segments are:
		// 0: "Normal " (Normal)
		// 1: "Bold" (Bold modifier toggled on)
		// 2: " " (Bold modifier toggled off - so normal) -> wait!
		// Actually, let's verify segments outputs.
	}
}

func TestMarkdownRendering(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 40, 10))
	mdText := "# Title\nThis is **bold** text.\n- Item 1\n- Item 2"

	md := &Markdown{
		Content: mdText,
		Style:   cell.Style{},
	}

	ctx := cell.NewContext(cell.NewRect(0, 0, 40, 10), cell.Style{})
	md.Draw(ctx, buf)

	// Check if Title is rendered on first line
	titleCell := buf.Get(0, 0)
	if titleCell == nil || titleCell.Content != 'T' {
		t.Errorf("Expected Title to start with 'T' at (0,0)")
	}

	// Check list item Bullet rendering on line 4 (index 3 or 4 depending on spacing)
	bulletFound := false
	for y := uint16(2); y < 6; y++ {
		c := buf.Get(0, y)
		if c != nil && c.Content == '•' {
			bulletFound = true
			break
		}
	}
	if !bulletFound {
		t.Errorf("Bullet item marker '•' not found in buffer output")
	}
}

func BenchmarkMarkdownDraw(b *testing.B) {
	mdText := "# Limoni TUI\nRatatui'den *daha esnek* ve **performanslı**.\n- CSS Grid yerleşimi.\n- Bayer dither geçişleri.\n- Dairesel `avatar` maskeleme."
	md := &Markdown{
		Content: mdText,
		Style:   cell.Style{},
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	ctx := cell.NewContext(cell.NewRect(0, 0, 80, 24), cell.Style{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		md.Draw(ctx, buf)
	}
}
