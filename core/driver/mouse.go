package driver

import (
	"strings"
	"sync/atomic"
)

// Mouse reporting: any-motion tracking and SGR coordinates.
const (
	mouseOnSeq  = "\x1b[?1003h\x1b[?1006h"
	mouseOffSeq = "\x1b[?1006l\x1b[?1003l"
)

// mouseCapture is whether the application takes the mouse — on unless it
// said otherwise — and whether the setup sequence has gone out, after which
// a change is written at once.
type mouseCapture struct {
	off    atomic.Bool
	active atomic.Bool
}

func (m *mouseCapture) enabled() bool { return !m.off.Load() }

// setupSequence is the setup for a full-screen or inline frame, with mouse
// reporting only if the application takes the mouse.
func setupSequence(inline uint16, mouse bool) string {
	seq := fullScreenSetupCmds()
	if inline > 0 {
		seq = inlineSetupCmds(inline)
	}
	if !mouse {
		seq = strings.Replace(seq, mouseOnSeq, "", 1)
	}
	return seq
}

// SetMouse says whether the application takes the mouse: clicks, the wheel
// and pointer movement reported to it. It is on by default. Off, the
// terminal keeps the mouse for itself, so text can be selected without
// holding Shift, and most terminals turn the wheel into arrow keys on the
// alternate screen.
//
// Called before Setup it decides what Setup sends; afterwards the change is
// written to the terminal at once. Either way it holds through a Suspend or
// a Release.
func (b *Backend) SetMouse(enabled bool) error {
	if b.mouse.off.Swap(!enabled) == !enabled {
		return nil // no change
	}
	if !b.mouse.active.Load() {
		return nil
	}
	seq := mouseOffSeq
	if enabled {
		seq = mouseOnSeq
	}
	_, err := b.Write([]byte(seq))
	return err
}

// MouseEnabled reports whether the application takes the mouse.
func (b *Backend) MouseEnabled() bool { return b.mouse.enabled() }
