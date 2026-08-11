package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

func TestRenderWidgetRecordsLayoutDiagnostics(t *testing.T) {
	f := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 20, 4)), NewFocusManager())
	f.RenderWidget(&widgets.Paragraph{Text: "intrinsic content", Wrap: false}, cell.NewRect(0, 0, 8, 2))
	if len(f.DebugRegions) != 1 {
		t.Fatalf("debug regions = %d, want 1", len(f.DebugRegions))
	}
	region := f.DebugRegions[0]
	if region.Allocated != cell.NewRect(0, 0, 8, 2) {
		t.Fatalf("allocated = %+v", region.Allocated)
	}
	if region.Measured.IdealWidth != 8 || region.Measured.IdealHeight != 1 {
		t.Fatalf("measured = %+v", region.Measured)
	}
	if region.Overflowed {
		t.Fatal("unexpected overflow for clipped-to-area paragraph")
	}
}
