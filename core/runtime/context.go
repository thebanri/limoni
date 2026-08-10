package runtime

import "context"

// Context is passed to runtime helpers and exposes the program lifecycle
// without exposing its internal queues.
type Context struct {
	context.Context
	program *Program
}

// Send injects a message into the owning program.
func (c Context) Send(msg Msg) error {
	if c.program == nil {
		return context.Canceled
	}
	return c.program.Send(c, msg)
}

// RequestRedraw coalesces a redraw request for the owning program.
func (c Context) RequestRedraw() {
	if c.program != nil {
		c.program.RequestRedraw()
	}
}
