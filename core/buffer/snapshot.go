package buffer

import (
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// Snapshot returns a deterministic plain-text representation of the buffer.
// Continuation cells are rendered as spaces so wide runes remain readable.
//
// A row is not one rune per column: a cell that holds a grapheme cluster writes
// all of the cluster's runes, so a row is at least Area.Width runes long.
// testkit.DiffSnapshot measures the columns and reads a row this way.
func (b *Buffer) Snapshot() string {
	if b == nil || b.Area.Width == 0 || b.Area.Height == 0 {
		return ""
	}
	var out strings.Builder
	for y := uint16(0); y < b.Area.Height; y++ {
		if y > 0 {
			out.WriteByte('\n')
		}
		for x := uint16(0); x < b.Area.Width; x++ {
			r := b.CellAt(x, y).Content
			if r == cell.RuneContinuation || r == 0 {
				r = ' '
			}
			if cell.IsCluster(r) {
				out.WriteString(cell.ClusterText(r))
				continue
			}
			out.WriteRune(r)
		}
	}
	return out.String()
}
