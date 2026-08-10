package backend

import "time"

// EventPhase controls the order in which an event reaches registered handlers.
type EventPhase uint8

const (
	CapturePhase EventPhase = iota
	TargetPhase
	BubblePhase
)

// PointerEventKind identifies pointer movement lifecycle events.
type PointerEventKind uint8

const (
	PointerMove PointerEventKind = iota
	PointerEnter
	PointerLeave
)

// EventContext is the mutable context passed to propagation handlers.
type EventContext struct {
	Mouse            MouseEvent
	Phase            EventPhase
	RegionID         string
	LayerID          string
	ZIndex           int
	PointerKind      PointerEventKind
	ClickCount       int
	EventTime        time.Time
	stopped          bool
	defaultPrevented bool
}

func (e *EventContext) StopPropagation()          { e.stopped = true }
func (e *EventContext) PreventDefault()           { e.defaultPrevented = true }
func (e EventContext) IsPropagationStopped() bool { return e.stopped }
func (e EventContext) IsDefaultPrevented() bool   { return e.defaultPrevented }
