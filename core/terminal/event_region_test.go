package terminal

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestEventRegionMetadataAndDisabledFiltering(t *testing.T) {
	frame := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 20, 4)), NewFocusManager())
	seenID := ""
	frame.RegisterEventRegion(EventRegion{
		Area: cell.NewRect(0, 0, 10, 2), ID: "disabled", Disabled: true,
		Phase: TargetPhase, Handler: func(*EventContext) { t.Fatal("disabled region was called") },
	})
	frame.RegisterEventRegion(EventRegion{
		Area: cell.NewRect(0, 0, 10, 2), ID: "button", LayerID: "popup", ZIndex: 8,
		Phase: TargetPhase, Handler: func(ctx *EventContext) {
			seenID = ctx.RegionID
			if ctx.LayerID != "popup" || ctx.ZIndex != 8 {
				t.Fatalf("metadata = %q/%q/%d", ctx.RegionID, ctx.LayerID, ctx.ZIndex)
			}
		},
	})
	if !frame.DispatchEventRegions(backend.MouseEvent{X: 2, Y: 1, Button: backend.MouseLeft}) {
		t.Fatal("expected enabled region to handle event")
	}
	if seenID != "button" {
		t.Fatalf("region ID = %q, want button", seenID)
	}
}

func TestEventRegionEnterLeave(t *testing.T) {
	frame := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 20, 4)), NewFocusManager())
	transitions := make([]backend.PointerEventKind, 0, 2)
	frame.RegisterEventRegion(EventRegion{
		Area: cell.NewRect(1, 1, 4, 2), ID: "hover", Phase: TargetPhase,
		OnEnter: func(ctx *EventContext) { transitions = append(transitions, ctx.PointerKind) },
		OnLeave: func(ctx *EventContext) { transitions = append(transitions, ctx.PointerKind) },
		Handler: func(*EventContext) {},
	})
	frame.DispatchPointerMove(backend.MouseEvent{X: 2, Y: 2, Button: backend.MouseNone})
	frame.DispatchPointerMove(backend.MouseEvent{X: 10, Y: 2, Button: backend.MouseNone})
	if len(transitions) != 2 || transitions[0] != backend.PointerEnter || transitions[1] != backend.PointerLeave {
		t.Fatalf("transitions = %v, want enter/leave", transitions)
	}
	if frame.HoveredRegionID() != "" {
		t.Fatalf("hovered region = %q, want empty", frame.HoveredRegionID())
	}
}

func TestEventRegionDoubleClick(t *testing.T) {
	frame := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 20, 4)), NewFocusManager())
	counts := []int{}
	frame.RegisterEventRegion(EventRegion{
		Area: cell.NewRect(1, 1, 4, 2), ID: "button", Phase: TargetPhase,
		Handler: func(ctx *EventContext) { counts = append(counts, ctx.ClickCount) },
	})
	start := time.Unix(100, 0)
	frame.DispatchClick(backend.MouseEvent{X: 2, Y: 2, Button: backend.MouseLeft}, start)
	frame.DispatchClick(backend.MouseEvent{X: 2, Y: 2, Button: backend.MouseLeft}, start.Add(200*time.Millisecond))
	if len(counts) != 2 || counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("click counts = %v, want [1 2]", counts)
	}
}
