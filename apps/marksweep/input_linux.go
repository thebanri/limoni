//go:build linux

package main

import (
	"errors"

	"github.com/thebanri/limoni"
)

// On Wayland the game's own window holds the mouse and the keyboard while
// a run or its cards are up (wayland_linux.go): the pointer is locked and
// hidden, its movement turns the view, and the keys come from the
// compositor rather than the terminal. Elsewhere, or if the compositor
// cannot, the terminal's mouse reports turn the view (game.mouse).
type pointer struct {
	wl    *wlClient
	why   string // a line for the title screen
	shown bool
	away  bool // the window lost the keyboard during the cards: not again until they go
}

const (
	btnLeft  = 0x110
	btnRight = 0x111
)

func newPointer() *pointer {
	c, err := dialWayland()
	switch {
	case errors.Is(err, errNoWayland):
		return &pointer{why: kittyHint()}
	case err != nil:
		return &pointer{why: "the compositor cannot lock the mouse: " + err.Error()}
	}
	return &pointer{wl: c}
}

func (p *pointer) drain(g *game) {
	if p.wl == nil {
		g.locked, g.lockHint = false, false
		return
	}
	g.locked = p.shown && p.wl.locked.Load()
	g.lockHint = p.shown && !g.locked
	dx, _ := p.wl.motion()
	switch {
	case g.locked && g.phase == phCards:
		g.nudgeCard(dx)
	case g.locked:
		g.lookDX += dx
	}
	for {
		select {
		case e := <-p.wl.events:
			if p.shown {
				p.event(g, e)
			}
		default:
			return
		}
	}
}

func (p *pointer) event(g *game, e wlEvent) {
	switch e.kind {
	case wlEnter:
		g.held = [nAct]bool{} // the keys held now follow, as presses
	case wlLeave:
		g.held = [nAct]bool{}
		if g.phase == phPlay {
			g.pauseRun()
		} else {
			p.away = true
		}
	case wlKey:
		if k, ok := evdevKey(e.code); ok {
			k.Release = !e.down
			g.key(k)
		}
	case wlButton:
		switch e.code {
		case btnLeft:
			g.pointerButton(0, e.down)
		case btnRight:
			g.pointerButton(2, e.down)
		}
	}
}

// want opens the window during a run and its cards, and closes it
// otherwise, which gives the mouse and the keyboard back to the terminal.
func (p *pointer) want(g *game) {
	if g.phase != phCards {
		p.away = false
	}
	w := p.wl != nil && (g.phase == phPlay || g.phase == phCards) && !p.away
	g.grab = w
	if w == p.shown {
		return
	}
	p.shown = w
	if w {
		p.wl.show()
	} else {
		p.wl.hide()
	}
}

func (p *pointer) close() {
	if p.wl != nil {
		p.wl.hide()
		p.wl.conn.Close()
	}
}

func (p *pointer) problem() string { return p.why }

// evdevKey is the key the terminal would have sent for an evdev key code:
// the keys are where they sit on the keyboard, whatever its layout.
func evdevKey(code uint32) (limoni.KeyEvent, bool) {
	switch code {
	case 1:
		return limoni.KeyEvent{Type: limoni.KeyEsc}, true
	case 28, 96: // Enter, and the keypad's
		return limoni.KeyEvent{Type: limoni.KeyEnter}, true
	case 57:
		return limoni.KeyEvent{Type: limoni.KeySpace}, true
	case 103:
		return limoni.KeyEvent{Type: limoni.KeyUp}, true
	case 105:
		return limoni.KeyEvent{Type: limoni.KeyLeft}, true
	case 106:
		return limoni.KeyEvent{Type: limoni.KeyRight}, true
	case 108:
		return limoni.KeyEvent{Type: limoni.KeyDown}, true
	}
	if r, ok := evdevRunes[code]; ok {
		return limoni.KeyEvent{Type: limoni.KeyRune, Ch: r}, true
	}
	return limoni.KeyEvent{}, false
}

var evdevRunes = map[uint32]rune{
	2: '1', 3: '2', 4: '3',
	16: 'q', 17: 'w', 18: 'e', 19: 'r', 25: 'p',
	30: 'a', 31: 's', 32: 'd', 33: 'f', 50: 'm',
}
