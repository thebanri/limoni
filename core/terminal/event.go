package terminal

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
)

// Aliases keep the terminal API concise while the event contract lives in the backend layer.
type EventPhase = backend.EventPhase
type EventContext = backend.EventContext

const (
	CapturePhase = backend.CapturePhase
	TargetPhase  = backend.TargetPhase
	BubblePhase  = backend.BubblePhase
)

type eventRegion struct {
	Area    cell.Rect
	Phase   backend.EventPhase
	Handler func(*backend.EventContext)
}
