package buffer

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// Buffer represents the cellular grid of the terminal screen.
// Using a 1D flat slice (`[]cell.Cell`) instead of a 2D slice minimizes memory fragmentation and cache misses.
type Buffer struct {
	Area       cell.Rect             // Bounding box dimensions of the buffer
	Content    []cell.Cell           // Contiguous slice of cells in memory
	IsDirty    bool                  // Indicates whether the buffer has modified content
	StyleCache map[cell.Style][]byte // Cache for ANSI style transition escape sequences

	// clean indicates whether all cells are in the default (space + zero style) state.
	// This flag lets Clear() avoid scanning the entire cell array during empty frame passes.
	clean bool
}

// NewBuffer creates a new Buffer with the specified dimensions.
func NewBuffer(area cell.Rect) *Buffer {
	needed := int(area.Width) * int(area.Height)
	content := make([]cell.Cell, needed)
	b := &Buffer{
		Area:       area,
		Content:    content,
		IsDirty:    true,
		StyleCache: make(map[cell.Style][]byte),
	}
	b.Clear()
	return b
}

// NewEmptyBuffer creates an empty (0x0) Buffer.
func NewEmptyBuffer() *Buffer {
	return NewBuffer(cell.Rect{})
}

// Clear resets all cells in the buffer to the default state (space character, default style).
func (b *Buffer) Clear() {
	if b.clean {
		return
	}
	for i := range b.Content {
		if b.Content[i].Content != ' ' || b.Content[i].Style != (cell.Style{}) {
			b.Content[i].Reset()
			b.IsDirty = true
		}
	}
	b.clean = true
}

// Invalidate marks the buffer content as modified externally.
// Callers directly manipulating the Content slice should call Invalidate to ensure proper buffer diffing.
func (b *Buffer) Invalidate() {
	b.clean = false
	b.IsDirty = true
}

// Resize resizes the buffer area.
// Zero-Allocation Optimization: If the existing slice capacity is sufficient,
// re-slices without allocating new heap memory.
func (b *Buffer) Resize(area cell.Rect) {
	b.Area = area
	needed := int(area.Width) * int(area.Height)
	if cap(b.Content) >= needed {
		b.Content = b.Content[:needed]
	} else {
		b.Content = make([]cell.Cell, needed)
	}
	b.IsDirty = true
	b.clean = false
	b.Clear()
}

// Get returns a direct pointer to the cell at the specified coordinates.
// Returns nil if coordinates are out of bounds.
func (b *Buffer) Get(x, y uint16) *cell.Cell {
	if x >= b.Area.Width || y >= b.Area.Height {
		return nil
	}
	b.clean = false
	return &b.Content[y*b.Area.Width+x]
}

// SetCell writes a cell at the specified coordinate.
func (b *Buffer) SetCell(x, y uint16, c cell.Cell) {
	if x >= b.Area.Width || y >= b.Area.Height {
		return
	}
	idx := int(y)*int(b.Area.Width) + int(x)
	if b.Content[idx] != c {
		b.Content[idx] = c
		b.IsDirty = true
		b.clean = false
	}
}

// SetString writes a string starting at the specified coordinate with the given style.
func (b *Buffer) SetString(x, y uint16, s string, style cell.Style) {
	if y >= b.Area.Height || x >= b.Area.Width {
		return
	}

	currX := x
	input := s
	for len(input) > 0 && currX < b.Area.Width {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError {
			break
		}

		w := cell.RuneWidth(r)
		if w == 0 {
			input = input[size:]
			continue // Skip zero-width combining characters
		}
		if currX+uint16(w) > b.Area.Width {
			break // Prevent clipping overflow
		}

		idx := y*b.Area.Width + currX
		if b.Content[idx].Content != r || b.Content[idx].Style != style {
			b.Content[idx].Content = r
			b.Content[idx].Style = style
			b.IsDirty = true
			b.clean = false
		}

		if w == 2 {
			if b.Content[idx+1].Content != cell.RuneContinuation {
				b.Content[idx+1].Content = cell.RuneContinuation
				b.IsDirty = true
				b.clean = false
			}
		}

		currX += uint16(w)
		input = input[size:]
	}
}

// index maps 2D coordinates to the 1D flat slice index. Returns -1 if out of bounds.
func (b *Buffer) index(x, y uint16) int {
	if x >= b.Area.Width || y >= b.Area.Height {
		return -1
	}
	return int(y)*int(b.Area.Width) + int(x)
}
