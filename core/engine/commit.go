package engine

import (
	"sync"

	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// commits are the widgets UpdateResult.Commit asked for, waiting for a
// frame. They are queued by the goroutine that runs Update, so they keep
// the order the Updates ran in — a Cmd would not: each runs on its own
// goroutine. They are never dropped: a lost commit is a hole in the user's
// scrollback.
type commits struct {
	mu      sync.Mutex
	pending []widgets.Widget
	drawing []widgets.Widget // owned by the drawing goroutine, reused
}

func (p *Program) queueCommits(ws []widgets.Widget) {
	if len(ws) == 0 {
		return
	}
	p.commits.mu.Lock()
	p.commits.pending = append(p.commits.pending, ws...)
	p.commits.mu.Unlock()
}

// commitPending hands the waiting commits to the frame being drawn.
func (p *Program) commitPending(frame *terminal.Frame) {
	p.commits.mu.Lock()
	p.commits.drawing = append(p.commits.drawing[:0], p.commits.pending...)
	clear(p.commits.pending)
	p.commits.pending = p.commits.pending[:0]
	p.commits.mu.Unlock()
	for i, w := range p.commits.drawing {
		_ = frame.Commit(0, w)
		p.commits.drawing[i] = nil
	}
}
