package buffer

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestBufferSnapshot(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 4, 2))
	buf.SetString(0, 0, "🔍A", cell.Style{})
	buf.SetString(0, 1, "ok", cell.Style{})
	if got := buf.Snapshot(); got != "🔍 A \nok  " {
		t.Fatalf("snapshot = %q", got)
	}
}
