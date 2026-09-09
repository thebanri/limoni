package terminal

import (
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// Aliases keep the terminal API concise while the event contract lives in the backend layer.
type EventPhase = driver.EventPhase
type EventContext = driver.EventContext

const (
	CapturePhase = driver.CapturePhase
	TargetPhase  = driver.TargetPhase
	BubblePhase  = driver.BubblePhase
)

type eventRegion struct {
	Area     cell.Rect
	ID       string
	LayerID  string
	ZIndex   int
	Disabled bool
	Phase    driver.EventPhase
	Handler  func(*driver.EventContext)
	OnEnter  func(*driver.EventContext)
	OnLeave  func(*driver.EventContext)
}

// EventRegion describes an event target independent of its visual widget.
type EventRegion struct {
	Area     cell.Rect
	ID       string
	LayerID  string
	ZIndex   int
	Disabled bool
	Phase    EventPhase
	Handler  func(*EventContext)
	OnEnter  func(*EventContext)
	OnLeave  func(*EventContext)
}
