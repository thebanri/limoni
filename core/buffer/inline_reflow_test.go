package buffer

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestInlineExtentsAndVisualRows(t *testing.T) {
	b := NewBuffer(cell.NewRect(0, 0, 10, 4))
	set := func(x, y int, r rune) { b.Content[y*10+x].Content = r }
	for x := range 10 {
		set(x, 0, 'a')
	}
	set(3, 1, 'b')
	set(8, 3, '世')
	set(9, 3, cell.RuneContinuation)
	ext := InlineExtents(nil, b, 4)
	want := []uint16{10, 4, 0, 10}
	for i := range want {
		if ext[i] != want[i] {
			t.Fatalf("extents %v, want %v", ext, want)
		}
	}
	for _, c := range []struct {
		w    uint16
		rows int
	}{{10, 4}, {5, 2 + 1 + 1 + 2}, {4, 3 + 1 + 1 + 3}, {20, 4}} {
		if got := InlineVisualRows(ext, c.w); got != c.rows {
			t.Errorf("width %d: %d rows, want %d", c.w, got, c.rows)
		}
	}
	if got := string(InlineUp(nil, 5)); got != "\x1b[4A\r" {
		t.Errorf("InlineUp = %q", got)
	}
	if got := string(InlineUp(nil, 1)); got != "\r" {
		t.Errorf("InlineUp(1) = %q", got)
	}
}
