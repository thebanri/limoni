package buffer

import (
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// Snapshot returns a deterministic plain-text representation of the buffer.
// Continuation cells are rendered as spaces so wide runes remain readable.
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
			r := b.Get(x, y).Content
			if r == cell.RuneContinuation || r == 0 {
				r = ' '
			}
			out.WriteRune(r)
		}
	}
	return out.String()
}
