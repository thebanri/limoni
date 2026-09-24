package buffer

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// Copying a row of cells — a wide character, then its continuation — keeps
// the character. The continuation used to count as a narrow cell written
// over the character's right half, which blanked the character: every CJK
// character and emoji in Markdown, and wide glyphs in TextInput, drew as
// spaces.
func TestWritingContinuationKeepsTheWideCharacter(t *testing.T) {
	family := cell.ClusterContent("\U0001F468\u200D\U0001F469\u200D\U0001F467", 2)
	for _, wide := range []rune{'你', family} {
		for name, set := range map[string]func(*Buffer, uint16, cell.Cell){
			"SetCell":       func(b *Buffer, x uint16, c cell.Cell) { b.SetCell(x, 0, c) },
			"SetCellDirect": func(b *Buffer, x uint16, c cell.Cell) { b.SetCellDirect(x, 0, c) },
		} {
			b := NewBuffer(cell.NewRect(0, 0, 6, 1))
			set(b, 1, cell.Cell{Content: wide})
			set(b, 2, cell.Cell{Content: cell.RuneContinuation})
			set(b, 3, cell.Cell{Content: 'x'})
			if got := b.CellAt(1, 0).Content; got != wide {
				t.Errorf("%s %U: the wide character became %U", name, wide, got)
			}
			if got := b.CellAt(2, 0).Content; got != cell.RuneContinuation {
				t.Errorf("%s %U: its right half became %U", name, wide, got)
			}
		}
	}
	// A continuation with nothing wide to its left is a space.
	b := NewBuffer(cell.NewRect(0, 0, 4, 1))
	b.SetCell(0, 0, cell.Cell{Content: 'a'})
	b.SetCellDirect(1, 0, cell.Cell{Content: cell.RuneContinuation})
	if got := b.CellAt(1, 0).Content; got != ' ' {
		t.Errorf("orphan continuation is %U, want a space", got)
	}
}
