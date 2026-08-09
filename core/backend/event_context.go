package backend

// EventPhase controls the order in which an event reaches registered handlers.
type EventPhase uint8

const (
	CapturePhase EventPhase = iota
	TargetPhase
	BubblePhase
)

// EventContext is the mutable context passed to propagation handlers.
type EventContext struct {
	Mouse            MouseEvent
	Phase            EventPhase
	stopped          bool
	defaultPrevented bool
}

func (e *EventContext) StopPropagation()          { e.stopped = true }
func (e *EventContext) PreventDefault()           { e.defaultPrevented = true }
func (e EventContext) IsPropagationStopped() bool { return e.stopped }
func (e EventContext) IsDefaultPrevented() bool   { return e.defaultPrevented }
