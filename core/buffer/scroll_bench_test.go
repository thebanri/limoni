package buffer

import (
	"testing"
	"unsafe"

	"github.com/thebanri/limoni/core/cell"
)

// The diff with every encoding on, as a real Terminal runs it. The scroll
// search must not tax frames that did not scroll: it once hashed every row
// of both buffers on every frame, and a one-cell update went from 9.8 µs to
// 26.9 µs in the cross-framework runner.
var scrollOpts = DiffOptions{TrueColor: true, Colors256: true, EraseChar: true, RepeatChar: true, ScrollRegions: true, InsertDelete: true}

func scrollBenchBuffers() (*Buffer, *Buffer, []byte) {
	area := cell.NewRect(0, 0, 120, 40)
	front, back := NewBuffer(area), NewBuffer(area)
	style := cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}
	for y := uint16(0); y < 40; y++ {
		for x := uint16(0); x < 120; x++ {
			front.SetCell(x, y, cell.Cell{Content: rune('a' + (int(x)+int(y))%26), Style: style})
		}
	}
	out, _ := DiffWithOptions(front, back, make([]byte, 0, 1<<16), scrollOpts)
	return front, back, out
}

// Nothing changed, but the frame was drawn: the buffer is dirty, as it is
// after every Draw, and the diff has to look.
func BenchmarkDiff_ScrollRegions_NoChanges(b *testing.B) {
	front, back, out := scrollBenchBuffers()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		front.Invalidate()
		out, _ = DiffWithOptions(front, back, out[:0], scrollOpts)
	}
}

func BenchmarkDiff_ScrollRegions_OneCell(b *testing.B) {
	front, back, out := scrollBenchBuffers()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// The character changes on every pass over the row, so every frame has
		// one changed cell; repeating a pass would write what is already there.
		front.SetCell(uint16(i%120), 5, cell.Cell{Content: rune('0' + (i+i/120)%10)})
		out, _ = DiffWithOptions(front, back, out[:0], scrollOpts)
	}
}

func BenchmarkDiff_ScrollRegions_FullChanges(b *testing.B) {
	front, back, out := scrollBenchBuffers()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for y := uint16(0); y < 40; y++ {
			for x := uint16(0); x < 120; x++ {
				front.SetCell(x, y, cell.Cell{Content: rune('a' + (int(x)+int(y)+i)%26)})
			}
		}
		out, _ = DiffWithOptions(front, back, out[:0], scrollOpts)
	}
}

// A log gaining a line: every row moves up one, a new one at the bottom.
func BenchmarkDiff_ScrollRegions_LogScroll(b *testing.B) {
	front, back, out := scrollBenchBuffers()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(front.Content, front.Content[120:])
		for x := uint16(0); x < 120; x++ {
			front.SetCell(x, 39, cell.Cell{Content: rune('A' + (int(x)+i)%26)})
		}
		out, _ = DiffWithOptions(front, back, out[:0], scrollOpts)
	}
}

// The scroll search hashes and compares rows as raw memory, which is only
// the same as comparing their cells if a Cell has no padding: bytes the
// fields do not cover could differ between equal cells.
func TestCellHasNoPadding(t *testing.T) {
	var c cell.Cell
	fields := unsafe.Sizeof(c.Content) + unsafe.Sizeof(c.Style.Fg) + unsafe.Sizeof(c.Style.Bg) +
		unsafe.Sizeof(c.Style.Modifier) + unsafe.Sizeof(c.Style.Link)
	if unsafe.Sizeof(c) != fields {
		t.Fatalf("Cell is %d bytes but its fields are %d: rowHash and rowsEqual would see padding", unsafe.Sizeof(c), fields)
	}
}
