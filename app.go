package limoni

// Run starts an event-driven Limoni application loop.
// appFn is invoked with the active Frame and the triggering Event.
// Returning false from appFn cleanly exits the application.
// On the first invocation, ev is nil for the initial render.
// Ctrl+C automatically terminates the application.
func Run(appFn func(f *Frame, ev *Event) bool) error {
	term, err := New()
	if err != nil {
		return err
	}
	defer term.Close()

	running := true
	// Initial render
	err = term.Draw(func(f *Frame) {
		running = appFn(f, nil)
	})
	if err != nil || !running {
		return err
	}

	events := term.Events()
	for running {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			// Automatic graceful exit on Ctrl+C
			if ev.Type == EventKey && ev.Key.Ctrl && (ev.Key.Ch == 'c' || ev.Key.Ch == 'C') {
				return nil
			}
			// Route mouse event through terminal hit-test router
			if ev.Type == EventMouse {
				term.RouteMouseEvent(ev.Mouse)
			}
			err = term.Draw(func(f *Frame) {
				running = appFn(f, &ev)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Start displays a static or single-state frame and cleanly exits when 'q', 'ESC', or Ctrl+C is pressed.
func Start(drawFn func(f *Frame)) error {
	return Run(func(f *Frame, ev *Event) bool {
		drawFn(f)
		if ev != nil && ev.Type == EventKey {
			if ev.Key.Type == KeyEsc || ev.Key.Ch == 'q' || ev.Key.Ch == 'Q' {
				return false
			}
		}
		return true
	})
}
