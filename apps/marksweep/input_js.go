//go:build js

package main

import "syscall/js"

// In the browser the page locks the pointer to the terminal while a run is
// on, so the mouse turns the view as in any first-person game and never
// leaves the window. The page reports movement and buttons here:
//
//	window.__marksweep_pointer('move', dx, dy)
//	window.__marksweep_pointer('button', button, down)
//	window.__marksweep_pointer('lock', locked)
//
// and reads window.__marksweep_wantLock to know whether a click should lock
// it. The calls arrive between frames; drain hands them to the game.
type pointer struct {
	dx      float64
	locked  bool
	changed bool
	buttons []pointerEvent
	wanted  bool
	init    bool
}

type pointerEvent struct {
	button int
	down   bool
}

func newPointer() *pointer {
	p := &pointer{buttons: make([]pointerEvent, 0, 16)}
	js.Global().Set("__marksweep_pointer", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		switch args[0].String() {
		case "move":
			if len(args) >= 2 {
				p.dx += args[1].Float()
			}
		case "button":
			if len(args) >= 3 && len(p.buttons) < cap(p.buttons) {
				p.buttons = append(p.buttons, pointerEvent{args[1].Int(), args[2].Bool()})
			}
		case "lock":
			if len(args) >= 2 {
				p.locked = args[1].Bool()
				p.changed = true
			}
		}
		return nil
	}))
	return p
}

func (p *pointer) drain(g *game) {
	if p.changed {
		p.changed = false
		g.locked = p.locked
		if !p.locked {
			g.held[actFire] = false
			if g.phase == phPlay {
				g.pauseRun() // Esc let the pointer go
			}
		}
	}
	g.lookDX += p.dx
	p.dx = 0
	for _, b := range p.buttons {
		g.pointerButton(b.button, b.down)
	}
	p.buttons = p.buttons[:0]
}

// want tells the page whether a click should lock the pointer: during a
// run, yes; on a card, a menu or the end of a run, the pointer is let go so
// the cards can be clicked.
func (p *pointer) want(w bool) {
	if p.init && w == p.wanted {
		return
	}
	p.init, p.wanted = true, w
	js.Global().Set("__marksweep_wantLock", w)
	switch {
	case !w && p.locked:
		js.Global().Get("document").Call("exitPointerLock")
	case w && !p.locked:
		// Right after the click or key that started the run, the browser
		// still counts it as the player's gesture and grants the lock.
		if req := js.Global().Get("__marksweep_requestLock"); req.Type() == js.TypeFunction {
			req.Invoke()
		}
	}
}
