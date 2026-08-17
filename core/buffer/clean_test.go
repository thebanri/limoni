package buffer

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// TestClearFastPathClearsWrittenCells, hızlı yolun yazılmış hücreleri atlamadığını doğrular.
func TestClearFastPathClearsWrittenCells(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 10, 3))
	var style cell.Style
	style.Reset()
	buf.SetString(0, 1, "limoni", style)

	buf.Clear()
	for i, c := range buf.Content {
		if c.Content != ' ' {
			t.Fatalf("cell %d was not cleared: %q", i, c.Content)
		}
	}
}

// TestClearFastPathSkipsCleanBuffer, temiz tamponda Clear'ın IsDirty bayrağını
// yeniden tetiklemediğini doğrular.
func TestClearFastPathSkipsCleanBuffer(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 10, 3))
	buf.IsDirty = false
	buf.Clear()
	if buf.IsDirty {
		t.Fatal("Clear on a clean buffer must not mark the buffer dirty")
	}
}

// TestInvalidateForcesFullClear, Content dilimine doğrudan yazıldıktan sonra
// Invalidate çağrısının hızlı yolu devre dışı bıraktığını doğrular.
func TestInvalidateForcesFullClear(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 4, 1))
	buf.Invalidate()
	buf.Content[2].Content = 'X'

	buf.Clear()
	if buf.Content[2].Content != ' ' {
		t.Fatalf("directly written cell was not cleared: %q", buf.Content[2].Content)
	}
	if !buf.IsDirty {
		t.Fatal("clearing a modified buffer must mark it dirty")
	}
}
