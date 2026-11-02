package main

import "time"

// A terminal shows its scrollback by itself (the wheel, Shift+PgUp) and
// tells no program. The overlay reads the terminal's window to tell how far
// it has scrolled and moves the pictures with the lines (Overlay.Anchor and
// Follow); termimg says when: once its own output has settled, and that a
// typed key has brought the terminal back to the bottom.
// shortcut: output that arrives while the scrollback is shown hides the
// pictures until a key is typed, as where the lines then are is not known.

const (
	scrollWatch = 200 * time.Millisecond // how often the window is read
	settle      = 300 * time.Millisecond // output this recent may not be drawn yet
)

type scrollback struct {
	wrote    time.Time // the last output, or resize
	anchored bool      // the window was read after it
}

// watchScroll follows the terminal's scrollback until the session closes.
func (s *session) watchScroll() {
	t := time.NewTicker(scrollWatch)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		closed := s.ov == nil
		if !closed {
			s.checkScroll()
		}
		s.mu.Unlock()
		if closed {
			return
		}
	}
}

// changed notes output or a resize: the window changes by termimg's doing,
// and is read again once that has settled.
func (s *session) changed() {
	s.scroll.wrote, s.scroll.anchored = time.Now(), false
	if s.ov != nil && s.ov.Moved() {
		_ = s.ov.Hide()
	}
}

// typed notes input from the user: the terminal is at the bottom again.
func (s *session) typed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ov != nil && s.ov.Moved() {
		_ = s.ov.Reset()
	}
	s.scroll.wrote, s.scroll.anchored = time.Now(), false
}

func (s *session) checkScroll() {
	if len(s.shown) == 0 || time.Since(s.scroll.wrote) < settle {
		return
	}
	if !s.scroll.anchored {
		s.scroll.anchored = s.ov.Anchor(s.emu.CursorPosition().Y)
		return
	}
	_ = s.ov.Follow()
}
