package engine

import (
	"context"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
)

// Observer sees what the runtime does, in the order it does it. It is the
// attachment point for session recording; the runtime itself knows nothing
// about files, formats or redaction.
//
// All three methods are called with the model lock held, from the goroutine
// running the model, so they observe a consistent sequence: step counts are
// dense and increasing, and a frame reported at step N was rendered from the
// model exactly as it stood after message N. Implementations must be quick and
// must not call back into the Program.
type Observer interface {
	// Message reports the message about to be passed to Update, as step N.
	Message(step uint64, msg Msg)
	// Frame reports the semantic tree a View produced after step N messages.
	Frame(step uint64, tree []accessibility.Node)
	// Panic reports that Update or Init panicked while handling step N.
	Panic(step uint64, value any)
}

// WithObserver attaches an observer to the program.
func WithObserver(observer Observer) Option {
	return func(opts *programOptions) { opts.observer = observer }
}

// TimeMsg carries the wall clock into a model.
//
// Reading time.Now inside Update makes a model impossible to replay: the
// replayed Update reads a different clock. Asking for the time with NowCmd
// instead routes it through a message, and messages are what a recording
// captures, so the replay sees exactly the time the original run saw.
type TimeMsg struct {
	Time time.Time
}

// NowCmd returns a command that delivers the current time as a TimeMsg.
func NowCmd() Cmd {
	return func(context.Context) Msg { return TimeMsg{Time: time.Now()} }
}
