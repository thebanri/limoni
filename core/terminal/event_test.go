package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestEventPropagationOrderAndStop(t *testing.T) {
	frame := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), NewFocusManager())
	terminal := &Terminal{frame: frame}
	order := make([]EventPhase, 0, 3)
	frame.RegisterEventHandler(cell.NewRect(0, 0, 10, 2), CapturePhase, func(*EventContext) { order = append(order, CapturePhase) })
	frame.RegisterEventHandler(cell.NewRect(1, 0, 3, 1), TargetPhase, func(ctx *EventContext) { order = append(order, TargetPhase); ctx.StopPropagation() })
	frame.RegisterEventHandler(cell.NewRect(0, 0, 10, 2), BubblePhase, func(*EventContext) { order = append(order, BubblePhase) })
	if !terminal.RouteMouseEvent(backend.MouseEvent{Button: backend.MouseLeft, X: 2, Y: 0}) {
		t.Fatal("event should be handled")
	}
	if len(order) != 2 || order[0] != CapturePhase || order[1] != TargetPhase {
		t.Fatalf("propagation order = %v; want capture,target", order)
	}
}
