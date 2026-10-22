//go:build linux

package main

import (
	"sync/atomic"
)

// On Linux the desktop locks the mouse for the game (portal_linux.go): the
// pointer is hidden and held while a run or its cards are up, and its
// movement and buttons come here. Until the desktop has answered, or if it
// cannot, the terminal's mouse reports turn the view (game.mouse).
type pointer struct {
	capture atomic.Pointer[capture]
	why     atomic.Value // a line for the title screen
	enabled bool
}

type pointerEvent struct {
	button int
	down   bool
}

func newPointer() *pointer {
	p := &pointer{}
	p.why.Store("")
	go func() {
		c, err := startCapture()
		if err != nil {
			p.why.Store("the desktop did not lock the mouse: " + err.Error())
			return
		}
		p.capture.Store(c)
	}()
	return p
}

func (p *pointer) drain(g *game) {
	c := p.capture.Load()
	if c == nil {
		g.locked, g.lockHint = false, false
		return
	}
	g.locked = c.active.Load()
	g.lockHint = p.enabled && !g.locked
	dx, _ := c.ei.motion()
	switch {
	case g.locked && g.phase == phCards:
		g.nudgeCard(dx)
	case g.locked:
		g.lookDX += dx
	}
	for {
		select {
		case b := <-c.ei.press:
			if g.locked {
				g.pointerButton(b.button, b.down)
			}
		default:
			return
		}
	}
}

// want arms the capture during a run and its cards, and gives the mouse
// back otherwise.
func (p *pointer) want(g *game) {
	w := g.phase == phPlay || g.phase == phCards
	c := p.capture.Load()
	if c == nil || w == p.enabled {
		return
	}
	p.enabled = w
	c.enable(w) // a D-Bus round trip, on the frame: in order, and only when the phase changes
}

func (p *pointer) close() {
	if c := p.capture.Load(); c != nil {
		c.close()
	}
}

func (p *pointer) problem() string { return p.why.Load().(string) }
