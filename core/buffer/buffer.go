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

// Get returns a direct mutable pointer to the cell at the specified coordinates.
// Calling Get marks the buffer as dirty and non-clean, ensuring subsequent diff passes
// do not skip mutations performed through the returned pointer.
// Returns nil if coordinates are out of bounds.
func (b *Buffer) Get(x, y uint16) *cell.Cell {
	if x >= b.Area.Width || y >= b.Area.Height {
		return nil
	}
	b.clean = false
	b.IsDirty = true
	return &b.Content[y*b.Area.Width+x]
}

// CellAt returns a copy of the cell at the specified coordinates without modifying the buffer's dirty state.
// If coordinates are out of bounds, it returns a zero Cell.
func (b *Buffer) CellAt(x, y uint16) cell.Cell {
	if x >= b.Area.Width || y >= b.Area.Height {
		return cell.Cell{}
	}
	return b.Content[y*b.Area.Width+x]
}

// clearOrphanWideAround cleans up broken/dangling wide characters or continuation cells
// around (x, y) when overwriting a cell.
func (b *Buffer) clearOrphanWideAround(x, y uint16, newWidth int) {
	idx := int(y)*int(b.Area.Width) + int(x)

	// 1. If cell at (x, y) is currently a continuation cell, the character to its left
	// was a 2-width character whose right half is being replaced. Clear that left character.
	if b.Content[idx].Content == cell.RuneContinuation && x > 0 {
		leftIdx := idx - 1
		b.Content[leftIdx].Content = ' '
		b.Content[leftIdx].Style = cell.Style{}
		b.IsDirty = true
		b.clean = false
	}

	// 2. If cell at (x, y) is currently a 2-width character, and we're writing a 1-width character,
	// the continuation cell to its right is now an orphan. Clear that right cell.
	if newWidth == 1 && cell.RuneWidth(b.Content[idx].Content) == 2 && x+1 < b.Area.Width {
		rightIdx := idx + 1
		b.Content[rightIdx].Content = ' '
		b.Content[rightIdx].Style = cell.Style{}
		b.IsDirty = true
		b.clean = false
	}

	// 3. If we are writing a 2-width character, it will occupy (x, y) and (x+1, y).
	// If (x+1, y) was previously a 2-width character, its right half at (x+2, y) is now an orphan. Clear it.
	if newWidth == 2 && x+2 < b.Area.Width {
		rightIdx := idx + 1
		if cell.RuneWidth(b.Content[rightIdx].Content) == 2 {
			b.Content[idx+2].Content = ' '
			b.Content[idx+2].Style = cell.Style{}
			b.IsDirty = true
			b.clean = false
		}
	}
}

// SetCell writes a cell at the specified coordinate.
// If the style's background is ColorDefault (unstyled), it preserves the cell's existing background color.
func (b *Buffer) SetCell(x, y uint16, c cell.Cell) {
	if x >= b.Area.Width || y >= b.Area.Height {
		return
	}
	w := cell.RuneWidth(c.Content)
	if w == 0 {
		return
	}
	if w == 2 && x+1 >= b.Area.Width {
		return
	}

	b.clearOrphanWideAround(x, y, w)

	idx := int(y)*int(b.Area.Width) + int(x)
	mergedStyle := b.Content[idx].Style.Merge(c.Style)
	mergedCell := cell.Cell{
		Content: c.Content,
		Style:   mergedStyle,
	}
	if b.Content[idx] != mergedCell {
		b.Content[idx] = mergedCell
		b.IsDirty = true
		b.clean = false
	}

	if w == 2 && x+1 < b.Area.Width {
		contIdx := idx + 1
		if b.Content[contIdx].Content != cell.RuneContinuation || b.Content[contIdx].Style != mergedStyle {
			b.Content[contIdx].Content = cell.RuneContinuation
			b.Content[contIdx].Style = mergedStyle
			b.IsDirty = true
			b.clean = false
		}
	}
}

// SetCellDirect writes a cell at the specified coordinate without style merging (exact overwrite).
func (b *Buffer) SetCellDirect(x, y uint16, c cell.Cell) {
	if x >= b.Area.Width || y >= b.Area.Height {
		return
	}
	w := cell.RuneWidth(c.Content)
	if w == 0 {
		return
	}
	if w == 2 && x+1 >= b.Area.Width {
		return
	}

	b.clearOrphanWideAround(x, y, w)

	idx := int(y)*int(b.Area.Width) + int(x)
	if b.Content[idx] != c {
		b.Content[idx] = c
		b.IsDirty = true
		b.clean = false
	}

	if w == 2 && x+1 < b.Area.Width {
		contIdx := idx + 1
		if b.Content[contIdx].Content != cell.RuneContinuation || b.Content[contIdx].Style != c.Style {
			b.Content[contIdx].Content = cell.RuneContinuation
			b.Content[contIdx].Style = c.Style
			b.IsDirty = true
			b.clean = false
		}
	}
}

// SetStringWithin writes a string starting at the specified coordinate with the given style,
// strictly clipping text within maxWidth columns and buffer boundaries.
// It returns the number of columns actually written.
func (b *Buffer) SetStringWithin(x, y uint16, s string, style cell.Style, maxWidth uint16) uint16 {
	if y >= b.Area.Height || x >= b.Area.Width || maxWidth == 0 {
		return 0
	}

	limitX := x + maxWidth
	if limitX > b.Area.Width {
		limitX = b.Area.Width
	}

	currX := x
	input := s
	for len(input) > 0 && currX < limitX {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError {
			break
		}

		w := cell.RuneWidth(r)
		if w == 0 {
			input = input[size:]
			continue // Skip zero-width combining characters
		}
		if currX+uint16(w) > limitX {
			break // Prevent clipping overflow beyond maxWidth
		}

		b.clearOrphanWideAround(currX, y, w)

		idx := y*b.Area.Width + currX
		merged := b.Content[idx].Style.Merge(style)
		if b.Content[idx].Content != r || b.Content[idx].Style != merged {
			b.Content[idx].Content = r
			b.Content[idx].Style = merged
			b.IsDirty = true
			b.clean = false
		}

		if w == 2 {
			contIdx := idx + 1
			if b.Content[contIdx].Content != cell.RuneContinuation || b.Content[contIdx].Style != merged {
				b.Content[contIdx].Content = cell.RuneContinuation
				b.Content[contIdx].Style = merged
				b.IsDirty = true
				b.clean = false
			}
		}

		currX += uint16(w)
		input = input[size:]
	}
	return currX - x
}

// SetString writes a string starting at the specified coordinate with the given style.
// It returns the number of columns actually written.
func (b *Buffer) SetString(x, y uint16, s string, style cell.Style) uint16 {
	if x >= b.Area.Width {
		return 0
	}
	return b.SetStringWithin(x, y, s, style, b.Area.Width-x)
}

// index maps 2D coordinates to the 1D flat slice index. Returns -1 if out of bounds.
func (b *Buffer) index(x, y uint16) int {
	if x >= b.Area.Width || y >= b.Area.Height {
		return -1
	}
	return int(y)*int(b.Area.Width) + int(x)
}
