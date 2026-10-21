package buffer

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func snapRows(b *Buffer) []string {
	rows := strings.Split(b.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return rows
}

func TestClipLimitsWrites(t *testing.T) {
	b := NewBuffer(cell.NewRect(0, 0, 10, 3))
	b.SetClip(cell.NewRect(2, 1, 5, 1))

	b.SetString(0, 1, "abcdefghij", cell.Style{}) // starts left of the clip, ends right of it
	b.SetString(0, 0, "outside", cell.Style{})
	b.SetCell(8, 1, cell.Cell{Content: 'X'})
	if c := b.Get(1, 1); c != nil {
		t.Fatal("Get outside the clip must return nil: its pointer is for writing")
	}
	if got := snapRows(b); got[0] != "" || got[1] != "  cdefg" || got[2] != "" {
		t.Fatalf("rows = %q, want only columns 2-6 of row 1 written", got)
	}
	if n := b.SetString(0, 1, "abcdefghij", cell.Style{}); n != 7 {
		t.Fatalf("SetString returned %d, want 7: the columns up to the clip's right edge", n)
	}

	// A wide character that would cross the clip's right edge is left out.
	b.SetString(5, 1, "日本", cell.Style{})
	if got := snapRows(b)[1]; got != "  cde日" { // 日 at 5-6; 本 would need 7-8
		t.Fatalf("row 1 = %q", got)
	}
	b.SetCell(6, 1, cell.Cell{Content: '本'})
	if b.CellAt(6, 1).Content != cell.RuneContinuation {
		t.Fatal("SetCell drew a wide character across the clip's right edge")
	}

	b.ClearClip()
	b.SetString(0, 0, "whole", cell.Style{})
	if got := snapRows(b)[0]; got != "whole" {
		t.Fatalf("after ClearClip row 0 = %q", got)
	}
}

func TestClipIsCutToTheBufferAndClearedByResize(t *testing.T) {
	b := NewBuffer(cell.NewRect(0, 0, 4, 2))
	b.SetClip(cell.NewRect(2, 0, 50, 50))
	if got := b.Clip(); got != cell.NewRect(2, 0, 2, 2) {
		t.Fatalf("Clip() = %+v, want it cut to the buffer", got)
	}
	b.Resize(cell.NewRect(0, 0, 6, 2))
	if got := b.Clip(); got != cell.NewRect(0, 0, 6, 2) {
		t.Fatalf("after Resize Clip() = %+v, want the whole buffer", got)
	}
}
